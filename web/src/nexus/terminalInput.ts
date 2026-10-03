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
    // Navigation bytes are never classified as metadata here. A split xterm
    // callback may deliver ESC first and "[A"/""[B]" next; the server-side
    // stateful coalescer must receive both pieces intact.
    // eslint-disable-next-line no-control-regex
    /^\x1b\[\d+;\d+R$/.test(data) ||
    /^\[\d+;\d+R$/.test(data) ||
    // SGR mouse tracking reports (with or without leading ESC) are terminal
    // metadata, never user input. The host terminal emits these when mouse
    // tracking modes (?1000/?1003 + ?1006) are active.
    // eslint-disable-next-line no-control-regex
    /^\x1b?\[<\d+;\d+;\d+[Mm]$/.test(data)
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
// eslint-disable-next-line no-control-regex
const EMBEDDED_MOUSE_RE = /\x1b\[<\d+;\d+;\d+[Mm]/g;

// Some bridges have been observed to lose ESC entirely and coalesce multiple
// navigation reports into one onData chunk (for example "[B[A[A[B[C").
// A single bare "[A" is ambiguous user text and must remain untouched; two or
// more navigation-shaped reports in the same chunk are transport corruption.
const BARE_NAVIGATION_RE = /\[(?:[0-9;?]*\[)?[A-DHFO]/g;

function stripBareNavigationBurst(data: string): string {
  const matches = [...data.matchAll(BARE_NAVIGATION_RE)].filter(
    (match) => match.index === 0 || data[match.index! - 1] !== '\x1b',
  );
  if (matches.length < 2) return data;
  return data.replace(BARE_NAVIGATION_RE, (match, offset: number) =>
    offset > 0 && data[offset - 1] === '\x1b' ? match : '',
  );
}

/** Removes terminal metadata and unambiguous bare navigation corruption. */
export function stripEmbeddedTerminalMetadata(data: string): string {
  if (!data) return data;
  const withoutMetadata = data
    .replace(EMBEDDED_FOCUS_RE, '')
    .replace(EMBEDDED_CPR_RE, '')
    .replace(EMBEDDED_MOUSE_RE, '');
  return stripBareNavigationBurst(withoutMetadata);
}

/**
 * Privacy-safe terminal byte signature for NEXUS_TERMINAL_TRACE.
 * Ordinary printable content is masked; only control bytes and terminal
 * syntax remain visible so CSI/SS3 fragmentation can be correlated.
 */
export function terminalTraceShape(data: string): { size: number; shape: string } {
  const bytes = new TextEncoder().encode(data);
  const limit = Math.min(bytes.length, 256);
  let shape = '';
  const visibleSyntax = new Set(
    Array.from('[];<>?~OABCDHFIRMmuhl').map((value) => value.charCodeAt(0)),
  );
  for (let index = 0; index < limit; index += 1) {
    const value = bytes[index]!;
    if (value === 0x1b) {
      shape += '<1b>';
    } else if (value < 0x20 || value === 0x7f) {
      shape += `<${value.toString(16).padStart(2, '0')}>`;
    } else if (value >= 0x30 && value <= 0x39) {
      shape += '#';
    } else if (visibleSyntax.has(value)) {
      shape += String.fromCharCode(value);
    } else if (value >= 0x20 && value <= 0x7e) {
      shape += '.';
    } else {
      shape += '*';
    }
  }
  if (bytes.length > limit) shape += '…';
  return { size: bytes.length, shape };
}
