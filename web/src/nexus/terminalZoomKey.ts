import {
  adjustTerminalFontSize,
  resetTerminalFontSize,
  storeTerminalFontSize,
} from './terminalSettings';

export type TerminalZoomKeyEvent = Pick<
  KeyboardEvent,
  'ctrlKey' | 'metaKey' | 'key' | 'code' | 'type'
>;

/** Interpret shortcuts without consuming unrelated provider/TUI keystrokes. */
export function terminalZoomAction(event: TerminalZoomKeyEvent): 'in' | 'out' | 'reset' | null {
  if (!event.ctrlKey && !event.metaKey) return null;
  if (['=', '+'].includes(event.key) || ['Equal', 'NumpadAdd'].includes(event.code)) return 'in';
  if (['-', '_'].includes(event.key) || ['Minus', 'NumpadSubtract'].includes(event.code)) {
    return 'out';
  }
  if (event.key === '0' || ['Digit0', 'Numpad0'].includes(event.code)) return 'reset';
  return null;
}

/** xterm expects false for intercepted keys, including keyup (no double zoom). */
export function handleTerminalZoomKey(event: KeyboardEvent): boolean {
  const action = terminalZoomAction(event);
  if (action === null) return true;
  if (event.type !== 'keydown') return false;
  const fontSize =
    action === 'reset'
      ? resetTerminalFontSize()
      : adjustTerminalFontSize(action === 'in' ? 1 : -1);
  storeTerminalFontSize(fontSize);
  return false;
}
