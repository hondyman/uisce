import React, { useState } from 'react';
import {
  Box,
  Typography,
  Stack,
  Switch,
  FormControlLabel,
  TextField,
  Button,
  Divider,
  useTheme,
  alpha,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Card,
  CardContent,
  CardActions,
  Grid,
  Chip,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { Edit as EditIcon, Delete as DeleteIcon, Visibility as VisibilityIcon, Download as DownloadIcon, GridView as GridViewIcon, ViewAgenda as ViewAgendaIcon } from '@mui/icons-material';
import { useNotification } from '../../../hooks/useNotification';

interface IPWhitelistSettings {
  enforceValidation: boolean;
  allowWildcards: boolean;
  maxIPsPerTenant: number;
  requireApprovalForGlobal: boolean;
  notifyOnConflict: boolean;
}

interface AccessPolicy {
  id: string;
  name: string;
  description: string;
  effect: 'allow' | 'deny';
  resources: string[];
  actions: string[];
  enabled: boolean;
}

const SettingsPage: React.FC = () => {
  const theme = useTheme();
  const notification = useNotification();
  const [settings, setSettings] = useState<IPWhitelistSettings>({
    enforceValidation: true,
    allowWildcards: false,
    maxIPsPerTenant: 100,
    requireApprovalForGlobal: true,
    notifyOnConflict: true,
  });
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [policies, _setPolicies] = useState<AccessPolicy[]>([
    {
      id: '1',
      name: 'IP Validation Policy',
      description: 'Enforces strict IP address validation format',
      effect: 'allow',
      resources: ['ip-whitelist'],
      actions: ['create', 'read'],
      enabled: true,
    },
    {
      id: '2',
      name: 'Tenant Isolation Policy',
      description: 'Prevents cross-tenant IP access',
      effect: 'deny',
      resources: ['ip-whitelist', 'tenant-assignment'],
      actions: ['cross-tenant-access'],
      enabled: true,
    },
    {
      id: '3',
      name: 'Global IP Approval Policy',
      description: 'Requires admin approval for global IP assignments',
      effect: 'allow',
      resources: ['global-ip'],
      actions: ['create'],
      enabled: true,
    },
  ]);
  const [selectedPolicy, setSelectedPolicy] = useState<AccessPolicy | null>(null);
  const [policyDetailOpen, setPolicyDetailOpen] = useState(false);
  const [policyViewMode, setPolicyViewMode] = useState<'tile' | 'table'>('tile');

  const handleSettingChange = (key: keyof IPWhitelistSettings, value: any) => {
    setSettings(prev => ({ ...prev, [key]: value }));
  };

  const handleExportPolicies = (format: 'csv' | 'json') => {
    if (format === 'csv') {
      const headers = ['ID', 'Name', 'Description', 'Effect', 'Resources', 'Actions', 'Enabled'];
      const rows = policies.map(p => [
        p.id,
        p.name,
        p.description,
        p.effect,
        p.resources.join(';'),
        p.actions.join(';'),
        p.enabled ? 'Yes' : 'No'
      ]);
      
      const csv = [headers, ...rows].map(row => 
        row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(',')
      ).join('\n');
      
      const blob = new Blob([csv], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `policies-${new Date().toISOString().split('T')[0]}.csv`;
      a.click();
      window.URL.revokeObjectURL(url);
    } else {
      const json = JSON.stringify(policies, null, 2);
      const blob = new Blob([json], { type: 'application/json' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `policies-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
      window.URL.revokeObjectURL(url);
    }
    notification.success(`Policies exported as ${format.toUpperCase()}`);
  };

  const handleSaveSettings = () => {
    setSaveDialogOpen(true);
  };

  const confirmSaveSettings = () => {
    // In a real app, you'd save to backend here
    localStorage.setItem('ipWhitelistSettings', JSON.stringify(settings));
    setSaveDialogOpen(false);
    notification.success('Settings saved successfully');
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack spacing={3}>
        {/* Header */}
        <Box>
          <Typography variant="h4" fontWeight={900} gutterBottom>
            Settings
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Configure IP whitelist policies and validation rules
          </Typography>
        </Box>

        {/* Settings Cards Grid */}
        <Grid container spacing={2}>
          {/* IP Validation Card */}
          <Grid item xs={12} sm={6} md={4}>
            <Card>
              <CardContent>
                <Typography variant="h6" fontWeight={700} gutterBottom>
                  IP Validation
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                  Control how IP addresses are validated
                </Typography>
                <Stack spacing={2}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={settings.enforceValidation}
                        onChange={(e) => handleSettingChange('enforceValidation', e.target.checked)}
                      />
                    }
                    label="Enforce validation"
                  />
                  <Typography variant="caption" color="text.secondary">
                    All IPs must be valid IPv4/IPv6
                  </Typography>

                  <Divider />

                  <FormControlLabel
                    control={
                      <Switch
                        checked={settings.allowWildcards}
                        onChange={(e) => handleSettingChange('allowWildcards', e.target.checked)}
                      />
                    }
                    label="Allow wildcards"
                  />
                  <Typography variant="caption" color="text.secondary">
                    Allow patterns like 192.168.*.*
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          {/* Tenant Assignment Card */}
          <Grid item xs={12} sm={6} md={4}>
            <Card>
              <CardContent>
                <Typography variant="h6" fontWeight={700} gutterBottom>
                  Tenant Assignment
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                  Rules for assigning IPs to tenants
                </Typography>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="body2" fontWeight={600} sx={{ mb: 1 }}>
                      Max IPs per Tenant
                    </Typography>
                    <TextField
                      type="number"
                      size="small"
                      value={settings.maxIPsPerTenant}
                      onChange={(e) => handleSettingChange('maxIPsPerTenant', parseInt(e.target.value, 10))}
                      inputProps={{ min: 1, max: 10000 }}
                      fullWidth
                    />
                  </Box>

                  <Divider />

                  <FormControlLabel
                    control={
                      <Switch
                        checked={settings.requireApprovalForGlobal}
                        onChange={(e) => handleSettingChange('requireApprovalForGlobal', e.target.checked)}
                      />
                    }
                    label="Require approval"
                  />
                  <Typography variant="caption" color="text.secondary">
                    Global IPs need admin approval
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          {/* Notifications Card */}
          <Grid item xs={12} sm={6} md={4}>
            <Card>
              <CardContent>
                <Typography variant="h6" fontWeight={700} gutterBottom>
                  Notifications
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                  Configure alerts and notifications
                </Typography>
                <Stack spacing={2}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={settings.notifyOnConflict}
                        onChange={(e) => handleSettingChange('notifyOnConflict', e.target.checked)}
                      />
                    }
                    label="Notify on conflicts"
                  />
                  <Typography variant="caption" color="text.secondary">
                    Alert when duplicates or overlaps detected
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Divider */}
        <Divider sx={{ my: 2 }} />

        {/* Access Control Policies Section */}
        <Box sx={{ mb: 4 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
            <Box>
              <Typography variant="h5" fontWeight={700} gutterBottom>
                Access Control Policies
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Manage policies that control access to IP whitelist resources
              </Typography>
            </Box>
            <Stack direction="row" spacing={0.5}>
              <IconButton
                size="small"
                onClick={() => setPolicyViewMode('tile')}
                title="Tile View"
                color={policyViewMode === 'tile' ? 'primary' : 'default'}
              >
                <GridViewIcon fontSize="small" />
              </IconButton>
              <IconButton
                size="small"
                onClick={() => setPolicyViewMode('table')}
                title="Table View"
                color={policyViewMode === 'table' ? 'primary' : 'default'}
              >
                <ViewAgendaIcon fontSize="small" />
              </IconButton>
              <IconButton
                size="small"
                onClick={() => handleExportPolicies('csv')}
                title="Export as CSV"
              >
                <DownloadIcon fontSize="small" />
              </IconButton>
              <IconButton
                size="small"
                onClick={() => handleExportPolicies('json')}
                title="Export as JSON"
              >
                <DownloadIcon fontSize="small" />
              </IconButton>
            </Stack>
          </Stack>

          {/* Tile View */}
          {policyViewMode === 'tile' && (
            <Grid container spacing={2}>
                {policies.map((policy) => (
                  <Grid item xs={12} sm={6} md={4} key={policy.id}>
                    <Card 
                      sx={{
                        height: '100%',
                        display: 'flex',
                        flexDirection: 'column',
                        transition: 'all 0.2s ease-in-out',
                        '&:hover': {
                          boxShadow: theme.shadows[8],
                          transform: 'translateY(-2px)',
                        }
                      }}
                    >
                      <CardContent sx={{ flexGrow: 1 }}>
                        <Stack spacing={1.5}>
                          <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                            <Typography variant="h6" fontWeight={700}>
                              {policy.name}
                            </Typography>
                            <Chip
                              label={policy.effect.toUpperCase()}
                              color={policy.effect === 'allow' ? 'success' : 'error'}
                              size="small"
                              variant="filled"
                            />
                          </Box>
                          <Typography variant="body2" color="text.secondary">
                            {policy.description}
                          </Typography>
                          <Box>
                            <Typography variant="caption" fontWeight={600} color="text.secondary">
                              Resources:
                            </Typography>
                            <Stack direction="row" spacing={0.5} flexWrap="wrap" sx={{ mt: 0.5 }}>
                              {policy.resources.map((resource) => (
                                <Chip
                                  key={resource}
                                  label={resource}
                                  size="small"
                                  variant="outlined"
                                  sx={{ mb: 0.5 }}
                                />
                              ))}
                            </Stack>
                          </Box>
                          <Box>
                            <Typography variant="caption" fontWeight={600} color="text.secondary">
                              Actions:
                            </Typography>
                            <Stack direction="row" spacing={0.5} flexWrap="wrap" sx={{ mt: 0.5 }}>
                              {policy.actions.map((action) => (
                                <Chip
                                  key={action}
                                  label={action}
                                  size="small"
                                  variant="outlined"
                                  sx={{ mb: 0.5 }}
                                />
                              ))}
                            </Stack>
                          </Box>
                          <Box display="flex" alignItems="center" gap={1} sx={{ mt: 1 }}>
                            <Typography variant="caption">
                              {policy.enabled ? '✅ Active' : '❌ Inactive'}
                            </Typography>
                          </Box>
                        </Stack>
                      </CardContent>
                      <CardActions>
                        <IconButton 
                          size="small" 
                          onClick={() => {
                            setSelectedPolicy(policy);
                            setPolicyDetailOpen(true);
                          }}
                          title="View details"
                        >
                          <VisibilityIcon fontSize="small" />
                        </IconButton>
                        <IconButton 
                          size="small"
                          title="Edit policy"
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton 
                          size="small"
                          color="error"
                          title="Delete policy"
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </CardActions>
                    </Card>
                  </Grid>
                ))}
              </Grid>
          )}

          {/* Table View */}
          {policyViewMode === 'table' && (
            <Card>
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? 'grey.800' : 'grey.100' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Name</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Description</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Effect</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Resources</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Actions</TableCell>
                      <TableCell sx={{ fontWeight: 700 }} align="center">Status</TableCell>
                      <TableCell sx={{ fontWeight: 700 }} align="center">Options</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {policies.map((policy) => (
                      <TableRow key={policy.id} hover>
                        <TableCell>
                          <Typography variant="body2" fontWeight={600}>
                            {policy.name}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" color="text.secondary">
                            {policy.description}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={policy.effect.toUpperCase()}
                            color={policy.effect === 'allow' ? 'success' : 'error'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Stack direction="row" spacing={0.5} flexWrap="wrap">
                            {policy.resources.map((resource) => (
                              <Chip
                                key={resource}
                                label={resource}
                                size="small"
                                variant="outlined"
                              />
                            ))}
                          </Stack>
                        </TableCell>
                        <TableCell>
                          <Stack direction="row" spacing={0.5} flexWrap="wrap">
                            {policy.actions.map((action) => (
                              <Chip
                                key={action}
                                label={action}
                                size="small"
                                variant="outlined"
                              />
                            ))}
                          </Stack>
                        </TableCell>
                        <TableCell align="center">
                          <Chip
                            label={policy.enabled ? 'Active' : 'Inactive'}
                            color={policy.enabled ? 'success' : 'default'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell align="center">
                          <Stack direction="row" spacing={0.5} justifyContent="center">
                            <IconButton 
                              size="small" 
                              onClick={() => {
                                setSelectedPolicy(policy);
                                setPolicyDetailOpen(true);
                              }}
                              title="View details"
                            >
                              <VisibilityIcon fontSize="small" />
                            </IconButton>
                            <IconButton 
                              size="small"
                              title="Edit policy"
                            >
                              <EditIcon fontSize="small" />
                            </IconButton>
                            <IconButton 
                              size="small"
                              color="error"
                              title="Delete policy"
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Stack>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Card>
          )}
        </Box>

        {/* Divider */}
        <Divider sx={{ my: 2 }} />

        {/* Danger Zone Card */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h5" fontWeight={700} gutterBottom>
            Danger Zone
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Be careful with these settings
          </Typography>
          <Card sx={{ bgcolor: alpha(theme.palette.error.main, 0.05), border: `1px solid ${theme.palette.error.main}` }}>
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6" fontWeight={700}>
                  Reset All Settings
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Restore all settings to their default values
                </Typography>
                <Button
                  variant="outlined"
                  color="error"
                  onClick={() => {
                    setSettings({
                      enforceValidation: true,
                      allowWildcards: false,
                      maxIPsPerTenant: 100,
                      requireApprovalForGlobal: true,
                      notifyOnConflict: true,
                    });
                    notification.success('Settings reset to defaults');
                  }}
                >
                  Reset to Defaults
                </Button>
              </Stack>
            </CardContent>
          </Card>
        </Box>

        {/* Divider */}
        <Divider sx={{ my: 3 }} />

        {/* Save Button */}
        <Stack direction="row" justifyContent="flex-end" spacing={2}>
          <Button variant="outlined">
            Cancel
          </Button>
          <Button variant="contained" onClick={handleSaveSettings}>
            Save Settings
          </Button>
        </Stack>
      </Stack>

      {/* Save Confirmation Dialog */}
      <Dialog open={saveDialogOpen} onClose={() => setSaveDialogOpen(false)}>
        <DialogTitle>Save Settings</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to save these changes? This will update the IP whitelist configuration for all users.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSaveDialogOpen(false)}>Cancel</Button>
          <Button onClick={confirmSaveSettings} variant="contained">
            Confirm Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Policy Detail Modal */}
      <Dialog open={policyDetailOpen} onClose={() => setPolicyDetailOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          Policy Details: {selectedPolicy?.name}
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {selectedPolicy && (
            <Stack spacing={2}>
              <Box>
                <Typography variant="subtitle2" fontWeight={700}>Effect</Typography>
                <Chip
                  label={selectedPolicy.effect.toUpperCase()}
                  color={selectedPolicy.effect === 'allow' ? 'success' : 'error'}
                  sx={{ mt: 1 }}
                />
              </Box>

              <Box>
                <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>Description</Typography>
                <Typography variant="body2">{selectedPolicy.description}</Typography>
              </Box>

              <Box>
                <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>Resources</Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap">
                  {selectedPolicy.resources.map((resource) => (
                    <Chip key={resource} label={resource} variant="outlined" />
                  ))}
                </Stack>
              </Box>

              <Box>
                <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>Actions</Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap">
                  {selectedPolicy.actions.map((action) => (
                    <Chip key={action} label={action} variant="outlined" />
                  ))}
                </Stack>
              </Box>

              <Box>
                <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>Status</Typography>
                <Chip
                  label={selectedPolicy.enabled ? 'Active' : 'Inactive'}
                  color={selectedPolicy.enabled ? 'success' : 'default'}
                />
              </Box>
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPolicyDetailOpen(false)}>Close</Button>
          <Button variant="contained">Edit Policy</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default SettingsPage;
