export type NexAssistantMode = 'manual' | 'contextual';

export interface NexAssistantPreferences {
  version: 1;
  mode: NexAssistantMode;
}

export const NEX_ASSISTANT_PREFERENCES_KEY = 'iapro:nexus:nex:assistant:v1';

export function readNexAssistantPreferences(): NexAssistantPreferences {
  if (typeof window === 'undefined') return { version: 1, mode: 'manual' };
  try {
    const raw = JSON.parse(
      window.localStorage.getItem(NEX_ASSISTANT_PREFERENCES_KEY) || '{}',
    ) as Partial<NexAssistantPreferences>;
    return { version: 1, mode: raw.mode === 'contextual' ? 'contextual' : 'manual' };
  } catch {
    return { version: 1, mode: 'manual' };
  }
}

export function saveNexAssistantPreferences(value: NexAssistantPreferences): void {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(NEX_ASSISTANT_PREFERENCES_KEY, JSON.stringify(value));
}
