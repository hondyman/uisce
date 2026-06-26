import React, { useMemo, useCallback, useState } from 'react';
import { Card, Select, Badge, Tooltip, Empty, Spin, message } from 'antd';
import { GlobalOutlined, CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined } from '@ant-design/icons';
import './RegionSelector.module.css';

// ============================================================================
// Phase 3.3: Region Selector Component
// Multi-region incident response with region selection and status tracking
// ============================================================================

interface Region {
  code: string;
  name: string;
  tier: 'tier-1' | 'tier-2' | 'tier-3';
  isHealthy: boolean;
  healthScore: number;
  availableCapacity: number;
  avgLatencyMs: number;
  tenants: number;
}

interface RegionPreference {
  tenantId: string;
  preferredRegion: string;
  allowedRegions: string[];
  localityPreference: 'latency' | 'data_residency' | 'cost';
  latencyThreshold: number;
}

interface RegionSelectorProps {
  tenantId?: string;
  onRegionSelect: (region: string, preference: RegionPreference) => void;
  selectedRegion?: string;
  loading?: boolean;
  regions: Region[];
}

export const RegionSelector: React.FC<RegionSelectorProps> = ({
  tenantId = 'default',
  onRegionSelect,
  selectedRegion = 'us-east-1',
  loading = false,
  regions,
}) => {
  const [localityPref, setLocalityPref] = useState<'latency' | 'data_residency' | 'cost'>('latency');
  const [customThreshold, setCustomThreshold] = useState<number>(200);

  const handleRegionChange = useCallback((value: string) => {
    const selectedRegionData = regions.find(r => r.code === value);
    if (!selectedRegionData) {
      message.error('Region not found');
      return;
    }

    const preference: RegionPreference = {
      tenantId,
      preferredRegion: value,
      allowedRegions: regions.filter(r => r.isHealthy).map(r => r.code),
      localityPreference: localityPref,
      latencyThreshold: customThreshold,
    };

    onRegionSelect(value, preference);
    message.success(`Region changed to ${selectedRegionData.name}`);
  }, [tenantId, regions, localityPref, customThreshold, onRegionSelect]);

  const healthStatusIcon = (region: Region) => {
    if (!region.isHealthy) {
      return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
    }
    if (region.healthScore > 0.8) {
      return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
    }
    return <ClockCircleOutlined style={{ color: '#faad14' }} />;
  };

  const regionOptions = useMemo(
    () =>
      regions.map((region) => ({
        label: (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px' }}>
            <span>{region.name}</span>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              {healthStatusIcon(region)}
              <span style={{ fontSize: '12px', color: '#666' }}>
                {region.healthScore * 100}% • {region.avgLatencyMs}ms
              </span>
            </div>
          </div>
        ),
        value: region.code,
      })),
    [regions]
  );

  const selectedRegionData = regions.find(r => r.code === selectedRegion);

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <GlobalOutlined />
          <span>Region Selector</span>
        </div>
      }
      className="region-selector-card"
      loading={loading}
    >
      <Spin spinning={loading}>
        <div className="region-selector-container">
          {/* Region Selection */}
          <div className="region-selection">
            <label>Primary Region:</label>
            <Select
              value={selectedRegion}
              onChange={handleRegionChange}
              options={regionOptions}
              style={{ width: '100%' }}
            />
          </div>

          {/* Selected Region Details */}
          {selectedRegionData && (
            <div className="region-details">
              <h4>{selectedRegionData.name}</h4>
              <div className="detail-grid">
                <div className="detail-item">
                  <span className="label">Code:</span>
                  <span className="value">{selectedRegionData.code}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Tier:</span>
                  <Badge color={selectedRegionData.tier === 'tier-1' ? '#52c41a' : '#faad14'} text={selectedRegionData.tier} />
                </div>
                <div className="detail-item">
                  <span className="label">Health:</span>
                  <span className="value">{(selectedRegionData.healthScore * 100).toFixed(0)}%</span>
                </div>
                <div className="detail-item">
                  <span className="label">Latency:</span>
                  <span className="value">{selectedRegionData.avgLatencyMs}ms</span>
                </div>
                <div className="detail-item">
                  <span className="label">Capacity:</span>
                  <span className="value">{selectedRegionData.availableCapacity}%</span>
                </div>
                <div className="detail-item">
                  <span className="label">Tenants:</span>
                  <span className="value">{selectedRegionData.tenants}</span>
                </div>
              </div>
            </div>
          )}

          {/* Locality Preferences */}
          <div className="locality-preferences">
            <label>Locality Preference:</label>
            <Select
              value={localityPref}
              onChange={setLocalityPref}
              options={[
                { label: 'Optimize for Latency', value: 'latency' },
                { label: 'Data Residency', value: 'data_residency' },
                { label: 'Optimize for Cost', value: 'cost' },
              ]}
              style={{ width: '100%' }}
            />
          </div>

          {/* Latency Threshold */}
          <div className="latency-threshold">
            <label>Max Latency Threshold (ms):</label>
            <input
              type="number"
              aria-label="Max Latency Threshold"
              value={customThreshold}
              onChange={(e) => setCustomThreshold(Math.max(50, parseInt(e.target.value) || 200))}
              min="50"
              max="1000"
              step="50"
              style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid #d9d9d9' }}
            />
          </div>

          {/* Fallback Regions */}
          <div className="fallback-regions">
            <label>Fallback Regions:</label>
            <div className="fallback-list">
              {regions
                .filter(r => r.code !== selectedRegion && r.isHealthy)
                .map((region) => (
                  <Tooltip key={region.code} title={`Latency: ${region.avgLatencyMs}ms, Health: ${(region.healthScore * 100).toFixed(0)}%`}>
                    <Badge
                      color={region.healthScore > 0.8 ? '#52c41a' : '#faad14'}
                      text={`${region.name} (${region.avgLatencyMs}ms)`}
                      style={{ marginRight: '12px' }}
                    />
                  </Tooltip>
                ))}
            </div>
          </div>
        </div>
      </Spin>
    </Card>
  );
};

export default RegionSelector;
