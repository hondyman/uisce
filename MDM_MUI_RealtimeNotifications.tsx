import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Card,
  CardContent,
  CardHeader,
  Grid,
  Button,
  AppBar,
  Toolbar,
  Typography,
  Avatar,
  IconButton,
  Tabs,
  Tab,
  Badge,
  Chip,
  Stack,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Divider,
  Switch,
  FormControlLabel,
  Dialog,
  DialogTitle,
  DialogContent,
  Tooltip,
  LinearProgress,
  CssBaseline,
} from '@mui/material';
import {
  Notifications as NotificationsIcon,
  Speed as SpeedIcon,
  Cloud as CloudIcon,
  Warning as WarningIcon,
  Info as InfoIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  PlayArrow as PlayArrowIcon,
  Pause as PauseIcon,
  Refresh as RefreshIcon,
  Settings as SettingsIcon,
  Close as CloseIcon,
  MoreVert as MoreVertIcon,
  TrendingUp as TrendingUpIcon,
  EventNote as ActivityIcon,
  Radio as RadioIcon,
  Bolt as BoltIcon,
  Bolt as ZapIcon,
  DoneAll as DoneAllIcon,
} from '@mui/icons-material';
import { createTheme, ThemeProvider } from '@mui/material/styles';

const theme = createTheme({
  palette: {
    primary: {
      main: '#137fec',
    },
    background: {
      default: '#f6f7f8',
    },
    success: {
      main: '#10b981',
    },
    warning: {
      main: '#f59e0b',
    },
    error: {
      main: '#ef4444',
    },
    info: {
      main: '#3b82f6',
    },
  },
  typography: {
    fontFamily: '"Inter", sans-serif',
  },
});

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ pt: 2 }}>{children}</Box>}
    </div>
  );
}

interface StreamEvent {
  id: string;
  type: 'success' | 'warning' | 'error' | 'info';
  title: string;
  message: string;
  timestamp: string;
  progress?: number;
  source?: string;
}

export const RealtimeNotificationsDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [isStreaming, setIsStreaming] = useState(true);
  const [streamEvents, setStreamEvents] = useState<StreamEvent[]>([
    {
      id: '1',
      type: 'success',
      title: 'Export Job Completed',
      message: 'Export-2024-01-28-CUST-001 finished processing 2.3M records in 14.2s',
      timestamp: 'Just now',
      source: 'Export Service',
    },
    {
      id: '2',
      type: 'info',
      title: 'Scheduler Triggered',
      message: 'RULE-USR-092 scheduled job started processing rules for Wealth Management',
      timestamp: '2 seconds ago',
      progress: 45,
      source: 'Scheduler',
    },
    {
      id: '3',
      type: 'warning',
      title: 'High Latency Detected',
      message: 'Database query took 8.3s (threshold: 5s) for rule evaluation',
      timestamp: '8 seconds ago',
      source: 'Rules Engine',
    },
  ]);
  const [notificationSettings, setNotificationSettings] = useState({
    enableSSE: true,
    enableWebSocket: true,
    enableEmail: false,
    enableSlack: true,
  });
  const [selectedNotification, setSelectedNotification] = useState<StreamEvent | null>(null);
  const [unreadCount, setUnreadCount] = useState(3);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  // Simulate streaming events
  useEffect(() => {
    if (!isStreaming) return;

    const interval = setInterval(() => {
      const newEvent: StreamEvent = {
        id: Date.now().toString(),
        type: ['success', 'info', 'warning'][Math.floor(Math.random() * 3)] as any,
        title: ['Rule Validated', 'Export Started', 'Conflict Detected'][Math.floor(Math.random() * 3)],
        message: `New event generated at ${new Date().toLocaleTimeString()}`,
        timestamp: 'Just now',
        source: ['Rules Engine', 'Export Service', 'Scheduler'][Math.floor(Math.random() * 3)],
      };
      setStreamEvents((prev) => [newEvent, ...prev.slice(0, 9)]);
    }, 5000);

    return () => clearInterval(interval);
  }, [isStreaming]);

  const getIconForType = (type: StreamEvent['type']) => {
    switch (type) {
      case 'success':
        return <CheckCircleIcon sx={{ color: '#10b981' }} />;
      case 'error':
        return <ErrorIcon sx={{ color: '#ef4444' }} />;
      case 'warning':
        return <WarningIcon sx={{ color: '#f59e0b' }} />;
      default:
        return <InfoIcon sx={{ color: '#3b82f6' }} />;
    }
  };

  const getColorForType = (type: StreamEvent['type']) => {
    switch (type) {
      case 'success':
        return '#10b981';
      case 'error':
        return '#ef4444';
      case 'warning':
        return '#f59e0b';
      default:
        return '#3b82f6';
    }
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: '#f6f7f8' }}>
        {/* Header */}
        <AppBar position="static" elevation={1} sx={{ bgcolor: '#ffffff', color: '#1f2937' }}>
          <Toolbar>
            <Stack direction="row" alignItems="center" spacing={2} sx={{ flexGrow: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#137fec' }}>
                <Badge badgeContent={unreadCount} color="error">
                  <NotificationsIcon sx={{ fontSize: '2rem' }} />
                </Badge>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  Real-time Operations Monitor
                </Typography>
              </Box>

              <Stack direction="row" spacing={3} sx={{ display: { xs: 'none', md: 'flex' }, ml: 4 }}>
                <Button color="inherit">Dashboard</Button>
                <Button color="inherit" sx={{ color: '#137fec', fontWeight: 700 }}>
                  Live Events
                </Button>
                <Button color="inherit">Alerts</Button>
                <Button color="inherit">Analytics</Button>
              </Stack>
            </Stack>

            <Stack direction="row" spacing={2} alignItems="center">
              <Stack direction="row" alignItems="center" spacing={0.5}>
                <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: '#10b981', animation: 'pulse 2s infinite' }} />
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#6b7280' }}>
                  Live
                </Typography>
              </Stack>
              <IconButton color="inherit">
                <SettingsIcon />
              </IconButton>
              <Avatar sx={{ width: 32, height: 32, bgcolor: '#137fec' }} />
            </Stack>
          </Toolbar>
        </AppBar>

        {/* Main Content */}
        <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
          {/* Sidebar - Event Sources */}
          <Paper
            elevation={0}
            sx={{
              width: 240,
              borderRight: '1px solid #e5e7eb',
              borderRadius: 0,
              display: 'flex',
              flexDirection: 'column',
              bgcolor: '#ffffff',
              overflow: 'auto',
            }}
          >
            <List sx={{ p: 1 }}>
              {[
                { label: 'All Events', count: 287, icon: <ActivityIcon /> },
                { label: 'Export Service', count: 45, icon: <ZapIcon /> },
                { label: 'Scheduler', count: 89, icon: <RadioIcon /> },
                { label: 'Rules Engine', count: 112, icon: <BoltIcon /> },
                { label: 'Conflicts', count: 41, icon: <WarningIcon /> },
              ].map((item, idx) => (
                <ListItemButton
                  key={idx}
                  selected={idx === 0}
                  sx={{
                    mb: 0.5,
                    borderRadius: 1,
                    '&.Mui-selected': {
                      bgcolor: '#eff6ff',
                      color: '#137fec',
                      '& .MuiListItemIcon-root': { color: '#137fec' },
                    },
                  }}
                >
                  <ListItemIcon sx={{ minWidth: 36 }}>{item.icon}</ListItemIcon>
                  <ListItemText
                    primary={item.label}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                  />
                  <Chip label={item.count} size="small" sx={{ ml: 1, bgcolor: '#f3f4f6' }} />
                </ListItemButton>
              ))}
            </List>

            <Divider sx={{ my: 2 }} />

            <Box sx={{ p: 2 }}>
              <Typography
                variant="caption"
                sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280', mb: 2, display: 'block' }}
              >
                Stream Settings
              </Typography>
              <Stack spacing={1}>
                <FormControlLabel
                  control={<Switch checked={isStreaming} onChange={(e) => setIsStreaming(e.target.checked)} />}
                  label={
                    <Typography variant="body2" sx={{ fontWeight: 600 }}>
                      {isStreaming ? 'Streaming' : 'Paused'}
                    </Typography>
                  }
                />
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<RefreshIcon />}
                  fullWidth
                  sx={{ textTransform: 'none' }}
                >
                  Clear All
                </Button>
              </Stack>
            </Box>
          </Paper>

          {/* Main Content Area */}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Header Section */}
            <Paper
              elevation={0}
              sx={{
                p: 3,
                borderBottom: '1px solid #e5e7eb',
                bgcolor: '#ffffff',
              }}
            >
              <Stack direction="row" justifyContent="space-between" alignItems="start" sx={{ mb: 2 }}>
                <Box>
                  <Typography variant="caption" sx={{ color: '#6b7280' }}>
                    Operations / Real-time Events
                  </Typography>
                  <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                    Live Event Dashboard
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#6b7280', mt: 0.5 }}>
                    Streaming updates from export, scheduler, and rules engine services
                  </Typography>
                </Box>

                <Stack direction="row" spacing={2}>
                  <Button
                    variant={isStreaming ? 'contained' : 'outlined'}
                    startIcon={isStreaming ? <PauseIcon /> : <PlayArrowIcon />}
                    onClick={() => setIsStreaming(!isStreaming)}
                    size="small"
                  >
                    {isStreaming ? 'Streaming' : 'Paused'}
                  </Button>
                </Stack>
              </Stack>

              {/* Tabs */}
              <Tabs value={activeTab} onChange={handleTabChange}>
                <Tab
                  icon={<NotificationsIcon sx={{ mr: 1 }} />}
                  label="Event Stream"
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                />
                <Tab
                  icon={<SpeedIcon sx={{ mr: 1 }} />}
                  label="Performance Metrics"
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                />
                <Tab
                  icon={<SettingsIcon sx={{ mr: 1 }} />}
                  label="Notification Settings"
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                />
              </Tabs>
            </Paper>

            {/* Tab Content */}
            <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
              {/* Event Stream Tab */}
              <TabPanel value={activeTab} index={0}>
                <Stack spacing={1}>
                  {streamEvents.map((event, idx) => (
                    <Card
                      key={event.id}
                      variant="outlined"
                      sx={{
                        cursor: 'pointer',
                        transition: 'all 0.2s',
                        borderLeft: `4px solid ${getColorForType(event.type)}`,
                        borderRadius: 1,
                        '&:hover': {
                          boxShadow: 3,
                          transform: 'translateX(4px)',
                        },
                      }}
                      onClick={() => setSelectedNotification(event)}
                    >
                      <Box sx={{ p: 2, display: 'flex', gap: 2 }}>
                        <Box sx={{ mt: 0.5 }}>{getIconForType(event.type)}</Box>
                        <Box sx={{ flex: 1 }}>
                          <Stack direction="row" justifyContent="space-between" alignItems="start" sx={{ mb: 0.5 }}>
                            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                              {event.title}
                            </Typography>
                            <Typography variant="caption" sx={{ color: '#6b7280' }}>
                              {event.timestamp}
                            </Typography>
                          </Stack>
                          <Typography variant="body2" sx={{ color: '#6b7280', mb: 1 }}>
                            {event.message}
                          </Typography>
                          {event.progress !== undefined && (
                            <Box>
                              <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                  Processing
                                </Typography>
                                <Typography variant="caption" sx={{ fontWeight: 600, color: '#137fec' }}>
                                  {event.progress}%
                                </Typography>
                              </Stack>
                              <LinearProgress
                                variant="determinate"
                                value={event.progress}
                                sx={{ height: 6, borderRadius: 1 }}
                              />
                            </Box>
                          )}
                          <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                            <Chip
                              label={event.source}
                              size="small"
                              variant="outlined"
                              icon={<ZapIcon />}
                            />
                            {idx === 0 && <Chip label="NEW" size="small" color="primary" />}
                          </Stack>
                        </Box>
                        <IconButton size="small">
                          <MoreVertIcon fontSize="small" />
                        </IconButton>
                      </Box>
                    </Card>
                  ))}
                </Stack>
              </TabPanel>

              {/* Performance Metrics Tab */}
              <TabPanel value={activeTab} index={1}>
                <Grid container spacing={3}>
                  {/* Real-time Stats */}
                  <Grid item xs={12} sm={6} md={3}>
                    <Card>
                      <CardContent>
                        <Typography
                          variant="caption"
                          sx={{
                            fontWeight: 700,
                            textTransform: 'uppercase',
                            color: '#6b7280',
                          }}
                        >
                          Events/Minute
                        </Typography>
                        <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                          2.4K
                        </Typography>
                        <Box sx={{ mt: 2, height: 30, display: 'flex', alignItems: 'flex-end', gap: 0.5 }}>
                          {Array.from({ length: 12 }).map((_, i) => (
                            <Box
                              key={i}
                              sx={{
                                flex: 1,
                                height: `${40 + Math.random() * 60}%`,
                                bgcolor: '#137fec',
                                borderRadius: '2px 2px 0 0',
                              }}
                            />
                          ))}
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>

                  <Grid item xs={12} sm={6} md={3}>
                    <Card>
                      <CardContent>
                        <Typography
                          variant="caption"
                          sx={{
                            fontWeight: 700,
                            textTransform: 'uppercase',
                            color: '#6b7280',
                          }}
                        >
                          Avg Latency
                        </Typography>
                        <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                          142ms
                        </Typography>
                        <Typography
                          variant="caption"
                          sx={{ color: '#10b981', fontWeight: 700, mt: 1, display: 'block' }}
                        >
                          ↓ 23% vs last hour
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>

                  <Grid item xs={12} sm={6} md={3}>
                    <Card>
                      <CardContent>
                        <Typography
                          variant="caption"
                          sx={{
                            fontWeight: 700,
                            textTransform: 'uppercase',
                            color: '#6b7280',
                          }}
                        >
                          Success Rate
                        </Typography>
                        <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                          99.8%
                        </Typography>
                        <Typography
                          variant="caption"
                          sx={{ color: '#10b981', fontWeight: 700, mt: 1, display: 'block' }}
                        >
                          ↑ 2 errors in 2h
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>

                  <Grid item xs={12} sm={6} md={3}>
                    <Card>
                      <CardContent>
                        <Typography
                          variant="caption"
                          sx={{
                            fontWeight: 700,
                            textTransform: 'uppercase',
                            color: '#6b7280',
                          }}
                        >
                          Active Connections
                        </Typography>
                        <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                          847
                        </Typography>
                        <Typography
                          variant="caption"
                          sx={{ color: '#6b7280', fontWeight: 700, mt: 1, display: 'block' }}
                        >
                          WebSocket + SSE
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>

                  {/* Throughput Chart */}
                  <Grid item xs={12}>
                    <Card>
                      <CardContent>
                        <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
                          Service Throughput Over Time
                        </Typography>
                        <Box sx={{ height: 200, display: 'flex', alignItems: 'flex-end', gap: 1 }}>
                          {Array.from({ length: 24 }).map((_, i) => {
                            const services = [
                              { height: 60 + Math.random() * 30, color: '#3b82f6' },
                              { height: 40 + Math.random() * 25, color: '#8b5cf6' },
                              { height: 30 + Math.random() * 20, color: '#ec4899' },
                            ];
                            return (
                              <Box key={i} sx={{ flex: 1, display: 'flex', gap: 0.25 }}>
                                {services.map((s, idx) => (
                                  <Box
                                    key={idx}
                                    sx={{
                                      flex: 1,
                                      height: `${s.height}%`,
                                      bgcolor: s.color,
                                      borderRadius: '2px 2px 0 0',
                                    }}
                                  />
                                ))}
                              </Box>
                            );
                          })}
                        </Box>
                        <Stack direction="row" spacing={3} sx={{ justifyContent: 'center', mt: 2 }}>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#3b82f6' }} />
                            <Typography variant="caption" sx={{ fontWeight: 700 }}>
                              Export Service
                            </Typography>
                          </Stack>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#8b5cf6' }} />
                            <Typography variant="caption" sx={{ fontWeight: 700 }}>
                              Scheduler
                            </Typography>
                          </Stack>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#ec4899' }} />
                            <Typography variant="caption" sx={{ fontWeight: 700 }}>
                              Rules Engine
                            </Typography>
                          </Stack>
                        </Stack>
                      </CardContent>
                    </Card>
                  </Grid>
                </Grid>
              </TabPanel>

              {/* Settings Tab */}
              <TabPanel value={activeTab} index={2}>
                <Grid container spacing={3}>
                  <Grid item xs={12} md={6}>
                    <Card>
                      <CardHeader title="Transport Channels" />
                      <CardContent>
                        <Stack spacing={2}>
                          <Stack direction="row" justifyContent="space-between" alignItems="center">
                            <Box>
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                Server-Sent Events (SSE)
                              </Typography>
                              <Typography variant="caption" sx={{ color: '#6b7280' }}>
                                HTTP streaming, one-way, 847 active connections
                              </Typography>
                            </Box>
                            <Switch
                              checked={notificationSettings.enableSSE}
                              onChange={(e) =>
                                setNotificationSettings({
                                  ...notificationSettings,
                                  enableSSE: e.target.checked,
                                })
                              }
                            />
                          </Stack>

                          <Divider />

                          <Stack direction="row" justifyContent="space-between" alignItems="center">
                            <Box>
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                WebSocket
                              </Typography>
                              <Typography variant="caption" sx={{ color: '#6b7280' }}>
                                Bidirectional, low-latency, real-time updates
                              </Typography>
                            </Box>
                            <Switch
                              checked={notificationSettings.enableWebSocket}
                              onChange={(e) =>
                                setNotificationSettings({
                                  ...notificationSettings,
                                  enableWebSocket: e.target.checked,
                                })
                              }
                            />
                          </Stack>
                        </Stack>
                      </CardContent>
                    </Card>
                  </Grid>

                  <Grid item xs={12} md={6}>
                    <Card>
                      <CardHeader title="Notification Channels" />
                      <CardContent>
                        <Stack spacing={2}>
                          <Stack direction="row" justifyContent="space-between" alignItems="center">
                            <Box>
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                Email Notifications
                              </Typography>
                              <Typography variant="caption" sx={{ color: '#6b7280' }}>
                                Daily digest of critical events
                              </Typography>
                            </Box>
                            <Switch
                              checked={notificationSettings.enableEmail}
                              onChange={(e) =>
                                setNotificationSettings({
                                  ...notificationSettings,
                                  enableEmail: e.target.checked,
                                })
                              }
                            />
                          </Stack>

                          <Divider />

                          <Stack direction="row" justifyContent="space-between" alignItems="center">
                            <Box>
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                Slack Integration
                              </Typography>
                              <Typography variant="caption" sx={{ color: '#6b7280' }}>
                                Critical alerts sent to #data-ops channel
                              </Typography>
                            </Box>
                            <Switch
                              checked={notificationSettings.enableSlack}
                              onChange={(e) =>
                                setNotificationSettings({
                                  ...notificationSettings,
                                  enableSlack: e.target.checked,
                                })
                              }
                            />
                          </Stack>
                        </Stack>
                      </CardContent>
                    </Card>
                  </Grid>
                </Grid>
              </TabPanel>
            </Box>
          </Box>

          {/* Right Panel - Selected Event Details */}
          {selectedNotification && (
            <Paper
              elevation={0}
              sx={{
                width: 320,
                borderLeft: '1px solid #e5e7eb',
                borderRadius: 0,
                display: 'flex',
                flexDirection: 'column',
                bgcolor: '#ffffff',
                overflow: 'hidden',
              }}
            >
              <Box sx={{ p: 2, borderBottom: '1px solid #e5e7eb', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  Event Details
                </Typography>
                <IconButton size="small" onClick={() => setSelectedNotification(null)}>
                  <CloseIcon />
                </IconButton>
              </Box>

              <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280' }}>
                      Type
                    </Typography>
                    <Stack direction="row" alignItems="center" spacing={1} sx={{ mt: 1 }}>
                      {getIconForType(selectedNotification.type)}
                      <Chip
                        label={selectedNotification.type.toUpperCase()}
                        size="small"
                        sx={{
                          bgcolor: getColorForType(selectedNotification.type),
                          color: '#ffffff',
                        }}
                      />
                    </Stack>
                  </Box>

                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280' }}>
                      Title
                    </Typography>
                    <Typography variant="body2" sx={{ mt: 1, fontWeight: 700 }}>
                      {selectedNotification.title}
                    </Typography>
                  </Box>

                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280' }}>
                      Message
                    </Typography>
                    <Typography variant="body2" sx={{ mt: 1, color: '#6b7280' }}>
                      {selectedNotification.message}
                    </Typography>
                  </Box>

                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280' }}>
                      Source Service
                    </Typography>
                    <Chip
                      label={selectedNotification.source || 'Unknown'}
                      size="small"
                      variant="outlined"
                      sx={{ mt: 1 }}
                    />
                  </Box>

                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280' }}>
                      Timestamp
                    </Typography>
                    <Typography variant="body2" sx={{ mt: 1, fontFamily: 'monospace' }}>
                      {new Date().toISOString()}
                    </Typography>
                  </Box>

                  {selectedNotification.progress !== undefined && (
                    <Box>
                      <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280' }}>
                        Progress
                      </Typography>
                      <Box sx={{ mt: 1 }}>
                        <LinearProgress
                          variant="determinate"
                          value={selectedNotification.progress}
                          sx={{ height: 8, borderRadius: 1, mb: 0.5 }}
                        />
                        <Typography variant="caption" sx={{ fontWeight: 700 }}>
                          {selectedNotification.progress}%
                        </Typography>
                      </Box>
                    </Box>
                  )}
                </Stack>
              </Box>

              <Box sx={{ p: 2, borderTop: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
                <Stack spacing={1}>
                  <Button variant="contained" fullWidth size="small" startIcon={<DoneAllIcon />}>
                    Mark as Read
                  </Button>
                  <Button variant="text" size="small" fullWidth>
                    More Actions
                  </Button>
                </Stack>
              </Box>
            </Paper>
          )}
        </Box>
      </Box>

      <style>
        {`
          @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
          }
        `}
      </style>
    </ThemeProvider>
  );
};

export default RealtimeNotificationsDashboard;
