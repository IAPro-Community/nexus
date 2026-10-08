import { describe, expect, it } from 'vitest';
import { terminalZoomAction, type TerminalZoomKeyEvent } from './terminalZoomKey';

const key = (
  changes: Partial<TerminalZoomKeyEvent>,
): TerminalZoomKeyEvent => ({
  ctrlKey: false,
  metaKey: false,
  key: '',
  code: '',
  type: 'keydown',
  ...changes,
});

describe('terminal zoom shortcuts', () => {
  it('reserves existing Ctrl/Cmd shortcuts without changing provider input', () => {
    expect(terminalZoomAction(key({ ctrlKey: true, key: '+', code: 'Equal' }))).toBe('in');
    expect(terminalZoomAction(key({ metaKey: true, key: '=', code: 'Equal' }))).toBe('in');
    expect(terminalZoomAction(key({ ctrlKey: true, code: 'NumpadAdd' }))).toBe('in');
    expect(terminalZoomAction(key({ ctrlKey: true, key: '_', code: 'Minus' }))).toBe('out');
    expect(terminalZoomAction(key({ ctrlKey: true, code: 'NumpadSubtract' }))).toBe('out');
    expect(terminalZoomAction(key({ ctrlKey: true, code: 'Numpad0' }))).toBe('reset');
  });

  it('does not steal unmodified keys, arrows, or other control chords', () => {
    expect(terminalZoomAction(key({ key: '+', code: 'Equal' }))).toBeNull();
    expect(terminalZoomAction(key({ key: 'ArrowDown', code: 'ArrowDown', ctrlKey: true }))).toBeNull();
    expect(terminalZoomAction(key({ key: 'c', code: 'KeyC', ctrlKey: true }))).toBeNull();
  });
});
