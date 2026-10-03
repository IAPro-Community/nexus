import { describe, expect, it } from 'vitest';
import {
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
  ])('drops leaked SGR mouse tracking report %j', (data) => {
    expect(shouldForwardTerminalInput(data)).toBe(false);
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

  it('strips unambiguous bare navigation bursts but preserves a single literal', () => {
    expect(stripEmbeddedTerminalMetadata('> [B[A[A[B[C[D[D[C')).toBe('> ');
    expect(stripEmbeddedTerminalMetadata('[[A[B')).toBe('');
    expect(stripEmbeddedTerminalMetadata('[A')).toBe('[A');
    expect(stripEmbeddedTerminalMetadata('typed [A text')).toBe('typed [A text');
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

  it('strips embedded SGR mouse tracking reports from mixed chunks', () => {
    expect(stripEmbeddedTerminalMetadata('hello\x1b[<35;10;20Mworld')).toBe('helloworld');
    expect(stripEmbeddedTerminalMetadata('\x1b[<0;10;4Mls\r')).toBe('ls\r');
    expect(stripEmbeddedTerminalMetadata('go\x1b[<35;58;48M\r')).toBe('go\r');
    expect(stripEmbeddedTerminalMetadata('\x1b[<0;1;1m')).toBe('');
  });
});

describe('terminalTraceShape', () => {
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
