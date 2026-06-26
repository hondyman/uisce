/**
 * useValidationRulesAPI Hook
 * Handles all backend API interactions for validation rules
 * Features: optimistic updates, error recovery, retry logic
 */

import { useCallback, useState, useRef } from 'react';
import InvestmentValidationEngine, { ValidationRule, RuleSeverity, RuleFrequency } from '../services/validationEngine';
import { ValidationRuleFormData, buildCreateRulePayload, buildUpdateRulePayload } from '../lib/ruleUtils';
import { devLog, devWarn } from '../utils/devLogger';

export interface UseValidationRulesAPIOptions {
  tenantId: string | undefined;
  datasourceId: string | undefined;
  // onSuccess may receive a single rule or the full list when loading
  onSuccess?: (action: 'load' | 'create' | 'update' | 'delete', rule?: ValidationRule | ValidationRule[]) => void;
  onError?: (action: 'load' | 'create' | 'update' | 'delete', error: Error) => void;
}

export interface ValidationRulesAPIState {
  rules: ValidationRule[];
  loading: boolean;
  saving: boolean;
  error: Error | null;
}

export interface RuleOperation {
  timestamp: number;
  action: 'create' | 'update' | 'delete';
  ruleId?: string;
  // Keep data flexible for retries (payload or partial form data)
  data: any;
  status: 'pending' | 'completed' | 'failed';
}

export const useValidationRulesAPI = (options: UseValidationRulesAPIOptions) => {
  const { tenantId, datasourceId, onSuccess, onError } = options;

  const [state, setState] = useState<ValidationRulesAPIState>({
    rules: [],
    loading: false,
    saving: false,
    error: null,
  });

  // Track recent operations for optimistic updates
  const operationsRef = useRef<Map<string, RuleOperation>>(new Map());
  const engineRef = useRef<InvestmentValidationEngine | null>(null);
  const retryAttemptsRef = useRef<Map<string, number>>(new Map());

  // Initialize engine
  const getEngine = useCallback(() => {
    if (!tenantId || !datasourceId) {
      devWarn('[useValidationRulesAPI] Tenant or datasource missing');
      return null;
    }

    if (!engineRef.current) {
      engineRef.current = new InvestmentValidationEngine(tenantId, datasourceId);
    }
    return engineRef.current;
  }, [tenantId, datasourceId]);

  /**
   * Load all rules
   */
  const loadRules = useCallback(
    async (_force = false) => {
      const engine = getEngine();
      if (!engine) return [];

      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        devLog('[useValidationRulesAPI] Loading rules', { tenantId, datasourceId });
        const rules = await engine.getRules();

        setState((prev) => ({
          ...prev,
          rules: rules || [],
          loading: false,
        }));

        // Note: callbacks are called but not in deps to prevent infinite loops
        onSuccess?.('load', rules);
        return rules;
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        setState((prev) => ({ ...prev, loading: false, error: err }));
        onError?.('load', err);
        return [];
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [getEngine, tenantId, datasourceId] // Intentionally exclude onSuccess/onError to prevent loops
  );

  /**
   * Create new rule with optimistic update
   */
  const createRule = useCallback(
    async (formData: ValidationRuleFormData): Promise<ValidationRule> => {
      const engine = getEngine();
      if (!engine) throw new Error('Engine not initialized');

      const operationId = `create-${Date.now()}`;
      const payload = buildCreateRulePayload(formData, tenantId, datasourceId);

      // Optimistic update: construct a minimally valid ValidationRule to satisfy typing
      const tempRule: ValidationRule = {
        id: `temp-${Date.now()}`,
        name: formData.name,
        description: formData.description || '',
        ruleType: formData.ruleType,
        scope: formData.accountTypes || [],
        severity: (formData.severity as unknown as RuleSeverity) || RuleSeverity.INFO,
        isActive: formData.isActive !== false,
        effectiveFrom: new Date(),
        effectiveTo: undefined,
        frequency: RuleFrequency.CONTINUOUS,
        evaluationOrder: formData.evaluationOrder ?? 100,
        overrideConditions: undefined,
        requiredAuthority: formData.requiredAuthority,
        parameters: formData.parameters || {},
        createdAt: new Date(),
        updatedAt: new Date(),
        tenantId: tenantId || '',
        datasourceId: datasourceId || '',
      };

      setState((prev) => ({
        ...prev,
        rules: [...prev.rules, tempRule],
      }));

      // Track operation
      operationsRef.current.set(operationId, {
        timestamp: Date.now(),
        action: 'create',
        data: payload,
        status: 'pending',
      });

      setState((prev) => ({ ...prev, saving: true, error: null }));

      try {
        devLog('[useValidationRulesAPI] Creating rule', { name: formData.name });
        const newRule = await engine.createRule(payload as unknown as Omit<ValidationRule, 'createdAt' | 'updatedAt'>);

        // Replace temp rule with real rule
        setState((prev) => ({
          ...prev,
          rules: prev.rules.map((r) => (r.id === tempRule.id ? newRule : r)),
          saving: false,
        }));

        operationsRef.current.get(operationId)!.status = 'completed';
        operationsRef.current.get(operationId)!.ruleId = newRule.id;

        onSuccess?.('create', newRule);
        return newRule;
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));

        // Rollback optimistic update
        setState((prev) => ({
          ...prev,
          rules: prev.rules.filter((r) => r.id !== tempRule.id),
          saving: false,
          error: err,
        }));

        operationsRef.current.get(operationId)!.status = 'failed';
        onError?.('create', err);
        throw err;
      }
    },
    [getEngine, tenantId, datasourceId, onSuccess, onError]
  );

  /**
   * Update existing rule with optimistic update
   */
  const updateRule = useCallback(
    async (ruleId: string, formData: ValidationRuleFormData): Promise<ValidationRule> => {
      const engine = getEngine();
      if (!engine) throw new Error('Engine not initialized');

      const operationId = `update-${ruleId}-${Date.now()}`;
      const payload = buildUpdateRulePayload(formData, tenantId, datasourceId);

      // Find original rule for rollback
      const originalRule = state.rules.find((r) => r.id === ruleId);

      // Optimistic update - apply a typed partial update to avoid spreading mismatched severity/type shapes
      setState((prev) => ({
        ...prev,
        rules: prev.rules.map((r) =>
          r.id === ruleId
            ? ({
              ...r, ...{
                name: formData.name,
                description: formData.description,
                ruleType: formData.ruleType,
                scope: formData.accountTypes,
                severity: formData.severity as unknown as RuleSeverity,
                isActive: formData.isActive,
                evaluationOrder: formData.evaluationOrder,
                parameters: formData.parameters,
                updatedAt: new Date(),
              }
            } as ValidationRule)
            : r
        ),
      }));

      operationsRef.current.set(operationId, {
        timestamp: Date.now(),
        action: 'update',
        ruleId,
        data: payload,
        status: 'pending',
      });

      setState((prev) => ({ ...prev, saving: true, error: null }));

      try {
        devLog('[useValidationRulesAPI] Updating rule', { ruleId, name: formData.name });
        const updatedRule = await engine.updateRule(ruleId, payload as unknown as Partial<ValidationRule>);

        setState((prev) => ({
          ...prev,
          rules: prev.rules.map((r) => (r.id === ruleId ? updatedRule : r)),
          saving: false,
        }));

        operationsRef.current.get(operationId)!.status = 'completed';
        onSuccess?.('update', updatedRule);
        return updatedRule;
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));

        // Rollback to original
        setState((prev) => ({
          ...prev,
          rules: originalRule
            ? prev.rules.map((r) => (r.id === ruleId ? originalRule : r))
            : prev.rules.filter((r) => r.id !== ruleId),
          saving: false,
          error: err,
        }));

        operationsRef.current.get(operationId)!.status = 'failed';
        onError?.('update', err);
        throw err;
      }
    },
    [getEngine, tenantId, datasourceId, state.rules, onSuccess, onError]
  );

  /**
   * Delete rule with optimistic update
   */
  const deleteRule = useCallback(
    async (ruleId: string) => {
      const engine = getEngine();
      if (!engine) throw new Error('Engine not initialized');

      const operationId = `delete-${ruleId}-${Date.now()}`;
      const originalRule = state.rules.find((r) => r.id === ruleId);

      // Optimistic update: remove immediately
      setState((prev) => ({
        ...prev,
        rules: prev.rules.filter((r) => r.id !== ruleId),
      }));

      operationsRef.current.set(operationId, {
        timestamp: Date.now(),
        action: 'delete',
        ruleId,
        data: { ruleId },
        status: 'pending',
      });

      setState((prev) => ({ ...prev, saving: true, error: null }));

      try {
        devLog('[useValidationRulesAPI] Deleting rule', { ruleId });
        await engine.deleteRule(ruleId);

        setState((prev) => ({ ...prev, saving: false }));

        operationsRef.current.get(operationId)!.status = 'completed';
        onSuccess?.('delete');
        return true;
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));

        // Rollback: add rule back
        if (originalRule) {
          setState((prev) => ({
            ...prev,
            rules: [...prev.rules, originalRule],
            saving: false,
            error: err,
          }));
        } else {
          setState((prev) => ({ ...prev, saving: false, error: err }));
        }

        operationsRef.current.get(operationId)!.status = 'failed';
        onError?.('delete', err);
        throw err;
      }
    },
    [getEngine, state.rules, onSuccess, onError]
  );

  /**
   * Retry failed operation
   */
  const retryOperation = useCallback(
    async (operationId: string) => {
      const operation = operationsRef.current.get(operationId);
      if (!operation || operation.status !== 'failed') {
        devWarn('[useValidationRulesAPI] Cannot retry operation', { operationId });
        return false;
      }

      const retryCount = (retryAttemptsRef.current.get(operationId) || 0) + 1;
      if (retryCount > 3) {
        devWarn('[useValidationRulesAPI] Max retries exceeded', { operationId, retryCount });
        return false;
      }

      retryAttemptsRef.current.set(operationId, retryCount);

      try {
        if (operation.action === 'create') {
          await createRule(operation.data as ValidationRuleFormData);
        } else if (operation.action === 'update' && operation.ruleId) {
          await updateRule(operation.ruleId, operation.data as ValidationRuleFormData);
        } else if (operation.action === 'delete' && operation.ruleId) {
          await deleteRule(operation.ruleId);
        }
        return true;
      } catch (error) {
        devWarn('[useValidationRulesAPI] Retry failed', { operationId, attempt: retryCount });
        return false;
      }
    },
    [createRule, updateRule, deleteRule]
  );

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setState((prev) => ({ ...prev, error: null }));
  }, []);

  /**
   * Get pending operations count
   */
  const getPendingOperationsCount = useCallback(() => {
    let count = 0;
    operationsRef.current.forEach((op) => {
      if (op.status === 'pending') count++;
    });
    return count;
  }, []);

  /**
   * Simulate validation rule with a business object instance
   */
  const simulateWithInstance = useCallback(
    async (ruleId: string, instanceId: string) => {
      if (!tenantId || !datasourceId) {
        throw new Error('Tenant ID and Datasource ID are required');
      }

      const url = `/api/validation-rules/${ruleId}/simulate-with-instance?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`;

      try {
        devLog('[useValidationRulesAPI] Simulating rule with instance', { ruleId, instanceId });

        const response = await fetch(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ instance_id: instanceId }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
          throw new Error(errorData.error || `HTTP error ${response.status}`);
        }

        const result = await response.json();
        devLog('[useValidationRulesAPI] Simulation successful', result);
        return result;
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        devWarn('[useValidationRulesAPI] Simulation failed', err);
        throw err;
      }
    },
    [tenantId, datasourceId]
  );

  return {
    // State
    ...state,
    rules: state.rules,
    loading: state.loading,
    saving: state.saving,
    error: state.error,

    // Methods
    loadRules,
    createRule,
    updateRule,
    deleteRule,
    retryOperation,
    clearError,
    getPendingOperationsCount,
    simulateWithInstance,
  };
};
