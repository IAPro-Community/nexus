import { useEffect, useRef } from 'react';

export interface VisibilityIntervalOptions {
  /** Trigger callback immediately when the document returns to visible state. */
  refreshOnVisible?: boolean;
}

export function isDocumentVisible(): boolean {
  return typeof document === 'undefined' || !document.hidden;
}

/**
 * Registers a window interval that skips ticks while `document.hidden` is true,
 * and optionally triggers an immediate refresh when the document becomes visible.
 */
export function registerVisibilityInterval(
  tick: () => void,
  delayMs: number,
  options: VisibilityIntervalOptions = {},
): () => void {
  const { refreshOnVisible = false } = options;
  const onVisible = () => {
    if (refreshOnVisible && isDocumentVisible()) {
      tick();
    }
  };
  const timer = window.setInterval(() => {
    if (isDocumentVisible()) {
      tick();
    }
  }, delayMs);
  if (refreshOnVisible && typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisible);
  }
  return () => {
    window.clearInterval(timer);
    if (refreshOnVisible && typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisible);
    }
  };
}

/**
 * Runs a periodic callback only while the active tab/window is visible.
 * Pass `delayMs = null` to suspend polling declaratively.
 */
export function useVisibilityInterval(
  callback: () => void,
  delayMs: number | null,
  options: VisibilityIntervalOptions = {},
): void {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;
  const refreshOnVisible = Boolean(options.refreshOnVisible);

  useEffect(() => {
    if (delayMs === null || delayMs <= 0) return;
    return registerVisibilityInterval(() => callbackRef.current(), delayMs, {
      refreshOnVisible,
    });
  }, [delayMs, refreshOnVisible]);
}
