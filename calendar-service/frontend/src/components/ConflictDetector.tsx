import React, { useState } from 'react';
import {
  Form,
  Input,
  Button,
  Card,
  DatePicker,
  Space,
  Alert,
  Table,
  Row,
  Col,
  Statistic,
  Badge,
  Empty,
  Spin,
  Modal,
  Tag,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';

const ConflictDetector = ({ profileId, tenantId }) => {
  const [form] = Form.useForm();
  const [conflicts, setConflicts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [isAvailable, setIsAvailable] = useState(null);
  const [selectedConflict, setSelectedConflict] = useState(null);
  const [isDetailModalVisible, setIsDetailModalVisible] = useState(false);

  const checkAvailability = async (values) => {
    setLoading(true);
    try {
      const payload = {
        profile_id: profileId,
        start_time: values.startTime.toISOString(),
        end_time: values.endTime.toISOString(),
      };

      const response = await fetch('/api/v1/conflicts/check', {
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

      const data = await response.json();
      setConflicts(data.conflicts || []);
      setIsAvailable(!data.has_conflicts);
    } catch (error) {
      console.error('Failed to check conflicts:', error);
      setIsAvailable(false);
    } finally {
      setLoading(false);
    }
  };

  const conflictColumns = [
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: '100px',
      render: (text) => {
        const colorMap = {
          overlap: 'red',
          blackout: 'orange',
          back_to_back: 'yellow',
        };
        return <Tag color={colorMap[text] || 'blue'}>{text.toUpperCase()}</Tag>;
      },
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      width: '100px',
      render: (text) => {
        const colorMap = {
          critical: 'red',
          high: 'orange',
          medium: 'blue',
          low: 'green',
        };
        return <Tag color={colorMap[text] || 'blue'}>{text.toUpperCase()}</Tag>;
      },
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      render: (text) => <span style={{ fontSize: '12px' }}>{text}</span>,
    },
    {
      title: 'Start Time',
      dataIndex: 'startTime',
      key: 'startTime',
      width: '150px',
      render: (text) => dayjs(text).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'End Time',
      dataIndex: 'endTime',
      key: 'endTime',
      width: '150px',
      render: (text) => dayjs(text).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Duration',
      key: 'duration',
      width: '80px',
      render: (_, record) => {
        if (!record.startTime || !record.endTime) return 'N/A';
        const duration = dayjs(record.endTime).diff(dayjs(record.startTime), 'minute');
        if (duration < 60) return `${duration}m`;
        return `${Math.floor(duration / 60)}h`;
      },
    },
    {
      title: 'Action',
      key: 'action',
      width: '80px',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          onClick={() => {
            setSelectedConflict(record);
            setIsDetailModalVisible(true);
          }}
        >
          Details
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <span>
            <SearchOutlined /> Availability Checker
          </span>
        }
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={checkAvailability}
          autoComplete="off"
        >
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

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              Check Availability
            </Button>
          </Form.Item>
        </Form>

        {isAvailable !== null && (
          <>
            <div style={{ margin: '20px 0' }}>
              {isAvailable ? (
                <Alert
                  message="Time Slot Available"
                  description="✓ This time slot is available for scheduling"
                  type="success"
                  showIcon
                  icon={<CheckCircleOutlined />}
                />
              ) : (
                <Alert
                  message={`Conflicts Found (${conflicts.length})`}
                  description="✗ This time slot has scheduling conflicts"
                  type="error"
                  showIcon
                  icon={<CloseCircleOutlined />}
                />
              )}
            </div>

            {conflicts.length > 0 && (
              <div style={{ marginTop: '20px' }}>
                <h3>Conflict Details</h3>
                <Row gutter={16} style={{ marginBottom: '16px' }}>
                  <Col span={8}>
                    <Statistic
                      title="Total Conflicts"
                      value={conflicts.length}
                      suffix="conflicts"
                      valueStyle={{ color: '#cf1322' }}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="Critical"
                      value={
                        conflicts.filter((c) => c.severity === 'critical').length
                      }
                      valueStyle={{ color: '#cf1322' }}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="High"
                      value={
                        conflicts.filter((c) => c.severity === 'high').length
                      }
                      valueStyle={{ color: '#ff7a45' }}
                    />
                  </Col>
                </Row>

                <Table
                  columns={conflictColumns}
                  dataSource={conflicts}
                  rowKey={(record, index) => index}
                  pagination={{ pageSize: 10 }}
                  size="small"
                />
              </div>
            )}
          </>
        )}

        {/* Conflict Detail Modal */}
        <Modal
          title="Conflict Details"
          visible={isDetailModalVisible}
          onCancel={() => setIsDetailModalVisible(false)}
          footer={null}
          width={600}
        >
          {selectedConflict && (
            <div>
              <Row gutter={16} style={{ marginBottom: '16px' }}>
                <Col span={12}>
                  <strong>Type:</strong>
                  <br />
                  <Tag color="blue">{selectedConflict.type?.toUpperCase()}</Tag>
                </Col>
                <Col span={12}>
                  <strong>Severity:</strong>
                  <br />
                  <Tag
                    color={
                      selectedConflict.severity === 'critical'
                        ? 'red'
                        : selectedConflict.severity === 'high'
                        ? 'orange'
                        : 'blue'
                    }
                  >
                    {selectedConflict.severity?.toUpperCase()}
                  </Tag>
                </Col>
              </Row>

              <div style={{ marginBottom: '16px' }}>
                <strong>Description:</strong>
                <p>{selectedConflict.description}</p>
              </div>

              <div style={{ marginBottom: '16px' }}>
                <strong>Time Range:</strong>
                <p>
                  {dayjs(selectedConflict.startTime).format('YYYY-MM-DD HH:mm:ss')} -{' '}
                  {dayjs(selectedConflict.endTime).format('YYYY-MM-DD HH:mm:ss')}
                </p>
              </div>

              {selectedConflict.affectedIds?.length > 0 && (
                <div>
                  <strong>Affected Events:</strong>
                  <div style={{ marginTop: '8px' }}>
                    {selectedConflict.affectedIds.map((id) => (
                      <Tag key={id}>{id.substring(0, 8)}...</Tag>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </Modal>
      </Card>
    </div>
  );
};

export default ConflictDetector;
