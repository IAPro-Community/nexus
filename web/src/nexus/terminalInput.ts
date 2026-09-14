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
