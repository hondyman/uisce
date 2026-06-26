import React, { useState, useRef, useEffect } from 'react';
import {
  Form,
  Input,
  Button,
  Modal,
  Table,
  Space,
  Alert,
  Select,
  DatePicker,
  TimePicker,
  Card,
  Row,
  Col,
  Badge,
  Popconfirm,
  Drawer,
  Tag,
  Divider,
  Statistic,
  Empty,
} from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery } from '@apollo/client';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';

dayjs.extend(utc);
dayjs.extend(timezone);

const RecurringEventManager = ({ profileId, tenantId }) => {
  const [form] = Form.useForm();
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState(null);
  const [selectedRuleId, setSelectedRuleId] = useState(null);
  const [showOccurrences, setShowOccurrences] = useState(false);
  const [occurrences, setOccurrences] = useState([]);
  const [detectedConflicts, setDetectedConflicts] = useState([]);
  const [availableTimezones, setAvailableTimezones] = useState([]);

  // Fetch available timezones
  useEffect(() => {
    fetch('/api/v1/timezones', {
      headers: {
        'X-Tenant-ID': tenantId,
      },
    })
      .then((res) => res.json())
      .then((data) => setAvailableTimezones(data.timezones || []))
      .catch(console.error);
  }, [tenantId]);

  // Fetch recurring rules
  const { data: rulesData, loading: rulesLoading, refetch: refetchRules } = useQuery(
    `query ListRecurrenceRules {
      listRecurrenceRules(profileId: "${profileId}") {
        id
        rrule
        startTime
        endTime
        timezoneId
        description
        maxOccurrence
        createdAt
      }
    }`
  );

  const createRecurrenceRule = async (values) => {
    try {
      const payload = {
        profile_id: profileId,
        rrule: values.rrule,
        start_time: values.startTime.toISOString(),
        end_time: values.endTime.toISOString(),
        timezone_id: values.timezoneId,
        max_occurrence: values.maxOccurrence || 100,
        description: values.description,
      };

      const response = await fetch('/api/v1/recurring-events', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      form.resetFields();
      setIsModalVisible(false);
      refetchRules();
    } catch (error) {
      console.error('Failed to create recurrence rule:', error);
    }
  };

  const deleteRecurrenceRule = async (id) => {
    try {
      const response = await fetch(`/api/v1/recurring-events/${id}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': tenantId,
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      refetchRules();
    } catch (error) {
      console.error('Failed to delete recurrence rule:', error);
    }
  };

  const generateOccurrences = async (ruleId) => {
    try {
      const fromDate = dayjs().startOf('month').toISOString();
      const toDate = dayjs().endOf('month').add(3, 'month').toISOString();

      const response = await fetch(`/api/v1/recurring-events/${ruleId}/occurrences`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          from_date: fromDate,
          to_date: toDate,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setOccurrences(data.occurrences || []);
      setSelectedRuleId(ruleId);
      setShowOccurrences(true);
    } catch (error) {
      console.error('Failed to generate occurrences:', error);
    }
  };

  const checkConflicts = async (startTime, endTime) => {
    try {
      const response = await fetch('/api/v1/conflicts/check', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          profile_id: profileId,
          start_time: startTime.toISOString(),
          end_time: endTime.toISOString(),
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setDetectedConflicts(data.conflicts || []);
      return data.conflicts;
    } catch (error) {
      console.error('Failed to check conflicts:', error);
      return [];
    }
  };

  const columns = [
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      width: '25%',
    },
    {
      title: 'RRULE',
      dataIndex: 'rrule',
      key: 'rrule',
      width: '30%',
      render: (text) => (
        <code style={{ fontSize: '11px', wordBreak: 'break-word' }}>{text}</code>
      ),
    },
    {
      title: 'Start Time',
      dataIndex: 'startTime',
      key: 'startTime',
      width: '15%',
      render: (text) => dayjs(text).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Timezone',
      dataIndex: 'timezoneId',
      key: 'timezoneId',
      width: '15%',
      render: (text) => <Tag>{text}</Tag>,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: '15%',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="primary"
            size="small"
            onClick={() => generateOccurrences(record.id)}
          >
            View
          </Button>
          <Popconfirm
            title="Delete Recurring Event"
            description="Are you sure you want to delete this recurring event?"
            onConfirm={() => deleteRecurrenceRule(record.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button type="primary" danger size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const occurrenceColumns = [
    {
      title: 'Date',
      dataIndex: 'startTime',
      key: 'date',
      render: (text) => dayjs(text).format('YYYY-MM-DD'),
    },
    {
      title: 'Start Time',
      dataIndex: 'startTime',
      key: 'startTime',
      render: (text) => dayjs(text).format('HH:mm:ss'),
    },
    {
      title: 'End Time',
      dataIndex: 'endTime',
      key: 'endTime',
      render: (text) => dayjs(text).format('HH:mm:ss'),
    },
    {
      title: 'Duration',
      key: 'duration',
      render: (_, record) => {
        const duration = dayjs(record.endTime).diff(record.startTime, 'minute');
        return `${Math.floor(duration / 60)}h ${duration % 60}m`;
      },
    },
    {
      title: '#',
      dataIndex: 'occurrenceNum',
      key: 'occurrenceNum',
      width: '60px',
    },
  ];

  const rules = rulesData?.listRecurrenceRules || [];

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <span>
            <CalendarOutlined /> Recurring Events Manager
          </span>
        }
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setIsModalVisible(true)}
          >
            Add Recurring Event
          </Button>
        }
      >
        {detectedConflicts.length > 0 && (
          <Alert
            message="Conflicts Detected"
            description={`Found ${detectedConflicts.length} scheduling conflict(s)`}
            type="warning"
            showIcon
            closable
            style={{ marginBottom: '16px' }}
          />
        )}

        <Table
          columns={columns}
          dataSource={rules}
          loading={rulesLoading}
          rowKey="id"
          pagination={{ pageSize: 10 }}
        />
      </Card>

      {/* Create/Edit Modal */}
      <Modal
        title={editingRule ? 'Edit Recurring Event' : 'Create Recurring Event'}
        visible={isModalVisible}
        onCancel={() => {
          setIsModalVisible(false);
          setEditingRule(null);
          form.resetFields();
        }}
        footer={null}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={createRecurrenceRule}
          initialValues={{
            maxOccurrence: 100,
          }}
        >
          <Form.Item
            label="Description"
            name="description"
            rules={[{ required: true, message: 'Please provide a description' }]}
          >
            <Input placeholder="e.g., Weekly team meetings" />
          </Form.Item>

          <Form.Item
            label="RRULE (RFC 5545 Format)"
            name="rrule"
            rules={[{ required: true, message: 'Please provide RRULE' }]}
            tooltip="Examples: FREQ=DAILY, FREQ=WEEKLY;BYDAY=MO,WE,FR, FREQ=MONTHLY;BYMONTHDAY=1"
          >
            <Input.TextArea
              placeholder="FREQ=WEEKLY;BYDAY=MO,WE,FR;UNTIL=20260630"
              rows={3}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Start Time"
                name="startTime"
                rules={[{ required: true, message: 'Select start time' }]}
              >
                <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="End Time"
                name="endTime"
                rules={[{ required: true, message: 'Select end time' }]}
              >
                <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="Timezone"
            name="timezoneId"
            rules={[{ required: true, message: 'Select timezone' }]}
          >
            <Select
              placeholder="Select timezone"
              showSearch
              optionLabelProp="label"
            >
              {availableTimezones.map((tz) => (
                <Select.Option key={tz} value={tz} label={tz}>
                  {tz}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            label="Max Occurrences"
            name="maxOccurrence"
            tooltip="Maximum number of occurrences to generate"
          >
            <Input type="number" min={1} max={365} />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              {editingRule ? 'Update' : 'Create'} Recurring Event
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      {/* Occurrences Drawer */}
      <Drawer
        title={`Generated Occurrences (${occurrences.length})`}
        placement="right"
        width={800}
        onClose={() => setShowOccurrences(false)}
        open={showOccurrences}
      >
        {occurrences.length === 0 ? (
          <Empty description="No occurrences generated" />
        ) : (
          <>
            <Row gutter={16} style={{ marginBottom: '20px' }}>
              <Col span={8}>
                <Statistic
                  title="Total Occurrences"
                  value={occurrences.length}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="Date Range"
                  value={
                    occurrences.length > 0
                      ? `${dayjs(occurrences[0].startTime).format(
                          'MM/DD'
                        )} - ${dayjs(
                          occurrences[occurrences.length - 1].startTime
                        ).format('MM/DD')}`
                      : 'N/A'
                  }
                  valueStyle={{ fontSize: '14px' }}
                />
              </Col>
            </Row>
            <Table
              columns={occurrenceColumns}
              dataSource={occurrences}
              rowKey={(record, index) => index}
              pagination={{ pageSize: 15 }}
              size="small"
            />
          </>
        )}
      </Drawer>
    </div>
  );
};

export default RecurringEventManager;
