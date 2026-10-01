import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { buildPieOption } from './pie';

export function buildDonutOption(results: QueryResultsData, config?: ChartConfig): EChartsOption {
  return buildPieOption(results, config, true);
}
