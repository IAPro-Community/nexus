import { describe, expect, it } from 'vitest';
import { sendVoiceTranscriptToTerminal, type TerminalVoiceSocket } from './terminalVoice';

function socket(state = 1) {
  const sent: string[] = [];
  const transport: TerminalVoiceSocket = {
    readyState: state,
    send: (data) => sent.push(data),
  };
  return { transport, sent };
}

describe('terminal voice input contract', () => {
  it('inserts the transcript exactly without Enter, CR or LF', () => {
    const { transport, sent } = socket();

    sendVoiceTranscriptToTerminal(transport, 'CONTROL', 'corrija a API');

    expect(sent).toHaveLength(1);
    expect(JSON.parse(sent[0])).toEqual({ type: 'input', data: 'corrija a API' });
    expect(JSON.parse(sent[0]).data).not.toMatch(/[\r\n]$/);
  });

  it('preserves transcript whitespace instead of turning dictation into submit', () => {
    const { transport, sent } = socket();

    sendVoiceTranscriptToTerminal(transport, 'CONTROL', '  texto ditado  ');

    expect(JSON.parse(sent[0]).data).toBe('  texto ditado  ');
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
