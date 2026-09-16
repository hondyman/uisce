import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface ScenarioLineageEntry {
  valuation_date: string;
  portfolio_id: string;
  pnl: number;
  etl_run_id: string;
}

export function useScenarioLineage(scenarioId: string, params: Record<string, any> = {}) {
  const query = new URLSearchParams(params);
  return useQuery({
    queryKey: ['scenario-lineage', scenarioId, params],
    queryFn: async () => {
      const res = await apiFetch(`/api/scenarios/${scenarioId}/lineage?${query}`);
      const json = await res.json();
      return (json.results ?? []) as ScenarioLineageEntry[];
    },
    enabled: !!scenarioId,
  });
}
