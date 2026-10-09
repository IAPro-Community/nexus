import React, {
  createContext,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
} from 'react';
import {
  defaultThemePreferences,
  MANAGED_THEME_COLOR_VARS,
  normalizeThemePreferences,
  resolveScheme,
  resolveThemeStyleVariables,
  themeStorageKey,
  type ThemeAccent,
  type ThemeDensity,
  type ThemePreferences,
  type ThemePresetKey,
  type ThemeScheme,
  THEME_PRESETS,
} from './theme';

interface ThemeContextValue extends ThemePreferences {
  resolvedScheme: 'dark' | 'light' | 'high-contrast';
  setScheme: (scheme: ThemeScheme) => void;
  setAccent: (accent: ThemeAccent) => void;
  setDensity: (density: ThemeDensity) => void;
  setReducedMotion: (value: boolean) => void;
  setPreset: (preset: ThemePresetKey) => void;
  setFontScale: (scale: number) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

function loadPreferences(): ThemePreferences {
  if (typeof window === 'undefined') return defaultThemePreferences;
  try {
    const raw = JSON.parse(window.localStorage.getItem(themeStorageKey) || 'null') || {};
    if (typeof raw.fontScale !== 'number') {
      const legacy = parseFloat(window.localStorage.getItem('iapro:nexus:font-scale') || '');
      if (Number.isFinite(legacy) && legacy >= 0.7 && legacy <= 2.0) {
        raw.fontScale = legacy;
      }
    }
    return normalizeThemePreferences(raw);
  } catch {
    return defaultThemePreferences;
  }
}

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [preferences, setPreferences] = useState<ThemePreferences>(loadPreferences);
  const [systemDark, setSystemDark] = useState(
    () =>
      typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches,
  );

  useEffect(() => {
    const media = window.matchMedia?.('(prefers-color-scheme: dark)');
    if (!media) return;
    const update = () => setSystemDark(media.matches);
    media.addEventListener?.('change', update);
    return () => media.removeEventListener?.('change', update);
  }, []);

  const resolvedScheme = resolveScheme(preferences.scheme, systemDark);

  useEffect(() => {
    window.localStorage.setItem(themeStorageKey, JSON.stringify(preferences));
  }, [preferences]);

  useLayoutEffect(() => {
    const root = document.documentElement;
    root.dataset.theme = resolvedScheme;
    root.dataset.accent = preferences.accent;
    root.dataset.density = preferences.density;
    // The flat visual treatment is opt-in per preset; legacy themes stay unchanged.
    // High-contrast overrides are never flattened.
    root.dataset.finish =
      resolvedScheme === 'high-contrast'
        ? 'standard'
        : THEME_PRESETS[preferences.preset]?.finish || 'standard';
    root.dataset.reducedMotion = preferences.reducedMotion ? 'true' : 'false';
    const scale = preferences.fontScale || 1.0;
    root.dataset.fontScale = String(scale);
    root.style.setProperty('--nx-font-scale', String(scale));
    root.style.colorScheme = resolvedScheme === 'light' ? 'light' : 'dark';

    // Limpa todas as variáveis CSS gerenciadas para evitar vazamento entre presets e customizações
    for (const v of MANAGED_THEME_COLOR_VARS) {
      root.style.removeProperty(v);
    }

    if (preferences.preset && THEME_PRESETS[preferences.preset]) {
      root.dataset.preset = preferences.preset;
      const dynamicVars = resolveThemeStyleVariables(preferences, resolvedScheme);
      for (const [varName, varVal] of Object.entries(dynamicVars)) {
        root.style.setProperty(varName, varVal);
      }
    } else {
      delete root.dataset.preset;
    }
  }, [preferences, resolvedScheme]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      ...preferences,
      resolvedScheme,
      setScheme: (scheme) =>
        setPreferences((current) => ({ ...current, scheme, isCustomized: true })),
      setAccent: (accent) =>
        setPreferences((current) => ({ ...current, accent, isCustomized: true })),
      setDensity: (density) => setPreferences((current) => ({ ...current, density })),
      setReducedMotion: (reducedMotion) =>
        setPreferences((current) => ({ ...current, reducedMotion })),
      setPreset: (preset) =>
        setPreferences((current) => {
          const def = THEME_PRESETS[preset];
          return {
            ...current,
            preset,
            scheme: def ? def.scheme : current.scheme,
            isCustomized: false, // Ao trocar de preset, restaura fidelidade total ao preset
          };
        }),
      setFontScale: (scale) => {
        const clamped = Math.min(Math.max(scale, 0.7), 2.0);
        setPreferences((current) => ({ ...current, fontScale: clamped }));
        if (typeof window !== 'undefined') {
          localStorage.setItem('iapro:nexus:font-scale', String(clamped));
        }
      },
    }),
    [preferences, resolvedScheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
};

export function useTheme(): ThemeContextValue {
  const value = useContext(ThemeContext);
  if (!value) throw new Error('useTheme must be used inside ThemeProvider');
  return value;
}
