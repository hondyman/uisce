import React, { useState, useEffect } from 'react';
import { Box, Typography, TextField, Button, Paper, Chip, Select, MenuItem, FormControl, InputLabel } from '@mui/material';
import {
  AlertTriangle,
  CheckCircle2,
  ShieldAlert,
  Bot,
  ArrowRight,
  TrendingUp,
  Clock,
  Sparkles,
  RefreshCw,
  Sliders,
  Send,
  Building2,
  Layers,
  Database,
  ExternalLink,
} from 'lucide-react';

export interface CompetingVendorValue {
  vendor: string;
  value: any;
  confidence: number;
  timestamp?: string;
  isStale?: boolean;
}

export interface MDMExceptionItem {
  exceptionId: string;
  tenantId: string;
  domainKey: string; // SECURITY, PRICING, CORP_ACTION, ISSUER, FUND, ACCOUNT
  masterEntitySid: string;
  entityName?: string;
  fieldName: string;
  anomalyType: string; // PRICE_TOLERANCE_BREACH, CHECKSUM_FAILURE, UNRESOLVED_XREF, STALE_FEED
  status: 'OPEN' | 'IN_REVIEW' | 'RESOLVED' | 'OVERRIDDEN';
  competingValues: CompetingVendorValue[];
  maxDeviationPct: number;
  createdAt: string;
  aiDiagnosis?: {
    recommendation: string;
    winningVendor: string;
    suggestedValue: any;
    confidenceScore: number;
    rationale: string;
  };
}

export const MDMExceptionWorkbench: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [exceptions, setExceptions] = useState<MDMExceptionItem[]>([]);
  const [selectedException, setSelectedException] = useState<MDMExceptionItem | null>(null);
  const [selectedDomain, setSelectedDomain] = useState<string>('ALL');
  const [selectedAnomaly, setSelectedAnomaly] = useState<string>('ALL');
  const [overrideReason, setOverrideReason] = useState<string>('');
  const [customValue, setCustomValue] = useState<string>('');
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [notification, setNotification] = useState<string | null>(null);

  useEffect(() => {
    fetchExceptions();
  }, [tenantId, selectedDomain, selectedAnomaly]);

  const fetchExceptions = async () => {
    try {
      const queryParams = new URLSearchParams({
        domain: selectedDomain,
        anomaly: selectedAnomaly,
      });
      const res = await fetch(`/api/v1/mdm/exceptions?${queryParams.toString()}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        const data = await res.json();
        setExceptions(data);
        if (data.length > 0 && !selectedException) {
          setSelectedException(data[0]);
        }
      }
    } catch (err) {
      console.error('Failed fetching MDM exceptions:', err);
    }
  };

  const handleApplyOverride = async (vendor: string, overrideVal: any, reason: string) => {
    if (!selectedException) return;
    setIsSubmitting(true);

    try {
      const payload = {
        exceptionId: selectedException.exceptionId,
        masterEntitySid: selectedException.masterEntitySid,
        domainKey: selectedException.domainKey,
        fieldName: selectedException.fieldName,
        chosenVendor: vendor,
        overrideValue: overrideVal,
        overrideReason: reason || 'Manual Data Steward Override',
        signalTemporalWorkflow: true,
      };

      const res = await fetch(`/api/v1/mdm/exceptions/${selectedException.exceptionId}/override`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        setNotification(`Override applied for ${selectedException.masterEntitySid}. Temporal workflow signaled.`);
        setTimeout(() => setNotification(null), 4000);
        await fetchExceptions();
      }
    } finally {
      setIsSubmitting(false);
      setOverrideReason('');
      setCustomValue('');
    }
  };

  const filteredExceptions = exceptions.filter((item) => {
    const matchDomain = selectedDomain === 'ALL' || item.domainKey === selectedDomain;
    const matchAnomaly = selectedAnomaly === 'ALL' || item.anomalyType === selectedAnomaly;
    return matchDomain && matchAnomaly;
  });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', bgcolor: '#030914', color: '#f1f5f9', border: '1px solid #1e293b', borderRadius: 2, overflow: 'hidden', fontFamily: 'sans-serif' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 3, py: 2, bgcolor: '#071526', borderBottom: '1px solid #1e293b' }}>
        <Box>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#f1f5f9', letterSpacing: '-0.025em', display: 'flex', alignItems: 'center', gap: 1 }}>
            <ShieldAlert size={20} style={{ color: '#fbbf24' }} />
            MDM Data Stewardship & Exception Workbench
          </Typography>
          <Typography variant="caption" sx={{ color: '#94a3b8', mt: 0.5, display: 'block' }}>
            Resolve price tolerance breaches, vendor feed collisions, and checksum anomalies across 8 master domains.
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, bgcolor: '#0f172a', border: '1px solid #1e293b', borderRadius: 1, px: 1.5, py: 0.75, fontSize: '0.75rem' }}>
            <Layers size={14} style={{ color: '#22d3ee' }} />
            <FormControl size="small" sx={{ minWidth: 140, '& .MuiSelect-select': { py: 0.5, color: '#e2e8f0' } }}>
              <Select
                value={selectedDomain}
                onChange={(e) => setSelectedDomain(e.target.value)}
                sx={{ bgcolor: 'transparent', color: '#e2e8f0', fontSize: '0.75rem', '& .MuiOutlinedInput-notchedOutline': { border: 'none' } }}
              >
                <option value="ALL">All Master Domains</option>
                <option value="PRICING">Pricing & Curves</option>
                <option value="SECURITY">Security Master</option>
                <option value="CORP_ACTION">Corporate Actions</option>
                <option value="ISSUER">Legal Entity & Issuer</option>
                <option value="FUND">Fund & Vehicle Master</option>
              </Select>
            </FormControl>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, bgcolor: '#0f172a', border: '1px solid #1e293b', borderRadius: 1, px: 1.5, py: 0.75, fontSize: '0.75rem' }}>
            <AlertTriangle size={14} style={{ color: '#fbbf24' }} />
            <FormControl size="small" sx={{ minWidth: 140, '& .MuiSelect-select': { py: 0.5, color: '#e2e8f0' } }}>
              <Select
                value={selectedAnomaly}
                onChange={(e) => setSelectedAnomaly(e.target.value)}
                sx={{ bgcolor: 'transparent', color: '#e2e8f0', fontSize: '0.75rem', '& .MuiOutlinedInput-notchedOutline': { border: 'none' } }}
              >
                <option value="ALL">All Anomaly Types</option>
                <option value="PRICE_TOLERANCE_BREACH">Price Breach (&gt;10%)</option>
                <option value="CHECKSUM_FAILURE">Checksum Failure</option>
                <option value="UNRESOLVED_XREF">Unresolved XREF</option>
                <option value="STALE_FEED">Stale Feed</option>
              </Select>
            </FormControl>
          </Box>

          <Button
            onClick={fetchExceptions}
            variant="outlined"
            size="small"
            sx={{ p: 1, bgcolor: '#0f172a', borderColor: '#1e293b', color: '#cbd5e1', '&:hover': { bgcolor: '#1e293b' } }}
            title="Refresh Exceptions"
          >
            <RefreshCw size={14} />
          </Button>
        </Box>
      </Box>

      {notification && (
        <Box sx={{ bgcolor: 'rgba(16, 185, 129, 0.2)', borderBottom: '1px solid rgba(16, 185, 129, 0.4)', px: 3, py: 1, display: 'flex', alignItems: 'center', gap: 1, color: '#6ee7b7', fontSize: '0.75rem' }}>
          <CheckCircle2 size={16} style={{ color: '#34d399' }} />
          {notification}
        </Box>
      )}

      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(12, 1fr)', flex: 1, overflow: 'hidden' }}>
        <Box sx={{ gridColumn: 'span 4', borderRight: '1px solid #1e293b', overflowY: 'auto', bgcolor: 'rgba(5, 14, 29, 0.6)' }}>
          <Box sx={{ p: 1.5, bgcolor: 'rgba(30, 41, 59, 0.4)', borderBottom: '1px solid rgba(30, 41, 59, 0.8)', display: 'flex', justifyContent: 'space-between', fontSize: '0.6875rem', fontWeight: 600, color: '#94a3b8', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            <span>Open Break Queue</span>
            <span style={{ color: '#fbbf24', fontFamily: 'monospace' }}>{filteredExceptions.length} Items</span>
          </Box>

          <Box sx={{ borderBottom: '1px solid rgba(30, 41, 59, 0.6)' }}>
            {filteredExceptions.map((item) => {
              const isSelected = selectedException?.exceptionId === item.exceptionId;
              return (
                <Box
                  key={item.exceptionId}
                  onClick={() => setSelectedException(item)}
                  sx={{
                    p: 2,
                    cursor: 'pointer',
                    transition: 'all 0.2s',
                    bgcolor: isSelected ? 'rgba(34, 211, 238, 0.1)' : 'transparent',
                    borderLeft: isSelected ? '4px solid #22d3ee' : '4px solid transparent',
                    '&:hover': { bgcolor: 'rgba(30, 41, 59, 0.4)' },
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
                    <Box>
                      <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#e2e8f0', display: 'block' }}>
                        {item.masterEntitySid}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#94a3b8', mt: 0.5, display: 'block', fontSize: '0.6875rem' }}>
                        {item.entityName || item.fieldName}
                      </Typography>
                    </Box>
                    <Chip
                      label={`Δ ${(item.maxDeviationPct || 0).toFixed(1)}%`}
                      size="small"
                      sx={{
                        bgcolor: (item.maxDeviationPct || 0) > 10 ? 'rgba(239, 68, 68, 0.2)' : 'rgba(245, 158, 11, 0.2)',
                        color: (item.maxDeviationPct || 0) > 10 ? '#f87171' : '#fbbf24',
                        fontWeight: 700,
                        fontSize: '0.625rem',
                        height: 20,
                        borderRadius: 'full',
                        border: `1px solid ${(item.maxDeviationPct || 0) > 10 ? 'rgba(239, 68, 68, 0.3)' : 'rgba(245, 158, 11, 0.3)'}`,
                      }}
                    />
                  </Box>

                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 1.5, fontSize: '0.625rem', color: '#94a3b8' }}>
                    <Chip label={item.domainKey} size="small" sx={{ bgcolor: '#1e293b', color: '#cbd5e1', fontWeight: 600, fontSize: '0.625rem', height: 18 }} />
                    <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>{item.fieldName}</Typography>
                    <Box sx={{ ml: 'auto', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Clock size={12} />
                      {new Date(item.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </Box>
                  </Box>
                </Box>
              );
            })}

            {filteredExceptions.length === 0 && (
              <Box sx={{ p: 6, textAlign: 'center', color: '#94a3b8', fontSize: '0.75rem' }}>
                No active breaks found in the selected domain.
              </Box>
            )}
          </Box>
        </Box>

        {selectedException ? (
          <Box sx={{ gridColumn: 'span 8', overflowY: 'auto', p: 3, bgcolor: '#030914', display: 'flex', flexDirection: 'column', gap: 3 }}>
            <Paper sx={{ p: 2, bgcolor: 'rgba(30, 41, 59, 0.6)', border: '1px solid #1e293b', borderRadius: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Chip label={selectedException.domainKey} size="small" sx={{ bgcolor: 'rgba(34, 211, 238, 0.2)', color: '#22d3ee', fontWeight: 700, fontSize: '0.75rem', border: '1px solid rgba(34, 211, 238, 0.3)' }} />
                  <Typography variant="h6" sx={{ fontWeight: 700, color: '#f1f5f9', fontFamily: 'monospace' }}>
                    {selectedException.masterEntitySid}
                  </Typography>
                </Box>
                <Typography variant="caption" sx={{ color: '#94a3b8', mt: 0.5, display: 'block' }}>
                  Evaluating field: <span style={{ color: '#e2e8f0', fontWeight: 600 }}>{selectedException.fieldName}</span>
                  {' '}| Anomaly: <span style={{ color: '#fbbf24', fontWeight: 600 }}>{selectedException.anomalyType}</span>
                </Typography>
              </Box>

              <Box sx={{ textAlign: 'right' }}>
                <Typography variant="caption" sx={{ color: '#94a3b8', display: 'block', textTransform: 'uppercase', fontWeight: 600, fontSize: '0.625rem' }}>Max Deviation</Typography>
                <Typography variant="h5" sx={{ fontWeight: 700, fontFamily: 'monospace', color: '#f87171' }}>
                  +{(selectedException.maxDeviationPct || 0).toFixed(2)}%
                </Typography>
              </Box>
            </Paper>

            {selectedException.aiDiagnosis && (
              <Paper sx={{ p: 2, background: 'linear-gradient(to right, rgba(88, 28, 135, 0.3), rgba(30, 41, 59, 0.6), rgba(88, 28, 135, 0.2))', border: '1px solid rgba(168, 85, 247, 0.4)', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: '#c4b5fd', textTransform: 'uppercase', letterSpacing: '0.05em', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Bot size={16} style={{ color: '#c4b5fd' }} />
                    MCP AI Data Steward Copilot Recommendation
                  </Typography>
                  <Chip label={`Confidence: ${(selectedException.aiDiagnosis.confidenceScore * 100).toFixed(0)}%`} size="small" sx={{ bgcolor: 'rgba(168, 85, 247, 0.2)', color: '#c4b5fd', border: '1px solid rgba(168, 85, 247, 0.4)', fontWeight: 700, fontSize: '0.625rem', borderRadius: 'full' }} />
                </Box>

                <Typography variant="body2" sx={{ color: '#e2e8f0', lineHeight: 1.6, fontFamily: 'sans-serif' }}>
                  {selectedException.aiDiagnosis.rationale}
                </Typography>

                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pt: 1, borderTop: '1px solid rgba(168, 85, 247, 0.2)' }}>
                  <Typography variant="caption" sx={{ color: '#e9d5ff' }}>
                    Suggested Action: <strong style={{ color: '#fff', fontFamily: 'monospace' }}>{selectedException.aiDiagnosis.recommendation}</strong>
                  </Typography>
                  <Button
                    variant="contained"
                    onClick={() =>
                      handleApplyOverride(
                        selectedException.aiDiagnosis!.winningVendor,
                        selectedException.aiDiagnosis!.suggestedValue,
                        `AI Recommendation: ${selectedException.aiDiagnosis!.rationale}`
                      )
                    }
                    disabled={isSubmitting}
                    startIcon={<Sparkles size={14} />}
                    sx={{
                      px: 2,
                      py: 0.75,
                      bgcolor: '#9333ea',
                      '&:hover': { bgcolor: '#a855f7' },
                      color: '#fff',
                      fontWeight: 700,
                      fontSize: '0.75rem',
                      borderRadius: 1,
                      textTransform: 'none',
                      disabled: { opacity: 0.5 },
                    }}
                  >
                    1-Click Apply AI Fix
                  </Button>
                </Box>
              </Paper>
            )}

            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#cbd5e1', textTransform: 'uppercase', letterSpacing: '0.05em', display: 'flex', alignItems: 'center', gap: 1 }}>
                <Database size={16} style={{ color: '#22d3ee' }} />
                Competing Multi-Source Vendor Payloads
              </Typography>

              <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2 }}>
                {(selectedException.competingValues || []).map((feed) => {
                  const isWinningCandidate = selectedException.aiDiagnosis?.winningVendor === feed.vendor;
                  return (
                    <Paper
                      key={feed.vendor}
                      sx={{
                        p: 2,
                        borderRadius: 2,
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'space-between',
                        bgcolor: isWinningCandidate ? 'rgba(30, 41, 59, 0.8)' : 'rgba(30, 41, 59, 0.4)',
                        border: '1px solid',
                        borderColor: isWinningCandidate ? 'rgba(34, 211, 238, 0.6)' : '#1e293b',
                        boxShadow: isWinningCandidate ? '0 10px 15px -3px rgba(34, 211, 238, 0.1)' : 'none',
                      }}
                    >
                      <Box>
                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 1, borderBottom: '1px solid #1e293b' }}>
                          <Typography variant="caption" sx={{ fontWeight: 700, color: '#f1f5f9', textTransform: 'uppercase' }}>
                            {feed.vendor}
                          </Typography>
                          <Typography variant="caption" sx={{ color: '#22d3ee', fontFamily: 'monospace', fontSize: '0.625rem' }}>
                            {((feed.confidence || 0.9) * 100).toFixed(0)}% Trust
                          </Typography>
                        </Box>

                        <Box sx={{ my: 2, textAlign: 'center' }}>
                          <Typography variant="caption" sx={{ color: '#94a3b8', display: 'block', textTransform: 'uppercase', fontWeight: 500 }}>Reported Value</Typography>
                          <Typography variant="h5" sx={{ fontWeight: 700, fontFamily: 'monospace', color: '#f1f5f9' }}>
                            {typeof feed.value === 'number' ? `$${feed.value.toFixed(2)}` : String(feed.value)}
                          </Typography>
                        </Box>
                      </Box>

                      <Button
                        variant="contained"
                        onClick={() =>
                          handleApplyOverride(
                            feed.vendor,
                            feed.value,
                            `Manual Steward selection: Accepted ${feed.vendor} feed`
                          )
                        }
                        disabled={isSubmitting}
                        startIcon={<CheckCircle2 size={14} />}
                        sx={{
                          width: '100%',
                          py: 1,
                          bgcolor: isWinningCandidate ? '#F5A623' : '#1e293b',
                          '&:hover': { bgcolor: isWinningCandidate ? '#fbbf24' : '#334155' },
                          color: isWinningCandidate ? '#0f172a' : '#e2e8f0',
                          fontWeight: 700,
                          fontSize: '0.75rem',
                          borderRadius: 1,
                          textTransform: 'none',
                        }}
                      >
                        Accept {feed.vendor}
                      </Button>
                    </Paper>
                  );
                })}
              </Box>
            </Box>

            <Paper sx={{ p: 2, bgcolor: 'rgba(30, 41, 59, 0.5)', border: '1px solid #1e293b', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#cbd5e1', textTransform: 'uppercase', letterSpacing: '0.05em', display: 'flex', alignItems: 'center', gap: 1 }}>
                <Sliders size={16} style={{ color: '#fbbf24' }} />
                Manual Data Steward Override & Audit Reason
              </Typography>

              <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 600, color: '#cbd5e1', display: 'block', mb: 0.5 }}>Custom Field Value</Typography>
                  <TextField
                    fullWidth
                    size="small"
                    value={customValue}
                    onChange={(e) => setCustomValue(e.target.value)}
                    placeholder="Enter explicit golden value..."
                    sx={{
                      '& .MuiOutlinedInput-root': { bgcolor: '#020617', '& fieldset': { borderColor: '#1e293b' } },
                      '& input': { color: '#f1f5f9', fontSize: '0.75rem', fontFamily: 'monospace' },
                    }}
                  />
                </Box>
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 600, color: '#cbd5e1', display: 'block', mb: 0.5 }}>Steward Override Rationale</Typography>
                  <TextField
                    fullWidth
                    size="small"
                    value={overrideReason}
                    onChange={(e) => setOverrideReason(e.target.value)}
                    placeholder="e.g. Verified with trading desk, IDC feed shifted decimal"
                    sx={{
                      '& .MuiOutlinedInput-root': { bgcolor: '#020617', '& fieldset': { borderColor: '#1e293b' } },
                      '& input': { color: '#f1f5f9', fontSize: '0.75rem' },
                    }}
                  />
                </Box>
              </Box>

              <Box sx={{ display: 'flex', justifyContent: 'flex-end', pt: 1 }}>
                <Button
                  variant="contained"
                  onClick={() => handleApplyOverride('MANUAL_STEWARD', customValue, overrideReason)}
                  disabled={isSubmitting || !customValue || !overrideReason}
                  startIcon={<Send size={14} />}
                  sx={{
                    px: 2,
                    py: 1,
                    background: 'linear-gradient(to right, #f59e0b, #F5A623)',
                    '&:hover': { opacity: 0.95 },
                    color: '#0f172a',
                    fontWeight: 700,
                    fontSize: '0.75rem',
                    borderRadius: 1,
                    textTransform: 'none',
                    disabled: { opacity: 0.4 },
                  }}
                >
                  Commit Manual Override & Signal Workflow
                </Button>
              </Box>
            </Paper>
          </Box>
        ) : (
          <Box sx={{ gridColumn: 'span 8', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#94a3b8', fontSize: '0.75rem', bgcolor: '#030914' }}>
            Select an open exception from the left queue to review vendor feeds and apply break overrides.
          </Box>
        )}
      </Box>
    </Box>
  );
};
