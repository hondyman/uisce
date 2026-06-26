/**
 * Phase 3.25: Query Planner UI Components
 *
 * Components for displaying query plans, explanations, and planner configuration.
 * Integrates with the Global Query Planner API.
 */

import React, { useEffect, useState } from 'react';
import {
  Modal,
  Card,
  Table,
  Tabs,
  Badge,
  Button,
  Skeleton,
  Alert,
  Collapse,
  Row,
  Col,
  Tooltip,
  Statistic,
  Drawer,
  Timeline,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  ThunderboltOutlined,
  DollarOutlined,
  GlobalOutlined,
} from '@ant-design/icons';

// ============================================================================
// Type Definitions (mirrors Go types)
// ============================================================================

interface QueryRequest {
  tenant_id: string;
  query_type: 'feature' | 'metric' | 'ts' | 'drift' | 'importance' | 'discovery';
  semantic_target: string;
  region_hint?: string;
  consistency_level?: 'strong' | 'eventual' | 'region_preferred';
  priority: 'interactive' | 'batch' | 'background';
  freshness_requirement?: string;
  time_range?: {
    from: string;
    to: string;
  };
}

interface QueryPlan {
  plan_id: string;
  plan_type: 'single_region' | 'multi_region_fanout' | 'global_federated';
  selected_regions: string[];
  engine_routes: EngineRoute[];
  estimated_cost: number;
  estimated_latency_ms: number;
  degradation_strategy: DegradationStrategy;
  explain: string;
}

interface EngineRoute {
  engine_type: string;
  region: string;
  endpoint: string;
  catalog?: string;
  table?: string;
  notes?: string;
}

interface DegradationStrategy {
  mode: 'fail_fast' | 'partial_results' | 'fallback_region' | 'use_cache';
  fallback_regions?: string[];
  max_staleness?: string;
}

interface ExplainPlan {
  plan_id: string;
  summary: SummarySummary;
  routing: ExplanationRouting;
  engines: ExplanationEngine[];
  explain: ExplainDetails;
}

interface SummarySummary {
  plan_type: string;
  regions: string[];
  latency_ms: number;
  cost: number;
  degraded: boolean;
}

interface ExplanationRouting {
  selected_regions: string[];
  fallback_regions: string[];
  consistency: string;
  freshness_requirement: string;
}

interface ExplanationEngine {
  engine_type: string;
  region: string;
  endpoint: string;
  catalog?: string;
  notes: string;
}

interface ExplainDetails {
  decision_text: string;
  region_selection_reason: string;
  engine_selection_reason: string;
  latency_estimate_reason: string;
  cost_estimate_reason: string;
  degradation_strategy_reason: string;
}

interface PlannerDecision {
  plan_id: string;
  created_at: string;
  query_type: string;
  semantic_target: string;
  plan_type: string;
  selected_regions: string[];
  estimated_latency_ms: number;
  actual_latency_ms?: number;
  estimated_cost: number;
  execution_status: 'success' | 'partial_failure' | 'failed' | 'pending';
}

interface FeaturePlannerConfig {
  feature_id: string;
  preferred_regions: string[];
  disallowed_regions: string[];
  default_consistency: string;
  default_freshness: string;
  interactive_latency_budget_ms: number;
  batch_latency_budget_ms: number;
  use_cache_if_stale: boolean;
  max_cache_staleness: string;
}

interface RegionPerformance {
  region: string;
  is_healthy: boolean;
  latency_ms_p50: number;
  latency_ms_p95: number;
  latency_ms_p99: number;
  error_rate: number;
  cache_hit_rate: number;
}

// ============================================================================
// Explain Plan Modal Component
// ============================================================================

interface ExplainPlanModalProps {
  planId: string;
  visible: boolean;
  onClose: () => void;
}

export const ExplainPlanModal: React.FC<ExplainPlanModalProps> = ({
  planId,
  visible,
  onClose,
}) => {
  const [explain, setExplain] = useState<ExplainPlan | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!visible || !planId) return;

    setLoading(true);
    setError(null);

    fetch(`/api/v1/plan/${planId}/explain`)
      .then(res => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then(setExplain)
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, [planId, visible]);

  if (!visible) return null;

  return (
    <Modal
      title="Query Plan Explanation"
      open={visible}
      onCancel={onClose}
      width={900}
      footer={null}
    >
      {loading && <Skeleton active paragraph={{ rows: 8 }} />}

      {error && (
        <Alert
          message="Error Loading Plan"
          description={error}
          type="error"
          showIcon
        />
      )}

      {explain && !loading && (
        <div className="space-y-4">
          {/* Summary Section */}
          <Card>
            <Card.Meta
              title={
                <div className="flex items-center gap-2">
                  <span>Summary</span>
                  {explain.summary.degraded && (
                    <Badge
                      status="error"
                      text="Degraded"
                      style={{ marginLeft: 'auto' }}
                    />
                  )}
                </div>
              }
            />
            <Row gutter={16} style={{ marginTop: 16 }}>
              <Col span={6}>
                <Statistic
                  title="Plan Type"
                  value={explain.summary.plan_type}
                  valueStyle={{ fontSize: 14 }}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="Regions"
                  value={explain.summary.regions.length}
                  suffix={`(${explain.summary.regions.join(', ')})`}
                  valueStyle={{ fontSize: 14 }}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="Latency"
                  value={explain.summary.latency_ms}
                  suffix="ms"
                  prefix={<ThunderboltOutlined />}
                  valueStyle={{ fontSize: 14 }}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="Cost"
                  value={explain.summary.cost}
                  prefix={<DollarOutlined />}
                  valueStyle={{ fontSize: 14 }}
                />
              </Col>
            </Row>
          </Card>

          {/* Region Selection */}
          <Card title="Region Selection">
            <p style={{ marginBottom: 12 }}>
              <strong>Selected Regions:</strong> {explain.routing.selected_regions.join(', ')}
            </p>
            {explain.routing.fallback_regions.length > 0 && (
              <p style={{ marginBottom: 12 }}>
                <strong>Fallback Regions:</strong> {explain.routing.fallback_regions.join(', ')}
              </p>
            )}
            <p style={{ color: '#666' }}>{explain.explain.region_selection_reason}</p>
          </Card>

          {/* Engine Selection */}
          <Card title="Engine Selection">
            {explain.engines.map((engine, idx) => (
              <div key={idx} style={{ marginBottom: 12, paddingBottom: 12, borderBottom: idx < explain.engines.length - 1 ? '1px solid #f0f0f0' : 'none' }}>
                <div>
                  <strong>{engine.engine_type}</strong>
                  <span style={{ marginLeft: 8, color: '#666' }}>
                    {engine.region} → {engine.endpoint}
                  </span>
                </div>
                {engine.catalog && (
                  <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                    Catalog: {engine.catalog}
                  </div>
                )}
                {engine.notes && (
                  <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>
                    {engine.notes}
                  </div>
                )}
              </div>
            ))}
            <Alert
              message={explain.explain.engine_selection_reason}
              type="info"
              style={{ marginTop: 12 }}
              showIcon
            />
          </Card>

          {/* Cost & Latency Estimation */}
          <Card title="Cost & Latency Estimation">
            <Alert
              message={explain.explain.latency_estimate_reason}
              type="info"
              style={{ marginBottom: 12 }}
              showIcon
            />
            <Alert
              message={explain.explain.cost_estimate_reason}
              type="info"
              showIcon
            />
          </Card>

          {/* Degradation Strategy */}
          <Card title="Degradation Strategy">
            <Alert
              message={explain.explain.degradation_strategy_reason}
              type={explain.summary.degraded ? 'error' : 'success'}
              showIcon
            />
          </Card>

          {/* Raw Plan JSON */}
          <Collapse
            items={[
              {
                key: '1',
                label: 'Raw Plan (JSON)',
                children: (
                  <pre
                    style={{
                      backgroundColor: '#f5f5f5',
                      padding: 12,
                      borderRadius: 4,
                      overflow: 'auto',
                      fontSize: 12,
                    }}
                  >
                    {JSON.stringify(explain, null, 2)}
                  </pre>
                ),
              },
            ]}
          />
        </div>
      )}
    </Modal>
  );
};

// ============================================================================
// Query Planner Behavior Panel
// ============================================================================

interface QueryPlannerBehaviorPanelProps {
  featureId: string;
  onPlanCreated?: (plan: QueryPlan) => void;
}

export const QueryPlannerBehaviorPanel: React.FC<QueryPlannerBehaviorPanelProps> = ({
  featureId,
  onPlanCreated,
}) => {
  const [config, setConfig] = useState<FeaturePlannerConfig | null>(null);
  const [plans, setPlans] = useState<PlannerDecision[]>([]);
  const [regionHealth, setRegionHealth] = useState<RegionPerformance[]>([]);

  const [explainModalVisible, setExplainModalVisible] = useState(false);
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);

  const [configLoading, setConfigLoading] = useState(false);
  const [plansLoading, setPlansLoading] = useState(false);

  useEffect(() => {
    loadData();
  }, [featureId]);

  const loadData = () => {
    setConfigLoading(true);
    setPlansLoading(true);

    Promise.all([
      fetch(`/api/v1/planner/config/${featureId}`).then(r => r.json()).catch(() => null),
      fetch(`/api/v1/plan/target/${featureId}?limit=10`).then(r => r.json()).catch(() => []),
      fetch(`/api/v1/planner/region-health`).then(r => r.json()).catch(() => []),
    ]).then(([cfg, pln, health]) => {
      setConfig(cfg);
      setPlans(Array.isArray(pln) ? pln : []);
      setRegionHealth(Array.isArray(health) ? health : []);
    }).finally(() => {
      setConfigLoading(false);
      setPlansLoading(false);
    });
  };

  const handleExplainClick = (planId: string) => {
    setSelectedPlanId(planId);
    setExplainModalVisible(true);
  };

  const planColumns = [
    {
      title: 'Timestamp',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => new Date(text).toLocaleString(),
      width: 180,
    },
    {
      title: 'Plan Type',
      dataIndex: 'plan_type',
      key: 'plan_type',
      render: (text: string) => (
        <Badge
          color={
            text === 'single_region'
              ? 'blue'
              : text === 'multi_region_fanout'
              ? 'green'
              : 'orange'
          }
          text={text}
        />
      ),
    },
    {
      title: 'Regions',
      dataIndex: 'selected_regions',
      key: 'regions',
      render: (regions: string[]) => regions.join(', '),
    },
    {
      title: 'Latency (Est/Actual)',
      key: 'latency',
      render: (_, record: PlannerDecision) => (
        <span>
          {record.estimated_latency_ms.toFixed(1)}ms{' '}
          {record.actual_latency_ms ? `/ ${record.actual_latency_ms.toFixed(1)}ms` : '/ -'}
        </span>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'execution_status',
      key: 'status',
      render: (status: string) => (
        <Badge
          icon={status === 'success' ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
          color={status === 'success' ? 'green' : 'red'}
          text={status}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record: PlannerDecision) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleExplainClick(record.plan_id)}
        >
          Explain
        </Button>
      ),
    },
  ];

  const regionHealthColumns = [
    {
      title: 'Region',
      dataIndex: 'region',
      key: 'region',
    },
    {
      title: 'Health',
      dataIndex: 'is_healthy',
      key: 'health',
      render: (healthy: boolean) => (
        <Badge
          status={healthy ? 'success' : 'error'}
          text={healthy ? 'Healthy' : 'Unhealthy'}
        />
      ),
    },
    {
      title: 'P99 Latency',
      dataIndex: 'latency_ms_p99',
      key: 'p99',
      render: (val: number) => `${val.toFixed(1)}ms`,
    },
    {
      title: 'Error Rate',
      dataIndex: 'error_rate',
      key: 'error_rate',
      render: (val: number) => `${(val * 100).toFixed(2)}%`,
    },
    {
      title: 'Cache Hit Rate',
      dataIndex: 'cache_hit_rate',
      key: 'cache_hit_rate',
      render: (val: number) => `${(val * 100).toFixed(1)}%`,
    },
  ];

  return (
    <Card title="Query Planner Behavior" extra={<Button onClick={loadData}>Refresh</Button>}>
      <Tabs
        items={[
          {
            key: 'config',
            label: 'Configuration',
            children: (
              <Skeleton active loading={configLoading} paragraph={{ rows: 4 }}>
                {config ? (
                  <div>
                    <div style={{ marginBottom: 12 }}>
                      <strong>Preferred Regions:</strong>{' '}
                      {config.preferred_regions?.length > 0
                        ? config.preferred_regions.join(', ')
                        : 'None'}
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <strong>Disallowed Regions:</strong>{' '}
                      {config.disallowed_regions?.length > 0
                        ? config.disallowed_regions.join(', ')
                        : 'None'}
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <strong>Default Consistency:</strong> {config.default_consistency}
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <strong>Default Freshness:</strong> {config.default_freshness}
                    </div>
                    <div style={{ marginBottom: 12 }}>
                      <strong>Interactive Latency Budget:</strong>{' '}
                      {config.interactive_latency_budget_ms}ms
                    </div>
                    <div>
                      <strong>Batch Latency Budget:</strong> {config.batch_latency_budget_ms}ms
                    </div>
                  </div>
                ) : (
                  <Alert
                    message="No configuration found"
                    type="info"
                    showIcon
                  />
                )}
              </Skeleton>
            ),
          },
          {
            key: 'plans',
            label: `Recent Plans (${plans.length})`,
            children: (
              <Skeleton active loading={plansLoading} paragraph={{ rows: 8 }}>
                <Table
                  columns={planColumns}
                  dataSource={plans}
                  rowKey="plan_id"
                  size="small"
                  pagination={{ pageSize: 10 }}
                />
              </Skeleton>
            ),
          },
          {
            key: 'health',
            label: 'Region Health',
            children: (
              <Skeleton active loading={plansLoading} paragraph={{ rows: 4 }}>
                <Table
                  columns={regionHealthColumns}
                  dataSource={regionHealth}
                  rowKey="region"
                  size="small"
                />
              </Skeleton>
            ),
          },
        ]}
      />

      {selectedPlanId && (
        <ExplainPlanModal
          planId={selectedPlanId}
          visible={explainModalVisible}
          onClose={() => setExplainModalVisible(false)}
        />
      )}
    </Card>
  );
};

// ============================================================================
// Plan Creator Component (for testing/debugging)
// ============================================================================

interface PlanCreatorProps {
  featureId: string;
  onPlanCreated?: (plan: QueryPlan) => void;
}

export const PlanCreator: React.FC<PlanCreatorProps> = ({ featureId, onPlanCreated }) => {
  const [visible, setVisible] = useState(false);
  const [priority, setPriority] = useState<'interactive' | 'batch' | 'background'>('interactive');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<QueryPlan | null>(null);

  const handleCreate = async () => {
    setLoading(true);

    try {
      const response = await fetch('/api/v1/plan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          semantic_target: featureId,
          query_type: 'feature',
          priority,
        }),
      });

      if (!response.ok) throw new Error(`HTTP ${response.status}`);

      const plan = await response.json();
      setResult(plan);
      onPlanCreated?.(plan);
    } catch (error) {
      alert(`Error creating plan: ${error}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Button onClick={() => setVisible(true)}>Create Plan</Button>

      <Drawer
        title="Create Query Plan"
        placement="right"
        onClose={() => setVisible(false)}
        open={visible}
      >
        <div style={{ marginBottom: 16 }}>
          <label>
            Priority:
            <select
              value={priority}
              onChange={e => setPriority(e.target.value as any)}
              style={{ marginLeft: 8, padding: 4 }}
            >
              <option value="interactive">Interactive</option>
              <option value="batch">Batch</option>
              <option value="background">Background</option>
            </select>
          </label>
        </div>

        <Button
          type="primary"
          onClick={handleCreate}
          loading={loading}
          style={{ marginBottom: 16 }}
        >
          Generate Plan
        </Button>

        {result && (
          <Card title="Plan Created">
            <div>
              <div><strong>Plan ID:</strong> {result.plan_id}</div>
              <div><strong>Plan Type:</strong> {result.plan_type}</div>
              <div><strong>Regions:</strong> {result.selected_regions.join(', ')}</div>
              <div><strong>Latency:</strong> {result.estimated_latency_ms}ms</div>
              <div><strong>Cost:</strong> {result.estimated_cost}</div>
            </div>
          </Card>
        )}
      </Drawer>
    </>
  );
};

// ============================================================================
// Region Performance Chart Component
// ============================================================================

interface RegionPerformanceChartProps {
  featureId?: string;
}

export const RegionPerformanceChart: React.FC<RegionPerformanceChartProps> = () => {
  const [data, setData] = useState<RegionPerformance[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/planner/region-health')
      .then(r => r.json())
      .then(setData)
      .catch(err => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  return (
    <Card title="Region Performance" loading={loading}>
      <Table
        columns={[
          {
            title: 'Region',
            dataIndex: 'region',
            key: 'region',
          },
          {
            title: 'Health',
            dataIndex: 'is_healthy',
            key: 'healthy',
            render: (v: boolean) => (
              <Badge status={v ? 'success' : 'error'} text={v ? 'Healthy' : 'Unhealthy'} />
            ),
          },
          {
            title: 'P50 Latency',
            dataIndex: 'latency_ms_p50',
            key: 'p50',
            render: (v: number) => `${v.toFixed(1)}ms`,
          },
          {
            title: 'P99 Latency',
            dataIndex: 'latency_ms_p99',
            key: 'p99',
            render: (v: number) => `${v.toFixed(1)}ms`,
          },
        ]}
        dataSource={data}
        rowKey="region"
        pagination={false}
      />
    </Card>
  );
};

// ============================================================================
// Export All Components
// ============================================================================

export default {
  ExplainPlanModal,
  QueryPlannerBehaviorPanel,
  PlanCreator,
  RegionPerformanceChart,
};
