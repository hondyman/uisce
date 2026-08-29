import { describe, it, expect } from 'vitest';
import {
  executeExplorer,
  previewExplorerSQL,
  generateDialectSQL,
  generateCodeSnippet,
} from '../../../features/data-explorer/services/dataExplorerApi';
import type {
  ExplorerSource,
  ExplorerQueryState,
} from '../../../features/data-explorer/types/dataExplorerTypes';

describe('Data Explorer Query Services', () => {
  const mockSource: ExplorerSource = {
    kind: 'business_object',
    id: 'oms.account',
    bindingId: 'default-binding',
    datasourceId: 'alpha-pg',
    displayName: 'Account & Wealth Management',
    fields: [
      {
        id: 'f_account_id',
        name: 'account_id',
        displayName: 'Account ID',
        technicalName: 'account_id',
        category: 'dimension',
        type: 'string',
        isCore: true,
      },
      {
        id: 'f_client_name',
        name: 'client_name',
        displayName: 'Client Name',
        technicalName: 'client_name',
        category: 'dimension',
        type: 'string',
        isCore: true,
      },
      {
        id: 'f_open_date',
        name: 'open_date',
        displayName: 'Open Date',
        technicalName: 'open_date',
        category: 'time',
        type: 'date',
        isCore: true,
      },
      {
        id: 'f_total_valuation',
        name: 'total_valuation',
        displayName: 'Total Valuation',
        technicalName: 'total_valuation',
        category: 'measure',
        type: 'number',
        defaultAggregation: 'SUM',
        isCore: true,
      },
    ],
  };

  it('generates valid synthesized preview SQL when fields are assigned', async () => {
    const queryState: ExplorerQueryState = {
      sourceId: 'oms.account',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'client_name' }],
      measures: [{ fieldId: 'total_valuation', agg: 'SUM' }],
      timeDimensions: [{ fieldId: 'open_date', granularity: 'month' }],
      calculations: [],
      parameters: [],
      filters: [{ fieldId: 'client_name', operator: 'equals', values: ['Acme Corp'] }],
      sorts: [{ fieldId: 'client_name', direction: 'asc' }],
      limit: 50,
    };

    const sql = await previewExplorerSQL(mockSource, queryState);
    expect(sql).toContain('SELECT');
    expect(sql).toContain('client_name');
    expect(sql).toContain('SUM(total_valuation)');
    expect(sql).toContain("DATE_TRUNC('month', open_date)");
    expect(sql).toContain('FROM\n  oms.account');
    expect(sql).toContain("WHERE\n  client_name = 'Acme Corp'");
    expect(sql).toContain('GROUP BY\n  client_name, open_date');
    expect(sql).toContain('LIMIT 50;');
  });

  it('handles field lookup by id or technicalName in SQL generation', async () => {
    const queryState: ExplorerQueryState = {
      sourceId: 'oms.account',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'f_client_name' }],
      measures: [{ fieldId: 'f_total_valuation', agg: 'AVG' }],
      timeDimensions: [],
      calculations: [],
      parameters: [],
      filters: [],
      sorts: [],
      limit: 25,
    };

    const sql = await previewExplorerSQL(mockSource, queryState);
    expect(sql).toContain('client_name');
    expect(sql).toContain('AVG(total_valuation)');
  });

  it('executes explorer and returns structured rows, columns, and SQL', async () => {
    const queryState: ExplorerQueryState = {
      sourceId: 'oms.account',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'client_name' }],
      measures: [{ fieldId: 'total_valuation', agg: 'SUM' }],
      timeDimensions: [],
      calculations: [],
      parameters: [],
      filters: [],
      sorts: [],
      limit: 15,
    };

    const result = await executeExplorer(mockSource, queryState);
    expect(result).toBeDefined();
    expect(result.columns.length).toBeGreaterThanOrEqual(2);
    expect(result.rows.length).toBeGreaterThan(0);
    expect(result.sql).toContain('SELECT');
    expect(result.rowCount).toBe(result.rows.length);
  });

  it('generates multi-dialect SQL (Snowflake, BigQuery, ClickHouse)', () => {
    const queryState: ExplorerQueryState = {
      sourceId: 'oms.account',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'client_name' }],
      measures: [{ fieldId: 'total_valuation', agg: 'SUM' }],
      timeDimensions: [{ fieldId: 'open_date', granularity: 'month' }],
      calculations: [],
      parameters: [],
      filters: [],
      sorts: [],
      limit: 100,
    };

    const snowflakeSql = generateDialectSQL(mockSource, queryState, 'snowflake');
    expect(snowflakeSql).toContain('SNOWFLAKE');
    expect(snowflakeSql).toContain('"CLIENT_NAME"');

    const bigQuerySql = generateDialectSQL(mockSource, queryState, 'bigquery');
    expect(bigQuerySql).toContain('BIGQUERY');
    expect(bigQuerySql).toContain('TIMESTAMP_TRUNC');
  });

  it('generates runnable code snippets for TypeScript, Python, and cURL', () => {
    const queryState: ExplorerQueryState = {
      sourceId: 'oms.account',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'client_name' }],
      measures: [{ fieldId: 'total_valuation', agg: 'SUM' }],
      timeDimensions: [],
      calculations: [],
      parameters: [],
      filters: [],
      sorts: [],
      limit: 50,
    };

    const tsSnippet = generateCodeSnippet(mockSource, queryState, 'typescript');
    expect(tsSnippet).toContain('@cubejs-client/core');
    expect(tsSnippet).toContain('oms_account.client_name');

    const pySnippet = generateCodeSnippet(mockSource, queryState, 'python');
    expect(pySnippet).toContain('import pandas as pd');
    expect(pySnippet).toContain('pd.DataFrame');

    const curlSnippet = generateCodeSnippet(mockSource, queryState, 'curl');
    expect(curlSnippet).toContain('curl -X POST');
  });
});
