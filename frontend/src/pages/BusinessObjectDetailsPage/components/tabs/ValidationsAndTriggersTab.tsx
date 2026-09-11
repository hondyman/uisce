import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Switch,
  Tooltip,
  CircularProgress,
  Alert,
  Link as MuiLink,
  Divider,
} from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import apiClient from '../../../../utils/apiClient';

// Validations & Triggers - a single BO-scoped surface for the two things
// govern what happens on a write to this BO. Validations are real today
// (backend/internal/metadata/shadow_evaluation.go); Triggers is a
// designated placeholder for the pipeline trigger list/toggle once the
// outbox emitter lands - built now so that feature has somewhere to go,
// rather than this tab needing to be split into two later.
interface ValidationsAndTriggersTabProps {
  businessObject: any;
}

interface RuleRow {
  id: string;
  name: string;
  bo_name: string;
  severity: string;
  timing: string;
  is_active: boolean;
}

interface HealthRow {
  rule_id: string;
  rule_name: string;
  bo_key: string;
  severity: string;
  violation_count: number;
  rule_error_count: number;
  blocked_count: number;
  logged_count: number;
  last_fired_at: string | null;
  suspect: boolean;
  suspect_reason?: string;
}

export function ValidationsAndTriggersTab({ businessObject }: ValidationsAndTriggersTabProps) {
  const navigate = useNavigate();
  const boKey: string | undefined = businessObject?.key;
  const editorPath = boKey ? `/core/validation-rules/editor?bo_name=${encodeURIComponent(boKey)}` : '#';
  const [rules, setRules] = useState<RuleRow[]>([]);
  const [health, setHealth] = useState<Record<string, HealthRow>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!boKey) return;
    setLoading(true);
    setError(null);
    try {
      const [ruleRes, healthRes] = await Promise.all([
        apiClient<{ validationRules: RuleRow[] }>(
          `/validation-rule-nodes?bo_name=${encodeURIComponent(boKey)}`
        ),
        apiClient<{ health: HealthRow[] }>('/validation-rule-nodes/health'),
      ]);
      setRules(ruleRes.validationRules || []);
      const byRuleId: Record<string, HealthRow> = {};
      for (const h of healthRes.health || []) {
        if (h.bo_key === boKey) byRuleId[h.rule_id] = h;
      }
      setHealth(byRuleId);
    } catch (err: any) {
      setError(err?.message || 'Failed to load validation rules');
    } finally {
      setLoading(false);
    }
  }, [boKey]);

  useEffect(() => {
    load();
  }, [load]);

  const handleToggleActive = async (rule: RuleRow) => {
    setTogglingId(rule.id);
    try {
      await apiClient(`/validation-rule-nodes/${rule.id}/active`, {
        method: 'PATCH',
        body: JSON.stringify({ active: !rule.is_active }),
      });
      await load();
    } catch (err: any) {
      setError(err?.message || 'Failed to update rule');
    } finally {
      setTogglingId(null);
    }
  };

  if (!boKey) {
    return <Alert severity="info">Save this business object before configuring validations.</Alert>;
  }

  return (
    <Box sx={{ p: 2 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h6">Validations</Typography>
          <Typography variant="body2" color="text.secondary">
            Rules governing writes to <code>{boKey}</code> - evaluated on every create/update. A rule
            producing violations or rule errors on nearly every recent write is flagged as suspect below.
          </Typography>
        </Box>
        <MuiLink
          component="button"
          onClick={() => navigate(editorPath)}
          underline="hover"
        >
          Open in editor →
        </MuiLink>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
          <CircularProgress size={24} />
        </Box>
      ) : rules.length === 0 ? (
        <Alert severity="info">No validation rules target this business object yet.</Alert>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Rule</TableCell>
                <TableCell>Severity</TableCell>
                <TableCell>Timing</TableCell>
                <TableCell>Active</TableCell>
                <TableCell align="right">Violations</TableCell>
                <TableCell align="right">Blocked / Logged</TableCell>
                <TableCell align="right">Rule errors</TableCell>
                <TableCell>Last fired</TableCell>
                <TableCell>Health</TableCell>
                <TableCell />
              </TableRow>
            </TableHead>
            <TableBody>
              {rules.map((rule) => {
                const h = health[rule.id];
                return (
                  <TableRow key={rule.id} hover>
                    <TableCell>{rule.name}</TableCell>
                    <TableCell>
                      <Chip
                        label={rule.severity}
                        size="small"
                        color={rule.severity === 'BLOCK' ? 'error' : 'warning'}
                        variant={rule.severity === 'BLOCK' ? 'filled' : 'outlined'}
                      />
                    </TableCell>
                    <TableCell>{rule.timing}</TableCell>
                    <TableCell>
                      <Switch
                        size="small"
                        checked={rule.is_active}
                        disabled={togglingId === rule.id}
                        onChange={() => handleToggleActive(rule)}
                      />
                    </TableCell>
                    <TableCell align="right">{h?.violation_count ?? 0}</TableCell>
                    <TableCell align="right">
                      {h ? `${h.blocked_count} / ${h.logged_count}` : '0 / 0'}
                    </TableCell>
                    <TableCell align="right">
                      {h && h.rule_error_count > 0 ? (
                        <Chip label={h.rule_error_count} size="small" color="error" variant="outlined" />
                      ) : (
                        0
                      )}
                    </TableCell>
                    <TableCell>
                      {h?.last_fired_at ? new Date(h.last_fired_at).toLocaleString() : '—'}
                    </TableCell>
                    <TableCell>
                      {h?.suspect ? (
                        <Tooltip title={h.suspect_reason || 'Possibly broken'}>
                          <Chip
                            icon={<WarningAmberIcon fontSize="small" />}
                            label="Suspect"
                            size="small"
                            color="error"
                            variant="outlined"
                          />
                        </Tooltip>
                      ) : (
                        <Chip label="OK" size="small" variant="outlined" />
                      )}
                    </TableCell>
                    <TableCell>
                      <MuiLink
                        component="button"
                        onClick={() => navigate(editorPath)}
                        underline="hover"
                        sx={{ fontSize: '0.8rem' }}
                      >
                        Edit
                      </MuiLink>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Divider sx={{ my: 3 }} />

      <Typography variant="h6" sx={{ mb: 1 }}>
        Triggers
      </Typography>
      <Alert severity="info" variant="outlined">
        No pipeline trigger integration yet - this section is the designated home for the trigger
        list and enable/disable toggle once the outbox emitter lands, so that feature won't need a
        second page later.
      </Alert>
    </Box>
  );
}
