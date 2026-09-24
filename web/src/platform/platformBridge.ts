import {
  PlatformCapabilities,
  FilePickerOptions,
  NotificationOptions,
  DesktopBootstrapInfo,
  NativeVoiceRecording,
} from './capabilities';

export interface PlatformBridge {
  readonly kind: 'web' | 'desktop';

  getCapabilities(): PlatformCapabilities;

  selectDirectory(title?: string): Promise<string | null>;
  selectFile(options?: FilePickerOptions): Promise<string | null>;

  showNotification(options: NotificationOptions): Promise<void>;

  openExternal(url: string): Promise<void>;

  getSystemTheme(): Promise<'light' | 'dark' | 'unknown'>;

  getBootstrapInfo?(): Promise<DesktopBootstrapInfo | null>;

  startNativeVoiceCapture?(): Promise<void>;
  stopNativeVoiceCapture?(): Promise<NativeVoiceRecording>;
  cancelNativeVoiceCapture?(): Promise<void>;

  isMaximized?(): Promise<boolean>;
  minimizeWindow?(): Promise<void>;
  maximizeWindow?(): Promise<void>;
  unmaximizeWindow?(): Promise<void>;
  closeWindow?(): Promise<void>;
  quitApp?(): Promise<void>;
}

let activeBridge: PlatformBridge | null = null;

export function setPlatformBridge(bridge: PlatformBridge): void {
  activeBridge = bridge;
}

export function getPlatformBridge(): PlatformBridge {
  if (!activeBridge) {
    throw new Error('PlatformBridge has not been initialized');
  }
  return activeBridge;
}
