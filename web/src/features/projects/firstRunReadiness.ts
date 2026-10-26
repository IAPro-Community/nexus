import type { ProviderInfo } from '../../types';

// Installation detection is a hint, NOT proof that a provider is authenticated
// or that any quota/billing is available. Registration is always opt-in.
export type FirstRunDetectionState = 'checking' | 'loaded' | 'unavailable';

export type FirstRunReadiness = {
  status:
    | 'checking'
    | 'ready'
    | 'needs-account'
    | 'needs-login'
    | 'verification-unknown'
    | 'needs-provider'
    | 'unavailable';
  titleKey: string;
  titleParams?: { count: number };
  descriptionKey: string;
  suggestedProvider?: string;
  canImportSessions: boolean;
};

type DetectedProvider = Pick<ProviderInfo, 'id' | 'installed' | 'registration_state'>;

export const deriveFirstRunReadiness = (
  providers: DetectedProvider[],
  detectionState: FirstRunDetectionState,
): FirstRunReadiness => {
  if (detectionState === 'checking') {
    return {
      status: 'checking',
      titleKey: 'projects.readinessCheckingTitle',
      descriptionKey: 'projects.readinessCheckingDescription',
      canImportSessions: false,
    };
  }
  if (detectionState === 'unavailable') {
    return {
      status: 'unavailable',
      titleKey: 'projects.readinessUnavailableTitle',
      descriptionKey: 'projects.readinessUnavailableDescription',
      canImportSessions: false,
    };
  }

  const installed = providers.filter((provider) => provider.installed);
  const authenticated = installed.filter(
    (provider) => provider.registration_state === 'REGISTERED_AUTHENTICATED',
  );
  if (authenticated.length > 0) {
    return {
      status: 'ready',
      titleKey:
        authenticated.length === 1
          ? 'projects.readinessAuthenticatedOne'
          : 'projects.readinessAuthenticatedMany',
      titleParams: { count: authenticated.length },
      descriptionKey:
        authenticated.length < installed.length
          ? 'projects.readinessAuthenticatedImportMore'
          : 'projects.readinessAuthenticatedDescription',
      canImportSessions: authenticated.length < installed.length,
    };
  }
  const pending = installed.find(
    (provider) =>
      provider.registration_state === 'PENDING_AUTH' ||
      provider.registration_state === 'REGISTERED_AUTH_EXPIRED',
  );
  if (pending) {
    return {
      status: 'needs-login',
      titleKey: 'projects.readinessNeedsLoginTitle',
      descriptionKey: 'projects.readinessNeedsLoginDescription',
      suggestedProvider: pending.id,
      canImportSessions: authenticated.length < installed.length,
    };
  }
  const unverified = installed.find(
    (provider) => provider.registration_state === 'REGISTRATION_UNKNOWN',
  );
  if (unverified) {
    return {
      status: 'verification-unknown',
      titleKey: 'projects.readinessVerificationUnknownTitle',
      descriptionKey: 'projects.readinessVerificationUnknownDescription',
      suggestedProvider: unverified.id,
      canImportSessions: true,
    };
  }
  if (installed.length > 0) {
    return {
      status: 'needs-account',
      titleKey:
        installed.length === 1
          ? 'projects.readinessInstalledOne'
          : 'projects.readinessInstalledMany',
      titleParams: { count: installed.length },
      descriptionKey: 'projects.readinessNeedsAccountDescription',
      suggestedProvider: installed[0].id,
      canImportSessions: true,
    };
  }
  return {
    status: 'needs-provider',
    titleKey: 'projects.readinessNoProviderTitle',
    descriptionKey: 'projects.readinessNoProviderDescription',
    canImportSessions: false,
  };
};

export const OFFICIAL_PROVIDER_INSTALL_GUIDES = [
  { label: 'Codex', href: 'https://github.com/openai/codex' },
  { label: 'Claude Code', href: 'https://code.claude.com/docs/en/setup' },
  { label: 'OpenCode', href: 'https://opencode.ai/docs/' },
] as const;
