import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
  LinearProgress
} from '@mui/material';
import {
  Security as ZkIcon,
  Shield as ShieldValidIcon,
  VisibilityOff as PrivateIcon,
  Key as KeyIcon,
  Analytics as BenchmarkIcon,
  Lock as LockIcon,
  Speed as LatencyIcon
} from '@mui/icons-material';

interface CovenantRow {
  facilityId: string;
  facilityName: string;
  borrowerKey: string;
  minDSCR: number;
  maxLeverage: number;
  minLiquidityUSD: number;
  verificationStatus: 'PROVEN_VALID' | 'PENDING_PROOF' | 'BREACH_FLAGGED';
  zkProofLatencyMs: number;
  merkleSeal: string;
}

export const ZKCleanRoomStudio: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [isProving, setIsProving] = useState(false);
  const [epsilonBudget, setEpsilonBudget] = useState(4.25);
  const totalEpsilon = 10.0;

  const [covenants, setCovenants] = useState<CovenantRow[]>([
    {
      facilityId: 'fac-direct-001',
      facilityName: 'Senior Secured Term Loan B (Tech Sponsor)',
      borrowerKey: 'BORROWER_9182_BLINDED',
      minDSCR: 1.35,
      maxLeverage: 4.20,
      minLiquidityUSD: 5000000,
      verificationStatus: 'PROVEN_VALID',
      zkProofLatencyMs: 12.45,
      merkleSeal: '3f8a92b1c4e789...'
    },
    {
      facilityId: 'fac-direct-002',
      facilityName: 'Revolving Credit Facility (Healthcare)',
      borrowerKey: 'BORROWER_4011_BLINDED',
      minDSCR: 1.25,
      maxLeverage: 4.50,
      minLiquidityUSD: 7500000,
      verificationStatus: 'PENDING_PROOF',
      zkProofLatencyMs: 0.0,
      merkleSeal: '—'
    }
  ]);

  const handleGenerateZkProof = (facilityId: string) => {
    setIsProving(true);
    setTimeout(() => {
      setCovenants((prev) =>
        prev.map((c) =>
          c.facilityId === facilityId
            ? {
                ...c,
                verificationStatus: 'PROVEN_VALID',
                zkProofLatencyMs: 14.12,
                merkleSeal: '8e1b99c72f10aa...'
              }
            : c
        )
      );
      setEpsilonBudget((prev) => prev + 0.25);
      setIsProving(false);
    }, 900);
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <ZkIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Zero-Knowledge Private Credit & Syndicate Clean Room
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Groth16 / BN254 Non-Interactive Proofs & Differential Privacy ($\epsilon$-DP) Analytics
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            icon={<ShieldValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />}
            label="Zero Financial Data Leakage"
            size="small"
            sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
          />
          <Chip
            icon={<LatencyIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
            label="Prover SLA: < 15 ms"
            size="small"
            sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, fontFamily: 'monospace' }}
          />
        </Stack>
      </Box>

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <KeyIcon sx={{ color: '#38BDF8', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                ZK Cryptographic Verification
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399', fontFamily: 'monospace' }}>
              100% Mathematically Proven
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              BN254 Pairing Curve (256-Bit Proof)
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <BenchmarkIcon sx={{ color: '#F59E0B', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Differential Privacy Budget (ε)
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#FBBF24', fontFamily: 'monospace' }}>
              {epsilonBudget.toFixed(2)} / {totalEpsilon.toFixed(2)} ε
            </Typography>
            <LinearProgress
              variant="determinate"
              value={(epsilonBudget / totalEpsilon) * 100}
              sx={{ mt: 1, height: 6, borderRadius: 1, bgcolor: '#071526', '& .MuiLinearProgress-bar': { bgcolor: '#F59E0B' } }}
            />
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Stack direction="row" spacing={1} alignItems="center" mb={0.5}>
              <PrivateIcon sx={{ color: '#A855F7', fontSize: 20 }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                Syndicate Exposure Gating
              </Typography>
            </Stack>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
              Blind Witness Storage
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              AES-256-GCM Hardware Security
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Syndicate Facility Covenant Verification Matrix (Zero-Knowledge Verified)
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Credit Facility</TableCell>
              <TableCell>Borrower Entity</TableCell>
              <TableCell align="center">DSCR Covenant</TableCell>
              <TableCell align="center">Leverage Limit</TableCell>
              <TableCell align="center">Min Liquidity</TableCell>
              <TableCell align="center">ZK Proof Status</TableCell>
              <TableCell>Merkle Receipt</TableCell>
              <TableCell align="center">Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {covenants.map((c) => (
              <TableRow key={c.facilityId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                  {c.facilityName}
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 11, color: '#94A3B8' }}>
                  {c.borrowerKey}
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontSize: 12, color: '#F8FAFC' }}>
                  ≥ {c.minDSCR.toFixed(2)}x
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontSize: 12, color: '#F8FAFC' }}>
                  ≤ {c.maxLeverage.toFixed(2)}x
                </TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontSize: 12, color: '#34D399' }}>
                  ${(c.minLiquidityUSD / 1000000).toFixed(1)}M
                </TableCell>
                <TableCell align="center">
                  <Chip
                    icon={c.verificationStatus === 'PROVEN_VALID' ? <ShieldValidIcon sx={{ fontSize: 12 }} /> : undefined}
                    label={c.verificationStatus === 'PROVEN_VALID' ? 'ZK Proven Valid' : 'Pending Witness'}
                    size="small"
                    sx={{
                      bgcolor: c.verificationStatus === 'PROVEN_VALID' ? '#064E3B' : '#7C2D12',
                      color: c.verificationStatus === 'PROVEN_VALID' ? '#34D399' : '#FDBA74',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 10, color: '#CBD5E1' }}>
                  {c.merkleSeal}
                </TableCell>
                <TableCell align="center">
                  {c.verificationStatus === 'PENDING_PROOF' ? (
                    <Button
                      variant="contained"
                      size="small"
                      startIcon={isProving ? <CircularProgress size={12} color="inherit" /> : <LockIcon />}
                      onClick={() => handleGenerateZkProof(c.facilityId)}
                      disabled={isProving}
                      sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 10, py: 0.2, '&:hover': { bgcolor: '#0369A1' } }}
                    >
                      Generate Proof
                    </Button>
                  ) : (
                    <Typography variant="caption" sx={{ color: '#34D399', fontWeight: 600 }}>
                      Verified ({c.zkProofLatencyMs} ms)
                    </Typography>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default ZKCleanRoomStudio;
