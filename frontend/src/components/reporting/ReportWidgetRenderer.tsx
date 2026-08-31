import React, { useEffect, useMemo, useState } from 'react';
import { Box, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, CircularProgress, Alert, Breadcrumbs, Link, Chip } from '@mui/material';
import ReactECharts from 'echarts-for-react';
import { executeQuery, fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { QueryDef, QueryExecuteResult, SemanticTermView } from '../../features/query-builder/types/queryDef';
import { apiFetch } from '../../lib/apiClient';
import { useCrossFilterStore, crossFilterKey } from '../../store/useCrossFilterStore';

export interface ReportWidgetBinding {
  boId: string;
  bindingId: string;
  tenantId: string;
  /** [{ termNodeId, alias }] */
  dimensions?: { termNodeId: string; alias: string }[];
  /** [{ termNodeId, alias, agg }] */
  measures?: { termNodeId: string; alias: string; agg?: string }[];
  chartType?: 'bar' | 'line' | 'pie';
  limit?: number;
}

interface ReportWidgetRendererProps {
  type: string; // 'table' | 'matrix' | 'list' | 'chart' | 'sparkline' | 'gauge'
  binding: ReportWidgetBinding;
}

interface DrillTargetResponse {
  available: boolean;
  boId?: string;
  termNodeId?: string;
  isSameBO?: boolean;
}

interface DrillStep {
  /** The filter this step applies: the level drilled *from*, and the value clicked. */
  filter: { boId: string; termNodeId: string; value: unknown; label: string };
  /** The dimension now in view as a result of this step. */
  resultDim: { boId: string; termNodeId: string; alias: string };
}

/**
 * Renders a Report Builder canvas element against real, live data instead of
 * the static text-label stub every widget type previously fell back to.
 * Runs the same QueryDef contract Query Builder uses — no SQL is
 * constructed here, the backend resolves and executes it.
 *
 * Two click behaviors, both server-resolved rather than configured per
 * widget: if the clicked dimension's semantic term has a drillPath (set
 * once on the term in the semantic layer, inherited by every BO field bound
 * to it — see backend SemanticTermView.DrillPath), clicking drills into the
 * next level, adding the clicked value as a filter and swapping in the
 * drill target as the new dimension (possibly from a related BO, via the
 * same relationship graph the multi-BO query generator uses). Otherwise it
 * falls back to plain cross-filtering: the clicked value is published to
 * the shared store and every other widget bound to the same (boId, field)
 * picks it up on its next fetch.
 */
const ReportWidgetRenderer: React.FC<ReportWidgetRendererProps> = ({ type, binding }) => {
  const [result, setResult] = useState<QueryExecuteResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [terms, setTerms] = useState<SemanticTermView[]>([]);
  const [drillSteps, setDrillSteps] = useState<DrillStep[]>([]);
  const crossFilters = useCrossFilterStore((s) => s.filters);
  const setCrossFilter = useCrossFilterStore((s) => s.setFilter);
  const clearCrossFilter = useCrossFilterStore((s) => s.clearFilter);

  useEffect(() => {
    setDrillSteps([]);
    if (!binding.boId || !binding.bindingId) {
      setTerms([]);
      return;
    }
    let cancelled = false;
    fetchBOTerms(binding.boId, binding.bindingId)
      .then((t) => {
        if (!cancelled) setTerms(t);
      })
      .catch(() => {
        if (!cancelled) setTerms([]);
      });
    return () => {
      cancelled = true;
    };
  }, [binding.boId, binding.bindingId]);

  const primaryDimension = binding.dimensions?.[0];

  // The dimension actually queried: the deepest drill level if the user has
  // drilled in, otherwise the widget's configured primary dimension.
  const effectiveDim = drillSteps.length > 0
    ? drillSteps[drillSteps.length - 1].resultDim
    : primaryDimension
      ? { boId: binding.boId, termNodeId: primaryDimension.termNodeId, alias: primaryDimension.alias }
      : undefined;

  const queryDef: QueryDef | null = useMemo(() => {
    if (!binding.boId || !binding.bindingId || !binding.tenantId) return null;
    const dims = effectiveDim ? [{ termNodeId: effectiveDim.termNodeId, alias: effectiveDim.alias, boId: effectiveDim.boId }] : [];
    if (dims.length === 0 && (binding.measures?.length || 0) === 0) return null;

    const relatedBoIds = Array.from(new Set(dims.filter((d) => d.boId !== binding.boId).map((d) => d.boId)));

    const drillFilters = drillSteps.map((step) => ({
      termNodeId: step.filter.termNodeId,
      operator: 'eq' as const,
      value: step.filter.value as any,
      boId: step.filter.boId,
    }));
    // Cross-filter applies to whichever dimension is currently in play,
    // scoped by its own boId so it never leaks across unrelated fields.
    const crossFilterFilters = dims
      .map((d) => {
        const key = crossFilterKey(d.boId, d.termNodeId);
        if (!(key in crossFilters)) return null;
        return { termNodeId: d.termNodeId, operator: 'eq' as const, value: crossFilters[key] as any, boId: d.boId };
      })
      .filter((f): f is NonNullable<typeof f> => f !== null);

    return {
      context: { boId: binding.boId, bindingId: binding.bindingId, tenantId: binding.tenantId, relatedBoIds: relatedBoIds.length > 0 ? relatedBoIds : undefined },
      query: {
        dimensions: dims,
        measures: (binding.measures || []).map((m) => ({ ...m, agg: (m.agg as any) || 'NONE' })),
        filters: [...drillFilters, ...crossFilterFilters],
        limit: binding.limit || 200,
      },
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [binding.boId, binding.bindingId, binding.tenantId, JSON.stringify(effectiveDim), JSON.stringify(binding.measures), binding.limit, JSON.stringify(crossFilters), JSON.stringify(drillSteps)]);

  useEffect(() => {
    if (!queryDef) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    executeQuery(queryDef)
      .then((res) => {
        if (!cancelled) setResult(res);
      })
      .catch((err) => {
        if (!cancelled) setError(err?.message || 'Query failed');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queryDef]);

  const handlePointClick = async (value: unknown) => {
    if (!effectiveDim) return;

    const term = terms.find((t) => t.termNodeId === effectiveDim.termNodeId);
    const nextTermId = term?.drillPath?.[0];

    if (nextTermId) {
      try {
        const res = await apiFetch(
          `/api/relationships/bo/${encodeURIComponent(effectiveDim.boId)}/drill-target/${encodeURIComponent(nextTermId)}`,
          { headers: { 'X-Tenant-ID': binding.tenantId } }
        );
        const target: DrillTargetResponse = await res.json();
        if (target.available && target.boId && target.termNodeId) {
          const targetTerm = terms.find((t) => t.termNodeId === target.termNodeId);
          setDrillSteps((prev) => [
            ...prev,
            {
              filter: { boId: effectiveDim.boId, termNodeId: effectiveDim.termNodeId, value, label: effectiveDim.alias },
              resultDim: { boId: target.boId!, termNodeId: target.termNodeId!, alias: targetTerm?.displayName || target.termNodeId! },
            },
          ]);
          return;
        }
      } catch {
        // fall through to plain cross-filter
      }
    }

    setCrossFilter(crossFilterKey(effectiveDim.boId, effectiveDim.termNodeId), value);
  };

  // Truncate back to having applied `count` drill steps: 0 returns to the
  // widget's original primary dimension, k keeps the first k levels.
  const resetDrill = (count: number) => {
    setDrillSteps((prev) => prev.slice(0, count));
  };

  const breadcrumb = drillSteps.length > 0 && (
    <Breadcrumbs sx={{ fontSize: '0.65rem', px: 0.5, py: 0.25 }}>
      <Link component="button" variant="caption" onClick={() => resetDrill(0)}>
        {primaryDimension?.alias}
      </Link>
      {drillSteps.map((step, i) => (
        <Link key={i} component="button" variant="caption" onClick={() => resetDrill(i + 1)}>
          {String(step.filter.value)}
        </Link>
      ))}
    </Breadcrumbs>
  );

  if (!queryDef) {
    return (
      <Box sx={{ p: 1, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Typography variant="caption" color="text.secondary">
          Bind this {type} to a Business Object and fields in Properties
        </Typography>
      </Box>
    );
  }

  if (loading && !result) {
    return (
      <Box sx={{ p: 1, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <CircularProgress size={20} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 1 }}>
        <Alert severity="error" sx={{ fontSize: '0.7rem' }}>{error}</Alert>
      </Box>
    );
  }

  if (!result || result.rows.length === 0) {
    return (
      <Box sx={{ p: 1, height: '100%', display: 'flex', flexDirection: 'column' }}>
        {breadcrumb}
        <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Typography variant="caption" color="text.secondary">No data</Typography>
        </Box>
      </Box>
    );
  }

  if (type === 'chart' || type === 'sparkline') {
    const dimCol = result.columns[0]?.name;
    const measureCol = result.columns.find((c) => c.name !== dimCol)?.name || result.columns[1]?.name;
    const categories = result.rows.map((r) => String(r[dimCol] ?? ''));
    const values = result.rows.map((r) => Number(r[measureCol]) || 0);
    const chartType = type === 'sparkline' ? 'line' : binding.chartType || 'bar';

    const option =
      chartType === 'pie'
        ? {
            tooltip: { trigger: 'item' },
            series: [{ type: 'pie', radius: '65%', data: categories.map((c, i) => ({ name: c, value: values[i] })) }],
          }
        : {
            grid: { left: 40, right: 12, top: 12, bottom: type === 'sparkline' ? 12 : 30 },
            xAxis: type === 'sparkline' ? { show: false, type: 'category', data: categories } : { type: 'category', data: categories, axisLabel: { fontSize: 9, rotate: 30 } },
            yAxis: type === 'sparkline' ? { show: false } : { type: 'value' },
            tooltip: { trigger: 'axis' },
            series: [{ type: chartType, data: values, smooth: chartType === 'line' }],
          };

    return (
      <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        {breadcrumb}
        <Box sx={{ flex: 1 }}>
          <ReactECharts
            option={option}
            style={{ height: '100%', width: '100%' }}
            onEvents={{ click: (params: any) => handlePointClick(categories[params.dataIndex]) }}
          />
        </Box>
      </Box>
    );
  }

  if (type === 'slicer') {
    // A slicer never drills — it's an explicit filter control, not a
    // navigation control — so it always publishes straight to the shared
    // cross-filter store regardless of whether the bound term has a
    // drillPath.
    const dimCol = result.columns[0]?.name;
    const distinctValues = Array.from(new Set(result.rows.map((r) => r[dimCol]))).filter((v) => v !== null && v !== undefined);
    const activeKey = effectiveDim ? crossFilterKey(effectiveDim.boId, effectiveDim.termNodeId) : '';
    const activeValue = activeKey ? crossFilters[activeKey] : undefined;

    return (
      <Box sx={{ p: 1, height: '100%', overflowY: 'auto' }}>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
          {primaryDimension?.alias}
        </Typography>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
          {distinctValues.map((v, i) => (
            <Chip
              key={i}
              size="small"
              label={String(v)}
              color={activeValue === v ? 'primary' : 'default'}
              variant={activeValue === v ? 'filled' : 'outlined'}
              onClick={() => {
                if (!effectiveDim) return;
                if (activeValue === v) {
                  clearCrossFilter(activeKey);
                } else {
                  setCrossFilter(activeKey, v);
                }
              }}
            />
          ))}
        </Box>
      </Box>
    );
  }

  if (type === 'gauge') {
    const measureCol = result.columns[result.columns.length - 1]?.name;
    const value = Number(result.rows[0]?.[measureCol]) || 0;
    return (
      <Box sx={{ p: 1, height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
        <Typography variant="h4" fontWeight={700}>{value.toLocaleString()}</Typography>
        <Typography variant="caption" color="text.secondary">{measureCol}</Typography>
      </Box>
    );
  }

  if (type === 'matrix') {
    // Real cross-tab pivot: dimension 0 = rows, dimension 1 = columns,
    // remaining measure(s) fill cells. Falls back to a flat table when
    // fewer than 2 dimensions are bound — there's nothing to pivot on.
    const rowDim = binding.dimensions?.[0];
    const colDim = binding.dimensions?.[1];
    const measureCol = result.columns.find((c) => !binding.dimensions?.some((d) => d.alias === c.name))?.name;

    if (rowDim && colDim && measureCol) {
      const rowKeys: string[] = [];
      const colKeys: string[] = [];
      const cellMap = new Map<string, number>();
      for (const row of result.rows) {
        const rk = String(row[rowDim.alias] ?? '');
        const ck = String(row[colDim.alias] ?? '');
        if (!rowKeys.includes(rk)) rowKeys.push(rk);
        if (!colKeys.includes(ck)) colKeys.push(ck);
        cellMap.set(`${rk}|${ck}`, Number(row[measureCol]) || 0);
      }
      return (
        <TableContainer sx={{ height: '100%' }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold', fontSize: '0.7rem', bgcolor: '#f9f9f9' }}>{rowDim.alias}</TableCell>
                {colKeys.map((ck) => (
                  <TableCell key={ck} sx={{ fontWeight: 'bold', fontSize: '0.7rem', bgcolor: '#f9f9f9' }} align="right">{ck}</TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {rowKeys.map((rk) => (
                <TableRow key={rk} hover>
                  <TableCell sx={{ fontSize: '0.75rem', fontWeight: 600 }}>{rk}</TableCell>
                  {colKeys.map((ck) => (
                    <TableCell key={ck} align="right" sx={{ fontSize: '0.75rem' }}>
                      {(cellMap.get(`${rk}|${ck}`) ?? 0).toLocaleString()}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      );
    }
    // fall through to flat table below
  }

  // table / matrix (fallback) / list
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {breadcrumb}
      <TableContainer sx={{ flex: 1 }}>
        <Table stickyHeader size="small">
          <TableHead>
            <TableRow>
              {result.columns.map((col) => (
                <TableCell key={col.name} sx={{ fontWeight: 'bold', fontSize: '0.7rem', bgcolor: '#f9f9f9' }}>
                  {col.name}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {result.rows.map((row, i) => (
              <TableRow
                key={i}
                hover
                onClick={() => effectiveDim && handlePointClick(row[result.columns[0]?.name])}
                sx={{ cursor: effectiveDim ? 'pointer' : 'default' }}
              >
                {result.columns.map((col) => (
                  <TableCell key={col.name} sx={{ fontSize: '0.75rem' }}>
                    {typeof row[col.name] === 'object' ? JSON.stringify(row[col.name]) : String(row[col.name] ?? '-')}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default ReportWidgetRenderer;
