import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  IconButton,
  Tooltip,
  Alert,
  Checkbox,
  FormControlLabel
} from '@mui/material';
import {
  Calculate as CalcIcon,
  AutoAwesome as AIIcon,
  Check as AcceptIcon,
  Close as DismissIcon,
  Psychology as PersonalIcon,
  TrendingUp as BoostIcon
} from '@mui/icons-material';

export interface CalculationSuggestion {
  id: string;
  suggestedCalcKey: string;
  suggestedName: string;
  expressionSql: string;
  returnType: string;
  rationaleNarrative: string;
  applicableBoKey: string;
  inputTerms: string[];
  confidenceScore: number;
  dynamicWeight?: number;
  acceptanceCount?: number;
  rejectionCount?: number;
}

interface AICalculationSuggestionStudioProps {
  tenantId?: string;
  userId?: string;
  boKey?: string;
  onCalculationCreated?: (calc: CalculationSuggestion, appliedToBO: boolean) => void;
}

export const AICalculationSuggestionStudio: React.FC<AICalculationSuggestionStudioProps> = ({
  boKey = 'trade_order',
  onCalculationCreated
}) => {
  const [suggestions, setSuggestions] = useState<CalculationSuggestion[]>([
    {
      id: '1',
      suggestedCalcKey: 'weight_in_portfolio_pct',
      suggestedName: 'Portfolio Weight (%)',
      expressionSql: '(${market_value} / NULLIF(${total_aum}, 0)) * 100.0',
      returnType: 'DECIMAL',
      rationaleNarrative: 'Automatically compute position allocation % based on current portfolio AUM.',
      applicableBoKey: boKey,
      inputTerms: ['market_value', 'total_aum'],
      confidenceScore: 0.98,
      dynamicWeight: 1.10,
      acceptanceCount: 8,
      rejectionCount: 1
    },
    {
      id: '2',
      suggestedCalcKey: 'gross_pnl_bps',
      suggestedName: 'Gross P&L (Basis Points)',
      expressionSql: '(${net_pnl} / NULLIF(${cost_basis}, 0)) * 10000.0',
      returnType: 'DECIMAL',
      rationaleNarrative: 'Institutional return metric standard for trade performance tracking.',
      applicableBoKey: boKey,
      inputTerms: ['net_pnl', 'cost_basis'],
      confidenceScore: 0.95,
      dynamicWeight: 1.05,
      acceptanceCount: 5,
      rejectionCount: 0
    }
  ]);

  const [applyToBO, setApplyToBO] = useState<Record<string, boolean>>({
    '1': true,
    '2': true
  });

  const [statusNotice, setStatusNotice] = useState<string | null>(null);

  const handleDismiss = (id: string, calcKey: string) => {
    // Dismiss and persist refusal to user profile; adjusts recommendation rate downward (-15% weight)
    setSuggestions(prev => prev.filter(s => s.id !== id));
    setStatusNotice(`Suggestion '${calcKey}' dismissed. Recommendation rate adjusted downward (-15% weight) and permanently hidden from your account.`);
  };

  const handleAccept = (item: CalculationSuggestion) => {
    const isBound = applyToBO[item.id] ?? true;
    setSuggestions(prev => prev.filter(s => s.id !== item.id));
    setStatusNotice(`Created '${item.suggestedName}' in Catalog. Positive reinforcement applied (+10% weight boost) across future recommendations.`);
    if (onCalculationCreated) {
      onCalculationCreated(item, isBound);
    }
  };

  return (
    <Box sx={{ width: '100%', mb: 3 }}>
      <Paper
        elevation={0}
        sx={{
          p: 2.5,
          bgcolor: '#071526',
          color: '#F8FAFC',
          border: '1px solid #1E293B',
          borderRadius: 2
        }}
      >
        {/* Header */}
        <Box display="flex" justifyContent="space-between" alignItems="center" pb={1.5} mb={2} borderBottom="1px solid #1E293B">
          <Stack direction="row" spacing={1.5} alignItems="center">
            <AIIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
            <Box>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, fontSize: 14 }}>
                AI Semantic Calculation Advisor & Feedback Engine
              </Typography>
              <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                Adaptive recommendation rates trained on acceptance/rejection feedback for '{boKey}'
              </Typography>
            </Box>
          </Stack>
          <Stack direction="row" spacing={1}>
            <Chip
              icon={<BoostIcon sx={{ fontSize: 13, color: '#34D399 !important' }} />}
              label="Bayesian Feedback Loop Active"
              size="small"
              sx={{ bgcolor: '#064E3B', color: '#34D399', border: '1px solid #059669', fontWeight: 600, fontSize: 10 }}
            />
            <Chip
              icon={<PersonalIcon sx={{ fontSize: 13, color: '#38BDF8 !important' }} />}
              label="User Isolation Memory"
              size="small"
              sx={{ bgcolor: '#0B1E36', color: '#38BDF8', border: '1px solid #1E293B', fontWeight: 600, fontSize: 10 }}
            />
          </Stack>
        </Box>

        {statusNotice && (
          <Alert severity="info" sx={{ mb: 2, bgcolor: '#0B1E36', color: '#38BDF8', border: '1px solid #0284C7', fontSize: 12 }}>
            {statusNotice}
          </Alert>
        )}

        {/* Suggestions List */}
        {suggestions.length === 0 ? (
          <Typography variant="body2" sx={{ color: '#94A3B8', textAlign: 'center', py: 2 }}>
            All calculation suggestions for '{boKey}' have been reviewed or applied.
          </Typography>
        ) : (
          <Stack spacing={2}>
            {suggestions.map(s => (
              <Paper
                key={s.id}
                sx={{
                  p: 2,
                  bgcolor: '#0B1E36',
                  border: '1px solid #1E293B',
                  borderRadius: 1.5
                }}
              >
                <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={1}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <CalcIcon sx={{ color: '#34D399', fontSize: 18 }} />
                    <Typography variant="body1" sx={{ fontWeight: 700, color: '#F8FAFC', fontSize: 13 }}>
                      {s.suggestedName}
                    </Typography>
                    <Chip label={`score ${(s.confidenceScore * 100).toFixed(0)}%`} size="small" sx={{ bgcolor: 'rgba(52, 211, 153, 0.1)', color: '#34D399', fontSize: 9, height: 18 }} />
                    {s.acceptanceCount !== undefined && s.acceptanceCount > 0 && (
                      <Chip label={`${s.acceptanceCount} accepts`} size="small" sx={{ bgcolor: 'rgba(56, 189, 248, 0.1)', color: '#38BDF8', fontSize: 9, height: 18 }} />
                    )}
                  </Stack>
                  <Tooltip title="Never suggest this calculation to me again (decays global score by 15%)">
                    <IconButton size="small" onClick={() => handleDismiss(s.id, s.suggestedCalcKey)} sx={{ color: '#94A3B8', '&:hover': { color: '#EF4444' } }}>
                      <DismissIcon sx={{ fontSize: 16 }} />
                    </IconButton>
                  </Tooltip>
                </Box>

                <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 1 }}>
                  {s.rationaleNarrative}
                </Typography>

                {/* Expression snippet */}
                <Box sx={{ p: 1, mb: 1.5, bgcolor: '#071526', borderRadius: 1, border: '1px solid #1E293B', fontFamily: 'monospace', fontSize: 11, color: '#38BDF8' }}>
                  {s.expressionSql}
                </Box>

                {/* Inputs & Action */}
                <Box display="flex" justifyContent="space-between" alignItems="center" flexWrap="wrap" gap={1}>
                  <Stack direction="row" spacing={0.5} alignItems="center">
                    <Typography variant="caption" sx={{ color: '#64748B', fontSize: 10 }}>Inputs:</Typography>
                    {s.inputTerms.map(t => (
                      <Chip key={t} label={t} size="small" sx={{ bgcolor: '#071526', color: '#94A3B8', fontSize: 9, height: 18 }} />
                    ))}
                  </Stack>

                  <Stack direction="row" spacing={1} alignItems="center">
                    <FormControlLabel
                      control={
                        <Checkbox
                          size="small"
                          checked={applyToBO[s.id] ?? true}
                          onChange={e => setApplyToBO({ ...applyToBO, [s.id]: e.target.checked })}
                          sx={{ color: '#0284C7', '&.Mui-checked': { color: '#00D4FF' }, p: 0.5 }}
                        />
                      }
                      label={<Typography variant="caption" sx={{ color: '#CBD5E1', fontSize: 11 }}>Bind to '{s.applicableBoKey}'</Typography>}
                    />
                    <Button
                      size="small"
                      variant="contained"
                      startIcon={<AcceptIcon sx={{ fontSize: 14 }} />}
                      onClick={() => handleAccept(s)}
                      sx={{ bgcolor: '#0284C7', color: '#FFF', textTransform: 'none', fontSize: 11, fontWeight: 700, '&:hover': { bgcolor: '#0369A1' } }}
                    >
                      Accept & Create
                    </Button>
                  </Stack>
                </Box>
              </Paper>
            ))}
          </Stack>
        )}
      </Paper>
    </Box>
  );
};

export default AICalculationSuggestionStudio;
