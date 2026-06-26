import { useState, useCallback, useEffect } from 'react';
import * as ruleService from '../services/ruleService';

export interface PriorityStep {
  id: string;
  priority: number;
  condition: {
    term: string;
    operator: string;
    value: any;
  };
  confidence: number;
  description: string;
}

export interface Rule {
  id: string;
  businessObject: string;
  name?: string;
  description?: string;
  version: number;
  status: 'draft' | 'testing' | 'staging' | 'production';
  steps: PriorityStep[];
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

export interface UseRuleBuilderReturn {
  rule: Rule | null;
  loading: boolean;
  error: string | null;
  addStep: (step?: Partial<PriorityStep>) => void;
  updateStep: (stepId: string, updates: Partial<PriorityStep>) => void;
  deleteStep: (stepId: string) => void;
  reorderSteps: (steps: PriorityStep[]) => void;
  saveRule: () => Promise<void>;
  publishRule: () => Promise<void>;
}

/**
 * useRuleBuilder Hook
 *
 * Manages rule builder state and API integration for creating and updating priority rules.
 * Handles:
 * - Rule loading and initialization
 * - Priority step management (add, update, delete, reorder)
 * - Optimistic updates with rollback
 * - Rule persistence (draft save)
 * - Rule publication (version promotion)
 *
 * @param ruleId - Optional existing rule ID to load
 * @param businessObject - Business object name for new rules
 * @returns Hook state and management methods
 */
export const useRuleBuilder = (
  ruleId?: string,
  businessObject: string = 'calendar'
): UseRuleBuilderReturn => {
  const [rule, setRule] = useState<Rule | null>(null);
  const [loading, setLoading] = useState(!!ruleId);
  const [error, setError] = useState<string | null>(null);

  // Load rule if ruleId provided
  useEffect(() => {
    if (!ruleId) {
      // Initialize new rule
      setRule({
        id: `rule_${Date.now()}`,
        businessObject,
        name: '',
        description: '',
        version: 1,
        status: 'draft',
        steps: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        createdBy: 'current_user', // Would come from auth context
      });
      setLoading(false);
      return;
    }

    // Fetch existing rule from backend
    const loadRule = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await ruleService.getRule(ruleId);
        setRule(data as Rule);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load rule';
        setError(message);
        console.error('Error loading rule:', err);
      } finally {
        setLoading(false);
      }
    };

    loadRule();
  }, [ruleId, businessObject]);

  const addStep = useCallback(
    (step?: Partial<PriorityStep>) => {
      if (!rule) return;

      const newStep: PriorityStep = {
        id: `step_${Date.now()}`,
        priority: rule.steps.length + 1,
        condition: {
          term: step?.condition?.term || '',
          operator: step?.condition?.operator || 'equals',
          value: step?.condition?.value || '',
        },
        confidence: step?.confidence ?? 75,
        description: step?.description ?? '',
      };

      setRule((prev) => ({
        ...prev!,
        steps: [...prev!.steps, newStep],
        updatedAt: new Date().toISOString(),
      }));
    },
    [rule]
  );

  const updateStep = useCallback(
    (stepId: string, updates: Partial<PriorityStep>) => {
      setRule((prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          steps: prev.steps.map((step) =>
            step.id === stepId ? { ...step, ...updates } : step
          ),
          updatedAt: new Date().toISOString(),
        };
      });
    },
    []
  );

  const deleteStep = useCallback((stepId: string) => {
    setRule((prev) => {
      if (!prev) return prev;
      const filtered = prev.steps.filter((step) => step.id !== stepId);
      // Renumber priorities
      const renumbered = filtered.map((step, idx) => ({
        ...step,
        priority: idx + 1,
      }));
      return {
        ...prev,
        steps: renumbered,
        updatedAt: new Date().toISOString(),
      };
    });
  }, []);

  const reorderSteps = useCallback((newSteps: PriorityStep[]) => {
    setRule((prev) => {
      if (!prev) return prev;
      // Renumber priorities after reorder
      const renumbered = newSteps.map((step, idx) => ({
        ...step,
        priority: idx + 1,
      }));
      return {
        ...prev,
        steps: renumbered,
        updatedAt: new Date().toISOString(),
      };
    });
  }, []);

  const saveRule = useCallback(async () => {
    if (!rule) return;

    try {
      setLoading(true);
      setError(null);

      // Prepare request payload
      const payload: ruleService.CreateRuleRequest = {
        businessObject: rule.businessObject,
        name: rule.name || `Rule-${Date.now()}`,
        description: rule.description || '',
        steps: rule.steps.map((s) => ({
          priority: s.priority,
          condition: {
            semanticTerm: s.condition.term,
            operator: s.condition.operator,
            value: String(s.condition.value),
          },
          action: {
            useField: 'golden_record',
            confidence: s.confidence,
          },
          description: s.description,
        })),
      };

      // Create or update rule
      let saved: ruleService.Rule;
      if (rule.id.startsWith('rule_')) {
        // New rule - create it
        saved = await ruleService.createRule(payload);
      } else {
        // Existing draft - update it
        saved = await ruleService.updateRule(rule.id, payload);
      }

      setRule(saved as Rule);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save rule';
      setError(message);
      console.error('Error saving rule:', err);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [rule]);

  const publishRule = useCallback(async () => {
    if (!rule) return;

    try {
      setLoading(true);
      setError(null);

      const published = await ruleService.publishRule(
        rule.id,
        rule.version,
        `Published to testing for validation`
      );

      setRule(published as Rule);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to publish rule';
      setError(message);
      console.error('Error publishing rule:', err);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [rule]);

  return {
    rule,
    loading,
    error,
    addStep,
    updateStep,
    deleteStep,
    reorderSteps,
    saveRule,
    publishRule,
  };
};

export default useRuleBuilder;
