import { useQuery } from '@tanstack/react-query';
import apiClient from '../../../utils/apiClient';

export interface JobResult {
  id: string;
  name: string;
  reused_existing: boolean;
  business_term_id: string;
  business_term_name: string;
  business_term_reused: boolean;
  definition_source: string;
  columns_linked: number;
  columns_total: number;
  error?: string;
}

export interface JobStatus {
  id: string;
  tenant_id: string;
  datasource_id: string;
  status: 'running' | 'completed' | 'failed';
  total: number;
  done: number;
  failed: number;
  results: JobResult[];
  errors: string[];
  started_at: string;
  finished_at: string;
}

export function useJobPolling(
  jobId: string | null,
  tenantId: string,
  enabled: boolean
) {
  return useQuery<JobStatus>({
    queryKey: ['glossary-bulk-job', jobId, tenantId],
    queryFn: () => apiClient<JobStatus>(`/api/glossary/jobs/${jobId}`),
    enabled: enabled && !!jobId && !!tenantId,
    // Retry on 404 for the first 3 attempts — the job store is in-memory and
    // the goroutine may not have called store.Create yet when the first poll
    // arrives. After 3 retries (~4.5s), a persistent 404 is surfaced as an error.
    retry: (failureCount, error) => {
      if (failureCount >= 3) return false;
      const msg = error instanceof Error ? error.message : String(error);
      if (msg.includes('404') || msg.includes('not found')) return true;
      return false;
    },
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === 'completed' || status === 'failed') return false;
      return 1500;
    },
  });
}
