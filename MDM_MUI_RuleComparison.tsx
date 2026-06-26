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
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Chip,
  LinearProgress,
  Divider,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  CssBaseline,
} from '@mui/material';
import {
  Search as SearchIcon,
  Architecture as ArchitectureIcon,
  Notifications as NotificationsIcon,
  History as HistoryIcon,
  Code as CodeIcon,
  BarChart as BarChartIcon,
  TableRows as TableRowsIcon,
  Send as SendIcon,
  Info as InfoIcon,
  Gavel as GavelIcon,
  Check as CheckIcon,
  Edit as EditIcon,
  Lock as LockIcon,
  TrendingUp as TrendingUpIcon,
  Dataset as DatasetIcon,
  Verified as VerifiedIcon,
  Cloud as CloudIcon,
  Database as DatabaseIcon,
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

export const RuleImpactComparison: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedRule, setSelectedRule] = useState('rule-092');
  const [justification, setJustification] = useState('');

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const rules = [
    { id: 'rule-091', name: 'RULE-USR-091 Validation' },
    { id: 'rule-092', name: 'RULE-USR-092 Comparison', active: true },
    { id: 'rule-093', name: 'RULE-USR-093 Matching' },
    { id: 'rule-001', name: 'RULE-FIN-001 Ledger Map' },
  ];

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: '#f6f7f8' }}>
        {/* Header */}
        <AppBar position="static" elevation={1} sx={{ bgcolor: '#ffffff', color: '#1f2937' }}>
          <Toolbar>
            <Stack direction="row" alignItems="center" spacing={2} sx={{ flexGrow: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#137fec' }}>
                <ArchitectureIcon sx={{ fontSize: '2rem' }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  Usice Architecture
                </Typography>
              </Box>

              <Stack direction="row" spacing={3} sx={{ display: { xs: 'none', md: 'flex' }, ml: 4 }}>
                <Button color="inherit">Dashboard</Button>
                <Button color="inherit" sx={{ color: '#137fec', fontWeight: 700 }}>
                  Rule Sets
                </Button>
                <Button color="inherit">Data Sources</Button>
                <Button color="inherit">Governance</Button>
              </Stack>
            </Stack>

            <Stack direction="row" spacing={2} alignItems="center">
              <TextField
                placeholder="Search rules..."
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
        <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
          {/* Sidebar */}
          <Paper
            elevation={0}
            sx={{
              width: 256,
              borderRight: '1px solid #e5e7eb',
              borderRadius: 0,
              display: 'flex',
              flexDirection: 'column',
              bgcolor: '#ffffff',
            }}
          >
            <Box sx={{ p: 2, borderBottom: '1px solid #e5e7eb' }}>
              <Typography
                variant="caption"
                sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280', mb: 1, display: 'block' }}
              >
                📁 Enterprise Data Management
              </Typography>
              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                Customer Master Rule Set
              </Typography>
            </Box>

            <List sx={{ flex: 1, overflow: 'auto', p: 1 }}>
              {rules.map((rule) => (
                <ListItemButton
                  key={rule.id}
                  selected={selectedRule === rule.id}
                  onClick={() => setSelectedRule(rule.id)}
                  sx={{
                    mb: 0.5,
                    borderRadius: 1,
                    '&.Mui-selected': {
                      bgcolor: '#eff6ff',
                      color: '#137fec',
                      borderLeft: '3px solid #137fec',
                      fontWeight: 700,
                    },
                  }}
                >
                  <ListItemText
                    primary={rule.name}
                    primaryTypographyProps={{ variant: 'body2', sx: { fontWeight: 600 } }}
                  />
                </ListItemButton>
              ))}
            </List>
          </Paper>

          {/* Main Content Area */}
          <Box sx={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
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
                    Rule Sets / {selectedRule}
                  </Typography>
                  <Typography variant="h4" sx={{ fontWeight: 900, mt: 1 }}>
                    Rule Impact & Version Comparison
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#6b7280', mt: 0.5 }}>
                    Comparing <strong style={{ color: '#374151' }}>v2.1 (Production)</strong> with{' '}
                    <strong style={{ color: '#137fec' }}>v2.2 (Draft)</strong>
                  </Typography>
                </Box>

                <Stack direction="row" spacing={2}>
                  <Button
                    variant="outlined"
                    startIcon={<HistoryIcon />}
                    size="small"
                  >
                    View History
                  </Button>
                  <Button
                    variant="contained"
                    size="small"
                  >
                    Refresh Analysis
                  </Button>
                </Stack>
              </Stack>

              {/* Tabs */}
              <Tabs value={activeTab} onChange={handleTabChange} sx={{ mt: 2 }}>
                <Tab
                  icon={<CodeIcon sx={{ mr: 1 }} />}
                  label="Logic Diff"
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                />
                <Tab
                  icon={<BarChartIcon sx={{ mr: 1 }} />}
                  label="Impact Analysis"
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                />
                <Tab
                  icon={<TableRowsIcon sx={{ mr: 1 }} />}
                  label="Sample Results"
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                />
              </Tabs>
            </Paper>

            {/* Tab Content */}
            <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
              {/* Logic Diff Tab */}
              <TabPanel value={activeTab} index={0}>
                <Grid container spacing={3}>
                  {/* V2.1 */}
                  <Grid item xs={12} md={6}>
                    <Card>
                      <CardHeader
                        title="V2.1 - PRODUCTION"
                        subheader="Oct 12, 2023"
                        sx={{ bgcolor: '#f3f4f6', pb: 1 }}
                        titleTypographyProps={{ variant: 'caption', sx: { fontWeight: 700, textTransform: 'uppercase' } }}
                      />
                      <CardContent sx={{ fontFamily: 'monospace', fontSize: '0.875rem', lineHeight: 1.8 }}>
                        <Box sx={{ '& .removed': { textDecoration: 'line-through', color: '#991b1b' } }}>
                          1 DEFINE RULE "Customer_Source_Priority"
                          <br />
                          2 PRIORITY_ORDER [ Salesforce, SAP_ERP, Legacy_DB ]
                          <br />
                          <Box component="span" className="removed" sx={{ display: 'block', bgcolor: '#fef2f2', pl: 1, borderLeft: '3px solid #ef4444' }}>
                            3 IF Salesforce.confidence &gt; 0.85 THEN
                          </Box>
                          <Box component="span" className="removed" sx={{ display: 'block', bgcolor: '#fef2f2', pl: 1, borderLeft: '3px solid #ef4444' }}>
                            4     USE Salesforce.data
                          </Box>
                          5 ELSE IF SAP_ERP.is_active THEN
                          <br />
                          6     USE SAP_ERP.data
                          <br />
                          7 END IF
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>

                  {/* V2.2 */}
                  <Grid item xs={12} md={6}>
                    <Card>
                      <CardHeader
                        title="V2.2 - DRAFT"
                        subheader="Unsaved Changes"
                        sx={{
                          bgcolor: '#eff6ff',
                          pb: 1,
                          '& .MuiCardHeader-title': { color: '#137fec', fontWeight: 700 },
                        }}
                        titleTypographyProps={{ variant: 'caption', sx: { fontWeight: 700, textTransform: 'uppercase' } }}
                      />
                      <CardContent sx={{ fontFamily: 'monospace', fontSize: '0.875rem', lineHeight: 1.8 }}>
                        <Box sx={{ '& .added': { backgroundColor: '#ecfdf5', borderLeft: '3px solid #10b981', pl: 1 } }}>
                          1 DEFINE RULE "Customer_Source_Priority"
                          <br />
                          2 PRIORITY_ORDER [ Salesforce, SAP_ERP, Legacy_DB ]
                          <br />
                          <Box component="span" className="added" sx={{ display: 'block' }}>
                            3 IF Salesforce.confidence &gt; 0.75 THEN
                          </Box>
                          <Box component="span" className="added" sx={{ display: 'block' }}>
                            4     AND Salesforce.last_updated &lt; 30_DAYS
                          </Box>
                          <Box component="span" className="added" sx={{ display: 'block' }}>
                            5     USE Salesforce.data
                          </Box>
                          6 ELSE IF SAP_ERP.is_active THEN
                          <br />
                          7     USE SAP_ERP.data
                          <br />
                          8 END IF
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>
                </Grid>
              </TabPanel>

              {/* Impact Analysis Tab */}
              <TabPanel value={activeTab} index={1}>
                <Stack spacing={3}>
                  {/* Distribution Chart */}
                  <Card>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="start" sx={{ mb: 3 }}>
                        <Box>
                          <Typography variant="h6" sx={{ fontWeight: 700 }}>
                            Confidence Impact Distribution
                          </Typography>
                          <Typography variant="body2" sx={{ color: '#6b7280', mt: 0.5 }}>
                            Visualizing record-level confidence shifts across 1.2M rows
                          </Typography>
                        </Box>
                        <Stack direction="row" spacing={2}>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#d1d5db' }} />
                            <Typography variant="caption">v2.1</Typography>
                          </Stack>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <Box sx={{ width: 12, height: 12, borderRadius: '2px', bgcolor: '#137fec' }} />
                            <Typography variant="caption">v2.2 (Draft)</Typography>
                          </Stack>
                        </Stack>
                      </Stack>

                      {/* Chart Visualization */}
                      <Box sx={{ height: 200, display: 'flex', alignItems: 'flex-end', gap: 0.5, mb: 2 }}>
                        {[15, 22, 38, 55, 75, 92, 82, 65, 45, 32, 18, 10].map((height, idx) => (
                          <Box
                            key={idx}
                            sx={{
                              flex: 1,
                              height: `${height}%`,
                              bgcolor: idx >= 5 && idx <= 7 ? '#137fec' : '#d1d5db',
                              borderRadius: '4px 4px 0 0',
                              opacity: idx >= 5 && idx <= 7 ? 1 : 0.5,
                            }}
                          />
                        ))}
                      </Box>

                      <Stack direction="row" justifyContent="space-between" sx={{ fontSize: '0.75rem', color: '#6b7280', fontWeight: 700, mb: 3 }}>
                        <span>Low Confidence (0.0)</span>
                        <span>0.5</span>
                        <span>High Confidence (1.0)</span>
                      </Stack>

                      {/* Metrics */}
                      <Grid container spacing={2}>
                        <Grid item xs={12} sm={6} md={3}>
                          <Card variant="outlined">
                            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                              <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', display: 'block' }}>
                                Avg Confidence Shift
                              </Typography>
                              <Stack direction="row" alignItems="center" spacing={1} sx={{ mt: 1 }}>
                                <Typography variant="h5" sx={{ fontWeight: 900 }}>
                                  +4.2%
                                </Typography>
                                <Box sx={{ p: 1, bgcolor: '#ecfdf5', borderRadius: '50%', color: '#10b981' }}>
                                  <TrendingUpIcon fontSize="small" />
                                </Box>
                              </Stack>
                            </CardContent>
                          </Card>
                        </Grid>

                        <Grid item xs={12} sm={6} md={3}>
                          <Card variant="outlined">
                            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                              <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', display: 'block' }}>
                                Records Affected
                              </Typography>
                              <Stack direction="row" alignItems="center" spacing={1} sx={{ mt: 1 }}>
                                <Typography variant="h5" sx={{ fontWeight: 900 }}>
                                  14.2k
                                </Typography>
                                <Box sx={{ p: 1, bgcolor: '#eff6ff', borderRadius: '50%', color: '#137fec' }}>
                                  <DatasetIcon fontSize="small" />
                                </Box>
                              </Stack>
                            </CardContent>
                          </Card>
                        </Grid>

                        <Grid item xs={12} sm={6} md={3}>
                          <Card variant="outlined">
                            <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                              <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', display: 'block' }}>
                                Conflict Resolution Rate
                              </Typography>
                              <Stack direction="row" alignItems="center" spacing={1} sx={{ mt: 1 }}>
                                <Typography variant="h5" sx={{ fontWeight: 900 }}>
                                  +12%
                                </Typography>
                                <Box sx={{ p: 1, bgcolor: '#dbeafe', borderRadius: '50%', color: '#0284c7' }}>
                                  <VerifiedIcon fontSize="small" />
                                </Box>
                              </Stack>
                            </CardContent>
                          </Card>
                        </Grid>
                      </Grid>
                    </CardContent>
                  </Card>

                  {/* Source Trust Shift */}
                  <Card>
                    <CardContent>
                      <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
                        Source Trust Shift
                      </Typography>
                      <Typography variant="body2" sx={{ color: '#6b7280', mb: 2 }}>
                        Distribution of &quot;Winning&quot; data sources between rule versions
                      </Typography>

                      <Stack spacing={3}>
                        {/* Salesforce */}
                        <Box>
                          <Stack direction="row" justifyContent="space-between" sx={{ mb: 1 }}>
                            <Stack direction="row" alignItems="center" spacing={1}>
                              <CloudIcon sx={{ color: '#4f46e5' }} />
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                Salesforce
                              </Typography>
                            </Stack>
                            <Typography variant="caption" sx={{ fontWeight: 700, color: '#6b7280' }}>
                              68% of Total Records
                            </Typography>
                          </Stack>

                          <Box sx={{ display: 'flex', height: 32, borderRadius: 1, overflow: 'hidden', gap: 0.25 }}>
                            <Box sx={{ flex: '54%', bgcolor: '#d1d5db' }} />
                            <Box sx={{ flex: '14%', bgcolor: '#4f46e5', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                              <Typography sx={{ fontSize: '0.625rem', color: '#ffffff', fontWeight: 700 }}>
                                +14%
                              </Typography>
                            </Box>
                          </Box>
                        </Box>

                        {/* SAP ERP */}
                        <Box>
                          <Stack direction="row" justifyContent="space-between" sx={{ mb: 1 }}>
                            <Stack direction="row" alignItems="center" spacing={1}>
                              <DatabaseIcon sx={{ color: '#3b82f6' }} />
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                SAP ERP
                              </Typography>
                            </Stack>
                            <Typography variant="caption" sx={{ fontWeight: 700, color: '#6b7280' }}>
                              22% of Total Records
                            </Typography>
                          </Stack>

                          <Box sx={{ display: 'flex', height: 32, borderRadius: 1, overflow: 'hidden', gap: 0.25 }}>
                            <Box sx={{ flex: '22%', bgcolor: '#3b82f6' }} />
                            <Box sx={{ flex: '8%', bgcolor: '#e5e7eb', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                              <Typography sx={{ fontSize: '0.625rem', color: '#6b7280', fontWeight: 700 }}>
                                -8%
                              </Typography>
                            </Box>
                          </Box>
                        </Box>
                      </Stack>
                    </CardContent>
                  </Card>
                </Stack>
              </TabPanel>

              {/* Results Tab */}
              <TabPanel value={activeTab} index={2}>
                <Card>
                  <CardHeader
                    title="Sample Validation Results"
                    subheader="Detailed row-level analysis for specific data points"
                  />
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow sx={{ bgcolor: '#f3f4f6' }}>
                          <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                            Row ID / Date
                          </TableCell>
                          <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                            Customer Attribute
                          </TableCell>
                          <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                            v2.1 Source
                          </TableCell>
                          <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                            v2.2 Source
                          </TableCell>
                          <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                            Status
                          </TableCell>
                          <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase' }}>
                            Shift
                          </TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {[
                          { id: '#ORD-99120', attr: 'Primary Email', v21: 'SAP_ERP', v22: 'Salesforce', color: '#f59e0b', shift: '+12%' },
                          { id: '#ORD-99125', attr: 'Postal Code (HQ)', v21: 'Salesforce', v22: 'Salesforce', color: '#6b7280', shift: '0%' },
                          { id: '#ORD-99132', attr: 'Credit Score (Ext)', v21: 'Legacy_DB', v22: 'SAP_ERP', color: '#f59e0b', shift: '+34%' },
                        ].map((row, idx) => (
                          <TableRow key={idx} hover>
                            <TableCell>
                              <Box>
                                <Typography variant="body2" sx={{ fontWeight: 700 }}>
                                  {row.id}
                                </Typography>
                                <Typography variant="caption" sx={{ color: '#6b7280' }}>
                                  Oct {24 + idx}, 2023
                                </Typography>
                              </Box>
                            </TableCell>
                            <TableCell>{row.attr}</TableCell>
                            <TableCell>
                              <Chip label={row.v21} size="small" variant="outlined" />
                            </TableCell>
                            <TableCell>
                              <Chip label={row.v22} size="small" variant="outlined" />
                            </TableCell>
                            <TableCell>
                              <Typography variant="caption" sx={{ fontWeight: 700, color: row.color }}>
                                {row.shift !== '0%' ? '🔄 Switched' : 'No Change'}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Typography
                                variant="body2"
                                sx={{
                                  fontWeight: 700,
                                  color: row.shift !== '0%' ? '#10b981' : '#6b7280',
                                }}
                              >
                                {row.shift}
                              </Typography>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                  <Box sx={{ p: 2, textAlign: 'center', borderTop: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
                    <Button size="small">View All 1.2M Analyzed Rows</Button>
                  </Box>
                </Card>
              </TabPanel>
            </Box>
          </Box>

          {/* Right Sidebar: Governance */}
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
            <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
              <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 3 }}>
                <GavelIcon sx={{ color: '#137fec' }} />
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  Governance
                </Typography>
              </Stack>

              {/* Approval Chain */}
              <Box sx={{ mb: 4 }}>
                <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280', mb: 2, display: 'block' }}>
                  Approval Chain
                </Typography>

                <Stack spacing={2}>
                  {[
                    { step: 'Logic Validation', status: 'Completed by John Doe', icon: 'check', color: '#10b981', active: false },
                    { step: 'Drafting Justification', status: 'In Progress (You)', icon: 'edit', color: '#137fec', active: true },
                    { step: 'Manager Review', status: 'Pending Submission', icon: '3', color: '#9ca3af', active: false },
                  ].map((item, idx) => (
                    <Stack key={idx} direction="row" spacing={2}>
                      <Box>
                        <Avatar
                          sx={{
                            width: 28,
                            height: 28,
                            bgcolor: item.color,
                            color: '#ffffff',
                            fontSize: '0.875rem',
                            fontWeight: 700,
                          }}
                        >
                          {item.icon === 'check' && <CheckIcon fontSize="small" />}
                          {item.icon === 'edit' && <EditIcon fontSize="small" />}
                          {item.icon === '3' && '3'}
                        </Avatar>
                      </Box>
                      <Box sx={{ flex: 1 }}>
                        <Typography
                          variant="body2"
                          sx={{ fontWeight: 700, color: item.active ? item.color : '#374151' }}
                        >
                          {item.step}
                        </Typography>
                        <Typography
                          variant="caption"
                          sx={{ color: item.active ? item.color : '#6b7280', display: 'block', mt: 0.25 }}
                        >
                          {item.status}
                        </Typography>
                      </Box>
                    </Stack>
                  ))}
                </Stack>
              </Box>

              {/* Justification Input */}
              <Box sx={{ mb: 3 }}>
                <Typography
                  variant="caption"
                  sx={{ fontWeight: 700, textTransform: 'uppercase', color: '#6b7280', mb: 1, display: 'block' }}
                >
                  Change Justification
                </Typography>
                <TextField
                  multiline
                  rows={6}
                  placeholder="Explain the business impact..."
                  fullWidth
                  size="small"
                  value={justification}
                  onChange={(e) => setJustification(e.target.value)}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      bgcolor: '#f3f4f6',
                    },
                  }}
                />
                <Stack direction="row" spacing={0.5} sx={{ mt: 1, fontSize: '0.75rem', color: '#6b7280' }}>
                  <InfoIcon sx={{ fontSize: '0.875rem' }} />
                  <span>Minimum 50 characters required for audit trail.</span>
                </Stack>
              </Box>

              {/* Risk Level */}
              <Box sx={{ mb: 3, p: 2, bgcolor: '#f3f4f6', borderRadius: 1 }}>
                <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 1 }}>
                  Estimated Risk Level
                </Typography>
                <Stack direction="row" alignItems="center" spacing={1}>
                  <Box sx={{ flex: 1, height: 6, bgcolor: '#10b981', borderRadius: 1 }} />
                  <Box sx={{ flex: 1, height: 6, bgcolor: '#10b981', borderRadius: 1 }} />
                  <Box sx={{ flex: 1, height: 6, bgcolor: '#e5e7eb', borderRadius: 1 }} />
                  <Box sx={{ flex: 1, height: 6, bgcolor: '#e5e7eb', borderRadius: 1 }} />
                  <Typography variant="caption" sx={{ fontWeight: 700, ml: 1 }}>
                    Low-Med
                  </Typography>
                </Stack>
              </Box>
            </Box>

            {/* Footer Actions */}
            <Paper
              elevation={0}
              sx={{
                p: 2,
                borderTop: '1px solid #e5e7eb',
                bgcolor: '#f9fafb',
                borderRadius: 0,
              }}
            >
              <Stack spacing={1}>
                <Button
                  variant="contained"
                  fullWidth
                  endIcon={<SendIcon />}
                  sx={{ fontWeight: 700 }}
                >
                  Submit for Review
                </Button>
                <Button fullWidth variant="text" size="small">
                  Save as Draft
                </Button>
              </Stack>
            </Paper>
          </Paper>
        </Box>
      </Box>
    </ThemeProvider>
  );
};

export default RuleImpactComparison;
