export type TerminalVoiceSocket = {
  readyState: number;
  send: (data: string) => void;
  url?: string;
};

/**
 * Destination of a voice action. A voice dictation must name the runtime and,
 * when known, the Agent that receives it; an unidentified destination fails
 * closed before any byte reaches the terminal.
 */
export type TerminalVoiceTarget = {
  runtimeId?: string;
  agentId?: string;
  label?: string;
  agentLabel?: string;
};

const SOCKET_OPEN = 1;

/** Control characters would submit the turn (CR/LF) or forge PTY sequences. */
function containsControlCharacter(text: string): boolean {
  for (const character of text) {
    const code = character.charCodeAt(0);
    if (code <= 0x1f || code === 0x7f) return true;
  }
  return false;
}

function resolveVoiceTarget(
  socket: TerminalVoiceSocket,
  target?: TerminalVoiceTarget,
): { runtimeId?: string; agentId?: string } {
  let runtimeId = target?.runtimeId?.trim() || '';
  let agentId = target?.agentId?.trim() || '';

  if (socket.url) {
    try {
      const url = new URL(socket.url);
      runtimeId = runtimeId || url.searchParams.get('runtime_id')?.trim() || '';
      if (!agentId) {
        const match = url.pathname.match(/\/api\/v1\/agents\/([^/]+)\/terminal(?:\/|$)/);
        if (match) agentId = decodeURIComponent(match[1]);
      }
      if (!runtimeId) {
        const match = url.pathname.match(/\/api\/v1\/runtimes\/([^/]+)\/terminal(?:\/|$)/);
        if (match) runtimeId = decodeURIComponent(match[1]);
      }
    } catch {
      // Invalid URL cannot provide a destination; explicit target remains usable.
    }
  }

  return { runtimeId: runtimeId || undefined, agentId: agentId || undefined };
}

export function sendVoiceTranscriptToTerminal(
  socket: TerminalVoiceSocket | null,
  role: string,
  text: string,
  target?: TerminalVoiceTarget,
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
  const destination = resolveVoiceTarget(socket, target);
  if (!destination.runtimeId && !destination.agentId) {
    throw new Error('Destino de voz não identificado.');
  }
  if (containsControlCharacter(text)) {
    throw new Error('Transcrição de voz contém caracteres de controle.');
  }

  // Voice dictation is an insertion operation, not a submit operation.
  // Preserve the transcript byte-for-byte and never append Enter/CR/LF.
  socket.send(
    JSON.stringify({
      type: 'input',
      data: text,
      runtime_id: destination.runtimeId,
      agent_id: destination.agentId,
    }),
  );
}
