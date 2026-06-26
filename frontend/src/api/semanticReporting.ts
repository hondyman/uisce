/**
 * Semantic Reporting API Client
 * 
 * TypeScript client for the semantic reporting platform APIs.
 * Supports core/extension pattern and Cube.dev integration.
 */

import axios, { AxiosInstance } from 'axios';
import axiosClient from '../utils/axiosClient';

// ============================================================================
// TYPES
// ============================================================================

export interface ReportDefinition {
  id: string;
  tenant_id: string;
  tenant_tenant_instance_id: string;
  report_key: string;
  display_name: string;
  description?: string;
  category?: string;
  tags?: string[];
  report_type: 'paginated' | 'interactive' | 'dashboard';
  output_formats: string[];
  definition: ReportLayout;
  parameters_schema?: Parameter[];
  semantic_cube_id?: string;
  semantic_query?: Record<string, any>;
  version: number;
  is_current: boolean;
  is_core: boolean;
  status: 'draft' | 'published' | 'deprecated' | 'archived';
  created_at: string;
  updated_at?: string;
}

export interface ReportExtension {
  id: string;
  tenant_id: string;
  tenant_tenant_instance_id: string;
  base_report_id: string;
  extension_key: string;
  extension_name?: string;
  description?: string;
  overrides?: Record<string, any>;
  additions?: Record<string, any>;
  removals?: Record<string, any>;
  parameter_defaults?: Record<string, any>;
  version: number;
  is_current: boolean;
  core_version_target?: number;
  status: 'draft' | 'published' | 'deprecated' | 'archived';
  created_at: string;
  updated_at?: string;
}

export interface ReportLayout {
  metadata: LayoutMetadata;
  data_bindings: Record<string, DataBinding>;
  layout: LayoutStructure;
  parameters?: Parameter[];
  conditional_styles?: Record<string, ConditionalStyle>;
  calculated_fields?: CalculatedField[];
  drillthrough?: DrillthroughConfig[];
  export_options?: ExportOptions;
}

export interface LayoutMetadata {
  display_name: string;
  description?: string;
  category?: string;
  page_size?: string;
  orientation?: 'portrait' | 'landscape';
  margins?: Margins;
}

export interface Margins {
  top: number;
  right: number;
  bottom: number;
  left: number;
}

export interface DataBinding {
  cube: string;
  measures: string[];
  dimensions: string[];
  filters?: CubeFilter[];
  time_dimension?: TimeDimension;
  order?: OrderSpec[];
  limit?: number;
  conditional?: {
    parameter: string;
    equals: any;
  };
}

export interface CubeFilter {
  member: string;
  operator: 'equals' | 'notEquals' | 'contains' | 'notContains' | 'gt' | 'gte' | 'lt' | 'lte' | 'set' | 'notSet' | 'inDateRange' | 'notInDateRange' | 'beforeDate' | 'afterDate';
  values: any[];
}

export interface TimeDimension {
  dimension: string;
  date_range: string | [string, string];
  granularity?: 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year';
}

export interface OrderSpec {
  member: string;
  direction: 'asc' | 'desc';
}

export interface LayoutStructure {
  header?: LayoutRegion;
  body: LayoutBody;
  footer?: LayoutRegion;
}

export interface LayoutRegion {
  height?: number;
  elements: LayoutElement[];
}

export interface LayoutBody {
  sections: ReportSection[];
}

export interface ReportSection {
  id: string;
  type: 'summary' | 'table' | 'chart' | 'text' | 'subreport';
  title?: string;
  data_binding?: string;
  columns?: TableColumn[];
  elements?: LayoutElement[];
  chart_config?: ChartConfig;
  grouping?: GroupConfig;
  page_break?: 'before' | 'after' | 'both' | 'none';
  visibility?: VisibilityCondition;
  subreport_id?: string;
  subreport_parameters?: Record<string, string>;
}

export interface TableColumn {
  dimension?: string;
  measure?: string;
  label: string;
  width?: number;
  format?: string;
  alignment?: 'left' | 'center' | 'right';
  conditional_style?: string;
}

export interface LayoutElement {
  type: 'text' | 'image' | 'pageNumber' | 'kpiCard' | 'rectangle' | 'line';
  content?: string;
  src?: string;
  title?: string;
  value?: string;
  change?: string;
  change_type?: string;
  benchmark?: string;
  benchmark_label?: string;
  position?: ElementPosition;
  style?: Record<string, any>;
}

export interface ElementPosition {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface ChartConfig {
  type: 'bar' | 'line' | 'pie' | 'area' | 'scatter' | 'combo';
  x_axis: string;
  y_axis: string[];
  series?: string;
  colors?: string[];
  show_legend?: boolean;
  show_data_labels?: boolean;
  stacked?: boolean;
}

export interface GroupConfig {
  group_by: string;
  show_subtotals?: boolean;
  show_grand_total?: boolean;
  sort_direction?: 'asc' | 'desc';
}

export interface VisibilityCondition {
  expression: string;
  hide_when_empty?: boolean;
}

export interface Parameter {
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean' | 'select' | 'multiselect';
  label: string;
  default_value?: any;
  required?: boolean;
  description?: string;
  options?: SelectOption[];
  validation?: ValidationRule;
}

export interface SelectOption {
  label: string;
  value: any;
}

export interface ValidationRule {
  min?: number;
  max?: number;
  pattern?: string;
  message?: string;
}

export interface ConditionalStyle {
  positive?: Record<string, string>;
  negative?: Record<string, string>;
  zero?: Record<string, string>;
}

export interface CalculatedField {
  name: string;
  expression: string;
  format?: string;
  description?: string;
}

export interface DrillthroughConfig {
  target_report: string;
  parameters: Record<string, string>;
  column?: string;
}

export interface ExportOptions {
  pdf?: {
    include_toc?: boolean;
    password?: string;
  };
  excel?: {
    sheet_per_group?: boolean;
    include_formulas?: boolean;
  };
  html?: {
    include_styles?: boolean;
    interactive?: boolean;
  };
}

export interface ReportInstance {
  id: string;
  tenant_id: string;
  tenant_tenant_instance_id: string;
  report_definition_id: string;
  report_extension_id?: string;
  context_type?: string;
  context_id?: string;
  context_name?: string;
  parameters?: Record<string, any>;
  output_format: string;
  status: 'pending' | 'generating' | 'completed' | 'failed';
  error_message?: string;
  output_url?: string;
  generation_time_ms?: number;
  requested_by?: string;
  requested_at: string;
  completed_at?: string;
}

export interface ReportSchedule {
  id: string;
  tenant_id: string;
  tenant_tenant_instance_id: string;
  report_definition_id: string;
  report_extension_id?: string;
  schedule_name: string;
  cron_expression: string;
  timezone: string;
  is_active: boolean;
  context_type?: string;
  fixed_context_id?: string;
  parameters_template?: Record<string, any>;
  output_formats: string[];
  delivery_channels: DeliveryChannel[];
  next_run_at?: string;
  last_run_at?: string;
}

export interface DeliveryChannel {
  type: 'email' | 'slack' | 'teams' | 'webhook' | 'file_share';
  config: Record<string, any>;
}

export interface ReportCategory {
  id: string;
  tenant_id: string;
  tenant_tenant_instance_id: string;
  name: string;
  description?: string;
  parent_id?: string;
  icon?: string;
  sort_order: number;
}

// Request/Response types
export interface CreateDefinitionRequest {
  report_key: string;
  display_name: string;
  description?: string;
  category?: string;
  tags?: string[];
  report_type?: 'paginated' | 'interactive' | 'dashboard';
  is_core?: boolean;
  definition: ReportLayout;
}

export interface CreateExtensionRequest {
  base_report_id: string;
  extension_key: string;
  extension_name?: string;
  description?: string;
  overrides?: Record<string, any>;
  additions?: Record<string, any>;
  removals?: Record<string, any>;
  parameter_defaults?: Record<string, any>;
}

export interface RenderReportRequest {
  report_definition_id: string;
  report_extension_id?: string;
  output_format: 'pdf' | 'html' | 'excel';
  context_type?: string;
  context_id?: string;
  context_name?: string;
  parameters?: Record<string, any>;
}

// ============================================================================
// API CLIENT
// ============================================================================

export class SemanticReportingClient {
  private client: AxiosInstance;

  constructor(_baseURL?: string, _tenantId?: string, _datasourceId?: string) {
    // We use the singleton axiosClient which already has interceptors for auth and tenant isolation.
    // The arguments are kept for signature compatibility but ignored in favor of global context.
    this.client = axiosClient;
  }

  private getParams() {
    // axiosClient already injects headers.
    // If some endpoints still strictly require them in query params, we keep this,
    // but usually headers are preferred.
    return {};
  }

  // Definitions
  async listDefinitions(filters?: {
    category?: string;
    status?: string;
    is_core?: boolean;
  }): Promise<ReportDefinition[]> {
    const params = { ...this.getParams(), ...filters };
    const response = await this.client.get<ReportDefinition[]>('/reports/definitions', { params });
    return response.data;
  }

  async getDefinition(id: string): Promise<ReportDefinition> {
    const response = await this.client.get<ReportDefinition>(`/reports/definitions/${id}`, {
      params: this.getParams(),
    });
    return response.data;
  }

  async createDefinition(request: CreateDefinitionRequest): Promise<ReportDefinition> {
    const response = await this.client.post<ReportDefinition>('/reports/definitions', request, {
      params: this.getParams(),
    });
    return response.data;
  }

  async updateDefinition(id: string, updates: Partial<ReportDefinition>): Promise<ReportDefinition> {
    const response = await this.client.put<ReportDefinition>(`/reports/definitions/${id}`, updates, {
      params: this.getParams(),
    });
    return response.data;
  }

  async deleteDefinition(id: string): Promise<void> {
    await this.client.delete(`/reports/definitions/${id}`, {
      params: this.getParams(),
    });
  }

  async publishDefinition(id: string): Promise<{ status: string }> {
    const response = await this.client.post<{ status: string }>(`/reports/definitions/${id}/publish`, null, {
      params: this.getParams(),
    });
    return response.data;
  }

  // Extensions
  async listExtensions(baseReportId?: string): Promise<ReportExtension[]> {
    const params = { ...this.getParams(), base_report_id: baseReportId };
    const response = await this.client.get<ReportExtension[]>('/reports/extensions', { params });
    return response.data;
  }

  async getExtension(id: string): Promise<ReportExtension> {
    const response = await this.client.get<ReportExtension>(`/reports/extensions/${id}`, {
      params: this.getParams(),
    });
    return response.data;
  }

  async createExtension(request: CreateExtensionRequest): Promise<ReportExtension> {
    const response = await this.client.post<ReportExtension>('/reports/extensions', request, {
      params: this.getParams(),
    });
    return response.data;
  }

  async updateExtension(id: string, updates: Partial<ReportExtension>): Promise<ReportExtension> {
    const response = await this.client.put<ReportExtension>(`/reports/extensions/${id}`, updates, {
      params: this.getParams(),
    });
    return response.data;
  }

  async deleteExtension(id: string): Promise<void> {
    await this.client.delete(`/reports/extensions/${id}`, {
      params: this.getParams(),
    });
  }

  // Rendering
  async renderReport(request: RenderReportRequest): Promise<ReportInstance> {
    const response = await this.client.post<ReportInstance>('/reports/render', request, {
      params: this.getParams(),
    });
    return response.data;
  }

  async renderReportAsync(request: RenderReportRequest): Promise<ReportInstance> {
    const response = await this.client.post<ReportInstance>('/reports/render/async', request, {
      params: this.getParams(),
    });
    return response.data;
  }

  // Instances
  async listInstances(limit?: number): Promise<ReportInstance[]> {
    const params = { ...this.getParams(), limit };
    const response = await this.client.get<ReportInstance[]>('/reports/instances', { params });
    return response.data;
  }

  async getInstance(id: string): Promise<ReportInstance> {
    const response = await this.client.get<ReportInstance>(`/reports/instances/${id}`, {
      params: this.getParams(),
    });
    return response.data;
  }

  async downloadInstance(id: string): Promise<Blob> {
    const response = await this.client.get(`/reports/instances/${id}/download`, {
      params: this.getParams(),
      responseType: 'blob',
    });
    return response.data;
  }

  // Schedules
  async listSchedules(): Promise<ReportSchedule[]> {
    const response = await this.client.get<ReportSchedule[]>('/reports/schedules', {
      params: this.getParams(),
    });
    return response.data;
  }

  async createSchedule(schedule: Omit<ReportSchedule, 'id' | 'tenant_id' | 'tenant_tenant_instance_id'>): Promise<ReportSchedule> {
    const response = await this.client.post<ReportSchedule>('/reports/schedules', schedule, {
      params: this.getParams(),
    });
    return response.data;
  }

  async getSchedule(id: string): Promise<ReportSchedule> {
    const response = await this.client.get<ReportSchedule>(`/reports/schedules/${id}`, {
      params: this.getParams(),
    });
    return response.data;
  }

  async updateSchedule(id: string, updates: Partial<ReportSchedule>): Promise<ReportSchedule> {
    const response = await this.client.put<ReportSchedule>(`/reports/schedules/${id}`, updates, {
      params: this.getParams(),
    });
    return response.data;
  }

  async deleteSchedule(id: string): Promise<void> {
    await this.client.delete(`/reports/schedules/${id}`, {
      params: this.getParams(),
    });
  }
}

// ============================================================================
// REACT HOOKS FACTORY
// ============================================================================

export function createReportingHooks(apiBaseUrl: string) {
  return {
    useReportingClient: (tenantId: string, datasourceId: string) => {
      return new SemanticReportingClient(apiBaseUrl, tenantId, datasourceId);
    },
  };
}

export default SemanticReportingClient;
