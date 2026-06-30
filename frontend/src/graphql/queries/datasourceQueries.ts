import { gql } from '@apollo/client';

export const GET_AVAILABLE_DATASOURCES = gql`
  query GetAvailableDatasources {
    alpha_datasource {
      id
      datasource_code
      config
    }
  }
`;

// Query to get semantic terms for a datasource
export const GET_SEMANTIC_TERMS = gql`
  query GetSemanticTerms($datasourceId: uuid!) {
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
      parent_id
      created_at
      updated_at
    }
  }
`;

// Combined query to get all business term data at once
export const GET_ALL_BUSINESS_DATA = gql`
  query GetAllBusinessData($datasourceId: uuid!) {
    business_terms: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
        node_type_id: { _eq: "21645d21-de5f-4feb-af99-99273ea75626" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_name
      description
      properties
      qualified_path
      parent_id
      created_at
      updated_at
    }

    semantic_terms: catalog_node(
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
      parent_id
      created_at
      updated_at
    }

    semantic_views: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
        node_type_id: { _eq: "c53f9e99-8d02-4dfb-bc1b-914747d35edb" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_name
      description
      properties
      qualified_path
      parent_id
      created_at
      updated_at
    }

    business_edges: catalog_edge(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
        relationship_type: { _in: ["SemanticToView", "SemanticViewToColumn"] }
      }
    ) {
      id
      source_node_id
      target_node_id
      relationship_type
      properties
      created_at
    }
  }
`;


// Query to fetch tables for a datasource (used by TableTypeahead)
export const GET_TABLES_FOR_DATASOURCE = gql`
  query GetTablesForDatasource($datasourceId: uuid!, $q: String, $limit: Int = 100) {
    catalog_node_vw(
      where: {
        _and: [
          { tenant_datasource_id: { _eq: $datasourceId } },
          { node_type_id: { _eq: "49a50271-ae58-4d3e-ae1c-2f5b89d89192" } },
          {
            _or: [
              { node_name: { _ilike: $q } },
              { qualified_path: { _ilike: $q } },
              { source_name: { _ilike: $q } }
            ]
          }
        ]
      }
      limit: $limit
      order_by: { qualified_path: asc }
    ) {
      tenant_tenant_instance_id
      source_name
      node_id
      node_name
      catalog_type_name
      catalog_defn
      node_type_id
      description
      qualified_path
      properties
      parent_id
    }
  }
`;

// Query to get schema tables and columns for Schema Explorer
export const GET_SCHEMA_TABLES = gql`
  query GetSchemaTables($datasourceId: uuid!) {
    tables: catalog_node(
      where: {
        tenant_tenant_instance_id: { _eq: $datasourceId }
        node_type_id: { _eq: "49a50271-ae58-4d3e-ae1c-2f5b89d89192" }
      }
      order_by: { qualified_path: asc }
    ) {
      id
      node_name
      qualified_path
      description
      properties
      parent_id
    }
    columns: catalog_node(
      where: {
        tenant_tenant_instance_id: { _eq: $datasourceId }
        node_type_id: { _eq: "a64c1011-16e8-4ddf-b447-363bf8e15c9a" }
      }
      order_by: { qualified_path: asc }
    ) {
      id
      node_name
      qualified_path
      description
      properties
      parent_id
    }
  }
`;

// Query to fetch a single catalog node by id (used for resolving string values)
export const GET_CATALOG_NODE_BY_ID = gql`
  query GetCatalogNodeById($datasourceId: uuid!, $nodeId: uuid!) {
    catalog_node_vw(
      where: {
        tenant_tenant_instance_id: { _eq: $datasourceId },
        node_id: { _eq: $nodeId }
      }
    ) {
      tenant_tenant_instance_id
      source_name
      node_id
      node_name
      catalog_type_name
      catalog_defn
      node_type_id
      description
      qualified_path
      properties
      parent_id
    }
  }
`;

// Query to fetch columns for a table (used by SimpleColumnAutocomplete)
export const GET_COLUMNS_FOR_TABLE = gql`
  query GetColumnsForTable($datasourceId: uuid!, $parentId: uuid!, $q: String, $limit: Int = 100) {
    catalog_node_vw(
      where: {
        _and: [
          { tenant_tenant_instance_id: { _eq: $datasourceId } },
          { parent_id: { _eq: $parentId } },
          { node_type_id: { _eq: "a64c1011-16e8-4ddf-b447-363bf8e15c9a" } },
          {
            _or: [
              { node_name: { _ilike: $q } },
              { qualified_path: { _ilike: $q } }
            ]
          }
        ]
      }
      limit: $limit
      order_by: { node_name: asc }
    ) {
      tenant_tenant_instance_id
      source_name
      node_id
      node_name
      catalog_type_name
      catalog_defn
      node_type_id
      description
      qualified_path
      properties
      parent_id
    }
  }
`;
