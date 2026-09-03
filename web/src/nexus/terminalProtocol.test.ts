import { describe, expect, it } from 'vitest';
import { attentionNotificationFromFrame, parseTerminalFrame } from './terminalProtocol';

describe('terminal protocol', () => {
  it('renders provider JSON as output when it is not a control envelope', () => {
    expect(parseTerminalFrame('{"hello":"world"}')).toEqual({ type: 'output', data: '{"hello":"world"}' });
  });

  it('parses agent_state without terminal output', () => {
    expect(parseTerminalFrame(JSON.stringify({
      type: 'agent_state',
      agent_id: 'a1',
      state: 'WORKING',
    }))).toEqual({ type: 'agent_state', agent_id: 'a1', state: 'WORKING' });
  });

  it('parses continuity_state without terminal output', () => {
    expect(parseTerminalFrame(JSON.stringify({
      type: 'continuity_state',
      agent_id: 'a1',
      continuity_state: 'PRESERVED',
    }))).toEqual({ type: 'continuity_state', agent_id: 'a1', continuity_state: 'PRESERVED' });
  });

  it('keeps ANSI payload byte-for-byte', () => {
    const ansi = '\u001b[32mPASS\u001b[0m\r\n';
    expect(parseTerminalFrame(JSON.stringify({ type: 'output', data: ansi })))
      .toEqual({ type: 'output', data: ansi });
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

  it('maps attention control frames into push payloads with routing metadata only', () => {
    const payload = attentionNotificationFromFrame(
      parseTerminalFrame(JSON.stringify({
        type: 'attention',
        runtime_id: 'rt-1',
        project_id: 'prj-1',
        project_name: 'Payments',
        attention_reason: 'QUESTION',
        attention_event_id: 'evt-1',
        view: 'work',
        context: 'Need approval',
        summary: 'Approval pending',
        dynamic_title: 'transcript: Bearer sk-live-secret',
        transcript: 'full transcript should not be forwarded',
        credentials: { token: 'sk-live-secret' },
      })),
      'fallback-runtime',
    );

    expect(payload).toEqual({
      runtimeId: 'rt-1',
      projectId: 'prj-1',
      projectName: 'Payments',
      reason: 'QUESTION',
      attentionEventId: 'evt-1',
      view: 'work',
      context: 'Need approval',
      summary: 'Approval pending',
      dynamicTitle: 'transcript: Bearer sk-live-secret',
    });
    expect(payload && 'transcript' in payload).toBe(false);
    expect(payload && 'credentials' in payload).toBe(false);
  });
});
