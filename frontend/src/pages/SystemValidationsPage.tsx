import { useEffect, useMemo, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
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
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormControlLabel,
  TextField,
  Link as MuiLink,
} from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import apiClient from '../utils/apiClient';

// System-wide validation rules view - every validation-rule-node across
// every BO, one place, filterable. The per-BO "Validations & Triggers"
// tab (BusinessObjectDetailsPage) is the scoped view of the same data;
// this is the unscoped one, for "what rules exist tenant-wide, which
// ones look broken" without visiting every BO in turn. Distinct from
// /core/validation-rules (ValidationRulesBuilderPage), which is the
// older catalog_validation_rules-era engine and doesn't touch
// /api/validation-rule-nodes at all.
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

export default function SystemValidationsPage() {
  const navigate = useNavigate();
  const [rules, setRules] = useState<RuleRow[]>([]);
  const [health, setHealth] = useState<Record<string, HealthRow>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const [boFilter, setBoFilter] = useState('all');
  const [severityFilter, setSeverityFilter] = useState('all');
  const [showInactive, setShowInactive] = useState(false);
  const [suspectOnly, setSuspectOnly] = useState(false);
  const [search, setSearch] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [ruleRes, healthRes] = await Promise.all([
        apiClient<{ validationRules: RuleRow[] }>(
          `/validation-rule-nodes?include_inactive=${showInactive ? 'true' : 'false'}`
        ),
        apiClient<{ health: HealthRow[] }>('/validation-rule-nodes/health'),
      ]);
      setRules(ruleRes.validationRules || []);
      const byRuleId: Record<string, HealthRow> = {};
      for (const h of healthRes.health || []) byRuleId[h.rule_id] = h;
      setHealth(byRuleId);
    } catch (err: any) {
      setError(err?.message || 'Failed to load validation rules');
    } finally {
      setLoading(false);
    }
  }, [showInactive]);

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

  const boOptions = useMemo(() => {
    const set = new Set(rules.map((r) => r.bo_name));
    return Array.from(set).sort();
  }, [rules]);

  const filtered = useMemo(() => {
    return rules.filter((r) => {
      if (boFilter !== 'all' && r.bo_name !== boFilter) return false;
      if (severityFilter !== 'all' && r.severity !== severityFilter) return false;
      if (search && !r.name.toLowerCase().includes(search.toLowerCase())) return false;
      if (suspectOnly && !health[r.id]?.suspect) return false;
      return true;
    });
  }, [rules, boFilter, severityFilter, search, suspectOnly, health]);

  const suspectCount = rules.filter((r) => health[r.id]?.suspect).length;
  const inactiveCount = rules.filter((r) => !r.is_active).length;

  return (
    <Container maxWidth="xl" sx={{ py: 3 }}>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 0.5 }}>
        Validation Rules
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Every validation-rule-node across every business object - {rules.length} rule
        {rules.length === 1 ? '' : 's'} shown
        {showInactive ? ` (${inactiveCount} retired)` : ''}
        {suspectCount > 0 ? `, ${suspectCount} flagged as possibly broken` : ''}.
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Stack direction="row" spacing={2} flexWrap="wrap" alignItems="center" useFlexGap>
          <TextField
            size="small"
            label="Search rule name"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            sx={{ minWidth: 220 }}
          />
          <FormControl size="small" sx={{ minWidth: 160 }}>
            <InputLabel>Business Object</InputLabel>
            <Select value={boFilter} label="Business Object" onChange={(e) => setBoFilter(e.target.value)}>
              <MenuItem value="all">All</MenuItem>
              {boOptions.map((bo) => (
                <MenuItem key={bo} value={bo}>
                  {bo}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 140 }}>
            <InputLabel>Severity</InputLabel>
            <Select value={severityFilter} label="Severity" onChange={(e) => setSeverityFilter(e.target.value)}>
              <MenuItem value="all">All</MenuItem>
              <MenuItem value="BLOCK">BLOCK</MenuItem>
              <MenuItem value="WARN">WARN</MenuItem>
            </Select>
          </FormControl>
          <FormControlLabel
            control={<Switch checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} />}
            label="Show retired rules"
          />
          <FormControlLabel
            control={<Switch checked={suspectOnly} onChange={(e) => setSuspectOnly(e.target.checked)} />}
            label="Suspect only"
          />
        </Stack>
      </Paper>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
          <CircularProgress size={24} />
        </Box>
      ) : filtered.length === 0 ? (
        <Alert severity="info">No rules match the current filters.</Alert>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Rule</TableCell>
                <TableCell>BO</TableCell>
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
              {filtered.map((rule) => {
                const h = health[rule.id];
                return (
                  <TableRow key={rule.id} hover sx={{ opacity: rule.is_active ? 1 : 0.55 }}>
                    <TableCell>
                      {rule.name}
                      {!rule.is_active && (
                        <Chip label="retired" size="small" variant="outlined" sx={{ ml: 1 }} />
                      )}
                    </TableCell>
                    <TableCell>
                      <code>{rule.bo_name}</code>
                    </TableCell>
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
                        onClick={() =>
                          navigate(`/core/validation-rules/editor?bo_name=${encodeURIComponent(rule.bo_name)}`)
                        }
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
    </Container>
  );
}
