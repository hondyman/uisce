# Advanced Condition Builder - Usage Examples

This document provides practical code examples for using the Advanced Condition Builder in various scenarios.

## Example 1: Basic Rule - Employee Age Verification

**Scenario**: Create a rule that checks if an employee is at least 18 years old.

### Implementation

```tsx
import { useState } from 'react';
import AdvancedConditionBuilder, { 
  ConditionGroup,
  evaluateCondition 
} from '../ExpressionBuilder/AdvancedConditionBuilder';

export function BasicAgeVerificationRule() {
  const [conditions, setConditions] = useState<ConditionGroup>({
    id: 'root',
    type: 'group',
    operator: 'AND',
    conditions: [
      {
        id: 'age_check',
        field: 'age',
        operator: 'greater_equal',
        value: '18',
        fieldType: 'number'
      }
    ]
  });

  const availableFields = [
    { name: 'age', type: 'number', label: 'Age' },
    { name: 'first_name', type: 'string', label: 'First Name' },
    { name: 'last_name', type: 'string', label: 'Last Name' }
  ];

  const handleSave = (conditionTree: ConditionGroup) => {
    console.log('Saving rule:', JSON.stringify(conditionTree, null, 2));
  };

  return (
    <AdvancedConditionBuilder
      value={conditions}
      onChange={setConditions}
      availableFields={availableFields}
      entityName="Employee"
    />
  );
}
```

### Generated JSON Output

```json
{
  "id": "root",
  "type": "group",
  "operator": "AND",
  "conditions": [
    {
      "id": "age_check",
      "field": "age",
      "operator": "greater_equal",
      "value": "18",
      "fieldType": "number"
    }
  ]
}
```

## Example 2: Complex Rule - Employee Eligibility

**Scenario**: Create a rule that checks if an employee is eligible for a benefit program.
- Must be at least 18 years old
- Must be in "Active" status OR have VIP privilege
- Must have a valid email

### Building the Condition Tree

```tsx
const eligibilityRule: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    // Condition 1: Age >= 18
    {
      id: 'age_check',
      field: 'age',
      operator: 'greater_equal',
      value: '18',
      fieldType: 'number'
    },
    
    // Condition 2: Nested group - Status OR VIP
    {
      id: 'status_vip_group',
      type: 'group',
      operator: 'OR',
      conditions: [
        {
          id: 'status_check',
          field: 'status',
          operator: 'equals',
          value: 'Active',
          fieldType: 'string'
        },
        {
          id: 'vip_check',
          field: 'is_vip',
          operator: 'is_true',
          value: 'true',
          fieldType: 'boolean'
        }
      ]
    },
    
    // Condition 3: Email is not empty
    {
      id: 'email_check',
      field: 'email',
      operator: 'is_not_empty',
      value: '',
      fieldType: 'string'
    }
  ]
};
```

### Evaluation

```tsx
const employeeData = {
  age: 28,
  status: 'Active',
  is_vip: false,
  email: 'john.doe@company.com'
};

const isEligible = evaluateCondition(eligibilityRule, employeeData);
// Result: true
// Reasoning: (age >= 18) AND ((status = Active) OR (is_vip = true)) AND (email is not empty)
```

## Example 3: With Autosave Integration

**Scenario**: Create a validation rule with automatic persistence.

### Component Implementation

```tsx
import ExpressionBuilder from '../ExpressionBuilder/ExpressionBuilder';

export function ValidationRuleWithAutosave() {
  const [ruleId, setRuleId] = useState<string | null>(null);
  const [ruleName, setRuleName] = useState('Income Validation Rule');
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle');

  const handleDraftCreated = (draftId: string, draftName?: string) => {
    // Update parent state when draft is created
    setRuleId(draftId);
    if (draftName) {
      setRuleName(draftName);
    }
    console.log('Draft created:', draftId);
  };

  return (
    <ExpressionBuilder
      // Enable autosave for existing rules (when ruleId is set)
      autosave={!!ruleId}
      debounceMs={1000}
      
      // Rule metadata
      ruleName={ruleName}
      targetEntity="Employee"
      ruleId={ruleId}
      
      // Callbacks
      onDraftCreated={handleDraftCreated}
      onChange={(conditions) => {
        setSaveStatus('saving');
        // Autosave will trigger in 1000ms if debounce enabled
      }}
    />
  );
}
```

### Autosave Flow

```
1. User adds first condition
   → schedulePersist() sets timer
   → Waits 1000ms

2. No more changes for 1000ms
   → persistNow() executes
   → Since no ruleId: insert_catalog_validation_rules_one (draft)
   → onDraftCreated called
   → setRuleId(newId)

3. User makes next change
   → schedulePersist() sets new timer
   → Since ruleId now exists: update_catalog_validation_rules_by_pk
   → Toast: "Rule autosaved"

4. User navigates away
   → useEffect cleanup called
   → Flush any pending saves
   → Final save executes (best-effort)
```

## Example 4: Date Range Validation

**Scenario**: Create a rule for hire date validation.

```tsx
const hireDateRule: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    {
      id: 'hire_after',
      field: 'hire_date',
      operator: 'after',
      value: '2020-01-01',
      fieldType: 'date'
    },
    {
      id: 'hire_before',
      field: 'hire_date',
      operator: 'before',
      value: '2025-01-01',
      fieldType: 'date'
    }
  ]
};

// Evaluation
const testData = { hire_date: '2023-06-15' };
const isValid = evaluateCondition(hireDateRule, testData);
// Result: true (date is between 2020-01-01 and 2025-01-01)
```

## Example 5: Complex Nested Structure - Department Policy

**Scenario**: Complex rule with multiple nesting levels.

```
Rule Logic:
IF (Department = "Engineering" OR Department = "Product")
AND (
  (Level = "Senior" AND Salary >= 150000)
  OR
  (Level = "Lead" AND Salary >= 200000)
  OR
  (Level = "Manager" AND Salary >= 180000)
)
AND (Performance = "Meets Expectations" OR Performance = "Exceeds")
```

### Implementation

```tsx
const departmentPolicyRule: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    // Condition 1: Department filter
    {
      id: 'dept_group',
      type: 'group',
      operator: 'OR',
      conditions: [
        {
          id: 'dept_eng',
          field: 'department',
          operator: 'equals',
          value: 'Engineering',
          fieldType: 'string'
        },
        {
          id: 'dept_prod',
          field: 'department',
          operator: 'equals',
          value: 'Product',
          fieldType: 'string'
        }
      ]
    },

    // Condition 2: Complex salary/level matrix
    {
      id: 'salary_group',
      type: 'group',
      operator: 'OR',
      conditions: [
        // Senior with $150k+
        {
          id: 'senior_check',
          type: 'group',
          operator: 'AND',
          conditions: [
            {
              id: 'level_senior',
              field: 'level',
              operator: 'equals',
              value: 'Senior',
              fieldType: 'string'
            },
            {
              id: 'salary_senior',
              field: 'salary',
              operator: 'greater_equal',
              value: '150000',
              fieldType: 'number'
            }
          ]
        },
        
        // Lead with $200k+
        {
          id: 'lead_check',
          type: 'group',
          operator: 'AND',
          conditions: [
            {
              id: 'level_lead',
              field: 'level',
              operator: 'equals',
              value: 'Lead',
              fieldType: 'string'
            },
            {
              id: 'salary_lead',
              field: 'salary',
              operator: 'greater_equal',
              value: '200000',
              fieldType: 'number'
            }
          ]
        },
        
        // Manager with $180k+
        {
          id: 'manager_check',
          type: 'group',
          operator: 'AND',
          conditions: [
            {
              id: 'level_mgr',
              field: 'level',
              operator: 'equals',
              value: 'Manager',
              fieldType: 'string'
            },
            {
              id: 'salary_mgr',
              field: 'salary',
              operator: 'greater_equal',
              value: '180000',
              fieldType: 'number'
            }
          ]
        }
      ]
    },

    // Condition 3: Performance check
    {
      id: 'perf_group',
      type: 'group',
      operator: 'OR',
      conditions: [
        {
          id: 'perf_meets',
          field: 'performance',
          operator: 'equals',
          value: 'Meets Expectations',
          fieldType: 'string'
        },
        {
          id: 'perf_exceeds',
          field: 'performance',
          operator: 'equals',
          value: 'Exceeds',
          fieldType: 'string'
        }
      ]
    }
  ]
};
```

### Test Data

```tsx
const employee1 = {
  department: 'Engineering',
  level: 'Senior',
  salary: 155000,
  performance: 'Meets Expectations'
};

const employee2 = {
  department: 'Product',
  level: 'Lead',
  salary: 210000,
  performance: 'Exceeds'
};

const employee3 = {
  department: 'Finance', // Wrong dept
  level: 'Senior',
  salary: 155000,
  performance: 'Meets Expectations'
};

console.log(evaluateCondition(departmentPolicyRule, employee1)); // true
console.log(evaluateCondition(departmentPolicyRule, employee2)); // true
console.log(evaluateCondition(departmentPolicyRule, employee3)); // false
```

## Example 6: String Pattern Validation

**Scenario**: Validate email and name patterns.

```tsx
const emailValidationRule: ConditionGroup = {
  id: 'root',
  type: 'group',
  operator: 'AND',
  conditions: [
    {
      id: 'email_contains',
      field: 'email',
      operator: 'contains',
      value: '@company.com',
      fieldType: 'string'
    },
    {
      id: 'email_not_empty',
      field: 'email',
      operator: 'is_not_empty',
      value: '',
      fieldType: 'string'
    },
    {
      id: 'name_not_empty',
      field: 'first_name',
      operator: 'is_not_empty',
      value: '',
      fieldType: 'string'
    }
  ]
};

const testData = {
  email: 'john.doe@company.com',
  first_name: 'John'
};

console.log(evaluateCondition(emailValidationRule, testData)); // true
```

## Example 7: Testing and Debugging

**Scenario**: Test your conditions with sample data.

```tsx
export function RuleDebugger({ conditionTree }: { conditionTree: ConditionGroup }) {
  const [testData, setTestData] = useState({
    age: 25,
    status: 'Active',
    is_vip: false,
    salary: 75000,
    email: 'test@example.com'
  });

  const result = evaluateCondition(conditionTree, testData);

  return (
    <div style={{ padding: '20px', border: '1px solid #ccc' }}>
      <h3>Rule Debugger</h3>
      
      <div>
        <h4>Condition Tree</h4>
        <pre>{JSON.stringify(conditionTree, null, 2)}</pre>
      </div>

      <div>
        <h4>Test Data</h4>
        {Object.entries(testData).map(([key, value]) => (
          <input
            key={key}
            type="text"
            value={value}
            placeholder={key}
            onChange={(e) => setTestData(prev => ({
              ...prev,
              [key]: e.target.value
            }))}
          />
        ))}
      </div>

      <div>
        <h4>Evaluation Result</h4>
        <p style={{
          fontSize: '20px',
          fontWeight: 'bold',
          color: result ? 'green' : 'red'
        }}>
          {result ? '✅ PASS' : '❌ FAIL'}
        </p>
      </div>
    </div>
  );
}
```

## Example 8: Programmatic Condition Creation

**Scenario**: Create conditions programmatically based on configuration.

```tsx
function createFieldCondition(
  field: string,
  operator: string,
  value: string,
  type: string
) {
  return {
    id: `cond_${Date.now()}_${Math.random()}`,
    field,
    operator,
    value,
    fieldType: type
  };
}

function createGroup(operator: 'AND' | 'OR', conditions: any[]) {
  return {
    id: `group_${Date.now()}_${Math.random()}`,
    type: 'group' as const,
    operator,
    conditions
  };
}

// Usage
const config = {
  rules: [
    { field: 'age', operator: 'greater_equal', value: '18', type: 'number' },
    { field: 'status', operator: 'equals', value: 'Active', type: 'string' }
  ]
};

const conditions = createGroup(
  'AND',
  config.rules.map(r => createFieldCondition(r.field, r.operator, r.value, r.type))
);

console.log(JSON.stringify(conditions, null, 2));
```

## Example 9: Form Integration

**Scenario**: Integrate the builder into a larger form.

```tsx
import { Form, Button, Input } from 'antd';
import ExpressionBuilder from '../ExpressionBuilder/ExpressionBuilder';
import { ConditionGroup } from '../ExpressionBuilder/AdvancedConditionBuilder';

export function ValidationRuleForm() {
  const [form] = Form.useForm();
  const [conditions, setConditions] = useState<ConditionGroup>({
    id: 'root',
    type: 'group',
    operator: 'AND',
    conditions: []
  });

  const onFinish = (formValues: any) => {
    const ruleData = {
      ...formValues,
      conditionTree: conditions
    };
    console.log('Saving rule:', ruleData);
    // Submit to backend
  };

  return (
    <Form form={form} onFinish={onFinish} layout="vertical">
      <Form.Item
        name="ruleName"
        label="Rule Name"
        rules={[{ required: true }]}
      >
        <Input placeholder="e.g., Senior Employee Bonus" />
      </Form.Item>

      <Form.Item
        name="description"
        label="Description"
      >
        <Input.TextArea placeholder="Describe what this rule does..." />
      </Form.Item>

      <Form.Item label="Conditions">
        <ExpressionBuilder
          value={conditions}
          onChange={setConditions}
          ruleName={form.getFieldValue('ruleName')}
        />
      </Form.Item>

      <Button type="primary" htmlType="submit">
        Save Rule
      </Button>
    </Form>
  );
}
```

## Example 10: Error Handling in Evaluation

**Scenario**: Handle missing fields or type mismatches.

```tsx
function safeEvaluateCondition(
  conditionTree: ConditionGroup,
  data: Record<string, any>
): { success: boolean; result: boolean; error?: string } {
  try {
    // Check if all referenced fields exist in data
    const missingFields = findMissingFields(conditionTree, data);
    if (missingFields.length > 0) {
      return {
        success: false,
        result: false,
        error: `Missing fields: ${missingFields.join(', ')}`
      };
    }

    const result = evaluateCondition(conditionTree, data);
    return { success: true, result };
  } catch (error) {
    return {
      success: false,
      result: false,
      error: `Evaluation failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    };
  }
}

function findMissingFields(
  node: ConditionNode,
  data: Record<string, any>,
  missing: string[] = []
): string[] {
  if (isCondition(node)) {
    if (!(node.field in data)) {
      missing.push(node.field);
    }
  } else if (isGroup(node)) {
    node.conditions.forEach(child => findMissingFields(child, data, missing));
  }
  return missing;
}
```

These examples demonstrate the flexibility and power of the Advanced Condition Builder for creating sophisticated validation rules.
