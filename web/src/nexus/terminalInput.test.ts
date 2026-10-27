import { describe, expect, it } from 'vitest';
import {
  shouldAutoScrollToBottomOnInput,
  shouldForwardTerminalInput,
  stripEmbeddedTerminalMetadata,
  terminalTraceShape,
} from './terminalInput';

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
    '\x1b[12;34R',
    '[12;34R',
  ])('drops leaked focus/CPR sequence %j', (data) => {
    expect(shouldForwardTerminalInput(data)).toBe(false);
  });

  it('forwards real arrow-key input and split-escape continuations', () => {
    for (const data of ['[A', '[B', '[C', '[D', '[A', '[B', '[C', '[D']) {
      expect(shouldForwardTerminalInput(data)).toBe(true);
    }
  });

  it.each([
    '\x1b[<0;10;4M', // SGR click
    '\x1b[<35;58;48M', // SGR any-event motion
    '[<35;58;48M', // bare (no ESC) motion
    '\x1b[<0;10;4m', // SGR release (lowercase m)
    '\x1b[M\x60\x48\x28', // X10 wheel Up (96; 40; 8)
    '\x1b[M\x61\x48\x28', // X10 wheel Down (97; 40; 8)
    '\x1b[96;40;8M', // URXVT wheel Up
    '\x1b[97;40;8M', // URXVT wheel Down
  ])('drops leaked mouse tracking report %j', (data) => {
    expect(shouldForwardTerminalInput(data)).toBe(false);
  });

  it('allows mouse packets through xterm when host will validate negotiated tracking', () => {
    for (const packet of [
      '\x1b[<0;10;4M', // SGR click
      '\x1b[<65;10;4M', // SGR wheel
      '\x1b[M\x60\x48\x28', // X10 wheel
      '\x1b[96;40;8M', // URXVT wheel
    ]) {
      expect(shouldForwardTerminalInput(packet, true)).toBe(true);
      expect(stripEmbeddedTerminalMetadata(packet, true)).toBe(packet);
    }
    expect(stripEmbeddedTerminalMetadata('start\x1b[<0;10;4M\x1b[Ifinish', true)).toBe(
      'start\x1b[<0;10;4Mfinish',
    );
    expect(shouldForwardTerminalInput('[<0;10;4M', true)).toBe(false);
    expect(shouldForwardTerminalInput('[96;40;8M', true)).toBe(false);
    expect(shouldForwardTerminalInput('\x1b[I', true)).toBe(false);
    expect(shouldForwardTerminalInput('\x1b[12;34R', true)).toBe(false);
    expect(shouldForwardTerminalInput('\x1bOA', true)).toBe(true);
  });

  it('keeps keyboard, UTF-8 and explicit Ctrl+C bytes', () => {
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

describe('shouldAutoScrollToBottomOnInput', () => {
  it.each([
    '\x1b[<65;58;48M', // SGR wheel
    '\x1b[M\x60\x48\x28', // X10 wheel
    '\x1b[96;58;48M', // URXVT wheel
    '\x1b[5~', // Page Up
    '\x1b[6;2~', // Shift + Page Down
    '\x1b[H', // Home / start of terminal buffer
    '\x1b[7~', // Alternate Home
    '\x1b[8;2~', // Shift + alternate End
    '\x1bOH', // Application mode Home
    '\x1bOF', // Application mode End
  ])('preserves the current viewport for navigation input %j', (data) => {
    expect(shouldAutoScrollToBottomOnInput(data)).toBe(false);
  });

  it('keeps ordinary keyboard input following live output', () => {
    expect(shouldAutoScrollToBottomOnInput('hello')).toBe(true);
    expect(shouldAutoScrollToBottomOnInput('\r')).toBe(true);
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

  it('preserves all ambiguous bare navigation bytes, including bursts', () => {
    expect(stripEmbeddedTerminalMetadata('> [B[A[A[B[C[D[D[C')).toBe('> [B[A[A[B[C[D[D[C');
    expect(stripEmbeddedTerminalMetadata('[[A[B')).toBe('[[A[B');
    expect(stripEmbeddedTerminalMetadata('[A')).toBe('[A');
    expect(stripEmbeddedTerminalMetadata('typed [A text')).toBe('typed [A text');
    expect(stripEmbeddedTerminalMetadata('[A[B test [D[C')).toBe('[A[B test [D[C');
  });

  it('preserves coalesced ESC-prefixed navigation input', () => {
    expect(stripEmbeddedTerminalMetadata('\x1b[C\x1b[D')).toBe('\x1b[C\x1b[D');
    expect(stripEmbeddedTerminalMetadata('typed\x1b[C\x1b[D')).toBe('typed\x1b[C\x1b[D');
  });

  it('leaves plain text, Kitty and literals intact', () => {
    expect(stripEmbeddedTerminalMetadata('ação')).toBe('ação');
    expect(stripEmbeddedTerminalMetadata('\x1b[1;3u')).toBe('\x1b[1;3u');
    expect(stripEmbeddedTerminalMetadata('/e[31m literal')).toBe('/e[31m literal');
    expect(stripEmbeddedTerminalMetadata('[I] array index')).toBe('[I] array index');
    expect(stripEmbeddedTerminalMetadata('typed\x1b[C')).toBe('typed\x1b[C');
  });

  it('removes coalesced legacy mouse-wheel reports without dropping real arrow keys', () => {
    const mouseUp = '\x1b[M\x60\x48\x28';
    const mouseDown = '\x1b[M\x61\x48\x28';
    expect(
      stripEmbeddedTerminalMetadata(`start${mouseUp}\x1b[A${mouseDown}\x1bOB\x1b[96;40;8M\r`),
    ).toBe('start\x1b[A\x1bOB\r');
    expect(stripEmbeddedTerminalMetadata('choice\x1b[97;40;8M')).toBe('choice');
    expect(shouldForwardTerminalInput('\x1b[M')).toBe(true);
    expect(shouldForwardTerminalInput('\x1b[A')).toBe(true);
    expect(shouldForwardTerminalInput('\x1bOB')).toBe(true);
    expect(shouldForwardTerminalInput('\x03')).toBe(true);
  });

  it('strips embedded SGR mouse tracking reports from mixed chunks', () => {
    expect(stripEmbeddedTerminalMetadata('hello\x1b[<35;10;20Mworld')).toBe('helloworld');
    expect(stripEmbeddedTerminalMetadata('\x1b[<0;10;4Mls\r')).toBe('ls\r');
    expect(stripEmbeddedTerminalMetadata('go\x1b[<35;58;48M\r')).toBe('go\r');
    expect(stripEmbeddedTerminalMetadata('\x1b[<0;1;1m')).toBe('');
  });
});

describe('terminalTraceShape', () => {
  it('does not leak uppercase words that resemble ANSI finals', () => {
    const traced = terminalTraceShape('ARM COMMAND MARCH');
    expect(traced.shape).not.toMatch(/[A-Z]/);
  });

  it('masks ordinary text while preserving control-sequence structure', () => {
    const traced = terminalTraceShape('secret\x1b[A123\r\x03');
    expect(traced.size).toBe(new TextEncoder().encode('secret\x1b[A123\r\x03').length);
    expect(traced.shape).not.toContain('secret');
    expect(traced.shape).not.toContain('123');
    expect(traced.shape).toContain('<1b>[A');
    expect(traced.shape).toContain('<0d>');
    expect(traced.shape).toContain('<03>');
  });

  it('reports UTF-8 byte size without exposing characters', () => {
    const traced = terminalTraceShape('ação');
    expect(traced.size).toBe(new TextEncoder().encode('ação').length);
    expect(traced.shape).not.toContain('ação');
  });
});
