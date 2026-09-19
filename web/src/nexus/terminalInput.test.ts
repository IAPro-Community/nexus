import { describe, expect, it } from 'vitest';
import { shouldForwardTerminalInput, stripEmbeddedTerminalMetadata } from './terminalInput';

describe('shouldForwardTerminalInput', () => {
  it.each([
    '\x1b[I',
    '\x1b[O',
    '[I',
    '[O',
    '\x1b[i',
    '[i',
    '\x1b[0[I',
    '[0[I',
    '\x1b[1;2O',
    '[1;2O',
    '[A',
    '\x1b[12;34R',
    '[12;34R',
  ])('drops leaked focus/CPR sequence %j', (data) => {
    expect(shouldForwardTerminalInput(data)).toBe(false);
  });

  it('forwards real arrow-key input', () => {
    for (const data of ['[A', '[B', '[C', '[D']) {
      expect(shouldForwardTerminalInput(data)).toBe(true);
    }
  });

  it('keeps mouse, keyboard, UTF-8 and explicit Ctrl+C bytes', () => {
    expect(shouldForwardTerminalInput('\x1b[<0;10;4M')).toBe(true);
    expect(shouldForwardTerminalInput('\x1b[1;3u')).toBe(true);
    expect(shouldForwardTerminalInput('ação')).toBe(true);
    expect(shouldForwardTerminalInput('\x03')).toBe(true);
    expect(shouldForwardTerminalInput('\x16')).toBe(true);
  });

  it('keeps literal provider text that resembles a control sequence', () => {
    expect(shouldForwardTerminalInput('/e[31m literal')).toBe(true);
    expect(shouldForwardTerminalInput('\\e[31m literal')).toBe(true);
  });
});

describe('stripEmbeddedTerminalMetadata', () => {
  it('strips coalesced ESC focus/CPR reports but keeps typed keystrokes', () => {
    expect(stripEmbeddedTerminalMetadata('\x1b[Ils\r')).toBe('ls\r');
    expect(stripEmbeddedTerminalMetadata('/mo\x1b[0[I')).toBe('/mo');
    expect(stripEmbeddedTerminalMetadata('\x1b[1;2Ogo')).toBe('go');
    expect(stripEmbeddedTerminalMetadata('go\x1b[12;34R\r')).toBe('go\r');
    expect(stripEmbeddedTerminalMetadata('\x1b[I')).toBe('');
  });

  it('drops standalone bare reports but preserves literal command text', () => {
    expect(shouldForwardTerminalInput('[I')).toBe(false);
    expect(shouldForwardTerminalInput('[12;34R')).toBe(false);
    expect(stripEmbeddedTerminalMetadata('[I] literal')).toBe('[I] literal');
    expect(stripEmbeddedTerminalMetadata('/status [12;34R literal')).toBe(
      '/status [12;34R literal',
    );
  });

  it('strips bursts whose ESC bytes were lost by the terminal bridge', () => {
    expect(stripEmbeddedTerminalMetadata('> [B[A[A[B[C[D[D[C')).toBe('> ');
    expect(stripEmbeddedTerminalMetadata('[[A[B')).toBe('');
    expect(stripEmbeddedTerminalMetadata('[A')).toBe('[A');
    expect(stripEmbeddedTerminalMetadata('typed [A text')).toBe('typed [A text');
  });

  it('preserves coalesced ESC-prefixed navigation input', () => {
    expect(stripEmbeddedTerminalMetadata('\x1b[C\x1b[D')).toBe('\x1b[C\x1b[D');
    expect(stripEmbeddedTerminalMetadata('typed\x1b[C\x1b[D')).toBe('typed\x1b[C\x1b[D');
  });

  it('leaves plain text, mouse, Kitty and literals intact', () => {
    expect(stripEmbeddedTerminalMetadata('ação')).toBe('ação');
    expect(stripEmbeddedTerminalMetadata('\x1b[<0;10;4M')).toBe('\x1b[<0;10;4M');
    expect(stripEmbeddedTerminalMetadata('\x1b[1;3u')).toBe('\x1b[1;3u');
    expect(stripEmbeddedTerminalMetadata('/e[31m literal')).toBe('/e[31m literal');
    expect(stripEmbeddedTerminalMetadata('[I] array index')).toBe('[I] array index');
    expect(stripEmbeddedTerminalMetadata('typed\x1b[C')).toBe('typed\x1b[C');
  });
});
