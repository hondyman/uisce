import React, { useState } from 'react';
import { Table, Tag, Space, Modal, Spin, Empty, Statistic, Row, Col, Button } from 'antd';
import { EyeOutlined, CheckCircleOutlined, CloseCircleOutlined, WarningOutlined } from '@ant-design/icons';
import { gql, useQuery } from '@apollo/client';

// GraphQL Query
const GET_SYNC_LOGS = gql`
  query GetSyncLogs($configId: UUID!, $limit: Int!, $offset: Int!) {
    sync_logs(where: { config_id: { _eq: $configId } }, limit: $limit, offset: $offset, order_by: { executed_at: desc }) {
      id
      config_id
      status
      holidays_added
      holidays_updated
      error_message
      execution_time_ms
      executed_at
    }
    sync_logs_aggregate(where: { config_id: { _eq: $configId } }) {
      aggregate {
        count
      }
    }
  }
`;

const GET_LAST_SYNC_LOG = gql`
  query GetLastSyncLog($configId: UUID!) {
    sync_logs(where: { config_id: { _eq: $configId } }, limit: 1, order_by: { executed_at: desc }) {
      id
      status
      holidays_added
      holidays_updated
      error_message
      execution_time_ms
      executed_at
    }
  }
`;

interface SyncLog {
  id: string;
  config_id: string;
  status: 'success' | 'failed' | 'partial';
  holidays_added: number;
  holidays_updated: number;
  error_message?: string;
  execution_time_ms: number;
  executed_at: string;
}

interface ExternalSyncLogsProps {
  configId: string;
  tenantId: string;
}

const ExternalSyncLogs: React.FC<ExternalSyncLogsProps> = ({ configId, tenantId }) => {
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<SyncLog | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });

  // Query logs
  const { data, loading, refetch } = useQuery(GET_SYNC_LOGS, {
    variables: {
      configId,
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    },
    skip: !configId,
    pollInterval: 30000, // Auto-refresh every 30 seconds
  });

  // Query last sync for summary
  const { data: lastSyncData, loading: lastSyncLoading } = useQuery(GET_LAST_SYNC_LOG, {
    variables: { configId },
    skip: !configId,
  });

  const logs = data?.sync_logs || [];
  const total = data?.sync_logs_aggregate?.aggregate?.count || 0;
  const lastSync = lastSyncData?.sync_logs?.[0];

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case 'failed':
        return <CloseCircleOutlined style={{ color: '#f5222d' }} />;
      case 'partial':
        return <WarningOutlined style={{ color: '#faad14' }} />;
      default:
        return null;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success':
        return 'green';
      case 'failed':
        return 'red';
      case 'partial':
        return 'orange';
      default:
        return 'default';
    }
  };

  const columns = [
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Space>
          {getStatusIcon(status)}
          <Tag color={getStatusColor(status)}>
            {status.charAt(0).toUpperCase() + status.slice(1)}
          </Tag>
        </Space>
      ),
    },
    {
      title: 'Executed',
      dataIndex: 'executed_at',
      key: 'executed_at',
      width: 180,
      render: (date: string) => new Date(date).toLocaleString(),
      sorter: (a: SyncLog, b: SyncLog) => 
        new Date(b.executed_at).getTime() - new Date(a.executed_at).getTime(),
    },
    {
      title: 'Holidays Added',
      dataIndex: 'holidays_added',
      key: 'holidays_added',
      width: 120,
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: 'Holidays Updated',
      dataIndex: 'holidays_updated',
      key: 'holidays_updated',
      width: 140,
      render: (count: number) => <Tag color="cyan">{count}</Tag>,
    },
    {
      title: 'Duration',
      dataIndex: 'execution_time_ms',
      key: 'execution_time_ms',
      width: 100,
      render: (ms: number) => `${ms}ms`,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 80,
      render: (_: any, log: SyncLog) => (
        <Button
          type="text"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => {
            setSelectedLog(log);
            setDetailVisible(true);
          }}
        >
          View
        </Button>
      ),
    },
  ];

  const handleTableChange = (newPagination: any) => {
    setPagination(newPagination);
  };

  return (
    <div>
      {/* Summary Cards */}
      {!lastSyncLoading && lastSync && (
        <Row gutter={16} style={{ marginBottom: '24px' }}>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="Last Status"
              value={lastSync.status}
              prefix={getStatusIcon(lastSync.status)}
              valueStyle={{ color: getStatusColor(lastSync.status) === 'green' ? '#52c41a' : '#f5222d' }}
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="Holidays Added"
              value={lastSync.holidays_added}
              suffix="items"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="Execution Time"
              value={lastSync.execution_time_ms}
              suffix="ms"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Statistic
              title="Last Sync"
              value={new Date(lastSync.executed_at).toLocaleDateString()}
            />
          </Col>
        </Row>
      )}

      {/* Logs Table */}
      <Spin spinning={loading}>
        {logs.length === 0 ? (
          <Empty description="No sync logs found" />
        ) : (
          <Table
            columns={columns}
            dataSource={logs}
            rowKey="id"
            pagination={{
              current: pagination.current,
              pageSize: pagination.pageSize,
              total: total,
              showSizeChanger: true,
              showTotal: (total) => `Total ${total} syncs`,
            }}
            onChange={handleTableChange}
            scroll={{ x: 1000 }}
          />
        )}
      </Spin>

      {/* Detail Modal */}
      {selectedLog && (
        <Modal
          title="Sync Log Details"
          open={detailVisible}
          onCancel={() => setDetailVisible(false)}
          footer={[
            <Button key="close" onClick={() => setDetailVisible(false)}>
              Close
            </Button>,
          ]}
          width={700}
        >
          <div style={{ lineHeight: '2.5' }}>
            <div>
              <strong>Status:</strong>{' '}
              <Space>
                {getStatusIcon(selectedLog.status)}
                <Tag color={getStatusColor(selectedLog.status)}>
                  {selectedLog.status.charAt(0).toUpperCase() + selectedLog.status.slice(1)}
                </Tag>
              </Space>
            </div>

            <div>
              <strong>Executed At:</strong> {new Date(selectedLog.executed_at).toLocaleString()}
            </div>

            <div>
              <strong>Holidays Added:</strong>{' '}
              <Tag color="blue">{selectedLog.holidays_added}</Tag>
            </div>

            <div>
              <strong>Holidays Updated:</strong>{' '}
              <Tag color="cyan">{selectedLog.holidays_updated}</Tag>
            </div>

            <div>
              <strong>Execution Time:</strong> {selectedLog.execution_time_ms}ms
            </div>

            {selectedLog.error_message && (
              <div style={{ marginTop: '16px' }}>
                <strong>Error Message:</strong>
                <div
                  style={{
                    padding: '12px',
                    backgroundColor: '#fff1f0',
                    border: '1px solid #ffccc7',
                    borderRadius: '4px',
                    marginTop: '8px',
                    fontFamily: 'monospace',
                    fontSize: '12px',
                  }}
                >
                  {selectedLog.error_message}
                </div>
              </div>
            )}

            {selectedLog.status === 'success' && !selectedLog.error_message && (
              <div style={{
                marginTop: '16px',
                padding: '12px',
                backgroundColor: '#f6ffed',
                border: '1px solid #b7eb8f',
                borderRadius: '4px',
              }}>
                <CheckCircleOutlined style={{ color: '#52c41a', marginRight: '8px' }} />
                Sync completed successfully
              </div>
            )}
          </div>
        </Modal>
      )}
    </div>
  );
};

export default ExternalSyncLogs;
