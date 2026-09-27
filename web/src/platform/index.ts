import { PlatformBridge, setPlatformBridge } from './platformBridge';
import { WebBridge } from './webBridge';
import { DesktopBridge } from './desktopBridge';

export * from './capabilities';
export * from './platformBridge';
export * from './webBridge';
export * from './desktopBridge';
export * from './wailsV3';

export function isDesktopApp(): boolean {
  if (typeof window === 'undefined') return false;
  const protocol = window.location?.protocol ?? '';
  const hostname = window.location?.hostname ?? '';
  return (
    protocol === 'wails:' ||
    hostname === 'wails' ||
    hostname === 'wails.localhost' ||
    (typeof navigator !== 'undefined' &&
      (navigator.userAgent?.toLowerCase().includes('wails') ?? false))
  );
}

export function initPlatformBridge(): PlatformBridge {
  const bridge = isDesktopApp() ? new DesktopBridge() : new WebBridge();
  setPlatformBridge(bridge);
  return bridge;
}
