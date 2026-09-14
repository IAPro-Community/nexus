import { describe, expect, it } from 'vitest';
import {
  isProtocolControlLine,
  parseTerminalFrame,
  ProtocolScrubber,
  scrubProtocolOutput,
} from './terminalProtocol';

describe('terminal protocol', () => {
  it('renders provider JSON as output when it is not a control envelope', () => {
    expect(parseTerminalFrame('{"hello":"world"}')).toEqual({
      type: 'output',
      data: '{"hello":"world"}',
    });
  });

  it('parses agent_state without terminal output', () => {
    expect(
      parseTerminalFrame(
        JSON.stringify({
          type: 'agent_state',
          agent_id: 'a1',
          state: 'WORKING',
        }),
      ),
    ).toEqual({ type: 'agent_state', agent_id: 'a1', state: 'WORKING' });
  });

  it('parses continuity_state without terminal output', () => {
    expect(
      parseTerminalFrame(
        JSON.stringify({
          type: 'continuity_state',
          agent_id: 'a1',
          continuity_state: 'PRESERVED',
        }),
      ),
    ).toEqual({ type: 'continuity_state', agent_id: 'a1', continuity_state: 'PRESERVED' });
  });

  it('keeps ANSI payload byte-for-byte', () => {
    const ansi = '\u001b[32mPASS\u001b[0m\r\n';
    expect(parseTerminalFrame(JSON.stringify({ type: 'output', data: ansi }))).toEqual({
      type: 'output',
      data: ansi,
    });
  });

  it('does not rewrite provider text that resembles an escaped ANSI sequence', () => {
    const literal = '/e[31m literal text and \\e[31m literal text';
    expect(parseTerminalFrame(JSON.stringify({ type: 'output', data: literal }))).toEqual({
      type: 'output',
      data: literal,
    });
  });

  it('isolates valid control frames from terminal output', () => {
    expect(parseTerminalFrame('{"type":"lease","role":"CONTROL"}')).toEqual({
      type: 'control',
      event: 'lease',
      payload: { role: 'CONTROL' },
    });
  });

  it('does not leak malformed or unknown typed frames into xterm', () => {
    expect(parseTerminalFrame('{"type":"secret_control","token":"secret"}')).toEqual({
      type: 'unknown',
      raw: '{"type":"secret_control","token":"secret"}',
    });
    expect(parseTerminalFrame('{"type":"output"}')).toEqual({
      type: 'unknown',
      raw: '{"type":"output"}',
    });
  });

  it('scrubs lease_acquire frames from polluted PTY output', () => {
    const polluted =
      'welcome\n' +
      '{"version":1,"id":"req-1","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}\n' +
      'real line\n' +
      '{"version":1,"ok":true,"data":"\\"lease_acquired\\""}\n' +
      '{"note":"keep me"}\n';
    expect(scrubProtocolOutput(polluted)).toBe('welcome\nreal line\n{"note":"keep me"}\n');
    expect(scrubProtocolOutput('{"version":1,"command":"lease_acquire"}')).toBe('');
    expect(isProtocolControlLine('hello')).toBe(false);
  });

  it('drops control RPC split across two output messages', () => {
    const scrubber = new ProtocolScrubber();
    expect(scrubber.scrub('welcome\n{"version":1,"comm')).toBe('welcome\n');
    expect(scrubber.scrub('and":"lease_acquire"}\nreal line\n')).toBe('real line\n');
  });

  it('renders ordinary trailing text immediately and flushes leftovers', () => {
    const scrubber = new ProtocolScrubber();
    expect(scrubber.scrub('prompt$ ')).toBe('prompt$ ');
    expect(scrubber.scrub('{"note":"keep me"}')).toBe('');
    expect(scrubber.flush()).toBe('{"note":"keep me"}');
  });
});
