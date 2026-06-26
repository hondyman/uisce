// note: useApolloClient removed because this module uses fetch/react-query helpers
import { useTenant } from '../../contexts/TenantContext';
import resolveApiUrl from '../../utils/resolveApiUrl';
import { useMutation as useReactMutation, useQuery as useReactQuery } from '@tanstack/react-query';

// Types for advanced conditional logic
export interface Condition {
  field: string;
  operator: '==' | '!=' | '>' | '<' | '>=' | '<=' | 'in' | 'contains' | 'startsWith' | 'endsWith';
  value: any;
}

export interface ConditionBranch {
  operator: 'AND' | 'OR' | 'NOT';
  conditions: Condition[];
  trueBranch?: string[]; // Step IDs to execute if true
  falseBranch?: string[]; // Step IDs to execute if false
}

// Types for approval chains
export interface ApprovalChain {
  type: 'role' | 'org_hierarchy' | 'custom' | 'multi_role';
  levels?: number; // For org_hierarchy: how many levels up
  roles?: string[]; // For multi_role: list of roles
  approvalMode: 'all' | 'any' | 'majority'; // How many approvals needed
  escalationPath?: string[]; // Fallback roles if timeout
}

// Types for notification templates
export interface NotificationConfig {
  templateId: string;
  channels: ('email' | 'in_app' | 'sms' | 'slack')[];
  recipients: {
    type: 'role' | 'user' | 'dynamic';
    value: string; // role name, user ID, or expression
  }[];
  mergeFields?: Record<string, string>; // Personalization tokens
}

export interface BPStep {
  id: string;
  stepOrder: number;
  stepType: 'data_entry' | 'validate' | 'approve' | 'notify' | 'integrate' | 'condition';
  stepName: string;
  durationHours: number;
  assigneeRole?: string;
  validationRules?: string[];
  notificationTemplate?: string;
  
  // Advanced conditional logic
  conditionLogic?: ConditionBranch;
  
  // Parallel execution support
  executionMode: 'sequential' | 'parallel';
  parallelGroup?: string; // Steps with same group execute in parallel
  waitForAll?: boolean; // true = all in group must complete, false = any can complete
  
  // Approval chain configuration
  approvalChain?: ApprovalChain;
  
  // Step dependencies
  dependsOn?: string[]; // Step IDs that must complete before this step
  skipCondition?: ConditionBranch; // Skip this step if condition is true
  
  // Enhanced notifications
  notificationConfig?: NotificationConfig;
  
  description?: string;
  escalationThresholdHours?: number;
}

export interface BusinessProcess {
  id: string;
  processName: string;
  entity: string;
  description: string;
  steps: BPStep[];
  isActive: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt?: string;
  version: number;
  tags?: string[];
}

export interface BPBuilderAPIResponse {
  success: boolean;
  data?: BusinessProcess;
  error?: string;
  timestamp: string;
}

// API_BASE intentionally omitted; resolveApiUrl is used with explicit paths.

// Helper to build headers with tenant scope
function getTenantHeaders(tenant?: { id: string }, datasource?: { id: string }) {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (tenant?.id) {
    headers['X-Tenant-ID'] = tenant.id;
  }
  if (datasource?.id) {
    headers['X-Tenant-Datasource-ID'] = datasource.id;
  }
  return headers;
}

// Build query params for tenant scoping
function getTenantParams(tenant?: { id: string }, datasource?: { id: string }) {
  const params = new URLSearchParams();
  if (tenant?.id) {
    params.append('tenant_id', tenant.id);
  }
  if (datasource?.id) {
    params.append('tenant_instance_id', datasource.id);
  }
  return params.toString();
}

/**
 * Fetch all business processes for the current tenant
 */
export function useFetchBusinessProcesses() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async () => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes?${params}`);

      const res = await fetch(url, {
        method: 'GET',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch business processes');
      }

      const data = (await res.json()) as BPBuilderAPIResponse;
      if (!data.success) {
        throw new Error(data.error || 'API returned success=false');
      }
      return data.data;
    },
  });
}

/**
 * Fetch a single business process by ID
 */
export function useFetchBusinessProcess(processId: string | null) {
  const { tenant, datasource } = useTenant();

  return useReactQuery({
    queryKey: ['business-process', processId, tenant?.id, datasource?.id],
    queryFn: async () => {
      if (!processId) return null;

      const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes/${processId}?${params}`);

      const res = await fetch(url, {
        method: 'GET',
        credentials: 'include',
        headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch business process');
      }

      const data = (await res.json()) as BPBuilderAPIResponse;
      if (!data.success) {
        throw new Error(data.error || 'API returned success=false');
      }
      return data.data;
    },
    enabled: !!processId && !!tenant?.id,
  });
}

/**
 * Create a new business process
 */
export function useCreateBusinessProcess() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async (process: Omit<BusinessProcess, 'id' | 'createdAt' | 'updatedAt' | 'version'>) => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes?${params}`);

      const res = await fetch(url, {
        method: 'POST',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
        body: JSON.stringify(process),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to create business process');
      }

      const data = (await res.json()) as BPBuilderAPIResponse;
      if (!data.success) {
        throw new Error(data.error || 'API returned success=false');
      }
      return data.data;
    },
  });
}

/**
 * Update an existing business process
 */
export function useUpdateBusinessProcess() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async (process: BusinessProcess) => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes/${process.id}?${params}`);

      const res = await fetch(url, {
        method: 'PUT',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
        body: JSON.stringify(process),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to update business process');
      }

      const data = (await res.json()) as BPBuilderAPIResponse;
      if (!data.success) {
        throw new Error(data.error || 'API returned success=false');
      }
      return data.data;
    },
  });
}

/**
 * Delete a business process
 */
export function useDeleteBusinessProcess() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async (processId: string) => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes/${processId}?${params}`);

      const res = await fetch(url, {
        method: 'DELETE',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete business process');
      }

      return true;
    },
  });
}

/**
 * Publish/activate a business process
 */
export function usePublishBusinessProcess() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async (processId: string) => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes/${processId}/publish?${params}`);

      const res = await fetch(url, {
        method: 'POST',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to publish business process');
      }

      const data = (await res.json()) as BPBuilderAPIResponse;
      if (!data.success) {
        throw new Error(data.error || 'API returned success=false');
      }
      return data.data;
    },
  });
}

/**
 * Simulate a business process
 */
export function useSimulateBusinessProcess() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async (payload: { processId: string; testData?: Record<string, any> }) => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes/${payload.processId}/simulate?${params}`);

      const res = await fetch(url, {
        method: 'POST',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
        body: JSON.stringify({ testData: payload.testData || {} }),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to simulate business process');
      }

      return res.json();
    },
  });
}

/**
 * Clone/duplicate a business process
 */
export function useDuplicateBusinessProcess() {
  const { tenant, datasource } = useTenant();

  return useReactMutation({
    mutationFn: async (processId: string) => {
  const params = getTenantParams(tenant ?? undefined, datasource ?? undefined);
      const url = resolveApiUrl(`/api/business-processes/${processId}/duplicate?${params}`);

      const res = await fetch(url, {
        method: 'POST',
        credentials: 'include',
  headers: getTenantHeaders(tenant ?? undefined, datasource ?? undefined),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to duplicate business process');
      }

      const data = (await res.json()) as BPBuilderAPIResponse;
      if (!data.success) {
        throw new Error(data.error || 'API returned success=false');
      }
      return data.data;
    },
  });
}
