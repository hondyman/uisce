import React, { useState } from 'react';
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
  WaterDrop,
} from '@mui/icons-material';
import { FundSelector } from './components/FundSelector';
import { IRRCurveChart } from './components/IRRCurveChart';
import { JCurvePlot } from './components/JCurvePlot';
import { MultipleOverlayPanel } from './components/MultipleOverlayPanel';
// BenchmarkComparison used elsewhere; not needed in LP dashboard file
import { LiquidityPanel } from './components/LiquidityPanel';
import { PerformanceAttributionChart } from './components/PerformanceAttributionChart';
import { RiskMetricsPanel } from './components/RiskMetricsPanel';
import { useExplorer } from './ExplorerContext';

interface LPDashboardProps {
  userId?: string;
  realTimeData?: { [fundId: string]: { metrics: any; lastUpdate: string } };
}

export const LPPrivateMarketsDashboard: React.FC<LPDashboardProps> = ({ realTimeData = {} }) => {
  const { bundle, excelResults, selectedEntities, setSelectedEntities, isLoading: contextLoading, error: contextError } = useExplorer();
  const [activeTab, setActiveTab] = useState(0);
  const selectedFunds = selectedEntities;

  // Get available funds from bundle or use mock data
  const availableFunds = bundle?.metrics?.map(m => ({
    id: m.node_id,
    name: m.name,
    vintage: 2020, // Mock data
    manager: 'Sample Manager',
    strategy: m.subcategory || 'General',
    geography: 'Global',
    status: 'active' as const
  })) || [];

  const handleFundSelection = (fundIds: string[]) => {
    setSelectedEntities(fundIds);
  };

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  if (contextLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (contextError) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {contextError}
      </Alert>
    );
  }

  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          LP Private Markets Dashboard
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
          selectedFunds={selectedEntities}
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
                      <Typography variant="h6">Selected Funds</Typography>
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
                      <Typography variant="h6">Avg Net IRR</Typography>
                      <Typography variant="h4">12.4%</Typography>
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
                      <Typography variant="h6">Avg TVPI</Typography>
                      <Typography variant="h4">1.8x</Typography>
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
                      <Typography variant="h6">Liquidity Ratio</Typography>
                      <Typography variant="h4">68%</Typography>
                    </Box>
                    <WaterDrop fontSize="large" />
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
              <Tab label="Performance Analysis" />
              <Tab label="Liquidity & Cash Flow" />
              <Tab label="Benchmarking" />
              <Tab label="Risk & Attribution" />
            </Tabs>

            <Box sx={{ p: 3 }}>
              {activeTab === 0 && (
                <Grid container spacing={3}>
                  <Grid item xs={12} lg={8}>
                    <IRRCurveChart 
                      selectedFunds={selectedEntities} 
                      excelResults={excelResults}
                      bundle={bundle}
                    />
                  </Grid>
                  <Grid item xs={12} lg={4}>
                    <MultipleOverlayPanel 
                      selectedFunds={selectedEntities} 
                      excelResults={excelResults}
                      bundle={bundle}
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <JCurvePlot 
                      selectedFunds={selectedEntities} 
                      excelResults={excelResults}
                      bundle={bundle}
                    />
                  </Grid>
                </Grid>
              )}

              {activeTab === 1 && (
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <LiquidityPanel 
                      selectedFunds={selectedEntities} 
                      excelResults={excelResults}
                      bundle={bundle}
                    />
                  </Grid>
                </Grid>
              )}

              {activeTab === 3 && (
                <Grid container spacing={3}>
                  <Grid item xs={12}>
                    <PerformanceAttributionChart 
                      selectedFunds={selectedEntities} 
                      excelResults={excelResults}
                      bundle={bundle}
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <RiskMetricsPanel 
                      selectedFunds={selectedEntities} 
                      excelResults={excelResults}
                      bundle={bundle}
                    />
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
