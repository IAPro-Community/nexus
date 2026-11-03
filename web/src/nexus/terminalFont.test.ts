import { afterEach, describe, expect, it, vi } from 'vitest';
import { resolveTerminalFontFamily, TERMINAL_FONT_FALLBACK } from './terminalFont';

describe('resolveTerminalFontFamily', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('passes an actual monospace font stack to xterm, never a CSS var expression', () => {
    vi.stubGlobal('getComputedStyle', () => ({
      getPropertyValue: (name: string) =>
        name === '--nx-font-mono'
          ? "'JetBrains Mono', 'Fira Code', Menlo, Monaco, monospace"
          : '',
    }));
    const family = resolveTerminalFontFamily({} as Element);
    expect(family).toBe("'JetBrains Mono', 'Fira Code', Menlo, Monaco, monospace");
    expect(family).not.toContain('var(');
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
