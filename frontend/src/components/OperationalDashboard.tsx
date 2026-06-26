/**
 * Operational Dashboard Component
 * 
 * Material-UI implementation of the enterprise risk platform dashboard.
 * Features:
 * - Responsive grid layout (mobile-first)
 * - Dark mode support
 * - KPI cards with metrics
 * - Sparkline trends
 * - ETL health monitoring
 * - Critical alerts feed
 * - Top navigation with search and notifications
 * - Left sidebar navigation
 */

import React, { useEffect, useState } from 'react';
import {
  AppBar,
  Toolbar,
  Drawer,
  Box,
  Container,
  Grid,
  Card,
  CardContent,
  CardHeader,
  TextField,
  InputAdornment,
  IconButton,
  Avatar,
  Badge,
  Chip,
  Alert,
  AlertTitle,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  Paper,
  List,
  ListItem,
  ListItemText,
  Divider,
  Typography,
  LinearProgress,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Search as SearchIcon,
  Menu as MenuIcon,
  Notifications as NotificationsIcon,
  Settings as SettingsIcon,
  Dashboard as DashboardIcon,
  CheckCircle as CheckCircleIcon,
  Info as InfoIcon,
  Home,
  BarChart as BarChart3Icon,
  TrendingUp as TrendingUpIcon,
  ReportProblem as AlertTriangleIcon,
  ChevronRight as ChevronRightIcon,
} from '@mui/icons-material';
import { LineChart, Line, ResponsiveContainer, XAxis, YAxis } from 'recharts';

// Type definitions
interface KPICard {
  id: string;
  label: string;
  value: string | number;
  unit?: string;
  change?: number;
  status?: 'good' | 'warning' | 'critical';
}

interface SparklineData {
  id: string;
  label: string;
  data: Array<{ name: string; value: number }>;
  unit?: string;
  color?: string;
}

interface AlertItem {
  id: string;
  severity: 'info' | 'warning' | 'error' | 'success';
  title: string;
  message: string;
  timestamp: string;
}

interface ETLStatus {
  name: string;
  status: 'active' | 'idle' | 'error';
  lastRun?: string;
  duration?: string;
  recordsProcessed?: number;
}

// Sparkline component with mini chart
const SparklineCard: React.FC<{ data: SparklineData }> = ({ data }) => {
  return (
    <Card>
      <CardContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="subtitle2" color="textSecondary">
              {data.label}
            </Typography>
            {data.unit && (
              <Typography variant="caption" sx={{ color: 'primary.main', fontWeight: 600 }}>
                {data.unit}
              </Typography>
            )}
          </Box>
          <Box sx={{ height: 60 }}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data.data}>
                <XAxis dataKey="name" height={0} />
                <YAxis hide domain={['auto', 'auto']} />
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke={data.color || '#137fec'}
                  dot={false}
                  isAnimationActive={false}
                  strokeWidth={2}
                />
              </LineChart>
            </ResponsiveContainer>
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
};

// KPI card with trend indicator
const KPICardComponent: React.FC<{ kpi: KPICard }> = ({ kpi }) => {
  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'good':
        return '#66bb6a';
      case 'warning':
        return '#ffa726';
      case 'critical':
        return '#f44336';
      default:
        return '#137fec';
    }
  };

  return (
    <Card>
      <CardContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
            <Box>
              <Typography variant="subtitle2" color="textSecondary">
                {kpi.label}
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 0.5, mt: 1 }}>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  {kpi.value}
                </Typography>
                {kpi.unit && (
                  <Typography variant="body2" color="textSecondary">
                    {kpi.unit}
                  </Typography>
                )}
              </Box>
            </Box>
            {kpi.change !== undefined && (
              <Chip
                label={`${kpi.change >= 0 ? '+' : ''}${kpi.change}%`}
                size="small"
                color={kpi.change >= 0 ? 'success' : 'error'}
                variant="outlined"
              />
            )}
          </Box>
          {kpi.status && (
            <Box
              sx={{
                height: 4,
                backgroundColor: getStatusColor(kpi.status),
                borderRadius: 2,
                mt: 1,
              }}
            />
          )}
        </Box>
      </CardContent>
    </Card>
  );
};

// ETL Health monitoring table
const ETLHealthSection: React.FC<{ etlStatus: ETLStatus[] }> = ({ etlStatus }) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'success';
      case 'error':
        return 'error';
      default:
        return 'default';
    }
  };

  return (
    <Card>
      <CardHeader title="ETL Health" subheader="Real-time pipeline status" />
      <TableContainer>
        <Table>
          <TableBody>
            {etlStatus.map((item) => (
              <TableRow key={item.name}>
                <TableCell>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Chip
                      label={item.status}
                      size="small"
                      color={getStatusColor(item.status)}
                      variant="filled"
                    />
                    <Typography variant="body2">{item.name}</Typography>
                  </Box>
                </TableCell>
                <TableCell align="right">
                  {item.lastRun && (
                    <Typography variant="caption" color="textSecondary">
                      Last: {item.lastRun}
                    </Typography>
                  )}
                </TableCell>
                <TableCell align="right">
                  {item.duration && (
                    <Typography variant="caption" color="textSecondary">
                      {item.duration}
                    </Typography>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Card>
  );
};

// The main dashboard component
interface OperationalDashboardProps {
  kpiData?: KPICard[];
  sparklineData?: SparklineData[];
  etlStatus?: ETLStatus[];
  alerts?: AlertItem[];
}

export const OperationalDashboard: React.FC<OperationalDashboardProps> = ({
  kpiData = [
    {
      id: '1',
      label: 'Compliance Score',
      value: '94.2',
      unit: '%',
      change: 2.3,
      status: 'good',
    },
    {
      id: '2',
      label: 'Risk Profile',
      value: 'Medium',
      change: -1.2,
      status: 'warning',
    },
  ],
  sparklineData = [
    {
      id: '1',
      label: 'Pass Rate',
      unit: '98.5%',
      data: [
        { name: '1', value: 95 },
        { name: '2', value: 97 },
        { name: '3', value: 98.5 },
      ],
      color: '#66bb6a',
    },
    {
      id: '2',
      label: 'Hard Breaches',
      unit: '3',
      data: [
        { name: '1', value: 5 },
        { name: '2', value: 4 },
        { name: '3', value: 3 },
      ],
      color: '#f44336',
    },
    {
      id: '3',
      label: 'Volatility',
      unit: '12.4%',
      data: [
        { name: '1', value: 15 },
        { name: '2', value: 13 },
        { name: '3', value: 12.4 },
      ],
      color: '#ffa726',
    },
    {
      id: '4',
      label: 'ETL Latency',
      unit: '245ms',
      data: [
        { name: '1', value: 300 },
        { name: '2', value: 270 },
        { name: '3', value: 245 },
      ],
      color: '#137fec',
    },
  ],
  etlStatus = [
    { name: 'Fund Data Pipeline', status: 'active', lastRun: '2 min ago', duration: '4s' },
    { name: 'Compliance Validator', status: 'active', lastRun: '5 min ago', duration: '8s' },
    { name: 'Risk Calculator', status: 'idle', lastRun: '15 min ago', duration: '12s' },
  ],
  alerts = [
    {
      id: '1',
      severity: 'warning',
      title: 'High Volatility Detected',
      message: 'Crypto asset volatility exceeded threshold',
      timestamp: '10 min ago',
    },
    {
      id: '2',
      severity: 'error',
      title: 'Compliance Breach Alert',
      message: 'Fund position violates sector allocation limits',
      timestamp: '25 min ago',
    },
    {
      id: '3',
      severity: 'info',
      title: 'Risk Report Generated',
      message: 'Daily risk report is ready for review',
      timestamp: '1 hour ago',
    },
  ],
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const isTablet = useMediaQuery(theme.breakpoints.down('lg'));
  const [mobileOpen, setMobileOpen] = React.useState(false);
  const [searchQuery, setSearchQuery] = React.useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  // Sidebar navigation items
  const navigationItems = [
    { label: 'Dashboard', icon: DashboardIcon },
    { label: 'Compliance', icon: CheckCircleIcon },
    { label: 'Risk Assessment', icon: AlertTriangleIcon },
    { label: 'Operations', icon: BarChart3Icon },
    { label: 'Logs', icon: InfoIcon },
  ];

  // Sidebar drawer content
  const drawerContent = (
    <Box sx={{ width: 280, pt: 2 }}>
      <Box sx={{ px: 2, mb: 3 }}>
        <Typography variant="h6" sx={{ fontWeight: 700, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Home sx={{ fontSize: 20 }} />
          SemLayer
        </Typography>
      </Box>

      <Divider />

      <List>
        {navigationItems.map((item) => {
          const IconComponent = item.icon;
          return (
            <ListItem
              key={item.label}
              sx={{
                px: 2,
                py: 1.5,
                '&:hover': { backgroundColor: 'action.hover' },
                cursor: 'pointer',
              }}
            >
              <IconComponent sx={{ mr: 2, fontSize: 20 }} />
              <ListItemText
                primary={item.label}
                primaryTypographyProps={{ variant: 'body2' }}
              />
              <ChevronRightIcon sx={{ fontSize: 18, color: 'action.disabled' }} />
            </ListItem>
          );
        })}
      </List>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', backgroundColor: 'background.default' }}>
      {/* Top Navigation Bar */}
      <AppBar
        position="fixed"
        sx={{
          zIndex: (theme) => theme.zIndex.drawer + 1,
          backgroundColor: 'background.paper',
          color: 'text.primary',
          boxShadow: 1,
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            edge="start"
            onClick={handleDrawerToggle}
            sx={{ mr: 2, display: { lg: 'none' } }}
          >
            <MenuIcon />
          </IconButton>

          <Typography variant="h6" sx={{ flexGrow: 1, fontWeight: 600 }}>
            Risk & Compliance Console
          </Typography>

          {/* Search Bar */}
          <TextField
            placeholder="Search..."
            size="small"
            variant="outlined"
            sx={{
              width: { xs: 200, md: 300 },
              mr: 3,
              '& .MuiOutlinedInput-root': {
                backgroundColor: 'action.hover',
              },
            }}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon sx={{ color: 'action.active' }} />
                </InputAdornment>
              ),
            }}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />

          {/* Right side icons */}
          <IconButton size="small" sx={{ mr: 1 }}>
            <Badge badgeContent={3} color="error">
              <NotificationsIcon />
            </Badge>
          </IconButton>

          <IconButton size="small" sx={{ mr: 2 }}>
            <SettingsIcon />
          </IconButton>

          <Avatar sx={{ width: 32, height: 32, backgroundColor: 'primary.main' }}>
            JD
          </Avatar>
        </Toolbar>
      </AppBar>

      {/* Sidebar - Permanent on desktop, Mobile drawer on mobile */}
      <Drawer
        variant={isMobile ? 'temporary' : 'permanent'}
        open={isMobile ? mobileOpen : true}
        onClose={handleDrawerToggle}
        sx={{
          width: 280,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: 280,
            backgroundColor: 'background.paper',
            borderRight: 1,
            borderColor: 'divider',
            mt: 8,
          },
        }}
      >
        {drawerContent}
      </Drawer>

      {/* Main Content Area */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: { xs: 2, sm: 3 },
          mt: 8,
          backgroundColor: 'background.default',
          minHeight: 'calc(100vh - 64px)',
        }}
      >
        <Container maxWidth="xl">
          {/* Breadcrumbs and Header */}
          <Box sx={{ mb: 4 }}>
            <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
              Dashboard / Compliance / Risk Profile
            </Typography>
            <Typography variant="h4" sx={{ fontWeight: 700 }}>
              Operational Overview
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Real-time compliance and risk monitoring dashboard
            </Typography>
          </Box>

          {/* KPI Cards Section - 2 columns */}
          <Grid container spacing={3} sx={{ mb: 4 }}>
            {kpiData.map((kpi) => (
              <Grid item xs={12} sm={6} lg={6} key={kpi.id}>
                <KPICardComponent kpi={kpi} />
              </Grid>
            ))}
          </Grid>

          {/* Sparkline Trends Section - 4 columns, responsive */}
          <Grid container spacing={3} sx={{ mb: 4 }}>
            {sparklineData.map((spark) => (
              <Grid
                item
                xs={12}
                sm={6}
                md={6}
                lg={3}
                key={spark.id}
              >
                <SparklineCard data={spark} />
              </Grid>
            ))}
          </Grid>

          {/* Operations and Alerts Section - 2 columns */}
          <Grid container spacing={3}>
            {/* ETL Health */}
            <Grid item xs={12} lg={6}>
              <ETLHealthSection etlStatus={etlStatus} />
            </Grid>

            {/* Critical Alerts */}
            <Grid item xs={12} lg={6}>
              <Card>
                <CardHeader
                  title="Critical Alerts"
                  subheader="Recent compliance and risk events"
                />
                <CardContent sx={{ p: 0 }}>
                  <List disablePadding>
                    {alerts.map((alert, index) => (
                      <React.Fragment key={alert.id}>
                        <ListItem sx={{ px: 2, py: 1.5 }}>
                          <Alert
                            severity={alert.severity}
                            sx={{
                              width: '100%',
                              '& .MuiAlert-icon': {
                                mr: 1,
                              },
                            }}
                          >
                            <AlertTitle sx={{ fontWeight: 600, mb: 0.5 }}>
                              {alert.title}
                            </AlertTitle>
                            {alert.message}
                            <Typography variant="caption" sx={{ display: 'block', mt: 0.5 }}>
                              {alert.timestamp}
                            </Typography>
                          </Alert>
                        </ListItem>
                        {index < alerts.length - 1 && <Divider />}
                      </React.Fragment>
                    ))}
                  </List>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        </Container>
      </Box>
    </Box>
  );
};

export default OperationalDashboard;
