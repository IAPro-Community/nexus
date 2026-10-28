import type { AgentConfig } from '../../types';

/** Provider-native transports currently exposed by the experimental SessionHost. */
export type StructuredHarnessProtocol = '' | 'acp' | 'pi_rpc';

const protocols: Record<string, Exclude<StructuredHarnessProtocol, ''>[]> = {
  opencode: ['acp'],
  pi: ['pi_rpc'],
  omp: ['acp', 'pi_rpc'],
  mimo: ['acp'],
  dsh: ['acp'],
};

export function structuredProtocolsForProvider(
  provider: string,
  installedAndRegistered: boolean,
): Exclude<StructuredHarnessProtocol, ''>[] {
  if (!installedAndRegistered) return [];
  return protocols[provider.toLowerCase()] ?? [];
}

export function isStructuredProtocolAllowed(
  provider: string,
  mode: StructuredHarnessProtocol,
  installedAndRegistered: boolean,
): boolean {
  return mode === '' || structuredProtocolsForProvider(provider, installedAndRegistered).includes(mode);
}

/**
 * Structured sessions must bind to the requested profile. A mere scheduler
 * preference can silently choose a different provider on a quota fallback.
 */
export function bindStructuredHarnessProfile(
  config: AgentConfig,
  provider: string,
  profile: string,
): AgentConfig {
  if (!provider.trim() || !profile.trim()) {
    throw new Error('Structured sessions require an explicit registered provider profile');
  }
  return { ...config, provider, profile, allocation: undefined };
}
