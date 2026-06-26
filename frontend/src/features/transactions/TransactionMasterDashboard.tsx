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
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import FileUploadIcon from '@mui/icons-material/FileUpload';
import { format } from 'date-fns';

// Minimal types for the dashboard
interface TransactionRecord {
  transaction_id: string;
  portfolio_id: string;
  security_id?: string;
  trade_date: string;
  settlement_date?: string;
  transaction_type: string;
  quantity?: string;
  price?: string;
  gross_amount?: string;
  transaction_currency: string;
  status: string;
  source_system: string;
  external_reference?: string;
}

const fetchTransactions = async (): Promise<TransactionRecord[]> => {
  const response = await fetch('/api/v1/transactions');
  if (!response.ok) {
    throw new Error('Failed to load transactions');
  }
  const result = await response.json();
  return result.data || [];
};

const ingestDemoTransactions = async (): Promise<void> => {
	// Send fake payload representing demo ingest
  const payload = [
    {
      portfolio_id: "00000000-0000-0000-0000-000000000001",
      trade_date: new Date().toISOString(),
      transaction_type: "BUY",
      quantity: "150",
      price: "185.25",
      transaction_currency: "USD",
      source_system: "OMS",
      external_reference: `DEMO-${Date.now()}`,
      status: "EXECUTED"
    }
  ];
  
  const response = await fetch('/api/v1/transactions/ingest', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  
  if (!response.ok) {
    throw new Error('Failed to ingest demo transactions');
  }
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
      id={`tx-tabpanel-${index}`}
      aria-labelledby={`tx-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ p: 3 }}>
          {children}
        </Box>
      )}
    </div>
  );
}

const TransactionMasterDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const queryClient = useQueryClient();

  const { data: transactions, isLoading, error } = useQuery({
    queryKey: ['transactions'],
    queryFn: fetchTransactions,
  });

  const mutation = useMutation({
    mutationFn: ingestDemoTransactions,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
    },
  });

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'SETTLED': return 'success';
      case 'EXECUTED': return 'warning';
      case 'CANCELLED': return 'error';
      default: return 'default';
    }
  };

  const trades = transactions?.filter(t => ['BUY', 'SELL', 'SHORT', 'COVER'].includes(t.transaction_type)) || [];
  const corpActions = transactions?.filter(t => ['DIVIDEND', 'SPLIT', 'MERGER'].includes(t.transaction_type)) || [];
  const cashFlows = transactions?.filter(t => ['CONTRIBUTION', 'WITHDRAWAL', 'FEE', 'INTEREST'].includes(t.transaction_type)) || [];

  return (
    <Box sx={{ width: '100%', p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Transaction Master
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button 
            variant="outlined" 
            startIcon={<RefreshIcon />}
            onClick={() => queryClient.invalidateQueries({ queryKey: ['transactions'] })}
            disabled={isLoading || mutation.isPending}
          >
            Refresh
          </Button>
          <Button 
            variant="contained" 
            startIcon={<FileUploadIcon />}
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
          >
            Ingest Demo Trade
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Error loading transactions: {(error as Error).message}
        </Alert>
      )}
      
      {mutation.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Failed to ingest demo transaction
        </Alert>
      )}

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={activeTab} onChange={handleTabChange} aria-label="transaction master tabs">
          <Tab label={`Trades (${trades.length})`} />
          <Tab label={`Corporate Actions (${corpActions.length})`} />
          <Tab label={`Cash Flows (${cashFlows.length})`} />
          <Tab label="Reconciliation" />
        </Tabs>
      </Box>

      {/* Trades Tab */}
      <CustomTabPanel value={activeTab} index={0}>
        <TableContainer component={Paper} elevation={0} variant="outlined">
          <Table sx={{ minWidth: 650 }} size="small">
            <TableHead sx={{ bgcolor: 'grey.50' }}>
              <TableRow>
                <TableCell>Trade Date</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Security ID</TableCell>
                <TableCell align="right">Quantity</TableCell>
                <TableCell align="right">Price</TableCell>
                <TableCell align="right">Gross Amt</TableCell>
                <TableCell>Ccy</TableCell>
                <TableCell>Source</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={9} align="center" sx={{ py: 3 }}>
                    <CircularProgress size={24} />
                  </TableCell>
                </TableRow>
              ) : trades.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} align="center" sx={{ py: 3, color: 'text.secondary' }}>
                    No trades found
                  </TableCell>
                </TableRow>
              ) : (
                trades.map((row) => (
                  <TableRow key={row.transaction_id} hover>
                    <TableCell>{format(new Date(row.trade_date), 'yyyy-MM-dd HH:mm')}</TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight="bold" color={row.transaction_type === 'BUY' ? 'success.main' : 'error.main'}>
                        {row.transaction_type}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                      {row.security_id?.substring(0, 8)}...
                    </TableCell>
                    <TableCell align="right">{row.quantity || '-'}</TableCell>
                    <TableCell align="right">{row.price || '-'}</TableCell>
                    <TableCell align="right">{row.gross_amount || '-'}</TableCell>
                    <TableCell>{row.transaction_currency}</TableCell>
                    <TableCell>
                      <Chip label={row.source_system} size="small" variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={row.status} 
                        size="small" 
                        color={getStatusColor(row.status)}
                      />
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </CustomTabPanel>

      {/* Corporate Actions Tab */}
      <CustomTabPanel value={activeTab} index={1}>
        <Paper elevation={0} variant="outlined" sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body1">Corporate Actions processing implementation pending (Phase 13+).</Typography>
          <Typography variant="body2" sx={{ mt: 1 }}>{corpActions.length} actions recorded.</Typography>
        </Paper>
      </CustomTabPanel>

      {/* Cash Flows Tab */}
      <CustomTabPanel value={activeTab} index={2}>
        <Paper elevation={0} variant="outlined" sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body1">Cash Flow lifecycle implementation pending.</Typography>
          <Typography variant="body2" sx={{ mt: 1 }}>{cashFlows.length} flows recorded.</Typography>
        </Paper>
      </CustomTabPanel>
      
      {/* Reconciliation Tab */}
      <CustomTabPanel value={activeTab} index={3}>
        <Paper elevation={0} variant="outlined" sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body1">Reconciliation exceptions and Gold Copy trace viewer pending.</Typography>
        </Paper>
      </CustomTabPanel>
    </Box>
  );
};

export default TransactionMasterDashboard;
