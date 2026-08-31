import { describe, expect, it } from 'vitest';
import { parseTerminalFrame } from './terminalProtocol';

describe('terminal protocol', () => {
  it('renders provider JSON as output when it is not a control envelope', () => {
    expect(parseTerminalFrame('{"hello":"world"}')).toEqual({ type: 'output', data: '{"hello":"world"}' });
  });
  it('keeps ANSI payload byte-for-byte', () => {
    const ansi = '\u001b[32mPASS\u001b[0m\r\n';
    expect(parseTerminalFrame(JSON.stringify({ type: 'output', data: ansi }))).toEqual({ type: 'output', data: ansi });
  });
  it('isolates valid control frames from terminal output', () => {
    expect(parseTerminalFrame('{"type":"lease","role":"CONTROL"}')).toEqual({ type: 'control', event: 'lease', payload: { role: 'CONTROL' } });
  });
  it('parses agent and continuity state without terminal output', () => {
    expect(parseTerminalFrame(JSON.stringify({ type: 'agent_state', agent_id: 'a1', state: 'WORKING' })))
      .toEqual({ type: 'agent_state', agent_id: 'a1', state: 'WORKING' });
    expect(parseTerminalFrame(JSON.stringify({ type: 'continuity_state', agent_id: 'a1', continuity_state: 'NATIVE_RESUME_VERIFIED' })))
      .toEqual({ type: 'continuity_state', agent_id: 'a1', continuity_state: 'NATIVE_RESUME_VERIFIED' });
  });
  it('reports malformed/unknown control frames without leaking JSON', () => {
    expect(parseTerminalFrame('{"type":"secret_control"}')).toEqual({ type: 'unknown', raw: '{"type":"secret_control"}' });
    expect(parseTerminalFrame('{"type":"output"}')).toEqual({ type: 'unknown', raw: '{"type":"output"}' });
  });
});
