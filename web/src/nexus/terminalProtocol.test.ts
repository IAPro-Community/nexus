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
      '{"version":1,"ok":true,"data":"\\"lease_acquired\\"","timestamp":"2026-01-01T00:00:00Z"}\n' +
      '{"note":"keep me"}\n';
    expect(scrubProtocolOutput(polluted)).toBe('welcome\nreal line\n{"note":"keep me"}\n');
    expect(scrubProtocolOutput('{"version":1,"id":"req-2","command":"lease_acquire","timestamp":"2026-01-01T00:00:00Z"}')).toBe('');
    expect(isProtocolControlLine('hello')).toBe(false);
    // Provider diagnostics are content even when they use common RPC names.
    expect(isProtocolControlLine('{"command":"npm install","ok":true}')).toBe(false);
    expect(isProtocolControlLine('{"version":1,"ok":true,"data":"provider"}')).toBe(false);
    expect(scrubProtocolOutput('{"command":"npm install","ok":true}\n')).toBe(
      '{"command":"npm install","ok":true}\n',
    );
  });

  it('drops control RPC split across two output messages', () => {
    const scrubber = new ProtocolScrubber();
    expect(scrubber.scrub('welcome\n{"version":1,"comm')).toBe('welcome\n');
    expect(scrubber.scrub('and":"lease_acquire","id":"req-1","timestamp":"2026-01-01T00:00:00Z"}\nreal line\n')).toBe('real line\n');
  });

  it('renders provider JSON immediately instead of holding it as a hypothetical RPC', () => {
    const scrubber = new ProtocolScrubber();
    expect(scrubber.scrub('prompt$ ')).toBe('prompt$ ');
    expect(scrubber.scrub('{"note":"keep me"}')).toBe('{"note":"keep me"}');
    expect(scrubber.scrub('{"command":"npm test","ok":true}')).toBe('{"command":"npm test","ok":true}');
    expect(scrubber.flush()).toBe('');
  });

  it('holds only partial Nexus envelopes until their final bytes arrive', () => {
    const scrubber = new ProtocolScrubber();
    expect(scrubber.scrub('{"ver')).toBe('');
    expect(scrubber.scrub('sion":1,"ok":true,"timestamp":"2026-01-01T00:00:00Z"}\n')).toBe('');
  });
});
