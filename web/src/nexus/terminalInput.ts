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
const EMBEDDED_FOCUS_RE = /\x1b\[[iIO]/g;
// eslint-disable-next-line no-control-regex
const EMBEDDED_CPR_RE = /\x1b\[\d+;\d+R/g;

/** Removes embedded ESC-prefixed focus/CPR metadata from a forwarded chunk. */
export function stripEmbeddedTerminalMetadata(data: string): string {
  if (!data || !data.includes('\x1b')) return data;
  return data.replace(EMBEDDED_FOCUS_RE, '').replace(EMBEDDED_CPR_RE, '');
}
