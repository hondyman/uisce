import React, { useState } from 'react';
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
  LinearProgress,
  Chip,
  Stack,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  Tooltip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import {
  Search as SearchIcon,
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  Business as BusinessIcon,
  People as PeopleIcon,
  Notifications as NotificationsIcon,
  Scale as ScaleIcon,
  WarningAmber as WarningAmberIcon,
  CheckCircle as CheckCircleIcon,
  MoreVert as MoreVertIcon,
  Download as DownloadIcon,
  Share as ShareIcon,
  BarChart as BarChartIcon,
  PieChart as PieChartIcon,
  Timeline as TimelineIcon,
  CloudUpload as CloudUploadIcon,
  Settings as SettingsIcon,
  Lock as LockIcon,
  Visibility as VisibilityIcon,
  MoreHoriz as MoreHorizIcon,
} from '@mui/icons-material';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';

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

export const ImpactAnalysisDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedBusinessUnit, setSelectedBusinessUnit] = useState('all');
  const [openDialog, setOpenDialog] = useState(false);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
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
                <BarChartIcon sx={{ fontSize: '2rem' }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  Impact Analysis Studio
                </Typography>
              </Box>

              <Stack direction="row" spacing={3} sx={{ display: { xs: 'none', md: 'flex' }, ml: 4 }}>
                <Button color="inherit">Dashboard</Button>
                <Button color="inherit" sx={{ color: '#137fec', fontWeight: 700 }}>
                  Business Impact
                </Button>
                <Button color="inherit">Distribution Maps</Button>
              </Stack>
            </Stack>

            <Stack direction="row" spacing={2} alignItems="center">
              <TextField
                placeholder="Search datasets..."
                size="small"
                variant="outlined"
                InputProps={{
                  startAdornment: <SearchIcon sx={{ mr: 1, color: '#9ca3af' }} />,
                }}
                sx={{
                  display: { xs: 'none', sm: 'block' },
                  '& .MuiOutlinedInput-root': {
                    bgcolor: '#f3f4f6',
                    '& fieldset': { border: 'none' },
                  },
                }}
              />
              <IconButton color="inherit">
                <NotificationsIcon />
              </IconButton>
              <Avatar sx={{ width: 32, height: 32, bgcolor: '#137fec' }} />
            </Stack>
          </Toolbar>
        </AppBar>

        {/* Main Content */}
        <Box sx={{ flex: 1, overflow: 'auto' }}>
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
                  Semantic Rules / Impact Analysis
                </Typography>
                <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                  Enterprise Business Impact Analysis
                </Typography>
                <Typography variant="body2" sx={{ color: '#6b7280', mt: 0.5 }}>
                  Comprehensive analysis of rule application across <strong style={{ color: '#374151' }}>14 business units</strong> and{' '}
                  <strong style={{ color: '#374151' }}>328 data domains</strong>
                </Typography>
              </Box>

              <Stack direction="row" spacing={2}>
                <Button
                  variant="outlined"
                  startIcon={<DownloadIcon />}
                  size="small"
                >
                  Export Report
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<ShareIcon />}
                  size="small"
                >
                  Share Analysis
                </Button>
              </Stack>
            </Stack>

            {/* Controls */}
            <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
              <FormControl size="small" sx={{ minWidth: 180 }}>
                <InputLabel>Business Unit</InputLabel>
                <Select
                  value={selectedBusinessUnit}
                  label="Business Unit"
                  onChange={(e) => setSelectedBusinessUnit(e.target.value)}
                >
                  <MenuItem value="all">All Units</MenuItem>
                  <MenuItem value="wealth">Wealth Management</MenuItem>
                  <MenuItem value="capital">Capital Markets</MenuItem>
                  <MenuItem value="banking">Banking Services</MenuItem>
                  <MenuItem value="ops">Operations</MenuItem>
                </Select>
              </FormControl>
              <TextField
                label="Date Range"
                type="date"
                InputLabelProps={{ shrink: true }}
                size="small"
                defaultValue="2024-01-01"
              />
              <TextField
                label="End Date"
                type="date"
                InputLabelProps={{ shrink: true }}
                size="small"
                defaultValue="2024-01-31"
              />
              <Button variant="text" size="small" sx={{ alignSelf: 'center' }}>
                Apply Filters
              </Button>
            </Stack>

            {/* Tabs */}
            <Tabs value={activeTab} onChange={handleTabChange}>
              <Tab
                icon={<BarChartIcon sx={{ mr: 1 }} />}
                label="Overview"
                sx={{ textTransform: 'none', fontWeight: 600 }}
              />
              <Tab
                icon={<BusinessIcon sx={{ mr: 1 }} />}
                label="Business Unit Breakdown"
                sx={{ textTransform: 'none', fontWeight: 600 }}
              />
              <Tab
                icon={<PieChartIcon sx={{ mr: 1 }} />}
                label="Process Distribution"
                sx={{ textTransform: 'none', fontWeight: 600 }}
              />
              <Tab
                icon={<TimelineIcon sx={{ mr: 1 }} />}
                label="Trend Analysis"
                sx={{ textTransform: 'none', fontWeight: 600 }}
              />
            </Tabs>
          </Paper>

          {/* Tab Content */}
          <Box sx={{ p: 3 }}>
            {/* Overview Tab */}
            <TabPanel value={activeTab} index={0}>
              <Grid container spacing={3}>
                {/* KPI Cards */}
                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start">
                        <Box>
                          <Typography
                            variant="caption"
                            sx={{
                              fontWeight: 700,
                              textTransform: 'uppercase',
                              color: '#6b7280',
                            }}
                          >
                            Total Impacted
                          </Typography>
                          <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                            234.2M
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{ color: '#10b981', fontWeight: 700, mt: 0.5, display: 'block' }}
                          >
                            <TrendingUpIcon sx={{ fontSize: '0.875rem', mr: 0.5, verticalAlign: 'middle' }} />
                            +12.5% vs last month
                          </Typography>
                        </Box>
                        <Avatar sx={{ width: 50, height: 50, bgcolor: '#dbeafe' }}>
                          <Box sx={{ color: '#137fec', fontSize: '1.5rem' }}>📊</Box>
                        </Avatar>
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start">
                        <Box>
                          <Typography
                            variant="caption"
                            sx={{
                              fontWeight: 700,
                              textTransform: 'uppercase',
                              color: '#6b7280',
                            }}
                          >
                            Accuracy Score
                          </Typography>
                          <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                            98.7%
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{ color: '#10b981', fontWeight: 700, mt: 0.5, display: 'block' }}
                          >
                            <CheckCircleIcon sx={{ fontSize: '0.875rem', mr: 0.5, verticalAlign: 'middle' }} />
                            Validated Last Hour
                          </Typography>
                        </Box>
                        <Avatar sx={{ width: 50, height: 50, bgcolor: '#ecfdf5' }}>
                          <Box sx={{ color: '#10b981', fontSize: '1.5rem' }}>✓</Box>
                        </Avatar>
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start">
                        <Box>
                          <Typography
                            variant="caption"
                            sx={{
                              fontWeight: 700,
                              textTransform: 'uppercase',
                              color: '#6b7280',
                            }}
                          >
                            Data Conflicts
                          </Typography>
                          <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                            3.2K
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{ color: '#f59e0b', fontWeight: 700, mt: 0.5, display: 'block' }}
                          >
                            <WarningAmberIcon sx={{ fontSize: '0.875rem', mr: 0.5, verticalAlign: 'middle' }} />
                            15 require attention
                          </Typography>
                        </Box>
                        <Avatar sx={{ width: 50, height: 50, bgcolor: '#fef3c7' }}>
                          <Box sx={{ color: '#f59e0b', fontSize: '1.5rem' }}>⚠️</Box>
                        </Avatar>
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start">
                        <Box>
                          <Typography
                            variant="caption"
                            sx={{
                              fontWeight: 700,
                              textTransform: 'uppercase',
                              color: '#6b7280',
                            }}
                          >
                            Remediation Time
                          </Typography>
                          <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                            2.1h
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{ color: '#6b7280', fontWeight: 700, mt: 0.5, display: 'block' }}
                          >
                            Avg Resolution Duration
                          </Typography>
                        </Box>
                        <Avatar sx={{ width: 50, height: 50, bgcolor: '#e0e7ff' }}>
                          <Box sx={{ color: '#4f46e5', fontSize: '1.5rem' }}>⏱️</Box>
                        </Avatar>
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>

                {/* Top Impact Areas */}
                <Grid item xs={12}>
                  <Card>
                    <CardHeader
                      title="Top Impact Areas by Revenue"
                      subheader="Departments with highest data impact from semantic rules"
                    />
                    <CardContent>
                      <Stack spacing={3}>
                        {[
                          { area: 'Wealth Management', impact: 78, revenue: '$34.2B', color: '#3b82f6', pct: '42%' },
                          { area: 'Capital Markets Trading', impact: 65, revenue: '$28.9B', color: '#8b5cf6', pct: '36%' },
                          { area: 'Global Operations', impact: 45, revenue: '$12.1B', color: '#ec4899', pct: '15%' },
                          { area: 'Risk Management', impact: 32, revenue: '$8.7B', color: '#f59e0b', pct: '7%' },
                        ].map((item, idx) => (
                          <Stack key={idx} spacing={1}>
                            <Stack direction="row" justifyContent="space-between" alignItems="center">
                              <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
                                <Box sx={{ flex: 0 }}>
                                  <Avatar sx={{ width: 40, height: 40, bgcolor: item.color }}>
                                    <BusinessIcon />
                                  </Avatar>
                                </Box>
                                <Box sx={{ flex: 1 }}>
                                  <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                    {item.area}
                                  </Typography>
                                  <Typography variant="caption" sx={{ color: '#6b7280' }}>
                                    Revenue Impact: {item.revenue}
                                  </Typography>
                                </Box>
                              </Stack>
                              <Typography variant="body2" sx={{ fontWeight: 700, minWidth: 50, textAlign: 'right' }}>
                                {item.pct}
                              </Typography>
                            </Stack>
                            <Box sx={{ display: 'flex', height: 24, borderRadius: 1, overflow: 'hidden', bgcolor: '#e5e7eb' }}>
                              <Box
                                sx={{
                                  width: `${(item.impact / 100) * 100}%`,
                                  bgcolor: item.color,
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'flex-end',
                                  pr: 1,
                                  color: '#ffffff',
                                  fontSize: '0.75rem',
                                  fontWeight: 700,
                                }}
                              >
                                {item.impact}%
                              </Box>
                            </Box>
                          </Stack>
                        ))}
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>
            </TabPanel>

            {/* Business Unit Breakdown Tab */}
            <TabPanel value={activeTab} index={1}>
              <Grid container spacing={3}>
                <Grid item xs={12}>
                  <Card>
                    <CardHeader title="Business Unit Analysis" />
                    <TableContainer>
                      <Table size="small">
                        <TableHead>
                          <TableRow sx={{ bgcolor: '#f3f4f6' }}>
                            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                              Business Unit
                            </TableCell>
                            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                              Records Affected
                            </TableCell>
                            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                              Accuracy
                            </TableCell>
                            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                              Coverage
                            </TableCell>
                            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                              Status
                            </TableCell>
                            <TableCell></TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {[
                            { bu: 'Wealth Management', records: '89.2M', accuracy: 99.2, coverage: 100, status: 'Active' },
                            { bu: 'Capital Markets', records: '76.5M', accuracy: 98.8, coverage: 100, status: 'Active' },
                            { bu: 'Banking Services', records: '45.3M', accuracy: 98.1, coverage: 95, status: 'Active' },
                            { bu: 'Operations', records: '23.2M', accuracy: 97.5, coverage: 85, status: 'Monitoring' },
                          ].map((row, idx) => (
                            <TableRow key={idx} hover>
                              <TableCell>
                                <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                  {row.bu}
                                </Typography>
                              </TableCell>
                              <TableCell>
                                <Typography variant="body2">{row.records}</Typography>
                              </TableCell>
                              <TableCell>
                                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                  <Box sx={{ flex: 1, height: 6, bgcolor: '#e5e7eb', borderRadius: 1, overflow: 'hidden' }}>
                                    <Box sx={{ width: `${row.accuracy}%`, height: '100%', bgcolor: '#10b981' }} />
                                  </Box>
                                  <Typography variant="caption" sx={{ fontWeight: 700, minWidth: 40 }}>
                                    {row.accuracy}%
                                  </Typography>
                                </Box>
                              </TableCell>
                              <TableCell>
                                <Chip
                                  label={`${row.coverage}%`}
                                  size="small"
                                  variant="outlined"
                                  color={row.coverage >= 95 ? 'success' : 'warning'}
                                />
                              </TableCell>
                              <TableCell>
                                <Chip
                                  label={row.status}
                                  size="small"
                                  color={row.status === 'Active' ? 'success' : 'warning'}
                                  variant="outlined"
                                />
                              </TableCell>
                              <TableCell>
                                <IconButton size="small">
                                  <MoreHorizIcon />
                                </IconButton>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  </Card>
                </Grid>
              </Grid>
            </TabPanel>

            {/* Process Distribution Tab */}
            <TabPanel value={activeTab} index={2}>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Card>
                    <CardHeader title="Rule Application by Process Type" />
                    <CardContent>
                      {[
                        { process: 'Data Validation', pct: 34, count: '1,248 rules' },
                        { process: 'Master Data Matching', pct: 28, count: '1,024 rules' },
                        { process: 'Conflict Resolution', pct: 22, count: '804 rules' },
                        { process: 'Governance Checks', pct: 16, count: '585 rules' },
                      ].map((item, idx) => (
                        <Box key={idx} sx={{ mb: 2 }}>
                          <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                            <Typography variant="body2" sx={{ fontWeight: 700 }}>
                              {item.process}
                            </Typography>
                            <Typography variant="caption" sx={{ color: '#6b7280' }}>
                              {item.count}
                            </Typography>
                          </Stack>
                          <Box sx={{ height: 24, bgcolor: '#e5e7eb', borderRadius: 1, overflow: 'hidden' }}>
                            <Box
                              sx={{
                                width: `${item.pct}%`,
                                height: '100%',
                                bgcolor: ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b'][idx],
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: idx === 0 ? 'flex-end' : 'flex-end',
                                pr: 1,
                              }}
                            >
                              <Typography sx={{ fontSize: '0.75rem', color: '#ffffff', fontWeight: 700 }}>
                                {item.pct}%
                              </Typography>
                            </Box>
                          </Box>
                        </Box>
                      ))}
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} md={6}>
                  <Card>
                    <CardHeader title="Platform Distribution" />
                    <CardContent>
                      {[
                        { platform: 'Salesforce', pct: 45, impact: '#137fec' },
                        { platform: 'SAP ERP', pct: 32, impact: '#8b5cf6' },
                        { platform: 'Legacy Systems', pct: 15, impact: '#f59e0b' },
                        { platform: 'Data Lake', pct: 8, impact: '#10b981' },
                      ].map((item, idx) => (
                        <Box key={idx} sx={{ mb: 2 }}>
                          <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                            <Typography variant="body2" sx={{ fontWeight: 700 }}>
                              {item.platform}
                            </Typography>
                            <Typography variant="caption" sx={{ color: '#6b7280', fontWeight: 700 }}>
                              {item.pct}%
                            </Typography>
                          </Stack>
                          <Box sx={{ height: 24, bgcolor: '#e5e7eb', borderRadius: 1, overflow: 'hidden' }}>
                            <Box
                              sx={{
                                width: `${item.pct}%`,
                                height: '100%',
                                bgcolor: item.impact,
                              }}
                            />
                          </Box>
                        </Box>
                      ))}
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>
            </TabPanel>

            {/* Trend Analysis Tab */}
            <TabPanel value={activeTab} index={3}>
              <Grid container spacing={3}>
                <Grid item xs={12}>
                  <Card>
                    <CardHeader
                      title="7-Day Performance Trend"
                      subheader="Accuracy, Coverage, and Impact Metrics"
                    />
                    <CardContent>
                      <Box sx={{ height: 300, display: 'flex', alignItems: 'flex-end', gap: 1, mb: 2 }}>
                        {Array.from({ length: 7 }).map((_, i) => (
                          <Box key={i} sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                            <Box sx={{ display: 'flex', gap: 0.25, flex: 1, alignItems: 'flex-end' }}>
                              <Box
                                sx={{
                                  flex: 1,
                                  height: `${80 + Math.random() * 20}%`,
                                  bgcolor: '#dbeafe',
                                  borderRadius: '4px 4px 0 0',
                                }}
                              />
                              <Box
                                sx={{
                                  flex: 1,
                                  height: `${70 + Math.random() * 25}%`,
                                  bgcolor: '#10b981',
                                  borderRadius: '4px 4px 0 0',
                                }}
                              />
                              <Box
                                sx={{
                                  flex: 1,
                                  height: `${60 + Math.random() * 30}%`,
                                  bgcolor: '#f59e0b',
                                  borderRadius: '4px 4px 0 0',
                                }}
                              />
                            </Box>
                            <Typography sx={{ fontSize: '0.625rem', textAlign: 'center', fontWeight: 700, color: '#6b7280' }}>
                              Day {i + 1}
                            </Typography>
                          </Box>
                        ))}
                      </Box>

                      <Stack direction="row" spacing={4} sx={{ justifyContent: 'center' }}>
                        <Stack direction="row" alignItems="center" spacing={1}>
                          <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#dbeafe' }} />
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>
                            Accuracy
                          </Typography>
                        </Stack>
                        <Stack direction="row" alignItems="center" spacing={1}>
                          <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#10b981' }} />
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>
                            Coverage
                          </Typography>
                        </Stack>
                        <Stack direction="row" alignItems="center" spacing={1}>
                          <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#f59e0b' }} />
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>
                            Anomalies
                          </Typography>
                        </Stack>
                      </Stack>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>
            </TabPanel>
          </Box>
        </Box>
      </Box>
    </ThemeProvider>
  );
};

export default ImpactAnalysisDashboard;
