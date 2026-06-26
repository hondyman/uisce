import { gql, useQuery } from '@apollo/client';
import { devWarn, devDebug, devError } from '../../utils/devLogger';
import pako from 'pako';

// Query for technical lineage data (database relationships)
export const GET_TECHNICAL_LINEAGE_CHART = gql`
  query GetTechnicalLineageChart($datasourceId: uuid!) {
    tenant_chart(where: {
      tenant_datasource_id: { _eq: $datasourceId },
      chart_name: { _eq: "technical_lineage_chart" }
    }) {
      id
      chart_name
      chart
      created_at
      updated_at
    }
  }
`;

// Query for semantic lineage data (business terms and semantic relationships)
export const GET_SEMANTIC_LINEAGE_CHART = gql`
  query GetSemanticLineageChart($datasourceId: uuid!) {
    tenant_chart(where: {
      tenant_datasource_id: { _eq: $datasourceId },
      chart_name: { _eq: "semantic_lineage_chart" }
    }) {
      id
      chart_name
      chart
      created_at
      updated_at
    }
  }
`;

// Query for all charts for a datasource (allows chart type selection)
export const GET_COMBINED_CHART = gql`
  query GetCombinedChart($datasourceId: uuid!) {
    tenant_chart(where: {
      tenant_datasource_id: { _eq: $datasourceId }
    }, order_by: { chart_name: asc }) {
      id
      chart_name
      chart
      created_at
      updated_at
    }
  }
`;

// Query for specific asset lineage (dynamic lineage generation)
export const GET_ASSET_LINEAGE = gql`
  query GetAssetLineage(
    $datasourceId: String!,
    $assetId: String!,
    $assetType: String!,
    $lineageType: String!
  ) {
    asset_lineage(
      tenant_instance_id: $datasourceId,
      asset_id: $assetId,
      asset_type: $assetType,
      lineage_type: $lineageType
    ) {
      nodes {
        id
        label
        type
        nodeType
        isCenter
        direction
        description
        properties
      }
      edges {
        id
        source
        target
        type
        label
        relationship_type
        properties
      }
      metadata {
        lineageType
        centerAsset {
          id
          type
          name
          nodeType
        }
      }
    }
  }
`;

// Query for business terms and semantic data (enhanced)
export const GET_ALL_SEMANTIC_DATA = gql`
  query GetAllSemanticData($datasourceId: uuid!) {
    # Business Terms
    business_terms: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: "21645d21-de5f-4feb-af99-99273ea75626" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }
    
    # Semantic Terms
    semantic_terms: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: "820b942a-9c9e-4abc-acdc-84616db33098" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }
    
    # Semantic Columns
    semantic_columns: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: "1439f761-606a-44cb-b4f8-7aa6b27a9bf5" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }
    
    # Semantic Relationships/Edges
    semantic_edges: catalog_edge(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
      }
      order_by: { created_at: desc }
    ) {
      id
      source_node_id
      target_node_id
      edge_type_id
      edge_type_name
      properties
      created_at
      updated_at
    }
  }
`;

// Query for lineage context (tables and foreign keys for technical lineage)
export const GET_TECHNICAL_LINEAGE_CONTEXT = gql`
  query GetTechnicalLineageContext($datasourceId: uuid!) {
    # Tables
    tables: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: "49a50271-ae58-4d3e-ae1c-2f5b89d89192" }
      }
      order_by: { qualified_path: asc }
    ) {
      id
      node_name
      qualified_path
      properties
      core_id
      
      # Columns for each table
      children: catalog_nodes(
        where: {
          node_type_id: { _eq: "a64c1011-16e8-4ddf-b447-363bf8e15c9a" }
        }
        order_by: { properties: { path: ["ordinal_position"], order_by: asc } }
      ) {
        id
        node_name
        properties
        core_id
      }
    }
    
    # Foreign Key relationships
    foreign_keys: catalog_edge(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        relationship_type: { _eq: "foreign_key" }
      }
    ) {
      id
      source_node_id
      target_node_id
      relationship_type
      edge_type_name
      properties
    }
  }
`;

// Hook for using technical lineage data
export const useTechnicalLineage = (datasourceId: string) => {
  return useQuery(GET_TECHNICAL_LINEAGE_CHART, {
    variables: { datasourceId },
    skip: !datasourceId
  });
};

// Hook for using semantic lineage data  
export const useSemanticLineage = (datasourceId: string) => {
  return useQuery(GET_SEMANTIC_LINEAGE_CHART, {
    variables: { datasourceId },
    skip: !datasourceId
  });
};

// Hook for using combined semantic data
export const useSemanticData = (datasourceId: string) => {
  return useQuery(GET_ALL_SEMANTIC_DATA, {
    variables: { datasourceId },
    skip: !datasourceId
  });
};

// Hook for dynamic asset lineage
export const useAssetLineage = (
  datasourceId: string,
  assetId: string,
  assetType: string,
  lineageType: 'technical' | 'semantic' | 'combined'
) => {
  return useQuery(GET_ASSET_LINEAGE, {
    variables: {
      datasourceId,
      assetId,
      assetType,
      lineageType
    },
    skip: !datasourceId || !assetId || !assetType
  });
};

// Mutation for building/refreshing charts
export const BUILD_LINEAGE_CHARTS = gql`
  mutation BuildLineageCharts(
    $datasourceId: uuid!,
    $chartTypes: [String!]!,
    $isGoldCopy: Boolean = false
  ) {
    build_lineage_charts(
      tenant_instance_id: $datasourceId,
      chart_types: $chartTypes,
      is_gold_copy: $isGoldCopy
    ) {
      success
      message
      charts_built
      errors
    }
  }
`;

// Utility functions for data transformation
export const transformChartData = (compressedData: string | null | undefined) => {
  if (!compressedData) {
    devWarn('transformChartData: No compressed data provided');
    return null;
  }

  try {
    // The data from GraphQL for a bytea column is a hex string like "\x1f8b..."
    devDebug('transformChartData: Received compressed data (first 50 chars):', compressedData.substring(0, 50));
    const bytes = hexToUint8Array(compressedData);
    devDebug('transformChartData: Converted to Uint8Array, size:', bytes.length);

    // Check for gzip magic number. If it's not present, the data is not compressed.
    if (bytes.length < 2 || bytes[0] !== 0x1f || bytes[1] !== 0x8b) {
      // It's not gzipped. It might be raw JSON that was incorrectly stored as bytea.
      // We need to convert the Uint8Array back to a string to parse it.
      devWarn("Chart data is not compressed. First 2 bytes:", bytes[0]?.toString(16), bytes[1]?.toString(16), "Expected: 1f 8b");
      const jsonString = new TextDecoder().decode(bytes);
      const parsed = JSON.parse(jsonString);
      devDebug('transformChartData: Parsed uncompressed JSON. Nodes:', parsed.nodes?.length);
      return normalizeChart(parsed);
    }

    // It looks like gzipped data, so decompress it.
    devDebug('transformChartData: Data is gzipped, decompressing...');
    const decompressed = pako.ungzip(bytes, { to: 'string' });
    devDebug('transformChartData: Decompressed to string, size:', decompressed.length);
    const parsed = JSON.parse(decompressed);
    devDebug('transformChartData: Parsed JSON. Structure: nodes=', parsed.nodes?.length, 'edges=', parsed.edges?.length);

    // Log sample node structure
    if (parsed.nodes && parsed.nodes.length > 0) {
      devDebug('transformChartData: Sample node structure:', JSON.stringify({
        id: parsed.nodes[0].id,
        type: parsed.nodes[0].type,
        position: parsed.nodes[0].position,
        data: parsed.nodes[0].data ? '{ exists }' : 'undefined',
        keys: Object.keys(parsed.nodes[0]).slice(0, 5)
      }));
    }

    const normalized = normalizeChart(parsed);
    devDebug('transformChartData: Chart normalization complete.');
    return normalized;
  } catch (error) {
    devError('transformChartData: Failed to decompress chart data:', error);
    // Show problematic data only in dev
    devDebug('transformChartData: Problematic data (first 100 chars):', compressedData.substring(0, 100));
    return null;
  }
};

// Normalize chart node shapes so front-end consumers can rely on `node.data.isCore` and `node.data.core_id`.
export const normalizeChart = (chart: any) => {
  if (!chart || !chart.nodes || !Array.isArray(chart.nodes)) return chart;

  let normalizedCount = 0;

  chart.nodes.forEach((node: any) => {
    // Ensure node.data exists
    node.data = node.data || {};

    const data = node.data;

    // Helper to read possible keys from multiple places
    const pick = (keys: string[]) => {
      for (const k of keys) {
        // top-level on node
        if (node[k] !== undefined && node[k] !== null) return node[k];
        // direct data
        if (data[k] !== undefined && data[k] !== null) return data[k];
        // nested data.data
        if (data.data && data.data[k] !== undefined && data.data[k] !== null) return data.data[k];
        // properties
        if (data.properties && data.properties[k] !== undefined && data.properties[k] !== null) return data.properties[k];
        // catalog_defn
        if (data.catalog_defn && data.catalog_defn[k] !== undefined && data.catalog_defn[k] !== null) return data.catalog_defn[k];
      }
      return undefined;
    };

    const possibleCoreId = pick(['core_id', 'coreId', 'CoreID']);

    // Conservative fallback: look for common alternative key names (gold_id, goldCopyId, coreid, etc.)
    const findShallowCoreKey = (obj: any) => {
      if (!obj || typeof obj !== 'object') return undefined;
      const re = /^(core[_-]?id|coreid|gold[_-]?id|goldcopy[_-]?id)$/i;
      for (const k of Object.keys(obj)) {
        if (re.test(k)) {
          const v = obj[k];
          if (v !== undefined && v !== null && String(v) !== '') return v;
        }
      }
      return undefined;
    };

    const fallbackCore = findShallowCoreKey(node) || findShallowCoreKey(data) ||
      (data.properties && findShallowCoreKey(data.properties)) ||
      (data.catalog_defn && findShallowCoreKey(data.catalog_defn));

    const finalCoreCandidate = possibleCoreId !== undefined && possibleCoreId !== null && String(possibleCoreId) !== ''
      ? possibleCoreId
      : fallbackCore;
    const possibleIsCore = pick(['isCore', 'is_core']);

    if (finalCoreCandidate !== undefined && finalCoreCandidate !== null && String(finalCoreCandidate) !== '') {
      try {
        data.core_id = String(finalCoreCandidate);
      } catch (e) {
        data.core_id = finalCoreCandidate;
      }
    }

    // If explicit boolean present or we have a core_id, mark as core
    data.isCore = Boolean(possibleIsCore) || Boolean(data.core_id) || Boolean(data.coreId) || Boolean(data.CoreID);

    if (data.isCore || data.core_id) normalizedCount++;
  });

  // If nothing was detected as core, print a small diagnostic in dev so we can inspect node shapes
  if (normalizedCount === 0) {
    try {
      const sample = (chart.nodes || []).slice(0, 8).map((n: any) => ({
        id: n.id,
        label: n.data?.label || n.label || n.data?.node_name || n.node_name,
        nodeType: n.data?.nodeType || n.type || n.nodeType,
        core_candidates: {
          top_level: n.core_id || n.coreId || n.CoreID || null,
          data_field: n.data && (n.data.core_id || n.data.coreId || n.data.CoreID) || null,
          properties: n.data && n.data.properties && (n.data.properties.core_id || n.data.properties.coreId) || null,
          catalog_defn: n.data && n.data.catalog_defn && (n.data.catalog_defn.core_id || n.data.catalog_defn.coreId) || null
        }
      }));
      devDebug('transformChartData: no core nodes found. Sample node shapes:', sample);
    } catch (e) {
      devDebug('transformChartData: failed to serialize sample nodes for debug', e);
    }
  }
  if (normalizedCount > 0) {
    devDebug(`transformChartData: normalized ${normalizedCount} core nodes`);
  }

  return chart;
};

export const hexToUint8Array = (hexString: string): Uint8Array => {
  if (hexString.startsWith('\\x')) {
    hexString = hexString.substring(2);
  }
  const bytes = new Uint8Array(hexString.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(hexString.substr(i * 2, 2), 16);
  }
  return bytes;
};

// Type definitions for the query responses
export interface TechnicalLineageResponse {
  tenant_chart: Array<{
    id: string;
    chart_name: string;
    chart: string; // compressed data
    created_at: string;
    updated_at: string;
  }>;
}

export interface SemanticDataResponse {
  business_terms: SemanticNodeData[];
  semantic_terms: SemanticNodeData[];
  semantic_columns: SemanticNodeData[];
  semantic_edges: SemanticEdgeData[];
}

export interface SemanticNodeData {
  id: string;
  node_type_id: string;
  node_name: string;
  description: string;
  qualified_path: string;
  parent_id?: string;
  properties: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface SemanticEdgeData {
  id: string;
  source_node_id: string;
  target_node_id: string;
  edge_type_id: string;
  relationship_type: string;
  properties: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface AssetLineageResponse {
  asset_lineage: {
    nodes: Array<{
      id: string;
      label: string;
      type: string;
      nodeType: string;
      isCenter?: boolean;
      direction?: 'upstream' | 'downstream';
      description?: string;
      properties?: Record<string, any>;
    }>;
    edges: Array<{
      id: string;
      source: string;
      target: string;
      type: string;
      label?: string;
      relationship_type: string;
      properties?: Record<string, any>;
    }>;
    metadata: {
      lineageType: string;
      centerAsset: {
        id: string;
        type: string;
        name: string;
        nodeType?: string;
      };
    };
  };
}