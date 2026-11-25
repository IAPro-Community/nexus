import { Call } from '@wailsio/runtime';
import type {
  NexV2ChatResponse,
  NexV2ChatTurn,
  NexV2EventsPage,
  NexV2HudProvider,
  NexV2HudSnapshotResult,
  NexV2HudState,
  NexV2ModelCatalogWatchSnapshot,
  NexV2PlatformCapabilities,
  NexV2PresenceState,
  NexV2PresetId,
} from './nexV2Types';

const LOCAL_STATE_KEY = 'nexus.nexV2.presenceState.v1';
const DESKTOP_PKG = 'github.com/IAPro-Community/nexus/internal/desktop';

let desktopToken = '';
let cachedCsrfToken = '';
let sessionInitDone = false;

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

async function callController<T>(method: string, ...args: unknown[]): Promise<T> {
  return (await Call.ByName(`${DESKTOP_PKG}.NexV2Controller.${method}`, ...args)) as T;
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

async function ensureSessionAuth(): Promise<void> {
  if (sessionInitDone) return;
  sessionInitDone = true;
  if (isDesktopShell()) {
    try {
      const info = (await Call.ByName(`${DESKTOP_PKG}.App.GetBootstrapInfo`)) as {
        sessionToken?: string;
        csrfToken?: string;
      } | null;
      if (info?.sessionToken) desktopToken = info.sessionToken;
      if (info?.csrfToken) cachedCsrfToken = info.csrfToken;
    } catch {
      // Keep unauthenticated probe path; HUD poll will surface status cleanly.
    }
  }
  try {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (desktopToken) {
      headers.Authorization = `Bearer ${desktopToken}`;
    }
    const res = await fetch('/api/v1/session', { headers, credentials: 'same-origin' });
    if (res.ok) {
      const payload = (await res.json()) as { csrf_token?: string };
      if (payload.csrf_token) {
        cachedCsrfToken = payload.csrf_token;
      }
    }
  } catch {
    // Offline handled by callers.
  }
}

function buildHeaders(includeJson = false): Record<string, string> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (includeJson) headers['Content-Type'] = 'application/json';
  if (desktopToken) headers.Authorization = `Bearer ${desktopToken}`;
  if (cachedCsrfToken) headers['X-CSRF-Token'] = cachedCsrfToken;
  return headers;
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

export async function fetchNexV2Hud(): Promise<NexV2HudSnapshotResult> {
  await ensureSessionAuth();
  try {
    const res = await fetch('/api/v1/hud', {
      headers: buildHeaders(false),
      credentials: 'same-origin',
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

export async function fetchNexV2Events(since = 0): Promise<NexV2EventsPage> {
  await ensureSessionAuth();
  try {
    const query = since > 0 ? `?since=${encodeURIComponent(String(since))}` : '';
    const res = await fetch(`/api/v1/nex/events${query}`, {
      headers: buildHeaders(false),
      credentials: 'same-origin',
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
  await ensureSessionAuth();
  try {
    const res = await fetch('/api/v1/nex/models/watch', {
      headers: buildHeaders(false),
      credentials: 'same-origin',
    });
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
  await ensureSessionAuth();
  try {
    const res = await fetch('/api/v1/capacity/ack', {
      method: 'POST',
      headers: buildHeaders(true),
      credentials: 'same-origin',
      body: JSON.stringify({ fingerprint }),
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function snoozeCapacityAlert(fingerprint: string, minutes = 30): Promise<boolean> {
  await ensureSessionAuth();
  try {
    const until = new Date(Date.now() + minutes * 60_000).toISOString();
    const res = await fetch('/api/v1/capacity/snooze', {
      method: 'POST',
      headers: buildHeaders(true),
      credentials: 'same-origin',
      body: JSON.stringify({ fingerprint, until }),
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
): Promise<NexV2ChatResponse & { offline?: boolean }> {
  await ensureSessionAuth();
  try {
    const res = await fetch('/api/v1/nex/assistant', {
      method: 'POST',
      headers: buildHeaders(true),
      credentials: 'same-origin',
      body: JSON.stringify({
        project_id: projectId ?? '',
        message,
        history: history.map((turn) => ({ role: turn.role, content: turn.content })),
        include_project_context: false,
        persist: false,
      }),
    });
    const body = (await res.json().catch(() => ({}))) as {
      answer?: string;
      detail?: string;
      source?: string;
      proposal?: NexV2ChatResponse['proposal'];
    };
    if (!res.ok) {
      if (res.status === 503 && body.detail) {
        return { answer: body.detail, source: 'unavailable' };
      }
      return { answer: '', offline: true };
    }
    return {
      answer: body.answer ?? '',
      source: body.source,
      proposal: body.proposal,
    };
  } catch {
    return { answer: '', offline: true };
  }
}
