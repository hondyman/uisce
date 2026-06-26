import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
  Chip,
  Tabs,
  Tab,
  Alert,
  CircularProgress
} from '@mui/material';
import {
  TrendingUp,
  AccountBalance,
  Timeline,
  // Assessment icon not used directly here
  AttachMoney,
} from '@mui/icons-material';
import { FundSelector } from './components/FundSelector';
import { DeploymentPacingChart } from './components/DeploymentPacingChart';
import { GrossIRRChart } from './components/GrossIRRChart';
import { NAVEvolutionChart } from './components/NAVEvolutionChart';
import { FeeCarryPanel } from './components/FeeCarryPanel';
import { ValueAttributionBridge } from './components/ValueAttributionBridge';
// ExitTrackingTable and BenchmarkComparison imported where used in other views
import { PerformanceAttributionChart } from './components/PerformanceAttributionChart';
import { RiskMetricsPanel } from './components/RiskMetricsPanel';

interface Fund {
  id: string;
  name: string;
  vintage: number;
  manager: string;
  strategy: string;
  geography: string;
  status: 'active' | 'liquidated' | 'realizing';
}

interface GPPrivateMarketsDashboardProps {
  userId?: string;
  realTimeData?: { [fundId: string]: { metrics: any; lastUpdate: string } };
}

export const GPPrivateMarketsDashboard: React.FC<GPPrivateMarketsDashboardProps> = ({ realTimeData = {} }) => {
  const [selectedFunds, setSelectedFunds] = useState<string[]>([]);
  const [availableFunds, setAvailableFunds] = useState<Fund[]>([]);
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAvailableFunds();
  }, []);

  const loadAvailableFunds = async () => {
    try {
      setLoading(true);
      // In a real implementation, this would call your API
      const response = await fetch('/api/funds?role=gp');
      if (!response.ok) {
        throw new Error('Failed to load funds');
      }
      const funds = await response.json();
      setAvailableFunds(funds);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load funds');
    } finally {
      setLoading(false);
    }
  };

  const handleFundSelection = (fundIds: string[]) => {
    setSelectedFunds(fundIds);
  };

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
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
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          GP Private Markets Dashboard
        </Typography>

        {/* Real-time Data Indicator */}
        {Object.keys(realTimeData).length > 0 && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Chip
              size="small"
              color="success"
              label={`Live: ${Object.keys(realTimeData).length} funds`}
              icon={<TrendingUp />}
            />
            <Typography variant="caption" color="text.secondary">
              Last update: {new Date(Math.max(...Object.values(realTimeData).map(d => new Date(d.lastUpdate).getTime()))).toLocaleTimeString()}
            </Typography>
          </Box>
        )}
      </Box>

      {/* Fund Selector */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <FundSelector
          availableFunds={availableFunds}
          selectedFunds={selectedFunds}
          onSelectionChange={handleFundSelection}
        />
      </Paper>

      {/* Main Dashboard Content */}
      <Grid container spacing={3}>
        {/* Performance Overview Cards */}
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item xs={12} md={3}>
              <Card sx={{ bgcolor: 'primary.light', color: 'primary.contrastText' }}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography variant="h6">Active Funds</Typography>
                      <Typography variant="h4">{selectedFunds.length}</Typography>
                    </Box>
                    <AccountBalance fontSize="large" />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} md={3}>
              <Card sx={{ bgcolor: 'success.light', color: 'success.contrastText' }}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography variant="h6">Avg Gross IRR</Typography>
                      <Typography variant="h4">18.2%</Typography>
                    </Box>
                    <TrendingUp fontSize="large" />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} md={3}>
              <Card sx={{ bgcolor: 'warning.light', color: 'warning.contrastText' }}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography variant="h6">Deployment %</Typography>
                      <Typography variant="h4">78%</Typography>
                    </Box>
                    <Timeline fontSize="large" />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} md={3}>
              <Card sx={{ bgcolor: 'info.light', color: 'info.contrastText' }}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography variant="h6">Fee Income</Typography>
                      <Typography variant="h4">$2.4M</Typography>
                    </Box>
                    <AttachMoney fontSize="large" />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        </Grid>

        {/* Tabbed Content */}
        <Grid item xs={12}>
          <Paper sx={{ width: '100%' }}>
            <Tabs
              value={activeTab}
              onChange={handleTabChange}
              indicatorColor="primary"
              textColor="primary"
              variant="fullWidth"
            >
              <Tab label="Fund Management" />
              <Tab label="Performance & NAV" />
              <Tab label="Fees & Carry" />
              <Tab label="Exits & Benchmarking" />
              <Tab label="Risk & Attribution" />
            </Tabs>

            <Box sx={{ p: 3 }}>
              {activeTab === 0 && (
                <Grid container spacing={3}>
                  <Grid item xs={12} lg={6}>
                    <DeploymentPacingChart selectedFunds={selectedFunds} />
                  </Grid>
                  <Grid item xs={12} lg={6}>
                    <ValueAttributionBridge selectedFunds={selectedFunds} />
                  </Grid>
                </Grid>
              )}

              {activeTab === 1 && (
                <Grid container spacing={3}>
                  <Grid item xs={12} lg={6}>
                    <GrossIRRChart selectedFunds={selectedFunds} />
                  </Grid>
                  <Grid item xs={12} lg={6}>
                    <NAVEvolutionChart selectedFunds={selectedFunds} />
                  </Grid>
                </Grid>
              )}

              {activeTab === 2 && (
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <FeeCarryPanel selectedFunds={selectedFunds} />
                  </Grid>
                </Grid>
              )}

              {activeTab === 4 && (
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <PerformanceAttributionChart selectedFunds={selectedFunds} />
                  </Grid>
                  <Grid item xs={12}>
                    <RiskMetricsPanel selectedFunds={selectedFunds} />
                  </Grid>
                </Grid>
              )}
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};
