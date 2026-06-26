import { useQuery } from '@tanstack/react-query';

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
      const res = await fetch(`/api/rules/${ruleId}/lineage?${query}`);
      if (!res.ok) throw new Error('Failed to load lineage');
      const json = await res.json();
      return (json.evaluations ?? []) as RuleLineageEntry[];
    },
    enabled: !!ruleId,
  });
}
