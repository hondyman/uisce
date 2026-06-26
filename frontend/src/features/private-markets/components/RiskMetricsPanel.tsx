import type { FC } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
} from '@mui/material';
import { TrendingUp, Warning, CheckCircle } from '@mui/icons-material';

interface RiskMetricsPanelProps {
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: any | null;
}

export const RiskMetricsPanel: FC<RiskMetricsPanelProps> = ({ selectedFunds: _selectedFunds = [] }) => {
  // Mock risk metrics data
  const riskMetrics = {
    volatility: 12.5,
    sharpeRatio: 1.8,
    maxDrawdown: -8.3,
    var95: -15.2,
    beta: 0.85,
    trackingError: 4.2
  };

  const getRiskColor = (value: number, threshold: number, isPositive: boolean = false) => {
    if (isPositive) {
      return value >= threshold ? 'success.main' : 'warning.main';
    }
    return Math.abs(value) <= threshold ? 'success.main' : 'warning.main';
  };

  const getRiskIcon = (value: number, threshold: number, isPositive: boolean = false) => {
    const isGood = isPositive ? value >= threshold : Math.abs(value) <= threshold;
    return isGood ? <CheckCircle color="success" /> : <Warning color="warning" />;
  };

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Risk Metrics
      </Typography>

      <Grid container spacing={2}>
        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                {getRiskIcon(riskMetrics.volatility, 15)}
                <Typography variant="body2" sx={{ ml: 1 }}>
                  Volatility
                </Typography>
              </Box>
              <Typography variant="h5" color={getRiskColor(riskMetrics.volatility, 15)}>
                {riskMetrics.volatility}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                {getRiskIcon(riskMetrics.sharpeRatio, 1.5, true)}
                <Typography variant="body2" sx={{ ml: 1 }}>
                  Sharpe Ratio
                </Typography>
              </Box>
              <Typography variant="h5" color={getRiskColor(riskMetrics.sharpeRatio, 1.5, true)}>
                {riskMetrics.sharpeRatio}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                {getRiskIcon(riskMetrics.maxDrawdown, 10)}
                <Typography variant="body2" sx={{ ml: 1 }}>
                  Max Drawdown
                </Typography>
              </Box>
              <Typography variant="h5" color={getRiskColor(riskMetrics.maxDrawdown, 10)}>
                {riskMetrics.maxDrawdown}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                {getRiskIcon(riskMetrics.var95, 20)}
                <Typography variant="body2" sx={{ ml: 1 }}>
                  VaR (95%)
                </Typography>
              </Box>
              <Typography variant="h5" color={getRiskColor(riskMetrics.var95, 20)}>
                {riskMetrics.var95}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <TrendingUp color="primary" />
                <Typography variant="body2" sx={{ ml: 1 }}>
                  Beta
                </Typography>
              </Box>
              <Typography variant="h5">
                {riskMetrics.beta}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                {getRiskIcon(riskMetrics.trackingError, 5)}
                <Typography variant="body2" sx={{ ml: 1 }}>
                  Tracking Error
                </Typography>
              </Box>
              <Typography variant="h5" color={getRiskColor(riskMetrics.trackingError, 5)}>
                {riskMetrics.trackingError}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Paper>
  );
};
