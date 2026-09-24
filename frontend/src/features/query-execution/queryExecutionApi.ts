/**
 * queryExecutionApi - the only allowed SQL-fetching surface for any
 * consumer of QueryResultsPanel.
 *
 * Wraps previewQuery / executeQuery from features/query-builder/services/
 * queryBuilderApi.ts and funnels responses through the canonical adapters.
 * Never constructs SQL. Saved-query run paths reuse runSavedQuery from
 * savedQueryApi and adapt through savedQueryResultToSet.
 */
import {
  previewQuery,
  executeQuery,
} from '../query-builder/services/queryBuilderApi';
import type {
  QueryDef,
  PreviewResult,
} from '../query-builder/types/queryDef';
import { liveQueryResultToSet } from './adapters';
import type { QueryResultSet } from './types';

export interface CompiledSql {
  sql: string;
  dialect?: string;
}

export async function fetchCompiledSql(def: QueryDef): Promise<CompiledSql> {
  const res: PreviewResult = await previewQuery(def);
  return { sql: res.sql, dialect: res.dialect };
}

export async function runExecute(def: QueryDef): Promise<QueryResultSet> {
  const res = await executeQuery(def);
  return liveQueryResultToSet(res);
}
