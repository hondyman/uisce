import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  IconButton,
  Button,
  Chip,
  Avatar,
  Divider,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  TextField,
  InputAdornment,
  Tabs,
  Tab,
  Card,
  CardContent,
  LinearProgress,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import { alpha } from '@mui/material/styles';
import {
  Search as SearchIcon,
  AccountBalance as PortfolioIcon,
  Gavel as MandateIcon,
  Speed as PerformanceIcon,
  History as HistoryIcon,
  AutoGraph as LineageIcon,
  VerifiedUser as VerifiedIcon,
  Warning as WarningIcon,
  ArrowForwardIos as ChevronRightIcon,
  MoreVert as MoreIcon,
} from '@mui/icons-material';

// Mock Data for demonstration
const MOCK_PORTFOLIOS = [
  { id: '1', code: 'EQUITY_GLOBAL_1', name: 'Global Equities Fund', type: 'Fund', status: 'Gold', confidence: 98 },
  { id: '2', code: 'DMA_US_BLUE_G', name: 'US Blue Chip Growth', type: 'SMA', status: 'Gold', confidence: 95 },
  { id: '3', code: 'ETF_TECH_DIS', name: 'Tech Disruptors ETF', type: 'ETF', status: 'Warning', confidence: 78 },
  { id: '4', code: 'MANDATE_FIX_INC', name: 'Fixed Income Mandate', type: 'Mandate', status: 'Gold', confidence: 92 },
];

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function CustomTabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

export const PortfolioMasterDashboard: React.FC = () => {
  const [selectedId, setSelectedId] = useState('1');
  const [tabValue, setTabValue] = useState(0);

  const selectedPortfolio = MOCK_PORTFOLIOS.find(p => p.id === selectedId) || MOCK_PORTFOLIOS[0];

  const handleTabChange = (_: any, newValue: number) => {
    setTabValue(newValue);
  };

  return (
    <Box sx={{ display: 'flex', height: '100vh', bgcolor: '#f8fafc' }}>
      {/* Sidebar: Portfolio List */}
      <Paper elevation={0} sx={{ width: 320, borderRight: '1px solid #e2e8f0', display: 'flex', flexDirection: 'column' }}>
        <Box sx={{ p: 2, borderBottom: '1px solid #e2e8f0' }}>
          <Typography variant="h6" sx={{ fontWeight: 800, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
            <PortfolioIcon color="primary" /> Portfolio Master
          </Typography>
          <TextField
            fullWidth
            size="small"
            placeholder="Search portfolios..."
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" color="action" />
                </InputAdornment>
              ),
            }}
            sx={{ '& .MuiOutlinedInput-root': { borderRadius: 2 } }}
          />
        </Box>
        <List sx={{ flex: 1, overflow: 'auto', py: 0 }}>
          {MOCK_PORTFOLIOS.map((p) => (
            <ListItem key={p.id} disablePadding border-bottom="1px solid #f1f5f9">
              <ListItemButton
                selected={selectedId === p.id}
                onClick={() => setSelectedId(p.id)}
                sx={{
                  py: 2,
                  '&.Mui-selected': {
                    bgcolor: alpha('#137fec', 0.08),
                    borderRight: '3px solid #137fec',
                    '& .MuiListItemText-primary': { fontWeight: 700, color: 'primary.main' },
                  },
                }}
              >
                <ListItemText
                  primary={p.name}
                  secondary={
                    <Stack direction="row" spacing={1} sx={{ mt: 0.5 }}>
                      <Typography variant="caption" sx={{ fontWeight: 600 }}>{p.code}</Typography>
                      <Chip 
                        label={p.status} 
                        size="small" 
                        color={p.status === 'Gold' ? 'success' : 'warning'} 
                        sx={{ height: 16, fontSize: '0.625rem', fontWeight: 800 }} 
                      />
                    </Stack>
                  }
                />
                <ChevronRightIcon fontSize="small" sx={{ opacity: 0.3 }} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </Paper>

      {/* Main Content Area */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Header */}
        <Box sx={{ p: 3, bgcolor: 'white', borderBottom: '1px solid #e2e8f0' }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Stack direction="row" spacing={2} alignItems="center">
              <Avatar sx={{ bgcolor: 'primary.main', width: 48, height: 48 }}>
                <PortfolioIcon />
              </Avatar>
              <Box>
                <Typography variant="h4" sx={{ fontWeight: 800, letterSpacing: '-0.02em' }}>
                  {selectedPortfolio.name}
                </Typography>
                <Stack direction="row" spacing={2} divider={<Divider orientation="vertical" flexItem />}>
                  <Typography variant="body2" sx={{ color: 'text.secondary', fontWeight: 600 }}>
                    Code: {selectedPortfolio.code}
                  </Typography>
                  <Typography variant="body2" sx={{ color: 'text.secondary', fontWeight: 600 }}>
                    Type: {selectedPortfolio.type}
                  </Typography>
                  <Typography variant="body2" sx={{ color: 'text.secondary', fontWeight: 600, display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <VerifiedIcon fontSize="inherit" color="success" /> Master Gold Copy
                  </Typography>
                </Stack>
              </Box>
            </Stack>
            <Stack direction="row" spacing={1}>
              <Button variant="outlined" startIcon={<HistoryIcon />}>History</Button>
              <Button variant="contained" color="primary" startIcon={<VerifiedIcon />}>Re-Master</Button>
              <IconButton><MoreIcon /></IconButton>
            </Stack>
          </Stack>
        </Box>

        {/* Dynamic Detail Sections */}
        <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
          <Grid container spacing={3}>
            {/* KPI Cards */}
            <Grid size={{ xs: 12, md: 8 }}>
              <Paper sx={{ borderRadius: 3, p: 2, mb: 3 }}>
                <Tabs value={tabValue} onChange={handleTabChange} sx={{ borderBottom: 1, borderColor: 'divider' }}>
                  <Tab label="Core Metadata" icon={<PortfolioIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                  <Tab label="Mandate & Strategy" icon={<MandateIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                  <Tab label="Performance Policies" icon={<PerformanceIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                  <Tab label="Lineage & Rules" icon={<LineageIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                </Tabs>

                <CustomTabPanel value={tabValue} index={0}>
                  <Grid container spacing={3}>
                    {[
                      { label: 'Inception Date', value: '2018-05-12', source: 'AccountingSystem' },
                      { label: 'Base Currency', value: 'USD', source: 'Custodian' },
                      { label: 'Domicile', value: 'Luxembourg', source: 'FundAdmin' },
                      { label: 'Legal Structure', value: 'SICAV', source: 'AccountingSystem' },
                      { label: 'Asset Class', value: 'Equity', source: 'Internal' },
                      { label: 'Status', value: 'Active', source: 'OMS' },
                    ].map((item) => (
                      <Grid size={{ xs: 6, sm: 4 }} key={item.label}>
                        <Box>
                          <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 700, textTransform: 'uppercase' }}>
                            {item.label}
                          </Typography>
                          <Typography variant="body1" sx={{ fontWeight: 600, mb: 0.5 }}>{item.value}</Typography>
                          <Chip label={item.source} size="small" variant="outlined" sx={{ height: 18, fontSize: '0.625rem' }} />
                        </Box>
                      </Grid>
                    ))}
                  </Grid>
                </CustomTabPanel>

                <CustomTabPanel value={tabValue} index={2}>
                  {/* Performance Settings Section */}
                  <Typography variant="subtitle2" sx={{ fontWeight: 800, mb: 2 }}>Calculation Methodologies</Typography>
                  <Grid container spacing={3}>
                    <Grid size={{ xs: 6 }}>
                      <Card variant="outlined" sx={{ borderRadius: 2 }}>
                        <CardContent>
                          <Typography variant="caption" color="text.secondary">Valuation Method</Typography>
                          <Typography variant="body1" sx={{ fontWeight: 700 }}>Daily Time-Weighted (TIB)</Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                    <Grid size={{ xs: 6 }}>
                      <Card variant="outlined" sx={{ borderRadius: 2 }}>
                        <CardContent>
                          <Typography variant="caption" color="text.secondary">Fee Treatment</Typography>
                          <Typography variant="body1" sx={{ fontWeight: 700 }}>Net of All Fees</Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                      <Typography variant="caption" sx={{ fontWeight: 700, color: 'text.secondary', display: 'block', mt: 2, mb: 1 }}>
                        CURRENCY HEDGING POLICY
                      </Typography>
                      <Paper sx={{ p: 2, bgcolor: 'grey.50', border: '1px dashed #cbd5e1' }}>
                        <Typography variant="body2">
                          100% hedge of all non-base currency exposures back to USD using 1-month rolling forward contracts.
                        </Typography>
                      </Paper>
                    </Grid>
                  </Grid>
                </CustomTabPanel>
              </Paper>
            </Grid>

            {/* Right Panel: Health & Lineage */}
            <Grid size={{ xs: 12, md: 4 }}>
              <Stack spacing={3}>
                {/* Confidence Score Gauge */}
                <Paper sx={{ p: 3, borderRadius: 3, textAlign: 'center', position: 'relative', overflow: 'hidden' }}>
                  <Box sx={{ position: 'absolute', top: 0, left: 0, width: '100%', height: 4, bgcolor: selectedPortfolio.confidence > 90 ? 'success.main' : 'warning.main' }} />
                  <Typography variant="subtitle2" sx={{ fontWeight: 800, color: 'text.secondary', mb: 2 }}>
                    GOLD COPY CONFIDENCE
                  </Typography>
                  <Typography variant="h2" sx={{ fontWeight: 900, color: selectedPortfolio.confidence > 90 ? 'success.main' : 'warning.main' }}>
                    {selectedPortfolio.confidence}%
                  </Typography>
                  <Typography variant="caption" sx={{ color: 'text.secondary', mb: 2, display: 'block' }}>
                    Based on 4 corroborated sources
                  </Typography>
                  <LinearProgress 
                    variant="determinate" 
                    value={selectedPortfolio.confidence} 
                    color={selectedPortfolio.confidence > 90 ? 'success' : 'warning'}
                    sx={{ height: 8, borderRadius: 4 }}
                  />
                </Paper>

                {/* Survivorship Breakdown */}
                <Paper sx={{ p: 2, borderRadius: 3 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 800, mb: 2 }}>Survivorship Winning Ratio</Typography>
                  {[
                    { source: 'AccountingSystem', percentage: 65, color: '#137fec' },
                    { source: 'OMS', percentage: 20, color: '#8b5cf6' },
                    { source: 'Custodian', percentage: 10, color: '#10b981' },
                    { source: 'Manual', percentage: 5, color: '#f59e0b' },
                  ].map((item) => (
                    <Box key={item.source} sx={{ mb: 1.5 }}>
                      <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                        <Typography variant="caption" sx={{ fontWeight: 700 }}>{item.source}</Typography>
                        <Typography variant="caption" sx={{ fontWeight: 700 }}>{item.percentage}%</Typography>
                      </Stack>
                      <LinearProgress 
                        variant="determinate" 
                        value={item.percentage} 
                        sx={{ 
                          height: 6, 
                          borderRadius: 3,
                          bgcolor: 'grey.100',
                          '& .MuiLinearProgress-bar': { bgcolor: item.color }
                        }} 
                      />
                    </Box>
                  ))}
                </Paper>

                {/* DQ Alerts */}
                <Card sx={{ borderRadius: 3, border: '1px solid #fee2e2', bgcolor: '#fef2f2' }}>
                  <CardContent sx={{ p: 2 }}>
                    <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                      <WarningIcon color="error" fontSize="small" />
                      <Typography variant="subtitle2" sx={{ fontWeight: 800, color: '#991b1b' }}>Data Quality Issues</Typography>
                    </Stack>
                    <Typography variant="caption" sx={{ color: '#b91c1c', display: 'block', mb: 1 }}>
                      2 soft violations detected in the latest gold copy run.
                    </Typography>
                    <Button size="small" variant="contained" color="error" sx={{ textTransform: 'none', borderRadius: 2 }}>
                      Inspect Violations
                    </Button>
                  </CardContent>
                </Card>
              </Stack>
            </Grid>
          </Grid>
        </Box>
      </Box>
    </Box>
  );
};

export default PortfolioMasterDashboard;
