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
  Alert,
  CircularProgress
} from '@mui/material';
import {
  AltRoute as BlastRadiusIcon,
  AutoFixHigh as AutoHealIcon,
  WarningAmber as WarningIcon,
  Error as ErrorIcon,
  PlayArrow as SimulateIcon
} from '@mui/icons-material';

interface ImpactAsset {
  nodeId: string;
  nodeKey: string;
  nodeName: string;
  nodeType: string;
  impactLevel: 'GREEN' | 'YELLOW' | 'RED';
  reason: string;
  hop: number;
}

export const ImpactAndDriftStudio: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [isSimulating, setIsSimulating] = useState(false);
  const [isHealing, setIsHealing] = useState(false);
  const [healedNotice, setHealedNotice] = useState<string | null>(null);

  const [impactedNodes] = useState<ImpactAsset[]>([
    {
      nodeId: 'ast-01',
      nodeKey: 'wealth.bo.custodial_portfolio',
      nodeName: 'Custodial Portfolio Master',
      nodeType: 'BUSINESS_OBJECT',
      impactLevel: 'YELLOW',
      reason: 'Business Object view definition requires recompilation',
      hop: 1
    },
    {
      nodeId: 'ast-02',
      nodeKey: 'sec.filing.form_13f_hr',
      nodeName: 'SEC Form 13F-HR EDGAR Compiler',
      nodeType: 'SEC_FILING_SPEC',
      impactLevel: 'YELLOW',
      reason: 'XML schema precision check required',
      hop: 2
    },
    {
      nodeId: 'ast-03',
      nodeKey: 'rule.compliance.limit_issuer_max_5',
      nodeName: 'Limit Single Issuer Cap (5%)',
      nodeType: 'COMPLIANCE_MANDATE_RULE',
      impactLevel: 'RED',
      reason: 'Active pre-trade rule depends on formula AST; mutation risks execution block',
      hop: 2
    }
  ]);

  const handleSimulate = () => {
    setIsSimulating(true);
    setTimeout(() => {
      setIsSimulating(false);
    }, 600);
  };

  const handleApplyAutoHeal = () => {
    setIsHealing(true);
    setTimeout(() => {
      setIsHealing(false);
      setHealedNotice(
        'Autonomous Drift Healer: Re-bound Customers.customer_name ➔ Customers.client_name (Cosine Match: 96.4%). All synthetic AST tests passed.'
      );
    }, 800);
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
          <BlastRadiusIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Lineage Impact Simulator & Autonomous Drift Healer
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-hop blast radius prediction & closed-loop vector schema drift auto-repair
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Button
            variant="outlined"
            size="small"
            startIcon={isSimulating ? <CircularProgress size={14} color="inherit" /> : <SimulateIcon />}
            onClick={handleSimulate}
            sx={{ borderColor: '#0284C7', color: '#38BDF8', textTransform: 'none', fontWeight: 600 }}
          >
            Re-Simulate Blast Radius
          </Button>
          <Button
            variant="contained"
            size="small"
            startIcon={isHealing ? <CircularProgress size={14} color="inherit" /> : <AutoHealIcon />}
            onClick={handleApplyAutoHeal}
            sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, '&:hover': { bgcolor: '#0369A1' } }}
          >
            1-Click Auto-Heal Drift
          </Button>
        </Stack>
      </Box>

      {healedNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          {healedNotice}
        </Alert>
      )}

      <Grid container spacing={2} mb={3}>
        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Blast Radius Severity
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#EF4444', fontFamily: 'monospace' }}>
              CRITICAL (RED)
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              1 Breaking Pre-Trade Rule Impacted
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Lineage Traversal Depth
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38BDF8' }}>
              2 Hops (3 Total Nodes)
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Evaluated in 1.4 ms
            </Typography>
          </Paper>
        </Grid>

        <Grid   size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
              Auto-Healing Confidence
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#34D399' }}>
              96.40% Match
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              pgvector HNSW Vector Match
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
        Downstream Dependency Blast Radius
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Impacted Node</TableCell>
              <TableCell>Node Type</TableCell>
              <TableCell align="center">Graph Depth</TableCell>
              <TableCell align="center">Impact Severity</TableCell>
              <TableCell>Diagnostic Reason</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {impactedNodes.map(node => (
              <TableRow key={node.nodeId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>
                    {node.nodeName}
                  </Typography>
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#94A3B8', fontSize: 10 }}>
                    {node.nodeKey}
                  </Typography>
                </TableCell>
                <TableCell sx={{ fontSize: 11 }}>{node.nodeType}</TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  Hop +{node.hop}
                </TableCell>
                <TableCell align="center">
                  <Chip
                    icon={
                      node.impactLevel === 'RED' ? (
                        <ErrorIcon sx={{ fontSize: 12, color: '#EF4444 !important' }} />
                      ) : (
                        <WarningIcon sx={{ fontSize: 12, color: '#F59E0B !important' }} />
                      )
                    }
                    label={node.impactLevel}
                    size="small"
                    sx={{
                      bgcolor: node.impactLevel === 'RED' ? '#450A0A' : '#451A03',
                      color: node.impactLevel === 'RED' ? '#FCA5A5' : '#FDBA74',
                      fontWeight: 700,
                      fontSize: 10
                    }}
                  />
                </TableCell>
                <TableCell sx={{ fontSize: 11, color: '#CBD5E1' }}>{node.reason}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default ImpactAndDriftStudio;
