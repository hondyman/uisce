import React from 'react';
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom';
import {
  LEGACY_PAGE_SLUGS,
  legacyPageReplacement,
} from '../../pages/page-studio/legacyPageRedirects';

/**
 * The legacy Workflow Designer (core-workflow-designer) was retired in favour of Process
 * Designer (client-workflow-studio) by
 * backend/db/migrations/20261225_009_retire_legacy_workflow_designer.
 *
 * SlugPage in AppRoutes.tsx resolves the redirect through legacyPageReplacement before it looks
 * a page up, so /p/core-workflow-designer reaches Process Designer rather than PageBrowser's
 * "page is not in this environment yet". This file pins the lookup and the redirect itself; the
 * map is the only seam, because AppRoutes eagerly imports every page module in the app.
 */

describe('legacyPageReplacement', () => {
  it('maps the retired Workflow Designer slug to Process Designer', () => {
    expect(legacyPageReplacement('core-workflow-designer')).toBe('client-workflow-studio');
  });

  it('leaves a live page alone, so it is looked up normally', () => {
    expect(legacyPageReplacement('client-workflow-studio')).toBeUndefined();
    expect(legacyPageReplacement('core-process-catalog')).toBeUndefined();
  });

  it('returns undefined for an unknown slug rather than guessing', () => {
    expect(legacyPageReplacement('no-such-page')).toBeUndefined();
    expect(legacyPageReplacement('')).toBeUndefined();
  });

  it('matches exactly, never by prefix or inherited object keys', () => {
    // A prefix of a retired slug is a different page and must not be redirected.
    expect(legacyPageReplacement('core-workflow-designer-v2')).toBeUndefined();
    expect(legacyPageReplacement('x-core-workflow-designer')).toBeUndefined();
    // Inherited Object.prototype keys are not pages; a plain property read would return one.
    expect(legacyPageReplacement('toString')).toBeUndefined();
    expect(legacyPageReplacement('constructor')).toBeUndefined();
  });

  it('only points at pages that survive', () => {
    // A redirect to a retired page would bounce a visitor straight back out of the app.
    for (const [from, to] of Object.entries(LEGACY_PAGE_SLUGS)) {
      expect(LEGACY_PAGE_SLUGS[to], `${from} redirects to another retired page`).toBeUndefined();
    }
  });
});

describe('SlugPage redirect behaviour', () => {
  /**
   * The redirect branch of SlugPage, with the two things it touches stubbed: StudioPageContent
   * stands in for the page lookup, and the locale is fixed so the expected href is readable.
   */
  const StudioPageContent = ({ slug }: { slug: string }) => (
    <div data-testid="page-content">{slug}</div>
  );

  const SlugPageUnderTest = () => {
    const { slug } = useParams<{ slug: string }>();
    // Stands in for useLocale(), which production reads the same way: off the route.
    const { locale } = useParams<{ locale: string }>();
    const replacement = legacyPageReplacement(slug ?? '');
    if (replacement) {
      return <Navigate to={`/${locale ?? 'en'}/p/${replacement}`} replace />;
    }
    return <StudioPageContent slug={slug ?? ''} />;
  };

  const renderAt = (path: string) =>
    render(
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/:locale/p/:slug" element={<SlugPageUnderTest />} />
        </Routes>
      </MemoryRouter>,
    );

  /**
   * Records the pathname after the redirect settles. The probe lives beside <Routes> rather than
   * in a catch-all route because the redirect target matches the same route pattern, so a route
   * level probe would be shadowed by it.
   */
  const renderFollowingRedirect = (path: string) => {
    const seen: string[] = [];
    const Probe = () => {
      seen.push(useLocation().pathname);
      return null;
    };
    render(
      <MemoryRouter initialEntries={[path]}>
        <Probe />
        <Routes>
          <Route path="/:locale/p/:slug" element={<SlugPageUnderTest />} />
        </Routes>
      </MemoryRouter>,
    );
    return seen;
  };

  it('sends /p/core-workflow-designer to Process Designer without rendering the old page', () => {
    const seen = renderFollowingRedirect(
      '/en/p/core-workflow-designer?tenantId=00000000-0000-4000-a000-000000000002',
    );

    expect(seen.at(-1)).toBe('/en/p/client-workflow-studio');
    // What lands on screen is Process Designer; the retired page's own content never renders.
    expect(screen.getByTestId('page-content')).toHaveTextContent('client-workflow-studio');
    expect(screen.getByTestId('page-content')).not.toHaveTextContent('core-workflow-designer');
  });

  it('preserves the locale prefix on redirect', () => {
    expect(renderFollowingRedirect('/de/p/core-workflow-designer').at(-1)).toBe(
      '/de/p/client-workflow-studio',
    );
  });

  it('renders a live page itself, with no redirect in the way', () => {
    renderAt('/en/p/client-workflow-studio');
    expect(screen.getByTestId('page-content')).toHaveTextContent('client-workflow-studio');
  });

  it('still lets an unknown slug fall through to the page lookup', () => {
    // The lookup owns the "not in this environment yet" message; the redirect must not swallow it.
    renderAt('/en/p/no-such-page');
    expect(screen.getByTestId('page-content')).toHaveTextContent('no-such-page');
  });
});