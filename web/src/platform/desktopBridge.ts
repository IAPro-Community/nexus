import { PlatformBridge } from './platformBridge';
import {
  PlatformCapabilities,
  FilePickerOptions,
  NotificationOptions,
  DesktopBootstrapInfo,
  NativeVoiceRecording,
} from './capabilities';
import { safeExternalUrl } from './externalUrl';
import { appV3 } from './wailsV3';

export class DesktopBridge implements PlatformBridge {
  readonly kind = 'desktop' as const;

  private cachedBootstrap: DesktopBootstrapInfo | null = null;
  private bootstrapRequest: Promise<DesktopBootstrapInfo | null> | null = null;
  private cachedCapabilities: PlatformCapabilities | null = null;
  private capabilitiesRequest: Promise<PlatformCapabilities | null> | null = null;

  private async loadCapabilities(): Promise<PlatformCapabilities | null> {
    if (this.cachedCapabilities) return this.cachedCapabilities;
    if (this.capabilitiesRequest) return this.capabilitiesRequest;

    this.capabilitiesRequest = appV3
      .getCapabilities()
      .then((capabilities) => {
        this.cachedCapabilities = capabilities;
        return capabilities;
      })
      .catch(() => null)
      .finally(() => {
        this.capabilitiesRequest = null;
      });
    return this.capabilitiesRequest;
  }

  async getBootstrapInfo(): Promise<DesktopBootstrapInfo | null> {
    if (this.cachedBootstrap) return this.cachedBootstrap;
    if (this.bootstrapRequest) return this.bootstrapRequest;

    this.bootstrapRequest = this.loadBootstrapInfo().finally(() => {
      this.bootstrapRequest = null;
    });
    return this.bootstrapRequest;
  }

  private async loadBootstrapInfo(): Promise<DesktopBootstrapInfo | null> {
    try {
      const info = await appV3.getBootstrapInfo();
      if (info?.serverUrl) {
        await this.loadCapabilities();
        this.cachedBootstrap = info;
        return info;
      }
    } catch (err) {
      console.warn('Wails v3 App.GetBootstrapInfo error:', err);
    }
    return null;
  }

  getCapabilities(): PlatformCapabilities {
    return (
      this.cachedCapabilities ?? {
        native: true,
        filePicker: false,
        folderPicker: false,
        notifications: false,
        tray: false,
        nativeMenus: false,
        deepLinks: false,
        autoStart: false,
        windowManagement: true,
        nativeVoiceCapture: false,
      }
    );
  }

  async selectDirectory(title?: string): Promise<string | null> {
    try {
      const path = await appV3.selectDirectory(title);
      return path || null;
    } catch {
      return null;
    }
  }

  async selectFile(options?: FilePickerOptions): Promise<string | null> {
    try {
      const path = await appV3.selectFile(options);
      return path || null;
    } catch {
      return null;
    }
  }

  async showNotification(options: NotificationOptions): Promise<void> {
    try {
      await appV3.showNotification(options);
      return;
    } catch {
      // Browser notification remains a presentation fallback.
    }
    if (
      typeof window !== 'undefined' &&
      'Notification' in window &&
      Notification.permission === 'granted'
    ) {
      new Notification(options.title, { body: options.body, icon: options.icon });
    }
  }

  async openExternal(url: string): Promise<void> {
    const safeUrl = safeExternalUrl(url);
    if (!safeUrl) return;
    try {
      await appV3.openExternal(safeUrl);
      return;
    } catch {
      window.open(safeUrl, '_blank', 'noopener,noreferrer');
    }
  }

  async getSystemTheme(): Promise<'light' | 'dark' | 'unknown'> {
    try {
      return await appV3.getSystemTheme();
    } catch {
      if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) return 'dark';
      return 'light';
    }
  }

  async startNativeVoiceCapture(): Promise<void> {
    await appV3.startVoiceCapture();
  }

  async stopNativeVoiceCapture(): Promise<NativeVoiceRecording> {
    return await appV3.stopVoiceCapture();
  }

  async cancelNativeVoiceCapture(): Promise<void> {
    await appV3.cancelVoiceCapture();
  }

  async isMaximized(): Promise<boolean> {
    try {
      return await appV3.isMaximized();
    } catch {
      return false;
    }
  }

  async minimizeWindow(): Promise<void> {
    await appV3.minimizeWindow();
  }

  async maximizeWindow(): Promise<void> {
    await appV3.maximizeWindow();
  }

  async unmaximizeWindow(): Promise<void> {
    await appV3.unmaximizeWindow();
  }

  async closeWindow(): Promise<void> {
    await appV3.closeWindow();
  }

  async quitApp(): Promise<void> {
    await appV3.quitApp();
  }
}
