import React, { useEffect, useState, useMemo } from 'react';
import { Box, Typography, Chip, CircularProgress, Alert, Breadcrumbs, Link, Button, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper } from '@mui/material';
import FilterAltOffIcon from '@mui/icons-material/FilterAltOff';
import ReactECharts from 'echarts-for-react';
import {
  getSavedQuery,
  runSavedQuery,
  isSafeToRollUpAcrossRows,
  isAdditiveSafe,
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
import {
  assertCubeSubjectMirror,
  RouteBadge,
  routeBadgeFromPreview,
  type QuerySubject,
} from '../../features/analytical-subject';

export interface SavedQueryParamBinding {
  mode: 'static' | 'selection';
  value?: string;
}

export interface SavedQueryWidgetProps {
  widgetId?: string;
  savedQueryId: string;
  /**
   * Mirrored analytical subject from the page definition (PR1a).
   * When kind=cube, compared to the saved query before execute — mismatch fail-closed.
   */
  subject?: QuerySubject | null;
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
  subject: mirroredSubject,
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
  const subjectKey = JSON.stringify(mirroredSubject ?? null);

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
    setResult(null);

    const run = async () => {
      if (mirroredSubject?.kind === 'cube') {
        const sq = await getSavedQuery(savedQueryId);
        const pin = assertCubeSubjectMirror(mirroredSubject, {
          subject: sq.subject ?? sq.state?.subject,
          sourceKind: sq.sourceKind,
          boId: sq.boId,
        });
        if (!pin.ok) {
          throw new Error(`Cube subject pin mismatch: ${pin.reason}`);
        }
      }
      return runSavedQuery(savedQueryId, resolvedParams, runtimeFilters);
    };

    run()
      .then((r) => { if (!cancelled) setResult(r); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to run saved query'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [savedQueryId, paramsKey, runtimeFiltersKey, subjectKey]);

  const routeBadge = useMemo(
    () => routeBadgeFromPreview({ cubeHit: result?.cubeHit, cubeMiss: result?.cubeMiss }),
    [result?.cubeHit, result?.cubeMiss],
  );

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
  if (error) return <Alert severity="error" sx={{ fontSize: '0.75rem' }} data-testid="saved-query-pin-error">{error}</Alert>;

  const badge = (mirroredSubject?.kind === 'cube' || routeBadge) ? (
    <Box sx={{ px: 1, pt: 1 }} data-testid="saved-query-route-badge">
      <RouteBadge
        route={routeBadge ?? (mirroredSubject?.kind === 'cube' ? {
          servedFrom: 'raw',
          cubeId: mirroredSubject.cubeId,
          contractVersion: typeof mirroredSubject.contractVersion === 'number' ? mirroredSubject.contractVersion : undefined,
        } : null)}
        emptyLabel={mirroredSubject?.kind === 'cube' ? 'raw · pinned cube (no hit yet)' : undefined}
      />
    </Box>
  ) : null;

  if (!result || result.rows.length === 0) {
    return (
      <Box>
        {badge}
        <Typography variant="caption" color="text.secondary" sx={{ px: 1 }}>No data</Typography>
      </Box>
    );
  }

  if (widgetType === 'slicer') {
    const col = result.columns[0]?.name;
    const distinct = Array.from(new Set(result.rows.map((r) => String(r[col] ?? ''))));
    return (
      <Box>
        {badge}
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
    const measureColDef = result.columns.find((c) => typeof result.rows[0][c.name] === 'string' && !isNaN(Number(result.rows[0][c.name]))) || result.columns[result.columns.length - 1];

    // Fail-safe, not wait-and-see: this re-sums measureCol across EVERY
    // returned row, client-side, independent of whatever grouping/
    // aggregation the saved query itself already applied. isSafeToRollUpAcrossRows
    // checks TWO things, both required - the query's row grain is intact
    // (no join ANYWHERE in the result fans out, not just on measureCol's
    // own path - a clean column can ride next to a fanning-out one in
    // the same query), and measureCol's own ownership is unique. A
    // single-BO query (hasRelatedBOs: false) passes both by construction,
    // with no per-column metadata needed. Absence of a positive signal -
    // any related-BO query the backend hasn't fully classified - must
    // read as "can't establish it's safe," not as "probably fine."
    //
    // Two independent safety signals, both required, AND-composed here
    // at the call site (NOT merged into one predicate at the source -
    // see isSafeToRollUpAcrossRows and isAdditiveSafe for why the
    // two failures stay distinct for downstream error copy):
    //
    //   1. isSafeToRollUpAcrossRows: grain is intact (no join fan-out)
    //      AND measureCol's own ownership is unique. Answers "is every
    //      row's value representable once at this grain?"
    //   2. isAdditiveSafe: measureCol's underlying aggregation is the
    //      one aggregation whose row values are themselves additive
    //      under `+` (currently just "sum"). Answers "is `+` a
    //      meaningful way to combine these values across rows at all?"
    //
    // Both must pass. A clean SUM column on a one-side grain is safe
    // (both true); an AVG column on a one-side grain is NOT safe, even
    // though grain/ownership are clean, because a sum of averages is
    // not the average of sums.
    //
    // Default-fail polarity carries through: missing aggregation
    // (single-BO today, where the backend's GenerateSQLFromSemantic
    // branch does not populate columns), unrecognized aggregation, and
    // non-additive aggregations ALL read as unsafe here, surfacing
    // "Needs review" until the row-grain and additivity signals both
    // resolve positively.
    //
    // The inline guard preserves TS narrowing of measureColDef past the
    // early-return into the reduce below - hoisting the conjunction
    // into a `const safe: boolean` would lose that narrowing and force
    // either an `!`/`!== undefined` re-check or a non-null assertion
    // at the use site, so the conjunction stays inline even though
    // it's three terms now.
    //
    // Also NOT covered (separate, tracked): measureColDef itself is a
    // heuristic guess (first numeric-looking column, or just the last
    // column) that can land on a dimension or an id rather than an
    // intended measure. That needs its own fix - this check only
    // answers "IF this is the right column, is summing it across rows
    // safe."
    if (!measureColDef || !isSafeToRollUpAcrossRows(result, measureColDef) || !isAdditiveSafe(measureColDef)) {
      return (
        <Box sx={{ textAlign: 'center', p: 1 }}>
          {badge}
          <Typography variant="body2" color="text.secondary">
            Needs review
          </Typography>
          {style?.label && <Typography variant="caption" color="text.secondary">{style.label}</Typography>}
        </Box>
      );
    }

    // measureColDef is narrowed non-undefined by the guard above; using
    // its .name here directly (rather than a pre-computed measureCol
    // captured before that guard ran) lets that narrowing actually reach
    // the value used as the row-index key, instead of a separately-typed
    // `string | undefined` that happened to be safe in practice but
    // wasn't provably so at its own point of use.
    const total = result.rows.reduce((sum, r) => sum + (Number(r[measureColDef.name]) || 0), 0);
    return (
      <Box sx={{ textAlign: 'center', p: 1 }}>
        {badge}
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
        {badge}
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
      {badge}
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
