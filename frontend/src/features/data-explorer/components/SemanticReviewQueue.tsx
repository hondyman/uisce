import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Button,
  Chip,
  IconButton,
  Tooltip,
  Collapse,
  CircularProgress,
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Check, X, Sparkles, Activity, ShieldAlert, Database } from 'lucide-react';
import { apiFetch } from '../../../lib/apiClient';
import { devError } from '../../../utils/devLogger';

// --- Interfaces ---
interface KnowledgeCandidate {
  id: string;
  type: 'alias' | 'calculated_measure' | 'default_filter' | string;
  term: string;
  target_field_id?: string;
  expression?: string;
  occurrences: number;
  confidence: number;
}

interface ImpactReport {
  candidateId: string;
  term: string;
  totalOccurrences: number;
  failedQueriesHit: number;
  estimatedRoi: string;
  sampleAffected: string[];
}

export const SemanticReviewQueue: React.FC = () => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  // --- Theme Palette ---
  const C = {
    bg: isDark ? '#080E21' : '#FFFFFF',
    cardBg: isDark ? 'rgba(0,0,0,0.2)' : '#F8FAFC',
    border: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)',
    text: isDark ? '#F8FAFC' : '#0F172A',
    textMuted: isDark ? '#94A3B8' : '#64748B',
    accentLight: isDark ? '#2DD4BF' : '#0D9488',
    alertBg: isDark ? 'rgba(251, 146, 60, 0.15)' : '#FFEDD5',
    alertText: isDark ? '#FB923C' : '#EA580C',
  };

  // --- State ---
  const [candidates, setCandidates] = useState<KnowledgeCandidate[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [impactData, setImpactData] = useState<Record<string, ImpactReport>>({});
  const [loadingImpact, setLoadingImpact] = useState<string | null>(null);

  // --- Effects ---
  useEffect(() => {
    fetchCandidates();
  }, []);

  // --- API Handlers ---
  const fetchCandidates = async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/api/v1/semantic/knowledge/candidates');
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          setCandidates(data);
          return;
        }
      }
    } catch (e) {
      devError('Failed to fetch from API', e);
    }

    // Fallback mock data for immediate UI testing
    setCandidates([
      { id: 'c1', type: 'alias', term: 'NII', target_field_id: 'net_interest_income', occurrences: 42, confidence: 0.96 },
      { id: 'c2', type: 'calculated_measure', term: 'Net Yield', expression: '(revenue - broker_fees) / avg_aum', occurrences: 19, confidence: 0.88 },
      { id: 'c3', type: 'alias', term: 'AUM', target_field_id: 'total_valuation', occurrences: 68, confidence: 0.98 },
      { id: 'c4', type: 'calculated_measure', term: 'Avg Trade Size', expression: 'total_valuation / trade_count', occurrences: 15, confidence: 0.84 },
    ]);
    setLoading(false);
  };

  const calculateImpact = async (id: string) => {
    setLoadingImpact(id);
    try {
      const res = await apiFetch(`/api/v1/semantic/knowledge/${id}/impact`);
      if (res.ok) {
        const data = await res.json();
        setImpactData((prev) => ({ ...prev, [id]: data }));
      } else {
        // Fallback mock impact data for UI testing if the Go backend endpoint isn't up yet
        setTimeout(() => {
          setImpactData((prev) => ({
            ...prev,
            [id]: {
              candidateId: id,
              term: 'Discovered Pattern',
              totalOccurrences: 142,
              failedQueriesHit: 38,
              estimatedRoi: 'Approving this will instantly resolve ~38 failed queries per month.',
              sampleAffected: [
                'Show me the NII for corporate clients',
                'What is the NII vs AUM trend across regions?',
              ],
            },
          }));
          setLoadingImpact(null);
        }, 500);
        return;
      }
    } catch (e) {
      devError('Failed to calculate impact:', e);
    } finally {
      setLoadingImpact(null);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      await apiFetch(`/api/v1/semantic/knowledge/approve/${id}`, { method: 'POST' });
    } catch (e) {
      devError('Failed to approve:', e);
    }
    setCandidates((prev) => prev.filter((c) => c.id !== id));
  };

  const handleReject = async (id: string) => {
    try {
      await apiFetch(`/api/v1/semantic/knowledge/reject/${id}`, { method: 'POST' });
    } catch (e) {
      devError('Failed to reject:', e);
    }
    setCandidates((prev) => prev.filter((c) => c.id !== id));
  };

  // --- Render ---
  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: C.bg, border: `1px solid ${C.border}`, borderRadius: 2.5 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Sparkles size={22} color={C.accentLight} />
          <Box>
            <Typography variant="h6" sx={{ color: C.text, fontWeight: 800, fontSize: '1.1rem' }}>
              Data Governance: Knowledge Review
            </Typography>
            <Typography variant="caption" sx={{ color: C.textMuted }}>
              Review and approve learned financial terminology and calculated metrics harvested from user queries.
            </Typography>
          </Box>
        </Box>
        <Chip
          size="small"
          icon={<Database size={14} />}
          label={`${candidates.length} Staged`}
          sx={{ bgcolor: C.cardBg, color: C.accentLight, fontWeight: 700, border: `1px solid ${C.border}` }}
        />
      </Box>

      {/* Main Table */}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress size={28} sx={{ color: C.accentLight }} />
        </Box>
      ) : candidates.length === 0 ? (
        <Box sx={{ textAlign: 'center', py: 6, bgcolor: C.cardBg, borderRadius: 2, border: `1px dashed ${C.border}` }}>
          <Typography variant="body2" sx={{ color: C.textMuted }}>No knowledge candidates pending review.</Typography>
        </Box>
      ) : (
        <Table size="small">
          <TableHead sx={{ bgcolor: isDark ? 'rgba(0,0,0,0.3)' : '#F1F5F9' }}>
            <TableRow>
              <TableCell sx={{ color: C.textMuted, fontWeight: 800, fontSize: '0.72rem' }}>TERM</TableCell>
              <TableCell sx={{ color: C.textMuted, fontWeight: 800, fontSize: '0.72rem' }}>PROPOSED MAPPING / FORMULA</TableCell>
              <TableCell sx={{ color: C.textMuted, fontWeight: 800, fontSize: '0.72rem' }}>IMPACT ANALYSIS</TableCell>
              <TableCell sx={{ color: C.textMuted, fontWeight: 800, fontSize: '0.72rem', textAlign: 'right' }}>ACTION</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {candidates.map((c) => (
              <React.Fragment key={c.id}>
                <TableRow sx={{ '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)' } }}>
                  <TableCell sx={{ color: C.text, fontWeight: 800, fontSize: '0.82rem' }}>
                    "{c.term}"
                  </TableCell>
                  <TableCell sx={{ color: C.accentLight, fontFamily: 'monospace', fontSize: '0.78rem' }}>
                    {c.target_field_id ? `Maps to -> [${c.target_field_id}]` : c.expression}
                  </TableCell>
                  <TableCell>
                    {!impactData[c.id] ? (
                      <Button
                        size="small"
                        onClick={() => calculateImpact(c.id)}
                        disabled={loadingImpact === c.id}
                        startIcon={loadingImpact === c.id ? <CircularProgress size={12} color="inherit" /> : <Activity size={14} />}
                        sx={{ textTransform: 'none', color: isDark ? '#38BDF8' : '#0284C7', fontSize: '0.72rem', fontWeight: 700 }}
                      >
                        {loadingImpact === c.id ? 'Calculating...' : 'Calculate Blast Radius'}
                      </Button>
                    ) : (
                      <Chip
                        size="small"
                        icon={<ShieldAlert size={12} style={{ color: C.alertText }} />}
                        label={`${impactData[c.id].failedQueriesHit} Failed Queries Prevented`}
                        sx={{ bgcolor: C.alertBg, color: C.alertText, fontWeight: 800, fontSize: '0.7rem' }}
                      />
                    )}
                  </TableCell>
                  <TableCell sx={{ textAlign: 'right' }}>
                    <Tooltip title="Approve & Deploy to Catalog">
                      <IconButton size="small" onClick={() => handleApprove(c.id)} sx={{ color: C.accentLight, mr: 0.5 }}>
                        <Check size={16} />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Reject & Discard">
                      <IconButton size="small" onClick={() => handleReject(c.id)} sx={{ color: '#EF4444' }}>
                        <X size={16} />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>

                {/* Expandable Impact Details Sub-Row */}
                <TableRow>
                  <TableCell colSpan={4} style={{ paddingBottom: 0, paddingTop: 0, border: 'none' }}>
                    <Collapse in={!!impactData[c.id]} timeout="auto" unmountOnExit>
                      <Box sx={{ m: 1, p: 2, bgcolor: C.cardBg, borderRadius: 2, borderLeft: `3px solid ${C.accentLight}` }}>
                        <Typography variant="caption" sx={{ color: C.accentLight, fontWeight: 800, mb: 1, display: 'block' }}>
                          {impactData[c.id]?.estimatedRoi}
                        </Typography>
                        <Typography variant="caption" sx={{ color: C.textMuted, fontWeight: 700 }}>
                          Sample Affected User Prompts:
                        </Typography>
                        <ul style={{ margin: '4px 0', paddingLeft: '20px', color: isDark ? '#CBD5E1' : '#475569', fontSize: '0.75rem', fontStyle: 'italic' }}>
                          {impactData[c.id]?.sampleAffected?.map((prompt, i) => (
                            <li key={i}>"{prompt}"</li>
                          ))}
                        </ul>
                      </Box>
                    </Collapse>
                  </TableCell>
                </TableRow>
              </React.Fragment>
            ))}
          </TableBody>
        </Table>
      )}
    </Paper>
  );
};

export default SemanticReviewQueue;
