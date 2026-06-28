/**
 * ImpersonationBanner
 *
 * A persistent, unmissable banner rendered at the top of every page whenever
 * a global admin has assumed a tenant context.
 *
 * Visual language:
 *   - READ-ONLY mode:  amber/orange gradient — "caution, elevated access"
 *   - BREAK-GLASS mode: red gradient with a pulse animation — "danger, full write access"
 *
 * The banner is designed so that it CANNOT be dismissed or hidden.
 * It disappears only when the session expires or the admin clicks "Exit".
 */

import React, { useMemo } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  AdminPanelSettings as ShieldIcon,
  ExitToApp as ExitIcon,
  LocalFireDepartment as FireIcon,
  Schedule as ClockIcon,
  ConfirmationNumber as TicketIcon,
} from '@mui/icons-material';
import { keyframes } from '@mui/system';
import { useImpersonation } from '../contexts/ImpersonationContext';

// ---------------------------------------------------------------------------
// Animations
// ---------------------------------------------------------------------------

const readOnlyPulse = keyframes`
  0%   { border-color: rgba(245, 158, 11, 0.6); }
  50%  { border-color: rgba(245, 158, 11, 1); }
  100% { border-color: rgba(245, 158, 11, 0.6); }
`;

const breakGlassPulse = keyframes`
  0%   { border-color: rgba(239, 68, 68, 0.5); box-shadow: 0 0 0 0 rgba(239,68,68,0.4); }
  50%  { border-color: rgba(239, 68, 68, 1);   box-shadow: 0 0 0 6px rgba(239,68,68,0); }
  100% { border-color: rgba(239, 68, 68, 0.5); box-shadow: 0 0 0 0 rgba(239,68,68,0); }
`;

// ---------------------------------------------------------------------------
// Countdown formatter
// ---------------------------------------------------------------------------

function formatCountdown(seconds: number): string {
  if (seconds <= 0) return '00:00';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) {
    return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  }
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const ImpersonationBanner: React.FC = () => {
  const { isImpersonating, session, isLoading, exitImpersonation } = useImpersonation();

  const isBreakGlass = session?.mode === 'break_glass';

  // Colour palette
  const palette = useMemo(() => {
    if (isBreakGlass) {
      return {
        background: 'linear-gradient(135deg, #7f1d1d 0%, #991b1b 40%, #b91c1c 100%)',
        borderColor: 'rgba(239,68,68,0.8)',
        chipBg: 'rgba(239,68,68,0.25)',
        chipColor: '#fca5a5',
        textPrimary: '#fee2e2',
        textSecondary: '#fca5a5',
        animation: `${breakGlassPulse} 1.5s ease-in-out infinite`,
        iconColor: '#fca5a5',
      };
    }
    return {
      background: 'linear-gradient(135deg, #78350f 0%, #92400e 40%, #b45309 100%)',
      borderColor: 'rgba(245,158,11,0.7)',
      chipBg: 'rgba(245,158,11,0.2)',
      chipColor: '#fde68a',
      textPrimary: '#fef3c7',
      textSecondary: '#fde68a',
      animation: `${readOnlyPulse} 2s ease-in-out infinite`,
      iconColor: '#fde68a',
    };
  }, [isBreakGlass]);

  if (!isImpersonating || !session) return null;

  const countdown = formatCountdown(session.secondsRemaining);
  const urgentTime = session.secondsRemaining < 300; // < 5 min = urgent

  return (
    <Box
      role="alert"
      aria-live="assertive"
      aria-label="Admin impersonation mode active"
      id="impersonation-banner"
      sx={{
        width: '100%',
        background: palette.background,
        borderBottom: `2px solid`,
        borderColor: palette.borderColor,
        animation: palette.animation,
        zIndex: 1300,
        position: 'relative',
      }}
    >
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        flexWrap="wrap"
        gap={1}
        sx={{ px: 3, py: 1.25 }}
      >
        {/* ── Left: Identity + mode indicator ── */}
        <Stack direction="row" alignItems="center" gap={1.5} flexWrap="wrap">
          {isBreakGlass ? (
            <FireIcon sx={{ color: palette.iconColor, fontSize: 20 }} />
          ) : (
            <ShieldIcon sx={{ color: palette.iconColor, fontSize: 20 }} />
          )}

          <Typography
            variant="body2"
            fontWeight={700}
            letterSpacing={0.5}
            sx={{ color: palette.textPrimary, textTransform: 'uppercase', fontSize: '0.75rem' }}
          >
            {isBreakGlass ? '⚡ Break-Glass Active' : '⚠ Admin Impersonation Active'}
          </Typography>

          <Chip
            label={`Tenant: ${session.targetTenantName}`}
            size="small"
            sx={{
              bgcolor: palette.chipBg,
              color: palette.chipColor,
              fontWeight: 600,
              fontSize: '0.72rem',
              border: `1px solid ${palette.borderColor}`,
            }}
          />

          <Tooltip title={`Full Tenant UUID: ${session.targetTenantId}`} placement="bottom">
            <Chip
              label={`ID: …${session.targetTenantId.slice(-8)}`}
              size="small"
              sx={{
                bgcolor: 'rgba(0,0,0,0.25)',
                color: palette.textSecondary,
                fontSize: '0.68rem',
                fontFamily: 'monospace',
                border: `1px solid rgba(255,255,255,0.1)`,
                cursor: 'help',
              }}
            />
          </Tooltip>

          <Chip
            label={isBreakGlass ? 'WRITE ACCESS ENABLED' : 'READ-ONLY'}
            size="small"
            sx={{
              bgcolor: isBreakGlass ? 'rgba(239,68,68,0.35)' : 'rgba(34,197,94,0.15)',
              color: isBreakGlass ? '#fca5a5' : '#86efac',
              fontWeight: 700,
              fontSize: '0.68rem',
              border: isBreakGlass
                ? '1px solid rgba(239,68,68,0.5)'
                : '1px solid rgba(34,197,94,0.3)',
            }}
          />

          {session.ticketReference && (
            <Tooltip title="Support ticket reference" placement="bottom">
              <Stack direction="row" alignItems="center" gap={0.5}>
                <TicketIcon sx={{ fontSize: 14, color: palette.textSecondary }} />
                <Typography variant="caption" sx={{ color: palette.textSecondary, fontFamily: 'monospace' }}>
                  {session.ticketReference}
                </Typography>
              </Stack>
            </Tooltip>
          )}
        </Stack>

        {/* ── Right: Countdown + exit ── */}
        <Stack direction="row" alignItems="center" gap={2}>
          <Tooltip title={`Session expires at ${session.expiresAt.toLocaleTimeString()}`} placement="bottom">
            <Stack direction="row" alignItems="center" gap={0.5}>
              <ClockIcon
                sx={{
                  fontSize: 16,
                  color: urgentTime ? '#f87171' : palette.textSecondary,
                  animation: urgentTime ? `${breakGlassPulse} 1s ease-in-out infinite` : 'none',
                }}
              />
              <Typography
                variant="body2"
                fontFamily="monospace"
                fontWeight={700}
                sx={{
                  color: urgentTime ? '#f87171' : palette.textPrimary,
                  fontSize: '0.85rem',
                  minWidth: 52,
                }}
              >
                {countdown}
              </Typography>
            </Stack>
          </Tooltip>

          <Button
            id="exit-impersonation-btn"
            variant="outlined"
            size="small"
            startIcon={<ExitIcon />}
            disabled={isLoading}
            onClick={() => void exitImpersonation()}
            sx={{
              borderColor: palette.borderColor,
              color: palette.textPrimary,
              fontWeight: 600,
              fontSize: '0.72rem',
              textTransform: 'none',
              whiteSpace: 'nowrap',
              '&:hover': {
                borderColor: palette.textPrimary,
                bgcolor: 'rgba(255,255,255,0.1)',
              },
              '&:disabled': {
                opacity: 0.5,
              },
            }}
          >
            {isLoading ? 'Exiting…' : 'Exit Impersonation'}
          </Button>
        </Stack>
      </Stack>

      {/* Reason tooltip row */}
      <Box sx={{ px: 3, pb: 0.75 }}>
        <Typography variant="caption" sx={{ color: palette.textSecondary, opacity: 0.8 }}>
          Reason: {session.reason} · All actions are being recorded in the platform audit log.
        </Typography>
      </Box>
    </Box>
  );
};

export default ImpersonationBanner;
