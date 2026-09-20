import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Box, Typography, LinearProgress, Button, Alert } from '@mui/material';
import apiClient from '../../../utils/apiClient';

interface JobResult {
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

interface JobStatus {
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

interface BulkGenerateProgressModalProps {
  open: boolean;
  jobId: string;
  tenantId: string;
  totalItems: number;
  onClose: () => void;
  onSuccess: (created: number, reused: number) => void;
}

export default function BulkGenerateProgressModal({
  open,
  jobId,
  tenantId,
  totalItems,
  onClose,
  onSuccess,
}: BulkGenerateProgressModalProps) {
  const [closed, setClosed] = useState(false);

  const { data: job, isError, dataUpdatedAt } = useQuery<JobStatus>({
    queryKey: ['glossary-bulk-job', jobId, tenantId],
    queryFn: () => apiClient<JobStatus>(
      `/api/glossary/jobs/${jobId}`
    ),
    enabled: open && !!jobId && !!tenantId && !closed,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === 'completed' || status === 'failed') return false;
      return 1500;
    },
  });

  if (!open || closed) return null;

  const status = job?.status ?? 'running';
  const done = job?.done ?? 0;
  const failed = job?.failed ?? 0;
  const total = job?.total ?? totalItems;
  const pct = total > 0 ? Math.round((done / total) * 100) : 0;
  const errors = job?.errors ?? [];

  const handleClose = () => {
    if (status === 'running') return;
    setClosed(true);
    onClose();
  };

  const handleDone = () => {
    setClosed(true);
    if (job?.results) {
      const created = job.results.filter(r => !r.reused_existing).length;
      const reused = job.results.filter(r => r.reused_existing).length;
      onSuccess(created, reused);
    }
    onClose();
  };

  const isTerminal = status === 'completed' || status === 'failed';

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 200,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        background: '#0F1218', borderRadius: 12, width: 520, padding: 32,
        border: '1px solid #2A2D3A', display: 'flex', flexDirection: 'column', gap: 20,
      }}>
        <Typography variant="h6" sx={{ color: '#E2E8F0', margin: 0 }}>
          Generating Semantic Terms
        </Typography>

        <Box>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
            <Typography variant="body2" sx={{ color: '#94A3B8' }}>
              {status === 'running' ? `${done} / ${total} terms processed` :
               status === 'completed' ? `Completed — ${done} terms` :
               `Failed — ${failed} of ${total} failed`}
            </Typography>
            <Typography variant="body2" sx={{ color: '#94A3B8' }}>{pct}%</Typography>
          </Box>
          <LinearProgress
            variant="determinate"
            value={pct}
            sx={{
              height: 6, borderRadius: 3,
              backgroundColor: '#1E2130',
              '& .MuiLinearProgress-bar': {
                backgroundColor: status === 'failed' ? '#EF4444' : '#6366F1',
                borderRadius: 3,
              },
            }}
          />
        </Box>

        <Box sx={{ display: 'flex', gap: 3 }}>
          <Box sx={{ flex: 1, background: '#1E2130', borderRadius: 8, p: 1.5, textAlign: 'center' }}>
            <Typography variant="h5" sx={{ color: '#E2E8F0', lineHeight: 1 }}>{total}</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Total</Typography>
          </Box>
          <Box sx={{ flex: 1, background: '#1E2130', borderRadius: 8, p: 1.5, textAlign: 'center' }}>
            <Typography variant="h5" sx={{ color: '#22C55E', lineHeight: 1 }}>{done - failed}</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Succeeded</Typography>
          </Box>
          <Box sx={{ flex: 1, background: '#1E2130', borderRadius: 8, p: 1.5, textAlign: 'center' }}>
            <Typography variant="h5" sx={{ color: '#EF4444', lineHeight: 1 }}>{failed}</Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>Failed</Typography>
          </Box>
        </Box>

        {isError && dataUpdatedAt > 0 && (
          <Alert severity="error" sx={{ background: '#1E2130', color: '#F87171' }}>
            Failed to fetch job status. Is the backend running?
          </Alert>
        )}

        {isTerminal && errors.length > 0 && (
          <Box sx={{ background: '#1E2130', borderRadius: 8, p: 2, maxHeight: 120, overflowY: 'auto' }}>
            <Typography variant="caption" sx={{ color: '#EF4444', display: 'block', mb: 1 }}>
              Errors ({errors.length})
            </Typography>
            {errors.slice(0, 10).map((e, i) => (
              <Typography key={i} variant="caption" sx={{ color: '#F87171', display: 'block', fontFamily: 'monospace' }}>
                {e}
              </Typography>
            ))}
          </Box>
        )}

        <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
          <Button
            onClick={handleClose}
            disabled={status === 'running'}
            sx={{ color: '#94A3B8' }}
          >
            {status === 'running' ? 'Running…' : 'Close'}
          </Button>
          {isTerminal && (
            <Button
              onClick={handleDone}
              variant="contained"
              sx={{ background: '#6366F1', '&:hover': { background: '#4F46E5' } }}
            >
              Done
            </Button>
          )}
        </Box>
      </div>
    </div>
  );
}
