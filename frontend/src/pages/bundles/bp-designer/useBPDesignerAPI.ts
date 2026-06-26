/**
 * useBPDesignerAPI.ts
 * React hooks for Business Process Designer API calls
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ProcessStepType,
  ValidationOperator,
  WorkflowEvent,
  BusinessObject,
  Process,
  ValidationRule,
} from './types';
import { getSelectedRegion } from '../../../lib/region';

/**
 * Fetch function that includes tenant scope
 */
async function fetchWithTenant(url: string, options?: RequestInit) {
  const tenantData = localStorage.getItem('selected_tenant');
  const datasourceData = localStorage.getItem('selected_datasource');

  const tenant = tenantData ? JSON.parse(tenantData) : null;
  const datasource = datasourceData ? JSON.parse(datasourceData) : null;

  if (!tenant?.id || !datasource?.id) {
    throw new Error('Tenant and datasource must be selected');
  }

  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenant.id,
    'X-Tenant-Datasource-ID': datasource.id,
    'X-Tenant-Region': getSelectedRegion(),
    ...options?.headers,
  };

  const separator = url.includes('?') ? '&' : '?';
  const urlWithScope = `${url}${separator}tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`;

  const response = await fetch(urlWithScope, {
    ...options,
    headers,
  });

  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`);
  }

  return response.json();
}

// Query Hooks

export const useStepTypes = () => {
  return useQuery({
    queryKey: ['stepTypes'],
    queryFn: () => fetchWithTenant('/api/step-types'),
  });
};

export const useValidationOperators = () => {
  return useQuery({
    queryKey: ['validationOperators'],
    queryFn: () => fetchWithTenant('/api/validation-operators'),
  });
};

export const useWorkflowEvents = () => {
  return useQuery({
    queryKey: ['workflowEvents'],
    // Use the v1 triggers events endpoint to avoid collision with legacy /api/events
    queryFn: () => fetchWithTenant('/api/v1/triggers/events'),
  });
};

export const useBusinessObjects = () => {
  return useQuery({
    queryKey: ['businessObjects'],
    queryFn: () => fetchWithTenant('/api/business-objects/list'),
  });
};

export const useProcess = (processId: string | null) => {
  return useQuery({
    queryKey: ['process', processId],
    queryFn: () =>
      processId
        ? fetchWithTenant(`/api/processes/${processId}`)
        : Promise.resolve(null),
    enabled: !!processId,
  });
};

export const useValidationRules = (processId: string | null, nodeId: string | null) => {
  return useQuery({
    queryKey: ['validationRules', processId, nodeId],
    queryFn: () =>
      processId && nodeId
        ? fetchWithTenant(`/api/processes/${processId}/nodes/${nodeId}/rules`)
        : Promise.resolve([]),
    enabled: !!processId && !!nodeId,
  });
};

// Mutation Hooks

export const useCreateProcess = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (process: Omit<Process, 'id' | 'created_at' | 'updated_at'>) =>
      fetchWithTenant('/api/processes', {
        method: 'POST',
        body: JSON.stringify(process),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['processes'] });
    },
  });
};

export const useUpdateProcess = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, nodes, edges }: { id: string; nodes: any[]; edges: any[] }) =>
      fetchWithTenant(`/api/processes/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ nodes, edges }),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['process', id] });
    },
  });
};

export const useSaveValidationRules = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ processId, nodeId, rules }: { processId: string; nodeId: string; rules: ValidationRule[] }) =>
      fetchWithTenant(`/api/processes/${processId}/nodes/${nodeId}/rules`, {
        method: 'POST',
        body: JSON.stringify(rules),
      }),
    onSuccess: (_, { processId, nodeId }) => {
      queryClient.invalidateQueries({ queryKey: ['validationRules', processId, nodeId] });
    },
  });
};
