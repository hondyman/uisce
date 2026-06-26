/**
 * Semantic Objects Management Hook
 *
 * Custom hook for fetching and managing semantic objects
 */

import { useState, useEffect, useCallback } from 'react';
import { useTenant } from '../../contexts/TenantContext';
import { useAuthFetch } from '../../utils/authFetch';
import { useValidationErrors } from '../../hooks/useValidationErrors';
import { SemanticObjectReference } from '../../types/bundles';

export const useSemanticObjects = () => {
  const [allObjects, setAllObjects] = useState<SemanticObjectReference[]>([]);
  const [loadingObjects, setLoadingObjects] = useState<boolean>(true);
  const [objectsError, setObjectsError] = useState<string | null>(null);
  const [hasFetchedObjects, setHasFetchedObjects] = useState<boolean>(false);
  const [fetchKey, setFetchKey] = useState(0);

  const { tenant, datasource } = useTenant();
  const { authFetch } = useAuthFetch();
  const { handleResponseError } = useValidationErrors();

  const tenantId = tenant?.id?.trim() ?? '';
  const datasourceId = (datasource?.id ?? datasource?.alpha_datasource?.datasource_name ?? '').trim();
  const selectionMissing = !tenantId || !datasourceId;

  const handleRefreshObjects = useCallback(() => {
    setFetchKey((prev) => prev + 1);
  }, []);

  useEffect(() => {
    const controller = new AbortController();

    if (selectionMissing) {
      setAllObjects([]);
      setObjectsError('Select a tenant and datasource to browse views.');
      setLoadingObjects(false);
      setHasFetchedObjects(false);
      return () => controller.abort();
    }

    const fetchAllViews = async () => {
      setLoadingObjects(true);
      setObjectsError(null);
      setHasFetchedObjects(false);

      try {
        const params = new URLSearchParams();
        params.set('tenant_id', tenantId);
        params.set('tenant_instance_id', datasourceId);
        params.set('page_size', '200');

        const result = await authFetch<any>(`/api/views?${params.toString()}`, {
          signal: controller.signal,
          cache: 'no-store'
        });

        if (!result.ok) {
          await handleResponseError(result.response, 'Failed to fetch views');
        }

        // Handle 304 Not Modified
        if (result.status === 304) {
          if (allObjects && allObjects.length > 0) {
            setLoadingObjects(false);
            setHasFetchedObjects(true);
            return;
          }

          // Try cache-busting retry
          try {
            const bustParams = new URLSearchParams(params.toString());
            bustParams.set('_', String(Date.now()));
            const bustRes = await authFetch<any>(`/api/views?${bustParams.toString()}`, {
              signal: controller.signal,
              cache: 'no-store'
            });

            if (bustRes.ok && bustRes.status !== 304) {
              const views = Array.isArray(bustRes.data?.views) ? bustRes.data.views : [];
              const mapped = views.map((view: any) => ({
                id: view.id || view.name,
                name: view.name,
                title: view.title,
                description: view.description,
                type: 'view',
                modelId: view.id || view.name
              }));
              setAllObjects(mapped);
              setLoadingObjects(false);
              setHasFetchedObjects(true);
              return;
            }
          } catch (e) {
            // fall through to error handling
          }

          setLoadingObjects(false);
          setHasFetchedObjects(true);
          return;
        }

        // Map API response to SemanticObjectReference[]
        const views = Array.isArray(result.data?.views) ? result.data.views : [];
        const mapped = views.map((view: any) => ({
          id: view.id || view.name,
          name: view.name,
          title: view.title,
          description: view.description,
          type: 'view',
          modelId: view.id || view.name
        }));

        setAllObjects(mapped);
      } catch (err: any) {
        if (err?.name === 'AbortError') {
          return;
        }
        setObjectsError(err?.message || 'Failed to fetch views');
      } finally {
        if (!controller.signal.aborted) {
          setLoadingObjects(false);
          setHasFetchedObjects(true);
        }
      }
    };

    fetchAllViews();

    return () => controller.abort();
  }, [tenantId, datasourceId, selectionMissing, fetchKey, authFetch, allObjects]);

  return {
    allObjects,
    loadingObjects,
    objectsError,
    hasFetchedObjects,
    handleRefreshObjects,
    tenantId,
    datasourceId,
    selectionMissing,
  };
};