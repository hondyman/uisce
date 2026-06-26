// frontend/src/components/catalog/RelatedObjectsPanel.tsx
// JSX only — React runtime import not needed with the new JSX transform and there's no React.* usage here.
import { lazy, Suspense, useState, useEffect as _useEffect, useMemo } from "react";
import { useQuery, useMutation, gql } from "@apollo/client";
// AISuggestButton is loaded lazily below; don't import the default directly to avoid unused-import lint warnings
import SuggestionPreviewPanel from "./SuggestionPreviewPanel";  // inline preview panel
import type { Entity, Field } from "../../types/entity-schema";

const GET_RELATED_OBJECTS = gql`
  query GetRelatedObjects($tenantId: ID!, $datasourceId: ID!, $entity: String!) {
    getRelatedObjects(tenantId: $tenantId, datasourceId: $datasourceId, entity: $entity) {
      edgeId
      direction
      edgeType
      cardinality
      source { id name description }
      target { id name description }
    }
  }
`;

const _GET_RELATIONSHIP_SUGGESTIONS = gql`
  query GetRelationshipSuggestions($tenantId: ID!, $datasourceId: ID!, $entity: String!, $limit: Int) {
    getRelationshipSuggestions(tenantId: $tenantId, datasourceId: $datasourceId, entity: $entity, limit: $limit) {
      id
      title
      description
      sourceEntity
      targetEntity
      edgeType
      cardinality
      fkColumn
      confidence
      reasoning
      dismissible
    }
  }
`;

const GET_SEMANTIC_TERM_CATALOG_NODES = gql`
  query GetSemanticTermCatalogNodes($datasourceId: ID!, $semanticTermIds: [ID!]!) {
    catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        _or: [
          { properties: { path: ["semantic_term_id"], value: { _in: $semanticTermIds } } }
        ]
      }
    ) {
      id
      node_name
      qualified_path
      properties
      node_type_id
      
      # Find foreign key relationships
      outbound_edges: catalog_edge(
        where: {
          tenant_datasource_id: { _eq: $datasourceId },
          relationship_type: { _eq: "foreign_key" }
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
  }
`;

const APPLY_RELATIONSHIP = gql`
  mutation ApplyRelationship($tenantId: ID!, $datasourceId: ID!, $sourceEntity: String!, $targetEntity: String!, $edgeType: EdgeType!, $cardinality: String, $fkColumn: String, $confidence: Float) {
    applyRelationship(tenantId: $tenantId, datasourceId: $datasourceId, sourceEntity: $sourceEntity, targetEntity: $targetEntity, edgeType: $edgeType, cardinality: $cardinality, fkColumn: $fkColumn, confidence: $confidence) {
      edgeId
      direction
      edgeType
      cardinality
      source { id name description }
      target { id name description }
    }
  }
`;

const DISMISS_SUGGESTION = gql`
  mutation DismissRelationshipSuggestion($tenantId: ID!, $datasourceId: ID!, $suggestionId: ID!, $reason: String) {
    dismissRelationshipSuggestion(tenantId: $tenantId, datasourceId: $datasourceId, suggestionId: $suggestionId, reason: $reason)
  }
`;

type Props = {
  tenantId: string;
  datasourceId: string;
  entity: string;
  entityData?: Entity; // Full entity data for semantic term analysis
};

const LazyAISuggestButton = lazy(() => import("../validation/AISuggestButton"));

type RelationshipSuggestion = {
  id: string;
  title: string;
  description: string;
  sourceEntity: string;
  targetEntity: string;
  edgeType: string;
  cardinality: string;
  fkColumn?: string;
  confidence: number;
  reasoning?: string;
  dismissible: boolean;
};

function isRelationshipSuggestion(obj: any): obj is RelationshipSuggestion {
  return obj && typeof obj.id === 'string' && typeof obj.sourceEntity === 'string' && typeof obj.targetEntity === 'string';
}

export default function RelatedObjectsPanel({ tenantId, datasourceId, entity, entityData }: Props) {
  const { data, loading, error, refetch } = useQuery(GET_RELATED_OBJECTS, {
    variables: { tenantId, datasourceId, entity },
    fetchPolicy: "cache-and-network",
  });
  const [applyRelationship] = useMutation(APPLY_RELATIONSHIP);
  const [dismissSuggestion] = useMutation(DISMISS_SUGGESTION);
  const [selectedSuggestion, setSelectedSuggestion] = useState<RelationshipSuggestion | null>(null);

  // Extract semantic term IDs from entity data
  const semanticTermIds = useMemo(() => {
    if (!entityData) return [];
    
    const termIds = new Set<string>();
    
    // Add semantic terms from entity fields
    if (entityData.entity_fields) {
      entityData.entity_fields.forEach((field: Field) => {
        if (field.semanticTermId) {
          termIds.add(field.semanticTermId);
        }
      });
    }
    
    // Add semantic terms from subtype fields
    if (entityData.subtypes) {
      Object.values(entityData.subtypes).forEach((subtype) => {
        if (subtype.subtype_fields) {
          subtype.subtype_fields.forEach((field: Field) => {
            if (field.semanticTermId) {
              termIds.add(field.semanticTermId);
            }
          });
        }
      });
    }
    
    return Array.from(termIds);
  }, [entityData]);

  // Query for catalog nodes and relationships based on semantic terms
  const { data: catalogData, loading: catalogLoading } = useQuery(GET_SEMANTIC_TERM_CATALOG_NODES, {
    variables: { datasourceId, semanticTermIds },
    skip: semanticTermIds.length === 0,
    fetchPolicy: "cache-and-network",
  });

  // Generate auto-suggestions based on catalog data
  const autoSuggestions = useMemo(() => {
    if (!catalogData?.catalog_node || !entityData) return [];
    
    const suggestions: RelationshipSuggestion[] = [];
    const processedTargets = new Set<string>();
    
    catalogData.catalog_node.forEach((catalogNode: any) => {
      if (catalogNode.outbound_edges) {
        catalogNode.outbound_edges.forEach((edge: any) => {
          const targetNode = edge.target_node;
          if (targetNode && !processedTargets.has(targetNode.id)) {
            processedTargets.add(targetNode.id);
            
            // Extract table names from qualified paths
            const sourceTable = catalogNode.qualified_path?.split('.').pop() || catalogNode.node_name;
            const targetTable = targetNode.qualified_path?.split('.').pop() || targetNode.node_name;
            
            // Create a more meaningful suggestion
            const suggestionId = `auto_fk_${catalogNode.id}_${targetNode.id}`;
            
            // Try to infer relationship type from foreign key properties
            const fkProperties = edge.properties || {};
            const isNullable = fkProperties.is_nullable === true || fkProperties.is_nullable === 'YES';
            const cardinality = isNullable ? '0:N' : '1:N';
            
            suggestions.push({
              id: suggestionId,
              title: `Relationship to ${targetTable}`,
              description: `Foreign key from ${sourceTable}.${catalogNode.node_name} → ${targetTable}.${targetNode.node_name}`,
              sourceEntity: entity, // Current entity
              targetEntity: targetTable, // Target table name (could be improved to find actual entity name)
              edgeType: 'references', // More accurate than 'has_many'
              cardinality: cardinality,
              fkColumn: `${sourceTable}.${catalogNode.node_name}`,
              confidence: 0.9, // Higher confidence for schema-based suggestions
              reasoning: `Database foreign key relationship detected in schema`,
              dismissible: true,
            });
          }
        });
      }
    });
    
    // Sort by confidence and limit results
    return suggestions
      .sort((a, b) => b.confidence - a.confidence)
      .slice(0, 5);
  }, [catalogData, entityData, entity]);

  if (loading || catalogLoading) return <p>Loading related objects...</p>;
  if (error) return <p>Error loading related objects: {String(error)}</p>;

  const handleApply = async (sugg: RelationshipSuggestion) => {
    // Map suggestion fields to mutation variables as needed by the backend
    const edgeType = typeof sugg.edgeType === 'string' ? sugg.edgeType : 'references';
    const cardinality = typeof sugg.cardinality === 'string' ? sugg.cardinality : undefined;
    await applyRelationship({ variables: { tenantId, datasourceId, sourceEntity: sugg.sourceEntity, targetEntity: sugg.targetEntity, edgeType, cardinality, fkColumn: sugg.fkColumn, confidence: sugg.confidence } });
    await refetch();
    setSelectedSuggestion(null);
  };

  const handleDismiss = async (sugg: RelationshipSuggestion, reason: string) => {
    await dismissSuggestion({ variables: { tenantId, datasourceId, suggestionId: sugg.id, reason } });
    setSelectedSuggestion(null);
  };

  const items = data?.getRelatedObjects ?? [];
  const outbound = items.filter((r: any) => r.direction === "OUTBOUND");
  const inbound = items.filter((r: any) => r.direction === "INBOUND");

  return (
    <div className="space-y-4" role="region" aria-label="Related Objects Panel">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Related Objects</h3>
        <Suspense fallback={<div>Loading AI Suggest...</div>}>
          <LazyAISuggestButton
            context="cross_entity"
            entity={entity}
            tenantId={tenantId}
            datasourceId={datasourceId}
            variant="button"
            onSuggestionSelected={(sugg: any | null) => setSelectedSuggestion(isRelationshipSuggestion(sugg) ? sugg : null)}
          />
        </Suspense>
      </div>
      
      {/* Auto-suggested relationships based on semantic terms and foreign keys */}
      {autoSuggestions.length > 0 && (
        <div>
          <h4 className="font-medium text-gray-700 mb-2">💡 Suggested Relationships</h4>
          <div className="space-y-2">
            {autoSuggestions.map((suggestion) => (
              <div key={suggestion.id} className="border rounded p-3 bg-blue-50">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">{suggestion.title}</div>
                    <div className="text-sm text-gray-600">{suggestion.description}</div>
                    <div className="text-xs text-gray-500 mt-1">
                      {suggestion.sourceEntity} → {suggestion.targetEntity} • {suggestion.edgeType} • {suggestion.cardinality}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleApply(suggestion)}
                      className="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
                    >
                      Apply
                    </button>
                    <button
                      onClick={() => handleDismiss(suggestion, 'not_relevant')}
                      className="px-3 py-1 bg-gray-300 text-gray-700 text-sm rounded hover:bg-gray-400"
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <h4 className="font-medium text-gray-700">Links to</h4>
          <ul className="mt-2 space-y-2">
            {outbound.map((r: any) => (
              <li key={r.edgeId} className="border rounded p-2" aria-label={`Outbound link to ${r.target.name}`}>
                {entity} → {r.target.name} • {r.edgeType} • {r.cardinality}
              </li>
            ))}
          </ul>
        </div>
        <div>
          <h4 className="font-medium text-gray-700">Links from</h4>
          <ul className="mt-2 space-y-2">
            {inbound.map((r: any) => (
              <li key={r.edgeId} className="border rounded p-2" aria-label={`Inbound link from ${r.source.name}`}>
                {r.source.name} → {entity} • {r.edgeType} • {r.cardinality}
              </li>
            ))}
          </ul>
        </div>
      </div>
      {selectedSuggestion && (
        <SuggestionPreviewPanel
          suggestion={{ ...selectedSuggestion, fkColumn: selectedSuggestion.fkColumn ?? '', reasoning: selectedSuggestion.reasoning ?? '' }}
          onApply={handleApply}
          onDismiss={handleDismiss}
          onClose={() => setSelectedSuggestion(null)}
        />
      )}
    </div>
  );
}