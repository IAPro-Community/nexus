import { describe, expect, it } from 'vitest';
import { shouldForwardTerminalInput } from './terminalInput';

describe('shouldForwardTerminalInput', () => {
  it.each(['\x1b[I', '\x1b[O', '[I', '[O', '\x1b[i', '[i', '\x1b[12;34R', '[12;34R'])(
    'drops leaked focus/CPR sequence %j',
    (data) => {
      expect(shouldForwardTerminalInput(data)).toBe(false);
    },
  );

  it('keeps mouse, keyboard, UTF-8 and explicit Ctrl+C bytes', () => {
    expect(shouldForwardTerminalInput('\x1b[<0;10;4M')).toBe(true);
    expect(shouldForwardTerminalInput('\x1b[1;3u')).toBe(true);
    expect(shouldForwardTerminalInput('ação')).toBe(true);
    expect(shouldForwardTerminalInput('\x03')).toBe(true);
  });

  it('keeps literal provider text that resembles a control sequence', () => {
    expect(shouldForwardTerminalInput('/e[31m literal')).toBe(true);
    expect(shouldForwardTerminalInput('\\e[31m literal')).toBe(true);
  });
});
