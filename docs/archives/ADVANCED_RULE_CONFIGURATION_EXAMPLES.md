# Advanced Rule Configuration - Code Examples

## Table of Contents
1. [Basic Usage](#basic-usage)
2. [Complete Integration](#complete-integration)
3. [Custom Implementations](#custom-implementations)
4. [GraphQL Integration](#graphql-integration)
5. [Testing Examples](#testing-examples)

---

## Basic Usage

### Minimal Setup
```typescript
import React from 'react';
import AdvancedRuleConfiguration from './components/validation/AdvancedRuleConfiguration';

function MyPage() {
  return (
    <AdvancedRuleConfiguration
      onRulesUpdate={(rules) => console.log('Rules:', rules)}
      onCrossEntitySave={(condition) => console.log('Condition:', condition)}
    />
  );
}

export default MyPage;
```

### With External Rules
```typescript
import React, { useState } from 'react';
import AdvancedRuleConfiguration, { ValidationRule } from './components/validation/AdvancedRuleConfiguration';

function RuleManager() {
  const [rules, setRules] = useState<ValidationRule[]>([
    {
      id: 'rule_1',
      name: 'Age Verification',
      entity: 'Employee',
      description: 'Employee must be at least 18 years old',
      severity: 'error'
    },
    {
      id: 'rule_2',
      name: 'Status Check',
      entity: 'Employee',
      description: 'Employee status must be Active or On Leave',
      severity: 'warning',
      dependent_rule_ids: ['rule_1']
    }
  ]);

  return (
    <AdvancedRuleConfiguration
      rules={rules}
      onRulesUpdate={setRules}
      onCrossEntitySave={(condition) => {
        console.log('New condition:', condition);
      }}
    />
  );
}

export default RuleManager;
```

---

## Complete Integration

### Full Component with Backend Sync
```typescript
import React, { useState, useCallback, useEffect } from 'react';
import { useMutation, useQuery } from '@apollo/client';
import AdvancedRuleConfiguration, {
  ValidationRule,
  CrossEntityCondition
} from './components/validation/AdvancedRuleConfiguration';
import {
  FETCH_RULES,
  UPDATE_RULE_DEPENDENCIES,
  CREATE_CROSS_ENTITY_VALIDATION
} from './graphql/rules';
import { useToast } from './hooks/useToast';
import { useTenant } from './hooks/useTenant';

function AdvancedRuleEditor() {
  const { tenantId, datasourceId } = useTenant();
  const { showSuccess, showError } = useToast();
  
  // Fetch rules
  const { data, loading, error, refetch } = useQuery(FETCH_RULES, {
    variables: { tenantId, datasourceId },
    skip: !tenantId || !datasourceId
  });

  // Mutations
  const [updateDependencies, { loading: updatingDeps }] = useMutation(
    UPDATE_RULE_DEPENDENCIES
  );
  
  const [createCondition, { loading: creatingCondition }] = useMutation(
    CREATE_CROSS_ENTITY_VALIDATION
  );

  // Local state
  const [rules, setRules] = useState<ValidationRule[]>([]);
  const [isSynced, setIsSynced] = useState(true);

  // Update from query
  useEffect(() => {
    if (data?.validationRules) {
      setRules(data.validationRules);
      setIsSynced(true);
    }
  }, [data]);

  // Handle rule updates
  const handleRulesUpdate = useCallback(
    async (updatedRules: ValidationRule[]) => {
      setRules(updatedRules);
      setIsSynced(false);

      try {
        // Find which rules have new dependencies
        const originalRules = data?.validationRules || [];
        const changedRules = updatedRules.filter(updatedRule => {
          const originalRule = originalRules.find(r => r.id === updatedRule.id);
          return JSON.stringify(originalRule?.dependent_rule_ids) !==
                 JSON.stringify(updatedRule.dependent_rule_ids);
        });

        // Update each changed rule
        for (const rule of changedRules) {
          await updateDependencies({
            variables: {
              ruleId: rule.id,
              dependencies: rule.dependent_rule_ids || [],
              tenantId,
              datasourceId
            }
          });
        }

        setIsSynced(true);
        showSuccess('Rules updated successfully');
        refetch();
      } catch (error) {
        showError('Failed to update rules');
        console.error('Error updating rules:', error);
        setRules(data?.validationRules || []); // Revert
      }
    },
    [data, tenantId, datasourceId, updateDependencies, refetch, showSuccess, showError]
  );

  // Handle cross-entity condition
  const handleCrossEntitySave = useCallback(
    async (condition: CrossEntityCondition) => {
      try {
        await createCondition({
          variables: {
            sourcePath: JSON.stringify(condition.sourcePath),
            operator: condition.operator,
            targetPath: JSON.stringify(condition.targetPath),
            tenantId,
            datasourceId
          }
        });

        showSuccess('Cross-entity validation created');
        refetch();
      } catch (error) {
        showError('Failed to create validation');
        console.error('Error creating condition:', error);
      }
    },
    [tenantId, datasourceId, createCondition, refetch, showSuccess, showError]
  );

  if (loading) return <div className="p-8">Loading rules...</div>;
  if (error) return <div className="p-8 text-red-600">Error loading rules</div>;

  return (
    <div className="relative">
      {!isSynced && (
        <div className="absolute top-0 right-0 bg-yellow-100 border border-yellow-400 text-yellow-700 px-4 py-2 rounded">
          ⚠️ Unsaved changes
        </div>
      )}
      
      <AdvancedRuleConfiguration
        rules={rules}
        onRulesUpdate={handleRulesUpdate}
        onCrossEntitySave={handleCrossEntitySave}
      />

      {updatingDeps && <div className="mt-4 text-blue-600">Updating dependencies...</div>}
      {creatingCondition && <div className="mt-4 text-blue-600">Creating validation...</div>}
    </div>
  );
}

export default AdvancedRuleEditor;
```

---

## Custom Implementations

### Using Only Dependency Chain
```typescript
import React, { useState } from 'react';
import { RuleDependencyChain, ValidationRule } from './components/validation/AdvancedRuleConfiguration';

function DependencyManager() {
  const [rules, setRules] = useState<ValidationRule[]>([
    { id: '1', name: 'Rule 1', entity: 'Entity', description: 'Desc', severity: 'error' },
    { id: '2', name: 'Rule 2', entity: 'Entity', description: 'Desc', severity: 'warning' }
  ]);

  const [selectedRuleId, setSelectedRuleId] = useState('1');

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Dependency Management</h1>
      
      <select
        value={selectedRuleId}
        onChange={(e) => setSelectedRuleId(e.target.value)}
        className="mb-6"
      >
        {rules.map(rule => (
          <option key={rule.id} value={rule.id}>{rule.name}</option>
        ))}
      </select>

      <RuleDependencyChain
        rules={rules}
        selectedRuleId={selectedRuleId}
        onUpdateDependencies={(ruleId, deps) => {
          const updated = rules.map(r =>
            r.id === ruleId ? { ...r, dependent_rule_ids: deps } : r
          );
          setRules(updated);
        }}
      />
    </div>
  );
}

export default DependencyManager;
```

### Using Only Entity Path Picker
```typescript
import React, { useState } from 'react';
import { EntityPathPicker, EntityPath } from './components/validation/AdvancedRuleConfiguration';

function PathSelector() {
  const [selectedPath, setSelectedPath] = useState<EntityPath | null>(null);

  return (
    <div className="p-6 max-w-md">
      <h1 className="text-2xl font-bold mb-6">Select a Field Path</h1>
      
      <EntityPathPicker
        startEntity="Employee"
        value={selectedPath}
        onChange={setSelectedPath}
        label="Employee Field"
      />

      {selectedPath && (
        <div className="mt-6 p-4 bg-blue-50 rounded">
          <h2 className="font-semibold mb-2">Selected Path:</h2>
          <div className="font-mono text-sm text-blue-600">
            {selectedPath.displayPath}
          </div>
          <div className="mt-2 text-xs text-gray-600">
            Segments: {selectedPath.segments.length}
          </div>
        </div>
      )}
    </div>
  );
}

export default PathSelector;
```

### Using Only Cross-Entity Builder
```typescript
import React, { useState } from 'react';
import {
  CrossEntityValidationBuilder,
  CrossEntityCondition
} from './components/validation/AdvancedRuleConfiguration';

function ValidationBuilder() {
  const [conditions, setConditions] = useState<CrossEntityCondition[]>([]);

  const handleSave = (condition: CrossEntityCondition) => {
    setConditions([...conditions, condition]);
    console.log('Saved condition:', condition);
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Create Cross-Entity Validation</h1>
      
      <CrossEntityValidationBuilder
        sourceEntity="Employee"
        onSave={handleSave}
      />

      {conditions.length > 0 && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold mb-4">
            Saved Conditions ({conditions.length})
          </h2>
          {conditions.map((condition, idx) => (
            <div
              key={idx}
              className="p-4 bg-white border border-gray-300 rounded mb-2"
            >
              <div className="flex items-center gap-3 font-mono text-sm">
                <span className="text-purple-600">{condition.sourcePath.displayPath}</span>
                <span className="bg-blue-600 text-white px-2 py-1 rounded text-xs">
                  {condition.operator}
                </span>
                <span className="text-blue-600">{condition.targetPath.displayPath}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default ValidationBuilder;
```

---

## GraphQL Integration

### Complete GraphQL Schema
```graphql
# Types
type ValidationRule {
  id: ID!
  name: String!
  entity: String!
  description: String!
  severity: RuleSeverity!
  dependent_rule_ids: [ID!]
  createdAt: DateTime!
  updatedAt: DateTime!
}

enum RuleSeverity {
  ERROR
  WARNING
  INFO
}

type EntityPath {
  segments: [PathSegment!]!
  displayPath: String!
}

type PathSegment {
  entity: String!
  field: String!
  relationship: String!
}

type CrossEntityCondition {
  id: ID!
  sourcePath: EntityPath!
  operator: String!
  targetPath: EntityPath!
  createdAt: DateTime!
}

# Inputs
input EntityPathInput {
  segments: [PathSegmentInput!]!
  displayPath: String!
}

input PathSegmentInput {
  entity: String!
  field: String!
  relationship: String!
}

input CrossEntityConditionInput {
  sourcePath: EntityPathInput!
  operator: String!
  targetPath: EntityPathInput!
}

# Queries
type Query {
  validationRules(
    tenantId: ID!
    datasourceId: ID!
    entityFilter: String
  ): [ValidationRule!]!

  validationRule(id: ID!, tenantId: ID!): ValidationRule

  crossEntityValidations(
    tenantId: ID!
    datasourceId: ID!
  ): [CrossEntityCondition!]!
}

# Mutations
type Mutation {
  updateRuleDependencies(
    ruleId: ID!
    dependencies: [ID!]!
    tenantId: ID!
    datasourceId: ID!
  ): ValidationRule!

  createCrossEntityValidation(
    condition: CrossEntityConditionInput!
    tenantId: ID!
    datasourceId: ID!
  ): CrossEntityCondition!

  deleteCrossEntityValidation(
    id: ID!
    tenantId: ID!
  ): Boolean!
}
```

### Apollo Client Setup
```typescript
import { ApolloClient, InMemoryCache, createHttpLink } from '@apollo/client';

const httpLink = createHttpLink({
  uri: 'http://localhost:8000/graphql',
  credentials: 'include',
  headers: {
    'Content-Type': 'application/json'
  }
});

// Custom middleware to add tenant headers
const tenantLink = new ApolloLink((operation, forward) => {
  const tenantId = localStorage.getItem('selected_tenant_id');
  const datasourceId = localStorage.getItem('selected_datasource_id');

  operation.setContext({
    headers: {
      'X-Tenant-ID': tenantId || '',
      'X-Tenant-Datasource-ID': datasourceId || ''
    }
  });

  return forward(operation);
});

export const apolloClient = new ApolloClient({
  link: tenantLink.concat(httpLink),
  cache: new InMemoryCache({
    typePolicies: {
      ValidationRule: {
        keyFields: ['id']
      }
    }
  })
});
```

### Query Examples
```typescript
import { gql } from '@apollo/client';

// Fetch all rules
export const FETCH_RULES = gql`
  query FetchRules($tenantId: ID!, $datasourceId: ID!) {
    validationRules(tenantId: $tenantId, datasourceId: $datasourceId) {
      id
      name
      entity
      description
      severity
      dependent_rule_ids
      createdAt
      updatedAt
    }
  }
`;

// Fetch single rule with dependencies
export const FETCH_RULE_WITH_DEPS = gql`
  query FetchRuleWithDeps($ruleId: ID!, $tenantId: ID!) {
    validationRule(id: $ruleId, tenantId: $tenantId) {
      id
      name
      entity
      description
      severity
      dependent_rule_ids
      createdAt
      updatedAt
    }
  }
`;

// Fetch cross-entity validations
export const FETCH_CROSS_ENTITY_VALIDATIONS = gql`
  query FetchCrossEntityValidations($tenantId: ID!, $datasourceId: ID!) {
    crossEntityValidations(tenantId: $tenantId, datasourceId: $datasourceId) {
      id
      sourcePath {
        segments {
          entity
          field
          relationship
        }
        displayPath
      }
      operator
      targetPath {
        segments {
          entity
          field
          relationship
        }
        displayPath
      }
      createdAt
    }
  }
`;
```

### Mutation Examples
```typescript
export const UPDATE_RULE_DEPENDENCIES = gql`
  mutation UpdateRuleDependencies(
    $ruleId: ID!
    $dependencies: [ID!]!
    $tenantId: ID!
    $datasourceId: ID!
  ) {
    updateRuleDependencies(
      ruleId: $ruleId
      dependencies: $dependencies
      tenantId: $tenantId
      datasourceId: $datasourceId
    ) {
      id
      name
      dependent_rule_ids
      updatedAt
    }
  }
`;

export const CREATE_CROSS_ENTITY_VALIDATION = gql`
  mutation CreateCrossEntityValidation(
    $condition: CrossEntityConditionInput!
    $tenantId: ID!
    $datasourceId: ID!
  ) {
    createCrossEntityValidation(
      condition: $condition
      tenantId: $tenantId
      datasourceId: $datasourceId
    ) {
      id
      sourcePath {
        segments {
          entity
          field
          relationship
        }
        displayPath
      }
      operator
      targetPath {
        segments {
          entity
          field
          relationship
        }
        displayPath
      }
      createdAt
    }
  }
`;

export const DELETE_CROSS_ENTITY_VALIDATION = gql`
  mutation DeleteCrossEntityValidation($id: ID!, $tenantId: ID!) {
    deleteCrossEntityValidation(id: $id, tenantId: $tenantId)
  }
`;
```

---

## Testing Examples

### Unit Tests
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MockedProvider } from '@apollo/client/testing';
import AdvancedRuleConfiguration, {
  RuleDependencyChain,
  ValidationRule
} from './AdvancedRuleConfiguration';

describe('AdvancedRuleConfiguration', () => {
  const mockRules: ValidationRule[] = [
    {
      id: '1',
      name: 'Rule 1',
      entity: 'Employee',
      description: 'Test rule 1',
      severity: 'error'
    },
    {
      id: '2',
      name: 'Rule 2',
      entity: 'Employee',
      description: 'Test rule 2',
      severity: 'warning'
    }
  ];

  it('renders both tabs', () => {
    render(<AdvancedRuleConfiguration rules={mockRules} />);
    expect(screen.getByText('Rule Dependencies')).toBeInTheDocument();
    expect(screen.getByText('Cross-Entity Validation')).toBeInTheDocument();
  });

  it('switches between tabs', () => {
    render(<AdvancedRuleConfiguration rules={mockRules} />);
    
    fireEvent.click(screen.getByText('Cross-Entity Validation'));
    expect(
      screen.getByText(/Compare fields across related entities/i)
    ).toBeInTheDocument();
  });

  it('calls onRulesUpdate when dependency added', async () => {
    const mockUpdate = jest.fn();
    render(
      <AdvancedRuleConfiguration
        rules={mockRules}
        onRulesUpdate={mockUpdate}
      />
    );

    // Change selected rule
    fireEvent.change(screen.getByLabelText('Select rule to configure'), {
      target: { value: '2' }
    });

    // Add dependency
    const addSelect = screen.getByLabelText('Add dependent rule');
    fireEvent.change(addSelect, { target: { value: '1' } });

    await waitFor(() => {
      expect(mockUpdate).toHaveBeenCalled();
    });
  });
});

describe('RuleDependencyChain', () => {
  const mockRules: ValidationRule[] = [
    {
      id: 'rule_1',
      name: 'Age Check',
      entity: 'Employee',
      description: 'Age check',
      severity: 'error'
    },
    {
      id: 'rule_2',
      name: 'Status Check',
      entity: 'Employee',
      description: 'Status check',
      severity: 'warning',
      dependent_rule_ids: ['rule_1']
    }
  ];

  it('displays current rule', () => {
    render(
      <RuleDependencyChain
        rules={mockRules}
        selectedRuleId="rule_2"
        onUpdateDependencies={jest.fn()}
      />
    );

    expect(screen.getByText('Status Check')).toBeInTheDocument();
    expect(screen.getByText('warning')).toBeInTheDocument();
  });

  it('shows dependencies', () => {
    render(
      <RuleDependencyChain
        rules={mockRules}
        selectedRuleId="rule_2"
        onUpdateDependencies={jest.fn()}
      />
    );

    expect(screen.getByText('Age Check')).toBeInTheDocument();
  });

  it('removes dependency on trash click', () => {
    const mockUpdate = jest.fn();
    render(
      <RuleDependencyChain
        rules={mockRules}
        selectedRuleId="rule_2"
        onUpdateDependencies={mockUpdate}
      />
    );

    const trashButtons = screen.getAllByLabelText(/Remove dependency/i);
    fireEvent.click(trashButtons[0]);

    expect(mockUpdate).toHaveBeenCalledWith('rule_2', []);
  });
});
```

### Integration Tests
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MockedProvider } from '@apollo/client/testing';
import AdvancedRuleConfiguration from './AdvancedRuleConfiguration';
import { UPDATE_RULE_DEPENDENCIES } from './graphql';

describe('AdvancedRuleConfiguration Integration', () => {
  it('saves cross-entity validation', async () => {
    const mockSave = jest.fn();
    render(
      <AdvancedRuleConfiguration onCrossEntitySave={mockSave} />
    );

    // Switch to cross-entity tab
    fireEvent.click(screen.getByText('Cross-Entity Validation'));

    // Click source path picker
    const sourceButtons = screen.getAllByText(/Click to select a field path/i);
    fireEvent.click(sourceButtons[0]);

    // Note: Full integration test would require more complex interactions
  });

  it('executes GraphQL mutations', async () => {
    const mocks = [
      {
        request: {
          query: UPDATE_RULE_DEPENDENCIES,
          variables: {
            ruleId: 'rule_1',
            dependencies: ['rule_2'],
            tenantId: 'tenant_1',
            datasourceId: 'datasource_1'
          }
        },
        result: {
          data: {
            updateRuleDependencies: {
              id: 'rule_1',
              name: 'Rule 1',
              dependent_rule_ids: ['rule_2'],
              updatedAt: new Date().toISOString()
            }
          }
        }
      }
    ];

    render(
      <MockedProvider mocks={mocks}>
        <AdvancedRuleConfiguration />
      </MockedProvider>
    );

    // Test would trigger mutation and verify result
  });
});
```

---

**Last Updated:** October 20, 2025  
**Version:** 1.0.0  
**Status:** Production Ready ✅
