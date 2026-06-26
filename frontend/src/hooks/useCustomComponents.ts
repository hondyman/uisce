import { useState, useEffect, useCallback } from 'react';
import { devError } from '../utils/devLogger';
import { useTenant } from '../contexts/TenantContext';
import { customComponentService } from '../services/customComponentService';
import { CustomComponent } from '../components/CustomComponentManager/CustomComponentManager';

export interface UseCustomComponentsReturn {
  components: CustomComponent[];
  loading: boolean;
  error: string | null;
  addComponent: (type: string) => void;
  updateComponent: (id: string, component: CustomComponent) => Promise<void>;
  deleteComponent: (id: string) => Promise<void>;
  saveComponent: (component: CustomComponent) => Promise<void>;
  refreshComponents: () => Promise<void>;
}

export const useCustomComponents = (): UseCustomComponentsReturn => {
  const { tenant, datasource } = useTenant();
  const [components, setComponents] = useState<CustomComponent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch components
  const fetchComponents = useCallback(async () => {
    if (!tenant || !datasource) {
      setComponents([]);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await customComponentService.listComponents(
        tenant.id,
        datasource.id
      );
      setComponents(data || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load components';
      setError(message);
      devError('Error fetching components:', err);
    } finally {
      setLoading(false);
    }
  }, [tenant, datasource]);

  // Refresh on tenant/datasource change
  useEffect(() => {
    fetchComponents();
  }, [fetchComponents]);

  // Add component
  const addComponent = useCallback((type: string) => {
    // Narrow incoming `type` to the known CustomComponent types to avoid `any`.
    const allowed = new Set<CustomComponent['type']>([
      'web_component', 'iframe', 'api_integration', 'custom_widget', 'chart', 'custom_code'
    ]);
    const compType: CustomComponent['type'] = (allowed.has(type as CustomComponent['type']) ? (type as CustomComponent['type']) : 'custom_widget');

    const newComponent: CustomComponent = {
      id: `comp_${Date.now()}`,
      name: `New Component`,
      type: compType,
      config: {},
      events: [],
      filters: [],
      tenantId: tenant?.id,
      datasourceId: datasource?.id,
    };
    setComponents(prev => [...prev, newComponent]);
  }, [tenant, datasource]);

  // Update component locally
  const updateComponent = useCallback(async (id: string, updated: CustomComponent) => {
    setComponents(prev => prev.map(c => c.id === id ? updated : c));
    // Auto-save after 1 second of no changes
    try {
      await customComponentService.updateComponent(
        tenant!.id,
        datasource!.id,
        updated
      );
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update component';
      setError(message);
      devError('Error updating component:', err);
    }
  }, [tenant, datasource]);

  // Delete component
  const deleteComponent = useCallback(async (id: string) => {
    setComponents(prev => prev.filter(c => c.id !== id));
    try {
      await customComponentService.deleteComponent(
        tenant!.id,
        datasource!.id,
        id
      );
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete component';
      setError(message);
      devError('Error deleting component:', err);
      // Re-fetch to restore state
      await fetchComponents();
    }
  }, [tenant, datasource, fetchComponents]);

  // Save component
  const saveComponent = useCallback(async (component: CustomComponent) => {
    try {
      await customComponentService.createComponent(
        tenant!.id,
        datasource!.id,
        component
      );
      await fetchComponents();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save component';
      setError(message);
      devError('Error saving component:', err);
      throw err;
    }
  }, [tenant, datasource, fetchComponents]);

  return {
    components,
    loading,
    error,
    addComponent,
    updateComponent,
    deleteComponent,
    saveComponent,
    refreshComponents: fetchComponents,
  };
};
