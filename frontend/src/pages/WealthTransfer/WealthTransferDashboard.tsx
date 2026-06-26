import React, { useState } from 'react';
import {
  Box,
  Container,
  Tabs,
  Tab,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  Button,
  Chip,
  IconButton,
} from '@mui/material';
import {
  AccountTree as TreeIcon,
  Assessment as ChartIcon,
  CardGiftcard as GiftIcon,
  Gavel as TrustIcon,
  Calculate as CalculatorIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { FamilyTreeVisualization } from './FamilyTreeVisualization';
import { ScenarioComparisonTable } from './ScenarioComparisonTable';
import { GiftTrackingDashboard } from './GiftTrackingDashboard';
import { TrustManagementPanel } from './TrustManagementPanel';
import { TaxCalculator } from './TaxCalculator';
import { WealthTransferFlow } from './WealthTransferFlow';

interface WealthTransferDashboardProps {
  familyId: string;
}

export const WealthTransferDashboard: React.FC<WealthTransferDashboardProps> = ({ familyId }) => {
  const [currentTab, setCurrentTab] = useState(0);
  const [familyData, setFamilyData] = useState<any>(null);
  const [scenarios, setScenarios] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setCurrentTab(newValue);
  };

  const handleGeneratePlan = async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/wealth-transfer/families/${familyId}/generate-plan`, {
       method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          maxScenarios: 15,
          generateNarratives: true,
        }),
      });
      const data = await response.json();
      setScenarios(data.scenarios || []);
    } catch (error) {
      console.error('Failed to generate plan:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs>
            <Typography variant="h4" component="h1" gutterBottom>
              Wealth Transfer Planning
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Family ID: {familyId}
            </Typography>
          </Grid>
          <Grid item>
            <Button
              variant="contained"
              size="large"
              startIcon={loading ? <RefreshIcon className="spin" /> : <ChartIcon />}
              onClick={handleGeneratePlan}
              disabled={loading}
            >
              {loading ? 'Generating...' : 'Generate Estate Plan'}
            </Button>
          </Grid>
        </Grid>
      </Box>

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Total Net Worth
              </Typography>
              <Typography variant="h4">
                $25.0M
              </Typography>
              <Chip label="+12% YoY" color="success" size="small" sx={{ mt: 1 }} />
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Estimated Estate Tax
              </Typography>
              <Typography variant="h4" color="error">
                $10.0M
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Without planning
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Potential Tax Savings
              </Typography>
              <Typography variant="h4" color="success.main">
                $7.2M
              </Typography>
              <Typography variant="caption" color="text.secondary">
                With recommended plan
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Exemption Used
              </Typography>
              <Typography variant="h4">
                12%
              </Typography>
              <Typography variant="caption" color="text.secondary">
                $1.7M of $13.99M
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper elevation={1}>
        <Tabs
          value={currentTab}
          onChange={handleTabChange}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab icon={<TreeIcon />} label="Family Tree" />
          <Tab icon={<ChartIcon />} label="Scenarios" />
          <Tab icon={<ChartIcon />} label="Wealth Flow" />
          <Tab icon={<GiftIcon />} label="Gift Tracking" />
          <Tab icon={<TrustIcon />} label="Trusts & Entities" />
          <Tab icon={<CalculatorIcon />} label="Tax Calculator" />
        </Tabs>

        <Box sx={{ p: 3 }}>
          {/* Tab Panels */}
          {currentTab === 0 && <FamilyTreeVisualization familyId={familyId} />}
          {currentTab === 1 && <ScenarioComparisonTable scenarios={scenarios} familyId={familyId} />}
          {currentTab === 2 && <WealthTransferFlow familyId={familyId} />}
          {currentTab === 3 && <GiftTrackingDashboard familyId={familyId} />}
          {currentTab === 4 && <TrustManagementPanel familyId={familyId} />}
          {currentTab === 5 && <TaxCalculator familyId={familyId} />}
        </Box>
      </Paper>

      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        .spin {
          animation: spin 1s linear infinite;
        }
      `}</style>
    </Container>
  );
};
