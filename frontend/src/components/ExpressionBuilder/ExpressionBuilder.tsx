import type React from 'react';
import { useState } from 'react';
import { devError } from '../../utils/devLogger';
import { Card } from '@mui/material';
import { useNotification } from '../../hooks/useNotification';
import ActionButton from '../ui/ActionButton';
import AdvancedConditionBuilder, {
  ConditionGroup,
  evaluateCondition
} from './AdvancedConditionBuilder';
import styles from './ExpressionBuilder.module.css';

// A pure condition-tree editor: it reports the tree through onChange/onSave
// and persists nothing itself. (It used to autosave into RuleFabric, a second
// rule engine that has been retired - rules persist through their owning
// screen, evaluated by internal/rules/vm.)
interface ExpressionBuilderProps {
  onSave?: (conditionJson: ConditionGroup) => void;
  onChange?: (conditionJson: ConditionGroup) => void;
  availableFields?: Array<{ name: string; type: string; label: string }>;
}

const ExpressionBuilder: React.FC<ExpressionBuilderProps> = ({
  onSave,
  onChange,
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
  };

  const handleSave = () => {
    try {
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
