import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Button,
  IconButton,
  Grid
} from '@mui/material';
import {
  AutoAwesome as AutoAwesomeIcon,
  TrendingUp as TrendingUpIcon,
  LightbulbOutlined as LightbulbIcon,
  ArrowForward as ArrowForwardIcon,
  Close as CloseIcon,
} from '@mui/icons-material';

interface SuggestionItem {
  id: string;
  type: string;
  title: string;
  subtitle: string;
  confidenceScore: number;
  actionPayload: string;
}

interface AIRecommendationDockProps {
  tenantId?: string;
  focalNodeId?: string;
  currentPrompt?: string;
  onApplySuggestion?: (payload: string) => void;
}

export const AIRecommendationDock: React.FC<AIRecommendationDockProps> = ({
  tenantId: _tenantId,
  focalNodeId: _focalNodeId,
  currentPrompt: _currentPrompt = '',
  onApplySuggestion
}) => {
  const [suggestions, _setSuggestions] = useState<SuggestionItem[]>([
    {
      id: 'sug-1',
      type: 'QUERY_SUGGESTION',
      title: 'Compare 3-Year Rolling XIRR by Country',
      subtitle: '84% of analysts analyze regional fund returns next',
      confidenceScore: 0.94,
      actionPayload: 'EXECUTE_ROLLING_XIRR_COUNTRY'
    },
    {
      id: 'sug-2',
      type: 'RELATED_FIELD',
      title: 'Include [order_freight_total]',
      subtitle: 'Frequently bound alongside customer order totals',
      confidenceScore: 0.88,
      actionPayload: 'ADD_FIELD:order_freight_total'
    },
    {
      id: 'sug-3',
      type: 'PROACTIVE_INSIGHT',
      title: 'Top 5 Issuers Represent 72% of Exposure',
      subtitle: 'Concentration anomaly detected across high-yield bonds',
      confidenceScore: 0.98,
      actionPayload: 'OPEN_INSIGHT_DRAWER'
    }
  ]);

  const [dismissedIds, setDismissedIds] = useState<string[]>([]);

  const handleDismiss = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setDismissedIds((prev) => [...prev, id]);
  };

  const visibleSuggestions = suggestions.filter((s) => !dismissedIds.includes(s.id));

  if (visibleSuggestions.length === 0) {
    return null;
  }

  return (
    <Paper
      elevation={0}
      sx={{
        p: 2,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={1.5}>
        <Stack direction="row" spacing={1} alignItems="center">
          <AutoAwesomeIcon sx={{ color: '#00D4FF', fontSize: 18 }} />
          <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: 12, color: '#F8FAFC' }}>
            Contextual AI Suggestions & Insights
          </Typography>
        </Stack>
        <Typography variant="caption" sx={{ color: '#64748B', fontSize: 11 }}>
          Adaptive Telemetry Active
        </Typography>
      </Box>

      <Grid container spacing={1.5}>
        {visibleSuggestions.map((item) => (
          <Grid   key={item.id} size={{ xs: 12, md: 4 }}>
            <Paper
              sx={{
                p: 1.5,
                bgcolor: '#0B1E36',
                border: '1px solid #1E293B',
                borderRadius: 1.5,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
                height: '100%',
                transition: 'all 0.2s',
                '&:hover': {
                  borderColor: '#0284C7',
                  bgcolor: '#0E2442'
                }
              }}
            >
              <Box>
                <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={0.5}>
                  <Chip
                    icon={
                      item.type === 'PROACTIVE_INSIGHT' ? (
                        <TrendingUpIcon sx={{ fontSize: 12, color: '#F59E0B !important' }} />
                      ) : (
                        <LightbulbIcon sx={{ fontSize: 12, color: '#00D4FF !important' }} />
                      )
                    }
                    label={`${Math.round(item.confidenceScore * 100)}% Match`}
                    size="small"
                    sx={{
                      bgcolor: item.type === 'PROACTIVE_INSIGHT' ? '#451A03' : '#082F49',
                      color: item.type === 'PROACTIVE_INSIGHT' ? '#FCD34D' : '#38BDF8',
                      fontSize: 10,
                      fontWeight: 700,
                      height: 20
                    }}
                  />
                  <IconButton
                    size="small"
                    onClick={(e) => handleDismiss(item.id, e)}
                    sx={{ color: '#64748B', p: 0.2, '&:hover': { color: '#EF4444' } }}
                  >
                    <CloseIcon sx={{ fontSize: 14 }} />
                  </IconButton>
                </Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, fontSize: 12, color: '#F8FAFC', mt: 0.5 }}>
                  {item.title}
                </Typography>
                <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 11, display: 'block', mt: 0.2 }}>
                  {item.subtitle}
                </Typography>
              </Box>

              <Button
                variant="outlined"
                size="small"
                endIcon={<ArrowForwardIcon sx={{ fontSize: 12 }} />}
                onClick={() => onApplySuggestion && onApplySuggestion(item.actionPayload)}
                sx={{
                  mt: 1.5,
                  alignSelf: 'flex-start',
                  borderColor: '#334155',
                  color: '#38BDF8',
                  fontSize: 11,
                  textTransform: 'none',
                  p: '2px 8px',
                  '&:hover': { borderColor: '#38BDF8', bgcolor: 'rgba(56, 189, 248, 0.08)' }
                }}
              >
                Apply Action
              </Button>
            </Paper>
          </Grid>
        ))}
      </Grid>
    </Paper>
  );
};

export default AIRecommendationDock;
