import { useState, useEffect, useCallback } from 'react';
import type { ModelCatalogNode } from './model';
import { useAuth } from '../../../frontend/src/contexts/AuthContext';

interface UseModelCatalogResult {
  models: ModelCatalogNode[];
  selectedModel: ModelCatalogNode | null;
  setSelectedModel: (model: ModelCatalogNode | null) => void;
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  loading: boolean;
  error: Error | null;
  createCustomModel: (baseModelKey: string) => Promise<void>;
  refreshModels: () => Promise<void>;
  updateModel: (modelId: string, updates: Partial<ModelCatalogNode>) => Promise<void>;
  deleteModel: (modelId: string) => Promise<void>;
}

export const useModelCatalog = (
  tenantId: string,
  datasourceId: string
): UseModelCatalogResult => {
  const [models, setModels] = useState<ModelCatalogNode[]>([]);
  const [selectedModel, setSelectedModel] = useState<ModelCatalogNode | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const { user } = useAuth();

  // API base URL - adjust to match your backend
  const API_BASE = (import.meta as any).env?.VITE_API_BASE || '/api';

  // Fetch models from the backend
  const fetchModels = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(
        // The original endpoint /tenants/... was not found (404).
        // We are adapting the existing /fabric/models endpoint to serve the catalog data.
        // This endpoint only requires datasource_id, as tenant_id can be derived on the backend.
        `${API_BASE}/fabric/models?datasource_id=${datasourceId}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            // Add any auth headers as needed
            // 'Authorization': `Bearer ${token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch models: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      setModels(data.models || []);
    } catch (err) {
      console.error('Error fetching models:', err);
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, API_BASE]);

  // Load models on component mount and when dependencies change
  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  // Create a custom model
  const createCustomModel = useCallback(async (baseModelKey: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${API_BASE}/tenants/${tenantId}/datasources/${datasourceId}/models/custom`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-User-ID': user?.id || '',
            // Add any auth headers as needed
          },
          body: JSON.stringify({
            base_model_key: baseModelKey,
          }),
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to create custom model: ${response.status} ${response.statusText}`);
      }

      const newModel = await response.json();
      
      // Update local state
      setModels(prev => [...prev, newModel]);
      setSelectedModel(newModel);
      
      return newModel;
    } catch (err) {
      console.error('Error creating custom model:', err);
      setError(err instanceof Error ? err : new Error('Failed to create custom model'));
      throw err;
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, API_BASE, user]);

  // Update a model
  const updateModel = useCallback(async (modelId: string, updates: Partial<ModelCatalogNode>) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${API_BASE}/tenants/${tenantId}/datasources/${datasourceId}/models/${modelId}`,
        {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            'X-User-ID': user?.id || '',
          },
          body: JSON.stringify(updates),
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to update model: ${response.status} ${response.statusText}`);
      }

      const updatedModel = await response.json();
      
      // Update local state
      setModels(prev => 
        prev.map(model => 
          model.id === modelId ? { ...model, ...updatedModel } : model
        )
      );
      
      // Update selected model if it's the one being updated
      if (selectedModel?.id === modelId) {
        setSelectedModel({ ...selectedModel, ...updatedModel });
      }
      
      return updatedModel;
    } catch (err) {
      console.error('Error updating model:', err);
      setError(err instanceof Error ? err : new Error('Failed to update model'));
      throw err;
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, selectedModel, API_BASE, user]);

  // Delete a model
  const deleteModel = useCallback(async (modelId: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${API_BASE}/tenants/${tenantId}/datasources/${datasourceId}/models/${modelId}`,
        {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json',
            'X-User-ID': user?.id || '',
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to delete model: ${response.status} ${response.statusText}`);
      }

      // Update local state
      setModels(prev => prev.filter(model => model.id !== modelId));
      
      // Clear selected model if it's the one being deleted
      if (selectedModel?.id === modelId) {
        setSelectedModel(null);
      }
    } catch (err) {
      console.error('Error deleting model:', err);
      setError(err instanceof Error ? err : new Error('Failed to delete model'));
      throw err;
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, selectedModel, API_BASE, user]);

  // Refresh models (alias for fetchModels)
  const refreshModels = useCallback(async () => {
    await fetchModels();
  }, [fetchModels]);

  return {
    models,
    selectedModel,
    setSelectedModel,
    searchTerm,
    setSearchTerm,
    loading,
    error,
    createCustomModel,
    refreshModels,
    updateModel,
    deleteModel,
  };
};