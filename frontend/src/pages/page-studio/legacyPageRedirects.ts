/**
 * Redirects for Page Studio pages that have been retired.
 *
 * A retired page keeps its slug reachable so old bookmarks, shared links, and nav entries land on
 * whatever replaced it, rather than on PageBrowser's "page is not in this environment yet"
 * notice. See PageContent in pages/PageBrowser.tsx for the notice itself.
 *
 * Retired so far:
 *   core-workflow-designer → client-workflow-studio
 *     The legacy Workflow Designer was replaced by Process Designer. Its domain component,
 *     workflow.WorkflowDesigner, never persisted a flow (its Save handler only logged), so there
 *     was no flow data to migrate across. Retired by
 *     backend/db/migrations/20261225_009_retire_legacy_workflow_designer, which also repoints the
 *     /core/workflow-designer route alias at the replacement.
 *
 * Add an entry here when another page is retired; keep the values on the right-hand side to the
 * slug that survives.
 */
export const LEGACY_PAGE_SLUGS: Record<string, string> = {
  'core-workflow-designer': 'client-workflow-studio',
};

/**
 * The page that replaced `slug`, or undefined when `slug` is a live page (or was never a page at
 * all — in which case the caller's normal lookup handles it and reports it missing).
 *
 * Lookup is exact-match only: a slug must never be resolved by prefix or fuzzy match, or a live
 * page whose slug merely starts with a retired one would be silently redirected.
 */
export function legacyPageReplacement(slug: string): string | undefined {
  return Object.prototype.hasOwnProperty.call(LEGACY_PAGE_SLUGS, slug)
    ? LEGACY_PAGE_SLUGS[slug]
    : undefined;
}