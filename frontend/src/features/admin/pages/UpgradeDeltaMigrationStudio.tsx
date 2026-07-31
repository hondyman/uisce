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
  CircularProgress
} from '@mui/material';
import MergeTypeIcon from '@mui/icons-material/MergeType';
import PublishedWithChangesIcon from '@mui/icons-material/PublishedWithChanges';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import { useTenant } from '../../../contexts/TenantContext';

interface ConflictItem {
  property_path: string;
  reason: string;
  ancestor_value: any;
  modified_value: any;
  target_value: any;
}

interface TenantDelta {
  tenant_id: string;
  base_version: string;
  core_master_spec: any;
  tenant_overlay: any;
  custom_fields: any[];
  modified_count: number;
}

export const UpgradeDeltaMigrationStudio: React.FC = () => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || 'core';

  const [delta, setDelta] = useState<TenantDelta | null>(null);
  const [loading, setLoading] = useState(false);
  const [upgradeStatus, setUpgradeStatus] = useState<string | null>(null);
  const [conflicts, setConflicts] = useState<ConflictItem[]>([]);

  const fetchDeltas = () => {
    setLoading(true);
    fetch(`/api/admin/tenants/deltas?tenant_id=${tenantId}`)
      .then((res) => res.json())
      .then((data) => {
        setDelta(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => {
    fetchDeltas();
  }, [tenantId]);

  const handleRunUpgrade = async () => {
    setLoading(true);
    setUpgradeStatus(null);
    setConflicts([]);

    try {
      const res = await fetch(`/api/admin/tenants/upgrade?tenant_id=${tenantId}&target_version=v1.3.0`, {
        method: 'POST',
      });
      const data = await res.json();
      setUpgradeStatus(data.status);
      setConflicts(data.conflicts || []);
      fetchDeltas();
    } catch (err: any) {
      setUpgradeStatus('UPGRADE_FAILED');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#0f172a', minHeight: '100vh', color: '#f8fafc' }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Box>
          <Typography variant="h4" fontWeight="700" color="#f8fafc">
            3-Way Merge & Upgrade Migration Studio
          </Typography>
          <Typography variant="body1" color="#94a3b8">
            Upgrade-Safe Metadata Migration: Compares System Core (master gold_copy) against Tenant Custom Overlay deltas.
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={loading ? <CircularProgress color="inherit" sx={{ fontSize: 20 }}/> : <PublishedWithChangesIcon />}
          onClick={handleRunUpgrade}
          disabled={loading}
          sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
        >
          Run 3-Way Merge Upgrade (v1.3.0)
        </Button>
      </Box>

      {/* Upgrade Status Banner */}
      {upgradeStatus && (
        <Box mb={3}>
          {upgradeStatus === 'UPG_SUCCESS' ? (
            <Alert icon={<CheckCircleIcon />} severity="success">
              Upgrade Executed Successfully! Zero conflicts detected. Client overlay modifications preserved cleanly.
            </Alert>
          ) : upgradeStatus === 'UPGRADE_PENDING_REVIEW' ? (
            <Alert icon={<WarningAmberIcon />} severity="warning">
              Upgrade Completed with Review Items! Non-conflicting changes auto-merged; conflicts routed to Exception Queue below.
            </Alert>
          ) : (
            <Alert severity="error">Upgrade Execution Failed.</Alert>
          )}
        </Box>
      )}

      {/* Delta Overview Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">Core Master Version</Typography>
              <Typography variant="h4" fontWeight="700" color="#38bdf8">v1.3.0</Typography>
              <Typography variant="caption" color="#94a3b8">Master Tenant (gold_copy = true)</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">Active Tenant Custom Overlay</Typography>
              <Typography variant="h4" fontWeight="700" color="#4ade80">
                {delta?.modified_count || 0} <span style={{ fontSize: '16px', color: '#94a3b8' }}>Deltas</span>
              </Typography>
              <Typography variant="caption" color="#94a3b8">Tenant {tenantId} Overlay</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Typography variant="body2" color="#94a3b8">3-Way Conflict Status</Typography>
              <Typography variant="h4" fontWeight="700" color={conflicts.length > 0 ? '#f43f5e' : '#4ade80'}>
                {conflicts.length} <span style={{ fontSize: '16px', color: '#94a3b8' }}>Conflicts</span>
              </Typography>
              <Typography variant="caption" color="#94a3b8">Painless Seamless Migration Guard</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* 3-Way Merge Visual Diff Queue */}
      <Typography variant="h5" fontWeight="600" mb={2}>
        3-Way Diff & Conflict Resolution Queue
      </Typography>
      <Paper sx={{ bgcolor: '#1e293b', border: '1px solid #334155', overflow: 'hidden', mb: 4 }}>
        <Table>
          <TableHead sx={{ bgcolor: '#0f172a' }}>
            <TableRow>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Property Path</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Ancestor (Base v1.0.0)</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Client Overlay (Modified)</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Target Core (Base v1.3.0)</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Resolution Action</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {conflicts.length > 0 ? (
              conflicts.map((item, idx) => (
                <TableRow key={idx}>
                  <TableCell sx={{ color: '#38bdf8', fontWeight: 600 }}>{item.property_path}</TableCell>
                  <TableCell sx={{ color: '#94a3b8' }}>{JSON.stringify(item.ancestor_value)}</TableCell>
                  <TableCell sx={{ color: '#4ade80' }}>{JSON.stringify(item.modified_value)}</TableCell>
                  <TableCell sx={{ color: '#facc15' }}>{JSON.stringify(item.target_value)}</TableCell>
                  <TableCell>
                    <Chip label="Auto-Kept Core Upgrade" size="small" color="warning" />
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ color: '#94a3b8', py: 3 }}>
                  No active 3-way merge conflicts. All client custom overlays match cleanly with core updates.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </Paper>

      {/* Tenant Custom Attributes Summary */}
      <Typography variant="h5" fontWeight="600" mb={2}>
        Tenant Custom Attributes Overlay Metadata
      </Typography>
      <Paper sx={{ bgcolor: '#1e293b', border: '1px solid #334155', p: 3 }}>
        {delta?.custom_fields && delta.custom_fields.length > 0 ? (
          <Table size="small">
            <TableHead sx={{ bgcolor: '#0f172a' }}>
              <TableRow>
                <TableCell sx={{ color: '#94a3b8' }}>Attribute</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>Display Name</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>Data Type</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>JSONB Path</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {delta.custom_fields.map((attr, i) => (
                <TableRow key={i}>
                  <TableCell sx={{ color: '#38bdf8' }}>{attr.attribute_name}</TableCell>
                  <TableCell sx={{ color: '#f8fafc' }}>{attr.display_name}</TableCell>
                  <TableCell sx={{ color: '#4ade80' }}>{attr.data_type}</TableCell>
                  <TableCell sx={{ color: '#cbd5e1', fontFamily: 'monospace' }}>{attr.jsonb_path}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <Typography color="#94a3b8">No tenant custom attribute overlays registered for tenant {tenantId}.</Typography>
        )}
      </Paper>
    </Box>
  );
};
