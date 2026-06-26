import React, { useEffect, useState, useCallback } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Table,
  Empty,
  Spin,
  Alert,
  Progress,
  Badge,
  Button,
  Space,
  Divider,
  Tag,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ExclamationCircleOutlined,
  ReloadOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import axios from 'axios';

interface AuthMetrics {
  total_requests: number;
  success_rate: number;
  failure_count: number;
  failed_auth_by_reason: Record<string, number>;
}

interface AuthorizationMetrics {
  total_failures: number;
  failures_by_tenant: Record<string, number>;
  failures_by_reason: Record<string, number>;
}

interface RateLimitMetrics {
  total_exceeded: number;
  exceeded_by_tenant: Record<string, number>;
}

interface AuditMetrics {
  total_logs: number;
  logs_by_type: Record<string, number>;
  logs_by_action: Record<string, number>;
}

interface ComplianceStatus {
  data_residency: boolean;
  audit_completeness: boolean;
  encryption_enabled: boolean;
  last_check: string;
  overall_status: string;
}

interface SecurityEvent {
  timestamp: string;
  type: string;
  severity: string;
  tenant_id?: string;
  user_id?: string;
  details: string;
}

interface DashboardData {
  timestamp: string;
  auth_metrics: AuthMetrics;
  authorization_metrics: AuthorizationMetrics;
  rate_limit_metrics: RateLimitMetrics;
  audit_metrics: AuditMetrics;
  compliance_status: ComplianceStatus;
  recent_security_events: SecurityEvent[];
}

interface SecurityDashboardProps {
  refreshInterval?: number; // milliseconds
  tenantId?: string;
}

export const SecurityDashboard: React.FC<SecurityDashboardProps> = ({ 
  refreshInterval = 60000, 
  tenantId 
}) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DashboardData | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  // Fetch security dashboard data
  const fetchDashboard = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const params = tenantId ? { tenant_id: tenantId } : {};
      const response = await axios.get('/api/security/dashboard', { params });
      
      setData(response.data);
      setLastRefresh(new Date());
    } catch (err: any) {
      setError(err.message || 'Failed to load security dashboard');
      console.error('Dashboard error:', err);
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  // Set up auto-refresh
  useEffect(() => {
    fetchDashboard();
    
    const interval = setInterval(fetchDashboard, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchDashboard, refreshInterval]);

  const handleDownloadReport = async () => {
    try {
      const response = await axios.post(
        '/api/v1/audit/reports',
        {
          format: 'csv',
          start_date: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
          end_date: new Date().toISOString(),
        },
        { 
          responseType: 'blob',
          headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
        }
      );
      
      const url = window.URL.createObjectURL(response.data);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit-report-${new Date().toISOString().split('T')[0]}.csv`;
      a.click();
    } catch (err) {
      console.error('Failed to download report:', err);
    }
  };

  if (loading && !data) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 0' }}>
        <Spin size="large" />
        <p style={{ marginTop: '20px' }}>Loading security dashboard...</p>
      </div>
    );
  }

  if (error && !data) {
    return (
      <Alert
        message="Error Loading Dashboard"
        description={error}
        type="error"
        showIcon
        action={
          <Button size="small" onClick={fetchDashboard}>
            Retry
          </Button>
        }
      />
    );
  }

  if (!data) return <Empty description="No data available" />;

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />;
      case 'high':
        return <WarningOutlined style={{ color: '#ff7a45' }} />;
      case 'medium':
        return <WarningOutlined style={{ color: '#faad14' }} />;
      default:
        return <InfoCircleOutlined style={{ color: '#1890ff' }} />;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'red';
      case 'high':
        return 'volcano';
      case 'medium':
        return 'orange';
      default:
        return 'blue';
    }
  };

  const getEventTypeColor = (type: string) => {
    switch (type) {
      case 'auth_failure':
        return 'red';
      case 'rate_limit':
        return 'orange';
      case 'token_revoked':
        return 'purple';
      case 'compliance_failure':
        return 'magenta';
      default:
        return 'blue';
    }
  };

  const failedAuthReasons = Object.entries(data.auth_metrics.failed_auth_by_reason).map(
    ([reason, count]) => ({
      reason,
      count,
      percentage: data.auth_metrics.failure_count > 0 
        ? Math.round((count / data.auth_metrics.failure_count) * 100)
        : 0,
    })
  );

  const complianceOverall = 
    data.compliance_status.data_residency &&
    data.compliance_status.audit_completeness &&
    data.compliance_status.encryption_enabled;

  return (
    <div style={{ padding: '24px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <h1 style={{ margin: 0 }}>🔐 Security Dashboard</h1>
        <Space>
          <span style={{ fontSize: '12px', color: '#666' }}>
            Last updated: {lastRefresh.toLocaleTimeString()}
          </span>
          <Button icon={<ReloadOutlined />} onClick={fetchDashboard} loading={loading}>
            Refresh
          </Button>
          <Button icon={<DownloadOutlined />} onClick={handleDownloadReport} type="primary">
            Download Report
          </Button>
        </Space>
      </div>

      {error && (
        <Alert
          message="Dashboard Warning"
          description={error}
          type="warning"
          showIcon
          closable
          style={{ marginBottom: '24px' }}
        />
      )}

      {/* Compliance Status Alert */}
      {!complianceOverall && (
        <Alert
          message="Compliance Alert"
          description="Some compliance checks are failing. Immediate attention required."
          type="error"
          showIcon
          style={{ marginBottom: '24px' }}
        />
      )}

      {/* Key Metrics */}
      <Row gutter={16} style={{ marginBottom: '24px' }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Auth Success Rate"
              value={data.auth_metrics.success_rate}
              suffix="%"
              valueStyle={{
                color: data.auth_metrics.success_rate > 95 ? '#52c41a' : '#faad14',
              }}
              prefix={data.auth_metrics.success_rate > 95 ? <CheckCircleOutlined /> : <WarningOutlined />}
            />
            <Progress 
              percent={data.auth_metrics.success_rate} 
              strokeColor={data.auth_metrics.success_rate > 95 ? '#52c41a' : '#faad14'}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Auth Failures"
              value={data.auth_metrics.failure_count}
              valueStyle={{ color: '#ff7a45' }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Rate Limit Events"
              value={data.rate_limit_metrics.total_exceeded}
              valueStyle={{ color: data.rate_limit_metrics.total_exceeded > 10 ? '#ff4d4f' : '#1890ff' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Audit Logs"
              value={data.audit_metrics.total_logs}
              prefix={data.audit_metrics.total_logs > 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* Compliance Status */}
      <Row gutter={16} style={{ marginBottom: '24px' }}>
        <Col xs={24} lg={12}>
          <Card title="Compliance Status" extra={<Badge status={complianceOverall ? 'success' : 'error'} />}>
            <Row gutter={16}>
              <Col xs={12}>
                <div style={{ textAlign: 'center', padding: '10px' }}>
                  {data.compliance_status.data_residency ? (
                    <CheckCircleOutlined style={{ fontSize: '24px', color: '#52c41a' }} />
                  ) : (
                    <CloseCircleOutlined style={{ fontSize: '24px', color: '#ff4d4f' }} />
                  )}
                  <div>Data Residency</div>
                </div>
              </Col>
              <Col xs={12}>
                <div style={{ textAlign: 'center', padding: '10px' }}>
                  {data.compliance_status.audit_completeness ? (
                    <CheckCircleOutlined style={{ fontSize: '24px', color: '#52c41a' }} />
                  ) : (
                    <CloseCircleOutlined style={{ fontSize: '24px', color: '#ff4d4f' }} />
                  )}
                  <div>Audit Completeness</div>
                </div>
              </Col>
              <Col xs={12} style={{ marginTop: '10px' }}>
                <div style={{ textAlign: 'center', padding: '10px' }}>
                  {data.compliance_status.encryption_enabled ? (
                    <CheckCircleOutlined style={{ fontSize: '24px', color: '#52c41a' }} />
                  ) : (
                    <CloseCircleOutlined style={{ fontSize: '24px', color: '#ff4d4f' }} />
                  )}
                  <div>Encryption Enabled</div>
                </div>
              </Col>
              <Col xs={12} style={{ marginTop: '10px' }}>
                <div style={{ textAlign: 'center', padding: '10px' }}>
                  <span style={{ fontSize: '12px', color: '#666' }}>
                    Last check: {new Date(data.compliance_status.last_check).toLocaleString()}
                  </span>
                </div>
              </Col>
            </Row>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="Overall Status">
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: '48px', marginBottom: '10px' }}>
                {complianceOverall ? '✅' : '⚠️'}
              </div>
              <div style={{ fontSize: '24px', fontWeight: 'bold' }}>
                {data.compliance_status.overall_status.toUpperCase()}
              </div>
              <Divider />
              <p style={{ color: '#666' }}>
                All security compliance checks are {complianceOverall ? 'passing' : 'not passing'}
              </p>
            </div>
          </Card>
        </Col>
      </Row>

      {/* Failed Auth Analysis */}
      <Card title="Authentication Failures by Reason" style={{ marginBottom: '24px' }}>
        <Table
          dataSource={failedAuthReasons}
          columns={[
            {
              title: 'Reason',
              dataIndex: 'reason',
              render: (reason) => <Tag color="red">{reason}</Tag>,
            },
            {
              title: 'Count',
              dataIndex: 'count',
              align: 'right' as const,
            },
            {
              title: 'Percentage',
              dataIndex: 'percentage',
              render: (percentage) => (
                <Progress 
                  type="circle" 
                  percent={percentage} 
                  width={40}
                  format={(percent) => `${percent}%`}
                />
              ),
              align: 'right' as const,
            },
          ]}
          pagination={false}
          size="small"
          locale={{ emptyText: 'No auth failures' }}
        />
      </Card>

      {/* Rate Limit by Tenant */}
      {Object.keys(data.rate_limit_metrics.exceeded_by_tenant).length > 0 && (
        <Card title="Rate Limit Violations by Tenant" style={{ marginBottom: '24px' }}>
          <Table
            dataSource={Object.entries(data.rate_limit_metrics.exceeded_by_tenant).map(([tenant, count]) => ({
              tenant,
              count,
            }))}
            columns={[
              {
                title: 'Tenant',
                dataIndex: 'tenant',
                render: (tenant) => <Tag>{tenant}</Tag>,
              },
              {
                title: 'Violations',
                dataIndex: 'count',
                align: 'right' as const,
                render: (count) => (
                  <Badge count={count} style={{ backgroundColor: '#ff4d4f' }} />
                ),
              },
            ]}
            pagination={false}
            size="small"
          />
        </Card>
      )}

      {/* Recent Security Events */}
      <Card title="Recent Security Events" style={{ marginBottom: '24px' }}>
        <Table
          dataSource={data.recent_security_events.map((event, i) => ({ ...event, key: i }))}
          columns={[
            {
              title: 'Time',
              dataIndex: 'timestamp',
              render: (timestamp) => new Date(timestamp).toLocaleString(),
              width: '180px',
            },
            {
              title: 'Type',
              dataIndex: 'type',
              render: (type) => <Tag color={getEventTypeColor(type)}>{type}</Tag>,
              width: '120px',
            },
            {
              title: 'Severity',
              dataIndex: 'severity',
              render: (severity) => (
                <Tag icon={getSeverityIcon(severity)} color={getSeverityColor(severity)}>
                  {severity.toUpperCase()}
                </Tag>
              ),
              width: '100px',
            },
            {
              title: 'Details',
              dataIndex: 'details',
              ellipsis: true,
            },
          ]}
          pagination={{ pageSize: 10 }}
          size="small"
          locale={{ emptyText: 'No recent security events' }}
        />
      </Card>

      {/* Audit Summary */}
      <Row gutter={16}>
        <Col xs={24} lg={12}>
          <Card title="Audit Logs by Type">
            <Table
              dataSource={Object.entries(data.audit_metrics.logs_by_type).map(([type, count]) => ({
                type,
                count,
              }))}
              columns={[
                { title: 'Entity Type', dataIndex: 'type' },
                {
                  title: 'Count',
                  dataIndex: 'count',
                  align: 'right' as const,
                  render: (count) => <strong>{count.toLocaleString()}</strong>,
                },
              ]}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="Audit Logs by Action">
            <Table
              dataSource={Object.entries(data.audit_metrics.logs_by_action).map(([action, count]) => ({
                action,
                count,
              }))}
              columns={[
                { title: 'Action', dataIndex: 'action' },
                {
                  title: 'Count',
                  dataIndex: 'count',
                  align: 'right' as const,
                  render: (count) => <strong>{count.toLocaleString()}</strong>,
                },
              ]}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default SecurityDashboard;
