import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  TextField,
  InputAdornment,
  CircularProgress,
  Alert,
  Tooltip,
  Divider,
  LinearProgress,
  IconButton,
  Collapse,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet';
import LayersIcon from '@mui/icons-material/Layers';
import PaymentsIcon from '@mui/icons-material/Payments';
import HistoryIcon from '@mui/icons-material/History';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';

// ─── Types ──────────────────────────────────────────────────────────────────

interface Position {
  id: string;
  portfolio_id: string;
  security_id: string;
  position_date: string;
  position_quantity: number;
  position_side: string;
  position_currency: string;
  market_value_local?: number;
  market_value_base?: number;
  cost_basis_local?: number;
  unrealized_pl_local?: number;
  unrealized_pl_pct?: number;
  position_weight_pct?: number;
  position_source: string;
  position_confidence: number;
  is_reconciled: boolean;
  reconciliation_diff?: number;
}

interface Lot {
  id: string;
  position_id: string;
  lot_reference?: string;
  acquisition_date: string;
  settlement_date?: string;
  lot_quantity: number;
  cost_per_unit: number;
  total_cost_basis: number;
  lot_method: string;
  is_closed: boolean;
  realized_pl?: number;
}

interface CashPosition {
  id: string;
  portfolio_id: string;
  cash_currency: string;
  value_date: string;
  balance_amount: number;
  available_balance?: number;
  interest_accrued: number;
  cash_source: string;
}

interface Snapshot {
  id: string;
  snapshot_date: string;
  snapshot_quantity?: number;
  snapshot_market_value?: number;
  snapshot_price_used?: number;
  snapshot_source: string;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const fmtMoney = (v?: number, ccy = 'USD') =>
  v == null ? '—' : new Intl.NumberFormat('en-US', { style: 'currency', currency: ccy, maximumFractionDigits: 0 }).format(v);

const fmtNum = (v?: number, dp = 2) =>
  v == null ? '—' : v.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp });

const fmtPct = (v?: number) => v == null ? '—' : `${(v * 100).toFixed(2)}%`;

function PLCell({ value }: { value?: number }) {
  if (value == null) return <TableCell align="right">—</TableCell>;
  const positive = value >= 0;
  return (
    <TableCell align="right" sx={{ color: positive ? 'success.main' : 'error.main', fontWeight: 600 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 0.5 }}>
        {positive ? <TrendingUpIcon fontSize="small" /> : <TrendingDownIcon fontSize="small" />}
        {fmtMoney(value)}
      </Box>
    </TableCell>
  );
}

function ConfidenceBar({ score }: { score: number }) {
  const color = score >= 90 ? 'success' : score >= 70 ? 'warning' : 'error';
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 100 }}>
      <LinearProgress
        variant="determinate"
        value={score}
        color={color}
        sx={{ flex: 1, height: 6, borderRadius: 3 }}
      />
      <Typography variant="caption" sx={{ minWidth: 30, fontWeight: 600 }}>{score}%</Typography>
    </Box>
  );
}

function ReconciliationBadge({ reconciled, diff }: { reconciled: boolean; diff?: number }) {
  if (reconciled) {
    return (
      <Tooltip title="Reconciled with custodian">
        <CheckCircleIcon color="success" fontSize="small" />
      </Tooltip>
    );
  }
  return (
    <Tooltip title={`Unreconciled — diff: ${diff != null ? fmtNum(diff) : 'unknown'}`}>
      <WarningAmberIcon color="warning" fontSize="small" />
    </Tooltip>
  );
}

// ─── Holdings Tab with lot drilldown ─────────────────────────────────────────

function PositionRow({ pos, tenantId }: { pos: Position; tenantId: string }) {
  const [open, setOpen] = useState(false);
  const [lots, setLots] = useState<Lot[]>([]);
  const [lotsLoading, setLotsLoading] = useState(false);

  const handleExpand = () => {
    if (!open && lots.length === 0) {
      setLotsLoading(true);
      fetch(`/api/v1/positions/${pos.id}/lots?limit=20`, {
        headers: { 'X-Tenant-ID': tenantId },
      })
        .then((r) => r.json())
        .then((d) => { setLots(d.lots ?? []); setLotsLoading(false); })
        .catch(() => setLotsLoading(false));
    }
    setOpen((v) => !v);
  };

  return (
    <>
      <TableRow hover sx={{ '& > *': { borderBottom: 'unset' } }}>
        <TableCell padding="checkbox">
          <IconButton size="small" onClick={handleExpand}>
            {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
          </IconButton>
        </TableCell>
        <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>{pos.security_id.slice(0, 8)}…</TableCell>
        <TableCell>
          <Chip
            label={pos.position_side}
            color={pos.position_side === 'Long' ? 'success' : 'warning'}
            size="small"
            variant="outlined"
          />
        </TableCell>
        <TableCell>{pos.position_date?.slice(0, 10)}</TableCell>
        <TableCell align="right" sx={{ fontWeight: 600 }}>{fmtNum(pos.position_quantity)}</TableCell>
        <TableCell align="right">{fmtMoney(pos.market_value_local, pos.position_currency)}</TableCell>
        <PLCell value={pos.unrealized_pl_local} />
        <TableCell align="right">
          {pos.position_weight_pct != null ? (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <LinearProgress
                variant="determinate"
                value={Math.min(pos.position_weight_pct * 100, 100)}
                sx={{ width: 50, height: 6, borderRadius: 3 }}
              />
              <Typography variant="caption">{fmtPct(pos.position_weight_pct)}</Typography>
            </Box>
          ) : '—'}
        </TableCell>
        <TableCell>
          <Chip label={pos.position_source} size="small" variant="outlined" />
        </TableCell>
        <TableCell>
          <ConfidenceBar score={pos.position_confidence} />
        </TableCell>
        <TableCell align="center">
          <ReconciliationBadge reconciled={pos.is_reconciled} diff={pos.reconciliation_diff} />
        </TableCell>
      </TableRow>

      <TableRow>
        <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={11}>
          <Collapse in={open} timeout="auto" unmountOnExit>
            <Box sx={{ m: 1.5 }}>
              <Typography variant="subtitle2" gutterBottom>Tax Lots</Typography>
              {lotsLoading ? <CircularProgress size={20} /> : (
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ backgroundColor: 'action.hover' }}>
                      <TableCell>Lot Ref</TableCell>
                      <TableCell>Acquisition Date</TableCell>
                      <TableCell align="right">Qty</TableCell>
                      <TableCell align="right">Cost/Unit</TableCell>
                      <TableCell align="right">Total Cost</TableCell>
                      <TableCell>Method</TableCell>
                      <TableCell align="center">Status</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {lots.map((lot) => (
                      <TableRow key={lot.id} hover>
                        <TableCell sx={{ fontFamily: 'monospace', fontSize: 11 }}>{lot.lot_reference ?? '—'}</TableCell>
                        <TableCell>{lot.acquisition_date?.slice(0, 10)}</TableCell>
                        <TableCell align="right">{fmtNum(lot.lot_quantity)}</TableCell>
                        <TableCell align="right">{fmtNum(lot.cost_per_unit, 4)}</TableCell>
                        <TableCell align="right">{fmtMoney(lot.total_cost_basis)}</TableCell>
                        <TableCell><Chip label={lot.lot_method} size="small" variant="outlined" /></TableCell>
                        <TableCell align="center">
                          {lot.is_closed
                            ? <Chip label="Closed" color="default" size="small" />
                            : <Chip label="Open" color="success" size="small" />}
                        </TableCell>
                      </TableRow>
                    ))}
                    {lots.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={7} align="center" sx={{ color: 'text.secondary', py: 2 }}>
                          No lots found
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              )}
            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
}

function HoldingsTab({ tenantId }: { tenantId: string }) {
  const [positions, setPositions] = useState<Position[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/positions?limit=100', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => { setPositions(d.positions ?? []); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [tenantId]);

  const filtered = positions.filter((p) =>
    p.security_id.toLowerCase().includes(search.toLowerCase()) ||
    p.position_source.toLowerCase().includes(search.toLowerCase()) ||
    p.position_side.toLowerCase().includes(search.toLowerCase()),
  );

  // Summary stats
  const totalMV = positions.reduce((s, p) => s + (p.market_value_local ?? 0), 0);
  const totalPL = positions.reduce((s, p) => s + (p.unrealized_pl_local ?? 0), 0);
  const unreconciledCount = positions.filter((p) => !p.is_reconciled).length;

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box>
      {/* Summary cards */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
        {[
          { label: 'Total Market Value', value: fmtMoney(totalMV), color: 'primary.main' },
          { label: 'Total Unrealized P&L', value: fmtMoney(totalPL), color: totalPL >= 0 ? 'success.main' : 'error.main' },
          { label: 'Positions', value: positions.length, color: 'text.primary' },
          { label: 'Unreconciled', value: unreconciledCount, color: unreconciledCount > 0 ? 'warning.main' : 'success.main' },
        ].map((s) => (
          <Card key={s.label} sx={{ flex: 1 }} elevation={0} variant="outlined">
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Typography variant="caption" color="text.secondary">{s.label}</Typography>
              <Typography variant="h6" fontWeight={700} sx={{ color: s.color }}>{s.value}</Typography>
            </CardContent>
          </Card>
        ))}
      </Box>

      <TextField
        fullWidth
        size="small"
        placeholder="Filter by security, source, or side…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment> }}
        sx={{ mb: 2 }}
      />

      <TableContainer component={Paper} elevation={0} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow sx={{ backgroundColor: 'action.hover' }}>
              <TableCell padding="checkbox" />
              <TableCell>Security ID</TableCell>
              <TableCell>Side</TableCell>
              <TableCell>Date</TableCell>
              <TableCell align="right">Qty</TableCell>
              <TableCell align="right">Market Value</TableCell>
              <TableCell align="right">Unrealized P&L</TableCell>
              <TableCell align="right">Weight</TableCell>
              <TableCell>Source</TableCell>
              <TableCell>Confidence</TableCell>
              <TableCell align="center">Recon</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((pos) => (
              <PositionRow key={pos.id} pos={pos} tenantId={tenantId} />
            ))}
            {filtered.length === 0 && (
              <TableRow>
                <TableCell colSpan={11} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                  No positions found
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}

function CashTab({ tenantId }: { tenantId: string }) {
  const [cash, setCash] = useState<CashPosition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/positions/cash?limit=50', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => { setCash(d.cash_positions ?? []); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [tenantId]);

  const totalCash = cash.reduce((s, c) => s + c.balance_amount, 0);

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box>
      <Box sx={{ mb: 2 }}>
        <Card elevation={0} variant="outlined" sx={{ display: 'inline-block', px: 3, py: 1.5 }}>
          <Typography variant="caption" color="text.secondary">Total Cash (All CCY)</Typography>
          <Typography variant="h5" fontWeight={700} color="primary.main">{fmtMoney(totalCash)}</Typography>
        </Card>
      </Box>

      <TableContainer component={Paper} elevation={0} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow sx={{ backgroundColor: 'action.hover' }}>
              <TableCell>Currency</TableCell>
              <TableCell>Value Date</TableCell>
              <TableCell align="right">Balance</TableCell>
              <TableCell align="right">Available</TableCell>
              <TableCell align="right">Interest Accrued</TableCell>
              <TableCell>Source</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {cash.map((c) => (
              <TableRow key={c.id} hover>
                <TableCell sx={{ fontWeight: 700, letterSpacing: 1 }}>{c.cash_currency}</TableCell>
                <TableCell>{c.value_date?.slice(0, 10)}</TableCell>
                <TableCell align="right" sx={{ fontWeight: 600 }}>{fmtMoney(c.balance_amount, c.cash_currency)}</TableCell>
                <TableCell align="right">
                  {c.available_balance != null ? fmtMoney(c.available_balance, c.cash_currency) : '—'}
                </TableCell>
                <TableCell align="right">{fmtMoney(c.interest_accrued, c.cash_currency)}</TableCell>
                <TableCell><Chip label={c.cash_source} size="small" variant="outlined" /></TableCell>
              </TableRow>
            ))}
            {cash.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                  No cash positions found
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}

function SnapshotsTab({ tenantId }: { tenantId: string }) {
  const [positions, setPositions] = useState<Position[]>([]);
  const [selected, setSelected] = useState<Position | null>(null);
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [snapLoading, setSnapLoading] = useState(false);

  useEffect(() => {
    fetch('/api/v1/positions?limit=20', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => {
        const list = d.positions ?? [];
        setPositions(list);
        if (list.length > 0) {
          setSelected(list[0]);
        }
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [tenantId]);

  useEffect(() => {
    if (!selected) return;
    setSnapLoading(true);
    fetch(`/api/v1/positions/${selected.id}/snapshots?limit=30`, {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => { setSnapshots(d.snapshots ?? []); setSnapLoading(false); })
      .catch(() => setSnapLoading(false));
  }, [selected, tenantId]);

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;

  return (
    <Box sx={{ display: 'flex', gap: 2 }}>
      {/* Position selector */}
      <Box sx={{ width: 240, flexShrink: 0 }}>
        {positions.map((p) => (
          <Card
            key={p.id}
            onClick={() => setSelected(p)}
            sx={{
              mb: 1, cursor: 'pointer',
              border: selected?.id === p.id ? '2px solid' : '1px solid',
              borderColor: selected?.id === p.id ? 'primary.main' : 'divider',
            }}
          >
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                {p.security_id.slice(0, 8)}…
              </Typography>
              <Typography variant="body2">{p.position_date?.slice(0, 10)}</Typography>
              <Chip label={p.position_source} size="small" sx={{ mt: 0.5 }} />
            </CardContent>
          </Card>
        ))}
      </Box>

      {/* Snapshot time-series */}
      {selected && (
        <Box sx={{ flex: 1 }}>
          <Typography variant="h6" gutterBottom>
            Historical Snapshots — {selected.security_id.slice(0, 8)}…
          </Typography>
          {snapLoading ? <CircularProgress size={24} /> : (
            <TableContainer component={Paper} elevation={0} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ backgroundColor: 'action.hover' }}>
                    <TableCell>Snapshot Date</TableCell>
                    <TableCell align="right">Quantity</TableCell>
                    <TableCell align="right">Market Value</TableCell>
                    <TableCell align="right">Price Used</TableCell>
                    <TableCell>Source</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {snapshots.map((s) => (
                    <TableRow key={s.id} hover>
                      <TableCell>{s.snapshot_date?.slice(0, 10)}</TableCell>
                      <TableCell align="right">{fmtNum(s.snapshot_quantity)}</TableCell>
                      <TableCell align="right">{fmtMoney(s.snapshot_market_value)}</TableCell>
                      <TableCell align="right" sx={{ fontFamily: 'monospace' }}>
                        {s.snapshot_price_used != null ? fmtNum(s.snapshot_price_used, 4) : '—'}
                      </TableCell>
                      <TableCell><Chip label={s.snapshot_source} size="small" variant="outlined" /></TableCell>
                    </TableRow>
                  ))}
                  {snapshots.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                        No snapshots yet — they are created automatically on each gold copy run
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Box>
      )}
    </Box>
  );
}

// ─── Main Dashboard ───────────────────────────────────────────────────────────

export interface PositionMasterDashboardProps {
  tenantId: string;
}

const tabs = [
  { label: 'Holdings',   icon: <AccountBalanceWalletIcon fontSize="small" /> },
  { label: 'Tax Lots',   icon: <LayersIcon fontSize="small" /> },
  { label: 'Cash',       icon: <PaymentsIcon fontSize="small" /> },
  { label: 'Snapshots',  icon: <HistoryIcon fontSize="small" /> },
];

export const PositionMasterDashboard: React.FC<PositionMasterDashboardProps> = ({ tenantId }) => {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
        <AccountBalanceWalletIcon sx={{ fontSize: 36, color: 'primary.main' }} />
        <Box>
          <Typography variant="h4" fontWeight={700}>Position Master</Typography>
          <Typography variant="body2" color="text.secondary">
            Book of record — holdings, tax lots, cash balances, and historical snapshots
          </Typography>
        </Box>
      </Box>

      <Divider sx={{ mb: 3 }} />

      <Tabs
        value={activeTab}
        onChange={(_, v) => setActiveTab(v)}
        sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
      >
        {tabs.map((t, i) => (
          <Tab
            key={i}
            label={t.label}
            icon={t.icon}
            iconPosition="start"
            sx={{ minHeight: 48, textTransform: 'none', fontWeight: 600 }}
          />
        ))}
      </Tabs>

      {activeTab === 0 && <HoldingsTab tenantId={tenantId} />}
      {activeTab === 1 && (
        <Alert severity="info" sx={{ mb: 2 }}>
          Click the expand arrow on any position in the <strong>Holdings</strong> tab to drill into its tax lots.
        </Alert>
      )}
      {activeTab === 2 && <CashTab tenantId={tenantId} />}
      {activeTab === 3 && <SnapshotsTab tenantId={tenantId} />}
    </Box>
  );
};

export default PositionMasterDashboard;
