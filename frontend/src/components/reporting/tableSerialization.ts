import {
  ColumnConfig,
  TableElementProperties,
  TotalsConfig,
  BandingConfig,
  createDefaultColumnConfig,
  createDefaultBandingConfig,
  createDefaultTotalsConfig,
  createDefaultPaginationConfig,
  createDefaultFreezePaneConfig,
  DEFAULT_HEADER_STYLE,
  DEFAULT_CELL_STYLE,
} from './tableColumnModel';

const SCHEMA_VERSION = 2;

export interface TokenEntry {
  id: string;
  text: string;
  mode: 'static' | 'expression';
  expression?: string;
}

export interface LayoutSettings {
  pageBreakBeforeGroup: boolean;
  pageBreakAfterGroup: boolean;
  pageBreakBetweenRegions: boolean;
  fixedPageSize: boolean;
  columns: number;
  columnSpacing: number;
  headerTokens: (string | TokenEntry)[];
  footerTokens: (string | TokenEntry)[];
  includeExecutionTime: boolean;
  includeUserName: boolean;
}

export interface SectionConfigEntry {
  visible?: boolean;
  visibilityCondition?: unknown;
  backgroundColor?: string;
  pageBreakBefore?: boolean;
  pageBreakAfter?: boolean;
}

export interface BuilderDefinition {
  _schemaVersion: number;
  elements: BuilderElement[];
  groupDefinitions?: unknown[];
  reportTitle?: string;
  sections?: unknown[];
  sectionConfig?: Record<string, SectionConfigEntry>;
  layoutSettings?: LayoutSettings;
}

export interface BuilderElement {
  id: string;
  type: string;
  section?: string;
  position?: { x: number; y: number };
  size?: { width: number; height: number };
  properties: Record<string, unknown>;
}

function uid(): string {
  return `${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
}

export function migrateV1ToV2(def: BuilderDefinition): BuilderDefinition {
  const migratedElements = def.elements.map(el => {
    if (el.type !== 'table' && el.type !== 'matrix') return el;

    const rawEl = el as unknown as Record<string, unknown>;
    const props = (el.properties || {}) as Record<string, unknown>;
    const legacyColumns: string[] =
      (rawEl.columns as string[] | undefined) ||
      (props.columns as string[] | undefined) ||
      [];

    const newColumns: ColumnConfig[] = legacyColumns.map((field: string) => {
      const col = createDefaultColumnConfig(uid(), field);

      if (props.fontSize) {
        col.bodyStyle = { ...DEFAULT_CELL_STYLE, fontSize: Number(props.fontSize) };
        col.headerStyle = { ...DEFAULT_HEADER_STYLE, fontSize: Number(props.fontSize) };
      }
      if (props.textColor) {
        col.bodyStyle = { ...col.bodyStyle, color: String(props.textColor) };
        col.headerStyle = { ...col.headerStyle, color: String(props.textColor) };
      }
      if (props.backgroundColor) {
        col.bodyStyle = { ...col.bodyStyle, backgroundColor: String(props.backgroundColor) };
      }
      if (props.showGridLines === false) {
        // will be overridden by BandingConfig.gridlines
      }

      return col;
    });

    const newProps: TableElementProperties = {
      name: (props.name as string) || (el.type === 'table' ? 'BO Data Table' : 'Matrix'),
      columns: newColumns,
      totals: props.totals as TotalsConfig || createDefaultTotalsConfig(),
      banding: props.banding as BandingConfig || createDefaultBandingConfig(),
      pagination: createDefaultPaginationConfig(),
    };

    return {
      ...el,
      properties: {
        ...props,
        columns: newColumns,
        totals: newProps.totals,
        banding: newProps.banding,
        pagination: newProps.pagination,
        _schemaVersion: SCHEMA_VERSION,
      },
    };
  });

  return {
    ...def,
    _schemaVersion: SCHEMA_VERSION,
    elements: migratedElements,
  };
}

export function serializeForBackend(def: BuilderDefinition): Record<string, unknown> {
  const rawSections = def.sections || [];
  const effectiveSections = rawSections.length > 0
    ? rawSections
    : [{ id: 'default', title: def.reportTitle || 'Report Section' }];

  const outputSections = effectiveSections.map((s: any) => {
    const sectionElements = def.elements.filter(
      (el) => el.section === s.id || (!el.section && s.id === 'default')
    );

    const tableElements = sectionElements
      .filter((el) => el.type === 'table' || el.type === 'matrix')
      .map((el) => {
        const props = el.properties as unknown as TableElementProperties;
        return {
          id: el.id,
          type: 'table',
          columns: (props.columns || []).map((col: ColumnConfig) => ({
            dimension: col.field,
            label: col.headerText,
            width: col.widthPx,
            format: col.formatType,
            formatMask: col.formatMask,
            alignment: col.align,
            verticalAlign: col.verticalAlign,
            wrap: col.wrap,
            visible: col.visible,
            headerStyle: col.headerStyle,
            bodyStyle: col.bodyStyle,
            aggregate: col.aggregate,
            sparkline: col.sparkline,
          })),
          totals: props.totals,
          banding: props.banding,
          freezePane: props.freezePane,
          pagination: props.pagination,
          conditionalRules: props.conditionalRules,
          namedStyles: props.namedStyles,
          sparklines: props.columns
            ?.filter((c: ColumnConfig) => c.sparkline)
            ?.map((c: ColumnConfig) => ({ columnId: c.id, ...c.sparkline })),
          grandTotal: props.totals?.grandTotal?.enabled ?? true,
          subtotals: props.totals?.subtotals?.enabled ?? false,
        };
      });

    const otherElements = sectionElements
      .filter((el) => el.type !== 'table' && el.type !== 'matrix')
      .map((el) => ({
        id: el.id,
        type: el.type,
        content: el.properties?.text || el.properties?.content || '',
        style: {
          fontSize: el.properties?.fontSize,
          textColor: el.properties?.textColor,
          backgroundColor: el.properties?.backgroundColor,
          borderWidth: el.properties?.borderWidth,
          borderStyle: el.properties?.borderStyle,
          borderColor: el.properties?.borderColor,
          borderRadius: el.properties?.borderRadius,
          padding: el.properties?.padding,
          textAlign: el.properties?.textAlign,
          format: el.properties?.formatType,
          prefix: el.properties?.formatPrefix,
          suffix: el.properties?.formatSuffix,
        },
        position: el.position,
        size: el.size,
      }));

    return {
      ...s,
      elements: [...tableElements, ...otherElements],
    };
  });

  return {
    _schemaVersion: SCHEMA_VERSION,
    title: def.reportTitle,
    groupDefinitions: def.groupDefinitions,
    sections: outputSections,
    sectionConfig: def.sectionConfig,
    layoutSettings: def.layoutSettings,
  };
}

export function deserializeFromBackend(raw: Record<string, unknown>): BuilderDefinition {
  const schemaVersion = (raw._schemaVersion as number | undefined) || 1;

  // Handle ReportLayout schema (the actual shape stored in report_definitions.definition)
  // This is the DB format: { metadata, parameters, dataBindings, layout: { body: { sections: [] } } }
  const reportLayoutSections = (raw as any)?.layout?.body?.sections;
  if (Array.isArray(reportLayoutSections) && reportLayoutSections.length > 0) {
    const elements: BuilderElement[] = [];
    reportLayoutSections.forEach((section: any, sIdx: number) => {
      const cols = Array.isArray(section.columns) ? section.columns : [];
      if (cols.length > 0) {
        const newColumns: ColumnConfig[] = cols.map((c: any) => ({
          id: uid(),
          field: c.dimension || c.measure || c.field || '',
          headerText: c.label || c.dimension || c.measure || c.field || '',
          widthPx: c.width || 120,
          visible: c.visible !== false,
          headerStyle: {},
          bodyStyle: {},
          align: (c.alignment as 'left' | 'center' | 'right') || 'left',
          verticalAlign: (c.verticalAlign as 'top' | 'middle' | 'bottom') || 'middle',
          wrap: c.wrap || false,
          formatType: (c.format as any) || 'Auto',
          formatMask: c.formatMask || '',
          formatPrefix: c.formatPrefix || '',
          formatSuffix: c.formatSuffix || '',
          aggregate: c.aggregate,
          sparkline: c.sparkline,
        }));

        elements.push({
          id: section.id || `table_${sIdx}`,
          type: section.type === 'matrix' ? 'matrix' : section.type === 'list' ? 'list' : 'table',
          section: section.id || 'body',
          position: { x: 20, y: 20 + sIdx * 240 },
          size: { width: 760, height: 220 },
          properties: {
            name: section.title || `Section ${sIdx + 1}`,
            columns: newColumns,
            totals: createDefaultTotalsConfig(),
            banding: (() => {
              const b = createDefaultBandingConfig();
              if (section.banding === 'none') { b.bandedRows = false; b.bandedColumns = false; }
              else if (section.banding === 'column') { b.bandedRows = false; b.bandedColumns = true; }
              else { b.bandedRows = true; b.bandedColumns = false; }
              return b;
            })(),
            freezePane: typeof section.freezePane === 'object'
              ? { ...createDefaultFreezePaneConfig(), ...section.freezePane }
              : createDefaultFreezePaneConfig(),
            pagination: createDefaultPaginationConfig(),
            conditionalRules: [],
            namedStyles: [],
          },
        });
      }

      if (section.title) {
        elements.push({
          id: `${section.id || `section_${sIdx}`}_title`,
          type: 'textbox',
          section: section.id || 'header',
          position: { x: 20, y: 10 + sIdx * 240 },
          size: { width: 600, height: 28 },
          properties: {
            name: `${section.title}_label`,
            text: section.title,
            fontSize: 14,
            bold: true,
            textColor: '#1A1A2E',
            backgroundColor: '',
            textAlign: 'left',
            formatType: 'Auto',
            formatPrefix: '',
            formatSuffix: '',
            borderWidth: 0,
            borderStyle: '',
            borderColor: '',
            borderRadius: 0,
            padding: 4,
          },
        });
      }
    });

    return migrateV1ToV2({
      _schemaVersion: 2,
      elements,
      reportTitle:
        (raw as any)?.metadata?.displayName ||
        (raw as any)?.metadata?.name ||
        (raw as any)?.title ||
        'Untitled Report',
      groupDefinitions: [],
      sectionConfig: {},
      layoutSettings: undefined,
    });
  }

  if (schemaVersion < 2) {
    const sections = (raw.sections as any[] || []) as any[];
    const flatElements: BuilderElement[] = sections.flatMap(
      (s: any) => (s.elements || []) as BuilderElement[]
    );
    return migrateV1ToV2({
      _schemaVersion: schemaVersion,
      elements: flatElements,
      reportTitle: (raw.title as string) || '',
      groupDefinitions: [],
    });
  }

  const sections = (raw.sections as any[] || []) as any[];
  const elements: BuilderElement[] = [];

  sections.forEach((section) => {
    (section.elements || []).forEach((el: any) => {
      if (el.type === 'table') {
        const cols: any[] = el.columns || [];
        const newColumns: ColumnConfig[] = cols.map((col: any) => ({
          id: uid(),
          field: col.dimension || col.measure || col.label || '',
          headerText: col.label || '',
          widthPx: col.width || 120,
          visible: col.visible !== false,
          headerStyle: col.headerStyle || {},
          bodyStyle: col.bodyStyle || {},
          align: (col.alignment as 'left' | 'center' | 'right') || 'left',
          verticalAlign: (col.verticalAlign as 'top' | 'middle' | 'bottom') || 'middle',
          wrap: col.wrap || false,
          formatType: (col.format as any) || 'Auto',
          formatMask: col.formatMask || '',
          formatPrefix: col.formatPrefix || '',
          formatSuffix: col.formatSuffix || '',
          aggregate: col.aggregate,
          sparkline: col.sparkline,
        }));

        elements.push({
          id: el.id,
          type: 'table',
          section: section.id,
          position: el.position,
          size: el.size,
          properties: {
            name: section.title || 'Table',
            columns: newColumns,
            totals: el.totals || createDefaultTotalsConfig(),
            banding: el.banding || createDefaultBandingConfig(),
            freezePane: el.freezePane,
            pagination: el.pagination || createDefaultPaginationConfig(),
            conditionalRules: el.conditionalRules || [],
            namedStyles: el.namedStyles || [],
          },
        });
      } else {
        elements.push({
          id: el.id,
          type: el.type,
          section: section.id,
          position: el.position,
          size: el.size,
          properties: {
            text: el.content,
            fontSize: el.style?.fontSize,
            textColor: el.style?.textColor,
            backgroundColor: el.style?.backgroundColor,
            borderWidth: el.style?.borderWidth,
            borderStyle: el.style?.borderStyle,
            borderColor: el.style?.borderColor,
            borderRadius: el.style?.borderRadius,
            padding: el.style?.padding,
            textAlign: el.style?.textAlign,
            formatType: el.style?.format,
            formatPrefix: el.style?.prefix,
            formatSuffix: el.style?.suffix,
          },
        });
      }
    });
  });

  return {
    _schemaVersion: schemaVersion,
    elements,
    reportTitle: (raw.title as string) || '',
    groupDefinitions: Array.isArray(raw.groupDefinitions) ? raw.groupDefinitions : [],
    sections,
    sectionConfig: (raw.sectionConfig as Record<string, SectionConfigEntry>) || {},
    layoutSettings: migrateLayoutSettings(raw.layoutSettings as LayoutSettings | undefined),
  };
}

function migrateLayoutSettings(raw?: LayoutSettings): LayoutSettings | undefined {
  if (!raw) return undefined;
  return {
    ...raw,
    headerTokens: migrateTokens(raw.headerTokens),
    footerTokens: migrateTokens(raw.footerTokens),
  };
}

function migrateTokens(tokens?: (string | TokenEntry)[]): (string | TokenEntry)[] {
  if (!tokens) return [];
  return tokens.map((t) => {
    if (typeof t === 'string') {
      return { id: `tok_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`, text: t, mode: 'static' as const };
    }
    return t;
  });
}

export function needsMigration(def: BuilderDefinition): boolean {
  return (def._schemaVersion || 1) < SCHEMA_VERSION;
}
