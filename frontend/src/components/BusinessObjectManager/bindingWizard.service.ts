/**
 * Service layer for the binding-first BO create wizard.
 *
 * Uses the existing REST surface while exposing a binding-shaped API.
 * When the backend introduces business_object_binding / field_binding,
 * only this module needs to change.
 */

import { fetchAPI } from '../../api';
import { catalogApi } from '../../api/catalogApi';
import type {
  CatalogTableNode,
  CreateBusinessObjectPayload,
  CreateBusinessObjectFieldPayload,
  WizardBusinessObject,
  WizardBinding,
  WizardField,
  WizardSemanticTerm,
  TermColumnMapping,
  EligibilitySource,
  BindingRequirement,
  BindingStatus,
} from './bindingWizard.types';

export interface FetchTablesOptions {
  tenantId: string;
  datasourceId: string;
  search?: string;
}

export async function fetchCatalogTables(
  opts: FetchTablesOptions
): Promise<CatalogTableNode[]> {
  const { tenantId, datasourceId, search } = opts;
  const params = new URLSearchParams({
    tenant_id: tenantId,
    tenant_instance_id: datasourceId,
    type: 'table',
  });
  if (search) {
    params.append('q', search);
  }

  const data = await fetchAPI<any>(`/catalog/nodes?${params.toString()}`);
  const nodes = Array.isArray(data) ? data : data?.nodes || data?.data || [];

  return nodes.map((n: any) => ({
    node_id: n.node_id || n.id,
    node_name: n.node_name || n.name,
    qualified_path: n.qualified_path || n.node_name || n.name,
    node_type: n.node_type,
    description: n.description,
  }));
}

/**
 * Fetch semantic terms that are mapped to columns on the given table.
 *
 * The backend endpoint `/catalog/semantic-terms-by-table/:tableId` is already
 * used by FieldSelectionWizard. We extend its response by extracting column
 * mappings from the term payload. If the backend does not yet return mappings,
 * we fall back to the term's qualified_path / properties so the wizard can
 * still auto-map fields.
 */
export async function fetchSemanticTermsByTable(
  tableId: string,
  datasourceId: string,
  tableName?: string
): Promise<WizardSemanticTerm[]> {
  const terms = await catalogApi.getSemanticTermsByTable(tableId, datasourceId);
  const rawTerms = Array.isArray(terms) ? terms : [];

  return rawTerms.map((term: any, idx: number): WizardSemanticTerm => {
    const termId = term.id || term.node_id || `term-${idx}`;
    const termName = term.node_name || term.name || term.termName || '';
    const termKey = term.term_key || term.key || termName.toLowerCase().replace(/\s+/g, '_');
    const properties = typeof term.properties === 'string'
      ? safeJsonParse(term.properties, {})
      : term.properties || {};

    const dataType = properties.data_type || term.data_type || term.dataType || 'text';
    const role = properties.role || term.role || 'DIMENSION';

    const mappings = extractMappings(term, properties, tableId, tableName);

    return {
      termNodeId: termId,
      termKey,
      termName,
      description: term.description || properties.description,
      dataType,
      role,
      eligibilitySource: 'DIRECT',
      mappings,
    };
  });
}

function safeJsonParse<T>(value: string, fallback: T): T {
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

function extractMappings(
  term: any,
  properties: Record<string, any>,
  tableId?: string,
  tableName?: string
): TermColumnMapping[] {
  // If the backend already supplies explicit mappings, prefer them.
  if (Array.isArray(term.mappings) && term.mappings.length > 0) {
    return term.mappings.map((m: any) => ({
      columnNodeId: m.column_node_id || m.columnNodeId || '',
      columnName: m.column_name || m.columnName || '',
      tableNodeId: m.table_node_id || m.tableNodeId || tableId || '',
      tableName: m.table_name || m.tableName || tableName || '',
      isPrimarySource: m.is_primary_source ?? m.isPrimarySource ?? false,
    }));
  }

  // Try properties that some endpoints return.
  const explicitColumn =
    properties.source_column ||
    properties.column_name ||
    properties.columnName ||
    properties.sql;

  if (explicitColumn) {
    return [
      {
        columnNodeId: properties.column_node_id || properties.columnNodeId || '',
        columnName: explicitColumn,
        tableNodeId: tableId || '',
        tableName:
          properties.table_name ||
          properties.tableName ||
          tableName ||
          '',
        isPrimarySource: true,
      },
    ];
  }

  // Fallback: derive a synthetic column from qualified_path when it looks like
  // schema.table.column or table.column.
  const qp = term.qualified_path || properties.qualified_path;
  if (typeof qp === 'string') {
    const parts = qp.split('.');
    const maybeColumn = parts.length >= 2 ? parts[parts.length - 1] : undefined;
    if (maybeColumn) {
      return [
        {
          columnNodeId: '',
          columnName: maybeColumn,
          tableNodeId: tableId || '',
          tableName: parts.length >= 3 ? parts[parts.length - 2] : tableName || '',
          isPrimarySource: true,
        },
      ];
    }
  }

  return [];
}

/**
 * Fetch semantic terms reachable through related tables.
 *
 * NOTE: This is a placeholder. The backend does not yet expose an endpoint
 * that returns related-table semantic term eligibility. When it does, replace
 * the body below with the real call.
 */
export async function fetchRelatedSemanticTerms(
  _drivingTableId: string,
  _datasourceId: string
): Promise<WizardSemanticTerm[]> {
  return [];
}

/**
 * Fetch calculated semantic terms whose inputs are available.
 *
 * NOTE: This is a placeholder. The backend does not yet expose an endpoint
 * that returns calculated-term eligibility. When it does, replace the body
 * below with the real call.
 */
export async function fetchCalculatedSemanticTerms(
  _drivingTableId: string,
  _datasourceId: string,
  _selectedTermIds: string[]
): Promise<WizardSemanticTerm[]> {
  return [];
}

/**
 * Fetch all semantic terms (used by the Manual tab).
 *
 * Reuses the existing semantic term catalog endpoint. Terms added manually are
 * marked MANUAL / UNRESOLVED until a binding is defined.
 */
export async function fetchAllSemanticTerms(
  tenantId: string,
  datasourceId: string,
  search?: string
): Promise<WizardSemanticTerm[]> {
  const params = new URLSearchParams({
    tenant_id: tenantId,
    tenant_instance_id: datasourceId,
    type: 'semantic_term',
  });
  if (search) {
    params.append('q', search);
  }

  const data = await fetchAPI<any>(`/catalog/nodes?${params.toString()}`);
  const nodes = Array.isArray(data) ? data : data?.nodes || data?.data || [];

  return nodes.map((term: any, idx: number): WizardSemanticTerm => {
    const properties = typeof term.properties === 'string'
      ? safeJsonParse(term.properties, {})
      : term.properties || {};

    return {
      termNodeId: term.id || term.node_id || `manual-term-${idx}`,
      termKey: term.term_key || term.key || (term.node_name || '').toLowerCase().replace(/\s+/g, '_'),
      termName: term.node_name || term.name || '',
      description: term.description || properties.description,
      dataType: properties.data_type || term.data_type || 'text',
      role: properties.role || 'DIMENSION',
      eligibilitySource: 'MANUAL',
      mappings: [],
    };
  });
}

export async function createBusinessObject(
  payload: CreateBusinessObjectPayload
): Promise<{ id: string; [key: string]: any }> {
  return fetchAPI('/business-objects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

// ─── Binding API ──────────────────────────────────────────────────────────────

export async function fetchBindings(boId: string): Promise<any[]> {
  const data = await fetchAPI<any>(`/business-objects/${boId}/bindings`);
  return Array.isArray(data) ? data : data?.bindings || [];
}

export async function createBinding(
  boId: string,
  payload: {
    backendId: string;
    drivingNodeId: string;
    bindingName?: string;
    baseSql?: string;
    temporalMode?: string;
    isCore?: boolean;
    coreReferenceBindingId?: string;
  }
): Promise<any> {
  return fetchAPI(`/business-objects/${boId}/bindings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function upsertFieldMappings(
  boId: string,
  bindingId: string,
  fields: Array<{
    fieldName: string;
    fieldRole?: string;
    dataType?: string;
    termNodeId?: string;
    isCore?: boolean;
    bindingRequirement?: string;
    bindingStatus?: string;
    eligibilitySource?: string;
    ordinalPosition?: number;
    isExposed?: boolean;
    isPrimaryKey?: boolean;
    coreReferenceFieldId?: string;
  }>
): Promise<void> {
  await fetchAPI(`/business-objects/${boId}/bindings/${bindingId}/fields`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ fields }),
  });
}

// ─── Physical Backends (datasource templates) ─────────────────────────────────

export async function fetchPhysicalBackends(): Promise<
  Array<{ backendId: string; backendName: string; description: string }>
> {
  const data = await fetchAPI<any>('/backends');
  return Array.isArray(data) ? data : data?.backends || [];
}

export async function createPhysicalBackend(payload: {
  backendName: string;
  description?: string;
}): Promise<any> {
  return fetchAPI('/backends', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

// ─── Tenant Datasources ───────────────────────────────────────────────────────

export async function fetchTenantDatasources(): Promise<
  Array<{
    datasourceId: string;
    tenantId: string;
    datasourceType: string;
    host: string;
    port: number;
    databaseName: string;
    username: string;
    provisioningStatus: string;
  }>
> {
  const data = await fetchAPI<any>('/datasources');
  return Array.isArray(data) ? data : data?.datasources || [];
}

export async function createTenantDatasource(payload: {
  datasourceType: string;
  host: string;
  port?: number;
  databaseName: string;
  username?: string;
  password?: string;
}): Promise<any> {
  return fetchAPI('/datasources', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function rebaseField(boId: string, fieldId: string): Promise<any> {
  return fetchAPI(`/business-objects/${boId}/fields/${fieldId}/rebase`, {
    method: 'POST',
  });
}

/**
 * Full wizard save: creates the BO, then persists the binding and field mappings
 * as real DB rows. Falls back gracefully — if binding persist fails, the BO is
 * still created and the error surfaced to the user.
 */
export async function saveBindingWithFields(
  bo: WizardBusinessObject,
  publish = false
): Promise<{ id: string; bindingId?: string; error?: string }> {
  // Step 1: Create the BO
  const boPayload = buildCreateBusinessObjectPayload(bo, publish);
  const boResult = await createBusinessObject(boPayload);
  const boId = boResult.id || boResult.bo_id;
  if (!boId) throw new Error('Business Object created but no ID returned.');

  const binding = bo.bindings[0];
  if (!binding?.backendId || !binding?.drivingTableId) {
    // No backend/table set — return BO-only result
    return { id: boId };
  }

  try {
    // Step 2: Persist the binding row
    const bindingResult = await createBinding(boId, {
      backendId: binding.backendId,
      drivingNodeId: binding.drivingTableId,
      bindingName: binding.backendName
        ? `${binding.backendName} Binding`
        : `${bo.name} Binding`,
      temporalMode: 'NONE',
      isCore: false,
    });
    const bindingId = bindingResult?.boBindingId || bindingResult?.bo_binding_id;

    if (bindingId && binding.fields.length > 0) {
      // Step 3: Persist field mappings
      const fieldPayloads = binding.fields.map((f, idx) => ({
        fieldName: f.name,
        fieldRole: f.role || 'DIMENSION',
        dataType: f.dataType || 'text',
        termNodeId: f.semanticTermId || undefined,
        isCore: false,
        bindingRequirement: f.bindingRequirement || 'REQUIRED',
        bindingStatus: f.bindingStatus || 'UNRESOLVED',
        eligibilitySource: f.eligibilitySource || 'DIRECT',
        ordinalPosition: idx,
        isExposed: true,
        isPrimaryKey: false,
        coreReferenceFieldId: f.coreReferenceFieldId || f.core_reference_field_id,
        core_reference_field_id: f.core_reference_field_id || f.coreReferenceFieldId,
      }));
      await upsertFieldMappings(boId, bindingId, fieldPayloads);
    }

    return { id: boId, bindingId };
  } catch (bindErr: any) {
    // BO created successfully but binding persist failed — surface as warning
    console.warn('[bindingWizard] Binding/field persist failed:', bindErr);
    return {
      id: boId,
      error: `BO created but binding could not be saved: ${bindErr?.message || 'unknown error'}`,
    };
  }
}

function generateLocalId(prefix = 'id'): string {
  // crypto.randomUUID is not available in all test environments, so fall back to a timestamped id.
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function createEmptyBinding(backendId: string, backendName: string): WizardBinding {
  return {
    localId: generateLocalId('binding'),
    backendId,
    backendName,
    isDefault: true,
    fields: [],
  };
}

export function createEmptyBusinessObject(
  backendId: string,
  backendName: string
): WizardBusinessObject {
  return {
    name: '',
    key: '',
    displayName: '',
    description: '',
    status: 'draft',
    enableHistory: false,
    historyMode: 'EXPLICIT_RANGE',
    bindings: [createEmptyBinding(backendId, backendName)],
  };
}

function inferRole(term: WizardSemanticTerm): string {
  return term.role || 'DIMENSION';
}

function inferBindingRequirement(source: EligibilitySource): BindingRequirement {
  switch (source) {
    case 'DIRECT':
      return 'REQUIRED';
    case 'CALCULATED':
      return 'CALCULATED';
    case 'MANUAL':
      return 'OPTIONAL';
    default:
      return 'OPTIONAL';
  }
}

function inferBindingStatus(field: WizardField): BindingStatus {
  if (field.eligibilitySource === 'MANUAL') {
    return field.selectedMapping ? 'PARTIAL' : 'UNRESOLVED';
  }
  if (field.eligibilitySource === 'CALCULATED') {
    return 'RESOLVED';
  }
  return field.selectedMapping ? 'RESOLVED' : 'UNRESOLVED';
}

function pickDefaultMapping(term: WizardSemanticTerm): TermColumnMapping | undefined {
  if (!term.mappings || term.mappings.length === 0) {
    return undefined;
  }
  const primary = term.mappings.find((m) => m.isPrimarySource);
  return primary || term.mappings[0];
}

export function createFieldFromTerm(
  term: WizardSemanticTerm,
  sequence: number
): WizardField {
  const selectedMapping = pickDefaultMapping(term);
  const eligibilitySource = term.eligibilitySource;
  const bindingRequirement = inferBindingRequirement(eligibilitySource);
  const role = inferRole(term);

  const draft: WizardField = {
    key: term.termKey,
    name: term.termName,
    displayName: term.termName,
    description: term.description,
    semanticTermId: term.termNodeId,
    semanticTermName: term.termName,
    dataType: term.dataType || 'text',
    role,
    bindingRequirement,
    eligibilitySource,
    bindingStatus: 'UNRESOLVED',
    selectedMapping,
    sequence,
  };

  draft.bindingStatus = inferBindingStatus(draft);
  return draft;
}

export function buildCreateBusinessObjectPayload(
  bo: WizardBusinessObject,
  publish = false
): CreateBusinessObjectPayload {
  const binding = bo.bindings[0];
  if (!binding) {
    throw new Error('At least one binding is required to create a Business Object.');
  }

  const status = publish ? 'active' : bo.status;

  const fields: CreateBusinessObjectFieldPayload[] = binding.fields.map((field, idx) => {
    const mapping = field.selectedMapping;
    return {
      key: field.key || field.name.toLowerCase().replace(/\s+/g, '_'),
      name: field.name,
      businessName: field.displayName,
      displayName: field.displayName,
      technicalName: field.key,
      description: field.description,
      type: field.dataType || 'text',
      role: field.role,
      semanticTermId: field.semanticTermId,
      semanticTermName: field.semanticTermName,
      sequence: typeof field.sequence === 'number' ? field.sequence : idx,
      isCore: field.bindingRequirement === 'REQUIRED',
      bindingRequirement: field.bindingRequirement,
      eligibilitySource: field.eligibilitySource,
      bindingStatus: field.bindingStatus,
      sourceColumnNodeId: mapping?.columnNodeId,
      sourceColumnName: mapping?.columnName,
      sourceTableNodeId: mapping?.tableNodeId,
      sourceTableName: mapping?.tableName,
      coreReferenceFieldId: field.coreReferenceFieldId || field.core_reference_field_id,
      core_reference_field_id: field.core_reference_field_id || field.coreReferenceFieldId,
      hasLocalOverride: field.hasLocalOverride || field.has_local_override,
      has_local_override: field.has_local_override || field.hasLocalOverride,
    };
  });

  return {
    name: bo.name,
    display_name: bo.displayName || bo.name,
    description: bo.description || undefined,
    status,
    enable_history: bo.enableHistory,
    history_mode: bo.historyMode,
    driver_table_id: binding.drivingTableId,
    driver_table_name: binding.drivingTableName,
    config: {
      is_active: publish,
      fields,
    },
  };
}
