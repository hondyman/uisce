import React from 'react';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  Chip,
  Paper,
  Stack,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import ShieldIcon from '@mui/icons-material/Shield';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import BlockIcon from '@mui/icons-material/Block';

export interface PlanNode {
  id: string;
  name: string;
  engine: 'StarRocks' | 'DataFusion' | 'Postgres' | 'ArrowMemory';
  costScore: number;
  latencyEstMs: number;
  pushdownFilters: string[];
  partitionPruning: boolean;
}

export interface ExplainPlanDrawerProps {
  open: boolean;
  onClose: () => void;
  queryHash?: string;
  complexityScore?: number;
  circuitBreakerStatus?: 'ALLOWED' | 'WARNING' | 'FORBIDDEN';
  nodes?: PlanNode[];
  evaluationTimeUs?: number;
}

export const ExplainPlanDrawer: React.FC<ExplainPlanDrawerProps> = ({
  open,
  onClose,
  queryHash = 'sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
  complexityScore = 25.0,
  circuitBreakerStatus = 'ALLOWED',
  nodes = [
    {
      id: 'node_1',
      name: 'Hot Partition Scan (T >= Wt)',
      engine: 'StarRocks',
      costScore: 12.5,
      latencyEstMs: 3.2,
      pushdownFilters: ['tenant_id = core', 'as_of_date >= 2026-01-01'],
      partitionPruning: true,
    },
    {
      id: 'node_2',
      name: 'Cold Lakehouse Scan (T < Wt)',
      engine: 'DataFusion',
      costScore: 28.0,
      latencyEstMs: 14.8,
      pushdownFilters: ['tenant_id = core', 'as_of_date < 2026-01-01'],
      partitionPruning: true,
    },
    {
      id: 'node_3',
      name: 'Vectorized Window Deduplication',
      engine: 'ArrowMemory',
      costScore: 5.0,
      latencyEstMs: 1.1,
      pushdownFilters: ['Dedupe by account_bk'],
      partitionPruning: false,
    },
  ],
  evaluationTimeUs = 280,
}) => {
  const getStatusChip = () => {
    switch (circuitBreakerStatus) {
      case 'FORBIDDEN':
        return <Chip icon={<BlockIcon sx={{ fontSize: 14 }} />} label="FORBIDDEN (Blocked by Rule 8)" color="error" size="small" sx={{ fontWeight: 700 }} />;
      case 'WARNING':
        return <Chip icon={<WarningAmberIcon sx={{ fontSize: 14 }} />} label="WARNING (High Complexity)" color="warning" size="small" sx={{ fontWeight: 700 }} />;
      default:
        return <Chip icon={<CheckCircleIcon sx={{ fontSize: 14 }} />} label="ALLOWED (Circuit OK)" color="success" size="small" sx={{ fontWeight: 700 }} />;
    }
  };

  const getEngineColor = (engine: PlanNode['engine']) => {
    switch (engine) {
      case 'StarRocks': return '#00D4FF';
      case 'DataFusion': return '#10B981';
      case 'ArrowMemory': return '#8B5CF6';
      default: return '#3B82F6';
    }
  };

  return (
    <Drawer anchor="right" open={open} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', sm: 480 }, bgcolor: '#071526', color: '#fff', p: 3 } }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <ShieldIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
          <Typography variant="subtitle1" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
            Visual Explain-Plan & Cost Governor
          </Typography>
        </Box>
        <IconButton size="small" onClick={onClose} sx={{ color: 'text.secondary' }}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      {/* Summary Card */}
      <Paper sx={{ p: 2, mb: 2.5, bgcolor: '#0B1E36', border: '1px solid rgba(0, 212, 255, 0.2)', borderRadius: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
          <Typography variant="caption" color="text.secondary">FinOps Circuit Breaker</Typography>
          {getStatusChip()}
        </Box>

        <Stack direction="row" spacing={2} sx={{ mb: 1.5 }}>
          <Box>
            <Typography variant="caption" color="text.secondary">Complexity Score</Typography>
            <Typography variant="h6" fontWeight="700" sx={{ color: complexityScore >= 85 ? '#EF4444' : '#00D4FF' }}>
              {complexityScore.toFixed(1)} <Typography component="span" variant="caption" color="text.secondary">/ 100</Typography>
            </Typography>
          </Box>
          <Box>
            <Typography variant="caption" color="text.secondary">Pre-Flight Gating</Typography>
            <Typography variant="h6" fontWeight="700" color="#10B981">
              {evaluationTimeUs} <Typography component="span" variant="caption" color="text.secondary">&micro;s</Typography>
            </Typography>
          </Box>
        </Stack>

        <Typography variant="caption" sx={{ display: 'block', fontFamily: 'monospace', fontSize: '10px', color: '#64748B', wordBreak: 'break-all' }}>
          Fingerprint: {queryHash}
        </Typography>
      </Paper>

      {/* Execution DAG Steps */}
      <Typography variant="caption" fontWeight="700" sx={{ letterSpacing: '0.05em', color: '#00D4FF', textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Federated Execution DAG (Pushdown Paths)
      </Typography>

      <Stack spacing={1.5}>
        {nodes.map((node, idx) => (
          <Paper key={node.id} sx={{ p: 2, bgcolor: '#050D1A', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="body2" fontWeight="700" sx={{ fontSize: '0.8rem' }}>
                Step {idx + 1}: {node.name}
              </Typography>
              <Chip size="small" label={node.engine} sx={{ bgcolor: `${getEngineColor(node.engine)}20`, color: getEngineColor(node.engine), fontWeight: 700, fontSize: '0.65rem', height: 18 }} />
            </Box>

            <Stack direction="row" spacing={3} sx={{ mb: 1 }}>
              <Box>
                <Typography variant="caption" color="text.secondary">Cost Unit</Typography>
                <Typography variant="body2" fontWeight="600" sx={{ fontSize: '0.75rem' }}>{node.costScore.toFixed(1)}</Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary">Est. Latency</Typography>
                <Typography variant="body2" fontWeight="600" sx={{ fontSize: '0.75rem' }}>{node.latencyEstMs} ms</Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary">Pruning</Typography>
                <Typography variant="body2" fontWeight="600" sx={{ fontSize: '0.75rem', color: node.partitionPruning ? '#10B981' : '#94A3B8' }}>
                  {node.partitionPruning ? 'Active' : 'N/A'}
                </Typography>
              </Box>
            </Stack>

            {node.pushdownFilters.length > 0 && (
              <Box sx={{ mt: 1, pt: 1, borderTop: '1px solid rgba(255,255,255,0.05)' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5, fontSize: '0.65rem' }}>
                  Pushdown Predicates:
                </Typography>
                {node.pushdownFilters.map((p, pIdx) => (
                  <Typography key={pIdx} component="div" sx={{ fontFamily: 'monospace', fontSize: '10px', color: '#00D4FF' }}>
                    &bull; {p}
                  </Typography>
                ))}
              </Box>
            )}
          </Paper>
        ))}
      </Stack>
    </Drawer>
  );
};

export default ExplainPlanDrawer;
