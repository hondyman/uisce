import React, { useState } from 'react';
import { useParams } from 'react-router-dom';
import { usePortfolioData } from '../../hooks/usePortfolioData';
import { useDashboardContext } from '../../contexts/DashboardContext';
import {
  PortfolioOverviewCard,
  RiskSnapshotCard,
  ComplianceSnapshotCard,
} from './PortfolioCards';
import {
  HoldingsTable,
  SectorWeights,
  ScenarioChart,
} from './PortfolioCharts';
import { FactorExposureChart } from './FactorExposureChart';
import { RuleBreachTable } from './RuleBreachTable';
import { ScenarioPnLChart } from './ScenarioPnLChart';
import {
  Box,
  Container,
  Tabs,
  Tab,
  Typography,
  Paper,
  Alert,
  AlertTitle,
  Button,
  Grid,
  Card,
  CardContent,
  LinearProgress,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Download as DownloadIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { useMaterialTheme } from '../../hooks/useMaterialTheme';

type TabType = 'overview' | 'holdings' | 'risk' | 'compliance' | 'scenarios';

interface TabConfig {
  id: TabType;
  label: string;
  icon: React.ReactNode;
}

const TABS: TabConfig[] = [
  { id: 'overview', label: 'Overview', icon: '📊' },
  { id: 'holdings', label: 'Holdings', icon: '📋' },
  { id: 'risk', label: 'Risk & Factors', icon: '⚠️' },
  { id: 'compliance', label: 'Compliance', icon: '✓' },
  { id: 'scenarios', label: 'Scenario Analysis', icon: '📈' },
];

interface ComplianceBreachProps {
  rule_code: string;
  metric_value: number;
  threshold_value: number;
  severity: 'hard' | 'soft';
}

const ComplianceBreach: React.FC<ComplianceBreachProps> = ({
  rule_code,
  metric_value,
  threshold_value,
  severity,
}) => {
  const theme = useTheme();
  const isHard = severity === 'hard';

  const color = isHard ? 'error' : 'warning';
  const icon = isHard ? '🚨' : '⚠️';
  const title = isHard ? 'Hard Breach' : 'Soft Breach';

  return (
    <Paper
      elevation={0}
      sx={{
        p: 2,
        borderLeft: 4,
        borderColor: isHard ? 'error.main' : 'warning.main',
        backgroundColor: isHard 
          ? theme.palette.error.lighter || 'rgba(211, 47, 47, 0.05)'
          : theme.palette.warning.lighter || 'rgba(251, 188, 28, 0.05)',
      }}
    >
      <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-start' }}>
        <Typography sx={{ fontSize: '1.5rem' }}>{icon}</Typography>
        <Box sx={{ flex: 1 }}>
          <Typography
            variant="subtitle2"
            sx={{
              fontWeight: 700,
              color: isHard ? 'error.dark' : 'warning.dark',
              mb: 0.5,
            }}
          >
            {title}: {rule_code}
          </Typography>
          <Typography variant="caption" color="textSecondary">
            Current: {metric_value.toFixed(4)} | Limit: {threshold_value.toFixed(4)}
          </Typography>
        </Box>
      </Box>
    </Paper>
  );
};

export const PortfolioDetailPage: React.FC = () => {
  const { portfolioId } = useParams<{ portfolioId: string }>();
  const { valuationDate } = useDashboardContext();
  const portfolio = usePortfolioData(portfolioId || null, valuationDate);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const { backgroundColor, borderColor } = useMaterialTheme();

  const [activeTab, setActiveTab] = useState<number>(0);

  if (!portfolioId) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">Portfolio ID not found</Alert>
      </Box>
    );
  }

  const overview = portfolio.overview.data;
  const tabIndex = Math.max(0, TABS.findIndex((tab) => tab.id === (TABS[activeTab]?.id || 'overview')));

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Box sx={{ minHeight: '100vh', backgroundColor: theme.palette.background.default }}>
      {/* Header */}
      <Paper elevation={0} sx={{ borderBottom: 1, borderColor, p: 3, mb: 3 }}>
        <Container maxWidth="lg">
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              gap: 2,
              flexDirection: isMobile ? 'column' : 'row',
              mb: 2,
            }}
          >
            <Box>
              <Typography
                variant="h4"
                sx={{ fontWeight: 700, mb: 1 }}
              >
                {overview?.name || 'Loading...'}
              </Typography>
              <Typography variant="body2" color="textSecondary">
                Portfolio ID: <code>{portfolioId}</code>
              </Typography>
            </Box>

            <Button
              variant="contained"
              startIcon={<DownloadIcon />}
              sx={{ whiteSpace: 'nowrap' }}
            >
              Export PDF
            </Button>
          </Box>

          {/* Error Alert */}
          {portfolio.isError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              <AlertTitle>Error</AlertTitle>
              {portfolio.error?.message || 'Failed to load portfolio data'}
            </Alert>
          )}
        </Container>
      </Paper>

      {/* Main Content */}
      <Container maxWidth="lg" sx={{ pb: 6 }}>
        {/* Tab Navigation */}
        <Paper elevation={0} sx={{ borderBottom: 1, borderColor, mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={handleTabChange}
            variant={isMobile ? 'scrollable' : 'standard'}
            scrollButtonsDisplay={isMobile ? 'auto' : 'off'}
            sx={{
              '& .MuiTab-root': {
                fontSize: isMobile ? '0.875rem' : '1rem',
                textTransform: 'none',
                fontWeight: 600,
                gap: 1,
              },
            }}
          >
            {TABS.map((tab) => (
              <Tab
                key={tab.id}
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <span>{tab.icon}</span>
                    {tab.label}
                  </Box>
                }
                id={`portfolio-tab-${tab.id}`}
                aria-controls={`portfolio-tabpanel-${tab.id}`}
              />
            ))}
          </Tabs>
        </Paper>

        {/* Tab Content: Overview */}
        {activeTab === 0 && (
          <Box sx={{ display: 'grid', gap: 3 }}>
            {/* Summary Cards */}
            <Grid container spacing={2}>
              <Grid item xs={12} sm={6} md={4}>
                <PortfolioOverviewCard
                  data={portfolio.overview.data}
                  isLoading={portfolio.overview.isLoading}
                  error={portfolio.overview.error}
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <RiskSnapshotCard
                  data={portfolio.risk.data}
                  isLoading={portfolio.risk.isLoading}
                  error={portfolio.risk.error}
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <ComplianceSnapshotCard
                  data={portfolio.compliance.data}
                  isLoading={portfolio.compliance.isLoading}
                  error={portfolio.compliance.error}
                />
              </Grid>
            </Grid>

            {/* Holdings & Scenarios */}
            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <HoldingsTable
                  data={portfolio.holdings.data}
                  isLoading={portfolio.holdings.isLoading}
                  error={portfolio.holdings.error}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <ScenarioChart
                  data={portfolio.scenarios.data?.results}
                  isLoading={portfolio.scenarios.isLoading}
                  error={portfolio.scenarios.error}
                />
              </Grid>
            </Grid>

            {/* Compliance Breaches */}
            {portfolio.compliance.data &&
              (portfolio.compliance.data.hard_breaches.length > 0 ||
                portfolio.compliance.data.soft_breaches.length > 0) && (
                <Box sx={{ mt: 2 }}>
                  <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
                    Compliance Status
                  </Typography>
                  <Box sx={{ display: 'grid', gap: 1.5 }}>
                    {portfolio.compliance.data.hard_breaches.map((breach) => (
                      <ComplianceBreach
                        key={breach.rule_code}
                        rule_code={breach.rule_code}
                        metric_value={breach.metric_value}
                        threshold_value={breach.threshold_value}
                        severity="hard"
                      />
                    ))}
                    {portfolio.compliance.data.soft_breaches.map((breach) => (
                      <ComplianceBreach
                        key={breach.rule_code}
                        rule_code={breach.rule_code}
                        metric_value={breach.metric_value}
                        threshold_value={breach.threshold_value}
                        severity="soft"
                      />
                    ))}
                  </Box>
                </Box>
              )}
          </Box>
        )}

        {/* Tab Content: Holdings */}
        {activeTab === 1 && (
          <Box sx={{ display: 'grid', gap: 3 }}>
            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <Paper elevation={0} sx={{ p: 2, backgroundColor, borderColor, border: 1 }}>
                  <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
                    Sector Breakdown
                  </Typography>
                  <SectorWeights
                    data={portfolio.holdings.data?.sector_weights}
                    isLoading={portfolio.holdings.isLoading}
                  />
                </Paper>
              </Grid>
              <Grid item xs={12} md={6}>
                <Paper elevation={0} sx={{ p: 2, backgroundColor, borderColor, border: 1 }}>
                  <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
                    Geographic Distribution
                  </Typography>
                  <SectorWeights
                    data={portfolio.holdings.data?.country_weights?.map((c) => ({
                      sector: c.country,
                      weight: c.weight,
                    }))}
                    isLoading={portfolio.holdings.isLoading}
                  />
                </Paper>
              </Grid>
            </Grid>
            <HoldingsTable
              data={portfolio.holdings.data}
              isLoading={portfolio.holdings.isLoading}
              error={portfolio.holdings.error}
            />
          </Box>
        )}

        {/* Tab Content: Risk & Factors */}
        {activeTab === 2 && (
          <Box sx={{ display: 'grid', gap: 3 }}>
            <FactorExposureChart
              data={portfolio.risk.data?.factor_exposures}
              isLoading={portfolio.risk.isLoading}
              error={portfolio.risk.error}
            />

            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <RiskSnapshotCard
                  data={portfolio.risk.data}
                  isLoading={portfolio.risk.isLoading}
                  error={portfolio.risk.error}
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <Paper elevation={0} sx={{ p: 2, backgroundColor, borderColor, border: 1 }}>
                  <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
                    Factor Exposures (Legacy View)
                  </Typography>
                  <Box sx={{ display: 'grid', gap: 2 }}>
                    {portfolio.risk.data?.factor_exposures.map((factor) => (
                      <Box key={factor.factor_id}>
                        <Box
                          sx={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            mb: 1,
                            gap: 1,
                          }}
                        >
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>
                            {factor.factor_id}
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{
                              fontWeight: 700,
                              fontFamily: 'monospace',
                              color: factor.exposure > 0 ? 'success.main' : 'error.main',
                            }}
                          >
                            {factor.exposure > 0 ? '+' : ''}
                            {factor.exposure.toFixed(2)}
                          </Typography>
                        </Box>
                        <LinearProgress
                          variant="determinate"
                          value={Math.min(
                            Math.abs(factor.exposure) * 20,
                            100
                          )}
                          sx={{
                            height: 6,
                            borderRadius: 1,
                            backgroundColor: theme.palette.action.hover,
                            '& .MuiLinearProgress-bar': {
                              backgroundColor:
                                factor.exposure > 0
                                  ? 'success.main'
                                  : 'error.main',
                            },
                          }}
                        />
                      </Box>
                    ))}
                  </Box>
                </Paper>
              </Grid>
            </Grid>
          </Box>
        )}

        {/* Tab Content: Compliance */}
        {activeTab === 3 && (
          <Box sx={{ display: 'grid', gap: 3 }}>
            <ComplianceSnapshotCard
              data={portfolio.compliance.data}
              isLoading={portfolio.compliance.isLoading}
              error={portfolio.compliance.error}
            />

            {portfolio.compliance.data &&
              (portfolio.compliance.data.hard_breaches.length > 0 ||
                portfolio.compliance.data.soft_breaches.length > 0) && (
                <RuleBreachTable
                  hard_breaches={portfolio.compliance.data.hard_breaches}
                  soft_breaches={portfolio.compliance.data.soft_breaches}
                  isLoading={portfolio.compliance.isLoading}
                  error={portfolio.compliance.error}
                />
              )}

            {(!portfolio.compliance.data ||
              (portfolio.compliance.data.hard_breaches.length === 0 &&
                portfolio.compliance.data.soft_breaches.length === 0)) &&
              !portfolio.compliance.isLoading && (
                <Alert severity="success">
                  <AlertTitle>Compliance Status</AlertTitle>
                  ✓ No compliance breaches detected
                </Alert>
              )}
          </Box>
        )}

        {/* Tab Content: Scenarios */}
        {activeTab === 4 && (
          <Box sx={{ display: 'grid', gap: 3 }}>
            <ScenarioPnLChart
              data={portfolio.scenarios.data?.results}
              isLoading={portfolio.scenarios.isLoading}
              error={portfolio.scenarios.error}
            />

            {portfolio.scenarios.data && (
              <Paper elevation={0} sx={{ p: 3, backgroundColor, borderColor, border: 1 }}>
                <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
                  Detailed Results
                </Typography>
                <Box sx={{ display: 'grid', gap: 1 }}>
                  {portfolio.scenarios.data.results.map((scenario) => (
                    <Card
                      key={scenario.scenario_id}
                      elevation={0}
                      sx={{
                        p: 2,
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        backgroundColor: theme.palette.action.hover,
                        border: 1,
                        borderColor: 'divider',
                      }}
                    >
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {scenario.name}
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          fontWeight: 700,
                          fontFamily: 'monospace',
                          color:
                            scenario.pnl < 0 ? 'error.main' : 'success.main',
                        }}
                      >
                        {scenario.pnl < 0 ? '-' : '+'}$
                        {Math.abs(scenario.pnl / 1000000).toFixed(1)}M
                      </Typography>
                    </Card>
                  ))}
                </Box>
              </Paper>
            )}
          </Box>
        )}
      </Container>
    </Box>
  );
};

export default PortfolioDetailPage;
