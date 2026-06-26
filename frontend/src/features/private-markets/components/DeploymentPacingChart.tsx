import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  LinearProgress,
  Alert,
  CircularProgress,
  Chip
} from '@mui/material';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  LineChart,
  Line,
  ReferenceLine
} from 'recharts';

interface DeploymentData {
  fundId: string;
  fundName: string;
  vintage: number;
  committedCapital: number;
  deployedCapital: number;
  deploymentPercentage: number;
  targetDeployment: number;
  deploymentByQuarter: Array<{
    quarter: string;
    deployed: number;
    target: number;
    cumulativeDeployed: number;
    cumulativeTarget: number;
  }>;
  sectorDeployment: Array<{
    sector: string;
    deployed: number;
    percentage: number;
  }>;
}

interface DeploymentPacingChartProps {
  selectedFunds?: string[];
}

export const DeploymentPacingChart: React.FC<DeploymentPacingChartProps> = ({ selectedFunds = [] }) => {
  const [deploymentData, setDeploymentData] = useState<DeploymentData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadDeploymentData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // In a real implementation, this would call your API
      const response = await fetch(`/api/funds/deployment?funds=${selectedFunds.join(',')}`);
      if (!response.ok) {
        throw new Error('Failed to load deployment data');
      }
      const data = await response.json();
      setDeploymentData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load deployment data');
    } finally {
      setLoading(false);
    }
  }, [selectedFunds]);

  useEffect(() => {
    if (selectedFunds.length > 0) {
      loadDeploymentData();
    }
  }, [selectedFunds, loadDeploymentData]);

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

  const getDeploymentStatus = (actual: number, target: number) => {
    const ratio = actual / target;
    if (ratio >= 0.95) return { color: 'success', label: 'On Track' };
    if (ratio >= 0.85) return { color: 'warning', label: 'Slight Delay' };
    return { color: 'error', label: 'Behind Schedule' };
  };

  if (selectedFunds.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          Select funds to view deployment pacing
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
        Capital Deployment & Pacing
      </Typography>

      {deploymentData.map((fund) => {
        const status = getDeploymentStatus(fund.deploymentPercentage, fund.targetDeployment);

        return (
          <Box key={fund.fundId} sx={{ mb: 4 }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="subtitle1" fontWeight="bold">
                {fund.fundName} ({fund.vintage})
              </Typography>
              <Chip
                label={status.label}
                color={status.color as any}
                size="small"
              />
            </Box>

            {/* Deployment Summary Cards */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
              <Grid item xs={12} md={4}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" color="primary">
                      Committed Capital
                    </Typography>
                    <Typography variant="h4">
                      {formatCurrency(fund.committedCapital)}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={4}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" color="success.main">
                      Deployed Capital
                    </Typography>
                    <Typography variant="h4">
                      {formatCurrency(fund.deployedCapital)}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {formatPercentage(fund.deploymentPercentage)} of committed
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={4}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" color="warning.main">
                      Target Deployment
                    </Typography>
                    <Typography variant="h4">
                      {formatPercentage(fund.targetDeployment)}
                    </Typography>
                    <LinearProgress
                      variant="determinate"
                      value={Math.min((fund.deploymentPercentage / fund.targetDeployment) * 100, 100)}
                      sx={{ mt: 1, height: 8, borderRadius: 4 }}
                      color={fund.deploymentPercentage >= fund.targetDeployment ? 'success' : 'warning'}
                    />
                  </CardContent>
                </Card>
              </Grid>
            </Grid>

            {/* Quarterly Deployment Chart */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle2" gutterBottom>
                Quarterly Deployment Pace
              </Typography>
              <Box sx={{ width: '100%', height: 300 }}>
                <ResponsiveContainer>
                  <BarChart data={fund.deploymentByQuarter}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="quarter"
                      tick={{ fontSize: 12 }}
                      angle={-45}
                      textAnchor="end"
                      height={80}
                    />
                    <YAxis tickFormatter={formatCurrency} />
                    <Tooltip formatter={(value) => [formatCurrency(value as number), '']} />
                    <Legend />
                    <Bar dataKey="deployed" fill="#8884d8" name="Actual Deployment" />
                    <Bar dataKey="target" fill="#82ca9d" name="Target Deployment" />
                  </BarChart>
                </ResponsiveContainer>
              </Box>
            </Box>

            {/* Cumulative Deployment Trend */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle2" gutterBottom>
                Cumulative Deployment Trend
              </Typography>
              <Box sx={{ width: '100%', height: 250 }}>
                <ResponsiveContainer>
                  <LineChart data={fund.deploymentByQuarter}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="quarter"
                      tick={{ fontSize: 12 }}
                      angle={-45}
                      textAnchor="end"
                      height={80}
                    />
                    <YAxis tickFormatter={formatCurrency} />
                    <Tooltip formatter={(value) => [formatCurrency(value as number), '']} />
                    <Legend />

                    {/* Target reference line */}
                    <ReferenceLine
                      y={fund.committedCapital * fund.targetDeployment}
                      stroke="#ff0000"
                      strokeDasharray="5 5"
                      label="Target Deployment"
                    />

                    <Line
                      type="monotone"
                      dataKey="cumulativeDeployed"
                      stroke="#8884d8"
                      strokeWidth={3}
                      name="Cumulative Deployed"
                    />
                    <Line
                      type="monotone"
                      dataKey="cumulativeTarget"
                      stroke="#82ca9d"
                      strokeWidth={2}
                      strokeDasharray="5 5"
                      name="Cumulative Target"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </Box>
            </Box>

            {/* Sector Deployment Breakdown */}
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Deployment by Sector
              </Typography>
              <Box sx={{ width: '100%', height: 250 }}>
                <ResponsiveContainer>
                  <BarChart data={fund.sectorDeployment} layout="horizontal">
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis type="number" tickFormatter={formatCurrency} />
                    <YAxis dataKey="sector" type="category" width={100} />
                    <Tooltip formatter={(value) => [formatCurrency(value as number), 'Deployed']} />
                    <Bar dataKey="deployed" fill="#8884d8" />
                  </BarChart>
                </ResponsiveContainer>
              </Box>
            </Box>
          </Box>
        );
      })}
    </Paper>
  );
};
