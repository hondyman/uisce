import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { extractShelves } from './utils';

export function buildScatterOption(results: QueryResultsData, config?: ChartConfig): EChartsOption {
  const { dimCol, measureCols } = extractShelves(results, config);
  const rows = results.rows || [];

  if (!dimCol || measureCols.length < 2 || rows.length === 0) {
    return {
      title: config?.title ? { text: config.title } : undefined,
      xAxis: { type: 'value' },
      yAxis: { type: 'value' },
      series: [],
    };
  }

  const dimKey = dimCol.name;
  const xMeasure = measureCols[0];
  const yMeasure = measureCols[1];
  const sizeMeasure = measureCols[2];

  const data = rows.map((r) => {
    const xVal = Number(r[xMeasure.name]) || 0;
    const yVal = Number(r[yMeasure.name]) || 0;
    const sizeVal = sizeMeasure ? Number(r[sizeMeasure.name]) || 10 : 10;
    const member = String(r[dimKey] ?? '');

    return {
      value: [xVal, yVal, sizeVal],
      name: member,
      termNodeId: dimCol.termNodeId,
      memberValue: member,
    };
  });

  return {
    title: config?.title ? { text: config.title } : undefined,
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        const d = params.data;
        return `<strong>${d.name}</strong><br/>${xMeasure.alias || xMeasure.name}: ${d.value[0]}<br/>${yMeasure.alias || yMeasure.name}: ${d.value[1]}`;
      },
    },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'value',
      name: xMeasure.alias || xMeasure.name,
      scale: true,
    },
    yAxis: {
      type: 'value',
      name: yMeasure.alias || yMeasure.name,
      scale: true,
    },
    series: [
      {
        type: 'scatter',
        symbolSize: (dataItem: any) => {
          if (!sizeMeasure) return 12;
          const val = dataItem[2];
          return Math.max(6, Math.min(val / 10, 40));
        },
        data,
      },
    ],
  };
}
