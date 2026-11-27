import type {
  AutoConfigureIntelligencePayload,
  NexV2ChatRecovery,
  NexV2ChatResponse,
  NexV2ChatSetupAssistance,
  NexV2ActionReceipt,
  NexV2ChatTurn,
  NexV2EventsPage,
  NexV2HudMemoryMode,
  NexV2HudProvider,
  NexV2HudSnapshotResult,
  NexV2HudState,
  NexV2ModelCatalogWatchSnapshot,
  NexV2NotificationCooldownSec,
  NexV2NotificationLevel,
  NexV2PersonalMemoryEntry,
  NexV2PlatformCapabilities,
  NexV2PresenceState,
  NexV2PresetId,
} from './nexV2Types';

const LOCAL_STATE_KEY = 'nexus.nexV2.presenceState.v1';
const DESKTOP_PKG = 'github.com/IAPro-Community/nexus/internal/desktop';

let desktopToken = '';
let cachedCsrfToken = '';
let sessionInitDone = false;
let sessionInitPromise: Promise<void> | null = null;

export function asSafeArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

export function presetSizePixels(presetId: string): number {
  switch (presetId) {
    case 'compact':
      return 128;
    case 'large':
      return 192;
    default:
      return 160;
  }
}

export function isDesktopShell(): boolean {
  if (typeof window === 'undefined') return false;
  const protocol = window.location?.protocol ?? '';
  const hostname = window.location?.hostname ?? '';
  const ua = typeof navigator !== 'undefined' ? (navigator.userAgent?.toLowerCase() ?? '') : '';
  return (
    protocol === 'wails:' ||
    hostname === 'wails' ||
    hostname === 'wails.localhost' ||
    ua.includes('wails')
  );
}

export function defaultPresenceState(): NexV2PresenceState {
  return {
    placement: {
      displayId: 'current',
      anchor: 'bottom-right',
      offsetX: 16,
      offsetY: 16,
      presetId: 'standard',
      scaleFactor: 1,
      alwaysOnTop: true,
    },
    showAvatar: true,
    compactBar: false,
    doNotDisturb: false,
    privacyMode: false,
    paused: false,
    reducedMotion: false,
    hotkey: 'Ctrl+Shift+N',
    respectFullscreen: true,
    notificationLevel: 'important',
    notificationCooldownSec: 30,
    hudMemoryMode: 'ondemand',
  };
}

export function defaultPlatformCapabilities(): NexV2PlatformCapabilities {
  return {
    globalPositioning: true,
    clickThrough: false,
    globalHotkey: false,
    globalHotkeyReason: 'Use tray or Ctrl+Shift+N inside Nex.',
    secondaryWindow: true,
    tray: true,
    autoStart: true,
    platform: typeof navigator !== 'undefined' ? navigator.platform || 'linux' : 'linux',
  };
}

function readLocalState(): NexV2PresenceState {
  const base = defaultPresenceState();
  if (typeof window === 'undefined' || !window.localStorage) {
    return base;
  }
  try {
    const raw = window.localStorage.getItem(LOCAL_STATE_KEY);
    if (!raw) return base;
    const parsed = JSON.parse(raw) as Partial<NexV2PresenceState>;
    return {
      ...base,
      ...parsed,
      placement: {
        ...base.placement,
        ...(parsed.placement ?? {}),
      },
    };
  } catch {
    return base;
  }
}

function writeLocalState(state: NexV2PresenceState): NexV2PresenceState {
  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      window.localStorage.setItem(LOCAL_STATE_KEY, JSON.stringify(state));
    } catch {
      // Ignore storage quota errors in restricted sandboxes.
    }
  }
  return state;
}

async function invokeDesktop<T>(fullMethod: string, ...args: unknown[]): Promise<T> {
  const { Call } = await import('@wailsio/runtime');
  return (await Call.ByName(fullMethod, ...args)) as T;
}

async function callController<T>(method: string, ...args: unknown[]): Promise<T> {
  return invokeDesktop<T>(`${DESKTOP_PKG}.NexV2Controller.${method}`, ...args);
}

export async function loadNexV2Presence(): Promise<{
  state: NexV2PresenceState;
  capabilities: NexV2PlatformCapabilities;
}> {
  if (isDesktopShell()) {
    try {
      const [state, capabilities] = await Promise.all([
        callController<NexV2PresenceState>('GetPresenceState'),
        callController<NexV2PlatformCapabilities>('GetCapabilities'),
      ]);
      return { state, capabilities };
    } catch {
      // Fall through to deterministic browser fallback if binding unavailable.
    }
  }
  return {
    state: readLocalState(),
    capabilities: defaultPlatformCapabilities(),
  };
}

export async function updateNexV2PanelOpen(open: boolean): Promise<boolean> {
  if (isDesktopShell()) {
    try {
      return await callController<boolean>('SetPanelOpen', open);
    } catch {
      return open;
    }
  }
  return open;
}

export async function updateNexV2SettingsOpen(open: boolean): Promise<boolean> {
  if (isDesktopShell()) {
    try {
      return await callController<boolean>('SetSettingsOpen', open);
    } catch {
      return open;
    }
  }
  return open;
}

export async function updateNexV2OverlayVisible(visible: boolean): Promise<boolean> {
  if (isDesktopShell()) {
    try {
      return await callController<boolean>('SetOverlayVisible', visible);
    } catch {
      return visible;
    }
  }
  return visible;
}

export async function updateNexV2Preset(
  presetId: NexV2PresetId,
  current: NexV2PresenceState,
): Promise<NexV2PresenceState> {
  if (isDesktopShell()) {
    try {
      return await callController<NexV2PresenceState>('SetPreset', presetId);
    } catch {
      // Fallback below
    }
  }
  return writeLocalState({
    ...current,
    placement: { ...current.placement, presetId },
  });
}

async function updateBooleanControllerSetting(
  method: string,
  value: boolean,
  updater: (prev: NexV2PresenceState) => NexV2PresenceState,
  current: NexV2PresenceState,
): Promise<NexV2PresenceState> {
  if (isDesktopShell()) {
    try {
      return await callController<NexV2PresenceState>(method, value);
    } catch {
      // Fallback below
    }
  }
  return writeLocalState(updater(current));
}

export const updateNexV2ShowAvatar = (visible: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting(
    'SetShowAvatar',
    visible,
    (prev) => ({ ...prev, showAvatar: visible }),
    current,
  );

export const updateNexV2CompactBar = (enabled: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting(
    'SetCompactBar',
    enabled,
    (prev) => ({ ...prev, compactBar: enabled }),
    current,
  );

export async function updateNexV2DisplayMode(
  showAvatar: boolean,
  compactBar: boolean,
  current: NexV2PresenceState,
): Promise<NexV2PresenceState> {
  if (isDesktopShell()) {
    try {
      return await callController<NexV2PresenceState>('SetDisplayMode', showAvatar, compactBar);
    } catch {
      // Fallback below
    }
  }
  return writeLocalState({
    ...current,
    showAvatar,
    compactBar,
  });
}

export const updateNexV2AlwaysOnTop = (enabled: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting(
    'SetAlwaysOnTop',
    enabled,
    (prev) => ({ ...prev, placement: { ...prev.placement, alwaysOnTop: enabled } }),
    current,
  );

export const updateNexV2DoNotDisturb = (enabled: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting(
    'SetDoNotDisturb',
    enabled,
    (prev) => ({ ...prev, doNotDisturb: enabled }),
    current,
  );

export const updateNexV2PrivacyMode = (enabled: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting(
    'SetPrivacyMode',
    enabled,
    (prev) => ({ ...prev, privacyMode: enabled }),
    current,
  );

export const updateNexV2Paused = (paused: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting('SetPaused', paused, (prev) => ({ ...prev, paused }), current);

export const updateNexV2ReducedMotion = (enabled: boolean, current: NexV2PresenceState) =>
  updateBooleanControllerSetting(
    'SetReducedMotion',
    enabled,
    (prev) => ({ ...prev, reducedMotion: enabled }),
    current,
  );

export async function updateNexV2NotificationLevel(
  level: NexV2NotificationLevel,
  current: NexV2PresenceState,
): Promise<NexV2PresenceState> {
  if (isDesktopShell()) {
    try {
      const updated = await callController<NexV2PresenceState>('SetNotificationLevel', level);
      return writeLocalState(updated);
    } catch {
      // Fallback to localStorage below.
    }
  }
  return writeLocalState({ ...current, notificationLevel: level });
}

export async function updateNexV2NotificationCooldown(
  seconds: NexV2NotificationCooldownSec,
  current: NexV2PresenceState,
): Promise<NexV2PresenceState> {
  if (isDesktopShell()) {
    try {
      const updated = await callController<NexV2PresenceState>('SetNotificationCooldown', seconds);
      return writeLocalState(updated);
    } catch {
      // Fallback to localStorage below.
    }
  }
  return writeLocalState({ ...current, notificationCooldownSec: seconds });
}

export async function updateNexV2HudMemoryMode(
  mode: NexV2HudMemoryMode,
  current: NexV2PresenceState,
): Promise<NexV2PresenceState> {
  if (isDesktopShell()) {
    try {
      const updated = await callController<NexV2PresenceState>('SetHudMemoryMode', mode);
      return writeLocalState(updated);
    } catch {
      // Fallback to localStorage below.
    }
  }
  return writeLocalState({ ...current, hudMemoryMode: mode });
}

function readBootstrapFragmentToken(): string {
  if (typeof window === 'undefined' || !window.location?.hash) return '';
  const hash = window.location.hash.replace(/^#/, '');
  const params = new URLSearchParams(hash);
  return params.get('nexus_bootstrap')?.trim() ?? '';
}

function clearBootstrapFragment(): void {
  if (typeof window === 'undefined' || !window.location?.hash) return;
  if (!window.location.hash.includes('nexus_bootstrap=')) return;
  try {
    window.history?.replaceState?.(
      null,
      '',
      `${window.location.pathname}${window.location.search}`,
    );
  } catch {
    // Ignore read-only location errors.
  }
}

async function performSessionAuth(): Promise<void> {
  if (isDesktopShell()) {
    try {
      const info = await invokeDesktop<{
        sessionToken?: string;
        csrfToken?: string;
      } | null>(`${DESKTOP_PKG}.App.GetBootstrapInfo`);
      if (info?.sessionToken) desktopToken = info.sessionToken;
      if (info?.csrfToken) cachedCsrfToken = info.csrfToken;
    } catch {
      // Keep unauthenticated probe path; HUD poll will surface status cleanly.
    }
  } else {
    const fragmentToken = readBootstrapFragmentToken();
    if (fragmentToken) {
      try {
        const bootRes = await fetch('/api/v1/auth/bootstrap', {
          method: 'POST',
          headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
          credentials: 'same-origin',
          body: JSON.stringify({ token: fragmentToken }),
        });
        if (bootRes.ok) {
          const bootPayload = (await bootRes.json()) as {
            authenticated?: boolean;
            csrf_token?: string;
          };
          if (bootPayload.csrf_token) {
            cachedCsrfToken = bootPayload.csrf_token;
          }
          clearBootstrapFragment();
          if (bootPayload.authenticated) {
            return;
          }
        }
      } catch {
        // Fall through to normal session probe.
      }
    }
  }

  try {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (desktopToken) {
      headers.Authorization = `Bearer ${desktopToken}`;
    }
    const res = await fetch('/api/v1/session', { headers, credentials: 'same-origin' });
    if (res.ok) {
      const payload = (await res.json()) as {
        authenticated?: boolean;
        csrf_token?: string;
      };
      if (payload.csrf_token) {
        cachedCsrfToken = payload.csrf_token;
      }
      if (payload.authenticated) {
        return;
      }
    }
    if (isDesktopShell()) {
      try {
        const refreshed = await invokeDesktop<{
          sessionToken?: string;
          csrfToken?: string;
        } | null>(`${DESKTOP_PKG}.App.RefreshBootstrapFromCore`);
        if (refreshed?.sessionToken) desktopToken = refreshed.sessionToken;
        if (refreshed?.csrfToken) cachedCsrfToken = refreshed.csrfToken;
      } catch {
        // Offline handled by callers.
      }
    } else {
      const reauthRes = await fetch('/api/v1/auth/loopback-reauth', {
        method: 'POST',
        headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
        credentials: 'same-origin',
      });
      if (reauthRes.ok) {
        const reauthPayload = (await reauthRes.json()) as { csrf_token?: string };
        if (reauthPayload.csrf_token) {
          cachedCsrfToken = reauthPayload.csrf_token;
        }
      }
    }
  } catch {
    // Offline handled by callers.
  }
}

async function ensureSessionAuth(forceRefresh = false): Promise<void> {
  if (forceRefresh) {
    sessionInitDone = false;
    sessionInitPromise = null;
  }
  if (sessionInitDone) return;
  if (!sessionInitPromise) {
    sessionInitPromise = performSessionAuth().finally(() => {
      sessionInitDone = true;
      sessionInitPromise = null;
    });
  }
  await sessionInitPromise;
}

function buildHeaders(
  includeJson = false,
  extraHeaders?: Record<string, string>,
): Record<string, string> {
  const headers: Record<string, string> = { Accept: 'application/json', ...extraHeaders };
  if (includeJson) headers['Content-Type'] = 'application/json';
  if (desktopToken) headers.Authorization = `Bearer ${desktopToken}`;
  if (cachedCsrfToken) headers['X-CSRF-Token'] = cachedCsrfToken;
  return headers;
}

async function authedFetch(
  input: string,
  options: {
    method?: string;
    includeJson?: boolean;
    body?: string;
    signal?: AbortSignal;
    extraHeaders?: Record<string, string>;
    keepalive?: boolean;
  } = {},
): Promise<Response> {
  await ensureSessionAuth();
  let res = await fetch(input, {
    method: options.method ?? 'GET',
    headers: buildHeaders(options.includeJson ?? false, options.extraHeaders),
    credentials: 'same-origin',
    signal: options.signal,
    body: options.body,
    keepalive: options.keepalive,
  });
  if (res.status === 401 || res.status === 403) {
    await ensureSessionAuth(true);
    res = await fetch(input, {
      method: options.method ?? 'GET',
      headers: buildHeaders(options.includeJson ?? false, options.extraHeaders),
      credentials: 'same-origin',
      signal: options.signal,
      body: options.body,
      keepalive: options.keepalive,
    });
  }
  return res;
}

type NexV2HubPresenceInput = Pick<
  NexV2PresenceState,
  'paused' | 'showAvatar' | 'compactBar' | 'doNotDisturb'
> &
  Partial<Pick<NexV2PresenceState, 'notificationLevel' | 'hudMemoryMode'>>;

export function buildNexHubHeaders(presence?: NexV2HubPresenceInput): Record<string, string> {
  if (!presence) return {};
  // Keep NEX active across quiet/critical/important/on-demand modes so OS native popups stay suppressed.
  const hub = presence.paused ? 'paused' : 'active';
  const dnd = presence.doNotDisturb || presence.notificationLevel === 'quiet';
  return {
    'X-Nexus-Nex-Hub': hub,
    'X-Nexus-Nex-Dnd': dnd ? '1' : '0',
  };
}

export async function reportNexHubStatus(
  presence: NexV2HubPresenceInput,
  offline: boolean,
  keepalive = false,
): Promise<void> {
  const canNotify = !offline && !presence.paused;
  const effectiveDnd = presence.doNotDisturb || presence.notificationLevel === 'quiet';
  const reason = offline
    ? 'nex_offline'
    : presence.paused
      ? 'nex_paused'
      : effectiveDnd
        ? 'nex_dnd'
        : 'nex_active';
  try {
    await authedFetch('/api/v1/nex/hub-status', {
      method: 'POST',
      includeJson: true,
      keepalive,
      body: JSON.stringify({
        can_notify: canNotify,
        do_not_disturb: effectiveDnd,
        reason,
      }),
    });
  } catch {
    // Offline or closing windows rely on backend TTL lease expiration as fail-safe.
  }
}

export async function ensureNexCoreConnection(): Promise<{
  ok: boolean;
  error?: string;
}> {
  if (isDesktopShell()) {
    try {
      const info = await invokeDesktop<{
        sessionToken?: string;
        csrfToken?: string;
      } | null>(`${DESKTOP_PKG}.App.EnsureLocalCoreRunning`);
      if (info?.sessionToken) desktopToken = info.sessionToken;
      if (info?.csrfToken) cachedCsrfToken = info.csrfToken;
    } catch (err) {
      return {
        ok: false,
        error: err instanceof Error ? err.message : undefined,
      };
    }
  }
  await ensureSessionAuth(true);
  try {
    const res = await fetch('/api/v1/session', {
      headers: buildHeaders(false),
      credentials: 'same-origin',
    });
    if (!res.ok) {
      return { ok: false, error: `HTTP ${res.status}` };
    }
    const payload = (await res.json()) as {
      authenticated?: boolean;
      csrf_token?: string;
    };
    if (payload.csrf_token) {
      cachedCsrfToken = payload.csrf_token;
    }
    return payload.authenticated ? { ok: true } : { ok: false, error: 'authentication required' };
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : undefined,
    };
  }
}

function normalizeProvider(p: NexV2HudProvider): NexV2HudProvider {
  return {
    ...p,
    windows: asSafeArray(p.windows),
  };
}

export function normalizeHudState(
  raw: Partial<NexV2HudState> | null | undefined,
): NexV2HudState | null {
  if (!raw) return null;
  return {
    generated_at: raw.generated_at ?? '',
    project_id: raw.project_id,
    providers: asSafeArray(raw.providers).map(normalizeProvider),
    agents: asSafeArray(raw.agents),
    missions: asSafeArray(raw.missions),
    alerts: asSafeArray(raw.alerts),
    runtime_attention: asSafeArray(raw.runtime_attention),
    cost: raw.cost,
    primary: raw.primary ?? { kind: 'EMPTY' },
    system: {
      capacity: raw.system?.capacity || 'UNKNOWN',
      resources: raw.system?.resources || 'UNKNOWN',
      agents: raw.system?.agents || 'UNKNOWN',
      missions: raw.system?.missions || 'UNKNOWN',
      cost: raw.system?.cost || 'UNKNOWN',
      cache_ttl_millis: raw.system?.cache_ttl_millis || 15_000,
    },
  };
}

export async function fetchNexV2Hud(
  presence?: NexV2HubPresenceInput,
): Promise<NexV2HudSnapshotResult> {
  try {
    const res = await authedFetch('/api/v1/hud', {
      extraHeaders: buildNexHubHeaders(presence),
    });
    if (res.status >= 500) {
      return { offline: true, enabled: false, state: null, error: `HTTP ${res.status}` };
    }
    if (!res.ok) {
      return { offline: false, enabled: false, state: null, error: `HTTP ${res.status}` };
    }
    const payload = (await res.json()) as { enabled?: boolean; state?: Partial<NexV2HudState> };
    return {
      offline: false,
      enabled: Boolean(payload.enabled),
      state: normalizeHudState(payload.state),
    };
  } catch (err) {
    return {
      offline: true,
      enabled: false,
      state: null,
      error: err instanceof Error ? err.message : 'network error',
    };
  }
}

export function computeReconnectBackoffMs(consecutiveFailures: number): number {
  if (consecutiveFailures <= 0 || !Number.isFinite(consecutiveFailures)) {
    return 0;
  }
  const shift = Math.min(Math.floor(consecutiveFailures) - 1, 3);
  return Math.min(2000 * 2 ** shift, 15_000);
}

export async function fetchNexV2Events(
  since = 0,
  presence?: NexV2HubPresenceInput,
): Promise<NexV2EventsPage> {
  try {
    const query = since > 0 ? `?since=${encodeURIComponent(String(since))}` : '';
    const res = await authedFetch(`/api/v1/nex/events${query}`, {
      extraHeaders: buildNexHubHeaders(presence),
    });
    if (res.status >= 500 || !res.ok) {
      return {
        offline: res.status >= 500,
        revision: since,
        replay_truncated: false,
        events: [],
        model_watch_freshness: 'UNKNOWN',
      };
    }
    const payload = (await res.json()) as Partial<NexV2EventsPage>;
    return {
      offline: false,
      revision: typeof payload.revision === 'number' ? payload.revision : since,
      replay_truncated: Boolean(payload.replay_truncated),
      events: asSafeArray(payload.events),
      model_watch_freshness: payload.model_watch_freshness || 'UNKNOWN',
    };
  } catch {
    return {
      offline: true,
      revision: since,
      replay_truncated: false,
      events: [],
      model_watch_freshness: 'UNKNOWN',
    };
  }
}

export async function fetchNexV2ModelWatch(): Promise<NexV2ModelCatalogWatchSnapshot> {
  try {
    const res = await authedFetch('/api/v1/nex/models/watch');
    if (!res.ok) {
      return { freshness: 'UNKNOWN', entries: [], changes: [] };
    }
    const payload = (await res.json()) as Partial<NexV2ModelCatalogWatchSnapshot>;
    return {
      freshness: payload.freshness || 'UNKNOWN',
      observed_at: payload.observed_at,
      entries: asSafeArray(payload.entries),
      changes: asSafeArray(payload.changes),
    };
  } catch {
    return { freshness: 'UNKNOWN', entries: [], changes: [] };
  }
}

export async function ackCapacityAlert(fingerprint: string): Promise<boolean> {
  try {
    const res = await authedFetch('/api/v1/capacity/ack', {
      method: 'POST',
      includeJson: true,
      body: JSON.stringify({ fingerprint }),
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function snoozeCapacityAlert(fingerprint: string, minutes = 30): Promise<boolean> {
  try {
    const until = new Date(Date.now() + minutes * 60_000).toISOString();
    const res = await authedFetch('/api/v1/capacity/snooze', {
      method: 'POST',
      includeJson: true,
      body: JSON.stringify({ fingerprint, until }),
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function loadNexV2Conversation(projectId = ''): Promise<NexV2ChatTurn[]> {
  try {
    const q = projectId ? `?project_id=${encodeURIComponent(projectId)}` : '';
    const res = await authedFetch(`/api/v1/nex/conversation${q}`);
    if (!res.ok) return [];
    const payload = (await res.json()) as { turns?: NexV2ChatTurn[] };
    return asSafeArray(payload.turns);
  } catch {
    return [];
  }
}

export async function clearNexV2Conversation(projectId = ''): Promise<boolean> {
  try {
    const q = projectId ? `?project_id=${encodeURIComponent(projectId)}` : '';
    const res = await authedFetch(`/api/v1/nex/conversation${q}`, {
      method: 'DELETE',
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function loadNexV2Memories(): Promise<NexV2PersonalMemoryEntry[]> {
  try {
    const res = await authedFetch('/api/v1/nex/memory');
    if (!res.ok) return [];
    const payload = (await res.json()) as { entries?: NexV2PersonalMemoryEntry[] };
    return asSafeArray(payload.entries);
  } catch {
    return [];
  }
}

export async function saveNexV2Memory(
  key: string,
  value: string,
): Promise<NexV2PersonalMemoryEntry | null> {
  try {
    const res = await authedFetch('/api/v1/nex/memory', {
      method: 'POST',
      includeJson: true,
      body: JSON.stringify({ key, value, provenance: 'user_explicit' }),
    });
    if (!res.ok) return null;
    const payload = (await res.json()) as { entry?: NexV2PersonalMemoryEntry };
    return payload.entry ?? null;
  } catch {
    return null;
  }
}

export async function deleteNexV2Memory(id: string): Promise<boolean> {
  try {
    const res = await authedFetch(`/api/v1/nex/memory?id=${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function sendNexV2Chat(
  message: string,
  history: NexV2ChatTurn[],
  projectId?: string,
  options: { persist?: boolean; requestId?: string; signal?: AbortSignal } = {},
): Promise<NexV2ChatResponse & { offline?: boolean; aborted?: boolean }> {
  try {
    const persist = options.persist ?? false;
    const requestId =
      options.requestId || `nex-v2-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
    const res = await authedFetch('/api/v1/nex/assistant', {
      method: 'POST',
      includeJson: true,
      signal: options.signal,
      body: JSON.stringify({
        project_id: projectId ?? '',
        message,
        history: history.map((turn) => ({ role: turn.role, content: turn.content })),
        include_project_context: false,
        persist,
        request_id: persist ? requestId : undefined,
      }),
    });
    const body = (await res.json().catch(() => ({}))) as {
      answer?: string;
      error?: string;
      detail?: string;
      source?: string;
      proposal?: NexV2ChatResponse['proposal'];
      setup_assistance?: NexV2ChatSetupAssistance;
    };
    if (!res.ok) {
      if ((res.status === 503 || res.status === 402) && (body.detail || body.setup_assistance)) {
        return {
          answer: body.detail ?? '',
          source: 'unavailable',
          setup_assistance: body.setup_assistance,
        };
      }
      const kind: NexV2ChatRecovery['kind'] =
        res.status === 401 || res.status === 403 ? 'auth_required' : 'core_error';
      return {
        answer: body.detail || body.error || '',
        offline: true,
        recovery: {
          kind,
          status: res.status,
          detail: body.detail || body.error,
          command: kind === 'auth_required' ? 'nexus web open' : 'nexus web',
          canStartCore: isDesktopShell(),
        },
      };
    }
    return {
      answer: body.answer ?? '',
      source: body.source,
      proposal: body.proposal,
    };
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      return { answer: '', aborted: true };
    }
    return {
      answer: '',
      offline: true,
      recovery: {
        kind: 'core_offline',
        command: 'nexus web',
        canStartCore: isDesktopShell(),
      },
    };
  }
}

export async function autoConfigureNexIntelligence(
  payload: AutoConfigureIntelligencePayload,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await authedFetch('/api/v1/nex/intelligence/auto-configure', {
      method: 'POST',
      includeJson: true,
      body: JSON.stringify(payload),
    });
    if (res.ok) {
      return { ok: true };
    }
    const body = (await res.json().catch(() => ({}))) as { detail?: string; error?: string };
    return { ok: false, error: body.detail || body.error || `HTTP ${res.status}` };
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : 'network error' };
  }
}

export async function loadNexV2Receipts(): Promise<NexV2ActionReceipt[]> {
  try {
    const res = await authedFetch('/api/v1/nex/actions');
    if (!res.ok) return [];
    const payload = (await res.json()) as { receipts?: NexV2ActionReceipt[] };
    return asSafeArray(payload.receipts);
  } catch {
    return [];
  }
}

// Runs one allowlisted action through propose -> explicit confirm. The server
// executes through the canonical owner and returns the audited receipt.
export async function runNexV2Action(
  actionId: string,
  params: Record<string, string> = {},
  approved = true,
): Promise<NexV2ActionReceipt | null> {
  try {
    const proposeRes = await authedFetch('/api/v1/nex/actions/propose', {
      method: 'POST',
      includeJson: true,
      body: JSON.stringify({ action_id: actionId, params }),
    });
    if (!proposeRes.ok) return null;
    const proposed = (await proposeRes.json()) as { receipt?: NexV2ActionReceipt };
    if (!proposed.receipt) return null;
    const confirmRes = await authedFetch('/api/v1/nex/actions/confirm', {
      method: 'POST',
      includeJson: true,
      body: JSON.stringify({ receipt_id: proposed.receipt.receipt_id, approved }),
    });
    if (!confirmRes.ok) return proposed.receipt;
    const done = (await confirmRes.json()) as { receipt?: NexV2ActionReceipt };
    return done.receipt ?? proposed.receipt;
  } catch {
    return null;
  }
}
