import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  LinearProgress,
  Tabs,
  Tab,
  Alert,
  Button,
} from '@mui/material';
import {
  TrendingUp as UpIcon,
  AccountBalance as FundIcon,
  Home as RealEstateIcon,
  ShowChart as HedgeIcon,
} from '@mui/icons-material';
import { PieChart, Pie, Cell, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';

interface AlternativeInvestment {
  investment_id: string;
  investment_name: string;
  asset_class: string;
  committed_capital: number;
  funded_capital: number;
  unfunded_commitment: number;
  current_nav: number;
  total_distributions: number;
  irr: number;
  moic: number;
  tvpi: number;
  dpi: number;
  rvpi: number;
  investment_status: string;
}

interface CapitalCall {
  call_id: string;
  investment_name: string;
  call_date: string;
  due_date: string;
  amount: number;
  call_status: string;
  days_until_due: number;
}

const COLORS = ['#1976d2', '#dc004e', '#9c27b0', '#ff9800', '#4caf50', '#00bcd4'];

export const AlternativeInvestmentsDashboard: React.FC = () => {
  const [investments, setInvestments] = useState<AlternativeInvestment[]>([]);
  const [capitalCalls, setCapitalCalls] = useState<CapitalCall[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState(0);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [investmentsRes, callsRes] = await Promise.all([
        fetch('/api/alternative-investments'),
        fetch('/api/alternative-investments/capital-calls?status=PENDING'),
      ]);

      const investmentsData = await investmentsRes.json();
      const callsData = await callsRes.json();

      setInvestments(investmentsData.investments || []);
      setCapitalCalls(callsData.capital_calls || []);
    } catch (error) {
      console.error('Failed to fetch data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <LinearProgress />;
  }

  // Calculate summary metrics
  const totalCommitted = investments.reduce((sum, inv) => sum + inv.committed_capital, 0);
  const totalFunded = investments.reduce((sum, inv) => sum + inv.funded_capital, 0);
  const totalNAV = investments.reduce((sum, inv) => sum + inv.current_nav, 0);
  const totalDistributions = investments.reduce((sum, inv) => sum + inv.total_distributions, 0);
  const totalValue = totalNAV + totalDistributions;
  const portfolioMOIC = totalFunded > 0 ? totalValue / totalFunded : 0;

  // Allocation by asset class
  const allocationData = investments.reduce((acc, inv) => {
    const existing = acc.find(item => item.name === inv.asset_class);
    if (existing) {
      existing.value += inv.current_nav;
    } else {
      acc.push({ name: inv.asset_class, value: inv.current_nav });
    }
    return acc;
  }, [] as Array<{ name: string; value: number }>);

  // Upcoming capital calls
  const urgentCalls = capitalCalls.filter(call => call.days_until_due <= 7);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ACTIVE': return 'success';
      case 'COMMITTED': return 'info';
      case 'DISTRIBUTING': return 'warning';
      default: return 'default';
    }
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Typography variant="h4" gutterBottom>
        Alternative Investments
      </Typography>

      {/* Urgent Capital Calls Alert */}
      {urgentCalls.length > 0 && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          <Typography variant="subtitle2">
            {urgentCalls.length} capital call{urgentCalls.length > 1 ? 's' : ''} due within 7 days
          </Typography>
          {urgentCalls.map(call => (
            <Typography key={call.call_id} variant="body2">
              {call.investment_name}: {formatCurrency(call.amount)} due in {call.days_until_due} days
            </Typography>
          ))}
        </Alert>
      )}

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Total Committed
              </Typography>
              <Typography variant="h5">{formatCurrency(totalCommitted)}</Typography>
              <Typography variant="caption" color="text.secondary">
                Funded: {formatCurrency(totalFunded)} ({((totalFunded / totalCommitted) * 100).toFixed(1)}%)
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Current NAV
              </Typography>
              <Typography variant="h5">{formatCurrency(totalNAV)}</Typography>
              <Typography variant="caption" color="text.secondary">
                + Distributions: {formatCurrency(totalDistributions)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Portfolio MOIC
              </Typography>
              <Typography variant="h5">{portfolioMOIC.toFixed(2)}x</Typography>
              <Typography variant="caption" color="success.main">
                <UpIcon fontSize="small" sx={{ verticalAlign: 'middle' }} />
                {((portfolioMOIC - 1) * 100).toFixed(1)}% gain
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Total Investments
              </Typography>
              <Typography variant="h5">{investments.length}</Typography>
              <Typography variant="caption" color="text.secondary">
                Across {allocationData.length} asset classes
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tab} onChange={(_, v) => setTab(v)}>
          <Tab label="All Investments" />
          <Tab label="Capital Calls" />
          <Tab label="Allocation" />
        </Tabs>
      </Box>

      {/* Tab: All Investments */}
      {tab === 0 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Investment</TableCell>
                <TableCell>Asset Class</TableCell>
                <TableCell align="right">Committed</TableCell>
                <TableCell align="right">Funded</TableCell>
                <TableCell align="right">Current NAV</TableCell>
                <TableCell align="right">MOIC</TableCell>
                <TableCell align="right">IRR</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {investments.map((inv) => (
                <TableRow key={inv.investment_id} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {inv.investment_name}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={inv.asset_class.replace('_', ' ')} size="small" />
                  </TableCell>
                  <TableCell align="right">{formatCurrency(inv.committed_capital)}</TableCell>
                  <TableCell align="right">
                    {formatCurrency(inv.funded_capital)}
                    <Box sx={{ width: '100%', mt: 0.5 }}>
                      <LinearProgress
                        variant="determinate"
                        value={(inv.funded_capital / inv.committed_capital) * 100}
                        sx={{ height: 4 }}
                      />
                    </Box>
                  </TableCell>
                  <TableCell align="right">{formatCurrency(inv.current_nav)}</TableCell>
                  <TableCell align="right">
                    <Typography variant="body2" color={inv.moic >= 1 ? 'success.main' : 'error.main'}>
                      {inv.moic?.toFixed(2)}x
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="body2" color={inv.irr >= 0 ? 'success.main' : 'error.main'}>
                      {inv.irr?.toFixed(1)}%
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={inv.investment_status} size="small" color={getStatusColor(inv.investment_status)} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Tab: Capital Calls */}
      {tab === 1 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Investment</TableCell>
                <TableCell>Call Date</TableCell>
                <TableCell>Due Date</TableCell>
                <TableCell align="right">Amount</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {capitalCalls.map((call) => (
                <TableRow key={call.call_id} hover>
                  <TableCell>{call.investment_name}</TableCell>
                  <TableCell>{new Date(call.call_date).toLocaleDateString()}</TableCell>
                  <TableCell>
                    {new Date(call.due_date).toLocaleDateString()}
                    {call.days_until_due <= 7 && (
                      <Chip
                        label={`${call.days_until_due} days`}
                        size="small"
                        color="warning"
                        sx={{ ml: 1 }}
                      />
                    )}
                  </TableCell>
                  <TableCell align="right">{formatCurrency(call.amount)}</TableCell>
                  <TableCell>
                    <Chip label={call.call_status} size="small" />
                  </TableCell>
                  <TableCell>
                    <Button size="small" variant="outlined">
                      Pay Now
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Tab: Allocation */}
      {tab === 2 && (
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Asset Class Allocation
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={allocationData}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    outerRadius={100}
                    label={(entry) => `${entry.name}: ${((entry.value / totalNAV) * 100).toFixed(1)}%`}
                  >
                    {allocationData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip formatter={(value: number) => formatCurrency(value)} />
                </PieChart>
              </ResponsiveContainer>
            </Paper>
          </Grid>

          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Allocation Breakdown
              </Typography>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Asset Class</TableCell>
                    <TableCell align="right">Value</TableCell>
                    <TableCell align="right">%</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {allocationData.map((item, index) => (
                    <TableRow key={item.name}>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Box
                            sx={{
                              width: 12,
                              height: 12,
                              bgcolor: COLORS[index % COLORS.length],
                              borderRadius: '50%',
                            }}
                          />
                          {item.name.replace('_', ' ')}
                        </Box>
                      </TableCell>
                      <TableCell align="right">{formatCurrency(item.value)}</TableCell>
                      <TableCell align="right">{((item.value / totalNAV) * 100).toFixed(1)}%</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Paper>
          </Grid>
        </Grid>
      )}
    </Box>
  );
};
