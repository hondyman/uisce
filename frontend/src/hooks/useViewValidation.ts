import { useCallback } from 'react';
import { devError } from '../utils/devLogger';
import { assertValidViewName } from '../utils/viewNameValidation';

interface UseViewValidationProps {
  tenantId?: string;
  datasourceId?: string;
}

interface ValidationResult {
  valid: boolean;
  issues: Array<{
    level: 'error' | 'warning' | 'info';
    message: string;
    path?: string;
  }>;
}

export const useViewValidation = ({ tenantId, datasourceId }: UseViewValidationProps) => {
  const validateView = useCallback(async (_viewName: string, viewData: any): Promise<ValidationResult> => {
    try {
      const params = new URLSearchParams();
      if (tenantId) params.append('tenant_id', tenantId);
      if (datasourceId) params.append('tenant_instance_id', datasourceId);

      const response = await fetch(`/api/views/validate?${params}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(viewData),
      });

      if (!response.ok) {
        throw new Error(`Validation failed: ${response.statusText}`);
      }

      const result = await response.json();
      
      // Transform backend validation result to our expected format
      return {
        valid: result.issues?.length === 0,
        issues: result.issues?.map((issue: any) => ({
          level: issue.severity || issue.level || 'error',
          message: issue.message || issue.description || 'Unknown validation issue',
          path: issue.path || issue.location
        })) || []
      };
    } catch (error) {
      try { devError('View validation error:', error); } catch {}
      return {
        valid: false,
        issues: [{
          level: 'error' as const,
          message: `Validation request failed: ${error}`,
          path: undefined
        }]
      };
    }
  }, [tenantId, datasourceId]);

  const saveView = useCallback(async (viewName: string, viewData: any): Promise<any> => {
    // Validate view name before making API call
    assertValidViewName(viewName);

    try {
      const params = new URLSearchParams();
      if (tenantId) params.append('tenant_id', tenantId);
      if (datasourceId) params.append('tenant_instance_id', datasourceId);

      // Use the canonical view name for the endpoint. If the caller provided a view object
      // with a 'name' field, prefer that as the resource identifier. This prevents accidental
      // upserts using UUIDs when editing by id.
      const targetName = (viewData && typeof viewData.name === 'string' && viewData.name.trim()) ? String(viewData.name) : viewName;
      const response = await fetch(`/api/views/${encodeURIComponent(targetName)}?${params}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(viewData),
      });

      if (!response.ok) {
        throw new Error(`Save failed: ${response.statusText}`);
      }
      // Return parsed server response to allow caller to use canonical id
      try {
        const body = await response.json();
        return body;
      } catch (e) {
        return null;
      }
    } catch (error) {
      try { devError('View save error:', error); } catch {}
      throw error;
    }
  }, [tenantId, datasourceId]);

  const loadView = useCallback(async (viewName: string, create = false): Promise<any> => {
    // Validate view name before making API call
    assertValidViewName(viewName);

    try {
      const params = new URLSearchParams();
      if (tenantId) params.append('tenant_id', tenantId);
      if (datasourceId) params.append('tenant_instance_id', datasourceId);
      if (create) params.append('create', 'true');

      const response = await fetch(`/api/views/${encodeURIComponent(viewName)}?${params}`);

      if (!response.ok) {
        if (response.status === 404 && !create) {
          // Try again with create=true for skeleton
          return loadView(viewName, true);
        }
        throw new Error(`Load failed: ${response.statusText}`);
      }

      const data = await response.json();
      
      // If response contains a 'view' property, return the view content
      // Otherwise return the data as-is for backward compatibility
      return data.view || data;
    } catch (error) {
      try { devError('View load error:', error); } catch {}
      throw error;
    }
  }, [tenantId, datasourceId]);

  return {
    validateView,
    saveView,
    loadView
  };
};

export default useViewValidation;
