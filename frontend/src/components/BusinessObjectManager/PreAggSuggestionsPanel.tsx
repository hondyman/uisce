import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
  Grid,
  Skeleton,
  Alert,
  LinearProgress,
} from '@mui/material';
import {
  TrendingUp,
  Speed,
  DataUsage,
  AddCircleOutline,
  AutoAwesome,
} from '@mui/icons-material';

interface PreAggSuggestion {
  tenant_id: string;
  datasource: string;
  fingerprint: string;
  group_by: string[];
  filters: string[];
  measures: string[];
  avg_latency_ms: number;
  avg_rows: number;
  freq: number;
  score: number;
  reason: string;
}

interface PreAggSuggestionsPanelProps {
  tenantId: string;
  onPromote: (suggestion: PreAggSuggestion) => void;
}

export const PreAggSuggestionsPanel: React.FC<PreAggSuggestionsPanelProps> = ({
  tenantId,
  onPromote,
}) => {
  const [suggestions, setSuggestions] = useState<PreAggSuggestion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchSuggestions();
  }, [tenantId]);

  const fetchSuggestions = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/preaggs/suggestions`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        const data = await res.json();
        setSuggestions(data || []);
      } else {
        setError('Failed to fetch suggestions');
      }
    } catch (e) {
      devError('Failed to fetch suggestions', e);
      setError('Failed to fetch suggestions');
    } finally {
      setLoading(false);
    }
  };

  const getScoreColor = (score: number): 'success' | 'warning' | 'error' => {
    if (score >= 0.7) return 'success';
    if (score >= 0.4) return 'warning';
    return 'error';
  };

  const formatLatency = (ms: number) => {
    if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
    return `${ms.toFixed(0)}ms`;
  };

  if (loading) {
    return (
      <Box>
        <Typography variant="h6" sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <AutoAwesome color="primary" />
          AI Suggestions
        </Typography>
        <Grid container spacing={2}>
          {[1, 2, 3].map((i) => (
            <Grid item xs={12} md={4} key={i}>
              <Skeleton variant="rectangular" height={200} sx={{ borderRadius: 2 }} />
            </Grid>
          ))}
        </Grid>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="warning" sx={{ mb: 2 }}>
        {error}
      </Alert>
    );
  }

  if (suggestions.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center', bgcolor: 'background.default' }}>
        <AutoAwesome sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
        <Typography color="text.secondary">
          No pre-aggregation suggestions at this time.
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Suggestions are generated based on query patterns over the last 7 days.
        </Typography>
      </Paper>
    );
  }

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
        <AutoAwesome color="primary" />
        AI Suggestions ({suggestions.length})
      </Typography>
      <Grid container spacing={2}>
        {suggestions.map((sug, idx) => (
          <Grid item xs={12} md={4} key={sug.fingerprint || idx}>
            <Card 
              variant="outlined" 
              sx={{ 
                height: '100%', 
                display: 'flex', 
                flexDirection: 'column',
                transition: 'all 0.2s',
                '&:hover': { boxShadow: 4, borderColor: 'primary.main' },
              }}
            >
              <CardContent sx={{ flexGrow: 1 }}>
                {/* Score badge */}
                <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                  <Chip 
                    label={`${(sug.score * 100).toFixed(0)}% Impact`}
                    color={getScoreColor(sug.score)}
                    size="small"
                  />
                  <Typography variant="caption" color="text.secondary">
                    {sug.datasource}
                  </Typography>
                </Box>

                {/* Score bar */}
                <LinearProgress 
                  variant="determinate" 
                  value={sug.score * 100} 
                  color={getScoreColor(sug.score)}
                  sx={{ mb: 2, borderRadius: 1 }}
                />

                {/* Reason */}
                <Typography variant="body2" sx={{ mb: 2 }}>
                  {sug.reason}
                </Typography>

                {/* Metrics */}
                <Box display="flex" gap={2} mb={2}>
                  <Box display="flex" alignItems="center" gap={0.5}>
                    <TrendingUp fontSize="small" color="action" />
                    <Typography variant="caption">
                      {sug.freq} queries
                    </Typography>
                  </Box>
                  <Box display="flex" alignItems="center" gap={0.5}>
                    <Speed fontSize="small" color="action" />
                    <Typography variant="caption">
                      {formatLatency(sug.avg_latency_ms)}
                    </Typography>
                  </Box>
                  <Box display="flex" alignItems="center" gap={0.5}>
                    <DataUsage fontSize="small" color="action" />
                    <Typography variant="caption">
                      {sug.avg_rows.toLocaleString()} rows
                    </Typography>
                  </Box>
                </Box>

                {/* Group by chips */}
                <Typography variant="caption" color="text.secondary" display="block" mb={0.5}>
                  Group By:
                </Typography>
                <Box display="flex" flexWrap="wrap" gap={0.5} mb={1}>
                  {sug.group_by?.slice(0, 3).map((col) => (
                    <Chip key={col} label={col} size="small" variant="outlined" />
                  ))}
                  {(sug.group_by?.length || 0) > 3 && (
                    <Chip label={`+${sug.group_by.length - 3}`} size="small" />
                  )}
                </Box>

                {/* Measures chips */}
                <Typography variant="caption" color="text.secondary" display="block" mb={0.5}>
                  Measures:
                </Typography>
                <Box display="flex" flexWrap="wrap" gap={0.5}>
                  {sug.measures?.slice(0, 2).map((m) => (
                    <Chip key={m} label={m} size="small" variant="outlined" color="primary" />
                  ))}
                  {(sug.measures?.length || 0) > 2 && (
                    <Chip label={`+${sug.measures.length - 2}`} size="small" color="primary" />
                  )}
                </Box>
              </CardContent>
              <CardActions>
                <Button 
                  startIcon={<AddCircleOutline />}
                  onClick={() => onPromote(sug)}
                  size="small"
                  fullWidth
                >
                  Create Pre-Aggregation
                </Button>
              </CardActions>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
};

export default PreAggSuggestionsPanel;
