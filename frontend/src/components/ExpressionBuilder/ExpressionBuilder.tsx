import type React from 'react';
import { useState, useEffect, useRef } from 'react';
import { devError } from '../../utils/devLogger';
import { Card } from '@mui/material';
import { useNotification } from '../../hooks/useNotification';
import ActionButton from '../ui/ActionButton';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../../lib/apiClient';
import AdvancedConditionBuilder, {
  ConditionGroup,
  ConditionNode as _ConditionNode,
  evaluateCondition
} from './AdvancedConditionBuilder';
import styles from './ExpressionBuilder.module.css';

interface ExpressionBuilderProps {
  onSave?: (conditionJson: ConditionGroup) => void;
  onChange?: (conditionJson: ConditionGroup) => void;
  autosave?: boolean; // default false
  debounceMs?: number; // default 1000
  ruleName?: string;
  targetEntity?: string;
  ruleId?: string;
  onDraftCreated?: (id: string, ruleName?: string) => void;
  availableFields?: Array<{ name: string; type: string; label: string }>;
}

const ExpressionBuilder: React.FC<ExpressionBuilderProps> = ({ 
  onSave, 
  onChange, 
  autosave = false, 
  debounceMs = 1000, 
  ruleName, 
  targetEntity, 
  ruleId, 
  onDraftCreated,
  availableFields: propAvailableFields
}) => {
  const notification = useNotification();
  const queryClient = useQueryClient();

  // Initialize with empty root condition group
  const [conditionTree, setConditionTree] = useState<ConditionGroup>({
    id: 'root',
    type: 'group',
    operator: 'AND',
    conditions: []
  });

  // Persists to RuleFabric (backend/internal/rulefabric), whose ConditionGroup/
  // Condition types are defined to match AdvancedConditionBuilder's schema
  // directly - see backend/internal/rulefabric/evaluator.go. Rules and their
  // condition logic are separate, versioned resources there: creating a rule
  // creates metadata only, and the condition tree is saved as rule_logic
  // version 1 via a follow-up call; every subsequent save creates a new
  // version rather than mutating one in place.
  const createRule = useMutation({
    mutationFn: async (object: Record<string, unknown>) => {
      const res = await apiFetch('/api/rule-fabric/rules', {
        method: 'POST',
        body: JSON.stringify(object),
      });
      return res.json();
    },
  });
  const createRuleVersion = useMutation({
    mutationFn: async ({ id, conditionJson, changeReason }: { id: string; conditionJson: ConditionGroup; changeReason: string }) => {
      const res = await apiFetch(`/api/rule-fabric/rules/${id}/versions`, {
        method: 'POST',
        body: JSON.stringify({ condition_json: conditionJson, change_reason: changeReason }),
      });
      return res.json();
    },
  });

  const saveTimer = useRef<number | null>(null);
  const lastPayload = useRef<ConditionGroup | null>(null);
  const [draftId, setDraftId] = useState<string | null>(null);

  // Available fields for the builder - use prop or default/hardcoded
  const defaultFields = [
    { name: 'age', type: 'number', label: 'Age' },
    { name: 'salary', type: 'number', label: 'Salary' },
    { name: 'email', type: 'string', label: 'Email' },
    { name: 'status', type: 'string', label: 'Status' },
    { name: 'is_vip', type: 'boolean', label: 'Is VIP' },
    { name: 'hire_date', type: 'date', label: 'Hire Date' },
    { name: 'first_name', type: 'string', label: 'First Name' },
    { name: 'last_name', type: 'string', label: 'Last Name' }
  ];

  const availableFields = propAvailableFields || defaultFields;

  // Handle condition tree changes
  const handleConditionChange = (newTree: ConditionGroup) => {
    setConditionTree(newTree);
    onChange && onChange(newTree);
    schedulePersist(newTree);
  };

  // Slugify a rule name into a rule_code (RuleFabric requires one at creation).
  const toRuleCode = (name: string): string =>
    name
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '') || `rule_${Date.now()}`;

  // Persist helper (non-debounced). Called by the debounced autosave
  // scheduler, and directly by the explicit Save button regardless of the
  // autosave flag.
  const persistNow = async (conditionJson: ConditionGroup | null, opts?: { force?: boolean }) => {
    if (!autosave && !opts?.force) return;

    const tenant = localStorage.getItem('selected_tenant') ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id : null;

    if (!tenant) {
      notification.warning('Select a tenant to persist this rule');
      return;
    }

    const maxRetries = 3;
    let attempt = 0;

    const doPersist = async (): Promise<void> => {
      attempt += 1;
      try {
        const effectiveId = ruleId || draftId;
        if (effectiveId) {
          // Rule already exists: RuleLogic is versioned, so every save
          // creates a new rule_logic version rather than mutating one in
          // place.
          await createRuleVersion.mutateAsync({
            id: effectiveId,
            conditionJson: conditionJson || { id: 'root', type: 'group', operator: 'AND', conditions: [] },
            changeReason: opts?.force ? 'manual save' : 'autosave',
          });
          notification.success(opts?.force ? 'Rule saved' : 'Rule autosaved');
        } else {
          // No rule yet: create it, then save the condition tree as its
          // first version.
          const name = ruleName || `Draft Rule ${Date.now()}`;
          const draftObject: Record<string, unknown> = {
            tenant_id: tenant,
            rule_code: toRuleCode(name),
            name,
            category: 'custom',
            primary_context: targetEntity ? 'data_record' : 'data_record',
            severity: 'warning',
            environment: 'production',
            scope_entity: targetEntity || undefined,
          };

          const created = await createRule.mutateAsync(draftObject);
          const newId = created?.id || created?.[0]?.id;
          if (!newId) {
            throw new Error('No id returned from rule creation');
          }

          await createRuleVersion.mutateAsync({
            id: newId,
            conditionJson: conditionJson || { id: 'root', type: 'group', operator: 'AND', conditions: [] },
            changeReason: 'initial version',
          });

          setDraftId(newId);
          onDraftCreated && onDraftCreated(newId, name);
          notification.success('Rule created');
        }
      } catch (err: any) {
        devError('Persist attempt failed', attempt, err);
        if (attempt < maxRetries) {
          const backoffMs = 200 * Math.pow(2, attempt - 1);
          await new Promise(resolve => setTimeout(resolve, backoffMs));
          return doPersist();
        }
        notification.error('Failed to persist rule. Please check your tenant selection and network.');
      }
    };

    await doPersist();
  };

  // Schedule a debounced save
  const schedulePersist = (conditionJson: any) => {
    lastPayload.current = conditionJson;
    if (!autosave) return;
    if (saveTimer.current) {
      window.clearTimeout(saveTimer.current);
    }
    saveTimer.current = window.setTimeout(async () => {
      await persistNow(lastPayload.current);
      saveTimer.current = null;
    }, debounceMs);
  };

  // Flush pending save on unmount
  useEffect(() => {
    return () => {
      if (saveTimer.current) {
        window.clearTimeout(saveTimer.current);
        persistNow(lastPayload.current);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSave = async () => {
    try {
      await persistNow(conditionTree, { force: true });
      onSave && onSave(conditionTree);
    } catch (e) {
      devError('onSave callback threw', e);
      notification.error('Failed to save rule');
    }
  };

  // Test evaluation function
  const testEvaluation = () => {
    const testData = {
      age: 25,
      salary: 75000,
      email: 'user@example.com',
      status: 'Active',
      is_vip: true,
      hire_date: '2022-01-15',
      first_name: 'John',
      last_name: 'Doe'
    };

    const result = evaluateCondition(conditionTree, testData);
    notification.info(`Test evaluation result: ${result ? '✅ PASS' : '❌ FAIL'}`);
  };

  return (
    <div className={styles.builderWrapper}>
      <Card className={styles.panel}>
        <h4 style={{ margin: '0 0 8px 0' }}>🎨 Advanced Expression Builder</h4>
        <p style={{ margin: '0 0 16px 0' }}>Build complex validation logic with nested groups and AND/OR combinations</p>
        
        <AdvancedConditionBuilder
          value={conditionTree}
          onChange={handleConditionChange}
          availableFields={availableFields}
          entityName="Entity"
        />

        <div className={styles.builderActions}>
          <ActionButton variant="primary" onClick={handleSave}>
            💾 Save Rule
          </ActionButton>
          <ActionButton variant="secondary" onClick={testEvaluation}>
            🧪 Test Rule
          </ActionButton>
        </div>
      </Card>
    </div>
  );
};

export default ExpressionBuilder;
