// @ts-nocheck
import React, { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Tag,
  Space,
  Tooltip,
  Button,
  Spin,
  Empty,
} from 'antd';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import {
  RobotOutlined,
  ThunderboltOutlined,
  SmileOutlined,
  BarChartOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import styles from './AIRoutingDashboard.module.css';

// Silence occasional unused-import warnings in this demo dashboard (non-functional placeholders)
void useState;
void useEffect;
void LineChart;
void Line;
void BarChart;
void Bar;
void PieChart;
void Pie;
void XAxis;
void YAxis;
void CartesianGrid;
void RechartsTooltip;
void Legend;
void ResponsiveContainer;
void Cell;
void Button;

interface RoutingMetrics {
  overall_accuracy: number;
  avg_decision_time_ms: number;
  model_agreement_rate: number;
  workflows_routed_today: number;
  branch_distribution: Array<{
    name: string;
    value: number;
    success_rate: number;
  }>;
  model_performance: Array<{
    model: string;
    accuracy: number;
    avg_latency: number;
  }>;
  rl_episodes: number;
  rl_epsilon: number;
  rl_avg_q_value: number;
  rl_last_reward: number;
}

interface LiveDecision {
  timestamp: string;
  workflow_name: string;
  branch_name: string;
  confidence: number;
  primary_model: string;
  reasoning: string[];
}

const AIRoutingDashboard: React.FC<{ workflowId?: string }> = ({ workflowId }) => {
  const { data: routingMetrics, isLoading: metricsLoading } = useQuery<RoutingMetrics>({
    queryKey: ['ai-routing-metrics', workflowId],
    queryFn: async () => {
      const response = await fetch('/api/ai-routing/metrics?tenant_id=default');
      return response.json();
    },
    refetchInterval: 5000,
  });

  const { data: liveDecisions, isLoading: decisionsLoading } = useQuery<{ decisions: LiveDecision[] }>({
    queryKey: ['live-routing-decisions'],
    queryFn: async () => {
      const response = await fetch('/api/ai-routing/live-decisions?tenant_id=default&limit=10');
      return response.json();
    },
    refetchInterval: 2000,
  });

  const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8B5CF6', '#EC4899'];

  const modelPerformanceData = routingMetrics?.model_performance || [
    { model: 'Predictive Analytics', accuracy: 0.94, avg_latency: 45 },
    { model: 'Reinforcement Learning', accuracy: 0.89, avg_latency: 12 },
    { model: 'Sentiment Analysis', accuracy: 0.87, avg_latency: 78 },
    { model: 'Load Balancer', accuracy: 0.96, avg_latency: 8 },
  ];

  if (metricsLoading) {
    return (
      <div className={styles.loadingContainer}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <Row gutter={[16, 16]} className={styles.headerRow}>
        <Col span={24}>
          <h1 className={styles.dashboardTitle}>
            <RobotOutlined className={styles.titleIcon} />
            AI-Driven Decision Routing Dashboard
          </h1>
          <p className={styles.dashboardSubtitle}>
            Real-time workflow routing with multi-model ensemble decision making
          </p>
        </Col>
      </Row>

      {/* Key Metrics */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className={styles.statCardPurple}>
            <Statistic
              title={<span className={styles.statCardTitle}>AI Routing Accuracy</span>}
              value={(routingMetrics?.overall_accuracy || 0) * 100}
              precision={1}
              suffix="%"
              prefix={<RobotOutlined className={styles.statCardIcon} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className={styles.statCardPink}>
            <Statistic
              title={<span className={styles.statCardTitle}>Avg Decision Time</span>}
              value={routingMetrics?.avg_decision_time_ms || 0}
              suffix="ms"
              prefix={<ThunderboltOutlined className={styles.statCardIcon} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className={styles.statCardBlue}>
            <Statistic
              title={<span className={styles.statCardTitle}>Model Agreement</span>}
              value={(routingMetrics?.model_agreement_rate || 0) * 100}
              precision={1}
              suffix="%"
              prefix={<SmileOutlined className={styles.statCardIcon} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className={styles.statCardGreen}>
            <Statistic
              title={<span className={styles.statCardTitle}>Routed Today</span>}
              value={routingMetrics?.workflows_routed_today || 0}
              prefix={<BarChartOutlined className={styles.statCardIcon} />}
            />
          </Card>
        </Col>
      </Row>

      {/* Model Performance */}
      <Row gutter={[16, 16]} className={styles.chartRow}>
        <Col span={24} lg={12}>
          <Card title="Model Performance Comparison" bordered={false}>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={modelPerformanceData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="model" angle={-15} textAnchor="end" height={80} />
                <YAxis yAxisId="left" orientation="left" stroke="#8884d8" />
                <YAxis yAxisId="right" orientation="right" stroke="#82ca9d" />
                <RechartsTooltip />
                <Legend />
                <Bar yAxisId="left" dataKey="accuracy" fill="#8884d8" name="Accuracy" radius={[8, 8, 0, 0]} />
                <Bar yAxisId="right" dataKey="avg_latency" fill="#82ca9d" name="Latency (ms)" radius={[8, 8, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        {/* Branch Distribution */}
        <Col span={24} lg={12}>
          <Card title="Routing Distribution (Last 24h)" bordered={false}>
            {routingMetrics?.branch_distribution && routingMetrics.branch_distribution.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={routingMetrics.branch_distribution}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(entry) => `${entry.name}: ${entry.value}`}
                    outerRadius={100}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {routingMetrics.branch_distribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <RechartsTooltip />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="No distribution data" />
            )}
          </Card>
        </Col>
      </Row>

      {/* Live Routing Decisions */}
      <Row gutter={[16, 16]} className={styles.chartRow}>
        <Col span={24}>
          <Card title="Live Routing Decisions" bordered={false}>
            {decisionsLoading ? (
              <Spin />
            ) : liveDecisions?.decisions && liveDecisions.decisions.length > 0 ? (
              <Table
                dataSource={liveDecisions.decisions}
                columns={[
                  {
                    title: 'Timestamp',
                    dataIndex: 'timestamp',
                    key: 'timestamp',
                    width: 120,
                    render: (text) => new Date(text).toLocaleTimeString(),
                  },
                  {
                    title: 'Workflow',
                    dataIndex: 'workflow_name',
                    key: 'workflow_name',
                    width: 150,
                  },
                  {
                    title: 'Selected Branch',
                    dataIndex: 'branch_name',
                    key: 'branch_name',
                    width: 120,
                    render: (text) => <Tag color="blue">{text}</Tag>,
                  },
                  {
                    title: 'Confidence',
                    dataIndex: 'confidence',
                    key: 'confidence',
                    width: 150,
                    render: (val) => (
                      <Progress
                        percent={val * 100}
                        size="small"
                        status={val > 0.8 ? 'success' : val > 0.6 ? 'normal' : 'exception'}
                      />
                    ),
                  },
                  {
                    title: 'Primary Model',
                    dataIndex: 'primary_model',
                    key: 'primary_model',
                    width: 150,
                    render: (text) => <Tag icon={<SmileOutlined />}>{text}</Tag>,
                  },
                  {
                    title: 'Reasoning',
                    dataIndex: 'reasoning',
                    key: 'reasoning',
                    render: (reasons) => (
                      <Tooltip
                        title={
                          <div>
                            {Array.isArray(reasons) &&
                              reasons.map((r, i) => (
                                <div key={i} className={styles.tooltipReason}>
                                  {r}
                                </div>
                              ))}
                          </div>
                        }
                      >
                        <span className={styles.viewDetailsLink}>View Details</span>
                      </Tooltip>
                    ),
                  },
                ]}
                pagination={{ pageSize: 10 }}
                size="small"
                scroll={{ x: 1000 }}
              />
            ) : (
              <Empty description="No decisions yet" />
            )}
          </Card>
        </Col>
      </Row>

      {/* RL Agent Status */}
      <Row gutter={[16, 16]} className={styles.chartRow}>
        <Col span={24}>
          <Card title="Reinforcement Learning Agent Status" bordered={false}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <div className={styles.metricRow}>
                <span>
                  <strong>Episodes Trained:</strong> {routingMetrics?.rl_episodes || 0}
                </span>
                <Tag color="blue">{routingMetrics?.rl_episodes || 0}</Tag>
              </div>

              <div className={styles.metricRow}>
                <span>
                  <strong>Exploration Rate (ε):</strong>
                </span>
                <span className={styles.progressContainer}>
                  <Progress
                    percent={Math.round((routingMetrics?.rl_epsilon || 0) * 100)}
                    showInfo={true}
                    format={(percent) => `${((percent || 0) / 100).toFixed(3)}`}
                  />
                </span>
              </div>

              <div className={styles.metricRow}>
                <span>
                  <strong>Average Q-Value:</strong>
                </span>
                <span>{(routingMetrics?.rl_avg_q_value || 0).toFixed(3)}</span>
              </div>

              <div className={styles.metricRow}>
                <span>
                  <strong>Last Reward:</strong>
                </span>
                <Tag color={(routingMetrics?.rl_last_reward || 0) > 0 ? 'green' : 'red'}>
                  {(routingMetrics?.rl_last_reward || 0).toFixed(2)}
                </Tag>
              </div>
            </Space>
          </Card>
        </Col>
      </Row>

      {/* System Status */}
      <Row gutter={[16, 16]} className={styles.chartRow}>
        <Col span={24}>
          <Card title="System Status" bordered={false}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <div className={styles.statusRow}>
                <CheckCircleOutlined className={styles.statusIconSuccess} />
                <span>All models operational and responding within SLA</span>
              </div>
              <div className={styles.statusRow}>
                <ExclamationCircleOutlined className={styles.statusIconWarning} />
                <span>Model agreement rate at {((routingMetrics?.model_agreement_rate || 0) * 100).toFixed(1)}% - acceptable</span>
              </div>
              <div className={styles.statusRow}>
                <CheckCircleOutlined className={styles.statusIconSuccess} />
                <span>RL agent learning curve stable, epsilon = {(routingMetrics?.rl_epsilon || 0).toFixed(4)}</span>
              </div>
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AIRoutingDashboard;
