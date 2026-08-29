import { apiClient } from '../utils/apiClient';

const PAGE_STUDIO_BASE = '/api/page-studio';

export interface PageStudioPage {
  id: string;
  name: string;
  slug: string;
  layout: unknown;
  components: unknown[];
  createdAt: string;
  updatedAt: string;
}

export const PageStudioApi = {
  listPages: async (): Promise<PageStudioPage[]> => {
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
};
