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
}

export interface GeneratedPageSpec {
  title: string;
  /** One of layoutTemplates.ts's plain section-template ids (single-column/two-column/three-column/dashboard-grid). */
  layoutTemplate: string;
  /** Related Business Objects actually used by one or more sections below - enough for generatePageDraft.ts to build a data source per BO without a second round trip. */
  relatedBusinessObjects: GeneratedRelatedBO[];
  sections: GeneratedPageSection[];
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
  generateSpec: async (boId: string, boKey: string, boName: string, description: string): Promise<GeneratedPageSpec> => {
    return apiClient<GeneratedPageSpec>(`${PAGE_STUDIO_BASE}/generate`, {
      method: 'POST',
      body: JSON.stringify({ boId, boKey, boName, description }),
      headers: { 'Content-Type': 'application/json' },
    });
  },
};
