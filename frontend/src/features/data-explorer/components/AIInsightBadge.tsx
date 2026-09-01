import React from 'react';
import { Box, Typography, Paper, Chip, Tooltip } from '@mui/material';
import { Sparkles, TrendingUp, AlertCircle, Zap } from 'lucide-react';

interface AIInsightBadgeProps {
  summaryText: string;
  anomalies?: string[];
  topDriver?: string;
  isDark?: boolean;
  isCacheHit?: boolean;
}

export const AIInsightBadge: React.FC<AIInsightBadgeProps> = ({
  summaryText,
  anomalies = [],
  topDriver,
  isDark = false,
  isCacheHit = false,
}) => {
  return (
    <Paper
      elevation={0}
      sx={{
        p: 1.5,
        mb: 2,
        borderRadius: 2,
        bgcolor: isDark ? 'rgba(13, 148, 136, 0.12)' : '#F0FDFA',
        border: '1px solid rgba(45, 212, 191, 0.3)',
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Sparkles size={16} color="#0D9488" />
          <Typography
            variant="caption"
            sx={{ fontWeight: 800, color: '#0D9488', letterSpacing: 0.6, textTransform: 'uppercase' }}
          >
            Automated Executive Insights
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {isCacheHit && (
            <Tooltip title="Intent recognized instantly via Vector Semantic Cache (< 50ms). LLM bypassed.">
              <Chip
                size="small"
                icon={<Zap size={12} color="#F59E0B" fill="#F59E0B" />}
                label="Cache Hit"
                sx={{
                  height: 20,
                  fontSize: '0.65rem',
                  fontWeight: 800,
                  bgcolor: 'rgba(245, 158, 11, 0.15)',
                  color: '#D97706',
                  border: '1px solid rgba(245, 158, 11, 0.3)',
                }}
              />
            </Tooltip>
          )}
          {topDriver && (
            <Chip
              size="small"
              icon={<TrendingUp size={12} color="#0D9488" />}
              label={`Primary Driver: ${topDriver}`}
              sx={{
                height: 20,
                fontSize: '0.65rem',
                fontWeight: 700,
                bgcolor: 'rgba(45, 212, 191, 0.25)',
                color: isDark ? '#2DD4BF' : '#0F766E',
              }}
            />
          )}
        </Box>
      </Box>

      <Typography variant="body2" sx={{ fontSize: '0.8rem', color: isDark ? '#E2E8F0' : '#1E293B', lineHeight: 1.45 }}>
        {summaryText}
      </Typography>

      {anomalies.length > 0 && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
          <AlertCircle size={14} color="#EA580C" />
          <Typography variant="caption" sx={{ color: '#EA580C', fontWeight: 700 }}>
            Detected Anomaly: {anomalies.join(' · ')}
          </Typography>
        </Box>
      )}
    </Paper>
  );
};

export default AIInsightBadge;
