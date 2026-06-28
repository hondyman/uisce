import React, { useEffect, useState } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  FormControlLabel,
  FormLabel,
  Radio,
  RadioGroup,
  Slider,
  TextField,
  Typography,
  Box,
  Alert,
} from '@mui/material';
import {
  useImpersonation,
  type ImpersonationMode,
  type ImpersonationScope,
  MIN_REASON_LENGTH,
  TICKET_REQUIRED_FOR_BREAK_GLASS,
  DEFAULT_SESSION_DURATION_MINUTES,
  MAX_SESSION_DURATION_MINUTES,
} from '../../contexts/ImpersonationContext';

interface ImpersonationModalProps {
open: boolean;
onClose: () => void;
targetTenantId: string;
targetTenantName: string;
  /**
   * Optional pre-selected scope from the picker. When omitted, defaults to
   * tenant-wide. Re-choosing inside the modal is still allowed.
 */
initialScope?: ImpersonationScope;
}

const DURATION_MARKS = [
  { value: 15, label: '15m' },
  { value: 30, label: '30m' },
{ value: 60, label: '1h' },
{ value: 120, label: '2h' },
];

export const ImpersonationModal: React.FC<ImpersonationModalProps> = ({
  open,
  onClose,
  targetTenantId,
  targetTenantName,
  initialScope,
}) => {
  const { assumeTenantContext, isLoading } = useImpersonation();

  const [reason, setReason] = useState('');
  const [ticketReference, setTicketReference] = useState('');
  const [mode, setMode] = useState<ImpersonationMode>('read_only');
  const [durationMinutes, setDurationMinutes] = useState(DEFAULT_SESSION_DURATION_MINUTES);
  const [error, setError] = useState<string | null>(null);
  // The default scope is tenant-wide. The picker modal can extend this to instance/product/datasource.
  const [scope, setScope] = useState<ImpersonationScope>(
    initialScope ?? { kind: 'tenant', id: targetTenantId },
  );

  // Update scope when the picker provides a new initialScope.
  useEffect(() => {
    if (initialScope) {
      setScope(initialScope);
    }
  }, [initialScope]);

  // Keep the scope id in sync when the chosen tenant changes (e.g. picker is reused).
  useEffect(() => {
    setScope((prev) => ({ ...prev, id: targetTenantId }));
  }, [targetTenantId]);

  const handleAssume = async () => {
    setError(null);

    if (reason.trim().length < MIN_REASON_LENGTH) {
      setError(`Reason must be at least ${MIN_REASON_LENGTH} characters long.`);
      return;
    }
    if (mode === 'break_glass' && TICKET_REQUIRED_FOR_BREAK_GLASS && !ticketReference.trim()) {
      setError('Ticket reference is mandatory for break_glass mode.');
      return;
    }
    if (durationMinutes < 1 || durationMinutes > MAX_SESSION_DURATION_MINUTES) {
      setError(`Duration must be between 1 and ${MAX_SESSION_DURATION_MINUTES} minutes.`);
      return;
    }

    try {
      await assumeTenantContext({
        targetTenantId,
        targetTenantName,
        reason: reason.trim(),
        ticketReference: ticketReference.trim(),
        mode,
        durationMinutes,
        scope,
      });
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to assume tenant context');
    }
  };

  const handleClose = () => {
    if (!isLoading) {
      onClose();
      // Reset state after close animation
      setTimeout(() => {
        setReason('');
        setTicketReference('');
        setMode('read_only');
        setDurationMinutes(30);
        setError(null);
      }, 300);
    }
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Assume Tenant Context</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 3 }}>
          You are about to assume the context of tenant <strong>{targetTenantName}</strong>. 
          This action will be recorded in the platform audit log.
        </DialogContentText>

        {error && (
          <Alert severity="error" sx={{ mb: 3 }}>
            {error}
          </Alert>
        )}

        <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          <TextField
            required
            label="Reason"
            multiline
            rows={3}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Describe why you need to access this tenant..."
            fullWidth
            disabled={isLoading}
          />

          <TextField
            required={mode === 'break_glass'}
            label="Support Ticket Reference"
            value={ticketReference}
            onChange={(e) => setTicketReference(e.target.value)}
            placeholder="e.g. SUP-1234"
            fullWidth
            disabled={isLoading}
          />

          <FormControl disabled={isLoading}>
            <FormLabel id="impersonation-mode-label">Access Mode</FormLabel>
            <RadioGroup
              aria-labelledby="impersonation-mode-label"
              value={mode}
              onChange={(e) => setMode(e.target.value as ImpersonationMode)}
            >
              <FormControlLabel 
                value="read_only" 
                control={<Radio />} 
                label="Read-Only (Default)" 
              />
              <FormControlLabel 
                value="break_glass" 
                control={<Radio color="error" />} 
                label={
                  <Typography color="error">
                    Break-Glass (Write Access)
                  </Typography>
                }
              />
            </RadioGroup>
          </FormControl>

          <Box>
            <Typography gutterBottom>Duration (Minutes)</Typography>
            <Slider
              value={durationMinutes}
              onChange={(_, value) => setDurationMinutes(value as number)}
              step={15}
              marks={DURATION_MARKS}
              min={15}
              max={120}
              valueLabelDisplay="auto"
              disabled={isLoading}
              sx={{ px: 2 }}
            />
          </Box>
        </Box>
      </DialogContent>
      <DialogActions sx={{ p: 3, pt: 0 }}>
        <Button onClick={handleClose} disabled={isLoading}>
          Cancel
        </Button>
        <Button 
          onClick={handleAssume} 
          variant="contained" 
          color={mode === 'break_glass' ? 'error' : 'primary'}
          disabled={isLoading}
        >
          {isLoading ? 'Assuming...' : 'Confirm & Assume'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
