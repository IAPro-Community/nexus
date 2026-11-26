import { isDesktopApp } from '../platform';

/**
 * Leaf detector for the native HUD window (`?nativeHud=1`). Lives outside the
 * HUD component module so the boot bundle can branch on it without pulling
 * `NexusHud` and its panels into the entry chunk.
 */
export const isDesktopHudApp = () => {
  if (typeof window === 'undefined' || !isDesktopApp()) return false;
  return new URLSearchParams(window.location.search).get('nativeHud') === '1';
};
