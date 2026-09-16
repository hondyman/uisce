import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface RuleLineageEntry {
  valuation_date: string;
  portfolio_id: string;
  status: string;
  metric_value: number;
  threshold_value: number;
  etl_run_id: string;
}

export function useRuleLineage(ruleId: string, params: Record<string, any> = {}) {
  const query = new URLSearchParams(params);
  return useQuery({
    queryKey: ['rule-lineage', ruleId, params],
    queryFn: async () => {
      const res = await apiFetch(`/api/rules/${ruleId}/lineage?${query}`);
      const json = await res.json();
      return (json.evaluations ?? []) as RuleLineageEntry[];
    },
    enabled: !!ruleId,
  });
}
