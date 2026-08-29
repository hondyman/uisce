export interface CellStyle {
  fontSize?: number;
  fontWeight?: string;
  fontFamily?: string;
  color?: string;
  backgroundColor?: string;
  textAlign?: 'left' | 'center' | 'right';
  verticalAlign?: 'top' | 'middle' | 'bottom';
  wrap?: boolean;
}

export interface ColumnConfig {
  id: string;
  field: string;
  headerText?: string;
  headerName?: string;
  width?: number;
  widthPx?: number;
  align?: 'left' | 'center' | 'right';
  verticalAlign?: 'top' | 'middle' | 'bottom';
  wrap?: boolean;
  visible?: boolean;
  formatType?: string;
  formatMask?: string;
  headerStyle?: CellStyle;
  bodyStyle?: CellStyle;
  aggregate?: string;
  sparkline?: Record<string, unknown>;
}

export interface TotalsConfig {
  grandTotal?: { enabled?: boolean };
  subtotals?: { enabled?: boolean };
}

export interface BandingConfig {
  rowBanding?: boolean;
  columnBanding?: boolean;
  gridlines?: boolean;
}

export interface PaginationConfig {
  enabled?: boolean;
  pageSize?: number;
}

export interface FreezePaneConfig {
  enabled?: boolean;
  rowCount?: number;
  columnCount?: number;
}

export interface TableElementProperties {
  name?: string;
  columns?: ColumnConfig[];
  totals?: TotalsConfig;
  banding?: BandingConfig;
  pagination?: PaginationConfig;
  freezePane?: FreezePaneConfig;
  conditionalRules?: unknown[];
  namedStyles?: unknown[];
}

export const DEFAULT_HEADER_STYLE: CellStyle = {
  fontWeight: 'bold',
  backgroundColor: 'transparent',
};

export const DEFAULT_CELL_STYLE: CellStyle = {
  fontSize: 12,
};

export function createDefaultColumnConfig(id: string, field: string): ColumnConfig {
  return {
    id,
    field,
    headerText: field,
    headerName: field,
    visible: true,
    align: 'left',
    verticalAlign: 'middle',
    wrap: false,
    headerStyle: { ...DEFAULT_HEADER_STYLE },
    bodyStyle: { ...DEFAULT_CELL_STYLE },
  };
}

export function createDefaultBandingConfig(): BandingConfig {
  return {
    rowBanding: false,
    columnBanding: false,
    gridlines: true,
  };
}

export function createDefaultTotalsConfig(): TotalsConfig {
  return {
    grandTotal: { enabled: true },
    subtotals: { enabled: false },
  };
}

export function createDefaultPaginationConfig(): PaginationConfig {
  return {
    enabled: false,
    pageSize: 25,
  };
}

export function createDefaultFreezePaneConfig(): FreezePaneConfig {
  return {
    enabled: false,
    rowCount: 0,
    columnCount: 0,
  };
}
