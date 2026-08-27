import React, { useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Alert,
  Grid,
} from '@mui/material';
import type {
  ExplorerField,
  ExplorerResult,
  ExplorerSource,
  ViewMode,
} from '../types/dataExplorerTypes';
import LazyECharts from '../../../components/LazyECharts';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface VisualizationPaneProps {
  source: ExplorerSource;
  result: ExplorerResult | null;
  mode: ViewMode;
}

function inferColumnType(
  columnName: string,
  rows: Record<string, unknown>[]
): ExplorerField['type'] {
  for (const row of rows.slice(0, 20)) {
    const v = row[columnName];
    if (v === null || v === undefined) continue;
    if (typeof v === 'number') return 'number';
    if (typeof v === 'boolean') return 'boolean';
    if (v instanceof Date) return 'date';
    if (typeof v === 'string') {
      const num = Number(v);
      if (!Number.isNaN(num) && /^-?\d/.test(v.trim())) return 'number';
      const ts = Date.parse(v);
      if (!Number.isNaN(ts) && /\d{4}-\d{2}-\d{2}/.test(v)) return 'date';
      return 'string';
    }
  }
  return 'unknown';
}

interface AxisSelection {
  dimensionColumn: string | null;
  measureColumns: string[];
}

function pickAxes(source: ExplorerSource, result: ExplorerResult | null): AxisSelection {
  if (!result || result.rows.length === 0) {
    return { dimensionColumn: null, measureColumns: [] };
  }
  const orderedColumns = result.columns.map((c) => c.name);
  const dimFromState = source.fields.find((f) => f.category === 'dimension')?.name;
  const measureFromState = source.fields.find((f) => f.category === 'measure')?.name;
  const fallbackDim =
    orderedColumns.find((c) => {
      const t = inferColumnType(c, result.rows);
      return t === 'string' || t === 'date';
    }) || orderedColumns[0];
  const dimColumn = (dimFromState && orderedColumns.includes(dimFromState) ? dimFromState : fallbackDim) ?? null;
  const measureColumns = orderedColumns.filter((c) => c !== dimColumn && inferColumnType(c, result.rows) === 'number');
  if (measureColumns.length === 0 && measureFromState && orderedColumns.includes(measureFromState)) {
    return { dimensionColumn: dimColumn, measureColumns: [measureFromState] };
  }
  if (measureColumns.length === 0) {
    const numeric = orderedColumns
      .filter((c) => c !== dimColumn)
      .filter((c) => inferColumnType(c, result.rows) === 'number');
    return { dimensionColumn: dimColumn, measureColumns: numeric.slice(0, 1) };
  }
  return { dimensionColumn: dimColumn, measureColumns: measureColumns.slice(0, 4) };
}

function formatNumber(n: number): string {
  if (Number.isInteger(n)) return n.toLocaleString();
  const abs = Math.abs(n);
  if (abs >= 1000) return n.toLocaleString(undefined, { maximumFractionDigits: 1 });
  return Number(n.toFixed(2)).toString();
}

function buildOption(
  mode: ViewMode,
  xField: string,
  yFields: string[],
  rows: Record<string, unknown>[],
  chartPalette: string[],
  textColor: string,
  borderColor: string,
  bgColor: string
): any {
  const categories = rows.map((r) => {
    const v = r[xField];
    if (v instanceof Date) return v.toISOString().slice(0, 10);
    if (v === null || v === undefined) return '—';
    return String(v);
  });

  if (mode === 'pie') {
    if (yFields.length === 0) {
      return {
        tooltip: { trigger: 'item' },
        series: [],
      };
    }
    const y = yFields[0];
    return {
      tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
      legend: { orient: 'vertical', left: 'left', top: 'middle', textStyle: { color: textColor } },
      series: [
        {
          name: y,
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['60%', '50%'],
          avoidLabelOverlap: true,
          itemStyle: { borderRadius: 4, borderColor: bgColor, borderWidth: 2 },
          label: { color: textColor, formatter: '{b}: {d}%' },
          data: rows.slice(0, 50).map((row, i) => ({
            name: String(row[xField] ?? '—'),
            value: Number(row[y]) || 0,
            itemStyle: { color: chartPalette[i % chartPalette.length] },
          })),
        },
      ],
    };
  }

  const series = yFields.map((y, i) => ({
    name: y,
    type:
      mode === 'line'
        ? 'line'
        : mode === 'area'
          ? 'line'
          : mode === 'scatter'
            ? 'scatter'
            : 'bar',
    smooth: mode === 'area' || mode === 'line',
    symbolSize: 8,
    areaStyle: mode === 'area' ? { opacity: 0.25 } : undefined,
    itemStyle: { color: chartPalette[i % chartPalette.length] },
    data: rows.map((row) => {
      const v = Number(row[y]);
      return Number.isFinite(v) ? v : 0;
    }),
  }));

  return {
    tooltip: {
      trigger: 'axis',
      valueFormatter: (val: unknown) => (typeof val === 'number' ? formatNumber(val) : String(val)),
    },
    legend: {
      show: series.length > 1,
      top: 0,
      textStyle: { color: textColor },
    },
    grid: { top: series.length > 1 ? 32 : 16, left: 64, right: 24, bottom: 64, containLabel: true },
    xAxis: {
      type: 'category',
      data: categories.slice(0, 50),
      axisLabel: { color: textColor, rotate: 30 },
      axisLine: { lineStyle: { color: borderColor } },
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: textColor,
        formatter: (v: number) => formatNumber(v),
      },
      splitLine: { lineStyle: { color: borderColor } },
    },
    series,
  };
}

function KpiGrid({
  measureColumns,
  rows,
  theme,
}: {
  measureColumns: string[];
  rows: Record<string, unknown>[];
  theme: ReturnType<typeof useExplorerTheme>;
}) {
  if (measureColumns.length === 0) return null;
  return (
    <Grid container spacing={2}>
      {measureColumns.map((col, i) => {
        const numeric = rows
          .map((r) => Number(r[col]))
          .filter((n) => Number.isFinite(n));
        if (numeric.length === 0) return null;
        const sum = numeric.reduce((a, b) => a + b, 0);
        const avg = sum / numeric.length;
        const max = Math.max(...numeric);
        const min = Math.min(...numeric);
        const card = (label: string, value: string) => (
          <Grid size={{ xs: 12, sm: 6, md: 3 }} key={`${col}-${label}`}>
            <Paper
              elevation={0}
              sx={{
                p: 2.5,
                borderRadius: 3,
                border: `1px solid ${theme.border}`,
                bgcolor: theme.backgroundElevated,
                position: 'relative',
                overflow: 'hidden',
              }}
            >
              <Box
                sx={{
                  position: 'absolute',
                  top: 0,
                  right: 0,
                  width: 64,
                  height: 64,
                  bgcolor: theme.background,
                  borderRadius: '50%',
                  transform: 'translate(40%, -40%)',
                  opacity: 0.6,
                }}
              />
              <Typography variant="overline" sx={{ color: theme.textMuted, fontWeight: 700, letterSpacing: 1 }}>
                {label} {col}
              </Typography>
              <Typography variant="h4" fontWeight={800} sx={{ color: theme.text, mt: 1 }}>
                {value}
              </Typography>
            </Paper>
          </Grid>
        );
        return (
          <React.Fragment key={col + i}>
            {card('Sum', formatNumber(sum))}
            {card('Avg', formatNumber(avg))}
            {card('Max', formatNumber(max))}
            {card('Min', formatNumber(min))}
          </React.Fragment>
        );
      })}
    </Grid>
  );
}

export const VisualizationPane: React.FC<VisualizationPaneProps> = ({
  source,
  result,
  mode,
}) => {
  const theme = useExplorerTheme();
  const axes = useMemo(() => pickAxes(source, result), [source, result]);

  if (!result || result.rows.length === 0) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: theme.background,
          p: 6,
        }}
      >
        <Box sx={{ textAlign: 'center', maxWidth: 480 }}>
          <Typography variant="h6" fontWeight={700} sx={{ color: theme.text, mb: 1 }}>
            No data to visualize
          </Typography>
          <Typography variant="body2" sx={{ color: theme.textMuted }}>
            Run the query to render {mode} charts.
          </Typography>
        </Box>
      </Box>
    );
  }

  if (mode === 'kpi') {
    return (
      <Box sx={{ flex: 1, overflowY: 'auto', p: 3, bgcolor: theme.background }}>
        <KpiGrid measureColumns={axes.measureColumns} rows={result.rows} theme={theme} />
      </Box>
    );
  }

  if (!axes.dimensionColumn || axes.measureColumns.length === 0) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="info">
          Could not auto-detect dim/measure axes. Add at least one dimension and one numeric measure.
        </Alert>
      </Box>
    );
  }

  const option = buildOption(
    mode,
    axes.dimensionColumn,
    axes.measureColumns,
    result.rows,
    theme.chartPalette,
    theme.text,
    theme.border,
    theme.background
  );

  return (
    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', bgcolor: theme.background, p: 3 }}>
      <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 2 }}>
        <Box
          sx={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            bgcolor: theme.accent,
          }}
        />
        <Typography variant="caption" sx={{ color: theme.textMuted, fontWeight: 700, letterSpacing: 1, textTransform: 'uppercase' }}>
          Visualization · {mode}
        </Typography>
        <Typography variant="caption" sx={{ color: theme.textMuted, ml: 2 }}>
          x: <strong style={{ color: theme.text }}>{axes.dimensionColumn}</strong> · y:{' '}
          <strong style={{ color: theme.text }}>{axes.measureColumns.join(', ')}</strong>
        </Typography>
      </Stack>
      <Paper
        elevation={0}
        sx={{
          flex: 1,
          minHeight: 380,
          p: 2,
          borderRadius: 3,
          border: `1px solid ${theme.border}`,
          bgcolor: theme.backgroundElevated,
        }}
      >
        <LazyECharts
          option={option}
          style={{ height: '100%', width: '100%' }}
        />
      </Paper>
    </Box>
  );
};

export default VisualizationPane;
