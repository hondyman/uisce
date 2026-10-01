import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { extractShelves } from './utils';

export function buildLineOption(results: QueryResultsData, config?: ChartConfig, isArea = false): EChartsOption {
  const { dimCol, secDimCol, measureCols } = extractShelves(results, config);
  const rows = results.rows || [];

  if (!dimCol || rows.length === 0) {
    return {
      title: config?.title ? { text: config.title } : undefined,
      xAxis: { type: 'category', data: [] },
      yAxis: { type: 'value' },
      series: [],
    };
  }

  const dimKey = dimCol.name;

  // Case 1: 2 Dimensions -> Grouped line series per secondary dimension value
  if (secDimCol) {
    const secDimKey = secDimCol.name;
    const categories = Array.from(new Set(rows.map((r) => String(r[dimKey] ?? ''))));
    const secCategories = Array.from(new Set(rows.map((r) => String(r[secDimKey] ?? ''))));
    const primaryMeasure = measureCols[0]?.name || Object.keys(rows[0] || {}).find((k) => k !== dimKey && k !== secDimKey) || '';

    const series = secCategories.map((secVal) => {
      const data = categories.map((cat) => {
        const matchingRow = rows.find((r) => String(r[dimKey] ?? '') === cat && String(r[secDimKey] ?? '') === secVal);
        const val = matchingRow ? Number(matchingRow[primaryMeasure]) || 0 : 0;
        return {
          value: val,
          termNodeId: dimCol.termNodeId,
          memberValue: cat,
          secTermNodeId: secDimCol.termNodeId,
          secMemberValue: secVal,
        };
      });

      return {
        name: secVal,
        type: 'line' as const,
        smooth: config?.smooth ?? true,
        areaStyle: isArea ? { opacity: 0.3 } : undefined,
        stack: isArea ? 'total' : undefined,
        data,
      };
    });

    return {
      title: config?.title ? { text: config.title } : undefined,
      tooltip: { trigger: 'axis' },
      legend: { data: secCategories, top: 0 },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: { type: 'category', data: categories },
      yAxis: { type: 'value' },
      series,
    };
  }

  // Case 2: 1 Dimension + Multiple Measures -> Multi-line chart
  const categories = rows.map((r) => String(r[dimKey] ?? ''));
  const effectiveMeasures = measureCols.length > 0 ? measureCols : [{ name: 'value', termNodeId: 'value', alias: 'Value' }];

  const series = effectiveMeasures.map((m) => ({
    name: m.alias || m.name,
    type: 'line' as const,
    smooth: config?.smooth ?? true,
    areaStyle: isArea ? { opacity: 0.3 } : undefined,
    stack: isArea ? 'total' : undefined,
    data: rows.map((r) => ({
      value: Number(r[m.name]) || 0,
      termNodeId: dimCol.termNodeId,
      memberValue: String(r[dimKey] ?? ''),
      measureTermNodeId: m.termNodeId,
    })),
  }));

  return {
    title: config?.title ? { text: config.title } : undefined,
    tooltip: { trigger: 'axis' },
    legend: effectiveMeasures.length > 1 ? { data: effectiveMeasures.map((m) => m.alias || m.name), top: 0 } : undefined,
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: categories },
    yAxis: { type: 'value' },
    series,
  };
}
