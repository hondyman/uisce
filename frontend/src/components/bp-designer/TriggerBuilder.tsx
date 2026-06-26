// @ts-nocheck
import React, { useState, useCallback } from 'react';
import { useNotification } from '../../hooks/useNotification';
import styles from './TriggerBuilder.module.css';

/**
 * TriggerBuilder Component
 * 
 * UI for configuring all 13 Workday trigger types:
 * 1. Save, 2. Field Change, 3. Delete, 4. Create, 5. Sub-Entity Change
 * 6. FK Relationship, 7. Integration Event, 8. Workflow Step, 9. Status Change
 * 10. Bulk Load, 11. Calculated Field, 12. Timeout, 13. Security Role
 */

const TRIGGER_TYPES = {
  // LIVE Triggers (7)
  'save': {
    label: 'Save',
    description: 'Fires when entity is persisted to database',
    status: 'live',
    icon: '💾',
  },
  'field_change': {
    label: 'Field Change',
    description: 'Fires when specific field is modified',
    status: 'live',
    icon: '✏️',
  },
  'delete': {
    label: 'Delete',
    description: 'Fires when entity is removed',
    status: 'live',
    icon: '🗑️',
  },
  'create': {
    label: 'Create',
    description: 'Fires when new entity is instantiated',
    status: 'live',
    icon: '✨',
  },
  'sub_entity_change': {
    label: 'Sub-Entity Change',
    description: 'Fires when child record in hierarchy is modified',
    status: 'live',
    icon: '🔗',
  },
  'fk_change': {
    label: 'FK Relationship',
    description: 'Fires when foreign key is updated',
    status: 'live',
    icon: '⚙️',
  },
  'integration_event': {
    label: 'Integration Event',
    description: 'Fires when external API/webhook triggers',
    status: 'live',
    icon: '🌐',
  },

  // PENDING Triggers (6)
  'workflow_step': {
    label: 'Workflow Step',
    description: 'Fires when BP step completes',
    status: 'pending',
    icon: '📋',
  },
  'status_change': {
    label: 'Status Change',
    description: 'Fires when status field updates',
    status: 'pending',
    icon: '📊',
  },
  'bulk_load': {
    label: 'Bulk Load',
    description: 'Fires when batch import (CSV) is processed',
    status: 'pending',
    icon: '📦',
  },
  'calculated_field': {
    label: 'Calculated Field',
    description: 'Fires when formula field recalculates',
    status: 'pending',
    icon: '🧮',
  },
  'timeout': {
    label: 'Timeout',
    description: 'Fires when timer expires (escalations)',
    status: 'pending',
    icon: '⏰',
  },
  'role_change': {
    label: 'Security Role',
    description: 'Fires when user role is assigned',
    status: 'pending',
    icon: '👤',
  },
};

const ESCALATION_ACTIONS = [
  { label: 'Notify Manager', value: 'notify' },
  { label: 'Escalate to Hierarchy', value: 'escalate' },
  { label: 'Auto-Approve', value: 'auto_approve' },
  { label: 'Auto-Reject', value: 'auto_reject' },
];

const TIMEOUT_UNITS = [
  { label: 'Hours', value: 'hours' },
  { label: 'Days', value: 'days' },
  { label: 'SLA', value: 'sla' },
  { label: 'Custom', value: 'custom' },
];

interface Trigger {
  id?: string;
  trigger_type: string;
  target_entity: string;
  event_config?: Record<string, any>;
  condition_config?: Record<string, any>[];
  action_config?: Record<string, any>;
  enabled: boolean;
}

interface TriggerBuilderProps {
  tenantId: string;
  datasourceId: string;
  onTriggersChange?: (triggers: Trigger[]) => void;
}

export const TriggerBuilder: React.FC<TriggerBuilderProps> = ({
  tenantId,
  datasourceId,
  onTriggersChange,
}) => {
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingTrigger, setEditingTrigger] = useState<Trigger | null>(null);
  const [form] = Form.useForm();

  const [formData, setFormData] = useState<Trigger>({
    trigger_type: 'save',
    target_entity: '',
    enabled: true,
    event_config: {},
    condition_config: [],
    action_config: {},
  });

  const handleOpenModal = (trigger?: Trigger) => {
    if (trigger) {
      setEditingTrigger(trigger);
      setFormData(trigger);
    } else {
      setEditingTrigger(null);
      setFormData({
        trigger_type: 'save',
        target_entity: '',
        enabled: true,
        event_config: {},
        condition_config: [],
        action_config: {},
      });
    }
    setIsModalOpen(true);
  };

  const notification = useNotification();

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setEditingTrigger(null);
  };

  const handleSaveTrigger = () => {
    if (!formData.target_entity) {
      notification.error('Please select a target entity');
      return;
    }

    if (editingTrigger?.id) {
      setTriggers(triggers.map(t => t.id === editingTrigger.id ? formData : t));
    } else {
      setTriggers([...triggers, { ...formData, id: Date.now().toString() }]);
    }

    onTriggersChange?.(triggers);
    handleCloseModal();
  };

  const handleDeleteTrigger = (id: string) => {
    setTriggers(triggers.filter(t => t.id !== id));
  };

  const getTriggerStatus = (triggerType: string) => {
    const trigger = TRIGGER_TYPES[triggerType as keyof typeof TRIGGER_TYPES];
    if (trigger?.status === 'live') return <Tag color="green">✅ Live</Tag>;
    return <Tag color="orange">⏳ Pending</Tag>;
  };

  const columns = [
    {
      title: 'Type',
      dataIndex: 'trigger_type',
      key: 'trigger_type',
      render: (type: string) => (
        <Space>
          <span>{TRIGGER_TYPES[type as keyof typeof TRIGGER_TYPES]?.icon}</span>
          <span>{TRIGGER_TYPES[type as keyof typeof TRIGGER_TYPES]?.label}</span>
        </Space>
      ),
    },
    {
      title: 'Target Entity',
      dataIndex: 'target_entity',
      key: 'target_entity',
    },
    {
      title: 'Status',
      key: 'status',
      render: (_: any, record: Trigger) => getTriggerStatus(record.trigger_type),
    },
    {
      title: 'Enabled',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean) => enabled ? <CheckCircleOutlined style={{ color: 'green' }} /> : '—',
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: Trigger) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleOpenModal(record)}
          />
          <Popconfirm
            title="Delete trigger?"
            onConfirm={() => record.id && handleDeleteTrigger(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className={styles.triggerBuilder}>
      <Card
        title="Validation Triggers (13 Types)"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => handleOpenModal()}
          >
            Add Trigger
          </Button>
        }
      >
        <Table
          dataSource={triggers}
          columns={columns}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Card>

      <Modal
        title={editingTrigger ? 'Edit Trigger' : 'Create Trigger'}
        open={isModalOpen}
        onOk={handleSaveTrigger}
        onCancel={handleCloseModal}
        width={700}
      >
        <Form layout="vertical">
          <Form.Item label="Trigger Type" required>
            <Select
              value={formData.trigger_type}
              onChange={(value) => setFormData({ ...formData, trigger_type: value })}
              options={Object.entries(TRIGGER_TYPES).map(([key, config]) => ({
                label: `${config.icon} ${config.label} ${config.status === 'live' ? '✅' : '⏳'}`,
                value: key,
                title: config.description,
              }))}
            />
          </Form.Item>

          <Form.Item label="Target Entity" required>
            <Input
              placeholder="e.g., orders, customers, employees"
              value={formData.target_entity}
              onChange={(e) => setFormData({ ...formData, target_entity: e.target.value })}
            />
          </Form.Item>

          {/* Conditional fields based on trigger type */}
          {formData.trigger_type === 'field_change' && (
            <Form.Item label="Watch Field">
              <Input
                placeholder="e.g., phone, status, total"
                value={formData.event_config?.field || ''}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    event_config: { ...formData.event_config, field: e.target.value },
                  })
                }
              />
            </Form.Item>
          )}

          {formData.trigger_type === 'status_change' && (
            <>
              <Form.Item label="From Status">
                <Input
                  placeholder="e.g., pending"
                  value={formData.event_config?.from || ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      event_config: { ...formData.event_config, from: e.target.value },
                    })
                  }
                />
              </Form.Item>
              <Form.Item label="To Status">
                <Input
                  placeholder="e.g., approved"
                  value={formData.event_config?.to || ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      event_config: { ...formData.event_config, to: e.target.value },
                    })
                  }
                />
              </Form.Item>
            </>
          )}

          {formData.trigger_type === 'timeout' && (
            <>
              <Form.Item label="Timeout Value" required>
                <Input
                  type="number"
                  placeholder="e.g., 2, 48, 7"
                  value={formData.event_config?.value || ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      event_config: { ...formData.event_config, value: e.target.value },
                    })
                  }
                />
              </Form.Item>
              <Form.Item label="Timeout Unit" required>
                <Select
                  value={formData.event_config?.unit}
                  onChange={(value) =>
                    setFormData({
                      ...formData,
                      event_config: { ...formData.event_config, unit: value },
                    })
                  }
                  options={TIMEOUT_UNITS}
                />
              </Form.Item>
              <Form.Item label="Escalation Action" required>
                <Select
                  value={formData.action_config?.escalation}
                  onChange={(value) =>
                    setFormData({
                      ...formData,
                      action_config: { ...formData.action_config, escalation: value },
                    })
                  }
                  options={ESCALATION_ACTIONS}
                />
              </Form.Item>
            </>
          )}

          <Form.Item label="Enabled">
            <Select
              value={formData.enabled ? 'true' : 'false'}
              onChange={(value) => setFormData({ ...formData, enabled: value === 'true' })}
              options={[
                { label: 'Yes', value: 'true' },
                { label: 'No', value: 'false' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TriggerBuilder;
