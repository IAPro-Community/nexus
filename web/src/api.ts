import {
  Workspace,
  RuntimeSession,
  ProviderInfo,
  ProfileInfo,
  EventRecord,
  EffectiveCapabilities,
} from './types';
import { shouldAttachWebSocketQueryToken } from './lib/networkHost';
import { isDesktopApp, getPlatformBridge } from './platform';

let csrfToken = '';
let csrfRefreshInFlight: Promise<boolean> | null = null;
let desktopAuthToken = '';
// Ordinary Wails HTTP requests stay relative to the document origin. Only
// WebSocket upgrades need the real loopback Core listener.
let desktopWebSocketBaseUrl = '';

export type BrowserSession = {
  authenticated: boolean;
  csrf_token?: string;
  expires_at?: string;
  idle_timeout?: number;
};

export type APIErrorPayload = {
  error?: string;
  code?: string;
  [key: string]: unknown;
};

export class NexusRequestError<TPayload extends APIErrorPayload = APIErrorPayload> extends Error {
  readonly status: number;
  readonly code: string | undefined;
  readonly payload: TPayload;

  constructor(status: number, payload: TPayload, message: string) {
    super(message);
    this.name = 'NexusRequestError';
    this.status = status;
    this.code = payload.code;
    this.payload = payload;
  }
}

export function setDesktopAuth(token: string, baseUrl = '') {
  desktopAuthToken = token;
  desktopWebSocketBaseUrl = baseUrl.replace(/\/+$/, '');
}

export function clearDesktopAuth() {
  desktopAuthToken = '';
  desktopWebSocketBaseUrl = '';
}

export function setCSRFToken(token: string) {
  csrfToken = token;
}

export function getCSRFToken(): string {
  return csrfToken;
}

export function getDesktopAuthToken(): string {
  return desktopAuthToken;
}

// Compatibility accessor for HTTP callers. Wails HTTP must remain relative.
export function getDesktopBaseUrl(): string {
  return '';
}

export function getDesktopWebSocketBaseUrl(): string {
  return desktopWebSocketBaseUrl;
}

export function getWebSocketEndpoint(path: string): string {
  const token = getDesktopAuthToken();
  const baseUrl = getDesktopWebSocketBaseUrl();
  let proto =
    typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  let host = typeof window !== 'undefined' ? window.location.host : 'localhost';

  if (baseUrl) {
    try {
      const parsed = new URL(baseUrl);
      proto = parsed.protocol === 'https:' ? 'wss:' : 'ws:';
      host = parsed.host;
    } catch {
      // ignore
    }
  }

  const cleanPath = path.startsWith('/') ? path : `/${path}`;
  const url = `${proto}//${host}${cleanPath}`;
  if (!token) return url;

  // Loopback only: private --remote and Quick Tunnel use HttpOnly cookie so the
  // session ID is not retained in proxy/access logs or LAN sniffers.
  if (!shouldAttachWebSocketQueryToken(host)) {
    return url;
  }

  const delimiter = url.includes('?') ? '&' : '?';
  return `${url}${delimiter}token=${encodeURIComponent(token)}`;
}

function consumeBrowserBootstrapToken(): string {
  if (typeof window === 'undefined' || !window.location.hash) return '';
  const params = new URLSearchParams(window.location.hash.slice(1));
  const token = params.get('nexus_bootstrap') || '';
  if (!token) return '';

  // Remove the token before navigation or subsequent requests can retain it.
  window.history.replaceState(
    null,
    typeof document === 'undefined' ? '' : document.title,
    `${window.location.pathname}${window.location.search}`,
  );
  return token;
}

function notifySessionExpired() {
  clearDesktopAuth();
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('nexus:session-expired'));
  }
}

export async function initSession(): Promise<BrowserSession> {
  try {
    const bootstrapToken = consumeBrowserBootstrapToken();
    if (bootstrapToken && !isDesktopApp()) {
      const bootstrapResponse = await fetch('/api/v1/auth/bootstrap', {
        method: 'POST',
        headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: bootstrapToken }),
      });
      if (bootstrapResponse.ok) {
        const bootstrapData = (await bootstrapResponse.json()) as BrowserSession;
        if (bootstrapData.csrf_token) {
          csrfToken = bootstrapData.csrf_token;
        }
        if (bootstrapData.authenticated) {
          return bootstrapData;
        }
      }
    }
    if (isDesktopApp()) {
      try {
        const bridge = getPlatformBridge();
        if (bridge.getBootstrapInfo) {
          const bootstrap = await bridge.getBootstrapInfo();
          if (bootstrap && bootstrap.sessionToken) {
            // Wails serves the Core handler under the same origin as the webview.
            // Keep API calls relative; using the loopback URL here turns every
            // request into a cross-origin fetch from wails://wails and fails
            // without CORS headers before auth can even be evaluated.
            setDesktopAuth(bootstrap.sessionToken, bootstrap.serverUrl);
            if (bootstrap.csrfToken) {
              csrfToken = bootstrap.csrfToken;
            }
            const targetUrl = '/api/v1/session';
            try {
              const res = await fetch(targetUrl, {
                headers: {
                  Accept: 'application/json',
                  Authorization: `Bearer ${bootstrap.sessionToken}`,
                },
              });
              if (res.ok) {
                const data = await res.json();
                if (data.csrf_token) csrfToken = data.csrf_token;
                return {
                  authenticated: true,
                  csrf_token: csrfToken || bootstrap.csrfToken,
                  expires_at: data.expires_at,
                  idle_timeout: data.idle_timeout,
                };
              }
            } catch (verifyErr) {
              console.warn(
                'Desktop session verify probe error, using pre-authenticated bootstrap token:',
                verifyErr,
              );
            }
            return {
              authenticated: true,
              csrf_token: bootstrap.csrfToken,
            };
          }
        }
      } catch (bridgeErr) {
        console.warn('Desktop session bootstrap initialization failed:', bridgeErr);
      }
    }

    const headers: Record<string, string> = { Accept: 'application/json' };
    if (desktopAuthToken) {
      headers['Authorization'] = `Bearer ${desktopAuthToken}`;
    }
    const targetUrl = '/api/v1/session';
    const res = await fetch(targetUrl, { headers });
    if (res.ok) {
      const data = (await res.json()) as BrowserSession;
      if (data.csrf_token) {
        csrfToken = data.csrf_token;
      }
      if (data.authenticated) {
        if (!desktopAuthToken && isDesktopApp()) {
          try {
            const b = await getPlatformBridge().getBootstrapInfo?.();
            if (b && b.sessionToken) {
              setDesktopAuth(b.sessionToken, b.serverUrl);
            }
          } catch {
            /* ignore */
          }
        }
        return data;
      }
    }

    // Chromium --app= often reuses the window and drops #nexus_bootstrap=.
    // On loopback, reclaim a session without requiring the fragment again.
    if (!isDesktopApp()) {
      const reauth = await tryLoopbackReauth();
      if (reauth?.authenticated) {
        return reauth;
      }
    }
    return { authenticated: false };
  } catch {
    return { authenticated: false };
  }
}

/** Trusted localhost recovery when the bootstrap fragment was lost. */
export async function tryLoopbackReauth(): Promise<BrowserSession | null> {
  try {
    const res = await fetch('/api/v1/auth/loopback-reauth', {
      method: 'POST',
      headers: { Accept: 'application/json' },
    });
    if (!res.ok) return null;
    const data = (await res.json()) as BrowserSession;
    if (data.csrf_token) csrfToken = data.csrf_token;
    return data.authenticated ? data : null;
  } catch {
    return null;
  }
}

export async function rotateSession(): Promise<BrowserSession> {
  const headers: Record<string, string> = {};
  if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
  if (desktopAuthToken) headers['Authorization'] = `Bearer ${desktopAuthToken}`;
  const targetUrl = '/api/v1/session/rotate';
  const res = await fetch(targetUrl, {
    method: 'POST',
    headers,
  });
  if (!res.ok) {
    notifySessionExpired();
    return { authenticated: false };
  }
  const data = (await res.json()) as BrowserSession;
  if (data.csrf_token) csrfToken = data.csrf_token;
  return data;
}

async function refreshCSRFToken(): Promise<boolean> {
  if (!csrfRefreshInFlight) {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (desktopAuthToken) headers.Authorization = `Bearer ${desktopAuthToken}`;
    const targetUrl = '/api/v1/session';
    csrfRefreshInFlight = fetch(targetUrl, { headers })
      .then(async (res) => {
        if (!res.ok) return false;
        const data = (await res.json()) as BrowserSession;
        if (!data.authenticated || !data.csrf_token) return false;
        csrfToken = data.csrf_token;
        return true;
      })
      .catch(() => false)
      .finally(() => {
        csrfRefreshInFlight = null;
      });
  }
  return csrfRefreshInFlight;
}

export async function request<T>(
  path: string,
  options: RequestInit = {},
  retried = false,
): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');

  if (desktopAuthToken) {
    headers.set('Authorization', `Bearer ${desktopAuthToken}`);
  }

  const mutating = Boolean(options.method && options.method !== 'GET' && options.method !== 'HEAD');
  if (mutating) {
    const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData;
    if (!isFormData) {
      headers.set('Content-Type', 'application/json');
    }
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken);
    }
  }

  const res = await fetch(path, { ...options, headers });
  if (res.status === 403 && mutating && !retried && (await refreshCSRFToken())) {
    return request<T>(path, options, true);
  }
  if (!res.ok) {
    if (res.status === 401) notifySessionExpired();
    const errBody = (await res.json().catch(() => ({ error: res.statusText }))) as APIErrorPayload;
    throw new NexusRequestError(res.status, errBody, errBody.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export const api = {
  getWorkspaces: () => request<Workspace[]>('/api/v1/workspaces'),
  addWorkspace: (path: string, name?: string) =>
    request<Workspace>('/api/v1/workspaces', {
      method: 'POST',
      body: JSON.stringify({ path, name }),
    }),
  removeWorkspace: (pathOrId: string) =>
    request<{ status: string }>(`/api/v1/workspaces?path=${encodeURIComponent(pathOrId)}`, {
      method: 'DELETE',
    }),
  getRuntimes: () => request<RuntimeSession[]>('/api/v1/runtimes'),
  getRuntime: (id: string) =>
    request<{ session: RuntimeSession; capabilities: EffectiveCapabilities | null }>(
      `/api/v1/runtimes/${id}`,
    ),
  startRuntime: (
    provider: string,
    profile: string,
    workspace: string,
    args: string[] = [],
    title?: string,
  ) =>
    request<RuntimeSession>('/api/v1/runtimes', {
      method: 'POST',
      body: JSON.stringify({ provider, profile, workspace, args, title }),
    }),
  updateRuntimeTitle: (id: string, title: string) =>
    request<{ status: string; title: string }>(`/api/v1/runtimes/${id}/title`, {
      method: 'POST',
      body: JSON.stringify({ title }),
    }),
  respondRuntime: (id: string, input: string) =>
    request<{ status: string; sent: string }>(`/api/v1/runtimes/${id}/respond`, {
      method: 'POST',
      body: JSON.stringify({ input }),
    }),
  stopRuntime: (id: string) =>
    request<import('./types').RuntimeControlResponse>(`/api/v1/runtimes/${id}/stop`, {
      method: 'POST',
    }),
  accountHandoff: (id: string, target: string) =>
    request<RuntimeSession>(`/api/v1/runtimes/${id}/handoff`, {
      method: 'POST',
      body: JSON.stringify({ target }),
    }),
  contextContinue: (id: string, provider: string, profile: string) =>
    request<RuntimeSession>(`/api/v1/runtimes/${id}/continue`, {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  getProviders: () => request<ProviderInfo[]>('/api/v1/providers'),
  getProfiles: () => request<ProfileInfo[]>('/api/v1/profiles'),
  getEvents: (runtimeId?: string) =>
    request<EventRecord[]>(`/api/v1/events${runtimeId ? `?runtime_id=${runtimeId}` : ''}`),
  deleteRuntime: (id: string) =>
    request<{ status: string }>(`/api/v1/runtimes/${id}`, { method: 'DELETE' }),
  cleanRuntimes: () =>
    request<{ cleaned: number; purged: number }>('/api/v1/runtimes', { method: 'DELETE' }),
};
