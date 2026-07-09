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
  Paper,
  Snackbar,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tab,
  Tabs,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
  useTheme,
  alpha,
} from "@mui/material";
import {
  AccountTree as AccountTreeIcon,
  Api as ApiIcon,
  ArrowBack as ArrowBackIcon,
  Block as BlockIcon,
  Category as CategoryIcon,
  CheckCircle as CheckCircleIcon,
  Close as CloseIcon,
  Code as CodeIcon,
  Edit as EditIcon,
  Extension as ExtensionIcon,
  Lock as LockIcon,
  PersonOutline as PersonOutlineIcon,
  Refresh as RefreshIcon,
  Save as SaveIcon,
  Security as SecurityIcon,
  Verified as VerifiedIcon,
} from "@mui/icons-material";
import { studioApi, StudioApiError } from "./studioApi";
import type {
  ComponentEntitlement,
  EntitlementType,
  OverrideState,
} from "./types";

interface EntitlementMatrixProps {
  profileKey: string;
  isCustom: boolean;
}

const PLATFORM_NODES: Record<EntitlementType, Array<{ path: string; label: string; baseline: OverrideState }>> = {
  MENU_PAGE: [
    { path: "gl.trial_balance", label: "Trial Balance Workspace", baseline: "INHERIT_BASELINE" },
    { path: "gl.fee_accrual", label: "Fee Accrual Parameters", baseline: "INHERIT_BASELINE" },
    { path: "trade_ops.manual_allocation", label: "Manual Allocation Trigger", baseline: "INHERIT_BASELINE" },
    { path: "trade_ops.block_settlement", label: "Block-Trade Settlement Lock", baseline: "INHERIT_BASELINE" },
  ],
  WORKFLOW_STEP: [
    { path: "trade_ops.workflow.allocate", label: "Allocate Step", baseline: "INHERIT_BASELINE" },
    { path: "trade_ops.workflow.clear", label: "Clear Step", baseline: "INHERIT_BASELINE" },
  ],
  PUBLIC_API: [
    { path: "/api/v1/public/trades/fetch", label: "GET /trades/fetch", baseline: "INHERIT_BASELINE" },
    { path: "/api/v1/public/orders/submit", label: "POST /orders/submit", baseline: "INHERIT_BASELINE" },
  ],
};

const TAB_META: Record<EntitlementType, { label: string; icon: React.ReactNode; description: string }> = {
  MENU_PAGE: {
    label: "Pages & Menus",
    icon: <CategoryIcon fontSize="small" />,
    description: "Navigation entries and route-level gating",
  },
  WORKFLOW_STEP: {
    label: "Workflow Steps",
    icon: <AccountTreeIcon fontSize="small" />,
    description: "Steps within Temporal workflows and orchestration",
  },
  PUBLIC_API: {
    label: "Public Edge APIs",
    icon: <ApiIcon fontSize="small" />,
    description: "Anonymous-facing API routes",
  },
};

export const EntitlementMatrix: React.FC<EntitlementMatrixProps> = ({ profileKey, isCustom }) => {
  const theme = useTheme();
  const [tab, setTab] = useState<EntitlementType>("MENU_PAGE");
  const [entitlements, setEntitlements] = useState<ComponentEntitlement[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [editingNode, setEditingNode] = useState<string | null>(null);
  const [editState, setEditState] = useState<OverrideState>("INHERIT_BASELINE");
  const [editCondition, setEditCondition] = useState<string>("");
  const [snack, setSnack] = useState<{ open: boolean; severity: "success" | "error" | "info"; message: string }>({
    open: false,
    severity: "info",
    message: "",
  });

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await studioApi.listEntitlements(profileKey, tab);
      setEntitlements(res.entitlements);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
      setEntitlements([]);
    } finally {
      setLoading(false);
    }
  }, [profileKey, tab]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const stateFor = (nodePath: string): OverrideState => {
    const e = entitlements.find((x) => x.nodePath === nodePath);
    return e ? e.overrideState : "INHERIT_BASELINE";
  };

  const conditionFor = (nodePath: string): string => {
    const e = entitlements.find((x) => x.nodePath === nodePath);
    return e?.conditionalDsl || "";
  };

  const handleSave = async (nodePath: string) => {
    setError(null);
    try {
      await studioApi.upsertEntitlement({
        targetProfileKey: profileKey,
        entitlementType: tab,
        nodePath,
        overrideState: editState,
        conditionalDsl: editCondition || undefined,
      });
      setEditingNode(null);
      setEditCondition("");
      setSnack({ open: true, severity: "success", message: `Override saved for ${nodePath}` });
      await refresh();
    } catch (e) {
      const msg = e instanceof StudioApiError ? `${e.message}: ${e.detail || ""}` : String(e);
      setError(msg);
      setSnack({ open: true, severity: "error", message: msg });
    }
  };

  const stateChip = (state: OverrideState) => {
    switch (state) {
      case "EXPLICIT_ALLOW":
        return (
          <Chip
            size="small"
            color="success"
            icon={<CheckCircleIcon style={{ fontSize: 14 }} />}
            label="EXPLICIT ALLOW"
          />
        );
      case "FORCE_DENY":
        return (
          <Chip
            size="small"
            color="error"
            icon={<BlockIcon style={{ fontSize: 14 }} />}
            label="FORCE DENY"
          />
        );
      default:
        return (
          <Chip
            size="small"
            variant="outlined"
            icon={<VerifiedIcon style={{ fontSize: 14 }} />}
            label="Inherit Baseline"
          />
        );
    }
  };

  const baselineChip = (state: OverrideState) => {
    if (state === "FORCE_DENY") {
      return (
        <Chip
          size="small"
          color="error"
          icon={<BlockIcon style={{ fontSize: 14 }} />}
          label="DENY"
        />
      );
    }
    return (
      <Chip
        size="small"
        color="success"
        variant="outlined"
        icon={<CheckCircleIcon style={{ fontSize: 14 }} />}
        label="ALLOW"
      />
    );
  };

  const nodes = PLATFORM_NODES[tab];
  const overrideCount = entitlements.length;
  const denyCount = entitlements.filter((e) => e.overrideState === "FORCE_DENY").length;
  const allowCount = entitlements.filter((e) => e.overrideState === "EXPLICIT_ALLOW").length;
  const inheritCount = nodes.length - overrideCount;

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
        <MuiLink underline="hover" color="text.secondary" href="/admin/entitlements" sx={{ fontSize: 13 }}>
          Entitlement Management
        </MuiLink>
        <Typography variant="body2" sx={{ fontSize: 13 }} color="text.primary">
          {profileKey}
        </Typography>
        <Typography variant="body2" sx={{ fontSize: 13 }} color="text.secondary">
          Components
        </Typography>
      </Breadcrumbs>

      <Stack direction={{ xs: "column", md: "row" }} spacing={2} alignItems={{ md: "flex-end" }} sx={{ mb: 3 }}>
        <Box sx={{ flex: 1 }}>
          <Button
            size="small"
            variant="text"
            startIcon={<ArrowBackIcon />}
            href={`/admin/entitlements/profiles/${profileKey}`}
            sx={{ mb: 0.5 }}
          >
            Back to {profileKey}
          </Button>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Avatar
              sx={{
                bgcolor: alpha(theme.palette.primary.main, 0.12),
                color: "primary.main",
                width: 40,
                height: 40,
              }}
            >
              <ExtensionIcon />
            </Avatar>
            <Box>
              <Typography variant="h4" fontWeight={800}>
                Functional Scope Matrix
              </Typography>
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.25 }}>
                <Chip
                  size="small"
                  label={isCustom ? "Custom Tenant Profile" : "System Baseline"}
                  color={isCustom ? "info" : "primary"}
                  variant="outlined"
                  icon={isCustom ? <PersonOutlineIcon style={{ fontSize: 14 }} /> : <LockIcon style={{ fontSize: 14 }} />}
                />
                <Typography variant="caption" color="text.secondary">
                  {profileKey}
                </Typography>
              </Stack>
            </Box>
          </Stack>
        </Box>
        <Tooltip title="Refresh">
          <IconButton onClick={refresh} disabled={loading} size="small">
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Stack>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Components"
            value={nodes.length}
            icon={TAB_META[tab].icon as React.ReactElement}
            color={theme.palette.primary.main}
            helper={TAB_META[tab].label}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Overrides"
            value={overrideCount}
            icon={<SecurityIcon />}
            color={theme.palette.info.main}
            helper={`${inheritCount} inheriting baseline`}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Explicit Allow"
            value={allowCount}
            icon={<CheckCircleIcon />}
            color={theme.palette.success.main}
            helper="Grants access"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatCard
            label="Force Deny"
            value={denyCount}
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
        <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
          <Tabs
            value={tab}
            onChange={(_, v) => setTab(v)}
            sx={{
              px: 2,
              "& .MuiTab-root": { minHeight: 56, textTransform: "none", fontWeight: 600 },
            }}
          >
            {(Object.keys(TAB_META) as EntitlementType[]).map((k) => {
              const meta = TAB_META[k];
              return (
                <Tab
                  key={k}
                  value={k}
                  icon={meta.icon as React.ReactElement}
                  iconPosition="start"
                  label={meta.label}
                />
              );
            })}
          </Tabs>
        </Box>

        <Box sx={{ p: 2.5, pb: 1.5 }}>
          <Typography variant="body2" color="text.secondary">
            {TAB_META[tab].description}. Click any row to override the ambient baseline.
          </Typography>
        </Box>
        <Divider />

        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: theme.palette.mode === "dark" ? "rgba(0,0,0,0.3)" : "grey.50" }}>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>
                  Component Node
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>
                  Ambient Baseline
                </TableCell>
                <TableCell sx={{ fontWeight: 700, textTransform: "uppercase", fontSize: "0.75rem" }}>
                  Enforced Override
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={3} align="center" sx={{ py: 6 }}>
                    <CircularProgress size={28} />
                  </TableCell>
                </TableRow>
              ) : nodes.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} align="center" sx={{ py: 6 }}>
                    <Typography color="text.secondary">No components registered in this category.</Typography>
                  </TableCell>
                </TableRow>
              ) : (
                nodes.map((n) => {
                  const state = stateFor(n.path);
                  const isEditing = editingNode === n.path;
                  const isOverridden = state !== "INHERIT_BASELINE";
                  return (
                    <TableRow
                      key={n.path}
                      hover
                      sx={{
                        "&:last-of-type td": { borderBottom: 0 },
                        bgcolor: state === "FORCE_DENY"
                          ? alpha(theme.palette.error.main, 0.04)
                          : state === "EXPLICIT_ALLOW"
                            ? alpha(theme.palette.success.main, 0.04)
                            : undefined,
                      }}
                    >
                      <TableCell sx={{ maxWidth: 380 }}>
                        <Stack direction="row" spacing={1} alignItems="flex-start">
                          <CodeIcon fontSize="small" color="action" sx={{ mt: 0.25 }} />
                          <Box sx={{ minWidth: 0 }}>
                            <Typography variant="body2" sx={{ fontFamily: "monospace", fontWeight: 600 }}>
                              {n.path}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {n.label}
                            </Typography>
                            {conditionFor(n.path) && (
                              <Box
                                sx={{
                                  mt: 0.75,
                                  fontFamily: "monospace",
                                  fontSize: 11,
                                  p: 0.5,
                                  borderRadius: 0.75,
                                  bgcolor: theme.palette.mode === "dark" ? "rgba(255,255,255,0.05)" : "grey.100",
                                  border: 1,
                                  borderColor: "divider",
                                }}
                              >
                                {conditionFor(n.path)}
                              </Box>
                            )}
                          </Box>
                        </Stack>
                      </TableCell>
                      <TableCell>{baselineChip(n.baseline)}</TableCell>
                      <TableCell>
                        <Stack direction="row" spacing={1} alignItems="center">
                          {stateChip(state)}
                          {!isEditing && (
                            <Tooltip title={isOverridden ? "Edit override" : "Add override"}>
                              <IconButton
                                size="small"
                                onClick={() => {
                                  setEditingNode(n.path);
                                  setEditState(state);
                                  setEditCondition(conditionFor(n.path));
                                }}
                              >
                                <EditIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          )}
                        </Stack>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      <Dialog open={!!editingNode} onClose={() => setEditingNode(null)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar sx={{ bgcolor: alpha(theme.palette.primary.main, 0.12), width: 32, height: 32 }}>
              <SecurityIcon color="primary" fontSize="small" />
            </Avatar>
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Override Component Entitlement
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
                {editingNode}
              </Typography>
            </Box>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ pt: 1 }}>
            <Box>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, textTransform: "uppercase", mb: 1, display: "block" }}>
                Override State
              </Typography>
              <ToggleButtonGroup
                value={editState}
                exclusive
                onChange={(_, v) => v && setEditState(v as OverrideState)}
                fullWidth
                sx={{
                  "& .MuiToggleButton-root": { py: 1.5, textTransform: "none", fontWeight: 600 },
                }}
              >
                <ToggleButton value="INHERIT_BASELINE">
                  <Stack alignItems="center" spacing={0.25}>
                    <VerifiedIcon fontSize="small" />
                    <Typography variant="caption">Inherit</Typography>
                  </Stack>
                </ToggleButton>
                <ToggleButton value="EXPLICIT_ALLOW" color="success">
                  <Stack alignItems="center" spacing={0.25}>
                    <CheckCircleIcon fontSize="small" />
                    <Typography variant="caption">Explicit Allow</Typography>
                  </Stack>
                </ToggleButton>
                <ToggleButton value="FORCE_DENY" color="error">
                  <Stack alignItems="center" spacing={0.25}>
                    <BlockIcon fontSize="small" />
                    <Typography variant="caption">Force Deny</Typography>
                  </Stack>
                </ToggleButton>
              </ToggleButtonGroup>
            </Box>

            <TextField
              label="Conditional DSL (optional)"
              value={editCondition}
              onChange={(e) => setEditCondition(e.target.value)}
              placeholder="resource.value <= 5000000"
              fullWidth
              multiline
              minRows={2}
              helperText="Apply this override only when the condition evaluates true"
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start" sx={{ alignItems: "flex-start", mt: 1 }}>
                    <CodeIcon fontSize="small" color="action" />
                  </InputAdornment>
                ),
              }}
            />

            {editState === "FORCE_DENY" && (
              <Alert severity="warning" variant="outlined">
                Force Deny shadows the baseline AND any tenant-scoped ALLOW rules at lower priority.
              </Alert>
            )}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setEditingNode(null)} startIcon={<CloseIcon />}>
            Cancel
          </Button>
          <Button onClick={() => editingNode && handleSave(editingNode)} variant="contained" startIcon={<SaveIcon />}>
            Save Override
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

export default EntitlementMatrix;