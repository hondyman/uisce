import React, { useState, useEffect } from 'react';
import { Table, Button, Tag, Space, Modal, Form, Select, Input, Switch, message, Popconfirm, Spin, Empty } from 'antd';
import { DeleteOutlined, EditOutlined, EyeOutlined, SyncOutlined } from '@ant-design/icons';
import { gql, useQuery, useMutation } from '@apollo/client';

// GraphQL Queries and Mutations
const LIST_EXTERNAL_SYNC_CONFIGS = gql`
  query ListExternalSyncConfigs($tenantId: UUID!) {
    external_sync_configs(where: { tenant_id: { _eq: $tenantId } }) {
      id
      profile_id
      provider
      country_code
      sync_enabled
      sync_frequency
      last_sync_at
      next_sync_at
      created_at
    }
  }
`;

const CREATE_EXTERNAL_SYNC_CONFIG = gql`
  mutation CreateExternalSyncConfig($input: CreateExternalSyncConfigInput!) {
    createExternalSyncConfig(input: $input) {
      id
      provider
      country_code
      sync_enabled
      sync_frequency
      created_at
    }
  }
`;

const UPDATE_EXTERNAL_SYNC_CONFIG = gql`
  mutation UpdateExternalSyncConfig($id: UUID!, $input: UpdateExternalSyncConfigInput!) {
    updateExternalSyncConfig(id: $id, input: $input) {
      id
      sync_enabled
      sync_frequency
      country_code
      update_at
    }
  }
`;

const DELETE_EXTERNAL_SYNC_CONFIG = gql`
  mutation DeleteExternalSyncConfig($id: UUID!) {
    deleteExternalSyncConfig(id: $id) {
      success
    }
  }
`;

const TRIGGER_SYNC = gql`
  mutation TriggerSync($configId: UUID!) {
    triggerSync(configId: $configId) {
      id
      status
      holidays_added
      execution_time_ms
      executed_at
    }
  }
`;

interface ExternalSyncConfig {
  id: string;
  profile_id: string;
  provider: 'nager_date' | 'calendarific';
  country_code: string;
  sync_enabled: boolean;
  sync_frequency: 'weekly' | 'monthly' | 'yearly';
  last_sync_at?: string;
  next_sync_at?: string;
  created_at: string;
}

interface ExternalSyncConfigListProps {
  profileId?: string;
  tenantId: string;
}

const ExternalSyncConfigList: React.FC<ExternalSyncConfigListProps> = ({ profileId, tenantId }) => {
  const [formVisible, setFormVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState<ExternalSyncConfig | null>(null);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState<ExternalSyncConfig | null>(null);
  const [form] = Form.useForm();

  // GraphQL hooks
  const { data, loading, refetch } = useQuery(LIST_EXTERNAL_SYNC_CONFIGS, {
    variables: { tenantId },
    skip: !tenantId,
  });

  const [createConfig] = useMutation(CREATE_EXTERNAL_SYNC_CONFIG, {
    onCompleted: () => {
      message.success('Sync configuration created');
      setFormVisible(false);
      form.resetFields();
      refetch();
    },
    onError: () => {
      message.error('Failed to create sync configuration');
    },
  });

  const [updateConfig] = useMutation(UPDATE_EXTERNAL_SYNC_CONFIG, {
    onCompleted: () => {
      message.success('Sync configuration updated');
      setFormVisible(false);
      form.resetFields();
      setEditingConfig(null);
      refetch();
    },
    onError: () => {
      message.error('Failed to update sync configuration');
    },
  });

  const [deleteConfig] = useMutation(DELETE_EXTERNAL_SYNC_CONFIG, {
    onCompleted: () => {
      message.success('Sync configuration deleted');
      refetch();
    },
    onError: () => {
      message.error('Failed to delete sync configuration');
    },
  });

  const [triggerSync] = useMutation(TRIGGER_SYNC, {
    onCompleted: (result) => {
      message.success(`Sync completed: ${result.triggerSync.holidays_added} holidays added`);
      refetch();
    },
    onError: () => {
      message.error('Failed to trigger sync');
    },
  });

  const handleCreateClick = () => {
    form.resetFields();
    setEditingConfig(null);
    setFormVisible(true);
  };

  const handleEditClick = (config: ExternalSyncConfig) => {
    setEditingConfig(config);
    form.setFieldsValue({
      provider: config.provider,
      country_code: config.country_code,
      sync_enabled: config.sync_enabled,
      sync_frequency: config.sync_frequency,
    });
    setFormVisible(true);
  };

  const handleFormSubmit = async (values: any) => {
    try {
      if (editingConfig) {
        await updateConfig({
          variables: {
            id: editingConfig.id,
            input: {
              sync_enabled: values.sync_enabled,
              sync_frequency: values.sync_frequency,
              country_code: values.country_code,
            },
          },
        });
      } else {
        await createConfig({
          variables: {
            input: {
              profile_id: profileId,
              provider: values.provider,
              country_code: values.country_code,
              api_key: values.api_key,
              sync_enabled: values.sync_enabled,
              sync_frequency: values.sync_frequency,
            },
          },
        });
      }
    } catch (error) {
      console.error('Form submission error:', error);
    }
  };

  const handleDeleteClick = async (id: string) => {
    await deleteConfig({
      variables: { id },
    });
  };

  const handleTriggerSync = async (id: string) => {
    await triggerSync({
      variables: { configId: id },
    });
  };

  const configs = data?.external_sync_configs || [];

  // Filter by profileId if provided
  const filteredConfigs = profileId
    ? configs.filter((c: ExternalSyncConfig) => c.profile_id === profileId)
    : configs;

  const columns = [
    {
      title: 'Provider',
      dataIndex: 'provider',
      key: 'provider',
      render: (provider: string) => {
        const colors: Record<string, string> = {
          nager_date: 'blue',
          calendarific: 'green',
        };
        const labels: Record<string, string> = {
          nager_date: 'Nager.Date',
          calendarific: 'Calendarific',
        };
        return <Tag color={colors[provider] || 'default'}>{labels[provider] || provider}</Tag>;
      },
    },
    {
      title: 'Country Code',
      dataIndex: 'country_code',
      key: 'country_code',
      render: (code: string) => <Tag>{code.toUpperCase()}</Tag>,
    },
    {
      title: 'Status',
      dataIndex: 'sync_enabled',
      key: 'sync_enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'orange'}>
          {enabled ? 'Enabled' : 'Disabled'}
        </Tag>
      ),
    },
    {
      title: 'Frequency',
      dataIndex: 'sync_frequency',
      key: 'sync_frequency',
      render: (freq: string) => (
        <Tag>{freq.charAt(0).toUpperCase() + freq.slice(1)}</Tag>
      ),
    },
    {
      title: 'Last Sync',
      dataIndex: 'last_sync_at',
      key: 'last_sync_at',
      render: (date: string) => date ? new Date(date).toLocaleDateString() : 'Never',
    },
    {
      title: 'Next Sync',
      dataIndex: 'next_sync_at',
      key: 'next_sync_at',
      render: (date: string) => date ? new Date(date).toLocaleDateString() : 'Not scheduled',
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, config: ExternalSyncConfig) => (
        <Space size="small">
          <Button
            type="primary"
            ghost
            size="small"
            icon={<SyncOutlined />}
            onClick={() => handleTriggerSync(config.id)}
            title="Trigger sync immediately"
          >
            Sync Now
          </Button>
          <Button
            type="default"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              setSelectedConfig(config);
              setDetailVisible(true);
            }}
          >
            Details
          </Button>
          <Button
            type="default"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEditClick(config)}
          >
            Edit
          </Button>
          <Popconfirm
            title="Delete Sync Config"
            description="Are you sure you want to delete this sync configuration?"
            onConfirm={() => handleDeleteClick(config.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button type="default" danger size="small" icon={<DeleteOutlined />}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: '16px' }}>
        <Button type="primary" onClick={handleCreateClick}>
          Add Sync Configuration
        </Button>
      </div>

      <Spin spinning={loading}>
        {filteredConfigs.length === 0 ? (
          <Empty description="No sync configurations found" />
        ) : (
          <Table
            columns={columns}
            dataSource={filteredConfigs}
            rowKey="id"
            pagination={{ pageSize: 10, showSizeChanger: true }}
          />
        )}
      </Spin>

      {/* Create/Edit Form Modal */}
      <Modal
        title={editingConfig ? 'Edit Sync Configuration' : 'Create Sync Configuration'}
        open={formVisible}
        onOk={() => form.submit()}
        onCancel={() => {
          setFormVisible(false);
          setEditingConfig(null);
        }}
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleFormSubmit}
        >
          {!editingConfig && (
            <Form.Item
              name="provider"
              label="Provider"
              rules={[{ required: true, message: 'Provider is required' }]}
            >
              <Select placeholder="Select a provider">
                <Select.Option value="nager_date">Nager.Date (Free)</Select.Option>
                <Select.Option value="calendarific">Calendarific (Requires API Key)</Select.Option>
              </Select>
            </Form.Item>
          )}

          <Form.Item
            name="country_code"
            label="Country Code (ISO 3166-1 Alpha-2)"
            rules={[
              { required: true, message: 'Country code is required' },
              { pattern: /^[A-Z]{2}$/, message: 'Must be 2 uppercase letters' },
            ]}
          >
            <Input placeholder="e.g., US, GB, DE" maxLength={2} />
          </Form.Item>

          {!editingConfig && (
            <Form.Item
              name="api_key"
              label="API Key (if required)"
              tooltip="Required for Calendarific provider"
            >
              <Input.Password placeholder="Leave empty for Nager.Date" />
            </Form.Item>
          )}

          <Form.Item
            name="sync_frequency"
            label="Sync Frequency"
            rules={[{ required: true, message: 'Sync frequency is required' }]}
          >
            <Select placeholder="Select frequency">
              <Select.Option value="weekly">Weekly</Select.Option>
              <Select.Option value="monthly">Monthly</Select.Option>
              <Select.Option value="yearly">Yearly</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="sync_enabled"
            label="Enable Syncing"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* Detail View Modal */}
      {selectedConfig && (
        <Modal
          title="Sync Configuration Details"
          open={detailVisible}
          onCancel={() => setDetailVisible(false)}
          footer={[
            <Button key="close" onClick={() => setDetailVisible(false)}>
              Close
            </Button>,
          ]}
          width={600}
        >
          <div style={{ lineHeight: '2' }}>
            <div><strong>Provider:</strong> {selectedConfig.provider === 'nager_date' ? 'Nager.Date' : 'Calendarific'}</div>
            <div><strong>Country Code:</strong> <Tag>{selectedConfig.country_code.toUpperCase()}</Tag></div>
            <div><strong>Status:</strong> <Tag color={selectedConfig.sync_enabled ? 'green' : 'orange'}>
              {selectedConfig.sync_enabled ? 'Enabled' : 'Disabled'}
            </Tag></div>
            <div><strong>Sync Frequency:</strong> {selectedConfig.sync_frequency.charAt(0).toUpperCase() + selectedConfig.sync_frequency.slice(1)}</div>
            <div><strong>Last Sync:</strong> {selectedConfig.last_sync_at ? new Date(selectedConfig.last_sync_at).toLocaleString() : 'Never'}</div>
            <div><strong>Next Sync:</strong> {selectedConfig.next_sync_at ? new Date(selectedConfig.next_sync_at).toLocaleString() : 'Not scheduled'}</div>
            <div><strong>Created:</strong> {new Date(selectedConfig.created_at).toLocaleString()}</div>
            
            <div style={{ marginTop: '16px', padding: '12px', backgroundColor: '#f5f5f5', borderRadius: '4px' }}>
              <strong>About this sync:</strong>
              <p style={{ fontSize: '12px', marginTop: '8px' }}>
                This configuration automatically fetches holiday data from {selectedConfig.provider === 'nager_date' ? 'Nager.Date' : 'Calendarific'} 
                {' '}{selectedConfig.sync_frequency} and adds them to your schedule profile.
              </p>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

export default ExternalSyncConfigList;
