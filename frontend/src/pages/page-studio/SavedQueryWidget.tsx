import React, { useEffect, useState } from 'react';
import { Box, Typography, Chip, CircularProgress, Alert } from '@mui/material';
import ReactECharts from 'echarts-for-react';
import { runSavedQuery, isSafeToRollUpAcrossRows } from '../../features/query-builder/services/savedQueryApi';
import { buildChartOption } from '../../features/query-builder/utils/chartOption';
import type { SavedQueryRunResult } from '../../features/query-builder/services/savedQueryApi';
import { useSelection } from './SelectionContext';
import type { WidgetStyleOptions } from '../../components/reporting/ReportWidgetRenderer';

export interface SavedQueryParamBinding {
  mode: 'static' | 'selection';
  value?: string;
}

interface SavedQueryWidgetProps {
  savedQueryId: string;
  /** How to render the result - matches the placing widget's type. */
  widgetType: 'slicer' | 'chart' | 'gauge';
  paramBindings?: Record<string, SavedQueryParamBinding>;
  style?: WidgetStyleOptions;
}

/**
 * Renders a Slicer/Chart/KPI widget whose data comes from a saved,
 * parameterized query (built in the Alpha Query Builder / Query Studio and
 * exposed at GET /api/explorer/saved-queries/{id}/preview) instead of the
 * widget auto-picking its own fields - see PropertiesPanel.tsx's "Data
 * Source: Use a saved query" toggle for how savedQueryId/paramBindings get
 * set.
 *
 * A "selection" param binding resolves to the page's current
 * SelectionContext record id (whatever a master Table/List row selected
 * elsewhere on the page) - the mechanism that lets one saved query like
 * "orders for account X" be reused as a Detail-page chart scoped to
 * whichever record the user is looking at, without a fresh query per page.
 *
 * Deliberately simpler than ReportWidgetRenderer's ad-hoc path: it does not
 * publish to useCrossFilterStore (the saved-query REST response carries no
 * termNodeId/boId to key a cross-filter on), so a Slicer here is a
 * read-only "current breakdown" display rather than an interactive filter.
 * That's a known, disclosed scope limit, not an oversight.
 */
const SavedQueryWidget: React.FC<SavedQueryWidgetProps> = ({ savedQueryId, widgetType, paramBindings, style }) => {
  const { selection } = useSelection();
  const [result, setResult] = useState<SavedQueryRunResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Resolve each parameter binding to a concrete value for this render -
  // "selection" mode re-resolves whenever the page's selected record
  // changes, so the widget refetches automatically.
  const resolvedParams: Record<string, string> = {};
  Object.entries(paramBindings || {}).forEach(([name, binding]) => {
    if (binding.mode === 'selection') {
      if (selection?.recordId) resolvedParams[name] = selection.recordId;
    } else if (binding.value) {
      resolvedParams[name] = binding.value;
    }
  });
  const paramsKey = JSON.stringify(resolvedParams);

  useEffect(() => {
    if (!savedQueryId) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    runSavedQuery(savedQueryId, resolvedParams)
      .then((r) => { if (!cancelled) setResult(r); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to run saved query'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [savedQueryId, paramsKey]);

  if (loading) return <Box sx={{ p: 2, display: 'flex', justifyContent: 'center' }}><CircularProgress size={20} /></Box>;
  if (error) return <Alert severity="error" sx={{ fontSize: '0.75rem' }}>{error}</Alert>;
  if (!result || result.rows.length === 0) return <Typography variant="caption" color="text.secondary">No data</Typography>;

  if (widgetType === 'slicer') {
    const col = result.columns[0]?.name;
    const distinct = Array.from(new Set(result.rows.map((r) => String(r[col] ?? ''))));
    return (
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
        {distinct.map((v) => <Chip key={v} label={v} size="small" variant={style?.variant || 'outlined'} />)}
      </Box>
    );
  }

  if (widgetType === 'gauge') {
    const measureColDef = result.columns.find((c) => typeof result.rows[0][c.name] === 'string' && !isNaN(Number(result.rows[0][c.name]))) || result.columns[result.columns.length - 1];
    const measureCol = measureColDef?.name;

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
    // NOT covered: whether `+` is even a meaningful way to combine
    // measureCol's values across rows at all (aggregation linearity - a
    // MIN or COUNT(DISTINCT) column summed with `+` is wrong regardless
    // of grain/ownership). This widget has no access to the underlying
    // aggregation to check that yet. See isSafeToRollUpAcrossRows's doc
    // comment for the full three-way split.
    //
    // Also NOT covered: a second, independent way this widget can show a
    // wrong number - measureColDef itself is a heuristic guess (first
    // numeric-looking column, or just the last column) that can land on
    // a dimension or an id rather than an intended measure. That needs
    // its own fix, tracked separately - this check only answers "IF this
    // is the right column, is summing it across rows safe."
    if (!measureColDef || !isSafeToRollUpAcrossRows(result, measureColDef)) {
      return (
        <Box sx={{ textAlign: 'center', p: 1 }}>
          <Typography variant="body2" color="text.secondary">
            Needs review
          </Typography>
          {style?.label && <Typography variant="caption" color="text.secondary">{style.label}</Typography>}
        </Box>
      );
    }

    const total = result.rows.reduce((sum, r) => sum + (Number(r[measureCol]) || 0), 0);
    return (
      <Box sx={{ textAlign: 'center', p: 1 }}>
        <Typography variant="h4" fontWeight={700} sx={{ color: style?.valueColor, fontSize: style?.valueFontSize ? `${style.valueFontSize}px` : undefined }}>
          {total.toLocaleString()}
        </Typography>
        {style?.label && <Typography variant="caption" color="text.secondary">{style.label}</Typography>}
      </Box>
    );
  }

  const chartOption = buildChartOption(result.rows, result.columns, result.chartType);
  if (style?.color) {
    (chartOption as { color?: string[] }).color = [style.color];
  }
  if (result.chartType === 'pie') {
    (chartOption as { legend?: { show: boolean } }).legend = { show: !!style?.showLegend };
  }
  return (
    <Box sx={{ height: 260 }}>
      <ReactECharts style={{ height: '100%', width: '100%' }} option={chartOption} notMerge />
    </Box>
  );
};

export default SavedQueryWidget;
