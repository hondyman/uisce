import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Chip,
  Button,
  TextField,
  InputAdornment,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
  Card,
  CardContent,
  CircularProgress,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  ListItemButton,
  Snackbar,
  Alert,
  useTheme,
} from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Storage as StorageIcon,
  CloudQueue as CloudQueueIcon,
  Speed as SpeedIcon,
  Analytics as AnalyticsIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Undo as UndoIcon,
  History as HistoryIcon,
  Restore as RestoreIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';

interface AuditHealth {
  status: string;
  datafusion_engine_url: string;
  engine_online: boolean;
  fallback_queue_size: number;
  global_ledger_table: string;
  repo2_oltp_history: string;
  repo3_olap_starrocks: string;
  timestamp: string;
}

interface AuditEventItem {
  event_id: string;
  tenant_id: string;
  tenant_instance_id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  user_id: string;
  timestamp: string;
}

interface HistoryRecord {
  event_id: string;
  timestamp: string;
  action: string;
  state: Record<string, any> | null;
}

export const AuditObservabilityPage: React.FC = () => {
  const theme = useTheme();
  const [health, setHealth] = useState<AuditHealth | null>(null);
  const [events, setEvents] = useState<AuditEventItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [restoringId, setRestoringId] = useState<string | null>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Time Machine Modal State
  const [isHistoryOpen, setIsHistoryOpen] = useState<boolean>(false);
  const [historyLoading, setHistoryLoading] = useState<boolean>(false);
  const [historyData, setHistoryData] = useState<HistoryRecord[]>([]);
  const [activeEntity, setActiveEntity] = useState<{ type: string; id: string }>({ type: '', id: '' });

  const fetchData = async () => {
    try {
      setLoading(true);
      const [healthRes, eventsRes] = await Promise.all([
        apiClient<AuditHealth>('/rbac/audit/health'),
        apiClient<AuditEventItem[]>('/rbac/audit/global'),
      ]);
      setHealth(healthRes);
      setEvents(Array.isArray(eventsRes) ? eventsRes : []);
    } catch (error) {
      console.error('Failed to load audit observability data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleOpenHistory = async (entityType: string, entityId: string) => {
    try {
      setActiveEntity({ type: entityType, id: entityId });
      setHistoryLoading(true);
      setIsHistoryOpen(true);
      const res = await apiClient<HistoryRecord[]>(`/rbac/audit/history/${entityType}/${entityId}`);
      setHistoryData(Array.isArray(res) ? res : []);
    } catch (err) {
      console.error('Failed to fetch entity history:', err);
      setHistoryData([]);
    } finally {
      setHistoryLoading(false);
    }
  };

  const handleRestoreState = async (eventID: string, statePayload?: Record<string, any> | null) => {
    try {
      setRestoringId(eventID);
      if (statePayload) {
        await apiClient(`/rbac/audit/recover/${activeEntity.type}/${activeEntity.id}`, {
          method: 'POST',
          body: JSON.stringify({ state: statePayload }),
        });
      } else {
        await apiClient('/rbac/audit/recover', {
          method: 'POST',
          body: JSON.stringify({
            event_id: eventID,
            entity_type: activeEntity.type,
            entity_id: activeEntity.id,
          }),
        });
      }
      setToastMessage(`Successfully restored ${activeEntity.type} state!`);
      setIsHistoryOpen(false);
      fetchData();
    } catch (err) {
      console.error('Failed to restore entity state:', err);
      setToastMessage(`Restoration failed: ${err}`);
    } finally {
      setRestoringId(null);
    }
  };

  const filteredEvents = events.filter(
    (e) =>
      e.event_id?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.tenant_id?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.entity_type?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.action?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Box sx={{ p: 4, bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Toast Notification */}
      <Snackbar
        open={Boolean(toastMessage)}
        autoHideDuration={6000}
        onClose={() => setToastMessage(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert onClose={() => setToastMessage(null)} severity="success" sx={{ width: '100%' }}>
          {toastMessage}
        </Alert>
      </Snackbar>

      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={4}>
        <Box>
          <Typography variant="h4" fontWeight={700}>
            Lakehouse Audit & Point-in-Time Recovery
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Real-time pipeline telemetry, 3-Tier Iceberg ledger, and Time Machine undo recovery
          </Typography>
        </Box>
        <Button
          variant="outlined"
          startIcon={<RefreshIcon />}
          onClick={fetchData}
          sx={{ borderRadius: 2 }}
        >
          Refresh Now
        </Button>
      </Stack>

      {/* Top Metric Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" color="text.secondary" fontWeight={600}>
                  DataFusion Engine
                </Typography>
                <CloudQueueIcon color="primary" />
              </Stack>
              <Stack direction="row" spacing={1} alignItems="center" mt={2}>
                <Chip
                  label={health?.engine_online ? 'ONLINE' : 'FALLBACK'}
                  color={health?.engine_online ? 'success' : 'warning'}
                  size="small"
                  icon={health?.engine_online ? <CheckCircleIcon /> : <WarningIcon />}
                />
                <Typography variant="caption" color="text.secondary">
                  Port 8081
                </Typography>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" color="text.secondary" fontWeight={600}>
                  Fallback Queue
                </Typography>
                <SpeedIcon color="warning" />
              </Stack>
              <Typography variant="h4" fontWeight={700} mt={1}>
                {health?.fallback_queue_size ?? 0}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Buffered in Postgres Log
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" color="text.secondary" fontWeight={600}>
                  Repo 1: Global Ledger
                </Typography>
                <StorageIcon color="secondary" />
              </Stack>
              <Typography variant="h6" fontWeight={700} mt={1.5}>
                uisce_global_audit
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Master System of Record
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" color="text.secondary" fontWeight={600}>
                  Repo 3: StarRocks OLAP
                </Typography>
                <AnalyticsIcon color="success" />
              </Stack>
              <Typography variant="h6" fontWeight={700} mt={1.5}>
                Active Gold Copy
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Flattened Star Schema
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Audit Ledger DataGrid */}
      <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={3}>
          <Box>
            <Typography variant="h6" fontWeight={700}>
              Global Audit Event Ledger
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Live audit events streamed into the DataFusion & Iceberg Lakehouse
            </Typography>
          </Box>
          <TextField
            size="small"
            placeholder="Filter audit events..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" sx={{ color: 'text.secondary' }} />
                </InputAdornment>
              ),
              sx: { borderRadius: 2 },
            }}
          />
        </Stack>

        {loading ? (
          <Box sx={{ display: 'flex', py: 6, justifyContent: 'center' }}>
            <CircularProgress />
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 700 }}>Timestamp</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Event ID</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Tenant ID</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Action</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Entity Type</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Entity ID</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>User</TableCell>
                  <TableCell sx={{ fontWeight: 700 }} align="right">Time Machine</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredEvents.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                      No audit events found.
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredEvents.map((evt) => (
                    <TableRow key={evt.event_id} hover>
                      <TableCell>{new Date(evt.timestamp).toLocaleString()}</TableCell>
                      <TableCell>
                        <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                          {evt.event_id?.substring(0, 8)}...
                        </Typography>
                      </TableCell>
                      <TableCell>{evt.tenant_id}</TableCell>
                      <TableCell>
                        <Chip
                          label={evt.action}
                          size="small"
                          color={
                            evt.action === 'created'
                              ? 'success'
                              : evt.action === 'deleted'
                              ? 'error'
                              : 'primary'
                          }
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>{evt.entity_type}</TableCell>
                      <TableCell>
                        <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                          {evt.entity_id}
                        </Typography>
                      </TableCell>
                      <TableCell>{evt.user_id || 'admin'}</TableCell>
                      <TableCell align="right">
                        <Stack direction="row" spacing={1} justifyContent="flex-end">
                          <Tooltip title="View Point-in-Time History Timeline">
                            <IconButton
                              size="small"
                              color="secondary"
                              onClick={() => handleOpenHistory(evt.entity_type, evt.entity_id)}
                            >
                              <HistoryIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Quick Undo & Restore to PostgreSQL">
                            <IconButton
                              size="small"
                              color="primary"
                              onClick={() => handleRestoreState(evt.event_id)}
                              disabled={restoringId === evt.event_id}
                            >
                              {restoringId === evt.event_id ? (
                                <CircularProgress size={16} />
                              ) : (
                                <UndoIcon fontSize="small" />
                              )}
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      {/* --- TIME MACHINE RECOVERY MODAL --- */}
      <Dialog
        open={isHistoryOpen}
        onClose={() => setIsHistoryOpen(false)}
        maxWidth="md"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ borderBottom: `1px solid ${theme.palette.divider}`, pb: 2 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <HistoryIcon color="secondary" />
            <Box>
              <Typography variant="h6" fontWeight={700}>
                Point-in-Time Recovery Timeline
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Entity: {activeEntity.type} ({activeEntity.id})
              </Typography>
            </Box>
          </Stack>
        </DialogTitle>

        <DialogContent sx={{ p: 3 }}>
          {historyLoading ? (
            <Box sx={{ display: 'flex', py: 6, justifyContent: 'center' }}>
              <CircularProgress />
            </Box>
          ) : historyData.length === 0 ? (
            <Typography variant="body2" color="text.secondary" align="center" sx={{ py: 4 }}>
              No historical snapshots found in Iceberg Repo 2.
            </Typography>
          ) : (
            <List disablePadding>
              {historyData.map((record) => (
                <Paper
                  key={record.event_id}
                  elevation={0}
                  sx={{
                    p: 2,
                    mb: 2,
                    borderRadius: 2,
                    border: `1px solid ${theme.palette.divider}`,
                    bgcolor: 'action.hover',
                  }}
                >
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Chip
                        label={record.action}
                        size="small"
                        color={record.action === 'deleted' ? 'error' : 'primary'}
                      />
                      <Typography variant="caption" color="text.secondary">
                        {new Date(record.timestamp).toLocaleString()}
                      </Typography>
                    </Stack>
                    <Button
                      size="small"
                      variant="contained"
                      startIcon={<RestoreIcon />}
                      onClick={() => handleRestoreState(record.event_id, record.state)}
                      disabled={restoringId === record.event_id}
                    >
                      Restore State
                    </Button>
                  </Stack>

                  {record.state ? (
                    <Box
                      component="pre"
                      sx={{
                        p: 1.5,
                        m: 0,
                        borderRadius: 1,
                        bgcolor: 'background.paper',
                        border: `1px solid ${theme.palette.divider}`,
                        fontFamily: 'monospace',
                        fontSize: '0.75rem',
                        overflowX: 'auto',
                      }}
                    >
                      {JSON.stringify(record.state, null, 2)}
                    </Box>
                  ) : (
                    <Typography variant="caption" color="text.secondary">
                      (No state payload available)
                    </Typography>
                  )}
                </Paper>
              ))}
            </List>
          )}
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2, borderTop: `1px solid ${theme.palette.divider}` }}>
          <Button onClick={() => setIsHistoryOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default AuditObservabilityPage;
