// @ts-nocheck
import React, { useState, useEffect } from 'react';
import { devLog } from '../../utils/devLogger';
import {
  Card,
  Form,
  Input,
  Select,
  InputNumber,
  Checkbox,
  Button,
  Table,
  Space,
  message,
  Modal,
  Tag,
  Collapse,
} from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import './WorkflowTimeoutTriggersPage.css';

// ============================================================================
// WORKFLOW TIMEOUT TRIGGERS PAGE
// ============================================================================
// Admin interface for creating and managing workflow step timeout triggers.
// Users can:
// 1. Create timeout triggers for any workflow step
// 2. Configure escalation, notification, and logging actions
// 3. Set due times and action thresholds (80%, 100%)
// 4. View/edit/delete existing triggers
// ============================================================================

interface TimeoutAction {
  percent: number;
  type: 'notify' | 'escalate' | 'log' | 'cancel';
  target: string;
  message: string;
}

interface TimeoutTrigger {
  id: string;
  workflow_name: string;
  step_name: string;
  due_hours: number;
  actions_json: TimeoutAction[];
  is_active: boolean;
  created_at: string;
}

interface FormData {
  workflow: string;
  step: string;
  due_hours: number;
  notify_enabled: boolean;
  escalate_enabled: boolean;
  log_enabled: boolean;
  escalate_target?: string;
  notify_message?: string;
}

// Sample workflows and steps (in real implementation, fetch from API)
const SAMPLE_WORKFLOWS = [
  { label: 'HireEmployee', value: 'HireEmployee' },
  { label: 'OrderApproval', value: 'OrderApproval' },
  { label: 'InvoiceProcessing', value: 'InvoiceProcessing' },
  { label: 'ProductLaunch', value: 'ProductLaunch' },
  { label: 'EmployeeTermination', value: 'EmployeeTermination' },
];

const WORKFLOW_STEPS: { [key: string]: string[] } = {
  HireEmployee: ['DataEntry', 'ManagerApproval', 'HRReview', 'OnboardingSetup'],
  OrderApproval: ['FinanceApproval', 'InventoryCheck', 'ShippingSetup'],
  InvoiceProcessing: ['PaymentSetup', 'Reconciliation', 'Archival'],
  ProductLaunch: ['PricingReview', 'MarketingApproval', 'ITSetup'],
  EmployeeTermination: ['OffboardingReview', 'HRFinalReview', 'DocumentsArchived'],
};

const ESCALATION_TARGETS = [
  { label: 'Manager', value: 'manager' },
  { label: 'Director', value: 'director' },
  { label: 'HR Director', value: 'hr_director' },
  { label: 'Finance Manager', value: 'finance_manager' },
  { label: 'C-Suite', value: 'c_suite' },
];

const WorkflowTimeoutTriggersPage: React.FC = () => {
  const [form] = Form.useForm();
  const [triggers, setTriggers] = useState<TimeoutTrigger[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>('');
  const [editingTrigger, setEditingTrigger] = useState<TimeoutTrigger | null>(null);
  const [isModalVisible, setIsModalVisible] = useState(false);

  // Fetch existing timeout triggers
  useEffect(() => {
    fetchTriggers();
  }, []);

  const fetchTriggers = async () => {
    setLoading(true);
    try {
      // In real implementation: const response = await api.get('/api/admin/timeout-triggers');
      // For now, use empty array
      setTriggers([]);
      message.success('Timeout triggers loaded');
    } catch (error) {
      console.error('Error fetching triggers:', error);
      message.error('Failed to load timeout triggers');
    } finally {
      setLoading(false);
    }
  };

  const handleWorkflowChange = (workflow: string) => {
    setSelectedWorkflow(workflow);
    form.setFieldsValue({ step: undefined });
  };

  const handleSubmit = async (values: FormData) => {
    try {
      // Build actions array
      const actions: TimeoutAction[] = [];

      if (values.notify_enabled) {
        actions.push({
          percent: 80,
          type: 'notify',
          target: 'assignee',
          message: values.notify_message || 'Approval due - please review',
        });
      }

      if (values.escalate_enabled) {
        actions.push({
          percent: 100,
          type: 'escalate',
          target: values.escalate_target || 'manager',
          message: `${values.workflow}.${values.step} approval overdue - escalated`,
        });
      }

      if (values.log_enabled) {
        actions.push({
          percent: 100,
          type: 'log',
          target: 'audit',
          message: `${values.workflow}.${values.step} timeout - escalation event`,
        });
      }

      const payload = {
        workflow_name: values.workflow,
        step_name: values.step,
        due_hours: values.due_hours,
        actions_json: actions,
      };

      // In real implementation: await api.post('/api/admin/timeout-triggers', payload);
      devLog('Submitting timeout trigger:', payload);

      message.success(
        `Timeout trigger created: ${values.workflow}.${values.step} (${values.due_hours}h)`
      );

      // Add to local state
      const newTrigger: TimeoutTrigger = {
        id: `trigger-${Date.now()}`,
        workflow_name: values.workflow,
        step_name: values.step,
        due_hours: values.due_hours,
        actions_json: actions,
        is_active: true,
        created_at: new Date().toISOString(),
      };

      setTriggers([...triggers, newTrigger]);
      form.resetFields();
      setIsModalVisible(false);
    } catch (error) {
      console.error('Error saving timeout trigger:', error);
      message.error('Failed to save timeout trigger');
    }
  };

  const handleEdit = (trigger: TimeoutTrigger) => {
    setEditingTrigger(trigger);
    form.setFieldsValue({
      workflow: trigger.workflow_name,
      step: trigger.step_name,
      due_hours: trigger.due_hours,
    });
    setSelectedWorkflow(trigger.workflow_name);
    setIsModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    Modal.confirm({
      title: 'Delete Timeout Trigger',
      content: 'Are you sure you want to delete this timeout trigger?',
      okText: 'Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          // In real implementation: await api.delete(`/api/admin/timeout-triggers/${id}`);
          setTriggers(triggers.filter(t => t.id !== id));
          message.success('Timeout trigger deleted');
        } catch (error) {
          message.error('Failed to delete timeout trigger');
        }
      },
    });
  };

  const columns = [
    {
      title: 'Workflow',
      dataIndex: 'workflow_name',
      key: 'workflow_name',
      width: 150,
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: 'Step',
      dataIndex: 'step_name',
      key: 'step_name',
      width: 150,
    },
    {
      title: 'Due Hours',
      dataIndex: 'due_hours',
      key: 'due_hours',
      width: 100,
      render: (hours: number) => (
        <Tag icon={<ClockCircleOutlined />} color="orange">
          {hours}h
        </Tag>
      ),
    },
    {
      title: 'Actions',
      dataIndex: 'actions_json',
      key: 'actions',
      render: (actions: TimeoutAction[]) => (
        <Space wrap>
          {actions.map((action, i) => (
            <Tag key={i} color={getActionColor(action.type)}>
              {action.type} ({action.percent}%)
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'is_active',
      key: 'status',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>{active ? 'Active' : 'Inactive'}</Tag>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      key: 'operations',
      width: 120,
      render: (_: any, record: TimeoutTrigger) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Button
            type="text"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record.id)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div className="timeout-triggers-container">
      <Card
        title={
          <div>
            <ClockCircleOutlined style={{ marginRight: '8px' }} />
            Workflow Timeout Triggers
          </div>
        }
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingTrigger(null);
              form.resetFields();
              setIsModalVisible(true);
            }}
          >
            New Timeout Trigger
          </Button>
        }
      >
        {/* FEATURE DESCRIPTION */}
        <Collapse
          items={[
            {
              key: '1',
              label: 'How Timeout Triggers Work',
              children: (
                <div>
                  <p>
                    Timeout triggers automatically monitor workflow steps for overdue
                    completions. When a step exceeds its due time, configured actions
                    are triggered:
                  </p>
                  <ul>
                    <li>
                      <strong>Notify (80%)</strong>: Send reminder email to assignee
                    </li>
                    <li>
                      <strong>Escalate (100%)</strong>: Automatically reassign to next level
                    </li>
                    <li>
                      <strong>Log (100%)</strong>: Record audit event for compliance
                    </li>
                  </ul>
                  <p>
                    <strong>Example:</strong> 48-hour Manager Approval
                    <br />
                    • Hour 38: Email sent: "Approval due in 10 hours"
                    <br />
                    • Hour 48: Auto-escalated to HR Director + audit logged
                  </p>
                </div>
              ),
            },
          ]}
        />

        {/* TIMEOUT TRIGGERS TABLE */}
        <div className="timeout-triggers-table">
          <Table
            dataSource={triggers}
            columns={columns}
            loading={loading}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            locale={{ emptyText: 'No timeout triggers configured' }}
          />
        </div>
      </Card>

      {/* CREATE/EDIT MODAL */}
      <Modal
        title={editingTrigger ? 'Edit Timeout Trigger' : 'Create Timeout Trigger'}
        visible={isModalVisible}
        okText={editingTrigger ? 'Update' : 'Create'}
        onOk={() => form.submit()}
        onCancel={() => {
          setIsModalVisible(false);
          setEditingTrigger(null);
          form.resetFields();
        }}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            notify_enabled: true,
            escalate_enabled: true,
            log_enabled: true,
            due_hours: 48,
          }}
        >
          {/* WORKFLOW SELECTION */}
          <Form.Item
            name="workflow"
            label="Workflow"
            rules={[{ required: true, message: 'Please select a workflow' }]}
          >
            <Select
              placeholder="Select workflow"
              options={SAMPLE_WORKFLOWS}
              onChange={handleWorkflowChange}
            />
          </Form.Item>

          {/* STEP SELECTION */}
          <Form.Item
            name="step"
            label="Step"
            rules={[{ required: true, message: 'Please select a step' }]}
          >
            <Select
              placeholder="Select step"
              options={
                selectedWorkflow && WORKFLOW_STEPS[selectedWorkflow]
                  ? WORKFLOW_STEPS[selectedWorkflow].map(step => ({
                      label: step,
                      value: step,
                    }))
                  : []
              }
            />
          </Form.Item>

          {/* DUE HOURS */}
          <Form.Item
            name="due_hours"
            label="Due Hours"
            rules={[
              { required: true, message: 'Please enter due hours' },
              { type: 'number', min: 1, message: 'Must be at least 1 hour' },
            ]}
          >
            <InputNumber min={1} max={999} />
          </Form.Item>

          {/* ACTIONS SECTION */}
          <Card size="small" title="Timeout Actions">
            {/* NOTIFY ACTION */}
            <Form.Item name="notify_enabled" valuePropName="checked">
              <Checkbox>
                Enable Notification (80% of due time)
              </Checkbox>
            </Form.Item>
            <Form.Item noStyle shouldUpdate={(prevValues, currentValues) => prevValues.notify_enabled !== currentValues.notify_enabled}>
              {({ getFieldValue }) =>
                getFieldValue('notify_enabled') ? (
                  <Form.Item
                    name="notify_message"
                    label="Notification Message"
                    rules={[{ required: true, message: 'Please enter notification message' }]}
                  >
                    <Input.TextArea
                      rows={2}
                      placeholder="E.g., Approval due - please review"
                    />
                  </Form.Item>
                ) : null
              }
            </Form.Item>

            {/* ESCALATE ACTION */}
            <Form.Item name="escalate_enabled" valuePropName="checked">
              <Checkbox>
                Enable Escalation (100% of due time)
              </Checkbox>
            </Form.Item>
            <Form.Item
              noStyle
              shouldUpdate={(prevValues, currentValues) => prevValues.escalate_enabled !== currentValues.escalate_enabled}
            >
              {({ getFieldValue }) =>
                getFieldValue('escalate_enabled') ? (
                  <Form.Item
                    name="escalate_target"
                    label="Escalate To"
                    rules={[{ required: true, message: 'Please select escalation target' }]}
                  >
                    <Select
                      placeholder="Select escalation target"
                      options={ESCALATION_TARGETS}
                    />
                  </Form.Item>
                ) : null
              }
            </Form.Item>

            {/* LOG ACTION */}
            <Form.Item name="log_enabled" valuePropName="checked">
              <Checkbox>
                Enable Audit Logging
              </Checkbox>
            </Form.Item>
          </Card>
        </Form>
      </Modal>
    </div>
  );
};

// HELPER: Get action color
function getActionColor(type: string): string {
  const colors: { [key: string]: string } = {
    notify: 'blue',
    escalate: 'red',
    log: 'green',
    cancel: 'orange',
  };
  return colors[type] || 'default';
}

export default WorkflowTimeoutTriggersPage;
