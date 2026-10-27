const DEFAULT_PROFILE_NAMES = new Set(['default']);

/**
 * Resolves the profile name that the CLI needs. The default profile (the one
 * named after the provider, or literally "default") is already implied by the
 * provider argument, so it is omitted to avoid commands that repeat themselves
 * (`nexus login codex codex`).
 */
export function visibleProfileName(provider: string, profile?: string): string {
  const providerName = (provider || '').trim();
  const profileName = (profile || '').trim();
  if (!profileName) return '';
  if (profileName.toLowerCase() === providerName.toLowerCase()) return '';
  if (DEFAULT_PROFILE_NAMES.has(profileName.toLowerCase())) return '';
  return profileName;
}

/** Terminal command for an official provider login. */
export function loginCommand(provider: string, profile?: string): string {
  const providerName = (provider || '').trim();
  const profileName = visibleProfileName(providerName, profile);
  return profileName ? `nexus login ${providerName} ${profileName}` : `nexus login ${providerName}`;
}

/**
 * Human label for a provider/profile pair. Avoids `codex (codex)` in titles and
 * toasts while keeping the profile visible when it differs from the provider.
 */
export function providerProfileLabel(provider: string, profile?: string): string {
  const providerName = (provider || '').trim();
  const profileName = visibleProfileName(providerName, profile);
  return profileName ? `${providerName} (${profileName})` : providerName;
}
