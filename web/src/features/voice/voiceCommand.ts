export type NexusVoiceCommand =
  | { kind: 'command'; commandId: string }
  | { kind: 'voice-settings' }
  | { kind: 'palette' }
  | { kind: 'unknown' };

const normalize = (value: string) =>
  value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();

const includesAny = (value: string, phrases: string[]) =>
  phrases.some((phrase) => value === phrase || value.includes(phrase));

export const interpretNexusVoiceCommand = (transcript: string): NexusVoiceCommand => {
  const value = normalize(transcript);
  if (!value) return { kind: 'unknown' };

  if (
    includesAny(value, [
      'configurar voz',
      'configuracoes de voz',
      'configuracao de voz',
      'voice settings',
      'acessibilidade de voz',
    ])
  ) {
    return { kind: 'voice-settings' };
  }
  if (
    includesAny(value, ['abrir comandos', 'paleta de comandos', 'command palette', 'mostrar comandos'])
  ) {
    return { kind: 'palette' };
  }

  const commands: Array<[string, string[]]> = [
    ['project-manager', ['abrir projetos', 'projetos', 'project manager', 'projects']],
    ['overview', ['abrir inicio', 'visao geral', 'overview', 'inicio']],
    ['terminals', ['abrir terminais', 'terminais', 'terminals']],
    ['agents', ['abrir agentes', 'agentes', 'agents']],
    ['work', ['abrir composer', 'composer', 'abrir trabalho', 'work']],
    ['plan', ['abrir missoes', 'missoes', 'missions', 'abrir plano']],
    ['resources', ['abrir recursos', 'recursos', 'resources', 'capacidade']],
    ['sessions', ['abrir sessoes', 'sessoes', 'sessions']],
    ['settings', ['abrir configuracoes', 'configuracoes', 'settings']],
    ['project-shell', ['novo terminal', 'abrir novo terminal', 'new terminal']],
    ['new-ai-session', ['nova sessao de ia', 'nova sessao ia', 'new ai session']],
  ];
  for (const [commandId, phrases] of commands) {
    if (includesAny(value, phrases)) return { kind: 'command', commandId };
  }
  return { kind: 'unknown' };
};
