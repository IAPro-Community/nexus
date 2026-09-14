import { describe, expect, it } from 'vitest';
import { shouldForwardTerminalInput, stripEmbeddedTerminalMetadata } from './terminalInput';

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

describe('stripEmbeddedTerminalMetadata', () => {
  it('strips coalesced ESC focus/CPR reports but keeps typed keystrokes', () => {
    expect(stripEmbeddedTerminalMetadata('\x1b[Ils\r')).toBe('ls\r');
    expect(stripEmbeddedTerminalMetadata('go\x1b[12;34R\r')).toBe('go\r');
    expect(stripEmbeddedTerminalMetadata('\x1b[I')).toBe('');
  });

  it('leaves plain text, mouse, Kitty and literals intact', () => {
    expect(stripEmbeddedTerminalMetadata('ação')).toBe('ação');
    expect(stripEmbeddedTerminalMetadata('\x1b[<0;10;4M')).toBe('\x1b[<0;10;4M');
    expect(stripEmbeddedTerminalMetadata('\x1b[1;3u')).toBe('\x1b[1;3u');
    expect(stripEmbeddedTerminalMetadata('/e[31m literal')).toBe('/e[31m literal');
    expect(stripEmbeddedTerminalMetadata('[I] array index')).toBe('[I] array index');
  });
});
