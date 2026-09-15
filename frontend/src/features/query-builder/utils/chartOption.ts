import type { SavedQueryChartType } from '../types/queryDef';

/**
 * Builds an ECharts option from a flat rows/columns result - shared between
 * the query builder's own Chart tab preview and Page Studio's "use a saved
 * query" Chart/KPI rendering (PageComponentRenderer.tsx), so a saved
 * query's preview and its live embedded rendering never diverge.
 */
export function buildChartOption(
  rows: Record<string, unknown>[],
  columns: { name: string }[],
  chartType: SavedQueryChartType,
  dimCol?: string,
  measureCol?: string
) {
  const dim = dimCol || columns[0]?.name;
  const measure = measureCol || columns.find((c) => c.name !== dim)?.name || columns[1]?.name;
  const categories = rows.map((r) => String(r[dim] ?? ''));
  const values = rows.map((r) => Number(r[measure]) || 0);

  if (chartType === 'pie') {
    return {
      tooltip: { trigger: 'item' },
      series: [{
        type: 'pie',
        radius: '65%',
        data: categories.map((c, i) => ({ name: c, value: values[i] })),
      }],
    };
  }
  return {
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: categories },
    yAxis: { type: 'value' },
    series: [{ type: chartType, data: values, smooth: chartType === 'line' }],
  };
}
