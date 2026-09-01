import React, { useEffect, useState } from 'react';
import {
  Box,
  Alert,
  Typography,
  CircularProgress,
  Stack,
} from '@mui/material';
import ExplainPlanVisualizer from '../../query-builder/components/ExplainPlanVisualizer';
import QueryPerformanceSummary from '../../query-builder/components/QueryPerformanceSummary';
import { previewExplorerPlan } from '../services/dataExplorerApi';
import type {
  ExplorerSource,
  ExplorerQueryState,
} from '../types/dataExplorerTypes';
import type { FederatedPlan } from '../../query-builder/types/queryDef';
import {
  EXPLORER_ACCENT,
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';

interface ExplainPlanPaneProps {
  source: ExplorerSource;
  state: ExplorerQueryState;
  initialPlan?: FederatedPlan;
}

export const ExplainPlanPane: React.FC<ExplainPlanPaneProps> = ({
  source,
  state,
  initialPlan,
}) => {
  const [plan, setPlan] = useState<FederatedPlan | null>(initialPlan ?? null);
  const [loading, setLoading] = useState(!initialPlan);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialPlan) {
      setPlan(initialPlan);
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    previewExplorerPlan(source, state)
      .then((next) => {
        if (cancelled) return;
        setPlan(next);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Failed to fetch plan.');
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [source, state, initialPlan]);

  if (loading) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: EXPLORER_BG,
        }}
      >
        <Stack spacing={2} alignItems="center">
          <CircularProgress sx={{ color: EXPLORER_ACCENT }} />
          <Typography variant="body2" sx={{ color: EXPLORER_MUTED }}>
            Generating federated explain plan…
          </Typography>
        </Stack>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (!plan) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: EXPLORER_BG,
          p: 6,
        }}
      >
        <Box sx={{ textAlign: 'center', maxWidth: 480 }}>
          <Typography variant="h6" fontWeight={700} sx={{ color: EXPLORER_TEXT, mb: 1 }}>
            No plan available
          </Typography>
          <Typography variant="body2" sx={{ color: EXPLORER_MUTED }}>
            Add dimensions or measures to generate a federated explain plan.
          </Typography>
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={{ flex: 1, overflow: 'auto', p: 3, bgcolor: EXPLORER_BG }}>
      <Box sx={{ maxWidth: 1200, mx: 'auto' }}>
        <Box sx={{ mb: 2 }}>
          <QueryPerformanceSummary plan={plan} />
        </Box>
        <Box
          sx={{
            border: `1px solid ${EXPLORER_BORDER}`,
            borderRadius: 3,
            bgcolor: 'white',
            p: 1,
          }}
        >
          <ExplainPlanVisualizer plan={plan} />
        </Box>
      </Box>
    </Box>
  );
};

export default ExplainPlanPane;
