import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { WebBridge } from './webBridge';
import { DesktopBridge } from './desktopBridge';
import { initPlatformBridge, getPlatformBridge } from './index';
import { safeExternalUrl } from './externalUrl';
import { setWailsV3InvokerForTests } from './wailsV3';

describe('PlatformBridge', () => {
  beforeEach(() => {
    (globalThis as any).window = {
      location: { protocol: 'http:', hostname: 'localhost', search: '' },
      open: () => undefined,
    };
  });

  afterEach(() => {
    setWailsV3InvokerForTests();
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
    expect(getPlatformBridge()).toBe(bridge);
  });

  it('initializes DesktopBridge from the Wails v3 origin', () => {
    (globalThis as any).window.location = {
      protocol: 'wails:',
      hostname: 'wails.localhost',
      search: '',
    };
    const bridge = initPlatformBridge();
    expect(bridge.kind).toBe('desktop');
    expect(bridge.getCapabilities().native).toBe(true);
    expect(bridge.getCapabilities().filePicker).toBe(false);
  });

  it('selectDirectory delegates to the Wails v3 App service', async () => {
    setWailsV3InvokerForTests(async (method, ...args) => {
      expect(method).toBe('github.com/IAPro-Community/nexus/internal/desktop.App.SelectDirectory');
      expect(args).toEqual(['Choose folder']);
      return '/selected/path/Choose folder';
    });
    const bridge = new DesktopBridge();
    await expect(bridge.selectDirectory('Choose folder')).resolves.toBe(
      '/selected/path/Choose folder',
    );
  });

  it('selectDirectory returns null in WebBridge', async () => {
    const bridge = new WebBridge();
    await expect(bridge.selectDirectory()).resolves.toBeNull();
  });

  it('getBootstrapInfo returns server connection and auth tokens', async () => {
    setWailsV3InvokerForTests(async (method) => {
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.GetBootstrapInfo') {
        return {
          serverUrl: 'http://127.0.0.1:45678',
          sessionToken: 'sess_12345',
          csrfToken: 'csrf_67890',
        };
      }
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.GetCapabilities') {
        return {
          native: true,
          filePicker: true,
          folderPicker: true,
          notifications: false,
          tray: true,
          nativeMenus: true,
          deepLinks: false,
          autoStart: true,
          windowManagement: true,
        };
      }
      throw new Error(`unexpected method ${method}`);
    });
    const bridge = new DesktopBridge();
    await expect(bridge.getBootstrapInfo()).resolves.toEqual({
      serverUrl: 'http://127.0.0.1:45678',
      sessionToken: 'sess_12345',
      csrfToken: 'csrf_67890',
    });
    expect(bridge.getCapabilities().tray).toBe(true);
  });

  it('coalesces concurrent desktop bootstrap requests', async () => {
    let bootstrapCalls = 0;
    setWailsV3InvokerForTests(async (method) => {
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.GetBootstrapInfo') {
        bootstrapCalls += 1;
        await Promise.resolve();
        return {
          serverUrl: 'http://127.0.0.1:45678',
          sessionToken: 'sess_12345',
          csrfToken: 'csrf_67890',
        };
      }
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.GetCapabilities') {
        return {
          native: true,
          filePicker: false,
          folderPicker: false,
          notifications: false,
          tray: true,
          nativeMenus: true,
          deepLinks: false,
          autoStart: true,
          windowManagement: true,
        };
      }
      throw new Error(`unexpected method ${method}`);
    });
    const bridge = new DesktopBridge();
    const [first, second] = await Promise.all([
      bridge.getBootstrapInfo(),
      bridge.getBootstrapInfo(),
    ]);
    expect(bootstrapCalls).toBe(1);
    expect(first).toEqual(second);
  });

  it('routes native voice capture through the Wails v3 App service', async () => {
    const calls: string[] = [];
    setWailsV3InvokerForTests(async (method) => {
      calls.push(method);
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.StopVoiceCapture') {
        return { dataBase64: 'UklGRg==', mimeType: 'audio/wav', durationMs: 321 };
      }
      return undefined;
    });
    const bridge = new DesktopBridge();
    await bridge.startNativeVoiceCapture();
    await expect(bridge.stopNativeVoiceCapture()).resolves.toEqual({
      dataBase64: 'UklGRg==',
      mimeType: 'audio/wav',
      durationMs: 321,
    });
    await bridge.cancelNativeVoiceCapture();
    expect(calls).toEqual([
      'github.com/IAPro-Community/nexus/internal/desktop.App.StartVoiceCapture',
      'github.com/IAPro-Community/nexus/internal/desktop.App.StopVoiceCapture',
      'github.com/IAPro-Community/nexus/internal/desktop.App.CancelVoiceCapture',
    ]);
  });

  it('uses backend capability evidence instead of method existence', async () => {
    setWailsV3InvokerForTests(async (method) => {
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.GetBootstrapInfo') {
        return {
          serverUrl: 'http://127.0.0.1:45678',
          sessionToken: 'sess_12345',
          csrfToken: 'csrf_67890',
        };
      }
      if (method === 'github.com/IAPro-Community/nexus/internal/desktop.App.GetCapabilities') {
        return {
          native: true,
          filePicker: false,
          folderPicker: false,
          notifications: false,
          tray: false,
          nativeMenus: false,
          deepLinks: false,
          autoStart: false,
          windowManagement: true,
        };
      }
      throw new Error(`unexpected method ${method}`);
    });
    const bridge = new DesktopBridge();
    expect(bridge.getCapabilities().filePicker).toBe(false);
    await bridge.getBootstrapInfo();
    expect(bridge.getCapabilities().filePicker).toBe(false);
    expect(bridge.getCapabilities().folderPicker).toBe(false);
  });
});
