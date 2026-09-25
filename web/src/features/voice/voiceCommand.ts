export type NexusVoiceCommand =
  | { kind: 'command'; commandId: string }
  | { kind: 'voice-settings' }
  | { kind: 'palette' }
  | { kind: 'terminal-dictation'; text: string }
  | { kind: 'unknown' };

export interface VoiceCommandDescriptor {
  id: string;
  label: string;
  keywords?: string[];
}

export interface TerminalVoiceDictationDetail {
  text: string;
  handled: boolean;
  reason?: string;
}

export const NEXUS_TERMINAL_VOICE_EVENT = 'nexus:voice-terminal-dictation';

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

const stripVoiceLead = (value: string) =>
  value
    .replace(
      /^(abrir|abra|mostrar|mostre|ir para|va para|navegar para|navegue para|focar|foque|comando|executar comando)\s+/,
      '',
    )
    .trim();

const terminalDictation = (transcript: string): string | null => {
  const patterns = [
    /^(?:ditar|dite|escrever|escreva|digitar|digite)\s+(?:no|para o|para)\s+terminal\s*[:,]?\s*(.+)$/i,
    /^terminal\s*[:,]?\s*(?:ditar|dite|escrever|escreva|digitar|digite)\s+(.+)$/i,
  ];
  for (const pattern of patterns) {
    const match = transcript.trim().match(pattern);
    if (match?.[1]?.trim()) return match[1].trim();
  }
  return null;
};

const SAFE_COMMAND_IDS = new Set([
  'projects',
  'project-manager',
  'overview',
  'terminals',
  'new-ai-session',
  'project-shell',
  'new-agent',
  'agents',
  'work',
  'plan',
  'resources',
  'maestro-control',
  'sessions',
  'settings',
  'runtime',
  'providers',
  'welcome',
  'tour',
  'toggle-zen',
]);

const SAFE_DYNAMIC_PREFIXES = ['terminal-', 'config-', 'flow-run-', 'preset-'];

export const isVoiceSafeCommandId = (commandId: string) =>
  SAFE_COMMAND_IDS.has(commandId) ||
  SAFE_DYNAMIC_PREFIXES.some((prefix) => commandId.startsWith(prefix));

export const interpretNexusVoiceCommand = (transcript: string): NexusVoiceCommand => {
  const dictated = terminalDictation(transcript);
  if (dictated) return { kind: 'terminal-dictation', text: dictated };

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
    includesAny(value, [
      'abrir comandos',
      'paleta de comandos',
      'command palette',
      'mostrar comandos',
    ])
  ) {
    return { kind: 'palette' };
  }

  const commands: Array<[string, string[]]> = [
    ['project-manager', ['abrir projetos', 'projetos', 'project manager', 'projects']],
    ['overview', ['abrir inicio', 'visao geral', 'overview', 'inicio']],
    ['terminals', ['abrir terminais', 'terminais', 'terminals']],
    ['agents', ['abrir agentes', 'agentes', 'agents']],
    ['new-agent', ['novo agente', 'criar agente', 'new agent']],
    ['work', ['abrir composer', 'composer', 'abrir trabalho', 'work']],
    ['plan', ['abrir missoes', 'missoes', 'missions', 'abrir plano']],
    ['resources', ['abrir recursos', 'recursos', 'resources', 'capacidade']],
    ['sessions', ['abrir sessoes', 'sessoes', 'sessions']],
    ['settings', ['abrir configuracoes', 'configuracoes', 'settings']],
    ['maestro-control', ['abrir maestro', 'controle do maestro', 'maestro control']],
    ['runtime', ['abrir runtimes', 'runtimes']],
    ['providers', ['abrir providers', 'providers', 'provedores']],
    ['project-shell', ['novo terminal', 'abrir novo terminal', 'new terminal']],
    ['new-ai-session', ['nova sessao de ia', 'nova sessao ia', 'new ai session']],
    ['toggle-zen', ['alternar modo zen', 'modo zen', 'zen mode']],
    ['welcome', ['abrir boas vindas', 'boas vindas', 'welcome']],
    ['tour', ['abrir tour', 'tour', 'iniciar tour']],
  ];
  for (const [commandId, phrases] of commands) {
    if (includesAny(value, phrases)) return { kind: 'command', commandId };
  }
  return { kind: 'unknown' };
};

export const resolveNexusVoiceCommand = (
  transcript: string,
  commands: VoiceCommandDescriptor[],
): NexusVoiceCommand => {
  const explicit = interpretNexusVoiceCommand(transcript);
  if (explicit.kind !== 'unknown') return explicit;

  const query = stripVoiceLead(normalize(transcript));
  if (!query) return explicit;

  const candidates = commands.filter((command) => {
    if (!isVoiceSafeCommandId(command.id)) return false;
    const aliases = [
      normalize(command.id.replace(/-/g, ' ')),
      normalize(command.label),
      stripVoiceLead(normalize(command.label)),
      ...(command.keywords ?? []).map(normalize),
    ].filter(Boolean);
    return aliases.includes(query);
  });

  if (candidates.length === 1) {
    return { kind: 'command', commandId: candidates[0].id };
  }
  return { kind: 'unknown' };
};

export const dispatchTerminalVoiceDictation = (text: string): TerminalVoiceDictationDetail => {
  const detail: TerminalVoiceDictationDetail = {
    text: text.trim(),
    handled: false,
  };
  if (!detail.text || typeof window === 'undefined') return detail;
  window.dispatchEvent(new CustomEvent<TerminalVoiceDictationDetail>(NEXUS_TERMINAL_VOICE_EVENT, { detail }));
  return detail;
};
