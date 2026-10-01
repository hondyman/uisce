import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { buildBarOption } from './bar';

export function buildStackedBarOption(results: QueryResultsData, config?: ChartConfig): EChartsOption {
  return buildBarOption(results, config, true);
}
