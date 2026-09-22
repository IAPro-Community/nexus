import { describe, expect, it } from 'vitest';
import {
  defaultThemePreferences,
  getThemePresetPalette,
  normalizeThemePreferences,
  resolveScheme,
  resolveThemeStyleVariables,
  themeStorageKey,
  type ThemePreferences,
} from './theme';
import { THEME_PRESETS } from './themePresets';

function parseHex(hex: string): [number, number, number] {
  const c = hex.replace('#', '');
  return [parseInt(c.slice(0, 2), 16), parseInt(c.slice(2, 4), 16), parseInt(c.slice(4, 6), 16)];
}

function srgbToLinear(c: number): number {
  const v = c / 255;
  return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
}

function relLuminance([r, g, b]: [number, number, number]): number {
  return 0.2126 * srgbToLinear(r) + 0.7152 * srgbToLinear(g) + 0.0722 * srgbToLinear(b);
}

function contrastRatio(hex1: string, hex2: string): number {
  const l1 = relLuminance(parseHex(hex1));
  const l2 = relLuminance(parseHex(hex2));
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

describe('theme model', () => {
  it('resolves system scheme from media preference', () => {
    expect(resolveScheme('system', true)).toBe('dark');
    expect(resolveScheme('system', false)).toBe('light');
  });

  it('keeps explicit and high contrast schemes', () => {
    expect(resolveScheme('dark', false)).toBe('dark');
    expect(resolveScheme('high-contrast', true)).toBe('high-contrast');
  });

  it('normalizes invalid persisted values', () => {
    const value = normalizeThemePreferences({
      scheme: 'pink',
      accent: 'orange',
      density: 'huge',
    } as unknown as ThemePreferences);
    expect(value).toEqual({
      scheme: 'system',
      accent: 'purple',
      density: 'compact',
      reducedMotion: false,
      preset: 'nexus-dark',
      isCustomized: false,
      fontScale: 1.0,
    });
  });

  it('preserves valid preset selection and handles round-trip serialization', () => {
    const original: ThemePreferences = {
      preset: 'nord',
      scheme: 'dark',
      accent: 'blue',
      density: 'comfortable',
      reducedMotion: true,
      isCustomized: false,
      fontScale: 1.15,
    };
    const serialized = JSON.stringify(original);
    const parsed = JSON.parse(serialized);
    const normalized = normalizeThemePreferences(parsed);
    expect(normalized).toEqual(original);
  });

  it('contains comprehensive suite of 10 presets', () => {
    const keys = Object.keys(THEME_PRESETS);
    expect(keys).toContain('nexus-dark');
    expect(keys).toContain('nexus-light');
    expect(keys).toContain('midnight');
    expect(keys).toContain('nord');
    expect(keys).toContain('dracula');
    expect(keys).toContain('monokai');
    expect(keys).toContain('solarized-dark');
    expect(keys).toContain('solarized-light');
    expect(keys).toContain('high-contrast-dark');
    expect(keys).toContain('high-contrast-light');
    expect(keys.length).toBeGreaterThanOrEqual(10);
  });

  it('normalizes out-of-bound fontScale values', () => {
    expect(
      normalizeThemePreferences({ fontScale: 0.1 } as unknown as ThemePreferences).fontScale,
    ).toBe(1.0);
    expect(
      normalizeThemePreferences({ fontScale: 5.0 } as unknown as ThemePreferences).fontScale,
    ).toBe(1.0);
    expect(
      normalizeThemePreferences({ fontScale: 1.3 } as unknown as ThemePreferences).fontScale,
    ).toBe(1.3);
  });

  it('uses a stable storage key', () => expect(themeStorageKey).toBe('iapro:nexus:theme:v1'));
});

describe('resolveThemeStyleVariables', () => {
  it('faz o preset padrão acompanhar o tema claro do sistema sem marcar customização', () => {
    const vars = resolveThemeStyleVariables(
      {
        ...defaultThemePreferences,
        scheme: 'system',
        preset: 'nexus-dark',
        isCustomized: false,
      },
      'light',
    );

    expect(vars['--nx-bg']).toBe('#f4f6fa');
    expect(vars['--nx-surface']).toBe('#ffffff');
    expect(vars['--nx-text']).toBe('#141821');
  });

  it('aplica variáveis nativas do preset quando isCustomized é false', () => {
    const vars = resolveThemeStyleVariables(
      {
        preset: 'nord',
        scheme: 'dark',
        accent: 'blue',
        density: 'compact',
        reducedMotion: false,
        isCustomized: false,
        fontScale: 1.0,
      },
      'dark',
    );
    expect(vars['--nx-bg']).toBe('#242933'); // Fundo nativo do Nord
    expect(vars['--nx-accent']).toBe('#88c0d0'); // Acento nativo do Nord
  });

  it('honra precedência autêntica de conflito: Nord nativo dark com override manual de esquema light e acento purple', () => {
    // Quando isCustomized é true, deve vencer o override manual
    const vars = resolveThemeStyleVariables(
      {
        preset: 'nord',
        scheme: 'light',
        accent: 'purple',
        density: 'compact',
        reducedMotion: false,
        isCustomized: true,
        fontScale: 1.0,
      },
      'light',
    );
    expect(vars['--nx-bg']).toBe('#f4f6fa'); // Sobrescrito pelo Light
    expect(vars['--nx-accent']).toBe('#8b5cf6'); // Sobrescrito pelo Purple
    expect(vars['--nx-accent-text']).toBe('#bba5ff');
  });

  it('não emite variáveis indefinidas, nulas ou vazias', () => {
    const vars = resolveThemeStyleVariables(defaultThemePreferences, 'dark');
    for (const [k, v] of Object.entries(vars)) {
      expect(k.startsWith('--nx-')).toBe(true);
      expect(typeof v).toBe('string');
      expect(v.length).toBeGreaterThan(0);
      expect(v).not.toBe('undefined');
    }
  });

  it('getThemePresetPalette extrai cores tipadas válidas de qualquer preset', () => {
    for (const def of Object.values(THEME_PRESETS)) {
      const palette = getThemePresetPalette(def);
      expect(palette.bg.startsWith('#')).toBe(true);
      expect(palette.surface.startsWith('#')).toBe(true);
      expect(palette.text.startsWith('#')).toBe(true);
      expect(palette.accent.startsWith('#')).toBe(true);
    }
  });
});

describe('WCAG AA Contrast Automation for all 10 presets', () => {
  it('satisfies WCAG AA contrast ratio for normal text (>= 4.5) and UI accent components (>= 3.0)', () => {
    for (const [id, def] of Object.entries(THEME_PRESETS)) {
      const bg = def.vars['--nx-bg'];
      const surface = def.vars['--nx-surface'];
      const text = def.vars['--nx-text'];
      const accent = def.accent;

      const textOnBg = contrastRatio(text, bg);
      const accentOnBg = contrastRatio(accent, bg);
      const accentOnSurface = contrastRatio(accent, surface);

      expect(textOnBg, `Preset ${id} text/bg contrast must be >= 4.5:1`).toBeGreaterThanOrEqual(
        4.5,
      );
      expect(accentOnBg, `Preset ${id} accent/bg contrast must be >= 3.0:1`).toBeGreaterThanOrEqual(
        3.0,
      );
      expect(
        accentOnSurface,
        `Preset ${id} accent/surface contrast must be >= 3.0:1`,
      ).toBeGreaterThanOrEqual(3.0);
    }
  });
});
