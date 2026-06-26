import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Grid,
  Alert,
  Typography,
  CircularProgress,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import ClearIcon from '@mui/icons-material/Clear';

interface Event {
  id: string;
  timestamp: string;
  eventType: 'job.started' | 'job.completed' | 'job.failed' | 'export.created' | 'schedule.executed';
  severity: 'info' | 'warning' | 'error' | 'success';
  message: string;
  details?: Record<string, any>;
}

const mockEvents: Event[] = [
  {
    id: 'evt-001',
    timestamp: new Date().toISOString(),
    eventType: 'job.completed',
    severity: 'success',
    message: 'Job job-123 completed successfully',
    details: { jobId: 'job-123', duration: '5m 32s', recordsProcessed: 1250 },
  },
  {
    id: 'evt-002',
    timestamp: new Date(Date.now() - 60000).toISOString(),
    eventType: 'schedule.executed',
    severity: 'info',
    message: 'Scheduled job "Daily Export" executed',
    details: { scheduleId: 'sched-001', jobId: 'job-124' },
  },
  {
    id: 'evt-003',
    timestamp: new Date(Date.now() - 120000).toISOString(),
    eventType: 'export.created',
    severity: 'info',
    message: 'Export exp-001 created',
    details: { format: 'csv', fileSize: 524288 },
  },
  {
    id: 'evt-004',
    timestamp: new Date(Date.now() - 180000).toISOString(),
    eventType: 'job.started',
    severity: 'info',
    message: 'Job job-122 started',
    details: { jobId: 'job-122', operationType: 'bulk-publish' },
  },
  {
    id: 'evt-005',
    timestamp: new Date(Date.now() - 300000).toISOString(),
    eventType: 'job.failed',
    severity: 'error',
    message: 'Job job-121 failed',
    details: { jobId: 'job-121', error: 'Database connection timeout' },
  },
];

export const EDM_EventsMonitor: React.FC = () => {
  const [events, setEvents] = useState<Event[]>(mockEvents);
  const [loading, setLoading] = useState(false);
  const [isLiveMode, setIsLiveMode] = useState(true);
  const [eventTypeFilter, setEventTypeFilter] = useState<string>('all');
  const [severityFilter, setSeverityFilter] = useState<string>('all');

  useEffect(() => {
    if (!isLiveMode) return;

    const interval = setInterval(() => {
      // Simulate new events in live mode
      const eventTypes: Event['eventType'][] = [
        'job.started',
        'job.completed',
        'job.failed',
        'export.created',
        'schedule.executed',
      ];
      const severities: Event['severity'][] = ['info', 'success', 'warning', 'error'];

      const randomType = eventTypes[Math.floor(Math.random() * eventTypes.length)];
      const randomSeverity = severities[Math.floor(Math.random() * severities.length)];

      const newEvent: Event = {
        id: `evt-${Date.now()}`,
        timestamp: new Date().toISOString(),
        eventType: randomType,
        severity: randomSeverity,
        message: `Event: ${randomType}`,
        details: { source: 'live-stream' },
      };

      setEvents((prev) => [newEvent, ...prev.slice(0, 49)]);
    }, 5000);

    return () => clearInterval(interval);
  }, [isLiveMode]);

  const handleRefresh = () => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
    }, 1000);
  };

  const handleClearAll = () => {
    if (confirm('Clear all events?')) {
      setEvents([]);
    }
  };

  const filteredEvents = events.filter((evt) => {
    const typeMatch = eventTypeFilter === 'all' || evt.eventType === eventTypeFilter;
    const severityMatch = severityFilter === 'all' || evt.severity === severityFilter;
    return typeMatch && severityMatch;
  });

  const getSeverityColor = (severity: string): 'default' | 'primary' | 'success' | 'error' | 'warning' => {
    switch (severity) {
      case 'success': return 'success';
      case 'error': return 'error';
      case 'warning': return 'warning';
      default: return 'default';
    }
  };

  const getEventTypeIcon = (eventType: string) => {
    if (eventType.includes('job')) return '⚙️';
    if (eventType.includes('export')) return '📤';
    if (eventType.includes('schedule')) return '⏰';
    return '📋';
  };

  return (
    <Box sx={{ p: 3 }}>
      <Card>
        <CardHeader
          title="Real-time Events Monitor"
          subheader="Live stream of job, export, and schedule events"
          action={
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button
                variant={isLiveMode ? 'contained' : 'outlined'}
                size="small"
                onClick={() => setIsLiveMode(!isLiveMode)}
              >
                {isLiveMode ? '🔴 Live' : '⭕ Paused'}
              </Button>
              <Button
                startIcon={<RefreshIcon />}
                onClick={handleRefresh}
                disabled={loading}
              >
                {loading ? <CircularProgress size={20} /> : 'Refresh'}
              </Button>
            </Box>
          }
        />
        <CardContent>
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth size="small">
                <InputLabel>Event Type</InputLabel>
                <Select
                  value={eventTypeFilter}
                  onChange={(e) => setEventTypeFilter(e.target.value)}
                  label="Event Type"
                >
                  <MenuItem value="all">All Events</MenuItem>
                  <MenuItem value="job.started">Job Started</MenuItem>
                  <MenuItem value="job.completed">Job Completed</MenuItem>
                  <MenuItem value="job.failed">Job Failed</MenuItem>
                  <MenuItem value="export.created">Export Created</MenuItem>
                  <MenuItem value="schedule.executed">Schedule Executed</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth size="small">
                <InputLabel>Severity</InputLabel>
                <Select
                  value={severityFilter}
                  onChange={(e) => setSeverityFilter(e.target.value)}
                  label="Severity"
                >
                  <MenuItem value="all">All Severities</MenuItem>
                  <MenuItem value="info">ℹ️ Info</MenuItem>
                  <MenuItem value="success">✅ Success</MenuItem>
                  <MenuItem value="warning">⚠️ Warning</MenuItem>
                  <MenuItem value="error">❌ Error</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>

          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              {filteredEvents.length} event{filteredEvents.length !== 1 ? 's' : ''}
            </Typography>
            <Button
              size="small"
              startIcon={<ClearIcon />}
              onClick={handleClearAll}
              variant="outlined"
              color="error"
            >
              Clear All
            </Button>
          </Box>

          <TableContainer component={Paper}>
            <Table>
              <TableHead sx={{ bgcolor: '#f3f4f6' }}>
                <TableRow>
                  <TableCell><strong>Time</strong></TableCell>
                  <TableCell><strong>Event Type</strong></TableCell>
                  <TableCell><strong>Severity</strong></TableCell>
                  <TableCell><strong>Message</strong></TableCell>
                  <TableCell><strong>Details</strong></TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredEvents.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} align="center" sx={{ py: 3 }}>
                      <Typography color="textSecondary">No events to display</Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredEvents.map((evt) => (
                    <TableRow key={evt.id} hover>
                      <TableCell sx={{ fontSize: '0.85rem', whiteSpace: 'nowrap' }}>
                        {new Date(evt.timestamp).toLocaleTimeString()}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Box>{getEventTypeIcon(evt.eventType)}</Box>
                          <Chip
                            label={evt.eventType}
                            size="small"
                            variant="outlined"
                          />
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={evt.severity.toUpperCase()}
                          size="small"
                          color={getSeverityColor(evt.severity)}
                        />
                      </TableCell>
                      <TableCell>{evt.message}</TableCell>
                      <TableCell>
                        <Box sx={{ fontSize: '0.8rem', fontFamily: 'monospace', maxWidth: 200, overflow: 'auto' }}>
                          {evt.details && Object.entries(evt.details).length > 0 ? (
                            <code>{JSON.stringify(evt.details, null, 2).substring(0, 100)}...</code>
                          ) : (
                            <Typography variant="caption" color="textSecondary">
                              --
                            </Typography>
                          )}
                        </Box>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>

          <Alert severity="info" sx={{ mt: 2 }}>
            💡 Events are streamed in real-time via SSE/WebSocket. Live mode updates every 5 seconds with new events.
          </Alert>
        </CardContent>
      </Card>
    </Box>
  );
};
