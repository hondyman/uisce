import React from 'react';
import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom';
import { useEffect } from 'react';
import {
  ACTIVE_LOCALES,
  DEFAULT_LOCALE,
  Locale,
  getPreferredLocale,
  resolveLocaleRoute,
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

function getBrowserLanguages(): readonly string[] {
  if (typeof navigator === 'undefined') return [];
  return navigator.languages ?? [navigator.language];
}

function getCachedLocale(): string | null {
  if (typeof localStorage === 'undefined') return null;
  return localStorage.getItem('appLocale');
}

function RootRedirect() {
  const { search, hash } = useLocation();
  const target = getPreferredLocale(getCachedLocale(), getBrowserLanguages());
  return <Navigate to={`/${target}${search}${hash}`} replace />;
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

  // Single source of truth for "what should the shell do with this URL?".
  // All branches of locale-shell.test.ts verify this function's output.
  const preferred: Locale = getPreferredLocale(
    getCachedLocale(),
    getBrowserLanguages(),
  ) ?? DEFAULT_LOCALE;

  const route = resolveLocaleRoute(pathname, search, hash, preferred);

  if (route.kind === 'redirect') {
    return <Navigate to={route.to} replace />;
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
