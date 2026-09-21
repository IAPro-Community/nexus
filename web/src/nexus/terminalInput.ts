/** Returns whether xterm input is safe to forward to the provider PTY. */
export function shouldForwardTerminalInput(data: string): boolean {
  if (!data) return false;

  // Browser focus reporting and CPR responses can leak from xterm while a
  // workspace panel is being switched. They are terminal metadata, not user
  // input. Some browser/PTY paths lose the leading ESC, so accept both forms.
  if (
    data === '\x1b[I' ||
    data === '\x1b[O' ||
    data === '[I' ||
    data === '[O' ||
    data === '\x1b[i' ||
    data === '[i' ||
    // Some terminal bridges encode focus reports with a parameter, e.g.
    // ESC[0[I. The ESC is invisible when this leaks into the prompt, leaving
    // the user-facing residue "0[I".
    // eslint-disable-next-line no-control-regex
    /^\x1b\[[0-9;]*(?:\[[iIO]|[iIO])$/.test(data) ||
    /^\[[0-9;]*(?:\[[iIO]|[iIO])$/.test(data) ||
    // ESC-prefixed CSI arrows are real user input and must pass through;
    // only ESC-less forms are eligible for navigation metadata filtering.
    /^\[[0-9;?]*[A-DHFO]$/.test(data) ||
    // eslint-disable-next-line no-control-regex
    /^\x1b\[\d+;\d+R$/.test(data) ||
    /^\[\d+;\d+R$/.test(data)
  ) {
    return false;
  }
  return true;
}

// xterm may coalesce a leaked focus/CPR report with real keystrokes in one
// onData chunk, bypassing the exact-match filter above. Strip embedded
// ESC-prefixed metadata and forward the remainder. Bare (ESC-less) forms are
// only dropped on exact match: inside a larger chunk they are
// indistinguishable from literal user text such as `/e[31m`.
// eslint-disable-next-line no-control-regex
const EMBEDDED_FOCUS_RE = /\x1b\[[0-9;]*(?:\[[iIO]|[iIO])/g;
// eslint-disable-next-line no-control-regex
const EMBEDDED_CPR_RE = /\x1b\[\d+;\d+R/g;
// A broken terminal bridge can drop ESC and coalesce arrow/focus reports into
// a burst such as "[B[A[A[B[C" (or even "[[A"). Only scrub bare reports when
// there is a burst, or when the whole chunk is one report, so ordinary text
// such as "[A" remains usable.
const BARE_NAVIGATION_RE = /\[(?:[0-9;?]*\[)?[A-DHFO]/g;

function stripBareNavigationBurst(data: string): string {
  const matches = [...data.matchAll(BARE_NAVIGATION_RE)].filter(
    (match) => match.index === 0 || data[match.index! - 1] !== '\x1b',
  );
  if (matches.length < 2) return data;
  return data.replace(BARE_NAVIGATION_RE, (match, offset: number) => {
    return offset > 0 && data[offset - 1] === '\x1b' ? match : '';
  });
}

/** Removes embedded ESC-prefixed focus/CPR metadata from a forwarded chunk. */
export function stripEmbeddedTerminalMetadata(data: string): string {
  if (!data) return data;
  const withoutEscMetadata = data.replace(EMBEDDED_FOCUS_RE, '').replace(EMBEDDED_CPR_RE, '');
  return stripBareNavigationBurst(withoutEscMetadata);
}
