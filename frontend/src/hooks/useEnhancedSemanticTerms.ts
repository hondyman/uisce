import { useQuery, gql } from '@apollo/client';
import { useEffect, useState, useMemo } from 'react';
import { devLog } from '../utils/devLogger';
import apiClient from '../utils/apiClient';

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

// GraphQL query to fetch semantic terms with full metadata
const GET_SEMANTIC_TERMS_WITH_METADATA = gql`
  query GetSemanticTermsWithMetadata($datasourceId: uuid!) {
    catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
        node_type_id: { _eq: "820b942a-9c9e-4abc-acdc-84616db33098" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_name
      description
      properties
      qualified_path
      created_at
      updated_at
    }
  }
`;

/**
 * Hook to fetch semantic terms with metadata
 * Enhances terms with computed fields for field creation
 */
export const useEnhancedSemanticTerms = (datasourceId: string | undefined) => {
  const { data, loading, error, refetch } = useQuery(GET_SEMANTIC_TERMS_WITH_METADATA, {
    variables: { datasourceId },
    skip: !datasourceId,
    errorPolicy: 'all',
  });

  // Local state to hold REST fallback rows when GraphQL yields nothing
  const [restFallback, setRestFallback] = useState<any[] | null>(null);
  const [restLoading, setRestLoading] = useState(false);
  const [restError, setRestError] = useState<string | null>(null);

  // Log GraphQL errors separately without onError callback
  useEffect(() => {
    if (error) {
      devLog('[useEnhancedSemanticTerms] GraphQL Error:', { error: error.message });
    }
  }, [error]);

  // If GraphQL returns no rows (or errors) fetch from REST fallback
  useEffect(() => {
    let cancelled = false;
    const shouldAttemptRest = (!loading && (error || (data && Array.isArray(data?.catalog_node) && data.catalog_node.length === 0)));
    if (!datasourceId) return;
    if (!shouldAttemptRest) return;

    const fetchRest = async () => {
      if (cancelled) return;
      setRestLoading(true);
      setRestError(null);
      try {
        // Request only semantic_term nodes from the catalog to avoid receiving columns/tables
        const url = `/catalog/nodes?type=semantic_term&tenant_instance_id=${datasourceId}`;
        const json = await apiClient(url);
        // apiClient returns parsed JSON, so no need to call .json()
        if (!cancelled) {
          setRestFallback(Array.isArray(json) ? json : (json.catalog_node || json));
          setRestLoading(false);
        }
      } catch (e: any) {
        if (!cancelled) {
          setRestError(e?.message || String(e));
          setRestLoading(false);
        }
      }
    };

    fetchRest();
    return () => { cancelled = true; };
  }, [datasourceId, data, loading, error]);

  const sourceRows = useMemo(() => (data?.catalog_node && Array.isArray(data.catalog_node) && data.catalog_node.length > 0)
    ? data.catalog_node
    : (restFallback || []), [data, restFallback]);

  // Transform and enhance terms
  const enhancedTerms: EnhancedSemanticTerm[] = useMemo(() => (sourceRows || []).map((term: any) => {
    const properties = typeof term.properties === 'string' ? JSON.parse(term.properties) : term.properties || {};

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
      qualified_path: term.qualified_path,
      properties,
      // Computed fields for convenience
      businessName: term.node_name,
      technicalName,

      dataType,
      role: properties.role || 'DIMENSION',
      title_short: properties.title_short
    } as EnhancedSemanticTerm;
  }), [sourceRows]);

  const combinedLoading = loading || restLoading;

  // If the REST fallback returned rows, prefer those results and suppress
  // GraphQL schema errors (for example: "field 'catalog_node' not found").
  // This avoids showing an error in the UI when the fallback succeeded.
  let combinedError: string | undefined;
  if (restFallback && Array.isArray(restFallback) && restFallback.length > 0) {
    combinedError = restError || undefined;
  } else {
    combinedError = error?.message || restError || undefined;
  }

  devLog('[useEnhancedSemanticTerms] Loaded terms:', {
    count: enhancedTerms.length,
    loading: combinedLoading,
    error: combinedError,
  });

  return {
    semanticTerms: enhancedTerms,
    loading: combinedLoading,
    error: combinedError ? new Error(combinedError) : undefined,
    refetch,
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
 * Group semantic terms by category
 */
export const groupSemanticTermsByCategory = (terms: EnhancedSemanticTerm[]) => {
  const grouped: Record<string, EnhancedSemanticTerm[]> = {};

  terms.forEach((term) => {
    const category = term.properties?.category || 'Other';
    if (!grouped[category]) {
      grouped[category] = [];
    }
    grouped[category].push(term);
  });

  return grouped;
};

/**
 * Search semantic terms by text
 */
export const searchSemanticTerms = (terms: EnhancedSemanticTerm[], query: string) => {
  const q = query.toLowerCase();
  return terms.filter(
    (term) =>
      term.node_name.toLowerCase().includes(q) ||
      term.description.toLowerCase().includes(q) ||
      (term.technicalName && term.technicalName.toLowerCase().includes(q))
  );
};
