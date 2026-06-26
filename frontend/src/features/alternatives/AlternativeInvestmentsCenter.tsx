import React, { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Box,
  Typography,
  Tabs,
  Tab,
  Card,
  CardContent,
  Grid,
  Button,
  CircularProgress,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div hidden={value !== index} {...other}>
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

interface AlternativeInvestment {
  id: string;
  fundName: string;
  fundManager: string;
  assetClass: string;
  vintageYear?: number;
  commitmentAmount: number;
  capitalCalled: number;
  capitalDistributed: number;
  unfundedCommitment: number;
  currentNav: number;
  inceptionDate: string;
}

interface PortfolioSummary {
  totalCommitment: number;
  totalCalled: number;
  totalNav: number;
  totalDistributed: number;
  unfundedCommitment: number;
  totalValue: number;
}

export const AlternativeInvestmentsCenter: React.FC<{ clientId: string }> = ({ clientId }) => {
  const [currentTab, setCurrentTab] = useState(0);

  const { data: investments, isLoading } = useQuery<AlternativeInvestment[]>({
    queryKey: ['alternative-investments', clientId],
    queryFn: async () => {
      const res = await fetch(`/api/v1/alternative-investments?client_id=${clientId}`);
      if (!res.ok) throw new Error('Failed to fetch investments');
      return res.json();
    },
  });

  // Calculate portfolio summary
  const portfolioSummary: PortfolioSummary | null = useMemo(() => {
    if (!investments) return null;

    return {
      totalCommitment: investments.reduce((sum, inv) => sum + inv.commitmentAmount, 0),
      totalCalled: investments.reduce((sum, inv) => sum + inv.capitalCalled, 0),
      totalNav: investments.reduce((sum, inv) => sum + inv.currentNav, 0),
      totalDistributed: investments.reduce((sum, inv) => sum + inv.capitalDistributed, 0),
      unfundedCommitment: investments.reduce((sum, inv) => sum + inv.unfundedCommitment, 0),
      totalValue: investments.reduce((sum, inv) => sum + (inv.currentNav + inv.capitalDistributed), 0),
    };
  }, [investments]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setCurrentTab(newValue);
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">Alternative Investments</Typography>
        <Button variant="contained" startIcon={<AddIcon />}>
          Add Investment
        </Button>
      </Box>

      {/* Portfolio Summary Cards */}
      {portfolioSummary && (
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Total Commitment
                </Typography>
                <Typography variant="h5" fontWeight="bold">
                  ${portfolioSummary.totalCommitment.toLocaleString()}
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Capital Called
                </Typography>
                <Typography variant="h5" fontWeight="bold">
                  ${portfolioSummary.totalCalled.toLocaleString()}
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  {((portfolioSummary.totalCalled / portfolioSummary.totalCommitment) * 100).toFixed(1)}% of commitment
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Current NAV
                </Typography>
                <Typography variant="h5" fontWeight="bold" color="primary">
                  ${portfolioSummary.totalNav.toLocaleString()}
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  + ${portfolioSummary.totalDistributed.toLocaleString()} distributed
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Unfunded Commitment
                </Typography>
                <Typography variant="h5" fontWeight="bold" color="warning.main">
                  ${portfolioSummary.unfundedCommitment.toLocaleString()}
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  Future capital required
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Tabbed Interface */}
      <Card>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={currentTab} onChange={handleTabChange}>
            <Tab label="Holdings" />
            <Tab label="Capital Calls" />
            <Tab label="Performance" />
            <Tab label="Documents" />
          </Tabs>
        </Box>

        <TabPanel value={currentTab} index={0}>
          <InvestmentsList investments={investments || []} />
        </TabPanel>

        <TabPanel value={currentTab} index={1}>
          <CapitalCallsDashboard clientId={clientId} />
        </TabPanel>

        <TabPanel value={currentTab} index={2}>
          <PerformanceAnalytics investments={investments || []} />
        </TabPanel>

        <TabPanel value={currentTab} index={3}>
          <DocumentUploadInterface clientId={clientId} />
        </TabPanel>
      </Card>
    </Box>
  );
};

// Investments List Component
const InvestmentsList: React.FC<{ investments: AlternativeInvestment[] }> = ({ investments }) => {
  return (
    <Grid container spacing={2}>
      {investments.map((inv) => (
        <Grid item xs={12} key={inv.id}>
          <Card variant="outlined">
            <CardContent>
              <Box display="flex" justifyContent="space-between">
                <Box>
                  <Typography variant="h6">{inv.fundName}</Typography>
                  <Typography variant="body2" color="textSecondary">
                    {inv.fundManager} • {inv.assetClass.replace(/_/g, ' ')} • Vintage {inv.vintageYear}
                  </Typography>
                </Box>
                <Box textAlign="right">
                  <Typography variant="h6" color="primary">
                    ${inv.currentNav.toLocaleString()}
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    Current NAV
                  </Typography>
                </Box>
              </Box>

              <Grid container spacing={2} sx={{ mt: 2 }}>
                <Grid item xs={3}>
                  <Typography variant="caption" color="textSecondary">Commitment</Typography>
                  <Typography variant="body2">${inv.commitmentAmount.toLocaleString()}</Typography>
                </Grid>
                <Grid item xs={3}>
                  <Typography variant="caption" color="textSecondary">Called</Typography>
                  <Typography variant="body2">${inv.capitalCalled.toLocaleString()}</Typography>
                </Grid>
                <Grid item xs={3}>
                  <Typography variant="caption" color="textSecondary">Distributed</Typography>
                  <Typography variant="body2">${inv.capitalDistributed.toLocaleString()}</Typography>
                </Grid>
                <Grid item xs={3}>
                  <Typography variant="caption" color="textSecondary">Unfunded</Typography>
                  <Typography variant="body2" color="warning.main">
                    ${inv.unfundedCommitment.toLocaleString()}
                  </Typography>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        </Grid>
      ))}
    </Grid>
  );
};

// Stub components for other tabs
const CapitalCallsDashboard: React.FC<{ clientId: string }> = ({ clientId }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>Capital Calls Dashboard</Typography>
      <Typography color="textSecondary">Coming soon: Capital call tracking and forecasting</Typography>
    </Box>
  );
};

const PerformanceAnalytics: React.FC<{ investments: AlternativeInvestment[] }> = ({ investments }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>Performance Analytics</Typography>
      <Typography color="textSecondary">Coming soon: IRR, TVPI, DPI, MOIC metrics and charts</Typography>
    </Box>
  );
};

const DocumentUploadInterface: React.FC<{ clientId: string }> = ({ clientId }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>Document Management</Typography>
      <Typography color="textSecondary">Coming soon: K-1, capital call, and statement uploads</Typography>
    </Box>
  );
};
