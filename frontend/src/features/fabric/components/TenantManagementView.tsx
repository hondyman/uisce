import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  IconButton,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Select,
  MenuItem,
  Stack,
  useTheme,
  alpha,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Menu,
  Checkbox,
  Avatar,
  LinearProgress,
} from '@mui/material';
import {
  Delete as DeleteIcon,
  Add as AddIcon,
  Download as DownloadIcon,
  Search as SearchIcon,
  Sort as SortIcon,
  GridView as GridViewIcon,
  ViewAgenda as ViewAgendaIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI, useIPWhitelistTable } from '../hooks/useIPWhitelist';
import { useNotification } from '../../../hooks/useNotification';
import { IPWhitelistEntry, Tenant } from '../types/ipWhitelist';
import IPAddEditDialog from './IPAddEditDialog';
import { exportTenantReport } from '../utils/exportUtils';
import TenantTypeahead from './TenantTypeahead';
import AddIPForm from './AddIPForm';
import DashboardPage from '../pages/DashboardPage';
import TenantsManagementPage from '../pages/TenantsManagementPage';
import AuditLogsPage from '../pages/AuditLogsPage';
import SettingsPage from '../pages/SettingsPage';

type SidebarView = 'ip-whitelist' | 'dashboard' | 'tenants' | 'audit-logs' | 'settings';

const TenantManagementView: React.FC = () => {
  const theme = useTheme();
  const [currentView, setCurrentView] = useState<SidebarView>('ip-whitelist');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [searchQuery, setSearchQuery] = useState('');
  const [tenantFilter, setTenantFilter] = useState<string>('');
  const [sortBy, setSortBy] = useState<'ipAddress' | 'label' | 'dateAdded'>('dateAdded');
  const [deleteConfirm, setDeleteConfirm] = useState<IPWhitelistEntry | null>(null);
  const [exportMenuAnchor, setExportMenuAnchor] = useState<null | HTMLElement>(null);
  const [addIPDialogOpen, setAddIPDialogOpen] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState<string>('');
  const [ipWhitelistViewMode, setIPWhitelistViewMode] = useState<'tile' | 'table'>('table');
  const [tenantsViewMode, setTenantsViewMode] = useState<'tile' | 'table'>('tile');

  const api = useIPWhitelistAPI();
  const notification = useNotification();
  const [entries, setEntries] = useState<IPWhitelistEntry[]>([]);
  const [tenants, setTenants] = useState<Tenant[]>([]);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [tenantsList, allEntries] = await Promise.all([
          api.fetchTenants(),
          api.fetchAllIPWhitelist()
        ]);
        setTenants(tenantsList);
        setEntries(allEntries);
      } catch (err) {
        notification.error('Failed to load IP whitelist data');
      }
    };
    loadData();
    // Load data once on component mount only - api and notification are stable
  }, []);

  const filteredEntries = useMemo(() => {
    let filtered = entries;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(entry =>
        entry.ipAddress.toLowerCase().includes(query) ||
        (entry.label && entry.label.toLowerCase().includes(query))
      );
    }

    if (tenantFilter) {
      filtered = filtered.filter(entry =>
        entry.tenantIds.some(id => id === tenantFilter) || (entry as any).allTenants
      );
    }

    return filtered.sort((a, b) => {
      switch (sortBy) {
        case 'ipAddress':
          return a.ipAddress.localeCompare(b.ipAddress);
        case 'label':
          return (a.label || '').localeCompare(b.label || '');
        case 'dateAdded':
        default:
          return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
      }
    });
  }, [entries, searchQuery, tenantFilter, sortBy]);

  const paginatedEntries = useMemo(() => {
    return filteredEntries.slice(page * rowsPerPage, (page + 1) * rowsPerPage);
  }, [filteredEntries, page, rowsPerPage]);

  const handleDeleteIP = async () => {
    if (!deleteConfirm) return;
    try {
      await api.removeIPWhitelist(deleteConfirm.tenantIds[0], deleteConfirm.ipAddress);
      setEntries(prev => prev.filter(e => e.ipAddress !== deleteConfirm.ipAddress));
      notification.success('IP address removed successfully');
      setDeleteConfirm(null);
    } catch (err) {
      notification.error('Failed to remove IP address');
    }
  };

  const handleAddIP = useCallback(() => {
    if (!selectedTenant) {
      notification.error('Please select a tenant first');
      return;
    }
    setAddIPDialogOpen(true);
  }, [selectedTenant, notification]);

  const handleAddIPSubmit = useCallback(async (ipAddress: string, label: string, description: string) => {
    try {
      const success = await api.addIPWhitelist(selectedTenant, ipAddress, label, description);
      if (success) {
        // Refresh entries
        const updated = await api.fetchAllIPWhitelist();
        setEntries(updated);
        notification.success('IP address added successfully');
        setAddIPDialogOpen(false);
      }
    } catch (err) {
      notification.error('Failed to add IP address');
    }
  }, [selectedTenant, api, notification]);

  const handleExport = (format: 'csv' | 'json') => {
    if (format === 'csv') {
      const headers = ['IP Address', 'Label', 'Assigned Tenants', 'Status', 'Date Added'];
      const rows = filteredEntries.map(entry => [
        entry.ipAddress,
        entry.label || '',
        entry.tenantIds.map(id => tenants.find(t => t.id === id)?.displayName || id).join('; '),
        (entry as any).allTenants ? 'Global' : 'Active',
        new Date(entry.createdAt).toLocaleDateString()
      ]);
      const csvContent = [headers, ...rows].map(row => row.map(cell => `"${cell}"`).join(',')).join('\n');
      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `ip-whitelist-${new Date().toISOString().split('T')[0]}.csv`;
      link.click();
      window.URL.revokeObjectURL(url);
    } else {
      const jsonData = filteredEntries.map(entry => ({
        ipAddress: entry.ipAddress,
        label: entry.label || '',
        assignedTenants: entry.tenantIds.map(id => tenants.find(t => t.id === id)?.displayName || id),
        status: (entry as any).allTenants ? 'Global' : 'Active',
        dateAdded: new Date(entry.createdAt).toLocaleDateString()
      }));
      const json = JSON.stringify(jsonData, null, 2);
      const blob = new Blob([json], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `ip-whitelist-${new Date().toISOString().split('T')[0]}.json`;
      link.click();
      window.URL.revokeObjectURL(url);
    }
    setExportMenuAnchor(null);
  };

  const handleExportTenants = (format: 'csv' | 'json') => {
    if (format === 'csv') {
      const headers = ['ID', 'Name', 'IP Count', 'Status'];
      const rows = tenants.map(tenant => [
        tenant.id,
        tenant.displayName,
        entries.filter(e => e.tenantIds.includes(tenant.id)).length,
        'Active'
      ]);
      const csvContent = [headers, ...rows].map(row => row.map(cell => `"${cell}"`).join(',')).join('\n');
      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `tenants-${new Date().toISOString().split('T')[0]}.csv`;
      link.click();
      window.URL.revokeObjectURL(url);
    } else {
      const jsonData = tenants.map(tenant => ({
        id: tenant.id,
        name: tenant.displayName,
        ipCount: entries.filter(e => e.tenantIds.includes(tenant.id)).length,
        status: 'Active'
      }));
      const json = JSON.stringify(jsonData, null, 2);
      const blob = new Blob([json], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `tenants-${new Date().toISOString().split('T')[0]}.json`;
      link.click();
      window.URL.revokeObjectURL(url);
    }
  };

  const getTenantInitials = (tenantId: string): string => {
    const tenant = tenants.find(t => t.id === tenantId);
    if (!tenant) return '?';
    const name = tenant.displayName || tenant.id;
    return name.split(/\s+/).map(part => part[0]).join('').toUpperCase().slice(0, 2);
  };

  return (
    <Box display="flex" height="100vh" bgcolor={theme.palette.mode === 'dark' ? 'grey.900' : 'grey.50'}>
      {/* Sidebar Navigation */}
      <Paper
        elevation={0}
        sx={{
          width: 256,
          borderRight: 1,
          borderColor: 'divider',
          borderRadius: 0,
          display: 'flex',
          flexDirection: 'column'
        }}
      >
        <Box p={3}>
          <Typography variant="h6" fontWeight={700}>
            Fabric Builder
          </Typography>
        </Box>
        <Stack component="nav" spacing={1} px={2} flex={1}>
          <Button
            fullWidth
            justifyContent="flex-start"
            startIcon={<span>📊</span>}
            onClick={() => setCurrentView('dashboard')}
            variant={currentView === 'dashboard' ? 'contained' : 'text'}
            disableElevation
            sx={{
              textTransform: 'none',
              color: currentView === 'dashboard' ? 'primary.contrastText' : 'text.primary',
              bgcolor: currentView === 'dashboard' ? 'primary.main' : 'transparent',
              '&:hover': { bgcolor: currentView === 'dashboard' ? 'primary.dark' : 'action.hover' }
            }}
          >
            Dashboard
          </Button>
          <Button
            fullWidth
            justifyContent="flex-start"
            startIcon={<span>👥</span>}
            onClick={() => setCurrentView('tenants')}
            variant={currentView === 'tenants' ? 'contained' : 'text'}
            disableElevation
            sx={{
              textTransform: 'none',
              color: currentView === 'tenants' ? 'primary.contrastText' : 'text.primary',
              bgcolor: currentView === 'tenants' ? 'primary.main' : 'transparent',
              '&:hover': { bgcolor: currentView === 'tenants' ? 'primary.dark' : 'action.hover' }
            }}
          >
            Tenants
          </Button>
          <Button
            fullWidth
            justifyContent="flex-start"
            startIcon={<span>🔐</span>}
            onClick={() => setCurrentView('ip-whitelist')}
            variant={currentView === 'ip-whitelist' ? 'contained' : 'text'}
            disableElevation
            sx={{
              textTransform: 'none',
              color: currentView === 'ip-whitelist' ? 'primary.contrastText' : 'text.primary',
              bgcolor: currentView === 'ip-whitelist' ? 'primary.main' : 'transparent',
              '&:hover': { bgcolor: currentView === 'ip-whitelist' ? 'primary.dark' : 'action.hover' }
            }}
          >
            IP Whitelist
          </Button>
          <Button
            fullWidth
            justifyContent="flex-start"
            startIcon={<span>📋</span>}
            onClick={() => setCurrentView('audit-logs')}
            variant={currentView === 'audit-logs' ? 'contained' : 'text'}
            disableElevation
            sx={{
              textTransform: 'none',
              color: currentView === 'audit-logs' ? 'primary.contrastText' : 'text.primary',
              bgcolor: currentView === 'audit-logs' ? 'primary.main' : 'transparent',
              '&:hover': { bgcolor: currentView === 'audit-logs' ? 'primary.dark' : 'action.hover' }
            }}
          >
            Audit Logs
          </Button>
          <Button
            fullWidth
            justifyContent="flex-start"
            startIcon={<span>⚙️</span>}
            onClick={() => setCurrentView('settings')}
            variant={currentView === 'settings' ? 'contained' : 'text'}
            disableElevation
            sx={{
              textTransform: 'none',
              color: currentView === 'settings' ? 'primary.contrastText' : 'text.primary',
              bgcolor: currentView === 'settings' ? 'primary.main' : 'transparent',
              '&:hover': { bgcolor: currentView === 'settings' ? 'primary.dark' : 'action.hover' }
            }}
          >
            Settings
          </Button>
        </Stack>
      </Paper>

      {/* Main Content */}
      {currentView === 'ip-whitelist' && (
        <Box flex={1} display="flex" flexDirection="column" overflow="hidden">
          <>
            {/* Header Bar */}
            <Paper
              elevation={0}
              sx={{
                borderBottom: 1,
                borderColor: 'divider',
                p: 3,
                borderRadius: 0,
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}
            >
              <Box>
                <Typography variant="h5" fontWeight={700}>
                  IP Whitelist Configuration
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                  Manage IP addresses and tenant assignments
                </Typography>
              </Box>
              <Stack direction="row" spacing={0.5} alignItems="center">
                <IconButton
                  size="small"
                  onClick={() => setIPWhitelistViewMode('tile')}
                  title="Tile View"
                  color={ipWhitelistViewMode === 'tile' ? 'primary' : 'default'}
                >
                  <GridViewIcon />
                </IconButton>
                <IconButton
                  size="small"
                  onClick={() => setIPWhitelistViewMode('table')}
                  title="Table View"
                  color={ipWhitelistViewMode === 'table' ? 'primary' : 'default'}
                >
                  <ViewAgendaIcon />
                </IconButton>
                <IconButton
                  size="small"
                  onClick={(e) => setExportMenuAnchor(e.currentTarget)}
                  title="Export"
                >
                  <DownloadIcon />
                </IconButton>
                <Button 
                  variant="contained" 
                  startIcon={<AddIcon />}
                  onClick={handleAddIP}
                >
                  Add New IP
                </Button>
              </Stack>
            </Paper>

            {/* Controls Bar */}
        <Box p={2} bgcolor={theme.palette.mode === 'dark' ? 'grey.800' : 'grey.100'} borderBottom={1} borderColor="divider">
          <Stack direction="row" spacing={2} alignItems="center">
            <TextField
              size="small"
              placeholder="Search IP or Label..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setPage(0);
              }}
              InputProps={{
                startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.active' }} />
              }}
              sx={{ flex: 1, maxWidth: 300 }}
            />
            <Select
              size="small"
              value={tenantFilter}
              onChange={(e) => {
                setTenantFilter(e.target.value);
                setSelectedTenant(e.target.value);
                setPage(0);
              }}
              displayEmpty
              sx={{ minWidth: 200 }}
            >
              <MenuItem value="">All Tenants</MenuItem>
              {tenants.map(tenant => (
                <MenuItem key={tenant.id} value={tenant.id}>
                  {tenant.displayName}
                </MenuItem>
              ))}
            </Select>
            <Select
              size="small"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as any)}
              sx={{ minWidth: 180 }}
            >
              <MenuItem value="dateAdded">Date Added (Newest)</MenuItem>
              <MenuItem value="ipAddress">IP Address</MenuItem>
              <MenuItem value="label">Label</MenuItem>
            </Select>
            <IconButton
              size="small"
              onClick={(e) => setExportMenuAnchor(e.currentTarget)}
              title="Export"
            >
              <DownloadIcon />
            </IconButton>
          </Stack>
        </Box>

        {/* Table */}
        <TableContainer
          sx={{
            flex: 1,
            overflow: 'auto',
            '&::-webkit-scrollbar': {
              width: '6px'
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
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  IP ADDRESS / CIDR
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  LABEL
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  ASSIGNED TENANTS
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  STATUS
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  DATE ADDED
                </TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  ACTIONS
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {paginatedEntries.map((entry, idx) => {
                const isConflict = api.conflictDetected;
                const borderColor = isConflict ? alpha(theme.palette.warning.main, 0.5) : 'transparent';
                return (
                  <TableRow
                    key={`${entry.ipAddress}-${idx}`}
                    hover
                    sx={{
                      borderLeft: `3px solid ${borderColor}`,
                      '&:hover .action-buttons': {
                        opacity: 1
                      }
                    }}
                  >
                    <TableCell>
                      <Typography fontFamily="monospace" fontSize="0.875rem">
                        {entry.ipAddress}/32
                      </Typography>
                    </TableCell>
                    <TableCell>{entry.label || '—'}</TableCell>
                    <TableCell>
                      <Stack direction="row" spacing={0.5} flexWrap="wrap">
                        {(entry as any).allTenants ? (
                          <Chip label="All Tenants" size="small" variant="outlined" />
                        ) : (
                          entry.tenantIds.slice(0, 2).map(tenantId => (
                            <Chip
                              key={tenantId}
                              avatar={<Avatar sx={{ width: 24, height: 24 }}>{getTenantInitials(tenantId)}</Avatar>}
                              label={tenants.find(t => t.id === tenantId)?.displayName || tenantId}
                              size="small"
                            />
                          ))
                        )}
                        {entry.tenantIds.length > 2 && (
                          <Chip label={`+${entry.tenantIds.length - 2} more`} size="small" variant="outlined" />
                        )}
                      </Stack>
                    </TableCell>
                    <TableCell>
                      {isConflict ? (
                        <Chip
                          label="Conflict Detected"
                          size="small"
                          icon={<span>⚠️</span>}
                          color="warning"
                          variant="outlined"
                        />
                      ) : (
                        <Chip
                          label={(entry as any).allTenants ? 'Global' : 'Active'}
                          size="small"
                          color={((entry as any).allTenants ? 'info' : 'success')}
                        />
                      )}
                    </TableCell>
                    <TableCell>
                      {new Date(entry.createdAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={0.5} justifyContent="flex-end" className="action-buttons" sx={{ opacity: 0, transition: 'opacity 0.2s' }}>
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => setDeleteConfirm(entry)}
                          title="Delete"
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>
                    </TableCell>
                  </TableRow>
                );
              })}
              {paginatedEntries.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      {entries.length === 0 ? 'No IP addresses configured' : 'No matching results'}
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Pagination Footer */}
        <Paper
          elevation={0}
          sx={{
            borderTop: 1,
            borderColor: 'divider',
            borderRadius: 0,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            p: 2
          }}
        >
          <Typography variant="body2" color="text.secondary">
            Showing {entries.length === 0 ? 0 : page * rowsPerPage + 1} to {Math.min((page + 1) * rowsPerPage, filteredEntries.length)} of {filteredEntries.length}
          </Typography>
          <Stack direction="row" spacing={2} alignItems="center">
            <Select
              size="small"
              value={rowsPerPage}
              onChange={(e) => {
                setRowsPerPage(e.target.value as number);
                setPage(0);
              }}
              sx={{ width: 80 }}
            >
              <MenuItem value={5}>5</MenuItem>
              <MenuItem value={10}>10</MenuItem>
              <MenuItem value={25}>25</MenuItem>
              <MenuItem value={50}>50</MenuItem>
            </Select>
            <Button
              variant="outlined"
              size="small"
              disabled={page === 0}
              onClick={() => setPage(p => p - 1)}
            >
              Previous
            </Button>
            <Typography variant="body2">{page + 1}</Typography>
            <Button
              variant="outlined"
              size="small"
              disabled={(page + 1) * rowsPerPage >= filteredEntries.length}
              onClick={() => setPage(p => p + 1)}
            >
              Next
            </Button>
          </Stack>
        </Paper>
          </>
        </Box>
      )}

      {/* Dashboard View */}
      {currentView === 'dashboard' && (
        <DashboardPage />
      )}

      {/* Tenants View */}
      {currentView === 'tenants' && (
        <TenantsManagementPage />
      )}

      {/* Audit Logs View */}
      {currentView === 'audit-logs' && (
        <AuditLogsPage />
      )}

      {/* Settings View */}
      {currentView === 'settings' && (
        <SettingsPage />
      )}

      {/* Delete Confirmation Dialog */}
      <Dialog open={Boolean(deleteConfirm)} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to remove <strong>{deleteConfirm?.ipAddress}</strong>?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirm(null)}>Cancel</Button>
          <Button onClick={handleDeleteIP} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Export Menu */}
      <Menu
        anchorEl={exportMenuAnchor}
        open={Boolean(exportMenuAnchor)}
        onClose={() => setExportMenuAnchor(null)}
      >
        <MenuItem onClick={() => handleExport('csv')}>
          <DownloadIcon sx={{ mr: 1 }} />
          Export as CSV
        </MenuItem>
        <MenuItem onClick={() => handleExport('json')}>
          <DownloadIcon sx={{ mr: 1 }} />
          Export as JSON
        </MenuItem>
      </Menu>

      {/* Add IP Dialog */}
      <Dialog open={addIPDialogOpen} onClose={() => setAddIPDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add New IP Address</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <AddIPForm
            tenantId={selectedTenant}
            onSubmit={handleAddIPSubmit}
            onCancel={() => setAddIPDialogOpen(false)}
          />
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default TenantManagementView;
