import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Card, CardContent, Typography, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, Chip, Button, Stack, CircularProgress, Alert, GridLegacy as Grid
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import RemoveIcon from '@mui/icons-material/Remove';
import CompareArrowsIcon from '@mui/icons-material/CompareArrows';
import HistoryIcon from '@mui/icons-material/History';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { mdmScoringApi, type ShadowValidationReport, type ShadowVendorComparison } from '../api';

export interface ShadowComparatorProps {
  asOf?: string;
  onRefresh?: () => void;
}

export const ShadowComparator: React.FC<ShadowComparatorProps> = ({ asOf }) => {
  const [data, setData] = useState<ShadowValidationReport | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchShadowReport = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await mdmScoringApi.shadowEval(asOf);
      setData(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to load shadow validation report');
    } finally {
      setLoading(false);
    }
  }, [asOf]);

  useEffect(() => {
    fetchShadowReport();
  }, [fetchShadowReport]);

  const renderRankDeltaBadge = (comp: ShadowVendorComparison) => {
    if (comp.rank_delta === null || comp.rank_delta === undefined) {
      return (
        <Chip
          size="small"
          variant="outlined"
          label="— (Degenerate Old Basis)"
          sx={{ fontStyle: 'italic', fontSize: '0.72rem', color: 'text.secondary' }}
        />
      );
    }
    if (comp.rank_delta > 0) {
      return (
        <Chip
          size="small"
          color="success"
          icon={<ArrowUpwardIcon fontSize="small" />}
          label={`+${comp.rank_delta} (Up)`}
        />
      );
    }
    if (comp.rank_delta < 0) {
      return (
        <Chip
          size="small"
          color="error"
          icon={<ArrowDownwardIcon fontSize="small" />}
          label={`${comp.rank_delta} (Down)`}
        />
      );
    }
    return (
      <Chip
        size="small"
        variant="outlined"
        icon={<RemoveIcon fontSize="small" />}
        label="0 (Unchanged)"
      />
    );
  };

  const renderStateChip = (comp: ShadowVendorComparison) => {
    switch (comp.stability) {
      case 'STABLE':
        return (
          <Chip
            size="small"
            variant="outlined"
            label="STABLE"
            sx={{ fontWeight: 600, color: 'text.secondary', borderColor: 'divider' }}
          />
        );
      case 'TIE_BOUND':
        return (
          <Chip
            size="small"
            color="warning"
            label={comp.neighbor_gap !== null && comp.neighbor_gap !== undefined ? `TIE_BOUND (${comp.neighbor_gap.toFixed(1)} pt gap)` : 'TIE_BOUND'}
            sx={{ fontWeight: 700 }}
          />
        );
      case 'MOVED':
        return (
          <Chip
            size="small"
            color="primary"
            label="MOVED"
            sx={{ fontWeight: 700 }}
          />
        );
      default:
        return <Chip size="small" label={comp.stability} />;
    }
  };

  const isModelChanged = data?.basis?.weights_changed || data?.comparisons.some(c => c.attribution === 'MODEL_CHANGED');
  const isOldDegenerate = data?.basis?.old_ranking === 'DEGENERATE';

  return (
    <Box sx={{ p: 2 }}>
      {/* Header and Controls */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <CompareArrowsIcon color="primary" sx={{ fontSize: 30 }} />
            <Typography variant="h5" sx={{ fontWeight: 700 }}>
              Dual-Run Shadow Validation Comparator
            </Typography>
            {data && (
              <Stack direction="row" spacing={1}>
                <Chip
                  label={data.is_stable ? 'STABLE RANKING' : 'RANK DRIFT DETECTED'}
                  color={data.is_stable ? 'success' : 'warning'}
                  sx={{ fontWeight: 700 }}
                />
                <Chip
                  label={`Scale: ${data.basis?.scale_factor || 1.0}×`}
                  variant="outlined"
                  size="small"
                />
              </Stack>
            )}
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Dual-run Shapley decomposition isolating model/weight shifts from underlying vendor data drift.
          </Typography>
        </Box>

        <Button
          variant="outlined"
          startIcon={<RefreshIcon />}
          onClick={fetchShadowReport}
          disabled={loading}
        >
          Re-evaluate
        </Button>
      </Box>

      {/* Loading / Error States */}
      {loading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 6 }}>
          <CircularProgress size={32} />
          <Typography variant="body2" sx={{ ml: 2 }}>Evaluating dual-run shadow comparison...</Typography>
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>
      )}

      {data && !loading && (
        <Box>
          {/* Executive Metrics Cards */}
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2 }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 600 }}>
                    Active Governance Profile
                  </Typography>
                  <Typography variant="h6" sx={{ fontWeight: 700, mt: 0.5 }}>
                    {data.weight_profile_name}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Decomposition: {data.basis?.decomposition || 'EXACT_MIDPOINT'}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2 }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 600 }}>
                    Pairwise Agreement Rate
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: data.pairwise_agreement_pct >= 50 ? 'success.main' : 'warning.main', mt: 0.5 }}>
                    {isOldDegenerate ? 'N/A' : `${data.pairwise_agreement_pct.toFixed(1)}%`}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {isOldDegenerate ? 'Old ranks tied (degenerate)' : 'Kendall rank order concordance'}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2 }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 600 }}>
                    Old Basis Spread
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: isOldDegenerate ? 'warning.main' : 'info.main', mt: 0.5 }}>
                    {data.basis?.old_spread?.toFixed(2) || '0.00'} pts
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {isOldDegenerate ? 'Degenerate (< 0.5 pt tie band)' : 'Separated ranking basis'}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2 }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 600 }}>
                    Max Rank Delta
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: isOldDegenerate ? 'text.secondary' : 'success.main', mt: 0.5 }}>
                    {isOldDegenerate ? '—' : `${data.max_rank_delta} position(s)`}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {isOldDegenerate ? 'Suppressed (old tie)' : 'Within ±2 stability boundary'}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          {/* Degenerate Old Ranking Alert */}
          {isOldDegenerate && (
            <Alert
              severity="warning"
              icon={<WarningAmberIcon />}
              sx={{ mb: 2.5, borderRadius: 2 }}
            >
              <strong>Degenerate Old Ranking:</strong> The legacy score spread across all vendors is only{' '}
              <strong>{data.basis.old_spread.toFixed(2)} pts</strong> (&lt; 0.5 pt tie threshold).
              Legacy ranks reflect arbitrary tie-breaker order; rank deltas are suppressed to avoid misleading procurement.
            </Alert>
          )}

          {/* Model Change Banner */}
          {isModelChanged && (
            <Alert
              severity="info"
              icon={<InfoOutlinedIcon />}
              sx={{ mb: 2.5, borderRadius: 2, alignItems: 'center' }}
            >
              <strong>Weight vector changed in this run</strong> — <code>Δ Model</code> is ruler change, not vendor movement.
            </Alert>
          )}

          {/* Dropped Vendors Note */}
          {data.dropped_vendors && data.dropped_vendors.length > 0 && (
            <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                Excluded Candidates / Fixtures:
              </Typography>
              {data.dropped_vendors.map((d) => (
                <Chip key={d.vendor_id} size="small" variant="outlined" label={`${d.vendor_id} (${d.reason})`} />
              ))}
            </Box>
          )}

          {/* Side-by-Side Dual-Run Ranking Table */}
          <Card variant="outlined" sx={{ mb: 3, borderRadius: 2 }}>
            <CardContent sx={{ p: 2.5 }}>
              <Typography variant="h6" sx={{ fontWeight: 700, mb: 1.5 }}>
                Vendor Attribution Breakdown (Exact Symmetric Shapley Midpoint)
              </Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'action.hover' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Vendor</TableCell>
                      <TableCell align="center" sx={{ fontWeight: 700 }}>Raw Old → Raw New</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Δ Total</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Δ Input (Data)</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Δ Model (Ruler)</TableCell>
                      <TableCell align="center" sx={{ fontWeight: 700 }}>Rank Δ</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Neighbor Gap</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>State</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {data.comparisons.map((comp) => (
                      <TableRow key={comp.vendor_id} hover>
                        <TableCell>
                          <strong>{comp.vendor_name}</strong> ({comp.vendor_id})
                          {comp.safety_alert && (
                            <Box sx={{ mt: 0.5 }}>
                              <Chip size="small" label={comp.safety_alert} color="error" />
                            </Box>
                          )}
                        </TableCell>

                        {/* Raw Old -> Raw New with Scale Badge */}
                        <TableCell align="center">
                          <Stack direction="row" spacing={0.75} alignItems="center" justifyContent="center">
                            <span>{comp.old_score.toFixed(1)} pts</span>
                            {comp.scale_factor !== 1.0 && (
                              <Chip size="small" variant="outlined" label={`×${comp.scale_factor}`} sx={{ height: 18, fontSize: '0.7rem' }} />
                            )}
                            <Typography variant="body2" color="text.secondary">→</Typography>
                            <strong>{comp.new_score.toFixed(1)} pts</strong>
                          </Stack>
                        </TableCell>

                        {/* Total Delta */}
                        <TableCell align="right" sx={{ fontWeight: 700, color: comp.score_delta >= 0 ? 'success.main' : 'error.main' }}>
                          {comp.score_delta >= 0 ? '+' : ''}{comp.score_delta.toFixed(1)} pts
                        </TableCell>

                        {/* Delta Input */}
                        <TableCell align="right" sx={{ color: comp.input_effect >= 0 ? 'success.dark' : 'error.dark' }}>
                          {comp.input_effect >= 0 ? '+' : ''}{comp.input_effect.toFixed(1)} pts
                        </TableCell>

                        {/* Delta Model */}
                        <TableCell align="right" sx={{ color: comp.model_effect >= 0 ? 'info.main' : 'text.secondary' }}>
                          {comp.model_effect >= 0 ? '+' : ''}{comp.model_effect.toFixed(1)} pts
                        </TableCell>

                        {/* Rank Delta */}
                        <TableCell align="center">
                          {renderRankDeltaBadge(comp)}
                        </TableCell>

                        {/* Neighbor Gap */}
                        <TableCell align="right" sx={{ fontWeight: 600 }}>
                          {comp.neighbor_gap !== null && comp.neighbor_gap !== undefined ? `${comp.neighbor_gap.toFixed(1)} pts` : '—'}
                        </TableCell>

                        {/* 3-State Stability Chip */}
                        <TableCell>
                          {renderStateChip(comp)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>

          {/* Recent Shadow Run Audit Log */}
          {data.recent_run_history && data.recent_run_history.length > 0 && (
            <Card variant="outlined" sx={{ borderRadius: 2 }}>
              <CardContent sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
                  <HistoryIcon color="primary" />
                  <Typography variant="h6" sx={{ fontWeight: 700 }}>
                    Shadow Run Audit Trail (mdm_eval.shadow_run_log)
                  </Typography>
                </Box>
                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow sx={{ bgcolor: 'action.hover' }}>
                        <TableCell sx={{ fontWeight: 700 }}>Run ID</TableCell>
                        <TableCell sx={{ fontWeight: 700 }}>Executed At</TableCell>
                        <TableCell sx={{ fontWeight: 700 }}>Dropped Vendors</TableCell>
                        <TableCell sx={{ fontWeight: 700 }}>Replacements</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700 }}>Gross Savings</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700 }}>Net TCO Benefit</TableCell>
                        <TableCell align="center" sx={{ fontWeight: 700 }}>Payback</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700 }}>Solver Latency</TableCell>
                        <TableCell align="center" sx={{ fontWeight: 700 }}>Status</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {data.recent_run_history.map((run) => (
                        <TableRow key={run.run_id} hover>
                          <TableCell><code>#{run.run_id}</code></TableCell>
                          <TableCell>{new Date(run.executed_at).toLocaleString()}</TableCell>
                          <TableCell>
                            <Box sx={{ display: 'flex', gap: 0.5 }}>
                              {run.dropped_vendor_ids?.map((v) => (
                                <Chip key={v} size="small" color="error" variant="outlined" label={v} />
                              ))}
                            </Box>
                          </TableCell>
                          <TableCell>
                            <Box sx={{ display: 'flex', gap: 0.5 }}>
                              {run.replacement_vendor_ids?.map((v) => (
                                <Chip key={v} size="small" color="success" variant="outlined" label={v} />
                              ))}
                            </Box>
                          </TableCell>
                          <TableCell align="right" sx={{ color: 'success.main', fontWeight: 600 }}>
                            ${run.gross_annual_savings?.toLocaleString()}
                          </TableCell>
                          <TableCell align="right" sx={{ fontWeight: 700, color: 'primary.main' }}>
                            ${run.net_tco_benefit?.toLocaleString()}
                          </TableCell>
                          <TableCell align="center">
                            {run.payback_months?.toFixed(1)} mos
                          </TableCell>
                          <TableCell align="right">
                            {run.solver_latency_ms?.toFixed(1)} ms
                          </TableCell>
                          <TableCell align="center">
                            <Chip
                              size="small"
                              label={run.status}
                              color={run.status === 'COMPLETED' ? 'success' : 'warning'}
                              sx={{ fontWeight: 700 }}
                            />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          )}
        </Box>
      )}
    </Box>
  );
};
