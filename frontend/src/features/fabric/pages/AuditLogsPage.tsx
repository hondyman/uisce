import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Stack,
  TextField,
  Select,
  MenuItem,
  useTheme,
  alpha,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterListIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI } from '../hooks/useIPWhitelist';
import { useNotification } from '../../../hooks/useNotification';

interface AuditLog {
  id: string;
  action: 'CREATE' | 'UPDATE' | 'DELETE' | 'ASSIGN';
  targetType: 'IP_ADDRESS' | 'TENANT_ASSIGNMENT';
  targetId: string;
  description: string;
  timestamp: string;
  user?: string;
}

const AuditLogsPage: React.FC = () => {
  const theme = useTheme();
  const api = useIPWhitelistAPI();
  const notification = useNotification();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [filteredLogs, setFilteredLogs] = useState<AuditLog[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [actionFilter, setActionFilter] = useState<'all' | 'CREATE' | 'UPDATE' | 'DELETE' | 'ASSIGN'>('all');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadAuditLogs = async () => {
      try {
        // Fetch all IPs to generate mock audit logs from their metadata
        const ips = await api.fetchAllIPWhitelist();
        
        // Generate mock audit logs from IP data
        const mockLogs: AuditLog[] = ips
          .map((ip, idx) => ({
            id: `${ip.ipAddress}-${idx}`,
            action: 'CREATE' as const,
            targetType: 'IP_ADDRESS' as const,
            targetId: ip.ipAddress,
            description: `IP address ${ip.ipAddress} was added${ip.label ? ` with label "${ip.label}"` : ''}`,
            timestamp: ip.createdAt || new Date().toISOString(),
            user: 'System Admin',
          }))
          .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

        setLogs(mockLogs);
        setFilteredLogs(mockLogs);
      } catch (err) {
        notification.error('Failed to load audit logs');
      } finally {
        setLoading(false);
      }
    };

    loadAuditLogs();
  }, [api]);

  useEffect(() => {
    let filtered = [...logs];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        log =>
          log.targetId.toLowerCase().includes(query) ||
          log.description.toLowerCase().includes(query) ||
          (log.user && log.user.toLowerCase().includes(query))
      );
    }

    if (actionFilter !== 'all') {
      filtered = filtered.filter(log => log.action === actionFilter);
    }

    setFilteredLogs(filtered);
  }, [searchQuery, actionFilter, logs]);

  const getActionColor = (action: string) => {
    switch (action) {
      case 'CREATE':
        return 'success';
      case 'UPDATE':
        return 'info';
      case 'DELETE':
        return 'error';
      case 'ASSIGN':
        return 'warning';
      default:
        return 'default';
    }
  };

  const getActionLabel = (action: string) => {
    switch (action) {
      case 'CREATE':
        return '✨ Created';
      case 'UPDATE':
        return '✏️ Updated';
      case 'DELETE':
        return '🗑️ Deleted';
      case 'ASSIGN':
        return '🔗 Assigned';
      default:
        return action;
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack spacing={3}>
        {/* Header */}
        <Box>
          <Typography variant="h4" fontWeight={900} gutterBottom>
            Audit Logs
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Track all changes to IP whitelist configurations and tenant assignments
          </Typography>
        </Box>

        {/* Filters */}
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
          <TextField
            size="small"
            placeholder="Search by IP, description, or user..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.active' }} />
            }}
            sx={{ flex: 1, maxWidth: { xs: '100%', md: 400 } }}
          />
          <Select
            size="small"
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value as any)}
            startAdornment={<FilterListIcon sx={{ mr: 1, color: 'action.active' }} />}
            sx={{ minWidth: 150 }}
          >
            <MenuItem value="all">All Actions</MenuItem>
            <MenuItem value="CREATE">Created</MenuItem>
            <MenuItem value="UPDATE">Updated</MenuItem>
            <MenuItem value="DELETE">Deleted</MenuItem>
            <MenuItem value="ASSIGN">Assigned</MenuItem>
          </Select>
        </Stack>

        {/* Logs Table */}
        <TableContainer
          component={Paper}
          sx={{
            borderRadius: 1,
            border: 1,
            borderColor: 'divider',
            '&::-webkit-scrollbar': {
              height: '6px'
            },
            '&::-webkit-scrollbar-track': {
              bgcolor: alpha(theme.palette.primary.main, 0.05)
            },
            '&::-webkit-scrollbar-thumb': {
              bgcolor: alpha(theme.palette.primary.main, 0.2),
              borderRadius: '3px',
              '&:hover': {
                bgcolor: alpha(theme.palette.primary.main, 0.4)
              }
            }
          }}
        >
          <Table size="small">
            <TableHead>
              <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(0, 0, 0, 0.3)' : 'grey.50' }}>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  Timestamp
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  Action
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  Target
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  Description
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  User
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">Loading audit logs...</Typography>
                  </TableCell>
                </TableRow>
              ) : filteredLogs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      {logs.length === 0 ? 'No audit logs available' : 'No matching logs found'}
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                filteredLogs.map((log) => (
                  <TableRow key={log.id} hover>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {new Date(log.timestamp).toLocaleDateString()} {new Date(log.timestamp).toLocaleTimeString()}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={getActionLabel(log.action)}
                        color={getActionColor(log.action) as any}
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontFamily="monospace" sx={{ wordBreak: 'break-all' }}>
                        {log.targetId}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {log.description}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {log.user || '—'}
                      </Typography>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Stats Footer */}
        <Paper sx={{ p: 2, bgcolor: alpha(theme.palette.info.main, 0.05) }}>
          <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" alignItems="center">
            <Typography variant="body2" color="text.secondary">
              Showing {filteredLogs.length} of {logs.length} audit logs
            </Typography>
            <Stack direction="row" spacing={3}>
              <Box>
                <Typography variant="caption" color="text.secondary">Creates:</Typography>
                <Typography variant="body2" fontWeight={600}>
                  {logs.filter(l => l.action === 'CREATE').length}
                </Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary">Updates:</Typography>
                <Typography variant="body2" fontWeight={600}>
                  {logs.filter(l => l.action === 'UPDATE').length}
                </Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary">Deletes:</Typography>
                <Typography variant="body2" fontWeight={600}>
                  {logs.filter(l => l.action === 'DELETE').length}
                </Typography>
              </Box>
            </Stack>
          </Stack>
        </Paper>
      </Stack>
    </Box>
  );
};

export default AuditLogsPage;
