import { DynamicProperty } from './evaluateDynamicProperty';

export interface ColumnConfig {
  id: string;
  field: string;
  headerText: string;
  headerTextDynamic?: DynamicProperty<string>;

  widthPx: number;
  minWidthPx?: number;
  visible: boolean;

  headerStyle: CellStyle;
  bodyStyle: CellStyle;

  align: 'left' | 'center' | 'right';
  verticalAlign: 'top' | 'middle' | 'bottom';
  wrap: boolean;

  formatType: FormatType;
  formatMask: string;
  formatPrefix: string;
  formatSuffix: string;

  aggregate?: AggregateConfig;
  conditionalOverrides?: string[];
  sparkline?: SparklineConfig;
}

export type FormatType = 'Auto' | 'Currency' | 'Percent' | 'Decimal' | 'Integer' | 'Date' | 'Text' | 'Custom';

export interface AggregateConfig {
  function: AggregateFunction | CustomAggregate;
  scope: 'column' | 'group' | 'report';
  enabled: boolean;
}

export type AggregateFunction = 'SUM' | 'AVG' | 'COUNT' | 'MIN' | 'MAX' | 'MEDIAN';

export interface CustomAggregate {
  customExpression: string;
}

export interface CellStyle {
  fontFamily?: string;
  fontSize?: number;
  fontWeight?: 300 | 400 | 500 | 600 | 700 | 800;
  fontStyle?: 'normal' | 'italic';
  textDecoration?: 'none' | 'underline' | 'line-through' | 'underline line-through';
  textTransform?: 'none' | 'uppercase' | 'lowercase' | 'capitalize';
  color?: string;
  backgroundColor?: string;
  borderTop?: BorderSide;
  borderRight?: BorderSide;
  borderBottom?: BorderSide;
  borderLeft?: BorderSide;
  paddingTop?: number;
  paddingRight?: number;
  paddingBottom?: number;
  paddingLeft?: number;
}

export interface BorderSide {
  width: number;
  style: 'none' | 'solid' | 'dashed' | 'dotted' | 'double';
  color: string;
}

export interface TotalsConfig {
  grandTotal: GrandTotalConfig;
  subtotals: SubtotalsConfig;
}

export interface GrandTotalConfig {
  enabled: boolean;
  position: 'top' | 'bottom';
  label: string;
}

export interface SubtotalsConfig {
  enabled: boolean;
  position: 'top' | 'bottom';
  label: string;
}

export interface BandingConfig {
  bandedRows: boolean;
  bandedColumns: boolean;
  bandColor: string;
  headerFill: string;
  headerTextColor: string;
  totalsFill: string;
  totalsTextColor: string;
  gridlines: GridlinesConfig;
}

export interface GridlinesConfig {
  horizontal: boolean;
  vertical: boolean;
  color: string;
  style: BorderSide['style'];
  width: number;
}

export interface FreezePaneConfig {
  frozenHeaderRows: number;
  frozenHeaderColumns: number;
  frozenTrailingRows: number;
  frozenTrailingColumns: number;
}

export interface PaginationConfig {
  mode: 'expand' | 'paginate';
  rowsPerPage: number;
  repeatHeadersOnEachPage: boolean;
  pageTotalEnabled: boolean;
  pageTotalPosition: 'top' | 'bottom';
  pageTotalLabel: string;
}

export interface ConditionalRule {
  id: string;
  name: string;
  appliesTo: 'all' | string[];
  type: 'colorScale' | 'dataBar' | 'iconSet' | 'expression';
  config: ColorScaleConfig | DataBarConfig | IconSetConfig | ExpressionConfig;
  precedence: number;
}

export interface ColorScaleConfig {
  minColor: string;
  midColor: string;
  maxColor: string;
  minValue?: number;
  maxValue?: number;
}

export interface DataBarConfig {
  color: string;
  showValue: boolean;
  axisColor?: string;
}

export interface IconSetConfig {
  iconSet: 'threeArrows' | 'threeArrowsGray' | 'threeSigns' | 'threeSymbols' | 'threeTrafficLights' | 'fourArrows' | 'fourRatings' | 'fiveArrows' | 'fiveRatings';
  reverse: boolean;
  showIconOnly: boolean;
}

export interface ExpressionConfig {
  expression: string;
  style: CellStyle;
}

export interface NamedStyle {
  id: string;
  name: string;
  cellStyle: CellStyle;
  scope: 'header' | 'body' | 'totals';
}

export interface SparklineConfig {
  type: 'line' | 'bar' | 'win-loss';
  color: string;
  highColor?: string;
  lowColor?: string;
  negativeColor?: string;
}

export interface TableElementProperties {
  name: string;
  dataSource?: string;
  columns: ColumnConfig[];
  totals?: TotalsConfig;
  banding?: BandingConfig;
  freezePane?: FreezePaneConfig;
  pagination?: PaginationConfig;
  conditionalRules?: ConditionalRule[];
  namedStyles?: NamedStyle[];
}

export function createDefaultColumnConfig(id: string, field: string): ColumnConfig {
  return {
    id,
    field,
    headerText: field,
    widthPx: 120,
    visible: true,
    headerStyle: {},
    bodyStyle: {},
    align: 'left',
    verticalAlign: 'middle',
    wrap: false,
    formatType: 'Auto',
    formatMask: '',
    formatPrefix: '',
    formatSuffix: '',
    aggregate: { enabled: false, function: 'SUM', scope: 'column' },
  };
}

export function createDefaultBandingConfig(): BandingConfig {
  return {
    bandedRows: true,
    bandedColumns: false,
    bandColor: 'rgba(0,0,0,0.04)',
    headerFill: '#071526',
    headerTextColor: '#E2E8F0',
    totalsFill: 'rgba(0,212,255,0.08)',
    totalsTextColor: '#00D4FF',
    gridlines: {
      horizontal: true,
      vertical: true,
      color: 'rgba(255,255,255,0.08)',
      style: 'solid',
      width: 1,
    },
  };
}

export function createDefaultTotalsConfig(): TotalsConfig {
  return {
    grandTotal: {
      enabled: true,
      position: 'bottom',
      label: 'Grand Total',
    },
    subtotals: {
      enabled: false,
      position: 'bottom',
      label: 'Total {groupValue}',
    },
  };
}

export function createDefaultFreezePaneConfig(): FreezePaneConfig {
  return {
    frozenHeaderRows: 1,
    frozenHeaderColumns: 0,
    frozenTrailingRows: 0,
    frozenTrailingColumns: 0,
  };
}

export function createDefaultPaginationConfig(): PaginationConfig {
  return {
    mode: 'expand',
    rowsPerPage: 20,
    repeatHeadersOnEachPage: true,
    pageTotalEnabled: false,
    pageTotalPosition: 'bottom',
    pageTotalLabel: 'Page Total',
  };
}

export const DEFAULT_CELL_STYLE: CellStyle = {
  fontFamily: 'Calibri',
  fontSize: 11,
  fontWeight: 400,
  fontStyle: 'normal',
  textDecoration: 'none',
  textTransform: 'none',
  color: '#E2E8F0',
  backgroundColor: 'transparent',
  paddingTop: 4,
  paddingRight: 8,
  paddingBottom: 4,
  paddingLeft: 8,
};

export const DEFAULT_HEADER_STYLE: CellStyle = {
  fontFamily: 'Calibri',
  fontSize: 11,
  fontWeight: 700,
  fontStyle: 'normal',
  textDecoration: 'none',
  textTransform: 'none',
  color: '#E2E8F0',
  backgroundColor: '#071526',
  paddingTop: 6,
  paddingRight: 8,
  paddingBottom: 6,
  paddingLeft: 8,
};
