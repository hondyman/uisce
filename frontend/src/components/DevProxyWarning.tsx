import React, { useState } from 'react';
import { Alert, AlertTitle, Link, Button, Dialog, DialogTitle, DialogContent, DialogActions } from '@mui/material';

const DevProxyWarning: React.FC = () => {
  const [dismissed, setDismissed] = useState(false);

  try {
    const env: any = (import.meta as any).env || {};
    const useProxy = String(env?.VITE_USE_PROXY || 'false').toLowerCase() === 'true';
    const backendTarget = String(env?.VITE_BACKEND_TARGET || env?.VITE_API_BASE_URL || '').toLowerCase();
    const runningOnHost = typeof window !== 'undefined' && window.location && (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1');

    const isProblem = useProxy && runningOnHost && backendTarget.includes('host.docker.internal');

    if (isProblem && !dismissed) {
      // Blocking modal — user must either 'Fix and reload' or 'Proceed anyway'
      return (
        <Dialog open={true} aria-labelledby="dev-proxy-warning-title">
          <DialogTitle id="dev-proxy-warning-title">Dev Proxy Misconfigured</DialogTitle>
          <DialogContent>
            <Alert severity="error" style={{ marginBottom: '8px' }}>
              <AlertTitle>Proxy points at <code>{backendTarget}</code></AlertTitle>
              You are running the frontend on <code>localhost</code> while the Vite proxy target points at <code>host.docker.internal</code>.
              This commonly prevents the proxy from reaching the backend. Set <code>VITE_BACKEND_TARGET</code> and <code>VITE_API_BASE_URL</code> to <code>http://localhost:8082</code> in <code>frontend/.env.local</code> and restart the dev server.
            </Alert>
            <div>Please choose an action below to continue.</div>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => window.location.reload()} color="primary">I've fixed it — reload</Button>
            <Button onClick={() => setDismissed(true)} color="inherit">Proceed anyway</Button>
          </DialogActions>
        </Dialog>
      );
    }
  } catch (e) {
    // ignore runtime errors in the check
  }
  // If dismissed, render a subtle non-blocking warning variant
  if (dismissed) {
    const envAny: any = (import.meta as any).env || {};
    const backendTarget = String(envAny?.VITE_BACKEND_TARGET || envAny?.VITE_API_BASE_URL || '');
    return (
      <Alert severity="warning" style={{ margin: '8px' }}>
        Dev Proxy may be misconfigured (target: <code>{backendTarget}</code>). Actions may fail.
      </Alert>
    );
  }

  return null;
};

export default DevProxyWarning;
