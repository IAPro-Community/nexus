export type NexAssistantMode = 'manual' | 'contextual' | 'proactive';

export interface NexAssistantPreferences {
  version: 1;
  mode: NexAssistantMode;
  persistHistory: boolean;
}

export const NEX_ASSISTANT_PREFERENCES_KEY = 'iapro:nexus:nex:assistant:v1';

export function readNexAssistantPreferences(): NexAssistantPreferences {
  if (typeof window === 'undefined') return { version: 1, mode: 'manual', persistHistory: false };
  try {
    const raw = JSON.parse(
      window.localStorage.getItem(NEX_ASSISTANT_PREFERENCES_KEY) || '{}',
    ) as Partial<NexAssistantPreferences>;
    return {
      version: 1,
      mode:
        raw.mode === 'proactive'
          ? 'proactive'
          : raw.mode === 'contextual'
            ? 'contextual'
            : 'manual',
      persistHistory: raw.persistHistory === true,
    };
  } catch {
    return { version: 1, mode: 'manual', persistHistory: false };
  }
}

export function saveNexAssistantPreferences(value: NexAssistantPreferences): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(NEX_ASSISTANT_PREFERENCES_KEY, JSON.stringify(value));
  } catch {
    // Local storage can be unavailable in private/embedded WebViews.
    // The in-memory preference remains usable for this session.
  }
}
