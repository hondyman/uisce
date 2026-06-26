import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Chip,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  LinearProgress,
  Tabs,
  Tab,
} from '@mui/material';
import {
  TrendingDown as LossIcon,
  AttachMoney as MoneyIcon,
  CheckCircle as SuccessIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';

interface DirectIndexAccount {
  account_id: string;
  account_name: string;
  total_market_value: number;
  ytd_tax_loss_harvested: number;
  ytd_tax_savings: number;
  ytd_return_pct: number;
  ytd_benchmark_return_pct: number;
  tracking_error_pct: number;
}

interface HarvestOpportunity {
  opportunity_id: string;
  ticker: string;
  shares_to_sell: number;
  current_price: number;
  unrealized_loss: number;
  unrealized_loss_pct: number;
  estimated_tax_savings: number;
  replacement_ticker: string;
  replacement_name: string;
  correlation_with_original: number;
  wash_sale_risk: boolean;
  detected_at: string;
}

export const DirectIndexingDashboard: React.FC = () => {
  const [accounts, setAccounts] = useState<DirectIndexAccount[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<DirectIndexAccount | null>(null);
  const [opportunities, setOpportunities] = useState<HarvestOpportunity[]>([]);
  const [loading, setLoading] = useState(true);
  const [executeDialogOpen, setExecuteDialogOpen] = useState(false);
  const [selectedOpp, setSelectedOpp] = useState<HarvestOpportunity | null>(null);
  const [tabValue, setTabValue] = useState(0);

  useEffect(() => {
    fetchAccounts();
  }, []);

  useEffect(() => {
    if (selectedAccount) {
      fetchOpportunities(selectedAccount.account_id);
    }
  }, [selectedAccount]);

  const fetchAccounts = async () => {
    try {
      const clientId = localStorage.getItem('client_id'); // Get from auth context
      const response = await fetch(`/api/direct-indexing/accounts?client_id=${clientId}`);
      const data = await response.json();
      setAccounts(data.accounts || []);
      if (data.accounts && data.accounts.length > 0) {
        setSelectedAccount(data.accounts[0]);
      }
    } catch (error) {
      console.error('Failed to fetch accounts:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchOpportunities = async (accountId: string) => {
    try {
      const response = await fetch(`/api/direct-indexing/accounts/${accountId}/opportunities?status=PENDING`);
      const data = await response.json();
      setOpportunities(data.opportunities || []);
    } catch (error) {
      console.error('Failed to fetch opportunities:', error);
    }
  };

  const handleExecute = async (opportunity: HarvestOpportunity) => {
    setSelectedOpp(opportunity);
    setExecuteDialogOpen(true);
  };

  const confirmExecute = async () => {
    if (!selectedOpp) return;

    try {
      const userId = localStorage.getItem('user_id');
      await fetch(`/api/direct-indexing/opportunities/${selectedOpp.opportunity_id}/execute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ approved_by: userId }),
      });

      // Refresh opportunities
      if (selectedAccount) {
        await fetchOpportunities(selectedAccount.account_id);
        await fetchAccounts(); // Refresh account metrics
      }

      setExecuteDialogOpen(false);
      setSelectedOpp(null);
    } catch (error) {
      console.error('Failed to execute harvest:', error);
    }
  };

  const handleDismiss = async (opportunityId: string) => {
    try {
      await fetch(`/api/direct-indexing/opportunities/${opportunityId}/dismiss`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: 'NOT_RELEVANT' }),
      });

      // Refresh opportunities
      if (selectedAccount) {
        await fetchOpportunities(selectedAccount.account_id);
      }
    } catch (error) {
      console.error('Failed to dismiss opportunity:', error);
    }
  };

  if (loading) {
    return <LinearProgress />;
  }

  const totalPotentialSavings = opportunities.reduce((sum, opp) => sum + opp.estimated_tax_savings, 0);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Typography variant="h4" gutterBottom>
        Direct Indexing & Tax-Loss Harvesting
      </Typography>
      <Typography variant="body2" color="text.secondary" gutterBottom>
        Automated tax optimization delivering 1.2-2.5% annual tax alpha
      </Typography>

      {/* Account Selector */}
      {accounts.length > 1 && (
        <Paper sx={{ mb: 3 }}>
          <Tabs value={tabValue} onChange={(_, v) => {
            setTabValue(v);
            setSelectedAccount(accounts[v]);
          }}>
            {accounts.map((acc, idx) => (
              <Tab key={acc.account_id} label={acc.account_name} />
            ))}
          </Tabs>
        </Paper>
      )}

      {selectedAccount && (
        <>
          {/* Stats Cards */}
          <Grid container spacing={3} sx={{ mb: 4 }}>
            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <MoneyIcon color="primary" sx={{ mr: 1 }} />
                    <Typography variant="h6">
                      ${selectedAccount.total_market_value.toLocaleString()}
                    </Typography>
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    Account Value
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <LossIcon color="success" sx={{ mr: 1 }} />
                    <Typography variant="h6">
                      ${selectedAccount.ytd_tax_loss_harvested.toLocaleString()}
                    </Typography>
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    YTD Losses Harvested
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <SuccessIcon color="success" sx={{ mr: 1 }} />
                    <Typography variant="h6">
                      ${selectedAccount.ytd_tax_savings.toLocaleString()}
                    </Typography>
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    YTD Tax Savings
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <Typography variant="h6">
                      {selectedAccount.ytd_return_pct?.toFixed(2) || 'N/A'}%
                    </Typography>
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    YTD Return vs {selectedAccount.ytd_benchmark_return_pct?.toFixed(2)}% Benchmark
                  </Typography>
                  {selectedAccount.tracking_error_pct && (
                    <Typography variant="caption" color="text.secondary">
                      Tracking Error: {selectedAccount.tracking_error_pct.toFixed(2)}%
                    </Typography>
                  )}
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          {/* Opportunities Alert */}
          {opportunities.length > 0 && (
            <Alert severity="info" sx={{ mb: 3 }}>
              <Typography variant="subtitle1">
                {opportunities.length} Tax-Loss Harvesting Opportunities Detected
              </Typography>
              <Typography variant="body2">
                Potential tax savings: ${totalPotentialSavings.toLocaleString()}
              </Typography>
            </Alert>
          )}

          {/* Opportunities Table */}
          <Paper>
            <Box sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                Pending Harvest Opportunities
              </Typography>
            </Box>

            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Ticker</TableCell>
                    <TableCell align="right">Shares</TableCell>
                    <TableCell align="right">Current Price</TableCell>
                    <TableCell align="right">Unrealized Loss</TableCell>
                    <TableCell align="right">Loss %</TableCell>
                    <TableCell align="right">Tax Savings</TableCell>
                    <TableCell>Replacement</TableCell>
                    <TableCell align="center">Action</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {opportunities.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={8} align="center">
                        <Box sx={{ py: 4 }}>
                          <SuccessIcon color="success" sx={{ fontSize: 48, mb: 1 }} />
                          <Typography variant="h6">All Caught Up!</Typography>
                          <Typography variant="body2" color="text.secondary">
                            No harvest opportunities detected. System scans daily at 4 PM ET.
                          </Typography>
                        </Box>
                      </TableCell>
                    </TableRow>
                  ) : (
                    opportunities.map((opp) => (
                      <TableRow key={opp.opportunity_id} hover>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            <strong>{opp.ticker}</strong>
                            {opp.wash_sale_risk && (
                              <Chip 
                                label="Wash Sale Risk" 
                                color="warning" 
                                size="small" 
                                sx={{ ml: 1 }}
                                icon={<WarningIcon />}
                              />
                            )}
                          </Box>
                        </TableCell>
                        <TableCell align="right">{opp.shares_to_sell.toFixed(2)}</TableCell>
                        <TableCell align="right">${opp.current_price.toFixed(2)}</TableCell>
                        <TableCell align="right" sx={{ color: 'error.main' }}>
                          ${Math.abs(opp.unrealized_loss).toLocaleString()}
                        </TableCell>
                        <TableCell align="right" sx={{ color: 'error.main' }}>
                          {opp.unrealized_loss_pct.toFixed(2)}%
                        </TableCell>
                        <TableCell align="right" sx={{ color: 'success.main', fontWeight: 'bold' }}>
                          ${opp.estimated_tax_savings.toLocaleString()}
                        </TableCell>
                        <TableCell>
                          <Box>
                            <Typography variant="body2">
                              <strong>{opp.replacement_ticker}</strong>
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {opp.replacement_name}
                            </Typography>
                            {opp.correlation_with_original && (
                              <Typography variant="caption" display="block" color="text.secondary">
                                Correlation: {(opp.correlation_with_original * 100).toFixed(1)}%
                              </Typography>
                            )}
                          </Box>
                        </TableCell>
                        <TableCell align="center">
                          <Box sx={{ display: 'flex', gap: 1, justifyContent: 'center' }}>
                            <Button
                              variant="contained"
                              color="primary"
                              size="small"
                              onClick={() => handleExecute(opp)}
                              disabled={opp.wash_sale_risk}
                            >
                              Execute
                            </Button>
                            <Button
                              variant="outlined"
                              color="secondary"
                              size="small"
                              onClick={() => handleDismiss(opp.opportunity_id)}
                            >
                              Dismiss
                            </Button>
                          </Box>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        </>
      )}

      {/* Execute Confirmation Dialog */}
      <Dialog open={executeDialogOpen} onClose={() => setExecuteDialogOpen(false)} maxWidth="sm" fullWidth>
        {selectedOpp && (
          <>
            <DialogTitle>Confirm Tax-Loss Harvest</DialogTitle>
            <DialogContent>
              <Alert severity="info" sx={{ mb: 2 }}>
                This will execute the following trades:
              </Alert>

              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="subtitle2">SELL</Typography>
                  <Typography variant="body2">
                    {selectedOpp.shares_to_sell.toFixed(2)} shares of {selectedOpp.ticker}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    @ ${selectedOpp.current_price.toFixed(2)}
                  </Typography>
                </Grid>

                <Grid item xs={6}>
                  <Typography variant="subtitle2">BUY</Typography>
                  <Typography variant="body2">
                    {selectedOpp.shares_to_sell.toFixed(2)} shares of {selectedOpp.replacement_ticker}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {selectedOpp.replacement_name}
                  </Typography>
                </Grid>
              </Grid>

              <Box sx={{ mt: 3, p: 2, bgcolor: 'success.light', borderRadius: 1 }}>
                <Typography variant="h6" color="success.dark">
                  Estimated Tax Savings: ${selectedOpp.estimated_tax_savings.toLocaleString()}
                </Typography>
                <Typography variant="body2" color="success.dark">
                  Realized Loss: ${Math.abs(selectedOpp.unrealized_loss).toLocaleString()}
                </Typography>
              </Box>

              <Alert severity="warning" sx={{ mt: 2 }}>
                A 30-day wash sale window will be created for {selectedOpp.ticker}
              </Alert>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setExecuteDialogOpen(false)}>Cancel</Button>
              <Button onClick={confirmExecute} variant="contained" color="primary">
                Confirm Execute
              </Button>
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  );
};
