import React, { useState } from 'react';
import { Box, Button, TextField, Typography, Alert, Stack, Dialog, DialogTitle, DialogContent, DialogActions } from '@mui/material';
import { useAuthFetch } from '../../../utils/authFetch';

const TemporalOpsPanel: React.FC = () => {
  const [workflowId, setWorkflowId] = useState('');
  const [runId, setRunId] = useState('');
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<string | null>(null);

  const { authFetch } = useAuthFetch();

  const apiPost = async (path: string, body: any) => {
    setMessage(null);
    setError(null);
    try {
  const resp = await authFetch(path, { method: 'POST', json: body });
      if (!resp.ok) {
        setError(resp.error || JSON.stringify(resp.data));
        return;
      }
      setMessage(JSON.stringify(resp.data || resp, null, 2));
    } catch (err: any) {
      setError(err.message || String(err));
    }
  };

  const doSignal = () => {
    apiPost(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/signal`, { run_id: runId, signal_name: 'unblock', input: {} });
  };

  const doUpdate = () => {
    apiPost(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/update`, { run_id: runId, update_name: 'changePriority', input: {} });
  };

  const confirmAction = (action: string) => {
    setPendingAction(action);
    setConfirmOpen(true);
  };

  const performPending = () => {
    setConfirmOpen(false);
    if (!pendingAction) return;
    switch (pendingAction) {
      case 'cancel':
        apiPost(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/cancel`, { run_id: runId, reason: 'cancelled via admin UI' });
        break;
      case 'terminate':
        apiPost(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/terminate`, { run_id: runId, reason: 'terminated via admin UI' });
        break;
      case 'reset':
        apiPost(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/reset`, { run_id: runId, reset_type: 'LastWorkflowTask', reason: 'reset via admin UI' });
        break;
      default:
        break;
    }
    setPendingAction(null);
  };

  const doStack = async () => {
    setMessage(null);
    setError(null);
    try {
  const resp = await authFetch(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/stack?run_id=${encodeURIComponent(runId || '')}`, { method: 'GET' });
      if (!resp.ok) {
        setError(resp.error || JSON.stringify(resp.data));
        return;
      }
      // Show formatted stack in the message area
      const data = resp.data || resp;
      setMessage(typeof data === 'string' ? data : JSON.stringify(data, null, 2));
    } catch (err: any) {
      setError(err.message || String(err));
    }
  };

  const downloadFile = (filename: string, content: string, mime = 'application/json') => {
    const blob = new Blob([content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  };

  const exportHistory = async (asCsv = false) => {
    setMessage(null); setError(null);
    try {
      const resp = await authFetch(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/history/export`, { method: 'POST', json: { run_id: runId } });
      if (!resp.ok) { setError(resp.error || JSON.stringify(resp.data)); return; }
      const data = resp.data || resp;
      if (asCsv) {
        // naive CSV conversion for array of objects
        const arr = Array.isArray(data) ? data : [data];
        const keys = Array.from(new Set(arr.flatMap((o: any) => Object.keys(o))));
        const csv = [keys.join(',')].concat(arr.map((r: any) => keys.map(k => JSON.stringify(r[k] ?? '')).join(','))).join('\n');
        downloadFile(`${workflowId}-history.csv`, csv, 'text/csv');
      } else {
        downloadFile(`${workflowId}-history.json`, JSON.stringify(data, null, 2));
      }
      setMessage('Export complete. Download should begin.');
    } catch (err: any) { setError(err.message || String(err)); }
  };

  const exportAudit = async (asCsv = false) => {
    setMessage(null); setError(null);
    try {
      const resp = await authFetch(`/api/temporal/workflows/${encodeURIComponent(workflowId)}/audit?run_id=${encodeURIComponent(runId || '')}`, { method: 'GET' });
      if (!resp.ok) { setError(resp.error || JSON.stringify(resp.data)); return; }
      const data = resp.data?.audit || resp.data || resp;
      if (asCsv) {
        const arr = Array.isArray(data) ? data : [data];
        const keys = Array.from(new Set(arr.flatMap((o: any) => Object.keys(o))));
        const csv = [keys.join(',')].concat(arr.map((r: any) => keys.map(k => JSON.stringify(r[k] ?? '')).join(','))).join('\n');
        downloadFile(`${workflowId}-audit.csv`, csv, 'text/csv');
      } else {
        downloadFile(`${workflowId}-audit.json`, JSON.stringify(data, null, 2));
      }
      setMessage('Audit export complete. Download should begin.');
    } catch (err: any) { setError(err.message || String(err)); }
  };

  const describeTaskQueue = async () => {
    setMessage(null); setError(null);
    try {
      const resp = await authFetch(`/api/temporal/taskqueue/describe?queue=${encodeURIComponent(workflowId)}`, { method: 'GET' });
      if (!resp.ok) { setError(resp.error || JSON.stringify(resp.data)); return; }
      setMessage(JSON.stringify(resp.data || resp, null, 2));
    } catch (err: any) { setError(err.message || String(err)); }
  };

  return (
    <Box sx={{ p: 2, border: '1px solid #e0e0e0', borderRadius: 1, background: '#fff' }}>
      <Typography variant="h6">Temporal Ops (runbook)</Typography>
      <Stack direction="row" spacing={2} sx={{ mt: 2, mb: 1 }}>
        <TextField label="Workflow ID" value={workflowId} onChange={e => setWorkflowId(e.target.value)} size="small" />
        <TextField label="Run ID (optional)" value={runId} onChange={e => setRunId(e.target.value)} size="small" />
        <Button variant="contained" onClick={doSignal} disabled={!workflowId}>Signal</Button>
        <Button variant="outlined" onClick={doUpdate} disabled={!workflowId}>Update</Button>
        <Button color="warning" variant="outlined" onClick={() => confirmAction('cancel')} disabled={!workflowId}>Cancel</Button>
        <Button color="error" variant="contained" onClick={() => confirmAction('terminate')} disabled={!workflowId}>Terminate</Button>
        <Button variant="contained" onClick={() => confirmAction('reset')} disabled={!workflowId}>Reset</Button>
        <Button variant="text" onClick={doStack} disabled={!workflowId}>Stack Trace</Button>
  <Button variant="text" onClick={() => exportHistory(false)} disabled={!workflowId}>Export History (JSON)</Button>
  <Button variant="text" onClick={() => exportHistory(true)} disabled={!workflowId}>Export History (CSV)</Button>
  <Button variant="text" onClick={() => exportAudit(false)} disabled={!workflowId}>Export Audit (JSON)</Button>
  <Button variant="text" onClick={() => exportAudit(true)} disabled={!workflowId}>Export Audit (CSV)</Button>
  <Button variant="text" onClick={describeTaskQueue} disabled={!workflowId}>Task Queue Info</Button>
      </Stack>

      {message && <Alert severity="success" sx={{ mt: 2, whiteSpace: 'pre-wrap' }}>{message}</Alert>}
      {error && <Alert severity="error" sx={{ mt: 2, whiteSpace: 'pre-wrap' }}>{error}</Alert>}

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogTitle>Confirm {pendingAction}</DialogTitle>
        <DialogContent>
          <Typography>Type "CONFIRM" to proceed with the {pendingAction} action on workflow <strong>{workflowId}</strong>.</Typography>
          <TextField autoFocus margin="dense" id="confirm" label="Type CONFIRM" fullWidth variant="standard" onKeyDown={(e) => {
            const target = e.target as HTMLInputElement;
            if (e.key === 'Enter' && target.value === 'CONFIRM') performPending();
          }} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmOpen(false)}>Cancel</Button>
          <Button color="error" onClick={performPending}>Confirm</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TemporalOpsPanel;
