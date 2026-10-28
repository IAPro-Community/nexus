import type { AgentConfig, ProviderInfo } from '../../types';

/** Provider-native transports advertised by the running Core, not authenticated E2E proof. */
export type StructuredHarnessProtocol = '' | 'acp' | 'pi_rpc';

type ReportedProvider = Pick<ProviderInfo, 'installed' | 'structured_modes'>;

/**
 * Core is the sole authority for experimental entrypoints. Older Core builds,
 * unavailable probes and malformed/unknown protocol names must fail closed.
 * This is discovery only: the launcher still validates and authorizes launch.
 */
export function structuredProtocolsForProvider(
  provider: ReportedProvider | null | undefined,
): Exclude<StructuredHarnessProtocol, ''>[] {
  if (!provider?.installed || !Array.isArray(provider.structured_modes)) return [];
  return provider.structured_modes
    .filter((mode): mode is Exclude<StructuredHarnessProtocol, ''> =>
      mode === 'acp' || mode === 'pi_rpc',
    )
    .filter((mode, index, modes) => modes.indexOf(mode) === index);
}

export function isStructuredProtocolAllowed(
  provider: ReportedProvider | null | undefined,
  mode: StructuredHarnessProtocol,
): boolean {
  return mode === '' || structuredProtocolsForProvider(provider).includes(mode);
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
