export interface TerminalFitState {
  disposed: boolean;
  sessionReady: boolean;
  terminalConnected: boolean;
  containerConnected: boolean;
  width: number;
  height: number;
}

export interface TerminalFrameState {
  disposed: boolean;
  socketOpen: boolean;
}

export function canFitTerminal(state: TerminalFitState): boolean {
  return (
    !state.disposed &&
    state.sessionReady &&
    state.terminalConnected &&
    state.containerConnected &&
    state.width > 0 &&
    state.height > 0
  );
}

export function canRunTerminalFrame(state: TerminalFrameState): boolean {
  return !state.disposed && state.socketOpen;
}

export interface TerminalDimensions {
  rows: number;
  cols: number;
}

/** Only send effective PTY dimensions, and avoid resize storms unless an explicit redraw is required. */
export function shouldSendTerminalResize(
  next: TerminalDimensions,
  previous: TerminalDimensions,
  force = false,
): boolean {
  return (
    Number.isInteger(next.rows) &&
    Number.isInteger(next.cols) &&
    next.rows > 0 &&
    next.cols > 0 &&
    (force || next.rows !== previous.rows || next.cols !== previous.cols)
  );
}
