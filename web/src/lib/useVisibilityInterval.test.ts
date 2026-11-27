import { afterEach, describe, expect, it, vi } from 'vitest';
import { isDocumentVisible, registerVisibilityInterval } from './useVisibilityInterval';

describe('useVisibilityInterval helpers', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('skips interval ticks when document is hidden and fires immediately on visibility restore', () => {
    vi.useFakeTimers();
    let hidden = true;
    const listeners = new Set<() => void>();
    const fakeDocument = {
      get hidden() {
        return hidden;
      },
      addEventListener: (_event: string, fn: () => void) => listeners.add(fn),
      removeEventListener: (_event: string, fn: () => void) => listeners.delete(fn),
    };
    const fakeWindow = {
      setInterval: (fn: () => void, ms: number) => globalThis.setInterval(fn, ms),
      clearInterval: (id: number) => globalThis.clearInterval(id),
    };
    vi.stubGlobal('document', fakeDocument);
    vi.stubGlobal('window', fakeWindow);

    const cb = vi.fn();
    const stop = registerVisibilityInterval(cb, 5000, { refreshOnVisible: true });

    expect(isDocumentVisible()).toBe(false);
    vi.advanceTimersByTime(15_000);
    expect(cb).not.toHaveBeenCalled();

    hidden = false;
    expect(isDocumentVisible()).toBe(true);
    for (const fn of listeners) fn();
    expect(cb).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(10_000);
    expect(cb).toHaveBeenCalledTimes(3);

    stop();
    vi.advanceTimersByTime(10_000);
    expect(cb).toHaveBeenCalledTimes(3);
  });
});
