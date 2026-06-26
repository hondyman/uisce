import { useState } from 'react';
import { useAuth } from '../../../contexts/AuthContext';
import Dialog from '@mui/material/Dialog';
import ModalHeader from '../../../components/ModalHeader';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import { Button, Typography, FormControl, InputLabel, Select, MenuItem, Checkbox, FormControlLabel, CircularProgress, Alert } from '@mui/material';

interface TenantRef { id: string; displayName: string }

interface Props {
  open: boolean;
  ownerTenantId: string | null;
  onClose: () => void;
  conflictingIp: string | null;
  tenants: TenantRef[];
  onCompleted?: () => void; // called when operation finishes successfully
}

const ManageConflictModal: React.FC<Props> = ({ open, ownerTenantId, onClose, conflictingIp, tenants, onCompleted }) => {
  const [transferTo, setTransferTo] = useState<string | null>(null);
  const [includeAssign, setIncludeAssign] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [pendingConfirmation, setPendingConfirmation] = useState(false);
  const [lastOperation, setLastOperation] = useState<{ owner?: string; target?: string; ip?: string } | null>(null);
  const { isAdmin } = useAuth();

  const doTransfer = async () => {
    setError(null);
    setSuccess(null);
    if (!isAdmin()) {
      setError('You do not have permission to perform this action.');
      return;
    }
    if (!ownerTenantId || !conflictingIp || !transferTo) return;

    // request a final confirmation from user
    if (!pendingConfirmation) {
      setPendingConfirmation(true);
      return;
    }

    setWorking(true);
    try {
      // Remove assignment from ownerTenantId
      const del = await fetch(`/api/tenants/${ownerTenantId}/ip-whitelist`, {
        method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ipAddress: conflictingIp, tenantIds: [ownerTenantId] })
      });
      if (!del.ok) {
        const body = await del.text();
        throw new Error(`Failed to remove assignment from owner: ${body}`);
      }

      if (includeAssign) {
        const add = await fetch(`/api/tenants/${transferTo}/ip-whitelist`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ipAddress: conflictingIp, tenantIds: [transferTo] }) });
        if (!add.ok) {
          const body = await add.text();
          throw new Error(`Failed to add assignment to target tenant: ${body}`);
        }
      }

      setSuccess('Operation completed. You can undo this action for a short time.');
      setLastOperation({ owner: ownerTenantId || undefined, target: transferTo || undefined, ip: conflictingIp || undefined });
      if (onCompleted) onCompleted();
      // allow undo for 10s then close
      setTimeout(() => {
        setWorking(false);
        setPendingConfirmation(false);
        setLastOperation(null);
        onClose();
      }, 10000);
    } catch (e: any) {
      setError(e?.message || 'Operation failed');
      setWorking(false);
      setPendingConfirmation(false);
    }
  };

  const undoLast = async () => {
    if (!lastOperation || !lastOperation.owner || !lastOperation.ip) return;
    setWorking(true);
    setError(null);
    try {
      // Re-assign to owner (best-effort)
      const add = await fetch(`/api/tenants/${lastOperation.owner}/ip-whitelist`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ipAddress: lastOperation.ip, tenantIds: [lastOperation.owner] }) });
      if (!add.ok) {
        const body = await add.text();
        throw new Error(`Undo failed: ${body}`);
      }
      setSuccess('Undo successful.');
      setLastOperation(null);
      setTimeout(() => { setWorking(false); onClose(); }, 800);
    } catch (e: any) {
      setError(e?.message || 'Undo failed');
      setWorking(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <ModalHeader title="Manage conflicting entry" onClose={onClose} />
      <DialogContent>
        {error && <Alert severity="error" sx={{ mb: 1 }}>{error}</Alert>}
        {success && (
          <Alert severity="success" sx={{ mb: 1 }} action={lastOperation ? <Button color="inherit" size="small" onClick={undoLast}>Undo</Button> : undefined}>{success}</Alert>
        )}
        <Typography sx={{ mb: 1 }}>Conflicting IP: <strong>{conflictingIp}</strong></Typography>
  <FormControlLabel control={<Checkbox checked={includeAssign} onChange={(e) => setIncludeAssign(e.target.checked)} />} label="Assign to new tenant after transfer" />
        <FormControl sx={{ mt: 2, width: '100%' }}>
          <InputLabel id="transfer-to-label">Transfer to tenant</InputLabel>
          <Select labelId="transfer-to-label" value={transferTo || ''} onChange={(e) => setTransferTo(e.target.value as string)} label="Transfer to tenant">
            <MenuItem value="">Select tenant</MenuItem>
            {tenants.map(t => <MenuItem key={t.id} value={t.id}>{t.displayName}</MenuItem>)}
          </Select>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => { setPendingConfirmation(false); onClose(); }} disabled={working}>Cancel</Button>
        <Button onClick={doTransfer} disabled={!transferTo || working} variant="contained" color={pendingConfirmation ? 'warning' : 'primary'} startIcon={working ? <CircularProgress size={16} /> : null}>{working ? 'Working...' : pendingConfirmation ? 'Confirm Execute' : 'Execute'}</Button>
      </DialogActions>
    </Dialog>
  );
};

export default ManageConflictModal;
