import { useState, useEffect, useCallback, SyntheticEvent } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Typography,
  TextField,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Chip,
  Dialog,
  DialogContent,
  DialogActions,
  Paper,
  Grid,
  CircularProgress,
  Tooltip,
  Alert,
} from '@mui/material';
import { Refresh, Visibility, Warning, Error as ErrorIcon, Info, CheckCircle, Security, Download } from '@mui/icons-material';
import { Tabs, Tab } from '@mui/material';
import ModalHeader from '../../components/ModalHeader';
// DatePicker intentionally not used in current UI, removed to satisfy noUnusedLocals
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';

interface AuditEvent {
  id: string;
  timestamp: string;
  event_type: string;
  severity: string;
  user_id: string;
  tenant_id: string;
  resource_type: string;
  resource_id: string;
  action: string;
  ip_address: string;
  success: boolean;
  error_message?: string;
  details: { [key: string]: any };
}

interface AuditSummary {
  total_events: number;
  events_by_type: { [key: string]: number };
  events_by_severity: { [key: string]: number };
  events_by_user: { [key: string]: number };
  recent_events: AuditEvent[];
  time_range: {
    start: string;
    end: string;
  };
}

export const AuditDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [summary, setSummary] = useState<AuditSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null);
  const [showEventDetails, setShowEventDetails] = useState(false);

  // Filters
  const [filters, setFilters] = useState({
    user_id: '',
    event_type: '',
    severity: '',
    resource_type: '',
    start_date: null as Date | null,
    end_date: null as Date | null,
    success: '',
  });

  const loadAuditData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Load summary
      const summaryResponse = await fetch('/api/audit/summary');
      if (!summaryResponse.ok) {
        throw { message: 'Failed to load audit summary' };
      }
      const summaryData = await summaryResponse.json();
      setSummary(summaryData);

      // Load events with filters
      const queryParams = new URLSearchParams();
      Object.entries(filters).forEach(([key, value]) => {
        if (value && value !== '') {
          if (value instanceof Date) {
            queryParams.append(key, value.toISOString());
          } else {
            queryParams.append(key, value.toString());
          }
        }
      });

      const eventsResponse = await fetch(`/api/audit/events?${queryParams}`);
      if (!eventsResponse.ok) {
        throw { message: 'Failed to load audit events' };
      }
      const eventsData = await eventsResponse.json();
      setEvents(eventsData.events || []);

    } catch (err: any) {
      setError(err?.message || 'Failed to load audit data');
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    loadAuditData();
  }, [loadAuditData]);

  const handleFilterChange = (field: string, value: any) => {
    setFilters(prev => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleTabChange = (_event: SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const handleViewEventDetails = (event: AuditEvent) => {
    setSelectedEvent(event);
    setShowEventDetails(true);
  };

  const handleExportEvents = async () => {
    try {
      const response = await fetch('/api/audit/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          filter: filters,
          format: 'csv',
          report_name: `audit_export_${new Date().toISOString().split('T')[0]}`,
        }),
      });

      if (!response.ok) {
        throw { message: 'Export failed' };
      }

      // Trigger download
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit_export_${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

    } catch (err) {
      setError('Failed to export audit events');
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <ErrorIcon color="error" />;
      case 'high':
        return <Warning color="warning" />;
      case 'medium':
        return <Info color="info" />;
      case 'low':
        return <CheckCircle color="success" />;
      default:
        return <Info />;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'error';
      case 'high':
        return 'warning';
      case 'medium':
        return 'info';
      case 'low':
        return 'success';
      default:
        return 'default';
    }
  };

  // mark helper as intentionally present for future UI uses
  void getSeverityColor;

  const getEventTypeColor = (eventType: string) => {
    if (eventType.includes('login')) return 'primary';
    if (eventType.includes('data')) return 'secondary';
    if (eventType.includes('config')) return 'warning';
    if (eventType.includes('violation')) return 'error';
    return 'default';
  };

  if (loading && !summary) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <LocalizationProvider dateAdapter={AdapterDateFns}>
      <Box sx={{ p: 3 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Box display="flex" alignItems="center">
            <Security sx={{ mr: 2, fontSize: 32 }} />
            <Typography variant="h4" component="h1">
              Audit Trail & Compliance
            </Typography>
          </Box>
          <Box>
            <Button
              variant="outlined"
              startIcon={<Refresh />}
              onClick={loadAuditData}
              sx={{ mr: 2 }}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              startIcon={<Download />}
              onClick={handleExportEvents}
            >
              Export
            </Button>
          </Box>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 3 }}>
            {error}
          </Alert>
        )}

        <Tabs value={activeTab} onChange={handleTabChange} sx={{ mb: 3 }}>
          <Tab label="Overview" />
          <Tab label="Audit Events" />
          <Tab label="Compliance Reports" />
        </Tabs>

        {activeTab === 0 && summary && (
          <Grid container spacing={3}>
            {/* Summary Cards */}
            <Grid item xs={12} md={3}>
              <Card>
                <CardContent>
                  <Typography color="textSecondary" gutterBottom>
                    Total Events
                  </Typography>
                  <Typography variant="h4">
                    {summary.total_events.toLocaleString()}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} md={3}>
              <Card>
                <CardContent>
                  <Typography color="textSecondary" gutterBottom>
                    Critical Events
                  </Typography>
                  <Typography variant="h4" color="error">
                    {(summary.events_by_severity?.critical || 0).toLocaleString()}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} md={3}>
              <Card>
                <CardContent>
                  <Typography color="textSecondary" gutterBottom>
                    Data Access Events
                  </Typography>
                  <Typography variant="h4" color="primary">
                    {(summary.events_by_type?.data_access || 0).toLocaleString()}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} md={3}>
              <Card>
                <CardContent>
                  <Typography color="textSecondary" gutterBottom>
                    Failed Operations
                  </Typography>
                  <Typography variant="h4" color="warning">
                    {events.filter(e => !e.success).length.toLocaleString()}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            {/* Recent Events */}
            <Grid item xs={12}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Recent Audit Events
                  </Typography>
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>Time</TableCell>
                          <TableCell>Event Type</TableCell>
                          <TableCell>User</TableCell>
                          <TableCell>Resource</TableCell>
                          <TableCell>Status</TableCell>
                          <TableCell>Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {summary.recent_events.slice(0, 10).map((event) => (
                          <TableRow key={event.id}>
                            <TableCell>
                              {new Date(event.timestamp).toLocaleString()}
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={event.event_type.replace('_', ' ')}
                                size="small"
                                color={getEventTypeColor(event.event_type)}
                              />
                            </TableCell>
                            <TableCell>{event.user_id || 'System'}</TableCell>
                            <TableCell>
                              {event.resource_type}: {event.resource_id}
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={event.success ? 'Success' : 'Failed'}
                                size="small"
                                color={event.success ? 'success' : 'error'}
                              />
                            </TableCell>
                            <TableCell>
                              <IconButton
                                size="small"
                                onClick={() => handleViewEventDetails(event)}
                              >
                                <Visibility />
                              </IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        )}

        {activeTab === 1 && (
          <Box>
            {/* Filters */}
            <Paper sx={{ p: 2, mb: 3 }}>
              <Typography variant="h6" gutterBottom>
                Filters
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6} md={3}>
                  <TextField
                    fullWidth
                    label="User ID"
                    value={filters.user_id}
                    onChange={(e) => handleFilterChange('user_id', e.target.value)}
                    size="small"
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Event Type</InputLabel>
                    <Select
                      value={filters.event_type}
                      onChange={(e) => handleFilterChange('event_type', e.target.value)}
                    >
                      <MenuItem value="">All</MenuItem>
                      <MenuItem value="login">Login</MenuItem>
                      <MenuItem value="data_access">Data Access</MenuItem>
                      <MenuItem value="data_modify">Data Modify</MenuItem>
                      <MenuItem value="calculation_run">Calculation</MenuItem>
                      <MenuItem value="policy_violation">Policy Violation</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Severity</InputLabel>
                    <Select
                      value={filters.severity}
                      onChange={(e) => handleFilterChange('severity', e.target.value)}
                    >
                      <MenuItem value="">All</MenuItem>
                      <MenuItem value="low">Low</MenuItem>
                      <MenuItem value="medium">Medium</MenuItem>
                      <MenuItem value="high">High</MenuItem>
                      <MenuItem value="critical">Critical</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Status</InputLabel>
                    <Select
                      value={filters.success}
                      onChange={(e) => handleFilterChange('success', e.target.value)}
                    >
                      <MenuItem value="">All</MenuItem>
                      <MenuItem value="true">Success</MenuItem>
                      <MenuItem value="false">Failed</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
              </Grid>
            </Paper>

            {/* Events Table */}
            <TableContainer component={Paper}>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Timestamp</TableCell>
                    <TableCell>Event Type</TableCell>
                    <TableCell>Severity</TableCell>
                    <TableCell>User</TableCell>
                    <TableCell>Resource</TableCell>
                    <TableCell>Action</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {events.map((event) => (
                    <TableRow key={event.id}>
                      <TableCell>
                        {new Date(event.timestamp).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={event.event_type.replace('_', ' ')}
                          size="small"
                          color={getEventTypeColor(event.event_type)}
                        />
                      </TableCell>
                      <TableCell>
                        <Box display="flex" alignItems="center">
                          {getSeverityIcon(event.severity)}
                          <Typography variant="body2" sx={{ ml: 1 }}>
                            {event.severity}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>{event.user_id || 'System'}</TableCell>
                      <TableCell>
                        {event.resource_type && event.resource_id
                          ? `${event.resource_type}:${event.resource_id}`
                          : '-'
                        }
                      </TableCell>
                      <TableCell>{event.action}</TableCell>
                      <TableCell>
                        <Chip
                          label={event.success ? 'Success' : 'Failed'}
                          size="small"
                          color={event.success ? 'success' : 'error'}
                        />
                      </TableCell>
                      <TableCell>
                        <Tooltip title="View Details">
                          <IconButton
                            size="small"
                            onClick={() => handleViewEventDetails(event)}
                          >
                            <Visibility />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>

            {events.length === 0 && !loading && (
              <Box textAlign="center" py={4}>
                <Typography variant="body1" color="textSecondary">
                  No audit events found matching the current filters.
                </Typography>
              </Box>
            )}
          </Box>
        )}

        {activeTab === 2 && (
          <Box>
            <Typography variant="h6" gutterBottom>
              Compliance Reports
            </Typography>
            <Alert severity="info">
              Compliance report generation will be available in the next update.
              This feature will allow you to generate detailed compliance reports
              for regulatory requirements and audit purposes.
            </Alert>
          </Box>
        )}

        {/* Event Details Dialog */}
        <Dialog
          open={showEventDetails}
          onClose={() => setShowEventDetails(false)}
          maxWidth="md"
          fullWidth
        >
          <ModalHeader title="Audit Event Details" onClose={() => setShowEventDetails(false)} />
          <DialogContent>
            {selectedEvent && (
              <Box>
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2">Event ID</Typography>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                      {selectedEvent.id}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2">Timestamp</Typography>
                    <Typography variant="body2">
                      {new Date(selectedEvent.timestamp).toLocaleString()}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2">Event Type</Typography>
                    <Chip
                      label={selectedEvent.event_type.replace('_', ' ')}
                      size="small"
                      color={getEventTypeColor(selectedEvent.event_type)}
                    />
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2">Severity</Typography>
                    <Box display="flex" alignItems="center">
                      {getSeverityIcon(selectedEvent.severity)}
                      <Typography variant="body2" sx={{ ml: 1 }}>
                        {selectedEvent.severity}
                      </Typography>
                    </Box>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2">User ID</Typography>
                    <Typography variant="body2">
                      {selectedEvent.user_id || 'System'}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2">IP Address</Typography>
                    <Typography variant="body2">
                      {selectedEvent.ip_address || 'N/A'}
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2">Details</Typography>
                    <Paper sx={{ p: 2, mt: 1, backgroundColor: 'grey.50' }}>
                      <Box component="pre" sx={{ m: 0, fontSize: '0.875rem', fontFamily: 'monospace' }}>
                        {JSON.stringify(selectedEvent.details, null, 2)}
                      </Box>
                    </Paper>
                  </Grid>
                  {!selectedEvent.success && selectedEvent.error_message && (
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Error Message</Typography>
                      <Alert severity="error" sx={{ mt: 1 }}>
                        {selectedEvent.error_message}
                      </Alert>
                    </Grid>
                  )}
                </Grid>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setShowEventDetails(false)}>Close</Button>
          </DialogActions>
        </Dialog>
      </Box>
    </LocalizationProvider>
  );
};
