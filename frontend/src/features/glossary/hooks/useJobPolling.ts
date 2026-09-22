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
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === 'completed' || status === 'failed') return false;
      return 1500;
    },
  });
}
