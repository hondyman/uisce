import { useQuery } from '@tanstack/react-query';

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
      const res = await fetch(`/api/scenarios/${scenarioId}/lineage?${query}`);
      if (!res.ok) throw new Error('Failed to load lineage');
      const json = await res.json();
      return (json.results ?? []) as ScenarioLineageEntry[];
    },
    enabled: !!scenarioId,
  });
}
