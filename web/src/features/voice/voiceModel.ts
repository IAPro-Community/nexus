export type VoiceProviderKind = 'groq' | 'cloudflare';

export type VoiceProviderUsage = {
  provider_id?: string;
  requests: number;
  successes: number;
  failures: number;
  observed_audio_ms: number;
  last_status?: string;
  last_used_at?: string;
};

export type VoiceProviderView = {
  id: string;
  kind: VoiceProviderKind;
  label: string;
  enabled: boolean;
  priority: number;
  model: string;
  account_id?: string;
  account_id_env?: string;
  base_url?: string;
  api_key_env?: string;
  api_key_file?: string;
  free_eligible: boolean;
  has_api_key: boolean;
  configured: boolean;
  usage: VoiceProviderUsage;
};

export type VoiceSettingsView = {
  version: number;
  enabled: boolean;
  language: string;
  terminal_enabled: boolean;
  nexus_control_enabled: boolean;
  automatic_fallback: boolean;
  max_capture_seconds: number;
  providers: VoiceProviderView[];
};

export type VoiceProviderUpdate = Omit<
  VoiceProviderView,
  'has_api_key' | 'configured' | 'usage'
> & {
  api_key?: string;
  clear_api_key?: boolean;
};

export type VoiceSettingsUpdate = Omit<VoiceSettingsView, 'version' | 'providers'> & {
  providers: VoiceProviderUpdate[];
};

export type VoiceProviderAttempt = {
  provider_id: string;
  provider: string;
  status: string;
  http_status?: number;
  message?: string;
};

export type VoiceTranscribeResult = {
  text: string;
  provider_id: string;
  provider: string;
  model: string;
  attempts: VoiceProviderAttempt[];
  duration_ms: number;
};

export const observedVoiceMinutes = (usage?: VoiceProviderUsage) =>
  Math.round(((usage?.observed_audio_ms || 0) / 60_000) * 10) / 10;

export const voiceProviderStatus = (provider: VoiceProviderView) => {
  if (!provider.enabled) return 'Desativado';
  if (!provider.has_api_key) return 'Chave não configurada';
  if (provider.kind === 'cloudflare' && !provider.configured) return 'Account ID ausente';
  if (provider.usage?.last_status) return provider.usage.last_status;
  return 'Pronto';
};
