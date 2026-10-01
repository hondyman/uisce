import React, { useEffect, useState, useMemo } from 'react';
import { Box, Typography, Chip, CircularProgress, Alert, Breadcrumbs, Link, Button, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper } from '@mui/material';
import FilterAltOffIcon from '@mui/icons-material/FilterAltOff';
import ReactECharts from 'echarts-for-react';
import {
  runSavedQuery,
  type SavedQueryRunResult,
  type RuntimeFilter,
} from '../../features/query-builder/services/savedQueryApi';
import { buildChartOption } from '../../features/query-builder/utils/chartOption';
import { echarts, CHART_ROW_SOFT_CAP, getChartSpec } from '../../features/query-builder/charts';
import { useSelection } from './SelectionContext';
import type { WidgetStyleOptions } from '../../components/reporting/ReportWidgetRenderer';
import {
  type DrillStep,
  type DrillThroughTarget,
  buildDrillThroughContext,
  resolveDrillPath,
} from '../../features/query-builder/utils/drilldown';
import {
  useCrossFilterBus,
  type CrossFilterBus,
  type CrossFilterConfig,
} from '../../features/query-builder/utils/crossFilterBus';

export interface SavedQueryParamBinding {
  mode: 'static' | 'selection';
  value?: string;
}

export interface SavedQueryWidgetProps {
  widgetId?: string;
  savedQueryId: string;
  /** How to render the result - matches the placing widget's type. */
  widgetType: 'slicer' | 'chart' | 'gauge';
  paramBindings?: Record<string, SavedQueryParamBinding>;
  style?: WidgetStyleOptions;
  drillThroughTargets?: DrillThroughTarget[];
  onDrillThrough?: (target: DrillThroughTarget, context: Record<string, unknown>) => void;
  isAuthorMode?: boolean;
  crossFilterConfig?: CrossFilterConfig;
  queryTerms?: string[];
  crossFilterBus?: CrossFilterBus;
}

/**
 * Renders a Slicer/Chart/KPI widget whose data comes from a saved,
 * parameterized query with drill-down, drill-through, and cross-filtering capabilities.
 */
const SavedQueryWidget: React.FC<SavedQueryWidgetProps> = ({
  widgetId,
  savedQueryId,
  widgetType,
  paramBindings,
  style,
  drillThroughTargets = [],
  onDrillThrough,
  isAuthorMode = false,
  crossFilterConfig,
  queryTerms,
  crossFilterBus: propBus,
}) => {
  const { selection } = useSelection();
  const contextBus = useCrossFilterBus();
  const bus = propBus || contextBus;

  const [result, setResult] = useState<SavedQueryRunResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [drillSteps, setDrillSteps] = useState<DrillStep[]>([]);
  const [busVersion, setBusVersion] = useState(0);

  // Subscribe to bus changes so cross-filters re-evaluate
  useEffect(() => {
    return bus.subscribe(() => {
      setBusVersion((v) => v + 1);
    });
  }, [bus]);

  const primaryCol = result?.columns[0]?.name || 'dimension';
  const effectiveTerms = useMemo(() => {
    if (queryTerms && queryTerms.length > 0) return queryTerms;
    if (result?.columns) return result.columns.map((c) => c.name);
    return [primaryCol];
  }, [queryTerms, result?.columns, primaryCol]);

  const resolution = useMemo(
    () => resolveDrillPath(primaryCol, primaryCol, undefined),
    [primaryCol]
  );

  // Resolve each parameter binding to a concrete value for this render
  const resolvedParams: Record<string, string> = {};
  Object.entries(paramBindings || {}).forEach(([name, binding]) => {
    if (binding.mode === 'selection') {
      if (selection?.recordId) resolvedParams[name] = selection.recordId;
    } else if (binding.value) {
      resolvedParams[name] = binding.value;
    }
  });

  // Include active drill step filters as query parameters if supported
  drillSteps.forEach((step) => {
    resolvedParams[`drill_${step.dimension.alias}`] = String(step.value);
  });

  // Get cross-filters applicable to this widget
  const effectiveWidgetId = widgetId || savedQueryId;
  const applicableCrossFilters = useMemo(() => {
    return bus.getFiltersForWidget(effectiveWidgetId, effectiveTerms, crossFilterConfig);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bus, effectiveWidgetId, effectiveTerms, crossFilterConfig, busVersion]);

  // Combine drill-down filters and cross filters into runtime filters
  const runtimeFilters: RuntimeFilter[] = useMemo(() => {
    const filters: RuntimeFilter[] = [];
    drillSteps.forEach((s) => {
      filters.push(s.filter);
    });
    applicableCrossFilters.forEach((cf) => {
      filters.push(cf);
    });
    return filters;
  }, [drillSteps, applicableCrossFilters]);

  const paramsKey = JSON.stringify(resolvedParams);
  const runtimeFiltersKey = JSON.stringify(runtimeFilters);

  useEffect(() => {
    if (!savedQueryId) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    runSavedQuery(savedQueryId, resolvedParams, runtimeFilters)
      .then((r) => { if (!cancelled) setResult(r); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to run saved query'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [savedQueryId, paramsKey, runtimeFiltersKey]);

  const handleChartClick = (params: any) => {
    if (!result || !params) return;
    const clickedVal = params.name || params.value;
    if (clickedVal === undefined || clickedVal === null || clickedVal === '') return;

    // 1. Drill-through target navigation if specified
    if (drillThroughTargets.length > 0 && onDrillThrough) {
      const newStep: DrillStep = {
        dimension: { termNodeId: primaryCol, alias: primaryCol },
        filter: { termNodeId: primaryCol, operator: 'eq', value: clickedVal },
        label: primaryCol,
        value: clickedVal,
      };
      const context = buildDrillThroughContext([...drillSteps, newStep], selection?.recordId);
      onDrillThrough(drillThroughTargets[0], context);
      return;
    }

    // 2. Cross-filtering emit if enabled
    if (crossFilterConfig?.enabled !== false && crossFilterConfig?.mode !== 'receive') {
      bus.emit(effectiveWidgetId, primaryCol, clickedVal);
    }

    // 3. Hierarchical Drill-down if hierarchy configured (for non-slicer widgets)
    if (widgetType !== 'slicer' && resolution.status === 'ok') {
      const nextDim = resolution.path[resolution.currentLevel + 1];
      const newStep: DrillStep = {
        dimension: { termNodeId: nextDim, alias: nextDim },
        filter: { termNodeId: primaryCol, operator: 'eq', value: clickedVal },
        label: primaryCol,
        value: clickedVal,
      };
      setDrillSteps((prev) => [...prev, newStep]);
    }
  };

  const resetDrill = (index: number) => {
    setDrillSteps((prev) => prev.slice(0, index));
  };

  const breadcrumbs = drillSteps.length > 0 && (
    <Breadcrumbs sx={{ fontSize: '0.75rem', px: 1, py: 0.5 }}>
      <Link component="button" variant="caption" onClick={() => resetDrill(0)}>
        {result?.name || 'Root'}
      </Link>
      {drillSteps.map((step, i) => (
        <Link key={i} component="button" variant="caption" onClick={() => resetDrill(i + 1)}>
          {String(step.value)}
        </Link>
      ))}
    </Breadcrumbs>
  );

  const crossFilterControls = bus.count > 0 && (
    <Box sx={{ px: 1, py: 0.5, display: 'flex', alignItems: 'center', gap: 1 }}>
      <Chip
        icon={<FilterAltOffIcon fontSize="small" />}
        label={`Clear filters (${bus.count})`}
        size="small"
        color="primary"
        variant="outlined"
        onClick={() => bus.clear()}
        onDelete={() => bus.clear()}
      />
    </Box>
  );

  if (loading && !result) return <Box sx={{ p: 2, display: 'flex', justifyContent: 'center' }}><CircularProgress size={20} /></Box>;
  if (error) return <Alert severity="error" sx={{ fontSize: '0.75rem' }}>{error}</Alert>;
  if (!result || result.rows.length === 0) return <Typography variant="caption" color="text.secondary">No data</Typography>;

  if (widgetType === 'slicer') {
    const col = result.columns[0]?.name;
    const distinct = Array.from(new Set(result.rows.map((r) => String(r[col] ?? ''))));
    return (
      <Box>
        {breadcrumbs}
        {crossFilterControls}
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, p: 0.5 }}>
          {distinct.map((v) => (
            <Chip
              key={v}
              label={v}
              size="small"
              variant={style?.variant || 'outlined'}
              onClick={() => handleChartClick({ name: v })}
            />
          ))}
        </Box>
      </Box>
    );
  }

  if (widgetType === 'gauge') {
    const measureCol = result.columns.find((c) => typeof result.rows[0][c.name] === 'string' && !isNaN(Number(result.rows[0][c.name])))?.name || result.columns[result.columns.length - 1]?.name;
    const total = result.rows.reduce((sum, r) => sum + (Number(r[measureCol]) || 0), 0);
    return (
      <Box sx={{ textAlign: 'center', p: 1 }}>
        {breadcrumbs}
        {crossFilterControls}
        <Typography variant="h4" fontWeight={700} sx={{ color: style?.valueColor, fontSize: style?.valueFontSize ? `${style.valueFontSize}px` : undefined }}>
          {total.toLocaleString()}
        </Typography>
        {style?.label && <Typography variant="caption" color="text.secondary">{style.label}</Typography>}
      </Box>
    );
  }

  // Soft cap check: if row count exceeds 10,000 rows, fall back to table view
  const isOverCap = result.rows.length > CHART_ROW_SOFT_CAP;
  const isKnownChart = !!getChartSpec(result.chartType);

  if (isOverCap || !isKnownChart || result.chartType === 'table') {
    return (
      <Box sx={{ p: 1, display: 'flex', flexDirection: 'column', height: '100%' }}>
        {breadcrumbs}
        {crossFilterControls}
        {isOverCap && (
          <Alert severity="info" sx={{ fontSize: '0.75rem', mb: 1 }}>
            Result set exceeds 10,000 rows. Displaying table view to maintain browser performance.
          </Alert>
        )}
        {!isKnownChart && result.chartType !== 'table' && (
          <Alert severity="warning" sx={{ fontSize: '0.75rem', mb: 1 }}>
            Chart type &apos;{result.chartType}&apos; is not supported. Displaying table view.
          </Alert>
        )}
        <TableContainer component={Paper} sx={{ flex: 1, maxHeight: 220 }}>
          <Table size="small" stickyHeader>
            <TableHead>
              <TableRow>
                {result.columns.map((col) => (
                  <TableCell key={col.name}>{col.name}</TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {result.rows.slice(0, 100).map((row, idx) => (
                <TableRow key={idx}>
                  {result.columns.map((col) => (
                    <TableCell key={col.name}>{String(row[col.name] ?? '')}</TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>
    );
  }

  const chartOption = buildChartOption(result.rows, result.columns, result.chartType);
  if (style?.color) {
    (chartOption as { color?: string[] }).color = [style.color];
  }
  if (result.chartType === 'pie' || result.chartType === 'donut') {
    (chartOption as { legend?: { show: boolean } }).legend = { show: !!style?.showLegend };
  }
  return (
    <Box sx={{ height: 260, display: 'flex', flexDirection: 'column' }}>
      {breadcrumbs}
      {crossFilterControls}
      <Box sx={{ flex: 1, minHeight: 0 }}>
        <ReactECharts
          echarts={echarts}
          style={{ height: '100%', width: '100%' }}
          option={chartOption}
          onEvents={{ click: handleChartClick }}
          notMerge
        />
      </Box>
    </Box>
  );
};

export default SavedQueryWidget;
