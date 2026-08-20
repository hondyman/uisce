import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface DriftRun {
  id: string;
  tenant_id: string;
  policy_id: string;
  status: 'running' | 'completed' | 'failed';
  started_at: string;
  completed_at?: string;
  drift_detected: boolean;
  drift_summary?: any;
  errors?: string[];
}

export interface DriftComparison {
  id: string;
  source_run_id: string;
  target_run_id: string;
  policy_id: string;
  differences: any;
  created_at: string;
}

export const driftKeys = {
  all: ['drift'] as const,
  runs: () => [...driftKeys.all, 'runs'] as const,
  run: (id: string) => [...driftKeys.all, 'run', id] as const,
  comparisons: () => [...driftKeys.all, 'comparisons'] as const,
};

export function useDriftRuns() {
  return useQuery({
    queryKey: driftKeys.runs(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/drift-runs');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<DriftRun[]>;
    },
  });
}

export function useDriftRun(id: string) {
  return useQuery({
    queryKey: driftKeys.run(id),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/drift-runs/${id}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data = await res.json();
      return Array.isArray(data) ? data[0] : data;
    },
    enabled: !!id,
  });
}

export function useDriftComparisons() {
  return useQuery({
    queryKey: driftKeys.comparisons(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/drift-comparisons');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<DriftComparison[]>;
    },
  });
}
