import type { EChartsOption } from 'echarts';

export type ChartType =
  | 'bar'
  | 'stackedBar'
  | 'line'
  | 'area'
  | 'pie'
  | 'donut'
  | 'scatter'
  | 'combo'
  | 'table';

export interface QueryColumnMeta {
  name: string;
  termNodeId?: string;
  alias?: string;
  type?: string;
}

export interface QueryResultsData {
  rows: Record<string, unknown>[];
  columns: QueryColumnMeta[];
  rowCount?: number;
}

export interface ChartConfig {
  dimCol?: string;
  secondaryDimCol?: string;
  measureCol?: string;
  measureCols?: string[];
  comboTypes?: Record<string, 'bar' | 'line'>;
  title?: string;
  smooth?: boolean;
}

export interface ShelfValidationResult {
  valid: boolean;
  message?: string;
}

export interface ChartTypeSpec {
  id: ChartType;
  label: string;
  icon: string;
  shelves: {
    dimensions: { min: number; max: number };
    measures: { min: number; max: number };
  };
  interactions: {
    crossFilter: boolean;
    drillDown: boolean;
    drillThrough: boolean;
  };
  validateShelves: (dimCount: number, measureCount: number) => ShelfValidationResult;
  buildOption: (results: QueryResultsData, config?: ChartConfig) => EChartsOption;
}

export const CHART_ROW_SOFT_CAP = 10000;
