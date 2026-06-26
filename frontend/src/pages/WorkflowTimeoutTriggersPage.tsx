// @ts-nocheck
import React, { useState, useEffect } from 'react';
import { devDebug, devError } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';
import styles from './WorkflowTimeoutTriggersPage.module.css';

interface TimeoutTrigger {
  id?: string;
  workflow_name: string;
  step_name: string;
  due_hours: number;
  actions: TimeoutAction[];
  is_active: boolean;
}

interface TimeoutAction {
  percent: number;
  type: 'escalate' | 'notify' | 'log' | 'cancel';
  target: string;
  message: string;
}

const WORKFLOW_STEPS = {
  HireEmployee: ['DataEntry', 'ManagerApproval', 'HRReview', 'Onboarding'],
  OrderApproval: ['DataEntry', 'CreditApproval', 'ExecutiveReview'],
  InvoiceProcessing: ['DataEntry', 'ApprovalQueue', 'PaymentApproval'],
};

const ESCALATION_TARGETS = {
  escalate: ['hr_director', 'finance_director', 'accounting_manager', 'operations_lead'],
  notify: ['assignee', 'manager', 'hr', 'finance'],
  log: ['audit', 'compliance', 'system'],
  cancel: ['auto', 'manual_review'],
};

const WorkflowTimeoutTriggersPage: React.FC = () => {
  const [form] = Form.useForm();
  const [triggers, setTriggers] = useState<TimeoutTrigger[]>([]);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<TimeoutTrigger | null>(null);
  const [actions, setActions] = useState<TimeoutAction[]>([
    { percent: 80, type: 'notify', target: 'assignee', message: 'Approval overdue - notification sent' },
    { percent: 100, type: 'escalate', target: 'hr_director', message: 'Escalated to HR Director' },
  ]);

  // Fetch existing triggers
  useEffect(() => {
    fetchTriggers();
  }, []);

  const getTenantHeaders = () => {
    const tenantData = localStorage.getItem('selected_tenant');
    const datasourceData = localStorage.getItem('selected_datasource');
    
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (tenantData) {
      try {
        const tenant = JSON.parse(tenantData);
        headers['X-Tenant-ID'] = tenant.id;
      } catch (e) {
        devError('Failed to parse tenant', e);
      }
    }

    if (datasourceData) {
      try {
        const datasource = JSON.parse(datasourceData);
        headers['X-Tenant-Datasource-ID'] = datasource.id;
        headers['X-Tenant-Region'] = getSelectedRegion();
      } catch (e) {
        devError('Failed to parse datasource', e);
      }
    }

    return headers;
  };

  const fetchTriggers = async () => {
    setLoading(true);
    try {
      const tenantId = localStorage.getItem('selected_tenant');
      const datasourceId = localStorage.getItem('selected_datasource');
      
      if (!tenantId || !datasourceId) {
        message.warning('Please select a tenant first');
        setLoading(false);
        return;
      }

      const response = await fetch('/api/workflow-timeout-triggers', {
        method: 'GET',
        headers: getTenantHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setTriggers(Array.isArray(data) ? data : []);
    } catch (error) {
      devError('Failed to load triggers:', error);
      message.error('Failed to load timeout triggers');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      
      const tenantId = localStorage.getItem('selected_tenant');
      const datasourceId = localStorage.getItem('selected_datasource');
      
      if (!tenantId || !datasourceId) {
        message.error('Please select a tenant first');
        return;
      }

      const newTrigger: TimeoutTrigger = {
        id: editing?.id,
        workflow_name: values.workflow,
        step_name: values.step,
        due_hours: values.due_hours,
        actions: actions,
        is_active: true,
      };

      setLoading(true);

      if (editing?.id) {
        const response = await fetch(
          `/api/workflow-timeout-triggers/${editing.id}`,
          {
            method: 'PUT',
            headers: getTenantHeaders(),
            body: JSON.stringify(newTrigger),
          }
        );

        if (!response.ok) throw new Error('Update failed');
        
        setTriggers(triggers.map(t => t.id === editing.id ? newTrigger : t));
        message.success('Timeout trigger updated');
      } else {
        const response = await fetch('/api/workflow-timeout-triggers', {
          method: 'POST',
          headers: getTenantHeaders(),
          body: JSON.stringify(newTrigger),
        });

        if (!response.ok) throw new Error('Create failed');
        
        const created = await response.json();
        setTriggers([...triggers, created]);
        message.success('Timeout trigger created');
      }

      form.resetFields();
      setActions([
        { percent: 80, type: 'notify', target: 'assignee', message: '' },
        { percent: 100, type: 'escalate', target: 'hr_director', message: '' },
      ]);
      setEditing(null);
    } catch (error) {
      devError('Save error:', error);
      message.error('Failed to save timeout trigger');
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (trigger: TimeoutTrigger) => {
    setEditing(trigger);
    form.setFieldsValue({
      workflow: trigger.workflow_name,
      step: trigger.step_name,
      due_hours: trigger.due_hours,
    });
    setActions(trigger.actions);
  };

  const handleDelete = (id?: string) => {
    Modal.confirm({
      title: 'Delete Timeout Trigger',
      content: 'Are you sure you want to delete this timeout trigger?',
      okText: 'Yes',
      cancelText: 'No',
      onOk: async () => {
        try {
          const tenantId = localStorage.getItem('selected_tenant');
          const datasourceId = localStorage.getItem('selected_datasource');
          
          if (!tenantId || !datasourceId) {
            message.error('Please select a tenant first');
            return;
          }

          const response = await fetch(`/api/workflow-timeout-triggers/${id}`, {
            method: 'DELETE',
            headers: getTenantHeaders(),
          });

          if (!response.ok) throw new Error('Delete failed');
          
          setTriggers(triggers.filter(t => t.id !== id));
          message.success('Timeout trigger deleted');
        } catch (error) {
          devError('Delete error:', error);
          message.error('Failed to delete timeout trigger');
        }
      },
    });
  };

  const handleTestTrigger = (trigger: TimeoutTrigger) => {
    Modal.confirm({
      title: 'Test Timeout Trigger',
      content: `This will simulate a timeout for ${trigger.workflow_name}.${trigger.step_name}. Continue?`,
      okText: 'Yes',
      cancelText: 'No',
      onOk: async () => {
        try {
          const tenantId = localStorage.getItem('selected_tenant');
          const datasourceId = localStorage.getItem('selected_datasource');
          
          if (!tenantId || !datasourceId) {
            message.error('Please select a tenant first');
            return;
          }

          const response = await fetch(
            `/api/workflow-timeout-triggers/${trigger.id}/test`,
            {
              method: 'POST',
              headers: getTenantHeaders(),
            }
          );

          if (!response.ok) throw new Error('Test failed');
          
          const result = await response.json();
          message.success(
            `Timeout trigger tested: ${trigger.actions.map(a => a.type).join(', ')}`
          );
          devDebug('Test result:', result);
        } catch (error) {
          devError('Test error:', error);
          message.error('Failed to test timeout trigger');
        }
      },
    });
  };

  const columns = [
    {
      title: 'Workflow',
      dataIndex: 'workflow_name',
      key: 'workflow_name',
    },
    {
      title: 'Step',
      dataIndex: 'step_name',
      key: 'step_name',
    },
    {
      title: 'Due (Hours)',
      dataIndex: 'due_hours',
      key: 'due_hours',
    },
    {
      title: 'Actions',
      dataIndex: 'actions',
      key: 'actions',
      render: (actions: TimeoutAction[]) => (
        <span>
          {actions.map((a, i) => (
            <span key={i}>
              {a.percent}%: {a.type}
              {i < actions.length - 1 && ', '}
            </span>
          ))}
        </span>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (is_active: boolean) => (is_active ? '✅ Active' : '❌ Inactive'),
    },
    {
      title: 'Operations',
      key: 'operations',
      render: (_: any, record: TimeoutTrigger) => (
        <Space>
          <Button
            type="primary"
            size="small"
            icon={<PlayCircleOutlined />}
            onClick={() => handleTestTrigger(record)}
          >
            Test
          </Button>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            Edit
          </Button>
          <Button
            danger
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record.id)}
          >
            Delete
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className={styles.container}>
      <Card title="🎯 Workflow Timeout Triggers" className="main-card">
        <Form form={form} layout="vertical" className={styles.formContainer}>
          <div className={styles.formGrid}>
            <Form.Item
              name="workflow"
              label="Workflow"
              rules={[{ required: true, message: 'Select a workflow' }]}
            >
              <Select placeholder="Select workflow">
                {Object.keys(WORKFLOW_STEPS).map(w => (
                  <Select.Option key={w} value={w}>
                    {w}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="step"
              label="Step"
              rules={[{ required: true, message: 'Select a step' }]}
            >
              <Select placeholder="Select step">
                {form.getFieldValue('workflow') &&
                  WORKFLOW_STEPS[form.getFieldValue('workflow') as keyof typeof WORKFLOW_STEPS]?.map(s => (
                    <Select.Option key={s} value={s}>
                      {s}
                    </Select.Option>
                  ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="due_hours"
              label="Due (Hours)"
              rules={[{ required: true, message: 'Enter due hours' }]}
            >
              <InputNumber min={1} max={999} placeholder="48" />
            </Form.Item>
          </div>
        </Form>

        <Card title="⏰ Timeout Actions" className={styles.actionsCard}>
          {actions.map((action, index) => (
            <div key={index} className={styles.actionItem}>
              <div className={styles.actionGrid}>
                <span className={styles.percentBadge}>{action.percent}%</span>
                <Select
                  value={action.type}
                  onChange={(type) => {
                    const newActions = [...actions];
                    newActions[index].type = type;
                    setActions(newActions);
                  }}
                >
                  <Select.Option value="notify">📧 Notify</Select.Option>
                  <Select.Option value="escalate">⬆️ Escalate</Select.Option>
                  <Select.Option value="log">📝 Log</Select.Option>
                  <Select.Option value="cancel">❌ Cancel</Select.Option>
                </Select>
                <Select
                  value={action.target}
                  onChange={(target) => {
                    const newActions = [...actions];
                    newActions[index].target = target;
                    setActions(newActions);
                  }}
                >
                  {(ESCALATION_TARGETS[action.type] || []).map(t => (
                    <Select.Option key={t} value={t}>
                      {t}
                    </Select.Option>
                  ))}
                </Select>
                <Input
                  placeholder="Message"
                  value={action.message}
                  onChange={(e) => {
                    const newActions = [...actions];
                    newActions[index].message = e.target.value;
                    setActions(newActions);
                  }}
                />
                <Button
                  danger
                  size="small"
                  onClick={() => setActions(actions.filter((_, i) => i !== index))}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
          <Button
            type="dashed"
            icon={<PlusOutlined />}
            onClick={() => setActions([...actions, { percent: 100, type: 'log', target: 'audit', message: '' }])}
            className={styles.addActionButton}
          >
            Add Action
          </Button>
        </Card>

        <Space>
          <Button type="primary" size="large" onClick={handleSave} loading={loading}>
            {editing ? '💾 Update Trigger' : '➕ Create Trigger'}
          </Button>
          {editing && (
            <Button size="large" onClick={() => {
              setEditing(null);
              form.resetFields();
              setActions([]);
            }}>
              Cancel
            </Button>
          )}
        </Space>
      </Card>

      <Card title="📋 Existing Timeout Triggers" className={styles.triggersCard}>
        <Table
          columns={columns}
          dataSource={triggers}
          loading={loading}
          rowKey="id"
          pagination={{ pageSize: 10 }}
        />
      </Card>
    </div>
  );
};

export default WorkflowTimeoutTriggersPage;
