import React, { useState, useEffect, useCallback } from 'react';
import { Bundle } from '../types';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  LinearProgress,
  Chip,
  Alert,
  CircularProgress,
  Tooltip
} from '@mui/material';
import {
  TrendingUp,
  AccountBalance,
  ShowChart
} from '@mui/icons-material';
// Assessment icon removed; not used after MetricCard icon set to null

interface FundMetrics {
  fundId: string;
  fundName: string;
  dpi: number;
  tvpi: number;
  rvpi: number;
  moic: number;
  status: 'active' | 'liquidated' | 'realizing';
  vintage: number;
}

interface MultipleOverlayPanelProps {
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: Bundle | null;
}

export const MultipleOverlayPanel: React.FC<MultipleOverlayPanelProps> = ({ selectedFunds = [] }) => {
  const [metrics, setMetrics] = useState<FundMetrics[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadFundMetrics = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // In a real implementation, this would call your API
      const response = await fetch(`/api/funds/metrics?funds=${selectedFunds.join(',')}`);
      if (!response.ok) {
        throw new Error('Failed to load fund metrics');
      }
      const data = await response.json();
      setMetrics(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load fund metrics');
    } finally {
      setLoading(false);
    }
  }, [selectedFunds]);

  useEffect(() => {
    if (selectedFunds.length > 0) {
      loadFundMetrics();
    }
  }, [selectedFunds, loadFundMetrics]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'primary';
      case 'liquidated': return 'success';
      case 'realizing': return 'warning';
      default: return 'default';
    }
  };

  const getMetricColor = (value: number, benchmark: number = 1.0) => {
    if (value >= benchmark * 1.2) return '#4caf50'; // Green for above benchmark
    if (value >= benchmark) return '#ff9800'; // Orange for at benchmark
    return '#f44336'; // Red for below benchmark
  };

  const formatMetric = (value: number, type: 'ratio' | 'multiple' = 'ratio') => {
    if (type === 'multiple') {
      return `${value.toFixed(2)}x`;
    }
    return `${(value * 100).toFixed(1)}%`;
  };

  const MetricCard: React.FC<{
    title: string;
    value: number;
    fundName: string;
    icon: React.ReactNode;
    type?: 'ratio' | 'multiple';
    benchmark?: number;
  }> = ({ title, value, fundName, icon, type = 'ratio', benchmark = 1.0 }) => {
    const color = getMetricColor(value, benchmark);
    const progressValue = Math.min((value / (benchmark * 2)) * 100, 100);

    return (
      <Card sx={{ height: '100%' }}>
        <CardContent>
          <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
            <Typography variant="h6" component="div">
              {title}
            </Typography>
            {icon}
          </Box>

          <Typography variant="h4" sx={{ color, fontWeight: 'bold', mb: 1 }}>
            {formatMetric(value, type)}
          </Typography>

          <Typography variant="body2" color="text.secondary" gutterBottom>
            {fundName}
          </Typography>

          <LinearProgress
            variant="determinate"
            value={progressValue}
            sx={{
              height: 8,
              borderRadius: 4,
              '& .MuiLinearProgress-bar': {
                backgroundColor: color,
              },
            }}
          />

          <Box display="flex" justifyContent="space-between" mt={1}>
            <Typography variant="caption" color="text.secondary">
              vs Benchmark
            </Typography>
            <Typography variant="caption" sx={{ color }}>
              {formatMetric(benchmark, type)}
            </Typography>
          </Box>
        </CardContent>
      </Card>
    );
  };

  if (selectedFunds.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          Select funds to view metrics
        </Typography>
      </Paper>
    );
  }

  if (loading) {
    return (
      <Paper sx={{ p: 3, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Paper>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Paper sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        Fund Performance Metrics
      </Typography>

      <Grid container spacing={2}>
        {metrics.map((fund) => (
          <Grid item xs={12} key={fund.fundId}>
            <Box mb={2}>
              <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
                <Typography variant="subtitle1" fontWeight="bold">
                  {fund.fundName}
                </Typography>
                <Box display="flex" alignItems="center" gap={1}>
                  <Chip
                    label={`${fund.vintage}`}
                    size="small"
                    variant="outlined"
                  />
                  <Chip
                    label={fund.status}
                    size="small"
                    color={getStatusColor(fund.status)}
                  />
                </Box>
              </Box>

              <Grid container spacing={2}>
                <Grid item xs={6} sm={3}>
                  <Tooltip title="Distributed to Paid-in Capital - Realized returns as multiple of invested capital">
                    <div>
                      <MetricCard
                        title="DPI"
                        value={fund.dpi}
                        fundName={fund.fundName}
                        icon={<AccountBalance />}
                        type="multiple"
                        benchmark={1.0}
                      />
                    </div>
                  </Tooltip>
                </Grid>

                <Grid item xs={6} sm={3}>
                  <Tooltip title="Total Value to Paid-in Capital - Total returns (realized + unrealized) as multiple of invested capital">
                    <div>
                      <MetricCard
                        title="TVPI"
                        value={fund.tvpi}
                        fundName={fund.fundName}
                        icon={<TrendingUp />}
                        type="multiple"
                        benchmark={1.5}
                      />
                    </div>
                  </Tooltip>
                </Grid>

                <Grid item xs={6} sm={3}>
                  <Tooltip title="Residual Value to Paid-in Capital - Unrealized returns as multiple of invested capital">
                    <div>
                      <MetricCard
                        title="RVPI"
                        value={fund.rvpi}
                        fundName={fund.fundName}
                        icon={<ShowChart />}
                        type="multiple"
                        benchmark={0.5}
                      />
                    </div>
                  </Tooltip>
                </Grid>

                <Grid item xs={6} sm={3}>
                  <Tooltip title="Multiple on Invested Capital - Total returns as multiple of invested capital (same as TVPI)">
                    <div>
                      <MetricCard
                        title="MOIC"
                        value={fund.moic}
                        fundName={fund.fundName}
                        icon={null}
                        type="multiple"
                        benchmark={1.5}
                      />
                    </div>
                  </Tooltip>
                </Grid>
              </Grid>
            </Box>
          </Grid>
        ))}
      </Grid>

      <Box mt={3} p={2} bgcolor="grey.50" borderRadius={1}>
        <Typography variant="body2" color="text.secondary">
          <strong>Understanding the Metrics:</strong><br />
          • <strong>DPI</strong>: Realized returns (distributions) as a multiple of invested capital<br />
          • <strong>TVPI</strong>: Total value (realized + unrealized) as a multiple of invested capital<br />
          • <strong>RVPI</strong>: Unrealized value (residual NAV) as a multiple of invested capital<br />
          • <strong>MOIC</strong>: Multiple on invested capital (same as TVPI)<br />
          <em>Note: TVPI = DPI + RVPI</em>
        </Typography>
      </Box>
    </Paper>
  );
};
