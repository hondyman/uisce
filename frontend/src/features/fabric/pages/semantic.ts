import { gql } from '@apollo/client';
import pako from 'pako';
import { devError } from '@/api';

export const GET_TECHNICAL_LINEAGE_CHART = gql`
  query GetTechnicalLineageChart($datasourceId: uuid!) {
    tenant_chart(where: {
      tenant_tenant_instance_id: {_eq: $datasourceId}, 
      chart_name: {_eq: "technical_lineage_chart"}
    }) {
      id
      chart
      updated_at
    }
  }
`;

export const GET_SEMANTIC_LINEAGE_CHART = gql`
  query GetSemanticLineageChart($datasourceId: uuid!) {
    tenant_chart(where: {
      tenant_tenant_instance_id: {_eq: $datasourceId}, 
      chart_name: {_eq: "semantic_lineage_chart"}
    }) {
      id
      chart
      updated_at
    }
  }
`;

export const GET_COMBINED_CHART = GET_TECHNICAL_LINEAGE_CHART; // Alias for now

export const GET_ALL_SEMANTIC_DATA = gql`
  query GetAllSemanticData($datasourceId: uuid!) {
    business_terms(where: { tenant_tenant_instance_id: { _eq: $datasourceId } }) { id node_name description parent_id properties }
    semantic_terms(where: { tenant_tenant_instance_id: { _eq: $datasourceId } }) { id node_name description parent_id properties }
    semantic_views(where: { tenant_tenant_instance_id: { _eq: $datasourceId } }) { id node_name description parent_id properties }
    semantic_edges(where: { tenant_tenant_instance_id: { _eq: $datasourceId } }) { id source_node_id target_node_id relationship_type properties }
  }
`;

export const transformChartData = (hexString: string) => {
  try {
    const compressedData = Uint8Array.from(Buffer.from(hexString, 'hex'));
    const decompressedData = pako.inflate(compressedData, { to: 'string' });
    return JSON.parse(decompressedData);
  } catch (error) {
    devError("Failed to decompress or parse chart data:", error);
    return null;
  }
};