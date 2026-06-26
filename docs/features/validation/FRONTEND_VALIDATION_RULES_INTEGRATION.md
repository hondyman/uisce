# Frontend Integration: Validation Rules with Backend API

## Overview

This guide explains how to integrate the frontend Validation Rules tab with the backend API endpoints. The integration layer provides HTTP communication, data management, and error handling between the React UI and the Go backend.

## Architecture

### Request Flow
```
React Component (ValidationRulesContainer)
    ↓
API Service Layer (ValidationRulesService)
    ↓
HTTP Client (with tenant scope)
    ↓
Backend Endpoints
    ↓
Database (PostgreSQL)
```

## API Service Layer

### Create `services/validationRulesService.ts`

```typescript
import { APIClient } from './apiClient';
import { TenantContext } from '../context/TenantContext';

export interface ValidationRule {
  id: string;
  name: string;
  description: string;
  condition: Record<string, any>;
  entityIds: string[];
  status: 'active' | 'inactive';
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
}

export interface ValidationRuleRequest {
  name: string;
  description: string;
  condition: Record<string, any>;
  entityIds: string[];
  status: 'active' | 'inactive';
}

export interface ValidationRuleExecutionResult {
  ruleId: string;
  isValid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
  executedAt: string;
}

export interface ValidationError {
  field: string;
  message: string;
  code: string;
}

export interface ValidationWarning {
  field: string;
  message: string;
}

export class ValidationRulesService {
  private apiClient: APIClient;

  constructor(apiClient: APIClient) {
    this.apiClient = apiClient;
  }

  /**
   * List all validation rules for the current tenant
   */
  async listRules(
    page: number = 1,
    limit: number = 50,
    filters?: {
      entityId?: string;
      status?: string;
      search?: string;
    }
  ): Promise<{ data: ValidationRule[]; pagination: { page: number; limit: number; total: number } }> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
      ...TenantContext.getCurrentScope(),
    });

    if (filters?.entityId) {
      params.append('entity_id', filters.entityId);
    }
    if (filters?.status) {
      params.append('status', filters.status);
    }
    if (filters?.search) {
      params.append('search', filters.search);
    }

    const response = await this.apiClient.get<{
      data: ValidationRule[];
      pagination: { page: number; limit: number; total: number };
    }>(`/validation-rules?${params}`);

    return response;
  }

  /**
   * Get a specific validation rule
   */
  async getRule(ruleId: string): Promise<ValidationRule> {
    const params = TenantContext.getCurrentScope();
    const response = await this.apiClient.get<ValidationRule>(
      `/validation-rules/${ruleId}?${new URLSearchParams(params)}`
    );
    return response;
  }

  /**
   * Create a new validation rule
   */
  async createRule(data: ValidationRuleRequest): Promise<ValidationRule> {
    const params = TenantContext.getCurrentScope();
    const response = await this.apiClient.post<ValidationRule>(
      `/validation-rules?${new URLSearchParams(params)}`,
      data
    );
    return response;
  }

  /**
   * Update an existing validation rule
   */
  async updateRule(ruleId: string, data: Partial<ValidationRuleRequest>): Promise<ValidationRule> {
    const params = TenantContext.getCurrentScope();
    const response = await this.apiClient.patch<ValidationRule>(
      `/validation-rules/${ruleId}?${new URLSearchParams(params)}`,
      data
    );
    return response;
  }

  /**
   * Delete a validation rule
   */
  async deleteRule(ruleId: string): Promise<void> {
    const params = TenantContext.getCurrentScope();
    await this.apiClient.delete(`/validation-rules/${ruleId}?${new URLSearchParams(params)}`);
  }

  /**
   * Execute a single validation rule
   */
  async executeRule(ruleId: string, data: Record<string, any>): Promise<ValidationRuleExecutionResult> {
    const params = TenantContext.getCurrentScope();
    const response = await this.apiClient.post<ValidationRuleExecutionResult>(
      `/validation-rules/${ruleId}/execute?${new URLSearchParams(params)}`,
      { data }
    );
    return response;
  }

  /**
   * Execute multiple validation rules in batch
   */
  async executeBatch(records: Record<string, any>[]): Promise<ValidationRuleExecutionResult[]> {
    const params = TenantContext.getCurrentScope();
    const response = await this.apiClient.post<ValidationRuleExecutionResult[]>(
      `/validation-rules/execute-batch?${new URLSearchParams(params)}`,
      { records }
    );
    return response;
  }

  /**
   * Get audit trail for a validation rule
   */
  async getAuditTrail(
    ruleId: string,
    page: number = 1
  ): Promise<{
    data: any[];
    pagination: { page: number; limit: number; total: number };
  }> {
    const params = new URLSearchParams({
      page: page.toString(),
      ...TenantContext.getCurrentScope(),
    });
    const response = await this.apiClient.get(
      `/validation-rules/${ruleId}/audit?${params}`
    );
    return response;
  }

  /**
   * Get all API endpoints for validation operations
   */
  async listValidationEndpoints(
    page: number = 1,
    limit: number = 50
  ): Promise<{ data: any[]; pagination: { page: number; limit: number; total: number } }> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
      category: 'validation',
      ...TenantContext.getCurrentScope(),
    });
    const response = await this.apiClient.get(
      `/api-endpoints?${params}`
    );
    return response;
  }

  /**
   * Get endpoints for a specific entity
   */
  async getEntityEndpoints(
    entityId: string,
    page: number = 1,
    limit: number = 50
  ): Promise<{ data: any[]; pagination: { page: number; limit: number; total: number } }> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
      ...TenantContext.getCurrentScope(),
    });
    const response = await this.apiClient.get(
      `/entities/${entityId}/api-endpoints?${params}`
    );
    return response;
  }
}

// Export singleton instance
export const validationRulesService = new ValidationRulesService(new APIClient());
```

## Updated EntityDetailsPage.tsx

Update the component to use the API service:

```typescript
import React, { useState, useEffect, useCallback } from 'react';
import { Card, Button, Space, Alert, Skeleton, Table, Tooltip, Popconfirm } from 'antd';
import { PlusOutlined, ReloadOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import styles from './EntityDetailsPage.module.css';
import AdvancedRuleConfiguration from '../components/AdvancedRuleConfiguration';
import { validationRulesService, ValidationRule } from '../services/validationRulesService';
import { TenantContext } from '../context/TenantContext';

interface Entity {
  id: string;
  name: string;
  businessName: string;
  [key: string]: any;
}

interface ValidationRulesContainerProps {
  rules: ValidationRule[];
  onRulesUpdate: (rules: ValidationRule[]) => void;
  onCrossEntitySave: (condition: any) => void;
  entity: Entity;
}

const ValidationRulesContainer: React.FC<ValidationRulesContainerProps> = ({
  rules,
  onRulesUpdate,
  onCrossEntitySave,
  entity,
}) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);

  // Fetch validation rules on component mount and when entity changes
  useEffect(() => {
    if (entity?.id && TenantContext.getCurrentScope()?.tenant_id) {
      loadRules();
    }
  }, [entity?.id, TenantContext.getCurrentScope()?.tenant_id]);

  const loadRules = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await validationRulesService.listRules(1, 100, {
        entityId: entity.id,
      });
      
      onRulesUpdate(response.data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to load validation rules'
      );
      console.error('Error loading validation rules:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRule = useCallback(async (ruleData: any) => {
    try {
      setLoading(true);
      const newRule = await validationRulesService.createRule({
        name: ruleData.name,
        description: ruleData.description,
        condition: ruleData.condition,
        entityIds: [entity.id],
        status: 'active',
      });
      
      onRulesUpdate([...rules, newRule]);
      setShowCreateForm(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create rule');
      console.error('Error creating rule:', err);
    } finally {
      setLoading(false);
    }
  }, [entity.id, rules, onRulesUpdate]);

  const handleUpdateRule = useCallback(async (ruleId: string, ruleData: any) => {
    try {
      setLoading(true);
      const updatedRule = await validationRulesService.updateRule(ruleId, {
        name: ruleData.name,
        description: ruleData.description,
        condition: ruleData.condition,
        status: ruleData.status,
      });
      
      onRulesUpdate(rules.map((r) => (r.id === ruleId ? updatedRule : r)));
      setEditingRuleId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update rule');
      console.error('Error updating rule:', err);
    } finally {
      setLoading(false);
    }
  }, [rules, onRulesUpdate]);

  const handleDeleteRule = useCallback(async (ruleId: string) => {
    try {
      setLoading(true);
      await validationRulesService.deleteRule(ruleId);
      onRulesUpdate(rules.filter((r) => r.id !== ruleId));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete rule');
      console.error('Error deleting rule:', err);
    } finally {
      setLoading(false);
    }
  }, [rules, onRulesUpdate]);

  const handleExecuteRule = useCallback(async (ruleId: string) => {
    try {
      setLoading(true);
      const result = await validationRulesService.executeRule(ruleId, {
        id: entity.id,
        ...entity,
      });
      
      if (!result.isValid) {
        setError(`Validation failed: ${result.errors.map((e) => e.message).join(', ')}`);
      } else {
        // Success - could show toast or modal
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to execute rule');
      console.error('Error executing rule:', err);
    } finally {
      setLoading(false);
    }
  }, [entity]);

  const rulesTableColumns = [
    {
      title: 'Rule Name',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: {
        showTitle: false,
      },
      render: (description: string) => (
        <Tooltip title={description}>
          {description?.substring(0, 50)}...
        </Tooltip>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <span style={{ color: status === 'active' ? '#52c41a' : '#d9d9d9' }}>
          {status === 'active' ? '✓ Active' : '○ Inactive'}
        </span>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: ValidationRule) => (
        <Space size="small">
          <Tooltip title="Edit Rule">
            <Button
              type="text"
              icon={<EditOutlined />}
              size="small"
              onClick={() => setEditingRuleId(record.id)}
            />
          </Tooltip>
          <Tooltip title="Execute Rule">
            <Button
              type="text"
              icon={<ReloadOutlined />}
              size="small"
              onClick={() => handleExecuteRule(record.id)}
              disabled={loading}
            />
          </Tooltip>
          <Popconfirm
            title="Delete Rule"
            description="Are you sure you want to delete this rule?"
            onConfirm={() => handleDeleteRule(record.id)}
            okText="Delete"
            cancelText="Cancel"
            okButtonProps={{ danger: true }}
          >
            <Tooltip title="Delete Rule">
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
                size="small"
                disabled={loading}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className={styles.validationRulesContainer}>
      <div className={styles.validationRulesHeader}>
        <div>
          <h5 className={styles.validationRulesTitle}>
            Validation Rules for {entity?.businessName || entity?.name}
          </h5>
          <p className={styles.validationRulesDescription}>
            Define and manage business logic and data quality rules for this entity. Rules are automatically executed during data operations.
          </p>
        </div>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={loadRules}
            loading={loading}
          >
            Refresh
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setShowCreateForm(true)}
            disabled={loading}
          >
            New Rule
          </Button>
        </Space>
      </div>

      {error && (
        <Alert
          message="Error"
          description={error}
          type="error"
          closable
          onClose={() => setError(null)}
          style={{ marginBottom: 16 }}
        />
      )}

      {loading && <Skeleton active paragraph={{ rows: 4 }} />}

      {!loading && showCreateForm && (
        <Card
          className={styles.validationRulesCard}
          title="Create New Validation Rule"
          extra={
            <Button
              type="text"
              onClick={() => setShowCreateForm(false)}
            >
              Cancel
            </Button>
          }
          style={{ marginBottom: 16 }}
        >
          <AdvancedRuleConfiguration
            onSave={handleCreateRule}
            entity={entity}
            onCrossEntitySave={onCrossEntitySave}
          />
        </Card>
      )}

      {!loading && editingRuleId && (
        <Card
          className={styles.validationRulesCard}
          title="Edit Validation Rule"
          extra={
            <Button
              type="text"
              onClick={() => setEditingRuleId(null)}
            >
              Cancel
            </Button>
          }
          style={{ marginBottom: 16 }}
        >
          <AdvancedRuleConfiguration
            rule={rules.find((r) => r.id === editingRuleId)}
            onSave={(data) => handleUpdateRule(editingRuleId, data)}
            entity={entity}
            onCrossEntitySave={onCrossEntitySave}
          />
        </Card>
      )}

      <Card className={styles.validationRulesCard}>
        {rules.length === 0 ? (
          <Alert
            message="No Validation Rules"
            description="Create your first validation rule to get started."
            type="info"
            showIcon
          />
        ) : (
          <Table
            dataSource={rules}
            columns={rulesTableColumns}
            rowKey="id"
            pagination={false}
            loading={loading}
            size="small"
          />
        )}
      </Card>
    </div>
  );
};

export default ValidationRulesContainer;
```

## Updated EntityConfigPageV2.tsx

Apply similar integration pattern to EntityConfigPageV2:

```typescript
import React, { useState, useEffect } from 'react';
import { Tabs } from 'antd';
import ValidationRulesContainer from './ValidationRulesContainer';
import { validationRulesService, ValidationRule } from '../services/validationRulesService';

interface EntityConfigPageV2Props {
  entityKey: string;
  // ... other props
}

const EntityConfigPageV2: React.FC<EntityConfigPageV2Props> = ({ entityKey, ...props }) => {
  const [validationRules, setValidationRules] = useState<ValidationRule[]>([]);

  const tabItems = [
    {
      key: 'entity',
      label: '⚙️ Entity',
      children: <EntityDetailsForm {...props} />,
    },
    {
      key: 'objects',
      label: '🔗 Related Objects',
      children: <RelatedObjectsPanel {...props} />,
    },
    {
      key: 'validations',
      label: '⚡ Validations',
      children: (
        <ValidationRulesContainer
          rules={validationRules}
          onRulesUpdate={setValidationRules}
          entity={props.entity}
          onCrossEntitySave={props.onCrossEntitySave}
        />
      ),
    },
  ];

  return <Tabs items={tabItems} />;
};
```

## Error Handling

Implement comprehensive error handling:

```typescript
export class ValidationRulesServiceError extends Error {
  constructor(
    public statusCode: number,
    public errorCode: string,
    message: string
  ) {
    super(message);
    this.name = 'ValidationRulesServiceError';
  }
}

// In service:
async listRules(...) {
  try {
    const response = await this.apiClient.get(`/validation-rules?${params}`);
    return response;
  } catch (err) {
    if (err instanceof APIError) {
      throw new ValidationRulesServiceError(
        err.statusCode,
        err.errorCode,
        err.message
      );
    }
    throw err;
  }
}
```

## State Management (Optional: React Query)

For better state management, consider using React Query:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

const useValidationRules = (entityId: string) => {
  return useQuery(
    ['validationRules', entityId],
    () => validationRulesService.listRules(1, 100, { entityId }),
    { enabled: !!entityId }
  );
};

const useCreateValidationRule = () => {
  const queryClient = useQueryClient();
  return useMutation(
    (data: ValidationRuleRequest) => validationRulesService.createRule(data),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['validationRules']);
      },
    }
  );
};
```

## Best Practices

### 1. Tenant Scope Management
Always ensure tenant scope is set before making API calls:

```typescript
if (!TenantContext.getCurrentScope()?.tenant_id) {
  setError('Please select a tenant first');
  return;
}
```

### 2. Loading States
Show appropriate loading indicators:

```typescript
{loading && <Skeleton active />}
```

### 3. Error Handling
Provide user-friendly error messages:

```typescript
setError(err instanceof Error ? err.message : 'An error occurred');
```

### 4. Pagination
Implement pagination for large datasets:

```typescript
const [page, setPage] = useState(1);
const response = await validationRulesService.listRules(page, 50);
```

### 5. Refresh Patterns
Allow manual refresh of data:

```typescript
<Button onClick={loadRules} icon={<ReloadOutlined />}>
  Refresh
</Button>
```

## Testing

### Mock Service
```typescript
jest.mock('../services/validationRulesService');

const mockService = validationRulesService as jest.Mock;
mockService.listRules.mockResolvedValue({
  data: mockRules,
  pagination: { page: 1, limit: 50, total: 3 },
});
```

### Component Tests
```typescript
test('displays validation rules after loading', async () => {
  render(<ValidationRulesContainer {...props} />);
  
  await waitFor(() => {
    expect(screen.getByText('Rule 1')).toBeInTheDocument();
  });
});
```

## Deployment Checklist

- [ ] Backend migrations applied
- [ ] API endpoints registered and tested
- [ ] API endpoints catalog seeded
- [ ] Frontend service layer implemented
- [ ] Component integration complete
- [ ] Error handling implemented
- [ ] Loading states added
- [ ] Tests passing
- [ ] Tenant scope validated
- [ ] Documentation updated
