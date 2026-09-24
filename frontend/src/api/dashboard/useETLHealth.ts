import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../lib/apiClient';

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

export function useETLHealth() {
  return useQuery({
    queryKey: ['dashboard-etl-health'],
    queryFn: async () => {
      const res = await apiFetch('/api/dashboard/etl-health');
      const data = await res.json();
      return data as ETLHealth;
    },
  });
}
