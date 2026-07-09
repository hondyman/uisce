import React, { useEffect, useState, useCallback } from "react";
import {
  Alert,
  Avatar,
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
  ArrowBack as ArrowBackIcon,
  Block as BlockIcon,
  CheckCircle as CheckCircleIcon,
  Close as CloseIcon,
  Code as CodeIcon,
  Extension as ExtensionIcon,
  Lock as LockIcon,
  PersonOutline as PersonOutlineIcon,
  Refresh as RefreshIcon,
  Save as SaveIcon,
  Security as SecurityIcon,
} from "@mui/icons-material";
import { studioApi, StudioApiError } from "./studioApi";
import type {
  AbacPolicy,
  AppendPolicyOverrideRequest,
  PolicyEffect,
} from "./types";

interface ProfileCustomizerProps {
  profileKey: string;
  isSystem: boolean;
}

export const ProfileCustomizer: React.FC<ProfileCustomizerProps> = ({ profileKey, isSystem }) => {
  const theme = useTheme();
  const [policies, setPolicies] = useState<AbacPolicy[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [adding, setAdding] = useState(false);
  const [action, setAction] = useState<string>("");
  const [effect, setEffect] = useState<PolicyEffect>("deny");
  const [priority, setPriority] = useState<number>(10);
  const [condition, setCondition] = useState<string>("");
  const [name, setName] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [snack, setSnack] = useState<{ open: boolean; severity: "success" | "error" | "info"; message: string }>({
    open: false,
    severity: "info",
    message: "",
  });

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(
        `${import.meta.env.VITE_API_URL || "http://localhost:8082/api"}/v1/tenant/policies?target_profile_key=${encodeURIComponent(profileKey)}`,
        { credentials: "include" }
      );
      if (res.ok) {
        const data = await res.json();
        setPolicies(data.policies || []);
      } else {
        setPolicies([]);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [profileKey]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleAdd = async () => {
    setError(null);
    try {
      const req: AppendPolicyOverrideRequest = {
        targetProfileKey: profileKey,
        actionAttribute: action,
        effect,
        priorityRank: priority,
        conditionDsl: condition || undefined,
        name: name || `Tenant override: ${action} on ${profileKey}`,
        description: description || undefined,
      };
      const result = await studioApi.appendPolicyOverride(req);
      setAdding(false);
      setAction("");
      setCondition("");
      setName("");
      setDescription("");
      setSnack({
        open: true,
        severity: "success",
        message: `Rule appended (priority ${result.priorityRank})`,
      });
      await refresh();
    } catch (e) {
      const msg = e instanceof StudioApiError ? `${e.message}: ${e.detail || ""}` : String(e);
      setError(msg);
      setSnack({ open: true, severity: "error", message: msg });
    }
  };

  const systemPolicies = policies.filter((p) => p.origin === "system");
  const tenantPolicies = policies.filter((p) => p.origin === "tenant");
  const totalPolicies = policies.length;

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

  const PolicyRow: React.FC<{ policy: AbacPolicy }> = ({ policy }) => (
    <TableRow hover sx={{ "&:last-of-type td": { borderBottom: 0 } }}>
      <TableCell>
        <Chip
          size="small"
          icon={policy.origin === "system" ? <LockIcon style={{ fontSize: 14 }} /> : <PersonOutlineIcon style={{ fontSize: 14 }} />}
          label={policy.origin === "system" ? "SYSTEM" : "TENANT"}
          color={policy.origin === "system" ? "primary" : "info"}
          variant={policy.origin === "system" ? "filled" : "outlined"}
        />
      </TableCell>
      <TableCell>
        <Stack direction="row" spacing={0.5} alignItems="center">
          <CodeIcon fontSize="small" color="action" />
          <Typography variant="body2" sx={{ fontFamily: "monospace", fontWeight: 600 }}>
            {policy.actionAttribute}
          </Typography>
        </Stack>
        <Typography variant="caption" color="text.secondary" sx={{ display: "block", mt: 0.5 }}>
          {policy.name}
        </Typography>
      </TableCell>
      <TableCell sx={{ maxWidth: 320 }}>
        {policy.conditionDsl ? (
          <Box
            sx={{
              fontFamily: "monospace",
              fontSize: 12,
              p: 0.75,
              borderRadius: 1,
              bgcolor: theme.palette.mode === "dark" ? "rgba(255,255,255,0.05)" : "grey.100",
              border: 1,
              borderColor: "divider",
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
            }}
          >
            {policy.conditionDsl}
          </Box>
        ) : (
          <Typography variant="caption" color="text.disabled" fontStyle="italic">
            no condition — applies always
          </Typography>
        )}
      </TableCell>
      <TableCell>
        {policy.effect === "allow" ? (
          <Chip
            size="small"
            color="success"
            icon={<CheckCircleIcon style={{ fontSize: 14 }} />}
            label="ALLOW"
          />
        ) : (
          <Chip
            size="small"
            color="error"
            icon={<BlockIcon style={{ fontSize: 14 }} />}
            label="DENY"
          />
        )}
      </TableCell>
      <TableCell align="right">
        <Tooltip
          title={
            policy.priorityRank <= 10
              ? "Highest override precedence"
              : policy.priorityRank <= 20
                ? "Standard precedence"
                : "Default precedence"
          }
        >
          <Chip
            size="small"
            label={policy.priorityRank}
            sx={{ fontWeight: 700, minWidth: 48, fontFamily: "monospace" }}
            color={policy.priorityRank <= 10 ? "warning" : "default"}
            variant={policy.priorityRank <= 10 ? "filled" : "outlined"}
          />
        </Tooltip>
      </TableCell>
    </TableRow>
  );

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1600, mx: "auto" }}>
      <Breadcrumbs sx={{ mb: 1.5 }}>
        <MuiLink underline="hover" color="text.secondary" href="/admin/entitlements" sx={{ fontSize: 13 }}>
          Entitlement Management
        </MuiLink>
        <Typography variant="body2" sx={{ fontSize: 13 }} color="text.primary">
          {profileKey}
        </Typography>
      </Breadcrumbs>

      <Stack direction={{ xs: "column", md: "row" }} spacing={2} alignItems={{ md: "flex-end" }} sx={{ mb: 3 }}>
        <Box sx={{ flex: 1 }}>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Button
              size="small"
              variant="text"
              startIcon={<ArrowBackIcon />}
              href="/admin/entitlements"
              sx={{ mr: 0.5 }}
            >
              All profiles
            </Button>
          </Stack>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Avatar
              sx={{
                bgcolor: alpha(isSystem ? theme.palette.primary.main : theme.palette.info.main, 0.12),
                color: isSystem ? "primary.main" : "info.main",
                width: 40,
                height: 40,
              }}
            >
              {isSystem ? <LockIcon /> : <PersonOutlineIcon />}
            </Avatar>
            <Box>
              <Typography variant="h4" fontWeight={800}>
                {profileKey}
              </Typography>
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.25 }}>
                <Chip
                  size="small"
                  label={isSystem ? "System Blueprint" : "Custom Tenant Profile"}
                  color={isSystem ? "primary" : "info"}
                  variant="outlined"
                />
                {isSystem && (
                  <Typography variant="caption" color="text.secondary">
                    Extends the immutable Platform Core Blueprint
                  </Typography>
                )}
              </Stack>
            </Box>
          </Stack>
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
            onClick={() => setAdding(true)}
            disabled={isSystem && systemPolicies.length === 0}
          >
            Append Rule
          </Button>
        </Stack>
      </Stack>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Total Rules"
            value={totalPolicies}
            icon={<SecurityIcon />}
            color={theme.palette.primary.main}
            helper={`${systemPolicies.length} system + ${tenantPolicies.length} tenant`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="System Rules"
            value={systemPolicies.length}
            icon={<LockIcon />}
            color={theme.palette.primary.main}
            helper="Inherited from blueprint"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Tenant Overrides"
            value={tenantPolicies.length}
            icon={<ExtensionIcon />}
            color={theme.palette.info.main}
            helper="Tenant-scoped rules"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Deny Rules"
            value={policies.filter((p) => p.effect === "deny").length}
            icon={<BlockIcon />}
            color={theme.palette.error.main}
            helper="Deny-overrides enforced"
          />
        </Grid>
      </Grid>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Paper elevation={0} sx={{ border: 1, borderColor: "divider", borderRadius: 2 }}>
        <Box sx={{ p: 2.5, pb: 2 }}>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), color: "primary.main", width: 32, height: 32 }}>
              <SecurityIcon fontSize="small" />
            </Avatar>
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Shadowing Engine Matrix
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Lower priority rank wins. Tenant overrides shadow system baselines at rank ≤ 20.
              </Typography>
            </Box>
          </Stack>
        </Box>
        <Divider />
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: theme.palette.mode === "dark" ? "rgba(0,0,0,0.3)" : "grey.50" }}>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Origin</TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Action</TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Condition DSL</TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>Effect</TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>
                  Priority
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
              ) : policies.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 6 }}>
                    <SecurityIcon sx={{ fontSize: 40, color: "text.disabled", mb: 1 }} />
                    <Typography variant="body1" color="text.secondary">
                      No ABAC rules yet for this profile
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Click "Append Rule" to add a tenant override
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                <>
                  {systemPolicies.map((p) => (
                    <PolicyRow key={p.policyId} policy={p} />
                  ))}
                  {tenantPolicies.map((p) => (
                    <PolicyRow key={p.policyId} policy={p} />
                  ))}
                </>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      <Dialog open={adding} onClose={() => setAdding(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), width: 32, height: 32 }}>
              <SecurityIcon color="primary" fontSize="small" />
            </Avatar>
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Policy Composer
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Append a tenant-scoped ABAC rule to <code>{profileKey}</code>.
              </Typography>
            </Box>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ pt: 1 }}>
            <Grid container spacing={2}>
              <Grid size={{ xs: 12, md: 8 }}>
                <TextField
                  label="Action Attribute"
                  value={action}
                  onChange={(e) => setAction(e.target.value)}
                  placeholder="read_ledger_data"
                  fullWidth
                  required
                  helperText="The capability or action being gated"
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <CodeIcon fontSize="small" color="action" />
                      </InputAdornment>
                    ),
                  }}
                />
              </Grid>
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  select
                  label="Effect"
                  value={effect}
                  onChange={(e) => setEffect(e.target.value as PolicyEffect)}
                  fullWidth
                  required
                >
                  <MenuItem value="deny">
                    <Stack direction="row" spacing={1} alignItems="center">
                      <BlockIcon fontSize="small" color="error" />
                      <Box>
                        <Typography variant="body2" fontWeight={600}>DENY</Typography>
                        <Typography variant="caption" color="text.secondary">
                          Deny-overrides enforced
                        </Typography>
                      </Box>
                    </Stack>
                  </MenuItem>
                  <MenuItem value="allow">
                    <Stack direction="row" spacing={1} alignItems="center">
                      <CheckCircleIcon fontSize="small" color="success" />
                      <Box>
                        <Typography variant="body2" fontWeight={600}>ALLOW</Typography>
                        <Typography variant="caption" color="text.secondary">
                          Explicit allow
                        </Typography>
                      </Box>
                    </Stack>
                  </MenuItem>
                </TextField>
              </Grid>
            </Grid>

            <TextField
              select
              label="Priority Rank"
              value={priority}
              onChange={(e) => setPriority(Number(e.target.value))}
              fullWidth
              helperText="Lower number = higher precedence. Rank ≤ 20 shadows system baselines."
            >
              <MenuItem value={10}>10 — Highest Override Precedence</MenuItem>
              <MenuItem value={20}>20 — Shadows System Baselines</MenuItem>
              <MenuItem value={50}>50 — Default Precedence</MenuItem>
            </TextField>

            <TextField
              label="Condition DSL"
              value={condition}
              onChange={(e) => setCondition(e.target.value)}
              placeholder="resource.bo_key == 'sensitive_cash_gl'"
              fullWidth
              multiline
              minRows={2}
              helperText="Optional — leave blank to apply unconditionally"
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start" sx={{ alignItems: "flex-start", mt: 1 }}>
                    <CodeIcon fontSize="small" color="action" />
                  </InputAdornment>
                ),
              }}
            />

            <Grid container spacing={2}>
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  label="Rule Name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Block ledger access for sensitive GL"
                  fullWidth
                />
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <TextField
                  label="Description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Optional"
                  fullWidth
                />
              </Grid>
            </Grid>

            <Alert severity="warning" variant="outlined">
              DENY rules at priority ≤ 20 will shadow matching system ALLOW rules. Review the shadowing matrix
              below before applying.
            </Alert>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setAdding(false)} startIcon={<CloseIcon />}>
            Cancel
          </Button>
          <Button
            onClick={handleAdd}
            variant="contained"
            disabled={!action || !effect}
            startIcon={<SaveIcon />}
          >
            Apply Rule
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

export default ProfileCustomizer;