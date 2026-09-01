import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  LinearProgress,
  Divider,
  Alert
} from '@mui/material';
import {
  Speed as SpeedIcon,
  Storage as StorageIcon,
  AttachMoney as MoneyIcon,
  AccountTree as DagIcon,
  Shield as ShieldIcon,
  PlayArrow as RunIcon
} from '@mui/icons-material';

export interface PreflightEstimate {
  hotPercentage: number;
  coldPercentage: number;
  estLatencyMs: number;
  scannedVolumeMb: number;
  estComputeUsd: number;
  complexityScore: number;
  passesBreaker: boolean;
  breakerMessage?: string;
  explainDagSteps: Array<{
    stepId: string;
    name: string;
    engine: string;
    operation: string;
    cost: number;
    rowCount: number;
    description: string;
  }>;
}

interface PreflightFinOpsAdvisorProps {
  estimate?: PreflightEstimate;
  onExecute: () => void;
  onOpenDAG: () => void;
  isLoading?: boolean;
}

export const PreflightFinOpsAdvisor: React.FC<PreflightFinOpsAdvisorProps> = ({
  estimate = {
    hotPercentage: 85,
    coldPercentage: 15,
    estLatencyMs: 180,
    scannedVolumeMb: 14.2,
    estComputeUsd: 0.0021,
    complexityScore: 24,
    passesBreaker: true,
    explainDagSteps: [
      {
        stepId: 'step-01',
        name: 'Partition Pruning & Filter Pushdown',
        engine: 'StarRocks',
        operation: 'PartitionPrune',
        cost: 1.2,
        rowCount: 154000,
        description: 'Pruned to active tenant partitions with pushdown predicates'
      },
      {
        stepId: 'step-02',
        name: 'Hot/Cold Lakehouse Federation',
        engine: 'Iceberg W_t',
        operation: 'FederatedJoin',
        cost: 4.5,
        rowCount: 32000,
        description: 'Federated union across StarRocks (85%) and Iceberg (15%)'
      }
    ]
  },
  onExecute,
  onOpenDAG,
  isLoading = false
}) => {
  return (
    <Paper
      elevation={0}
      sx={{
        p: 2,
        mb: 2,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={1.5}>
        <Stack direction="row" spacing={1} alignItems="center">
          <ShieldIcon sx={{ color: estimate.passesBreaker ? '#10B981' : '#EF4444', fontSize: 20 }} />
          <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: 13, textTransform: 'uppercase', letterSpacing: 0.5 }}>
            Pre-Flight Query Advisor & FinOps Guardrails
          </Typography>
        </Stack>
        <Chip
          label={`Complexity: ${estimate.complexityScore}/100`}
          size="small"
          sx={{
            bgcolor: estimate.complexityScore < 50 ? '#064E3B' : estimate.complexityScore < 85 ? '#451A03' : '#450A0A',
            color: estimate.complexityScore < 50 ? '#34D399' : estimate.complexityScore < 85 ? '#FBBF24' : '#F87171',
            fontWeight: 700,
            fontSize: 11
          }}
        />
      </Box>

      {/* Metrics Row */}
      <Stack direction="row" spacing={3} alignItems="center" flexWrap="wrap" mb={2}>
        <Box display="flex" alignItems="center" gap={1}>
          <StorageIcon sx={{ color: '#00D4FF', fontSize: 18 }} />
          <Box>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', fontSize: 10 }}>Routing</Typography>
            <Typography variant="body2" sx={{ fontWeight: 700, fontSize: 12 }}>
              {estimate.hotPercentage}% StarRocks (Hot) + {estimate.coldPercentage}% Iceberg (Cold)
            </Typography>
          </Box>
        </Box>

        <Divider orientation="vertical" flexItem sx={{ borderColor: '#1E293B' }} />

        <Box display="flex" alignItems="center" gap={1}>
          <SpeedIcon sx={{ color: '#34D399', fontSize: 18 }} />
          <Box>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', fontSize: 10 }}>Est. Latency</Typography>
            <Typography variant="body2" sx={{ fontWeight: 700, fontSize: 12, fontFamily: 'monospace' }}>
              ~{estimate.estLatencyMs}ms
            </Typography>
          </Box>
        </Box>

        <Divider orientation="vertical" flexItem sx={{ borderColor: '#1E293B' }} />

        <Box display="flex" alignItems="center" gap={1}>
          <MoneyIcon sx={{ color: '#FBBF24', fontSize: 18 }} />
          <Box>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', fontSize: 10 }}>Scanned Volume & Cost</Typography>
            <Typography variant="body2" sx={{ fontWeight: 700, fontSize: 12, fontFamily: 'monospace' }}>
              {estimate.scannedVolumeMb} MB · ${estimate.estComputeUsd.toFixed(4)}
            </Typography>
          </Box>
        </Box>
      </Stack>

      {!estimate.passesBreaker && estimate.breakerMessage && (
        <Alert severity="error" sx={{ mb: 2, bgcolor: '#450A0A', color: '#FCA5A5', border: '1px solid #DC2626' }}>
          {estimate.breakerMessage}
        </Alert>
      )}

      {/* Action Buttons */}
      <Stack direction="row" spacing={1.5} justifyContent="flex-end">
        <Button
          size="small"
          variant="outlined"
          startIcon={<DagIcon sx={{ fontSize: 14 }} />}
          onClick={onOpenDAG}
          sx={{
            color: '#38BDF8',
            borderColor: '#0284C7',
            textTransform: 'none',
            fontSize: 11,
            fontWeight: 600,
            '&:hover': { borderColor: '#38BDF8', bgcolor: 'rgba(56, 189, 248, 0.08)' }
          }}
        >
          View Explain Plan DAG
        </Button>
        <Button
          size="small"
          variant="contained"
          startIcon={<RunIcon sx={{ fontSize: 14 }} />}
          onClick={onExecute}
          disabled={isLoading || !estimate.passesBreaker}
          sx={{
            bgcolor: '#0284C7',
            color: '#FFFFFF',
            textTransform: 'none',
            fontSize: 11,
            fontWeight: 700,
            '&:hover': { bgcolor: '#0369A1' }
          }}
        >
          {isLoading ? 'Executing...' : 'Run Unified Execution'}
        </Button>
      </Stack>
    </Paper>
  );
};

export default PreflightFinOpsAdvisor;
