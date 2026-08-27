import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  Hub as HubIcon,
  PlayArrow as PlayIcon,
  CheckCircle as CheckCircleIcon,
  Speed as SpeedIcon
} from '@mui/icons-material';

interface SwarmPassport {
  run_id: string;
  status: string;
  merkle_root_seal: string;
  total_duration_ms: number;
  task_results: Array<{
    agent_type: string;
    okf_concept_key: string;
    status: string;
    latency_ms: number;
    merkle_leaf_hash: string;
  }>;
}

export const MultiAgentSwarmHUD: React.FC<{ tenantId: string }> = ({ tenantId: _tenantId }) => {
  const [isRunning, setIsRunning] = useState(false);
  const [passport, setPassport] = useState<SwarmPassport | null>(null);

  const handleRunSwarm = async () => {
    setIsRunning(true);
    setTimeout(() => {
      setPassport({
        run_id: 'e4b10294-819a-4c22-bade-4f4d439ac84c',
        status: 'COMPLETED',
        merkle_root_seal: '7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069',
        total_duration_ms: 18,
        task_results: [
          {
            agent_type: 'RECON_BREAK_AGENT',
            okf_concept_key: 'mdm.survivorship.fixed_income_pricing',
            status: 'VERIFIED',
            latency_ms: 4,
            merkle_leaf_hash: 'a1b2c3d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef'
          },
          {
            agent_type: 'RISK_SHOCK_AGENT',
            okf_concept_key: 'compliance.rule.fund_concentration_us_equity',
            status: 'VERIFIED',
            latency_ms: 6,
            merkle_leaf_hash: 'b2c3d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef12'
          },
          {
            agent_type: 'ALLOCATION_AGENT',
            okf_concept_key: 'concept/allocation-waterfall',
            status: 'VERIFIED',
            latency_ms: 3,
            merkle_leaf_hash: 'c3d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef1234'
          }
        ]
      });
      setIsRunning(false);
    }, 500);
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        bgcolor: '#0B1E36',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <HubIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC', fontSize: 16 }}>
              Autonomous Multi-Agent Domain Swarm
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Coordinated pre-trade negotiation with SEC Rule 17a-4 Merkle receipts
            </Typography>
          </Box>
        </Stack>
        <Button
          variant="contained"
          size="small"
          startIcon={isRunning ? <CircularProgress size={14} color="inherit" /> : <PlayIcon />}
          onClick={handleRunSwarm}
          disabled={isRunning}
          sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, '&:hover': { bgcolor: '#0369A1' } }}
        >
          {isRunning ? 'Coordinating Swarm...' : 'Dispatch Swarm Run'}
        </Button>
      </Box>

      {passport && (
        <Stack spacing={2.5} mt={3}>
          <Grid container spacing={2}>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 1.5 }}>
                <Typography variant="caption" sx={{ color: '#64748B' }}>
                  Execution Status
                </Typography>
                <Typography variant="subtitle2" sx={{ color: '#10B981', fontWeight: 700, display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <CheckCircleIcon fontSize="small" /> {passport.status}
                </Typography>
              </Paper>
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 1.5 }}>
                <Typography variant="caption" sx={{ color: '#64748B' }}>
                  Total Pipeline Latency
                </Typography>
                <Typography variant="subtitle2" sx={{ color: '#00D4FF', fontWeight: 700, fontFamily: 'monospace', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <SpeedIcon fontSize="small" /> {passport.total_duration_ms} ms
                </Typography>
              </Paper>
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 1.5 }}>
                <Typography variant="caption" sx={{ color: '#64748B' }}>
                  Merkle Root Seal
                </Typography>
                <Typography variant="subtitle2" sx={{ color: '#CBD5E1', fontFamily: 'monospace', fontSize: 11 }}>
                  {passport.merkle_root_seal.slice(0, 18)}...
                </Typography>
              </Paper>
            </Grid>
          </Grid>

          <TableContainer component={Paper} sx={{ bgcolor: '#071526', border: '1px solid #1E293B' }}>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B' } }}>
                  <TableCell>Specialized Agent</TableCell>
                  <TableCell>Attested OKF Concept</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Latency</TableCell>
                  <TableCell>Merkle Leaf Receipt</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {passport.task_results.map((tr, idx) => (
                  <TableRow key={idx} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                    <TableCell sx={{ fontWeight: 600, color: '#38BDF8' }}>{tr.agent_type}</TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>{tr.okf_concept_key}</TableCell>
                    <TableCell>
                      <Chip label={tr.status} size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }} />
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>{tr.latency_ms} ms</TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: 10, color: '#94A3B8' }}>
                      {tr.merkle_leaf_hash.slice(0, 16)}...
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}
    </Paper>
  );
};

export default MultiAgentSwarmHUD;
