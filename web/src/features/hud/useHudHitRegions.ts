import { useEffect } from 'react';
import { hudV3 } from '../../platform';
import { collectHudHitRegions } from './hudHitRegions';

const DEBOUNCE_MS = 120;
const REFRESH_MS = 2000;

/**
 * Keeps the native HUD window in sync with the interactive areas of the React
 * tree. The native watcher turns everything outside these rectangles into a
 * click-through surface, so transparent margins stop swallowing clicks meant
 * for the window behind the HUD.
 */
export function useHudHitRegions(enabled: boolean): void {
  useEffect(() => {
    if (!enabled || typeof document === 'undefined') return;
    let disposed = false;
    let timer: number | undefined;

    const push = () => {
      if (disposed) return;
      void hudV3.setHitRegions(collectHudHitRegions(document)).catch(() => undefined);
    };
    const schedule = () => {
      if (disposed) return;
      if (timer !== undefined) window.clearTimeout(timer);
      timer = window.setTimeout(push, DEBOUNCE_MS);
    };

    push();

    let resizeObserver: ResizeObserver | undefined;
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(schedule);
      resizeObserver.observe(document.documentElement);
      if (document.body) resizeObserver.observe(document.body);
    }

    let mutationObserver: MutationObserver | undefined;
    if (typeof MutationObserver !== 'undefined' && document.body) {
      mutationObserver = new MutationObserver(schedule);
      mutationObserver.observe(document.body, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ['class', 'style', 'hidden', 'aria-expanded', 'data-state'],
      });
    }

    window.addEventListener('resize', schedule);
    const interval = window.setInterval(push, REFRESH_MS);

    return () => {
      disposed = true;
      if (timer !== undefined) window.clearTimeout(timer);
      resizeObserver?.disconnect();
      mutationObserver?.disconnect();
      window.removeEventListener('resize', schedule);
      window.clearInterval(interval);
    };
  }, [enabled]);
}
