/**
 * Delegation Manager - Approval Authority Delegation
 * 
 * Features:
 * - Create temporary approval delegations
 * - Configure date ranges and conditions
 * - Delegation usage tracking and audit trail
 * - Backup and vacation coverage
 * - Active delegation visualization
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  InputAdornment,
  MenuItem,
  Paper,
  Select,
  TextField,
  Typography,
  FormControl,
  InputLabel,
  ToggleButton,
  ToggleButtonGroup,
  Stack,
  Tooltip,
  Autocomplete,
  AutocompleteRenderInputParams,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Avatar,
} from '@mui/material';
import {
  Search as SearchIcon,
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Event as CalendarIcon,
  Security as SecurityIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  TrendingUp as TrendingUpIcon,
  AssignmentInd as AssignmentIndIcon,
  ViewList as ViewListIcon,
  ViewModule as ViewModuleIcon,
} from '@mui/icons-material';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface Role {
  id: string;
  role_name: string;
  role_key: string;
}

interface Delegation {
  id: string;
  delegator_user_id: string;
  delegator_name?: string;
  delegate_user_id: string;
  delegate_name?: string;
  delegation_type: 'full' | 'partial' | 'backup';
  resource_type?: string;
  resource_id?: string;
  start_date: string;
  end_date?: string;
  reason?: string;
  is_active: boolean;
  usage_count?: number;
}

interface DelegationManagerProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const DelegationManager: React.FC<DelegationManagerProps> = ({
  tenant,
  datasource,
}) => {
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [users, setUsers] = useState<any[]>([]); // User list for autocomplete
  const [roles, setRoles] = useState<Role[]>([]); // Role list for resource autocomplete
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<string>('all');
  const [filterStatus, setFilterStatus] = useState<string>('active');
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('table');

  // Form state
  const [formData, setFormData] = useState({
    delegator_user_id: '',
    delegate_user_id: '',
    delegation_type: 'full' as 'full' | 'partial' | 'backup',
    resource_type_selection: 'Role', // 'Role' or 'Custom' (UI only helper)
    resource_type: 'role', // Actual value sent to API (default to role)
    resource_id: '',
    start_date: '',
    end_date: '',
    reason: '',
  });

  // Fetch users for autocomplete
  const fetchUsers = async () => {
    try {
      const response = await fetch(`/api/rbac/users?tenant_id=${tenant.id}`);
      const data = await response.json();
      setUsers(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setUsers([]);
    }
  };

  // Fetch roles for resource autocomplete
  const fetchRoles = async () => {
    try {
      const response = await fetch(`/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`);
      const data = await response.json();
      setRoles(Array.isArray(data) ? data : []);
    } catch (error) {
        console.error('Failed to fetch roles:', error);
        setRoles([]);
    }
  };

  // Fetch delegations
  const fetchDelegations = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `/api/rbac/delegations?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setDelegations(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch delegations:', error);
      setDelegations([]);
    } finally {
      setLoading(false);
    }
  };

  // Create delegation
  const createDelegation = async () => {
    try {
      setSaving(true);
      // Ensure resource type matches selection
      const finalResourceType = formData.delegation_type === 'partial' 
        ? (formData.resource_type_selection === 'Role' ? 'role' : formData.resource_type)
        : '';
        
      const finalResourceId = formData.delegation_type === 'partial' ? formData.resource_id : '';

      await fetch(`/api/rbac/delegations`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...formData,
          resource_type: finalResourceType,
          resource_id: finalResourceId,
          tenant_id: tenant.id,
          tenant_instance_id: datasource.id,
        }),
      });
      await fetchDelegations();
      setShowCreateModal(false);
      resetForm();
    } catch (error) {
      console.error('Failed to create delegation:', error);
    } finally {
      setSaving(false);
    }
  };

  // Delete delegation
  const deleteDelegation = async (delegationId: string) => {
    if (!confirm('Are you sure you want to revoke this delegation?')) return;

    try {
      await fetch(`/api/rbac/delegations/${delegationId}`, { method: 'DELETE' });
      await fetchDelegations();
    } catch (error) {
      console.error('Failed to delete delegation:', error);
    }
  };

  // Reset form
  const resetForm = () => {
    setFormData({
      delegator_user_id: '',
      delegate_user_id: '',
      delegation_type: 'full',
      resource_type_selection: 'Role',
      resource_type: '',
      resource_id: '',
      start_date: '',
      end_date: '',
      reason: '',
    });
  };

  // Filter delegations
  const filteredDelegations = useMemo(() => {
    return delegations.map(d => ({
      ...d,
      delegator_name: users.find(u => u.id === d.delegator_user_id)?.name || d.delegator_name,
      delegate_name: users.find(u => u.id === d.delegate_user_id)?.name || d.delegate_name,
    })).filter(delegation => {
      const searchLower = searchTerm.toLowerCase();
      const matchesSearch = searchTerm === '' ||
        (delegation.delegator_name || delegation.delegator_user_id || '').toLowerCase().includes(searchLower) ||
        (delegation.delegate_name || delegation.delegate_user_id || '').toLowerCase().includes(searchLower) ||
        (delegation.reason || '').toLowerCase().includes(searchLower);
      
      const matchesType = filterType === 'all' || delegation.delegation_type === filterType;
      
      let matchesStatus = true;
      if (filterStatus === 'active') {
        matchesStatus = delegation.is_active;
      } else if (filterStatus === 'inactive') {
        matchesStatus = !delegation.is_active;
      }
      
      return matchesSearch && matchesType && matchesStatus;
    });
  }, [delegations, users, searchTerm, filterType, filterStatus]);

  useEffect(() => {
    if (tenant.id) {
        fetchDelegations();
        fetchUsers();
        fetchRoles();
    }
  }, [tenant.id]);

  const isActive = (delegation: Delegation) => {
    const now = new Date();
    const start = new Date(delegation.start_date);
    const end = delegation.end_date ? new Date(delegation.end_date) : null;
    return delegation.is_active && now >= start && (!end || now <= end);
  };

  const getStatusChip = (delegation: Delegation) => {
    const active = isActive(delegation);
    const expired = delegation.end_date && new Date(delegation.end_date) < new Date();

    if (active) {
      return <Chip icon={<CheckCircleIcon />} label="Active" color="success" size="small" variant="outlined" />;
    }
    if (expired) {
      return <Chip icon={<CancelIcon />} label="Expired" color="error" size="small" variant="outlined" />;
    }
    return <Chip label="Inactive" size="small" variant="outlined" />;
  };

  if (loading) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" minHeight="50vh">
        <Typography>Loading delegations...</Typography>
      </Box>
    );
  }

  // Helper for Autocomplete
  const getUserOptions = () => users.map(u => ({ 
    label: `${u.name} (${u.username})`, 
    id: u.id,
    username: u.username 
  }));

  const getRoleOptions = () => roles.map(r => ({
      label: r.role_name,
      id: r.id,
      key: r.role_key
  }));

  const getDelegatorValue = () => {
    const user = users.find(u => u.id === formData.delegator_user_id);
    return user ? { label: `${user.name} (${user.username})`, id: user.id } : null;
  }

    const getDelegateValue = () => {
    const user = users.find(u => u.id === formData.delegate_user_id);
    return user ? { label: `${user.name} (${user.username})`, id: user.id } : null;
  }

  const getRoleValue = () => {
      const role = roles.find(r => r.id === formData.resource_id);
      return role ? { label: role.role_name, id: role.id, key: role.role_key } : null;
  }

  return (
    <Box sx={{ p: 4 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
        <Box>
          <Box display="flex" alignItems="center" gap={1.5} mb={0.5}>
            <AssignmentIndIcon color="primary" sx={{ fontSize: 32 }} />
            <Typography variant="h4" fontWeight={600}>
              Delegation Overview
            </Typography>
          </Box>
          <Typography variant="body1" color="text.secondary">
            Manage approval authority delegations for {tenant.display_name}
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => {
            resetForm();
            setShowCreateModal(true);
          }}
          sx={{ textTransform: 'none', fontWeight: 600 }}
        >
          Create Delegation
        </Button>
      </Box>

      {/* Filters and View Toggle */}
      <Paper elevation={0} variant="outlined" sx={{ p: 2, mb: 4, bgcolor: 'background.paper', borderRadius: 2 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={4}>
            <Autocomplete
              freeSolo
              options={delegations.flatMap(d => [
                d.delegator_name || d.delegator_user_id, 
                d.delegate_name || d.delegate_user_id
              ]).filter(Boolean)}
              value={searchTerm}
              onInputChange={(_, newValue) => setSearchTerm(newValue || '')}
              renderInput={(params) => (
                <TextField 
                    {...params}
                    fullWidth 
                    size="small" 
                    placeholder="Search delegations..." 
                    InputProps={{
                        ...params.InputProps,
                        startAdornment: (
                        <InputAdornment position="start">
                            <SearchIcon fontSize="small" />
                        </InputAdornment>
                        ),
                    }}
                />
              )}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControl fullWidth size="small">
              <Select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value)}
                displayEmpty
              >
                <MenuItem value="all">All Types</MenuItem>
                <MenuItem value="full">Full Authority</MenuItem>
                <MenuItem value="partial">Specific Approval</MenuItem>
                <MenuItem value="backup">Backup Only</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={3}>
            <ToggleButtonGroup
              value={filterStatus}
              exclusive
              onChange={(_, value) => value && setFilterStatus(value)}
              size="small"
              fullWidth
            >
              <ToggleButton value="active" sx={{ textTransform: 'none' }}>Active</ToggleButton>
              <ToggleButton value="inactive" sx={{ textTransform: 'none' }}>Inactive</ToggleButton>
            </ToggleButtonGroup>
          </Grid>
          <Grid item xs={12} md={2} display="flex" justifyContent="flex-end">
             <ToggleButtonGroup
              value={viewMode}
              exclusive
              onChange={(_, value) => value && setViewMode(value)}
              size="small"
            >
              <ToggleButton value="table" aria-label="table view">
                <ViewListIcon fontSize="small" />
              </ToggleButton>
              <ToggleButton value="grid" aria-label="grid view">
                <ViewModuleIcon fontSize="small" />
              </ToggleButton>
            </ToggleButtonGroup>
          </Grid>
        </Grid>
      </Paper>

      {/* Content */}
      {filteredDelegations.length === 0 ? (
        <Box 
          display="flex" 
          flexDirection="column" 
          alignItems="center" 
          justifyContent="center" 
          py={8}
          bgcolor="background.paper"
          borderRadius={2}
          border={1}
          borderColor="divider"
        >
          <AssignmentIndIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.primary" gutterBottom>
            No Delegations Found
          </Typography>
          <Typography color="text.secondary" mb={3}>
            Create your first delegation to enable approval authority transfer
          </Typography>
          <Button
            variant="contained"
            onClick={() => {
              resetForm();
              setShowCreateModal(true);
            }}
            sx={{ textTransform: 'none' }}
          >
            Create Delegation
          </Button>
        </Box>
      ) : viewMode === 'table' ? (
         <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
          <Table size="medium">
            <TableHead sx={{ bgcolor: 'grey.50' }}>
              <TableRow>
                <TableCell><Typography variant="subtitle2" fontWeight={600}>Delegator</Typography></TableCell>
                <TableCell><Typography variant="subtitle2" fontWeight={600}>Delegate</Typography></TableCell>
                <TableCell><Typography variant="subtitle2" fontWeight={600}>Type</Typography></TableCell>
                <TableCell><Typography variant="subtitle2" fontWeight={600}>Resource</Typography></TableCell>
                <TableCell><Typography variant="subtitle2" fontWeight={600}>Duration</Typography></TableCell>
                <TableCell><Typography variant="subtitle2" fontWeight={600}>Status</Typography></TableCell>
                <TableCell align="right"><Typography variant="subtitle2" fontWeight={600}>Actions</Typography></TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredDelegations.map((delegation) => (
                <TableRow key={delegation.id} hover>
                  <TableCell>
                    <Box display="flex" alignItems="center" gap={1}>
                       <Avatar sx={{ width: 24, height: 24, fontSize: '0.75rem' }}>
                          {(delegation.delegator_name || delegation.delegator_user_id || 'U')[0].toUpperCase()}
                       </Avatar>
                       <Typography variant="body2">{delegation.delegator_name || delegation.delegator_user_id}</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                     <Box display="flex" alignItems="center" gap={1}>
                       <Avatar sx={{ width: 24, height: 24, fontSize: '0.75rem', bgcolor: 'secondary.main' }}>
                          {(delegation.delegate_name || delegation.delegate_user_id || 'U')[0].toUpperCase()}
                       </Avatar>
                       <Typography variant="body2">{delegation.delegate_name || delegation.delegate_user_id}</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip 
                      label={delegation.delegation_type === 'partial' ? 'Specific' : delegation.delegation_type} 
                      size="small" 
                      color="primary" 
                      variant="outlined" 
                    />
                  </TableCell>
                  <TableCell>
                    {(delegation.resource_id || delegation.resource_type) ? (
                      <Box display="flex" alignItems="center" gap={0.5}>
                        <SecurityIcon fontSize="small" color="action" sx={{ fontSize: 16 }} />
                        <Typography variant="caption" color="text.secondary">
                          {getRoleOptions().find(r => r.id === delegation.resource_id)?.label || 
                           (delegation.resource_type === 'role' ? 'Role' : delegation.resource_type) + ': ' + (delegation.resource_id ? delegation.resource_id.substring(0, 8) + '...' : '')}
                        </Typography>
                      </Box>
                    ) : (
                      <Typography variant="caption" color="text.secondary">-</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">
                        {new Date(delegation.start_date).toLocaleDateString()}
                    </Typography>
                    {delegation.end_date && (
                        <Typography variant="caption" color="text.secondary">
                             - {new Date(delegation.end_date).toLocaleDateString()}
                        </Typography>
                    )}
                  </TableCell>
                  <TableCell>{getStatusChip(delegation)}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="Revoke Delegation">
                      <IconButton 
                        size="small" 
                        color="error" 
                        onClick={() => deleteDelegation(delegation.id)}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ) : (
        <Grid container spacing={3}>
          {filteredDelegations.map(delegation => (
            <Grid item xs={12} md={6} lg={4} key={delegation.id}>
              <Card elevation={2} sx={{ height: '100%', borderRadius: 2 }}>
                <CardContent>
                  <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
                    <Box>
                      <Box display="flex" alignItems="center" gap={1} mb={0.5}>
                        <Typography variant="subtitle1" fontWeight={600}>
                          {delegation.delegator_name || delegation.delegator_user_id}
                        </Typography>
                        <Typography color="text.secondary">→</Typography>
                        <Typography variant="subtitle1" fontWeight={600}>
                          {delegation.delegate_name || delegation.delegate_user_id}
                        </Typography>
                      </Box>
                      {delegation.reason && (
                        <Typography variant="body2" color="text.secondary" sx={{ 
                          display: '-webkit-box',
                          overflow: 'hidden',
                          WebkitBoxOrient: 'vertical',
                          WebkitLineClamp: 2,
                        }}>
                          {delegation.reason}
                        </Typography>
                      )}
                    </Box>
                    <IconButton 
                      size="small" 
                      color="error" 
                      onClick={() => deleteDelegation(delegation.id)}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Box>

                  <Box display="flex" flexWrap="wrap" gap={1} mb={2}>
                    {getStatusChip(delegation)}
                    <Chip 
                      label={delegation.delegation_type === 'partial' ? 'Specific Approval' : delegation.delegation_type} 
                      size="small" 
                      color="primary" 
                      variant="outlined" 
                    />
                    {delegation.usage_count !== undefined && (
                      <Chip 
                        icon={<TrendingUpIcon />} 
                        label={`${delegation.usage_count} uses`} 
                        size="small" 
                        variant="outlined" 
                      />
                    )}
                  </Box>

                  <Stack spacing={1}>
                    <Box display="flex" alignItems="center" gap={1}>
                      <CalendarIcon fontSize="small" color="action" />
                      <Typography variant="caption" color="text.secondary">
                        {new Date(delegation.start_date).toLocaleDateString()} 
                        {delegation.end_date && ` - ${new Date(delegation.end_date).toLocaleDateString()}`}
                      </Typography>
                    </Box>
                    {/* Display Role Name if it's a role delegation, otherwise show resource ID */}
                    {(delegation.resource_id || delegation.resource_type) && (
                      <Box display="flex" alignItems="center" gap={1}>
                        <SecurityIcon fontSize="small" color="action" />
                        <Typography variant="caption" color="text.secondary">
                          {getRoleOptions().find(r => r.id === delegation.resource_id)?.label || 
                           (delegation.resource_type === 'role' ? 'Role' : delegation.resource_type) + ': ' + (delegation.resource_id ? delegation.resource_id.substring(0, 8) + '...' : '')}
                        </Typography>
                      </Box>
                    )}
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {/* Create Modal */}
      <Dialog
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ elevation: 8, sx: { borderRadius: 2 } }}
      >
        <DialogTitle>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Typography variant="h6" fontWeight={600}>
              Create/Edit Delegation
            </Typography>
            <IconButton onClick={() => setShowCreateModal(false)} size="small">
              <CancelIcon />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={3} mt={1}>
            
            {/* Delegator Autocomplete */}
            <Autocomplete
                options={getUserOptions()}
                value={getDelegatorValue()}
                onChange={(_, newValue: { label: string; id: string; username?: string } | null) => {
                    setFormData({ ...formData, delegator_user_id: newValue ? newValue.id : '' });
                }}
                renderInput={(params: AutocompleteRenderInputParams) => (
                    <TextField {...params} label="Delegator User" required error={!formData.delegator_user_id} helperText={!formData.delegator_user_id ? "Required" : ""} />
                )}
            />

            {/* Delegate Autocomplete */}
            <Autocomplete
                options={getUserOptions()}
                value={getDelegateValue()}
                onChange={(_, newValue: { label: string; id: string; username?: string } | null) => {
                    setFormData({ ...formData, delegate_user_id: newValue ? newValue.id : '' });
                }}
                renderInput={(params: AutocompleteRenderInputParams) => (
                    <TextField {...params} label="Delegate User" required error={!formData.delegate_user_id} helperText={!formData.delegate_user_id ? "Required" : ""} />
                )}
            />
            
            <FormControl fullWidth required>
              <InputLabel>Delegation Type</InputLabel>
              <Select
                value={formData.delegation_type}
                label="Delegation Type"
                onChange={(e) => setFormData({ ...formData, delegation_type: e.target.value as any })}
              >
                <MenuItem value="full">Full Authority</MenuItem>
                <MenuItem value="partial">Specific Approval</MenuItem>
                <MenuItem value="backup">Backup Only</MenuItem>
              </Select>
            </FormControl>

            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField
                  label="Start Date"
                  type="datetime-local"
                  fullWidth
                  required
                  InputLabelProps={{ shrink: true }}
                  value={formData.start_date}
                  onChange={(e) => setFormData({ ...formData, start_date: e.target.value })}
                  helperText="This field is required."
                  error={!formData.start_date}
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  label="End Date"
                  type="datetime-local"
                  fullWidth
                  InputLabelProps={{ shrink: true }}
                  value={formData.end_date}
                  onChange={(e) => setFormData({ ...formData, end_date: e.target.value })}
                  placeholder="Optional"
                />
              </Grid>
            </Grid>

            <TextField
              label="Reason"
              fullWidth
              multiline
              rows={3}
              value={formData.reason}
              onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
              placeholder="e.g., Vacation coverage, Backup approver"
            />
            
            {/* Conditional Fields for Specific Approval */}
            {formData.delegation_type === 'partial' && (
                <>
                    <FormControl fullWidth required>
                        <InputLabel>Resource Type</InputLabel>
                        <Select
                            value={formData.resource_type_selection}
                            label="Resource Type"
                            onChange={(e) => {
                                const selection = e.target.value;
                                setFormData({ 
                                    ...formData, 
                                    resource_type_selection: selection,
                                    resource_type: selection === 'Role' ? 'role' : '', // Clear or set default
                                    resource_id: '' // Reset ID when type changes
                                });
                            }}
                        >
                            <MenuItem value="Role">Role (Standard)</MenuItem>
                            <MenuItem value="Custom">Custom / Other</MenuItem>
                        </Select>
                    </FormControl>

                    {formData.resource_type_selection === 'Role' ? (
                        <Autocomplete
                            options={getRoleOptions()}
                            value={getRoleValue()}
                            onChange={(_, newValue: { label: string; id: string; key: string } | null) => {
                                setFormData({ ...formData, resource_id: newValue ? newValue.id : '' });
                            }}
                            renderInput={(params: AutocompleteRenderInputParams) => (
                                <TextField {...params} label="Select Role" required error={!formData.resource_id} helperText={!formData.resource_id ? "Required" : ""} />
                            )}
                        />
                    ) : (
                        <>
                            <TextField
                            label="Custom Resource Type"
                            fullWidth
                            required
                            value={formData.resource_type}
                            onChange={(e) => setFormData({ ...formData, resource_type: e.target.value })}
                            />
                            <TextField
                            label="Custom Resource ID"
                            fullWidth
                            required
                            value={formData.resource_id}
                            onChange={(e) => setFormData({ ...formData, resource_id: e.target.value })}
                            error={!formData.resource_id}
                            helperText={!formData.resource_id ? "Required" : ""}
                            />
                        </>
                    )}
                </>
            )}
            
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setShowCreateModal(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={createDelegation}
            disabled={
                !formData.delegator_user_id || 
                !formData.delegate_user_id || 
                !formData.start_date || 
                (formData.delegation_type === 'partial' && (!formData.resource_id)) ||
                saving
            }
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            {saving ? 'Creating...' : 'Create Delegation'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
