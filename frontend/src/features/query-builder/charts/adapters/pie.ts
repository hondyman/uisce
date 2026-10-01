import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { extractShelves } from './utils';

export function buildPieOption(results: QueryResultsData, config?: ChartConfig, isDonut = false): EChartsOption {
  const { dimCol, measureCols } = extractShelves(results, config);
  const rows = results.rows || [];

  if (!dimCol || rows.length === 0) {
    return {
      title: config?.title ? { text: config.title } : undefined,
      series: [],
    };
  }

  const dimKey = dimCol.name;
  const measureKey = measureCols[0]?.name || Object.keys(rows[0] || {}).find((k) => k !== dimKey) || '';

  const data = rows.map((r) => ({
    name: String(r[dimKey] ?? ''),
    value: Number(r[measureKey]) || 0,
    termNodeId: dimCol.termNodeId,
    memberValue: String(r[dimKey] ?? ''),
  }));

  return {
    title: config?.title ? { text: config.title } : undefined,
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { orient: 'horizontal', top: 0 },
    series: [
      {
        type: 'pie',
        radius: isDonut ? ['40%', '70%'] : '65%',
        center: ['50%', '55%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: isDonut ? 6 : 0,
          borderColor: '#fff',
          borderWidth: 2,
        },
        data,
      },
    ],
  };
}
