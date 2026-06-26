import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Card, CardContent, CardHeader, Chip,
  Divider, FormControl, IconButton, LinearProgress,
  MenuItem, Select, SelectChangeEvent, Stack, Table,
  TableBody, TableCell, TableContainer, TableHead, TableRow,
  Tooltip, Typography, Button, Alert,
} from '@mui/material';
import Grid from '@mui/material/Grid';
import RefreshIcon from '@mui/icons-material/Refresh';
import DownloadIcon from '@mui/icons-material/Download';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import TrendingFlatIcon from '@mui/icons-material/TrendingFlat';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import HealthAndSafetyIcon from '@mui/icons-material/HealthAndSafety';
import { PortfolioImpactSimulation } from './PortfolioImpactSimulation';

// ── Types ──────────────────────────────────────────────────────────────────────

interface SourceRow {
  sourceSystem: string;
  confidence: number;
  confidenceDelta: number;
  coveragePercent: number;
  timeliness: string;
  lastUpdated: string;
  errorCount: number;
  firstPreferenceCount: number;
  totalSelections: number;
  impactedPortfolios: number;
  uptime: number;
}

const TERMS = ['Price', 'Quantity', 'MarketValue', 'Yield', 'Duration', 'Rating'];
const ACCOUNT_TYPES = ['institutional', 'retail', 'private_wealth', 'private_markets'];
const REGIONS = ['NAM', 'EMEA', 'APAC', 'LATAM', 'EM'];

const DEMO_SOURCES: SourceRow[] = [
  { sourceSystem: 'Bloomberg',  confidence: 98, confidenceDelta: 0,   coveragePercent: 99, timeliness: 'Real-time', lastUpdated: '2 mins ago',  errorCount: 12, firstPreferenceCount: 340, totalSelections: 360, impactedPortfolios: 150, uptime: 99.99 },
  { sourceSystem: 'Refinitiv',  confidence: 92, confidenceDelta: -6,  coveragePercent: 97, timeliness: 'Real-time', lastUpdated: '5 mins ago',  errorCount: 28, firstPreferenceCount: 120, totalSelections: 360, impactedPortfolios: 60,  uptime: 99.8  },
  { sourceSystem: 'FactSet',    confidence: 89, confidenceDelta: -9,  coveragePercent: 94, timeliness: 'T+1',       lastUpdated: '1 hour ago',  errorCount: 38, firstPreferenceCount: 60,  totalSelections: 360, impactedPortfolios: 30,  uptime: 99.2  },
  { sourceSystem: 'S&P',        confidence: 82, confidenceDelta: -16, coveragePercent: 91, timeliness: 'T+1',       lastUpdated: '2 hours ago', errorCount: 48, firstPreferenceCount: 20,  totalSelections: 360, impactedPortfolios: 10,  uptime: 98.4  },
];

const timelinessBadge = (t: string) => {
  const color = t === 'Real-time' ? 'success' : t === 'T+1' ? 'warning' : 'default';
  return <Chip label={t} color={color as any} size="small" variant="outlined" />;
};

const rankChip = (rank: number) => {
  if (rank === 0) return <Chip label="1st" color="success" size="small" sx={{ ml: 1 }} />;
  if (rank === 1) return <Chip label="2nd" color="warning" size="small" sx={{ ml: 1 }} />;
  return <Chip label={`${rank + 1}th`} color="default" size="small" sx={{ ml: 1 }} />;
};

const DeltaCell: React.FC<{ delta: number; showAbs?: boolean }> = ({ delta, showAbs }) => {
  if (delta === 0) return <TrendingFlatIcon color="disabled" fontSize="small" />;
  return (
    <Stack direction="row" alignItems="center" spacing={0.5}>
      {delta > 0
        ? <TrendingUpIcon color="success" fontSize="small" />
        : <TrendingDownIcon color="error" fontSize="small" />}
      <Typography variant="body2" color={delta > 0 ? 'success.main' : 'error.main'} fontWeight={700}>
        {delta > 0 ? '+' : ''}{showAbs ? Math.abs(delta) : delta}%
      </Typography>
    </Stack>
  );
};

// ── Main Component ─────────────────────────────────────────────────────────────

const EDM_SourceComparisonDashboard: React.FC = () => {
  const [semanticTerm, setSemanticTerm] = useState('Price');
  const [accountType, setAccountType] = useState('institutional');
  const [region, setRegion] = useState('NAM');
  const [sources, setSources] = useState<SourceRow[]>(DEMO_SOURCES);
  const [loading, setLoading] = useState(false);
  const [showSim, setShowSim] = useState(false);

  const fetchSources = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ business_object: 'Portfolio', semantic_term: semanticTerm, account_type: accountType, region });
      const res = await fetch(`/api/v1/portfolio/analytics/sources?${params}`);
      if (!res.ok) throw new Error('API error');
      const json = await res.json();
      if (json.sources?.length) setSources(json.sources);
      else setSources(DEMO_SOURCES);
    } catch {
      setSources(DEMO_SOURCES);
    } finally {
      setLoading(false);
    }
  }, [semanticTerm, accountType, region]);

  useEffect(() => { fetchSources(); }, [fetchSources]);

  const primary = sources[0];
  const secondary = sources[1];

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* ── Page Header ──────────────────────────────────────────────────── */}
      <Box sx={{ px: 4, pt: 4, pb: 2 }}>
        <Stack direction="row" alignItems="flex-start" justifyContent="space-between" flexWrap="wrap" gap={2}>
          <Box>
            <Typography variant="h4" fontWeight={800} gutterBottom>Source Comparison Matrix</Typography>
            <Typography variant="body2" color="text.secondary">
              Evaluation of data providers for <strong>{semanticTerm}</strong> · {accountType} portfolios · {region}
            </Typography>
          </Box>
          <Stack direction="row" spacing={1} alignItems="center">
            {/* Filters */}
            <FormControl size="small">
              <Select value={semanticTerm} onChange={(e: SelectChangeEvent) => setSemanticTerm(e.target.value)}>
                {TERMS.map(t => <MenuItem key={t} value={t}>{t}</MenuItem>)}
              </Select>
            </FormControl>
            <FormControl size="small">
              <Select value={accountType} onChange={(e: SelectChangeEvent) => setAccountType(e.target.value)}>
                {ACCOUNT_TYPES.map(t => <MenuItem key={t} value={t}>{t}</MenuItem>)}
              </Select>
            </FormControl>
            <FormControl size="small">
              <Select value={region} onChange={(e: SelectChangeEvent) => setRegion(e.target.value)}>
                {REGIONS.map(r => <MenuItem key={r} value={r}>{r}</MenuItem>)}
              </Select>
            </FormControl>
            <Tooltip title="Refresh">
              <IconButton onClick={fetchSources} disabled={loading} color="primary">
                <RefreshIcon />
              </IconButton>
            </Tooltip>
            <Button variant="contained" startIcon={<DownloadIcon />} size="small">Export</Button>
          </Stack>
        </Stack>
      </Box>

      {loading && <LinearProgress />}

      <Box sx={{ px: 4, pb: 4 }}>
        <Grid container spacing={3}>
          {/* ── Comparison Table ─────────────────────────────────────────── */}
          <Grid size={{ xs: 12, lg: 9 }}>
            <Card elevation={2}>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'action.hover' }}>
                      <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.72rem', letterSpacing: '0.06em', color: 'text.secondary' }}>
                        Metric
                      </TableCell>
                      {sources.map((s, i) => (
                        <TableCell key={s.sourceSystem}>
                          <Stack direction="row" alignItems="center">
                            <Typography variant="body2" fontWeight={700}>{s.sourceSystem}</Typography>
                            {rankChip(i)}
                          </Stack>
                        </TableCell>
                      ))}
                      <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.72rem', letterSpacing: '0.06em', color: 'text.secondary' }}>
                        1st vs 2nd δ
                      </TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {/* Confidence */}
                    <TableRow hover>
                      <TableCell><Typography variant="body2" fontWeight={600}>Confidence Score</Typography></TableCell>
                      {sources.map(s => (
                        <TableCell key={s.sourceSystem}>
                          <Stack direction="row" alignItems="center" spacing={1}>
                            <Box sx={{ width: 80 }}>
                              <LinearProgress variant="determinate" value={s.confidence} sx={{ height: 8, borderRadius: 4 }} color={s.confidence >= 95 ? 'success' : s.confidence >= 80 ? 'warning' : 'error'} />
                            </Box>
                            <Typography variant="body2" fontWeight={600}>{s.confidence}%</Typography>
                            {s.confidenceDelta !== 0 && (
                              <Typography variant="caption" color={s.confidenceDelta > 0 ? 'success.main' : 'error.main'} fontWeight={700}>
                                {s.confidenceDelta > 0 ? '+' : ''}{s.confidenceDelta}%
                              </Typography>
                            )}
                          </Stack>
                        </TableCell>
                      ))}
                      <TableCell>
                        {sources.length >= 2 && <DeltaCell delta={sources[0].confidence - sources[1].confidence} />}
                      </TableCell>
                    </TableRow>

                    {/* Coverage */}
                    <TableRow hover>
                      <TableCell><Typography variant="body2" fontWeight={600}>Coverage %</Typography></TableCell>
                      {sources.map(s => (
                        <TableCell key={s.sourceSystem}>
                          <Typography variant="body2" fontWeight={600}>{s.coveragePercent}%</Typography>
                        </TableCell>
                      ))}
                      <TableCell>
                        {sources.length >= 2 && <DeltaCell delta={sources[0].coveragePercent - sources[1].coveragePercent} />}
                      </TableCell>
                    </TableRow>

                    {/* Timeliness */}
                    <TableRow hover>
                      <TableCell><Typography variant="body2" fontWeight={600}>Timeliness</Typography></TableCell>
                      {sources.map(s => <TableCell key={s.sourceSystem}>{timelinessBadge(s.timeliness)}</TableCell>)}
                      <TableCell>
                        {sources.length >= 2 && sources[0].timeliness !== sources[1].timeliness && (
                          <Chip label="Critical" color="error" size="small" variant="outlined" />
                        )}
                      </TableCell>
                    </TableRow>

                    {/* Last updated */}
                    <TableRow hover>
                      <TableCell><Typography variant="body2" fontWeight={600}>Last Updated</Typography></TableCell>
                      {sources.map(s => (
                        <TableCell key={s.sourceSystem}>
                          <Typography variant="body2" color="text.secondary">{s.lastUpdated}</Typography>
                        </TableCell>
                      ))}
                      <TableCell><Typography variant="body2" color="text.disabled">N/A</Typography></TableCell>
                    </TableRow>

                    {/* Error count */}
                    <TableRow hover>
                      <TableCell><Typography variant="body2" fontWeight={600}>Error Count</Typography></TableCell>
                      {sources.map(s => (
                        <TableCell key={s.sourceSystem}>
                          <Chip
                            label={s.errorCount}
                            color={s.errorCount > 30 ? 'error' : 'default'}
                            size="small"
                            variant="outlined"
                          />
                        </TableCell>
                      ))}
                      <TableCell>
                        {sources.length >= 2 && (
                          <DeltaCell delta={-(sources[1].errorCount - sources[0].errorCount)} showAbs />
                        )}
                      </TableCell>
                    </TableRow>

                    {/* 1st Pref % */}
                    <TableRow hover>
                      <TableCell><Typography variant="body2" fontWeight={600}>1st Preference %</Typography></TableCell>
                      {sources.map(s => {
                        const pct = s.totalSelections ? Math.round((s.firstPreferenceCount / s.totalSelections) * 100) : 0;
                        return (
                          <TableCell key={s.sourceSystem}>
                            <Typography variant="body2" fontWeight={600}>{pct}%</Typography>
                          </TableCell>
                        );
                      })}
                      <TableCell />
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>

              {/* Business impact callout + Simulate button */}
              {primary && (
                <Box sx={{ p: 2, bgcolor: 'action.hover', borderTop: '1px solid', borderColor: 'divider' }}>
                  <Stack direction="row" alignItems="center" justifyContent="space-between" gap={2} flexWrap="wrap">
                    <Alert icon={<LightbulbIcon />} severity="info" sx={{ flex: 1, py: 0 }}>
                      Using <strong>{primary.sourceSystem}</strong> as primary impacts{' '}
                      <strong>{primary.impactedPortfolios}</strong> portfolios ·{' '}
                      {secondary && <>+{primary.confidence - secondary.confidence}% confidence vs {secondary.sourceSystem}</>}
                    </Alert>
                    <Button variant="outlined" size="small" onClick={() => setShowSim(v => !v)}>
                      {showSim ? 'Hide' : 'Run'} Impact Simulation
                    </Button>
                  </Stack>
                  {showSim && (
                    <Box sx={{ mt: 2 }}>
                      <PortfolioImpactSimulation portfolioId="*" semanticTerm={semanticTerm} newSourceSystem={primary.sourceSystem} />
                    </Box>
                  )}
                </Box>
              )}
            </Card>
          </Grid>

          {/* ── Sidebar ──────────────────────────────────────────────────── */}
          <Grid size={{ xs: 12, lg: 3 }}>
            <Stack spacing={3}>
              {/* Business Impact */}
              <Card elevation={2}>
                <CardHeader
                  avatar={<LightbulbIcon color="primary" />}
                  title="Business Impact"
                  titleTypographyProps={{ fontWeight: 700, variant: 'body1' }}
                />
                <Divider />
                <CardContent>
                  {primary ? (
                    <>
                      <Typography variant="body2" color="text.secondary" paragraph>
                        <strong>{primary.sourceSystem}</strong> provides superior coverage for{' '}
                        {accountType} {semanticTerm} data, justifying its{' '}
                        <Typography component="span" color="primary.main" fontWeight={700}>Rank 1</Typography> status.
                      </Typography>
                      {secondary && (
                        <Typography variant="body2" color="text.secondary" paragraph>
                          The {primary.confidence - secondary.confidence}pt confidence advantage over{' '}
                          {secondary.sourceSystem} has direct consequences for portfolio accuracy.
                        </Typography>
                      )}
                      <Box sx={{ mt: 2, p: 1.5, bgcolor: 'primary.50', border: '1px solid', borderColor: 'primary.200', borderRadius: 1 }}>
                        <Typography variant="body2" color="primary.main" fontWeight={500}>
                          Maintain <strong>{primary.sourceSystem}</strong> as primary source for the current fiscal year.
                        </Typography>
                      </Box>
                    </>
                  ) : (
                    <Typography variant="body2" color="text.secondary">Select filters to see impact analysis.</Typography>
                  )}
                </CardContent>
              </Card>

              {/* Source Uptime */}
              <Card elevation={2}>
                <CardHeader
                  avatar={<HealthAndSafetyIcon color="primary" />}
                  title="Source Uptime"
                  titleTypographyProps={{ fontWeight: 700, variant: 'body1' }}
                />
                <Divider />
                <CardContent>
                  <Stack spacing={2}>
                    {sources.map(s => (
                      <Box key={s.sourceSystem}>
                        <Stack direction="row" justifyContent="space-between" mb={0.5}>
                          <Typography variant="body2" color="text.secondary">{s.sourceSystem}</Typography>
                          <Typography variant="body2" fontWeight={700} color={s.uptime >= 99.9 ? 'success.main' : s.uptime >= 98 ? 'warning.main' : 'error.main'}>
                            {s.uptime.toFixed(2)}%
                          </Typography>
                        </Stack>
                        <LinearProgress
                          variant="determinate"
                          value={s.uptime}
                          sx={{ height: 6, borderRadius: 3 }}
                          color={s.uptime >= 99.9 ? 'success' : s.uptime >= 98 ? 'warning' : 'error'}
                        />
                      </Box>
                    ))}
                  </Stack>
                </CardContent>
              </Card>
            </Stack>
          </Grid>
        </Grid>
      </Box>
    </Box>
  );
};

export default EDM_SourceComparisonDashboard;
