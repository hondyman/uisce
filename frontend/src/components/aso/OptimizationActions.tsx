import React, { useState, useMemo } from 'react';
import {
  Box,
  Stack,
  Button,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Typography,
  Alert,
  Chip,
  CircularProgress,
} from '@mui/material';
import {
  PlayArrow,
  ThumbUp,
  ThumbDown,
  Visibility,
  Science,
  Lock,
  Warning,
} from '@mui/icons-material';
import { useAuth } from '../../contexts/AuthContext';

// ============================================================================
// Types
// ============================================================================

export interface Optimization {
  id: string;
  env: string;
  tenant_id?: string;
  scope: 'core' | 'tenant';
  status: string;
  optimization_type: string;
  score: number;
  ml_score?: number;
  risk_score?: number;
}

export interface ASOPolicy {
  mode: 'advisory' | 'auto_tune' | 'auto_apply';
  enabled: boolean;
}

export interface PermissionSet {
  can_view: boolean;
  can_approve: boolean;
  can_apply: boolean;
  can_reject: boolean;
  can_simulate: boolean;
}

// ============================================================================
// Permission Helpers
// ============================================================================

const ASORole = {
  GOLDCOPY_ADMIN: 'goldcopy_admin',
  GLOBAL_OPS: 'global_ops',
  TENANT_ADMIN: 'tenant_admin',
  TENANT_OPS: 'tenant_ops',
} as const;

export function hasRole(roles: string[], role: string): boolean {
  return roles.includes(role);
}

export function canViewCore(roles: string[]): boolean {
  return (
    hasRole(roles, ASORole.GOLDCOPY_ADMIN) ||
    hasRole(roles, ASORole.GLOBAL_OPS)
  );
}

export function canViewTenant(roles: string[], tenantId: string, userTenants: string[]): boolean {
  if (hasRole(roles, ASORole.GOLDCOPY_ADMIN) || hasRole(roles, ASORole.GLOBAL_OPS)) {
    return true;
  }
  return userTenants.includes(tenantId);
}

export function canApproveOptimization(
  roles: string[],
  opt: Optimization,
  userTenants: string[]
): boolean {
  if (opt.scope === 'core') {
    return hasRole(roles, ASORole.GOLDCOPY_ADMIN);
  }
  if (opt.tenant_id && canViewTenant(roles, opt.tenant_id, userTenants)) {
    return (
      hasRole(roles, ASORole.TENANT_ADMIN) ||
      hasRole(roles, ASORole.GLOBAL_OPS) ||
      hasRole(roles, ASORole.GOLDCOPY_ADMIN)
    );
  }
  return false;
}

export function canApplyOptimization(
  roles: string[],
  opt: Optimization,
  policy: ASOPolicy | null,
  userTenants: string[]
): boolean {
  if (opt.scope === 'core') {
    return hasRole(roles, ASORole.GOLDCOPY_ADMIN);
  }
  if (opt.tenant_id && canViewTenant(roles, opt.tenant_id, userTenants)) {
    if (policy?.mode === 'auto_apply') {
      return (
        hasRole(roles, ASORole.TENANT_ADMIN) ||
        hasRole(roles, ASORole.TENANT_OPS) ||
        hasRole(roles, ASORole.GLOBAL_OPS) ||
        hasRole(roles, ASORole.GOLDCOPY_ADMIN)
      );
    }
    return (
      hasRole(roles, ASORole.TENANT_ADMIN) ||
      hasRole(roles, ASORole.GLOBAL_OPS) ||
      hasRole(roles, ASORole.GOLDCOPY_ADMIN)
    );
  }
  return false;
}

export function getPermissions(
  roles: string[],
  opt: Optimization,
  policy: ASOPolicy | null,
  userTenants: string[]
): PermissionSet {
  const canView = opt.scope === 'core'
    ? canViewCore(roles)
    : opt.tenant_id
    ? canViewTenant(roles, opt.tenant_id, userTenants)
    : false;

  return {
    can_view: canView,
    can_approve: canApproveOptimization(roles, opt, userTenants),
    can_apply: canApplyOptimization(roles, opt, policy, userTenants),
    can_reject: canApproveOptimization(roles, opt, userTenants),
    can_simulate: canView,
  };
}

// ============================================================================
// Optimization Actions Component
// ============================================================================

interface OptimizationActionsProps {
  optimization: Optimization;
  policy?: ASOPolicy | null;
  onView?: () => void;
  onApprove?: () => Promise<void>;
  onApply?: () => Promise<void>;
  onReject?: (reason: string) => Promise<void>;
  onSimulate?: () => void;
  compact?: boolean;
}

export const OptimizationActions: React.FC<OptimizationActionsProps> = ({
  optimization,
  policy = null,
  onView,
  onApprove,
  onApply,
  onReject,
  onSimulate,
  compact = false,
}) => {
  const { user } = useAuth();
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [loading, setLoading] = useState<string | null>(null);

  // Get user roles and tenants
  const userRoles = useMemo(() => user?.roles || [], [user]);
  const userTenants = useMemo(() => user?.tenant_ids || [], [user]);

  // Calculate permissions
  const permissions = useMemo(
    () => getPermissions(userRoles, optimization, policy, userTenants),
    [userRoles, optimization, policy, userTenants]
  );

  // Check if optimization is actionable
  const isPending = optimization.status === 'proposed';
  const isApproved = optimization.status === 'approved';
  const canAct = isPending || isApproved;

  // Risk warning
  const isHighRisk = (optimization.risk_score ?? 0) > 0.5;
  const showRiskWarning = isHighRisk && canAct && permissions.can_apply;

  const handleAction = async (action: string, handler?: () => Promise<void>) => {
    if (!handler) return;
    setLoading(action);
    try {
      await handler();
    } finally {
      setLoading(null);
    }
  };

  const handleReject = async () => {
    if (!onReject) return;
    setLoading('reject');
    try {
      await onReject(rejectReason);
      setRejectDialogOpen(false);
      setRejectReason('');
    } finally {
      setLoading(null);
    }
  };

  // Compact mode (for table rows)
  if (compact) {
    return (
      <Stack direction="row" spacing={0.5} justifyContent="center">
        {onView && (
          <Tooltip title="View Details">
            <IconButton size="small" onClick={onView}>
              <Visibility fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {onSimulate && permissions.can_simulate && (
          <Tooltip title="Simulate">
            <IconButton size="small" color="secondary" onClick={onSimulate}>
              <Science fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {canAct && permissions.can_apply && onApply && (
          <Tooltip title={isHighRisk ? 'Apply (High Risk)' : 'Apply'}>
            <IconButton
              size="small"
              color={isHighRisk ? 'warning' : 'success'}
              onClick={() => handleAction('apply', onApply)}
              disabled={loading === 'apply'}
            >
              {loading === 'apply' ? (
                <CircularProgress size={18} />
              ) : (
                <PlayArrow fontSize="small" />
              )}
            </IconButton>
          </Tooltip>
        )}

        {isPending && permissions.can_approve && onApprove && (
          <Tooltip title="Approve">
            <IconButton
              size="small"
              color="primary"
              onClick={() => handleAction('approve', onApprove)}
              disabled={loading === 'approve'}
            >
              {loading === 'approve' ? (
                <CircularProgress size={18} />
              ) : (
                <ThumbUp fontSize="small" />
              )}
            </IconButton>
          </Tooltip>
        )}

        {isPending && permissions.can_reject && onReject && (
          <Tooltip title="Reject">
            <IconButton
              size="small"
              color="error"
              onClick={() => setRejectDialogOpen(true)}
            >
              <ThumbDown fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {!permissions.can_approve && !permissions.can_apply && canAct && (
          <Tooltip title="Insufficient permissions">
            <IconButton size="small" disabled>
              <Lock fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {/* Reject Dialog */}
        <RejectDialog
          open={rejectDialogOpen}
          loading={loading === 'reject'}
          reason={rejectReason}
          onReasonChange={setRejectReason}
          onConfirm={handleReject}
          onCancel={() => setRejectDialogOpen(false)}
        />
      </Stack>
    );
  }

  // Full mode (for detail page)
  return (
    <Box>
      {showRiskWarning && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Warning />
            <Typography variant="body2">
              This optimization has a high risk score ({(optimization.risk_score! * 100).toFixed(0)}%).
              Consider running a simulation before applying.
            </Typography>
          </Stack>
        </Alert>
      )}

      {!canAct && (
        <Alert severity="info" sx={{ mb: 2 }}>
          This optimization is in <strong>{optimization.status}</strong> status and cannot be modified.
        </Alert>
      )}

      <Stack direction="row" spacing={2} justifyContent="flex-end">
        {/* Permission indicator */}
        {!permissions.can_approve && !permissions.can_apply && (
          <Chip
            icon={<Lock />}
            label="View Only"
            size="small"
            color="default"
          />
        )}

        {onSimulate && permissions.can_simulate && (
          <Button
            variant="outlined"
            color="secondary"
            startIcon={<Science />}
            onClick={onSimulate}
          >
            Simulate
          </Button>
        )}

        {isPending && permissions.can_reject && onReject && (
          <Button
            variant="outlined"
            color="error"
            startIcon={<ThumbDown />}
            onClick={() => setRejectDialogOpen(true)}
          >
            Reject
          </Button>
        )}

        {isPending && permissions.can_approve && onApprove && (
          <Button
            variant="outlined"
            color="primary"
            startIcon={<ThumbUp />}
            onClick={() => handleAction('approve', onApprove)}
            disabled={loading === 'approve'}
          >
            {loading === 'approve' ? <CircularProgress size={20} /> : 'Approve'}
          </Button>
        )}

        {canAct && permissions.can_apply && onApply && (
          <Button
            variant="contained"
            color={isHighRisk ? 'warning' : 'success'}
            startIcon={<PlayArrow />}
            onClick={() => handleAction('apply', onApply)}
            disabled={loading === 'apply'}
          >
            {loading === 'apply' ? <CircularProgress size={20} color="inherit" /> : 'Apply Now'}
          </Button>
        )}
      </Stack>

      {/* Reject Dialog */}
      <RejectDialog
        open={rejectDialogOpen}
        loading={loading === 'reject'}
        reason={rejectReason}
        onReasonChange={setRejectReason}
        onConfirm={handleReject}
        onCancel={() => setRejectDialogOpen(false)}
      />
    </Box>
  );
};

// ============================================================================
// Reject Dialog
// ============================================================================

interface RejectDialogProps {
  open: boolean;
  loading: boolean;
  reason: string;
  onReasonChange: (reason: string) => void;
  onConfirm: () => void;
  onCancel: () => void;
}

const RejectDialog: React.FC<RejectDialogProps> = ({
  open,
  loading,
  reason,
  onReasonChange,
  onConfirm,
  onCancel,
}) => (
  <Dialog open={open} onClose={onCancel} maxWidth="sm" fullWidth>
    <DialogTitle>Reject Optimization</DialogTitle>
    <DialogContent>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Please provide a reason for rejecting this optimization.
      </Typography>
      <TextField
        autoFocus
        label="Rejection Reason"
        fullWidth
        multiline
        rows={3}
        value={reason}
        onChange={(e) => onReasonChange(e.target.value)}
        placeholder="e.g., Not aligned with current priorities, needs further analysis..."
      />
    </DialogContent>
    <DialogActions>
      <Button onClick={onCancel} disabled={loading}>
        Cancel
      </Button>
      <Button
        onClick={onConfirm}
        color="error"
        variant="contained"
        disabled={!reason.trim() || loading}
      >
        {loading ? <CircularProgress size={20} /> : 'Reject'}
      </Button>
    </DialogActions>
  </Dialog>
);

export default OptimizationActions;
