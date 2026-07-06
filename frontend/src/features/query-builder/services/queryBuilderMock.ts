/**
 * Development mock for the Alpha Query Builder endpoints.
 *
 * This mock is installed only in `import.meta.env.DEV`. It lets the Query
 * Builder UI demonstrate the QueryDef contract without requiring the backend
 * Resolution Engine / SQL Generator to be implemented first.
 *
 * When the backend endpoints are live, remove or disable this module.
 */

import type {
  QueryDef,
  SemanticTermView,
  PreviewResult,
  QueryExecuteResult,
} from '../types/queryDef';

let installed = false;

export function installQueryBuilderMock(): void {
  if (installed) return;
  installed = true;

  const originalFetch = window.fetch;

  window.fetch = async function mockFetch(
    input: RequestInfo | URL,
    init?: RequestInit
  ): Promise<Response> {
    const url = typeof input === 'string' ? input : input.toString();

    try {
      const termsMatch = url.match(/\/api\/business-objects\/([^/]+)\/terms\?bindingId=([^&]+)/);
      if (termsMatch && (!init || init.method === undefined || init.method === 'GET')) {
        const [, boId] = termsMatch;
        return mockBOTerms(boId);
      }

      if (url.endsWith('/api/query/preview') && init?.method === 'POST') {
        const body = JSON.parse((init.body as string) || '{}') as QueryDef;
        return mockPreview(body);
      }

      if (url.endsWith('/api/query/execute') && init?.method === 'POST') {
        const body = JSON.parse((init.body as string) || '{}') as QueryDef;
        return mockExecute(body);
      }
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('[QueryBuilderMock] Mock handling failed:', err);
    }

    return originalFetch(input, init);
  };
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function mockBOTerms(boId: string): Response {
  const terms: SemanticTermView[] = getMockTermsForBO(boId);
  return jsonResponse({ terms });
}

function getMockTermsForBO(boId: string): SemanticTermView[] {
  const orderTerms: SemanticTermView[] = [
    {
      termNodeId: 'order_id',
      termKey: 'order_id',
      termName: 'Order ID',
      displayName: 'Order ID',
      dataType: 'integer',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
    {
      termNodeId: 'order_date',
      termKey: 'order_date',
      termName: 'Order Date',
      displayName: 'Order Date',
      dataType: 'date',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
    {
      termNodeId: 'customer_id',
      termKey: 'customer_id',
      termName: 'Customer ID',
      displayName: 'Customer ID',
      dataType: 'integer',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
    {
      termNodeId: 'order_total',
      termKey: 'order_total',
      termName: 'Order Total',
      displayName: 'Order Total',
      dataType: 'decimal',
      role: 'MEASURE',
      bindingStatus: 'RESOLVED',
      defaultAggregation: 'SUM',
    },
    {
      termNodeId: 'order_status',
      termKey: 'order_status',
      termName: 'Order Status',
      displayName: 'Order Status',
      dataType: 'string',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
    {
      termNodeId: 'shipping_cost',
      termKey: 'shipping_cost',
      termName: 'Shipping Cost',
      displayName: 'Shipping Cost',
      dataType: 'decimal',
      role: 'MEASURE',
      bindingStatus: 'RESOLVED',
      defaultAggregation: 'SUM',
    },
  ];

  const customerTerms: SemanticTermView[] = [
    {
      termNodeId: 'customer_id',
      termKey: 'customer_id',
      termName: 'Customer ID',
      displayName: 'Customer ID',
      dataType: 'integer',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
    {
      termNodeId: 'customer_name',
      termKey: 'customer_name',
      termName: 'Customer Name',
      displayName: 'Customer Name',
      dataType: 'string',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
    {
      termNodeId: 'customer_segment',
      termKey: 'customer_segment',
      termName: 'Customer Segment',
      displayName: 'Customer Segment',
      dataType: 'string',
      role: 'DIMENSION',
      bindingStatus: 'RESOLVED',
    },
  ];

  if (/order/i.test(boId)) return orderTerms;
  if (/customer/i.test(boId)) return customerTerms;

  // Generic fallback
  return [
    ...orderTerms,
    ...customerTerms,
    {
      termNodeId: 'calculated_revenue_per_customer',
      termKey: 'calculated_revenue_per_customer',
      termName: 'Revenue per Customer',
      displayName: 'Revenue per Customer',
      dataType: 'decimal',
      role: 'CALCULATED',
      bindingStatus: 'RESOLVED',
      defaultAggregation: 'AVG',
    },
  ];
}

function mockPreview(queryDef: QueryDef): Response {
  const sql = generateMockSQL(queryDef);
  const plan: PreviewResult['plan'] = {
    tenantId: queryDef.context.tenantId,
    root: {
      id: 'root_orchestration',
      nodeType: 'Coordinator',
      dataSource: 'UisceFederatedEngine',
      cost: 0,
      isSecured: true,
      details: { enforced_tenant_id: queryDef.context.tenantId },
      children: [
        {
          id: 'step_0',
          nodeType: 'IndexScan',
          dataSource: 'Postgres',
          cost: 24.5,
          isSecured: true,
          details: {
            index_pruned: true,
            estimated_rows: 1240,
            table: 'orders',
          },
          children: [],
        },
      ],
    },
    metrics: {
      totalLatencyMs: 12,
      dataScannedBytes: 2048,
    },
    warnings: [
      'Alpha mock plan: real backend warnings will appear here for high-cost nodes, sequential scans, or excessive data scans.',
    ],
  };
  return jsonResponse({ sql, dialect: 'postgres', parameters: [queryDef.context.tenantId], plan } as PreviewResult);
}

function mockExecute(queryDef: QueryDef): Response {
  const sql = generateMockSQL(queryDef);
  const { columns, rows } = generateMockResults(queryDef);
  return jsonResponse({
    sql,
    columns,
    rows,
    rowCount: rows.length,
    executionTimeMs: Math.floor(Math.random() * 120) + 12,
  } as QueryExecuteResult);
}

function generateMockSQL(queryDef: QueryDef): string {
  const { context, query } = queryDef;

  const selectParts: string[] = [];
  query.dimensions.forEach((d) => {
    selectParts.push(`  "${d.termNodeId}" AS "${d.alias}"`);
  });
  query.measures.forEach((m) => {
    const expr = m.agg && m.agg !== 'NONE' ? `${m.agg}("${m.termNodeId}")` : `"${m.termNodeId}"`;
    selectParts.push(`  ${expr} AS "${m.alias}"`);
  });

  if (selectParts.length === 0) {
    return '-- Add dimensions or measures to generate SQL';
  }

  const groupByAliases = query.groupBy?.length
    ? query.groupBy
    : query.dimensions.map((d) => d.alias);

  let sql = `SELECT\n${selectParts.join(',\n')}\nFROM "${context.boId}"\nWHERE "tenant_id" = ?`;

  const params: string[] = [context.tenantId];

  query.filters.forEach((f) => {
    const { clause, value } = renderFilterClause(f);
    sql += `\n  AND ${clause}`;
    if (value !== undefined) params.push(String(value));
  });

  if (groupByAliases.length > 0) {
    sql += `\nGROUP BY ${groupByAliases.map((a) => `"${a}"`).join(', ')}`;
  }

  if (query.limit) {
    sql += `\nLIMIT ${query.limit}`;
  }

  sql += ';';

  // Append parameter summary as a comment for the preview pane.
  sql += `\n\n-- Parameters: [${params.join(', ')}]`;

  return sql;
}

function renderFilterClause(filter: QueryDef['query']['filters'][number]): {
  clause: string;
  value?: unknown;
} {
  const col = `"${filter.termNodeId}"`;
  switch (filter.operator) {
    case 'eq':
      return { clause: `${col} = ?`, value: filter.value };
    case 'neq':
      return { clause: `${col} != ?`, value: filter.value };
    case 'gt':
      return { clause: `${col} > ?`, value: filter.value };
    case 'gte':
      return { clause: `${col} >= ?`, value: filter.value };
    case 'lt':
      return { clause: `${col} < ?`, value: filter.value };
    case 'lte':
      return { clause: `${col} <= ?`, value: filter.value };
    case 'contains':
      return { clause: `${col} LIKE ?`, value: `%${filter.value}%` };
    case 'starts_with':
      return { clause: `${col} LIKE ?`, value: `${filter.value}%` };
    case 'ends_with':
      return { clause: `${col} LIKE ?`, value: `%${filter.value}` };
    case 'in':
      return {
        clause: `${col} IN (${Array.isArray(filter.value) ? filter.value.map(() => '?').join(', ') : '?'})`,
        value: filter.value,
      };
    case 'not_in':
      return {
        clause: `${col} NOT IN (${Array.isArray(filter.value) ? filter.value.map(() => '?').join(', ') : '?'})`,
        value: filter.value,
      };
    case 'is_null':
      return { clause: `${col} IS NULL` };
    case 'is_not_null':
      return { clause: `${col} IS NOT NULL` };
    case 'between':
      return { clause: `${col} BETWEEN ? AND ?`, value: filter.value };
    default:
      return { clause: `${col} = ?`, value: filter.value };
  }
}

function generateMockResults(queryDef: QueryDef): {
  columns: { name: string; type?: string }[];
  rows: Record<string, unknown>[];
} {
  const columns = [
    ...queryDef.query.dimensions.map((d) => ({ name: d.alias, type: 'string' })),
    ...queryDef.query.measures.map((m) => ({ name: m.alias, type: 'number' })),
  ];

  if (columns.length === 0) {
    return { columns: [], rows: [] };
  }

  const rowCount = 5;
  const rows: Record<string, unknown>[] = [];

  for (let i = 0; i < rowCount; i += 1) {
    const row: Record<string, unknown> = {};
    queryDef.query.dimensions.forEach((d) => {
      row[d.alias] = `${d.alias}_value_${i + 1}`;
    });
    queryDef.query.measures.forEach((m) => {
      row[m.alias] = Math.round((Math.random() * 10000 + 100) * 100) / 100;
    });
    rows.push(row);
  }

  return { columns, rows };
}
