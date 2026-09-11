import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box,
  Typography,
  Chip,
  Stack,
  CircularProgress,
  Collapse,
  IconButton,
  Tooltip,
  Paper,
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import CloudQueueIcon from '@mui/icons-material/CloudQueue';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { evaluateRulesForGrid } from './ruleGridValidatorService';

interface RuleEvaluationResult {
  ruleId: string;
  ruleName: string;
  ruleDescription?: string;
  passed: boolean;
  isServerSideOnly: boolean;
  serverSideReason?: string;
  error?: string;
}

interface RuleGridEvaluation {
  ruleResults: RuleEvaluationResult[];
  allPassed: boolean;
  summary: string;
}

interface RuleGridValidatorProps {
  boKey: string;
  parentRecord: Record<string, unknown>;
  collectionRows: Record<string, unknown>[];
  tenantId: string;
}

interface RunningTotal {
  sum: number;
  target: number;
  field: string;
  collectionKey: string;
}

function computeRunningTotal(
  boKey: string,
  parentRecord: Record<string, unknown>,
  collectionRows: Record<string, unknown>[]
): RunningTotal | null {
  if (boKey !== 'order') return null;

  const targetQty =
    parentRecord.TargetQuantity ?? parentRecord.target_qty ?? parentRecord.TargetQuantity ?? 0;

  const collectionKey = 'OrderAllocations';
  const rows = collectionRows ?? [];
  const sum = rows.reduce((acc: number, row: Record<string, unknown>) => {
    const val = row.target_qty ?? row.TargetQty ?? 0;
    return acc + (typeof val === 'number' ? val : parseFloat(String(val)) || 0);
  }, 0);

  return { sum, target: Number(targetQty), field: 'target_qty', collectionKey };
}

export function RuleGridValidator({
  boKey,
  parentRecord,
  collectionRows,
  tenantId,
}: RuleGridValidatorProps) {
  const [evaluation, setEvaluation] = useState<RuleGridEvaluation | null>(null);
  const [loading, setLoading] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMounted = useRef(true);

  const runEvaluation = useCallback(async () => {
    if (!boKey || !tenantId) return;
    setLoading(true);
    try {
      const result = await evaluateRulesForGrid(boKey, parentRecord, collectionRows, tenantId);
      if (isMounted.current) {
        setEvaluation(result);
      }
    } catch (err) {
      if (isMounted.current) {
        setEvaluation({
          ruleResults: [],
          allPassed: false,
          summary: err instanceof Error ? err.message : 'Evaluation failed',
        });
      }
    } finally {
      if (isMounted.current) {
        setLoading(false);
      }
    }
  }, [boKey, parentRecord, collectionRows, tenantId]);

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
  }, []);

  useEffect(() => {
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
    }
    debounceTimer.current = setTimeout(() => {
      runEvaluation();
    }, 150);

    return () => {
      if (debounceTimer.current) {
        clearTimeout(debounceTimer.current);
      }
    };
  }, [runEvaluation]);

  const total = computeRunningTotal(boKey, parentRecord, collectionRows);
  const isBalanced = total ? Math.abs(total.sum - total.target) < 0.001 : null;

  if (!total) return null;

  const overallColor = evaluation?.allPassed ? 'success' : 'error';
  const overallIcon = evaluation?.allPassed ? (
    <CheckCircleIcon fontSize="small" />
  ) : (
    <ErrorIcon fontSize="small" />
  );

  return (
    <Paper
      variant="outlined"
      sx={{
        borderColor: overallColor === 'success' ? 'success.main' : 'error.main',
        borderWidth: evaluation ? (overallColor === 'success' ? 1 : 2) : 1,
        bgcolor: overallColor === 'success' ? 'success.50' : 'error.50',
        transition: 'all 0.2s ease',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          px: 2,
          py: 1,
          cursor: 'pointer',
        }}
        onClick={() => setCollapsed((c) => !c)}
      >
        {loading ? (
          <CircularProgress size={16} thickness={5} />
        ) : (
          <Box sx={{ color: overallColor === 'success' ? 'success.main' : 'error.main' }}>
            {overallIcon}
          </Box>
        )}

        <Box sx={{ flex: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
            <Typography variant="body2" sx={{ fontWeight: 700, color: 'text.primary' }}>
              {total.sum.toLocaleString()}
              <Typography component="span" sx={{ color: 'text.secondary', mx: 0.5 }}>
                of
              </Typography>
              {total.target.toLocaleString()}
              <Typography component="span" sx={{ color: 'text.secondary', mx: 0.5 }}>
                {total.field}
              </Typography>
            </Typography>

            {isBalanced === false && (
              <Chip
                label={
                  evaluation?.allPassed
                    ? 'Balanced'
                    : `${total.sum} ≠ ${total.target} — rule violated`
                }
                size="small"
                color={evaluation?.allPassed ? 'success' : 'error'}
                variant="outlined"
                icon={
                  evaluation?.allPassed ? (
                    <CheckCircleIcon sx={{ fontSize: '0.9rem !important' }} />
                  ) : (
                    <ErrorIcon sx={{ fontSize: '0.9rem !important' }} />
                  )
                }
              />
            )}

            {isBalanced === true && (
              <Chip
                label="Allocations balanced"
                size="small"
                color="success"
                variant="outlined"
                icon={<CheckCircleIcon sx={{ fontSize: '0.9rem !important' }} />}
              />
            )}

            {evaluation?.ruleResults
              .filter((r) => !r.isServerSideOnly)
              .map((r) => (
                <Chip
                  key={r.ruleId}
                  label={r.ruleName}
                  size="small"
                  color={r.passed ? 'success' : 'error'}
                  variant={r.passed ? 'outlined' : 'filled'}
                  icon={
                    r.passed ? (
                      <CheckCircleIcon sx={{ fontSize: '0.9rem !important' }} />
                    ) : (
                      <ErrorIcon sx={{ fontSize: '0.9rem !important' }} />
                    )
                  }
                />
              ))}

            {evaluation?.ruleResults
              .filter((r) => r.isServerSideOnly)
              .map((r) => (
                <Tooltip key={r.ruleId} title={r.serverSideReason ?? 'Evaluated on save'}>
                  <Chip
                    label={r.ruleName}
                    size="small"
                    color="default"
                    variant="outlined"
                    icon={<CloudQueueIcon sx={{ fontSize: '0.9rem !important' }} />}
                  />
                </Tooltip>
              ))}
          </Stack>

          {evaluation && (
            <Typography variant="caption" color="text.secondary" sx={{ mt: 0.25, display: 'block' }}>
              {evaluation.summary}
              {evaluation.ruleResults.some((r) => r.error) &&
                ` · ${evaluation.ruleResults.find((r) => r.error)?.error}`}
            </Typography>
          )}
        </Box>

        <Tooltip title={collapsed ? 'Expand' : 'Collapse'}>
          <IconButton size="small">
            {collapsed ? <ExpandMoreIcon fontSize="small" /> : <ExpandLessIcon fontSize="small" />}
          </IconButton>
        </Tooltip>
      </Box>

      <Collapse in={!collapsed && evaluation !== null}>
        <Box sx={{ px: 2, pb: 1.5, borderTop: '1px solid', borderColor: 'divider', pt: 1 }}>
          {evaluation?.ruleResults
            .filter((r) => !r.isServerSideOnly)
            .map((r) => (
              <Box key={r.ruleId} sx={{ mb: 0.5 }}>
                <Stack direction="row" spacing={0.75} alignItems="flex-start">
                  <Box sx={{ mt: 0.35, color: r.passed ? 'success.main' : 'error.main' }}>
                    {r.passed ? (
                      <CheckCircleIcon sx={{ fontSize: '0.85rem' }} />
                    ) : (
                      <ErrorIcon sx={{ fontSize: '0.85rem' }} />
                    )}
                  </Box>
                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 600 }}>
                      {r.ruleName}
                    </Typography>
                    {r.ruleDescription && (
                      <Typography variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>
                        — {r.ruleDescription}
                      </Typography>
                    )}
                    {r.error && (
                      <Typography variant="caption" color="error.main" sx={{ display: 'block' }}>
                        Error: {r.error}
                      </Typography>
                    )}
                  </Box>
                </Stack>
              </Box>
            ))}

          {evaluation?.ruleResults.some((r) => r.isServerSideOnly) && (
            <Box sx={{ mt: 1, pt: 1, borderTop: '1px dashed', borderColor: 'divider' }}>
              <Typography variant="caption" color="text.disabled" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <CloudQueueIcon sx={{ fontSize: '0.85rem' }} />
                Server-side rules — evaluated on save
              </Typography>
              {evaluation.ruleResults
                .filter((r) => r.isServerSideOnly)
                .map((r) => (
                  <Box key={r.ruleId} sx={{ mt: 0.5 }}>
                    <Typography variant="caption" color="text.secondary">
                      {r.ruleName}
                      {r.serverSideReason && ` · ${r.serverSideReason}`}
                    </Typography>
                  </Box>
                ))}
            </Box>
          )}

          {(!evaluation || evaluation.ruleResults.length === 0) && (
            <Typography variant="caption" color="text.disabled">
              No active client-evaluable rules for this entity
            </Typography>
          )}
        </Box>
      </Collapse>
    </Paper>
  );
}
