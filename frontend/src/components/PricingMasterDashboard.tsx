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
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AttachMoneyIcon from '@mui/icons-material/AttachMoney';
import CurrencyExchangeIcon from '@mui/icons-material/CurrencyExchange';
import TimelineIcon from '@mui/icons-material/Timeline';
import GridOnIcon from '@mui/icons-material/GridOn';
import SignalCellularAltIcon from '@mui/icons-material/SignalCellularAlt';

// ─── Types ──────────────────────────────────────────────────────────────────

interface Price {
  id: string;
  security_id: string;
  price_type: string;
  price_date: string;
  price_value: number;
  price_currency: string;
  price_source: string;
  price_confidence: number;
  is_stale_price: boolean;
  is_composite_price: boolean;
}

interface FXRate {
  id: string;
  base_currency: string;
  quote_currency: string;
  fx_rate_date: string;
  fx_tenor: string;
  fx_rate: number;
  fx_source: string;
  fx_confidence: number;
}

interface TenorPoint {
  tenor: string;
  rate: number;
  discount_factor?: number;
}

interface Curve {
  id: string;
  curve_type: string;
  curve_currency: string;
  curve_as_of_date: string;
  curve_source: string;
  curve_tenor_points: TenorPoint[];
  curve_interpolation?: string;
  curve_confidence: number;
}

interface VolSurface {
  id: string;
  underlier_security_id: string;
  vol_surface_type: string;
  vol_as_of_date: string;
  vol_source: string;
  vol_grid: {
    strikes: number[];
    tenors: string[];
    vols: number[][];
  };
  vol_confidence: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const fmt = new Intl.NumberFormat('en-US');
const fmtPct = (v: number) => `${(v * 100).toFixed(4)}%`;
const fmtRate = (v: number, decimals = 6) => v.toFixed(decimals);

function ConfidenceChip({ score }: { score: number }) {
  const color = score >= 90 ? 'success' : score >= 70 ? 'warning' : 'error';
  return (
    <Chip
      label={`${score}%`}
      color={color}
      size="small"
      icon={<SignalCellularAltIcon style={{ fontSize: 14 }} />}
      sx={{ fontWeight: 'bold', minWidth: 68 }}
    />
  );
}

function SourceBadge({ source }: { source: string }) {
  const colorMap: Record<string, 'primary' | 'info' | 'default' | 'secondary'> = {
    Bloomberg: 'primary',
    Refinitiv: 'info',
    FactSet: 'secondary',
  };
  return (
    <Chip
      label={source}
      color={colorMap[source] ?? 'default'}
      variant="outlined"
      size="small"
    />
  );
}

// ─── Tab content components ───────────────────────────────────────────────────

function PricesTab({ tenantId }: { tenantId: string }) {
  const [prices, setPrices] = useState<Price[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/pricing/prices?limit=100', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => { setPrices(d.prices ?? []); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [tenantId]);

  const filtered = prices.filter(
    (p) =>
      p.security_id.toLowerCase().includes(search.toLowerCase()) ||
      p.price_type.toLowerCase().includes(search.toLowerCase()) ||
      p.price_source.toLowerCase().includes(search.toLowerCase()),
  );

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box>
      <TextField
        fullWidth
        size="small"
        placeholder="Filter by security, type or source…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment> }}
        sx={{ mb: 2 }}
      />
      <TableContainer component={Paper} elevation={0} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow sx={{ backgroundColor: 'action.hover' }}>
              <TableCell>Security ID</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Date</TableCell>
              <TableCell align="right">Price</TableCell>
              <TableCell>CCY</TableCell>
              <TableCell>Source</TableCell>
              <TableCell align="center">Confidence</TableCell>
              <TableCell align="center">Flags</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((p) => (
              <TableRow key={p.id} hover>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {p.security_id.slice(0, 8)}…
                </TableCell>
                <TableCell>
                  <Chip label={p.price_type} size="small" variant="outlined" />
                </TableCell>
                <TableCell>{p.price_date?.slice(0, 10)}</TableCell>
                <TableCell align="right" sx={{ fontWeight: 600 }}>
                  {fmt.format(p.price_value)}
                </TableCell>
                <TableCell>{p.price_currency}</TableCell>
                <TableCell><SourceBadge source={p.price_source} /></TableCell>
                <TableCell align="center">
                  <ConfidenceChip score={p.price_confidence} />
                </TableCell>
                <TableCell align="center">
                  {p.is_stale_price && <Chip label="Stale" color="error" size="small" sx={{ mr: 0.5 }} />}
                  {p.is_composite_price && <Chip label="Composite" color="info" size="small" />}
                </TableCell>
              </TableRow>
            ))}
            {filtered.length === 0 && (
              <TableRow>
                <TableCell colSpan={8} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                  No prices found
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}

function FXRatesTab({ tenantId }: { tenantId: string }) {
  const [rates, setRates] = useState<FXRate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/pricing/fx-rates?limit=100', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => { setRates(d.fx_rates ?? []); setLoading(false); })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [tenantId]);

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <TableContainer component={Paper} elevation={0} variant="outlined">
      <Table size="small">
        <TableHead>
          <TableRow sx={{ backgroundColor: 'action.hover' }}>
            <TableCell>Pair</TableCell>
            <TableCell>Tenor</TableCell>
            <TableCell>Date</TableCell>
            <TableCell align="right">Rate</TableCell>
            <TableCell>Source</TableCell>
            <TableCell align="center">Confidence</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rates.map((r) => (
            <TableRow key={r.id} hover>
              <TableCell sx={{ fontWeight: 700, letterSpacing: 1 }}>
                {r.base_currency}/{r.quote_currency}
              </TableCell>
              <TableCell>
                <Chip label={r.fx_tenor} size="small" variant="outlined" />
              </TableCell>
              <TableCell>{r.fx_rate_date?.slice(0, 10)}</TableCell>
              <TableCell align="right" sx={{ fontFamily: 'monospace', fontWeight: 600 }}>
                {fmtRate(r.fx_rate, 5)}
              </TableCell>
              <TableCell><SourceBadge source={r.fx_source} /></TableCell>
              <TableCell align="center">
                <ConfidenceChip score={r.fx_confidence} />
              </TableCell>
            </TableRow>
          ))}
          {rates.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                No FX rates found
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function CurvesTab({ tenantId }: { tenantId: string }) {
  const [curves, setCurves] = useState<Curve[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Curve | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/pricing/curves?limit=50', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => {
        const list = d.curves ?? [];
        setCurves(list);
        if (list.length > 0) setSelected(list[0]);
        setLoading(false);
      })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [tenantId]);

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box sx={{ display: 'flex', gap: 2 }}>
      {/* Curve list */}
      <Box sx={{ width: 240, flexShrink: 0 }}>
        {curves.map((c) => (
          <Card
            key={c.id}
            onClick={() => setSelected(c)}
            sx={{
              mb: 1, cursor: 'pointer',
              border: selected?.id === c.id ? '2px solid' : '1px solid',
              borderColor: selected?.id === c.id ? 'primary.main' : 'divider',
            }}
          >
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Typography variant="subtitle2" fontWeight={700}>{c.curve_type}</Typography>
              <Typography variant="caption" color="text.secondary">
                {c.curve_currency} · {c.curve_as_of_date?.slice(0, 10)}
              </Typography>
              <Box sx={{ mt: 0.5 }}>
                <ConfidenceChip score={c.curve_confidence} />
              </Box>
            </CardContent>
          </Card>
        ))}
        {curves.length === 0 && (
          <Typography color="text.secondary" variant="body2">No curves available</Typography>
        )}
      </Box>

      {/* Tenor point detail */}
      {selected && (
        <Box sx={{ flex: 1 }}>
          <Typography variant="h6" gutterBottom>
            {selected.curve_type} ({selected.curve_currency}) — {selected.curve_as_of_date?.slice(0, 10)}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, mb: 2, flexWrap: 'wrap' }}>
            <Chip label={`Source: ${selected.curve_source}`} size="small" />
            {selected.curve_interpolation && <Chip label={`Interp: ${selected.curve_interpolation}`} size="small" variant="outlined" />}
            <ConfidenceChip score={selected.curve_confidence} />
          </Box>
          <TableContainer component={Paper} elevation={0} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell>Tenor</TableCell>
                  <TableCell align="right">Rate</TableCell>
                  <TableCell align="right">Rate %</TableCell>
                  <TableCell align="right">Discount Factor</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {(selected.curve_tenor_points ?? []).map((tp, i) => (
                  <TableRow key={i} hover>
                    <TableCell sx={{ fontWeight: 600 }}>{tp.tenor}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace' }}>
                      {fmtRate(tp.rate, 6)}
                    </TableCell>
                    <TableCell align="right">{fmtPct(tp.rate)}</TableCell>
                    <TableCell align="right" sx={{ fontFamily: 'monospace' }}>
                      {tp.discount_factor != null ? fmtRate(tp.discount_factor, 4) : '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}
    </Box>
  );
}

function VolSurfacesTab({ tenantId }: { tenantId: string }) {
  const [surfaces, setSurfaces] = useState<VolSurface[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<VolSurface | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/pricing/vol-surfaces?limit=50', {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((d) => {
        const list = d.vol_surfaces ?? [];
        setSurfaces(list);
        if (list.length > 0) setSelected(list[0]);
        setLoading(false);
      })
      .catch((e) => { setError(e.message); setLoading(false); });
  }, [tenantId]);

  if (loading) return <Box sx={{ py: 4, textAlign: 'center' }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box sx={{ display: 'flex', gap: 2 }}>
      {/* Surface list */}
      <Box sx={{ width: 240, flexShrink: 0 }}>
        {surfaces.map((s) => (
          <Card
            key={s.id}
            onClick={() => setSelected(s)}
            sx={{
              mb: 1, cursor: 'pointer',
              border: selected?.id === s.id ? '2px solid' : '1px solid',
              borderColor: selected?.id === s.id ? 'primary.main' : 'divider',
            }}
          >
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Typography variant="subtitle2" fontWeight={700}>{s.vol_surface_type}</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace', fontSize: 11 }}>
                {s.underlier_security_id.slice(0, 8)}…
              </Typography>
              <Typography variant="caption" display="block" color="text.secondary">
                {s.vol_as_of_date?.slice(0, 10)}
              </Typography>
              <Box sx={{ mt: 0.5 }}>
                <ConfidenceChip score={s.vol_confidence} />
              </Box>
            </CardContent>
          </Card>
        ))}
        {surfaces.length === 0 && (
          <Typography color="text.secondary" variant="body2">No vol surfaces available</Typography>
        )}
      </Box>

      {/* Grid display */}
      {selected && selected.vol_grid && (
        <Box sx={{ flex: 1, overflowX: 'auto' }}>
          <Typography variant="h6" gutterBottom>
            {selected.vol_surface_type} Vol Surface — {selected.vol_as_of_date?.slice(0, 10)}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
            <Chip label={`Source: ${selected.vol_source}`} size="small" />
            <ConfidenceChip score={selected.vol_confidence} />
          </Box>
          <TableContainer component={Paper} elevation={0} variant="outlined">
            <Table size="small" stickyHeader>
              <TableHead>
                <TableRow sx={{ backgroundColor: 'action.hover' }}>
                  <TableCell sx={{ fontWeight: 700 }}>Tenor \ Strike</TableCell>
                  {(selected.vol_grid.strikes ?? []).map((k) => (
                    <TableCell key={k} align="center" sx={{ fontWeight: 600 }}>
                      {(k * 100).toFixed(0)}%
                    </TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {(selected.vol_grid.tenors ?? []).map((tenor, ti) => (
                  <TableRow key={tenor} hover>
                    <TableCell sx={{ fontWeight: 600 }}>{tenor}</TableCell>
                    {(selected.vol_grid.vols?.[ti] ?? []).map((vol, ki) => {
                      const pct = vol * 100;
                      const intensity = Math.min(1, vol / 0.5);
                      return (
                        <Tooltip key={ki} title={`Vol: ${fmtPct(vol)}`} arrow>
                          <TableCell
                            align="center"
                            sx={{
                              fontFamily: 'monospace',
                              backgroundColor: `rgba(33, 150, 243, ${intensity * 0.3})`,
                              fontWeight: 500,
                            }}
                          >
                            {pct.toFixed(1)}%
                          </TableCell>
                        </Tooltip>
                      );
                    })}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}
    </Box>
  );
}

// ─── Main Dashboard ───────────────────────────────────────────────────────────

export interface PricingMasterDashboardProps {
  tenantId: string;
}

const tabs = [
  { label: 'Prices',       icon: <AttachMoneyIcon fontSize="small" /> },
  { label: 'FX Rates',     icon: <CurrencyExchangeIcon fontSize="small" /> },
  { label: 'Curves',       icon: <TimelineIcon fontSize="small" /> },
  { label: 'Vol Surfaces', icon: <GridOnIcon fontSize="small" /> },
];

export const PricingMasterDashboard: React.FC<PricingMasterDashboardProps> = ({ tenantId }) => {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
        <AttachMoneyIcon sx={{ fontSize: 36, color: 'primary.main' }} />
        <Box>
          <Typography variant="h4" fontWeight={700}>
            Pricing Master
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Gold-copy prices, FX rates, rate curves and volatility surfaces
          </Typography>
        </Box>
      </Box>

      <Divider sx={{ mb: 3 }} />

      {/* Tabs */}
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

      {/* Tab panels */}
      {activeTab === 0 && <PricesTab tenantId={tenantId} />}
      {activeTab === 1 && <FXRatesTab tenantId={tenantId} />}
      {activeTab === 2 && <CurvesTab tenantId={tenantId} />}
      {activeTab === 3 && <VolSurfacesTab tenantId={tenantId} />}
    </Box>
  );
};

export default PricingMasterDashboard;
