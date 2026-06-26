import { useMemo } from 'react';
import { useQueries } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';

// Returns a map of propertyName -> Map<id, label>
// Accepts either a nodeTypeId string OR a nodeType object with properties already loaded
// Optional assetProperties parameter enables cascading lookup resolution
export function usePropertyLookupMaps(nodeTypeOrId?: string | { id?: string; properties?: any[] }, _assetProperties?: Record<string, any>) {
  const { tenant } = useTenant();
  const assetProperties = _assetProperties;
  
  // Extract properties from either the nodeType object or fallback to empty
  const properties = useMemo(() => {
    if (!nodeTypeOrId) {
      return undefined;
    }
    
    // If it's an object with properties, use them directly
    if (typeof nodeTypeOrId === 'object' && 'properties' in nodeTypeOrId) {
      return nodeTypeOrId.properties;
    }
    
    // Otherwise, it's a string ID - we'd need to fetch, but for now return undefined
    return undefined;
  }, [nodeTypeOrId]);

  // Build queries for each property that has a lookup_id
  // For cascading properties, fetch with parent_id parameter
  const queries = useMemo(() => {
    if (!properties || !Array.isArray(properties)) {
      return [];
    }
    const propsWithLookup = properties.filter((p: any) => p.lookup_id);
    
    return propsWithLookup.map((p: any) => {
      // Check if this property cascades from another
      const parentPropName = p.cascade_from;
      let queryUrl = `/api/lookups/${p.lookup_id}/values?tenant_id=${tenant?.id}`;
      
      if (parentPropName && _assetProperties && _assetProperties[parentPropName]) {
        const parentValue = _assetProperties[parentPropName];
        queryUrl += `&parent_id=${encodeURIComponent(parentValue)}`;
        devDebug(`[usePropertyLookupMaps] Property ${p.name} cascades from ${parentPropName}=${parentValue}`);
      }
      
      const isEnabled = !!tenant?.id && !!p.lookup_id && (!parentPropName || !!_assetProperties?.[parentPropName]);
      
      return {
        queryKey: ['property-lookup', tenant?.id, p.lookup_id, p.name, parentPropName, assetProperties?.[parentPropName]],
        queryFn: async () => {
          if (!tenant?.id || !p.lookup_id) return [];
          const res = await fetch(queryUrl, { credentials: 'include' });
          if (!res.ok) {
            return [];
          }
          const raw = await res.json();
          return raw.items || [];
        },
        enabled: isEnabled,
        staleTime: 1000 * 60 * 30, // 30 minutes
        cacheTime: 1000 * 60 * 60,  // 1 hour
        refetchOnWindowFocus: false,
      };
    });
  }, [properties, tenant?.id, assetProperties]);

  // useQueries wants a stable array of configs
  const results = useQueries({ queries });

  // Build map: propertyName -> Map<id,label>
  const lookupMaps = useMemo(() => {
    const out: Record<string, Map<string, string>> = {};
    if (!properties || !Array.isArray(properties)) return out;
    const lookups = properties.filter((p: any) => p.lookup_id);
    lookups.forEach((p: any, idx: number) => {
      const items = (results[idx] && results[idx].data) || [];
      const map = new Map<string, string>();
      (Array.isArray(items) ? items : []).forEach((it: any) => {
        if (it && it.id !== undefined) map.set(String(it.id), it.label || it.name || it.value || String(it.id));
      });
      out[p.name] = map;
      if (map.size > 0) {
        devDebug(`[usePropertyLookupMaps] ✓ Built map for '${p.name}': ${map.size} entries`);
        // Log first 3 entries
        const sampleEntries = Array.from(map.entries()).slice(0, 3);
        devDebug(`[usePropertyLookupMaps]   Sample entries:`, sampleEntries);
      } else {
        devDebug(`[usePropertyLookupMaps] ⚠️ Map for '${p.name}' is EMPTY. Results[${idx}]:`, results[idx]);
      }
    });
    devDebug('[usePropertyLookupMaps] Final lookupMaps keys:', Object.keys(out), `(${Object.keys(out).length} total)`);
    // Log which properties have lookups
    const emptyMaps = Object.entries(out).filter(([_, m]) => m.size === 0).map(([k]) => k);
    if (emptyMaps.length > 0) {
      devDebug('[usePropertyLookupMaps] ⚠️ EMPTY lookup maps (no data):', emptyMaps);
    }
    return out;
  }, [properties, results]);

  return lookupMaps;
}
