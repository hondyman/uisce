/**
 * useBusinessEntitySemanticLayer Hook
 * 
 * Provides integration between business entities and semantic layer
 * for core/custom model and view generation and management.
 */

import { useState, useCallback, useEffect } from 'react';
import BusinessEntitySemanticService, {
  SemanticModelMetadata,
  SemanticViewMetadata,
  CoreSemanticAssets,
  RelationshipSuggestion,
} from '../services/businessEntitySemanticService';
import { devLog, devError } from '../utils/devLogger';

interface UseBusinessEntitySemanticLayerOptions {
  tenantId: string;
  datasourceId: string;
  businessEntityId: string;
  businessEntityName: string;
  semanticTermIds: string[];
  sourceTableNames: string[];
}

export function useBusinessEntitySemanticLayer(options: UseBusinessEntitySemanticLayerOptions) {
  const { tenantId, datasourceId, businessEntityId, businessEntityName, semanticTermIds, sourceTableNames } =
    options;

  const [service] = useState(
    () => new BusinessEntitySemanticService(tenantId, datasourceId)
  );

  // State
  const [semanticAssets, setSemanticAssets] = useState<CoreSemanticAssets>({});
  const [relationshipSuggestions, setRelationshipSuggestions] = useState<RelationshipSuggestion[]>([]);
  const [linkedModels, setLinkedModels] = useState<SemanticModelMetadata[]>([]);
  const [relatedObjects, setRelatedObjects] = useState<{
    linksTo: SemanticModelMetadata[];
    linksFrom: SemanticModelMetadata[];
  }>({ linksTo: [], linksFrom: [] });

  // Loading states
  const [assetsLoading, setAssetsLoading] = useState(false);
  const [suggestionsLoading, setSuggestionsLoading] = useState(false);
  const [linkedModelsLoading, setLinkedModelsLoading] = useState(false);
  const [relatedObjectsLoading, setRelatedObjectsLoading] = useState(false);
  const [modelGenerationLoading, setModelGenerationLoading] = useState(false);
  const [viewGenerationLoading, setViewGenerationLoading] = useState(false);

  // Error states
  const [assetsError, setAssetsError] = useState<Error | null>(null);
  const [suggestionsError, setSuggestionsError] = useState<Error | null>(null);
  const [modelError, setModelError] = useState<Error | null>(null);
  const [viewError, setViewError] = useState<Error | null>(null);

  // Fetch semantic assets on mount or when entity changes
  useEffect(() => {
    if (businessEntityId) {
      fetchSemanticAssets();
    }
  }, [businessEntityId]);

  // Fetch relationship suggestions when entity changes
  useEffect(() => {
    if (businessEntityId && sourceTableNames.length > 0) {
      fetchRelationshipSuggestions();
    }
  }, [businessEntityId, sourceTableNames]);

  // Fetch linked models when core model is available
  useEffect(() => {
    if (semanticAssets.coreModel?.id) {
      fetchLinkedModels(semanticAssets.coreModel.id);
    }
  }, [semanticAssets.coreModel?.id]);

  // Fetch related objects
  useEffect(() => {
    if (businessEntityId) {
      fetchRelatedObjects();
    }
  }, [businessEntityId]);

  const fetchSemanticAssets = useCallback(async () => {
    setAssetsLoading(true);
    setAssetsError(null);
    try {
      const assets = await service.getSemanticAssets(businessEntityId);
      setSemanticAssets(assets);
      devLog('Semantic assets loaded:', assets);
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Failed to load semantic assets');
      setAssetsError(err);
      devError('Error loading semantic assets:', error);
    } finally {
      setAssetsLoading(false);
    }
  }, [businessEntityId, service]);

  const fetchRelationshipSuggestions = useCallback(async () => {
    setSuggestionsLoading(true);
    setSuggestionsError(null);
    try {
      const suggestions = await service.getRelationshipSuggestions(
        businessEntityId,
        sourceTableNames
      );
      setRelationshipSuggestions(suggestions);
      devLog('Relationship suggestions loaded:', suggestions);
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Failed to load suggestions');
      setSuggestionsError(err);
      devError('Error loading relationship suggestions:', error);
    } finally {
      setSuggestionsLoading(false);
    }
  }, [businessEntityId, sourceTableNames, service]);

  const fetchLinkedModels = useCallback(async (modelId: string) => {
    setLinkedModelsLoading(true);
    try {
      const models = await service.getLinkedModels(modelId);
      setLinkedModels(models);
      devLog('Linked models loaded:', models);
    } catch (error) {
      devError('Error loading linked models:', error);
    } finally {
      setLinkedModelsLoading(false);
    }
  }, [service]);

  const fetchRelatedObjects = useCallback(async () => {
    setRelatedObjectsLoading(true);
    try {
      const related = await service.getRelatedObjects(businessEntityId);
      setRelatedObjects(related);
      devLog('Related objects loaded:', related);
    } catch (error) {
      devError('Error loading related objects:', error);
    } finally {
      setRelatedObjectsLoading(false);
    }
  }, [businessEntityId, service]);

  // Actions
  const generateCoreModel = useCallback(async (): Promise<SemanticModelMetadata | null> => {
    setModelGenerationLoading(true);
    setModelError(null);
    try {
      const model = await service.generateOrUpdateCoreModel(
        businessEntityId,
        businessEntityName,
        semanticTermIds,
        sourceTableNames
      );
      setSemanticAssets((prev) => ({ ...prev, coreModel: model }));
      devLog('Core model generated:', model);
      return model;
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Failed to generate core model');
      setModelError(err);
      devError('Error generating core model:', error);
      return null;
    } finally {
      setModelGenerationLoading(false);
    }
  }, [businessEntityId, businessEntityName, semanticTermIds, sourceTableNames, service]);

  const generateCoreView = useCallback(async (): Promise<SemanticViewMetadata | null> => {
    if (!semanticAssets.coreModel?.id) {
      const err = new Error('Core model must exist before generating view');
      setViewError(err);
      return null;
    }

    setViewGenerationLoading(true);
    setViewError(null);
    try {
      const view = await service.generateOrUpdateCoreView(
        businessEntityId,
        businessEntityName,
        semanticAssets.coreModel.id,
        semanticTermIds
      );
      setSemanticAssets((prev) => ({ ...prev, coreView: view }));
      devLog('Core view generated:', view);
      return view;
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Failed to generate core view');
      setViewError(err);
      devError('Error generating core view:', error);
      return null;
    } finally {
      setViewGenerationLoading(false);
    }
  }, [businessEntityId, businessEntityName, semanticTermIds, semanticAssets.coreModel, service]);

  const createCustomModel = useCallback(
    async (customModelName: string, additionalDimensions?: any[], additionalMeasures?: any[]) => {
      if (!semanticAssets.coreModel?.id) {
        const err = new Error('Core model must exist before creating custom model');
        setModelError(err);
        return null;
      }

      setModelGenerationLoading(true);
      setModelError(null);
      try {
        const model = await service.createOrUpdateCustomModel(
          businessEntityId,
          semanticAssets.coreModel.id,
          customModelName,
          additionalDimensions,
          additionalMeasures
        );
        setSemanticAssets((prev) => ({ ...prev, customModel: model }));
        devLog('Custom model created:', model);
        return model;
      } catch (error) {
        const err = error instanceof Error ? error : new Error('Failed to create custom model');
        setModelError(err);
        devError('Error creating custom model:', error);
        return null;
      } finally {
        setModelGenerationLoading(false);
      }
    },
    [businessEntityId, semanticAssets.coreModel, service]
  );

  const createCustomView = useCallback(
    async (customViewName: string, customModelId?: string, additionalColumns?: any[]) => {
      if (!semanticAssets.coreView?.id) {
        const err = new Error('Core view must exist before creating custom view');
        setViewError(err);
        return null;
      }

      setViewGenerationLoading(true);
      setViewError(null);
      try {
        const view = await service.createOrUpdateCustomView(
          businessEntityId,
          semanticAssets.coreView.id,
          customViewName,
          customModelId || semanticAssets.customModel?.id,
          additionalColumns
        );
        setSemanticAssets((prev) => ({ ...prev, customView: view }));
        devLog('Custom view created:', view);
        return view;
      } catch (error) {
        const err = error instanceof Error ? error : new Error('Failed to create custom view');
        setViewError(err);
        devError('Error creating custom view:', error);
        return null;
      } finally {
        setViewGenerationLoading(false);
      }
    },
    [businessEntityId, semanticAssets.coreView, semanticAssets.customModel, service]
  );

  const applyRelationshipSuggestion = useCallback(
    async (suggestion: RelationshipSuggestion) => {
      try {
        const result = await service.applyRelationshipSuggestion(suggestion);
        devLog('Relationship applied:', result);
        // Refresh suggestions after applying one
        await fetchRelationshipSuggestions();
        return result;
      } catch (error) {
        devError('Error applying relationship:', error);
        throw error;
      }
    },
    [service, fetchRelationshipSuggestions]
  );

  const traverseObjectGraph = useCallback(
    async (startModelId: string, dotPath: string) => {
      try {
        const graph = await service.traverseObjectGraph(startModelId, dotPath);
        devLog('Object graph traversed:', graph);
        return graph;
      } catch (error) {
        devError('Error traversing object graph:', error);
        throw error;
      }
    },
    [service]
  );

  return {
    // State
    semanticAssets,
    relationshipSuggestions,
    linkedModels,
    relatedObjects,

    // Loading states
    assetsLoading,
    suggestionsLoading,
    linkedModelsLoading,
    relatedObjectsLoading,
    modelGenerationLoading,
    viewGenerationLoading,

    // Error states
    assetsError,
    suggestionsError,
    modelError,
    viewError,

    // Actions
    generateCoreModel,
    generateCoreView,
    createCustomModel,
    createCustomView,
    applyRelationshipSuggestion,
    traverseObjectGraph,
    fetchRelationshipSuggestions,
    fetchSemanticAssets,
    fetchLinkedModels,
    fetchRelatedObjects,
  };
}

export default useBusinessEntitySemanticLayer;
