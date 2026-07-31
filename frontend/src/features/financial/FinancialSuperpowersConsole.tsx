import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Tabs,
  Tab,
  Grid,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip,
  Alert,
  Card,
  CardContent,
  CircularProgress
} from '@mui/material';
import AccountBalanceIcon from '@mui/icons-material/AccountBalance';
import SecurityIcon from '@mui/icons-material/Security';
import MonetizationOnIcon from '@mui/icons-material/MonetizationOn';
import HubIcon from '@mui/icons-material/Hub';
import SmartToyIcon from '@mui/icons-material/SmartToy';

export const FinancialSuperpowersConsole: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(false);

  // Tab 0: Symbology
  const [symbologyInput, setSymbologyInput] = useState('US0378331005');
  const [symbologyResult, setSymbologyResult] = useState<any>(null);

  // Tab 1: Compliance
  const [order, setOrder] = useState({ symbol: 'AAPL', quantity: 1000, price: 220, amount: 220000 });
  const [complianceResult, setComplianceResult] = useState<any>(null);

  // Tab 2: IBOR/ABOR Posting
  const [postingResult, setPostingResult] = useState<any>(null);

  // Tab 3: Household TLH
  const [householdId, setHouseholdId] = useState('HH-SMITH-FAMILY');
  const [tlhResults, setTlhResults] = useState<any[]>([]);

  const handleResolveSymbology = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/financial/instrument-master/resolve', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ identifier_type: 'ISIN', identifier_value: symbologyInput }),
      });
      const data = await res.json();
      setSymbologyResult(data);
    } finally {
      setLoading(false);
    }
  };

  const handleTestCompliance = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/financial/compliance/evaluate-trade', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          portfolio_id: 'PORT-991',
          symbol: order.symbol,
          order_type: 'BUY',
          quantity: Number(order.quantity),
          limit_price: Number(order.price),
          estimated_amount: Number(order.amount),
        }),
      });
      const data = await res.json();
      setComplianceResult(data);
    } finally {
      setLoading(false);
    }
  };

  const handleExecutePosting = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/financial/posting/execute-trade-post', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ portfolio_id: 'PORT-991', symbol: 'AAPL', quantity: 1000, price: 220.0, asset_class: 'EQUITY' }),
      });
      const data = await res.json();
      setPostingResult(data);
    } finally {
      setLoading(false);
    }
  };

  const handleOptimizeHousehold = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/financial/household/optimize-harvesting?household_id=${householdId}`);
      const data = await res.json();
      setTlhResults(data.opportunities || []);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#0f172a', minHeight: '100vh', color: '#f8fafc' }}>
      <Box display="flex" alignItems="center" gap={1.5} mb={1}>
        <AccountBalanceIcon sx={{ color: '#38bdf8', fontSize: 36 }} />
        <Typography variant="h4" fontWeight="700">
          Financial Superpowers Operations Suite
        </Typography>
      </Box>
      <Typography variant="body1" color="#94a3b8" mb={3}>
        Institutional-Grade Core Engine: Instrument Master (EDM), Pre-Trade Compliance (CRD), IBOR/ABOR Postings (Aladdin), Household TLH & Native MCP AI Agents.
      </Typography>

      <Paper sx={{ bgcolor: '#1e293b', border: '1px solid #334155', mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, val) => setActiveTab(val)}
          textColor="inherit"
          indicatorColor="primary"
          sx={{ borderBottom: '1px solid #334155' }}
        >
          <Tab icon={<HubIcon />} iconPosition="start" label="1. Instrument Master (EDM)" />
          <Tab icon={<SecurityIcon />} iconPosition="start" label="2. Pre-Trade Compliance (CRD)" />
          <Tab icon={<MonetizationOnIcon />} iconPosition="start" label="3. IBOR / ABOR Posting (Aladdin)" />
          <Tab icon={<AccountBalanceIcon />} iconPosition="start" label="4. Household TLH (WealthTech)" />
          <Tab icon={<SmartToyIcon />} iconPosition="start" label="5. Native MCP Server Tools" />
        </Tabs>

        {/* Tab 0: Instrument Master */}
        {activeTab === 0 && (
          <Box p={3}>
            <Typography variant="h6" fontWeight="600" mb={2}>
              Multi-Feed Symbology Resolver & EDM Survivorship Priority Engine
            </Typography>
            <Grid container spacing={2} mb={3}>
              <Grid size={{ xs: 12, sm: 8 }}>
                <TextField
                  label="Search Identifier (ISIN / CUSIP / SEDOL / FIGI / Ticker)"
                  fullWidth
                  size="small"
                  value={symbologyInput}
                  onChange={(e) => setSymbologyInput(e.target.value)}
                  sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
                />
              </Grid>
              <Grid size={{ xs: 12, sm: 4 }}>
                <Button variant="contained" fullWidth onClick={handleResolveSymbology} disabled={loading} sx={{ bgcolor: '#0284c7' }}>
                  Resolve Instrument Symbology
                </Button>
              </Grid>
            </Grid>

            {symbologyResult && (
              <Card sx={{ bgcolor: '#0f172a', border: '1px solid #38bdf8', color: '#fff' }}>
                <CardContent>
                  <Typography variant="subtitle1" fontWeight="700" color="#38bdf8" mb={1}>
                    {symbologyResult.instrument_name} ({symbologyResult.primary_ticker})
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid size={{ xs: 6, sm: 3 }}><Typography variant="caption" color="#94a3b8">ISIN:</Typography> <Typography fontWeight="600">{symbologyResult.isin}</Typography></Grid>
                    <Grid size={{ xs: 6, sm: 3 }}><Typography variant="caption" color="#94a3b8">CUSIP:</Typography> <Typography fontWeight="600">{symbologyResult.cusip}</Typography></Grid>
                    <Grid size={{ xs: 6, sm: 3 }}><Typography variant="caption" color="#94a3b8">SEDOL:</Typography> <Typography fontWeight="600">{symbologyResult.sedol}</Typography></Grid>
                    <Grid size={{ xs: 6, sm: 3 }}><Typography variant="caption" color="#94a3b8">FIGI:</Typography> <Typography fontWeight="600">{symbologyResult.figi}</Typography></Grid>
                  </Grid>
                </CardContent>
              </Card>
            )}
          </Box>
        )}

        {/* Tab 1: Pre-Trade Compliance */}
        {activeTab === 1 && (
          <Box p={3}>
            <Typography variant="h6" fontWeight="600" mb={2}>
              Declarative Pre-Trade Regulatory & Concentration Rule Inspector
            </Typography>
            <Grid container spacing={2} mb={3}>
              <Grid size={{ xs: 3 }}>
                <TextField label="Symbol" size="small" fullWidth value={order.symbol} onChange={(e) => setOrder({ ...order, symbol: e.target.value })} sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }} />
              </Grid>
              <Grid size={{ xs: 3 }}>
                <TextField label="Quantity" size="small" fullWidth type="number" value={order.quantity} onChange={(e) => setOrder({ ...order, quantity: Number(e.target.value) })} sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }} />
              </Grid>
              <Grid size={{ xs: 3 }}>
                <TextField label="Price ($)" size="small" fullWidth type="number" value={order.price} onChange={(e) => setOrder({ ...order, price: Number(e.target.value) })} sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }} />
              </Grid>
              <Grid size={{ xs: 3 }}>
                <Button variant="contained" fullWidth onClick={handleTestCompliance} disabled={loading} sx={{ bgcolor: '#0284c7' }}>
                  Evaluate Pre-Trade Compliance
                </Button>
              </Grid>
            </Grid>

            {complianceResult && (
              <Box>
                {complianceResult.passed ? (
                  <Alert severity="success" sx={{ mb: 2 }}>Trade Passed Pre-Trade Compliance Check cleanly!</Alert>
                ) : (
                  <Alert severity="error" sx={{ mb: 2 }}>TRADE BLOCKED: Pre-Trade Compliance Violation Detected!</Alert>
                )}
                {complianceResult.violations?.map((v: string, i: number) => (
                  <Alert key={i} severity="error" sx={{ mb: 1 }}>{v}</Alert>
                ))}
                {complianceResult.warnings?.map((w: string, i: number) => (
                  <Alert key={i} severity="warning" sx={{ mb: 1 }}>{w}</Alert>
                ))}
              </Box>
            )}
          </Box>
        )}

        {/* Tab 2: IBOR / ABOR Posting */}
        {activeTab === 2 && (
          <Box p={3}>
            <Typography variant="h6" fontWeight="600" mb={2}>
              IBOR (Investment) & ABOR (Accounting Book of Record) Double-Entry Posting Preview
            </Typography>
            <Button variant="contained" onClick={handleExecutePosting} disabled={loading} sx={{ bgcolor: '#0284c7', mb: 3 }}>
              Execute Sample Trade Posting (1000 AAPL @ $220.00)
            </Button>

            {postingResult && (
              <Grid container spacing={3}>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #38bdf8' }}>
                    <Typography variant="subtitle2" color="#38bdf8" fontWeight="600" mb={1}>IBOR Ledger (Investment Book of Record)</Typography>
                    <Table size="small">
                      <TableHead><TableRow><TableCell sx={{ color: '#94a3b8' }}>Account</TableCell><TableCell sx={{ color: '#94a3b8' }}>Debit</TableCell><TableCell sx={{ color: '#94a3b8' }}>Credit</TableCell></TableRow></TableHead>
                      <TableBody>
                        {postingResult.ibor_postings?.map((entry: any, i: number) => (
                          <TableRow key={i}>
                            <TableCell sx={{ color: '#fff' }}>{entry.account_name}</TableCell>
                            <TableCell sx={{ color: '#4ade80' }}>${entry.debit.toLocaleString()}</TableCell>
                            <TableCell sx={{ color: '#f43f5e' }}>${entry.credit.toLocaleString()}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </Paper>
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #a855f7' }}>
                    <Typography variant="subtitle2" color="#a855f7" fontWeight="600" mb={1}>ABOR Ledger (Accounting Book of Record)</Typography>
                    <Table size="small">
                      <TableHead><TableRow><TableCell sx={{ color: '#94a3b8' }}>Account</TableCell><TableCell sx={{ color: '#94a3b8' }}>Debit</TableCell><TableCell sx={{ color: '#94a3b8' }}>Credit</TableCell></TableRow></TableHead>
                      <TableBody>
                        {postingResult.abor_postings?.map((entry: any, i: number) => (
                          <TableRow key={i}>
                            <TableCell sx={{ color: '#fff' }}>{entry.account_name}</TableCell>
                            <TableCell sx={{ color: '#4ade80' }}>${entry.debit.toLocaleString()}</TableCell>
                            <TableCell sx={{ color: '#f43f5e' }}>${entry.credit.toLocaleString()}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </Paper>
                </Grid>
              </Grid>
            )}
          </Box>
        )}

        {/* Tab 3: Household TLH */}
        {activeTab === 3 && (
          <Box p={3}>
            <Typography variant="h6" fontWeight="600" mb={2}>
              Household Graph Tax-Loss Harvesting & Wash-Sale Safe Optimizer
            </Typography>
            <Grid container spacing={2} mb={3}>
              <Grid size={{ xs: 8 }}>
                <TextField label="Target Household ID" fullWidth size="small" value={householdId} onChange={(e) => setHouseholdId(e.target.value)} sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }} />
              </Grid>
              <Grid size={{ xs: 4 }}>
                <Button variant="contained" fullWidth onClick={handleOptimizeHousehold} disabled={loading} sx={{ bgcolor: '#0284c7' }}>
                  Scan Household Graph for TLH
                </Button>
              </Grid>
            </Grid>

            {tlhResults.length > 0 && (
              <Table size="small">
                <TableHead sx={{ bgcolor: '#0f172a' }}>
                  <TableRow>
                    <TableCell sx={{ color: '#94a3b8' }}>Target Symbol</TableCell>
                    <TableCell sx={{ color: '#94a3b8' }}>Harvestable Loss</TableCell>
                    <TableCell sx={{ color: '#94a3b8' }}>Wash-Sale Substitute</TableCell>
                    <TableCell sx={{ color: '#94a3b8' }}>Recommended Action</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {tlhResults.map((opp, i) => (
                    <TableRow key={i}>
                      <TableCell sx={{ color: '#f43f5e', fontWeight: 600 }}>{opp.target_symbol}</TableCell>
                      <TableCell sx={{ color: '#4ade80', fontWeight: 600 }}>${opp.harvestable_loss_usd?.toLocaleString()}</TableCell>
                      <TableCell sx={{ color: '#38bdf8', fontWeight: 600 }}>{opp.substitute_symbol}</TableCell>
                      <TableCell sx={{ color: '#fff' }}>{opp.recommended_action}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Box>
        )}

        {/* Tab 4: MCP Server */}
        {activeTab === 4 && (
          <Box p={3}>
            <Typography variant="h6" fontWeight="600" mb={2}>
              Native Model Context Protocol (MCP) Server Tool Registrations
            </Typography>
            <Alert severity="info" sx={{ mb: 2 }}>
              Autonomous AI agents connect via MCP to dynamically discover Business Objects, run pre-trade compliance checks, simulate IBOR postings, and execute household tax-loss harvesting.
            </Alert>
            <Grid container spacing={2}>
              <Grid size={{ xs: 12, sm: 6 }}>
                <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155' }}>
                  <Typography variant="subtitle2" color="#38bdf8" fontWeight="600">resolve_instrument_symbology</Typography>
                  <Typography variant="caption" color="#94a3b8">Market EDM lookup across CUSIP, ISIN, SEDOL, FIGI with priority rules.</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155' }}>
                  <Typography variant="subtitle2" color="#38bdf8" fontWeight="600">evaluate_pretrade_compliance</Typography>
                  <Typography variant="caption" color="#94a3b8">CRD pre-trade compliance checks for sector concentration & restricted lists.</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155' }}>
                  <Typography variant="subtitle2" color="#38bdf8" fontWeight="600">post_ibor_abor_transaction</Typography>
                  <Typography variant="caption" color="#94a3b8">Aladdin double-entry ledger postings for IBOR and ABOR views.</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155' }}>
                  <Typography variant="subtitle2" color="#38bdf8" fontWeight="600">optimize_household_tax_harvesting</Typography>
                  <Typography variant="caption" color="#94a3b8">WealthTech household graph optimizer for wash-sale safe tax loss harvesting.</Typography>
                </Paper>
              </Grid>
            </Grid>
          </Box>
        )}
      </Paper>
    </Box>
  );
};
