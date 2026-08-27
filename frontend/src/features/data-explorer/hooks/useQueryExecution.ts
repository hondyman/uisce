import { useMutation } from '@tanstack/react-query';
import apiClient from '../../../utils/apiClient';
import type { ExplorerQueryState } from '../types/dataExplorerTypes';

export interface QueryExecutionResult {
  compiledSql: string;
  targetBackend: string;
  estimatedCostUsd: number;
  complexityScore: number;
  columns: string[];
  rows: Record<string, unknown>[];
}

export function useQueryExecution() {
  return useMutation({
    mutationFn: async (queryState: ExplorerQueryState) => {
      const payload = {
        boKey: queryState.sourceId,
        dimensions: queryState.dimensions.map((d) => d.fieldId),
        measures: queryState.measures.map((m) => m.fieldId),
        filters: queryState.filters.map((f) => ({
          fieldKey: f.fieldId,
          operator: f.operator,
          values: f.values,
        })),
        limit: queryState.limit || 100,
      };

      return apiClient<QueryExecutionResult>('/api/v1/query/preview', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
    },
  });
}
