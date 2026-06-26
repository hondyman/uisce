import React, { createContext, useContext, useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

/**
 * ABAC Provider - Attribute-Based Access Control Context
 * 
 * Provides policy evaluation and management across the application.
 * All policy decisions are tenant-scoped and multi-tenant safe.
 * 
 * Features:
 * - Policy evaluation (subject, action, resource, environment)
 * - Dynamic policy loading from backend
 * - Caching with React Query
 * - Delegation support (temporary role assignments)
 * - Audit trail integration
 */

export interface ABACPolicy {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  subject: {
    roles?: string[];
    users?: string[];
    departments?: string[];
  };
  action: {
    allowed?: string[];
    denied?: string[];
  };
  resource: {
    types?: string[];
    excluded?: string[];
  };
  environment?: {
    locations?: string[];
    time_windows?: any[];
  };
  effect: 'allow' | 'deny';
  priority: number;
  enabled: boolean;
}

export interface ABACEvaluationRequest {
  action: string;
  resource: string;
  subject?: {
    user_id?: string;
    role?: string;
    department?: string;
  };
  environment?: {
    ip_address?: string;
    time?: string;
    location?: string;
  };
}

export interface ABACEvaluationResult {
  allowed: boolean;
  policy_id?: string;
  reason?: string;
  matched_policies: ABACPolicy[];
  timestamp: string;
}

interface ABACContextType {
  policies: ABACPolicy[];
  loading: boolean;
  error?: Error;
  evaluate: (request: ABACEvaluationRequest) => Promise<ABACEvaluationResult>;
  createPolicy: (policy: Omit<ABACPolicy, 'id'>) => Promise<ABACPolicy>;
  updatePolicy: (id: string, policy: Partial<ABACPolicy>) => Promise<ABACPolicy>;
  deletePolicy: (id: string) => Promise<void>;
  canExecute: (action: string, resource: string) => Promise<boolean>;
  tenantId: string;
}

const ABACContext = createContext<ABACContextType | null>(null);

interface ABACProviderProps {
  children: React.ReactNode;
  tenantId: string;
  baseUrl?: string;
}

/**
 * ABACProvider - Context provider for ABAC functionality
 * 
 * Wraps your application to provide attribute-based access control.
 * All requests are automatically scoped to the tenant.
 * 
 * Usage:
 * ```tsx
 * <ABACProvider tenantId={tenantId}>
 *   <YourApp />
 * </ABACProvider>
 * ```
 */
export const ABACProvider: React.FC<ABACProviderProps> = ({
  children,
  tenantId,
  baseUrl = '/api',
}) => {
  const queryClient = useQueryClient();
  const [error, setError] = useState<Error | undefined>();

  // Fetch all policies for the tenant
  const { data: policies = [], isLoading } = useQuery<ABACPolicy[]>({
    queryKey: ['abac-policies', tenantId],
    queryFn: async () => {
      const response = await fetch(`${baseUrl}/abac/policies`, {
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json',
        },
      });
      if (!response.ok) throw new Error('Failed to load policies');
      return response.json();
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  // Evaluate access decision
  const evaluate = useCallback(
    async (request: ABACEvaluationRequest): Promise<ABACEvaluationResult> => {
      try {
        const response = await fetch(`${baseUrl}/abac/evaluate`, {
          method: 'POST',
          headers: {
            'X-Tenant-ID': tenantId,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(request),
        });

        if (!response.ok) {
          throw new Error('ABAC evaluation failed');
        }

        const result = await response.json();
        return result;
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Unknown error');
        setError(error);
        throw error;
      }
    },
    [tenantId, baseUrl]
  );

  // Create policy
  const createMutation = useMutation({
    mutationFn: (policy: Omit<ABACPolicy, 'id'>) =>
      fetch(`${baseUrl}/abac/policies`, {
        method: 'POST',
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(policy),
      })
        .then((r) => {
          if (!r.ok) throw new Error('Failed to create policy');
          return r.json();
        }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['abac-policies', tenantId] });
    },
  });

  // Update policy
  const updateMutation = useMutation({
    mutationFn: ({ id, policy }: { id: string; policy: Partial<ABACPolicy> }) =>
      fetch(`${baseUrl}/abac/policies/${id}`, {
        method: 'PUT',
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(policy),
      })
        .then((r) => {
          if (!r.ok) throw new Error('Failed to update policy');
          return r.json();
        }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['abac-policies', tenantId] });
    },
  });

  // Delete policy
  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      fetch(`${baseUrl}/abac/policies/${id}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': tenantId,
        },
      })
        .then((r) => {
          if (!r.ok) throw new Error('Failed to delete policy');
        }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['abac-policies', tenantId] });
    },
  });

  // Convenience method to check if action is allowed
  const canExecute = useCallback(
    async (action: string, resource: string): Promise<boolean> => {
      const result = await evaluate({
        action,
        resource,
      });
      return result.allowed;
    },
    [evaluate]
  );

  const value: ABACContextType = {
    policies,
    loading: isLoading,
    error,
    evaluate,
    createPolicy: createMutation.mutateAsync,
    updatePolicy: (id, policy) => updateMutation.mutateAsync({ id, policy }),
    deletePolicy: deleteMutation.mutateAsync,
    canExecute,
    tenantId,
  };

  return (
    <ABACContext.Provider value={value}>
      {children}
    </ABACContext.Provider>
  );
};

/**
 * useABAC Hook - Access ABAC functionality
 * 
 * Usage:
 * ```tsx
 * const { evaluate, canExecute, policies } = useABAC();
 * 
 * const allowed = await canExecute('edit_trigger', 'orders');
 * ```
 */
export const useABAC = () => {
  const context = useContext(ABACContext);
  if (!context) {
    throw new Error('useABAC must be used within ABACProvider');
  }
  return context;
};

export default ABACProvider;
