import React, { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip,
  Button,
  CircularProgress,
  Alert,
} from '@mui/material';
import { TrendingUp, TrendingDown, AccountBalance, Refresh } from '@mui/icons-material';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div hidden={value !== index} {...other}>
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

interface CryptoWallet {
  id: string;
  blockchain: string;
  address: string;
  label?: string;
  custodian: string;
}

interface CryptoHolding {
  assetSymbol: string;
  assetName?: string;
  quantity: number;
  costBasisTotal: number;
  averageCostPerUnit: number;
  currentPrice?: number;
  change24h?: number;
}

interface DeFiPosition {
  protocol: string;
  positionType: string;
  assetDeposited: string;
  quantityDeposited: number;
  currentValueUsd?: number;
  apr?: number;
  rewardsEarned: number;
  unclaimedRewardsUsd?: number;
}

interface CryptoTransaction {
  id: string;
  txnType: string;
  assetSymbol: string;
  quantity: number;
  fiatValueUsd?: number;
  blockTimestamp?: string;
  status: string;
  txnHash?: string;
}

export const CryptoPortfolioCenter: React.FC<{ clientId: string }> = ({ clientId }) => {
  const [currentTab, setCurrentTab] = useState(0);
  const [selectedWallet, setSelectedWallet] = useState<string | null>(null);

  // Fetch wallets
  const { data: wallets, isLoading: walletsLoading } = useQuery<CryptoWallet[]>({
    queryKey: ['crypto-wallets', clientId],
    queryFn: async () => {
      const res = await fetch(`/api/v1/crypto/wallets?client_id=${clientId}`);
      if (!res.ok) throw new Error('Failed to fetch wallets');
      return res.json();
    },
  });

  // Fetch holdings for selected wallet
  const { data: holdings, isLoading: holdingsLoading } = useQuery<CryptoHolding[]>({
    queryKey: ['crypto-holdings', selectedWallet],
    enabled: !!selectedWallet,
    queryFn: async () => {
      const res = await fetch(`/api/v1/crypto/wallets/${selectedWallet}/balances`);
      if (!res.ok) throw new Error('Failed to fetch holdings');
      return res.json();
    },
  });

  // Fetch DeFi positions
  const { data: defiPositions } = useQuery<DeFiPosition[]>({
    queryKey: ['defi-positions', selectedWallet],
    enabled: !!selectedWallet,
    queryFn: async () => {
      const res = await fetch(`/api/v1/crypto/defi/positions?wallet_id=${selectedWallet}`);
      if (!res.ok) return [];
      return res.json();
    },
  });

  // Fetch transactions
  const { data: transactions } = useQuery<CryptoTransaction[]>({
    queryKey: ['crypto-transactions', selectedWallet],
    enabled: !!selectedWallet,
    queryFn: async () => {
      const res = await fetch(`/api/v1/crypto/wallets/${selectedWallet}/transactions?limit=50`);
      if (!res.ok) return [];
      return res.json();
    },
  });

  // Calculate portfolio metrics
  const portfolioMetrics = useMemo(() => {
    if (!holdings) return null;

    const totalValue = holdings.reduce((sum, h) => {
      const value = h.currentPrice ? h.quantity * h.currentPrice : 0;
      return sum + value;
    }, 0);

    const totalCost = holdings.reduce((sum, h) => sum + h.costBasisTotal, 0);
    const unrealizedGain = totalValue - totalCost;
    const unrealizedGainPct = totalCost > 0 ? (unrealizedGain / totalCost) * 100 : 0;

    return {
      totalValue,
      totalCost,
      unrealizedGain,
      unrealizedGainPct,
    };
  }, [holdings]);

  // Select first wallet by default
  React.useEffect(() => {
    if (wallets && wallets.length > 0 && !selectedWallet) {
      setSelectedWallet(wallets[0].id);
    }
  }, [wallets, selectedWallet]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setCurrentTab(newValue);
  };

  if (walletsLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (!wallets || wallets.length === 0) {
    return (
      <Box p={3}>
        <Alert severity="info">
          No crypto wallets connected. Add a wallet to get started.
        </Alert>
        <Button variant="contained" sx={{ mt: 2 }}>
          Connect Wallet
        </Button>
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">Crypto Portfolio</Typography>
        <Button startIcon={<Refresh />} variant="outlined">
          Sync Prices
        </Button>
      </Box>

      {/* Wallet Selector */}
      <Box mb={3}>
        <Typography variant="subtitle2" color="textSecondary" gutterBottom>
          Select Wallet
        </Typography>
        <Grid container spacing={2}>
          {wallets.map((wallet) => (
            <Grid item xs={12} md={6} lg={4} key={wallet.id}>
              <Card
                variant={selectedWallet === wallet.id ? 'outlined' : 'elevation'}
                sx={{
                  cursor: 'pointer',
                  borderColor: selectedWallet === wallet.id ? 'primary.main' : undefined,
                  borderWidth: selectedWallet === wallet.id ? 2 : 1,
                }}
                onClick={() => setSelectedWallet(wallet.id)}
              >
                <CardContent>
                  <Box display="flex" alignItems="center" gap={1} mb={1}>
                    <AccountBalance fontSize="small" color="action" />
                    <Typography variant="subtitle1">{wallet.label || wallet.blockchain}</Typography>
                  </Box>
                  <Typography variant="caption" color="textSecondary">
                    {wallet.custodian} • {wallet.blockchain}
                  </Typography>
                  <Typography variant="caption" display="block" sx={{ mt: 0.5, wordBreak: 'break-all' }}>
                    {wallet.address.substring(0, 12)}...{wallet.address.substring(wallet.address.length - 8)}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      </Box>

      {/* Portfolio Summary Cards */}
      {portfolioMetrics && (
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Total Value
                </Typography>
                <Typography variant="h5" fontWeight="bold">
                  ${portfolioMetrics.totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Cost Basis
                </Typography>
                <Typography variant="h5" fontWeight="bold">
                  ${portfolioMetrics.totalCost.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Unrealized Gain/Loss
                </Typography>
                <Box display="flex" alignItems="center" gap={1}>
                  <Typography
                    variant="h5"
                    fontWeight="bold"
                    color={portfolioMetrics.unrealizedGain >= 0 ? 'success.main' : 'error.main'}
                  >
                    {portfolioMetrics.unrealizedGain >= 0 ? '+' : ''}
                    ${portfolioMetrics.unrealizedGain.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                  </Typography>
                  {portfolioMetrics.unrealizedGain >= 0 ? (
                    <TrendingUp color="success" />
                  ) : (
                    <TrendingDown color="error" />
                  )}
                </Box>
                <Typography variant="caption" color="textSecondary">
                  {portfolioMetrics.unrealizedGainPct >= 0 ? '+' : ''}
                  {portfolioMetrics.unrealizedGainPct.toFixed(2)}%
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={3}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="textSecondary" variant="body2" gutterBottom>
                  Assets Held
                </Typography>
                <Typography variant="h5" fontWeight="bold">
                  {holdings?.length || 0}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Tabbed Interface */}
      <Card>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={currentTab} onChange={handleTabChange}>
            <Tab label="Holdings" />
            <Tab label="DeFi Positions" />
            <Tab label="Transactions" />
            <Tab label="Tax Lots" />
          </Tabs>
        </Box>

        <TabPanel value={currentTab} index={0}>
          <HoldingsTable holdings={holdings || []} loading={holdingsLoading} />
        </TabPanel>

        <TabPanel value={currentTab} index={1}>
          <DeFiPositionsTable positions={defiPositions || []} />
        </TabPanel>

        <TabPanel value={currentTab} index={2}>
          <TransactionsTable transactions={transactions || []} />
        </TabPanel>

        <TabPanel value={currentTab} index={3}>
          <TaxLotsView walletId={selectedWallet} />
        </TabPanel>
      </Card>
    </Box>
  );
};

// Holdings Table Component
const HoldingsTable: React.FC<{ holdings: CryptoHolding[]; loading: boolean }> = ({ holdings, loading }) => {
  if (loading) return <CircularProgress />;

  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Asset</TableCell>
          <TableCell align="right">Quantity</TableCell>
          <TableCell align="right">Avg Cost</TableCell>
          <TableCell align="right">Current Price</TableCell>
          <TableCell align="right">Value</TableCell>
          <TableCell align="right">24h Change</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {holdings.map((holding) => (
          <TableRow key={holding.assetSymbol}>
            <TableCell>
              <Typography variant="body2" fontWeight="bold">{holding.assetSymbol}</Typography>
              <Typography variant="caption" color="textSecondary">{holding.assetName}</Typography>
            </TableCell>
            <TableCell align="right">{holding.quantity.toFixed(8)}</TableCell>
            <TableCell align="right">${holding.averageCostPerUnit.toLocaleString()}</TableCell>
            <TableCell align="right">
              {holding.currentPrice ? `$${holding.currentPrice.toLocaleString()}` : '-'}
            </TableCell>
            <TableCell align="right">
              {holding.currentPrice
                ? `$${(holding.quantity * holding.currentPrice).toLocaleString()}`
                : '-'}
            </TableCell>
            <TableCell align="right">
              {holding.change24h !== undefined && (
                <Chip
                  label={`${holding.change24h >= 0 ? '+' : ''}${holding.change24h.toFixed(2)}%`}
                  color={holding.change24h >= 0 ? 'success' : 'error'}
                  size="small"
                />
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};

// DeFi Positions Table
const DeFiPositionsTable: React.FC<{ positions: DeFiPosition[] }> = ({ positions }) => {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Protocol</TableCell>
          <TableCell>Type</TableCell>
          <TableCell align="right">Deposited</TableCell>
          <TableCell align="right">Value</TableCell>
          <TableCell align="right">APR</TableCell>
          <TableCell align="right">Unclaimed Rewards</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {positions.map((pos, idx) => (
          <TableRow key={idx}>
            <TableCell>{pos.protocol}</TableCell>
            <TableCell>
              <Chip label={pos.positionType} size="small" />
            </TableCell>
            <TableCell align="right">
              {pos.quantityDeposited.toFixed(4)} {pos.assetDeposited}
            </TableCell>
            <TableCell align="right">
              {pos.currentValueUsd ? `$${pos.currentValueUsd.toLocaleString()}` : '-'}
            </TableCell>
            <TableCell align="right">
              {pos.apr ? `${pos.apr.toFixed(2)}%` : '-'}
            </TableCell>
            <TableCell align="right">
              {pos.unclaimedRewardsUsd ? `$${pos.unclaimedRewardsUsd.toLocaleString()}` : '-'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};

// Transactions Table
const TransactionsTable: React.FC<{ transactions: CryptoTransaction[] }> = ({ transactions }) => {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Type</TableCell>
          <TableCell>Asset</TableCell>
          <TableCell align="right">Quantity</TableCell>
          <TableCell align="right">Value</TableCell>
          <TableCell>Date</TableCell>
          <TableCell>Status</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {transactions.map((txn) => (
          <TableRow key={txn.id}>
            <TableCell>
              <Chip label={txn.txnType} size="small" />
            </TableCell>
            <TableCell>{txn.assetSymbol}</TableCell>
            <TableCell align="right">{txn.quantity.toFixed(8)}</TableCell>
            <TableCell align="right">
              {txn.fiatValueUsd ? `$${txn.fiatValueUsd.toLocaleString()}` : '-'}
            </TableCell>
            <TableCell>
              {txn.blockTimestamp
                ? new Date(txn.blockTimestamp).toLocaleDateString()
                : '-'}
            </TableCell>
            <TableCell>
              <Chip
                label={txn.status}
                color={txn.status === 'CONFIRMED' ? 'success' : 'warning'}
                size="small"
              />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};

// Tax Lots Viewer (stub)
const TaxLotsView: React.FC<{ walletId: string | null }> = ({ walletId }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>Tax Lot Tracking</Typography>
      <Typography color="textSecondary">
        View cost basis and tax lots using FIFO, LIFO, or HIFO methods
      </Typography>
      <Button variant="outlined" sx={{ mt: 2 }}>
        Download Form 8949
      </Button>
    </Box>
  );
};

export default CryptoPortfolioCenter;
