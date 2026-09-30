export type TerminalVoiceSocket = {
  readyState: number;
  send: (data: string) => void;
};

const SOCKET_OPEN = 1;

export function sendVoiceTranscriptToTerminal(
  socket: TerminalVoiceSocket | null,
  role: string,
  text: string,
): void {
  if (role !== 'CONTROL') {
    if (socket?.readyState === SOCKET_OPEN) {
      socket.send(JSON.stringify({ type: 'lease_acquire' }));
    }
    throw new Error('Assuma CONTROL antes de inserir voz neste terminal.');
  }
  if (!socket || socket.readyState !== SOCKET_OPEN) {
    throw new Error('Terminal desconectado.');
  }

  // Voice dictation is an insertion operation, not a submit operation.
  // Preserve the transcript byte-for-byte and never append Enter/CR/LF.
  socket.send(JSON.stringify({ type: 'input', data: text }));
}
