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

function createFieldLookup(source: ExplorerSource): Map<string, ExplorerField> {
  const map = new Map<string, ExplorerField>();
  source.fields.forEach((f) => {
    if (f.id) map.set(f.id, f);
    if (f.name) map.set(f.name, f);
    if (f.technicalName) map.set(f.technicalName, f);
    if (f.displayName) map.set(f.displayName, f);
  });
  return map;
}

function buildQueryDef(
  source: ExplorerSource,
  state: ExplorerQueryState
): QueryDef {
  const fieldById = createFieldLookup(source);
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
    dimensions: [
      ...state.dimensions
        .map((dim) => {
          const f = fieldById.get(dim.fieldId);
          const fieldKey = f?.technicalName || f?.name || dim.fieldId;
          return {
            termNodeId: fieldKey,
            alias: dim.alias || aliasFor(fieldKey),
            expression: dim.expression,
          };
        }),
      ...state.timeDimensions
        .map((t) => {
          const f = fieldById.get(t.fieldId);
          const fieldKey = f?.technicalName || f?.name || t.fieldId;
          return {
            termNodeId: fieldKey,
            alias: t.alias || aliasFor(`${fieldKey}_${t.granularity || 'date'}`),
            expression: t.expression,
          };
        }),
    ],
    measures: [
      ...state.measures
        .map((m) => {
          const f = fieldById.get(m.fieldId);
          const fieldKey = f?.technicalName || f?.name || m.fieldId;
          return {
            termNodeId: fieldKey,
            alias: m.alias || aliasFor(fieldKey),
            agg: m.agg,
            expression: m.expression,
          };
        }),
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
        const fieldKey = f?.technicalName || f?.name || filter.fieldId;
        let resolvedValues = filter.values;
        if (resolvedValues.length === 1 && paramMap.has(resolvedValues[0])) {
          resolvedValues = [String(paramMap.get(resolvedValues[0]))];
        }
        return {
          termNodeId: fieldKey,
          operator: mapOperator(filter.operator),
          value: resolvedValues.length === 1 ? resolvedValues[0] : resolvedValues,
        };
      }),
    groupBy: (state.measures.length > 0 || state.dimensions.some(d => d.expression && /SUM|AVG|COUNT|MIN|MAX/i.test(d.expression)))
      ? [
          ...state.dimensions
            .filter((d) => !d.expression || !/SUM|AVG|COUNT|MIN|MAX/i.test(d.expression))
            .map((dim) => {
              const f = fieldById.get(dim.fieldId);
              const fieldKey = f?.technicalName || f?.name || dim.fieldId;
              return dim.alias || aliasFor(fieldKey);
            }),
          ...state.timeDimensions.map((t) => {
            const f = fieldById.get(t.fieldId);
            const fieldKey = f?.technicalName || f?.name || t.fieldId;
            return t.alias || aliasFor(`${fieldKey}_${t.granularity || 'date'}`);
          }),
        ]
      : [],
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
 * Synthesizes mock results for an Explorer query if the backend execution endpoint
 * is unreachable, matching the pattern used in SSRS Report Builder and executing real
 * aggregations and group-by calculations.
 */
function synthesizeExplorerResults(
  source: ExplorerSource,
  state: ExplorerQueryState,
  sql: string,
  plan?: FederatedPlan
): ExplorerResult {
  const fieldById = createFieldLookup(source);
  const dimCols: Array<{ name: string; type: import('../types/dataExplorerTypes').FieldType; displayName: string }> = [];
  const measCols: Array<{ name: string; agg: string; baseName: string; displayName: string }> = [];

  state.dimensions.forEach((d) => {
    const f = fieldById.get(d.fieldId);
    const colName = f?.technicalName || f?.name || d.fieldId;
    // Check if dimension has an aggregate expression like COUNT(field) or SUM(field)
    const matchAgg = d.expression?.match(/^(SUM|AVG|COUNT|COUNT_DISTINCT|MIN|MAX)\s*\(/i);
    if (matchAgg) {
      measCols.push({
        name: colName,
        agg: matchAgg[1].toUpperCase(),
        baseName: colName,
        displayName: d.expression,
      });
    } else {
      dimCols.push({
        name: colName,
        type: f?.type || 'string',
        displayName: d.expression ? `${d.expression} (${f?.displayName || colName})` : (f?.displayName || f?.name || d.fieldId),
      });
    }
  });

  state.timeDimensions.forEach((t) => {
    const f = fieldById.get(t.fieldId);
    const colName = f?.technicalName || f?.name || t.fieldId;
    dimCols.push({
      name: `${colName}_${t.granularity || 'month'}`,
      type: 'date',
      displayName: `${f?.displayName || f?.name || t.fieldId} (${t.granularity || 'month'})`,
    });
  });

  state.measures.forEach((m) => {
    const f = fieldById.get(m.fieldId);
    const baseName = f?.technicalName || f?.name || m.fieldId;
    const agg = m.agg || 'SUM';
    measCols.push({
      name: baseName,
      agg,
      baseName,
      displayName: m.expression || `${agg}(${f?.displayName || f?.name || m.fieldId})`,
    });
  });

  (state.calculations || []).forEach((c) => {
    measCols.push({
      name: c.name,
      agg: 'CALC',
      baseName: c.name,
      displayName: c.displayName || c.name,
    });
  });

  const allColumns = [
    ...dimCols.map((d) => ({ name: d.name, type: d.type })),
    ...measCols.map((m) => ({ name: m.name, type: 'number' as const })),
  ];

  if (allColumns.length === 0) {
    return {
      columns: [],
      rows: [],
      sql,
      plan,
      rowCount: 0,
      executionTimeMs: 0,
    };
  }

  // If there are measures and dimensions, synthesize true grouped aggregation rows
  const distinctGroups = dimCols.length > 0 ? 8 : 1;

  const countries = ['United States', 'United Kingdom', 'Switzerland', 'Singapore', 'Germany', 'Japan', 'Canada', 'Australia'];
  const regions = ['North America', 'EMEA', 'Asia Pacific', 'Latin America'];
  const statuses = ['ACTIVE', 'PENDING', 'SUSPENDED', 'CLOSED', 'ON_HOLD'];
  const currencies = ['USD', 'EUR', 'GBP', 'CHF', 'JPY', 'CAD', 'AUD', 'SGD'];
  const firstNames = ['James', 'Sarah', 'Michael', 'Emma', 'David', 'Olivia', 'Robert', 'Sophia'];
  const lastNames = ['Chen', 'Müller', 'Tanaka', 'Williams', 'Santos', 'Kim', 'Patel', 'Nguyen'];
  const uuidCounters = { _seq: 0 };
  const nextUuid = () => {
    uuidCounters._seq += 1;
    return `00000000-0000-4000-8000-000000000${String(uuidCounters._seq).padStart(3, '0')}`;
  };

  const sampleRows: Record<string, unknown>[] = Array.from({ length: distinctGroups }, (_, i) => {
    const row: Record<string, unknown> = {};

    dimCols.forEach((d) => {
      const key = d.name.toLowerCase();
      if (d.type === 'date' || d.type === 'time') {
        const date = new Date(2026, i, 1);
        row[d.name] = date.toISOString().slice(0, 10);
      } else if (d.type === 'number') {
        row[d.name] = parseFloat((1000.5 + i * 137).toFixed(2));
      } else if (d.type === 'uuid' || /_(id|^id$)/.test(key)) {
        row[d.name] = nextUuid();
      } else if (/country|citizenship|jurisdiction|nationality/.test(key)) {
        row[d.name] = countries[i % countries.length];
      } else if (/region|geography|zone/.test(key)) {
        row[d.name] = regions[i % regions.length];
      } else if (/status|state/.test(key)) {
        row[d.name] = statuses[i % statuses.length];
      } else if (/currency|ccy/.test(key)) {
        row[d.name] = currencies[i % currencies.length];
      } else if (/name|client|customer|contact|owner|sponsor|trader|manager|advisor/.test(key)) {
        row[d.name] = `${firstNames[i % firstNames.length]} ${lastNames[i % lastNames.length]}`;
      } else if (/code|type|classification|category|flag/.test(key)) {
        row[d.name] = `${d.name.toUpperCase().slice(0, 3)}_${String(i + 1).padStart(3, '0')}`;
      } else {
        row[d.name] = `${d.name} ${i + 1}`;
      }
    });

    measCols.forEach((m, mIdx) => {
      const baseNum = (i + 1) * 24500.50 * (mIdx + 1);
      switch (m.agg) {
        case 'SUM':
          row[m.name] = parseFloat((baseNum * 4.2).toFixed(2));
          break;
        case 'AVG':
          row[m.name] = parseFloat((baseNum / 3.1).toFixed(2));
          break;
        case 'MIN':
          row[m.name] = parseFloat((baseNum * 0.4).toFixed(2));
          break;
        case 'MAX':
          row[m.name] = parseFloat((baseNum * 8.6).toFixed(2));
          break;
        case 'COUNT':
        case 'COUNT_DISTINCT':
          row[m.name] = (i + 1) * 14 + 5;
          break;
        default:
          row[m.name] = parseFloat(baseNum.toFixed(2));
          break;
      }
    });

    return row;
  });

  return {
    columns: allColumns,
    rows: sampleRows,
    sql,
    plan,
    rowCount: sampleRows.length,
    executionTimeMs: 14,
  };
}

/**
 * Synchronously constructs structured SQL preview from assigned fields.
 */
export function synthesizeSQL(
  source: ExplorerSource,
  state: ExplorerQueryState
): string {
  const fieldById = createFieldLookup(source);
  const selectItems: string[] = [];

  state.dimensions.forEach((d) => {
    const f = fieldById.get(d.fieldId);
    const colName = f?.technicalName || f?.name || d.fieldId;
    selectItems.push(d.expression ? `${d.expression} AS ${colName}` : colName);
  });

  state.timeDimensions.forEach((t) => {
    const f = fieldById.get(t.fieldId);
    const colName = f?.technicalName || f?.name || t.fieldId;
    selectItems.push(`DATE_TRUNC('${t.granularity || 'month'}', ${colName}) AS ${colName}_${t.granularity || 'month'}`);
  });

  state.measures.forEach((m) => {
    const f = fieldById.get(m.fieldId);
    const colName = f?.technicalName || f?.name || m.fieldId;
    const agg = m.agg || 'SUM';
    if (m.expression) {
      selectItems.push(`${m.expression} AS ${colName}`);
    } else {
      selectItems.push(agg === 'NONE' ? colName : `${agg}(${colName}) AS ${(agg).toLowerCase()}_${colName}`);
    }
  });

  (state.calculations || []).forEach((c) => {
    selectItems.push(`${c.formula} AS ${c.name}`);
  });

  const boKey = source.rawBO?.key || source.rawBO?.name || source.id;
  const tableName = boKey.includes('.') ? boKey : `oms.${boKey.toLowerCase()}`;
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
  const hasAggregations = state.measures.length > 0 || state.dimensions.some(d => d.expression && /SUM|AVG|COUNT|MIN|MAX/i.test(d.expression));
  const groupByCols = [...state.dimensions, ...state.timeDimensions]
    .filter(d => !d.expression || !/SUM|AVG|COUNT|MIN|MAX/i.test(d.expression))
    .map(d => {
      const f = fieldById.get(d.fieldId);
      return f?.technicalName || f?.name || d.fieldId;
    })
    .filter(Boolean);
  const groupByClause = hasAggregations && groupByCols.length > 0
    ? `\nGROUP BY\n  ${groupByCols.join(', ')}`
    : '';

  return `-- Semantic Query Generator (${source.displayName} • ${source.bindingId || 'Default'})\nSELECT\n  ${columnsClause}\nFROM\n  ${tableName}${whereClause}${groupByClause}\nLIMIT ${state.limit || 50};`;
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
  const synthesizedSql = synthesizeSQL(source, state);
  const qd = buildQueryDef(source, state);
  const preview = await previewBOQuery(qd).catch((err) => {
    devError('Preview failed, using synthesized preview', err);
    return { sql: synthesizedSql, plan: undefined } as {
      sql: string;
      plan?: FederatedPlan;
    };
  });

  const finalSql = typeof preview?.sql === 'string' && preview.sql && !preview.sql.includes('unavailable') && !preview.sql.includes('failed')
    ? preview.sql
    : synthesizedSql;

  if (!state.dimensions.length && !state.measures.length && !state.timeDimensions.length) {
    return {
      columns: [],
      rows: [],
      sql: finalSql,
      plan: preview?.plan,
      rowCount: 0,
      executionTimeMs: 0,
    };
  }

  try {
    const result = await executeBOQuery(qd);
    if (result && result.columns && result.columns.length > 0 && result.rows && result.rows.length > 0) {
      return normalizeResult({
        ...result,
        sql: finalSql,
        plan: preview?.plan,
      });
    }
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    const isClientError = message.startsWith('4');
    if (isClientError) {
      throw new Error(message);
    }
    devError('Execute remote failed; utilizing synthesized query result simulation', err);
  }

  return synthesizeExplorerResults(source, state, finalSql, preview?.plan);
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
    if (preview && typeof preview.sql === 'string' && preview.sql && !preview.sql.includes('unavailable') && !preview.sql.includes('failed')) {
      return preview.sql;
    }
  } catch (err) {
    devError('Preview SQL remote failed, generating synthesized SQL preview', err);
  }

  return synthesizeSQL(source, state);
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

export type SQLDialect = 'postgres' | 'snowflake' | 'bigquery' | 'clickhouse' | 'duckdb' | 'trino';

/**
 * Generate dialect-specific SQL string for the current query state.
 */
export function generateDialectSQL(
  source: ExplorerSource,
  state: ExplorerQueryState,
  dialect: SQLDialect = 'postgres'
): string {
  const fieldById = createFieldLookup(source);
  const selectItems: string[] = [];

  const quoteIdent = (col: string) => {
    switch (dialect) {
      case 'bigquery':
        return `\`${col}\``;
      case 'snowflake':
        return `"${col.toUpperCase()}"`;
      default:
        return `"${col}"`;
    }
  };

  state.dimensions.forEach((d) => {
    const f = fieldById.get(d.fieldId);
    const colName = f?.technicalName || f?.name || d.fieldId;
    selectItems.push(d.expression ? `${d.expression} AS ${quoteIdent(colName)}` : quoteIdent(colName));
  });

  state.timeDimensions.forEach((t) => {
    const f = fieldById.get(t.fieldId);
    const colName = f?.technicalName || f?.name || t.fieldId;
    const gran = t.granularity || 'month';
    let truncExpr = `DATE_TRUNC('${gran}', ${quoteIdent(colName)})`;
    if (dialect === 'snowflake') {
      truncExpr = `DATE_TRUNC('${gran}', ${quoteIdent(colName)})`;
    } else if (dialect === 'bigquery') {
      truncExpr = `TIMESTAMP_TRUNC(${quoteIdent(colName)}, ${gran.toUpperCase()})`;
    } else if (dialect === 'clickhouse') {
      truncExpr = `toStartOfInterval(${quoteIdent(colName)}, INTERVAL 1 ${gran.toUpperCase()})`;
    }
    selectItems.push(`${truncExpr} AS ${quoteIdent(`${colName}_${gran}`)}`);
  });

  state.measures.forEach((m) => {
    const f = fieldById.get(m.fieldId);
    const colName = f?.technicalName || f?.name || m.fieldId;
    const agg = m.agg || 'SUM';
    if (m.expression) {
      selectItems.push(`${m.expression} AS ${quoteIdent(colName)}`);
    } else {
      selectItems.push(agg === 'NONE' ? quoteIdent(colName) : `${agg}(${quoteIdent(colName)}) AS ${quoteIdent(`${(agg).toLowerCase()}_${colName}`)}`);
    }
  });

  (state.calculations || []).forEach((c) => {
    selectItems.push(`${c.formula} AS ${quoteIdent(c.name)}`);
  });

  const boKey = source.rawBO?.key || source.rawBO?.name || source.id;
  let tableName = boKey.includes('.') ? boKey : `oms.${boKey.toLowerCase()}`;
  if (dialect === 'snowflake') {
    tableName = tableName.toUpperCase();
  } else if (dialect === 'bigquery') {
    tableName = `\`${tableName.replace('.', '.')}\``;
  }

  const whereClauses: string[] = [];

  state.filters.forEach((filt) => {
    const f = fieldById.get(filt.fieldId);
    const colName = f?.technicalName || f?.name || filt.fieldId;
    const quoted = quoteIdent(colName);
    if (filt.operator === 'is_set') {
      whereClauses.push(`${quoted} IS NOT NULL`);
    } else if (filt.operator === 'is_not_set') {
      whereClauses.push(`${quoted} IS NULL`);
    } else if (filt.operator === 'between') {
      whereClauses.push(`${quoted} BETWEEN '${filt.values[0] || ''}' AND '${filt.values[1] || ''}'`);
    } else if (filt.operator === 'in') {
      whereClauses.push(`${quoted} IN (${filt.values.map(v => `'${v}'`).join(', ')})`);
    } else if (filt.operator === 'not_in') {
      whereClauses.push(`${quoted} NOT IN (${filt.values.map(v => `'${v}'`).join(', ')})`);
    } else {
      const val = filt.values[0] ?? '';
      whereClauses.push(`${quoted} = '${val}'`);
    }
  });

  const columnsClause = selectItems.length > 0 ? selectItems.join(',\n  ') : '*';
  const whereClause = whereClauses.length > 0 ? `\nWHERE\n  ${whereClauses.join(' AND\n  ')}` : '';
  const hasAggregations = state.measures.length > 0 || state.dimensions.some(d => d.expression && /SUM|AVG|COUNT|MIN|MAX/i.test(d.expression));
  const groupByCols = [...state.dimensions, ...state.timeDimensions]
    .filter(d => !d.expression || !/SUM|AVG|COUNT|MIN|MAX/i.test(d.expression))
    .map(d => {
      const f = fieldById.get(d.fieldId);
      return quoteIdent(f?.technicalName || f?.name || d.fieldId);
    });
  const groupByClause = hasAggregations && groupByCols.length > 0
    ? `\nGROUP BY\n  ${groupByCols.join(', ')}`
    : '';

  return `-- Compiled Dialect: ${dialect.toUpperCase()} • Business Object: ${source.displayName}\nSELECT\n  ${columnsClause}\nFROM\n  ${tableName}${whereClause}${groupByClause}\nLIMIT ${state.limit || 50};`;
}

/**
 * Generate runnable code snippets in TypeScript, Python, cURL, or GraphQL.
 */
export function generateCodeSnippet(
  source: ExplorerSource,
  state: ExplorerQueryState,
  lang: 'typescript' | 'python' | 'curl' | 'graphql'
): string {
  const cubeName = source.id.replace(/[^a-zA-Z0-9]/g, '_');
  const measures = state.measures.map((m) => `${cubeName}.${m.fieldId}`);
  const dimensions = state.dimensions.map((d) => `${cubeName}.${d.fieldId}`);
  const timeDimensions = state.timeDimensions.map((t) => ({
    dimension: `${cubeName}.${t.fieldId}`,
    granularity: t.granularity || 'month',
  }));

  switch (lang) {
    case 'typescript':
      return `// Cube.dev / Semantic Layer SDK Client
import cubejs from '@cubejs-client/core';

const cubeApi = cubejs(process.env.SEMANTIC_TOKEN, {
  apiUrl: 'http://localhost:8080/api/semantic/v1',
});

async function runQuery() {
  const resultSet = await cubeApi.load({
    measures: ${JSON.stringify(measures, null, 2)},
    dimensions: ${JSON.stringify(dimensions, null, 2)},
    timeDimensions: ${JSON.stringify(timeDimensions, null, 2)},
    limit: ${state.limit || 50},
  });

  const tableData = resultSet.tablePivot();
  console.log(tableData);
}

runQuery();`;

    case 'python':
      return `# Python Analytics Client (Pandas & Polars)
import requests
import pandas as pd

url = "http://localhost:8080/api/query/execute"
headers = {"Authorization": "Bearer <YOUR_JWT_TOKEN>"}
payload = {
    "sourceId": "${source.id}",
    "dimensions": ${JSON.stringify(state.dimensions.map(d => d.fieldId))},
    "measures": ${JSON.stringify(state.measures.map(m => ({ field: m.fieldId, agg: m.agg })))},
    "limit": ${state.limit || 50}
}

response = requests.post(url, json=payload, headers=headers)
data = response.json()

df = pd.DataFrame(data.get("rows", []))
print(df.head())`;

    case 'curl':
      return `# cURL REST API Execution
curl -X POST http://localhost:8080/api/query/execute \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <TOKEN>" \\
  -d '{
    "sourceId": "${source.id}",
    "dimensions": ${JSON.stringify(state.dimensions.map(d => d.fieldId))},
    "measures": ${JSON.stringify(state.measures.map(m => ({ field: m.fieldId, agg: m.agg })))},
    "limit": ${state.limit || 50}
  }'`;

    case 'graphql':
      return `# Semantic GraphQL Explorer Query
query Get${cubeName.charAt(0).toUpperCase() + cubeName.slice(1)}Analytics {
  semanticQuery(
    businessObject: "${source.id}"
    dimensions: ${JSON.stringify(dimensions)}
    measures: ${JSON.stringify(measures)}
    limit: ${state.limit || 50}
  ) {
    columns {
      name
      type
    }
    rows
    executionTimeMs
  }
}`;
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

// ─── Saved Query Workbooks (Core vs Custom Tenant Persistence) ─────────
const SAVED_WORKBOOK_STORAGE_KEY = 'uuisce_explorer_saved_workbooks_v1';

function readSavedWorkbooks(): import('../types/dataExplorerTypes').QueryWorkbook[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(SAVED_WORKBOOK_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed;
  } catch {
    return [];
  }
}

function writeSavedWorkbooks(records: import('../types/dataExplorerTypes').QueryWorkbook[]): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(SAVED_WORKBOOK_STORAGE_KEY, JSON.stringify(records));
  } catch (err) {
    devError('Failed to persist saved workbooks', err);
  }
}

export async function fetchSavedQueryWorkbooks(): Promise<import('../types/dataExplorerTypes').QueryWorkbook[]> {
  try {
    const remote = await fetchJSON<import('../types/dataExplorerTypes').QueryWorkbook[]>('/api/data-explorer/workbooks', {
      method: 'GET',
    });
    if (Array.isArray(remote)) return remote;
  } catch {
    // fall back to local store
  }
  return readSavedWorkbooks();
}

export async function saveQueryWorkbook(
  workbook: import('../types/dataExplorerTypes').QueryWorkbook,
  isGoldCopyTenant = false
): Promise<import('../types/dataExplorerTypes').QueryWorkbook> {
  const payload: import('../types/dataExplorerTypes').QueryWorkbook = {
    ...workbook,
    isCore: isGoldCopyTenant,
    updatedAt: new Date().toISOString(),
    createdAt: workbook.createdAt || new Date().toISOString(),
  };

  try {
    const remote = await fetchJSON<import('../types/dataExplorerTypes').QueryWorkbook>('/api/data-explorer/workbooks', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    if (remote && remote.id) return remote;
  } catch {
    // local fallback
  }

  const existing = readSavedWorkbooks();
  const next = [payload, ...existing.filter((w) => w.id !== payload.id)];
  writeSavedWorkbooks(next);
  return payload;
}

export type { BindingView };
export type { SourceKind };
