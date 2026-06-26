import { useEffect, useRef, useMemo } from 'react';
import { BarChart3, LineChart, PieChart, Table, TrendingUp, Users, DollarSign, Calendar } from 'lucide-react';
import './DashboardPreview.css';

// (intentionally empty) placeholder removed

interface DashboardVisual {
  id: string;
  type: string;
  title: string;
  description: string;
  querySpec: {
    metrics: string[];
    dimensions: string[];
    sql: string;
  };
  config: {
    chartType: string;
    xAxis?: string;
    yAxis?: string;
    colorBy?: string;
    showLegend: boolean;
    showGrid: boolean;
  };
  compliance: {
    isCompliant: boolean;
    riskLevel: string;
    violations: Array<{
      policyId: string;
      severity: string;
      message: string;
      suggestion?: string;
    }>;
  };
  position: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
}

interface DashboardLayout {
  type: string;
  columns: number;
  rowHeight: number;
}

interface DashboardPreviewProps {
  visuals: DashboardVisual[];
  layout: DashboardLayout;
  className?: string;
}

export const DashboardPreview: React.FC<DashboardPreviewProps> = ({
  visuals,
  layout,
  className = ''
}) => {
  const gridRef = useRef<HTMLDivElement | null>(null);
  const instanceId = useMemo(() => `dashboard-preview-${Math.random().toString(36).slice(2,9)}`, []);
  const getChartIcon = (type: string) => {
    switch (type) {
      case 'line':
        return <LineChart className="chart-icon chart-icon-blue" />;
      case 'bar':
        return <BarChart3 className="chart-icon chart-icon-green" />;
      case 'pie':
        return <PieChart className="chart-icon chart-icon-purple" />;
      case 'table':
        return <Table className="chart-icon chart-icon-orange" />;
      default:
        return <BarChart3 className="chart-icon chart-icon-gray" />;
    }
  };

  const getMetricIcon = (metric: string) => {
    const lowerMetric = metric.toLowerCase();
    if (lowerMetric.includes('revenue') || lowerMetric.includes('sales') || lowerMetric.includes('value')) {
      return <DollarSign className="metric-icon metric-icon-green" />;
    }
    if (lowerMetric.includes('users') || lowerMetric.includes('customers') || lowerMetric.includes('count')) {
      return <Users className="metric-icon metric-icon-blue" />;
    }
    if (lowerMetric.includes('growth') || lowerMetric.includes('trend')) {
      return <TrendingUp className="metric-icon metric-icon-purple" />;
    }
    return <BarChart3 className="metric-icon metric-icon-gray" />;
  };

  const renderMockChart = (visual: DashboardVisual) => {
    const { type, config: _config, querySpec: _querySpec } = visual;

    // Mock data for preview
    const mockData = [
      { label: 'Jan', value: 120 },
      { label: 'Feb', value: 150 },
      { label: 'Mar', value: 180 },
      { label: 'Apr', value: 140 },
      { label: 'May', value: 200 },
      { label: 'Jun', value: 170 },
    ];

    switch (type) {
      case 'line':
        return (
          <div className="chart-container">
            {mockData.map((point, index) => (
              <div key={index} className="chart-bar-container">
                <div
                  className="chart-bar"
                  data-height={`${(point.value / 200) * 100}%`}
                />
                <span className="chart-bar-label">{point.label}</span>
              </div>
            ))}
          </div>
        );

      case 'bar':
        return (
          <div className="chart-container">
            {mockData.map((point, index) => (
              <div key={index} className="chart-bar-container">
                <div
                  className="chart-bar-green"
                  data-height={`${(point.value / 200) * 100}%`}
                />
                <span className="chart-bar-label">{point.label}</span>
              </div>
            ))}
          </div>
        );

      case 'pie':
        return (
          <div className="visual-content">
            <div className="chart-pie-container">
              <div className="chart-pie-gradient"></div>
              <div className="chart-pie-center"></div>
              <div className="chart-pie-overlay">
                <PieChart className="chart-pie-icon" />
              </div>
            </div>
          </div>
        );

      case 'table':
        return (
          <div className="table-container">
            <table className="table-element">
              <thead className="table-header">
                <tr>
                  <th>Month</th>
                  <th>Value</th>
                </tr>
              </thead>
              <tbody className="table-body">
                {mockData.slice(0, 4).map((row, index) => (
                  <tr key={index}>
                    <td>{row.label}</td>
                    <td>{row.value}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );

      default:
        return (
          <div className="visual-content">
            <BarChart3 className="default-chart-icon" />
          </div>
        );
    }
  };

  useEffect(() => {
    // This logic is moved to the style prop on the grid-layout div for better React practice.
  }, [layout.columns, layout.rowHeight]);

  return (
    <div className={`dashboard-preview ${className}`}>
      <div className="dashboard-header">
        <h2 className="dashboard-title">Dashboard Preview</h2>
        <p className="dashboard-subtitle">
          Interactive preview of your conversational dashboard
        </p>
      </div>

      {visuals.length === 0 ? (
        <div className="empty-state">
          <BarChart3 className="empty-icon" />
          <p className="empty-title">No visualizations yet</p>
          <p className="empty-subtitle">Start a conversation to add charts and tables</p>
        </div>
      ) : (
        <div className="grid-layout" ref={gridRef}>
          <style>{`
            .${instanceId} .grid-layout {
              --grid-columns: ${layout.columns};
              --grid-row-height: ${layout.rowHeight}px;
            }
            ${visuals.map(visual => {
              const idSafe = String(visual.id).replace(/[^a-zA-Z0-9_-]/g, '_');
              return `.${instanceId} .visual-card-${idSafe} { --grid-column-start: ${visual.position.x + 1}; --grid-column-span: ${visual.position.width}; --grid-row-start: ${visual.position.y + 1}; --grid-row-span: ${visual.position.height}; }`;
            }).join('\n')}
          `}</style>
          {visuals.map((visual) => (
            <div
              key={visual.id != null && typeof visual.id !== 'object' ? String(visual.id) : ''}
              className={`visual-card visual-card-${String(visual.id).replace(/[^a-zA-Z0-9_-]/g, '_')}`}
            >
              <div className="visual-header">
                <div className="visual-title">
                  {getChartIcon(visual.type)}
                  <h3>{visual.title}</h3>
                </div>
                <div className="compliance-indicator">
                  {visual.compliance.isCompliant ? (
                    <div className="compliance-dot compliance-dot-green" />
                  ) : (
                    <div className={`compliance-dot ${
                      visual.compliance.riskLevel === 'high' ? 'compliance-dot-red' : 'compliance-dot-yellow'
                    }`} />
                  )}
                </div>
              </div>

              {/* Chart Area */}
              <div className="visual-chart-area">
                {renderMockChart(visual)}
              </div>

              {/* Footer with metrics */}
              <div className="visual-footer">
                <div className="footer-content">
                  <div className="metric-group">
                    {visual.querySpec.metrics.slice(0, 2).map((metric, index) => (
                      <div key={String(index)} className="metric-item">
                        {getMetricIcon(metric)}
                        <span>{metric}</span>
                      </div>
                    ))}
                    {visual.querySpec.metrics.length > 2 && (
                      <span>+{visual.querySpec.metrics.length - 2} more</span>
                    )}
                  </div>
                  {visual.querySpec.dimensions.length > 0 && (
                    <div className="metric-item">
                      <Calendar className="metric-icon" />
                      <span>by {visual.querySpec.dimensions[0]}</span>
                    </div>
                  )}
                </div>
              </div>

              {/* Compliance Issues */}
              {visual.compliance.violations.length > 0 && (
                <div className="compliance-violations">
                  <div className="violation-content">
                    <div className="violation-dot" />
                    <div className="violation-text">
                      <p className="violation-title">Policy Violation</p>
                      <p>{visual.compliance.violations[0].message}</p>
                      {visual.compliance.violations[0].suggestion && (
                        <p className="violation-suggestion">
                          💡 {visual.compliance.violations[0].suggestion}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
