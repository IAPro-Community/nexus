import { THEME_PRESETS, type ThemeDefinition } from './themePresets';

export type ThemeScheme = 'system' | 'dark' | 'light' | 'high-contrast';
export type ThemeAccent = 'purple' | 'blue' | 'cyan' | 'neutral';
export type ThemeDensity = 'compact' | 'comfortable';
export type ThemeVisualFinish = 'flat' | 'standard' | 'subtle-gradient';
export type ThemeFontFamily = 'default' | 'system' | 'readable';
export type ThemeIconStyle = 'default' | 'light' | 'bold';
export type ThemePresetKey = keyof typeof THEME_PRESETS;

/** Legacy preset finish metadata. User preference now selects finish independently. */
export function getThemePresetFinish(
  preset: ThemePresetKey,
): 'flat' | 'subtle-gradient' | 'standard' {
  const definition: ThemeDefinition = THEME_PRESETS[preset];
  return definition.finish ?? 'standard';
}

export interface ThemePreferences {
  scheme: ThemeScheme;
  accent: ThemeAccent;
  density: ThemeDensity;
  visualFinish: ThemeVisualFinish;
  reducedMotion: boolean;
  preset: ThemePresetKey;
  isCustomized: boolean;
  fontScale: number;
  fontFamily: ThemeFontFamily;
  iconStyle: ThemeIconStyle;
}

export const themeStorageKey = 'iapro:nexus:theme:v1';
export const defaultThemePreferences: ThemePreferences = {
  scheme: 'system',
  accent: 'blue',
  density: 'compact',
  visualFinish: 'flat',
  reducedMotion: false,
  preset: 'nexus-dark',
  isCustomized: false,
  fontScale: 1.0,
  fontFamily: 'default',
  iconStyle: 'default',
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
  '--nx-warning-contrast',
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
    '--nx-bg': '#141414',
    '--nx-bg-elevated': '#1f1f1f',
    '--nx-surface': '#292929',
    '--nx-surface-2': '#333333',
    '--nx-surface-3': '#3d3d3d',
    '--nx-border': '#525252',
    '--nx-border-strong': '#737373',
    '--nx-text': '#f5f5f5',
    '--nx-text-soft': '#d6d6d6',
    '--nx-muted': '#adadad',
    '--nx-subtle': '#949494',
    '--nx-success': '#6ccb8a',
    '--nx-success-soft': 'rgba(108, 203, 138, 0.14)',
    '--nx-warning': '#fce100',
    '--nx-warning-soft': 'rgba(252, 225, 0, 0.14)',
    '--nx-danger': '#ff9090',
    '--nx-danger-soft': 'rgba(255, 144, 144, 0.12)',
    '--nx-info': '#62b5f5',
    '--nx-info-soft': 'rgba(98, 181, 245, 0.14)',
    '--nx-accent': '#2879b8',
    '--nx-accent-hover': '#2676b5',
    '--nx-accent-soft': 'rgba(40, 121, 184, 0.18)',
    '--nx-accent-text': '#62b5f5',
  },
  light: {
    '--nx-bg': '#f5f5f5',
    '--nx-bg-elevated': '#ffffff',
    '--nx-surface': '#ffffff',
    '--nx-surface-2': '#f5f5f5',
    '--nx-surface-3': '#e6e6e6',
    '--nx-border': '#d1d1d1',
    '--nx-border-strong': '#737373',
    '--nx-text': '#242424',
    '--nx-text-soft': '#424242',
    '--nx-muted': '#616161',
    '--nx-subtle': '#707070',
    '--nx-success': '#0e7a0d',
    '--nx-success-soft': 'rgba(14, 122, 13, 0.1)',
    '--nx-warning': '#806300',
    '--nx-warning-soft': 'rgba(128, 99, 0, 0.1)',
    '--nx-danger': '#c50f1f',
    '--nx-danger-soft': 'rgba(197, 15, 31, 0.1)',
    '--nx-info': '#0f6cbd',
    '--nx-info-soft': 'rgba(15, 108, 189, 0.1)',
    '--nx-accent': '#0f6cbd',
    '--nx-accent-hover': '#115ea3',
    '--nx-accent-soft': 'rgba(15, 108, 189, 0.1)',
    '--nx-accent-text': '#0f548c',
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
    '--nx-accent': '#2779bd',
    '--nx-accent-hover': '#2676b5',
    '--nx-accent-soft': 'rgba(39, 121, 189, 0.16)',
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

const ACCENT_TEXT_BY_SCHEME: Record<Exclude<ThemeScheme, 'system'>, Record<ThemeAccent, string>> = {
  dark: {
    purple: '#bba5ff',
    blue: '#62b5f5',
    cyan: '#70dbe5',
    neutral: '#c1c7d1',
  },
  light: {
    purple: '#6d28d9',
    blue: '#0f548c',
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

function resolveReadableContrast(background: string): string {
  const black = '#000000';
  const white = '#ffffff';
  return colorContrastRatio(background, black) >= colorContrastRatio(background, white)
    ? black
    : white;
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

  finalVars['--nx-accent-contrast'] = resolveReadableContrast(
    finalVars['--nx-accent'] || preset.accent,
  );
  finalVars['--nx-warning-contrast'] = resolveReadableContrast(finalVars['--nx-warning']);

  return finalVars;
}

/** The finish is independent of palette and density, with contrast mode protected. */
export function resolveVisualFinish(
  preferences: ThemePreferences,
  resolvedScheme: Exclude<ThemeScheme, 'system'>,
): ThemeVisualFinish {
  return resolvedScheme === 'high-contrast' ? 'standard' : preferences.visualFinish;
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
  const visualFinishes: ThemeVisualFinish[] = ['flat', 'standard', 'subtle-gradient'];
  const fontFamilies: ThemeFontFamily[] = ['default', 'system', 'readable'];
  const iconStyles: ThemeIconStyle[] = ['default', 'light', 'bold'];
  const presetKeys = Object.keys(THEME_PRESETS) as ThemePresetKey[];

  const preset =
    input?.preset && presetKeys.includes(input.preset as ThemePresetKey)
      ? (input.preset as ThemePresetKey)
      : defaultThemePreferences.preset;
  const isCustomized = Boolean(input?.isCustomized);
  const requestedScheme = schemes.includes(input?.scheme as ThemeScheme)
    ? input!.scheme!
    : defaultThemePreferences.scheme;
  const scheme =
    !isCustomized && !(preset === defaultThemePreferences.preset && requestedScheme === 'system')
      ? THEME_PRESETS[preset].scheme
      : requestedScheme;

  return {
    scheme,
    accent: accents.includes(input?.accent as ThemeAccent)
      ? input!.accent!
      : defaultThemePreferences.accent,
    density: densities.includes(input?.density as ThemeDensity)
      ? input!.density!
      : defaultThemePreferences.density,
    visualFinish: visualFinishes.includes(input?.visualFinish as ThemeVisualFinish)
      ? input!.visualFinish!
      : defaultThemePreferences.visualFinish,
    reducedMotion: Boolean(input?.reducedMotion),
    preset,
    isCustomized,
    fontFamily: fontFamilies.includes(input?.fontFamily as ThemeFontFamily)
      ? input!.fontFamily!
      : defaultThemePreferences.fontFamily,
    iconStyle: iconStyles.includes(input?.iconStyle as ThemeIconStyle)
      ? input!.iconStyle!
      : defaultThemePreferences.iconStyle,
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
