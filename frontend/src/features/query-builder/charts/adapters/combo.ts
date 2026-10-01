import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { extractShelves } from './utils';

export function buildComboOption(results: QueryResultsData, config?: ChartConfig): EChartsOption {
  const { dimCol, measureCols } = extractShelves(results, config);
  const rows = results.rows || [];

  if (!dimCol || measureCols.length < 2 || rows.length === 0) {
    return {
      title: config?.title ? { text: config.title } : undefined,
      xAxis: { type: 'category', data: [] },
      yAxis: [{ type: 'value' }, { type: 'value' }],
      series: [],
    };
  }

  const dimKey = dimCol.name;
  const categories = rows.map((r) => String(r[dimKey] ?? ''));
  const comboTypes = config?.comboTypes || {};

  const series = measureCols.map((m, idx) => {
    const isLine = comboTypes[m.name] === 'line' || (comboTypes[m.name] === undefined && idx >= 1);
    const seriesType = isLine ? ('line' as const) : ('bar' as const);
    const yAxisIndex = isLine && idx > 0 ? 1 : 0;

    return {
      name: m.alias || m.name,
      type: seriesType,
      yAxisIndex,
      data: rows.map((r) => ({
        value: Number(r[m.name]) || 0,
        termNodeId: dimCol.termNodeId,
        memberValue: String(r[dimKey] ?? ''),
        measureTermNodeId: m.termNodeId,
      })),
    };
  });

  return {
    title: config?.title ? { text: config.title } : undefined,
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    legend: { data: measureCols.map((m) => m.alias || m.name), top: 0 },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: categories },
    yAxis: [
      {
        type: 'value',
        name: measureCols[0]?.alias || measureCols[0]?.name,
      },
      {
        type: 'value',
        name: measureCols[1]?.alias || measureCols[1]?.name,
      },
    ],
    series,
  };
}
