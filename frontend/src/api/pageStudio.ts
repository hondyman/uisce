import axios from 'axios';
import { CorePageDefinition, PageOverlay, EffectivePageDefinition, UpgradeImpact } from '../types/pageStudio';
import { mergePage } from '../utils/pageMerge';

const API_BASE = '/api/page-studio';
const effectivePageCache = new Map<string, EffectivePageDefinition>();

export const PageStudioApi = {
    listPages: async (env: string): Promise<CorePageDefinition[]> => {
        const resp = await axios.get(`${API_BASE}/pages?env=${env}`);
        return resp.data;
    },
    getPage: async (id: string): Promise<CorePageDefinition> => {
        const resp = await axios.get(`${API_BASE}/pages/${id}`);
        return resp.data;
    },
    getPageBySlug: async (slug: string, env: string): Promise<CorePageDefinition> => {
        const resp = await axios.get(`${API_BASE}/pages/slug/${slug}?env=${env}`);
        return resp.data;
    },
    savePage: async (page: Partial<CorePageDefinition>): Promise<CorePageDefinition> => {
        const resp = await axios.post(`${API_BASE}/pages`, page);
        return resp.data;
    },
    getOverlay: async (pageId: string, tenantId: string, env: string): Promise<PageOverlay> => {
        const resp = await axios.get(`${API_BASE}/pages/${pageId}/overlay?tenant_id=${tenantId}&env=${env}`);
        return resp.data;
    },
    saveOverlay: async (overlay: Partial<PageOverlay>): Promise<PageOverlay> => {
        const resp = await axios.post(`${API_BASE}/pages/${overlay.parentId}/overlay`, overlay);
        return resp.data;
    },

    // AI Features
    generateLayout: async (boName: string, intent: string, tenantId: string): Promise<any> => {
        const resp = await axios.post(`${API_BASE}/ai/generate-layout`, { bo_name: boName, intent, tenant_id: tenantId });
        return resp.data;
    },

    // Epic 3: Multi-tenant Runtime Resolution
    resolveEffectivePage: async (slug: string, tenantId: string, env: string): Promise<EffectivePageDefinition> => {
        const cacheKey = `${tenantId}:${slug}:${env}`;
        if (effectivePageCache.has(cacheKey)) {
            return effectivePageCache.get(cacheKey)!;
        }

        const core = await PageStudioApi.getPageBySlug(slug, env);
        let overlay: PageOverlay | null = null;
        try {
            overlay = await PageStudioApi.getOverlay(core.id, tenantId, env);
        } catch (e) {
            // Overlay might not exist, which is fine
            console.log(`No overlay found for ${slug} / ${tenantId}`);
        }

        const effective = mergePage(core, overlay || undefined);
        effectivePageCache.set(cacheKey, effective);
        return effective;
    },

    getPageBundle: async (slug: string, tenantId: string, env: string, params: Record<string, any>): Promise<Record<string, any>> => {
        const query = new URLSearchParams({ env, tenant_id: tenantId, ...params }).toString();
        const resp = await axios.get(`/api/page-studio/runtime/page-bundle/${slug}?${query}`);
        // Response format: { pageSlug, tenantID, data }
        return resp.data.data;
    },

    invalidateCache: (slug?: string, tenantId?: string, env?: string) => {
        if (!slug) {
            effectivePageCache.clear();
        } else {
            const cacheKey = `${tenantId}:${slug}:${env}`;
            effectivePageCache.delete(cacheKey);
        }
    },

    // Epic 3: Tenant Upgrades
    getUpgradeImpacts: async (tenantId: string): Promise<UpgradeImpact[]> => {
        const resp = await axios.get(`${API_BASE}/upgrades?tenant_id=${tenantId}`);
        return resp.data;
    },
    applyUpgradeDecision: async (impactId: string, decisions: any): Promise<void> => {
        await axios.post(`${API_BASE}/upgrades/${impactId}/apply`, decisions);
    }
};
