// Locales the project ships translations for today (file present + values
// reviewed). Skeleton locales (de/pt-BR/ja/zh-CN/ar) exist as files but
// contain only English defaults — they are NOT in this set. Automatic
// locale detection (RootRedirect) restricts itself to ACTIVE_LOCALES
// so a ja-JP browser doesn't land on /ja and render English.
export const ACTIVE_LOCALES: readonly Locale[] = ['en', 'es', 'fr'];

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

/** Match the first candidate against LOCALES. May return a skeleton locale. */
export function matchLocale(candidates: readonly string[]): Locale {
  for (const c of candidates) {
    const hit = normalizeLocale(c);
    if (hit) return hit;
  }
  return DEFAULT_LOCALE;
}

/**
 * Match the first candidate against ACTIVE_LOCALES (real translations).
 * Use this for automatic detection where landing on a skeleton locale
 * (English fallback) is a worse outcome than landing on DEFAULT_LOCALE.
 */
export function matchActiveLocale(
  candidates: readonly string[],
  active: readonly Locale[] = ACTIVE_LOCALES,
): Locale {
  for (const c of candidates) {
    const hit = normalizeLocale(c);
    if (hit && active.includes(hit)) return hit;
  }
  return DEFAULT_LOCALE;
}

/**
 * Where to send an un-prefixed URL.
 *
 * Order of preference:
 *   1. Cached appLocale from localStorage — but ONLY if it's in ACTIVE_LOCALES.
 *      An explicit visit to /ja sets this to 'ja'; we must not honor that
 *      for the heal / redirect path, or a stale cache silently bypasses the
 *      ACTIVE_LOCALES gate. (Explicit URL prefixes still work via the
 *      LocaleLayout branch.)
 *   2. Best match from `candidates` (typically `navigator.languages`)
 *      against ACTIVE_LOCALES.
 *   3. DEFAULT_LOCALE.
 *
 * Pure: pass `cached` and `candidates` explicitly so this is unit-testable
 * without localStorage or navigator.
 */
export function getPreferredLocale(
  cached: string | null,
  candidates: readonly string[],
  active: readonly Locale[] = ACTIVE_LOCALES,
): Locale {
  const cachedLocale = normalizeLocale(cached);
  if (cachedLocale && active.includes(cachedLocale)) return cachedLocale;
  const matched = matchActiveLocale(candidates, active);
  return matched ?? DEFAULT_LOCALE;
}

export function localePath(locale: Locale, rest = '/'): string {
  const suffix = rest === '/' ? '' : rest.startsWith('/') ? rest : `/${rest}`;
  return `/${locale}${suffix}`;
}

/** Strip the optional `/{locale}` prefix from a path. */
export function stripLocale(pathname: string): string {
  const m = pathname.match(/^\/([^/]+)((?:\/.*)?)$/);
  if (!m || !normalizeLocale(m[1])) return pathname;
  return m[2] === '' ? '/' : m[2];
}

/**
 * Decide what the locale shell should do with a pathname. This is the
 * SINGLE pure function the shell calls — testing it IS testing the shell.
 *
 * Three branches:
 *   1. No locale prefix → heal to preferred locale, KEEP the full pathname
 *      so search/hash preservation upstream composes correctly. Preferred
 *      defaults to matchActiveLocale (en/es/fr only) so a ja-JP browser
 *      lands on /en, not /ja.
 *   2. Locale prefix is non-canonical (e.g. /ES/dashboard) → redirect to
 *      the canonical lower-case form. The rest is the path after the
 *      first segment (regex-stripped, NOT the original pathname).
 *   3. Locale prefix is canonical → render the locale (no redirect).
 */
export type LocaleRoute =
  | { kind: 'ok'; locale: Locale }
  | { kind: 'redirect'; to: string };

export function resolveLocaleRoute(
  pathname: string,
  search = '',
  hash = '',
  preferred: Locale = DEFAULT_LOCALE,
): LocaleRoute {
  const raw = pathname.split('/')[1] ?? '';
  const locale = normalizeLocale(raw);

  if (!locale) {
    // No (or invalid) locale prefix → heal with the FULL pathname. The
    // shell composes `${to}${search}${hash}` upstream so search/hash survive.
    return {
      kind: 'redirect',
      to: localePath(preferred, pathname) + search + hash,
    };
  }

  if (raw !== locale) {
    // Non-canonical prefix (e.g. "/ES/" or "/PT-BR/") → redirect to
    // canonical, stripping only the first segment.
    const rest = pathname.replace(/^\/[^/]+/, '') || '/';
    return {
      kind: 'redirect',
      to: localePath(locale, rest) + search + hash,
    };
  }

  return { kind: 'ok', locale };
}
