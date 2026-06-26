import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Drawer,
  LinearProgress,
  IconButton,
  Tooltip
} from '@mui/material';
import {
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  TrendingUp as TrendingUpIcon,
  Assessment as AssessmentIcon,
  Close as CloseIcon,
  Refresh as RefreshIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';

// Mock Data Types
interface Trade {
  id: string;
  orderId: string;
  customerName: string;
  amount: number;
  riskScore: number;
  status: 'pending' | 'approved' | 'rejected' | 'escalated';
  createdAt: string;
}

interface RiskExplanation {
  explanation: Array<{
    feature: string;
    feature_value: number;
    impact: number;
  }>;
  base_value: number;
}

const RiskDashboardPage: React.FC = () => {
  const { tenant } = useTenant();
  const [highRiskTrades, setHighRiskTrades] = useState<Trade[]>([]);
  const [selectedTrade, setSelectedTrade] = useState<Trade | null>(null);
  const [explanation, setExplanation] = useState<RiskExplanation | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    // Mock fetching high-risk trades
    // In production, fetch from API
    setHighRiskTrades([
      { id: 'T-1001', orderId: 'ORD-5501', customerName: 'Vins et alcools Chevalier', amount: 12500, riskScore: 0.82, status: 'pending', createdAt: '2025-10-27T10:00:00Z' },
      { id: 'T-1002', orderId: 'ORD-5504', customerName: 'Hanari Carnes', amount: 4500, riskScore: 0.76, status: 'escalated', createdAt: '2025-10-27T10:15:00Z' },
      { id: 'T-1003', orderId: 'ORD-5509', customerName: 'Suprêmes délices', amount: 8200, riskScore: 0.65, status: 'pending', createdAt: '2025-10-27T10:30:00Z' },
    ]);
  }, []);

  const handleSelectTrade = async (trade: Trade) => {
    setSelectedTrade(trade);
    setLoading(true);
    
    // Simulate fetching SHAP explanation from backend
    // In production: await fetch('/api/v1/risk/explain/' + trade.id)
    setTimeout(() => {
      setExplanation({
        base_value: 0.05,
        explanation: [
          { feature: 'is_cross_border', feature_value: 1, impact: 0.35 },
          { feature: 'customer_previous_fails', feature_value: 3, impact: 0.28 },
          { feature: 'line_item_count', feature_value: 25, impact: 0.15 },
          { feature: 'order_total_value', feature_value: 12500, impact: 0.12 },
          { feature: 'customer_country', feature_value: 4, impact: -0.05 },
        ]
      });
      setLoading(false);
    }, 800);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1" fontWeight="bold">
          Settlement Risk Dashboard
        </Typography>
        <Button startIcon={<RefreshIcon />} variant="outlined">Refresh</Button>
      </Box>

      {/* KPI Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} md={3}>
          <Paper sx={{ p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography color="textSecondary" variant="subtitle2">Model Accuracy</Typography>
              <Typography variant="h4" fontWeight="bold">94.2%</Typography>
            </Box>
            <CheckCircleIcon color="success" sx={{ fontSize: 40, opacity: 0.2 }} />
          </Paper>
        </Grid>
        <Grid item xs={12} md={3}>
          <Paper sx={{ p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography color="textSecondary" variant="subtitle2">Data Drift</Typography>
              <Typography variant="h4" fontWeight="bold" color="warning.main">Low</Typography>
            </Box>
            <AssessmentIcon color="warning" sx={{ fontSize: 40, opacity: 0.2 }} />
          </Paper>
        </Grid>
        <Grid item xs={12} md={3}>
          <Paper sx={{ p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography color="textSecondary" variant="subtitle2">High Risk Vol.</Typography>
              <Typography variant="h4" fontWeight="bold">12</Typography>
            </Box>
            <WarningIcon color="error" sx={{ fontSize: 40, opacity: 0.2 }} />
          </Paper>
        </Grid>
        <Grid item xs={12} md={3}>
          <Paper sx={{ p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography color="textSecondary" variant="subtitle2">Avg. Risk Score</Typography>
              <Typography variant="h4" fontWeight="bold">0.18</Typography>
            </Box>
            <TrendingUpIcon color="primary" sx={{ fontSize: 40, opacity: 0.2 }} />
          </Paper>
        </Grid>
      </Grid>

      {/* High Risk Queue */}
      <Paper elevation={0} variant="outlined">
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Typography variant="h6">High-Risk Trades Queue</Typography>
        </Box>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Trade ID</TableCell>
                <TableCell>Customer</TableCell>
                <TableCell align="right">Amount</TableCell>
                <TableCell align="center">Risk Score</TableCell>
                <TableCell>Status</TableCell>
                <TableCell align="right">Action</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {highRiskTrades.map((trade) => (
                <TableRow key={trade.id} hover onClick={() => handleSelectTrade(trade)} sx={{ cursor: 'pointer' }}>
                  <TableCell>{trade.id}</TableCell>
                  <TableCell>{trade.customerName}</TableCell>
                  <TableCell align="right">${trade.amount.toLocaleString()}</TableCell>
                  <TableCell align="center">
                    <Chip 
                      label={`${(trade.riskScore * 100).toFixed(0)}%`} 
                      color={trade.riskScore > 0.75 ? 'error' : 'warning'} 
                      size="small" 
                    />
                  </TableCell>
                  <TableCell>
                    <Chip label={trade.status} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell align="right">
                    <Button size="small" variant="contained">Review</Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Detail Drawer */}
      <Drawer
        anchor="right"
        open={Boolean(selectedTrade)}
        onClose={() => setSelectedTrade(null)}
        PaperProps={{ sx: { width: 450, p: 3 } }}
      >
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h5" fontWeight="bold">Risk Analysis</Typography>
          <IconButton onClick={() => setSelectedTrade(null)}>
            <CloseIcon />
          </IconButton>
        </Box>

        {selectedTrade && (
          <>
            <Box sx={{ mb: 4, textAlign: 'center' }}>
              <Typography color="textSecondary" gutterBottom>Overall Risk Score</Typography>
              <Box sx={{ position: 'relative', display: 'inline-flex' }}>
                <Typography variant="h2" color="error.main" fontWeight="bold">
                  {(selectedTrade.riskScore * 100).toFixed(0)}%
                </Typography>
              </Box>
              <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
                Order ID: {selectedTrade.orderId}
              </Typography>
            </Box>

            <Typography variant="h6" gutterBottom>Top Risk Drivers (SHAP)</Typography>
            <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
              Why the model predicted this risk score
            </Typography>

            {loading ? (
              <LinearProgress />
            ) : explanation ? (
              <Box>
                {explanation.explanation.map((item, idx) => (
                  <Box key={idx} sx={{ mb: 2 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                      <Typography variant="body2" fontWeight="medium">
                        {item.feature}
                      </Typography>
                      <Typography 
                        variant="body2" 
                        color={item.impact > 0 ? 'error.main' : 'success.main'}
                        fontWeight="bold"
                      >
                        {item.impact > 0 ? '+' : ''}{(item.impact * 100).toFixed(1)}%
                      </Typography>
                    </Box>
                    <Typography variant="caption" color="textSecondary" display="block" gutterBottom>
                      Value: {item.feature_value}
                    </Typography>
                    <LinearProgress 
                      variant="determinate" 
                      value={Math.min(Math.abs(item.impact) * 300, 100)} // Scale purely for visualization
                      color={item.impact > 0 ? 'error' : 'success'}
                      sx={{ height: 6, borderRadius: 3 }}
                    />
                  </Box>
                ))}
              </Box>
            ) : null}

            <Box sx={{ mt: 4, display: 'flex', gap: 2 }}>
              <Button fullWidth variant="contained" color="error">Reject</Button>
              <Button fullWidth variant="contained" color="success">Approve</Button>
            </Box>
            <Button fullWidth variant="outlined" sx={{ mt: 2 }}>Escalate to Compliance</Button>
          </>
        )}
      </Drawer>
    </Box>
  );
};

export default RiskDashboardPage;
