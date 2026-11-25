import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';
import { ptBR } from './locales/pt-BR';

export const supportedLanguages = ['pt-BR', 'en', 'es'] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];
export const languageStorageKey = 'iapro:nexus:language:v1';

export function normalizeLanguage(value?: string | null): SupportedLanguage {
  const tag = (value || '').trim().replace('_', '-').toLowerCase();
  if (tag === 'pt' || tag.startsWith('pt-')) return 'pt-BR';
  if (tag === 'es' || tag.startsWith('es-')) return 'es';
  if (tag === 'en' || tag.startsWith('en-')) return 'en';
  return 'en';
}

const savedLanguage =
  typeof window !== 'undefined'
    ? window.localStorage?.getItem(languageStorageKey) ||
      window.localStorage?.getItem('iapro:nexus:lang:v1')
    : null;

const initialLanguage = savedLanguage ? normalizeLanguage(savedLanguage) : 'pt-BR';

const loadLocale = async (language: SupportedLanguage): Promise<void> => {
  if (i18n.hasResourceBundle(language, 'translation')) {
    return;
  }
  if (language === 'en') {
    const mod = await import('./locales/en');
    i18n.addResourceBundle('en', 'translation', mod.en, true, true);
    return;
  }
  if (language === 'es') {
    const mod = await import('./locales/es');
    i18n.addResourceBundle('es', 'translation', mod.es, true, true);
  }
};

const initialization = i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: { 'pt-BR': { translation: ptBR } },
    supportedLngs: [...supportedLanguages],
    fallbackLng: 'pt-BR',
    lng: initialLanguage,
    load: 'currentOnly',
    // React auto-escapes interpolated values, so escapeValue must remain false
    // to avoid double-escaping (e.g., "&amp;" rendered literally in the UI).
    // All `t()` calls pass through React's JSX rendering which handles XSS.
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage'],
      lookupLocalStorage: languageStorageKey,
      caches: ['localStorage'],
      convertDetectedLanguage: normalizeLanguage,
    },
    react: { useSuspense: false },
  });

const originalChangeLanguage = i18n.changeLanguage.bind(i18n);
i18n.changeLanguage = async (lng, callback) => {
  const target = normalizeLanguage(typeof lng === 'string' ? lng : initialLanguage);
  await loadLocale(target);
  return originalChangeLanguage(target, callback);
};

const syncDocumentLanguage = (language: string) => {
  const normalized = normalizeLanguage(language);
  if (typeof document !== 'undefined') {
    document.documentElement.lang = normalized;
    document.documentElement.dir = 'ltr';
  }
  if (!i18n.hasResourceBundle(normalized, 'translation')) {
    void loadLocale(normalized).then(() => originalChangeLanguage(normalized));
    return;
  }
  if (language !== normalized) void originalChangeLanguage(normalized);
};

i18n.on('languageChanged', syncDocumentLanguage);
void initialization.then(async () => {
  if (initialLanguage !== 'pt-BR') {
    await loadLocale(initialLanguage);
    await originalChangeLanguage(initialLanguage);
  }
  syncDocumentLanguage(i18n.language);
});

export function translateStatus(status?: string): string {
  if (!status) return i18n.t('common.unknown');
  const key = `status.${status.toUpperCase()}`;
  return i18n.exists(key) ? i18n.t(key) : status;
}

export default i18n;
