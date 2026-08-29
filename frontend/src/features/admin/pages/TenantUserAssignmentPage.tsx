import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  InputAdornment,
  IconButton,
  Tooltip,
  CircularProgress,
  Alert,
  Avatar,
  Stack,
  useTheme,
  Autocomplete
} from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  Business as BusinessIcon,
  Add as AddIcon,
  Edit as EditIcon
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';
import { useAccess } from '../../../contexts/AccessContext';

interface Tenant {
  id: string;
  name: string;
  display_name: string;
  is_active?: boolean;
  isActive?: boolean;
  status?: string;
  is_deleted?: boolean;
}

interface User {
  id: string;
  username: string;
  email: string | null;
  name: string | null;
  first_name: string | null;
  last_name: string | null;
}

interface TenantAccessMapping {
  id: string;
  user_id: string;
  tenant_id: string;
  access_role: string;
  email: string;
  created_at: string;
  updated_at: string;
  tenant_name?: string;
  tenant_is_active?: boolean;
  tenant_is_deleted?: boolean;
}

const isTenantAvailable = (tenant: Tenant): boolean =>
  !tenant.is_deleted && (tenant.is_active ?? tenant.isActive ?? tenant.status === 'active') === true;

const TenantUserAssignmentPage: React.FC = () => {
  const theme = useTheme();
  const { isPlatformOperator, accessLevel, accessibleTenants, currentTenant } = useAccess();
  const isTenantManager = isPlatformOperator || accessLevel === 'tenant_admin';

  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [mappings, setMappings] = useState<TenantAccessMapping[]>([]);
  const [users, setUsers] = useState<User[]>([]);

  // Dialog state
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  // Selected items state
  const [selectedMapping, setSelectedMapping] = useState<TenantAccessMapping | null>(null);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedTenantId, setSelectedTenantId] = useState<string>(currentTenant?.id || accessibleTenants[0]?.id || '');
  const [selectedRole, setSelectedRole] = useState<string>('viewer');

  // Fetch access mappings
  const fetchData = async () => {
    try {
      setLoading(true);
      setErrorMsg(null);
      const res = await apiClient<any>('/api/admin/tenant-access');
      const list = Array.isArray(res) ? res : Array.isArray(res?.data) ? res.data : [];
      setMappings(list);
    } catch (err: any) {
      console.error('Failed to fetch tenant access mappings:', err);
      setErrorMsg(err.message || 'Failed to fetch tenant access mappings.');
      setMappings([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch users
  const fetchUsers = async () => {
    try {
      const data = await apiClient<User[]>('/api/rbac/users');
      setUsers(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error('Failed to fetch users:', err);
      setUsers([]);
    }
  };

  // Sync selectedTenantId when currentTenant or accessibleTenants changes
  useEffect(() => {
    if (currentTenant?.id) {
      setSelectedTenantId(currentTenant.id);
    } else if (accessibleTenants.length > 0 && !selectedTenantId) {
      setSelectedTenantId(accessibleTenants[0].id);
    }
  }, [currentTenant?.id, accessibleTenants]);

  // Fetch data on mount
  useEffect(() => {
    fetchData();
    fetchUsers();
  }, []);

  // Map tenant details by ID
  const tenantMap = useMemo(() => {
    const map = new Map<string, Tenant>();
    accessibleTenants.forEach(t => map.set(t.id, t));
    return map;
  }, [accessibleTenants]);

  // Filtered mappings list - always scoped to current tenant
  const filteredMappings = useMemo(() => {
    const effectiveTenantFilter = currentTenant?.id || accessibleTenants[0]?.id || 'all';
    return mappings.filter(m => {
      const matchesSearch =
        m.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
        m.user_id.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesTenant =
        effectiveTenantFilter === 'all' ||
        m.tenant_id === effectiveTenantFilter;

      return matchesSearch && matchesTenant;
    });
  }, [mappings, searchTerm, currentTenant, accessibleTenants]);

  // Grant access (Create mapping)
  const handleCreateAccess = async () => {
    if (!selectedUser || !selectedTenantId || !selectedRole) {
      setErrorMsg('Please select a user, tenant, and role.');
      return;
    }
    if (!tenantMap.has(selectedTenantId)) {
      setErrorMsg('That tenant is not active or is no longer available.');
      return;
    }
    setActionLoading(true);
    try {
      setErrorMsg(null);
      await apiClient('/api/admin/tenant-access', {
        method: 'POST',
        body: JSON.stringify({
          user_id: selectedUser.id,
          tenant_id: selectedTenantId,
          access_role: selectedRole
        })
      });
      
      setSuccessMsg(`Successfully granted access to tenant.`);
      setCreateDialogOpen(false);
      setSelectedUser(null);
      setSelectedTenantId('');
      setSelectedRole('viewer');
      fetchData();
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to create tenant access mapping.');
    } finally {
      setActionLoading(false);
    }
  };

  // Update role mapping
  const handleUpdateRole = async () => {
    if (!selectedMapping || !selectedRole) return;
    if (!tenantMap.has(selectedMapping.tenant_id)) {
      setErrorMsg('That tenant is not active or is no longer available.');
      return;
    }
    setActionLoading(true);
    try {
      setErrorMsg(null);
      await apiClient(`/api/admin/tenant-access/${selectedMapping.user_id}/${selectedMapping.tenant_id}`, {
        method: 'PUT',
        body: JSON.stringify({ access_role: selectedRole })
      });
      
      setSuccessMsg(`Successfully updated role mapping.`);
      setEditDialogOpen(false);
      setSelectedMapping(null);
      setSelectedRole('viewer');
      fetchData();
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to update access role.');
    } finally {
      setActionLoading(false);
    }
  };

  // Revoke access mapping
  const handleRevokeAccess = async (mapping: TenantAccessMapping) => {
    if (!tenantMap.has(mapping.tenant_id)) {
      setErrorMsg('That tenant is not active or is no longer available.');
      return;
    }
    if (!window.confirm(`Are you sure you want to revoke tenant access for ${mapping.email}?`)) {
      return;
    }
    setActionLoading(true);
    try {
      setErrorMsg(null);
      await apiClient(`/api/admin/tenant-access/${mapping.user_id}/${mapping.tenant_id}`, {
        method: 'DELETE'
      });
      
      setSuccessMsg(`Successfully revoked tenant access mapping.`);
      fetchData();
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to revoke tenant access.');
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="calc(100vh - 120px)">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 4, display: 'flex', flexDirection: 'column', gap: 3 }}>
      {/* Page Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center">
        <Box>
          <Typography variant="h4" fontWeight={900} gutterBottom>
            Tenant Access Controls
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Map specific tenant access scopes and roles to users to control data isolation.
          </Typography>
        </Box>
        <Stack direction="row" spacing={2}>
          <Button 
            variant="outlined" 
            startIcon={<RefreshIcon />} 
            onClick={fetchData}
          >
            Refresh
          </Button>
          <Button 
            variant="contained" 
            startIcon={<AddIcon />} 
            onClick={() => setCreateDialogOpen(true)}
          >
            Assign Tenant Access
          </Button>
        </Stack>
      </Box>

      {/* Alerts */}
      {errorMsg && <Alert severity="error">{errorMsg}</Alert>}
      {successMsg && <Alert severity="success">{successMsg}</Alert>}

      {/* Filter and Search Bar */}
      <Paper elevation={0} sx={{ p: 3, border: '1px solid', borderColor: 'divider', borderRadius: 2 }}>
        <TextField
          size="small"
          placeholder="Search by email or user ID..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: 'text.secondary' }} />
              </InputAdornment>
            )
          }}
          sx={{ width: { xs: '100%', sm: 400 } }}
        />
      </Paper>

      {/* Access Mappings Table */}
      <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(0, 0, 0, 0.2)' : 'grey.50' }}>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>User Email / ID</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Assigned Tenant</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Access Role</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Created At</TableCell>
              <TableCell align="right" sx={{ fontWeight: 'bold', py: 1.5 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredMappings.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                  <Typography variant="body1" color="text.secondary">
                    No access mappings found.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredMappings.map((m) => {
                const tenant = tenantMap.get(m.tenant_id);
                const tenantLabel = m.tenant_name || (tenant ? (tenant.display_name || tenant.name) : m.tenant_id);
                const tenantIsInactive = m.tenant_is_active === false || m.tenant_is_deleted === true;
                const tenantTooltip = m.tenant_is_deleted
                  ? 'Tenant has been soft-deleted'
                  : m.tenant_is_active === false
                  ? 'Tenant is inactive'
                  : '';
                return (
                  <TableRow key={m.id} hover>
                    <TableCell sx={{ py: 1.5 }}>
                      <Stack direction="row" spacing={1.5} alignItems="center">
                        <Avatar sx={{ width: 32, height: 32, bgcolor: theme.palette.primary.main }}>
                          {m.email[0].toUpperCase()}
                        </Avatar>
                        <Box>
                          <Typography variant="body2" fontWeight={600}>
                            {m.email}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" fontFamily="monospace">
                            {m.user_id}
                          </Typography>
                        </Box>
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Tooltip title={tenantTooltip}>
                        <span>
                          <Chip
                            icon={<BusinessIcon />}
                            label={tenantLabel}
                            color={tenantIsInactive ? 'default' : 'primary'}
                            variant="outlined"
                            size="small"
                          />
                        </span>
                      </Tooltip>
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Chip
                        label={m.access_role}
                        color={m.access_role === 'admin' ? 'error' : m.access_role === 'editor' ? 'warning' : 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Typography variant="body2" color="text.secondary">
                        {new Date(m.created_at).toLocaleString()}
                      </Typography>
                    </TableCell>
                    <TableCell align="right" sx={{ py: 1.5 }}>
                      <Stack direction="row" spacing={1} justifyContent="flex-end">
                        <Tooltip title="Edit Role">
                          <IconButton
                            size="small"
                            color="primary"
                            onClick={() => {
                              setSelectedMapping(m);
                              setSelectedRole(m.access_role);
                              setEditDialogOpen(true);
                            }}
                          >
                            <EditIcon />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Revoke Tenant Access">
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => handleRevokeAccess(m)}
                            disabled={actionLoading}
                          >
                            <DeleteIcon />
                          </IconButton>
                        </Tooltip>
                      </Stack>
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Grant Access Dialog */}
      <Dialog 
        open={createDialogOpen} 
        onClose={() => setCreateDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ fontWeight: 'bold' }}>Assign Tenant Access</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* User Selection autocomplete */}
            <Autocomplete
              options={users}
              getOptionLabel={(u) => `${u.username} (${u.email || 'No email'})`}
              value={selectedUser}
              onChange={(_, newValue) => setSelectedUser(newValue)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Select User"
                  placeholder="Search user..."
                  size="small"
                />
              )}
              fullWidth
            />

            {/* Tenant Selection (Read-only / Defaulted to scoped tenant) */}
            <FormControl fullWidth size="small" disabled={Boolean(currentTenant?.id) || (!isPlatformOperator && accessibleTenants.length <= 1)}>
              <InputLabel>Tenant</InputLabel>
              <Select
                value={selectedTenantId}
                label="Tenant"
                onChange={(e) => setSelectedTenantId(e.target.value)}
              >
              {accessibleTenants.map((t) => (
                  <MenuItem key={t.id} value={t.id}>
                    {t.display_name || t.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {/* Role Selection */}
            <FormControl fullWidth size="small">
              <InputLabel>Access Role</InputLabel>
              <Select
                value={selectedRole}
                label="Access Role"
                onChange={(e) => setSelectedRole(e.target.value)}
              >
                <MenuItem value="viewer">Viewer</MenuItem>
                <MenuItem value="editor">Editor</MenuItem>
                <MenuItem value="admin">Admin</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button 
            variant="contained" 
            onClick={handleCreateAccess} 
            disabled={actionLoading || !selectedUser || !selectedTenantId}
          >
            Assign
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Role Dialog */}
      <Dialog 
        open={editDialogOpen} 
        onClose={() => setEditDialogOpen(false)}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle sx={{ fontWeight: 'bold' }}>Edit Access Role</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="body2" color="text.secondary">
              Update role permission mapping for <strong>{selectedMapping?.email}</strong>.
            </Typography>
            <FormControl fullWidth size="small">
              <InputLabel>Access Role</InputLabel>
              <Select
                value={selectedRole}
                label="Access Role"
                onChange={(e) => setSelectedRole(e.target.value)}
              >
                <MenuItem value="viewer">Viewer</MenuItem>
                <MenuItem value="editor">Editor</MenuItem>
                <MenuItem value="admin">Admin</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setEditDialogOpen(false)}>Cancel</Button>
          <Button 
            variant="contained" 
            onClick={handleUpdateRole} 
            disabled={actionLoading}
          >
            Update
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TenantUserAssignmentPage;
