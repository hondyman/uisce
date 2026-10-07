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
  useTheme,
  alpha,
  Card,
  CardContent,
} from '@mui/material';

import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import RefreshIcon from '@mui/icons-material/Refresh';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CloseIcon from '@mui/icons-material/Close';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import SearchIcon from '@mui/icons-material/Search';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';

import { apiClient } from '../../utils/apiClient';

export interface ComplianceCalendarEvent {
  id: string;
  tenant_id: string;
  event_code: string;
  title: string;
  description: string;
  jurisdiction: 'US' | 'UK' | 'EU' | 'GLOBAL';
  regulation: string;
  deadline_type: 'STATUTORY_FILING' | 'MANDATE_ATTESTATION' | 'PORTFOLIO_REVIEW' | 'DISCLOSURE_CUTOFF' | 'REGULATORY_CHANGE';
  due_date: string;
  cutoff_time: string;
  cutoff_timestamp?: string;
  days_remaining: number;
  status: 'UPCOMING' | 'DUE_SOON' | 'OVERDUE' | 'FILED' | 'EXEMPT';
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  legal_citation?: string;
  affected_accounts?: string[];
  rule_ids?: string[];
  source: 'STATUTORY_SCHEDULE' | 'REGULATORY_CASE' | 'SURVEILLANCE_BREACH';
  source_id?: string;
  metadata?: Record<string, any>;
}

export const ComplianceCalendar: React.FC = () => {
  const theme = useTheme();
  const [events, setEvents] = useState<ComplianceCalendarEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [jurisdictionFilter, setJurisdictionFilter] = useState('ALL');
  const [deadlineTypeFilter, setDeadlineTypeFilter] = useState('ALL');
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchQuery, setSearchQuery] = useState('');

  // Pagination
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(15);

  // Detail Modal
  const [selectedEvent, setSelectedEvent] = useState<ComplianceCalendarEvent | null>(null);

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (jurisdictionFilter !== 'ALL') params.append('jurisdiction', jurisdictionFilter);
      if (deadlineTypeFilter !== 'ALL') params.append('deadline_type', deadlineTypeFilter);
      if (statusFilter !== 'ALL') params.append('status', statusFilter);

      const res = await apiClient.get<{ data: ComplianceCalendarEvent[]; total_count: number }>(
        `/compliance/calendar?${params.toString()}`
      );
      if (res && Array.isArray(res.data)) {
        setEvents(res.data);
      } else {
        setEvents([]);
      }
    } catch (err: any) {
      console.error('Failed to load compliance calendar:', err);
      setError(err?.message || 'Failed to retrieve compliance calendar deadlines.');
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [jurisdictionFilter, deadlineTypeFilter, statusFilter]);

  useEffect(() => {
    fetchEvents();
  }, [fetchEvents]);

  // Client-side search filter
  const filteredEvents = useMemo(() => {
    return events.filter((e) => {
      if (!searchQuery) return true;
      const q = searchQuery.toLowerCase();
      return (
        e.title.toLowerCase().includes(q) ||
        e.description.toLowerCase().includes(q) ||
        e.regulation.toLowerCase().includes(q) ||
        e.event_code.toLowerCase().includes(q) ||
        e.jurisdiction.toLowerCase().includes(q) ||
        (e.rule_ids && e.rule_ids.some((r) => r.toLowerCase().includes(q)))
      );
    });
  }, [events, searchQuery]);

  // KPI calculations
  const kpis = useMemo(() => {
    const total = events.length;
    const overdue = events.filter((e) => e.days_remaining < 0).length;
    const dueSoon = events.filter((e) => e.days_remaining >= 0 && e.days_remaining <= 5).length;
    const upcoming = events.filter((e) => e.days_remaining > 5).length;
    return { total, overdue, dueSoon, upcoming };
  }, [events]);

  const renderStatusBadge = (daysRemaining: number, status: string) => {
    if (daysRemaining < 0) {
      return (
        <Chip
          icon={<ErrorOutlineIcon style={{ color: '#fff', fontSize: '16px' }} />}
          label={`OVERDUE (${Math.abs(daysRemaining)}d)`}
          size="small"
          sx={{
            bgcolor: theme.palette.error.main,
            color: '#fff',
            fontWeight: 700,
            fontSize: '11px',
          }}
        />
      );
    }
    if (daysRemaining <= 5) {
      return (
        <Chip
          icon={<WarningAmberIcon style={{ color: '#fff', fontSize: '16px' }} />}
          label={`DUE SOON (${daysRemaining}d)`}
          size="small"
          sx={{
            bgcolor: theme.palette.warning.dark,
            color: '#fff',
            fontWeight: 700,
            fontSize: '11px',
          }}
        />
      );
    }
    return (
      <Chip
        icon={<CheckCircleOutlineIcon style={{ color: '#fff', fontSize: '16px' }} />}
        label={`UPCOMING (${daysRemaining}d)`}
        size="small"
        sx={{
          bgcolor: alpha(theme.palette.success.main, 0.85),
          color: '#fff',
          fontWeight: 600,
          fontSize: '11px',
        }}
      />
    );
  };

  const renderJurisdictionChip = (jurisdiction: string) => {
    const colors: Record<string, { bg: string; text: string }> = {
      US: { bg: alpha('#1976d2', 0.15), text: '#1976d2' },
      UK: { bg: alpha('#7b1fa2', 0.15), text: '#7b1fa2' },
      EU: { bg: alpha('#0288d1', 0.15), text: '#0288d1' },
      GLOBAL: { bg: alpha('#388e3c', 0.15), text: '#388e3c' },
    };
    const c = colors[jurisdiction] || { bg: alpha(theme.palette.text.secondary, 0.1), text: theme.palette.text.primary };
    return (
      <Chip
        label={jurisdiction}
        size="small"
        sx={{
          bgcolor: c.bg,
          color: c.text,
          fontWeight: 700,
          fontSize: '11px',
          border: `1px solid ${c.text}`,
        }}
      />
    );
  };

  const renderDeadlineTypeChip = (deadlineType: string) => {
    const labels: Record<string, string> = {
      STATUTORY_FILING: 'Statutory Filing',
      MANDATE_ATTESTATION: 'Mandate Attestation',
      PORTFOLIO_REVIEW: 'Portfolio Review',
      DISCLOSURE_CUTOFF: 'Dealing Cutoff',
      REGULATORY_CHANGE: 'Regulatory Change',
    };
    return (
      <Chip
        label={labels[deadlineType] || deadlineType}
        size="small"
        variant="outlined"
        sx={{ fontSize: '11px', fontWeight: 500 }}
      />
    );
  };

  return (
    <Box sx={{ width: '100%', p: 2 }}>
      {/* Top Header & KPI Summary Banner */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: alpha(theme.palette.primary.main, 0.05), border: `1px solid ${alpha(theme.palette.primary.main, 0.2)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Active Deadlines
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.primary.main }}>
                    {kpis.total}
                  </Typography>
                </Box>
                <CalendarTodayIcon sx={{ fontSize: 32, color: alpha(theme.palette.primary.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: alpha(theme.palette.error.main, 0.05), border: `1px solid ${alpha(theme.palette.error.main, 0.3)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="error.main" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                    Overdue Filings
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.error.main }}>
                    {kpis.overdue}
                  </Typography>
                </Box>
                <ErrorOutlineIcon sx={{ fontSize: 32, color: alpha(theme.palette.error.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: alpha(theme.palette.warning.main, 0.05), border: `1px solid ${alpha(theme.palette.warning.main, 0.3)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="warning.dark" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                    Due &le; 5 Days
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.warning.dark }}>
                    {kpis.dueSoon}
                  </Typography>
                </Box>
                <AccessTimeIcon sx={{ fontSize: 32, color: alpha(theme.palette.warning.main, 0.4) }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: alpha(theme.palette.success.main, 0.05), border: `1px solid ${alpha(theme.palette.success.main, 0.2)}` }}>
            <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="caption" color="success.dark" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Upcoming Windows
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: theme.palette.success.dark }}>
                    {kpis.upcoming}
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
              placeholder="Search regulation, title, code, rule..."
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
              <InputLabel>Jurisdiction</InputLabel>
              <Select
                value={jurisdictionFilter}
                label="Jurisdiction"
                onChange={(e) => setJurisdictionFilter(e.target.value)}
              >
                <MenuItem value="ALL">All Jurisdictions</MenuItem>
                <MenuItem value="US">United States (SEC/CFTC)</MenuItem>
                <MenuItem value="UK">United Kingdom (FCA/Panel)</MenuItem>
                <MenuItem value="EU">European Union (ESMA)</MenuItem>
                <MenuItem value="GLOBAL">Global / Multi-Jurisdiction</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={6} sm={2.5}>
            <FormControl fullWidth size="small">
              <InputLabel>Deadline Type</InputLabel>
              <Select
                value={deadlineTypeFilter}
                label="Deadline Type"
                onChange={(e) => setDeadlineTypeFilter(e.target.value)}
              >
                <MenuItem value="ALL">All Deadline Types</MenuItem>
                <MenuItem value="STATUTORY_FILING">Statutory Filing</MenuItem>
                <MenuItem value="DISCLOSURE_CUTOFF">Dealing Disclosure Cutoff</MenuItem>
                <MenuItem value="PORTFOLIO_REVIEW">Periodic Portfolio Review</MenuItem>
                <MenuItem value="MANDATE_ATTESTATION">Mandate Attestation</MenuItem>
                <MenuItem value="REGULATORY_CHANGE">Regulatory Change Due Date</MenuItem>
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
                <MenuItem value="OVERDUE">Overdue Only</MenuItem>
                <MenuItem value="DUE_SOON">Due Soon (&le; 5d)</MenuItem>
                <MenuItem value="UPCOMING">Upcoming (&gt; 5d)</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={6} sm={1}>
            <Tooltip title="Refresh Deadlines">
              <Button
                variant="outlined"
                fullWidth
                size="medium"
                onClick={fetchEvents}
                disabled={loading}
                sx={{ height: 40 }}
              >
                <RefreshIcon fontSize="small" />
              </Button>
            </Tooltip>
          </Grid>
        </Grid>
      </Paper>

      {/* Error Banner */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Calendar Deadlines Data Table */}
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
              <TableCell sx={{ fontWeight: 700, width: 140 }}>Due Date & Cutoff</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 150 }}>Countdown / Status</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 110 }}>Jurisdiction</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 160 }}>Regulation</TableCell>
              <TableCell sx={{ fontWeight: 700 }}>Deadline Title & Statutory Description</TableCell>
              <TableCell sx={{ fontWeight: 700, width: 150 }}>Type</TableCell>
              <TableCell align="center" sx={{ fontWeight: 700, width: 90 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredEvents.length === 0 && !loading ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <Typography variant="body2" color="text.secondary">
                    No compliance deadlines found matching selected criteria.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredEvents
                .slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)
                .map((ev) => (
                  <TableRow
                    key={ev.id}
                    hover
                    sx={{
                      cursor: 'pointer',
                      bgcolor: ev.days_remaining < 0 ? alpha(theme.palette.error.main, 0.03) : undefined,
                    }}
                    onClick={() => setSelectedEvent(ev)}
                  >
                    <TableCell>
                      <Typography variant="body2" sx={{ fontWeight: 700 }}>
                        {ev.due_date}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {ev.cutoff_time}
                      </Typography>
                    </TableCell>
                    <TableCell>{renderStatusBadge(ev.days_remaining, ev.status)}</TableCell>
                    <TableCell>{renderJurisdictionChip(ev.jurisdiction)}</TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>
                        {ev.regulation}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                        {ev.event_code}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontWeight: 600, color: theme.palette.text.primary }}>
                        {ev.title}
                      </Typography>
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}
                      >
                        {ev.description}
                      </Typography>
                      {ev.legal_citation && (
                        <Typography variant="caption" sx={{ color: theme.palette.primary.main, display: 'block', mt: 0.5, fontStyle: 'italic' }}>
                          Citation: {ev.legal_citation}
                        </Typography>
                      )}
                      {ev.rule_ids && ev.rule_ids.length > 0 && (
                        <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }}>
                          {ev.rule_ids.map((rid) => (
                            <Chip key={rid} label={rid} size="small" sx={{ fontSize: '10px', height: 18 }} />
                          ))}
                        </Stack>
                      )}
                    </TableCell>
                    <TableCell>{renderDeadlineTypeChip(ev.deadline_type)}</TableCell>
                    <TableCell align="center">
                      <Tooltip title="View Detailed Filing Instructions">
                        <IconButton
                          size="small"
                          color="primary"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedEvent(ev);
                          }}
                        >
                          <VisibilityIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))
            )}
          </TableBody>
        </Table>

        <TablePagination
          rowsPerPageOptions={[10, 15, 25, 50]}
          component="div"
          count={filteredEvents.length}
          rowsPerPage={rowsPerPage}
          page={page}
          onPageChange={(_, p) => setPage(p)}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10));
            setPage(0);
          }}
        />
      </TableContainer>

      {/* Detailed Deadline Inspection Modal */}
      {selectedEvent && (
        <Dialog
          open={Boolean(selectedEvent)}
          onClose={() => setSelectedEvent(null)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle sx={{ pb: 1 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Compliance Deadline Details
              </Typography>
              <IconButton size="small" onClick={() => setSelectedEvent(null)}>
                <CloseIcon />
              </IconButton>
            </Stack>
          </DialogTitle>
          <Divider />
          <DialogContent sx={{ pt: 2 }}>
            <Stack spacing={2.5}>
              <Box>
                <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                  {renderJurisdictionChip(selectedEvent.jurisdiction)}
                  {renderDeadlineTypeChip(selectedEvent.deadline_type)}
                  {renderStatusBadge(selectedEvent.days_remaining, selectedEvent.status)}
                </Stack>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  {selectedEvent.title}
                </Typography>
                <Typography variant="subtitle2" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                  Event Code: {selectedEvent.event_code}
                </Typography>
              </Box>

              <Paper variant="outlined" sx={{ p: 2, bgcolor: alpha(theme.palette.background.default, 0.5) }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                  Statutory Description & Obligations
                </Typography>
                <Typography variant="body2" sx={{ lineHeight: 1.6 }}>
                  {selectedEvent.description}
                </Typography>
                {selectedEvent.legal_citation && (
                  <Typography variant="caption" sx={{ color: theme.palette.primary.main, display: 'block', mt: 1, fontWeight: 600 }}>
                    Legal Citation: {selectedEvent.legal_citation}
                  </Typography>
                )}
              </Paper>

              <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                  <Paper variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Statutory Due Date
                    </Typography>
                    <Typography variant="body1" sx={{ fontWeight: 700 }}>
                      {selectedEvent.due_date} ({selectedEvent.cutoff_time})
                    </Typography>
                  </Paper>
                </Grid>

                <Grid item xs={12} sm={6}>
                  <Paper variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                      Regulation / Authority
                    </Typography>
                    <Typography variant="body1" sx={{ fontWeight: 700 }}>
                      {selectedEvent.regulation} ({selectedEvent.jurisdiction})
                    </Typography>
                  </Paper>
                </Grid>

                {selectedEvent.rule_ids && selectedEvent.rule_ids.length > 0 && (
                  <Grid item xs={12}>
                    <Paper variant="outlined" sx={{ p: 1.5 }}>
                      <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
                        Associated Compliance Rules
                      </Typography>
                      <Stack direction="row" spacing={1} flexWrap="wrap">
                        {selectedEvent.rule_ids.map((rid) => (
                          <Chip key={rid} label={rid} size="small" color="primary" variant="outlined" />
                        ))}
                      </Stack>
                    </Paper>
                  </Grid>
                )}

                {selectedEvent.affected_accounts && selectedEvent.affected_accounts.length > 0 && (
                  <Grid item xs={12}>
                    <Paper variant="outlined" sx={{ p: 1.5 }}>
                      <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
                        Affected Account Disclosures
                      </Typography>
                      <Stack direction="row" spacing={1} flexWrap="wrap">
                        {selectedEvent.affected_accounts.map((acc) => (
                          <Chip key={acc} label={acc} size="small" variant="filled" />
                        ))}
                      </Stack>
                    </Paper>
                  </Grid>
                )}
              </Grid>
            </Stack>
          </DialogContent>
          <Divider />
          <DialogActions sx={{ px: 3, py: 1.5 }}>
            <Button onClick={() => setSelectedEvent(null)}>Close</Button>
            <Button variant="contained" color="primary" startIcon={<OpenInNewIcon />}>
              Open Filing Checklist
            </Button>
          </DialogActions>
        </Dialog>
      )}
    </Box>
  );
};
