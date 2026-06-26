import React, { useState, useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  Tab,
  Stack,
  LinearProgress,
  Divider,
  IconButton,
  Tooltip,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  AccountBalance as AccountIcon,
  PieChart as PieChartIcon,
  ShowChart as ChartIcon,
  Refresh as RefreshIcon,
  Info as InfoIcon,
} from '@mui/icons-material';

// ============================================================================
// Types
// ============================================================================

interface PositionSummary {
  security_id: string;
  security_name: string;
  ticker: string;
  quantity: number;
  market_value: number;
  weight: number;
  day_change: number;
  day_change_percent: number;
  unrealized_gain_loss: number;
  unrealized_gain_loss_percent: number;
}

interface AllocationItem {
  category: string;
  market_value: number;
  weight: number;
  target_weight?: number;
  drift?: number;
}

interface PortfolioSummary {
  id: string;
  name: string;
  total_market_value: number;
  total_cost_basis: number;
  total_unrealized_gain_loss: number;
  unrealized_gain_loss_percent: number;
  day_change: number;
  day_change_percent: number;
  ytd_return: number;
  as_of_date: string;
  position_count: number;
  account_count: number;
  top_holdings: PositionSummary[];
  asset_allocation: AllocationItem[];
  sector_allocation: AllocationItem[];
}

interface PerformanceMetrics {
  portfolio_id: string;
  as_of_date: string;
  mtd_return: number;
  qtd_return: number;
  ytd_return: number;
  one_year_return: number;
  three_year_return: number;
  five_year_return: number;
  since_inception: number;
  alpha?: number;
  beta?: number;
  sharpe_ratio?: number;
  volatility?: number;
  max_drawdown?: number;
}

// ============================================================================
// Helper Functions
// ============================================================================

const formatCurrency = (value: number): string => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
};

const formatPercent = (value: number, decimals: number = 2): string => {
  return `${value >= 0 ? '+' : ''}${value.toFixed(decimals)}%`;
};

const getChangeColor = (value: number): string => {
  if (value > 0) return '#4caf50';
  if (value < 0) return '#f44336';
  return 'inherit';
};

// ============================================================================
// MetricCard Component
// ============================================================================

interface MetricCardProps {
  title: string;
  value: string;
  change?: number;
  subtitle?: string;
  icon?: React.ReactNode;
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, change, subtitle, icon }) => (
  <Card sx={{ height: '100%' }}>
    <CardContent>
      <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
        <Box>
          <Typography variant="caption" color="text.secondary" gutterBottom>
            {title}
          </Typography>
          <Typography variant="h5" fontWeight="bold">
            {value}
          </Typography>
          {change !== undefined && (
            <Stack direction="row" alignItems="center" spacing={0.5}>
              {change >= 0 ? (
                <TrendingUpIcon sx={{ fontSize: 16, color: '#4caf50' }} />
              ) : (
                <TrendingDownIcon sx={{ fontSize: 16, color: '#f44336' }} />
              )}
              <Typography variant="body2" sx={{ color: getChangeColor(change) }}>
                {formatPercent(change)}
              </Typography>
            </Stack>
          )}
          {subtitle && (
            <Typography variant="caption" color="text.secondary">
              {subtitle}
            </Typography>
          )}
        </Box>
        {icon && (
          <Box sx={{ color: 'primary.main', opacity: 0.7 }}>
            {icon}
          </Box>
        )}
      </Stack>
    </CardContent>
  </Card>
);

// ============================================================================
// Allocation Chart Component
// ============================================================================

interface AllocationChartProps {
  data: AllocationItem[];
  title: string;
  showDrift?: boolean;
}

const COLORS = ['#2196f3', '#4caf50', '#ff9800', '#9c27b0', '#f44336', '#00bcd4', '#795548', '#607d8b'];

const AllocationChart: React.FC<AllocationChartProps> = ({ data, title, showDrift }) => {
  const sortedData = useMemo(() => 
    [...data].sort((a, b) => b.weight - a.weight), 
    [data]
  );

  return (
    <Paper sx={{ p: 2, height: '100%' }}>
      <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
        {title}
      </Typography>
      <Stack spacing={1.5}>
        {sortedData.map((item, index) => (
          <Box key={item.category}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.5 }}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <Box
                  sx={{
                    width: 12,
                    height: 12,
                    borderRadius: 1,
                    bgcolor: COLORS[index % COLORS.length],
                  }}
                />
                <Typography variant="body2">{item.category}</Typography>
              </Stack>
              <Stack direction="row" alignItems="center" spacing={1}>
                <Typography variant="body2" fontWeight="medium">
                  {item.weight.toFixed(1)}%
                </Typography>
                {showDrift && item.drift !== undefined && (
                  <Chip
                    label={formatPercent(item.drift, 1)}
                    size="small"
                    sx={{
                      bgcolor: Math.abs(item.drift) > 5 ? '#ffebee' : '#e8f5e9',
                      color: Math.abs(item.drift) > 5 ? '#c62828' : '#2e7d32',
                    }}
                  />
                )}
              </Stack>
            </Stack>
            <LinearProgress
              variant="determinate"
              value={Math.min(item.weight, 100)}
              sx={{
                height: 8,
                borderRadius: 1,
                bgcolor: 'grey.200',
                '& .MuiLinearProgress-bar': {
                  bgcolor: COLORS[index % COLORS.length],
                },
              }}
            />
          </Box>
        ))}
      </Stack>
    </Paper>
  );
};

// ============================================================================
// Holdings Table Component
// ============================================================================

interface HoldingsTableProps {
  holdings: PositionSummary[];
}

const HoldingsTable: React.FC<HoldingsTableProps> = ({ holdings }) => (
  <TableContainer component={Paper}>
    <Table size="small">
      <TableHead>
        <TableRow sx={{ bgcolor: 'grey.50' }}>
          <TableCell>Security</TableCell>
          <TableCell align="right">Market Value</TableCell>
          <TableCell align="right">Weight</TableCell>
          <TableCell align="right">Day Change</TableCell>
          <TableCell align="right">Unrealized G/L</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {holdings.map((holding) => (
          <TableRow key={holding.security_id} hover>
            <TableCell>
              <Box>
                <Typography variant="body2" fontWeight="medium">
                  {holding.ticker}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {holding.security_name}
                </Typography>
              </Box>
            </TableCell>
            <TableCell align="right">{formatCurrency(holding.market_value)}</TableCell>
            <TableCell align="right">{holding.weight.toFixed(2)}%</TableCell>
            <TableCell align="right" sx={{ color: getChangeColor(holding.day_change) }}>
              {formatCurrency(holding.day_change)}
              <Typography variant="caption" display="block" sx={{ color: getChangeColor(holding.day_change_percent) }}>
                {formatPercent(holding.day_change_percent)}
              </Typography>
            </TableCell>
            <TableCell align="right" sx={{ color: getChangeColor(holding.unrealized_gain_loss) }}>
              {formatCurrency(holding.unrealized_gain_loss)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  </TableContainer>
);

// ============================================================================
// Performance Table Component
// ============================================================================

interface PerformanceTableProps {
  metrics: PerformanceMetrics;
}

const PerformanceTable: React.FC<PerformanceTableProps> = ({ metrics }) => {
  const periods = [
    { label: 'MTD', value: metrics.mtd_return },
    { label: 'QTD', value: metrics.qtd_return },
    { label: 'YTD', value: metrics.ytd_return },
    { label: '1 Year', value: metrics.one_year_return },
    { label: '3 Year', value: metrics.three_year_return },
    { label: '5 Year', value: metrics.five_year_return },
    { label: 'Since Inception', value: metrics.since_inception },
  ];

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
        Time-Weighted Returns
      </Typography>
      <Grid container spacing={2}>
        {periods.map((period) => (
          <Grid item xs={6} sm={4} md key={period.label}>
            <Box textAlign="center" sx={{ p: 1 }}>
              <Typography variant="caption" color="text.secondary">
                {period.label}
              </Typography>
              <Typography
                variant="h6"
                fontWeight="bold"
                sx={{ color: getChangeColor(period.value) }}
              >
                {formatPercent(period.value)}
              </Typography>
            </Box>
          </Grid>
        ))}
      </Grid>
      
      <Divider sx={{ my: 2 }} />
      
      <Typography variant="subtitle2" gutterBottom>
        Risk Metrics
      </Typography>
      <Grid container spacing={2}>
        <Grid item xs={6} sm={3}>
          <Box>
            <Typography variant="caption" color="text.secondary">Volatility</Typography>
            <Typography variant="body1" fontWeight="medium">{metrics.volatility?.toFixed(2)}%</Typography>
          </Box>
        </Grid>
        <Grid item xs={6} sm={3}>
          <Box>
            <Typography variant="caption" color="text.secondary">Sharpe Ratio</Typography>
            <Typography variant="body1" fontWeight="medium">{metrics.sharpe_ratio?.toFixed(2)}</Typography>
          </Box>
        </Grid>
        <Grid item xs={6} sm={3}>
          <Box>
            <Typography variant="caption" color="text.secondary">Max Drawdown</Typography>
            <Typography variant="body1" fontWeight="medium" sx={{ color: '#f44336' }}>
              -{metrics.max_drawdown?.toFixed(2)}%
            </Typography>
          </Box>
        </Grid>
        <Grid item xs={6} sm={3}>
          <Box>
            <Typography variant="caption" color="text.secondary">Beta</Typography>
            <Typography variant="body1" fontWeight="medium">{metrics.beta?.toFixed(2) || 'N/A'}</Typography>
          </Box>
        </Grid>
      </Grid>
    </Paper>
  );
};

// ============================================================================
// Main Portfolio Dashboard Component
// ============================================================================

interface PortfolioDashboardProps {
  summary: PortfolioSummary;
  performance: PerformanceMetrics;
  onRefresh?: () => void;
}

export const PortfolioDashboard: React.FC<PortfolioDashboardProps> = ({
  summary,
  performance,
  onRefresh,
}) => {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <Box>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight="bold">
            {summary.name}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            As of {new Date(summary.as_of_date).toLocaleDateString()} • {summary.position_count} positions • {summary.account_count} accounts
          </Typography>
        </Box>
        {onRefresh && (
          <IconButton onClick={onRefresh}>
            <RefreshIcon />
          </IconButton>
        )}
      </Stack>

      {/* Summary Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Total Market Value"
            value={formatCurrency(summary.total_market_value)}
            change={summary.day_change_percent}
            subtitle={`${formatCurrency(summary.day_change)} today`}
            icon={<AccountIcon />}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Unrealized Gain/Loss"
            value={formatCurrency(summary.total_unrealized_gain_loss)}
            change={summary.unrealized_gain_loss_percent}
            subtitle="Total P&L"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="YTD Return"
            value={formatPercent(summary.ytd_return)}
            icon={<ChartIcon />}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Cost Basis"
            value={formatCurrency(summary.total_cost_basis)}
            subtitle="Original investment"
          />
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label="Holdings" icon={<AccountIcon />} iconPosition="start" />
          <Tab label="Allocation" icon={<PieChartIcon />} iconPosition="start" />
          <Tab label="Performance" icon={<ChartIcon />} iconPosition="start" />
        </Tabs>
      </Paper>

      {/* Tab Content */}
      {activeTab === 0 && (
        <HoldingsTable holdings={summary.top_holdings} />
      )}

      {activeTab === 1 && (
        <Grid container spacing={2}>
          <Grid item xs={12} md={6}>
            <AllocationChart
              data={summary.asset_allocation}
              title="Asset Allocation"
              showDrift
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <AllocationChart
              data={summary.sector_allocation}
              title="Sector Allocation"
            />
          </Grid>
        </Grid>
      )}

      {activeTab === 2 && (
        <PerformanceTable metrics={performance} />
      )}
    </Box>
  );
};

export default PortfolioDashboard;
