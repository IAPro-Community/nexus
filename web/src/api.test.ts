import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  api,
  clearDesktopAuth,
  getDesktopBaseUrl,
  getDesktopWebSocketBaseUrl,
  initSession,
  request,
  setCSRFToken,
  setDesktopAuth,
} from './api';
import { setPlatformBridge, type PlatformBridge, WebBridge } from './platform';

describe('api request layer', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.stubGlobal('fetch', vi.fn());
    setPlatformBridge(new WebBridge());
    setCSRFToken('');
    clearDesktopAuth();
  });

  it('keeps Wails HTTP same-origin while using the Core listener for WebSocket', async () => {
    setDesktopAuth('desktop-token', 'http://127.0.0.1:43123/');
    expect(getDesktopBaseUrl()).toBe('');
    expect(getDesktopWebSocketBaseUrl()).toBe('http://127.0.0.1:43123');

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    });
    vi.stubGlobal('fetch', fetchMock);

    await api.getWorkspaces();

    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/workspaces');
    expect(new Headers(fetchMock.mock.calls[0][1]?.headers).get('Authorization')).toBe(
      'Bearer desktop-token',
    );
  });

  it('exposes the shared request transport and CSRF setter for domain clients', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    setCSRFToken('shared-csrf');

    await request('/api/v1/shared-contract', { method: 'POST', body: '{}' });

    const [, init] = fetchMock.mock.calls[0];
    expect(init.headers.get('X-CSRF-Token')).toBe('shared-csrf');
  });

  it('throws on non-ok responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: () => Promise.resolve({ error: 'authentication required' }),
      }),
    );

    await expect(api.getWorkspaces()).rejects.toThrow('authentication required');
  });

  it('preserves stable API error codes for the shared request layer', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: () => Promise.resolve({ error: 'project not found', code: 'PROJECT_NOT_FOUND' }),
      }),
    );

    await expect(api.getWorkspaces()).rejects.toMatchObject({
      status: 404,
      code: 'PROJECT_NOT_FOUND',
    });
  });

  it('sends JSON bodies on state-changing calls', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'stopping' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await api.stopRuntime('rt-1');

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/runtimes/rt-1/stop');
    expect(init.method).toBe('POST');
    expect(init.headers.get('Content-Type')).toBe('application/json');
  });

  it('keeps the runtime detail capability contract typed and nullable', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          session: { runtime_id: 'rt-1', workspace: '/tmp', state: 'RUNNING' },
          capabilities: null,
        }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.getRuntime('rt-1')).resolves.toMatchObject({
      session: { runtime_id: 'rt-1' },
      capabilities: null,
    });
  });

  it('includes CSRF token header after session bootstrap', async () => {
    const sessionFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ authenticated: true, csrf_token: 'tok-123' }),
    });
    vi.stubGlobal('fetch', sessionFetch);
    await initSession();

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await api.stopRuntime('rt-2');

    const [, init] = fetchMock.mock.calls[0];
    expect(init.headers.get('X-CSRF-Token')).toBe('tok-123');
  });

  it('refreshes a stale CSRF token and retries a mutation exactly once', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: false,
        status: 403,
        statusText: 'Forbidden',
        json: () => Promise.resolve({ error: 'invalid CSRF token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ authenticated: true, csrf_token: 'fresh-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ status: 'stopping' }),
      });
    vi.stubGlobal('fetch', fetchMock);

    await api.stopRuntime('rt-stale');

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1][0]).toBe('/api/v1/session');
    const retryInit = fetchMock.mock.calls[2]?.[1] as RequestInit | undefined;
    expect(new Headers(retryInit?.headers).get('X-CSRF-Token')).toBe('fresh-token');
  });

  it('exchanges the browser fragment bootstrap token and removes it from history', async () => {
    const replaceState = vi.fn();
    vi.stubGlobal('window', {
      location: {
        hash: '#nexus_bootstrap=bootstrap-token',
        pathname: '/',
        search: '',
        protocol: 'http:',
        hostname: '127.0.0.1',
      },
      history: { replaceState },
    });
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ authenticated: true, csrf_token: 'csrf-from-browser' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(initSession()).resolves.toMatchObject({
      authenticated: true,
      csrf_token: 'csrf-from-browser',
    });
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/auth/bootstrap');
    expect(fetchMock.mock.calls[0][1].body).toBe(JSON.stringify({ token: 'bootstrap-token' }));
    expect(replaceState).toHaveBeenCalledWith(null, '', '/');
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('never blocks Wails bootstrap when the embedded window has no hostname', async () => {
    const bridge: PlatformBridge = {
      kind: 'desktop',
      getCapabilities: () => ({
        native: true,
        filePicker: true,
        folderPicker: true,
        notifications: false,
        tray: false,
        nativeMenus: false,
        deepLinks: false,
        autoStart: false,
        windowManagement: true,
      }),
      getBootstrapInfo: async () => ({
        serverUrl: 'http://127.0.0.1:43123',
        sessionToken: 'desktop-session',
        csrfToken: 'desktop-csrf',
      }),
      selectDirectory: async () => null,
      selectFile: async () => null,
      showNotification: async () => undefined,
      openExternal: async () => undefined,
      getSystemTheme: async () => 'unknown',
    };
    setPlatformBridge(bridge);
    vi.stubGlobal('window', { go: {}, location: { protocol: 'wails:' } });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ authenticated: true, csrf_token: 'desktop-csrf' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(initSession()).resolves.toMatchObject({ authenticated: true });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/session', {
      headers: { Accept: 'application/json', Authorization: 'Bearer desktop-session' },
    });
  });

  it('allows HTTPS tunnel bootstrap before checking an existing browser cookie', async () => {
    const replaceState = vi.fn();
    vi.stubGlobal('window', {
      location: {
        hash: '#nexus_bootstrap=remote-invite',
        pathname: '/',
        search: '',
        protocol: 'https:',
        hostname: 'example.trycloudflare.com',
      },
      history: { replaceState },
    });
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        authenticated: true,
        csrf_token: 'remote-csrf',
        expires_at: '2026-10-09T00:00:00Z',
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(initSession()).resolves.toMatchObject({
      authenticated: true,
      csrf_token: 'remote-csrf',
    });
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/auth/bootstrap');
    expect(replaceState).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('refuses plaintext LAN bootstrap and scrubs invite fragment without sending credentials', async () => {
    const replaceState = vi.fn();
    vi.stubGlobal('window', {
      location: {
        hash: '#nexus_bootstrap=secret',
        pathname: '/',
        search: '',
        protocol: 'http:',
        hostname: '192.168.0.33',
      },
      history: { replaceState },
    });
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    await expect(initSession()).resolves.toMatchObject({ authenticated: false });
    expect(replaceState).toHaveBeenCalledTimes(1);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('keeps desktop bootstrap API calls same-origin inside the Wails webview', async () => {
    const desktopBridge: PlatformBridge = {
      kind: 'desktop',
      getCapabilities: () => ({
        native: true,
        filePicker: true,
        folderPicker: true,
        notifications: false,
        tray: false,
        nativeMenus: false,
        deepLinks: false,
        autoStart: false,
        windowManagement: true,
      }),
      getBootstrapInfo: async () => ({
        serverUrl: 'http://127.0.0.1:43123',
        sessionToken: 'desktop-session',
        csrfToken: 'desktop-csrf',
      }),
      selectDirectory: async () => null,
      selectFile: async () => null,
      showNotification: async () => undefined,
      openExternal: async () => undefined,
      getSystemTheme: async () => 'unknown',
    };
    setPlatformBridge(desktopBridge);
    vi.stubGlobal('window', { go: {}, location: { protocol: 'wails:' } });
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ authenticated: true, csrf_token: 'desktop-csrf' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(initSession()).resolves.toMatchObject({ authenticated: true });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/session', {
      headers: { Accept: 'application/json', Authorization: 'Bearer desktop-session' },
    });
  });
});
