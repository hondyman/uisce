import React, { useState, useEffect } from 'react';
import type { Bundle } from '../types';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  // LinearProgress unused in this component
  Alert,
  CircularProgress,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  Warning,
  CheckCircle
} from '@mui/icons-material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area
} from 'recharts';

interface LiquidityData {
  fundId: string;
  fundName: string;
  committedCapital: number;
  investedCapital: number;
  unfundedCommitment: number;
  unfundedRatio: number;
  capitalCalls: Array<{
    date: string;
    amount: number;
    cumulativeAmount: number;
  }>;
  projectedLiquidity: Array<{
    date: string;
    projectedLiquidity: number;
    scenario: 'base' | 'optimistic' | 'pessimistic';
  }>;
  nextCallDate: string;
  nextCallAmount: number;
  liquidityStressTest: {
    severe: number;
    moderate: number;
    mild: number;
  };
}

interface LiquidityPanelProps {
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: Bundle | null;
}

export const LiquidityPanel: React.FC<LiquidityPanelProps> = ({ selectedFunds = [] }) => {
  const [liquidityData, setLiquidityData] = useState<LiquidityData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedScenario, _setSelectedScenario] = useState<'base' | 'optimistic' | 'pessimistic'>('base');

  const loadLiquidityData = React.useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // In a real implementation, this would call your API
      const response = await fetch(`/api/funds/liquidity?funds=${selectedFunds.join(',')}`);
      if (!response.ok) {
        throw new Error('Failed to load liquidity data');
      }
      const data = await response.json();
      setLiquidityData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load liquidity data');
    } finally {
      setLoading(false);
    }
  }, [selectedFunds]);

  useEffect(() => {
    if (selectedFunds.length > 0) {
      loadLiquidityData();
    }
  }, [selectedFunds, loadLiquidityData]);

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatPercentage = (value: number) => {
    return `${(value * 100).toFixed(1)}%`;
  };

  const getLiquidityStatus = (ratio: number) => {
    if (ratio < 0.1) return { color: 'success', label: 'Well Funded', icon: <CheckCircle /> };
    if (ratio < 0.3) return { color: 'warning', label: 'Moderate', icon: <Warning /> };
    return { color: 'error', label: 'High Commitment', icon: <Warning /> };
  };

  const LiquidityCard: React.FC<{
    title: string;
    value: number;
    subtitle?: string;
    format: 'currency' | 'percentage';
    color?: string;
  }> = ({ title, value, subtitle, format, color = 'primary' }) => (
    <Card>
      <CardContent>
        <Typography variant="h6" color={color} gutterBottom>
          {title}
        </Typography>
        <Typography variant="h4" component="div" sx={{ fontWeight: 'bold' }}>
          {format === 'currency' ? formatCurrency(value) : formatPercentage(value)}
        </Typography>
        {subtitle && (
          <Typography variant="body2" color="text.secondary">
            {subtitle}
          </Typography>
        )}
      </CardContent>
    </Card>
  );

  if (selectedFunds.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          Select funds to view liquidity analysis
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
    <Box>
      <Typography variant="h6" gutterBottom sx={{ mb: 3 }}>
        Liquidity & Commitment Analysis
      </Typography>

      {liquidityData.map((fund) => {
  const status = getLiquidityStatus(fund.unfundedRatio);

        return (
          <Paper key={fund.fundId} sx={{ p: 3, mb: 3 }}>
            <Box display="flex" alignItems="center" justifyContent="space-between" mb={3}>
              <Typography variant="h6">{fund.fundName}</Typography>
              <Chip
                label={status.label}
                color={status.color as any}
                icon={status.icon}
              />
            </Box>

            {/* Key Metrics */}
            <Grid container spacing={3} sx={{ mb: 3 }}>
              <Grid item xs={12} md={3}>
                <LiquidityCard
                  title="Committed Capital"
                  value={fund.committedCapital}
                  format="currency"
                  color="primary"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <LiquidityCard
                  title="Invested Capital"
                  value={fund.investedCapital}
                  format="currency"
                  color="success"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <LiquidityCard
                  title="Unfunded Commitment"
                  value={fund.unfundedCommitment}
                  subtitle={`Remaining obligation`}
                  format="currency"
                  color="warning"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <LiquidityCard
                  title="Unfunded Ratio"
                  value={fund.unfundedRatio}
                  subtitle={`Of total commitment`}
                  format="percentage"
                  color={fund.unfundedRatio > 0.3 ? 'error' : 'info'}
                />
              </Grid>
            </Grid>

            {/* Capital Call Pacing */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" gutterBottom>
                Capital Call Pacing
              </Typography>
              <Box sx={{ width: '100%', height: 300 }}>
                <ResponsiveContainer>
                  <AreaChart data={fund.capitalCalls}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="date"
                      tick={{ fontSize: 12 }}
                      angle={-45}
                      textAnchor="end"
                      height={80}
                    />
                    <YAxis tickFormatter={(value) => formatCurrency(value)} />
                    <Tooltip formatter={(value) => [formatCurrency(value as number), 'Cumulative Calls']} />
                    <Area
                      type="monotone"
                      dataKey="cumulativeAmount"
                      stroke="#8884d8"
                      fill="#8884d8"
                      fillOpacity={0.6}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </Box>
            </Box>

            {/* Next Capital Call */}
            {fund.nextCallDate && (
              <Alert severity="info" sx={{ mb: 3 }}>
                <Typography variant="body2">
                  <strong>Next Capital Call:</strong> {formatCurrency(fund.nextCallAmount)} expected on {fund.nextCallDate}
                </Typography>
              </Alert>
            )}

            {/* Projected Liquidity */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" gutterBottom>
                Projected Liquidity Timeline
              </Typography>
              <Box sx={{ width: '100%', height: 300 }}>
                <ResponsiveContainer>
                  <LineChart data={fund.projectedLiquidity.filter(d => d.scenario === selectedScenario)}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="date"
                      tick={{ fontSize: 12 }}
                      angle={-45}
                      textAnchor="end"
                      height={80}
                    />
                    <YAxis tickFormatter={(value) => formatCurrency(value)} />
                    <Tooltip formatter={(value) => [formatCurrency(value as number), 'Projected Liquidity']} />
                    <Line
                      type="monotone"
                      dataKey="projectedLiquidity"
                      stroke="#82ca9d"
                      strokeWidth={3}
                      dot={{ r: 4 }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </Box>
            </Box>

            {/* Stress Test Results */}
            <Box>
              <Typography variant="subtitle1" gutterBottom>
                Liquidity Stress Test
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Scenario</TableCell>
                      <TableCell align="right">Liquidity Impact</TableCell>
                      <TableCell align="right">Days to Depletion</TableCell>
                      <TableCell>Risk Level</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell>Mild Market Stress</TableCell>
                      <TableCell align="right">{formatCurrency(fund.liquidityStressTest.mild)}</TableCell>
                      <TableCell align="right">90 days</TableCell>
                      <TableCell>
                        <Chip label="Low" size="small" color="success" />
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>Moderate Market Stress</TableCell>
                      <TableCell align="right">{formatCurrency(fund.liquidityStressTest.moderate)}</TableCell>
                      <TableCell align="right">45 days</TableCell>
                      <TableCell>
                        <Chip label="Medium" size="small" color="warning" />
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>Severe Market Stress</TableCell>
                      <TableCell align="right">{formatCurrency(fund.liquidityStressTest.severe)}</TableCell>
                      <TableCell align="right">15 days</TableCell>
                      <TableCell>
                        <Chip label="High" size="small" color="error" />
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          </Paper>
        );
      })}
    </Box>
  );
};
