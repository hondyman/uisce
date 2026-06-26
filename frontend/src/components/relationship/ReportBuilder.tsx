// @ts-nocheck
import React, { useState } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Tabs,
  Table,
  message,
  Space,
  Empty,
  Badge,
  Modal,
  Row,
  Col,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  CopyOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import './ReportBuilder.module.css';

interface Metric {
  field: string;
  aggregation: 'SUM' | 'AVG' | 'COUNT' | 'MIN' | 'MAX';
  alias: string;
}

interface Filter {
  field: string;
  operator: '=' | '>' | '<' | 'LIKE' | 'IN';
  value: string;
}

interface ReportQueryConfig {
  baseEntityId: string;
  baseEntityName?: string;
  relatedEntities: string[];
  metrics: Metric[];
  dimensions: string[];
  filters: Filter[];
}

interface ReportBuilderProps {
  tenantId: string;
  datasourceId: string;
  entities: Array<{ id: string; name: string }>;
  onExecuteReport: (config: ReportQueryConfig) => Promise<void>;
}

const ReportBuilder: React.FC<ReportBuilderProps> = ({
  tenantId,
  datasourceId,
  entities,
}) => {
  const [config, setConfig] = useState<ReportQueryConfig>({
    baseEntityId: '',
    relatedEntities: [],
    metrics: [],
    dimensions: [],
    filters: [],
  });

  const [executing, setExecuting] = useState(false);
  const [generatedSQL, setGeneratedSQL] = useState<string>('');
  const [sqlVisible, setSqlVisible] = useState(false);
  const [results, setResults] = useState<any[]>([]);

  // Generate report query
  const handleGenerateSQL = async () => {
    if (!config.baseEntityId) {
      message.error('Please select a base entity');
      return;
    }

    try {
      const response = await fetch('/api/reports/generate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify(config),
      });

      if (!response.ok) {
        throw new Error(`Failed to generate query: ${response.statusText}`);
      }

      const data = await response.json();
      setGeneratedSQL(data.query);
      setSqlVisible(true);
      message.success('Query generated successfully');
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to generate query';
      message.error(errorMsg);
    }
  };

  // Execute report
  const handleExecuteReport = async () => {
    if (!config.baseEntityId) {
      message.error('Please select a base entity');
      return;
    }

    setExecuting(true);

    try {
      const response = await fetch('/api/reports/preview', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          ...config,
          limit: 100,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to execute report: ${response.statusText}`);
      }

      const data = await response.json();
      setGeneratedSQL(data.query);
      setResults(data.results || []);
      message.success(`Report executed: ${data.results?.length || 0} rows`);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to execute report';
      message.error(errorMsg);
    } finally {
      setExecuting(false);
    }
  };

  // Add metric
  const handleAddMetric = () => {
    const newMetric: Metric = {
      field: '',
      aggregation: 'SUM',
      alias: '',
    };
    setConfig({
      ...config,
      metrics: [...config.metrics, newMetric],
    });
  };

  // Remove metric
  const handleRemoveMetric = (index: number) => {
    setConfig({
      ...config,
      metrics: config.metrics.filter((_, i) => i !== index),
    });
  };

  // Add filter
  const handleAddFilter = () => {
    const newFilter: Filter = {
      field: '',
      operator: '=',
      value: '',
    };
    setConfig({
      ...config,
      filters: [...config.filters, newFilter],
    });
  };

  // Remove filter
  const handleRemoveFilter = (index: number) => {
    setConfig({
      ...config,
      filters: config.filters.filter((_, i) => i !== index),
    });
  };

  return (
    <div className="report-builder">
      <Card className="report-builder-card" title="Self-Service Report Builder">
        <Form layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="Base Entity" required>
                <Select
                  placeholder="Select base entity"
                  value={config.baseEntityId || undefined}
                  onChange={(value) => {
                    const entityName = entities.find((e) => e.id === value)?.name || '';
                    setConfig({
                      ...config,
                      baseEntityId: value,
                      baseEntityName: entityName,
                    });
                  }}
                  options={entities.map((e) => ({
                    label: e.name,
                    value: e.id,
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="Related Entities">
                <Select
                  mode="multiple"
                  placeholder="Select related entities"
                  value={config.relatedEntities}
                  onChange={(value) => {
                    setConfig({
                      ...config,
                      relatedEntities: value,
                    });
                  }}
                  options={entities
                    .filter((e) => e.id !== config.baseEntityId)
                    .map((e) => ({
                      label: e.name,
                      value: e.id,
                    }))}
                />
              </Form.Item>
            </Col>
          </Row>

          <Tabs
            defaultActiveKey="metrics"
            items={[
              {
                key: 'metrics',
                label: (
                  <span>
                    Metrics
                    <Badge
                      count={config.metrics.length}
                      style={{ backgroundColor: '#52c41a', marginLeft: '8px' }}
                    />
                  </span>
                ),
                children: (
                  <div className="config-section">
                    {config.metrics.length === 0 ? (
                      <Empty description="No metrics added" />
                    ) : (
                      <div className="config-list">
                        {config.metrics.map((metric, index) => (
                          <div key={index} className="config-item">
                            <div className="config-item-content">
                              <Select
                                placeholder="Field"
                                style={{ width: '200px' }}
                                value={metric.field || undefined}
                                onChange={(value) => {
                                  const updated = [...config.metrics];
                                  updated[index].field = value;
                                  setConfig({
                                    ...config,
                                    metrics: updated,
                                  });
                                }}
                              />
                              <Select
                                placeholder="Aggregation"
                                style={{ width: '150px' }}
                                value={metric.aggregation}
                                onChange={(value) => {
                                  const updated = [...config.metrics];
                                  updated[index].aggregation = value;
                                  setConfig({
                                    ...config,
                                    metrics: updated,
                                  });
                                }}
                                options={[
                                  { label: 'SUM', value: 'SUM' },
                                  { label: 'AVG', value: 'AVG' },
                                  { label: 'COUNT', value: 'COUNT' },
                                  { label: 'MIN', value: 'MIN' },
                                  { label: 'MAX', value: 'MAX' },
                                ]}
                              />
                              <Input
                                placeholder="Alias"
                                value={metric.alias}
                                onChange={(e) => {
                                  const updated = [...config.metrics];
                                  updated[index].alias = e.target.value;
                                  setConfig({
                                    ...config,
                                    metrics: updated,
                                  });
                                }}
                              />
                            </div>
                            <Button
                              danger
                              icon={<DeleteOutlined />}
                              onClick={() => handleRemoveMetric(index)}
                            />
                          </div>
                        ))}
                      </div>
                    )}
                    <Button
                      icon={<PlusOutlined />}
                      onClick={handleAddMetric}
                      block
                      className="add-button"
                    >
                      Add Metric
                    </Button>
                  </div>
                ),
              },
              {
                key: 'dimensions',
                label: (
                  <span>
                    Dimensions
                    <Badge
                      count={config.dimensions.length}
                      style={{ backgroundColor: '#1890ff', marginLeft: '8px' }}
                    />
                  </span>
                ),
                children: (
                  <div className="config-section">
                    <Select
                      mode="multiple"
                      placeholder="Select dimension fields"
                      value={config.dimensions}
                      onChange={(value) => {
                        setConfig({
                          ...config,
                          dimensions: value,
                        });
                      }}
                      style={{ width: '100%' }}
                    />
                  </div>
                ),
              },
              {
                key: 'filters',
                label: (
                  <span>
                    Filters
                    <Badge
                      count={config.filters.length}
                      style={{ backgroundColor: '#faad14', marginLeft: '8px' }}
                    />
                  </span>
                ),
                children: (
                  <div className="config-section">
                    {config.filters.length === 0 ? (
                      <Empty description="No filters added" />
                    ) : (
                      <div className="config-list">
                        {config.filters.map((filter, index) => (
                          <div key={index} className="config-item">
                            <div className="config-item-content">
                              <Input placeholder="Field" value={filter.field} disabled />
                              <Select
                                placeholder="Operator"
                                style={{ width: '100px' }}
                                value={filter.operator}
                                onChange={(value) => {
                                  const updated = [...config.filters];
                                  updated[index].operator = value;
                                  setConfig({
                                    ...config,
                                    filters: updated,
                                  });
                                }}
                                options={[
                                  { label: '=', value: '=' },
                                  { label: '>', value: '>' },
                                  { label: '<', value: '<' },
                                  { label: 'LIKE', value: 'LIKE' },
                                  { label: 'IN', value: 'IN' },
                                ]}
                              />
                              <Input
                                placeholder="Value"
                                value={filter.value}
                                onChange={(e) => {
                                  const updated = [...config.filters];
                                  updated[index].value = e.target.value;
                                  setConfig({
                                    ...config,
                                    filters: updated,
                                  });
                                }}
                              />
                            </div>
                            <Button
                              danger
                              icon={<DeleteOutlined />}
                              onClick={() => handleRemoveFilter(index)}
                            />
                          </div>
                        ))}
                      </div>
                    )}
                    <Button
                      icon={<PlusOutlined />}
                      onClick={handleAddFilter}
                      block
                      className="add-button"
                    >
                      Add Filter
                    </Button>
                  </div>
                ),
              },
            ]}
          />

          <div className="report-actions">
            <Space>
              <Button onClick={handleGenerateSQL} icon={<CopyOutlined />}>
                Generate SQL
              </Button>
              <Button
                type="primary"
                onClick={handleExecuteReport}
                loading={executing}
                icon={<PlayCircleOutlined />}
              >
                Execute Report
              </Button>
            </Space>
          </div>
        </Form>
      </Card>

      <Modal
        title="Generated SQL"
        visible={sqlVisible}
        onCancel={() => setSqlVisible(false)}
        footer={[
          <Button key="close" onClick={() => setSqlVisible(false)}>
            Close
          </Button>,
          <Button
            key="copy"
            type="primary"
            icon={<CopyOutlined />}
            onClick={() => {
              navigator.clipboard.writeText(generatedSQL);
              message.success('SQL copied to clipboard');
            }}
          >
            Copy SQL
          </Button>,
        ]}
        width={900}
      >
        <pre className="sql-code">{generatedSQL}</pre>
      </Modal>

      {results.length > 0 && (
        <Card
          className="results-card"
          title="Report Results"
          extra={
            <Button
              icon={<DownloadOutlined />}
              onClick={() => {
                // Export to CSV
                message.info('Export functionality coming soon');
              }}
            >
              Export
            </Button>
          }
        >
          <Table
            columns={
              results.length > 0
                ? Object.keys(results[0]).map((key) => ({
                    title: key,
                    dataIndex: key,
                    key,
                    render: (value) => {
                      if (value === null || value === undefined) {
                        return <span className="null-value">NULL</span>;
                      }
                      return value;
                    },
                  }))
                : []
            }
            dataSource={results.map((row, idx) => ({
              ...row,
              key: idx,
            }))}
            size="small"
            pagination={{
              pageSize: 20,
              showTotal: (total) => `Total ${total} rows`,
            }}
          />
        </Card>
      )}
    </div>
  );
};

export default ReportBuilder;
