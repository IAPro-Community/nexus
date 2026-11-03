import { afterEach, describe, expect, it, vi } from 'vitest';
import { resolveTerminalFontFamily, TERMINAL_FONT_FALLBACK, watchTerminalFontMetrics } from './terminalFont';

describe('resolveTerminalFontFamily', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('passes an actual monospace font stack to xterm, never a CSS var expression', () => {
    vi.stubGlobal('getComputedStyle', () => ({
      getPropertyValue: (name: string) =>
        name === '--nx-font-mono' ? "'JetBrains Mono', 'Fira Code', Menlo, Monaco, monospace" : '',
    }));
    const family = resolveTerminalFontFamily({} as Element);
    expect(family).toBe("'JetBrains Mono', 'Fira Code', Menlo, Monaco, monospace");
    expect(family).not.toContain('var(');
  });

  it('updates metrics when fonts are ready, and disposes listeners', async () => {
    const events = new Map<string, () => void>();
    let ready!: () => void;
    const promise = new Promise<void>((resolve) => { ready = resolve; });
    const add = vi.fn((type: string, listener: () => void) => events.set(type, listener));
    const remove = vi.fn((type: string) => events.delete(type));
    vi.stubGlobal('document', {
      fonts: { ready: promise, addEventListener: add, removeEventListener: remove },
    });
    const onChange = vi.fn();
    const dispose = watchTerminalFontMetrics(onChange);
    ready();
    await promise;
    await Promise.resolve();
    expect(onChange).toHaveBeenCalledOnce();
    events.get('loadingdone')?.();
    expect(onChange).toHaveBeenCalledTimes(2);
    dispose();
    expect(remove).toHaveBeenCalledWith('loadingdone', expect.any(Function));
    expect(events.has('loadingdone')).toBe(false);
  });

  it('ignores deferred font readiness after teardown', async () => {
    let ready!: () => void;
    const promise = new Promise<void>((resolve) => { ready = resolve; });
    vi.stubGlobal('document', {
      fonts: {
        ready: promise,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      },
    });
    const onChange = vi.fn();
    const dispose = watchTerminalFontMetrics(onChange);
    dispose();
    ready();
    await promise;
    await Promise.resolve();
    expect(onChange).not.toHaveBeenCalled();
  });

  it('falls back when theme tokens are missing or unresolved', () => {
    vi.stubGlobal('getComputedStyle', () => ({
      getPropertyValue: () => 'var(--missing-font)',
    }));
    expect(resolveTerminalFontFamily({} as Element)).toBe(TERMINAL_FONT_FALLBACK);
    vi.stubGlobal('getComputedStyle', () => ({ getPropertyValue: () => '' }));
    expect(resolveTerminalFontFamily({} as Element)).toBe(TERMINAL_FONT_FALLBACK);
  });
});
