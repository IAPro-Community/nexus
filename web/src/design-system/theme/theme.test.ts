import { describe, expect, it } from 'vitest';
import {
  defaultThemePreferences,
  getThemePresetPalette,
  getThemePresetFinish,
  normalizeThemePreferences,
  resolveScheme,
  resolveThemeStyleVariables,
  themeStorageKey,
  type ThemePreferences,
  type ThemePresetKey,
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
      fontFamily: 'default',
      iconStyle: 'default',
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
      fontFamily: 'readable',
      iconStyle: 'light',
    };
    const serialized = JSON.stringify(original);
    const parsed = JSON.parse(serialized);
    const normalized = normalizeThemePreferences(parsed);
    expect(normalized).toEqual(original);
  });

  it('repairs persisted preset/scheme mismatches when the theme is not customized', () => {
    expect(
      normalizeThemePreferences({
        ...defaultThemePreferences,
        preset: 'nord-light',
        scheme: 'dark',
        isCustomized: false,
      }).scheme,
    ).toBe('light');

    expect(
      normalizeThemePreferences({
        ...defaultThemePreferences,
        preset: 'high-contrast-dark',
        scheme: 'light',
        isCustomized: false,
      }).scheme,
    ).toBe('high-contrast');
  });

  it('preserves system-following defaults and explicit customized schemes', () => {
    expect(normalizeThemePreferences(defaultThemePreferences).scheme).toBe('system');

    expect(
      normalizeThemePreferences({
        ...defaultThemePreferences,
        ...defaultThemePreferences,
        preset: 'nord',
        scheme: 'light',
        isCustomized: true,
      }).scheme,
    ).toBe('light');
  });

  it('contains the Nexus 1.0 preset suite, including the approved visual themes', () => {
    const keys = Object.keys(THEME_PRESETS);
    expect(keys).toEqual(
      expect.arrayContaining([
        'nexus-dark',
        'nexus-light',
        'slate-flat',
        'paper-flat',
        'clay-flat',
        'nexus-dusk',
        'ember-studio',
        'midnight',
        'nord',
        'dracula',
        'monokai',
        'solarized-dark',
        'solarized-light',
        'graphite-focus',
        'vasco-noir',
        'nord-light',
        'aurora-ops',
        'night',
        'obsidian',
        'high-contrast-dark',
        'high-contrast-light',
      ]),
    );
    expect(keys.length).toBeGreaterThanOrEqual(20);
  });

  it('keeps Night cool-toned and explicitly non-pink', () => {
    const night = THEME_PRESETS.night;
    expect(night.scheme).toBe('dark');
    expect(night.vars['--nx-bg']).toBe('#090c12');
    expect(night.vars['--nx-accent']).toBe('#6f95e8');
    expect(night.vars['--nx-accent']).not.toMatch(/ff79c6|ff69b4|ec4899/i);
  });

  it('keeps registry keys, ids and primary accent tokens consistent', () => {
    const names = new Set<string>();

    for (const [key, definition] of Object.entries(THEME_PRESETS)) {
      expect(definition.id).toBe(key);
      expect(definition.accent).toBe(definition.vars['--nx-accent']);
      expect(names.has(definition.name), `Duplicate theme name: ${definition.name}`).toBe(false);
      names.add(definition.name);
    }
  });

  it('preserves the first three flat presets without changing legacy preset finishes', () => {
    for (const preset of ['slate-flat', 'paper-flat', 'clay-flat'] as const) {
      expect(THEME_PRESETS[preset].finish).toBe('flat');
      expect(THEME_PRESETS[preset].vars['--nx-bg']).not.toBe(
        THEME_PRESETS[preset].vars['--nx-surface'],
      );
    }

    expect(getThemePresetFinish('nexus-dark')).toBe('standard');
    expect(getThemePresetFinish('monokai')).toBe('standard');
    expect(getThemePresetFinish('obsidian')).toBe('standard');
  });

  it('uses a dedicated restrained gradient finish for Nexus Dusk only', () => {
    const orca = THEME_PRESETS['nexus-dusk'];
    expect(orca.finish).toBe('subtle-gradient');
    expect(getThemePresetFinish('nexus-dusk')).toBe('subtle-gradient');
    expect(orca.scheme).toBe('dark');
    expect(orca.vars['--nx-bg']).toBe('#0b0c0e');
    expect(orca.vars['--nx-surface']).toBe('#17191c');
    expect(getThemePresetFinish('slate-flat')).toBe('flat');
    expect(getThemePresetFinish('nexus-dark')).toBe('standard');
  });

  it('preserves Nexus Dusk selection and consistent scheme after a persisted reload', () => {
    const normalized = normalizeThemePreferences({
      ...defaultThemePreferences,
      preset: 'nexus-dusk',
      scheme: 'light',
      density: 'comfortable',
      isCustomized: false,
    });
    expect(normalized.preset).toBe('nexus-dusk');
    expect(normalized.scheme).toBe('dark');
    expect(normalized.density).toBe('comfortable');
    expect(resolveThemeStyleVariables(normalized, 'dark')['--nx-bg']).toBe('#0b0c0e');
  });

  it('restores a flat preset after persistence without affecting user density', () => {
    const normalized = normalizeThemePreferences({
      ...defaultThemePreferences,
      preset: 'paper-flat',
      scheme: 'dark',
      density: 'comfortable',
      isCustomized: false,
    });

    expect(normalized.preset).toBe('paper-flat');
    expect(normalized.scheme).toBe('light');
    expect(normalized.density).toBe('comfortable');
    expect(resolveThemeStyleVariables(normalized, 'light')['--nx-bg']).toBe('#f4f5f3');
  });

  it('keeps font and icon settings independent from the theme preset and scheme', () => {
    const prefs = normalizeThemePreferences({
      ...defaultThemePreferences,
      preset: 'ember-studio',
      scheme: 'light',
      fontFamily: 'readable',
      iconStyle: 'bold',
      isCustomized: false,
    });
    expect(prefs.preset).toBe('ember-studio');
    expect(prefs.scheme).toBe('dark');
    expect(prefs.fontFamily).toBe('readable');
    expect(prefs.iconStyle).toBe('bold');
    expect(THEME_PRESETS['ember-studio'].finish).toBe('flat');
  });

  it('repairs invalid font and icon settings from persisted storage', () => {
    const prefs = normalizeThemePreferences({
      fontFamily: 'remote-url',
      iconStyle: 'image-html',
    } as unknown as ThemePreferences);
    expect(prefs.fontFamily).toBe('default');
    expect(prefs.iconStyle).toBe('default');
  });

  it('provides defaults for legacy preferences saved before font/icon selectors', () => {
    const prefs = normalizeThemePreferences({
      ...defaultThemePreferences,
      fontFamily: undefined,
      iconStyle: undefined,
    });
    expect(prefs.fontFamily).toBe('default');
    expect(prefs.iconStyle).toBe('default');
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
    expect(vars['--nx-subtle']).toBe('#667085');
  });

  it('aplica variáveis nativas do preset quando isCustomized é false', () => {
    const vars = resolveThemeStyleVariables(
      {
        ...defaultThemePreferences,
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
        ...defaultThemePreferences,
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
    expect(vars['--nx-accent-text']).toBe('#6d28d9');
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

describe('WCAG AA contrast automation for every registered preset', () => {
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

  it('keeps accent text readable on application and panel surfaces', () => {
    for (const [id, def] of Object.entries(THEME_PRESETS)) {
      for (const background of ['--nx-bg', '--nx-surface'] as const) {
        expect(
          contrastRatio(def.vars['--nx-accent-text'], def.vars[background]),
          `Preset ${id} accent-text/${background} contrast must be >= 4.5:1`,
        ).toBeGreaterThanOrEqual(4.5);
      }
    }
  });

  it('keeps secondary text readable on both application and panel surfaces', () => {
    for (const [id, def] of Object.entries(THEME_PRESETS)) {
      const bg = def.vars['--nx-bg'];
      const surface = def.vars['--nx-surface'];

      for (const token of ['--nx-muted', '--nx-subtle'] as const) {
        expect(
          contrastRatio(def.vars[token], bg),
          `Preset ${id} ${token}/bg contrast must be >= 4.5:1`,
        ).toBeGreaterThanOrEqual(4.5);
        expect(
          contrastRatio(def.vars[token], surface),
          `Preset ${id} ${token}/surface contrast must be >= 4.5:1`,
        ).toBeGreaterThanOrEqual(4.5);
      }
    }
  });

  it('keeps semantic status text readable on every preset surface', () => {
    for (const [id, def] of Object.entries(THEME_PRESETS)) {
      const vars = resolveThemeStyleVariables(
        {
          ...defaultThemePreferences,
          preset: id as ThemePresetKey,
          scheme: def.scheme,
          isCustomized: false,
        },
        def.scheme,
      );

      for (const token of ['--nx-success', '--nx-warning', '--nx-danger', '--nx-info'] as const) {
        expect(
          contrastRatio(vars[token], vars['--nx-surface']),
          `Preset ${id} ${token}/surface contrast must be >= 4.5:1`,
        ).toBeGreaterThanOrEqual(4.5);
      }

      expect(
        contrastRatio(vars['--nx-warning'], vars['--nx-warning-contrast']),
        `Preset ${id} warning/control text contrast must be >= 4.5:1`,
      ).toBeGreaterThanOrEqual(4.5);
    }
  });

  it('derives readable text for accent-filled controls in every preset', () => {
    for (const [id, def] of Object.entries(THEME_PRESETS)) {
      const vars = resolveThemeStyleVariables(
        {
          ...defaultThemePreferences,
          preset: id as ThemePresetKey,
          scheme: def.scheme,
          isCustomized: false,
        },
        def.scheme,
      );

      expect(
        contrastRatio(vars['--nx-accent'], vars['--nx-accent-contrast']),
        `Preset ${id} accent/control text contrast must be >= 4.5:1`,
      ).toBeGreaterThanOrEqual(4.5);
      expect(
        contrastRatio(vars['--nx-accent-hover'], vars['--nx-accent-contrast']),
        `Preset ${id} accent-hover/control text contrast must be >= 4.5:1`,
      ).toBeGreaterThanOrEqual(4.5);
    }
  });

  it('keeps custom accent overrides readable in every explicit scheme', () => {
    for (const scheme of ['dark', 'light', 'high-contrast'] as const) {
      for (const accent of ['purple', 'blue', 'cyan', 'neutral'] as const) {
        const vars = resolveThemeStyleVariables(
          {
            ...defaultThemePreferences,
            preset: 'nexus-dark',
            scheme,
            accent,
            isCustomized: true,
          },
          scheme,
        );

        expect(
          contrastRatio(vars['--nx-accent'], vars['--nx-accent-contrast']),
          `${scheme}/${accent} accent/control text contrast must be >= 4.5:1`,
        ).toBeGreaterThanOrEqual(4.5);
        expect(
          contrastRatio(vars['--nx-accent-hover'], vars['--nx-accent-contrast']),
          `${scheme}/${accent} accent-hover/control text contrast must be >= 4.5:1`,
        ).toBeGreaterThanOrEqual(4.5);
        for (const background of ['--nx-bg', '--nx-surface'] as const) {
          expect(
            contrastRatio(vars['--nx-accent-text'], vars[background]),
            `${scheme}/${accent} accent-text/${background} contrast must be >= 4.5:1`,
          ).toBeGreaterThanOrEqual(4.5);
        }
      }
    }
  });

  it('keeps High Contrast Light semantic status text readable on its surface', () => {
    const definition = THEME_PRESETS['high-contrast-light'];
    const surface = definition.vars['--nx-surface'];

    for (const token of ['--nx-success', '--nx-warning', '--nx-danger', '--nx-info'] as const) {
      expect(
        contrastRatio(definition.vars[token], surface),
        `High Contrast Light ${token} must be >= 4.5:1 on surface`,
      ).toBeGreaterThanOrEqual(4.5);
    }
  });
});
