import { Call } from '@wailsio/runtime';
import type {
  DesktopBootstrapInfo,
  FilePickerOptions,
  NativeVoiceRecording,
  NotificationOptions,
  PlatformCapabilities,
} from './capabilities';

export interface HudPlatformCapabilities {
  transparentWindow: boolean;
  framelessWindow: boolean;
  alwaysOnTop: boolean;
  globalPositioning: boolean;
  multiMonitor: boolean;
  autoHide: boolean;
  nativeNotifications: boolean;
  tray: boolean;
  autoStart: boolean;
  displayCount: number;
  platform: string;
  session?: string;
}

export interface HudWindowState {
  displayId: string;
  displayScale?: number;
  placement: string;
  x: number;
  y: number;
  width: number;
  height: number;
  compactWidth?: number;
  compactHeight?: number;
  expandedWidth?: number;
  expandedHeight?: number;
  mode: string;
  opacity: number;
  alwaysOnTop: boolean;
  autoHide: boolean;
}

export interface HudSizePreset {
  id: string;
  label: string;
  compactWidth: number;
  compactHeight: number;
  expandedWidth: number;
  expandedHeight: number;
}

type WailsV3Invoker = (methodName: string, ...args: unknown[]) => Promise<unknown>;

const runtimeInvoker: WailsV3Invoker = async (methodName, ...args) =>
  await Call.ByName(methodName, ...args);

let activeInvoker: WailsV3Invoker = runtimeInvoker;

export function setWailsV3InvokerForTests(invoker?: WailsV3Invoker): void {
  activeInvoker = invoker ?? runtimeInvoker;
}

const desktopServicePackage = 'github.com/IAPro-Community/nexus/internal/desktop';

function call<T>(service: 'App' | 'HUDController', method: string, ...args: unknown[]): Promise<T> {
  return activeInvoker(`${desktopServicePackage}.${service}.${method}`, ...args) as Promise<T>;
}

export const appV3 = {
  getBootstrapInfo: () => call<DesktopBootstrapInfo>('App', 'GetBootstrapInfo'),
  getCapabilities: () => call<PlatformCapabilities>('App', 'GetCapabilities'),
  selectDirectory: (title = '') => call<string>('App', 'SelectDirectory', title),
  selectFile: (options: FilePickerOptions = { title: '', defaultPath: '', filters: [] }) =>
    call<string>('App', 'SelectFile', options),
  showNotification: (options: NotificationOptions) =>
    call<void>('App', 'ShowNotification', options),
  openExternal: (url: string) => call<void>('App', 'OpenExternal', url),
  getSystemTheme: () => call<'light' | 'dark' | 'unknown'>('App', 'GetSystemTheme'),
  startVoiceCapture: () => call<void>('App', 'StartVoiceCapture'),
  stopVoiceCapture: () => call<NativeVoiceRecording>('App', 'StopVoiceCapture'),
  cancelVoiceCapture: () => call<void>('App', 'CancelVoiceCapture'),
  isMaximized: () => call<boolean>('App', 'IsMaximized'),
  minimizeWindow: () => call<void>('App', 'MinimizeWindow'),
  maximizeWindow: () => call<void>('App', 'MaximizeWindow'),
  unmaximizeWindow: () => call<void>('App', 'UnmaximizeWindow'),
  closeWindow: () => call<void>('App', 'CloseWindow'),
  quitApp: () => call<void>('App', 'QuitApp'),
};

export const hudV3 = {
  getCapabilities: () => call<HudPlatformCapabilities>('HUDController', 'GetHUDCapabilities'),
  getWindowState: () => call<HudWindowState>('HUDController', 'GetHUDWindowState'),
  captureWindowState: () => call<void>('HUDController', 'CaptureHUDWindowState'),
  setExpanded: (expanded: boolean) => call<void>('HUDController', 'SetHUDExpanded', expanded),
  setBubbleVisible: (visible: boolean) => call<void>('HUDController', 'SetHUDBubbleVisible', visible),
  getSizePresets: () => call<HudSizePreset[]>('HUDController', 'GetHUDSizePresets'),
  setSizePreset: (preset: string, expanded: boolean) =>
    call<void>('HUDController', 'SetHUDSizePreset', preset, expanded),
  move: (dx: number, dy: number) => call<void>('HUDController', 'MoveHUD', dx, dy),
  hide: () => call<void>('HUDController', 'HideHUD'),
  show: () => call<void>('HUDController', 'ShowHUD'),
  getAutoStartEnabled: () => call<boolean>('HUDController', 'GetHUDAutoStartEnabled'),
  setAutoStartEnabled: (enabled: boolean) =>
    call<void>('HUDController', 'SetHUDAutoStartEnabled', enabled),
  setAutoHide: (enabled: boolean) => call<void>('HUDController', 'SetHUDAutoHide', enabled),
};
