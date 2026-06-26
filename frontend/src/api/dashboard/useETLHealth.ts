import { useQuery } from '@tanstack/react-query';

export interface ETLRun {
  etl_run_id: string;
  status: string;
  duration_ms: number;
  rules_evaluated: number;
  scenarios_evaluated: number;
  wasm_version: string;
  orchestrator_version: string;
  completed_at: string;
}

export interface ETLHealth {
  last_run: ETLRun;
  success_rate: number;
  avg_duration_ms: number;
  total_runs: number;
}

export function useETLHealth(tenantId: string) {
  return useQuery({
    queryKey: ['dashboard-etl-health', tenantId],
    queryFn: async () => {
      const res = await fetch(`/api/dashboard/etl-health?tenant_id=${tenantId}`);
      if (!res.ok) throw new Error('Failed to load ETL health');
      const data = await res.json();
      return data as ETLHealth;
    },
    enabled: !!tenantId,
  });
}
