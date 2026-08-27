import { serializeForBackend } from './tableSerialization';
import type { BuilderElement } from './tableSerialization';
import type { FormTemplateSpec } from './form/FormManagerTypes';
import type { AdvancedReportSection } from './sectionLayoutModel';

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

export type ParameterType = 'string' | 'number' | 'date' | 'boolean' | 'list';

export interface ReportParameter {
  id: string;
  name: string;
  type: ParameterType;
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
  
  // Enterprise Source, Cascading & User-Personalization Options
  sourceType?: 'manual' | 'query' | 'context';
  querySql?: string;
  contextKey?: string;
  userContextKey?: string;
  lockForUser?: boolean;
  hidden?: boolean;
  staticOptions?: { value: string; label: string }[];
}

export interface ReportBuilderState {
  elements: BuilderElement[];
  groupDefinitions?: unknown[];
  reportTitle: string;
  sectionConfig?: Record<string, SectionConfigEntry>;
  layoutSettings?: LayoutSettings;
  parameters?: ReportParameter[];
  formSpec?: FormTemplateSpec | null;
  formRegistry?: Record<string, FormTemplateSpec>;
  layoutSections?: AdvancedReportSection[];
}

export interface BOBinding {
  boKey?: string;
  qualifiedPath?: string;
  rootObject?: string;
  subtypeCode?: string;
  fieldAllowlist?: string[];
  filters?: { field: string; operator: string; value?: unknown; parameter?: string }[];
}

export function buildReportTitle(elements: BuilderElement[]): string {
  const titleEl = elements.find((el) => el.type === 'textbox' && el.properties?.name);
  return (titleEl?.properties?.name as string) || 'Untitled Report';
}

export function buildSavePayload(
  state: ReportBuilderState,
  dataSource?: BOBinding | null,
  reportId?: string
) {
  const def = serializeForBackend({
    _schemaVersion: 2,
    elements: state.elements,
    groupDefinitions: state.groupDefinitions || [],
    reportTitle: state.reportTitle,
    sectionConfig: state.sectionConfig,
    formSpec: state.formSpec ?? null,
    formRegistry: state.formRegistry ?? {},
    layoutSections: state.layoutSections ?? [],
  });

  const metadata: Record<string, unknown> = {};
  if (dataSource) {
    metadata.data_bindings = [
      {
        bo_path: dataSource.qualifiedPath || dataSource.boKey,
        field_allowlist: dataSource.fieldAllowlist || [],
        filters: dataSource.filters || [],
      },
    ];
  }

  // Include parameters in metadata and definition
  const params = state.parameters || [];
  metadata.parameters = params;
  (def as any).parameters = params;

  // Persist form layout manager spec
  metadata.formSpec = state.formSpec ?? null;
  metadata.formRegistry = state.formRegistry ?? {};
  metadata.layoutSections = state.layoutSections ?? [];

  return {
    name: state.reportTitle || 'Untitled Report',
    description: '',
    definition: def as unknown as Record<string, unknown>,
    metadata,
  };
}
