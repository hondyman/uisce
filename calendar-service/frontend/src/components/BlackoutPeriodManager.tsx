import React, { useState, useEffect } from 'react';
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
  Card,
  Row,
  Col,
  Badge,
  Popconfirm,
  Statistic,
  Empty,
  Tag,
} from 'antd';
import {
  DeleteOutlined,
  PlusOutlined,
  LockOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const BlackoutPeriodManager = ({ profileId, tenantId }) => {
  const [form] = Form.useForm();
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [blackoutPeriods, setBlackoutPeriods] = useState([]);
  const [loading, setLoading] = useState(false);
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

  // Fetch blackout periods
  const fetchBlackoutPeriods = async () => {
    setLoading(true);
    try {
      const fromDate = dayjs().startOf('month').toISOString();
      const toDate = dayjs().endOf('month').add(3, 'month').toISOString();

      const response = await fetch(
        `/api/v1/blackout-periods?profile_id=${profileId}&from_date=${fromDate}&to_date=${toDate}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setBlackoutPeriods(data.blackout_periods || []);
    } catch (error) {
      console.error('Failed to fetch blackout periods:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBlackoutPeriods();
  }, [profileId, tenantId]);

  const createBlackoutPeriod = async (values) => {
    try {
      const payload = {
        profile_id: profileId,
        start_time: values.startTime.toISOString(),
        end_time: values.endTime.toISOString(),
        reason: values.reason,
        timezone_id: values.timezoneId,
      };

      const response = await fetch('/api/v1/blackout-periods', {
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
      await fetchBlackoutPeriods();
    } catch (error) {
      console.error('Failed to create blackout period:', error);
    }
  };

  const deleteBlackoutPeriod = async (id) => {
    try {
      const response = await fetch(`/api/v1/blackout-periods/${id}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': tenantId,
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      await fetchBlackoutPeriods();
    } catch (error) {
      console.error('Failed to delete blackout period:', error);
    }
  };

  const columns = [
    {
      title: 'Reason',
      dataIndex: 'reason',
      key: 'reason',
      width: '20%',
      render: (text) => <strong>{text}</strong>,
    },
    {
      title: 'Start',
      dataIndex: 'startTime',
      key: 'startTime',
      width: '18%',
      render: (text) => dayjs(text).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'End',
      dataIndex: 'endTime',
      key: 'endTime',
      width: '18%',
      render: (text) => dayjs(text).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Duration',
      key: 'duration',
      width: '12%',
      render: (_, record) => {
        const duration = dayjs(record.endTime).diff(
          dayjs(record.startTime),
          'hour'
        );
        return `${duration}h`;
      },
    },
    {
      title: 'Timezone',
      dataIndex: 'timezoneId',
      key: 'timezoneId',
      width: '15%',
      render: (text) => <Tag>{text}</Tag>,
    },
    {
      title: 'Status',
      key: 'status',
      width: '10%',
      render: (_, record) => {
        const now = dayjs();
        const start = dayjs(record.startTime);
        const end = dayjs(record.endTime);

        if (now.isBefore(start)) {
          return <Badge status="processing" text="Upcoming" />;
        } else if (now.isAfter(end)) {
          return <Badge status="default" text="Completed" />;
        } else {
          return <Badge status="error" text="Active" />;
        }
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: '10%',
      render: (_, record) => (
        <Popconfirm
          title="Delete Blackout Period"
          description="Are you sure you want to delete this blackout period?"
          onConfirm={() => deleteBlackoutPeriod(record.id)}
          okText="Yes"
          cancelText="No"
        >
          <Button type="primary" danger size="small" icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  // Calculate statistics
  const activePeriods = blackoutPeriods.filter((p) => {
    const now = dayjs();
    return now.isAfter(dayjs(p.startTime)) && now.isBefore(dayjs(p.endTime));
  });

  const upcomingPeriods = blackoutPeriods.filter((p) => {
    return dayjs(p.startTime).isAfter(dayjs());
  });

  const totalDowntime = blackoutPeriods.reduce((total, p) => {
    return total + dayjs(p.endTime).diff(dayjs(p.startTime), 'hour');
  }, 0);

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <span>
            <LockOutlined /> Blackout Periods Manager
          </span>
        }
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setIsModalVisible(true)}
          >
            Add Blackout Period
          </Button>
        }
      >
        {activePeriods.length > 0 && (
          <Alert
            message="Active Blackout Period"
            description={`${activePeriods.length} blackout period(s) are currently active`}
            type="warning"
            showIcon
            closable
            style={{ marginBottom: '16px' }}
          />
        )}

        <Row gutter={16} style={{ marginBottom: '20px' }}>
          <Col span={6}>
            <Statistic
              title="Total Periods"
              value={blackoutPeriods.length}
              prefix={<LockOutlined />}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="Active Now"
              value={activePeriods.length}
              valueStyle={{
                color: activePeriods.length > 0 ? '#cf1322' : '#52c41a',
              }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="Upcoming"
              value={upcomingPeriods.length}
              valueStyle={{ color: '#faad14' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="Total Downtime"
              value={totalDowntime}
              suffix="hours"
            />
          </Col>
        </Row>

        {blackoutPeriods.length === 0 ? (
          <Empty description="No blackout periods scheduled" />
        ) : (
          <Table
            columns={columns}
            dataSource={blackoutPeriods}
            loading={loading}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            size="small"
          />
        )}
      </Card>

      {/* Create Modal */}
      <Modal
        title="Create Blackout Period"
        visible={isModalVisible}
        onCancel={() => {
          setIsModalVisible(false);
          form.resetFields();
        }}
        footer={null}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={createBlackoutPeriod}
        >
          <Form.Item
            label="Reason"
            name="reason"
            rules={[
              {
                required: true,
                message: 'Please provide a reason for the blackout',
              },
            ]}
            tooltip="e.g., Maintenance, Team Meeting, System Upgrade, Office Closed"
          >
            <Input placeholder="e.g., System maintenance, office closure" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Start Date & Time"
                name="startTime"
                rules={[
                  {
                    required: true,
                    message: 'Select start time',
                  },
                ]}
              >
                <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="End Date & Time"
                name="endTime"
                rules={[
                  {
                    required: true,
                    message: 'Select end time',
                  },
                ]}
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

          <Alert
            message="Note"
            description="During blackout periods, no events can be scheduled in this profile."
            type="info"
            showIcon
            style={{ marginBottom: '16px' }}
          />

          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              Create Blackout Period
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default BlackoutPeriodManager;
