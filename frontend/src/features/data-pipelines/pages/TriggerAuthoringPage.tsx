import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  TextField,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Chip,
  OutlinedInput,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  ToggleButton,
  ToggleButtonGroup,
  FormHelperText,
  Breadcrumbs,
  Link,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Switch,
  FormControlLabel,
  IconButton,
  Tooltip,
} from '@mui/material';
import { Save, X, Zap, PlayCircle, RefreshCw } from 'lucide-react';
import { fetchAPI } from '@/api';
import { PipelinePicker } from '../components/PipelinePicker';

interface ValidationRule {
  id: string;
  rule_name: string;
  rule_type: string;
  target_entity: string;
  description?: string;
}

interface TriggerFormData {
  trigger_type: string;
  target_entity: string;
  step_name: string;
  rule_ids: string[];
  pipeline_id: string;
  dispatch_mode: 'sync' | 'async';
}

interface TriggerRow {
  id: string;
  tenant_id: string;
  trigger_type: string;
  target_entity: string;
  step_name?: string;
  rule_ids: string;
  is_active: boolean;
  pipeline_id?: string;
  dispatch_mode?: string;
  last_fired_at?: string;
}

const TRIGGER_TYPES = [
  { value: 'row_insert', label: 'Row Inserted' },
  { value: 'row_update', label: 'Row Updated' },
  { value: 'row_delete', label: 'Row Deleted' },
  { value: 'page_load', label: 'Page Loaded' },
  { value: 'page_save', label: 'Page Saved' },
  { value: 'field_change', label: 'Field Changed' },
  { value: 'api_request', label: 'API Request' },
  { value: 'api_response', label: 'API Response' },
];

const EMPTY_FORM: TriggerFormData = {
  trigger_type: '',
  target_entity: '',
  step_name: '',
  rule_ids: [],
  pipeline_id: '',
  dispatch_mode: 'sync',
};

export const TriggerAuthoringPage: React.FC = () => {
  const navigate = useNavigate();
  const [form, setForm] = useState<TriggerFormData>(EMPTY_FORM);
  const [rules, setRules] = useState<ValidationRule[]>([]);
  const [rulesLoading, setRulesLoading] = useState(true);
  const [rulesError, setRulesError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [triggers, setTriggers] = useState<TriggerRow[]>([]);
  const [triggersLoading, setTriggersLoading] = useState(false);
  const [triggersError, setTriggersError] = useState<string | null>(null);

  useEffect(() => {
    fetchAPI<{ rules: ValidationRule[] }>('/api/validation-rules?limit=100')
      .then((res) => setRules(res.rules ?? []))
      .catch(() => setRulesError('Failed to load validation rules'))
      .finally(() => setRulesLoading(false));
  }, []);

  const fetchTriggers = (entity?: string) => {
    setTriggersLoading(true);
    setTriggersError(null);
    const url = entity
      ? `/api/admin/validation-triggers?target_entity=${encodeURIComponent(entity)}`
      : '/api/admin/validation-triggers';
    fetchAPI<TriggerRow[]>(url)
      .then((res) => setTriggers(Array.isArray(res) ? res : []))
      .catch(() => setTriggersError('Failed to load triggers'))
      .finally(() => setTriggersLoading(false));
  };

  useEffect(() => {
    fetchTriggers();
  }, []);

  useEffect(() => {
    if (form.target_entity) {
      fetchTriggers(form.target_entity);
    } else {
      fetchTriggers();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [form.target_entity]);

  const handleToggleActive = async (trigger: TriggerRow) => {
    const newActive = !trigger.is_active;
    setTriggers((prev) =>
      prev.map((t) => (t.id === trigger.id ? { ...t, is_active: newActive } : t)),
    );
    try {
      await fetchAPI(`/api/v1/triggers/${trigger.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: newActive }),
      });
    } catch {
      setTriggers((prev) =>
        prev.map((t) => (t.id === trigger.id ? { ...t, is_active: !newActive } : t)),
      );
    }
  };

  const handleChange = (field: keyof TriggerFormData, value: string | string[]) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const handlePipelineChange = (pipelineId: string, dispatchMode: 'sync' | 'async') => {
    setForm((prev) => ({ ...prev, pipeline_id: pipelineId, dispatch_mode: dispatchMode }));
  };

  const handleSubmit = async () => {
    setSaveError(null);
    if (!form.trigger_type || !form.target_entity || form.rule_ids.length === 0) {
      setSaveError('Trigger type, target entity, and at least one rule are required');
      return;
    }

    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        trigger_type: form.trigger_type,
        target_entity: form.target_entity,
        rule_ids: form.rule_ids,
        dispatch_mode: form.dispatch_mode,
      };
      if (form.step_name) payload.step_name = form.step_name;
      if (form.pipeline_id) payload.pipeline_id = form.pipeline_id;

      await fetchAPI('/api/admin/validation-triggers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      navigate('/core/validation-rules');
    } catch (err: any) {
      setSaveError(err?.message ?? 'Failed to create trigger');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ p: 3, maxWidth: 800, mx: 'auto' }}>
      <Box sx={{ mb: 3 }}>
        <Breadcrumbs>
          <Link
            component="button"
            variant="body2"
            onClick={() => navigate('/pipelines/studio')}
            sx={{ cursor: 'pointer', textDecoration: 'none' }}
          >
            Pipelines
          </Link>
          <Typography variant="body2" color="text.primary">
            New Validation Trigger
          </Typography>
        </Breadcrumbs>
      </Box>

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 3 }}>
        <Box
          sx={{
            width: 40,
            height: 40,
            borderRadius: '8px',
            background: 'linear-gradient(135deg, #7c3aed, #5b21b6)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#ffffff',
          }}
        >
          <Zap size={20} />
        </Box>
        <Box>
          <Typography variant="h6" fontWeight={800}>
            New Validation Trigger
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Bind a pipeline to a BO row event — fires validation rules on write
          </Typography>
        </Box>
      </Box>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="subtitle2" fontWeight={700} color="text.secondary">
              Existing Triggers
            </Typography>
            <Button size="small" startIcon={<RefreshCw size={14} />} onClick={() => fetchTriggers(form.target_entity || undefined)}>
              Refresh
            </Button>
          </Box>
          {triggersLoading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
              <CircularProgress size={20} />
            </Box>
          ) : triggersError ? (
            <Alert severity="error" sx={{ mb: 1 }}>{triggersError}</Alert>
          ) : triggers.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              No triggers found{form.target_entity ? ` for "${form.target_entity}"` : ''}.
            </Typography>
          ) : (
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Trigger Type</TableCell>
                    <TableCell>Target Entity</TableCell>
                    <TableCell>Pipeline</TableCell>
                    <TableCell>Dispatch</TableCell>
                    <TableCell>Last Fired</TableCell>
                    <TableCell>Active</TableCell>
                    <TableCell align="right">Runs</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {triggers.map((trig) => (
                    <TableRow key={trig.id} hover>
                      <TableCell>
                        <Chip label={trig.trigger_type} size="small" sx={{ fontSize: '0.7rem' }} />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                          {trig.target_entity}
                        </Typography>
                        {trig.step_name && (
                          <Typography variant="caption" color="text.secondary">
                            step: {trig.step_name}
                          </Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        {trig.pipeline_id ? (
                          <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.7rem' }}>
                            {trig.pipeline_id.slice(0, 8)}…
                          </Typography>
                        ) : (
                          <Typography variant="caption" color="text.secondary">—</Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption">{trig.dispatch_mode || 'sync'}</Typography>
                      </TableCell>
                      <TableCell>
                        {trig.last_fired_at ? (
                          <Typography variant="caption">
                            {new Date(trig.last_fired_at).toLocaleString()}
                          </Typography>
                        ) : (
                          <Typography variant="caption" color="text.secondary">Never</Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        <FormControlLabel
                          control={
                            <Switch
                              size="small"
                              checked={trig.is_active}
                              onChange={() => handleToggleActive(trig)}
                            />
                          }
                          label={trig.is_active ? 'On' : 'Off'}
                          sx={{ mr: 0 }}
                        />
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="View runs">
                          <IconButton
                            size="small"
                            onClick={() => navigate(`/pipelines/runs?trigger=${trig.id}`)}
                          >
                            <PlayCircle size={16} />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      <Card sx={{ mb: 3 }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
          {saveError && (
            <Alert severity="error" onClose={() => setSaveError(null)}>
              {saveError}
            </Alert>
          )}

          <FormControl fullWidth required size="small">
            <InputLabel>Trigger Type</InputLabel>
            <Select
              value={form.trigger_type}
              label="Trigger Type"
              onChange={(e) => handleChange('trigger_type', e.target.value)}
            >
              {TRIGGER_TYPES.map((t) => (
                <MenuItem key={t.value} value={t.value}>
                  {t.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <TextField
            label="Target Entity"
            required
            size="small"
            fullWidth
            placeholder="e.g. oms.trade_order, master.customer"
            value={form.target_entity}
            onChange={(e) => handleChange('target_entity', e.target.value)}
            helperText="The Business Object entity this trigger watches"
          />

          <TextField
            label="Step Name"
            size="small"
            fullWidth
            placeholder="Optional: restrict to a specific pipeline step"
            value={form.step_name}
            onChange={(e) => handleChange('step_name', e.target.value)}
          />

          <FormControl fullWidth required size="small" error={!!rulesError}>
            <InputLabel>Validation Rules</InputLabel>
            <Select
              multiple
              value={form.rule_ids}
              onChange={(e) => {
                const val = e.target.value as string[];
                handleChange('rule_ids', val);
              }}
              input={<OutlinedInput label="Validation Rules" />}
              renderValue={(selected) => (
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {selected.map((id) => {
                    const rule = rules.find((r) => r.id === id);
                    return (
                      <Chip key={id} label={rule?.rule_name ?? id} size="small" />
                    );
                  })}
                </Box>
              )}
              endAdornment={
                rulesLoading ? <CircularProgress size={16} sx={{ mr: 1 }} /> : null
              }
            >
              {rulesLoading && <MenuItem value="">Loading...</MenuItem>}
              {!rulesLoading && rules.length === 0 && (
                <MenuItem value="" disabled>
                  No validation rules found
                </MenuItem>
              )}
              {!rulesLoading &&
                rules.map((rule) => (
                  <MenuItem key={rule.id} value={rule.id}>
                    <Box>
                      <Typography variant="body2">{rule.rule_name}</Typography>
                      <Typography variant="caption" color="text.secondary">
                        {rule.rule_type} · {rule.target_entity}
                      </Typography>
                    </Box>
                  </MenuItem>
                ))}
            </Select>
            {rulesError && <FormHelperText>{rulesError}</FormHelperText>}
            <FormHelperText>
              Select one or more rules to execute when this trigger fires
            </FormHelperText>
          </FormControl>

          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
              Pipeline (optional — for durable async dispatch)
            </Typography>
            <PipelinePicker
              value={form.pipeline_id}
              onChange={handlePipelineChange}
            />
          </Box>

          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block' }}>
              Dispatch Mode
            </Typography>
            <ToggleButtonGroup
              value={form.dispatch_mode}
              exclusive
              onChange={(_, v) => v && handleChange('dispatch_mode', v)}
              size="small"
            >
              <ToggleButton value="sync">Sync</ToggleButton>
              <ToggleButton value="async">Async</ToggleButton>
            </ToggleButtonGroup>
            <FormHelperText sx={{ mt: 0.5 }}>
              {form.dispatch_mode === 'sync'
                ? 'Validation runs inline; a failure blocks the BO write'
                : 'Validation queued via outbox; write is not blocked'}
            </FormHelperText>
          </Box>
        </CardContent>
      </Card>

      <Box sx={{ display: 'flex', gap: 1.5, justifyContent: 'flex-end' }}>
        <Button
          variant="outlined"
          startIcon={<X size={16} />}
          onClick={() => navigate('/pipelines/studio')}
        >
          Cancel
        </Button>
        <Button
          variant="contained"
          startIcon={saving ? <CircularProgress size={16} color="inherit" /> : <Save size={16} />}
          onClick={handleSubmit}
          disabled={saving || rulesLoading}
          sx={{
            background: 'linear-gradient(135deg, #7c3aed, #5b21b6)',
            fontWeight: 700,
          }}
        >
          {saving ? 'Saving...' : 'Save Trigger'}
        </Button>
      </Box>
    </Box>
  );
};
