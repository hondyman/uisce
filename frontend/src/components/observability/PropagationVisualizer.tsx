import React, { useMemo, useCallback } from 'react';
import { Card, Empty, Tooltip, Tag, Progress, Badge } from 'antd';
import { ArrowRightOutlined, WarningOutlined, CheckCircleOutlined } from '@ant-design/icons';
import './PropagationVisualizer.module.css';

// ============================================================================
// Phase 3.3: Propagation Visualizer Component
// Visualize cross-region propagation paths and issue spread
// ============================================================================

interface PropagationPath {
  fromRegion: string;
  toRegion: string;
  hopCount: number;
  estimatedMs: number;
  isLikely: boolean;
  correlationScore: number;
  severity: 'critical' | 'high' | 'medium' | 'low';
}

interface RegionNode {
  region: string;
  isAffected: boolean;
  issueCount: number;
  health: number;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'none';
}

interface PropagationVisualizerProps {
  paths: PropagationPath[];
  regions: RegionNode[];
  loading?: boolean;
  onPathClick?: (path: PropagationPath) => void;
}

export const PropagationVisualizer: React.FC<PropagationVisualizerProps> = ({
  paths = [],
  regions = [],
  loading = false,
  onPathClick,
}) => {
  // Group paths by source region
  const groupedPaths = useMemo(() => {
    const grouped = new Map<string, PropagationPath[]>();
    paths.forEach((path) => {
      const key = path.fromRegion;
      if (!grouped.has(key)) {
        grouped.set(key, []);
      }
      grouped.get(key)!.push(path);
    });
    return grouped;
  }, [paths]);

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return '#ff4d4f';
      case 'high':
        return '#ff7a45';
      case 'medium':
        return '#faad14';
      case 'low':
        return '#52c41a';
      default:
        return '#1890ff';
    }
  };

  const getHealthStatus = (health: number) => {
    if (health > 0.8) return { status: 'success', text: 'Healthy' };
    if (health > 0.6) return { status: 'warning', text: 'Degraded' };
    return { status: 'error', text: 'Critical' };
  };

  const renderRegionNode = (region: RegionNode) => {
    const healthStatus = getHealthStatus(region.health);
    return (
      <Tooltip
        key={region.region}
        title={`Issues: ${region.issueCount}, Health: ${(region.health * 100).toFixed(0)}%`}
      >
        <div className="region-node">
          <div className="region-header">
            <Badge
              status={
                region.severity === 'none'
                  ? 'success'
                  : region.severity === 'critical'
                    ? 'error'
                    : region.severity === 'high'
                      ? 'warning'
                      : 'processing'
              }
              text={region.region}
            />
          </div>
          <div className="region-metrics">
            <span className="issue-count">Issues: {region.issueCount}</span>
            <Progress
              type="circle"
              percent={region.health * 100}
              width={40}
              strokeColor={getSeverityColor(region.severity)}
              format={(percent) => `${percent}%`}
            />
          </div>
        </div>
      </Tooltip>
    );
  };

  const renderPropagationPath = (path: PropagationPath, index: number) => {
    const sourceRegion = regions.find(r => r.region === path.fromRegion);
    const targetRegion = regions.find(r => r.region === path.toRegion);

    return (
      <div
        key={`${path.fromRegion}-${path.toRegion}-${index}`}
        className="propagation-path"
        onClick={() => onPathClick?.(path)}
        style={{ cursor: onPathClick ? 'pointer' : 'default' }}
      >
        <div className="path-source">
          <span className="region-label">{path.fromRegion}</span>
        </div>

        <div className="path-arrow">
          <ArrowRightOutlined />
          <div className="path-details">
            <span className="hop-count">{path.hopCount} hop</span>
            <span className="latency">{path.estimatedMs}ms</span>
          </div>
        </div>

        <div className="path-target">
          <span className="region-label">{path.toRegion}</span>
        </div>

        <div className="path-risk">
          <Tag
            color={getSeverityColor(path.severity)}
            icon={
              path.isLikely ? <WarningOutlined /> : <CheckCircleOutlined />
            }
          >
            {path.isLikely ? 'Likely' : 'Possible'} • {(path.correlationScore * 100).toFixed(0)}%
          </Tag>
        </div>
      </div>
    );
  };

  if (paths.length === 0) {
    return (
      <Card
        title="Cross-Region Propagation"
        loading={loading}
      >
        <Empty description="No propagation paths detected" />
      </Card>
    );
  }

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <WarningOutlined style={{ color: '#faad14' }} />
          <span>Cross-Region Propagation Paths ({paths.length})</span>
        </div>
      }
      loading={loading}
      className="propagation-visualizer-card"
    >
      <div className="propagation-container">
        {/* Region Overview */}
        {regions.length > 0 && (
          <div className="regions-overview">
            <h4>Region Status</h4>
            <div className="regions-grid">
              {regions.map(renderRegionNode)}
            </div>
          </div>
        )}

        {/* Propagation Paths */}
        <div className="propagation-paths">
          <h4>Detected Paths</h4>
          <div className="paths-list">
            {paths.map((path, index) => renderPropagationPath(path, index))}
          </div>
        </div>

        {/* Propagation Summary */}
        <div className="propagation-summary">
          <h4>Summary</h4>
          <div className="summary-grid">
            <div className="summary-item">
              <span className="label">Total Paths:</span>
              <span className="value">{paths.length}</span>
            </div>
            <div className="summary-item">
              <span className="label">Likely Paths:</span>
              <span className="value">{paths.filter(p => p.isLikely).length}</span>
            </div>
            <div className="summary-item">
              <span className="label">Critical Severity:</span>
              <span className="value">{paths.filter(p => p.severity === 'critical').length}</span>
            </div>
            <div className="summary-item">
              <span className="label">Affected Regions:</span>
              <span className="value">{new Set(paths.flatMap(p => [p.fromRegion, p.toRegion])).size}</span>
            </div>
          </div>
        </div>
      </div>
    </Card>
  );
};

export default PropagationVisualizer;
