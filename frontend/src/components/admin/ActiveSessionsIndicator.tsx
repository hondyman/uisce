/**
 * ActiveSessionsIndicator
 *
 * Topbar pill that shows the count of currently-active impersonation sessions
 * for this admin (read from /api/admin/impersonate/sessions/active).
 *
 * Clicking the pill opens a popover listing each active session with two
 * actions:
 *   - "Switch to this tenant" — exits the current session (if any) and opens the
 *     impersonation picker pre-populated with that tenant + its recorded scope.
 *   - "End this session" — calls DELETE /api/admin/impersonate/{sessionId} to
 *     clean up the stale session.
 *
 * The pill is hidden when the admin has no active sessions.
 */

import React, { useCallback, useEffect, useState, useRef } from 'react';
import {
  Popover,
  Box,
  Typography,
  IconButton,
  Button,
  Divider,
  Chip,
  CircularProgress,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  Stack,
} from '@mui/material';
import {
  Visibility as ViewIcon,
  Close as CloseIcon,
  SwapHoriz as SwitchIcon,
} from '@mui/icons-material';
import {
  useImpersonation,
  type ActiveImpersonationSession,
} from '../../contexts/ImpersonationContext';
import { useAuth } from '../../contexts/AuthContext';

// How often to re-query active sessions (ms).
const POLL_INTERVAL_MS = 30_000;

const API_BASE = import.meta.env.VITE_API_URL ?? '/api';

export interface ActiveSessionsIndicatorProps {
  /**
   * Called when the admin clicks "Switch to this tenant" on a row.
   * The parent (typically TenantsPage) is expected to close any current
   * impersonation session and open the picker with the given tenant preselected.
   */
  onSwitchToTenant: (s: ActiveImpersonationSession) => void;
}

/**
 * Format a duration like "13m left" / "45s left" / "expired".
 */
function formatRemaining(expiresAt: string): string {
  if (!expiresAt) return '';
  const ms = new Date(expiresAt).getTime() - Date.now();
  if (ms <= 0) return 'expired';
  const sec = Math.round(ms / 1000);
  if (sec < 60) return `${sec}s left`;
  const min = Math.round(sec / 60);
  if (min < 60) return `${min}m left`;
  const hr = Math.round(min / 60);
  return `${hr}h left`;
}

export const ActiveSessionsIndicator: React.FC<ActiveSessionsIndicatorProps> = ({
  onSwitchToTenant,
}) => {
  const { listActiveSessions, exitImpersonation } = useImpersonation();
  const { token: authToken } = useAuth();
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const [sessions, setSessions] = useState<ActiveImpersonationSession[]>([]);
  const [loading, setLoading] = useState(false);
  const anchorRef = useRef<HTMLButtonElement | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listActiveSessions();
      setSessions(data);
    } catch {
      setSessions([]);
    } finally {
      setLoading(false);
    }
  }, [listActiveSessions]);

  // Initial fetch on mount + poll every POLL_INTERVAL_MS.
  useEffect(() => {
    void refresh();
    const id = setInterval(() => {
      void refresh();
    }, POLL_INTERVAL_MS);
    return () => clearInterval(id);
  }, [refresh]);

  const open = Boolean(anchorEl);
  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(e.currentTarget);
    void refresh();
  };
  const handleClose = () => setAnchorEl(null);

  const handleEndSession = async (sessionID: string) => {
    // If this is the admin's CURRENT session, exitImpersonation() handles it
    // via the existing context path. If it's a stale session (different from
    // the current one), we hit the DELETE endpoint directly so an END row is
    // written and the audit invariant holds.
    try {
      // We use a direct fetch (not the context helper) because the context
      // helper only knows the *current* session, not arbitrary session IDs.
      const resp = await fetch(
        `${API_BASE}/admin/impersonate/${sessionID}`,
        {
          method: 'DELETE',
          headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
        },
      );
      if (!resp.ok) {
        // Fall back to the context helper — may be a no-op if not the current session.
        void exitImpersonation();
      }
      handleClose();
      setTimeout(() => void refresh(), 500);
    } catch {
      // ignore — best-effort cleanup
    }
  };

  // Hide entirely when there are no sessions and we're not loading.
  if (!loading && sessions.length === 0 && !open) {
    return null;
  }

  return (
    <>
      <IconButton
        ref={anchorRef}
        onClick={handleClick}
        size="small"
        sx={{
          color: 'warning.main',
          bgcolor: 'warning.light',
          '&:hover': { bgcolor: 'warning.main', color: 'warning.contrastText' },
          px: 1.5,
        }}
        aria-label={`${sessions.length} active impersonation session(s)`}
      >
        <Stack direction="row" spacing={0.5} alignItems="center">
          <ViewIcon fontSize="small" />
          <Typography variant="caption" sx={{ fontWeight: 600 }}>
            {loading ? '…' : `${sessions.length} active`}
          </Typography>
        </Stack>
      </IconButton>

      <Popover
        open={open}
        anchorEl={anchorEl}
        onClose={handleClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        PaperProps={{ sx: { minWidth: 360, maxWidth: 480 } }}
      >
        <Box sx={{ p: 2, pb: 1 }}>
          <Typography variant="subtitle2">Active Impersonation Sessions</Typography>
          <Typography variant="caption" color="text.secondary">
            {sessions.length === 0
              ? 'No active sessions.'
              : `You have ${sessions.length} active session${sessions.length === 1 ? '' : 's'}.`}
          </Typography>
        </Box>
        <Divider />
        {loading && sessions.length === 0 && (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <CircularProgress size={20} />
          </Box>
        )}
        {sessions.length > 0 && (
          <List dense disablePadding>
            {sessions.map((s) => (
              <ListItem key={s.session_id} sx={{ px: 2, py: 1.25, alignItems: 'flex-start' }}>
                <ListItemText
                  primary={
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>
                        Tenant {s.target_tenant_id.slice(0, 8)}…
                      </Typography>
                      <Chip
                        size="small"
                        label={s.mode}
                        color={s.mode === 'break_glass' ? 'error' : 'info'}
                        variant="outlined"
                      />
                      {s.scope_kind && s.scope_kind !== 'tenant' && (
                        <Chip
                          size="small"
                          label={`scope: ${s.scope_kind}`}
                          color="warning"
                          variant="outlined"
                        />
                      )}
                    </Stack>
                  }
                  secondary={
                    <Typography variant="caption" color="text.secondary">
                      {s.reason || '(no reason recorded)'}
                      {s.expires_at && (
                        <> \u00b7 {formatRemaining(s.expires_at)}</>
                      )}
                    </Typography>
                  }
                />
                <ListItemSecondaryAction>
                  <Stack direction="row" spacing={0.5}>
                    <Button
                      size="small"
                      variant="outlined"
                      color="primary"
                      startIcon={<SwitchIcon />}
                      onClick={() => {
                        onSwitchToTenant(s);
                        handleClose();
                      }}
                    >
                      Switch
                    </Button>
                    <IconButton
                      size="small"
                      color="error"
                      onClick={() => handleEndSession(s.session_id)}
                      aria-label="End this session"
                    >
                      <CloseIcon fontSize="small" />
                    </IconButton>
                  </Stack>
                </ListItemSecondaryAction>
              </ListItem>
            ))}
          </List>
        )}
      </Popover>
    </>
  );
};

export default ActiveSessionsIndicator;