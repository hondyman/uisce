import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Typography,
  Tab,
  Tabs,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Button,
  CircularProgress,
  Alert,
  TextField,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import FileUploadIcon from '@mui/icons-material/FileUpload';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import { format } from 'date-fns';

interface CashBalance {
  cash_balance_id: string;
  portfolio_id: string;
  currency: string;
  valuation_date: string;
  opening_balance: number;
  cash_inflows: number;
  cash_outflows: number;
  interest_accrual: number;
  fx_effect: number;
  closing_balance: number;
  source_system: string;
  is_closed: boolean;
}

interface CashLedgerEntry {
  cash_ledger_id: string;
  portfolio_id: string;
  currency: string;
  value_date: string;
  cash_event_type: 'SETTLEMENT' | 'INCOME' | 'FEE' | 'FX' | 'CONTRIBUTION' | 'WITHDRAWAL';
  amount: number;
  transaction_id?: string;
  security_id?: string;
  status: 'PENDING' | 'POSTED' | 'CANCELLED';
  source_system: string;
}

const fetchBalances = async (portfolioId: string): Promise<CashBalance[]> => {
  const url = portfolioId ? `/api/v1/cash/balances?portfolio_id=${portfolioId}` : '/api/v1/cash/balances';
  const response = await fetch(url);
  if (!response.ok) throw new Error('Failed to load cash balances');
  const result = await response.json();
  return result.data || [];
};

const fetchLedger = async (portfolioId: string): Promise<CashLedgerEntry[]> => {
  const url = portfolioId ? `/api/v1/cash/ledger?portfolio_id=${portfolioId}` : '/api/v1/cash/ledger';
  const response = await fetch(url);
  if (!response.ok) throw new Error('Failed to load cash ledger');
  const result = await response.json();
  return result.data || [];
};

const ingestDemoLedger = async (): Promise<void> => {
  const payload = [
    {
      portfolio_id: "00000000-0000-0000-0000-000000000001",
      currency: "USD",
      value_date: new Date().toISOString(),
      cash_event_type: "CONTRIBUTION",
      amount: 500000,
      source_system: "TreasurySystem",
      external_reference: `CASH-${Date.now()}`,
      status: "POSTED"
    }
  ];
  const response = await fetch('/api/v1/cash/ledger/ingest', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!response.ok) throw new Error('Failed to ingest demo cash ledger');
};

const rollForwardBalance = async (portfolioId: string, currency: string): Promise<void> => {
  const today = new Date().toISOString().split('T')[0];
  const url = `/api/v1/cash/balances/rollforward?portfolio_id=${portfolioId}&currency=${currency}&date=${today}`;
  const response = await fetch(url, { method: 'POST' });
  if (!response.ok) throw new Error('Failed to roll-forward cash balance');
};

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}
function CustomTabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const CashMasterDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [portfolioFilter, setPortfolioFilter] = useState('');
  const queryClient = useQueryClient();

  const { data: balances, isLoading: loadingBalances, error: balancesError } = useQuery({
    queryKey: ['cash-balances', portfolioFilter],
    queryFn: () => fetchBalances(portfolioFilter),
  });

  const { data: ledger, isLoading: loadingLedger, error: ledgerError } = useQuery({
    queryKey: ['cash-ledger', portfolioFilter],
    queryFn: () => fetchLedger(portfolioFilter),
  });

  const ingestMutation = useMutation({
    mutationFn: ingestDemoLedger,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cash-ledger'] });
    },
  });

  const rollForwardMutation = useMutation({
    mutationFn: () => rollForwardBalance("00000000-0000-0000-0000-000000000001", "USD"), // Using default ID for demo
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cash-balances'] });
    },
  });

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'POSTED': return 'success';
      case 'PENDING': return 'warning';
      case 'CANCELLED': return 'error';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ width: '100%', p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Cash Master
        </Typography>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
          <TextField
            size="small"
            placeholder="Filter by portfolio..."
            value={portfolioFilter}
            onChange={(e) => setPortfolioFilter(e.target.value)}
          />
          <Button 
            variant="outlined" 
            startIcon={<RefreshIcon />}
            onClick={() => {
              queryClient.invalidateQueries({ queryKey: ['cash-balances'] });
              queryClient.invalidateQueries({ queryKey: ['cash-ledger'] });
            }}
          >
            Refresh
          </Button>
          <Button 
            variant="outlined" 
            startIcon={<PlayArrowIcon />}
            onClick={() => rollForwardMutation.mutate()}
            disabled={rollForwardMutation.isPending}
            color="primary"
          >
            Run Roll-Forward
          </Button>
          <Button 
            variant="contained" 
            startIcon={<FileUploadIcon />}
            onClick={() => ingestMutation.mutate()}
            disabled={ingestMutation.isPending}
          >
            Ingest Demo Ledger
          </Button>
        </Box>
      </Box>
      
      {(balancesError || ledgerError) && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Error loading cash data: {((balancesError || ledgerError) as Error).message}
        </Alert>
      )}

      {rollForwardMutation.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Failed to compute balance roll-forward
        </Alert>
      )}

      {ingestMutation.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Failed to ingest demo ledger
        </Alert>
      )}

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="cash master tabs">
          <Tab label="Balances" />
          <Tab label="Ledger" />
          <Tab label="Reconciliation" />
          <Tab label="Cash Flow Analysis" />
        </Tabs>
      </Box>

      {/* Balances Tab */}
      <CustomTabPanel value={activeTab} index={0}>
        <TableContainer component={Paper} elevation={0} variant="outlined">
          <Table sx={{ minWidth: 650 }} size="small">
            <TableHead sx={{ bgcolor: 'grey.50' }}>
              <TableRow>
                <TableCell>Date</TableCell>
                <TableCell>Currency</TableCell>
                <TableCell align="right">Opening</TableCell>
                <TableCell align="right">Inflows</TableCell>
                <TableCell align="right">Outflows</TableCell>
                <TableCell align="right">Interest</TableCell>
                <TableCell align="right">FX Effect</TableCell>
                <TableCell align="right">Closing</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loadingBalances ? (
                <TableRow><TableCell colSpan={9} align="center" sx={{ py: 3 }}><CircularProgress size={24} /></TableCell></TableRow>
              ) : !balances || balances.length === 0 ? (
                <TableRow><TableCell colSpan={9} align="center" sx={{ py: 3, color: 'text.secondary' }}>No balances found</TableCell></TableRow>
              ) : (
                balances.map((row) => (
                  <TableRow key={row.cash_balance_id} hover>
                    <TableCell>{format(new Date(row.valuation_date), 'yyyy-MM-dd')}</TableCell>
                    <TableCell sx={{ fontWeight: 'bold' }}>{row.currency}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace' }}>{row.opening_balance?.toLocaleString()}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace', color: 'success.main' }}>{row.cash_inflows?.toLocaleString()}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace', color: 'error.main' }}>{row.cash_outflows?.toLocaleString()}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace' }}>{row.interest_accrual?.toLocaleString()}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace' }}>{row.fx_effect?.toLocaleString()}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 'bold' }}>{row.closing_balance?.toLocaleString()}</TableCell>
                    <TableCell>
                      <Chip label={row.is_closed ? "Closed" : "Open"} size="small" variant={row.is_closed ? "filled" : "outlined"} />
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </CustomTabPanel>

      {/* Ledger Tab */}
      <CustomTabPanel value={activeTab} index={1}>
        <TableContainer component={Paper} elevation={0} variant="outlined">
          <Table sx={{ minWidth: 650 }} size="small">
            <TableHead sx={{ bgcolor: 'grey.50' }}>
              <TableRow>
                <TableCell>Date</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Currency</TableCell>
                <TableCell align="right">Amount</TableCell>
                <TableCell>Transaction</TableCell>
                <TableCell>Security</TableCell>
                <TableCell>Source</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loadingLedger ? (
                <TableRow><TableCell colSpan={8} align="center" sx={{ py: 3 }}><CircularProgress size={24} /></TableCell></TableRow>
              ) : !ledger || ledger.length === 0 ? (
                <TableRow><TableCell colSpan={8} align="center" sx={{ py: 3, color: 'text.secondary' }}>No ledger entries found</TableCell></TableRow>
              ) : (
                ledger.map((row) => (
                  <TableRow key={row.cash_ledger_id} hover>
                    <TableCell>{format(new Date(row.value_date), 'yyyy-MM-dd')}</TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight="bold" color={(row.cash_event_type === 'INCOME' || row.cash_event_type === 'CONTRIBUTION') ? 'success.main' : 'error.main'}>
                        {row.cash_event_type}
                      </Typography>
                    </TableCell>
                    <TableCell>{row.currency}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace', color: row.amount < 0 ? 'error.main' : 'success.main' }}>
                      {row.amount?.toLocaleString()}
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{row.transaction_id?.substring(0, 8) || '-'}</TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{row.security_id?.substring(0, 8) || '-'}</TableCell>
                    <TableCell>
                      <Chip label={row.source_system} size="small" variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Chip label={row.status} size="small" color={getStatusColor(row.status) as any} />
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </CustomTabPanel>

      {/* Reconciliation Tab */}
      <CustomTabPanel value={activeTab} index={2}>
        <Paper elevation={0} variant="outlined" sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body1">Cash Reconciliation vs. Custodian implementation pending (Phase 15).</Typography>
        </Paper>
      </CustomTabPanel>

      {/* Analysis Tab */}
      <CustomTabPanel value={activeTab} index={3}>
        <Paper elevation={0} variant="outlined" sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body1">Cash Flow Waterfall Visualization pending.</Typography>
        </Paper>
      </CustomTabPanel>
    </Box>
  );
};

export default CashMasterDashboard;
