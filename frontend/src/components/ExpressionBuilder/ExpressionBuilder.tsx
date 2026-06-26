import type React from 'react';
import { useState, useEffect, useRef } from 'react';
import { devError } from '../../utils/devLogger';
import { Card } from '@mui/material';
import { useNotification } from '../../hooks/useNotification';
import ActionButton from '../ui/ActionButton';
import { useMutation, gql } from '@apollo/client';
import AdvancedConditionBuilder, { 
  ConditionGroup,
  ConditionNode as _ConditionNode,
  evaluateCondition
} from './AdvancedConditionBuilder';
import styles from './ExpressionBuilder.module.css';
// Apollo GraphQL mutation example (uncomment and adapt if using apollo client)
// import { useMutation, gql } from '@apollo/client';
// const INSERT_RULE = gql`
// mutation InsertRule($object: rules_insert_input!) {
//   insert_rules_one(object: $object) { id }
// }
// `;

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

// Upsert mutation for catalog_validation_rules. Adjust the on_conflict.constraint name if your Hasura
// schema generates a different constraint name for the UNIQUE(tenant_id, rule_name) constraint.
const INSERT_DRAFT_RULE = gql`
  mutation InsertDraftValidationRule($object: catalog_validation_rules_insert_input!) {
    insert_catalog_validation_rules_one(object: $object) { id }
  }
`;

const UPDATE_RULE_BY_PK = gql`
  mutation UpdateValidationRuleByPk($id: uuid!, $changes: catalog_validation_rules_set_input!) {
    update_catalog_validation_rules_by_pk(pk_columns: { id: $id }, _set: $changes) { id }
  }
`;

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
  
  // Initialize with empty root condition group
  const [conditionTree, setConditionTree] = useState<ConditionGroup>({
    id: 'root',
    type: 'group',
    operator: 'AND',
    conditions: []
  });

  const [insertDraftRule] = useMutation(INSERT_DRAFT_RULE);
  const [updateRuleByPk] = useMutation(UPDATE_RULE_BY_PK);

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

  // Persist helper (non-debounced) - called by the debounced scheduler when enabled
  const persistNow = async (conditionJson: ConditionGroup | null) => {
    // If autosave is not enabled, do nothing
    if (!autosave) return;

    const tenant = localStorage.getItem('selected_tenant') ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id : null;
    const datasource = localStorage.getItem('selected_datasource') ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id : null;

    if (!tenant || !datasource) {
      notification.warning('Select a tenant & datasource to persist visual rule');
      return;
    }

    const object: Record<string, unknown> = {
      tenant_id: tenant,
      rule_name: ruleName || 'Visual Rule',
      rule_type: 'business_logic',
      condition_json: conditionJson || {},
      target_entity: targetEntity || undefined,
    };

    const maxRetries = 3;
    let attempt = 0;
    
    const doPersist = async (): Promise<void> => {
      attempt += 1;
      try {
        const effectiveId = ruleId || draftId;
        if (effectiveId) {
          // If we have an id (existing rule or draft), update-by-pk
          const changes: any = { condition_json: conditionJson };
          if (targetEntity) changes.target_entity = targetEntity;
          await updateRuleByPk({ 
            variables: { id: effectiveId, changes }, 
            context: { 
              headers: { 
                'X-Tenant-ID': tenant, 
                'X-Tenant-Datasource-ID': datasource 
              } 
            } 
          });
          notification.success('Rule autosaved');
        } else {
          // No id yet: create a draft row
          const draftObject: Record<string, unknown> = {
            ...object,
            rule_name: ruleName || `Draft Rule ${Date.now()}`,
            is_active: false,
          };
          
          const res = await insertDraftRule({ 
            variables: { object: draftObject }, 
            context: { 
              headers: { 
                'X-Tenant-ID': tenant, 
                'X-Tenant-Datasource-ID': datasource 
              } 
            } 
          });
          
          const newId = res?.data?.insert_catalog_validation_rules_one?.id;
          if (newId) {
            setDraftId(newId);
            const draftName = typeof draftObject.rule_name === 'string' ? draftObject.rule_name : undefined;
            onDraftCreated && onDraftCreated(newId, draftName);
            notification.success('Draft created');
          } else {
            throw new Error('No id returned from draft insert');
          }
        }
      } catch (err: any) {
    devError('Autosave attempt failed', attempt, err);
        if (attempt < maxRetries) {
          const backoffMs = 200 * Math.pow(2, attempt - 1);
          await new Promise(resolve => setTimeout(resolve, backoffMs));
          return doPersist();
        }
        notification.error('Failed to persist rule (autosave). Please check your tenant selection and network.');
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
      onSave && onSave(conditionTree);
      notification.success('Rule saved successfully!');
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
