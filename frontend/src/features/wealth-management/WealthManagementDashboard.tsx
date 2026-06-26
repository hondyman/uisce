import React, { useState } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  Chip,
  Button,
  Tabs,
  Tab,
  Paper,
  Alert,
  CircularProgress,
  // IconButton, Tooltip not used
} from '@mui/material';
import {
  TrendingUp as TrendingUpIcon,
  Assessment as AssessmentIcon,
  AccountBalance as AccountBalanceIcon,
  PieChart as PieChartIcon,
  Timeline as TimelineIcon,
  ShowChart as ShowChartIcon,
  Calculate as CalculateIcon,
  Refresh as RefreshIcon,
  Info as InfoIcon,
  CheckCircle as CheckCircleIcon,
  Schedule as _ScheduleIcon
} from '@mui/icons-material';
import { Divider } from '@mui/material';
import MetricDetailDialog from './components/MetricDetailDialog';
import { wealthManagementService, WealthManagementMetric } from '../../services/wealthManagementService';
import { devError } from '../../utils/devLogger';

interface WealthManagementDashboardProps {
  tenantId: string;
  clientId?: string;
}

const WealthManagementDashboard: React.FC<WealthManagementDashboardProps> = ({
  tenantId,
  clientId: _clientId
}) => {
  const [activeTab, setActiveTab] = useState(0);
  const [metrics, setMetrics] = useState<WealthManagementMetric[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [selectedMetric, setSelectedMetric] = useState<WealthManagementMetric | null>(null);
  const [detailDialogOpen, setDetailDialogOpen] = useState(false);

    const handleRefresh = () => {
    // Trigger refresh of metrics data
    const loadMetrics = async () => {
      try {
        setLoading(true);
        const data = await wealthManagementService.getMetrics(tenantId);
        setMetrics(data);
        setLastRefresh(new Date());
        setError(null);
      } catch (err) {
        setError('Failed to load wealth management metrics');
        devError('Error loading metrics:', err);
      } finally {
        setLoading(false);
      }
    };
    loadMetrics();
  };

  const handleViewDetails = (metric: WealthManagementMetric) => {
    setSelectedMetric(metric);
    setDetailDialogOpen(true);
  };

  const handleCloseDetailDialog = () => {
    setDetailDialogOpen(false);
    setSelectedMetric(null);
  };

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'performance':
        return <TrendingUpIcon color="primary" />;
      case 'risk':
      case 'risk_adjusted_performance':
        return <AssessmentIcon color="warning" />;
      case 'composition':
        return <PieChartIcon color="success" />;
      case 'income':
        return <AccountBalanceIcon color="success" />;
      case 'client_kpi':
        return <TimelineIcon color="info" />;
      case 'business_efficiency':
        return <ShowChartIcon color="secondary" />;
      default:
        return <CalculateIcon />;
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'performance':
        return 'primary';
      case 'risk':
      case 'risk_adjusted_performance':
        return 'warning';
      case 'composition':
      case 'income':
        return 'success';
      case 'client_kpi':
        return 'info';
      case 'business_efficiency':
        return 'secondary';
      default:
        return 'default';
    }
  };

  const getGovernanceIcon = (status: string) => {
    return status === 'golden' ?
      <CheckCircleIcon color="success" fontSize="small" /> :
      <InfoIcon color="warning" fontSize="small" />;
  };

  const filteredMetrics = metrics.filter(metric => {
    if (activeTab === 0) return true; // All metrics
    if (activeTab === 1) return metric.governance_status === 'golden';
    if (activeTab === 2) return metric.category === 'performance' || metric.category === 'risk_adjusted_performance';
    if (activeTab === 3) return metric.category === 'client_kpi' || metric.category === 'business_efficiency';
    return false;
  });

  const categoryCounts = metrics.reduce((acc, metric) => {
    acc[metric.category] = (acc[metric.category] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
        <Typography variant="h6" sx={{ ml: 2 }}>
          Loading Wealth Management Analytics...
        </Typography>
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
    <Box sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Wealth Management Analytics
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Advanced portfolio insights and business intelligence for wealth management
          </Typography>
          {lastRefresh && (
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
              Last updated: {lastRefresh.toLocaleString()}
            </Typography>
          )}
        </Box>
        <Button
          variant="outlined"
          startIcon={<RefreshIcon />}
          onClick={handleRefresh}
          size="small"
        >
          Refresh Data
        </Button>
      </Box>

      {/* Summary Cards */}
      <Grid container spacing={3} mb={3}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Total Metrics
              </Typography>
              <Typography variant="h4">
                {metrics.length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Golden Status
              </Typography>
              <Typography variant="h4" color="success.main">
                {metrics.filter(m => m.governance_status === 'golden').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Categories
              </Typography>
              <Typography variant="h4">
                {Object.keys(categoryCounts).length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Auto-Refresh Enabled
              </Typography>
              <Typography variant="h4" color="primary.main">
                {metrics.filter(m => m.governance_status === 'golden').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          variant="fullWidth"
        >
          <Tab label={`All Metrics (${metrics.length})`} />
          <Tab label={`Golden (${metrics.filter(m => m.governance_status === 'golden').length})`} />
          <Tab label={`Performance & Risk (${metrics.filter(m => ['performance', 'risk', 'risk_adjusted_performance'].includes(m.category)).length})`} />
          <Tab label={`Business & Client (${metrics.filter(m => ['client_kpi', 'business_efficiency'].includes(m.category)).length})`} />
        </Tabs>
      </Paper>

      {/* Metrics List */}
      <Grid container spacing={3}>
        {filteredMetrics.map((metric) => (
          <Grid item xs={12} md={6} lg={4} key={metric.node_id}>
            <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
              <CardContent sx={{ flexGrow: 1 }}>
                <Box display="flex" alignItems="center" mb={2}>
                  {getCategoryIcon(metric.category)}
                  <Typography variant="h6" sx={{ ml: 1, flexGrow: 1 }}>
                    {metric.node_id.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                  </Typography>
                  <Box display="flex" alignItems="center">
                    {getGovernanceIcon(metric.governance_status)}
                    <Chip
                      label={metric.governance_status}
                      size="small"
                      color={metric.governance_status === 'golden' ? 'success' : 'warning'}
                      sx={{ ml: 1 }}
                    />
                  </Box>
                </Box>

                <Typography variant="body2" color="text.secondary" paragraph>
                  {metric.description}
                </Typography>

                <Box mb={2}>
                  <Chip
                    label={metric.category.replace(/_/g, ' ')}
                    size="small"
                    color={getCategoryColor(metric.category) as any}
                    sx={{ mr: 1, mb: 1 }}
                  />
                  <Chip
                    label={metric.formula_type}
                    size="small"
                    variant="outlined"
                    sx={{ mr: 1, mb: 1 }}
                  />
                </Box>

                <Box>
                  <Typography variant="caption" color="text.secondary">
                    Audience:
                  </Typography>
                  <Box sx={{ mt: 0.5 }}>
                    {metric.audience.map((audience) => (
                      <Chip
                        key={audience}
                        label={audience}
                        size="small"
                        variant="outlined"
                        sx={{ mr: 0.5, mb: 0.5 }}
                      />
                    ))}
                  </Box>
                </Box>
              </CardContent>

              <Divider />

              <Box p={2}>
                <Button
                  fullWidth
                  variant="outlined"
                  startIcon={<ShowChartIcon />}
                  onClick={() => handleViewDetails(metric)}
                  size="small"
                >
                  View Details
                </Button>
              </Box>
            </Card>
          </Grid>
        ))}
      </Grid>

      {filteredMetrics.length === 0 && (
        <Box textAlign="center" py={6}>
          <Typography variant="h6" color="text.secondary">
            No metrics found for the selected filter.
          </Typography>
        </Box>
      )}

      {/* Metric Detail Dialog */}
      <MetricDetailDialog
        open={detailDialogOpen}
        onClose={handleCloseDetailDialog}
        metric={selectedMetric}
      />
    </Box>
  );
};

export default WealthManagementDashboard;
