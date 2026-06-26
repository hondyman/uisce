/**
 * Business Entity Semantic Layer GraphQL Queries and Mutations
 * 
 * Provides GraphQL interface for core/custom model/view generation,
 * relationship suggestions, and object graph traversal.
 */

import { gql } from '@apollo/client';

/**
 * Queries
 */

export const GET_SEMANTIC_ASSETS = gql`
  query GetSemanticAssets($businessEntityId: uuid!, $datasourceId: uuid!) {
    semantic_assets(
      where: {
        business_entity_id: { _eq: $businessEntityId }
        tenant_instance_id: { _eq: $datasourceId }
      }
    ) {
      business_entity_id
      core_model_id
      core_view_id
      custom_model_id
      custom_view_id
      semantic_term_ids
      created_at
      updated_at
      core_model {
        id
        node_name
        description
        properties
        qualified_path
      }
      core_view {
        id
        node_name
        description
        properties
        qualified_path
      }
      custom_model {
        id
        node_name
        description
        properties
        qualified_path
      }
      custom_view {
        id
        node_name
        description
        properties
        qualified_path
      }
    }
  }
`;

export const GET_RELATIONSHIP_SUGGESTIONS = gql`
  query GetRelationshipSuggestions(
    $businessEntityId: uuid!
    $datasourceId: uuid!
    $limit: Int
  ) {
    relationship_suggestions(
      where: {
        source_entity_id: { _eq: $businessEntityId }
        tenant_instance_id: { _eq: $datasourceId }
      }
      limit: $limit
      order_by: { confidence: desc }
    ) {
      id
      source_entity_id
      target_entity_id
      confidence
      rationale
      scoring_breakdown
      accepted
      created_at
    }
  }
`;

export const GET_LINKED_MODELS = gql`
  query GetLinkedModels($modelId: uuid!, $datasourceId: uuid!) {
    catalog_edge(
      where: {
        source_node_id: { _eq: $modelId }
        relationship_type: { _in: ["joins", "references", "extends"] }
        tenant_instance_id: { _eq: $datasourceId }
      }
    ) {
      id
      target_node_id
      relationship_type
      properties
      created_at
      updated_at
    }
  }
`;

export const GET_RELATED_OBJECTS = gql`
  query GetRelatedObjects($businessEntityId: uuid!, $datasourceId: uuid!) {
    # Links To (Many-to-One)
    links_to: catalog_edge(
      where: {
        source_node_id: { _eq: $businessEntityId }
        relationship_type: { _in: ["references", "many_to_one"] }
        tenant_instance_id: { _eq: $datasourceId }
      }
    ) {
      id
      target_node_id
      relationship_type
      properties
      created_at
      updated_at
    }
    
    # Links From (One-to-Many)
    links_from: catalog_edge(
      where: {
        target_node_id: { _eq: $businessEntityId }
        relationship_type: { _in: ["referenced_by", "one_to_many"] }
        tenant_instance_id: { _eq: $datasourceId }
      }
    ) {
      id
      source_node_id
      relationship_type
      properties
    }
  }
`;

/**
 * Mutations
 */

export const GENERATE_CORE_MODEL = gql`
  mutation GenerateCoreModel(
    $businessEntityId: uuid!
    $businessEntityName: String!
    $semanticTermIds: [uuid!]!
    $sourceTableNames: [String!]!
    $datasourceId: uuid!
    $tenantId: uuid!
  ) {
    generate_core_model(
      input: {
        business_entity_id: $businessEntityId
        business_entity_name: $businessEntityName
        semantic_term_ids: $semanticTermIds
        source_tables: $sourceTableNames
        tenant_instance_id: $datasourceId
        tenant_id: $tenantId
      }
    ) {
      success
      semantic_model {
        id
        node_name
        description
        properties
        qualified_path
        created_at
      }
      error
    }
  }
`;

export const GENERATE_CORE_VIEW = gql`
  mutation GenerateCoreView(
    $businessEntityId: uuid!
    $businessEntityName: String!
    $coreModelId: uuid!
    $semanticTermIds: [uuid!]!
    $datasourceId: uuid!
    $tenantId: uuid!
  ) {
    generate_core_view(
      input: {
        business_entity_id: $businessEntityId
        business_entity_name: $businessEntityName
        core_model_id: $coreModelId
        semantic_term_ids: $semanticTermIds
        tenant_instance_id: $datasourceId
        tenant_id: $tenantId
      }
    ) {
      success
      semantic_view {
        id
        node_name
        description
        properties
        qualified_path
        created_at
      }
      error
    }
  }
`;

export const CREATE_CUSTOM_MODEL = gql`
  mutation CreateCustomModel(
    $businessEntityId: uuid!
    $coreModelId: uuid!
    $customModelName: String!
    $additionalDimensions: [JSON!]
    $additionalMeasures: [JSON!]
    $datasourceId: uuid!
    $tenantId: uuid!
  ) {
    create_custom_model(
      input: {
        business_entity_id: $businessEntityId
        core_model_id: $coreModelId
        custom_model_name: $customModelName
        additional_dimensions: $additionalDimensions
        additional_measures: $additionalMeasures
        tenant_instance_id: $datasourceId
        tenant_id: $tenantId
      }
    ) {
      success
      semantic_model {
        id
        node_name
        description
        properties
        qualified_path
        created_at
      }
      error
    }
  }
`;

export const CREATE_CUSTOM_VIEW = gql`
  mutation CreateCustomView(
    $businessEntityId: uuid!
    $coreViewId: uuid!
    $customViewName: String!
    $customModelId: uuid
    $additionalColumns: [JSON!]
    $datasourceId: uuid!
    $tenantId: uuid!
  ) {
    create_custom_view(
      input: {
        business_entity_id: $businessEntityId
        core_view_id: $coreViewId
        custom_view_name: $customViewName
        custom_model_id: $customModelId
        additional_columns: $additionalColumns
        tenant_instance_id: $datasourceId
        tenant_id: $tenantId
      }
    ) {
      success
      semantic_view {
        id
        node_name
        description
        properties
        qualified_path
        created_at
      }
      error
    }
  }
`;

export const APPLY_RELATIONSHIP_SUGGESTION = gql`
  mutation ApplyRelationshipSuggestion(
    $suggestionId: uuid!
    $sourceEntityId: uuid!
    $targetEntityId: uuid!
    $confidence: Float!
    $rationale: String!
    $scoringBreakdown: JSON!
    $datasourceId: uuid!
    $tenantId: uuid!
  ) {
    apply_relationship_suggestion(
      input: {
        suggestion_id: $suggestionId
        source_entity_id: $sourceEntityId
        target_entity_id: $targetEntityId
        confidence: $confidence
        rationale: $rationale
        scoring_breakdown: $scoringBreakdown
        tenant_instance_id: $datasourceId
        tenant_id: $tenantId
      }
    ) {
      success
      edge_id
      edge {
        id
        source_node_id
        target_node_id
        relationship_type
        properties
        created_at
      }
      error
    }
  }
`;

export const TRAVERSE_OBJECT_GRAPH = gql`
  mutation TraverseObjectGraph(
    $startModelId: uuid!
    $dotPath: String!
    $datasourceId: uuid!
    $tenantId: uuid!
  ) {
    traverse_object_graph(
      input: {
        start_model_id: $startModelId
        dot_path: $dotPath
        tenant_instance_id: $datasourceId
        tenant_id: $tenantId
      }
    ) {
      success
      graph {
        nodes {
          id
          node_name
          description
          properties
          nodeType
        }
        edges {
          id
          source
          target
          relationship_type
          properties
        }
      }
      path_traversed
      error
    }
  }
`;

/**
 * Hooks
 */

import { useQuery, useMutation } from '@apollo/client';

export const useGetSemanticAssets = (businessEntityId: string, datasourceId: string) => {
  return useQuery(GET_SEMANTIC_ASSETS, {
    variables: { businessEntityId, datasourceId },
    skip: !businessEntityId || !datasourceId,
  });
};

export const useGetRelationshipSuggestions = (businessEntityId: string, datasourceId: string) => {
  return useQuery(GET_RELATIONSHIP_SUGGESTIONS, {
    variables: { businessEntityId, datasourceId, limit: 10 },
    skip: !businessEntityId || !datasourceId,
  });
};

export const useGetLinkedModels = (modelId: string, datasourceId: string) => {
  return useQuery(GET_LINKED_MODELS, {
    variables: { modelId, datasourceId },
    skip: !modelId || !datasourceId,
  });
};

export const useGetRelatedObjects = (businessEntityId: string, datasourceId: string) => {
  return useQuery(GET_RELATED_OBJECTS, {
    variables: { businessEntityId, datasourceId },
    skip: !businessEntityId || !datasourceId,
  });
};

export const useGenerateCoreModel = () => {
  return useMutation(GENERATE_CORE_MODEL);
};

export const useGenerateCoreView = () => {
  return useMutation(GENERATE_CORE_VIEW);
};

export const useCreateCustomModel = () => {
  return useMutation(CREATE_CUSTOM_MODEL);
};

export const useCreateCustomView = () => {
  return useMutation(CREATE_CUSTOM_VIEW);
};

export const useApplyRelationshipSuggestion = () => {
  return useMutation(APPLY_RELATIONSHIP_SUGGESTION);
};

export const useTraverseObjectGraph = () => {
  return useMutation(TRAVERSE_OBJECT_GRAPH);
};
