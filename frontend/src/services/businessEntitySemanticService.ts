/**
 * Business Entity to Semantic Layer Service
 * 
 * Handles generation and management of core semantic models/views from business entities,
 * with support for custom extensions and AI-powered relationship suggestions.
 */

import { devError, devLog } from '../utils/devLogger';

// Type definitions
export interface SemanticModelMetadata {
  id: string;
  node_name: string;
  description?: string;
  properties: {
    is_core?: boolean;
    extends_model_id?: string;
    business_entity_id?: string;
    semantic_term_ids?: string[];
    source_tables?: string[];
    [key: string]: any;
  };
  created_at?: string;
  updated_at?: string;
}

export interface SemanticViewMetadata {
  id: string;
  node_name: string;
  description?: string;
  properties: {
    is_core?: boolean;
    extends_view_id?: string;
    business_entity_id?: string;
    semantic_term_ids?: string[];
    [key: string]: any;
  };
  created_at?: string;
  updated_at?: string;
}

export interface CoreSemanticAssets {
  coreModel?: SemanticModelMetadata;
  coreView?: SemanticViewMetadata;
  customModel?: SemanticModelMetadata;
  customView?: SemanticViewMetadata;
}

export interface RelationshipSuggestion {
  id: string;
  source_entity_id: string;
  target_entity_id: string;
  confidence: number;
  rationale: string;
  scoring_breakdown: {
    fk_presence: number;
    join_frequency: number;
    name_similarity: number;
    text_similarity: number;
    edge_type_prior: number;
  };
  accepted?: boolean;
}

export interface BusinessEntitySemanticLink {
  businessEntityId: string;
  businessEntityName: string;
  coreModelId?: string;
  coreViewId?: string;
  customModelId?: string;
  customViewId?: string;
  semanticTermIds: string[];
  datasourceId: string;
  tenantId: string;
}

export interface MappingResult {
  database_column: any;
  semantic_term: string;
  semantic_term_id?: string;
  confidence: number;
  is_new_term: boolean;
  selected: boolean;
  match_reason?: string;
  edge_exists: boolean;
  override: boolean;
}

import apiClient from '../utils/apiClient';

// ... (types)

export class BusinessEntitySemanticService {
  constructor(_tenantId?: string, _datasourceId?: string, _apiBase: string = '/api') {
    // These are now handled by the global context in apiClient
  }

  /**
   * Generate or update core semantic model from business entity and semantic terms
   */
  async generateOrUpdateCoreModel(
    businessEntityId: string,
    businessEntityName: string,
    semanticTermIds: string[],
    sourceTableNames: string[]
  ): Promise<SemanticModelMetadata> {
    try {
      devLog(`🔄 Generating core model for entity: ${businessEntityName}`);

      const response = await apiClient(`business-entities/generate-core-model`, {
        method: 'POST',
        body: JSON.stringify({
          business_entity_id: businessEntityId,
          business_entity_name: businessEntityName,
          semantic_term_ids: semanticTermIds,
          source_tables: sourceTableNames,
        }),
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Generate core model endpoint not implemented yet');
          throw new Error('Semantic layer features not yet implemented');
        }
        const error = await response.json();
        throw new Error(error.message || 'Failed to generate core model');
      }

      const result = await response.json();
      devLog('✅ Core model generated successfully:', result);
      return result.semantic_model;
    } catch (error) {
      devError('Error generating core model:', error);
      throw error;
    }
  }

  /**
   * Generate or update core semantic view from business entity
   */
  async generateOrUpdateCoreView(
    businessEntityId: string,
    businessEntityName: string,
    coreModelId: string,
    semanticTermIds: string[]
  ): Promise<SemanticViewMetadata> {
    try {
      devLog(`🔄 Generating core view for entity: ${businessEntityName}`);

      const response = await apiClient(`business-entities/generate-core-view`, {
        method: 'POST',
        body: JSON.stringify({
          business_entity_id: businessEntityId,
          business_entity_name: businessEntityName,
          core_model_id: coreModelId,
          semantic_term_ids: semanticTermIds,
        }),
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Generate core view endpoint not implemented yet');
          throw new Error('Semantic layer features not yet implemented');
        }
        const error = await response.json();
        throw new Error(error.message || 'Failed to generate core view');
      }

      const result = await response.json();
      devLog('✅ Core view generated successfully:', result);
      return result.semantic_view;
    } catch (error) {
      devError('Error generating core view:', error);
      throw error;
    }
  }

  /**
   * Create custom model extending core model
   */
  async createOrUpdateCustomModel(
    businessEntityId: string,
    coreModelId: string,
    customModelName: string,
    additionalDimensions?: any[],
    additionalMeasures?: any[]
  ): Promise<SemanticModelMetadata> {
    try {
      devLog(`🔄 Creating custom model extending core model`);

      const response = await apiClient(`business-entities/create-custom-model`, {
        method: 'POST',
        body: JSON.stringify({
          business_entity_id: businessEntityId,
          core_model_id: coreModelId,
          custom_model_name: customModelName,
          additional_dimensions: additionalDimensions || [],
          additional_measures: additionalMeasures || [],
        }),
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Create custom model endpoint not implemented yet');
          throw new Error('Semantic layer features not yet implemented');
        }
        const error = await response.json();
        throw new Error(error.message || 'Failed to create custom model');
      }

      const result = await response.json();
      devLog('✅ Custom model created successfully:', result);
      return result.semantic_model;
    } catch (error) {
      devError('Error creating custom model:', error);
      throw error;
    }
  }

  /**
   * Create custom view extending core view
   */
  async createOrUpdateCustomView(
    businessEntityId: string,
    coreViewId: string,
    customViewName: string,
    customModelId?: string,
    additionalColumns?: any[]
  ): Promise<SemanticViewMetadata> {
    try {
      devLog(`🔄 Creating custom view extending core view`);

      const response = await apiClient(`business-entities/create-custom-view`, {
        method: 'POST',
        body: JSON.stringify({
          business_entity_id: businessEntityId,
          core_view_id: coreViewId,
          custom_view_name: customViewName,
          custom_model_id: customModelId,
          additional_columns: additionalColumns || [],
        }),
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Create custom view endpoint not implemented yet');
          throw new Error('Semantic layer features not yet implemented');
        }
        const error = await response.json();
        throw new Error(error.message || 'Failed to create custom view');
      }

      const result = await response.json();
      devLog('✅ Custom view created successfully:', result);
      return result.semantic_view;
    } catch (error) {
      devError('Error creating custom view:', error);
      throw error;
    }
  }

  /**
   * Get core and custom semantic assets for a business entity
   */
  async getSemanticAssets(businessEntityId: string): Promise<CoreSemanticAssets> {
    try {
      devLog(`📦 Fetching semantic assets for entity: ${businessEntityId}`);

      const result = await apiClient(
        `business-entities/${businessEntityId}/semantic-assets`
      );

      // apiClient throws on error, so if we are here, it's success (and result is the JSON body)
      devLog('✅ Semantic assets fetched:', result);
      return (result as any).assets || {};
    } catch (error: any) {
      if (error.message && error.message.includes('404')) {
        devLog('ℹ️ Semantic assets endpoint not implemented yet, returning empty assets');
        return {};
      }
      devError('Error fetching semantic assets:', error);
      return {};
    }
  }

  /**
   * Get AI relationship suggestions based on FK analysis and other signals
   */
  async getRelationshipSuggestions(
    businessEntityId: string,
    _sourceTableNames: string[],
    limit: number = 5
  ): Promise<RelationshipSuggestion[]> {
    try {
      devLog(`🤖 Fetching relationship suggestions for entity: ${businessEntityId}`);

      const params = new URLSearchParams({
        limit: String(limit),
      });

      const response = await apiClient(`relationships/${businessEntityId}/suggestions?${params.toString()}`);

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Relationship suggestions endpoint not found, returning empty suggestions');
          return [];
        }
        throw new Error('Failed to fetch relationship suggestions');
      }

      const result = await response.json();
      devLog('✅ Relationship suggestions fetched:', result);
      return result.suggestions || [];
    } catch (error) {
      devError('Error fetching relationship suggestions:', error);
      return [];
    }
  }

  /**
   * Apply a relationship suggestion by creating edge in catalog
   */
  async applyRelationshipSuggestion(
    suggestion: RelationshipSuggestion
  ): Promise<{ edge_id: string; success: boolean }> {
    try {
      devLog(`✅ Applying relationship suggestion`);

      const response = await apiClient(`business-entities/apply-relationship`, {
        method: 'POST',
        body: JSON.stringify({
          source_entity_id: suggestion.source_entity_id,
          target_entity_id: suggestion.target_entity_id,
          confidence: suggestion.confidence,
          rationale: suggestion.rationale,
          scoring_breakdown: suggestion.scoring_breakdown,
        }),
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Apply relationship endpoint not implemented yet');
          throw new Error('Relationship features not yet implemented');
        }
        throw new Error('Failed to apply relationship');
      }

      const result = await response.json();
      devLog('✅ Relationship applied successfully:', result);
      return result;
    } catch (error) {
      devError('Error applying relationship suggestion:', error);
      throw error;
    }
  }

  /**
   * Get linked models for a given semantic model
   */
  async getLinkedModels(modelId: string): Promise<SemanticModelMetadata[]> {
    try {
      devLog(`🔗 Fetching linked models for model: ${modelId}`);

      const response = await apiClient(
        `semantic-models/${modelId}/linked-models`
      );

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Linked models endpoint not implemented yet, returning empty models');
          return [];
        }
        throw new Error('Failed to fetch linked models');
      }

      const result = await response.json();
      devLog('✅ Linked models fetched:', result);
      return result.models || [];
    } catch (error) {
      devError('Error fetching linked models:', error);
      return [];
    }
  }

  /**
   * Traverse object graph using dot-notation (e.g., Employee.department.company.name)
   */
  async traverseObjectGraph(
    startModelId: string,
    dotPath: string
  ): Promise<{ nodes: any[]; edges: any[] }> {
    try {
      devLog(`🌐 Traversing object graph with path: ${dotPath}`);

      const response = await apiClient(`semantic-models/traverse-graph`, {
        method: 'POST',
        body: JSON.stringify({
          start_model_id: startModelId,
          dot_path: dotPath,
        }),
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Traverse graph endpoint not implemented yet, returning empty graph');
          return { nodes: [], edges: [] };
        }
        throw new Error('Failed to traverse object graph');
      }

      const result = await response.json();
      devLog('✅ Object graph traversal complete:', result);
      return result.graph || { nodes: [], edges: [] };
    } catch (error) {
      devError('Error traversing object graph:', error);
      return { nodes: [], edges: [] };
    }
  }

  /**
   * Get related objects for a business entity (links to and links from)
   */
  async getRelatedObjects(
    businessEntityId: string
  ): Promise<{ linksTo: SemanticModelMetadata[]; linksFrom: SemanticModelMetadata[] }> {
    try {
      devLog(`🔄 Fetching related objects for entity: ${businessEntityId}`);

      const result = await apiClient(
        `business-entities/${businessEntityId}/related-objects`
      );

      devLog('✅ Related objects fetched:', result);
      return (result as any).related || { linksTo: [], linksFrom: [] };
    } catch (error: any) {
      if (error.message && error.message.includes('404')) {
        devLog('ℹ️ Related objects endpoint not implemented yet, returning empty related objects');
        return { linksTo: [], linksFrom: [] };
      }
      devError('Error fetching related objects:', error);
      return { linksTo: [], linksFrom: [] };
    }
  }

  /**
   * Generate semantic mappings (abbreviations, standard terms) for the current datasource
   */
  async generateSemanticMappings(): Promise<any> {
    try {
      devLog(`🔮 Generating semantic mappings`);

      const response = await apiClient(`semantic-mapping/generate`, {
        method: 'POST',
      });

      if (!response.ok) {
        if (response.status === 404) {
          devLog('ℹ️ Generate mappings endpoint not found');
          throw new Error('Semantic mapping feature not available');
        }
        const error = await response.json().catch(() => ({ message: 'Unknown error' }));
        throw new Error(error.message || 'Failed to generate mappings');
      }

      const result = await response.json();
      devLog('✅ Semantic mappings generated:', result);
      return result;
    } catch (error) {
      devError('Error generating semantic mappings:', error);
      throw error;
    }
  }

  /**
   * Apply selected semantic mappings
   */
  async applySemanticMappings(mappings: MappingResult[]): Promise<{ applied_count: number }> {
    try {
      devLog(`🚀 Applying ${mappings.length} mappings`);

      const response = await apiClient(`semantic-mapping/apply`, {
        method: 'POST',
        body: JSON.stringify({ mappings }),
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({ message: 'Unknown error' }));
        throw new Error(error.message || 'Failed to apply mappings');
      }

      const result = await response.json();
      devLog('✅ Semantic mappings applied:', result);
      return result;
    } catch (error) {
      devError('Error applying semantic mappings:', error);
      throw error;
    }
  }

}

export default BusinessEntitySemanticService;
