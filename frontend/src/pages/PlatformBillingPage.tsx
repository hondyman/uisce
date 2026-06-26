import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Alert,
  LinearProgress,
  IconButton,
  Tooltip,
  Button,
  Stack,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  TextField,
  Slider,
  Divider,
  Skeleton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  alpha,
  useTheme,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  AttachMoney as MoneyIcon,
  Storage as StorageIcon,
  Speed as SpeedIcon,
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  BarChart as ChartIcon,
  Receipt as ReceiptIcon,
  Calculate as CalculateIcon,
  CloudQueue as CloudIcon,
  Public as RegionIcon,
  TableChart as TableIcon,
  Assessment as AssessmentIcon,
  Download as DownloadIcon,
  Bolt as BoltIcon,
  Send as SendIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  Add as AddIcon,
  CardGiftcard as CreditIcon,
  Visibility as ViewIcon,
} from '@mui/icons-material';

// ── Types ────────────────────────────────────────────────────────

interface TenantCostBreakdown {
  computeUSD: number;
  storageUSD: number;
  eventsUSD: number;
  overageUSD: number;
  sloBreachUSD: number;
  totalUSD: number;
}

interface TenantUsage {
  eventsPublished: number;
  commits: number;
  s3Validations: number;
  idempotencyHits: number;
  computeMs: { p50: number; p95: number; p99: number; total: number };
  storage: { snapshotCount: number; totalBytes: number };
  regions: { region: string; commits: number; computeMs: number }[];
  tables: { table: string; commits: number; storageBytes: number }[];
}

interface TenantBillingResponse {
  tenantId: string;
  window: string;
  usage: TenantUsage;
  estimatedCost: TenantCostBreakdown;
}

interface PlatformTotals {
  computeUSD: number;
  storageUSD: number;
  eventsUSD: number;
  totalUSD: number;
}

interface RegionCost {
  region: string;
  totalUSD: number;
}

interface TenantCost {
  tenantId: string;
  totalUSD: number;
}

interface PlatformBillingResponse {
  window: string;
  totals: PlatformTotals;
  byRegion: RegionCost[];
  byTenant: TenantCost[];
  topTenants: TenantCost[];
  bottomTenants: TenantCost[];
}

interface BillingAnomaly {
  type: string;
  key: string;
  severity: string;
  ratio: number;
  reason: string;
  timestamp: string;
}

interface AnomalyResponse {
  tenantAnomalies: BillingAnomaly[];
  regionAnomalies: BillingAnomaly[];
  costAnomalies: BillingAnomaly[];
}

interface BillingForecast {
  forecastUSD: number;
  model: string;
  confidence: number;
}

interface TableCostItem {
  table: string;
  computeUSD: number;
  storageUSD: number;
}

interface CostSimulationResponse {
  estimatedCostUSD: number;
  breakdown: TenantCostBreakdown;
}

interface InvoiceResponse {
  tenantId: string;
  period: string;
  totalUSD: number;
  lineItems: { type: string; amountUSD: number }[];
}

interface DetailedLineItem {
  type: string;
  description: string;
  quantity: number;
  unitLabel: string;
  unitPrice: number;
  amountUSD: number;
}

interface AppliedCredit {
  creditId: string;
  reason: string;
  amountUSD: number;
}

interface Invoice {
  invoiceId: string;
  invoiceNumber: string;
  tenantId: string;
  status: 'DRAFT' | 'ISSUED' | 'PAID' | 'OVERDUE' | 'VOID';
  periodStart: string;
  periodEnd: string;
  periodLabel: string;
  lineItems: DetailedLineItem[];
  subtotalUSD: number;
  creditsUSD: number;
  discountUSD: number;
  taxUSD: number;
  totalDueUSD: number;
  appliedCredits: AppliedCredit[];
  issuedAt?: string;
  dueAt?: string;
  paidAt?: string;
  createdAt: string;
  notes?: string;
}

interface InvoiceSummary {
  invoiceId: string;
  invoiceNumber: string;
  tenantId: string;
  periodLabel: string;
  status: 'DRAFT' | 'ISSUED' | 'PAID' | 'OVERDUE' | 'VOID';
  totalDueUSD: number;
  issuedAt?: string;
  dueAt?: string;
}

// ── Gradient Presets ─────────────────────────────────────────────

const gradients = {
  compute: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  storage: 'linear-gradient(135deg, #11998e 0%, #38ef7d 100%)',
  events:  'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  total:   'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  anomaly: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
  forecast:'linear-gradient(135deg, #a18cd1 0%, #fbc2eb 100%)',
};

const severityColors: Record<string, string> = {
  critical: '#f44336',
  high: '#ff5722',
  medium: '#ff9800',
  low: '#ffc107',
};

// ── Main Component ───────────────────────────────────────────────

const PlatformBillingPage: React.FC = () => {
  const theme = useTheme();
  const [activeTab, setActiveTab] = useState(0);
  const [window, setWindow] = useState('30d');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Data state
  const [platformData, setPlatformData] = useState<PlatformBillingResponse | null>(null);
  const [tenantData, setTenantData] = useState<TenantBillingResponse | null>(null);
  const [anomalies, setAnomalies] = useState<AnomalyResponse | null>(null);
  const [forecast, setForecast] = useState<BillingForecast | null>(null);
  const [tableCosts, setTableCosts] = useState<TableCostItem[]>([]);

  // Invoice state
  const [invoices, setInvoices] = useState<InvoiceSummary[]>([]);
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null);
  const [invoiceDialogOpen, setInvoiceDialogOpen] = useState(false);
  const [generateMonth, setGenerateMonth] = useState(
    new Date().toISOString().slice(0, 7)
  );
  const [creditDialogOpen, setCreditDialogOpen] = useState(false);
  const [creditAmount, setCreditAmount] = useState('');
  const [creditReason, setCreditReason] = useState('');
  const [invoiceActionLoading, setInvoiceActionLoading] = useState(false);

  // Simulator state
  const [simEvents, setSimEvents] = useState(1000000);
  const [simCompute, setSimCompute] = useState(500000);
  const [simStorage, setSimStorage] = useState(100);
  const [simResult, setSimResult] = useState<CostSimulationResponse | null>(null);

  // Current tenant from header context
  const tenantId = localStorage.getItem('tenantId') || 'default';

  const fetchAll = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const headers: Record<string, string> = {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json',
      };

      const [platformRes, tenantRes, anomalyRes, forecastRes, tablesRes] = await Promise.allSettled([
        fetch(`/api/billing/platform?window=${window}`, { headers }),
        fetch(`/api/billing/tenant/${tenantId}?window=${window}`, { headers }),
        fetch('/api/billing/anomalies', { headers }),
        fetch('/api/billing/forecast', { headers }),
        fetch(`/api/billing/tables?window=${window}`, { headers }),
      ]);

      if (platformRes.status === 'fulfilled' && platformRes.value.ok) {
        setPlatformData(await platformRes.value.json());
      }
      if (tenantRes.status === 'fulfilled' && tenantRes.value.ok) {
        setTenantData(await tenantRes.value.json());
      }
      if (anomalyRes.status === 'fulfilled' && anomalyRes.value.ok) {
        setAnomalies(await anomalyRes.value.json());
      }
      if (forecastRes.status === 'fulfilled' && forecastRes.value.ok) {
        setForecast(await forecastRes.value.json());
      }
      if (tablesRes.status === 'fulfilled' && tablesRes.value.ok) {
        setTableCosts(await tablesRes.value.json());
      }

      // Fetch invoices
      try {
        const invRes = await fetch(`/api/invoices?tenantId=${tenantId}`, { headers });
        if (invRes.ok) {
          setInvoices(await invRes.json());
        }
      } catch { /* invoices are optional */ }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load billing data');
    } finally {
      setLoading(false);
    }
  }, [window, tenantId]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const runSimulation = async () => {
    try {
      const res = await fetch('/api/billing/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenantId,
          eventsPerMonth: simEvents,
          computeMs: simCompute,
          storageGB: simStorage,
          regions: ['us-east-1'],
          sloTier: 'standard',
        }),
      });
      if (res.ok) {
        setSimResult(await res.json());
      }
    } catch (err) {
      devError('Simulation failed:', err);
    }
  };

  // ── Invoice actions ──────────────────────────────────────────

  const generateInvoice = async () => {
    setInvoiceActionLoading(true);
    try {
      const res = await fetch('/api/invoices/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify({ tenantId, month: generateMonth }),
      });
      if (res.ok) {
        const inv = await res.json();
        setSelectedInvoice(inv);
        setInvoiceDialogOpen(true);
        fetchAll(); // refresh list
      }
    } catch (err) {
      devError('Generate invoice failed:', err);
    } finally {
      setInvoiceActionLoading(false);
    }
  };

  const viewInvoice = async (invoiceId: string) => {
    try {
      const res = await fetch(`/api/invoices/${invoiceId}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        setSelectedInvoice(await res.json());
        setInvoiceDialogOpen(true);
      }
    } catch (err) {
      devError('View invoice failed:', err);
    }
  };

  const issueInvoice = async (invoiceId: string) => {
    setInvoiceActionLoading(true);
    try {
      const res = await fetch(`/api/invoices/${invoiceId}/issue`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        setSelectedInvoice(await res.json());
        fetchAll();
      }
    } catch (err) {
      devError('Issue invoice failed:', err);
    } finally {
      setInvoiceActionLoading(false);
    }
  };

  const payInvoice = async (invoiceId: string) => {
    setInvoiceActionLoading(true);
    try {
      const res = await fetch(`/api/invoices/${invoiceId}/pay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        setSelectedInvoice(await res.json());
        fetchAll();
      }
    } catch (err) {
      devError('Pay invoice failed:', err);
    } finally {
      setInvoiceActionLoading(false);
    }
  };

  const voidInvoice = async (invoiceId: string) => {
    const reason = prompt('Reason for voiding this invoice:');
    if (!reason) return;
    setInvoiceActionLoading(true);
    try {
      const res = await fetch(`/api/invoices/${invoiceId}/void`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify({ reason }),
      });
      if (res.ok) {
        setSelectedInvoice(await res.json());
        fetchAll();
      }
    } catch (err) {
      devError('Void invoice failed:', err);
    } finally {
      setInvoiceActionLoading(false);
    }
  };

  const addCredit = async () => {
    if (!creditAmount || !creditReason) return;
    try {
      await fetch('/api/invoices/credits', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify({
          tenantId,
          amountUSD: parseFloat(creditAmount),
          reason: creditReason,
          expiresInDays: 365,
        }),
      });
      setCreditDialogOpen(false);
      setCreditAmount('');
      setCreditReason('');
    } catch (err) {
      devError('Add credit failed:', err);
    }
  };

  const statusColor = (status: string): 'default' | 'primary' | 'success' | 'warning' | 'error' => {
    switch (status) {
      case 'DRAFT': return 'default';
      case 'ISSUED': return 'primary';
      case 'PAID': return 'success';
      case 'OVERDUE': return 'warning';
      case 'VOID': return 'error';
      default: return 'default';
    }
  };

  const fmtUSD = (v: number) => `$${v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  const fmtBytes = (b: number) => {
    if (b >= 1e9) return `${(b / 1e9).toFixed(1)} GB`;
    if (b >= 1e6) return `${(b / 1e6).toFixed(1)} MB`;
    return `${(b / 1e3).toFixed(0)} KB`;
  };

  if (loading && !platformData) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <CircularProgress size={56} />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3, maxWidth: 1600, mx: 'auto' }}>
      {/* ── Header ───────────────────────────────────────────── */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Billing & Cost Analytics
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Platform-wide cost monitoring, forecasting, and usage attribution
          </Typography>
        </Box>
        <Stack direction="row" spacing={2} alignItems="center">
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel id="billing-window-label">Window</InputLabel>
            <Select
              labelId="billing-window-label"
              id="billing-window-select"
              value={window}
              label="Window"
              onChange={(e) => setWindow(e.target.value)}
            >
              <MenuItem value="1h">1 Hour</MenuItem>
              <MenuItem value="24h">24 Hours</MenuItem>
              <MenuItem value="7d">7 Days</MenuItem>
              <MenuItem value="30d">30 Days</MenuItem>
              <MenuItem value="90d">90 Days</MenuItem>
            </Select>
          </FormControl>
          <Tooltip title="Refresh data">
            <IconButton onClick={fetchAll} disabled={loading}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {error && (
        <Alert severity="warning" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* ── Tabs ─────────────────────────────────────────────── */}
      <Paper sx={{ mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, v) => setActiveTab(v)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab icon={<ChartIcon />} label="Overview" iconPosition="start" />
          <Tab icon={<MoneyIcon />} label="Tenant Billing" iconPosition="start" />
          <Tab icon={<RegionIcon />} label="Platform Costs" iconPosition="start" />
          <Tab icon={<WarningIcon />} label="Anomalies" iconPosition="start" />
          <Tab icon={<CalculateIcon />} label="Cost Simulator" iconPosition="start" />
          <Tab icon={<TableIcon />} label="Per-Table Costs" iconPosition="start" />
          <Tab icon={<ReceiptIcon />} label="Invoices" iconPosition="start" />
        </Tabs>
      </Paper>

      {/* ═══ TAB 0: Overview ═══════════════════════════════════ */}
      {activeTab === 0 && (
        <Box>
          {/* Summary Cards */}
          <Grid container spacing={3} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={6} md={3}>
              <Card sx={{ height: '100%', background: gradients.compute }}>
                <CardContent>
                  <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                    <Box>
                      <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                        Compute Cost
                      </Typography>
                      <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                        {fmtUSD(platformData?.totals.computeUSD || 0)}
                      </Typography>
                    </Box>
                    <SpeedIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Card sx={{ height: '100%', background: gradients.storage }}>
                <CardContent>
                  <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                    <Box>
                      <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                        Storage Cost
                      </Typography>
                      <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                        {fmtUSD(platformData?.totals.storageUSD || 0)}
                      </Typography>
                    </Box>
                    <StorageIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Card sx={{ height: '100%', background: gradients.events }}>
                <CardContent>
                  <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                    <Box>
                      <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                        Event Cost
                      </Typography>
                      <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                        {fmtUSD(platformData?.totals.eventsUSD || 0)}
                      </Typography>
                    </Box>
                    <BoltIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Card sx={{ height: '100%', background: gradients.total }}>
                <CardContent>
                  <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                    <Box>
                      <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                        Total Platform Cost
                      </Typography>
                      <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                        {fmtUSD(platformData?.totals.totalUSD || 0)}
                      </Typography>
                    </Box>
                    <MoneyIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
                  </Stack>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          {/* Forecast + Anomaly Summary */}
          <Grid container spacing={3} sx={{ mb: 3 }}>
            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 3, height: '100%' }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                  <Typography variant="h6" sx={{ fontWeight: 600 }}>
                    30-Day Forecast
                  </Typography>
                  <TrendingUpIcon color="primary" />
                </Stack>
                {forecast ? (
                  <Box>
                    <Typography variant="h3" sx={{ fontWeight: 700, color: theme.palette.primary.main }}>
                      {fmtUSD(forecast.forecastUSD)}
                    </Typography>
                    <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
                      <Chip
                        label={`Model: ${forecast.model}`}
                        size="small"
                        variant="outlined"
                      />
                      <Chip
                        label={`Confidence: ${(forecast.confidence * 100).toFixed(0)}%`}
                        size="small"
                        color={forecast.confidence > 0.85 ? 'success' : forecast.confidence > 0.7 ? 'warning' : 'error'}
                      />
                    </Stack>
                    <LinearProgress
                      variant="determinate"
                      value={forecast.confidence * 100}
                      sx={{ mt: 2, height: 8, borderRadius: 4 }}
                    />
                  </Box>
                ) : (
                  <Skeleton variant="rectangular" height={100} />
                )}
              </Paper>
            </Grid>
            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 3, height: '100%' }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                  <Typography variant="h6" sx={{ fontWeight: 600 }}>
                    Cost Anomalies
                  </Typography>
                  <WarningIcon color="warning" />
                </Stack>
                {anomalies ? (
                  <Box>
                    {[...anomalies.costAnomalies, ...anomalies.tenantAnomalies, ...anomalies.regionAnomalies].length === 0 ? (
                      <Stack alignItems="center" sx={{ py: 3 }}>
                        <AssessmentIcon sx={{ fontSize: 48, color: '#4caf50', mb: 1 }} />
                        <Typography color="text.secondary">No anomalies detected</Typography>
                      </Stack>
                    ) : (
                      <Stack spacing={1.5}>
                        {[...anomalies.costAnomalies, ...anomalies.tenantAnomalies, ...anomalies.regionAnomalies]
                          .slice(0, 4)
                          .map((a, i) => (
                            <Card
                              key={i}
                              variant="outlined"
                              sx={{
                                borderLeft: `4px solid ${severityColors[a.severity] || '#ff9800'}`,
                                backgroundColor: alpha(severityColors[a.severity] || '#ff9800', 0.05),
                              }}
                            >
                              <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
                                <Stack direction="row" justifyContent="space-between" alignItems="center">
                                  <Typography variant="body2" sx={{ fontWeight: 500 }}>
                                    {a.reason}
                                  </Typography>
                                  <Chip
                                    label={`${a.ratio}x`}
                                    size="small"
                                    sx={{
                                      backgroundColor: severityColors[a.severity],
                                      color: 'white',
                                      fontWeight: 700,
                                    }}
                                  />
                                </Stack>
                              </CardContent>
                            </Card>
                          ))}
                      </Stack>
                    )}
                  </Box>
                ) : (
                  <Skeleton variant="rectangular" height={100} />
                )}
              </Paper>
            </Grid>
          </Grid>

          {/* Top Tenants */}
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Top Tenants by Cost
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 600 }}>Tenant</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Cost</TableCell>
                        <TableCell sx={{ fontWeight: 600, width: 120 }}>Share</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {(platformData?.topTenants || []).slice(0, 8).map((t) => {
                        const pct = platformData?.totals.totalUSD
                          ? (t.totalUSD / platformData.totals.totalUSD) * 100
                          : 0;
                        return (
                          <TableRow key={t.tenantId} hover>
                            <TableCell>
                              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                                {t.tenantId}
                              </Typography>
                            </TableCell>
                            <TableCell align="right">{fmtUSD(t.totalUSD)}</TableCell>
                            <TableCell>
                              <LinearProgress
                                variant="determinate"
                                value={Math.min(pct, 100)}
                                sx={{ height: 6, borderRadius: 3 }}
                              />
                            </TableCell>
                          </TableRow>
                        );
                      })}
                      {(!platformData?.topTenants || platformData.topTenants.length === 0) && (
                        <TableRow>
                          <TableCell colSpan={3} align="center">
                            <Typography color="text.secondary" variant="body2">No data yet</Typography>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Paper>
            </Grid>
            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Cost by Region
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 600 }}>Region</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Cost</TableCell>
                        <TableCell sx={{ fontWeight: 600, width: 120 }}>Share</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {(platformData?.byRegion || []).map((r) => {
                        const pct = platformData?.totals.totalUSD
                          ? (r.totalUSD / platformData.totals.totalUSD) * 100
                          : 0;
                        return (
                          <TableRow key={r.region} hover>
                            <TableCell>
                              <Stack direction="row" spacing={1} alignItems="center">
                                <RegionIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
                                <Typography variant="body2" sx={{ fontWeight: 500 }}>
                                  {r.region}
                                </Typography>
                              </Stack>
                            </TableCell>
                            <TableCell align="right">{fmtUSD(r.totalUSD)}</TableCell>
                            <TableCell>
                              <LinearProgress
                                variant="determinate"
                                value={Math.min(pct, 100)}
                                color="secondary"
                                sx={{ height: 6, borderRadius: 3 }}
                              />
                            </TableCell>
                          </TableRow>
                        );
                      })}
                      {(!platformData?.byRegion || platformData.byRegion.length === 0) && (
                        <TableRow>
                          <TableCell colSpan={3} align="center">
                            <Typography color="text.secondary" variant="body2">No data yet</Typography>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Paper>
            </Grid>
          </Grid>
        </Box>
      )}

      {/* ═══ TAB 1: Tenant Billing ═════════════════════════════ */}
      {activeTab === 1 && (
        <Box>
          {/* Cost breakdown cards */}
          <Grid container spacing={3} sx={{ mb: 3 }}>
            {[
              { label: 'Compute', value: tenantData?.estimatedCost.computeUSD || 0, gradient: gradients.compute, icon: <SpeedIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 36 }} /> },
              { label: 'Storage', value: tenantData?.estimatedCost.storageUSD || 0, gradient: gradients.storage, icon: <StorageIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 36 }} /> },
              { label: 'Events', value: tenantData?.estimatedCost.eventsUSD || 0, gradient: gradients.events, icon: <BoltIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 36 }} /> },
              { label: 'Total Estimated', value: tenantData?.estimatedCost.totalUSD || 0, gradient: gradients.total, icon: <MoneyIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 36 }} /> },
            ].map((card) => (
              <Grid item xs={12} sm={6} md={3} key={card.label}>
                <Card sx={{ height: '100%', background: card.gradient }}>
                  <CardContent>
                    <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                      <Box>
                        <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                          {card.label}
                        </Typography>
                        <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                          {fmtUSD(card.value)}
                        </Typography>
                      </Box>
                      {card.icon}
                    </Stack>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>

          {/* Usage Details */}
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Usage Summary
                </Typography>
                <Stack spacing={2}>
                  <UsageRow label="Events Published" value={tenantData?.usage.eventsPublished?.toLocaleString() || '0'} />
                  <UsageRow label="Commits" value={tenantData?.usage.commits?.toLocaleString() || '0'} />
                  <UsageRow label="S3 Validations" value={tenantData?.usage.s3Validations?.toLocaleString() || '0'} />
                  <UsageRow label="Idempotency Hits" value={tenantData?.usage.idempotencyHits?.toLocaleString() || '0'} />
                  <Divider />
                  <UsageRow label="Total Compute" value={`${((tenantData?.usage.computeMs?.total || 0) / 1000).toFixed(1)}s`} />
                  <UsageRow label="p50 Latency" value={`${tenantData?.usage.computeMs?.p50?.toFixed(0) || 0}ms`} />
                  <UsageRow label="p95 Latency" value={`${tenantData?.usage.computeMs?.p95?.toFixed(0) || 0}ms`} />
                  <UsageRow label="p99 Latency" value={`${tenantData?.usage.computeMs?.p99?.toFixed(0) || 0}ms`} />
                  <Divider />
                  <UsageRow label="Storage" value={fmtBytes(tenantData?.usage.storage?.totalBytes || 0)} />
                  <UsageRow label="Snapshots" value={tenantData?.usage.storage?.snapshotCount?.toLocaleString() || '0'} />
                </Stack>
              </Paper>
            </Grid>

            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 3, mb: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Usage by Region
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 600 }}>Region</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Commits</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Compute</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {(tenantData?.usage.regions || []).map((r) => (
                        <TableRow key={r.region} hover>
                          <TableCell>{r.region}</TableCell>
                          <TableCell align="right">{r.commits.toLocaleString()}</TableCell>
                          <TableCell align="right">{(r.computeMs / 1000).toFixed(1)}s</TableCell>
                        </TableRow>
                      ))}
                      {(!tenantData?.usage.regions || tenantData.usage.regions.length === 0) && (
                        <TableRow>
                          <TableCell colSpan={3} align="center">
                            <Typography color="text.secondary" variant="body2">No region data</Typography>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Paper>

              <Paper sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Usage by Table
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 600 }}>Table</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Commits</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Storage</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {(tenantData?.usage.tables || []).map((t) => (
                        <TableRow key={t.table} hover>
                          <TableCell>{t.table}</TableCell>
                          <TableCell align="right">{t.commits.toLocaleString()}</TableCell>
                          <TableCell align="right">{fmtBytes(t.storageBytes)}</TableCell>
                        </TableRow>
                      ))}
                      {(!tenantData?.usage.tables || tenantData.usage.tables.length === 0) && (
                        <TableRow>
                          <TableCell colSpan={3} align="center">
                            <Typography color="text.secondary" variant="body2">No table data</Typography>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Paper>
            </Grid>
          </Grid>
        </Box>
      )}

      {/* ═══ TAB 2: Platform Costs ═════════════════════════════ */}
      {activeTab === 2 && (
        <Box>
          {/* Summary */}
          <Paper sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Platform Cost Breakdown ({window})
            </Typography>
            <Grid container spacing={3}>
              {[
                { label: 'Compute', value: platformData?.totals.computeUSD || 0, color: '#667eea' },
                { label: 'Storage', value: platformData?.totals.storageUSD || 0, color: '#11998e' },
                { label: 'Events', value: platformData?.totals.eventsUSD || 0, color: '#4facfe' },
              ].map((item) => {
                const pct = platformData?.totals.totalUSD
                  ? (item.value / platformData.totals.totalUSD) * 100
                  : 0;
                return (
                  <Grid item xs={12} sm={4} key={item.label}>
                    <Box sx={{ textAlign: 'center', p: 2 }}>
                      <Typography variant="overline" color="text.secondary">{item.label}</Typography>
                      <Typography variant="h4" sx={{ fontWeight: 700, my: 1 }}>{fmtUSD(item.value)}</Typography>
                      <LinearProgress
                        variant="determinate"
                        value={pct}
                        sx={{
                          height: 8,
                          borderRadius: 4,
                          '& .MuiLinearProgress-bar': { backgroundColor: item.color },
                        }}
                      />
                      <Typography variant="caption" color="text.secondary">{pct.toFixed(1)}% of total</Typography>
                    </Box>
                  </Grid>
                );
              })}
            </Grid>
          </Paper>

          {/* Full tenant list */}
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              All Tenants
            </Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 600 }}>#</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Tenant ID</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 600 }}>Total Cost</TableCell>
                    <TableCell sx={{ fontWeight: 600, width: 200 }}>Distribution</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(platformData?.byTenant || []).map((t, idx) => {
                    const pct = platformData?.totals.totalUSD
                      ? (t.totalUSD / platformData.totals.totalUSD) * 100
                      : 0;
                    return (
                      <TableRow key={t.tenantId} hover>
                        <TableCell>{idx + 1}</TableCell>
                        <TableCell>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>{t.tenantId}</Typography>
                        </TableCell>
                        <TableCell align="right">{fmtUSD(t.totalUSD)}</TableCell>
                        <TableCell>
                          <Stack direction="row" spacing={1} alignItems="center">
                            <LinearProgress
                              variant="determinate"
                              value={Math.min(pct, 100)}
                              sx={{ flex: 1, height: 6, borderRadius: 3 }}
                            />
                            <Typography variant="caption" sx={{ minWidth: 40 }}>
                              {pct.toFixed(1)}%
                            </Typography>
                          </Stack>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        </Box>
      )}

      {/* ═══ TAB 3: Anomalies ══════════════════════════════════ */}
      {activeTab === 3 && (
        <Box>
          {[
            { title: 'Global Cost Anomalies', items: anomalies?.costAnomalies || [], icon: <MoneyIcon /> },
            { title: 'Tenant Anomalies', items: anomalies?.tenantAnomalies || [], icon: <CloudIcon /> },
            { title: 'Region Anomalies', items: anomalies?.regionAnomalies || [], icon: <RegionIcon /> },
          ].map(({ title, items, icon }) => (
            <Paper key={title} sx={{ p: 3, mb: 3 }}>
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 2 }}>
                {icon}
                <Typography variant="h6" sx={{ fontWeight: 600 }}>{title}</Typography>
                <Chip label={items.length} size="small" color={items.length > 0 ? 'warning' : 'success'} />
              </Stack>
              {items.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <AssessmentIcon sx={{ fontSize: 48, color: '#4caf50', mb: 1 }} />
                  <Typography color="text.secondary">All clear — no anomalies detected</Typography>
                </Box>
              ) : (
                <Stack spacing={1.5}>
                  {items.map((a, i) => (
                    <Card
                      key={i}
                      variant="outlined"
                      sx={{
                        borderLeft: `4px solid ${severityColors[a.severity] || '#ff9800'}`,
                        backgroundColor: alpha(severityColors[a.severity] || '#ff9800', 0.04),
                      }}
                    >
                      <CardContent sx={{ py: 2, '&:last-child': { pb: 2 } }}>
                        <Stack direction="row" justifyContent="space-between" alignItems="center">
                          <Box>
                            <Typography variant="body1" sx={{ fontWeight: 600 }}>
                              {a.reason}
                            </Typography>
                            <Stack direction="row" spacing={1} sx={{ mt: 0.5 }}>
                              <Chip label={a.type} size="small" variant="outlined" />
                              <Chip label={a.key} size="small" variant="outlined" />
                              <Typography variant="caption" color="text.secondary">
                                {new Date(a.timestamp).toLocaleString()}
                              </Typography>
                            </Stack>
                          </Box>
                          <Chip
                            label={`${a.ratio}x spike`}
                            size="small"
                            sx={{
                              backgroundColor: severityColors[a.severity],
                              color: 'white',
                              fontWeight: 700,
                              fontSize: '0.85rem',
                            }}
                          />
                        </Stack>
                      </CardContent>
                    </Card>
                  ))}
                </Stack>
              )}
            </Paper>
          ))}
        </Box>
      )}

      {/* ═══ TAB 4: Cost Simulator ═════════════════════════════ */}
      {activeTab === 4 && (
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 3 }}>
                What-If Cost Simulator
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                Estimate your monthly costs based on projected usage. Adjust the sliders below.
              </Typography>

              <Stack spacing={4}>
                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Events per Month: {simEvents.toLocaleString()}
                  </Typography>
                  <Slider
                    value={simEvents}
                    onChange={(_, v) => setSimEvents(v as number)}
                    min={0}
                    max={100000000}
                    step={100000}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(v) => `${(v / 1e6).toFixed(1)}M`}
                  />
                </Box>

                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Compute (ms/month): {simCompute.toLocaleString()}
                  </Typography>
                  <Slider
                    value={simCompute}
                    onChange={(_, v) => setSimCompute(v as number)}
                    min={0}
                    max={10000000}
                    step={10000}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(v) => `${(v / 1e6).toFixed(1)}M ms`}
                  />
                </Box>

                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Storage (GB): {simStorage}
                  </Typography>
                  <Slider
                    value={simStorage}
                    onChange={(_, v) => setSimStorage(v as number)}
                    min={0}
                    max={10000}
                    step={10}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(v) => `${v} GB`}
                  />
                </Box>

                <Button
                  variant="contained"
                  size="large"
                  startIcon={<CalculateIcon />}
                  onClick={runSimulation}
                  sx={{ py: 1.5 }}
                >
                  Estimate Cost
                </Button>
              </Stack>
            </Paper>
          </Grid>

          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3, height: '100%' }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 3 }}>
                Estimated Monthly Cost
              </Typography>
              {simResult ? (
                <Box>
                  <Typography
                    variant="h2"
                    sx={{
                      fontWeight: 800,
                      background: gradients.total,
                      WebkitBackgroundClip: 'text',
                      WebkitTextFillColor: 'transparent',
                      mb: 3,
                    }}
                  >
                    {fmtUSD(simResult.estimatedCostUSD)}
                  </Typography>

                  <Stack spacing={2}>
                    <CostBreakdownRow label="Compute" value={simResult.breakdown.computeUSD} color="#667eea" />
                    <CostBreakdownRow label="Storage" value={simResult.breakdown.storageUSD} color="#11998e" />
                    <CostBreakdownRow label="Events" value={simResult.breakdown.eventsUSD} color="#4facfe" />
                  </Stack>
                </Box>
              ) : (
                <Box sx={{ textAlign: 'center', py: 6 }}>
                  <CalculateIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
                  <Typography color="text.secondary">
                    Adjust the sliders and click "Estimate Cost" to see your projected bill
                  </Typography>
                </Box>
              )}
            </Paper>
          </Grid>
        </Grid>
      )}

      {/* ═══ TAB 5: Per-Table Costs ════════════════════════════ */}
      {activeTab === 5 && (
        <Paper sx={{ p: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              Cost Attribution by Table
            </Typography>
            <Chip label={`${tableCosts.length} tables`} size="small" variant="outlined" />
          </Stack>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 600 }}>Table</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600 }}>Compute Cost</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600 }}>Storage Cost</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600 }}>Total</TableCell>
                  <TableCell sx={{ fontWeight: 600, width: 200 }}>Distribution</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tableCosts.map((t) => {
                  const total = t.computeUSD + t.storageUSD;
                  const maxTotal = tableCosts.reduce((max, tc) => Math.max(max, tc.computeUSD + tc.storageUSD), 1);
                  const pct = (total / maxTotal) * 100;
                  return (
                    <TableRow key={t.table} hover>
                      <TableCell>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <TableIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>{t.table}</Typography>
                        </Stack>
                      </TableCell>
                      <TableCell align="right">{fmtUSD(t.computeUSD)}</TableCell>
                      <TableCell align="right">{fmtUSD(t.storageUSD)}</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 600 }}>{fmtUSD(total)}</TableCell>
                      <TableCell>
                        <LinearProgress
                          variant="determinate"
                          value={pct}
                          sx={{
                            height: 6,
                            borderRadius: 3,
                            '& .MuiLinearProgress-bar': { backgroundColor: '#667eea' },
                          }}
                        />
                      </TableCell>
                    </TableRow>
                  );
                })}
                {tableCosts.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                      <TableIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
                      <Typography color="text.secondary">No table cost data available</Typography>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}

      {/* ═══ TAB 6: Invoices ═══════════════════════════════════ */}
      {activeTab === 6 && (
        <Box>
          {/* Actions bar */}
          <Paper sx={{ p: 3, mb: 3 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Box>
                <Typography variant="h6" sx={{ fontWeight: 600 }}>
                  Invoice Management
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Generate, review, and manage tenant invoices
                </Typography>
              </Box>
              <Stack direction="row" spacing={2}>
                <Button
                  variant="outlined"
                  startIcon={<CreditIcon />}
                  onClick={() => setCreditDialogOpen(true)}
                >
                  Add Credit
                </Button>
                <Stack direction="row" spacing={1} alignItems="center">
                  <TextField
                    type="month"
                    size="small"
                    value={generateMonth}
                    onChange={(e) => setGenerateMonth(e.target.value)}
                    sx={{ minWidth: 160 }}
                  />
                  <Button
                    variant="contained"
                    startIcon={invoiceActionLoading ? <CircularProgress size={16} /> : <AddIcon />}
                    onClick={generateInvoice}
                    disabled={invoiceActionLoading}
                  >
                    Generate Invoice
                  </Button>
                </Stack>
              </Stack>
            </Stack>
          </Paper>

          {/* Invoice List */}
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Recent Invoices
            </Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 600 }}>Invoice #</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Tenant</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Period</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 600 }}>Total Due</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Issued</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Due Date</TableCell>
                    <TableCell align="center" sx={{ fontWeight: 600 }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {invoices.map((inv) => (
                    <TableRow key={inv.invoiceId} hover>
                      <TableCell>
                        <Typography
                          variant="body2"
                          sx={{ fontWeight: 600, color: theme.palette.primary.main, cursor: 'pointer' }}
                          onClick={() => viewInvoice(inv.invoiceId)}
                        >
                          {inv.invoiceNumber}
                        </Typography>
                      </TableCell>
                      <TableCell>{inv.tenantId}</TableCell>
                      <TableCell>{inv.periodLabel}</TableCell>
                      <TableCell>
                        <Chip label={inv.status} size="small" color={statusColor(inv.status)} />
                      </TableCell>
                      <TableCell align="right">{fmtUSD(inv.totalDueUSD)}</TableCell>
                      <TableCell>
                        {inv.issuedAt ? new Date(inv.issuedAt).toLocaleDateString() : '—'}
                      </TableCell>
                      <TableCell>
                        {inv.dueAt ? new Date(inv.dueAt).toLocaleDateString() : '—'}
                      </TableCell>
                      <TableCell align="center">
                        <Stack direction="row" spacing={0.5} justifyContent="center">
                          <Tooltip title="View Details">
                            <IconButton size="small" onClick={() => viewInvoice(inv.invoiceId)}>
                              <ViewIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          {inv.status === 'DRAFT' && (
                            <Tooltip title="Issue Invoice">
                              <IconButton
                                size="small"
                                color="primary"
                                onClick={() => issueInvoice(inv.invoiceId)}
                              >
                                <SendIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          )}
                          {(inv.status === 'ISSUED' || inv.status === 'OVERDUE') && (
                            <Tooltip title="Mark Paid">
                              <IconButton
                                size="small"
                                color="success"
                                onClick={() => payInvoice(inv.invoiceId)}
                              >
                                <CheckCircleIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          )}
                          {inv.status !== 'PAID' && inv.status !== 'VOID' && (
                            <Tooltip title="Void Invoice">
                              <IconButton
                                size="small"
                                color="error"
                                onClick={() => voidInvoice(inv.invoiceId)}
                              >
                                <CancelIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          )}
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))}
                  {invoices.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={8} align="center">
                        <Stack alignItems="center" sx={{ py: 4 }}>
                          <ReceiptIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
                          <Typography color="text.secondary">
                            No invoices generated yet. Click "Generate Invoice" to create one.
                          </Typography>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>

          {/* Invoice Detail Dialog */}
          <Dialog
            open={invoiceDialogOpen}
            onClose={() => setInvoiceDialogOpen(false)}
            maxWidth="md"
            fullWidth
          >
            {selectedInvoice && (
              <>
                <DialogTitle>
                  <Stack direction="row" justifyContent="space-between" alignItems="center">
                    <Box>
                      <Typography variant="h6" sx={{ fontWeight: 700 }}>
                        Invoice {selectedInvoice.invoiceNumber}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Tenant: {selectedInvoice.tenantId} · Period: {selectedInvoice.periodLabel}
                      </Typography>
                    </Box>
                    <Chip
                      label={selectedInvoice.status}
                      color={statusColor(selectedInvoice.status)}
                      sx={{ fontWeight: 700 }}
                    />
                  </Stack>
                </DialogTitle>
                <DialogContent dividers>
                  {/* Dates */}
                  <Grid container spacing={2} sx={{ mb: 3 }}>
                    <Grid item xs={4}>
                      <Typography variant="overline" color="text.secondary">Created</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {new Date(selectedInvoice.createdAt).toLocaleDateString()}
                      </Typography>
                    </Grid>
                    <Grid item xs={4}>
                      <Typography variant="overline" color="text.secondary">Issued</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {selectedInvoice.issuedAt
                          ? new Date(selectedInvoice.issuedAt).toLocaleDateString()
                          : '—'}
                      </Typography>
                    </Grid>
                    <Grid item xs={4}>
                      <Typography variant="overline" color="text.secondary">Due Date</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {selectedInvoice.dueAt
                          ? new Date(selectedInvoice.dueAt).toLocaleDateString()
                          : '—'}
                      </Typography>
                    </Grid>
                  </Grid>

                  {/* Line Items */}
                  <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>Line Items</Typography>
                  <TableContainer sx={{ mb: 3 }}>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 600 }}>Type</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Description</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600 }}>Qty</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600 }}>Unit</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600 }}>Unit Price</TableCell>
                          <TableCell align="right" sx={{ fontWeight: 600 }}>Amount</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {selectedInvoice.lineItems.map((li, idx) => (
                          <TableRow key={idx}>
                            <TableCell>
                              <Chip label={li.type} size="small" variant="outlined" />
                            </TableCell>
                            <TableCell>{li.description}</TableCell>
                            <TableCell align="right">{li.quantity.toLocaleString()}</TableCell>
                            <TableCell align="right">{li.unitLabel}</TableCell>
                            <TableCell align="right">${li.unitPrice.toFixed(6)}</TableCell>
                            <TableCell align="right" sx={{ fontWeight: 600 }}>
                              {fmtUSD(li.amountUSD)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>

                  {/* Totals */}
                  <Paper variant="outlined" sx={{ p: 2 }}>
                    <Stack spacing={1}>
                      <Stack direction="row" justifyContent="space-between">
                        <Typography variant="body2">Subtotal</Typography>
                        <Typography variant="body2" sx={{ fontWeight: 500 }}>
                          {fmtUSD(selectedInvoice.subtotalUSD)}
                        </Typography>
                      </Stack>
                      {selectedInvoice.creditsUSD > 0 && (
                        <Stack direction="row" justifyContent="space-between">
                          <Typography variant="body2" color="success.main">Credits Applied</Typography>
                          <Typography variant="body2" color="success.main" sx={{ fontWeight: 500 }}>
                            −{fmtUSD(selectedInvoice.creditsUSD)}
                          </Typography>
                        </Stack>
                      )}
                      {selectedInvoice.discountUSD > 0 && (
                        <Stack direction="row" justifyContent="space-between">
                          <Typography variant="body2" color="info.main">Volume Discount</Typography>
                          <Typography variant="body2" color="info.main" sx={{ fontWeight: 500 }}>
                            −{fmtUSD(selectedInvoice.discountUSD)}
                          </Typography>
                        </Stack>
                      )}
                      <Divider />
                      <Stack direction="row" justifyContent="space-between">
                        <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>Total Due</Typography>
                        <Typography variant="subtitle1" sx={{ fontWeight: 700, color: theme.palette.primary.main }}>
                          {fmtUSD(selectedInvoice.totalDueUSD)}
                        </Typography>
                      </Stack>
                    </Stack>
                  </Paper>

                  {/* Applied Credits */}
                  {selectedInvoice.appliedCredits && selectedInvoice.appliedCredits.length > 0 && (
                    <Box sx={{ mt: 2 }}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>Applied Credits</Typography>
                      {selectedInvoice.appliedCredits.map((c, idx) => (
                        <Chip
                          key={idx}
                          icon={<CreditIcon />}
                          label={`${c.reason}: −${fmtUSD(c.amountUSD)}`}
                          size="small"
                          color="success"
                          variant="outlined"
                          sx={{ mr: 1, mb: 0.5 }}
                        />
                      ))}
                    </Box>
                  )}

                  {/* Notes */}
                  {selectedInvoice.notes && (
                    <Alert severity="info" sx={{ mt: 2 }}>
                      {selectedInvoice.notes}
                    </Alert>
                  )}
                </DialogContent>
                <DialogActions>
                  {selectedInvoice.status === 'DRAFT' && (
                    <Button
                      variant="contained"
                      startIcon={<SendIcon />}
                      onClick={() => issueInvoice(selectedInvoice.invoiceId)}
                      disabled={invoiceActionLoading}
                    >
                      Issue Invoice
                    </Button>
                  )}
                  {(selectedInvoice.status === 'ISSUED' || selectedInvoice.status === 'OVERDUE') && (
                    <Button
                      variant="contained"
                      color="success"
                      startIcon={<CheckCircleIcon />}
                      onClick={() => payInvoice(selectedInvoice.invoiceId)}
                      disabled={invoiceActionLoading}
                    >
                      Mark Paid
                    </Button>
                  )}
                  {selectedInvoice.status !== 'PAID' && selectedInvoice.status !== 'VOID' && (
                    <Button
                      variant="outlined"
                      color="error"
                      startIcon={<CancelIcon />}
                      onClick={() => voidInvoice(selectedInvoice.invoiceId)}
                      disabled={invoiceActionLoading}
                    >
                      Void
                    </Button>
                  )}
                  <Button onClick={() => setInvoiceDialogOpen(false)}>Close</Button>
                </DialogActions>
              </>
            )}
          </Dialog>

          {/* Add Credit Dialog */}
          <Dialog
            open={creditDialogOpen}
            onClose={() => setCreditDialogOpen(false)}
            maxWidth="sm"
            fullWidth
          >
            <DialogTitle>Add Billing Credit</DialogTitle>
            <DialogContent>
              <Stack spacing={2} sx={{ mt: 1 }}>
                <TextField
                  label="Tenant ID"
                  value={tenantId}
                  disabled
                  fullWidth
                  size="small"
                />
                <TextField
                  label="Credit Amount (USD)"
                  type="number"
                  value={creditAmount}
                  onChange={(e) => setCreditAmount(e.target.value)}
                  fullWidth
                  size="small"
                  inputProps={{ min: 0, step: 0.01 }}
                />
                <TextField
                  label="Reason"
                  value={creditReason}
                  onChange={(e) => setCreditReason(e.target.value)}
                  fullWidth
                  size="small"
                  multiline
                  rows={2}
                  placeholder="e.g. SLA breach compensation, promotional credit"
                />
              </Stack>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setCreditDialogOpen(false)}>Cancel</Button>
              <Button
                variant="contained"
                onClick={addCredit}
                disabled={!creditAmount || !creditReason}
              >
                Add Credit
              </Button>
            </DialogActions>
          </Dialog>
        </Box>
      )}
    </Box>
  );
};

// ── Sub-components ─────────────────────────────────────────────

const UsageRow: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <Stack direction="row" justifyContent="space-between" alignItems="center">
    <Typography variant="body2" color="text.secondary">{label}</Typography>
    <Typography variant="body2" sx={{ fontWeight: 600 }}>{value}</Typography>
  </Stack>
);

const CostBreakdownRow: React.FC<{ label: string; value: number; color: string }> = ({ label, value, color }) => {
  const fmtUSD = (v: number) => `$${v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
        <Typography variant="body2">{label}</Typography>
        <Typography variant="body2" sx={{ fontWeight: 600 }}>{fmtUSD(value)}</Typography>
      </Stack>
      <LinearProgress
        variant="determinate"
        value={100}
        sx={{
          height: 6,
          borderRadius: 3,
          '& .MuiLinearProgress-bar': { backgroundColor: color },
        }}
      />
    </Box>
  );
};

export default PlatformBillingPage;
