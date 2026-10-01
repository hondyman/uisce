import type { SavedQueryChartType } from '../types/queryDef';
import { buildOptionFromRegistry } from '../charts/registry';

/**
 * Builds an ECharts option from a flat rows/columns result - shared between
 * the query builder's own Chart tab preview and Page Studio's "use a saved
 * query" Chart/KPI rendering (PageComponentRenderer.tsx), so a saved
 * query's preview and its live embedded rendering never diverge.
 */
export function buildChartOption(
  rows: Record<string, unknown>[],
  columns: { name: string; termNodeId?: string; alias?: string; type?: string }[],
  chartType: SavedQueryChartType,
  dimCol?: string,
  measureCol?: string,
  secondaryDimCol?: string,
  measureCols?: string[],
  comboTypes?: Record<string, 'bar' | 'line'>
) {
  return buildOptionFromRegistry(
    chartType,
    { rows, columns },
    { dimCol, measureCol, secondaryDimCol, measureCols, comboTypes }
  );
}

