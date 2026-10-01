import { apiClient } from '../utils/apiClient';
import type { CorePageDefinition } from '../types/pageStudio';

const PAGE_STUDIO_BASE = '/api/page-studio';

export interface GeneratedPageSection {
  /** "" = the page's primary Business Object; otherwise a key from relatedBusinessObjects. */
  boKey: string;
  type: string;
  title: string;
}

export interface GeneratedRelatedBO {
  boId: string;
  boKey: string;
  displayName: string;
  cardinality?: string;
  joinCondition?: string;
}

export type GeneratedPageKind = 'list' | 'detail' | 'master-detail' | 'dashboard';

export interface GeneratedPageSpec {
  title: string;
  pageKind?: GeneratedPageKind;
  /** One of layoutTemplates.ts's section-template ids. */
  layoutTemplate: string;
  /** Related Business Objects actually used by one or more sections below - enough for generatePageDraft.ts to build a data source per BO without a second round trip. */
  relatedBusinessObjects: GeneratedRelatedBO[];
  sections: GeneratedPageSection[];
  filterBar?: GeneratedPageSection[];
  /** Which path produced this spec - lets the UI be honest about whether Gemini actually ran or the deterministic template did. */
  source: 'ai' | 'template';
}

/**
 * Alias, not a separate shape: page_studio_handler.go serializes exactly
 * CorePageDefinition's fields (id/name/slug/description/layout/components/
 * dataSources/version/createdAt/updatedAt). This used to be redeclared
 * here with its own (looser, array-typed) layout/components fields, which
 * drifted from the real wire shape and forced every caller to fight the
 * type checker back into the actual object-keyed runtime shape.
 */
export type PageStudioPage = CorePageDefinition;

/** One element's worth of changes (backend/internal/corecustom Group). */
export interface CoreChangeGroup {
  id: string;
  kind: 'page' | 'component' | 'layout' | 'tab' | 'dataSource' | 'rule' | 'app' | 'other' | string;
  label: string;
  summary: 'added' | 'removed' | 'changed';
  changes: { path: string; op: 'add' | 'remove' | 'change' | 'reorder'; old?: unknown; new?: unknown }[];
  /** The core also changed this element; keeping the customization means the tenant's version wins. */
  conflict: boolean;
  /** The core now ships exactly this customization. */
  inCore: boolean;
}

export interface CoreComparison {
  baseVersion: number;
  coreVersion: number;
  upgradeAvailable: boolean;
  customizations: CoreChangeGroup[];
  coreUpdates: CoreChangeGroup[];
}

/** A template as the gallery lists it (no bundle). */
export interface PageTemplateSummary {
  id: string;
  slug: string;
  version: number;
  name: string;
  description: string;
  category: string;
  isCore: boolean;
  createdAt: string;
}

/** What bringing a template in would do (or did) to the tenant's fragments. */
export interface TemplateImportPlan {
  create: { slug: string; version: number }[];
  reuse: { fragment: string; version: number }[];
  conflicts: string[];
}

export interface InstantiatedTemplate {
  plan: TemplateImportPlan;
  /** A fresh draft page document, for the ordinary page API to save. */
  page: Partial<CorePageDefinition>;
  applied: boolean;
}

export const PageStudioApi = {
  listPages: async (_env?: string): Promise<PageStudioPage[]> => {
    return apiClient<PageStudioPage[]>(`${PAGE_STUDIO_BASE}/pages`);
  },

  getPage: async (id: string): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}`);
  },

  getPageBySlug: async (slug: string): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/slug/${slug}`);
  },

  savePage: async (page: Partial<PageStudioPage>): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages`, {
      method: 'POST',
      body: JSON.stringify(page),
      headers: { 'Content-Type': 'application/json' }
    });
  },

  updatePage: async (id: string, page: Partial<PageStudioPage>): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}`, {
      method: 'PUT',
      body: JSON.stringify(page),
      headers: { 'Content-Type': 'application/json' }
    });
  },

  deletePage: async (id: string): Promise<void> => {
    return apiClient<void>(`${PAGE_STUDIO_BASE}/pages/${id}`, { method: 'DELETE' });
  },

  /** Publish or unpublish a page the caller owns. Status only - the version is not bumped, so publishing a core page is not an upgrade for tenants. */
  setStatus: async (id: string, status: 'draft' | 'published'): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  // ── Tenant lifecycle of a core page (page_studio_core.go) ──────────────
  // A tenant never writes the core page itself; these record how the
  // tenant uses it.

  /** Save the tenant's customized copy of a core page (extends it; the first save pins the core version it is based on). */
  saveExtension: async (id: string, page: Partial<PageStudioPage>): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}/extension`, {
      method: 'PUT',
      body: JSON.stringify(page),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  /** Switch a core page on or off in this tenant. */
  setCoreActive: async (id: string, active: boolean): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}/activation`, {
      method: 'PUT',
      body: JSON.stringify({ active }),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  /** Replace a core page with an independent tenant copy at the same slug. No upgrade path. */
  cloneCore: async (id: string): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}/clone`, { method: 'POST' });
  },

  /** Drop the tenant's extension or clone (the clone page is deleted) and use the core page as shipped. */
  revertToCore: async (id: string): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}/customization`, { method: 'DELETE' });
  },

  /** The tenant's customizations and the core's changes since the version they were made on. */
  compareWithCore: async (id: string): Promise<CoreComparison> => {
    return apiClient<CoreComparison>(`${PAGE_STUDIO_BASE}/pages/${id}/compare`);
  },

  /** Rebase the extension onto the current core, carrying every customization except those in `remove`. */
  upgradeExtension: async (id: string, remove: string[]): Promise<PageStudioPage> => {
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}/upgrade`, {
      method: 'POST',
      body: JSON.stringify({ remove }),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  /**
   * Clones a page: fetches its full content and re-POSTs it under a new
   * name/slug. Always creates a tenant-authored (isCore: false) copy,
   * even when cloning a core page - inheriting a gold-copy page and then
   * writing back to the original would violate the read-only inheritance
   * model; cloning is how a tenant gets their own editable starting point.
   */
  clonePage: async (id: string): Promise<PageStudioPage> => {
    const source = await apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages/${id}`);
    const slug = `${source.slug}-copy-${Math.random().toString(36).slice(2, 6)}`;
    return apiClient<PageStudioPage>(`${PAGE_STUDIO_BASE}/pages`, {
      method: 'POST',
      body: JSON.stringify({
        name: `${source.name} (Copy)`,
        slug,
        description: source.description,
        layout: source.layout,
        tabs: source.tabs,
        components: source.components,
        dataSources: source.dataSources,
        presentationEvents: source.presentationEvents,
        filterBar: source.filterBar,
        app: source.app,
        isCore: false,
        status: 'draft',
      }),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  /**
   * Asks the backend to pick a small widget mix (KPI/Chart/Table/Slicer)
   * for a Business Object - via Gemini when configured, a deterministic
   * template otherwise (backend/internal/handlers/page_studio_handler.go's
   * `generate`). Returns only the spec; expanding it into an actual
   * layout/components/dataSources draft is generatePageDraft's job, since
   * that expansion needs the BO's binding id and other client-side context
   * this endpoint doesn't have.
   */
  generateSpec: async (boId: string, boKey: string, boName: string, description: string, pageKind?: GeneratedPageKind): Promise<GeneratedPageSpec> => {
    return apiClient<GeneratedPageSpec>(`${PAGE_STUDIO_BASE}/generate`, {
      method: 'POST',
      body: JSON.stringify({ boId, boKey, boName, description, pageKind }),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  // -- Templates gallery ---------------------------------------------------------------

  listTemplates: async (category?: string): Promise<PageTemplateSummary[]> => {
    const q = category ? `?category=${encodeURIComponent(category)}` : '';
    return apiClient<PageTemplateSummary[]>(`${PAGE_STUDIO_BASE}/templates${q}`);
  },

  /** Save a page as the next version of a template. */
  publishTemplate: async (req: { slug: string; name: string; description: string; category: string; pageId: string }): Promise<PageTemplateSummary> => {
    return apiClient<PageTemplateSummary>(`${PAGE_STUDIO_BASE}/templates`, {
      method: 'POST',
      body: JSON.stringify(req),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  /** Bring a template's fragments in and get a fresh draft page. dryRun only plans. */
  instantiateTemplate: async (slug: string, version: number, req: { name: string; slug: string }, dryRun = false): Promise<InstantiatedTemplate> => {
    return apiClient<InstantiatedTemplate>(`${PAGE_STUDIO_BASE}/templates/${encodeURIComponent(slug)}/versions/${version}/instantiate${dryRun ? '?dryRun=true' : ''}`, {
      method: 'POST',
      body: JSON.stringify(req),
      headers: { 'Content-Type': 'application/json' },
    });
  },

  // -- Fragments -------------------------------------------------------------------------

  /** One immutable version of a fragment: its content (widgets, nodes, variables, queries, uses). */
  getFragmentVersion: async (slug: string, version: number): Promise<{ slug: string; version: number; name: string; contentHash: string; content: Record<string, unknown> }> => {
    return apiClient(`${PAGE_STUDIO_BASE}/fragments/${encodeURIComponent(slug)}/versions/${version}`);
  },

  /** The latest version of each fragment the tenant can see (no history). */
  listFragments: async (): Promise<{ slug: string; version: number; name: string; description: string; isCore: boolean }[]> => {
    return apiClient(`${PAGE_STUDIO_BASE}/fragments`);
  },
};
