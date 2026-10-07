import React, { useCallback, useEffect, useState, useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  TableContainer,
  TablePagination,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Chip,
  Stack,
  Grid,
  Button,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  CircularProgress,
  Divider,
  Alert,
  InputAdornment,
  LinearProgress,
  useTheme,
  alpha,
  Card,
  CardContent,
} from '@mui/material';

import SpeedIcon from '@mui/icons-material/Speed';
import RefreshIcon from '@mui/icons-material/Refresh';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CloseIcon from '@mui/icons-material/Close';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import TrendingFlatIcon from '@mui/icons-material/TrendingFlat';
import SearchIcon from '@mui/icons-material/Search';
import ShowChartIcon from '@mui/icons-material/ShowChart';

import { apiClient } from '../../utils/apiClient';

export interface LimitDataPoint {
  timestamp: string;
  value: number;
  utilization_pct: number;
  status: string;
}

export interface LimitUtilizationRecord {
  id: string;
  tenant_id: string;
  account_id: string;
  account_name: string;
  rule_id: string;
  rule_code: string;
  rule_name: string;
  rule_pack: string;
  category: string;
  threshold_limit: string;
  current_value: string;
  utilization_pct: string;
  headroom_pct: string;
  headroom_amount: string;
  currency: string;
  status: 'SAFE' | 'CAUTION' | 'WARNING' | 'BREACHED';
  trend: 'INCREASING' | 'DECREASING' | 'STABLE';
  evaluated_at: string;
  history?: LimitDataPoint[];
}

export const ComplianceLimitDashboard: React.FC = () => {
  const theme = useTheme();
  const [records, setRecords] = useState<LimitUtilizationRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [packFilter, setPackFilter] = useState('ALL');
  const [categoryFilter, setCategoryFilter] = useState('ALL');
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchQuery, setSearchQuery] = useState('');

  // Pagination
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(15);

  // Detail Modal
  const [selectedRecord, setSelectedRecord] = useState<LimitUtilizationRecord | null>(null);

  const fetchUtilization = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (packFilter !== 'ALL') params.append('rule_pack', packFilter);
      if (categoryFilter !== 'ALL') params.append('category', categoryFilter);
      if (statusFilter !== 'ALL') params.append('status', statusFilter);

      const res = await apiClient.get<{ data: LimitUtilizationRecord[]; total_count: number }>(
        `/compliance/limits/utilization?${params.toString()}`
      );
      if (res && Array.isArray(res.data)) {
        setRecords(res.data);
      } else {
        setRecords([]);
      }
    } catch (err: any) {
      console.error('Failed to load limit utilization:', err);
      setError(err?.message || 'Failed to retrieve limit utilization metrics.');
      setRecords([]);
    } finally {
      setLoading(false);
    }
  }, [packFilter, categoryFilter, statusFilter]);

  useEffect(() => {
    fetchUtilization();
  }, [fetchUtilization]);

  // Filtered list
  const filteredRecords = useMemo(() => {
    return records.filter((r) => {
      if (!searchQuery) return true;
      const q = searchQuery.toLowerCase();
      return (
        r.rule_name.toLowerCase().includes(q) ||
        r.rule_code.toLowerCase().includes(q) ||
        r.rule_pack.toLowerCase().includes(q) ||
        r.category.toLowerCase().includes(q) ||
        r.account_name.toLowerCase().includes(q)
      );
    });
  }, [records, searchQuery]);

  // KPIs
  const kpis = useMemo(() => {
    const total = records.length;
    const breached = records.filter((r) => r.status === 'BREACHED').length;
    const warning = records.filter((r) => r.status === 'WARNING').length;
    const caution = records.filter((r) => r.status === 'CAUTION').length;
    const safe = records.filter((r) => r.status === 'SAFE').length;
    return { total, breached, warning, caution, safe };
  }, [records]);

  const renderStatusBadge = (status: string, utilPct: number) => {
    switch (status) {
      case 'BREACHED':
        return (
          <Chip
            icon={<ErrorOutlineIcon style={{ color: '#fff', fontSize: '16px' }} />}
            label={`BREACH (${utilPct.toFixed(1)}%)`}
            size="small"
            sx={{ bgcolor: theme.palette.error.main, color: '#fff', fontWeight: 700, fontSize: '11px' }}
          />
        );
      case 'WARNING':
        return (
          <Chip
            icon={<WarningAmberIcon style={{ color: '#fff', fontSize: '16px' }} />}
            label={`WARNING (${utilPct.toFixed(1)}%)`}
            size="small"
            sx={{ bgcolor: theme.palette.warning.dark, color: '#fff', fontWeight: 700, fontSize: '11px' }}
          />
        );
      case 'CAUTION':
        return (
          <Chip
            label={`CAUTION (${utilPct.toFixed(1)}%)`}
            size="small"
            sx={{ bgcolor: '#ed6c02', color: '#fff', fontWeight: 600, fontSize: '11px' }}
          />
        );
      default:
        return (
          <Chip
            icon={<CheckCircleOutlineIcon style={{ color: '#fff', fontSize: '16px' }} />}
            label={`SAFE (${utilPct.toFixed(1)}%)`}
            size="small"
            sx={{ bgcolor: alpha(theme.palette.success.main, 0.85), color: '#fff', fontWeight: 600, fontSize: '11px' }}
          />
        );
    }
  };

  const renderProgressBar = (utilPct: number) => {
    let color: 'success' | 'warning' | 'error' = 'success';
    if (utilPct >= 100) color = 'error';
    else if (utilPct >= 85) color = 'error';
    else if (utilPct >= 70) color = 'warning';

    return (
      <Box sx={{ width: '100%', mr: 1 }}>
        <LinearProgress
          variant="determinate"
          value={Math.min(utilPct, 100)}
          color={color}
          sx={{ height: 8, borderRadius: 4 }}
        />
      </Box>
    );
  };

  const renderTrendIcon = (trend: string) => {
    if (trend === 'INCREASING') {
      return (
        <Tooltip title="Utilization Increasing (Proximity Risk)">
          <TrendingUpIcon fontSize="small" sx={{ color: theme.palette.error.main }} />
        </Tooltip>
      );
    }
    if (trend === 'DECREASING') {
      return (
        <Tooltip title="Utilization Decreasing (Headroom Expanding)">
          <TrendingDownIcon fontSize="small" sx={{ color: theme.palette.success.main }} />
        </Tooltip>
      );
    }
    return (
      <Tooltip title="Utilization Stable">
        <TrendingFlatIcon fontSize="small" sx={{ color: theme.palette.text.secondary }} />
      </Tooltip>
    );
  };

  const renderSparklineSVG = (history?: LimitDataPoint[]) => {
    if (!history || history.length < 2) return null;
    const values = history.map((h) => h.utilization_pct);
    const min = Math.min(...values) * 0.95;
    const max = Math.max(...values, 100);
    const range = max - min || 1;
    const width = 80;
    const height = 24;

    const points = values
      .map((val, idx) => {
        const x = (idx / (values.length - 1)) * width;
        const y = height - ((val - min) / range) * height;
        return `${x},${y}`;
      })
      .join(' ');

    const lastVal = values[values.length - 1];
    const strokeColor = lastVal >= 85 ? theme.palette.error.main : lastVal >= 70 ? '#ed6c02' : theme.palette.success.main;

    return (
      <svg width={width} height={height} style={{ overflow: 'visible' }}>
        <polyline fill="none" stroke={strokeColor} strokeWidth="2" points={points} />
      </svg>
    );
  };

  return (
    <Box sx={{ width: '100%', p: 2 }}>
      {/* KPI Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={2.4}>
          <Card sx={{ bgcolor: alpha(theme.palette.primary.main, 0.05), border: `1px solid ${alpha(theme.palette.primary.main, 0.2)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Limits Tracked
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.primary.main }}>
                    {kpis.total}
                  </Typography>
                </Box>
                <SpeedIcon sx={{ fontSize: 32, color: alpha(theme.palette.primary.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={2.4}>
          <Card sx={{ bgcolor: alpha(theme.palette.error.main, 0.05), border: `1px solid ${alpha(theme.palette.error.main, 0.3)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="error.main" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                    Active Breaches
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.error.main }}>
                    {kpis.breached}
                  </Typography>
                </Box>
                <ErrorOutlineIcon sx={{ fontSize: 32, color: alpha(theme.palette.error.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={2.4}>
          <Card sx={{ bgcolor: alpha(theme.palette.warning.main, 0.05), border: `1px solid ${alpha(theme.palette.warning.main, 0.3)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="warning.dark" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                    Warning (85-99%)
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.warning.dark }}>
                    {kpis.warning}
                  </Typography>
                </Box>
                <WarningAmberIcon sx={{ fontSize: 32, color: alpha(theme.palette.warning.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={2.4}>
          <Card sx={{ bgcolor: alpha('#ed6c02', 0.05), border: `1px solid ${alpha('#ed6c02', 0.3)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" sx={{ color: '#ed6c02', fontWeight: 600, textTransform: 'uppercase' }}>
                    Caution (70-85%)
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: '#ed6c02' }}>
                    {kpis.caution}
                  </Typography>
                </Box>
                <ShowChartIcon sx={{ fontSize: 32, color: alpha('#ed6c02', 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={2.4}>
          <Card sx={{ bgcolor: alpha(theme.palette.success.main, 0.05), border: `1px solid ${alpha(theme.palette.success.main, 0.2)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="success.dark" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Safe (&lt; 70%)
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.success.dark }}>
                    {kpis.safe}
                  </Typography>
                </Box>
                <CheckCircleOutlineIcon sx={{ fontSize: 32, color: alpha(theme.palette.success.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Filter Toolbar */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} sm={4}>
            <TextField
              fullWidth
              size="small"
              placeholder="Search rule, category, pack, fund..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" />
                  </InputAdornment>
                ),
                endAdornment: searchQuery ? (
                  <InputAdornment position="end">
                    <IconButton size="small" onClick={() => setSearchQuery('')}>
                      <CloseIcon fontSize="small" />
                    </IconButton>
                  </InputAdornment>
                ) : null,
              }}
            />
          </Grid>

          <Grid item xs={6} sm={2.5}>
            <FormControl fullWidth size="small">
              <InputLabel>Rule Pack</InputLabel>
              <Select
                value={packFilter}
                label="Rule Pack"
                onChange={(e) => setPackFilter(e.target.value)}
              >
                <MenuItem value="ALL">All Rule Packs</MenuItem>
                <MenuItem value="SEC">SEC Rules</MenuItem>
                <MenuItem value="UCITS">UCITS Directive</MenuItem>
                <MenuItem value="1940_ACT">1940 Act</MenuItem>
                <MenuItem value="MANDATE">Client Mandates</MenuItem>
                <MenuItem value="RISK">Risk & Counterparty</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={6} sm={2.5}>
            <FormControl fullWidth size="small">
              <InputLabel>Category</InputLabel>
              <Select
                value={categoryFilter}
                label="Category"
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                <MenuItem value="ALL">All Categories</MenuItem>
                <MenuItem value="CONCENTRATION">Concentration</MenuItem>
                <MenuItem value="LEVERAGE">Leverage</MenuItem>
                <MenuItem value="LIQUIDITY">Liquidity</MenuItem>
                <MenuItem value="ISSUER">Issuer Concentration</MenuItem>
                <MenuItem value="SECTOR">Sector Exposure</MenuItem>
                <MenuItem value="COUNTERPARTY">Counterparty</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={6} sm={2}>
            <FormControl fullWidth size="small">
              <InputLabel>Status</InputLabel>
              <Select
                value={statusFilter}
                label="Status"
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <MenuItem value="ALL">All Statuses</MenuItem>
                <MenuItem value="BREACHED">Breached (&ge;100%)</MenuItem>
                <MenuItem value="WARNING">Warning (85-99%)</MenuItem>
                <MenuItem value="CAUTION">Caution (70-85%)</MenuItem>
                <MenuItem value="SAFE">Safe (&lt;70%)</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={6} sm={1}>
            <Tooltip title="Refresh Limit Metrics">
              <Button
                variant="outlined"
                fullWidth
                size="medium"
                onClick={fetchUtilization}
                disabled={loading}
                sx={{ height: 40 }}
              >
                <RefreshIcon fontSize="small" />
              </Button>
            </Tooltip>
          </Grid>
        </Grid>
      </Paper>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Utilization Table */}
      <TableContainer component={Paper} sx={{ position: 'relative' }}>
        {loading && (
          <Box
            sx={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: alpha(theme.palette.background.paper, 0.7),
              zIndex: 2,
            }}
          >
            <CircularProgress size={36} />
          </Box>
        )}

        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: alpha(theme.palette.primary.main, 0.04) }}>
              <TableCell sx={{ fontWeight: 700, width: 220 }}>Rule & Limit Definition</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 110 }}>Category</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 100 }}>Pack</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 200 }}>Utilization Gauge</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 130 }}>Current vs Limit</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 140 }}>Remaining Headroom</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 100 }}>7d Sparkline</TableCell>
              <TableCell align="center" sx={{ fontWeight: 700, width: 60 }}>Trend</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 130 }}>Status</TableCell>
              <TableCell align="center" sx={{ fontWeight: 700, width: 60 }}>Inspect</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredRecords.length === 0 && !loading ? (
              <TableRow>
                <TableCell colSpan={10} align="center" sx={{ py: 4 }}>
                  <Typography variant="body2" color="text.secondary">
                    No limits found matching current filters.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredRecords
                .slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)
                .map((rec) => {
                  const utilFloat = parseFloat(rec.utilization_pct) || 0;
                  return (
                    <TableRow
                      key={rec.id}
                      hover
                      sx={{
                        cursor: 'pointer',
                        bgcolor: rec.status === 'BREACHED' ? alpha(theme.palette.error.main, 0.03) : undefined,
                      }}
                      onClick={() => setSelectedRecord(rec)}
                    >
                      <TableCell>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          {rec.rule_name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                          {rec.rule_code}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={rec.category} size="small" variant="outlined" sx={{ fontSize: '11px' }} />
                      </TableCell>
                      <TableCell>
                        <Chip label={rec.rule_pack} size="small" sx={{ fontSize: '11px', fontWeight: 600 }} />
                      </TableCell>
                      <TableCell>
                        <Stack spacing={0.5}>
                          <Stack direction="row" justifyContent="space-between">
                            <Typography variant="caption" sx={{ fontWeight: 700 }}>
                              {utilFloat.toFixed(1)}%
                            </Typography>
                          </Stack>
                          {renderProgressBar(utilFloat)}
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontWeight: 700 }}>
                          {(parseFloat(rec.current_value) * 100).toFixed(2)}%
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          Ceiling: {(parseFloat(rec.threshold_limit) * 100).toFixed(1)}%
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontWeight: 600, color: theme.palette.text.primary }}>
                          +{rec.headroom_pct}%
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          ${parseFloat(rec.headroom_amount).toLocaleString()}
                        </Typography>
                      </TableCell>
                      <TableCell>{renderSparklineSVG(rec.history)}</TableCell>
                      <TableCell align="center">{renderTrendIcon(rec.trend)}</TableCell>
                      <TableCell>{renderStatusBadge(rec.status, utilFloat)}</TableCell>
                      <TableCell align="center">
                        <Tooltip title="View Limit Headroom & Remediation">
                          <IconButton
                            size="small"
                            color="primary"
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedRecord(rec);
                            }}
                          >
                            <VisibilityIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  );
                })
            )}
          </TableBody>
        </Table>

        <TablePagination
          rowsPerPageOptions={[10, 15, 25, 50]}
          component="div"
          count={filteredRecords.length}
          rowsPerPage={rowsPerPage}
          page={page}
          onPageChange={(_, p) => setPage(p)}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10));
            setPage(0);
          }}
        />
      </TableContainer>

      {/* Detail Modal */}
      {selectedRecord && (
        <Dialog
          open={Boolean(selectedRecord)}
          onClose={() => setSelectedRecord(null)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle sx={{ pb: 1 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Limit Utilization & Headroom Analysis
              </Typography>
              <IconButton size="small" onClick={() => setSelectedRecord(null)}>
                <CloseIcon />
              </IconButton>
            </Stack>
          </DialogTitle>
          <Divider />
          <DialogContent sx={{ pt: 2 }}>
            <Stack spacing={2.5}>
              <Box>
                <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                  <Chip label={selectedRecord.rule_pack} size="small" color="primary" />
                  <Chip label={selectedRecord.category} size="small" variant="outlined" />
                  {renderStatusBadge(selectedRecord.status, parseFloat(selectedRecord.utilization_pct))}
                </Stack>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  {selectedRecord.rule_name}
                </Typography>
                <Typography variant="subtitle2" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                  Rule ID: {selectedRecord.rule_id} | Code: {selectedRecord.rule_code}
                </Typography>
              </Box>

              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                  Current Threshold Proximity & Capacity Buffer
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={4}>
                    <Typography variant="caption" color="text.secondary">
                      Current Portfolio Value
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 700 }}>
                      {(parseFloat(selectedRecord.current_value) * 100).toFixed(2)}%
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={4}>
                    <Typography variant="caption" color="text.secondary">
                      Regulatory / Mandate Limit
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 700 }}>
                      {(parseFloat(selectedRecord.threshold_limit) * 100).toFixed(2)}%
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={4}>
                    <Typography variant="caption" color="text.secondary">
                      Remaining Dollar Headroom
                    </Typography>
                    <Typography variant="h6" sx={{ fontWeight: 700, color: theme.palette.success.main }}>
                      ${parseFloat(selectedRecord.headroom_amount).toLocaleString()}
                    </Typography>
                  </Grid>
                </Grid>
                <Box sx={{ mt: 2 }}>
                  {renderProgressBar(parseFloat(selectedRecord.utilization_pct))}
                </Box>
              </Paper>

              <Paper variant="outlined" sx={{ p: 2, bgcolor: alpha(theme.palette.background.default, 0.5) }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                  Remediation & Pre-Trade Guidance
                </Typography>
                <Typography variant="body2" sx={{ lineHeight: 1.6 }}>
                  {parseFloat(selectedRecord.utilization_pct) >= 85
                    ? `⚠️ High proximity warning: Portfolio has consumed ${selectedRecord.utilization_pct}% of allowed ceiling. Additional buy orders in this category will trigger pre-trade warnings or hard order rejections. Consider rebalancing or requesting a temporary mandate relaxation.`
                    : `✅ Headroom buffer is healthy ($${parseFloat(selectedRecord.headroom_amount).toLocaleString()} remaining). New executions can be allocated without breaching regulatory constraints.`}
                </Typography>
              </Paper>
            </Stack>
          </DialogContent>
          <Divider />
          <DialogActions sx={{ px: 3, py: 1.5 }}>
            <Button onClick={() => setSelectedRecord(null)}>Close</Button>
            <Button variant="contained" color="primary">
              Simulate Pre-Trade Allocation
            </Button>
          </DialogActions>
        </Dialog>
      )}
    </Box>
  );
};
