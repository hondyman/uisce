import React from 'react';
import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom';
import { useEffect } from 'react';
import {
  DEFAULT_LOCALE,
  Locale,
  localePath,
  matchLocale,
  normalizeLocale,
} from '../i18n/locales';
import i18n from '../i18n';
import { AppRoutes } from '../AppRoutes';

// Un-prefixed top-level pages. v6 ranks static routes above `/:locale/*`
// order-independent, so these always win for direct visits and OAuth callbacks.
import LoginPage from '../pages/AuthPage';
import AuthCallbackPage from '../pages/AuthCallbackPage';
import APIStudioPage from '../pages/api-studio/APIStudioPage';
import PageStudioPage from '../pages/page-studio/PageStudioPage';
import { PageRuntimeRenderer as RuntimePage } from '../pages/PageRuntimeRenderer';
import ChangeReviewPage from '../pages/ChangeReviewPage';

function RootRedirect() {
  const cached =
    typeof localStorage !== 'undefined' ? localStorage.getItem('appLocale') : null;
  const target =
    normalizeLocale(cached) ??
    matchLocale(navigator.languages ?? [navigator.language]);
  return <Navigate to={localePath(target)} replace />;
}

function LocaleSync() {
  const { locale } = useParams<{ locale: string }>();
  useEffect(() => {
    if (locale && locale !== i18n.language) {
      void i18n.changeLanguage(locale);
    }
  }, [locale]);
  return null;
}

export function LocaleLayout() {
  const { pathname, search, hash } = useLocation();
  const trailing = `${search}${hash}`;
  const m = pathname.match(/^\/([^/]+)(\/.*)?$/);
  const raw = m?.[1] ?? '';
  const rest = m?.[2] ?? '/';
  const locale = normalizeLocale(raw);

  // Un-prefixed or non-canonical first segment: heal to the user's preferred
  // locale. Preserves search + hash so auth callbacks, filters, and hash
  // anchors survive the round-trip.
  if (!locale) {
    const preferred =
      normalizeLocale(
        typeof localStorage !== 'undefined' ? localStorage.getItem('appLocale') : null,
      ) ?? matchLocale(navigator.languages ?? [navigator.language]);
    return <Navigate to={`${localePath(preferred, pathname)}${trailing}`} replace />;
  }
  if (raw !== locale) {
    return <Navigate to={`${localePath(locale, rest)}${trailing}`} replace />;
  }

  return (
    <>
      <LocaleSync />
      <AppRoutes />
    </>
  );
}

export function LocaleShell() {
  return (
    <Routes>
      {/* Canonical un-prefixed: auth flow + entry/embed surfaces */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/callback" element={<AuthCallbackPage />} />
      <Route path="/api-studio" element={<APIStudioPage />} />
      <Route path="/page-studio" element={<PageStudioPage />} />
      <Route path="/app/:slug" element={<RuntimePage />} />
      <Route path="/change-review" element={<ChangeReviewPage />} />
      {/* Root redirect → preferred locale home */}
      <Route path="/" element={<RootRedirect />} />
      {/* Locale-prefixed app: single splat, single descendant <Routes> inside AppRoutes */}
      <Route path="/:locale/*" element={<LocaleLayout />} />
    </Routes>
  );
}
