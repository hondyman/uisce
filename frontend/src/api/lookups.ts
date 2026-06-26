import { useQuery } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';

export interface Lookup {
  id: string;
  tenant_id?: string;
  name: string;
  description?: string;
  is_core?: boolean;
}

export interface LookupValue {
  id: string;
  tenant_id?: string;
  name: string;
  parent_id?: string | null;
  metadata?: any;
  label?: string;
  value?: string;
  is_core?: boolean;
}

export function useLookups(tenantId?: string, q?: string, limit?: number) {
  return useQuery({
    queryKey: ['lookups', tenantId, q, limit],
    queryFn: async () => {
      if (!tenantId) return [] as Lookup[];
      const params = new URLSearchParams({ tenant_id: tenantId });
      if (q) params.set('q', q);
      if (limit && limit > 0) params.set('limit', String(limit));
      const res = await fetch(`/api/lookups?${params.toString()}`, { credentials: 'include' });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to fetch lookups');
      }
      // API returns { items: Lookup[], next_cursor: number } for paginated responses.
      // Normalize that shape to return the array of lookups so callers can assume an array.
      const raw = await res.json();
      if (Array.isArray(raw)) return raw as Lookup[];
      return (raw?.items as Lookup[]) || [];
    },
    enabled: !!tenantId,
  });
}

import { useInfiniteQuery } from '@tanstack/react-query';

// Infinite lookup hook for autocompletes — returns pages and fetchNextPage helper
export function useInfiniteLookups(tenantId?: string, q?: string, limit = 50) {
  return useInfiniteQuery({
    // Use a distinct key for infinite lookups to avoid colliding with the
    // simple `useLookups` query (which returns an array). Sharing the same
    // key causes cache shape mismatches (array vs {pages: []}).
    queryKey: ['lookups/infinite', tenantId, q, limit],
    queryFn: async ({ pageParam }) => {
      if (!tenantId) return { items: [], next_cursor: undefined };
      const params = new URLSearchParams({ tenant_id: tenantId, limit: String(limit), cursor: String(pageParam) });
      if (q) params.set('q', q);
      const res = await fetch(`/api/lookups?${params.toString()}`, { credentials: 'include' });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to fetch lookups');
      }
      const data = await res.json() as { items: Lookup[]; next_cursor?: number };
      return { items: data.items || [], next_cursor: data.next_cursor };
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage) => (lastPage?.next_cursor ? lastPage.next_cursor : undefined),
    enabled: !!tenantId,
  });
}

export function useLookupValues(tenantId?: string, lookupId?: string, parentId?: string | null, parentValue?: string | null) {
  return useQuery({
    queryKey: ['lookup-values', tenantId, lookupId, parentId, parentValue],
    queryFn: async () => {
      if (!tenantId || !lookupId) return [] as LookupValue[];
      let url = `/api/lookups/${lookupId}/values?tenant_id=${tenantId}`;
      if (parentId) {
        url += `&parent_id=${encodeURIComponent(parentId)}`;
      } else if (parentValue) {
        url += `&parent_value=${encodeURIComponent(parentValue)}`;
      }
      const res = await fetch(url, { credentials: 'include' });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to fetch lookup values');
      }
      const raw = await res.json();
      // Debug: log response and URL for troubleshooting cascading behavior
      // eslint-disable-next-line no-console
      devDebug('[useLookupValues] fetched', { url, parentId, items: raw.items || [], raw });
      // API returns { items: [...], next_cursor: ... }, so extract items array
      const items = raw.items || [];
      // Normalize server-side field names (label/value into a `name` field) and return lightweight objects
      return items.map((r: any) => ({ id: r.id, tenant_id: r.tenant_id, name: r.label || r.value || r.name, parent_id: r.parent_id || r.parentId || null, metadata: r.metadata, label: r.label, value: r.value, is_core: r.is_core }));
    },
    enabled: !!tenantId && !!lookupId,
  });
}

export function useInfiniteLookupValues(tenantId?: string, lookupId?: string, parentId?: string | null, parentValue?: string | null, limit = 50) {
  return useInfiniteQuery({
    queryKey: ['lookup-values/infinite', tenantId, lookupId, parentId, parentValue, limit],
    queryFn: async ({ pageParam }) => {
      if (!tenantId || !lookupId) return { items: [], next_cursor: undefined };
      let url = `/api/lookups/${lookupId}/values?tenant_id=${tenantId}&limit=${limit}`;
      if (pageParam) {
        url += `&cursor=${encodeURIComponent(String(pageParam))}`;
      }
      if (parentId) {
        url += `&parent_id=${encodeURIComponent(parentId)}`;
      } else if (parentValue) {
        url += `&parent_value=${encodeURIComponent(parentValue)}`;
      }
      const res = await fetch(url, { credentials: 'include' });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to fetch lookup values');
      }
      const raw = await res.json();
      const items = (raw.items || []).map((r: any) => ({ id: r.id, tenant_id: r.tenant_id, name: r.label || r.value || r.name, parent_id: r.parent_id || r.parentId || null, metadata: r.metadata, label: r.label, value: r.value, is_core: r.is_core }));
      return { items, next_cursor: raw.next_cursor };
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage) => (lastPage?.next_cursor ? lastPage.next_cursor : undefined),
    enabled: !!tenantId && !!lookupId,
  });
}

// CRUD helpers for lookups
export async function createLookup(tenantId: string, payload: { name: string, description?: string }) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`/api/lookups?tenant_id=${tenantId}`, { method: 'POST', credentials: 'include', headers, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function updateLookup(tenantId: string, id: string, payload: { name?: string, description?: string }) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`/api/lookups/${id}?tenant_id=${tenantId}`, { method: 'PUT', credentials: 'include', headers, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res;
}

export async function deleteLookup(tenantId: string, id: string) {
  const headers: Record<string, string> = {};
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`/api/lookups/${id}?tenant_id=${tenantId}`, { method: 'DELETE', credentials: 'include', headers });
  if (!res.ok) throw new Error(await res.text());
  return res;
}

export async function createLookupValue(tenantId: string, lookupId: string, payload: { value: string, label: string, parent_id?: string | null, metadata?: any }) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`/api/lookups/${lookupId}/values?tenant_id=${tenantId}`, { method: 'POST', credentials: 'include', headers, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function updateLookupValue(tenantId: string, lookupId: string, valueId: string, payload: { value?: string, label?: string, parent_id?: string | null, metadata?: any }) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`/api/lookups/${lookupId}/values/${valueId}?tenant_id=${tenantId}`, { method: 'PUT', credentials: 'include', headers, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res;
}

export async function deleteLookupValue(tenantId: string, lookupId: string, valueId: string) {
  const headers: Record<string, string> = {};
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const res = await fetch(`/api/lookups/${lookupId}/values/${valueId}?tenant_id=${tenantId}`, { method: 'DELETE', credentials: 'include', headers });
  if (!res.ok) throw new Error(await res.text());
  return res;
}
export async function propagateLookup(tenantId: string, id: string) {
  const res = await fetch(`/api/lookups/${id}/propagate?tenant_id=${tenantId}`, { method: 'POST', credentials: 'include' });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function exportLookupValues(tenantId: string, lookupId: string, lookupName: string, datasourceId?: string) {
  let url = `/api/lookups/${lookupId}/export?tenant_id=${tenantId}`;
  if (datasourceId) {
    url += `&tenant_instance_id=${datasourceId}`;
  }

  const res = await fetch(url, { credentials: 'include' });
  if (!res.ok) {
    throw new Error(await res.text() || 'Failed to export lookup values');
  }

  // Download as file
  const blob = await res.blob();
  const downloadUrl = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = downloadUrl;
  a.download = `${lookupName}.csv`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(downloadUrl);
}