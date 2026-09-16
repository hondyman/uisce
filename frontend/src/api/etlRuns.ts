import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface ETLRun {
  etl_run_id: string;
  tenant_id: string;
  valuation_date: string;
  started_at: string;
  completed_at: string | null;
  status: string;
  rules_evaluated: number;
  scenarios_evaluated: number;
  wasm_version: string;
  orchestrator_version: string;
  error_summary?: string;
}

export interface ETLRunParams {
  tenant_id?: string;
  status?: string;
  from?: string;
  to?: string;
  limit?: number;
}

export function useETLRuns(params: ETLRunParams) {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null) query.set(k, String(v));
  });

  return useQuery({
    queryKey: ['etl-runs', params],
    queryFn: async () => {
      const res = await apiFetch(`/api/etl-runs?${query.toString()}`);
      const json = await res.json();
      return (json.runs ?? []) as ETLRun[];
    },
  });
}

export function useETLRun(id: string) {
  return useQuery({
    queryKey: ['etl-run', id],
    queryFn: async () => {
      const res = await apiFetch(`/api/etl-runs/${id}`);
      return (await res.json()) as ETLRun;
    },
    enabled: !!id,
  });
}
