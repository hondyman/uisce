import React, { useMemo, useEffect } from 'react';
import { Box, Card, Typography, Stack, Tooltip, Chip } from '@mui/material';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import type { KpiTileConfig } from '../../types/pageStudio';

export interface KpiTileWidgetProps {
  id?: string;
  title?: string;
  config: KpiTileConfig;
  data?: {
    rows?: Record<string, unknown>[];
    columns?: { name: string; type: string }[];
    error?: string;
  };
  mode?: 'design' | 'preview';
  onEmitCrossFilter?: (termNodeId: string, value: unknown) => void;
  onRefresh?: () => void;
}

export function formatMetricValue(val: number | null | undefined, format: KpiTileConfig['format']): string {
  if (val === null || val === undefined || isNaN(val)) return '—';

  const precision = format.precision ?? 2;
  const prefix = format.prefix ?? '';
  const suffix = format.suffix ?? '';

  let formatted = '';
  switch (format.type) {
    case 'currency':
      const sym = format.currencySymbol ?? '$';
      formatted = `${sym}${val.toLocaleString(undefined, { minimumFractionDigits: precision, maximumFractionDigits: precision })}`;
      break;
    case 'percentage':
      formatted = `${val.toFixed(precision)}%`;
      break;
    case 'compact':
      if (Math.abs(val) >= 1_000_000_000) {
        formatted = `${(val / 1_000_000_000).toFixed(1)}B`;
      } else if (Math.abs(val) >= 1_000_000) {
        formatted = `${(val / 1_000_000).toFixed(1)}M`;
      } else if (Math.abs(val) >= 1_000) {
        formatted = `${(val / 1_000).toFixed(1)}K`;
      } else {
        formatted = val.toFixed(precision);
      }
      break;
    case 'number':
    default:
      formatted = val.toLocaleString(undefined, { minimumFractionDigits: precision, maximumFractionDigits: precision });
      break;
  }

  return `${prefix}${formatted}${suffix}`;
}

export function renderMiniSparkline(
  values: number[],
  isPositive: boolean,
  type: 'line' | 'bar' | 'area' = 'line'
) {
  if (!values || values.length < 2) return null;

  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const width = 120;
  const height = 36;
  const strokeColor = isPositive ? '#16a34a' : '#dc2626';
  const fillColor = isPositive ? 'rgba(22, 163, 74, 0.15)' : 'rgba(220, 38, 38, 0.15)';

  const points = values.map((v, i) => {
    const x = (i / (values.length - 1)) * width;
    const y = height - ((v - min) / range) * (height - 6) - 3;
    return `${x},${y}`;
  });

  const polylineStr = points.join(' ');
  const areaPoints = `0,${height} ${polylineStr} ${width},${height}`;

  return (
    <svg width={width} height={height} style={{ overflow: 'visible' }}>
      {type === 'area' && (
        <polygon points={areaPoints} fill={fillColor} />
      )}
      <polyline
        fill="none"
        stroke={strokeColor}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={polylineStr}
      />
    </svg>
  );
}

export const KpiTileWidget: React.FC<KpiTileWidgetProps> = ({
  id,
  title,
  config,
  data,
  mode = 'preview',
  onEmitCrossFilter,
  onRefresh,
}) => {
  const rows = data?.rows || [];
  // Resolution order: metricId -> measureAlias
  const targetFieldKey = config.metricId || config.measureAlias;

  // Live polling at >=60s with document visibility state check
  useEffect(() => {
    if (!config.refreshInterval || config.refreshInterval < 60 || !onRefresh) {
      return;
    }
    const intervalMs = config.refreshInterval * 1000;
    const intervalId = setInterval(() => {
      if (typeof document === 'undefined' || document.visibilityState === 'visible') {
        onRefresh();
      }
    }, intervalMs);
    return () => clearInterval(intervalId);
  }, [config.refreshInterval, onRefresh]);

  // Designer warning check: comparison enabled without trend dimension for period comparison
  const missingTrendDimensionWarning = useMemo(() => {
    if (
      config.comparison?.enabled &&
      config.comparison.type === 'previous_period' &&
      !config.trendDimensionAlias
    ) {
      return 'Comparison enabled without trend/period dimension in query';
    }
    return null;
  }, [config.comparison, config.trendDimensionAlias]);

  // Extract primary value and historical series
  const { currentValue, delta, trendPoints } = useMemo(() => {
    if (!rows.length || !targetFieldKey) {
      return { currentValue: null, delta: null, trendPoints: [] };
    }

    const series = rows.map((r) => {
      // Look up targetFieldKey (e.g. metricId, metric_val, or measureAlias)
      const raw = r[targetFieldKey] !== undefined ? r[targetFieldKey] : r['metric_val'] !== undefined ? r['metric_val'] : r[config.measureAlias];
      const val = Number(raw);
      return isNaN(val) ? 0 : val;
    });

    const current = series[series.length - 1];
    let calculatedDelta: { value: number; percent: number; isPositive: boolean } | null = null;

    if (config.comparison?.enabled) {
      let baseline: number | null = null;
      if (config.comparison.type === 'previous_period' && series.length > 1) {
        baseline = series[series.length - 2];
      } else if (config.comparison.type === 'target_literal' && config.comparison.targetValue !== undefined) {
        baseline = config.comparison.targetValue;
      }

      if (baseline !== null && baseline !== undefined) {
        const diff = current - baseline;
        const pct = baseline !== 0 ? (diff / Math.abs(baseline)) * 100 : 0;
        let isPos = diff >= 0;
        if (config.comparison.invertPolarity) {
          isPos = !isPos;
        }
        calculatedDelta = { value: diff, percent: pct, isPositive: isPos };
      }
    }

    return { currentValue: current, delta: calculatedDelta, trendPoints: series };
  }, [rows, targetFieldKey, config.measureAlias, config.comparison]);

  const handleClick = () => {
    if (config.interactionConfig?.crossFilter?.enabled && onEmitCrossFilter) {
      const termNodeId = config.interactionConfig.crossFilter.termNodeId;
      onEmitCrossFilter(termNodeId, currentValue);
    }
  };

  return (
    <Card
      variant="outlined"
      sx={{
        p: 2,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        cursor: config.interactionConfig?.crossFilter?.enabled ? 'pointer' : 'default',
        '&:hover': config.interactionConfig?.crossFilter?.enabled
          ? { borderColor: 'primary.main', boxShadow: 1 }
          : {},
      }}
      onClick={handleClick}
    >
      <Box>
        <Stack direction="row" justifyContent="space-between" alignItems="center" spacing={1}>
          <Typography variant="body2" color="text.secondary" fontWeight={500} noWrap>
            {title || config.measureAlias || 'KPI Metric'}
          </Typography>
          {mode === 'design' && !config.metricId && (
            <Tooltip title="Using legacy measureAlias binding. Consider binding to a Semantic Metric ID for catalog governance.">
              <Chip size="small" variant="outlined" label="Raw Field" sx={{ height: 18, fontSize: '0.65rem' }} />
            </Tooltip>
          )}
          {missingTrendDimensionWarning && mode === 'design' && (
            <Tooltip title={missingTrendDimensionWarning}>
              <Chip
                size="small"
                color="warning"
                icon={<WarningAmberIcon fontSize="small" />}
                label="Config Warning"
                sx={{ height: 20, fontSize: '0.7rem' }}
              />
            </Tooltip>
          )}
        </Stack>

        <Typography variant="h4" fontWeight={700} sx={{ mt: 0.5, letterSpacing: '-0.02em' }}>
          {formatMetricValue(currentValue, config.format)}
        </Typography>
      </Box>

      <Stack direction="row" justifyContent="space-between" alignItems="flex-end" sx={{ mt: 1 }}>
        {delta ? (
          <Stack
            direction="row"
            alignItems="center"
            spacing={0.5}
            sx={{
              color: delta.isPositive ? 'success.main' : 'error.main',
              fontWeight: 600,
              fontSize: '0.875rem',
            }}
          >
            {delta.isPositive ? (
              <TrendingUpIcon fontSize="small" />
            ) : (
              <TrendingDownIcon fontSize="small" />
            )}
            <span>
              {delta.percent > 0 ? '+' : ''}
              {config.comparison?.deltaFormat === 'percentage'
                ? `${delta.percent.toFixed(1)}%`
                : formatMetricValue(delta.value, config.format)}
            </span>
          </Stack>
        ) : (
          <Box />
        )}

        {config.sparkline?.enabled && trendPoints.length > 1 && (
          <Box sx={{ ml: 'auto' }}>
            {renderMiniSparkline(
              trendPoints,
              delta ? delta.isPositive : true,
              config.sparkline.type || 'line'
            )}
          </Box>
        )}
      </Stack>
    </Card>
  );
};

export default KpiTileWidget;
