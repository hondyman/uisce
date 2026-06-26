import { useState, useEffect, useCallback, SyntheticEvent } from 'react';
import { devLog, devError } from '../utils/devLogger';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CardActions as _CardActions,
  Button,
  Tabs,
  Tab,
  Chip as _Chip,
  LinearProgress,
  Alert,
  IconButton as _IconButton,
  Dialog as _Dialog,
  DialogContent as _DialogContent,
  DialogActions as _DialogActions,
} from '@mui/material';
import {
  Campaign as CampaignIcon,
  Settings as SettingsIcon,
  Analytics as AnalyticsIcon,
  Notifications as NotificationsIcon,
  TrendingUp as TrendingUpIcon,
  People as PeopleIcon,
  Schedule as ScheduleIcon,
} from '@mui/icons-material';
import { useNotificationAPI, EngagementAnalytics } from '../hooks/useNotificationAPI';
import CampaignManager from './CampaignManager';
import { NotificationPreferences } from './NotificationPreferences';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`notification-tabpanel-${index}`}
      aria-labelledby={`notification-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

export default function NotificationDashboard() {
  const [tabValue, setTabValue] = useState(0);
  const [analytics, setAnalytics] = useState<EngagementAnalytics | null>(null);
  const [loading, setLoading] = useState(true);
  const [userId] = useState('current-user'); // TODO: Get from auth context

  const { getEngagementAnalytics } = useNotificationAPI();

  const loadAnalytics = useCallback(async () => {
    try {
      setLoading(true);
      const endDate = new Date().toISOString().split('T')[0];
      const startDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];

      const data = await getEngagementAnalytics(startDate, endDate);
      setAnalytics(data);
    } catch (error) {
      devError('Failed to load analytics:', error);
    } finally {
      setLoading(false);
    }
  }, [getEngagementAnalytics]);

  useEffect(() => {
    loadAnalytics();
  }, [loadAnalytics]);

  const handleTabChange = (_event: SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <LinearProgress />
        <Typography sx={{ mt: 2 }}>Loading notification dashboard...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%' }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="notification dashboard tabs">
          <Tab
            icon={<AnalyticsIcon />}
            label="Overview"
            id="notification-tab-0"
            aria-controls="notification-tabpanel-0"
          />
          <Tab
            icon={<CampaignIcon />}
            label="Campaigns"
            id="notification-tab-1"
            aria-controls="notification-tabpanel-1"
          />
          <Tab
            icon={<SettingsIcon />}
            label="Preferences"
            id="notification-tab-2"
            aria-controls="notification-tabpanel-2"
          />
        </Tabs>
      </Box>

      <TabPanel value={tabValue} index={0}>
        <NotificationOverview
          analytics={analytics}
          onRefresh={loadAnalytics}
          onTabChange={setTabValue}
        />
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        <CampaignManager />
      </TabPanel>

      <TabPanel value={tabValue} index={2}>
        <NotificationPreferences
          userId={userId}
          onPreferencesUpdated={() => {
            // Handle preferences update
            devLog('Preferences updated');
          }}
        />
      </TabPanel>
    </Box>
  );
}

interface NotificationOverviewProps {
  analytics: EngagementAnalytics | null;
  onRefresh: () => void;
  onTabChange: (tabIndex: number) => void;
}

function NotificationOverview({ analytics, onRefresh, onTabChange }: NotificationOverviewProps) {
  if (!analytics) {
    return (
      <Alert severity="info">
        No analytics data available. Start sending notifications to see engagement metrics.
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Notification Analytics Overview
        </Typography>
        <Button variant="outlined" onClick={onRefresh}>
          Refresh
        </Button>
      </Box>

      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Last 30 days performance metrics
      </Typography>

      <Grid container spacing={3}>
        {/* Key Metrics */}
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <NotificationsIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6" color="primary">
                  {analytics.total_sent.toLocaleString()}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">
                Total Sent
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <PeopleIcon color="success" sx={{ mr: 1 }} />
                <Typography variant="h6" color="success.main">
                  {analytics.total_opened.toLocaleString()}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">
                Total Opened
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <TrendingUpIcon color="info" sx={{ mr: 1 }} />
                <Typography variant="h6" color="info.main">
                  {analytics.total_clicked.toLocaleString()}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">
                Total Clicked
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <ScheduleIcon color="warning" sx={{ mr: 1 }} />
                <Typography variant="h6" color="warning.main">
                  {(analytics.avg_open_rate * 100).toFixed(1)}%
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary">
                Avg Open Rate
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Performance Metrics */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Performance Overview
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={12} sm={6} md={3}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="h4" color="success.main">
                    {(analytics.avg_open_rate * 100).toFixed(1)}%
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Average Open Rate
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} sm={6} md={3}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="h4" color="info.main">
                    {(analytics.avg_click_rate * 100).toFixed(1)}%
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Average Click Rate
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} sm={6} md={3}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="h4" color="primary.main">
                    {analytics.total_sent > 0 ? ((analytics.total_opened / analytics.total_sent) * 100).toFixed(1) : 0}%
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Overall Open Rate
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} sm={6} md={3}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="h4" color="secondary.main">
                    {analytics.total_opened > 0 ? ((analytics.total_clicked / analytics.total_opened) * 100).toFixed(1) : 0}%
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Click-to-Open Rate
                  </Typography>
                </Box>
              </Grid>
            </Grid>
          </Paper>
        </Grid>

        {/* Quick Actions */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Quick Actions
            </Typography>
            <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
              <Button
                variant="contained"
                startIcon={<CampaignIcon />}
                onClick={() => onTabChange(1)}
              >
                Create Campaign
              </Button>
              <Button
                variant="outlined"
                startIcon={<SettingsIcon />}
                onClick={() => onTabChange(2)}
              >
                Manage Preferences
              </Button>
              <Button
                variant="outlined"
                startIcon={<AnalyticsIcon />}
                onClick={onRefresh}
              >
                Refresh Analytics
              </Button>
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}
