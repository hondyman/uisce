import { useState, useEffect, useCallback } from 'react';
import type { ModelCatalogNode } from '../types/model';
import { useAuth } from '../contexts/AuthContext';
import { useAuthFetch } from '../utils/authFetch';
import { devLog, devError } from '../utils/devLogger';
import resolveApiUrl from '../utils/resolveApiUrl';

interface UseModelCatalogResult {
  models: ModelCatalogNode[];
  selectedModel: ModelCatalogNode | null;
  setSelectedModel: (model: ModelCatalogNode | null) => void;
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  loading: boolean;
  error: Error | null;
  createCustomModel: (baseModelKey: string) => Promise<any>;
  cloneModel: (baseModelKey: string) => Promise<any>;
  refreshModels: () => Promise<void>;
  updateModel: (modelId: string, updates: Partial<ModelCatalogNode>) => Promise<void>;
  deleteModel: (modelId: string, isCore?: boolean, modelKey?: string) => Promise<{ success: boolean; error?: Error }>;
}

export const useModelCatalog = (tenantId: string, datasourceId: string): UseModelCatalogResult => {
  const [models, setModels] = useState<ModelCatalogNode[]>([]);
  const [selectedModel, setSelectedModel] = useState<ModelCatalogNode | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const { getValidToken, isLoading: authLoading } = useAuth();
  const { authFetch } = useAuthFetch();

  const DEV_ALLOW_UNAUTH_MODELS = ((import.meta.env.VITE_DEV_ALLOW_UNAUTH_MODELS as string) ?? 'true') === 'true';

  const fetchModels = useCallback(async () => {
    // Skip until we have real IDs
    if (!tenantId || tenantId === 'skip' || !datasourceId || datasourceId === 'skip') return;
    if (authLoading) return; // wait until auth state is known
  setLoading(true);
  setError(null);
  try {
      const validToken = await getValidToken();
      if (!validToken && !DEV_ALLOW_UNAUTH_MODELS) {
        // Guard: don't call API without a token unless dev bypass enabled
        setModels([]);
        setError(new Error('Not authenticated'));
        return;
      }

      const resp = await authFetch<{ models: ModelCatalogNode[] }>(
        resolveApiUrl(`/api/models?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`)
      );

  if (!resp.ok) {
  if (resp.status === 401 && !DEV_ALLOW_UNAUTH_MODELS) {
          setError(new Error('Not authenticated'));
          return;
        }
        devError('Failed to fetch models:', resp.error || resp.status);
        setModels([]);
        return;
      }

      setModels(Array.isArray(resp.data?.models) ? resp.data.models : []);
    } catch (err) {
      devError('Error fetching models:', err);
  const e = err instanceof Error ? err : new Error('Unknown error');
  setError(e);
    } finally {
      setLoading(false);
    }
    // tenantId/datasourceId intentionally included; authFetch/getValidToken/logout are stable hooks
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, datasourceId, authLoading]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  const createCustomModel = useCallback(async (baseModelKey: string) => {
    setLoading(true);
    setError(null);
    try {
  const validToken = await getValidToken();
  if (!validToken && !DEV_ALLOW_UNAUTH_MODELS) throw new Error('Not authenticated');

      const resp = await authFetch(
        resolveApiUrl('/api/models/custom'),
        { method: 'POST', json: { tenant_id: tenantId, tenant_instance_id: datasourceId, base_model_key: baseModelKey } }
      );

      if (!resp.ok) {
        devError('Failed to create custom model:', resp.error || resp.status);
        throw new Error(resp.error || `Create failed: ${resp.status}`);
      }

      const newModel = resp.data;
      setModels(prev => [...prev, newModel]);
      setSelectedModel(newModel as ModelCatalogNode);
      return newModel;
    } catch (err) {
      devError('Error creating custom model:', err);
  const e = err instanceof Error ? err : new Error('Failed to create custom model');
      setError(e);
      throw err;
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, datasourceId]);

  const cloneModel = useCallback(async (baseModelKey: string) => {
    setLoading(true);
    setError(null);
    try {
  const validToken = await getValidToken();
  if (!validToken && !DEV_ALLOW_UNAUTH_MODELS) throw new Error('Not authenticated');

      // Accept either a UUID id, a model_key, a model_key_custom, or a display_name.
      const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
      let resolvedModelId: string | undefined = undefined;

      if (uuidRegex.test(baseModelKey)) {
        resolvedModelId = baseModelKey;
      } else {
        const found = models.find(m => 
          m.id === baseModelKey ||
          m.model_key === baseModelKey ||
          m.model_key === `${baseModelKey}_custom` ||
          m.display_name === baseModelKey
        );
        if (found && found.id) resolvedModelId = found.id;
      }

      if (!resolvedModelId) {
        const e = new Error(`Base model not found: ${baseModelKey}`);
        devError('Base model not found for clone', baseModelKey);
        setError(e);
        throw e;
      }

      const resp = await authFetch(
        resolveApiUrl(`/api/models/clone?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`),
        { method: 'POST', json: { model_id: resolvedModelId } }
      );

      if (!resp.ok) {
        devError('Failed to clone model:', resp.error || resp.status);
        throw new Error(resp.error || `Clone failed: ${resp.status}`);
      }

      const cloned = resp.data;
      setModels(prev => [...prev, cloned]);
      setSelectedModel(cloned as ModelCatalogNode);
      devLog('Model cloned', cloned);
      return cloned;
    } catch (err) {
      devError('Error cloning model:', err);
  const e = err instanceof Error ? err : new Error('Failed to clone model');
      setError(e);
      throw err;
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, datasourceId, models]);

  const updateModel = useCallback(async (modelId: string, updates: Partial<ModelCatalogNode>) => {
    setLoading(true);
    setError(null);
    try {
  const validToken = await getValidToken();
  if (!validToken && !DEV_ALLOW_UNAUTH_MODELS) throw new Error('Not authenticated');

  const updateUrlObj = new URL(resolveApiUrl(`/api/models/${modelId}`));
      if (tenantId) updateUrlObj.searchParams.set('tenant_id', tenantId);
      if (datasourceId) updateUrlObj.searchParams.set('tenant_instance_id', datasourceId);
      const resp = await authFetch(
        updateUrlObj.toString(),
        { method: 'PATCH', json: updates }
      );

      if (!resp.ok) {
        if (resp.status === 401) throw new Error('Authentication expired. Please log in again.');
        throw new Error(resp.error || `Update failed: ${resp.status}`);
      }

      const updatedModel = resp.data;
  // Ensure we prefer `display_name` in the UI—backend sometimes returns `title`.
  const asRec = (v: unknown) => (v && typeof v === 'object' && !Array.isArray(v) ? v as Record<string, unknown> : {} as Record<string, unknown>);
  const uiRec = asRec(updatedModel);
  const displayName = uiRec.display_name ?? uiRec.title;
  const uiModelPatch: Partial<ModelCatalogNode> = {
    ...(uiRec as Partial<ModelCatalogNode>),
    ...(displayName ? { display_name: String(displayName) } : {}),
  };
  setModels(prev => prev.map(m => (m.id === modelId ? { ...m, ...uiModelPatch } : m)));
  if (selectedModel?.id === modelId) setSelectedModel({ ...selectedModel, ...uiModelPatch } as ModelCatalogNode);
  // Refresh full list to pick up any server-side computed changes and related derived data
  try { await fetchModels(); } catch (err) { devError('Failed to refresh models after update', err); }
  return updatedModel;
    } catch (err) {
      devError('Error updating model:', err);
  const e = err instanceof Error ? err : new Error('Failed to update model');
      setError(e);
      throw err;
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, datasourceId, models, selectedModel]);

  const deleteModel = useCallback(async (modelId: string, _isCore?: boolean, _modelKey?: string) => {
    // Accept either a UUID model id or a model_key/display name; try to resolve
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
    let resolvedId = modelId;
    if (!uuidRegex.test(modelId)) {
      // Try to resolve from known models by id, model_key, or display_name
      const found = models.find(m => m.id === modelId || m.model_key === modelId || m.model_key === `${modelId}_custom` || m.display_name === modelId);
      if (found && found.id) {
        resolvedId = found.id;
      } else if (_modelKey) {
        const byKey = models.find(m => m.model_key === _modelKey || m.model_key === `${_modelKey}_custom`);
        if (byKey && byKey.id) resolvedId = byKey.id;
      }
      if (!uuidRegex.test(resolvedId)) {
        const e = new Error(`Invalid model ID or unknown model key: ${modelId}`);
        devError('Invalid model ID', modelId);
        setError(e);
        return { success: false, error: e };
      }
    }

    setLoading(true);
    setError(null);
    try {
  const validToken = await getValidToken();
  if (!validToken && !DEV_ALLOW_UNAUTH_MODELS) throw new Error('Not authenticated');

  const deleteUrlObj = new URL(resolveApiUrl(`/api/models/${resolvedId}`));
      if (tenantId) deleteUrlObj.searchParams.set('tenant_id', tenantId);
      if (datasourceId) deleteUrlObj.searchParams.set('tenant_instance_id', datasourceId);
      const resp = await authFetch(
        deleteUrlObj.toString(),
        { method: 'DELETE' }
      );

      if (!resp.ok) {
        if (resp.status === 404) {
          try { await fetchModels(); } catch (err) { devError('Refresh after 404 failed', err); }
          // Ensure we remove the model locally in case server list is stale
          setModels(prev => prev.filter(m => m.id !== resolvedId));
          if (selectedModel?.id === modelId || selectedModel?.id === resolvedId) setSelectedModel(null);
          return { success: true };
        }
        // If backend rejects the provided ID as invalid, attempt to resolve again and retry once
        if (resp.status === 400 && (resp.error || '').toString().toLowerCase().includes('invalid model id')) {
          devLog('Backend reported invalid model id; attempting to resolve by key and retry', { modelId, resolvedId });
          // Try to resolve by modelKey/display_name and retry
          const altFound = models.find(m => m.model_key === modelId || m.model_key === `${modelId}_custom` || m.display_name === modelId || m.id === modelId);
          if (altFound && altFound.id && altFound.id !== resolvedId) {
            const retryId = altFound.id;
            const retryDeleteUrlObj = new URL(resolveApiUrl(`/api/models/${retryId}`));
            if (tenantId) retryDeleteUrlObj.searchParams.set('tenant_id', tenantId);
            if (datasourceId) retryDeleteUrlObj.searchParams.set('tenant_instance_id', datasourceId);
            const retryResp = await authFetch(
              retryDeleteUrlObj.toString(),
              { method: 'DELETE' }
            );
            if (retryResp.ok) {
              try { await fetchModels(); } catch (err) { devError('Refresh after delete retry failed', err); }
              // Remove locally
              setModels(prev => prev.filter(m => m.id !== retryId));
              if (selectedModel?.id === modelId || selectedModel?.id === resolvedId || selectedModel?.id === retryId) setSelectedModel(null);
              return { success: true };
            }
          }
        }
        devError('Failed to delete model', resp.error || resp.status);
        throw new Error(resp.error || `Delete failed: ${resp.status}`);
      }

    // Optimistically remove locally and notify listeners immediately so the UI updates fast.
    setModels(prev => prev.filter(m => m.id !== resolvedId));
    devLog('Optimistically removed model locally', { resolvedId });
    try { window.dispatchEvent(new CustomEvent('model.deleted', { detail: { id: resolvedId } })); } catch {}
    if (selectedModel?.id === modelId || selectedModel?.id === resolvedId) setSelectedModel(null);

    // Refresh authoritative list in background. If this fails, we'll log but keep the optimistic change so the UI isn't stuck.
    try { await fetchModels(); } catch (err) { devError('Refresh after delete failed', err); }
      return { success: true };
    } catch (err) {
      devError('Error deleting model', err);
  const e = err instanceof Error ? err : new Error('Failed to delete model');
      setError(e);
      return { success: false, error: e };
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, datasourceId, models, selectedModel]);

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
    cloneModel,
    refreshModels,
    updateModel,
    deleteModel,
  };
};