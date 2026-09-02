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
  Stack,
  useTheme,
} from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  Add as AddIcon,
  Groups as GroupsIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';
import { useAccess } from '../../../contexts/AccessContext';

interface GroupRoleMapping {
  id: string;
  tenant_id: string;
  tenant_name: string;
  idp_group_claim: string;
  role_key: string;
  role_name: string | null;
  created_at: string;
}

interface Role {
  id: string;
  role_key: string;
  role_name: string;
}

// Admin screen for security.idp_group_role_mappings: which IdP group claim
// (e.g. a Keycloak group path) grants which bp_roles.role_key, per tenant.
// The same group claim string can map to different role_keys in different
// tenants, which is how one identity (e.g. a support/professional-services
// user) ends up read-only in some tenants and full CRUD in others, purely
// from IdP group membership — see security.GroupRoleResolver on the backend.
const GroupRoleMappingPage: React.FC = () => {
  const theme = useTheme();
  const { accessibleTenants, currentTenant, isPlatformOperator } = useAccess();

  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [mappings, setMappings] = useState<GroupRoleMapping[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  const [groupClaim, setGroupClaim] = useState('');
  const [selectedRoleKey, setSelectedRoleKey] = useState('');
  const [selectedTenantId, setSelectedTenantId] = useState<string>(currentTenant?.id || accessibleTenants[0]?.id || '');

  const fetchData = async () => {
    try {
      setLoading(true);
      setErrorMsg(null);
      const [mappingRes, roleRes] = await Promise.all([
        apiClient<GroupRoleMapping[]>('/api/rbac/group-role-mappings'),
        apiClient<Role[]>('/api/rbac/roles'),
      ]);
      setMappings(Array.isArray(mappingRes) ? mappingRes : []);
      setRoles(Array.isArray(roleRes) ? roleRes : []);
    } catch (err: any) {
      console.error('Failed to fetch group role mappings:', err);
      setErrorMsg(err.message || 'Failed to fetch group role mappings.');
      setMappings([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    if (currentTenant?.id) {
      setSelectedTenantId(currentTenant.id);
    } else if (accessibleTenants.length > 0 && !selectedTenantId) {
      setSelectedTenantId(accessibleTenants[0].id);
    }
  }, [currentTenant?.id, accessibleTenants]);

  const filteredMappings = useMemo(() => {
    const q = searchTerm.toLowerCase();
    return mappings.filter(
      (m) =>
        m.idp_group_claim.toLowerCase().includes(q) ||
        m.role_key.toLowerCase().includes(q) ||
        m.tenant_name.toLowerCase().includes(q)
    );
  }, [mappings, searchTerm]);

  const handleCreate = async () => {
    if (!groupClaim.trim() || !selectedRoleKey || !selectedTenantId) {
      setErrorMsg('Please provide a tenant, an IdP group claim, and a role.');
      return;
    }
    setActionLoading(true);
    try {
      setErrorMsg(null);
      await apiClient('/api/rbac/group-role-mappings', {
        method: 'POST',
        body: JSON.stringify({
          tenant_id: selectedTenantId,
          idp_group_claim: groupClaim.trim(),
          role_key: selectedRoleKey,
        }),
      });
      setSuccessMsg('Group role mapping created.');
      setCreateDialogOpen(false);
      setGroupClaim('');
      setSelectedRoleKey('');
      fetchData();
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to create group role mapping.');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDelete = async (mapping: GroupRoleMapping) => {
    if (!window.confirm(`Remove mapping "${mapping.idp_group_claim}" -> "${mapping.role_key}" for ${mapping.tenant_name}?`)) {
      return;
    }
    setActionLoading(true);
    try {
      await apiClient(`/api/rbac/group-role-mappings/${mapping.id}`, { method: 'DELETE' });
      setSuccessMsg('Mapping removed.');
      fetchData();
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to remove mapping.');
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
      <Box display="flex" justifyContent="space-between" alignItems="center">
        <Box>
          <Typography variant="h4" fontWeight={900} gutterBottom>
            Group &rarr; Role Mappings
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Map an identity provider group claim to an internal role, per tenant. The same group can grant
            different access in different tenants — e.g. read-only in one, full access in another.
          </Typography>
        </Box>
        <Stack direction="row" spacing={2}>
          <Button variant="outlined" startIcon={<RefreshIcon />} onClick={fetchData}>
            Refresh
          </Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => setCreateDialogOpen(true)}>
            Add Mapping
          </Button>
        </Stack>
      </Box>

      {errorMsg && <Alert severity="error">{errorMsg}</Alert>}
      {successMsg && <Alert severity="success">{successMsg}</Alert>}

      <Paper elevation={0} sx={{ p: 3, border: '1px solid', borderColor: 'divider', borderRadius: 2 }}>
        <TextField
          size="small"
          placeholder="Search by group, role, or tenant..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: 'text.secondary' }} />
              </InputAdornment>
            ),
          }}
          sx={{ width: { xs: '100%', sm: 400 } }}
        />
      </Paper>

      <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? 'rgba(0, 0, 0, 0.2)' : 'grey.50' }}>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Tenant</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>IdP Group Claim</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Grants Role</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Created</TableCell>
              <TableCell align="right" sx={{ fontWeight: 'bold', py: 1.5 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredMappings.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                  <Typography variant="body1" color="text.secondary">
                    No group role mappings found.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredMappings.map((m) => (
                <TableRow key={m.id} hover>
                  <TableCell sx={{ py: 1.5 }}>
                    <Chip label={m.tenant_name} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell sx={{ py: 1.5 }}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <GroupsIcon fontSize="small" color="action" />
                      <Typography variant="body2" fontFamily="monospace">
                        {m.idp_group_claim}
                      </Typography>
                    </Stack>
                  </TableCell>
                  <TableCell sx={{ py: 1.5 }}>
                    <Chip label={m.role_name || m.role_key} size="small" color="primary" />
                  </TableCell>
                  <TableCell sx={{ py: 1.5 }}>
                    <Typography variant="body2" color="text.secondary">
                      {new Date(m.created_at).toLocaleString()}
                    </Typography>
                  </TableCell>
                  <TableCell align="right" sx={{ py: 1.5 }}>
                    <Tooltip title="Remove Mapping">
                      <IconButton size="small" color="error" onClick={() => handleDelete(m)} disabled={actionLoading}>
                        <DeleteIcon />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 'bold' }}>Add Group Role Mapping</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 3 }}>
            <FormControl fullWidth size="small" disabled={!isPlatformOperator && accessibleTenants.length <= 1}>
              <InputLabel>Tenant</InputLabel>
              <Select value={selectedTenantId} label="Tenant" onChange={(e) => setSelectedTenantId(e.target.value)}>
                {accessibleTenants.map((t) => (
                  <MenuItem key={t.id} value={t.id}>
                    {t.display_name || t.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <TextField
              label="IdP Group Claim"
              placeholder="/uisce-staff/northwind-support"
              value={groupClaim}
              onChange={(e) => setGroupClaim(e.target.value)}
              helperText="Must exactly match the group path/name your IdP projects into the JWT's groups claim."
              size="small"
              fullWidth
            />

            <FormControl fullWidth size="small">
              <InputLabel>Grants Role</InputLabel>
              <Select value={selectedRoleKey} label="Grants Role" onChange={(e) => setSelectedRoleKey(e.target.value)}>
                {roles.map((r) => (
                  <MenuItem key={r.id} value={r.role_key}>
                    {r.role_name} ({r.role_key})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleCreate}
            disabled={actionLoading || !groupClaim.trim() || !selectedRoleKey || !selectedTenantId}
          >
            Add Mapping
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default GroupRoleMappingPage;
