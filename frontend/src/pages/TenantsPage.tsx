import React, { useState } from 'react';
import {
  Box,
  Typography,
  List,
  ListItem,
  ListItemText,
  Button,
  CircularProgress,
  Alert,
  Tabs,
  Tab,
  Paper,
  TextField,
  Grid,
  Card,
  CardContent,
  CardActions,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  IconButton,
  Tooltip
} from '@mui/material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import SettingsIcon from '@mui/icons-material/Settings';

interface Tenant {
  id: string;
  name: string;
  display_name?: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`tenant-tabpanel-${index}`}
      aria-labelledby={`tenant-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const TenantsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);

  // Form states
  const [newTenantName, setNewTenantName] = useState('');
  const [newTenantDisplayName, setNewTenantDisplayName] = useState('');
  const [newTenantDescription, setNewTenantDescription] = useState('');
  const [editDisplayName, setEditDisplayName] = useState('');
  const [editDescription, setEditDescription] = useState('');

  const queryClient = useQueryClient();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['tenants'],
    queryFn: () => apiFetch('/api/rest/tenants').then(r => r.json()),
  });

  const createTenant = useMutation({
    mutationFn: (input: { name: string; display_name?: string; description?: string }) =>
      apiFetch('/api/rest/tenants', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tenants'] }),
  });

  const updateTenant = useMutation({
    mutationFn: ({ id, ...input }: { id: string; display_name?: string; description?: string }) =>
      apiFetch(`/api/rest/tenants/${id}`, { method: 'PATCH', body: JSON.stringify(input) }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tenants'] }),
  });

  const deleteTenant = useMutation({
    mutationFn: (id: string) => apiFetch(`/api/rest/tenants/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tenants'] }),
  });

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const handleSelectTenant = (tenant: Tenant) => {
    setSelectedTenant(tenant);
    console.log('Selected tenant:', tenant);
  };

  const handleCreateTenant = async () => {
    try {
      await createTenant.mutateAsync({
        name: newTenantName,
        display_name: newTenantDisplayName || undefined,
        description: newTenantDescription || undefined,
      });
      setCreateDialogOpen(false);
      resetCreateForm();
    } catch (err) {
      console.error('Error creating tenant:', err);
    }
  };

  const handleEditTenant = async () => {
    if (!editingTenant) return;

    try {
      await updateTenant.mutateAsync({
        id: editingTenant.id,
        display_name: editDisplayName || undefined,
        description: editDescription || undefined,
      });
      setEditDialogOpen(false);
      setEditingTenant(null);
      resetEditForm();
    } catch (err) {
      console.error('Error updating tenant:', err);
    }
  };

  const handleDeleteTenant = async (tenant: Tenant) => {
    if (!window.confirm(`Are you sure you want to delete tenant "${tenant.display_name || tenant.name}"?`)) {
      return;
    }

    try {
      await deleteTenant.mutateAsync(tenant.id);
    } catch (err) {
      console.error('Error deleting tenant:', err);
    }
  };

  const openEditDialog = (tenant: Tenant) => {
    setEditingTenant(tenant);
    setEditDisplayName(tenant.display_name || '');
    setEditDescription(tenant.description || '');
    setEditDialogOpen(true);
  };

  const resetCreateForm = () => {
    setNewTenantName('');
    setNewTenantDisplayName('');
    setNewTenantDescription('');
  };

  const resetEditForm = () => {
    setEditDisplayName('');
    setEditDescription('');
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ padding: 3 }}>
        <Alert severity="error">
          Error fetching tenants: {(error as Error).message}
        </Alert>
      </Box>
    );
  }

  const tenants = data || [];

  return (
    <Box sx={{ width: '100%' }}>
      <Paper sx={{ width: '100%', mb: 2 }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="tenant management tabs">
          <Tab label="Select Tenant" />
          <Tab label="Manage Tenants" />
          <Tab label="Create Tenant" />
        </Tabs>
      </Paper>

      <TabPanel value={activeTab} index={0}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Select a Tenant
          </Typography>
          <Typography variant="body1" sx={{ marginBottom: 3 }}>
            Choose a tenant to continue to the dashboard
          </Typography>
          {tenants.length === 0 ? (
            <Typography>No tenants available</Typography>
          ) : (
            <Grid container spacing={2}>
              {tenants.map((tenant: Tenant) => (
                <Grid size={{ 'xs': 12, 'sm': 6, 'md': 4 }}>
                  <Card
                    sx={{
                      height: '100%',
                      cursor: 'pointer',
                      border: selectedTenant?.id === tenant.id ? '2px solid #1976d2' : '1px solid #e0e0e0',
                    }}
                    onClick={() => handleSelectTenant(tenant)}
                  >
                    <CardContent>
                      <Typography variant="h6" component="div">
                        {tenant.display_name || tenant.name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                        {tenant.description || 'No description'}
                      </Typography>
                      <Chip
                        label={selectedTenant?.id === tenant.id ? 'Selected' : 'Select'}
                        color={selectedTenant?.id === tenant.id ? 'primary' : 'default'}
                        size="small"
                      />
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          )}
        </Box>
      </TabPanel>

      <TabPanel value={activeTab} index={1}>
        <Box>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
            <Typography variant="h4">Manage Tenants</Typography>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => setActiveTab(2)}
            >
              Create New Tenant
            </Button>
          </Box>
          {tenants.length === 0 ? (
            <Typography>No tenants available</Typography>
          ) : (
            <Grid container spacing={2}>
              {tenants.map((tenant: Tenant) => (
                <Grid size={{ 'xs': 12, 'md': 6 }}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6" component="div">
                        {tenant.display_name || tenant.name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                        {tenant.description || 'No description'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Created: {new Date(tenant.created_at || '').toLocaleDateString()}
                      </Typography>
                    </CardContent>
                    <CardActions>
                      <Tooltip title="Edit Tenant">
                        <IconButton onClick={() => openEditDialog(tenant)}>
                          <EditIcon />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete Tenant">
                        <IconButton
                          onClick={() => handleDeleteTenant(tenant)}
                          color="error"
                        >
                          <DeleteIcon />
                        </IconButton>
                      </Tooltip>
                    </CardActions>
                  </Card>
                </Grid>
              ))}
            </Grid>
          )}
        </Box>
      </TabPanel>

      <TabPanel value={activeTab} index={2}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Create New Tenant
          </Typography>
          <Typography variant="body1" sx={{ marginBottom: 3 }}>
            Add a new tenant to the system
          </Typography>
          <Card sx={{ maxWidth: 600, mx: 'auto' }}>
            <CardContent>
              <Grid container spacing={2}>
                <Grid size={12}>
                  <TextField
                    fullWidth
                    label="Tenant Name"
                    value={newTenantName}
                    onChange={(e) => setNewTenantName(e.target.value)}
                    required
                    helperText="Unique identifier for the tenant"
                  />
                </Grid>
                <Grid size={12}>
                  <TextField
                    fullWidth
                    label="Display Name"
                    value={newTenantDisplayName}
                    onChange={(e) => setNewTenantDisplayName(e.target.value)}
                    helperText="Human-readable name (optional)"
                  />
                </Grid>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    multiline
                    rows={3}
                    label="Description"
                    value={newTenantDescription}
                    onChange={(e) => setNewTenantDescription(e.target.value)}
                    helperText="Brief description of the tenant (optional)"
                  />
                </Grid>
              </Grid>
            </CardContent>
            <CardActions>
              <Button onClick={() => setActiveTab(1)}>Cancel</Button>
              <Button
                variant="contained"
                onClick={handleCreateTenant}
                disabled={!newTenantName.trim()}
              >
                Create Tenant
              </Button>
            </CardActions>
          </Card>
        </Box>
      </TabPanel>

      {/* Create Tenant Dialog */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Tenant</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Tenant Name"
                value={newTenantName}
                onChange={(e) => setNewTenantName(e.target.value)}
                required
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Display Name"
                value={newTenantDisplayName}
                onChange={(e) => setNewTenantDisplayName(e.target.value)}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                multiline
                rows={3}
                label="Description"
                value={newTenantDescription}
                onChange={(e) => setNewTenantDescription(e.target.value)}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleCreateTenant} variant="contained">
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Tenant Dialog */}
      <Dialog open={editDialogOpen} onClose={() => setEditDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Tenant</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Display Name"
                value={editDisplayName}
                onChange={(e) => setEditDisplayName(e.target.value)}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                multiline
                rows={3}
                label="Description"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleEditTenant} variant="contained">
            Update
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TenantsPage;