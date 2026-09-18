import { describe, it, expect, beforeEach } from 'vitest';
import { WebBridge } from './webBridge';
import { DesktopBridge } from './desktopBridge';
import { initPlatformBridge, getPlatformBridge } from './index';
import { safeExternalUrl } from './externalUrl';

describe('PlatformBridge', () => {
  beforeEach(() => {
    (globalThis as any).window = globalThis;
    delete (globalThis as any).window.go;
    delete (globalThis as any).window.runtime;
  });

  it('allows only structural HTTP(S) external URLs', () => {
    expect(safeExternalUrl('https://example.com/path?q=1')).toBe('https://example.com/path?q=1');
    for (const unsafe of [
      'javascript:alert(1)',
      'data:text/plain,hello',
      'file:///tmp/x',
      'not url',
    ]) {
      expect(safeExternalUrl(unsafe)).toBeNull();
    }
  });

  it('initializes WebBridge in standard browser environment', () => {
    const bridge = initPlatformBridge();
    expect(bridge.kind).toBe('web');
    expect(bridge.getCapabilities().native).toBe(false);
    expect(bridge.getCapabilities().filePicker).toBe(false);
    expect(bridge.getCapabilities().folderPicker).toBe(false);
    expect(getPlatformBridge()).toBe(bridge);
  });

  it('initializes DesktopBridge when Wails bindings are detected', () => {
    (globalThis as any).window.go = {
      desktop: {
        App: {
          SelectDirectory: async () => '/home/user/project',
        },
      },
    };
    const bridge = initPlatformBridge();
    expect(bridge.kind).toBe('desktop');
    expect(bridge.getCapabilities().native).toBe(true);
    expect(bridge.getCapabilities().filePicker).toBe(false);
    expect(bridge.getCapabilities().folderPicker).toBe(true);
    expect(bridge.getCapabilities().tray).toBe(false);
  });

  it('selectDirectory delegates to desktop App binding', async () => {
    (globalThis as any).window.go = {
      desktop: {
        App: {
          SelectDirectory: async (title?: string) => `/selected/path/${title || 'default'}`,
        },
      },
    };
    const bridge = new DesktopBridge();
    const result = await bridge.selectDirectory('Choose folder');
    expect(result).toBe('/selected/path/Choose folder');
  });

  it('selectDirectory returns null in WebBridge', async () => {
    const bridge = new WebBridge();
    const result = await bridge.selectDirectory();
    expect(result).toBeNull();
  });

  it('getBootstrapInfo returns server connection and auth tokens', async () => {
    (globalThis as any).window.go = {
      desktop: {
        App: {
          GetBootstrapInfo: async () => ({
            serverUrl: 'http://127.0.0.1:45678',
            sessionToken: 'sess_12345',
            csrfToken: 'csrf_67890',
          }),
        },
      },
    };
    const bridge = new DesktopBridge();
    const info = await bridge.getBootstrapInfo();
    expect(info).toEqual({
      serverUrl: 'http://127.0.0.1:45678',
      sessionToken: 'sess_12345',
      csrfToken: 'csrf_67890',
    });
  });

  it('coalesces concurrent desktop bootstrap requests', async () => {
    let calls = 0;
    (globalThis as any).window.go = {
      desktop: {
        App: {
          GetBootstrapInfo: async () => {
            calls += 1;
            await Promise.resolve();
            return {
              serverUrl: 'http://127.0.0.1:45678',
              sessionToken: 'sess_12345',
              csrfToken: 'csrf_67890',
            };
          },
        },
      },
    };
    const bridge = new DesktopBridge();

    const [first, second] = await Promise.all([
      bridge.getBootstrapInfo(),
      bridge.getBootstrapInfo(),
    ]);

    expect(calls).toBe(1);
    expect(first).toEqual(second);
  });

  it('uses backend capability evidence instead of Wails method presence', async () => {
    (globalThis as any).window.go = {
      desktop: {
        App: {
          GetBootstrapInfo: async () => ({
            serverUrl: 'http://127.0.0.1:45678',
            sessionToken: 'sess_12345',
            csrfToken: 'csrf_67890',
          }),
          GetCapabilities: async () => ({
            native: true,
            filePicker: false,
            folderPicker: false,
            notifications: false,
            tray: false,
            nativeMenus: false,
            deepLinks: false,
            autoStart: false,
            windowManagement: true,
          }),
          SelectFile: async () => '/should-not-imply-capability',
        },
      },
    };
    const bridge = new DesktopBridge();

    expect(bridge.getCapabilities().filePicker).toBe(false);
    await bridge.getBootstrapInfo();
    expect(bridge.getCapabilities().filePicker).toBe(false);
    expect(bridge.getCapabilities().folderPicker).toBe(false);
  });
});
