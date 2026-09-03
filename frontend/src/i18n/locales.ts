export const LOCALES = ['en', 'es', 'fr', 'de', 'pt-BR', 'ja', 'zh-CN', 'ar'] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = 'en';

const RTL = new Set(['ar', 'he', 'fa']);
export const isRtl = (l?: string): boolean => RTL.has((l ?? '').toLowerCase());

export const NATIVE_NAMES: Record<Locale, string> = {
  en: 'English',
  es: 'Español',
  fr: 'Français',
  de: 'Deutsch',
  'pt-BR': 'Português (Brasil)',
  ja: '日本語',
  'zh-CN': '简体中文',
  ar: 'العربية',
};

export function normalizeLocale(raw?: string | null): Locale | null {
  if (!raw) return null;
  const lower = raw.toLowerCase();
  return (
    LOCALES.find((l) => l.toLowerCase() === lower) ??
    LOCALES.find((l) => l.toLowerCase().split('-')[0] === lower.split('-')[0]) ??
    null
  );
}

export function matchLocale(candidates: readonly string[]): Locale {
  for (const c of candidates) {
    const hit = normalizeLocale(c);
    if (hit) return hit;
  }
  return DEFAULT_LOCALE;
}

export function localePath(locale: Locale, rest = '/'): string {
  const suffix = rest === '/' ? '' : rest.startsWith('/') ? rest : `/${rest}`;
  return `/${locale}${suffix}`;
}

/** Strip the optional `/{locale}` prefix from a path. */
export function stripLocale(pathname: string): string {
  const stripped = pathname.replace(/^\/[^/]+/, '');
  return stripped === '' ? '/' : stripped;
}
