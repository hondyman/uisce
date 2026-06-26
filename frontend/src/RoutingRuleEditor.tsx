import { useState, useEffect } from 'react';
import { devError } from './utils/devLogger';
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Typography,
  Chip,
  CircularProgress,
  Alert,
  Paper,
} from '@mui/material';
import ModalHeader from './components/ModalHeader';
import { NotificationRoutingRule } from './types';
import { updateNotificationRule, previewRecipients } from './api';

interface RoutingRuleEditorProps {
  rule: NotificationRoutingRule | null;
  onClose: () => void;
}

const availableRecipients: string[] = ["asset_owner", "domain_steward", "downstream_users", "requestor", "user"];

export default function RoutingRuleEditor({ rule, onClose }: RoutingRuleEditorProps) {
  const [formData, setFormData] = useState<NotificationRoutingRule | null>(rule);
  const [saving, setSaving] = useState(false);
  const [simulating, setSimulating] = useState(false);
  const [simulationResult, setSimulationResult] = useState<Record<string, string[]> | null>(null);

  useEffect(() => {
    setFormData(rule);
  }, [rule]);

  const handleSave = async () => {
    if (!formData) return;
    setSaving(true);
    try {
      await updateNotificationRule(formData);
      onClose();
    } catch (err) {
      try { devError("Failed to save rule", err); } catch {}
    } finally {
      setSaving(false);
    }
  };

  const handleSimulate = async () => {
    setSimulating(true);
    try {
      // In a real app, you'd have a way to select a test asset.
      const testAssetId = "d1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b";
      const result = await previewRecipients(testAssetId);
      setSimulationResult(result);
    } catch (err) {
      try { devError("Simulation failed", err); } catch {}
    } finally {
      setSimulating(false);
    }
  };

  if (!formData) return null;

  return (
    <Dialog open={true} onClose={onClose} fullWidth maxWidth="md">
      <ModalHeader title={`Edit Routing Rule`} subtitle={formData.rule_id} onClose={onClose} />
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          <TextField label="Trigger" value={formData.trigger} disabled fullWidth />
          <TextField label="Scope" value={formData.scope} disabled fullWidth />
          <TextField label="Asset Type" value={formData.asset_type} disabled fullWidth />

          <Typography variant="subtitle1" sx={{ mt: 2 }}>Recipient Logic</Typography>
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="body2" color="text.secondary" gutterBottom>Notify</Typography>
            <Box>
              {availableRecipients.map(r => (
                <Chip
                  key={r}
                  label={r}
                  clickable
                  color={formData.routing_logic?.notify?.includes(r) ? "primary" : "default"}
                  onClick={() => {
                    // Toggle recipient in routing_logic.notify
                    setFormData(prev => {
                      if (!prev) return prev;
                      const prevNotify: string[] = prev.routing_logic?.notify ?? [];
                      const has = prevNotify.includes(r);
                      const nextNotify = has ? prevNotify.filter(x => x !== r) : [...prevNotify, r];
                      return {
                        ...prev,
                        routing_logic: {
                          ...(prev.routing_logic ?? {}),
                          notify: nextNotify,
                        },
                      } as NotificationRoutingRule;
                    });
                  }}
                  sx={{ mr: 1, mb: 1 }}
                />
              ))}
            </Box>
          </Paper>

          <Box sx={{ mt: 2 }}>
            <Button onClick={handleSimulate} disabled={simulating}>
              {simulating ? <CircularProgress size={24} /> : 'Simulate Routing'}
            </Button>
            {simulationResult && (
              <Alert severity="info" sx={{ mt: 2 }}>
                <Typography variant="subtitle2">Simulation Preview</Typography>
                <pre><code>{JSON.stringify(simulationResult, null, 2)}</code></pre>
              </Alert>
            )}
          </Box>

        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}