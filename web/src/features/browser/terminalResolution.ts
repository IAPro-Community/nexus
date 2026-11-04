import type { Agent, RuntimeSession } from '../../types';

export interface ResolvedTerminalInfo {
  isAgent: boolean;
  terminalLabel: string;
  provider: string;
  profile: string;
  agentName?: string;
  tooltip: string;
}

/**
 * Resolves a human-readable terminal/agent identifier from a runtime session ID.
 * Accurately pairs the controlling runtime with agent names, provider profiles,
 * or descriptive fallbacks instead of cryptical hash strings.
 */
export function resolveTerminalInfo(
  ownerRuntimeId?: string,
  runtimes?: readonly RuntimeSession[],
  agents?: readonly Agent[],
): ResolvedTerminalInfo {
  if (!ownerRuntimeId) {
    return {
      isAgent: false,
      terminalLabel: 'Manual',
      provider: '',
      profile: '',
      tooltip: 'Aba aberta pelo usuário (sem agente vinculado)',
    };
  }

  const runtime = runtimes?.find((r) => r.runtime_id === ownerRuntimeId);
  const agent = runtime?.agent_id
    ? agents?.find((a) => a.id === runtime.agent_id)
    : agents?.find((a) => a.id === ownerRuntimeId);

  const provider = (runtime?.provider || runtime?.provider_id || '').toUpperCase();
  const profile = runtime?.profile || runtime?.profile_id || '';
  const agentName = agent?.name?.trim() || undefined;

  let terminalLabel = '';

  if (agentName) {
    terminalLabel = agentName;
  } else if (provider) {
    if (profile && profile !== 'default') {
      terminalLabel = `${provider} (${profile})`;
    } else {
      terminalLabel = provider;
    }
  } else if (runtime?.title?.trim()) {
    terminalLabel = runtime.title.trim();
  } else {
    terminalLabel = `Terminal · ${ownerRuntimeId.slice(0, 6)}`;
  }

  const tooltip = `Controlado por ${terminalLabel} (Runtime: ${ownerRuntimeId.slice(0, 8)})`;

  return {
    isAgent: true,
    terminalLabel,
    provider,
    profile,
    agentName,
    tooltip,
  };
}
