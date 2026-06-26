import React, { useEffect, useState, useCallback } from 'react';
import {
  Alert,
  Box,
  Card,
  CardHeader,
  Chip,
  Divider,
  IconButton,
  LinearProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import LightbulbOutlinedIcon from '@mui/icons-material/LightbulbOutlined';
import StarIcon from '@mui/icons-material/Star';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';

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
}

interface Props {
  businessObject: string;
  semanticTerm: string;
  accountType: string;
  region?: string;
}

const DEMO: SourceRow[] = [
  { sourceSystem: 'Bloomberg',  confidence: 98, confidenceDelta: 0,   coveragePercent: 99, timeliness: 'Real-time', lastUpdated: '2m ago',   errorCount: 12, firstPreferenceCount: 340, totalSelections: 360, impactedPortfolios: 150 },
  { sourceSystem: 'Refinitiv',  confidence: 92, confidenceDelta: -6,  coveragePercent: 97, timeliness: 'Real-time', lastUpdated: '5m ago',   errorCount: 28, firstPreferenceCount: 120, totalSelections: 360, impactedPortfolios: 60  },
  { sourceSystem: 'FactSet',    confidence: 89, confidenceDelta: -9,  coveragePercent: 94, timeliness: 'T+1',       lastUpdated: '1hr ago',  errorCount: 38, firstPreferenceCount: 60,  totalSelections: 360, impactedPortfolios: 30  },
  { sourceSystem: 'S&P',        confidence: 82, confidenceDelta: -16, coveragePercent: 91, timeliness: 'T+1',       lastUpdated: '2hrs ago', errorCount: 48, firstPreferenceCount: 20,  totalSelections: 360, impactedPortfolios: 10  },
];

const timelinessBadge = (t: string) => (
  <Chip
    label={t}
    size="small"
    color={t === 'Real-time' ? 'success' : t === 'T+1' ? 'warning' : 'default'}
    variant="outlined"
  />
);

export const SourceComparisonMatrix: React.FC<Props> = ({
  businessObject,
  semanticTerm,
  accountType,
  region = 'NAM',
}) => {
  const [sources, setSources] = useState<SourceRow[]>(DEMO);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch_ = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const p = new URLSearchParams({ business_object: businessObject, semantic_term: semanticTerm, account_type: accountType, region });
      const res = await fetch(`/api/v1/portfolio/analytics/sources?${p}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      if (json.sources?.length) setSources(json.sources);
      else setSources(DEMO);
    } catch {
      setSources(DEMO);
    } finally {
      setLoading(false);
    }
  }, [businessObject, semanticTerm, accountType, region]);

  useEffect(() => { fetch_(); }, [fetch_]);

  const best = sources[0];
  const second = sources[1];

  return (
    <Card elevation={2}>
      <CardHeader
        title={
          <Stack direction="row" alignItems="center" spacing={1}>
            <Typography variant="subtitle1" fontWeight={700}>Source Comparison</Typography>
            <Chip label={semanticTerm} color="primary" size="small" variant="outlined" />
            <Chip label={accountType} size="small" variant="outlined" />
            <Chip label={region} size="small" variant="outlined" />
          </Stack>
        }
        action={
          <Tooltip title="Refresh">
            <IconButton size="small" onClick={fetch_} disabled={loading}>
              <RefreshIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        }
      />
      <Divider />

      {loading && <LinearProgress />}
      {error && <Alert severity="error" sx={{ m: 2 }}>{error}</Alert>}

      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: 'action.hover' }}>
              <TableCell sx={{ fontWeight: 700 }}>Source</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Confidence</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Coverage</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Timeliness</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Last Updated</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Errors</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>1st Pref %</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {sources.map((row, i) => {
              const firstPct = row.totalSelections ? Math.round((row.firstPreferenceCount / row.totalSelections) * 100) : 0;
              return (
                <TableRow key={row.sourceSystem} hover selected={i === 0}>
                  <TableCell>
                    <Stack direction="row" alignItems="center" spacing={1}>
                      {i === 0 && <StarIcon fontSize="small" color="warning" />}
                      <Typography variant="body2" fontWeight={i === 0 ? 700 : 400}>{row.sourceSystem}</Typography>
                      {i === 0 && <Chip label="Recommended" size="small" color="success" />}
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Stack direction="row" alignItems="center" spacing={1}>
                      <Box sx={{ width: 70 }}>
                        <LinearProgress
                          variant="determinate"
                          value={row.confidence}
                          sx={{ height: 8, borderRadius: 4 }}
                          color={row.confidence >= 95 ? 'success' : row.confidence >= 80 ? 'warning' : 'error'}
                        />
                      </Box>
                      <Typography variant="body2" fontWeight={600}>{row.confidence}%</Typography>
                      {row.confidenceDelta !== 0 && (
                        <Stack direction="row" alignItems="center">
                          {row.confidenceDelta > 0
                            ? <TrendingUpIcon fontSize="small" color="success" />
                            : <TrendingDownIcon fontSize="small" color="error" />}
                          <Typography variant="caption" color={row.confidenceDelta > 0 ? 'success.main' : 'error.main'} fontWeight={700}>
                            {row.confidenceDelta}%
                          </Typography>
                        </Stack>
                      )}
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{row.coveragePercent}%</Typography>
                  </TableCell>
                  <TableCell>{timelinessBadge(row.timeliness)}</TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">{row.lastUpdated}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={row.errorCount}
                      size="small"
                      color={row.errorCount > 30 ? 'error' : 'default'}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" fontWeight={600}>{firstPct}%</Typography>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Business insight callout */}
      {best && second && (
        <Box sx={{ p: 2, bgcolor: 'action.hover', borderTop: '1px solid', borderColor: 'divider' }}>
          <Alert icon={<LightbulbOutlinedIcon />} severity="info" variant="outlined" sx={{ py: 0.5 }}>
            Using <strong>{best.sourceSystem}</strong> as primary raises confidence by{' '}
            <strong>+{best.confidence - second.confidence}%</strong> and affects{' '}
            <strong>{best.impactedPortfolios}</strong> portfolios vs {second.sourceSystem}.
          </Alert>
        </Box>
      )}
    </Card>
  );
};

export default SourceComparisonMatrix;
