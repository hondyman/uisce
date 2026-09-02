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
  Switch,
  FormControlLabel,
  Collapse,
  useTheme,
} from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  Add as AddIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  VpnKey as KeyIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';
import { useAccess } from '../../../contexts/AccessContext';

interface IdentityProvider {
  id: string;
  issuer: string;
  jwks_uri: string;
  display_name: string;
  is_cross_tenant: boolean;
  is_active: boolean;
  created_at: string;
  created_by: string;
}

interface IdPGrant {
  id: string;
  idp_id: string;
  tenant_id: string;
  tenant_name: string;
  granted_at: string;
  granted_by: string;
}

// Admin screen for semantic.tenant_identity_providers / _grants: which
// external issuers (Keycloak realms, Entra tenants, etc.) uisce trusts, and
// which tenant(s) each one is allowed to claim in its JWTs. This is the
// actual trust boundary — a token's issuer must be registered here, and its
// tenant_id claim must be within this grant list, or the request is
// rejected (see services.ValidateIssuerTenant). Deactivating an IdP here
// (rather than deleting) is deliberately reversible.
const IdentityProvidersPage: React.FC = () => {
  const theme = useTheme();
  const { accessibleTenants, isPlatformOperator } = useAccess();

  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [idps, setIdps] = useState<IdentityProvider[]>([]);
  const [grantsByIdp, setGrantsByIdp] = useState<Record<string, IdPGrant[]>>({});
  const [expandedIdp, setExpandedIdp] = useState<string | null>(null);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [grantDialogIdp, setGrantDialogIdp] = useState<IdentityProvider | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const [issuer, setIssuer] = useState('');
  const [jwksUri, setJwksUri] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [isCrossTenant, setIsCrossTenant] = useState(false);
  const [grantTenantId, setGrantTenantId] = useState('');

  const fetchIdps = async () => {
    try {
      setLoading(true);
      setErrorMsg(null);
      const data = await apiClient<IdentityProvider[]>('/api/rbac/idps');
      setIdps(Array.isArray(data) ? data : []);
    } catch (err: any) {
      console.error('Failed to fetch identity providers:', err);
      setErrorMsg(err.message || 'Failed to fetch identity providers.');
      setIdps([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchIdps();
  }, []);

  const fetchGrants = async (idpId: string) => {
    try {
      const data = await apiClient<IdPGrant[]>(`/api/rbac/idps/${idpId}/grants`);
      setGrantsByIdp((prev) => ({ ...prev, [idpId]: Array.isArray(data) ? data : [] }));
    } catch (err: any) {
      console.error('Failed to fetch grants:', err);
    }
  };

  const toggleExpand = (idp: IdentityProvider) => {
    if (expandedIdp === idp.id) {
      setExpandedIdp(null);
      return;
    }
    setExpandedIdp(idp.id);
    if (!grantsByIdp[idp.id]) {
      fetchGrants(idp.id);
    }
  };

  const filteredIdps = useMemo(() => {
    const q = searchTerm.toLowerCase();
    return idps.filter((i) => i.issuer.toLowerCase().includes(q) || i.display_name.toLowerCase().includes(q));
  }, [idps, searchTerm]);

  const handleCreate = async () => {
    if (!issuer.trim() || !jwksUri.trim()) {
      setErrorMsg('Issuer and JWKS URI are required.');
      return;
    }
    setActionLoading(true);
    try {
      setErrorMsg(null);
      await apiClient('/api/rbac/idps', {
        method: 'POST',
        body: JSON.stringify({
          issuer: issuer.trim(),
          jwks_uri: jwksUri.trim(),
          display_name: displayName.trim(),
          is_cross_tenant: isCrossTenant,
        }),
      });
      setSuccessMsg('Identity provider registered. Grant it to a tenant next, and the JWKS refresh loop will pick it up within its refresh interval.');
      setCreateDialogOpen(false);
      setIssuer('');
      setJwksUri('');
      setDisplayName('');
      setIsCrossTenant(false);
      fetchIdps();
      setTimeout(() => setSuccessMsg(null), 8000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to register identity provider.');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeactivate = async (idp: IdentityProvider) => {
    if (!window.confirm(`Deactivate "${idp.display_name || idp.issuer}"? Every user authenticating through it will immediately lose access.`)) {
      return;
    }
    setActionLoading(true);
    try {
      await apiClient(`/api/rbac/idps/${idp.id}`, {
        method: 'PUT',
        body: JSON.stringify({ is_active: false }),
      });
      setSuccessMsg('Identity provider deactivated.');
      fetchIdps();
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to deactivate identity provider.');
    } finally {
      setActionLoading(false);
    }
  };

  const handleGrant = async () => {
    if (!grantDialogIdp || !grantTenantId) return;
    setActionLoading(true);
    try {
      await apiClient(`/api/rbac/idps/${grantDialogIdp.id}/grants`, {
        method: 'POST',
        body: JSON.stringify({ tenant_id: grantTenantId }),
      });
      setSuccessMsg('Tenant granted.');
      fetchGrants(grantDialogIdp.id);
      setGrantDialogIdp(null);
      setGrantTenantId('');
      setTimeout(() => setSuccessMsg(null), 5000);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to grant tenant.');
    } finally {
      setActionLoading(false);
    }
  };

  const handleRevoke = async (idpId: string, tenantId: string) => {
    if (!window.confirm('Revoke this tenant grant? Users of that tenant authenticating through this IdP will immediately lose access.')) {
      return;
    }
    setActionLoading(true);
    try {
      await apiClient(`/api/rbac/idps/${idpId}/grants/${tenantId}`, { method: 'DELETE' });
      fetchGrants(idpId);
    } catch (err: any) {
      console.error(err);
      setErrorMsg(err.message || 'Failed to revoke grant.');
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
            Identity Providers
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Register the IdPs (Keycloak realms, Entra tenants, etc.) uisce trusts, and grant each one the tenant(s)
            it may claim. This is the enforced trust boundary — an unregistered issuer, or a tenant claim outside
            its grants, is rejected at authentication.
          </Typography>
        </Box>
        <Stack direction="row" spacing={2}>
          <Button variant="outlined" startIcon={<RefreshIcon />} onClick={fetchIdps}>
            Refresh
          </Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => setCreateDialogOpen(true)}>
            Register IdP
          </Button>
        </Stack>
      </Box>

      {errorMsg && <Alert severity="error">{errorMsg}</Alert>}
      {successMsg && <Alert severity="success">{successMsg}</Alert>}

      <Paper elevation={0} sx={{ p: 3, border: '1px solid', borderColor: 'divider', borderRadius: 2 }}>
        <TextField
          size="small"
          placeholder="Search by issuer or name..."
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
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }} />
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Name</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Issuer</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Status</TableCell>
              <TableCell sx={{ fontWeight: 'bold', py: 1.5 }}>Scope</TableCell>
              <TableCell align="right" sx={{ fontWeight: 'bold', py: 1.5 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredIdps.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 6 }}>
                  <Typography variant="body1" color="text.secondary">
                    No identity providers registered.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredIdps.map((idp) => (
                <React.Fragment key={idp.id}>
                  <TableRow hover>
                    <TableCell sx={{ py: 1.5 }}>
                      <IconButton size="small" onClick={() => toggleExpand(idp)}>
                        {expandedIdp === idp.id ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                      </IconButton>
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <KeyIcon fontSize="small" color="action" />
                        <Typography variant="body2" fontWeight={600}>
                          {idp.display_name || '(unnamed)'}
                        </Typography>
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Typography variant="body2" fontFamily="monospace" sx={{ wordBreak: 'break-all' }}>
                        {idp.issuer}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Chip label={idp.is_active ? 'Active' : 'Inactive'} color={idp.is_active ? 'success' : 'default'} size="small" />
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Chip label={idp.is_cross_tenant ? 'Cross-tenant' : 'Tenant-scoped'} size="small" variant="outlined" />
                    </TableCell>
                    <TableCell align="right" sx={{ py: 1.5 }}>
                      <Tooltip title="Grant to Tenant">
                        <IconButton size="small" color="primary" onClick={() => setGrantDialogIdp(idp)}>
                          <AddIcon />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Deactivate">
                        <IconButton size="small" color="error" onClick={() => handleDeactivate(idp)} disabled={actionLoading || !idp.is_active}>
                          <DeleteIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell colSpan={6} sx={{ p: 0, borderBottom: expandedIdp === idp.id ? undefined : 'none' }}>
                      <Collapse in={expandedIdp === idp.id} timeout="auto" unmountOnExit>
                        <Box sx={{ p: 2, bgcolor: 'action.hover' }}>
                          <Typography variant="subtitle2" gutterBottom>
                            Granted Tenants
                          </Typography>
                          {(grantsByIdp[idp.id] || []).length === 0 ? (
                            <Typography variant="body2" color="text.secondary">
                              No tenants granted yet — tokens from this issuer will be rejected until at least one is added.
                            </Typography>
                          ) : (
                            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                              {(grantsByIdp[idp.id] || []).map((g) => (
                                <Chip
                                  key={g.id}
                                  label={g.tenant_name}
                                  onDelete={() => handleRevoke(idp.id, g.tenant_id)}
                                  size="small"
                                />
                              ))}
                            </Stack>
                          )}
                        </Box>
                      </Collapse>
                    </TableCell>
                  </TableRow>
                </React.Fragment>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 'bold' }}>Register Identity Provider</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 3 }}>
            <TextField
              label="Display Name"
              placeholder="tenant123 Entra ID"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              size="small"
              fullWidth
            />
            <TextField
              label="Issuer URL"
              placeholder="https://login.microsoftonline.com/{tenant-guid}/v2.0"
              value={issuer}
              onChange={(e) => setIssuer(e.target.value)}
              helperText="Must exactly match the token's iss claim."
              size="small"
              fullWidth
            />
            <TextField
              label="JWKS URI"
              placeholder="https://.../discovery/v2.0/keys"
              value={jwksUri}
              onChange={(e) => setJwksUri(e.target.value)}
              size="small"
              fullWidth
            />
            <FormControlLabel
              control={<Switch checked={isCrossTenant} onChange={(e) => setIsCrossTenant(e.target.checked)} />}
              label="Cross-tenant issuer (e.g. a super-admin or professional-services realm serving multiple tenants)"
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreate} disabled={actionLoading || !issuer.trim() || !jwksUri.trim()}>
            Register
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!grantDialogIdp} onClose={() => setGrantDialogIdp(null)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 'bold' }}>Grant Tenant Access</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Allow <strong>{grantDialogIdp?.display_name || grantDialogIdp?.issuer}</strong> to issue tokens claiming this
              tenant.
            </Typography>
            <FormControl fullWidth size="small">
              <InputLabel>Tenant</InputLabel>
              <Select value={grantTenantId} label="Tenant" onChange={(e) => setGrantTenantId(e.target.value)}>
                {accessibleTenants.map((t) => (
                  <MenuItem key={t.id} value={t.id}>
                    {t.display_name || t.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setGrantDialogIdp(null)}>Cancel</Button>
          <Button variant="contained" onClick={handleGrant} disabled={actionLoading || !grantTenantId}>
            Grant
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default IdentityProvidersPage;
