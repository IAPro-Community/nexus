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
