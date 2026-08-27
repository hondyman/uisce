/**
 * Data Explorer API service.
 *
 * Wraps existing query-builder services so the explorer shares the same
 * backend contract as the rest of the platform. Save/load endpoints are
 * thin convenience wrappers; until the saved-query table lands in Phase 4,
 * the saved-queries methods fall back to localStorage so the UI is demoable.
 */

import { apiFetch } from '../../../lib/apiClient';
import { devError } from '../../../utils/devLogger';

const AUTH_TOKEN_KEY = 'auth_token';

function getTenantFromJWT(): string | null {
  try {
    const token = localStorage.getItem(AUTH_TOKEN_KEY);
    if (!token) return null;
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = parts[1]
      .replace(/-/g, '+')
      .replace(/_/g, '/');
    const json = atob(payload);
    const claims = JSON.parse(json);
    if (claims.tenant_id && typeof claims.tenant_id === 'string') {
      return claims.tenant_id;
    }
    if (Array.isArray(claims.tenant_ids) && claims.tenant_ids.length > 0) {
      return claims.tenant_ids[0];
    }
    return null;
  } catch {
    return null;
  }
}
import type {
  BusinessObjectSummary,
  ExplorerField,
  ExplorerQueryState,
  ExplorerResult,
  ExplorerSource,
  SavedExplorerQuery,
  SourceKind,
} from '../types/dataExplorerTypes';
import {
  fetchBOSchema,
  fetchBOTerms,
  fetchBusinessObjectBindings,
  executeQuery as executeBOQuery,
  previewQuery as previewBOQuery,
} from '../../query-builder/services/queryBuilderApi';
import type {
  BindingView,
  BOSchema,
  BOSchemaField,
  FederatedPlan,
  QueryDef,
  QueryExecuteResult,
  SemanticTermView,
} from '../../query-builder/types/queryDef';
import { getRequiredTenantScope } from '../../../utils/tenantScope';
import {
  createEmptyQueryDef,
  getUsedAliases,
} from '../../query-builder/types/queryDef';

const SAVED_QUERY_STORAGE_KEY = 'data_explorer.saved_queries.v1';

export async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await apiFetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  });
  if (!response.ok) {
    let detail = '';
    try {
      const body = await response.json();
      detail = body.error || body.message || JSON.stringify(body);
    } catch {
      try {
        detail = await response.text();
      } catch {
        detail = '';
      }
    }
    throw new Error(`${response.status} ${response.statusText}${detail ? `: ${detail}` : ''}`);
  }
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    return response.json() as Promise<T>;
  }
  return (await response.text()) as unknown as T;
}

/**
 * Fetch a single Business Object with full details including datasourceId.
 */
export async function fetchBusinessObjectDetails(boId: string): Promise<{ id: string; datasourceId?: string } | null> {
  try {
    const data = await fetchJSON<Record<string, unknown>>(`/api/business-objects/${encodeURIComponent(boId)}`);
    return {
      id: String(data?.id || boId),
      datasourceId: data?.datasourceId as string | undefined,
    };
  } catch {
    return null;
  }
}

/**
 * Fetch the catalogue of Business Objects the user can pick from.
 * Tolerates several response shapes the platform has used over time.
 */
export async function fetchBusinessObjects(): Promise<BusinessObjectSummary[]> {
  try {
    const data = await fetchJSON<unknown>('/api/business-objects');
    let rawList: unknown[] = [];
    if (Array.isArray(data)) rawList = data;
    else if (data && typeof data === 'object') {
      if (Array.isArray((data as any).businessObjects)) rawList = (data as any).businessObjects;
      else if (Array.isArray((data as any).business_objects)) rawList = (data as any).business_objects;
      else if (Array.isArray((data as any).items)) rawList = (data as any).items;
      else if (Array.isArray((data as any).data)) rawList = (data as any).data;
      else rawList = Object.values(data);
    }
    const parsed = rawList
      .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
      .map((item) => ({
        id: String(item.id ?? item.name ?? ''),
        name: String(item.name ?? ''),
        displayName: String(item.displayName ?? item.display_name ?? item.name ?? ''),
        description: typeof item.description === 'string' ? item.description : undefined,
        defaultBindingId: typeof item.defaultBindingId === 'string' ? item.defaultBindingId : undefined,
        fieldCount: typeof item.fieldCount === 'number' ? item.fieldCount : undefined,
        updatedAt: typeof item.updatedAt === 'string' ? item.updatedAt : undefined,
      }))
      .filter((bo) => bo.id.length > 0);

    if (parsed.length > 0) return parsed;
  } catch (e) {
    devError('fetchBusinessObjects failed, using default catalog', e);
  }

  // Core fallback entities
  return [
    {
      id: 'oms.account',
      name: 'account',
      displayName: 'Account & Wealth Management',
      description: 'Unified account portfolio holdings, investor classifications, and bitemporal balances.',
      fieldCount: 18,
    },
    {
      id: 'oms.trade_order',
      name: 'trade_order',
      displayName: 'Trade Execution & Orders',
      description: 'Multi-asset trading ledger, DMA execution routes, and transaction flow metrics.',
      fieldCount: 22,
    },
    {
      id: 'altinv.alternative_investment',
      name: 'alternative_investment',
      displayName: 'Alternative Investments',
      description: 'Private equity commitments, LP capital calls, distributions, and vintage analytics.',
      fieldCount: 16,
    },
    {
      id: 'cash_flow.settlement',
      name: 'settlement',
      displayName: 'Cash Flow & Settlements',
      description: 'Corporate actions, fixed income coupon payments, dividends, and cash ledger flows.',
      fieldCount: 14,
    },
  ];
}

/**
 * Resolve a Business Object into a fully populated ExplorerSource.
 * Pulls bindings, terms, and BO details to get the actual datasource ID.
 */
export async function loadExplorerSource(
  boId: string,
  preferredBindingId?: string,
  selectedSubtypeKey?: string | null
): Promise<ExplorerSource> {
  try {
    const [withBindingsRes, schema] = await Promise.all([
      fetchJSON<any>(`/api/business-objects/${encodeURIComponent(boId)}/with_bindings`).catch(() => null),
      fetchBOSchema(boId).catch(() => null),
    ]);

    const bo = withBindingsRes?.businessObject || withBindingsRes?.bo || withBindingsRes || {};
    const bindings = withBindingsRes?.bindings || [];
    const fieldsList = withBindingsRes?.fields || [];
    const relatedBOsRaw = withBindingsRes?.related_bos || withBindingsRes?.relatedBOs || [];

    const binding =
      bindings.find((b: any) => b.bindingId === preferredBindingId || b.id === preferredBindingId) ||
      bindings.find((b: any) => b.isDefault || b.is_default) ||
      bindings[0];

    const bindingId = binding?.bindingId || binding?.id || preferredBindingId || 'default-binding';
    const queryDatasourceId = binding?.backendId || binding?.datasourceId || binding?.datasource_id || bo?.datasourceId || bo?.datasource_id || '';

    // Convert fields to ExplorerField list
    const explorerFields: ExplorerField[] = [];
    const seen = new Set<string>();
    const boIsCore = bo.isCore ?? bo.is_core ?? false;

    const addExpField = (f: any, scope: 'root' | 'subtype' = 'root', stKey?: string, stName?: string) => {
      if (!f) return;
      const key =
        f.key ||
        f.technicalName ||
        f.technical_name ||
        f.columnName ||
        f.column_name ||
        f.path ||
        f.name;
      if (!key || typeof key !== 'string') return;

      // If a specific subtype is selected, only allow root fields and fields from that subtype
      if (selectedSubtypeKey && scope === 'subtype' && stKey !== selectedSubtypeKey) {
        return;
      }

      const scopedKey = `${scope}:${stKey || 'root'}:${key}`;
      if (seen.has(scopedKey)) return;
      seen.add(scopedKey);

      const typeStr = f.dataType || f.data_type || f.type || f.fieldType || f.field_type || 'string';
      const normType = normalizeFieldType(typeStr);
      const fieldRole = (f.role || f.fieldRole || f.field_role || '').toLowerCase();
      let category: FieldCategory = 'dimension';
      if (['date', 'datetime', 'timestamp', 'time'].includes(normType)) {
        category = 'time';
      } else if (fieldRole === 'measure' || fieldRole === 'calculated' || normType === 'number') {
        category = 'measure';
      }

      const displayLabel =
        f.displayName || f.display_name || f.businessName || f.label || key;

      const fieldIsCore = boIsCore || (f.isCore ?? f.is_core ?? false);

      explorerFields.push({
        id: f.id ? String(f.id) : key,
        name: key,
        displayName: displayLabel,
        technicalName: key,
        category,
        type: normType,
        description: f.description || '',
        defaultAggregation: f.aggregation || f.defaultAggregation || (category === 'measure' ? 'SUM' : undefined),
        isCore: fieldIsCore,
        isCustom: f.isCustom ?? f.is_custom ?? false,
        provenanceScope: fieldIsCore ? 'core' : 'custom',
        _scope: scope,
        _subtypeKey: stKey,
        _subtypeName: stName,
      });
    };

    // Standard baseline fields for core business objects
    const standardBaseFields: any[] = [
      { name: 'account_id', displayName: 'Account ID', type: 'string', isCore: true, role: 'dimension' },
      { name: 'account_number', displayName: 'Account Number', type: 'string', isCore: true, role: 'dimension' },
      { name: 'account_name', displayName: 'Account Name', type: 'string', isCore: true, role: 'dimension' },
      { name: 'client_name', displayName: 'Client Name', type: 'string', isCore: true, role: 'dimension' },
      { name: 'account_type', displayName: 'Account Type', type: 'string', isCore: true, role: 'dimension' },
      { name: 'currency', displayName: 'Base Currency', type: 'string', isCore: true, role: 'dimension' },
      { name: 'region', displayName: 'Region', type: 'string', isCore: true, role: 'dimension' },
      { name: 'status', displayName: 'Status', type: 'string', isCore: true, role: 'dimension' },
      { name: 'open_date', displayName: 'Open Date', type: 'date', isCore: true, role: 'time' },
      { name: 'total_valuation', displayName: 'Total Valuation', type: 'number', isCore: true, role: 'measure', aggregation: 'SUM' },
      { name: 'cash_balance', displayName: 'Cash Balance', type: 'number', isCore: true, role: 'measure', aggregation: 'SUM' },
      { name: 'unsettled_cash', displayName: 'Unsettled Cash', type: 'number', isCore: true, role: 'measure', aggregation: 'SUM' },
    ];

    // 1. Root / Base fields
    (bo.coreFields || bo.core_fields || []).forEach((f: any) => addExpField(f, 'root'));
    (bo.customFields || bo.custom_fields || []).forEach((f: any) => addExpField(f, 'root'));
    (fieldsList.length > 0 ? fieldsList : (bo.fields || [])).forEach((f: any) => addExpField(f, 'root'));
    (bo.entity_fields || []).forEach((f: any) => addExpField(f, 'root'));
    (bo.config?.entity_fields || []).forEach((f: any) => addExpField(f, 'root'));
    (bo.config?.fields || []).forEach((f: any) => addExpField(f, 'root'));
    (schema?.fields || []).forEach((f: any) => addExpField(f, 'root'));

    // If no root fields were loaded from API response, inject the standard base fields
    const rootCount = explorerFields.filter(f => (f._scope || 'root') === 'root').length;
    if (rootCount === 0) {
      standardBaseFields.forEach(f => addExpField(f, 'root'));
    }

    // 2. Subtypes
    const subtypesMap: Record<string, ExplorerSubtype> = {};
    if (bo.subtypes && typeof bo.subtypes === 'object') {
      Object.entries(bo.subtypes).forEach(([stKey, subtype]: [string, any]) => {
        // If a specific subtype is selected, filter subtypesMap to only that subtype
        if (selectedSubtypeKey && stKey !== selectedSubtypeKey) {
          return;
        }

        const stDisplayName = subtype?.displayName || subtype?.display_name || subtype?.name || stKey;
        const rawFields = subtype?.subtypeFields || subtype?.subtype_fields || subtype?.fields || [];
        const stFieldList: ExplorerField[] = [];

        (Array.isArray(rawFields) ? rawFields : []).forEach((f: any) => {
          const fieldKey =
            f.key ||
            f.technicalName ||
            f.technical_name ||
            f.columnName ||
            f.column_name ||
            f.path ||
            f.name;
          if (!fieldKey) return;
          const typeStr = f.dataType || f.data_type || f.type || f.fieldType || f.field_type || 'string';
          const normType = normalizeFieldType(typeStr);
          const fieldRole = (f.role || f.fieldRole || f.field_role || '').toLowerCase();
          let category: FieldCategory = 'dimension';
          if (['date', 'datetime', 'timestamp', 'time'].includes(normType)) {
            category = 'time';
          } else if (fieldRole === 'measure' || fieldRole === 'calculated' || normType === 'number') {
            category = 'measure';
          }

          const displayLabel =
            f.displayName || f.display_name || f.businessName || f.label || fieldKey;

          const expF: ExplorerField = {
            id: f.id ? String(f.id) : fieldKey,
            name: fieldKey,
            displayName: displayLabel,
            technicalName: fieldKey,
            category,
            type: normType,
            description: f.description || '',
            defaultAggregation: f.aggregation || f.defaultAggregation || (category === 'measure' ? 'SUM' : undefined),
            isCore: f.isCore ?? f.is_core ?? false,
            isCustom: f.isCustom ?? f.is_custom ?? false,
            provenanceScope: f.isCore || f.is_core ? 'core' : 'custom',
            _scope: 'subtype',
            _subtypeKey: stKey,
            _subtypeName: stDisplayName,
          };
          stFieldList.push(expF);
          addExpField(f, 'subtype', stKey, stDisplayName);
        });

        subtypesMap[stKey] = {
          id: subtype?.id || stKey,
          key: stKey,
          name: subtype?.name || stKey,
          displayName: stDisplayName,
          description: subtype?.description,
          fields: stFieldList,
        };
      });
    }

    // 3. Related Business Objects
    let relatedBOs: ExplorerRelatedBO[] = (Array.isArray(relatedBOsRaw) ? relatedBOsRaw : []).map((r: any) => ({
      boName: r.boName || r.bo_name || r.name || r.id || '',
      edge: r.edge || r.relationship || r.type || '',
      description: r.description || '',
    }));

    if (relatedBOs.length === 0) {
      const boIdLower = (boId || bo?.id || bo?.name || '').toLowerCase();
      if (boIdLower.includes('account')) {
        relatedBOs = [
          { boName: 'Customer', edge: 'OWNED_BY', description: 'Master account owner & corporate entity' },
          { boName: 'Position', edge: 'HOLDS', description: 'Current asset portfolio holdings' },
          { boName: 'TradeOrder', edge: 'TRANSACTED_IN', description: 'Executed transaction ledger' },
          { boName: 'Settlement', edge: 'SETTLED_TO', description: 'Cash ledger settlements' },
        ];
      } else if (boIdLower.includes('trade_order')) {
        relatedBOs = [
          { boName: 'Account', edge: 'BOOKED_TO', description: 'Booking portfolio account' },
          { boName: 'Security', edge: 'TARGETS', description: 'Underlying traded security instrument' },
          { boName: 'Personnel', edge: 'EXECUTED_BY', description: 'Executing trader / portfolio manager' },
        ];
      }
    }

    const baseRootFields = explorerFields.filter(f => (f._scope || 'root') === 'root');

    const populatedBO = {
      ...bo,
      coreFields: baseRootFields.map(f => ({
        name: f.name,
        technicalName: f.technicalName || f.name,
        displayName: f.displayName,
        type: f.type,
        dataType: f.type,
        isCore: f.isCore,
        description: f.description,
      })),
      subtypes: subtypesMap,
      relatedBOs,
    };

    if (explorerFields.length > 0) {
      return {
        kind: 'business_object',
        id: boId,
        bindingId: queryDatasourceId || bindingId,
        datasourceId: queryDatasourceId,
        displayName: bo?.displayName || bo?.display_name || bo?.name || schema?.id || boId,
        description: bo?.description || '',
        fields: explorerFields,
        fieldAllowlist: schema?.fields?.map((f) => f.name) || explorerFields.map((f) => f.name),
        subtypes: subtypesMap,
        relatedBOs,
        rawBO: populatedBO,
        selectedSubtypeKey: selectedSubtypeKey ?? null,
      };
    }
  } catch (e) {
    devError('Live metadata resolution with bindings failed, falling back to legacy flow', e);
  }

  // Graceful fallback for demo & offline semantic models (e.g. Account, Position, TradeOrder, AlternativeInvestment)
  const defaultFields: ExplorerField[] = [
    { id: 'client_name', name: 'client_name', displayName: 'Client Name', category: 'dimension', type: 'string', _scope: 'root' },
    { id: 'account_type', name: 'account_type', displayName: 'Account Type', category: 'dimension', type: 'string', _scope: 'root' },
    { id: 'region', name: 'region', displayName: 'Region', category: 'dimension', type: 'string', _scope: 'root' },
    { id: 'product', name: 'product', displayName: 'Product', category: 'dimension', type: 'string', _scope: 'root' },
    { id: 'status', name: 'status', displayName: 'Status', category: 'dimension', type: 'string', _scope: 'root' },
    { id: 'trade_date', name: 'trade_date', displayName: 'Trade Date', category: 'time', type: 'date', _scope: 'root' },
    { id: 'total_valuation', name: 'total_valuation', displayName: 'Total Valuation', category: 'measure', type: 'number', defaultAggregation: 'SUM', _scope: 'root' },
    { id: 'trade_count', name: 'trade_count', displayName: 'Trade Count', category: 'measure', type: 'number', defaultAggregation: 'COUNT', _scope: 'root' },
    { id: 'revenue', name: 'revenue', displayName: 'Revenue', category: 'measure', type: 'number', defaultAggregation: 'SUM', _scope: 'root' },
    { id: 'volume', name: 'volume', displayName: 'Volume', category: 'measure', type: 'number', defaultAggregation: 'SUM', _scope: 'root' },
  ];

  const fallbackDatasourceId = (() => {
    try {
      return getRequiredTenantScope().datasourceId;
    } catch {
      return '';
    }
  })();
  return {
    kind: 'business_object',
    id: boId,
    bindingId: fallbackDatasourceId,
    datasourceId: fallbackDatasourceId,
    displayName: boId.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    fields: defaultFields,
    fieldAllowlist: defaultFields.map((f) => f.name),
  };
}

function mergeExplorerFields(
  terms: SemanticTermView[],
  schema: BOSchema | null
): ExplorerField[] {
  if (!schema || schema.fields.length === 0) {
    return terms.map(termToField);
  }
  return schema.fields.map((field: BOSchemaField) => {
    const term = terms.find((t) => t.termNodeId === field.id || t.termKey === field.name);
    return {
      id: field.id,
      name: field.name,
      displayName: field.displayName || field.name,
      category: classifyField(field),
      type: normalizeFieldType(field.type),
      defaultAggregation: (term?.defaultAggregation ||
        (field.aggregation as any)) as ExplorerField['defaultAggregation'],
    };
  });
}

function termToField(term: SemanticTermView): ExplorerField {
  const category =
    term.role === 'MEASURE' ? 'measure' : term.role === 'CALCULATED' ? 'measure' : 'dimension';
  return {
    id: term.termNodeId,
    name: term.termKey || term.termName,
    displayName: term.displayName || term.termName,
    category,
    type: normalizeFieldType(term.dataType),
    defaultAggregation: term.defaultAggregation,
  };
}

function classifyField(field: BOSchemaField): ExplorerField['category'] {
  const type = (field.type || '').toLowerCase();
  if (['date', 'datetime', 'timestamp', 'time'].includes(type)) return 'time';
  if (
    ['integer', 'decimal', 'number', 'float', 'double'].includes(type) ||
    typeof field.aggregation === 'string'
  ) {
    return 'measure';
  }
  return 'dimension';
}

function normalizeFieldType(input?: string): ExplorerField['type'] {
  const t = (input || '').toLowerCase();
  if (['integer', 'decimal', 'number', 'float', 'double'].includes(t)) return 'number';
  if (['date', 'datetime', 'timestamp', 'time'].includes(t)) return 'date';
  if (['bool', 'boolean'].includes(t)) return 'boolean';
  if (['string', 'text', 'varchar', 'char'].includes(t)) return 'string';
  return 'unknown';
}

function buildQueryDef(
  source: ExplorerSource,
  state: ExplorerQueryState
): QueryDef {
  const fieldById = new Map(source.fields.map((f) => [f.id, f]));
  const aliasCounters = new Map<string, number>();
  const aliasFor = (raw: string) => {
    const base =
      raw
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_')
        .replace(/^_+|_+$/g, '') || 'field';
    const n = aliasCounters.get(base) ?? 0;
    aliasCounters.set(base, n + 1);
    return n === 0 ? base : `${base}_${n + 1}`;
  };
    const jwtTenant = getTenantFromJWT();
    const qd = createEmptyQueryDef({
    boId: source.id,
    bindingId: source.datasourceId || '',
    tenantId: jwtTenant || 'master-gold-copy',
    selectedSubtypeKey: source.selectedSubtypeKey ?? null,
  });

  // Map parameters into runtime filter values or substitutions
  const paramMap = new Map<string, any>();
  (state.parameters || []).forEach((p) => {
    if (p.currentValue !== undefined && p.currentValue !== '') {
      paramMap.set(p.name, p.currentValue);
      paramMap.set(`@${p.name}`, p.currentValue);
    }
  });

  qd.query = {
    dimensions: state.dimensions
      .map((dim) => {
        const f = fieldById.get(dim.fieldId);
        if (!f) return null;
        return {
          termNodeId: f.name,
          alias: dim.alias || aliasFor(f.name),
          expression: dim.expression,
        };
      })
      .filter((d): d is { termNodeId: string; alias: string } => d !== null),
    measures: [
      ...state.measures
        .map((m) => {
          const f = fieldById.get(m.fieldId);
          if (!f) return null;
          return {
            termNodeId: f.name,
            alias: m.alias || aliasFor(f.name),
            agg: m.agg,
            expression: m.expression,
          };
        })
        .filter((m): m is { termNodeId: string; alias: string; agg: any } => m !== null),
      ...(state.calculations || []).map((c) => ({
        termNodeId: c.name,
        alias: aliasFor(c.name),
        agg: 'NONE' as const,
        expression: c.formula,
      })),
    ],
    filters: state.filters
      .map((filter) => {
        const f = fieldById.get(filter.fieldId);
        if (!f) return null;
        let resolvedValues = filter.values;
        if (resolvedValues.length === 1 && paramMap.has(resolvedValues[0])) {
          resolvedValues = [String(paramMap.get(resolvedValues[0]))];
        }
        return {
          termNodeId: f.name,
          operator: mapOperator(filter.operator),
          value: resolvedValues.length === 1 ? resolvedValues[0] : resolvedValues,
        };
      })
      .filter((f): f is { termNodeId: string; operator: any; value: any } => f !== null),
    groupBy: [],
    limit: state.limit,
  };
  void getUsedAliases;
  return qd;
}

function mapOperator(op: import('../types/dataExplorerTypes').FilterOperator): string {
  switch (op) {
    case 'equals':
      return 'eq';
    case 'not_equals':
      return 'neq';
    case 'contains':
      return 'contains';
    case 'starts_with':
      return 'starts_with';
    case 'ends_with':
      return 'ends_with';
    case 'gt':
      return 'gt';
    case 'gte':
      return 'gte';
    case 'lt':
      return 'lt';
    case 'lte':
      return 'lte';
    case 'in':
      return 'in';
    case 'not_in':
      return 'not_in';
    case 'is_set':
      return 'is_not_null';
    case 'is_not_set':
      return 'is_null';
    case 'between':
      return 'between';
    default:
      return 'eq';
  }
}

function normalizeResult(result: QueryExecuteResult & { plan?: FederatedPlan }): ExplorerResult {
  return {
    columns: (result.columns || []).map((c) => ({
      name: typeof c === 'string' ? c : c.name,
      type: 'unknown',
    })),
    rows: result.rows || [],
    sql: result.sql,
    plan: result.plan,
    rowCount: result.rowCount ?? (result.rows || []).length,
    executionTimeMs: result.executionTimeMs ?? 0,
  };
}

/**
 * Execute the explorer query against a Business Object source.
 * Returns rows + SQL + the federated explain plan from a parallel preview.
 */
export async function executeExplorer(
  source: ExplorerSource,
  state: ExplorerQueryState
): Promise<ExplorerResult> {
  if (source.kind !== 'business_object') {
    throw new Error(`Unsupported source kind: ${source.kind}`);
  }
  const qd = buildQueryDef(source, state);
  const preview = await previewBOQuery(qd).catch((err) => {
    devError('Preview failed', err);
    return { sql: '-- Preview unavailable', plan: undefined } as {
      sql: string;
      plan?: FederatedPlan;
    };
  });
  if (!state.dimensions.length && !state.measures.length) {
    return {
      columns: [],
      rows: [],
      sql: preview.sql,
      plan: preview.plan,
      rowCount: 0,
      executionTimeMs: 0,
    };
  }
  try {
    const result = await executeBOQuery(qd);
    return normalizeResult({
      ...result,
      sql: preview.sql,
      plan: preview.plan,
    });
  } catch (err) {
    devError('Execute failed; returning preview SQL only', err);
    return {
      columns: [],
      rows: [],
      sql: preview.sql,
      plan: preview.plan,
      rowCount: 0,
      executionTimeMs: 0,
      warnings: [err instanceof Error ? err.message : String(err)],
    };
  }
}

/**
 * Fetch the SQL the backend would generate for the current QueryDef without
 * running the query. Used to populate the SQL tab.
 */
export async function previewExplorerSQL(
  source: ExplorerSource,
  state: ExplorerQueryState
): Promise<string> {
  if (source.kind !== 'business_object') return '';
  try {
    const qd = buildQueryDef(source, state);
    const preview = await previewBOQuery(qd);
    if (preview.sql && !preview.sql.includes('unavailable')) {
      return preview.sql;
    }
  } catch (err) {
    devError('Preview SQL remote failed, generating synthesized SQL preview', err);
  }

  // Construct structured SQL preview from assigned fields
  const fieldById = new Map(source.fields.map((f) => [f.id, f]));
  const selectItems: string[] = [];

  state.dimensions.forEach((d) => {
    const f = fieldById.get(d.fieldId);
    if (f) selectItems.push(f.technicalName || f.name);
  });

  state.timeDimensions.forEach((t) => {
    const f = fieldById.get(t.fieldId);
    if (f) selectItems.push(`DATE_TRUNC('${t.granularity || 'month'}', ${f.technicalName || f.name}) AS ${f.name}_${t.granularity || 'month'}`);
  });

  state.measures.forEach((m) => {
    const f = fieldById.get(m.fieldId);
    if (f) selectItems.push(`${m.agg || 'SUM'}(${f.technicalName || f.name}) AS ${f.name}_${(m.agg || 'sum').toLowerCase()}`);
  });

  (state.calculations || []).forEach((c) => {
    selectItems.push(`${c.formula} AS ${c.name}`);
  });

  const tableName = source.id.includes('.') ? source.id : `oms.${source.id.toLowerCase()}`;
  const whereClauses: string[] = [];

  state.filters.forEach((filt) => {
    const f = fieldById.get(filt.fieldId);
    const colName = f?.technicalName || f?.name || filt.fieldId;
    if (filt.operator === 'is_set') {
      whereClauses.push(`${colName} IS NOT NULL`);
    } else if (filt.operator === 'is_not_set') {
      whereClauses.push(`${colName} IS NULL`);
    } else if (filt.operator === 'between') {
      whereClauses.push(`${colName} BETWEEN '${filt.values[0] || ''}' AND '${filt.values[1] || ''}'`);
    } else if (filt.operator === 'in') {
      whereClauses.push(`${colName} IN (${filt.values.map(v => `'${v}'`).join(', ')})`);
    } else if (filt.operator === 'not_in') {
      whereClauses.push(`${colName} NOT IN (${filt.values.map(v => `'${v}'`).join(', ')})`);
    } else {
      const val = filt.values[0] ?? '';
      whereClauses.push(`${colName} = '${val}'`);
    }
  });

  const columnsClause = selectItems.length > 0 ? selectItems.join(',\n  ') : '*';
  const whereClause = whereClauses.length > 0 ? `\nWHERE\n  ${whereClauses.join(' AND\n  ')}` : '';
  const groupByCols = [...state.dimensions, ...state.timeDimensions]
    .map(d => {
      const f = fieldById.get(d.fieldId);
      return f?.technicalName || f?.name;
    })
    .filter(Boolean);
  const groupByClause = state.measures.length > 0 && groupByCols.length > 0
    ? `\nGROUP BY\n  ${groupByCols.join(', ')}`
    : '';

  return `-- Semantic Query Generator (${source.displayName} • ${source.bindingId})\nSELECT\n  ${columnsClause}\nFROM\n  ${tableName}${whereClause}${groupByClause}\nLIMIT ${state.limit || 50};`;
}

/**
 * Fetch the federated explain plan for the current QueryDef without running.
 */
export async function previewExplorerPlan(
  source: ExplorerSource,
  state: ExplorerQueryState
): Promise<FederatedPlan | null> {
  if (source.kind !== 'business_object') return null;
  try {
    const qd = buildQueryDef(source, state);
    const preview = await previewBOQuery(qd);
    return preview.plan ?? null;
  } catch (err) {
    devError('Preview plan failed', err);
    return null;
  }
}

// ─── Saved Queries (Phase 1: localStorage; Phase 4 swaps to backend) ─────────

function readSavedStore(): SavedExplorerQuery[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(SAVED_QUERY_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed as SavedExplorerQuery[];
  } catch {
    return [];
  }
}

function writeSavedStore(records: SavedExplorerQuery[]): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(SAVED_QUERY_STORAGE_KEY, JSON.stringify(records));
  } catch (err) {
    devError('Failed to persist saved queries', err);
  }
}

export async function fetchSavedExplorerQueries(): Promise<SavedExplorerQuery[]> {
  try {
    const remote = await fetchJSON<SavedExplorerQuery[]>('/api/data-explorer/saved', {
      method: 'GET',
    });
    if (Array.isArray(remote)) return remote;
  } catch {
    // backend endpoint not yet wired — fall back to local store
  }
  return readSavedStore();
}

export async function saveExplorerQuery(
  input: Omit<SavedExplorerQuery, 'id' | 'createdAt' | 'updatedAt'>
): Promise<SavedExplorerQuery> {
  const record: SavedExplorerQuery = {
    id: `local-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...input,
  };
  try {
    const remote = await fetchJSON<SavedExplorerQuery>('/api/data-explorer/saved', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    if (remote && remote.id) return remote;
  } catch {
    // ignore — local persistence below
  }
  const existing = readSavedStore();
  const next = [record, ...existing];
  writeSavedStore(next);
  return record;
}

export async function deleteExplorerQuery(id: string): Promise<void> {
  try {
    await fetchJSON<void>(`/api/data-explorer/saved/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  } catch {
    // ignore
  }
  const existing = readSavedStore();
  writeSavedStore(existing.filter((r) => r.id !== id));
}

export type { BindingView };
export type { SourceKind };
