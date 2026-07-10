import React, { useEffect, useState, useCallback } from "react";
import {
  Alert,
  Avatar,
  AvatarGroup,
  Box,
  Breadcrumbs,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  IconButton,
  InputAdornment,
  Link as MuiLink,
  MenuItem,
  Paper,
  Snackbar,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
  useTheme,
  alpha,
} from "@mui/material";
import {
  Add as AddIcon,
  ArrowForward as ArrowForwardIcon,
  CheckCircle as CheckCircleIcon,
  Close as CloseIcon,
  ContentCopy as ContentCopyIcon,
  Extension as ExtensionIcon,
  Groups as GroupsIcon,
  Lock as LockIcon,
  PersonOutline as PersonOutlineIcon,
  Refresh as RefreshIcon,
  Search as SearchIcon,
  Security as SecurityIcon,
  Shield as ShieldIcon,
  Storage as StorageIcon,
  Tune as TuneIcon,
  VpnKey as VpnKeyIcon,
} from "@mui/icons-material";
import { studioApi, StudioApiError } from "./studioApi";
import type { BackendSecurityProfile, CloneProfileRequest, IdpBroker, SystemProfile, TenantProfile } from "./types";

export const ProfilesDashboard: React.FC = () => {
  const theme = useTheme();
  const [tenantProfiles, setTenantProfiles] = useState<TenantProfile[]>([]);
  const [systemProfiles, setSystemProfiles] = useState<SystemProfile[]>([]);
  const [idpBrokers, setIdpBrokers] = useState<IdpBroker[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [cloneOpen, setCloneOpen] = useState(false);
  const [cloneSource, setCloneSource] = useState<string>("");
  const [cloneTarget, setCloneTarget] = useState<string>("");
  const [cloneTargetName, setCloneTargetName] = useState<string>("");
  const [idpOpen, setIdpOpen] = useState(false);
  const [idpAlias, setIdpAlias] = useState<string>("");
  const [idpProvider, setIdpProvider] = useState<string>("oidc");
  const [targetRealm] = useState<string>("uisce");
  const [snack, setSnack] = useState<{ open: boolean; severity: "success" | "error" | "info"; message: string }>({
    open: false,
    severity: "info",
    message: "",
  });

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await studioApi.listProfiles();
      const raw: BackendSecurityProfile[] = data.profiles || [];
      const tenant: TenantProfile[] = raw
        .filter((p) => p.tenant_id !== null)
        .map((p) => ({
          profileId: p.profile_id,
          profileKey: p.profile_key,
          profileName: p.profile_name,
          origin: "tenant" as const,
          ruleCount: 0,
          parentProfileKey: p.parent_profile_id ?? null,
          updatedAt: p.updated_at,
        }));
      const system: SystemProfile[] = raw
        .filter((p) => p.tenant_id === null)
        .map((p) => ({
          profileKey: p.profile_key,
          profileName: p.profile_name,
          origin: "system" as const,
          ruleCount: 0,
        }));
      setTenantProfiles(tenant);
      setSystemProfiles(system);
      setTenantProfiles(tenant);
      setSystemProfiles(system.map((p) => ({ profileKey: p.profileKey, profileName: p.profileName, origin: "system" as const, ruleCount: 0 })));
      try {
        const brokers = await studioApi.listIdpBrokers(targetRealm);
        setIdpBrokers(brokers.brokers);
      } catch {
        setIdpBrokers([]);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [targetRealm]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleClone = async () => {
    setError(null);
    try {
      const req: CloneProfileRequest = {
        sourceProfileKey: cloneSource,
        targetProfileKey: cloneTarget,
        targetProfileName: cloneTargetName,
      };
      const result = await studioApi.cloneProfile(req);
      setCloneOpen(false);
      setCloneSource("");
      setCloneTarget("");
      setCloneTargetName("");
      setSnack({
        open: true,
        severity: "success",
        message: `Cloned ${result.clonedRulesCount} rules from ${result.sourceProfileKey} → ${result.profileKey}`,
      });
      await refresh();
    } catch (e) {
      const msg = e instanceof StudioApiError ? `${e.message}: ${e.detail || ""}` : String(e);
      setError(msg);
      setSnack({ open: true, severity: "error", message: msg });
    }
  };

  const handleRegisterIdp = async () => {
    setError(null);
    try {
      await studioApi.registerIdpBroker(targetRealm, idpAlias, idpProvider);
      setIdpOpen(false);
      setIdpAlias("");
      setSnack({ open: true, severity: "success", message: `Registered IdP broker "${idpAlias}"` });
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
      setSnack({ open: true, severity: "error", message: msg });
    }
  };

  const openCloneFor = (profileKey: string) => {
    setCloneSource(profileKey);
    setCloneTarget("");
    setCloneTargetName("");
    setCloneOpen(true);
  };

  const totalSystemRules = systemProfiles.reduce((acc, p) => acc + p.ruleCount, 0);
  const totalTenantRules = tenantProfiles.reduce((acc, p) => acc + p.ruleCount, 0);
  const activeBrokers = idpBrokers.filter((b) => b.enabled).length;

  const filteredSystem = systemProfiles.filter(
    (p) =>
      !search ||
      p.profileName.toLowerCase().includes(search.toLowerCase()) ||
      p.profileKey.toLowerCase().includes(search.toLowerCase())
  );
  const filteredTenant = tenantProfiles.filter(
    (p) =>
      !search ||
      p.profileName.toLowerCase().includes(search.toLowerCase()) ||
      p.profileKey.toLowerCase().includes(search.toLowerCase())
  );

  const StatCard: React.FC<{
    label: string;
    value: number | string;
    icon: React.ReactNode;
    color: string;
    helper?: string;
  }> = ({ label, value, icon, color, helper }) => (
    <Card
      elevation={0}
      sx={{
        border: 1,
        borderColor: "divider",
        borderRadius: 2,
        height: "100%",
        transition: "border-color 0.15s ease, transform 0.15s ease",
        "&:hover": { borderColor: color, transform: "translateY(-1px)" },
      }}
    >
      <CardContent sx={{ p: 2.5 }}>
        <Stack direction="row" spacing={2} alignItems="flex-start">
          <Box
            sx={{
              width: 44,
              height: 44,
              borderRadius: 1.5,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              bgcolor: alpha(color, 0.12),
              color: color,
            }}
          >
            {icon}
          </Box>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, textTransform: "uppercase", letterSpacing: 0.5 }}>
              {label}
            </Typography>
            <Typography variant="h4" fontWeight={800} sx={{ mt: 0.5, lineHeight: 1.1 }}>
              {value}
            </Typography>
            {helper && (
              <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: "block" }}>
                {helper}
              </Typography>
            )}
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1600, mx: "auto" }}>
      <Breadcrumbs sx={{ mb: 1.5 }}>
        <MuiLink underline="hover" color="text.secondary" href="/fabric/tenants" sx={{ fontSize: 13 }}>
          Platform
        </MuiLink>
        <Typography variant="body2" sx={{ fontSize: 13 }} color="text.primary">
          Entitlement Management
        </Typography>
      </Breadcrumbs>

      <Stack direction={{ xs: "column", md: "row" }} spacing={2} alignItems={{ md: "flex-end" }} sx={{ mb: 3 }}>
        <Box sx={{ flex: 1 }}>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), color: "primary.main", width: 40, height: 40 }}>
              <ShieldIcon />
            </Avatar>
            <Typography variant="h4" fontWeight={800}>
              Entitlement Management
            </Typography>
          </Stack>
          <Typography variant="body1" color="text.secondary">
            Define tenant profiles, link corporate identity providers, and govern ABAC overrides.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1}>
          <Tooltip title="Refresh">
            <IconButton onClick={refresh} disabled={loading} size="small">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setCloneOpen(true)}
            disabled={systemProfiles.length === 0}
          >
            Clone Profile
          </Button>
        </Stack>
      </Stack>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="System Profiles"
            value={systemProfiles.length}
            icon={<LockIcon />}
            color={theme.palette.primary.main}
            helper={`${totalSystemRules} baseline rules`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Custom Profiles"
            value={tenantProfiles.length}
            icon={<PersonOutlineIcon />}
            color={theme.palette.info.main}
            helper={`${totalTenantRules} tenant rules`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="IdP Brokers"
            value={idpBrokers.length}
            icon={<VpnKeyIcon />}
            color={theme.palette.success.main}
            helper={`${activeBrokers} active`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Identity Mappings"
            value="—"
            icon={<GroupsIcon />}
            color={theme.palette.warning.main}
            helper="Configured in catalog"
          />
        </Grid>
      </Grid>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Paper elevation={0} sx={{ border: 1, borderColor: "divider", borderRadius: 2, mb: 3 }}>
        <Box sx={{ p: 2.5, pb: 2 }}>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Avatar sx={{ bgcolor: alpha(theme.palette.success.main, 0.12), color: "success.main", width: 32, height: 32 }}>
              <VpnKeyIcon fontSize="small" />
            </Avatar>
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Corporate Identity Provider Link
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Bring your own IdP — connect Azure AD, Okta, Ping, or Google to realm <strong>{targetRealm}</strong>.
              </Typography>
            </Box>
          </Stack>
        </Box>
        <Divider />
        <Box sx={{ p: 2.5 }}>
          {idpBrokers.length === 0 ? (
            <Box
              sx={{
                py: 5,
                px: 3,
                textAlign: "center",
                borderRadius: 1.5,
                border: 1,
                borderStyle: "dashed",
                borderColor: "divider",
                bgcolor: alpha(theme.palette.success.main, 0.04),
              }}
            >
              <VpnKeyIcon sx={{ fontSize: 40, color: "text.disabled", mb: 1 }} />
              <Typography variant="body1" color="text.secondary" sx={{ mb: 0.5 }}>
                No IdP brokers linked yet
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: "block", mb: 2 }}>
                Connect your corporate identity provider to enable SSO and AD group mapping.
              </Typography>
              <Button variant="outlined" startIcon={<AddIcon />} onClick={() => setIdpOpen(true)}>
                Register IdP
              </Button>
            </Box>
          ) : (
            <Grid container spacing={1.5}>
              {idpBrokers.map((b) => (
                <Grid size={{ xs: 12, sm: 6, md: 4 }} key={b.alias}>
                  <Card variant="outlined" sx={{ p: 2 }}>
                    <Stack direction="row" alignItems="center" spacing={1.5}>
                      <Avatar sx={{ bgcolor: alpha(theme.palette.success.main, 0.12), color: "success.main", width: 36, height: 36 }}>
                        <SecurityIcon fontSize="small" />
                      </Avatar>
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        <Typography variant="subtitle2" fontWeight={700} noWrap>
                          {b.alias}
                        </Typography>
                        <Typography variant="caption" color="text.secondary" noWrap>
                          {b.providerId} · {b.linkedRealm}
                        </Typography>
                      </Box>
                      <Chip
                        size="small"
                        label={b.enabled ? "Active" : "Disabled"}
                        color={b.enabled ? "success" : "default"}
                        variant={b.enabled ? "filled" : "outlined"}
                      />
                    </Stack>
                  </Card>
                </Grid>
              ))}
              <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                <Card
                  variant="outlined"
                  sx={{
                    p: 2,
                    borderStyle: "dashed",
                    cursor: "pointer",
                    transition: "all 0.15s",
                    "&:hover": { borderColor: "primary.main", bgcolor: alpha(theme.palette.primary.main, 0.04) },
                  }}
                  onClick={() => setIdpOpen(true)}
                >
                  <Stack direction="row" alignItems="center" spacing={1.5} justifyContent="center">
                    <AddIcon color="primary" />
                    <Typography variant="subtitle2" color="primary" fontWeight={700}>
                      Register another IdP
                    </Typography>
                  </Stack>
                </Card>
              </Grid>
            </Grid>
          )}
        </Box>
      </Paper>

      <Paper elevation={0} sx={{ border: 1, borderColor: "divider", borderRadius: 2, mb: 3 }}>
        <Box sx={{ p: 2.5, pb: 2 }}>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Avatar sx={{ bgcolor: alpha(theme.palette.warning.main, 0.12), color: "warning.main", width: 32, height: 32 }}>
              <GroupsIcon fontSize="small" />
            </Avatar>
            <Box sx={{ flex: 1 }}>
              <Typography variant="h6" fontWeight={700}>
                AD Group Mapping
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Configure corporate group → internal profile mappings via the catalog.
              </Typography>
            </Box>
          </Stack>
        </Box>
        <Divider />
        <Box sx={{ p: 2.5 }}>
          <Alert severity="info" variant="outlined" icon={<StorageIcon />}>
            Mappings are stored in <code>security.identity_profile_mappings</code>. Per{" "}
            <strong>Cardinal Rule 7.4</strong>, group IDs are passed through verbatim from the IdP — resolution
            is the database plane, not the application.
          </Alert>
        </Box>
      </Paper>

      <Paper elevation={0} sx={{ border: 1, borderColor: "divider", borderRadius: 2 }}>
        <Box sx={{ p: 2.5, pb: 2 }}>
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
            <Stack direction="row" spacing={1.5} alignItems="center">
              <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), color: "primary.main", width: 32, height: 32 }}>
                <ExtensionIcon fontSize="small" />
              </Avatar>
              <Box>
                <Typography variant="h6" fontWeight={700}>
                  Platform Profile Register
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  System blueprints and tenant-scoped custom profiles. Click a row to customize.
                </Typography>
              </Box>
            </Stack>
            <TextField
              size="small"
              placeholder="Search profiles…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" />
                  </InputAdornment>
                ),
              }}
              sx={{ minWidth: 240 }}
            />
          </Stack>

          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Chip
              icon={<LockIcon />}
              label={`System · ${systemProfiles.length}`}
              size="small"
              color="primary"
              variant="outlined"
            />
            <Chip
              icon={<PersonOutlineIcon />}
              label={`Custom · ${tenantProfiles.length}`}
              size="small"
              color="info"
              variant="outlined"
            />
            <Box sx={{ flex: 1 }} />
            <AvatarGroup max={4} sx={{ "& .MuiAvatar-root": { width: 24, height: 24, fontSize: 12 } }}>
              {[...systemProfiles, ...tenantProfiles].slice(0, 4).map((p) => (
                <Avatar key={p.profileKey}>{p.profileName.charAt(0)}</Avatar>
              ))}
            </AvatarGroup>
          </Stack>
        </Box>
        <Divider />
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: theme.palette.mode === "dark" ? "rgba(0,0,0,0.3)" : "grey.50" }}>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Profile</TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Type</TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Status</TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>
                  Rules
                </TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>
                  Actions
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                    <CircularProgress size={28} />
                  </TableCell>
                </TableRow>
              ) : filteredSystem.length === 0 && filteredTenant.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                    <Typography color="text.secondary">No profiles match "{search}"</Typography>
                  </TableCell>
                </TableRow>
              ) : (
                <>
                  {filteredSystem.map((p) => (
                    <TableRow
                      key={p.profileKey}
                      hover
                      sx={{ "&:last-of-type td": { borderBottom: 0 }, opacity: 0.95 }}
                    >
                      <TableCell>
                        <Stack direction="row" spacing={1.5} alignItems="center">
                          <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), width: 32, height: 32 }}>
                            <LockIcon fontSize="small" color="primary" />
                          </Avatar>
                          <Box>
                            <Typography variant="body2" fontWeight={600}>
                              {p.profileName}
                            </Typography>
                            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
                              {p.profileKey}
                            </Typography>
                          </Box>
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Chip label="SYSTEM" size="small" color="primary" variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Chip
                          label="Immutable Baseline"
                          size="small"
                          icon={<ShieldIcon style={{ fontSize: 14 }} />}
                          sx={{ bgcolor: alpha(theme.palette.primary.main, 0.08) }}
                        />
                      </TableCell>
                      <TableCell align="right">
                        <Typography variant="body2" fontWeight={600}>
                          {p.ruleCount}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="Clone to a custom tenant profile">
                          <Button
                            size="small"
                            variant="outlined"
                            startIcon={<ContentCopyIcon />}
                            onClick={() => openCloneFor(p.profileKey)}
                          >
                            Clone
                          </Button>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                  {filteredTenant.map((p) => (
                    <TableRow
                      key={p.profileId}
                      hover
                      sx={{ "&:last-of-type td": { borderBottom: 0 } }}
                    >
                      <TableCell>
                        <Stack direction="row" spacing={1.5} alignItems="center">
                          <Avatar sx={{ bgcolor: alpha(theme.palette.info.main, 0.12), width: 32, height: 32 }}>
                            <PersonOutlineIcon fontSize="small" color="info" />
                          </Avatar>
                          <Box>
                            <Typography variant="body2" fontWeight={600}>
                              {p.profileName}
                            </Typography>
                            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
                              {p.profileKey}
                            </Typography>
                          </Box>
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Chip label="CUSTOM" size="small" color="info" variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Chip
                          label="Active"
                          size="small"
                          color="success"
                          icon={<CheckCircleIcon style={{ fontSize: 14 }} />}
                        />
                      </TableCell>
                      <TableCell align="right">
                        <Typography variant="body2" fontWeight={600}>
                          {p.ruleCount}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                          <Tooltip title="Open functional scope matrix">
                            <Button
                              size="small"
                              variant="text"
                              startIcon={<TuneIcon />}
                              href={`/admin/entitlements/profiles/${p.profileKey}/components`}
                            >
                              Components
                            </Button>
                          </Tooltip>
                          <Tooltip title="Open profile customizer">
                            <Button
                              size="small"
                              variant="outlined"
                              startIcon={<ExtensionIcon />}
                              href={`/admin/entitlements/profiles/${p.profileKey}`}
                            >
                              Customize
                            </Button>
                          </Tooltip>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))}
                </>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      <Dialog open={cloneOpen} onClose={() => setCloneOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), width: 32, height: 32 }}>
              <ContentCopyIcon color="primary" fontSize="small" />
            </Avatar>
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Clone System Profile
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Create a tenant-scoped custom profile from a system baseline.
              </Typography>
            </Box>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ pt: 1 }}>
            <TextField
              select
              label="Source Profile"
              value={cloneSource}
              onChange={(e) => setCloneSource(e.target.value)}
              fullWidth
              required
            >
              {systemProfiles.map((p) => (
                <MenuItem key={p.profileKey} value={p.profileKey}>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ width: "100%" }}>
                    <LockIcon fontSize="small" color="primary" />
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="body2">{p.profileName}</Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
                        {p.profileKey} · {p.ruleCount} rules
                      </Typography>
                    </Box>
                  </Stack>
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Target Profile Key"
              value={cloneTarget}
              onChange={(e) => setCloneTarget(e.target.value)}
              placeholder="inv_senior_analyst"
              fullWidth
              required
              helperText="Lowercase, underscores only — used in API and condition DSL"
            />
            <TextField
              label="Target Profile Name"
              value={cloneTargetName}
              onChange={(e) => setCloneTargetName(e.target.value)}
              placeholder="InvestCo Senior Analyst"
              fullWidth
            />
            <Alert severity="info" variant="outlined" icon={<ShieldIcon />}>
              Cloned profiles inherit all system rules. You can then add tenant-scoped overrides in the Profile
              Customizer.
            </Alert>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setCloneOpen(false)} startIcon={<CloseIcon />}>
            Cancel
          </Button>
          <Button
            onClick={handleClone}
            variant="contained"
            disabled={!cloneSource || !cloneTarget || !cloneTargetName}
            startIcon={<ContentCopyIcon />}
          >
            Clone Profile
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={idpOpen} onClose={() => setIdpOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar sx={{ bgcolor: alpha(theme.palette.success.main, 0.12), width: 32, height: 32 }}>
              <VpnKeyIcon color="success" fontSize="small" />
            </Avatar>
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Register Identity Provider
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Link a corporate IdP to realm <strong>{targetRealm}</strong>.
              </Typography>
            </Box>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ pt: 1 }}>
            <TextField
              label="Alias"
              value={idpAlias}
              onChange={(e) => setIdpAlias(e.target.value)}
              placeholder="acme-oidc"
              fullWidth
              required
              helperText="Unique identifier within the realm"
            />
            <TextField
              select
              label="Provider Type"
              value={idpProvider}
              onChange={(e) => setIdpProvider(e.target.value)}
              fullWidth
            >
              <MenuItem value="oidc">OpenID Connect</MenuItem>
              <MenuItem value="saml">SAML 2.0</MenuItem>
              <MenuItem value="google">Google Workspace</MenuItem>
            </TextField>
            <Alert severity="warning" variant="outlined">
              After registration, configure the IdP metadata (client ID, secret, redirect URI) via the realm's
              identity provider console.
            </Alert>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setIdpOpen(false)} startIcon={<CloseIcon />}>
            Cancel
          </Button>
          <Button
            onClick={handleRegisterIdp}
            variant="contained"
            disabled={!idpAlias || !idpProvider}
            startIcon={<ArrowForwardIcon />}
          >
            Register Broker
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={snack.open}
        autoHideDuration={5000}
        onClose={() => setSnack((s) => ({ ...s, open: false }))}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={() => setSnack((s) => ({ ...s, open: false }))}
          severity={snack.severity}
          variant="filled"
          sx={{ minWidth: 320 }}
        >
          {snack.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default ProfilesDashboard;