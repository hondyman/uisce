import jsPDF from 'jspdf';
import 'jspdf-autotable';
import DOMPurify from 'dompurify';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';

// Type definitions
export const ELEMENT_TYPES = {
  TEXTBOX: 'textbox',
  TABLE: 'table',
  MATRIX: 'matrix',
  LIST: 'list',
  CHART: 'chart',
  IMAGE: 'image',
  SUBREPORT: 'subreport',
  RECTANGLE: 'rectangle',
  LINE: 'line',
  PARAMETER: 'parameter',
  GAUGE: 'gauge',
  SPARKLINE: 'sparkline',
  SLICER: 'slicer',
  FORM: 'form',
} as const;

export const REPORT_SECTIONS = {
  REPORT_HEADER: 'reportHeader',
  PAGE_HEADER: 'pageHeader',
  BODY: 'body',
  PAGE_FOOTER: 'pageFooter',
  REPORT_FOOTER: 'reportFooter',
} as const;

export type SectionName = typeof REPORT_SECTIONS[keyof typeof REPORT_SECTIONS];

export interface ReportElement {
  id: string;
  type: typeof ELEMENT_TYPES[keyof typeof ELEMENT_TYPES];
  section: SectionName;
  position: { x: number; y: number };
  size: { width: number; height: number };
  properties: Record<string, any>;
}

export interface AggregateDefinition {
  id: string;
  field: string;
  function: keyof typeof aggregateHandlers;
  scope: 'Group' | 'Report';
  displayName?: string;
}

export interface GroupDefinition {
  id: string;
  name: string;
  expression: string;
  parent?: string | null;
  aggregates: AggregateDefinition[];
  pageBreakBefore?: boolean;
  pageBreakAfter?: boolean;
}

export interface CalculatedField {
  id: string;
  name: string;
  expression: string;
  datasetId: string;
  format?: string;
}

export interface LayoutSettings {
  pageBreakBeforeGroup: boolean;
  pageBreakAfterGroup: boolean;
  pageBreakBetweenRegions: boolean;
  fixedPageSize: boolean;
  columns: number;
  columnSpacing: number;
  headerTokens: string[];
  footerTokens: string[];
  includeExecutionTime: boolean;
  includeUserName: boolean;
}

export interface ExportOptions {
  includePrintFriendly: boolean;
  includeDrillThrough: boolean;
  includeComments: boolean;
}

export const DRAG_TYPES = {
  REPORT_ELEMENT: 'reportElement',
} as const;

// Sample/static data moved here
export const dataSources = [
  {
    id: 'ds_primary',
    name: 'Primary Warehouse',
    type: 'SQL Server',
    connectionString: 'Server=sql.prod;Database=Warehouse;Trusted_Connection=True;',
  },
  {
    id: 'ds_marketing',
    name: 'Marketing Lakehouse',
    type: 'Fabric Lake',
    url: 'https://lakehouse.fabric.microsoft.com/marketing',
  },
];

export const datasets = [
  {
    id: 'dataset_sales',
    name: 'SalesSummary',
    dataSourceId: 'ds_primary',
    fields: [
      { name: 'Region', type: 'string' },
      { name: 'Manager', type: 'string' },
      { name: 'Sales', type: 'number' },
      { name: 'Growth', type: 'number' },
      { name: 'Quota', type: 'number' },
    ],
  },
  {
    id: 'dataset_margin',
    name: 'MarginDetail',
    dataSourceId: 'ds_marketing',
    fields: [
      { name: 'Category', type: 'string' },
      { name: 'Region', type: 'string' },
      { name: 'Revenue', type: 'number' },
      { name: 'Cost', type: 'number' },
      { name: 'Margin', type: 'number' },
    ],
  },
] as const;

export const reportParameters = [
  { name: 'StartDate', type: 'Date', prompt: 'Start Date', defaultValue: '2024-01-01' },
  { name: 'EndDate', type: 'Date', prompt: 'End Date', defaultValue: '2024-12-31' },
  { name: 'Region', type: 'String', prompt: 'Region', defaultValue: 'All' },
] as const;

export const sampleDetailData = [
  { region: 'North', sales: 125000, growth: 0.12, manager: 'Alex Johnson', quota: 110000 },
  { region: 'South', sales: 98500, growth: 0.08, manager: 'Priya Patel', quota: 102000 },
  { region: 'East', sales: 156000, growth: 0.15, manager: 'Michael Chen', quota: 150000 },
  { region: 'West', sales: 112000, growth: 0.05, manager: 'Laura Smith', quota: 120000 },
  { region: 'Europe', sales: 178500, growth: 0.18, manager: 'Markus Keller', quota: 165000 },
  { region: 'APAC', sales: 162750, growth: -0.03, manager: 'Yuki Nakamura', quota: 170000 },
] as const;

export const sampleMatrixData = [
  { category: 'Software', region: 'North', sales: 42000 },
  { category: 'Software', region: 'South', sales: 38000 },
  { category: 'Software', region: 'East', sales: 52000 },
  { category: 'Software', region: 'West', sales: 41000 },
  { category: 'Hardware', region: 'North', sales: 31000 },
  { category: 'Hardware', region: 'South', sales: 29500 },
  { category: 'Hardware', region: 'East', sales: 33500 },
  { category: 'Hardware', region: 'West', sales: 29000 },
  { category: 'Services', region: 'North', sales: 18000 },
  { category: 'Services', region: 'South', sales: 22500 },
  { category: 'Services', region: 'East', sales: 19800 },
  { category: 'Services', region: 'West', sales: 18400 },
] as const;

export const currencyFormatter = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

export const percentFormatter = new Intl.NumberFormat('en-US', {
  style: 'percent',
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

export const aggregateHandlers: Record<string, (values: number[]) => number> = {
  SUM: (values) => values.reduce((acc, value) => acc + value, 0),
  AVG: (values) => (values.length ? values.reduce((acc, value) => acc + value, 0) / values.length : 0),
  COUNT: (values) => values.length,
  MIN: (values) => (values.length ? Math.min(...values) : 0),
  MAX: (values) => (values.length ? Math.max(...values) : 0),
};

export const computeAggregate = (rows: Record<string, unknown>[], field: string, fn: keyof typeof aggregateHandlers) => {
  const numericValues = rows
    .map((row) => Number(row[field as keyof typeof row]))
    .filter((value) => Number.isFinite(value));
  return aggregateHandlers[fn]?.(numericValues) ?? 0;
};

export const formatValue = (fieldKey: string, value: unknown) => {
  if (value === null || value === undefined) {
    return '';
  }
  const normalizedKey = fieldKey.toLowerCase();
  if (typeof value === 'number') {
    if (normalizedKey.includes('sales') || normalizedKey.includes('amount') || normalizedKey.includes('revenue') || normalizedKey.includes('quota')) {
      return currencyFormatter.format(value);
    }
    if (normalizedKey.includes('growth') || normalizedKey.includes('margin') || normalizedKey.includes('percent')) {
      return percentFormatter.format(value);
    }
    return value.toLocaleString();
  }
  return String(value);
};

export const buildMatrix = (data: typeof sampleMatrixData) => {
  const rowKeys = Array.from(new Set(data.map((item) => item.category)));
  const columnKeys = Array.from(new Set(data.map((item) => item.region)));
  const matrix: Record<string, Record<string, number>> = {};
  rowKeys.forEach((rowKey) => {
    matrix[rowKey] = {};
    columnKeys.forEach((colKey) => {
      const match = data.find((item) => item.category === rowKey && item.region === colKey);
      matrix[rowKey][colKey] = match?.sales ?? 0;
    });
  });
  return { rowKeys, columnKeys, matrix };
};

export const dynamicTokens = [
  { key: '{PageNumber}', label: 'Page Number' },
  { key: '{TotalPages}', label: 'Total Pages' },
  { key: '{ExecutionTime}', label: 'Execution Time' },
  { key: '{UserName}', label: 'User Name' },
] as const;

export const exportFormatLabels: Record<keyof ExportOptions, string> = {
  includePrintFriendly: 'Print-friendly PDF',
  includeDrillThrough: 'Excel with drill-through',
  includeComments: 'Word with review comments',
};

export const exportOptionDescriptions: Record<keyof ExportOptions, string> = {
  includePrintFriendly: 'Generate a paginated, print-optimized PDF rendition.',
  includeDrillThrough: 'Preserve drill-through actions when exporting to Excel.',
  includeComments: 'Include comment threads for collaborative document review.',
};

/**
 * Dropping a BO field onto the canvas used to always produce the same
 * plain text box regardless of the field's actual type - a date and a
 * dollar amount landed identically formatted. This maps a field's
 * data_type (business_object_fields.data_type: 'datetime'/'date',
 * 'number'/'integer'/'numeric', 'boolean', everything else 'text') to a
 * sensible default presentation: a Format label (shown in Properties and
 * meant to drive value formatting once live evaluation reads it), a
 * natural alignment (numbers/dates right-aligned, like every spreadsheet
 * and report tool does), a wider default box for numbers/dates, and a
 * placeholder sample so the canvas isn't just "[FieldName]" for every type.
 */
export interface FieldWidgetPreset {
  format: 'Date' | 'Number' | 'Boolean' | 'Text';
  textAlign: 'left' | 'right' | 'center';
  width: number;
  sampleText: (label: string) => string;
}

export const defaultFieldWidgetPreset = (dataType?: string): FieldWidgetPreset => {
  const t = (dataType || '').toLowerCase();
  if (t.includes('date') || t.includes('time')) {
    return { format: 'Date', textAlign: 'right', width: 160, sampleText: (l) => `[${l}: MM/DD/YYYY]` };
  }
  if (t.includes('number') || t.includes('numeric') || t.includes('integer') || t.includes('decimal') || t.includes('float')) {
    return { format: 'Number', textAlign: 'right', width: 140, sampleText: (l) => `[${l}: #,##0.00]` };
  }
  if (t.includes('bool')) {
    return { format: 'Boolean', textAlign: 'center', width: 100, sampleText: (l) => `[${l}: Yes/No]` };
  }
  return { format: 'Text', textAlign: 'left', width: 220, sampleText: (l) => `[${l}]` };
};

/**
 * Builds a data-bound element's `dataBinding` (boId/bindingId/tenantId
 * plus per-type dimensions/measures), extracted from
 * SSRSReportBuilder.tsx's handleAddToolboxItem so the manual toolbox-drag
 * path and the AI report-generation feature can't silently drift on what
 * a widget's default binding looks like - both call this with a
 * DIMENSION/MEASURE-split field list, the manual path passing every
 * field the BO has, the AI path passing only the fields it already
 * chose. Per-type shape mirrors handleAddToolboxItem's original inline
 * logic exactly: slicer takes one dimension, gauge takes one
 * measure-or-dimension, chart/sparkline take one of each, table/matrix/
 * list take up to 4 dimensions + 2 measures, form takes none (it binds
 * the whole record via boId/tenantId alone).
 */
export function buildDataBindingForType(
  type: string,
  boId: string,
  bindingId: string,
  tenantId: string,
  dims: SemanticTermView[],
  measures: SemanticTermView[],
): Record<string, any> {
  if (type === ELEMENT_TYPES.FORM) {
    return { boId, tenantId };
  }
  const base = { boId, bindingId, tenantId };
  if (type === ELEMENT_TYPES.SLICER) {
    const d = dims[0];
    return d ? { ...base, dimensions: [{ termNodeId: d.termNodeId, alias: d.displayName }] } : {};
  }
  if (type === ELEMENT_TYPES.GAUGE) {
    const m = measures[0] || dims[0];
    return m ? { ...base, measures: [{ termNodeId: m.termNodeId, alias: m.displayName, agg: measures[0] ? 'SUM' : 'COUNT' }] } : {};
  }
  if (type === ELEMENT_TYPES.CHART || type === ELEMENT_TYPES.SPARKLINE) {
    const d = dims[0];
    const m = measures[0];
    if (!d || !m) return {};
    return {
      ...base,
      dimensions: [{ termNodeId: d.termNodeId, alias: d.displayName }],
      measures: [{ termNodeId: m.termNodeId, alias: m.displayName, agg: 'SUM' }],
      chartType: 'bar',
    };
  }
  // table / matrix / list: first few dimensions + measures as columns
  const picked = [...dims.slice(0, 4), ...measures.slice(0, 2)];
  if (picked.length === 0) return {};
  return {
    ...base,
    dimensions: dims.slice(0, 4).map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName })),
    measures: measures.slice(0, 2).map((t) => ({ termNodeId: t.termNodeId, alias: t.displayName, agg: 'SUM' })),
  };
}

// Sanitization function
export const sanitizeInput = (value: string): string => {
  return DOMPurify.sanitize(value, {
    ALLOWED_TAGS: [],
    ALLOWED_ATTR: [],
  });
};

// Pixel-Perfect PDF Export Function
export const generatePixelPerfectPDF = (elements: ReportElement[], layoutSettings: LayoutSettings) => {
  const doc = new jsPDF({
    orientation: layoutSettings.fixedPageSize ? 'portrait' : 'landscape',
    unit: 'pt',
    format: 'a4',
  });

  let yPosition = 40; // Start after header

  elements.forEach((element) => {
    if (element.position.y > yPosition) {
      yPosition = element.position.y;
    }

    switch (element.type) {
      case ELEMENT_TYPES.TEXTBOX: {
        const allowedAligns = ['left', 'center', 'right', 'justify'] as const;
        type Align = typeof allowedAligns[number];
        const rawAlign = (element.properties.textAlign as string) || 'left';
        const alignValue: Align = (allowedAligns.includes(rawAlign as Align) ? (rawAlign as Align) : 'left');

        doc.text(
          (element.properties.text as string) || 'Sample Text',
          element.position.x,
          yPosition + 10,
          { align: alignValue }
        );
        break;
      }
      case ELEMENT_TYPES.TABLE:
        const columns = (element.properties.columns as string[]) || [];
        const rows = (element.properties.previewData as any[]) || Array.from(sampleDetailData).map(row => Object.values(row));
        (doc as any).autoTable({
          head: [columns],
          body: rows,
          startY: yPosition,
          theme: 'grid',
          styles: { fontSize: 8, cellPadding: 3 },
          headStyles: { fillColor: [99, 102, 241] },
        });
        yPosition = (doc as any).lastAutoTable.finalY + 10;
        break;
      default:
        doc.rect(element.position.x, yPosition, element.size.width, element.size.height);
    }
  });

  doc.save('pixel-perfect-report.pdf');
};
