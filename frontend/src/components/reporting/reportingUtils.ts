import jsPDF from 'jspdf';
import 'jspdf-autotable';
import DOMPurify from 'dompurify';

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
  FORM_REFERENCE: 'formReference',
} as const;

export const REPORT_SECTIONS = {
  REPORT_HEADER: 'reportHeader',
  PAGE_HEADER: 'pageHeader',
  GROUP_HEADER: 'groupHeader',
  BODY: 'body',
  GROUP_FOOTER: 'groupFooter',
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

export interface EventScripts {
  onRowRender: string;
  onCellRender: string;
  onPageRender: string;
  onExport: string;
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
  MEDIAN: (values) => {
    if (!values.length) return 0;
    const sorted = [...values].sort((a, b) => a - b);
    const mid = Math.floor(sorted.length / 2);
    return sorted.length % 2 !== 0
      ? sorted[mid]
      : (sorted[mid - 1] + sorted[mid]) / 2;
  },
};

export const computeAggregate = (
  rows: Record<string, unknown>[],
  field: string,
  fn: keyof typeof aggregateHandlers | { customExpression: string }
) => {
  const numericValues = rows
    .map((row) => Number(row[field as keyof typeof row]))
    .filter((value) => Number.isFinite(value));

  if (typeof fn === 'object' && 'customExpression' in fn) {
    const { evaluateCustomAggregate } = require('./evaluateDynamicProperty');
    return evaluateCustomAggregate(rows, fn.customExpression, { _field: field });
  }

  return aggregateHandlers[fn]?.(numericValues) ?? 0;
};

export const formatReportValue = (value: unknown, formatType?: string, prefix?: string, suffix?: string): string => {
  if (value === null || value === undefined) {
    return '';
  }
  let formatted = '';
  if (typeof value === 'number') {
    switch (formatType) {
      case 'Currency':
        formatted = currencyFormatter.format(value);
        break;
      case 'Percent':
        formatted = percentFormatter.format(value / (Math.abs(value) > 1 ? 100 : 1));
        break;
      case 'Decimal':
        formatted = value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
        break;
      case 'Integer':
        formatted = Math.round(value).toLocaleString('en-US');
        break;
      default:
        formatted = value.toLocaleString('en-US');
    }
  } else if (formatType === 'Date' && typeof value === 'string') {
    try {
      formatted = new Date(value).toLocaleDateString();
    } catch {
      formatted = String(value);
    }
  } else {
    formatted = String(value);
  }

  if (prefix) formatted = `${prefix}${formatted}`;
  if (suffix) formatted = `${formatted}${suffix}`;
  return formatted;
};

export const evaluateReportExpression = (
  expr: string | undefined,
  dataRecord?: Record<string, any>,
  fallbackText?: string
): string => {
  if (!expr) return fallbackText ?? '';

  let result = expr;

  // Replace standard tokens
  result = result.replace(/\{PageNumber\}/g, '1');
  result = result.replace(/\{TotalPages\}/g, '1');
  result = result.replace(/\{ExecutionTime\}/g, new Date().toLocaleTimeString());
  result = result.replace(/\{UserName\}/g, 'Current User');

  // If we have row data record
  if (dataRecord && typeof dataRecord === 'object') {
    // Replace [Entity.field] or [field]
    result = result.replace(/\[(?:[a-zA-Z0-9_]+\.)?([a-zA-Z0-9_]+)\]/g, (_, fieldName) => {
      const val = dataRecord[fieldName] ?? dataRecord[fieldName.toLowerCase()];
      return val !== undefined && val !== null ? String(val) : `[${fieldName}]`;
    });

    // Replace =Fields!fieldName.Value
    result = result.replace(/=Fields!([a-zA-Z0-9_]+)\.Value/g, (_, fieldName) => {
      const val = dataRecord[fieldName] ?? dataRecord[fieldName.toLowerCase()];
      return val !== undefined && val !== null ? String(val) : `[${fieldName}]`;
    });

    // Evaluate basic math if starts with '=' and contains no unresolved tokens
    if (result.startsWith('=')) {
      const cleanMath = result.slice(1).trim();
      if (/^[\d\s+\-*/().]+$/.test(cleanMath)) {
        try {
          // eslint-disable-next-line no-eval
          const mathVal = Function(`"use strict"; return (${cleanMath})`)();
          return String(mathVal);
        } catch {
          // ignore error and return cleanMath
        }
      }
      return cleanMath;
    }
  }

  return result;
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

export const eventScriptLabels: Record<keyof EventScripts, string> = {
  onRowRender: 'On Row Render',
  onCellRender: 'On Cell Render',
  onPageRender: 'On Page Render',
  onExport: 'On Export',
};

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
