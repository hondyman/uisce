import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import ICU from 'i18next-icu';
import en from './locales/en.json';
import { DEFAULT_LOCALE, LOCALES, isRtl, normalizeLocale } from './i18n/locales';

// One-time migration from the previous localStorage key — must run BEFORE
// LanguageDetector reads localStorage during init.
try {
  if (typeof localStorage !== 'undefined') {
    const legacy = localStorage.getItem('selected_language');
    if (legacy && !localStorage.getItem('appLocale')) {
      localStorage.setItem('appLocale', legacy);
      localStorage.removeItem('selected_language');
    }
  }
} catch {
  /* SSR / locked-down storage */
}

// Resolve initial language from the URL before first paint to prevent
// a flash of English on /ja or /ar.
const urlLocale =
  typeof window !== 'undefined'
    ? normalizeLocale(window.location.pathname.split('/')[1])
    : null;

i18n
  .use(LanguageDetector)
  .use(ICU)
  .use({
    type: 'backend',
    read: (lng: string, _ns: string, cb: (err: unknown, data: unknown) => void) =>
      import(`./locales/${lng}.json`)
        .then((m) => cb(null, m.default))
        .catch((e: unknown) => cb(e, null)),
  })
  .use(initReactI18next)
  .init({
    lng: urlLocale ?? undefined,
    resources: { en: { translation: en } },
    partialBundledLanguages: true,
    fallbackLng: DEFAULT_LOCALE,
    supportedLngs: [...LOCALES],
    nonExplicitSupportedLngs: true,
    load: 'currentOnly',
    defaultNS: 'translation',
    ns: ['translation'],
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      lookupLocalStorage: 'appLocale',
    },
    returnNull: false,
  });

// WCAG 3.1.1 / 3.1.2: keep <html lang> and dir in sync for screen readers.
i18n.on('languageChanged', (lng: string) => {
  if (typeof document === 'undefined') return;
  const locale = normalizeLocale(lng) ?? DEFAULT_LOCALE;
  document.documentElement.lang = locale;
  document.documentElement.dir = isRtl(locale) ? 'rtl' : 'ltr';
});

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'translation';
    resources: { translation: typeof en };
  }
}

export default i18n;
