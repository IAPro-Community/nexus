import { describe, expect, it } from 'vitest';
import { sendVoiceTranscriptToTerminal, type TerminalVoiceSocket } from './terminalVoice';

function socket(state = 1, url?: string) {
  const sent: string[] = [];
  const transport: TerminalVoiceSocket = {
    readyState: state,
    send: (data) => sent.push(data),
    ...(url ? { url } : {}),
  };
  return { transport, sent };
}

describe('terminal voice input contract', () => {
  it('inserts the transcript exactly without Enter, CR or LF', () => {
    const { transport, sent } = socket();

    sendVoiceTranscriptToTerminal(transport, 'CONTROL', 'corrija a API', {
      runtimeId: 'runtime-1',
      agentId: 'agent-1',
    });

    expect(sent).toHaveLength(1);
    expect(JSON.parse(sent[0])).toEqual({
      type: 'input',
      data: 'corrija a API',
      runtime_id: 'runtime-1',
      agent_id: 'agent-1',
    });
    expect(JSON.parse(sent[0]).data).not.toMatch(/[\r\n]$/);
  });

  it('preserves transcript whitespace instead of turning dictation into submit', () => {
    const { transport, sent } = socket();

    sendVoiceTranscriptToTerminal(transport, 'CONTROL', '  texto ditado  ', {
      runtimeId: 'runtime-1',
    });

    expect(JSON.parse(sent[0]).data).toBe('  texto ditado  ');
  });

  it('identifies the destination Agent from the terminal socket', () => {
    const { transport, sent } = socket(
      1,
      'ws://127.0.0.1/api/v1/agents/agent-42/terminal?runtime_id=runtime-42',
    );

    sendVoiceTranscriptToTerminal(transport, 'CONTROL', 'continue');

    expect(JSON.parse(sent[0])).toMatchObject({
      runtime_id: 'runtime-42',
      agent_id: 'agent-42',
    });
  });

  it('rejects control characters that could submit or forge PTY input', () => {
    const { transport, sent } = socket();

    expect(() =>
      sendVoiceTranscriptToTerminal(transport, 'CONTROL', 'enviar\r', {
        runtimeId: 'runtime-1',
      }),
    ).toThrow('caracteres de controle');
    expect(sent).toEqual([]);
  });

  it('fails closed when the destination cannot be identified', () => {
    const { transport, sent } = socket();

    expect(() => sendVoiceTranscriptToTerminal(transport, 'CONTROL', 'não enviar')).toThrow(
      'Destino de voz não identificado',
    );
    expect(sent).toEqual([]);
  });

  it('requests CONTROL but refuses to inject voice while the lease is not owned', () => {
    const { transport, sent } = socket();

    expect(() => sendVoiceTranscriptToTerminal(transport, 'READ', 'não enviar')).toThrow(
      'Assuma CONTROL',
    );
    expect(sent.map((frame) => JSON.parse(frame))).toEqual([{ type: 'lease_acquire' }]);
  });

  it('fails closed when the terminal transport is disconnected', () => {
    const { transport, sent } = socket(3);

    expect(() => sendVoiceTranscriptToTerminal(transport, 'CONTROL', 'não enviar')).toThrow(
      'Terminal desconectado',
    );
    expect(sent).toEqual([]);
  });
});
