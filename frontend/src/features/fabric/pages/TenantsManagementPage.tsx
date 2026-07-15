import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
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
  Grid,
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
  SupervisorAccount as _SupervisorAccountIcon,
  WorkspacePremium as WorkspacePremiumIcon,
  OpenInNew as OpenInNewIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI } from '../hooks/useIPWhitelist';
import { useNotification } from '../../../hooks/useNotification';
import { Tenant as IPTenant } from '../types/ipWhitelist';
import { useAccess } from '../../../contexts/AccessContext';
import { useOrganizationEntitlement } from '../../../contexts/useOrganizationEntitlement';
import { useRegions } from '../../../hooks/useRegions';
import { apiFetch } from '../../../lib/apiClient';
import { Tenant } from '../../../types';

const DEFAULT_REGION = 'us-east-1';

interface TenantWithUsage extends IPTenant {
  status: 'active' | 'suspended' | 'inactive';
  ipUsageActive: number;
  ipUsageTotal: number;
  lastUpdated: string;
  plan?: string;
  gold_copy?: boolean;
  is_deleted?: boolean;
  region?: string;
}

const isActiveTenant = (tenant: Pick<Tenant, 'is_active' | 'is_deleted'> & {
  isActive?: boolean;
  status?: string;
}): boolean => {
  return !tenant.is_deleted && (tenant.is_active ?? tenant.isActive ?? tenant.status === 'active') === true;
};

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
  const [downloadMenuAnchor, setDownloadMenuAnchor] = useState<null | HTMLElement>(null);
  const [viewMode, setViewMode] = useState<'tile' | 'table'>('tile');
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createCode, setCreateCode] = useState('');
  const [createRegion, setCreateRegion] = useState(DEFAULT_REGION);
  const [createSubmitting, setCreateSubmitting] = useState(false);

  const api = useIPWhitelistAPI();
  const notification = useNotification();
  const navigate = useNavigate();
  const { accessibleTenants, scope } = useAccess();
  const organization = useOrganizationEntitlement();
  const canWriteOrganization = organization.canWrite;
  const { regions } = useRegions();
  const [tenants, setTenants] = useState<TenantWithUsage[]>([]);
  const [loading, setLoading] = useState(false);

  // Block direct URL access for users without any organization entitlement.
  // They wouldn't see the menu link, but they could still type the URL.
  if (!organization.isVisible) {
    return <Navigate to="/" replace />;
  }

  useEffect(() => {
    const loadTenants = async () => {
      setLoading(true);
      try {
        // Only show tenants the current user is authorized to access.
        // When scoped to a specific tenant, show only that tenant.
        let tenantsList = accessibleTenants.filter(isActiveTenant);
        if (scope.level === 'tenant' && scope.tenantId) {
          tenantsList = tenantsList.filter(t => t.id === scope.tenantId);
        }

        // Fetch IP whitelist for each tenant to get real usage data
        const enriched: TenantWithUsage[] = await Promise.all(
          tenantsList.map(async (t: Tenant) => {
            try {
              const ips = await api.fetchTenantIPWhitelist(t.id);
              const totalIPs = ips.length;
              const activeIPs = ips.filter((ip: any) => ip.isActive !== false).length;
              const lastUpdated = ips.length > 0
                ? new Date(Math.max(...ips.map(ip => new Date(ip.createdAt || 0).getTime()))).toLocaleDateString()
                : 'Never';

              return {
                id: t.id,
                displayName: t.display_name || t.name || t.id,
                name: t.name,
                tenant_code: (t as any).tenant_code,
                status: 'active',
                ipUsageActive: activeIPs,
                ipUsageTotal: totalIPs,
                lastUpdated,
                plan: t.name || 'Standard Plan',
                gold_copy: (t as any).gold_copy ?? false,
                is_deleted: t.is_deleted ?? false,
                region: (t as any).region || DEFAULT_REGION
              };
            } catch (err) {
              // If fetch fails for this tenant, return defaults
              return {
                id: t.id,
                displayName: t.display_name || t.name || t.id,
                name: t.name,
                tenant_code: (t as any).tenant_code,
                status: 'active',
                ipUsageActive: 0,
                ipUsageTotal: 0,
                lastUpdated: 'N/A',
                plan: t.name || 'Standard Plan',
                gold_copy: (t as any).gold_copy ?? false,
                is_deleted: t.is_deleted ?? false,
                region: (t as any).region || DEFAULT_REGION
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
  }, [accessibleTenants, scope]);

  const filteredTenants = useMemo(() => {
    let filtered = tenants;

    // Only active tenants are available for this management surface.
    filtered = filtered.filter(t => !t.is_deleted && t.status === 'active');

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
    if (tenant.status !== 'active' || tenant.is_deleted) return;
    setTenantMenuAnchor(event.currentTarget);
    setSelectedTenantMenu(tenant);
  };

  const handleTenantMenuClose = () => {
    setTenantMenuAnchor(null);
    setSelectedTenantMenu(null);
  };

  const handleEditTenant = useCallback(() => {
    if (!selectedTenantMenu) return;
    if (!canWriteOrganization) {
      notification.error('You do not have permission to edit tenants');
      handleTenantMenuClose();
      return;
    }
    if (selectedTenantMenu.status !== 'active' || selectedTenantMenu.is_deleted) {
      notification.error('Inactive tenants are not available');
      handleTenantMenuClose();
      return;
    }
    setEditingTenant(selectedTenantMenu);
    setEditName(selectedTenantMenu.displayName);
    setEditDialogOpen(true);
    handleTenantMenuClose();
  }, [selectedTenantMenu, notification, canWriteOrganization]);

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
    if (deleteConfirm.status !== 'active' || deleteConfirm.is_deleted) {
      notification.error('Inactive tenants are not available');
      setDeleteConfirm(null);
      return;
    }
    
    // Defensive guard: prevent deletion of gold copy (system) tenants
    if (deleteConfirm.gold_copy) {
      notification.error('Gold Copy tenants cannot be deleted');
      setDeleteConfirm(null);
      return;
    }
    
    try {
      const success = await api.deleteTenant(deleteConfirm.id);
      if (!success) {
        notification.error('Failed to delete tenant');
        return;
      }
      setTenants(prev => prev.filter(t => t.id !== deleteConfirm.id));
      notification.success('Tenant deleted successfully');
      setDeleteConfirm(null);
    } catch (err) {
      notification.error('Failed to delete tenant');
    }
  }, [deleteConfirm, notification, api]);

  const resetCreateDialog = useCallback(() => {
    setCreateDialogOpen(false);
    setCreateName('');
    setCreateCode('');
    setCreateRegion(DEFAULT_REGION);
    setCreateSubmitting(false);
  }, []);

  const handleCreateTenant = useCallback(async () => {
    if (!canWriteOrganization) {
      notification.error('You do not have permission to create tenants');
      return;
    }
    const name = createName.trim();
    const code = createCode.trim();
    const region = createRegion.trim();
    if (!name || !code) {
      notification.error('Tenant name and code are required');
      return;
    }
    if (!region) {
      notification.error('Region is required');
      return;
    }
    setCreateSubmitting(true);
    try {
      const response = await apiFetch('/api/admin/tenants', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name,
          tenant_code: code,
          display_name: name,
          region,
        }),
      });
      if (!response.ok) {
        const errorText = await response.text();
        let message = errorText || response.statusText;
        try {
          const parsed = JSON.parse(errorText);
          message = parsed.message || parsed.error || message;
        } catch {
          // keep raw text
        }
        throw new Error(`${response.status}: ${message}`);
      }
      const created = await response.json();
      const data = created?.data ?? created;
      const newId = data?.id;
      notification.success('Tenant created successfully');
      resetCreateDialog();
      if (newId) {
        setTenants(prev => [
          ...prev,
          {
            id: newId,
            displayName: data.display_name || data.displayName || name,
            name: data.name || name,
            tenant_code: data.tenant_code || code,
            status: 'active',
            ipUsageActive: 0,
            ipUsageTotal: 0,
            lastUpdated: 'Never',
            plan: data.name || name || 'Standard Plan',
            gold_copy: data.gold_copy ?? false,
            is_deleted: false,
            region,
          },
        ]);
      }
    } catch (err: any) {
      notification.error(err?.message || 'Failed to create tenant');
      setCreateSubmitting(false);
    }
  }, [canWriteOrganization, createName, createCode, createRegion, notification, resetCreateDialog]);

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
      const rows = filteredTenants.slice(0, loadedCount).map(t => [
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
      const jsonData = filteredTenants.slice(0, loadedCount).map(t => ({
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
              onClick={(e) => {
                setDownloadMenuAnchor(e.currentTarget);
              }}
              title="Export"
            >
              <DownloadIcon />
            </IconButton>
            {canWriteOrganization && (
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                size="large"
                onClick={() => setCreateDialogOpen(true)}
              >
                Add New Tenant
              </Button>
            )}
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
            value={statusFilter === 'active' ? 'active' : 'all'}
            onChange={(e) => {
              setStatusFilter(e.target.value as any);
              setLoadedCount(10);
            }}
            startAdornment={<FilterListIcon sx={{ mr: 1, color: 'action.active' }} />}
            sx={{ minWidth: 150 }}
          >
            <MenuItem value="all">Available Tenants</MenuItem>
            <MenuItem value="active">Active</MenuItem>
          </Select>
        </Stack>

        {/* Data View — Tile (default) or Table */}
        {viewMode === 'tile' ? (
          loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
              <Typography color="text.secondary">Loading tenants...</Typography>
            </Box>
          ) : visibleTenants.length === 0 ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
              <Typography color="text.secondary">
                {tenants.length === 0 ? 'No tenants configured' : 'No matching tenants'}
              </Typography>
            </Box>
          ) : (
            <Grid container spacing={2} sx={{ mb: 2 }}>
              {visibleTenants.map((tenant) => (
                <Grid key={tenant.id} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
                  <Paper
                    elevation={0}
                    sx={{
                      p: 2,
                      border: 1,
                      borderColor: 'divider',
                      borderRadius: 1,
                      height: '100%',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 1.5,
                      transition: 'border-color 0.15s ease, transform 0.15s ease',
                      '&:hover': {
                        borderColor: 'primary.main',
                        transform: 'translateY(-1px)',
                      },
                    }}
                  >
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <Stack direction="row" spacing={1.5} alignItems="center" sx={{ minWidth: 0, flex: 1 }}>
                        <Avatar sx={{ width: 40, height: 40, fontSize: '1.25rem' }}>
                          {getTenantAvatar(tenant.displayName)}
                        </Avatar>
                        <Box sx={{ minWidth: 0 }}>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Typography variant="body2" fontWeight={600} noWrap>
                              {tenant.displayName}
                            </Typography>
                            {tenant.gold_copy && (
                              <Chip
                                icon={<WorkspacePremiumIcon />}
                                label="Gold"
                                size="small"
                                color="warning"
                                variant="outlined"
                                sx={{ height: 20, fontSize: '0.65rem', fontWeight: 600 }}
                              />
                            )}
                          </Stack>
                          <Typography variant="caption" color="text.secondary" noWrap>
                            {tenant.plan || 'Standard Plan'}
                          </Typography>
                        </Box>
                      </Stack>
                      <IconButton
                        size="small"
                        onClick={(e) => handleTenantMenuOpen(e, tenant)}
                        aria-label="Tenant actions"
                      >
                        <MoreVertIcon fontSize="small" />
                      </IconButton>
                    </Box>

                    <Box>
                      <Typography variant="caption" color="text.secondary" display="block" sx={{ fontWeight: 700 }}>
                        TENANT ID
                      </Typography>
                      <Typography variant="body2" fontFamily="monospace" noWrap>
                        {tenant.id}
                      </Typography>
                    </Box>

                    <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                      <Chip
                        label={getStatusLabel(tenant.status)}
                        color={getStatusColor(tenant.status)}
                        size="small"
                      />
                      <Chip
                        label={
                          regions.find((r) => r.value === tenant.region)?.label
                            ? regions.find((r) => r.value === tenant.region)!.label
                            : (tenant.region || DEFAULT_REGION)
                        }
                        size="small"
                        variant="outlined"
                      />
                    </Stack>

                    <Box>
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
                          mt: 0.5,
                          height: 6,
                          borderRadius: 1,
                          backgroundColor: alpha(theme.palette.primary.main, 0.1),
                          '& .MuiLinearProgress-bar': {
                            backgroundColor:
                              tenant.status === 'suspended' ? theme.palette.error.main : theme.palette.primary.main,
                            borderRadius: 1,
                          },
                        }}
                      />
                    </Box>

                    <Typography variant="caption" color="text.secondary">
                      Updated {tenant.lastUpdated}
                    </Typography>
                  </Paper>
                </Grid>
              ))}
            </Grid>
          )
        ) : (
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
                  REGION
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
                  <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">Loading tenants...</Typography>
                  </TableCell>
                </TableRow>
              ) : visibleTenants.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
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
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Typography variant="body2" fontWeight={600}>
                              {tenant.displayName}
                            </Typography>
                            {tenant.gold_copy && (
                              <Chip
                                icon={<WorkspacePremiumIcon />}
                                label="Gold Copy"
                                size="small"
                                color="warning"
                                variant="outlined"
                                sx={{
                                  height: 22,
                                  fontSize: '0.65rem',
                                  fontWeight: 600,
                                  '& .MuiChip-icon': {
                                    fontSize: 14,
                                    ml: 0.5,
                                  },
                                }}
                              />
                            )}
                          </Stack>
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
                      <Typography variant="body2" color="text.secondary">
                        {regions.find((r) => r.value === tenant.region)?.label
                          ? `${regions.find((r) => r.value === tenant.region)!.label} (${tenant.region})`
                          : (tenant.region || DEFAULT_REGION)}
                      </Typography>
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
        )}

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
        {canWriteOrganization && (
          <MenuItem onClick={handleEditTenant}>
            <EditIcon sx={{ mr: 1 }} fontSize="small" />
            Edit
          </MenuItem>
        )}
        <MenuItem onClick={() => {
          handleTenantMenuClose();
          if (selectedTenantMenu) {
            navigate(`/tenants/${selectedTenantMenu.id}`);
          }
        }}>
          <OpenInNewIcon sx={{ mr: 1 }} fontSize="small" />
          Manage Resources
        </MenuItem>
        {canWriteOrganization && !selectedTenantMenu?.gold_copy && (
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
        )}
        {canWriteOrganization && selectedTenantMenu?.gold_copy && (
          <MenuItem disabled sx={{ color: 'text.disabled' }}>
            <DeleteIcon sx={{ mr: 1 }} fontSize="small" />
            Gold Copy (cannot delete)
          </MenuItem>
        )}
      </Menu>

      {/* Download Menu */}
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

      {/* Create Tenant Dialog */}
      <Dialog
        open={createDialogOpen}
        onClose={createSubmitting ? undefined : resetCreateDialog}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Add New Tenant</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Stack spacing={2}>
            <TextField
              fullWidth
              label="Tenant Name"
              value={createName}
              onChange={(e) => setCreateName(e.target.value)}
              variant="outlined"
              size="small"
              required
              autoFocus
            />
            <TextField
              fullWidth
              label="Tenant Code"
              value={createCode}
              onChange={(e) => setCreateCode(e.target.value)}
              variant="outlined"
              size="small"
              required
              helperText="Short identifier (e.g. acme-prod)"
            />
            <Select
              fullWidth
              size="small"
              value={createRegion}
              onChange={(e) => setCreateRegion(String(e.target.value))}
              required
              displayEmpty
              inputProps={{ 'aria-label': 'Region' }}
            >
              {regions.map((r) => (
                <MenuItem key={r.value} value={r.value}>
                  {r.label} ({r.value})
                </MenuItem>
              ))}
            </Select>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={resetCreateDialog} disabled={createSubmitting}>
            Cancel
          </Button>
          <Button
            onClick={handleCreateTenant}
            variant="contained"
            disabled={createSubmitting || !createName.trim() || !createCode.trim() || !createRegion.trim()}
          >
            {createSubmitting ? 'Creating...' : 'Create Tenant'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TenantsManagementPage;
