import React, { useState, useMemo, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  TextField,
  Select,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
  Chip,
  Avatar,
  Stack,
  IconButton,
  Menu,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  LinearProgress,
  useTheme,
  alpha,
  CircularProgress,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterListIcon,
  MoreVert as MoreVertIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Download as DownloadIcon,
  GridView as GridViewIcon,
  ViewAgenda as ViewAgendaIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI } from '../hooks/useIPWhitelist';
import { useNotification } from '../../../hooks/useNotification';
import { Tenant } from '../types/ipWhitelist';

interface TenantWithUsage extends Tenant {
  status: 'active' | 'suspended' | 'inactive';
  ipUsageActive: number;
  ipUsageTotal: number;
  lastUpdated: string;
  plan?: string;
}

const TenantsManagementPage: React.FC = () => {
  const theme = useTheme();
  const [loadedCount, setLoadedCount] = useState(10);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'suspended' | 'inactive'>('all');
  const [tenantMenuAnchor, setTenantMenuAnchor] = useState<null | HTMLElement>(null);
  const [selectedTenantMenu, setSelectedTenantMenu] = useState<TenantWithUsage | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<TenantWithUsage | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<TenantWithUsage | null>(null);
  const [editName, setEditName] = useState('');
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [selectedTenantDetails, setSelectedTenantDetails] = useState<TenantWithUsage | null>(null);
  const [downloadMenuAnchor, setDownloadMenuAnchor] = useState<null | HTMLElement>(null);
  const [viewMode, setViewMode] = useState<'tile' | 'table'>('tile');

  const api = useIPWhitelistAPI();
  const notification = useNotification();
  const [tenants, setTenants] = useState<TenantWithUsage[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const loadTenants = async () => {
      setLoading(true);
      try {
        const tenantsList = await api.fetchTenants();
        
        // Fetch IP whitelist for each tenant to get real usage data
        const enriched: TenantWithUsage[] = await Promise.all(
          tenantsList.map(async (t) => {
            try {
              const ips = await api.fetchTenantIPWhitelist(t.id);
              const totalIPs = ips.length;
              const activeIPs = ips.filter((ip: any) => ip.isActive !== false).length;
              const lastUpdated = ips.length > 0
                ? new Date(Math.max(...ips.map(ip => new Date(ip.createdAt || 0).getTime()))).toLocaleDateString()
                : 'Never';
              
              return {
                ...t,
                status: totalIPs === 0 ? 'inactive' : 'active' as any,
                ipUsageActive: activeIPs,
                ipUsageTotal: totalIPs,
                lastUpdated,
                plan: t.name || 'Standard Plan'
              };
            } catch (err) {
              // If fetch fails for this tenant, return defaults
              return {
                ...t,
                status: 'active' as any,
                ipUsageActive: 0,
                ipUsageTotal: 0,
                lastUpdated: 'N/A',
                plan: t.name || 'Standard Plan'
              };
            }
          })
        );
        setTenants(enriched);
      } catch (err) {
        notification.error('Failed to load tenants');
      } finally {
        setLoading(false);
      }
    };
    loadTenants();
  }, []);

  const filteredTenants = useMemo(() => {
    let filtered = tenants;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(t =>
        t.displayName.toLowerCase().includes(query) ||
        t.id.toLowerCase().includes(query)
      );
    }

    if (statusFilter !== 'all') {
      filtered = filtered.filter(t => t.status === statusFilter);
    }

    return filtered;
  }, [tenants, searchQuery, statusFilter]);

  const visibleTenants = useMemo(() => {
    return filteredTenants.slice(0, loadedCount);
  }, [filteredTenants, loadedCount]);

  const hasMore = useMemo(() => {
    return loadedCount < filteredTenants.length;
  }, [loadedCount, filteredTenants.length]);

  const loadMore = useCallback(() => {
    setIsLoadingMore(true);
    setTimeout(() => {
      setLoadedCount(prev => prev + 10);
      setIsLoadingMore(false);
    }, 300);
  }, []);

  const handleTenantMenuOpen = (event: React.MouseEvent<HTMLElement>, tenant: TenantWithUsage) => {
    setTenantMenuAnchor(event.currentTarget);
    setSelectedTenantMenu(tenant);
  };

  const handleTenantMenuClose = () => {
    setTenantMenuAnchor(null);
    setSelectedTenantMenu(null);
  };

  const handleEditTenant = useCallback(() => {
    if (!selectedTenantMenu) return;
    setEditingTenant(selectedTenantMenu);
    setEditName(selectedTenantMenu.displayName);
    setEditDialogOpen(true);
    handleTenantMenuClose();
  }, [selectedTenantMenu]);

  const handleSaveEdit = useCallback(async () => {
    if (!editingTenant || !editName.trim()) return;
    try {
      // TODO: Call API to update tenant
      setTenants(prev =>
        prev.map(t =>
          t.id === editingTenant.id
            ? { ...t, displayName: editName }
            : t
        )
      );
      notification.success('Tenant updated successfully');
      setEditDialogOpen(false);
      setEditingTenant(null);
      setEditName('');
    } catch (err) {
      notification.error('Failed to update tenant');
    }
  }, [editingTenant, editName, notification]);

  const handleDeleteTenant = useCallback(async () => {
    if (!deleteConfirm) return;
    try {
      // TODO: Call API to delete tenant
      setTenants(prev => prev.filter(t => t.id !== deleteConfirm.id));
      notification.success('Tenant deleted successfully');
      setDeleteConfirm(null);
    } catch (err) {
      notification.error('Failed to delete tenant');
    }
  }, [deleteConfirm, notification]);

  const handleViewDetails = useCallback(() => {
    if (!selectedTenantMenu) return;
    setSelectedTenantDetails(selectedTenantMenu);
    setDetailsDialogOpen(true);
    handleTenantMenuClose();
  }, [selectedTenantMenu]);

  const handleExportTenant = useCallback((format: 'csv' | 'json') => {
    if (!selectedTenantDetails) return;
    
    const data = {
      tenantName: selectedTenantDetails.displayName,
      tenantId: selectedTenantDetails.id,
      status: selectedTenantDetails.status,
      plan: selectedTenantDetails.plan,
      ipUsageActive: selectedTenantDetails.ipUsageActive,
      ipUsageTotal: selectedTenantDetails.ipUsageTotal,
      lastUpdated: selectedTenantDetails.lastUpdated,
      exportedAt: new Date().toISOString()
    };

    if (format === 'csv') {
      const headers = Object.keys(data).join(',');
      const values = Object.values(data).join(',');
      const csv = `${headers}\n${values}`;
      const blob = new Blob([csv], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `tenant-${selectedTenantDetails.id}-${new Date().toISOString().split('T')[0]}.csv`;
      link.click();
      window.URL.revokeObjectURL(url);
    } else {
      const json = JSON.stringify(data, null, 2);
      const blob = new Blob([json], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `tenant-${selectedTenantDetails.id}-${new Date().toISOString().split('T')[0]}.json`;
      link.click();
      window.URL.revokeObjectURL(url);
    }
    setDownloadMenuAnchor(null);
  }, [selectedTenantDetails]);

  const getStatusColor = (status: string): 'success' | 'error' | 'default' => {
    switch (status) {
      case 'active':
        return 'success';
      case 'suspended':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusLabel = (status: string): string => {
    return status.charAt(0).toUpperCase() + status.slice(1);
  };

  const getUsagePercentage = (active: number, total: number): number => {
    return total === 0 ? 0 : (active / total) * 100;
  };

  const _getTenantInitials = (displayName: string): string => {
    return displayName
      .split(/\s+/)
      .map(part => part[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const getTenantAvatar = (displayName: string): string => {
    // Map some known names to emojis for visual interest (matching mockup)
    const emojiMap: Record<string, string> = {
      'Acme': '🏢',
      'Globex': '🌐',
      'Soylent': '🧬',
      'Initech': '💾',
      'Umbrella': '🌂'
    };
    const key = Object.keys(emojiMap).find(k => displayName.includes(k));
    return key ? emojiMap[key] : '🏢';
  };

  const handleExportTenants = (format: 'csv' | 'json') => {
    if (format === 'csv') {
      const headers = ['ID', 'Name', 'IPs Used', 'IPs Total', 'Status'];
      const rows = tenants.slice(0, loadedCount).map(t => [
        t.id,
        t.displayName,
        t.ipUsageActive,
        t.ipUsageTotal,
        t.status
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
      const jsonData = tenants.slice(0, loadedCount).map(t => ({
        id: t.id,
        name: t.displayName,
        ipUsageActive: t.ipUsageActive,
        ipUsageTotal: t.ipUsageTotal,
        status: t.status
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
    setDownloadMenuAnchor(null);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Paper elevation={0} sx={{ p: 3 }}>
        {/* Header Section */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', mb: 4, gap: 2, flexWrap: 'wrap' }}>
          <Box>
            <Typography variant="h4" fontWeight={900} gutterBottom>
              Tenant Management
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Manage tenant access, configure IP whitelists, and monitor usage.
            </Typography>
          </Box>
          <Stack direction="row" spacing={0.5}>
            <IconButton
              size="small"
              onClick={() => setViewMode('tile')}
              title="Tile View"
              color={viewMode === 'tile' ? 'primary' : 'default'}
            >
              <GridViewIcon />
            </IconButton>
            <IconButton
              size="small"
              onClick={() => setViewMode('table')}
              title="Table View"
              color={viewMode === 'table' ? 'primary' : 'default'}
            >
              <ViewAgendaIcon />
            </IconButton>
            <IconButton
              size="small"
              onClick={(e) => setDownloadMenuAnchor(e.currentTarget)}
              title="Export"
            >
              <DownloadIcon />
            </IconButton>
            <Button variant="contained" startIcon={<AddIcon />} size="large">
              Add New Tenant
            </Button>
          </Stack>
        </Box>

        {/* Filters & Search Toolbar */}
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mb: 3 }}>
          {/* Search */}
          <TextField
            size="small"
            placeholder="Search tenants by name or ID..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setLoadedCount(10);
            }}
            InputProps={{
              startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.active' }} />
            }}
            sx={{ flex: 1, maxWidth: { xs: '100%', md: 350 } }}
          />

          {/* Status Filter */}
          <Select
            size="small"
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value as any);
              setLoadedCount(10);
            }}
            startAdornment={<FilterListIcon sx={{ mr: 1, color: 'action.active' }} />}
            sx={{ minWidth: 150 }}
          >
            <MenuItem value="all">All Statuses</MenuItem>
            <MenuItem value="active">Active</MenuItem>
            <MenuItem value="suspended">Suspended</MenuItem>
            <MenuItem value="inactive">Inactive</MenuItem>
          </Select>
        </Stack>

        {/* Data Table */}
        <TableContainer
          sx={{
            borderRadius: 1,
            border: 1,
            borderColor: 'divider',
            mb: 2,
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
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(0, 0, 0, 0.3)' : 'grey.50' }}>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  TENANT NAME
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  TENANT ID
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  STATUS
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  IP USAGE (ACTIVE/TOTAL)
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  LAST UPDATED
                </TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  ACTIONS
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">Loading tenants...</Typography>
                  </TableCell>
                </TableRow>
              ) : visibleTenants.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      {tenants.length === 0 ? 'No tenants configured' : 'No matching tenants'}
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                visibleTenants.map((tenant) => (
                  <TableRow
                    key={tenant.id}
                    hover
                    sx={{ '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.05) } }}
                  >
                    <TableCell>
                      <Stack direction="row" spacing={2} alignItems="center">
                        <Avatar sx={{ width: 40, height: 40, fontSize: '1.25rem' }}>
                          {getTenantAvatar(tenant.displayName)}
                        </Avatar>
                        <Box>
                          <Typography variant="body2" fontWeight={600}>
                            {tenant.displayName}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {tenant.plan || 'Standard Plan'}
                          </Typography>
                        </Box>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontFamily="monospace" color="text.secondary">
                        {tenant.id}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={getStatusLabel(tenant.status)}
                        color={getStatusColor(tenant.status)}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Stack spacing={1}>
                        <Stack direction="row" justifyContent="space-between" spacing={1}>
                          <Typography variant="caption" fontWeight={600}>
                            {tenant.ipUsageActive} Active
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {tenant.ipUsageTotal} Total
                          </Typography>
                        </Stack>
                        <LinearProgress
                          variant="determinate"
                          value={getUsagePercentage(tenant.ipUsageActive, tenant.ipUsageTotal)}
                          sx={{
                            height: 6,
                            borderRadius: 1,
                            backgroundColor: alpha(theme.palette.primary.main, 0.1),
                            '& .MuiLinearProgress-bar': {
                              backgroundColor: tenant.status === 'suspended' ? theme.palette.error.main : theme.palette.primary.main,
                              borderRadius: 1
                            }
                          }}
                        />
                      </Stack>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {tenant.lastUpdated}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <IconButton
                        size="small"
                        onClick={(e) => handleTenantMenuOpen(e, tenant)}
                      >
                        <MoreVertIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Load More Button */}
        {hasMore && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <Button
              variant="outlined"
              onClick={loadMore}
              disabled={isLoadingMore}
              startIcon={isLoadingMore ? <CircularProgress size={20} /> : undefined}
            >
              {isLoadingMore ? 'Loading...' : `Load More (${visibleTenants.length} of ${filteredTenants.length})`}
            </Button>
          </Box>
        )}

        {/* Results Summary */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderTop: 1, borderColor: 'divider', pt: 2 }}>
          <Typography variant="body2" color="text.secondary">
            {filteredTenants.length === 0
              ? 'No results'
              : `Showing ${visibleTenants.length} of ${filteredTenants.length} tenants`}
          </Typography>
        </Box>
      </Paper>

      {/* Tenant Actions Menu */}
      <Menu
        anchorEl={tenantMenuAnchor}
        open={Boolean(tenantMenuAnchor)}
        onClose={handleTenantMenuClose}
      >
        <MenuItem onClick={handleEditTenant}>
          <EditIcon sx={{ mr: 1 }} fontSize="small" />
          Edit
        </MenuItem>
        <MenuItem onClick={handleViewDetails}>
          View Details
        </MenuItem>
        <MenuItem
          onClick={() => {
            handleTenantMenuClose();
            if (selectedTenantMenu) {
              setDeleteConfirm(selectedTenantMenu);
            }
          }}
          sx={{ color: 'error.main' }}
        >
          <DeleteIcon sx={{ mr: 1 }} fontSize="small" />
          Delete
        </MenuItem>
      </Menu>

      {/* View Details Dialog */}
      <Dialog open={detailsDialogOpen} onClose={() => setDetailsDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>{selectedTenantDetails?.displayName} - Details</span>
          <IconButton
            size="small"
            onClick={(e) => {
              setDownloadMenuAnchor(e.currentTarget);
            }}
            sx={{ ml: 2 }}
          >
            📥
          </IconButton>
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Stack spacing={3}>
            {selectedTenantDetails && (
              <>
                <Box>
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5, fontWeight: 700 }}>
                    TENANT ID
                  </Typography>
                  <Typography variant="body2" fontFamily="monospace">
                    {selectedTenantDetails.id}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5, fontWeight: 700 }}>
                    DISPLAY NAME
                  </Typography>
                  <Typography variant="body2">
                    {selectedTenantDetails.displayName}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5, fontWeight: 700 }}>
                    PLAN
                  </Typography>
                  <Typography variant="body2">
                    {selectedTenantDetails.plan}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5, fontWeight: 700 }}>
                    STATUS
                  </Typography>
                  <Chip
                    label={getStatusLabel(selectedTenantDetails.status)}
                    color={getStatusColor(selectedTenantDetails.status)}
                    size="small"
                  />
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1, fontWeight: 700 }}>
                    IP USAGE ({selectedTenantDetails.ipUsageActive}/{selectedTenantDetails.ipUsageTotal})
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={getUsagePercentage(selectedTenantDetails.ipUsageActive, selectedTenantDetails.ipUsageTotal)}
                    sx={{
                      height: 8,
                      borderRadius: 1,
                      backgroundColor: alpha(theme.palette.primary.main, 0.1),
                      '& .MuiLinearProgress-bar': {
                        backgroundColor: selectedTenantDetails.status === 'suspended' ? theme.palette.error.main : theme.palette.primary.main,
                        borderRadius: 1
                      }
                    }}
                  />
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5, fontWeight: 700 }}>
                    LAST UPDATED
                  </Typography>
                  <Typography variant="body2">
                    {selectedTenantDetails.lastUpdated}
                  </Typography>
                </Box>
              </>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Download Menu */}
      <Menu
        anchorEl={downloadMenuAnchor}
        open={Boolean(downloadMenuAnchor)}
        onClose={() => setDownloadMenuAnchor(null)}
      >
        <MenuItem onClick={() => handleExportTenant('csv')}>
          📊 Export as CSV
        </MenuItem>
        <MenuItem onClick={() => handleExportTenant('json')}>
          📄 Export as JSON
        </MenuItem>
      </Menu>

      {/* Export Menu */}
      <Menu
        anchorEl={downloadMenuAnchor}
        open={Boolean(downloadMenuAnchor)}
        onClose={() => setDownloadMenuAnchor(null)}
      >
        <MenuItem onClick={() => handleExportTenants('csv')}>
          📊 Export as CSV
        </MenuItem>
        <MenuItem onClick={() => handleExportTenants('json')}>
          📄 Export as JSON
        </MenuItem>
      </Menu>

      {/* Edit Tenant Dialog */}
      <Dialog open={editDialogOpen} onClose={() => setEditDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Tenant</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Stack spacing={2}>
            <TextField
              fullWidth
              label="Tenant Name"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              variant="outlined"
              size="small"
            />
            {editingTenant && (
              <Box>
                <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 1 }}>
                  Tenant ID: {editingTenant.id}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Plan: {editingTenant.plan}
                </Typography>
              </Box>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleSaveEdit} variant="contained" disabled={!editName.trim()}>
            Save Changes
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={Boolean(deleteConfirm)} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Delete Tenant</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete <strong>{deleteConfirm?.displayName}</strong>? This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirm(null)}>Cancel</Button>
          <Button onClick={handleDeleteTenant} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TenantsManagementPage;
