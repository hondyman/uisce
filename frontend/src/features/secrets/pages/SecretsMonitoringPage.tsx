import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Card,
  CardContent,
  Grid,
  Chip,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Alert,
  Button,
} from '@mui/material';
import MonitorHeartIcon from '@mui/icons-material/MonitorHeart';
import RefreshIcon from '@mui/icons-material/Refresh';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import WarningIcon from '@mui/icons-material/Warning';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import ScheduleIcon from '@mui/icons-material/Schedule';
import LockIcon from '@mui/icons-material/Lock';
import { useQuery } from '@apollo/client';
import { gql } from '@apollo/client';

const GET_MONITORING_DATA = gql`
  query GetMonitoringData($tenantId: uuid!) {
    secret_metadata(where: { tenant_id: { _eq: $tenantId }, deleted_at: { _is_null: true } }) {
      id
      name
      path
      secret_type
      ttl
      updated_at
    }
    secrets_needing_rotation: secret_metadata(
      where: { 
        tenant_id: { _eq: $tenantId }, 
        deleted_at: { _is_null: true },
        ttl: { _is_null: false }
      }
    ) {
      id
      name
      ttl
      updated_at
    }
    recent_access: secret_access_log(
      where: { secret_metadata: { tenant_id: { _eq: $tenantId } } }
      order_by: { requested_at: desc }
      limit: 50
    ) {
      success
      action
      requested_at
    }
  }
`;

interface SecretsMonitoringPageProps {
  tenantId: string;
}

export default function SecretsMonitoringPage({ tenantId }: SecretsMonitoringPageProps) {
  const { data, loading, refetch } = useQuery(GET_MONITORING_DATA, {
    variables: { tenantId },
    skip: !tenantId,
    pollInterval: 30000, // Auto-refresh every 30 seconds
  });

  const secrets = data?.secret_metadata || [];
  const needsRotation = data?.secrets_needing_rotation || [];
  const recentAccess = data?.recent_access || [];

  // Calculate metrics
  const totalSecrets = secrets.length;
  const rotationEnabled = secrets.filter((s: any) => s.ttl).length;
  const successfulAccess = recentAccess.filter((a: any) => a.success).length;
  const failedAccess = recentAccess.filter((a: any) => !a.success).length;
  const successRate = recentAccess.length > 0 ? (successfulAccess / recentAccess.length) * 100 : 100;

  // Check for secrets needing rotation
  const overdueRotations = needsRotation.filter((s: any) => {
    if (!s.ttl || !s.updated_at) return false;
    const lastUpdate = new Date(s.updated_at);
    const ttlMatch = s.ttl.match(/(\d+)\s*(day|hour|minute)/i);
    if (!ttlMatch) return false;
    const value = parseInt(ttlMatch[1]);
    const unit = ttlMatch[2].toLowerCase();
    const msMultiplier = unit === 'day' ? 86400000 : unit === 'hour' ? 3600000 : 60000;
    const dueDate = new Date(lastUpdate.getTime() + value * msMultiplier);
    return new Date() > dueDate;
  });

  const getHealthStatus = () => {
    if (overdueRotations.length > 0 || successRate < 90) return 'warning';
    if (failedAccess > successfulAccess * 0.1) return 'error';
    return 'healthy';
  };

  const healthStatus = getHealthStatus();

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <MonitorHeartIcon /> Secrets Monitoring
          </Typography>
          <Typography color="text.secondary">
            Real-time health and status of your secrets infrastructure
          </Typography>
        </Box>
        <Button variant="outlined" startIcon={<RefreshIcon />} onClick={() => refetch()}>
          Refresh
        </Button>
      </Box>

      {/* Health Status Banner */}
      <Alert 
        severity={healthStatus === 'healthy' ? 'success' : healthStatus === 'warning' ? 'warning' : 'error'}
        sx={{ mb: 3 }}
        icon={healthStatus === 'healthy' ? <CheckCircleIcon /> : healthStatus === 'warning' ? <WarningIcon /> : <ErrorIcon />}
      >
        <strong>System Health: </strong>
        {healthStatus === 'healthy' && 'All secrets are healthy and up to date.'}
        {healthStatus === 'warning' && `${overdueRotations.length} secret(s) need rotation. Success rate: ${successRate.toFixed(1)}%`}
        {healthStatus === 'error' && 'Critical issues detected. Review failed access attempts immediately.'}
      </Alert>

      {/* Metrics Grid */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Box>
                  <Typography color="text.secondary" gutterBottom>Total Secrets</Typography>
                  <Typography variant="h4">{totalSecrets}</Typography>
                </Box>
                <LockIcon sx={{ fontSize: 40, color: 'primary.main', opacity: 0.5 }} />
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Box>
                  <Typography color="text.secondary" gutterBottom>Auto-Rotation</Typography>
                  <Typography variant="h4">{rotationEnabled}</Typography>
                </Box>
                <ScheduleIcon sx={{ fontSize: 40, color: 'info.main', opacity: 0.5 }} />
              </Box>
              <LinearProgress 
                variant="determinate" 
                value={totalSecrets > 0 ? (rotationEnabled / totalSecrets) * 100 : 0} 
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card sx={{ borderLeft: overdueRotations.length > 0 ? '4px solid orange' : '4px solid green' }}>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Box>
                  <Typography color="text.secondary" gutterBottom>Needs Rotation</Typography>
                  <Typography variant="h4" color={overdueRotations.length > 0 ? 'warning.main' : 'success.main'}>
                    {overdueRotations.length}
                  </Typography>
                </Box>
                {overdueRotations.length > 0 ? (
                  <TrendingUpIcon sx={{ fontSize: 40, color: 'warning.main', opacity: 0.5 }} />
                ) : (
                  <TrendingDownIcon sx={{ fontSize: 40, color: 'success.main', opacity: 0.5 }} />
                )}
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Box>
                  <Typography color="text.secondary" gutterBottom>Access Success Rate</Typography>
                  <Typography variant="h4" color={successRate >= 95 ? 'success.main' : successRate >= 80 ? 'warning.main' : 'error.main'}>
                    {successRate.toFixed(1)}%
                  </Typography>
                </Box>
              </Box>
              <LinearProgress 
                variant="determinate" 
                value={successRate} 
                color={successRate >= 95 ? 'success' : successRate >= 80 ? 'warning' : 'error'}
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Secrets Needing Rotation */}
      {overdueRotations.length > 0 && (
        <Paper sx={{ mb: 3 }}>
          <Box sx={{ p: 2, bgcolor: 'warning.lighter', borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <WarningIcon color="warning" /> Secrets Needing Rotation
            </Typography>
          </Box>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Secret</TableCell>
                <TableCell>TTL</TableCell>
                <TableCell>Last Rotated</TableCell>
                <TableCell>Status</TableCell>
                <TableCell align="right">Action</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {overdueRotations.map((secret: any) => (
                <TableRow key={secret.id}>
                  <TableCell>{secret.name}</TableCell>
                  <TableCell>{secret.ttl}</TableCell>
                  <TableCell>{new Date(secret.updated_at).toLocaleDateString()}</TableCell>
                  <TableCell>
                    <Chip label="Overdue" color="warning" size="small" />
                  </TableCell>
                  <TableCell align="right">
                    <Button size="small" variant="outlined" startIcon={<RefreshIcon />}>
                      Rotate Now
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}

      {/* Recent Activity */}
      <Paper>
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Typography variant="h6">Recent Access Activity</Typography>
        </Box>
        <Box sx={{ p: 2 }}>
          <Grid container spacing={1}>
            {recentAccess.slice(0, 20).map((access: any, index: number) => (
              <Grid item key={index}>
                <Tooltip title={`${access.action} - ${new Date(access.requested_at).toLocaleString()}`}>
                  <Box
                    sx={{
                      width: 12,
                      height: 12,
                      borderRadius: '2px',
                      bgcolor: access.success ? 'success.main' : 'error.main',
                      opacity: 0.8,
                    }}
                  />
                </Tooltip>
              </Grid>
            ))}
          </Grid>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            Last 20 access attempts • Green = Success, Red = Failed
          </Typography>
        </Box>
      </Paper>
    </Box>
  );
}
