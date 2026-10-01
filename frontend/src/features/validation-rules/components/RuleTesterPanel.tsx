import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  CircularProgress,
  Alert,
  Chip,
  Paper,
  Divider,
  Grid,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import BlockIcon from '@mui/icons-material/Block';
import WarningIcon from '@mui/icons-material/Warning';
import { validationRulesApi, type RecordEvaluationResponse } from '../api';

interface RuleTesterPanelProps {
  defaultBOName?: string;
  defaultDomain?: string;
}

export const RuleTesterPanel: React.FC<RuleTesterPanelProps> = ({
  defaultBOName = 'order',
  defaultDomain = 'validation',
}) => {
  const [boName, setBOName] = useState(defaultBOName);
  const [timing, setTiming] = useState('pre_write');
  const [domain, setDomain] = useState(defaultDomain);
  const [payloadText, setPayloadText] = useState(
    JSON.stringify(
      {
        order_id: 'ord-1001',
        TargetQuantity: 1500,
        LimitPrice: 105.25,
        OrderSide: 'BUY',
        account_status: 'ACTIVE',
      },
      null,
      2
    )
  );
  const [evaluating, setEvaluating] = useState(false);
  const [result, setResult] = useState<RecordEvaluationResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleRunEvaluation = async () => {
    setEvaluating(true);
    setError(null);
    try {
      const record = JSON.parse(payloadText);
      const res = await validationRulesApi.evaluateRecord(
        boName,
        record,
        domain || undefined,
        timing || undefined
      );
      setResult(res);
    } catch (err: any) {
      setError(err.message || 'Evaluation failed');
    } finally {
      setEvaluating(false);
    }
  };

  return (
    <Card variant="outlined" sx={{ borderRadius: 2 }}>
      <CardContent>
        <Typography variant="h6" fontWeight="bold" gutterBottom>
          Interactive Rule Evaluator & Payload Simulator
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Test ad-hoc JSON payloads against in-memory compiled rule snapshots with sub-millisecond evaluation.
        </Typography>

        <Grid container spacing={2}>
          <Grid item xs={12} md={6}>
            <Stack spacing={2}>
              <Stack direction="row" spacing={2}>
                <FormControl size="small" fullWidth>
                  <InputLabel>Business Object</InputLabel>
                  <Select
                    value={boName}
                    label="Business Object"
                    onChange={(e) => setBOName(e.target.value)}
                  >
                    <MenuItem value="order">Order (orm.order)</MenuItem>
                    <MenuItem value="execution">Execution (orm.execution)</MenuItem>
                    <MenuItem value="security">Security (oms.security)</MenuItem>
                    <MenuItem value="account">Account (oms.account)</MenuItem>
                    <MenuItem value="party">Party / Customer (master.customer)</MenuItem>
                  </Select>
                </FormControl>

                <FormControl size="small" sx={{ minWidth: 150 }}>
                  <InputLabel>Timing</InputLabel>
                  <Select
                    value={timing}
                    label="Timing"
                    onChange={(e) => setTiming(e.target.value)}
                  >
                    <MenuItem value="pre_write">Pre-Write</MenuItem>
                    <MenuItem value="reconcile">Reconcile</MenuItem>
                  </Select>
                </FormControl>
              </Stack>

              <TextField
                label="Evaluation Payload (JSON)"
                multiline
                rows={12}
                value={payloadText}
                onChange={(e) => setPayloadText(e.target.value)}
                sx={{ fontFamily: 'monospace' }}
                fullWidth
              />

              <Button
                variant="contained"
                color="primary"
                startIcon={<PlayArrowIcon />}
                onClick={handleRunEvaluation}
                disabled={evaluating}
              >
                {evaluating ? <CircularProgress size={20} /> : 'Run Evaluation'}
              </Button>
            </Stack>
          </Grid>

          <Grid item xs={12} md={6}>
            <Paper variant="outlined" sx={{ p: 2, height: '100%', bgcolor: 'background.default' }}>
              <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
                Evaluation Results
              </Typography>
              <Divider sx={{ mb: 2 }} />

              {error && <Alert severity="error">{error}</Alert>}

              {result && (
                <Stack spacing={2}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    {result.valid ? (
                      <Chip icon={<CheckCircleIcon />} label="VALID (PASS)" color="success" />
                    ) : result.blocked ? (
                      <Chip icon={<BlockIcon />} label="BLOCKED (FAILED BLOCK RULE)" color="error" />
                    ) : (
                      <Chip icon={<WarningIcon />} label="WARN (FAILED WARN RULE)" color="warning" />
                    )}
                    <Chip label={`Evaluated: ${result.evaluated_rules_count} Rules`} size="small" />
                  </Stack>

                  {result.error === 'ERR_SERVER_CONTEXT_REQUIRED' && (
                    <Alert severity="warning">
                      <Typography variant="body2" fontWeight="bold">Server Context Required:</Typography>
                      <Typography variant="caption">{result.hint}</Typography>
                      {result.missing_context_fields && (
                        <Box sx={{ mt: 1 }}>
                          {result.missing_context_fields.map((f, i) => (
                            <Chip key={i} label={f} size="small" color="secondary" sx={{ mr: 0.5 }} />
                          ))}
                        </Box>
                      )}
                    </Alert>
                  )}

                  {result.violations && result.violations.length > 0 && (
                    <Box>
                      <Typography variant="subtitle2" color="error" fontWeight="bold">
                        Violations ({result.violations.length}):
                      </Typography>
                      <Stack spacing={1} sx={{ mt: 1 }}>
                        {result.violations.map((v, i) => (
                          <Paper key={i} sx={{ p: 1.5, borderLeft: '4px solid #ef4444' }}>
                            <Stack direction="row" justifyContent="space-between" alignItems="center">
                              <Typography variant="body2" fontWeight="bold">{v.rule_name}</Typography>
                              <Chip label={v.severity} size="small" color={v.severity === 'BLOCK' ? 'error' : 'warning'} />
                            </Stack>
                            <Typography variant="caption" color="text.secondary" display="block">
                              {v.message || 'Condition not satisfied'}
                            </Typography>
                          </Paper>
                        ))}
                      </Stack>
                    </Box>
                  )}

                  {result.rule_errors && result.rule_errors.length > 0 && (
                    <Box>
                      <Typography variant="subtitle2" color="warning.main" fontWeight="bold">
                        Rule Errors ({result.rule_errors.length}):
                      </Typography>
                      <Stack spacing={1} sx={{ mt: 1 }}>
                        {result.rule_errors.map((re, i) => (
                          <Paper key={i} sx={{ p: 1.5, bgcolor: '#fffbeb', borderLeft: '4px solid #f59e0b' }}>
                            <Typography variant="body2" fontWeight="bold">{re.rule_name} ({re.rule_key})</Typography>
                            <Typography variant="caption" color="error">{re.message}</Typography>
                          </Paper>
                        ))}
                      </Stack>
                    </Box>
                  )}

                  {result.valid && (
                    <Alert severity="success">
                      Record satisfied all {result.evaluated_rules_count} active validation rules with zero blocking or warning conditions.
                    </Alert>
                  )}
                </Stack>
              )}

              {!result && !error && (
                <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic', mt: 4, textAlign: 'center' }}>
                  Click "Run Evaluation" to test the payload against active rules.
                </Typography>
              )}
            </Paper>
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  );
};
