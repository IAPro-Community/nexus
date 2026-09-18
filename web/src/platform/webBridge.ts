import { PlatformBridge } from './platformBridge';
import { safeExternalUrl } from './externalUrl';
import { PlatformCapabilities, FilePickerOptions, NotificationOptions } from './capabilities';

export class WebBridge implements PlatformBridge {
  readonly kind = 'web' as const;

  getCapabilities(): PlatformCapabilities {
    return {
      native: false,
      filePicker: false,
      folderPicker: false,
      notifications: typeof window !== 'undefined' && 'Notification' in window,
      tray: false,
      nativeMenus: false,
      deepLinks: false,
      autoStart: false,
      windowManagement: false,
    };
  }

  async selectDirectory(_title?: string): Promise<string | null> {
    return null;
  }

  async selectFile(_options?: FilePickerOptions): Promise<string | null> {
    return null;
  }

  async showNotification(options: NotificationOptions): Promise<void> {
    if (typeof window === 'undefined' || !('Notification' in window)) return;
    if (Notification.permission === 'granted') {
      try {
        new Notification(options.title, {
          body: options.body,
          icon: options.icon || './nexus-icon.png',
          silent: options.silent,
        });
      } catch {
        // Fallback or ignore in browser
      }
    }
  }

  async openExternal(url: string): Promise<void> {
    const safeUrl = safeExternalUrl(url);
    if (safeUrl && typeof window !== 'undefined') {
      window.open(safeUrl, '_blank', 'noopener,noreferrer');
    }
  }

  async getSystemTheme(): Promise<'light' | 'dark' | 'unknown'> {
    if (typeof window !== 'undefined' && window.matchMedia) {
      if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
        return 'dark';
      }
      return 'light';
    }
    return 'unknown';
  }
}
