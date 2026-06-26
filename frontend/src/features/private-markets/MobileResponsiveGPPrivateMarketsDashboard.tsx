import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
  Chip,
  Button,
  Tabs,
  Tab,
  Alert,
  CircularProgress,
  useTheme,
  useMediaQuery,
  Container,
  Stack,
  Divider,
  IconButton,
} from '@mui/material';
import {
  TrendingUp,
  AccountBalance,
  Assessment,
  // Compare icon not used in this file
  WaterDrop,
  FilterList,
  Download,
  Share,
  Fullscreen,
  FullscreenExit,
} from '@mui/icons-material';
import { FundSelector } from './components/FundSelector';
import { IRRCurveChart } from './components/IRRCurveChart';
// JCurvePlot not used in this mobile variant
import { MultipleOverlayPanel } from './components/MultipleOverlayPanel';
import { BenchmarkComparison } from './components/BenchmarkComparison';
import { RiskMetricsPanel } from './components/RiskMetricsPanel';
import { FundPerformanceTracker } from './components/FundPerformanceTracker';
import { CapitalDeploymentChart } from './components/CapitalDeploymentChart';
import { PortfolioCompositionChart } from './components/PortfolioCompositionChart';

interface Fund {
  id: string;
  name: string;
  vintage: number;
  manager: string;
  strategy: string;
  geography: string;
  status: 'active' | 'liquidated' | 'realizing';
}

interface MobileResponsiveGPPrivateMarketsDashboardProps {
  userId?: string;
  realTimeData?: { [fundId: string]: { metrics: any; lastUpdate: string } };
}

export const MobileResponsiveGPPrivateMarketsDashboard: React.FC<MobileResponsiveGPPrivateMarketsDashboardProps> = ({
  realTimeData = {}
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const isTablet = useMediaQuery(theme.breakpoints.down('md'));

  const [selectedFunds, setSelectedFunds] = useState<string[]>([]);
  const [availableFunds, setAvailableFunds] = useState<Fund[]>([]);
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showFilters, setShowFilters] = useState(false);
  const [fullscreenChart, setFullscreenChart] = useState<string | null>(null);

  useEffect(() => {
    loadAvailableFunds();
  }, []);

  const loadAvailableFunds = async () => {
    try {
      setLoading(true);
      // In a real implementation, this would call your API
      const response = await fetch('/api/funds');
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

  const toggleFilters = () => {
    setShowFilters(!showFilters);
  };

  const toggleFullscreen = (chartId: string) => {
    setFullscreenChart(fullscreenChart === chartId ? null : chartId);
  };

  // Key metrics cards for mobile - GP focused
  const renderKeyMetrics = () => (
    <Grid container spacing={isMobile ? 1 : 2} sx={{ mb: isMobile ? 2 : 3 }}>
      {[
        { title: 'Total Commitments', value: '$8.2B', change: '+12.3%', icon: <AccountBalance />, color: 'primary' },
        { title: 'IRR', value: '18.4%', change: '+3.2%', icon: <TrendingUp />, color: 'success' },
        { title: 'Active Funds', value: '12', change: '+1', icon: <Assessment />, color: 'info' },
        { title: 'Capital Deployed', value: '72%', change: '+5.1%', icon: <WaterDrop />, color: 'warning' },
      ].map((metric, index) => (
        <Grid item xs={6} sm={6} md={3} key={index}>
          <Card
            sx={{
              height: isMobile ? '80px' : '100px',
              cursor: 'pointer',
              transition: 'transform 0.2s',
              '&:hover': {
                transform: isMobile ? 'none' : 'translateY(-4px)',
              },
            }}
          >
            <CardContent sx={{
              p: isMobile ? 1.5 : 2,
              '&:last-child': { pb: isMobile ? 1.5 : 2 }
            }}>
              <Box display="flex" alignItems="center" justifyContent="space-between" mb={0.5}>
                <Box
                  sx={{
                    color: `${metric.color}.main`,
                    fontSize: isMobile ? '1.2rem' : '1.5rem'
                  }}
                >
                  {metric.icon}
                </Box>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{ fontSize: isMobile ? '0.7rem' : '0.75rem' }}
                >
                  {metric.change}
                </Typography>
              </Box>
              <Typography
                variant={isMobile ? "h6" : "h5"}
                sx={{
                  fontWeight: 'bold',
                  fontSize: isMobile ? '1rem' : '1.25rem',
                  mb: 0.5
                }}
              >
                {metric.value}
              </Typography>
              <Typography
                variant="caption"
                color="text.secondary"
                sx={{ fontSize: isMobile ? '0.7rem' : '0.75rem' }}
              >
                {metric.title}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      ))}
    </Grid>
  );

  // Mobile-optimized chart container
  const ChartContainer = ({
    title,
    children,
    chartId,
    height = isMobile ? 250 : 400
  }: {
    title: string;
    children: React.ReactNode;
    chartId: string;
    height?: number;
  }) => (
    <Paper
      sx={{
        p: isMobile ? 1.5 : 2,
        mb: isMobile ? 2 : 3,
        height: fullscreenChart === chartId ? '70vh' : 'auto',
        transition: 'height 0.3s ease',
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
        <Typography
          variant={isMobile ? "subtitle1" : "h6"}
          sx={{ fontWeight: 'bold', fontSize: isMobile ? '1rem' : '1.25rem' }}
        >
          {title}
        </Typography>
        <Box>
          <Button
            size={isMobile ? "small" : "medium"}
            startIcon={<Download />}
            sx={{ mr: 1, fontSize: isMobile ? '0.75rem' : '0.875rem' }}
          >
            {isMobile ? '' : 'Export'}
          </Button>
          <IconButton
            size={isMobile ? "small" : "medium"}
            onClick={() => toggleFullscreen(chartId)}
          >
            {fullscreenChart === chartId ? <FullscreenExit /> : <Fullscreen />}
          </IconButton>
        </Box>
      </Box>
      <Box sx={{ height: fullscreenChart === chartId ? 'calc(70vh - 60px)' : height }}>
        {children}
      </Box>
    </Paper>
  );

  if (loading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight={isMobile ? "60vh" : "400px"}
        sx={{ p: isMobile ? 2 : 3 }}
      >
        <CircularProgress size={isMobile ? 40 : 60} />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert
        severity="error"
        sx={{
          m: isMobile ? 2 : 3,
          fontSize: isMobile ? '0.875rem' : '1rem'
        }}
      >
        {error}
      </Alert>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ py: isMobile ? 1 : 3, px: isMobile ? 1 : 3 }}>
      {/* Mobile Header */}
      {isMobile && (
        <Box sx={{ mb: 2 }}>
          <Typography
            variant="h5"
            sx={{
              fontWeight: 'bold',
              mb: 1,
              fontSize: '1.25rem'
            }}
          >
            GP Dashboard
          </Typography>
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button
              size="small"
              startIcon={<FilterList />}
              onClick={toggleFilters}
              variant={showFilters ? "contained" : "outlined"}
              sx={{ fontSize: '0.75rem' }}
            >
              Filters
            </Button>
            <Button
              size="small"
              startIcon={<Download />}
              sx={{ fontSize: '0.75rem' }}
            >
              Export
            </Button>
            <Button
              size="small"
              startIcon={<Share />}
              sx={{ fontSize: '0.75rem' }}
            >
              Share
            </Button>
          </Stack>
        </Box>
      )}

      {/* Fund Selector - Collapsible on mobile */}
      {(!isMobile || showFilters) && (
        <Paper sx={{ p: isMobile ? 1.5 : 2, mb: isMobile ? 2 : 3 }}>
          <FundSelector
            availableFunds={availableFunds}
            selectedFunds={selectedFunds}
            onSelectionChange={handleFundSelection}
          />
        </Paper>
      )}

      {/* Key Metrics */}
      {renderKeyMetrics()}

      {/* Mobile Stack / Desktop Tabs */}
      {isMobile ? (
        <Stack spacing={2}>
          {/* Performance Tab */}
          <Box>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 'bold',
                mb: 2,
                fontSize: '1.1rem'
              }}
            >
              Performance Management
            </Typography>
            <ChartContainer title="Fund Performance Tracker" chartId="performance">
              <FundPerformanceTracker
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
            <ChartContainer title="IRR Curves" chartId="irr">
              <IRRCurveChart
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
          </Box>

          <Divider />

          {/* Capital Deployment Tab */}
          <Box>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 'bold',
                mb: 2,
                fontSize: '1.1rem'
              }}
            >
              Capital Deployment
            </Typography>
            <ChartContainer title="Capital Deployment" chartId="deployment">
              <CapitalDeploymentChart
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
            <ChartContainer title="Portfolio Composition" chartId="composition">
              <PortfolioCompositionChart
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
          </Box>

          <Divider />

          {/* Risk & Benchmarking Tab */}
          <Box>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 'bold',
                mb: 2,
                fontSize: '1.1rem'
              }}
            >
              Risk & Benchmarking
            </Typography>
            <ChartContainer title="Risk Metrics" chartId="risk">
              <RiskMetricsPanel
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
            <ChartContainer title="Benchmark Comparison" chartId="benchmark">
              <BenchmarkComparison
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
          </Box>
        </Stack>
      ) : (
        // Desktop Tabs
        <Box sx={{ width: '100%' }}>
          <Tabs
            value={activeTab}
            onChange={handleTabChange}
            variant={isTablet ? "scrollable" : "standard"}
            scrollButtons={isTablet ? "auto" : false}
            sx={{
              borderBottom: 1,
              borderColor: 'divider',
              mb: 3,
              '& .MuiTab-root': {
                fontSize: isTablet ? '0.875rem' : '1rem',
                minHeight: isTablet ? '48px' : '64px',
              },
            }}
          >
            <Tab label="Performance" />
            <Tab label="Capital Deployment" />
            <Tab label="Risk Analysis" />
            <Tab label="Advanced" />
          </Tabs>

          {activeTab === 0 && (
            <Grid container spacing={3}>
              <Grid item xs={12} lg={6}>
                <ChartContainer title="Fund Performance Tracker" chartId="performance">
                  <FundPerformanceTracker
                    selectedFunds={selectedFunds}
                  />
                </ChartContainer>
              </Grid>
              <Grid item xs={12} lg={6}>
                <ChartContainer title="IRR Curves" chartId="irr">
                  <IRRCurveChart
                    selectedFunds={selectedFunds}
                  />
                </ChartContainer>
              </Grid>
            </Grid>
          )}

          {activeTab === 1 && (
            <Grid container spacing={3}>
              <Grid item xs={12} lg={6}>
                <ChartContainer title="Capital Deployment" chartId="deployment">
                  <CapitalDeploymentChart
                    selectedFunds={selectedFunds}
                  />
                </ChartContainer>
              </Grid>
              <Grid item xs={12} lg={6}>
                <ChartContainer title="Portfolio Composition" chartId="composition">
                  <PortfolioCompositionChart
                    selectedFunds={selectedFunds}
                  />
                </ChartContainer>
              </Grid>
            </Grid>
          )}

          {activeTab === 2 && (
            <Grid container spacing={3}>
              <Grid item xs={12} lg={6}>
                <ChartContainer title="Risk Metrics" chartId="risk">
                  <RiskMetricsPanel
                    selectedFunds={selectedFunds}
                  />
                </ChartContainer>
              </Grid>
              <Grid item xs={12} lg={6}>
                <ChartContainer title="Benchmark Comparison" chartId="benchmark">
                  <BenchmarkComparison
                    selectedFunds={selectedFunds}
                  />
                </ChartContainer>
              </Grid>
            </Grid>
          )}

          {activeTab === 3 && (
            <ChartContainer title="Multiple Overlay Analysis" chartId="overlay">
              <MultipleOverlayPanel
                selectedFunds={selectedFunds}
              />
            </ChartContainer>
          )}
        </Box>
      )}

      {/* Real-time Data Indicator */}
      {Object.keys(realTimeData).length > 0 && (
        <Box sx={{
          position: 'fixed',
          bottom: isMobile ? 16 : 24,
          right: isMobile ? 16 : 24,
          zIndex: 1000
        }}>
          <Chip
            label={`${Object.keys(realTimeData).length} live updates`}
            color="success"
            size={isMobile ? "small" : "medium"}
            sx={{
              fontSize: isMobile ? '0.75rem' : '0.875rem',
              height: isMobile ? '24px' : '32px'
            }}
          />
        </Box>
      )}
    </Container>
  );
};
