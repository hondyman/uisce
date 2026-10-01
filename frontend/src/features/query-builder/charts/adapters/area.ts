import type { EChartsOption } from 'echarts';
import type { QueryResultsData, ChartConfig } from '../types';
import { buildLineOption } from './line';

export function buildAreaOption(results: QueryResultsData, config?: ChartConfig): EChartsOption {
  return buildLineOption(results, config, true);
}
