import { useLocation } from 'react-router-dom';
import i18n from '../i18n';
import { DEFAULT_LOCALE, Locale, normalizeLocale } from './locales';

/**
 * Locale is the source of truth from the URL, falling back to the i18n
 * instance (which mirrors the URL via LocaleSync), then to the default.
 *
 * Reads from useLocation (not useParams) because:
 * - MainNavigation is rendered outside the Route tree in App.tsx, so
 *   useParams() returns undefined there.
 * - useLocation re-renders on every navigation, so callers always see
 *   the current locale.
 */
export function useLocale(): Locale {
  const { pathname } = useLocation();
  return (
    normalizeLocale(pathname.split('/')[1]) ??
    normalizeLocale(i18n.language) ??
    DEFAULT_LOCALE
  );
}
