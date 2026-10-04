import { THEME_PRESETS, type ThemeDefinition } from './themePresets';

export type ThemeScheme = 'system' | 'dark' | 'light' | 'high-contrast';
export type ThemeAccent = 'purple' | 'blue' | 'cyan' | 'neutral';
export type ThemeDensity = 'compact' | 'comfortable';
export type ThemePresetKey = keyof typeof THEME_PRESETS;

export interface ThemePreferences {
  scheme: ThemeScheme;
  accent: ThemeAccent;
  density: ThemeDensity;
  reducedMotion: boolean;
  preset: ThemePresetKey;
  isCustomized: boolean;
  fontScale: number;
}

export const themeStorageKey = 'iapro:nexus:theme:v1';
export const defaultThemePreferences: ThemePreferences = {
  scheme: 'system',
  accent: 'purple',
  density: 'compact',
  reducedMotion: false,
  preset: 'nexus-dark',
  isCustomized: false,
  fontScale: 1.0,
};

export const MANAGED_THEME_COLOR_VARS = [
  '--nx-bg',
  '--nx-bg-elevated',
  '--nx-surface',
  '--nx-surface-2',
  '--nx-surface-3',
  '--nx-border',
  '--nx-border-strong',
  '--nx-text',
  '--nx-text-soft',
  '--nx-muted',
  '--nx-subtle',
  '--nx-accent',
  '--nx-accent-hover',
  '--nx-accent-soft',
  '--nx-accent-text',
  '--nx-accent-contrast',
  '--nx-window-accent',
  '--nx-success',
  '--nx-success-soft',
  '--nx-warning',
  '--nx-warning-soft',
  '--nx-danger',
  '--nx-danger-soft',
  '--nx-info',
  '--nx-info-soft',
] as const;

export const SCHEME_OVERRIDE_VARS: Record<
  Exclude<ThemeScheme, 'system'>,
  Record<string, string>
> = {
  dark: {
    '--nx-bg': '#080a0f',
    '--nx-bg-elevated': '#0c0f15',
    '--nx-surface': '#10131a',
    '--nx-surface-2': '#151922',
    '--nx-surface-3': '#1a1f2b',
    '--nx-border': '#202633',
    '--nx-border-strong': '#303849',
    '--nx-text': '#f1f3f7',
    '--nx-text-soft': '#d9dde6',
    '--nx-muted': '#8a93a5',
    '--nx-subtle': '#8b95a8',
    '--nx-success': '#51cf9a',
    '--nx-success-soft': 'rgba(81, 207, 154, 0.12)',
    '--nx-warning': '#f3bd67',
    '--nx-warning-soft': 'rgba(243, 189, 103, 0.12)',
    '--nx-danger': '#ff9090',
    '--nx-danger-soft': 'rgba(255, 144, 144, 0.12)',
    '--nx-info': '#62b4ff',
    '--nx-info-soft': 'rgba(98, 180, 255, 0.12)',
  },
  light: {
    '--nx-bg': '#f4f6fa',
    '--nx-bg-elevated': '#ffffff',
    '--nx-surface': '#ffffff',
    '--nx-surface-2': '#f0f3f8',
    '--nx-surface-3': '#e8edf4',
    '--nx-border': '#d7dde8',
    '--nx-border-strong': '#bcc5d5',
    '--nx-text': '#141821',
    '--nx-text-soft': '#2d3442',
    '--nx-muted': '#5e687b',
    '--nx-subtle': '#667085',
    '--nx-success': '#166534',
    '--nx-success-soft': 'rgba(22, 101, 52, 0.1)',
    '--nx-warning': '#92400e',
    '--nx-warning-soft': 'rgba(146, 64, 14, 0.1)',
    '--nx-danger': '#b91c1c',
    '--nx-danger-soft': 'rgba(185, 28, 28, 0.1)',
    '--nx-info': '#1d4ed8',
    '--nx-info-soft': 'rgba(29, 78, 216, 0.1)',
  },
  'high-contrast': {
    '--nx-bg': '#000000',
    '--nx-bg-elevated': '#000000',
    '--nx-surface': '#050505',
    '--nx-surface-2': '#0b0b0b',
    '--nx-surface-3': '#111111',
    '--nx-border': '#8f8f8f',
    '--nx-border-strong': '#ffffff',
    '--nx-text': '#ffffff',
    '--nx-text-soft': '#ffffff',
    '--nx-muted': '#e2e2e2',
    '--nx-subtle': '#c7c7c7',
    '--nx-success': '#5fff93',
    '--nx-success-soft': 'rgba(95, 255, 147, 0.18)',
    '--nx-warning': '#ffd75f',
    '--nx-warning-soft': 'rgba(255, 215, 95, 0.18)',
    '--nx-danger': '#ff7676',
    '--nx-danger-soft': 'rgba(255, 118, 118, 0.18)',
    '--nx-info': '#75d5ff',
    '--nx-info-soft': 'rgba(117, 213, 255, 0.18)',
  },
};

export const ACCENT_OVERRIDE_VARS: Record<ThemeAccent, Record<string, string>> = {
  purple: {
    '--nx-accent': '#8b5cf6',
    '--nx-accent-hover': '#9f75ff',
    '--nx-accent-soft': 'rgba(139, 92, 246, 0.14)',
  },
  blue: {
    '--nx-accent': '#4285ff',
    '--nx-accent-hover': '#6399ff',
    '--nx-accent-soft': 'rgba(66, 133, 255, 0.14)',
  },
  cyan: {
    '--nx-accent': '#13b8c8',
    '--nx-accent-hover': '#30c9d8',
    '--nx-accent-soft': 'rgba(19, 184, 200, 0.14)',
  },
  neutral: {
    '--nx-accent': '#8993a4',
    '--nx-accent-hover': '#a4adbb',
    '--nx-accent-soft': 'rgba(137, 147, 164, 0.14)',
  },
};

const ACCENT_TEXT_BY_SCHEME: Record<
  Exclude<ThemeScheme, 'system'>,
  Record<ThemeAccent, string>
> = {
  dark: {
    purple: '#bba5ff',
    blue: '#8db5ff',
    cyan: '#70dbe5',
    neutral: '#c1c7d1',
  },
  light: {
    purple: '#6d28d9',
    blue: '#1d4ed8',
    cyan: '#0e7490',
    neutral: '#475569',
  },
  'high-contrast': {
    purple: '#bba5ff',
    blue: '#8db5ff',
    cyan: '#70dbe5',
    neutral: '#c1c7d1',
  },
};

function parseHexColor(value: string): [number, number, number] | null {
  const normalized = value.trim().replace('#', '');
  if (!/^[0-9a-fA-F]{6}$/.test(normalized)) return null;
  return [
    Number.parseInt(normalized.slice(0, 2), 16),
    Number.parseInt(normalized.slice(2, 4), 16),
    Number.parseInt(normalized.slice(4, 6), 16),
  ];
}

function relativeLuminance(value: string): number | null {
  const rgb = parseHexColor(value);
  if (!rgb) return null;

  const [r, g, b] = rgb.map((channel) => {
    const srgb = channel / 255;
    return srgb <= 0.03928 ? srgb / 12.92 : Math.pow((srgb + 0.055) / 1.055, 2.4);
  });

  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function colorContrastRatio(left: string, right: string): number {
  const leftLuminance = relativeLuminance(left);
  const rightLuminance = relativeLuminance(right);
  if (leftLuminance === null || rightLuminance === null) return 0;

  const lighter = Math.max(leftLuminance, rightLuminance);
  const darker = Math.min(leftLuminance, rightLuminance);
  return (lighter + 0.05) / (darker + 0.05);
}

function resolveAccentContrast(accent: string): string {
  const black = '#000000';
  const white = '#ffffff';
  return colorContrastRatio(accent, black) >= colorContrastRatio(accent, white) ? black : white;
}

export function resolveScheme(
  scheme: ThemeScheme,
  systemDark: boolean,
): Exclude<ThemeScheme, 'system'> {
  return scheme === 'system' ? (systemDark ? 'dark' : 'light') : scheme;
}

export function resolveThemeStyleVariables(
  prefs: ThemePreferences,
  resolvedScheme: Exclude<ThemeScheme, 'system'>,
): Record<string, string> {
  const presetKey = prefs.preset || defaultThemePreferences.preset;
  const preset = THEME_PRESETS[presetKey] || THEME_PRESETS['nexus-dark'];
  const finalVars: Record<string, string> = {
    ...SCHEME_OVERRIDE_VARS[preset.scheme],
    ...preset.vars,
  };

  if (prefs.isCustomized || prefs.scheme === 'system') {
    if (resolvedScheme !== preset.scheme) {
      Object.assign(finalVars, SCHEME_OVERRIDE_VARS[resolvedScheme] || {});
    }
    if (prefs.isCustomized) {
      Object.assign(finalVars, ACCENT_OVERRIDE_VARS[prefs.accent] || {});
      finalVars['--nx-accent-text'] = ACCENT_TEXT_BY_SCHEME[resolvedScheme][prefs.accent];
    }
  }

  finalVars['--nx-accent-contrast'] = resolveAccentContrast(
    finalVars['--nx-accent'] || preset.accent,
  );

  return finalVars;
}

export interface ThemeColorPalette {
  bg: string;
  surface: string;
  text: string;
  accent: string;
}

export function getThemePresetPalette(preset: ThemeDefinition): ThemeColorPalette {
  return {
    bg: preset.vars['--nx-bg'] || '#080a0f',
    surface: preset.vars['--nx-surface'] || '#10131a',
    text: preset.vars['--nx-text'] || '#f1f3f7',
    accent: preset.accent || '#8b5cf6',
  };
}

export function normalizeThemePreferences(
  input?: Partial<ThemePreferences> | null,
): ThemePreferences {
  const schemes: ThemeScheme[] = ['system', 'dark', 'light', 'high-contrast'];
  const accents: ThemeAccent[] = ['purple', 'blue', 'cyan', 'neutral'];
  const densities: ThemeDensity[] = ['compact', 'comfortable'];
  const presetKeys = Object.keys(THEME_PRESETS) as ThemePresetKey[];

  return {
    scheme: schemes.includes(input?.scheme as ThemeScheme)
      ? input!.scheme!
      : defaultThemePreferences.scheme,
    accent: accents.includes(input?.accent as ThemeAccent)
      ? input!.accent!
      : defaultThemePreferences.accent,
    density: densities.includes(input?.density as ThemeDensity)
      ? input!.density!
      : defaultThemePreferences.density,
    reducedMotion: Boolean(input?.reducedMotion),
    preset:
      input?.preset && presetKeys.includes(input.preset as ThemePresetKey)
        ? (input.preset as ThemePresetKey)
        : defaultThemePreferences.preset,
    isCustomized: Boolean(input?.isCustomized),
    fontScale:
      typeof input?.fontScale === 'number' &&
      Number.isFinite(input.fontScale) &&
      input.fontScale >= 0.7 &&
      input.fontScale <= 2.0
        ? input.fontScale
        : defaultThemePreferences.fontScale,
  };
}

export * from './themePresets';
