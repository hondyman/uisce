/**
 * App routes served by a Page Studio page rather than a hand-built one: the
 * route renders the page with this slug, as the tenant uses it. One map for
 * both AppRoutes.tsx (what renders) and the Page Studio list (where a page
 * is served), so the two cannot drift.
 */
export const STUDIO_ROUTES: { path: string; slug: string }[] = [
  { path: 'data/mastering', slug: 'mastering-console' },
  { path: 'data/staging-bindings', slug: 'staging-bindings' },
  { path: 'data/pipelines', slug: 'data-pipelines' },
  { path: 'data/pipelines/:id', slug: 'data-pipeline-editor' },
  { path: 'data/mdm/source-hierarchy', slug: 'mdm-source-hierarchy' },
  { path: 'data/mdm/match-rules', slug: 'mdm-match-rules' },
  { path: 'data/mdm/vendors', slug: 'mdm-vendors' },
];

/** The app routes a page is served at, e.g. ['/data/mastering']. */
export function routesForSlug(slug: string): string[] {
  return STUDIO_ROUTES.filter((r) => r.slug === slug).map((r) => `/${r.path}`);
}
