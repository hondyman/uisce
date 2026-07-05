import { useEffect, useState, useMemo, useCallback } from 'react';
import { devLog } from '../utils/devLogger';
import { useTenant } from '../contexts/TenantContext';
import { getSelectedRegion } from '../lib/region';

export interface EnhancedSemanticTerm {
  id: string;
  node_name: string; // Will become businessName
  description: string;
  qualified_path: string;
  properties: {
    data_type?: 'text' | 'number' | 'date' | 'boolean' | 'json' | 'array'; // Data type
    technical_name?: string; // e.g., "legal_name"
    category?: string;
    sub_category?: string;
    tags?: string[];
    [key: string]: any;
  };
  // Edge relationships (columns that link to this semantic term)
  edges_as_target?: Array<{
    source_node: {
      id: string;
      node_name: string;
      qualified_path: string;
      node_type_id: string;
      parent_node?: {
        id: string;
        node_name: string;
        qualified_path: string;
      };
    };
  }>;
  // Computed fields
  businessName?: string; // From node_name
  technicalName?: string; // From properties.technical_name or computed from node_name
  data_type?: 'text' | 'number' | 'date' | 'boolean' | 'json' | 'array'; // From properties.data_type
  dataType?: string; // CamelCase version
  role?: string; // From properties.role
  title_short?: string; // From properties.title_short
}

const SEMANTIC_TERM_NODE_TYPE_ID = '820b942a-9c9e-4abc-acdc-84616db33098';

/**
 * Build auth/tenant headers for direct fetch calls.
 * Mirrors the header injection in apiClient but avoids URL resolution issues
 * that can route /api requests to the wrong backend port during local dev.
 */
function buildHeaders(tenantId: string, datasourceId?: string): Record<string, string> {
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
  const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
    'X-Tenant-Region': getSelectedRegion(),
  };
  if (datasourceId) {
    headers['X-Tenant-Datasource-ID'] = datasourceId;
  }
  if (authHeader) {
    headers['Authorization'] = authHeader;
  }
  return headers;
}

/**
 * Hook to fetch semantic terms with metadata via REST.
 * GraphQL-free: reads from /api/catalog/nodes so it works against the local
 * platform backend without requiring a running Hasura/GraphQL endpoint.
 */
export const useEnhancedSemanticTerms = (datasourceId: string | undefined) => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';

  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTerms = useCallback(async () => {
    if (!tenantId) {
      setRows([]);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const url = `/api/catalog/nodes?type=semantic_term&node_type_id=${encodeURIComponent(SEMANTIC_TERM_NODE_TYPE_ID)}&limit=100000`;
      const response = await fetch(url, {
        headers: buildHeaders(tenantId, datasourceId),
      });
      if (!response.ok) {
        throw new Error(`Failed to fetch semantic terms: ${response.status} ${response.statusText}`);
      }
      const json = await response.json();
      const terms = Array.isArray(json) ? json : (json.catalog_node || json.data || []);
      setRows(terms);
    } catch (e: any) {
      setError(e?.message || String(e));
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    fetchTerms();
  }, [fetchTerms]);

  // Transform and enhance terms
  const enhancedTerms: EnhancedSemanticTerm[] = useMemo(() => (rows || []).map((term: any) => {
    const rawProps = typeof term.properties === 'string' ? JSON.parse(term.properties) : term.properties;
    const properties = rawProps || {};

    // Auto-generate technical name if not in properties
    const technicalName = properties.technical_name ||
      term.node_name
        .toLowerCase()
        .replace(/\s+/g, '_')
        .replace(/[^\w_]/g, '');

    // Extract data type from properties, default to 'text'
    const dataType = properties.data_type || 'text';

    return {
      id: term.id,
      node_name: term.node_name,
      description: term.description || '',
      qualified_path: term.qualified_path || term.node_name,
      properties,
      // Computed fields for convenience
      businessName: term.node_name,
      technicalName,

      dataType,
      role: properties.role || 'DIMENSION',
      title_short: properties.title_short
    } as EnhancedSemanticTerm;
  }), [rows]);

  devLog('[useEnhancedSemanticTerms] Loaded terms:', {
    count: enhancedTerms.length,
    loading,
    error,
  });

  return {
    semanticTerms: enhancedTerms,
    loading,
    error: error ? new Error(error) : undefined,
    refetch: fetchTerms,
  };
};

/**
 * Convert semantic term to field
 * Auto-generates businessName, technicalName, and type from semantic term
 */
export const semanticTermToField = (
  semanticTerm: EnhancedSemanticTerm,
  sequence: number = 0
) => {
  return {
    key: semanticTerm.technicalName,
    name: semanticTerm.businessName,
    businessName: semanticTerm.businessName,
    technicalName: semanticTerm.technicalName,
    type: semanticTerm.dataType || 'text',
    role: semanticTerm.role || 'DIMENSION',
    semanticTermId: semanticTerm.id,
    semanticTermName: semanticTerm.node_name,
    description: semanticTerm.description,
    sequence,
    isCore: false,
    lastModifiedAt: new Date().toISOString(),
  };
};

/**
 * Search semantic terms by name or technical name
 */
export const searchSemanticTerms = (
  terms: EnhancedSemanticTerm[],
  query: string
): EnhancedSemanticTerm[] => {
  if (!query || query.trim() === '') {
    return terms;
  }
  const normalizedQuery = query.toLowerCase().trim();
  return terms.filter((term) =>
    term.node_name.toLowerCase().includes(normalizedQuery) ||
    term.technicalName?.toLowerCase().includes(normalizedQuery) ||
    term.description?.toLowerCase().includes(normalizedQuery)
  );
};
