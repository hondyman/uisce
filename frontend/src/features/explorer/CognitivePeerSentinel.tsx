import React from 'react';
import {
  Box,
  Paper,
  Stack,
  Typography,
  Chip,
  Button,
  Alert,
  Tooltip,
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import GroupsIcon from '@mui/icons-material/Groups';
import HealingIcon from '@mui/icons-material/Healing';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';

export interface PeerRecommendation {
  suggestedFieldKey: string;
  peerAdoptionRate: number;
  rationaleNarrative: string;
}

export interface SelfHealingAlert {
  triggerCondition: string;
  remediationAction: string;
  suggestedFilter?: string;
}

interface CognitivePeerSentinelProps {
  boKey?: string;
  peerRecommendations?: PeerRecommendation[];
  healingAlert?: SelfHealingAlert | null;
  onAddField?: (fieldKey: string) => void;
  onApplyHealedFilter?: (filterExpr: string) => void;
}

export const CognitivePeerSentinel: React.FC<CognitivePeerSentinelProps> = ({
  boKey = 'trade_order',
  peerRecommendations = [
    {
      suggestedFieldKey: 'cash_drag',
      peerAdoptionRate: 87,
      rationaleNarrative: '87% of peer Sovereign Wealth institutions include [cash_drag] with Trade Orders.'
    },
    {
      suggestedFieldKey: 'fx_hedge_ratio',
      peerAdoptionRate: 74,
      rationaleNarrative: '74% of peer Sovereign Wealth institutions explore [fx_hedge_ratio] with Multi-Currency Trades.'
    }
  ],
  healingAlert,
  onAddField,
  onApplyHealedFilter,
}) => {
  return (
    <Stack spacing={2} sx={{ mb: 3 }}>
      {/* 1. Proactive Self-Healing Sentinel for Frustration / Zero Rows */}
      {healingAlert && (
        <Alert
          severity="warning"
          icon={<HealingIcon sx={{ color: '#F59E0B' }} />}
          sx={{
            bgcolor: 'rgba(245, 158, 11, 0.08)',
            border: '1px solid rgba(245, 158, 11, 0.25)',
            color: '#FDE68A',
            borderRadius: 2,
          }}
          action={
            healingAlert.suggestedFilter && onApplyHealedFilter && (
              <Button
                color="warning"
                size="small"
                variant="outlined"
                onClick={() => onApplyHealedFilter(healingAlert.suggestedFilter!)}
                sx={{ textTransform: 'none', fontWeight: 600 }}
              >
                Apply Fix
              </Button>
            )
          }
        >
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            Query Correction Detected
          </Typography>
          <Typography variant="caption" sx={{ color: '#FCD34D' }}>
            {healingAlert.remediationAction}
          </Typography>
        </Alert>
      )}

      {/* 2. Privacy-Preserving Peer Cohort Insights */}
      {peerRecommendations.length > 0 && (
        <Paper
          sx={{
            p: 2,
            bgcolor: '#091528',
            border: '1px solid #1E293B',
            borderRadius: 2.5,
          }}
        >
          <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1.5 }}>
            <GroupsIcon sx={{ color: '#00D4FF', fontSize: 18 }} />
            <Typography variant="subtitle2" sx={{ color: '#F8FAFC', fontWeight: 700 }}>
              Peer Intelligence (Anonymized Cohort Benchmarks)
            </Typography>
            <Chip
              label="Differential Privacy (ε=0.1)"
              size="small"
              sx={{
                bgcolor: 'rgba(0, 212, 255, 0.1)',
                color: '#38BDF8',
                fontSize: '10px',
                height: 20,
              }}
            />
          </Stack>

          <Stack direction="row" spacing={1.5} flexWrap="wrap">
            {peerRecommendations.map((rec) => (
              <Tooltip key={rec.suggestedFieldKey} title={rec.rationaleNarrative}>
                <Paper
                  sx={{
                    p: '8px 12px',
                    bgcolor: '#0E1E38',
                    border: '1px solid #2A4365',
                    borderRadius: 2,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                  }}
                >
                  <AutoAwesomeIcon sx={{ color: '#38BDF8', fontSize: 16 }} />
                  <Box>
                    <Typography variant="body2" sx={{ color: '#F1F5F9', fontWeight: 600 }}>
                      +{rec.suggestedFieldKey}
                    </Typography>
                    <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                      {rec.peerAdoptionRate.toFixed(0)}% peer adoption
                    </Typography>
                  </Box>
                  {onAddField && (
                    <Button
                      size="small"
                      startIcon={<AddCircleOutlineIcon />}
                      onClick={() => onAddField(rec.suggestedFieldKey)}
                      sx={{
                        ml: 1,
                        textTransform: 'none',
                        color: '#00D4FF',
                        fontSize: '11px',
                      }}
                    >
                      Add
                    </Button>
                  )}
                </Paper>
              </Tooltip>
            ))}
          </Stack>
        </Paper>
      )}
    </Stack>
  );
};

export default CognitivePeerSentinel;
