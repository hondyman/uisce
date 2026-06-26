import React, { useState } from 'react';
import {
  Box,
  Container,
  Paper,
  Card,
  CardContent,
  CardHeader,
  Grid,
  TextField,
  Button,
  AppBar,
  Toolbar,
  Typography,
  Avatar,
  IconButton,
  Tabs,
  Tab,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Chip,
  LinearProgress,
  Divider,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Stack,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Tooltip,
} from '@mui/material';
import {
  Search as SearchIcon,
  AccountTree as AccountTreeIcon,
  ChevronRight as ChevronRightIcon,
  Settings as SettingsIcon,
  Notifications as NotificationsIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  DragIndicator as DragIndicatorIcon,
  Edit as EditIcon,
  Download as DownloadIcon,
  Filter as FilterIcon,
  Publish as PublishIcon,
  TrendingUp as TrendingUpIcon,
  MoreVert as MoreVertIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';

// Custom theme matching your color scheme
const theme = createTheme({
  palette: {
    primary: {
      main: '#137fec',
      light: '#4a9eff',
      dark: '#0e5fb7',
    },
    background: {
      default: '#f6f7f8',
      paper: '#ffffff',
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
    fontFamily: '"Inter", "Helvetica", "Arial", sans-serif',
    h1: {
      fontSize: '2.5rem',
      fontWeight: 900,
      letterSpacing: '-0.02em',
    },
    h2: {
      fontSize: '1.875rem',
      fontWeight: 800,
    },
    h3: {
      fontSize: '1.25rem',
      fontWeight: 700,
    },
    body1: {
      fontSize: '0.875rem',
    },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 600,
          borderRadius: '0.5rem',
          transition: 'all 200ms ease-in-out',
        },
        contained: {
          boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
          '&:hover': {
            boxShadow: '0 8px 12px rgba(0, 0, 0, 0.15)',
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: '0.75rem',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
          '&:hover': {
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
            transition: 'box-shadow 200ms ease-in-out',
          },
        },
      },
    },
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
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

// Semantic Rule Builder Dashboard Component
export const SemanticRuleBuilderDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedRule, setSelectedRule] = useState('rule-1');

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: '#f6f7f8' }}>
        {/* Top Navigation */}
        <AppBar position="sticky" elevation={1} sx={{ bgcolor: '#ffffff', color: '#1f2937' }}>
          <Toolbar>
            <Stack direction="row" alignItems="center" spacing={2} sx={{ flexGrow: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#137fec' }}>
                <AccountTreeIcon sx={{ fontSize: '2rem' }} />
                <Typography variant="h3" sx={{ fontWeight: 700 }}>
                  Usice Rule Builder
                </Typography>
              </Box>
              <Divider orientation="vertical" flexItem />
              <Stack direction="row" spacing={3} sx={{ display: { xs: 'none', md: 'flex' } }}>
                <Button color="inherit" sx={{ fontSize: '0.875rem' }}>
                  MDM
                </Button>
                <ChevronRightIcon sx={{ color: '#9ca3af' }} />
                <Typography sx={{ fontSize: '0.875rem', color: '#374151' }}>
                  Holiday Rules
                </Typography>
              </Stack>
            </Stack>

            <Stack direction="row" spacing={2} alignItems="center">
              <Tabs
                value={activeTab}
                onChange={handleTabChange}
                sx={{
                  bgcolor: '#f3f4f6',
                  borderRadius: '0.5rem',
                  mr: 2,
                  '& .MuiTabs-indicator': { display: 'none' },
                }}
              >
                <Tab label="Editor" sx={{ fontWeight: 600, fontSize: '0.75rem' }} />
                <Tab label="Analytics" sx={{ fontWeight: 600, fontSize: '0.75rem' }} />
              </Tabs>

              <Button variant="outlined" size="small">
                Save Draft
              </Button>
              <Button
                variant="contained"
                size="small"
                startIcon={<PublishIcon />}
                sx={{ px: 2 }}
              >
                Publish
              </Button>
              <Avatar
                src="https://lh3.googleusercontent.com/aida-public/AB6AXuB3b4K6UiFDwAJ0O2M8FR-1Ge9fyU0N05jFT45NCL_FNbaS0bCRQpzyAA5A-6cvHwxVeVe1u64tl4rwPVxpxun6j4CBJsRxS9XarnCzfROxjobM7sWIK2daECCGg6RvG57a9AqcAzMEjNbon8_QOpzugTkRl1yYmlvJVFUmOUNit4AdF6cl7sg97MVkOCHv7_aLCxod9wTJK8KOSWosyLp20mCjyd5h6mkXG0rLuGoocj3wU92xMPyeXp7LY7Vt_qUoc4LP0WNbHPNs"
                sx={{ width: 32, height: 32 }}
              />
            </Stack>
          </Toolbar>
        </AppBar>

        {/* Main Content */}
        <Grid container sx={{ flex: 1, gap: 0 }}>
          {/* Left Panel: Semantic Catalog */}
          <Grid item xs={12} md={3} sx={{ borderRight: '1px solid #e5e7eb' }}>
            <Paper
              elevation={0}
              sx={{
                height: '100%',
                bgcolor: '#ffffff',
                borderRadius: 0,
                p: 2,
                overflow: 'auto',
              }}
            >
              <Stack spacing={2}>
                <Stack spacing={1}>
                  <Stack direction="row" justifyContent="space-between" alignItems="center">
                    <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                      Semantic Catalog
                    </Typography>
                    <Tooltip title="Filter">
                      <IconButton size="small">
                        <FilterIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </Stack>
                  <TextField
                    placeholder="Search objects or terms..."
                    size="small"
                    fullWidth
                    InputProps={{
                      startAdornment: <SearchIcon sx={{ mr: 1, color: '#9ca3af' }} />,
                    }}
                    variant="outlined"
                  />
                </Stack>

                {/* Business Objects */}
                <Box>
                  <Typography
                    variant="caption"
                    sx={{
                      fontWeight: 700,
                      textTransform: 'uppercase',
                      color: '#9ca3af',
                      mb: 1,
                      display: 'block',
                    }}
                  >
                    📊 Business Objects
                  </Typography>
                  <Stack spacing={1}>
                    {['HolidaySchedule', 'MarketClock'].map((obj) => (
                      <Card
                        key={obj}
                        sx={{
                          cursor: 'grab',
                          '&:hover': {
                            borderColor: '#137fec',
                            bgcolor: 'rgba(19, 127, 236, 0.05)',
                          },
                        }}
                      >
                        <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                          <Stack direction="row" justifyContent="space-between" alignItems="start">
                            <Typography variant="body2" sx={{ fontWeight: 600 }}>
                              {obj}
                            </Typography>
                            <DragIndicatorIcon fontSize="small" sx={{ color: '#d1d5db' }} />
                          </Stack>
                          <Typography variant="caption" sx={{ color: '#6b7280', mt: 0.5, display: 'block' }}>
                            {obj === 'HolidaySchedule'
                              ? 'Calendar rules for global financial markets'
                              : 'Trading session timing definitions'}
                          </Typography>
                        </CardContent>
                      </Card>
                    ))}
                  </Stack>
                </Box>

                {/* Semantic Terms */}
                <Box>
                  <Typography
                    variant="caption"
                    sx={{
                      fontWeight: 700,
                      textTransform: 'uppercase',
                      color: '#9ca3af',
                      mb: 1,
                      display: 'block',
                      mt: 2,
                    }}
                  >
                    🏷️ Semantic Terms
                  </Typography>
                  <Stack spacing={1}>
                    {[
                      { name: 'CalendarDate', type: 'Date', color: '#3b82f6' },
                      { name: 'IsBusinessDay', type: 'Boolean', color: '#10b981' },
                      { name: 'MarketClose', type: 'Time', color: '#8b5cf6' },
                    ].map((term) => (
                      <Card key={term.name}>
                        <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <Avatar
                              sx={{
                                width: 32,
                                height: 32,
                                bgcolor: `${term.color}15`,
                                color: term.color,
                              }}
                            >
                              {term.name.charAt(0)}
                            </Avatar>
                            <Box>
                              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                                {term.name}
                              </Typography>
                              <Typography variant="caption" sx={{ color: '#9ca3af', fontWeight: 600 }}>
                                Data Type: {term.type}
                              </Typography>
                            </Box>
                          </Stack>
                        </CardContent>
                      </Card>
                    ))}
                  </Stack>
                </Box>
              </Stack>
            </Paper>
          </Grid>

          {/* Center Panel: Priority Hierarchy Editor */}
          <Grid item xs={12} md={6} sx={{ bgcolor: '#f9fafb' }}>
            <Paper
              elevation={0}
              sx={{
                height: '100%',
                borderRadius: 0,
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              {/* Header */}
              <Box sx={{ p: 3, borderBottom: '1px solid #e5e7eb', bgcolor: '#ffffff' }}>
                <Stack direction="row" justifyContent="space-between" alignItems="start">
                  <Box>
                    <Typography variant="h3">Priority Hierarchy Editor</Typography>
                    <Typography variant="body2" sx={{ color: '#6b7280', mt: 0.5 }}>
                      Define rule precedence and logical fallback sequence.
                    </Typography>
                  </Box>
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<AddIcon />}
                    sx={{ color: '#137fec', borderColor: '#137fec' }}
                  >
                    Add Step
                  </Button>
                </Stack>
              </Box>

              {/* Content */}
              <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
                <Stack spacing={3}>
                  {/* Priority Step 1 */}
                  <Card sx={{ borderLeft: '4px solid #137fec' }}>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start" sx={{ mb: 2 }}>
                        <Stack direction="row" spacing={2} alignItems="center">
                          <Avatar
                            sx={{
                              width: 28,
                              height: 28,
                              bgcolor: '#137fec',
                              fontSize: '0.75rem',
                              fontWeight: 700,
                            }}
                          >
                            01
                          </Avatar>
                          <Typography variant="body1" sx={{ fontWeight: 700 }}>
                            Regional Overrides
                          </Typography>
                        </Stack>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Chip label="Active" size="small" color="success" variant="outlined" />
                          <IconButton size="small">
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Stack>
                      </Stack>

                      <Stack spacing={2}>
                        {/* IF Condition */}
                        <Box sx={{ p: 2, bgcolor: '#f3f4f6', borderRadius: '0.5rem' }}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <Typography
                              sx={{
                                fontSize: '0.75rem',
                                fontWeight: 700,
                                color: '#137fec',
                                width: '2rem',
                              }}
                            >
                              IF
                            </Typography>
                            <Stack direction="row" spacing={1} alignItems="center" sx={{ flex: 1 }}>
                              <Chip label="[Source.System]" size="small" variant="outlined" />
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                is
                              </Typography>
                              <Chip label="'TradingHours.com'" size="small" variant="filled" />
                            </Stack>
                          </Stack>
                        </Box>

                        {/* THEN Action */}
                        <Box sx={{ p: 2, bgcolor: '#f3f4f6', borderRadius: '0.5rem' }}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <Typography
                              sx={{
                                fontSize: '0.75rem',
                                fontWeight: 700,
                                color: '#137fec',
                                width: '2rem',
                              }}
                            >
                              THEN
                            </Typography>
                            <Stack direction="row" spacing={1} alignItems="center" sx={{ flex: 1 }}>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Use Value
                              </Typography>
                              <Chip
                                label="[Source.Value]"
                                size="small"
                                sx={{
                                  bgcolor: '#ecfdf5',
                                  color: '#10b981',
                                  fontWeight: 600,
                                }}
                              />
                            </Stack>
                          </Stack>
                        </Box>

                        {/* Confidence Slider */}
                        <Box>
                          <Stack direction="row" justifyContent="space-between" sx={{ mb: 1 }}>
                            <Typography
                              variant="caption"
                              sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#9ca3af' }}
                            >
                              Confidence Score
                            </Typography>
                            <Typography variant="caption" sx={{ fontWeight: 700, color: '#10b981' }}>
                              95%
                            </Typography>
                          </Stack>
                          <LinearProgress
                            variant="determinate"
                            value={95}
                            sx={{
                              height: 6,
                              borderRadius: 1,
                              backgroundColor: '#e5e7eb',
                              '& .MuiLinearProgress-bar': {
                                backgroundColor: '#10b981',
                              },
                            }}
                          />
                        </Box>
                      </Stack>
                    </CardContent>
                  </Card>

                  {/* Priority Step 2 */}
                  <Card sx={{ borderLeft: '4px solid #f59e0b', opacity: 0.9 }}>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start" sx={{ mb: 2 }}>
                        <Stack direction="row" spacing={2} alignItems="center">
                          <Avatar
                            sx={{
                              width: 28,
                              height: 28,
                              bgcolor: '#f59e0b',
                              fontSize: '0.75rem',
                              fontWeight: 700,
                            }}
                          >
                            02
                          </Avatar>
                          <Typography variant="body1" sx={{ fontWeight: 700 }}>
                            Legacy Mapping
                          </Typography>
                        </Stack>
                        <IconButton size="small">
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>

                      <Stack spacing={2}>
                        <Box sx={{ p: 2, bgcolor: '#f3f4f6', borderRadius: '0.5rem' }}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <Typography sx={{ fontSize: '0.75rem', fontWeight: 700, color: '#137fec' }}>
                              IF
                            </Typography>
                            <Stack direction="row" spacing={1} alignItems="center" sx={{ flex: 1 }}>
                              <Chip label="[Metadata.Source]" size="small" variant="outlined" />
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                contains
                              </Typography>
                              <Chip label="'V2_LEGACY'" size="small" variant="filled" />
                            </Stack>
                          </Stack>
                        </Box>

                        <Box sx={{ p: 2, bgcolor: '#f3f4f6', borderRadius: '0.5rem' }}>
                          <Stack direction="row" spacing={2} alignItems="center">
                            <Typography sx={{ fontSize: '0.75rem', fontWeight: 700, color: '#137fec' }}>
                              THEN
                            </Typography>
                            <Stack direction="row" spacing={1} alignItems="center" sx={{ flex: 1 }}>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Apply Function
                              </Typography>
                              <Chip
                                label="reformatDate()"
                                size="small"
                                sx={{
                                  bgcolor: '#ede9fe',
                                  color: '#7c3aed',
                                  fontWeight: 600,
                                }}
                              />
                            </Stack>
                          </Stack>
                        </Box>

                        <Box>
                          <Stack direction="row" justifyContent="space-between" sx={{ mb: 1 }}>
                            <Typography
                              variant="caption"
                              sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#9ca3af' }}
                            >
                              Confidence Score
                            </Typography>
                            <Typography variant="caption" sx={{ fontWeight: 700, color: '#f59e0b' }}>
                              65%
                            </Typography>
                          </Stack>
                          <LinearProgress
                            variant="determinate"
                            value={65}
                            sx={{
                              height: 6,
                              borderRadius: 1,
                              backgroundColor: '#e5e7eb',
                              '& .MuiLinearProgress-bar': {
                                backgroundColor: '#f59e0b',
                              },
                            }}
                          />
                        </Box>
                      </Stack>
                    </CardContent>
                  </Card>

                  {/* Default Fallback */}
                  <Card sx={{ bgcolor: '#f3f4f6', border: '2px dashed #e5e7eb' }}>
                    <CardContent>
                      <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
                        <SettingsIcon sx={{ color: '#9ca3af' }} />
                        <Typography
                          sx={{
                            fontSize: '0.875rem',
                            fontWeight: 700,
                            textTransform: 'uppercase',
                            color: '#6b7280',
                          }}
                        >
                          Default Fallback
                        </Typography>
                      </Stack>
                      <Box sx={{ p: 2, bgcolor: '#ffffff', borderRadius: '0.5rem' }}>
                        <Stack direction="row" spacing={2} alignItems="center">
                          <Typography sx={{ fontSize: '0.75rem', fontWeight: 700 }}>ELSE</Typography>
                          <Typography variant="caption" sx={{ fontWeight: 600 }}>
                            Return NULL or throw
                          </Typography>
                          <Chip label="Schema_Exception" size="small" variant="filled" />
                        </Stack>
                      </Box>
                    </CardContent>
                  </Card>
                </Stack>
              </Box>
            </Paper>
          </Grid>

          {/* Right Panel: Simulation & Impact */}
          <Grid item xs={12} md={3} sx={{ borderLeft: '1px solid #e5e7eb' }}>
            <Paper
              elevation={0}
              sx={{
                height: '100%',
                borderRadius: 0,
                bgcolor: '#ffffff',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              {/* Header */}
              <Box sx={{ p: 2, borderBottom: '1px solid #e5e7eb' }}>
                <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                  Simulation & Impact
                </Typography>

                <Stack spacing={1.5} sx={{ mt: 2 }}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Test Dataset</InputLabel>
                    <Select
                      label="Test Dataset"
                      defaultValue="nyse"
                      size="small"
                    >
                      <MenuItem value="nyse">NYSE_2024_H1_Sample.csv</MenuItem>
                      <MenuItem value="london">London_Exchange_Data.json</MenuItem>
                      <MenuItem value="custom">Custom Upload...</MenuItem>
                    </Select>
                  </FormControl>
                  <Button
                    variant="contained"
                    fullWidth
                    startIcon={<PublishIcon />}
                    sx={{ textTransform: 'none' }}
                  >
                    Run Simulation
                  </Button>
                </Stack>
              </Box>

              {/* Content */}
              <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
                <Stack spacing={3}>
                  {/* Impact Summary */}
                  <Box>
                    <Typography
                      variant="caption"
                      sx={{
                        fontWeight: 700,
                        textTransform: 'uppercase',
                        color: '#9ca3af',
                        mb: 1,
                        display: 'block',
                      }}
                    >
                      📊 Impact Summary
                    </Typography>
                    <Grid container spacing={1.5}>
                      <Grid item xs={6}>
                        <Card>
                          <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                            <Typography
                              variant="caption"
                              sx={{ fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 0.5 }}
                            >
                              Records
                            </Typography>
                            <Typography variant="h6" sx={{ fontWeight: 900 }}>
                              127
                            </Typography>
                            <Typography
                              variant="caption"
                              sx={{ color: '#10b981', fontWeight: 700, display: 'flex', alignItems: 'center', mt: 0.5 }}
                            >
                              <TrendingUpIcon sx={{ fontSize: '0.875rem', mr: 0.25 }} /> +5%
                            </Typography>
                          </CardContent>
                        </Card>
                      </Grid>
                      <Grid item xs={6}>
                        <Card>
                          <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                            <Typography
                              variant="caption"
                              sx={{ fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 0.5 }}
                            >
                              Accuracy
                            </Typography>
                            <Typography variant="h6" sx={{ fontWeight: 900 }}>
                              98.2%
                            </Typography>
                            <LinearProgress
                              variant="determinate"
                              value={98}
                              sx={{
                                mt: 1,
                                height: 4,
                                '& .MuiLinearProgress-bar': {
                                  backgroundColor: '#137fec',
                                },
                              }}
                            />
                          </CardContent>
                        </Card>
                      </Grid>
                    </Grid>
                  </Box>

                  {/* Execution Trace */}
                  <Box>
                    <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1.5 }}>
                      <Typography
                        variant="caption"
                        sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#9ca3af' }}
                      >
                        📋 Execution Trace
                      </Typography>
                      <Button size="small">View Details</Button>
                    </Stack>

                    <Stack spacing={2}>
                      {[
                        { rule: 'Rule 01', status: 'NO MATCH', color: '#f59e0b', icon: '✗' },
                        { rule: 'Rule 02', status: 'Logic Executed', color: '#10b981', icon: '✓' },
                        { rule: 'Default', status: 'Terminated', color: '#9ca3af', icon: '-', opacity: 0.5 },
                      ].map((trace, idx) => (
                        <Box
                          key={idx}
                          sx={{
                            p: 2,
                            bgcolor: '#f3f4f6',
                            borderRadius: '0.5rem',
                            opacity: trace.opacity || 1,
                            borderLeft: `3px solid ${trace.color}`,
                          }}
                        >
                          <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                            {trace.rule}
                          </Typography>
                          <Typography variant="caption" sx={{ display: 'block', color: trace.color, fontWeight: 600, mt: 0.5 }}>
                            {trace.status}
                          </Typography>
                        </Box>
                      ))}
                    </Stack>
                  </Box>
                </Stack>
              </Box>

              {/* Footer */}
              <Box sx={{ p: 2, borderTop: '1px solid #e5e7eb', display: 'flex', gap: 1 }}>
                <Button fullWidth variant="text" size="small">
                  Download Log
                </Button>
                <Button fullWidth variant="text" size="small">
                  Clear Cache
                </Button>
              </Box>
            </Paper>
          </Grid>
        </Grid>
      </Box>
    </ThemeProvider>
  );
};

export default SemanticRuleBuilderDashboard;
