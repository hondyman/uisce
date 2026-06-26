// @ts-nocheck
import React, { useState } from 'react';
import {
  Form,
  Button,
  Select,
  Input,
  Card,
  Space,
  Tag,
  Switch,
  Divider,
  message,
} from 'antd';
import { useABAC } from './ABACProvider';
import type { ABACPolicy } from './ABACProvider';

/**
 * PolicyBuilder Component
 * 
 * UI for creating and editing ABAC policies.
 * Supports:
 * - Subject rules (roles, users, departments)
 * - Action rules (allowed/denied actions)
 * - Resource rules (types, exclusions)
 * - Environment rules (locations, time windows)
 * 
 * All policies are tenant-scoped and automatically enforce multi-tenancy.
 */

interface PolicyBuilderProps {
  onPolicySaved?: (policy: ABACPolicy) => void;
  initialPolicy?: ABACPolicy;
}

export const PolicyBuilder: React.FC<PolicyBuilderProps> = ({
  onPolicySaved,
  initialPolicy,
}) => {
  const { createPolicy, updatePolicy, canExecute } = useABAC();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [policy, setPolicy] = useState<Partial<ABACPolicy>>(
    initialPolicy || {
      name: '',
      effect: 'allow',
      priority: 100,
      enabled: true,
      subject: {},
      action: {},
      resource: {},
      environment: {},
    }
  );

  const handleSave = async () => {
    try {
      // Check if user has permission to create/edit policies
      const hasPermission = await canExecute('create_policy', 'abac');
      if (!hasPermission) {
        message.error('You do not have permission to create policies');
        return;
      }

      setLoading(true);

      const policyData = {
        ...policy,
        name: policy.name || 'Untitled Policy',
        priority: policy.priority || 100,
      } as ABACPolicy;

      let result;
      if (initialPolicy?.id) {
        result = await updatePolicy(initialPolicy.id, policyData);
        message.success('Policy updated');
      } else {
        result = await createPolicy(policyData as Omit<ABACPolicy, 'id'>);
        message.success('Policy created');
      }

      onPolicySaved?.(result);
      form.resetFields();
    } catch (error) {
      const msg = error instanceof Error ? error.message : 'Failed to save policy';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="policy-builder-container">
      <Card title="Create ABAC Policy">
        <Form form={form} layout="vertical">
          {/* Basic Info */}
          <Form.Item label="Policy Name" required>
            <Input
              placeholder="e.g., Compliance Officer Read-Only"
              value={policy.name}
              onChange={(e) => setPolicy({ ...policy, name: e.target.value })}
            />
          </Form.Item>

          <Form.Item label="Description">
            <Input.TextArea
              placeholder="Policy description"
              rows={2}
              value={policy.description}
              onChange={(e) =>
                setPolicy({ ...policy, description: e.target.value })
              }
            />
          </Form.Item>

          <Divider />

          {/* Effect */}
          <Space style={{ marginBottom: 16 }}>
            <span>Effect:</span>
            <Select
              style={{ width: 150 }}
              value={policy.effect}
              onChange={(value) => setPolicy({ ...policy, effect: value })}
              options={[
                { label: '✅ Allow', value: 'allow' },
                { label: '❌ Deny', value: 'deny' },
              ]}
            />
            <span>Priority:</span>
            <Input
              type="number"
              style={{ width: 80 }}
              value={policy.priority}
              onChange={(e) =>
                setPolicy({ ...policy, priority: parseInt(e.target.value) })
              }
            />
            <Switch
              checked={policy.enabled}
              onChange={(checked) => setPolicy({ ...policy, enabled: checked })}
            />
            <span>{policy.enabled ? 'Enabled' : 'Disabled'}</span>
          </Space>

          <Divider />

          {/* Subject Rules */}
          <h4>Subject Rules</h4>
          <Form.Item label="Roles">
            <Select
              mode="multiple"
              placeholder="e.g., ComplianceOfficer, Analyst, Admin"
              value={policy.subject?.roles}
              onChange={(roles) =>
                setPolicy({ ...policy, subject: { ...policy.subject, roles } })
              }
            />
          </Form.Item>

          <Form.Item label="Users">
            <Select
              mode="multiple"
              placeholder="Specific user UUIDs"
              value={policy.subject?.users}
              onChange={(users) =>
                setPolicy({ ...policy, subject: { ...policy.subject, users } })
              }
            />
          </Form.Item>

          <Form.Item label="Departments">
            <Select
              mode="multiple"
              placeholder="e.g., Compliance, Risk, Operations"
              value={policy.subject?.departments}
              onChange={(departments) =>
                setPolicy({
                  ...policy,
                  subject: { ...policy.subject, departments },
                })
              }
            />
          </Form.Item>

          <Divider />

          {/* Action Rules */}
          <h4>Action Rules</h4>
          <Form.Item label="Allowed Actions">
            <Select
              mode="multiple"
              placeholder="e.g., read, list, view"
              value={policy.action?.allowed}
              onChange={(allowed) =>
                setPolicy({ ...policy, action: { ...policy.action, allowed } })
              }
            />
          </Form.Item>

          <Form.Item label="Denied Actions">
            <Select
              mode="multiple"
              placeholder="e.g., create, edit, delete"
              value={policy.action?.denied}
              onChange={(denied) =>
                setPolicy({ ...policy, action: { ...policy.action, denied } })
              }
            />
          </Form.Item>

          <Divider />

          {/* Resource Rules */}
          <h4>Resource Rules</h4>
          <Form.Item label="Resource Types">
            <Select
              mode="multiple"
              placeholder="e.g., triggers, policies, processes"
              value={policy.resource?.types}
              onChange={(types) =>
                setPolicy({ ...policy, resource: { ...policy.resource, types } })
              }
            />
          </Form.Item>

          <Form.Item label="Excluded Resources">
            <Select
              mode="multiple"
              placeholder="Resources to exclude"
              value={policy.resource?.excluded}
              onChange={(excluded) =>
                setPolicy({
                  ...policy,
                  resource: { ...policy.resource, excluded },
                })
              }
            />
          </Form.Item>

          <Divider />

          {/* Environment Rules */}
          <h4>Environment Rules (Optional)</h4>
          <Form.Item label="Allowed Locations">
            <Select
              mode="multiple"
              placeholder="e.g., US, EU, JP"
              value={policy.environment?.locations}
              onChange={(locations) =>
                setPolicy({
                  ...policy,
                  environment: { ...policy.environment, locations },
                })
              }
            />
          </Form.Item>

          <Divider />

          {/* Actions */}
          <Space>
            <Button
              type="primary"
              onClick={handleSave}
              loading={loading}
              icon={<PlusOutlined />}
            >
              {initialPolicy ? 'Update' : 'Create'} Policy
            </Button>
          </Space>
        </Form>
      </Card>
    </div>
  );
};

export default PolicyBuilder;
