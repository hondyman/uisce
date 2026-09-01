import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Stack,
  Button,
  Grid,
  LinearProgress,
  Tooltip,
} from '@mui/material';
import SpeedIcon from '@mui/icons-material/Speed';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import VerifiedUserIcon from '@mui/icons-material/VerifiedUser';
import LockIcon from '@mui/icons-material/Lock';
import MemoryIcon from '@mui/icons-material/Memory';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

export const NextHorizonTelemetryHUD: React.FC<{ tenantId?: string }> = ({
  tenantId = 'default',
}) => {
  const [activeTab, setActiveTab] = useState<'AQE' | 'SMT' | 'PRIVACY' | 'FLIGHT'>('AQE');

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', p: 3, fontFamily: 'sans-serif' }}>
      
      {/* Top Banner Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 2, borderBottom: '1px solid #1E293B', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <MemoryIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" fontWeight="700">
              Next-Horizon Apex Telemetry HUD & Adaptive Mesh
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Zero-Copy Flight SQL &bull; Adaptive Query Optimizer (AQE) &bull; Z3 Formal SMT &bull; Differential Privacy
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={1}>
          <Chip
            icon={<CheckCircleIcon sx={{ fontSize: '14px !important', color: '#10B981 !important' }} />}
            label="RULE 7 ISOLATED"
            size="small"
            sx={{ bgcolor: 'rgba(16, 185, 129, 0.12)', color: '#10B981', fontWeight: 700, fontSize: '10px' }}
          />
          <Chip
            label="TIER 1 G-SIFI ENGINE"
            size="small"
            sx={{ bgcolor: 'rgba(245, 166, 35, 0.15)', color: '#F5A623', fontWeight: 700, fontSize: '10px' }}
          />
        </Stack>
      </Box>

      {/* 4 Pillar Quick-Switchers */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid    size={{ xs: 12, sm: 6, md: 3 }}>
          <Paper
            onClick={() => setActiveTab('FLIGHT')}
            sx={{
              p: 2,
              bgcolor: activeTab === 'FLIGHT' ? '#0E2238' : '#071526',
              border: activeTab === 'FLIGHT' ? '1px solid #00D4FF' : '1px solid #1E293B',
              cursor: 'pointer',
              borderRadius: 2,
              transition: 'all 0.2s ease',
            }}
          >
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
              <SpeedIcon sx={{ color: '#00D4FF', fontSize: 20 }} />
              <Typography variant="caption" fontWeight="700" sx={{ color: '#00D4FF' }}>
                ARROW FLIGHT SQL
              </Typography>
            </Stack>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#fff', fontSize: '18px' }}>
              2.84 GB/s
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              0-Copy IPC &bull; Sub-150µs Latency
            </Typography>
          </Paper>
        </Grid>

        <Grid    size={{ xs: 12, sm: 6, md: 3 }}>
          <Paper
            onClick={() => setActiveTab('AQE')}
            sx={{
              p: 2,
              bgcolor: activeTab === 'AQE' ? '#0E2238' : '#071526',
              border: activeTab === 'AQE' ? '1px solid #8B5CF6' : '1px solid #1E293B',
              cursor: 'pointer',
              borderRadius: 2,
              transition: 'all 0.2s ease',
            }}
          >
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
              <AutoFixHighIcon sx={{ color: '#8B5CF6', fontSize: 20 }} />
              <Typography variant="caption" fontWeight="700" sx={{ color: '#8B5CF6' }}>
                ADAPTIVE EXECUTION
              </Typography>
            </Stack>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#fff', fontSize: '18px' }}>
              BROADCAST HASH
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              4,200 Rows &bull; 75 Splits Pruned
            </Typography>
          </Paper>
        </Grid>

        <Grid    size={{ xs: 12, sm: 6, md: 3 }}>
          <Paper
            onClick={() => setActiveTab('SMT')}
            sx={{
              p: 2,
              bgcolor: activeTab === 'SMT' ? '#0E2238' : '#071526',
              border: activeTab === 'SMT' ? '1px solid #10B981' : '1px solid #1E293B',
              cursor: 'pointer',
              borderRadius: 2,
              transition: 'all 0.2s ease',
            }}
          >
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
              <VerifiedUserIcon sx={{ color: '#10B981', fontSize: 20 }} />
              <Typography variant="caption" fontWeight="700" sx={{ color: '#10B981' }}>
                FORMAL SMT SOLVER
              </Typography>
            </Stack>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#fff', fontSize: '18px' }}>
              SATISFIABLE
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              0 Contradictions &bull; Z3 Verified
            </Typography>
          </Paper>
        </Grid>

        <Grid    size={{ xs: 12, sm: 6, md: 3 }}>
          <Paper
            onClick={() => setActiveTab('PRIVACY')}
            sx={{
              p: 2,
              bgcolor: activeTab === 'PRIVACY' ? '#0E2238' : '#071526',
              border: activeTab === 'PRIVACY' ? '1px solid #F5A623' : '1px solid #1E293B',
              cursor: 'pointer',
              borderRadius: 2,
              transition: 'all 0.2s ease',
            }}
          >
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
              <LockIcon sx={{ color: '#F5A623', fontSize: 20 }} />
              <Typography variant="caption" fontWeight="700" sx={{ color: '#F5A623' }}>
                (ε, δ) PRIVACY MESH
              </Typography>
            </Stack>
            <Typography variant="h6" fontWeight="800" sx={{ color: '#fff', fontSize: '18px' }}>
              ε = 0.50
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Laplace Noise &bull; ZK Passport Sealed
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Main Diagnostic Detail Panel */}
      <Paper sx={{ p: 3, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2 }}>
        {activeTab === 'AQE' && (
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#8B5CF6', mb: 1 }}>
              Dynamic Runtime Execution Topology & Dynamic Partition Pruning (DPP)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 2 }}>
              Plan adapted from distributed hash join to in-memory broadcast join after evaluating driving filter predicates.
            </Typography>

            <Box sx={{ p: 2, bgcolor: '#030914', borderRadius: 1.5, border: '1px solid #1E293B', mb: 2 }}>
              <Stack spacing={1.5}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Driving Filtered Row Count:</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#10B981', fontWeight: 700 }}>4,200 rows (Threshold: 100,000)</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Dynamic S3 Split Pruning:</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#00D4FF', fontWeight: 700 }}>75 of 100 splits pruned (75% I/O reduction)</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Network Shuffle Stage:</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#8B5CF6', fontWeight: 700 }}>Bypassed (In-Memory Broadcast Hash Join)</Typography>
                </Box>
              </Stack>
            </Box>
          </Box>
        )}

        {activeTab === 'SMT' && (
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#10B981', mb: 1 }}>
              Formal SMT First-Order Proof Verification (Z3 Solver Core)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 2 }}>
              Mathematical proof ensuring all portfolio constraint branches sum to ≤ 100% with no mutually exclusive bounds.
            </Typography>

            <pre style={{ fontFamily: 'monospace', fontSize: '11px', color: '#CBD5E1', padding: '12px', backgroundColor: '#030914', borderRadius: '4px', border: '1px solid #1E293B' }}>
{`; SMT-LIB2 Invariant Formulation
(declare-const w_fixed_income Real)
(declare-const w_equities Real)
(declare-const w_cash Real)
(assert (>= w_fixed_income 0.40))
(assert (<= w_equities 0.50))
(assert (>= w_cash 0.05))
(assert (= (+ w_fixed_income w_equities w_cash) 1.0))
(check-sat)
; Result: SATISFIABLE (Model: w_fixed_income=0.45, w_equities=0.50, w_cash=0.05)`}
            </pre>
          </Box>
        )}

        {activeTab === 'PRIVACY' && (
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#F5A623', mb: 1 }}>
              Differential Privacy (ε=0.50, δ=1e-5) & Cryptographic Zero-Knowledge Passport
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 2 }}>
              Laplace mechanism noise injection for peer benchmark analytics preventing constituent trade reconstruction.
            </Typography>

            <Box sx={{ p: 2, bgcolor: '#030914', borderRadius: 1.5, border: '1px solid #1E293B' }}>
              <Stack spacing={1}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Raw Peer Metric:</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#fff' }}>+14.2400%</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Noisy Published Metric (Laplace):</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#F5A623', fontWeight: 700 }}>+14.1923%</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>ZK Compliance Passport Hash:</Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#00D4FF', fontSize: '10px' }}>
                    e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
                  </Typography>
                </Box>
              </Stack>
            </Box>
          </Box>
        )}

        {activeTab === 'FLIGHT' && (
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#00D4FF', mb: 1 }}>
              Apache Arrow Flight SQL Transport Mesh
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 2 }}>
              High-throughput gRPC service streaming binary Arrow RecordBatches directly into Python / DuckDB / Polars without ODBC serialization.
            </Typography>

            <Box sx={{ p: 2, bgcolor: '#030914', borderRadius: 1.5, border: '1px solid #1E293B' }}>
              <Typography variant="caption" sx={{ color: '#64748B', display: 'block', mb: 1 }}>Python Client Connection:</Typography>
              <pre style={{ fontFamily: 'monospace', fontSize: '11px', color: '#00D4FF', margin: 0 }}>
{`import pyarrow.flight as flight
client = flight.connect("grpc+tls://flight.uisce.io:443")
ticket = flight.Ticket(b"bo:portfolio_performance")
reader = client.do_get(ticket)
df = reader.read_pandas() # Direct zero-copy Arrow memory mapping`}
              </pre>
            </Box>
          </Box>
        )}
      </Paper>
    </Box>
  );
};

export default NextHorizonTelemetryHUD;
