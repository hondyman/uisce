import React, { useState, useEffect } from 'react';
import {
  Grid,
  Card,
  CardContent,
  Typography,
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  Stack,
  Chip,
} from '@mui/material';
import {
  Download as DownloadIcon,
  Refresh as RefreshIcon,
  TrendingUp as TrendIcon,
  People as PeopleIcon,
  Sync as SyncIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell,
} from 'recharts';

const COLORS = ['#10b981', '#059669', '#047857', '#065f46', '#064e3b'];

const AnalyticsDashboard: React.FC = () => {
  const [timeRange, setTimeRange] = useState('30d');
  const [syncData, setSyncData] = useState([]);
  const [conflictData, setConflictData] = useState([]);
  const [cohortData, setCohortData] = useState([]);
  const [executiveMetrics, setExecutiveMetrics] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadAnalytics();
  }, [timeRange]);

  const loadAnalytics = async () => {
    try {
      setLoading(true);
      const [sync, conflict, cohort, executive] = await Promise.all([
        fetchAnalytics('sync'),
        fetchAnalytics('conflict'),
        fetchAnalytics('cohort'),
        fetchAnalytics('executive'),
      ]);
      setSyncData(sync);
      setConflictData(conflict);
      setCohortData(cohort);
      setExecutiveMetrics(executive);
    } catch (err) {
      console.error('Failed to load analytics:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchAnalytics = async (type: string) => {
    const response = await fetch(`/api/v1/analytics/${type}`, {
      headers: {
        'X-Hasura-Tenant-Id': localStorage.getItem('tenant_id') || '',
      },
    });
    const data = await response.json();
    return data.data || data.metrics || data.cohorts || [];
  };

  const handleExport = async (format: string) => {
    try {
      const response = await fetch('/api/v1/analytics/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Hasura-Tenant-Id': localStorage.getItem('tenant_id') || '',
        },
        body: JSON.stringify({
          format,
          report_type: 'comprehensive',
          start_date: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
          end_date: new Date().toISOString(),
          include_charts: true,
        }),
      });

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `analytics-export.${format}`;
      a.click();
    } catch (err) {
      console.error('Export failed:', err);
    }
  };

  if (loading) {
    return <div>Loading...</div>; // Replacing undefined LoadingSpinner
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">
          📊 Analytics Dashboard
        </Typography>
        <Stack direction="row" spacing={2}>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Time Range</InputLabel>
            <Select
              value={timeRange}
              label="Time Range"
              onChange={(e) => setTimeRange(e.target.value as string)}
            >
              <MenuItem value="7d">Last 7 Days</MenuItem>
              <MenuItem value="30d">Last 30 Days</MenuItem>
              <MenuItem value="90d">Last 90 Days</MenuItem>
              <MenuItem value="1y">Last Year</MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="outlined"
            startIcon={<DownloadIcon />}
            onClick={() => handleExport('csv')}
          >
            Export CSV
          </Button>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={loadAnalytics}
          >
            Refresh
          </Button>
        </Stack>
      </Box>

      {/* Executive Metrics */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        {executiveMetrics.map((metric: any, index: number) => (
          <Grid item xs={12} sm={6} md={3} key={index}>
            <Card>
              <CardContent>
                <Typography variant="body2" color="text.secondary">
                  {metric.metric}
                </Typography>
                <Typography variant="h4" sx={{ mt: 1 }}>
                  {metric.value}
                </Typography>
                {metric.trend && (
                  <Chip
                    label={`${metric.trend > 0 ? '+' : ''}${metric.trend}%`}
                    color={metric.trend > 0 ? 'success' : 'error'}
                    size="small"
                    sx={{ mt: 1 }}
                  />
                )}
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* Sync Performance Chart */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            <SyncIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
            Sync Performance ({timeRange})
          </Typography>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={syncData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis yAxisId="left" />
              <YAxis yAxisId="right" orientation="right" />
              <Tooltip />
              <Legend />
              <Area
                yAxisId="left"
                type="monotone"
                dataKey="total_syncs"
                stroke="#10b981"
                fill="#10b981"
                fillOpacity={0.3}
                name="Total Syncs"
              />
              <Area
                yAxisId="right"
                type="monotone"
                dataKey="success_rate"
                stroke="#059669"
                fill="#059669"
                fillOpacity={0.3}
                name="Success Rate %"
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Conflict Analytics */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                <WarningIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
                Conflict Resolution Trends
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={conflictData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="total_conflicts" fill="#f59e0b" name="Total Conflicts" />
                  <Bar dataKey="auto_resolved" fill="#10b981" name="Auto-Resolved" />
                  <Bar dataKey="user_overrides" fill="#ef4444" name="User Overrides" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Conflict Types
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={conflictData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {conflictData.map((entry: any, index: number) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* User Retention Cohort */}
      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            <PeopleIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
            User Retention Cohort Analysis
          </Typography>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={cohortData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="cohort_month" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="retention_rate" stroke="#10b981" name="Retention Rate %" />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </Box>
  );
};

export default AnalyticsDashboard;
