import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  IconButton,
  Tooltip,
  Card,
  CardContent,
  Grid,
  Alert,
} from '@mui/material';
import HistoryIcon from '@mui/icons-material/History';
import SearchIcon from '@mui/icons-material/Search';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import VisibilityIcon from '@mui/icons-material/Visibility';
import RefreshIcon from '@mui/icons-material/Refresh';
import WarningIcon from '@mui/icons-material/Warning';
import { useQuery } from '@apollo/client';
import { gql } from '@apollo/client';

const GET_AUDIT_LOGS = gql`
  query GetAuditLogs($tenantId: uuid!, $limit: Int, $offset: Int) {
    secret_access_log(
      where: { secret_metadata: { tenant_id: { _eq: $tenantId } } }
      order_by: { requested_at: desc }
      limit: $limit
      offset: $offset
    ) {
      id
      action
      user_id
      ip_address
      user_agent
      requested_at
      success
      error_message
      abac_result
      secret_metadata {
        name
        path
      }
    }
  }
`;

interface AuditLog {
  id: string;
  action: string;
  user_id?: string;
  ip_address?: string;
  user_agent?: string;
  requested_at: string;
  success: boolean;
  error_message?: string;
  abac_result?: Record<string, any>;
  secret_metadata?: {
    name: string;
    path: string;
  };
}

interface SecretsAuditPageProps {
  tenantId: string;
}

export default function SecretsAuditPage({ tenantId }: SecretsAuditPageProps) {
  const [actionFilter, setActionFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const { data, loading, refetch } = useQuery(GET_AUDIT_LOGS, {
    variables: { tenantId, limit: 100, offset: 0 },
    skip: !tenantId,
  });

  const logs: AuditLog[] = data?.secret_access_log || [];

  const filteredLogs = logs.filter((log) => {
    if (actionFilter !== 'all' && log.action !== actionFilter) return false;
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      return (
        log.secret_metadata?.name?.toLowerCase().includes(query) ||
        log.secret_metadata?.path?.toLowerCase().includes(query) ||
        log.user_id?.toLowerCase().includes(query) ||
        log.ip_address?.includes(query)
      );
    }
    return true;
  });

  const successCount = logs.filter(l => l.success).length;
  const failedCount = logs.filter(l => !l.success).length;
  const uniqueUsers = new Set(logs.map(l => l.user_id)).size;

  const getActionColor = (action: string) => {
    switch (action) {
      case 'read': return 'info';
      case 'rotate': return 'warning';
      case 'create': return 'success';
      case 'delete': return 'error';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <HistoryIcon /> Secrets Audit Log
          </Typography>
          <Typography color="text.secondary">
            Track all secret access, modifications, and rotations
          </Typography>
        </Box>
        <Tooltip title="Refresh">
          <IconButton onClick={() => refetch()}>
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Box>

      {/* Stats Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Total Events</Typography>
              <Typography variant="h4">{logs.length}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card sx={{ borderLeft: '4px solid green' }}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Successful</Typography>
              <Typography variant="h4" color="success.main">{successCount}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card sx={{ borderLeft: '4px solid red' }}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Failed</Typography>
              <Typography variant="h4" color="error.main">{failedCount}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Unique Users</Typography>
              <Typography variant="h4">{uniqueUsers}</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* AI Anomaly Alert */}
      {failedCount > logs.length * 0.1 && (
        <Alert severity="warning" sx={{ mb: 2 }} icon={<WarningIcon />}>
          <strong>Anomaly Detected:</strong> High failure rate detected ({((failedCount / logs.length) * 100).toFixed(1)}%). 
          Review failed access attempts for potential security issues.
        </Alert>
      )}

      {/* Filters */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
          <TextField
            size="small"
            placeholder="Search secrets, users, IPs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{ startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.active' }} /> }}
            sx={{ minWidth: 300 }}
          />
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel>Action</InputLabel>
            <Select
              value={actionFilter}
              onChange={(e) => setActionFilter(e.target.value)}
              label="Action"
            >
              <MenuItem value="all">All Actions</MenuItem>
              <MenuItem value="read">Read</MenuItem>
              <MenuItem value="rotate">Rotate</MenuItem>
              <MenuItem value="create">Create</MenuItem>
              <MenuItem value="delete">Delete</MenuItem>
            </Select>
          </FormControl>
        </Box>
      </Paper>

      {/* Audit Log Table */}
      <Paper>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Time</TableCell>
              <TableCell>Secret</TableCell>
              <TableCell>Action</TableCell>
              <TableCell>User</TableCell>
              <TableCell>IP Address</TableCell>
              <TableCell>Status</TableCell>
              <TableCell align="right">Details</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} align="center">Loading...</TableCell>
              </TableRow>
            ) : filteredLogs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">No audit logs found</TableCell>
              </TableRow>
            ) : (
              filteredLogs.map((log) => (
                <TableRow key={log.id} hover sx={{ bgcolor: !log.success ? 'error.lighter' : undefined }}>
                  <TableCell>
                    <Typography variant="body2">
                      {new Date(log.requested_at).toLocaleString()}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {log.secret_metadata?.name || 'Unknown'}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {log.secret_metadata?.path}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={log.action.toUpperCase()}
                      size="small"
                      color={getActionColor(log.action) as any}
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{log.user_id || 'system'}</Typography>
                  </TableCell>
                  <TableCell>
                    <code style={{ fontSize: '0.85em' }}>{log.ip_address || '-'}</code>
                  </TableCell>
                  <TableCell>
                    {log.success ? (
                      <Chip icon={<CheckCircleIcon />} label="Success" size="small" color="success" variant="outlined" />
                    ) : (
                      <Tooltip title={log.error_message || 'Unknown error'}>
                        <Chip icon={<ErrorIcon />} label="Failed" size="small" color="error" variant="outlined" />
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="View Details">
                      <IconButton size="small">
                        <VisibilityIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Paper>
    </Box>
  );
}
