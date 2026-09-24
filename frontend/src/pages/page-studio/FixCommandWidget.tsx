import React, { useState } from 'react';
import { Button, Stack, Alert, CircularProgress } from '@mui/material';
import { ComponentDefinition } from '../../types/pageStudio';
import { useSelection } from './SelectionContext';
import { apiClient } from '../../utils/apiClient';

interface FixCommandWidgetProps {
  component: ComponentDefinition;
  mode?: 'design' | 'preview';
}

const COMMANDS = ['NewOrderSingle', 'CancelReplace', 'Cancel'] as const;

/**
 * Presentation-only command button. Fires POST /api/oms/commands/fix-order-entry
 * which starts FIXOrderEntryWorkflow. Does not save the page or write fills.
 */
const FixCommandWidget: React.FC<FixCommandWidgetProps> = ({ component, mode }) => {
  const { selection } = useSelection();
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ severity: 'success' | 'error'; message: string } | null>(null);

  const label = (component.props?.label as string) || 'Send FIX';
  const command = String(component.props?.command || 'NewOrderSingle').replace(/^fix\./, '');
  const sessionId = component.props?.sessionId as string | undefined;
  const fieldMap = (component.props?.fieldMap as Record<string, string>) || {};
  const variant = (component.props?.variant as 'contained' | 'outlined' | 'text') || 'contained';

  const handleClick = async () => {
    if (mode === 'design') return;
    const orderId = selection?.recordId;
    if (!orderId) {
      setNotice({ severity: 'error', message: 'Select an order first (open this page from the list).' });
      return;
    }
    setBusy(true);
    setNotice(null);
    try {
      const body: Record<string, unknown> = { orderId, command, sessionId };
      if (fieldMap.symbol) body.symbolField = fieldMap.symbol;
      const res = await apiClient<{ clOrdId?: string; workflowId?: string; status?: string }>(
        '/api/oms/commands/fix-order-entry',
        { method: 'POST', body: JSON.stringify(body), headers: { 'Content-Type': 'application/json' } },
      );
      setNotice({
        severity: 'success',
        message: `${command} started (${res.clOrdId || res.workflowId || res.status || 'ok'})`,
      });
    } catch (err) {
      setNotice({ severity: 'error', message: err instanceof Error ? err.message : 'Command failed' });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Stack spacing={1}>
      <Button variant={variant} onClick={handleClick} disabled={busy || mode === 'design'}>
        {busy ? <CircularProgress size={16} /> : label}
      </Button>
      {mode === 'design' && (
        <Alert severity="info">
          Core FIX command ({COMMANDS.includes(command as typeof COMMANDS[number]) ? command : 'NewOrderSingle'}). Runtime starts
          FIXOrderEntryWorkflow — it does not save this page.
        </Alert>
      )}
      {notice && <Alert severity={notice.severity} onClose={() => setNotice(null)}>{notice.message}</Alert>}
    </Stack>
  );
};

export default FixCommandWidget;
