import { describe, expect, it, vi } from 'vitest';
import { isDynamicImportError, reloadForChunkErrorOnce } from './ErrorBoundary';

function createSessionStorage() {
  const values = new Map<string, string>();
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
  };
}

describe('ErrorBoundary chunk recovery', () => {
  it('recognizes dynamic import failures', () => {
    expect(isDynamicImportError(new TypeError('Failed to fetch dynamically imported module'))).toBe(
      true,
    );
    expect(isDynamicImportError(new Error('Workspace request failed'))).toBe(false);
  });

  it('reloads once per session and leaves a persistent failure recoverable', () => {
    const storage = createSessionStorage();
    const reload = vi.fn();
    const chunkError = new Error('Loading chunk BrowserSurface failed');

    expect(reloadForChunkErrorOnce(chunkError, storage, reload)).toBe(true);
    expect(reloadForChunkErrorOnce(chunkError, storage, reload)).toBe(false);
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it('does not reload for unrelated rendering errors', () => {
    const reload = vi.fn();
    expect(
      reloadForChunkErrorOnce(new Error('Render failed'), createSessionStorage(), reload),
    ).toBe(false);
    expect(reload).not.toHaveBeenCalled();
  });
});
