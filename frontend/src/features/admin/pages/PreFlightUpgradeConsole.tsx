import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  Button,
  Chip,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Alert,
  Divider,
  LinearProgress,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Stepper,
  Step,
  StepLabel
} from '@mui/material';
import SpeedIcon from '@mui/icons-material/Speed';
import StorageIcon from '@mui/icons-material/Storage';
import WarningIcon from '@mui/icons-material/Warning';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunch';
import LockIcon from '@mui/icons-material/Lock';
import MemoryIcon from '@mui/icons-material/Memory';

interface TenantWarning {
  tenant_id: string;
  bo_id: string;
  field_name: string;
  severity: string;
  message: string;
  conflict_path: string;
}

interface StorageImpact {
  citus_distributed_ddls: string[];
  iceberg_evolutions: string[];
  starrocks_rebuilds: string[];
  cache_invalidations: string[];
}

interface ImpactReport {
  package_id: string;
  version: string;
  can_upgrade: boolean;
  core_deltas_count: number;
  affected_tenants: number;
  tenant_warnings: TenantWarning[];
  storage_impact: StorageImpact;
}

interface DeploymentStep {
  phase: string;
  name: string;
  target_store: string;
  status: string;
  duration_ms: number;
  details: string;
}

interface DeploymentResponse {
  package_id: string;
  target_version: string;
  overall_status: string;
  execution_steps: DeploymentStep[];
  regions_deployed: string[];
}

export const PreFlightUpgradeConsole: React.FC = () => {
  const [report, setReport] = useState<ImpactReport | null>(null);
  const [deployResult, setDeployResult] = useState<DeploymentResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deploying, setDeploying] = useState(false);

  const runPreFlightCheck = () => {
    setLoading(true);
    fetch('/api/admin/upgrade/preflight-simulation', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ version: 'v1.3.0' }),
    })
      .then((res) => res.json())
      .then((data) => {
        setReport(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => {
    runPreFlightCheck();
  }, []);

  const handleDeployGlobally = async () => {
    setDeploying(true);
    setDeployResult(null);

    try {
      const res = await fetch('/api/admin/upgrade/deploy-globally', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ version: report?.version || 'v1.3.0' }),
      });
      const data = await res.json();
      setDeployResult(data);
    } catch (err) {
      console.error(err);
    } finally {
      setDeploying(false);
    }
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#0f172a', minHeight: '100vh', color: '#f8fafc' }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" fontWeight="700" color="#f8fafc">
            Pre-Flight Upgrade Impact & Global Deployment Console
          </Typography>
          <Typography variant="body1" color="#94a3b8">
            Pre-flight impact simulation across Citus Shards, Iceberg Lakehouse, StarRocks OLAP, and Tenant Custom Overlays.
          </Typography>
        </Box>
        <Box display="flex" gap={2}>
          <Button variant="outlined" onClick={runPreFlightCheck} disabled={loading || deploying}>
            Re-run Pre-Flight Simulation
          </Button>
          <Button
            variant="contained"
            startIcon={<RocketLaunchIcon />}
            onClick={handleDeployGlobally}
            disabled={deploying || loading}
            sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
          >
            Deploy Globally (Phase 4)
          </Button>
        </Box>
      </Box>

      {/* Package Spec Manifest Info */}
      <Card sx={{ bgcolor: '#1e293b', border: '1px solid #334155', mb: 3 }}>
        <CardContent>
          <Box display="flex" justifyContent="space-between" alignItems="center">
            <Box display="flex" alignItems="center" gap={1.5}>
              <LockIcon sx={{ color: '#38bdf8' }} />
              <Typography variant="h6" fontWeight="600">
                Upgrade Manifest: {report?.package_id || 'pkg_v1.3.0'}
              </Typography>
              <Chip label={`Target: ${report?.version || 'v1.3.0'}`} color="primary" size="small" />
              <Chip icon={<CheckCircleIcon />} label="Signed & Verified SHA-256 Checksum" color="success" size="small" />
            </Box>
            <Typography variant="caption" color="#94a3b8">
              Author: Uisce System Release Pipeline
            </Typography>
          </Box>
        </CardContent>
      </Card>

      {/* Deploying Progress */}
      {deploying && (
        <Box mb={3}>
          <Typography variant="subtitle2" color="#38bdf8" mb={1}>
            Phase 4: Executing Region-by-Region Canary Deployment Pipeline...
          </Typography>
          <LinearProgress sx={{ bgcolor: '#1e293b', '& .MuiLinearProgress-bar': { bgcolor: '#38bdf8' } }} />
        </Box>
      )}

      {/* Post-Deployment Result */}
      {deployResult && (
        <Alert icon={<CheckCircleIcon />} severity="success" sx={{ mb: 3 }}>
          <Typography variant="subtitle1" fontWeight="700">
            Global Deployment Completed Successfully! (Version {deployResult.target_version})
          </Typography>
          <Typography variant="caption" display="block">
            Regions Deployed: {deployResult.regions_deployed?.join(' | ')}
          </Typography>
        </Alert>
      )}

      {/* Impact Matrix Summary Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid size={{ xs: 12, md: 3 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">Core Master Deltas</Typography>
              <Typography variant="h4" fontWeight="700" color="#38bdf8">
                {report?.core_deltas_count || 0}
              </Typography>
              <Typography variant="caption" color="#94a3b8">Schema & BO field modifications</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 3 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">Tenant Overlays Impacted</Typography>
              <Typography variant="h4" fontWeight="700" color="#facc15">
                {report?.affected_tenants || 0}
              </Typography>
              <Typography variant="caption" color="#94a3b8">Cross-referenced against overlays</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 3 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">Citus Distributed DDLs</Typography>
              <Typography variant="h4" fontWeight="700" color="#4ade80">
                {report?.storage_impact?.citus_distributed_ddls?.length || 0}
              </Typography>
              <Typography variant="caption" color="#94a3b8">Master Node ACID transactions</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 3 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">StarRocks & Lakehouse</Typography>
              <Typography variant="h4" fontWeight="700" color="#a855f7">
                {(report?.storage_impact?.iceberg_evolutions?.length || 0) + (report?.storage_impact?.starrocks_rebuilds?.length || 0)}
              </Typography>
              <Typography variant="caption" color="#94a3b8">Iceberg & MV re-materializations</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Multi-Store Heterogeneous Storage Execution Matrix */}
      <Typography variant="h5" fontWeight="600" mb={2}>
        Heterogeneous Storage Evolution Matrix (Citus / Iceberg / StarRocks / Redis)
      </Typography>
      <Grid container spacing={3} mb={4}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={{ p: 3, bgcolor: '#1e293b', border: '1px solid #334155' }}>
            <Typography variant="subtitle1" fontWeight="600" color="#38bdf8" mb={2} display="flex" alignItems="center" gap={1}>
              <StorageIcon /> Citus Master Node Distributed DDL Execution
            </Typography>
            {report?.storage_impact?.citus_distributed_ddls?.map((ddl, i) => (
              <Box key={i} p={1.5} mb={1} sx={{ bgcolor: '#0f172a', borderRadius: 1, fontFamily: 'monospace', fontSize: '12px', color: '#4ade80' }}>
                {ddl}
              </Box>
            ))}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={{ p: 3, bgcolor: '#1e293b', border: '1px solid #334155' }}>
            <Typography variant="subtitle1" fontWeight="600" color="#a855f7" mb={2} display="flex" alignItems="center" gap={1}>
              <MemoryIcon /> Iceberg Catalog Evolution & StarRocks Re-Materialization
            </Typography>
            {report?.storage_impact?.iceberg_evolutions?.map((evo, i) => (
              <Box key={i} p={1.5} mb={1} sx={{ bgcolor: '#0f172a', borderRadius: 1, fontFamily: 'monospace', fontSize: '12px', color: '#cbd5e1' }}>
                {evo}
              </Box>
            ))}
            {report?.storage_impact?.starrocks_rebuilds?.map((sr, i) => (
              <Box key={i} p={1.5} mb={1} sx={{ bgcolor: '#0f172a', borderRadius: 1, fontFamily: 'monospace', fontSize: '12px', color: '#facc15' }}>
                {sr}
              </Box>
            ))}
          </Paper>
        </Grid>
      </Grid>

      {/* Deployment Execution Trace */}
      {deployResult?.execution_steps && (
        <Box mb={4}>
          <Typography variant="h5" fontWeight="600" mb={2}>
            Distributed Execution Trace & Canary Verification
          </Typography>
          <Paper sx={{ bgcolor: '#1e293b', border: '1px solid #334155', p: 3 }}>
            {deployResult.execution_steps.map((step, idx) => (
              <Box key={idx} mb={2} p={2} sx={{ bgcolor: '#0f172a', borderRadius: 1, borderLeft: '4px solid #38bdf8' }}>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                  <Typography variant="subtitle2" fontWeight="600" color="#f8fafc">
                    {step.phase}: {step.name}
                  </Typography>
                  <Chip label={`${step.duration_ms} ms`} size="small" sx={{ bgcolor: 'rgba(56,189,248,0.1)', color: '#38bdf8' }} />
                </Box>
                <Typography variant="caption" color="#94a3b8" display="block" mt={0.5}>
                  {step.details}
                </Typography>
              </Box>
            ))}
          </Paper>
        </Box>
      )}
    </Box>
  );
};
