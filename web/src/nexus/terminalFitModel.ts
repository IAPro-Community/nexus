export interface TerminalFitState {
  disposed: boolean;
  sessionReady: boolean;
  terminalConnected: boolean;
  containerConnected: boolean;
  width: number;
  height: number;
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
