import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Chip,
  Divider,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  TextField,
  InputAdornment,
  Tabs,
  Tab,
  Card,
  CardContent,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import { alpha } from '@mui/material/styles';
import {
  Search as SearchIcon,
  ShowChart as EquityIcon,
  AccountBalance as BondIcon,
  PieChart as FundIcon,
  TrendingUp as DerivIcon,
  VerifiedUser as GoldIcon,
  Warning as WarnIcon,
  Timeline as LineageIcon,
  CompareArrows as CompareIcon,
  Tune as AttrIcon,
  Info as InfoIcon,
} from '@mui/icons-material';

// ─── Types ────────────────────────────────────────────────────────────────────

type AssetClass = 'Equity' | 'FixedIncome' | 'Fund' | 'Derivative';
type SecurityStatus = 'Gold' | 'Warning' | 'Error';

interface Security {
  id: string;
  securityId: string;
  isin: string;
  cusip?: string;
  figi?: string;
  ticker: string;
  name: string;
  shortName: string;
  assetClass: AssetClass;
  subAssetClass: string;
  currency: string;
  country: string;
  sector: string;
  issuerName: string;
  exchange: string;
  status: SecurityStatus;
  confidence: number;
  sources: string[];
}

interface FixedIncomeDetails {
  couponType: string;
  couponRate: number;
  couponFreq: string;
  maturityDate: string;
  issueDate: string;
  parValue: number;
  seniority: string;
  ratingComposite: string;
  dayCount: string;
}

interface EquityDetails {
  shareClass: string;
  sharesOutstanding: string;
  freeFloat: string;
  dividendYield: string;
  dividendFreq: string;
}

interface FundDetails {
  fundType: string;
  managementCo: string;
  administrator: string;
  ter: string;
  distributionPolicy: string;
  domicile: string;
}

interface DerivativeDetails {
  underlierTicker: string;
  underlierType: string;
  contractSize: number;
  strikePrice: number;
  optionType: string;
  exerciseStyle: string;
  expiryDate: string;
}

// ─── Mock Data ────────────────────────────────────────────────────────────────

const MOCK_SECURITIES: Security[] = [
  {
    id: '1', securityId: 'SEC-AAPL', isin: 'US0378331005', cusip: '037833100',
    figi: 'BBG000B9XRY4', ticker: 'AAPL', name: 'Apple Inc.', shortName: 'Apple',
    assetClass: 'Equity', subAssetClass: 'Large Cap', currency: 'USD',
    country: 'USA', sector: 'Technology', issuerName: 'Apple Inc.',
    exchange: 'NASDAQ', status: 'Gold', confidence: 98, sources: ['Bloomberg', 'Refinitiv'],
  },
  {
    id: '2', securityId: 'SEC-GS-4.25-27', isin: 'US38141GXB92', cusip: '38141GXB9',
    ticker: 'GS', name: 'Goldman Sachs 4.25% 2027', shortName: 'GS 4.25 27',
    assetClass: 'FixedIncome', subAssetClass: 'IG Corporate', currency: 'USD',
    country: 'USA', sector: 'Financials', issuerName: 'Goldman Sachs Group',
    exchange: 'NYSE', status: 'Gold', confidence: 95, sources: ['Bloomberg', 'Refinitiv', 'ICE'],
  },
  {
    id: '3', securityId: 'SEC-IVV', isin: 'US4642872422', cusip: '464287242',
    ticker: 'IVV', name: 'iShares Core S&P 500 ETF', shortName: 'iShares S&P500',
    assetClass: 'Fund', subAssetClass: 'Passive ETF', currency: 'USD',
    country: 'USA', sector: 'Diversified', issuerName: 'BlackRock',
    exchange: 'NYSE Arca', status: 'Gold', confidence: 97, sources: ['Bloomberg', 'Admin'],
  },
  {
    id: '4', securityId: 'SEC-AAPL-C-200-JUN25', isin: '', cusip: '',
    figi: 'BBG00XYZ1234', ticker: 'AAPL US 06/20/25 C200', name: 'AAPL Jun25 Call 200',
    shortName: 'AAPL C200 Jun25', assetClass: 'Derivative', subAssetClass: 'Equity Option',
    currency: 'USD', country: 'USA', sector: 'Technology', issuerName: 'Apple Inc.',
    exchange: 'CBOE', status: 'Warning', confidence: 88, sources: ['Bloomberg', 'ICE'],
  },
  {
    id: '5', securityId: 'SEC-MSFT', isin: 'US5949181045', cusip: '594918104',
    ticker: 'MSFT', name: 'Microsoft Corporation', shortName: 'Microsoft',
    assetClass: 'Equity', subAssetClass: 'Large Cap', currency: 'USD',
    country: 'USA', sector: 'Technology', issuerName: 'Microsoft Corp.',
    exchange: 'NASDAQ', status: 'Gold', confidence: 99, sources: ['Bloomberg', 'Refinitiv', 'ICE'],
  },
];

const MOCK_FI_DETAILS: FixedIncomeDetails = {
  couponType: 'Fixed', couponRate: 4.25, couponFreq: 'Semi-Annual',
  maturityDate: '2027-10-21', issueDate: '2020-10-21', parValue: 1000.0,
  seniority: 'Senior Unsecured', ratingComposite: 'A+', dayCount: '30/360',
};

const MOCK_EQ_DETAILS: EquityDetails = {
  shareClass: 'Common', sharesOutstanding: '15.44B', freeFloat: '99.8%',
  dividendYield: '0.54%', dividendFreq: 'Quarterly',
};

const MOCK_FUND_DETAILS: FundDetails = {
  fundType: 'ETF', managementCo: 'BlackRock Fund Advisors',
  administrator: 'BlackRock', ter: '0.03%', distributionPolicy: 'Distributing', domicile: 'USA',
};

const MOCK_DERIV_DETAILS: DerivativeDetails = {
  underlierTicker: 'AAPL', underlierType: 'Single Name', contractSize: 100,
  strikePrice: 200.0, optionType: 'Call', exerciseStyle: 'American', expiryDate: '2025-06-20',
};

const MOCK_LINEAGE = [
  { field: 'isin', value: 'US0378331005', source: 'Bloomberg', rule: 'prefer_source', confidence: 10 },
  { field: 'security_name', value: 'Apple Inc.', source: 'Bloomberg', rule: 'prefer_source', confidence: 9 },
  { field: 'asset_class', value: 'Equity', source: 'Bloomberg', rule: 'prefer_source', confidence: 10 },
  { field: 'currency', value: 'USD', source: 'Bloomberg', rule: 'prefer_source', confidence: 10 },
  { field: 'listing_exchange', value: 'NASDAQ', source: 'Bloomberg', rule: 'prefer_source', confidence: 9 },
  { field: 'sector', value: 'Technology', source: 'Refinitiv', rule: 'highest_quality', confidence: 8 },
];

// ─── Helpers ──────────────────────────────────────────────────────────────────

const assetClassIcon = (ac: AssetClass) => {
  switch (ac) {
    case 'Equity':      return <EquityIcon fontSize="small" />;
    case 'FixedIncome': return <BondIcon fontSize="small" />;
    case 'Fund':        return <FundIcon fontSize="small" />;
    case 'Derivative':  return <DerivIcon fontSize="small" />;
  }
};

const assetClassColor = (ac: AssetClass): string => {
  switch (ac) {
    case 'Equity':      return '#2196f3';
    case 'FixedIncome': return '#9c27b0';
    case 'Fund':        return '#00897b';
    case 'Derivative':  return '#ef6c00';
  }
};

const statusChip = (status: SecurityStatus, confidence: number) => {
  const color = status === 'Gold' ? '#FFD700' : status === 'Warning' ? '#ff9800' : '#f44336';
  const bg = status === 'Gold' ? 'rgba(255,215,0,0.12)' : status === 'Warning' ? 'rgba(255,152,0,0.12)' : 'rgba(244,67,54,0.12)';
  return (
    <Chip
      size="small"
      icon={status === 'Gold' ? <GoldIcon sx={{ color: `${color} !important`, fontSize: 14 }} /> : <WarnIcon sx={{ color: `${color} !important`, fontSize: 14 }} />}
      label={`${status} · ${confidence}%`}
      sx={{ bgcolor: bg, color, fontWeight: 600, fontSize: 11 }}
    />
  );
};

// ─── Sub-panels ───────────────────────────────────────────────────────────────

function CoreAttributesPanel({ sec }: { sec: Security }) {
  const rows = [
    ['ISIN', sec.isin || '—'], ['CUSIP', sec.cusip || '—'], ['FIGI', sec.figi || '—'],
    ['Ticker', sec.ticker], ['Asset Class', sec.assetClass], ['Sub-type', sec.subAssetClass],
    ['Currency', sec.currency], ['Country', sec.country], ['Sector', sec.sector],
    ['Exchange', sec.exchange], ['Issuer', sec.issuerName],
    ['Sources', sec.sources.join(', ')],
  ];
  return (
    <TableContainer>
      <Table size="small">
        <TableBody>
          {rows.map(([label, value]) => (
            <TableRow key={label} sx={{ '&:last-child td': { border: 0 } }}>
              <TableCell sx={{ color: 'text.secondary', width: 160, fontWeight: 500, fontSize: 12 }}>{label}</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 13 }}>{value}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function FixedIncomePanel({ d }: { d: FixedIncomeDetails }) {
  const rows = [
    ['Coupon Type', d.couponType],
    ['Coupon Rate', `${d.couponRate}%`],
    ['Coupon Frequency', d.couponFreq],
    ['Day Count', d.dayCount],
    ['Issue Date', d.issueDate],
    ['Maturity Date', d.maturityDate],
    ['Par Value', `$${d.parValue.toLocaleString()}`],
    ['Seniority', d.seniority],
    ['Rating (Composite)', d.ratingComposite],
  ];
  return (
    <TableContainer>
      <Table size="small">
        <TableBody>
          {rows.map(([l, v]) => (
            <TableRow key={l} sx={{ '&:last-child td': { border: 0 } }}>
              <TableCell sx={{ color: 'text.secondary', width: 200, fontWeight: 500, fontSize: 12 }}>{l}</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 13 }}>{v}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function EquityPanel({ d }: { d: EquityDetails }) {
  const rows = [
    ['Share Class', d.shareClass],
    ['Shares Outstanding', d.sharesOutstanding],
    ['Free Float', d.freeFloat],
    ['Dividend Yield', d.dividendYield],
    ['Dividend Frequency', d.dividendFreq],
  ];
  return (
    <TableContainer>
      <Table size="small">
        <TableBody>
          {rows.map(([l, v]) => (
            <TableRow key={l} sx={{ '&:last-child td': { border: 0 } }}>
              <TableCell sx={{ color: 'text.secondary', width: 200, fontWeight: 500, fontSize: 12 }}>{l}</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 13 }}>{v}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function FundPanel({ d }: { d: FundDetails }) {
  const rows = [
    ['Fund Type', d.fundType],
    ['Management Company', d.managementCo],
    ['Administrator', d.administrator],
    ['Domicile', d.domicile],
    ['Total Expense Ratio', d.ter],
    ['Distribution Policy', d.distributionPolicy],
  ];
  return (
    <TableContainer>
      <Table size="small">
        <TableBody>
          {rows.map(([l, v]) => (
            <TableRow key={l} sx={{ '&:last-child td': { border: 0 } }}>
              <TableCell sx={{ color: 'text.secondary', width: 200, fontWeight: 500, fontSize: 12 }}>{l}</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 13 }}>{v}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function DerivativePanel({ d }: { d: DerivativeDetails }) {
  const rows = [
    ['Underlier', d.underlierTicker],
    ['Underlier Type', d.underlierType],
    ['Contract Size', d.contractSize.toLocaleString()],
    ['Strike Price', `$${d.strikePrice.toFixed(2)}`],
    ['Option Type', d.optionType],
    ['Exercise Style', d.exerciseStyle],
    ['Expiry Date', d.expiryDate],
  ];
  return (
    <TableContainer>
      <Table size="small">
        <TableBody>
          {rows.map(([l, v]) => (
            <TableRow key={l} sx={{ '&:last-child td': { border: 0 } }}>
              <TableCell sx={{ color: 'text.secondary', width: 200, fontWeight: 500, fontSize: 12 }}>{l}</TableCell>
              <TableCell sx={{ fontFamily: 'monospace', fontSize: 13 }}>{v}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function LineagePanel({ security }: { security: Security }) {
  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Field-level survivorship lineage for the gold copy record. Shows which source system won each field.
      </Typography>
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: 'action.hover' }}>
              <TableCell sx={{ fontWeight: 700, fontSize: 11 }}>FIELD</TableCell>
              <TableCell sx={{ fontWeight: 700, fontSize: 11 }}>CHOSEN VALUE</TableCell>
              <TableCell sx={{ fontWeight: 700, fontSize: 11 }}>SOURCE</TableCell>
              <TableCell sx={{ fontWeight: 700, fontSize: 11 }}>RULE</TableCell>
              <TableCell sx={{ fontWeight: 700, fontSize: 11 }} align="right">CONFIDENCE</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {MOCK_LINEAGE.map((row) => (
              <TableRow key={row.field} hover>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>{row.field}</TableCell>
                <TableCell sx={{ fontSize: 12 }}>{row.value}</TableCell>
                <TableCell>
                  <Chip label={row.source} size="small" sx={{ fontSize: 10 }} color="primary" variant="outlined" />
                </TableCell>
                <TableCell sx={{ fontSize: 11, color: 'text.secondary' }}>{row.rule}</TableCell>
                <TableCell align="right">
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, justifyContent: 'flex-end' }}>
                    <LinearProgress
                      variant="determinate"
                      value={row.confidence * 10}
                      sx={{ width: 60, height: 6, borderRadius: 3 }}
                    />
                    <Typography variant="caption">{row.confidence}/10</Typography>
                  </Box>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      {security.status === 'Warning' && (
        <Paper variant="outlined" sx={{ p: 2, mt: 2, borderColor: 'warning.main', bgcolor: 'rgba(255,152,0,0.06)' }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <WarnIcon sx={{ color: 'warning.main', fontSize: 18 }} />
            <Typography variant="body2" color="warning.main" fontWeight={600}>DQ Warning: Missing ISIN identifier</Typography>
          </Stack>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
            Rule: Security_RequiredIdentifiers · Severity: Soft · At least one cross-reference identifier recommended.
          </Typography>
        </Paper>
      )}
    </Box>
  );
}

// ─── Detail Panel ─────────────────────────────────────────────────────────────

function SecurityDetailPanel({ security }: { security: Security }) {
  const [tab, setTab] = useState(0);
  const color = assetClassColor(security.assetClass);

  const subtypeTab = () => {
    switch (security.assetClass) {
      case 'FixedIncome': return <FixedIncomePanel d={MOCK_FI_DETAILS} />;
      case 'Equity':      return <EquityPanel d={MOCK_EQ_DETAILS} />;
      case 'Fund':        return <FundPanel d={MOCK_FUND_DETAILS} />;
      case 'Derivative':  return <DerivativePanel d={MOCK_DERIV_DETAILS} />;
    }
  };

  const subtypeLabel = () => {
    switch (security.assetClass) {
      case 'FixedIncome': return 'Bond Attributes';
      case 'Equity':      return 'Equity Attributes';
      case 'Fund':        return 'Fund Attributes';
      case 'Derivative':  return 'Derivative Attributes';
    }
  };

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', p: 3, gap: 2 }}>
      {/* Header */}
      <Stack direction="row" alignItems="flex-start" justifyContent="space-between">
        <Box>
          <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
            <Box sx={{ color, display: 'flex' }}>{assetClassIcon(security.assetClass)}</Box>
            <Chip label={security.assetClass} size="small" sx={{ bgcolor: alpha(color, 0.12), color, fontWeight: 600, fontSize: 10 }} />
            <Chip label={security.subAssetClass} size="small" variant="outlined" sx={{ fontSize: 10 }} />
          </Stack>
          <Typography variant="h6" fontWeight={700}>{security.name}</Typography>
          <Typography variant="body2" color="text.secondary">{security.ticker} · {security.isin || security.figi}</Typography>
        </Box>
        {statusChip(security.status, security.confidence)}
      </Stack>

      {/* KPI Row */}
      <Grid container spacing={1.5}>
        {[
          { label: 'Currency', value: security.currency },
          { label: 'Exchange', value: security.exchange },
          { label: 'Country', value: security.country },
          { label: 'Sources', value: `${security.sources.length} feeds` },
        ].map((kpi) => (
          <Grid key={kpi.label} size={{ xs: 6, sm: 3 }}>
            <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
              <Typography variant="caption" color="text.secondary" display="block">{kpi.label}</Typography>
              <Typography variant="body2" fontWeight={700}>{kpi.value}</Typography>
            </Paper>
          </Grid>
        ))}
      </Grid>

      {/* Tabs */}
      <Tabs value={tab} onChange={(_, v) => setTab(v)} variant="scrollable" scrollButtons="auto" sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tab icon={<InfoIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="Core" sx={{ minHeight: 40, fontSize: 12 }} />
        <Tab icon={<AttrIcon sx={{ fontSize: 14 }} />} iconPosition="start" label={subtypeLabel()} sx={{ minHeight: 40, fontSize: 12 }} />
        <Tab icon={<LineageIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="Lineage" sx={{ minHeight: 40, fontSize: 12 }} />
        <Tab icon={<CompareIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="Source Compare" sx={{ minHeight: 40, fontSize: 12 }} />
      </Tabs>

      <Box sx={{ flex: 1, overflowY: 'auto' }}>
        {tab === 0 && <CoreAttributesPanel sec={security} />}
        {tab === 1 && subtypeTab()}
        {tab === 2 && <LineagePanel security={security} />}
        {tab === 3 && (
          <Box sx={{ mt: 1 }}>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Compare raw values from each source system side-by-side to understand divergence and survivorship decisions.
            </Typography>
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell sx={{ fontWeight: 700, fontSize: 11 }}>FIELD</TableCell>
                    {security.sources.map(s => <TableCell key={s} sx={{ fontWeight: 700, fontSize: 11 }}>{s.toUpperCase()}</TableCell>)}
                    <TableCell sx={{ fontWeight: 700, fontSize: 11, color: 'warning.main' }}>GOLD COPY</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {['security_name','currency','sector','country_of_issue'].map(field => (
                    <TableRow key={field} hover>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>{field}</TableCell>
                      {security.sources.map(s => (
                        <TableCell key={s} sx={{ fontSize: 12, color: 'text.secondary' }}>
                          {MOCK_LINEAGE.find(r => r.field === field)?.value || 'N/A'}
                        </TableCell>
                      ))}
                      <TableCell sx={{ fontSize: 12, fontWeight: 600 }}>
                        {MOCK_LINEAGE.find(r => r.field === field)?.value || '—'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}
      </Box>
    </Box>
  );
}

// ─── Main Dashboard ───────────────────────────────────────────────────────────

const SecurityMasterDashboard: React.FC = () => {
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string>('1');
  const [assetFilter, setAssetFilter] = useState<AssetClass | 'All'>('All');

  const filtered = MOCK_SECURITIES.filter(s => {
    const q = search.toLowerCase();
    const matchSearch = !q || s.name.toLowerCase().includes(q) || s.isin.toLowerCase().includes(q) || s.ticker.toLowerCase().includes(q) || s.securityId.toLowerCase().includes(q);
    const matchClass = assetFilter === 'All' || s.assetClass === assetFilter;
    return matchSearch && matchClass;
  });

  const selected = MOCK_SECURITIES.find(s => s.id === selectedId) ?? MOCK_SECURITIES[0];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: 'background.default' }}>
      {/* Top bar */}
      <Box sx={{ px: 3, py: 2, borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper' }}>
        <Stack direction="row" spacing={2} alignItems="center" justifyContent="space-between">
          <Box>
            <Typography variant="h5" fontWeight={800}>Security Master</Typography>
            <Typography variant="caption" color="text.secondary">Institutional Gold Copy · {MOCK_SECURITIES.length} instruments</Typography>
          </Box>
          <Stack direction="row" spacing={1}>
            {(['All', 'Equity', 'FixedIncome', 'Fund', 'Derivative'] as const).map(ac => {
              const count = ac === 'All' ? MOCK_SECURITIES.length : MOCK_SECURITIES.filter(s => s.assetClass === ac).length;
              const color = ac === 'All' ? undefined : assetClassColor(ac as AssetClass);
              return (
                <Chip
                  key={ac}
                  label={`${ac === 'FixedIncome' ? 'Fixed Income' : ac}  ${count}`}
                  size="small"
                  onClick={() => setAssetFilter(ac)}
                  sx={{
                    fontWeight: 600,
                    fontSize: 11,
                    bgcolor: assetFilter === ac ? (color ? alpha(color, 0.15) : 'action.selected') : 'transparent',
                    borderColor: assetFilter === ac ? (color || 'primary.main') : 'divider',
                    color: assetFilter === ac ? (color || 'primary.main') : 'text.secondary',
                    border: '1px solid',
                  }}
                />
              );
            })}
          </Stack>
        </Stack>
      </Box>

      {/* Body: left sidebar + right detail */}
      <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Sidebar */}
        <Paper
          elevation={0}
          sx={{ width: 310, flexShrink: 0, borderRight: 1, borderColor: 'divider', display: 'flex', flexDirection: 'column', overflowY: 'hidden' }}
        >
          <Box sx={{ p: 1.5 }}>
            <TextField
              fullWidth
              size="small"
              placeholder="Search by name, ISIN, ticker…"
              value={search}
              onChange={e => setSearch(e.target.value)}
              InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ fontSize: 18, color: 'text.disabled' }} /></InputAdornment> }}
            />
          </Box>
          <Divider />
          <List dense disablePadding sx={{ flex: 1, overflowY: 'auto' }}>
            {filtered.map(sec => {
              const isSelected = sec.id === selectedId;
              const color = assetClassColor(sec.assetClass);
              return (
                <React.Fragment key={sec.id}>
                  <ListItem disablePadding>
                    <ListItemButton
                      selected={isSelected}
                      onClick={() => setSelectedId(sec.id)}
                      sx={{
                        py: 1.5,
                        bgcolor: isSelected ? alpha(color, 0.08) : 'transparent',
                        '&:hover': { bgcolor: alpha(color, 0.05) },
                        '&.Mui-selected': { bgcolor: alpha(color, 0.1) },
                        borderLeft: isSelected ? `3px solid ${color}` : '3px solid transparent',
                      }}
                    >
                      <Box sx={{ mr: 1.5, color, display: 'flex', mt: 0.5 }}>{assetClassIcon(sec.assetClass)}</Box>
                      <ListItemText
                        primary={
                          <Typography variant="body2" fontWeight={isSelected ? 700 : 500} noWrap sx={{ fontSize: 13 }}>
                            {sec.name}
                          </Typography>
                        }
                        secondary={
                          <Stack direction="row" spacing={0.5} alignItems="center" sx={{ mt: 0.25 }}>
                            <Typography variant="caption" color="text.disabled" sx={{ fontFamily: 'monospace' }}>{sec.ticker}</Typography>
                            <Typography variant="caption" color="text.disabled">·</Typography>
                            <Typography variant="caption" color="text.disabled" sx={{ fontFamily: 'monospace' }}>{sec.isin || sec.figi}</Typography>
                          </Stack>
                        }
                      />
                      <Tooltip title={`${sec.status} · ${sec.confidence}% confidence`}>
                        <Box>
                          {sec.status === 'Gold'
                            ? <GoldIcon sx={{ fontSize: 14, color: '#FFD700' }} />
                            : <WarnIcon sx={{ fontSize: 14, color: 'warning.main' }} />
                          }
                        </Box>
                      </Tooltip>
                    </ListItemButton>
                  </ListItem>
                  <Divider sx={{ mx: 2 }} />
                </React.Fragment>
              );
            })}
            {filtered.length === 0 && (
              <Box sx={{ p: 3, textAlign: 'center' }}>
                <Typography color="text.disabled" variant="body2">No securities match your search.</Typography>
              </Box>
            )}
          </List>

          {/* Footer stats */}
          <Divider />
          <Box sx={{ p: 1.5 }}>
            <Grid container spacing={1}>
              {([['Gold', '#FFD700', 4], ['Warning', '#ff9800', 1]] as const).map(([label, color, count]) => (
                <Grid key={label} size={6}>
                  <Card variant="outlined" sx={{ textAlign: 'center', py: 0.5 }}>
                    <CardContent sx={{ p: '6px !important' }}>
                      <Typography variant="h6" fontWeight={800} sx={{ color }}>{count}</Typography>
                      <Typography variant="caption" color="text.secondary">{label}</Typography>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </Box>
        </Paper>

        {/* Detail pane */}
        <Box sx={{ flex: 1, overflowY: 'auto' }}>
          {selected ? <SecurityDetailPanel security={selected} /> : (
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
              <Typography color="text.disabled">Select a security to view details</Typography>
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
};

export default SecurityMasterDashboard;
