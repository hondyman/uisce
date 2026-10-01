import type { ChartType, ChartTypeSpec, ShelfValidationResult, QueryResultsData, ChartConfig } from './types';
import { buildBarOption } from './adapters/bar';
import { buildStackedBarOption } from './adapters/stackedBar';
import { buildLineOption } from './adapters/line';
import { buildAreaOption } from './adapters/area';
import { buildPieOption } from './adapters/pie';
import { buildDonutOption } from './adapters/donut';
import { buildScatterOption } from './adapters/scatter';
import { buildComboOption } from './adapters/combo';

function createValidator(
  label: string,
  minDim: number,
  maxDim: number,
  minMeas: number,
  maxMeas: number
): (dimCount: number, measCount: number) => ShelfValidationResult {
  return (dimCount: number, measCount: number): ShelfValidationResult => {
    if (dimCount < minDim) {
      return { valid: false, message: `${label} requires at least ${minDim} dimension${minDim > 1 ? 's' : ''}` };
    }
    if (dimCount > maxDim) {
      return { valid: false, message: `${label} supports at most ${maxDim} dimension${maxDim > 1 ? 's' : ''}` };
    }
    if (measCount < minMeas) {
      return { valid: false, message: `${label} requires at least ${minMeas} measure${minMeas > 1 ? 's' : ''}` };
    }
    if (measCount > maxMeas) {
      return { valid: false, message: `${label} supports at most ${maxMeas} measure${maxMeas > 1 ? 's' : ''}` };
    }
    return { valid: true };
  };
}

export const CHART_REGISTRY: Record<ChartType, ChartTypeSpec | undefined> = {
  bar: {
    id: 'bar',
    label: 'Bar Chart',
    icon: 'bar_chart',
    shelves: { dimensions: { min: 1, max: 2 }, measures: { min: 1, max: 3 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Bar chart', 1, 2, 1, 3),
    buildOption: (results, config) => buildBarOption(results, config, false),
  },
  stackedBar: {
    id: 'stackedBar',
    label: 'Stacked Bar',
    icon: 'stacked_bar_chart',
    shelves: { dimensions: { min: 1, max: 2 }, measures: { min: 1, max: 3 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Stacked bar', 1, 2, 1, 3),
    buildOption: (results, config) => buildStackedBarOption(results, config),
  },
  line: {
    id: 'line',
    label: 'Line Chart',
    icon: 'show_chart',
    shelves: { dimensions: { min: 1, max: 2 }, measures: { min: 1, max: 3 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Line chart', 1, 2, 1, 3),
    buildOption: (results, config) => buildLineOption(results, config, false),
  },
  area: {
    id: 'area',
    label: 'Area Chart',
    icon: 'area_chart',
    shelves: { dimensions: { min: 1, max: 2 }, measures: { min: 1, max: 3 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Area chart', 1, 2, 1, 3),
    buildOption: (results, config) => buildAreaOption(results, config),
  },
  pie: {
    id: 'pie',
    label: 'Pie Chart',
    icon: 'pie_chart',
    shelves: { dimensions: { min: 1, max: 1 }, measures: { min: 1, max: 1 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Pie chart', 1, 1, 1, 1),
    buildOption: (results, config) => buildPieOption(results, config, false),
  },
  donut: {
    id: 'donut',
    label: 'Donut Chart',
    icon: 'donut_large',
    shelves: { dimensions: { min: 1, max: 1 }, measures: { min: 1, max: 1 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Donut chart', 1, 1, 1, 1),
    buildOption: (results, config) => buildDonutOption(results, config),
  },
  scatter: {
    id: 'scatter',
    label: 'Scatter Plot',
    icon: 'scatter_plot',
    shelves: { dimensions: { min: 1, max: 1 }, measures: { min: 2, max: 3 } },
    interactions: { crossFilter: true, drillDown: false, drillThrough: true },
    validateShelves: createValidator('Scatter plot', 1, 1, 2, 3),
    buildOption: (results, config) => buildScatterOption(results, config),
  },
  combo: {
    id: 'combo',
    label: 'Combo (Bar & Line)',
    icon: 'multiline_chart',
    shelves: { dimensions: { min: 1, max: 1 }, measures: { min: 2, max: 3 } },
    interactions: { crossFilter: true, drillDown: true, drillThrough: true },
    validateShelves: createValidator('Combo chart', 1, 1, 2, 3),
    buildOption: (results, config) => buildComboOption(results, config),
  },
  table: undefined,
};

export const SUPPORTED_CHART_TYPES: ChartType[] = [
  'bar',
  'stackedBar',
  'line',
  'area',
  'pie',
  'donut',
  'scatter',
  'combo',
];

export function getChartSpec(chartType: string | undefined): ChartTypeSpec | undefined {
  if (!chartType) return undefined;
  return CHART_REGISTRY[chartType as ChartType];
}

export function buildOptionFromRegistry(
  chartType: string | undefined,
  results: QueryResultsData,
  config?: ChartConfig
) {
  const spec = getChartSpec(chartType);
  if (!spec) {
    return buildBarOption(results, config, false);
  }
  return spec.buildOption(results, config);
}
