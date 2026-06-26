import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Alert,
  CircularProgress,
  Typography,
  Switch,
  FormControlLabel,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import PauseIcon from '@mui/icons-material/Pause';
import DeleteIcon from '@mui/icons-material/Delete';
import { API_BASE_URL, DEFAULT_TENANT_ID, SCHEDULER_CONFIG, getApiHeaders } from '../api/config';

interface Schedule {
  id: string;
  name: string;
  operation_type: string;
  schedule_type: 'once' | 'daily' | 'weekly' | 'monthly' | 'cron';
  is_active: boolean;
  next_run_at: string;
  last_run_at?: string;
  timezone: string;
  run_count: number;
  success_count: number;
  failure_count: number;
  created_at: string;
  cron_expression?: string;
}

export const EDM_SchedulingManager: React.FC = () => {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [openDialog, setOpenDialog] = useState(false);
  const [newSchedule, setNewSchedule] = useState({
    name: '',
    operation_type: 'bulk-publish',
    schedule_type: SCHEDULER_CONFIG.DEFAULT_SCHEDULE_TYPE as const,
    timezone: SCHEDULER_CONFIG.DEFAULT_TIMEZONE,
    cron_expression: '0 2 * * *',
  });
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch schedules on mount
  useEffect(() => {
    fetchSchedules();
  }, []);

  const fetchSchedules = async () => {
    try {
      setInitialLoading(true);
      setError(null);
      const response = await fetch(`${API_BASE_URL}/schedules`, {
        headers: getApiHeaders(DEFAULT_TENANT_ID),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch schedules: ${response.statusText}`);
      }

      const data: { schedules: Schedule[] } = await response.json();
      setSchedules(data.schedules || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch schedules');
    } finally {
      setInitialLoading(false);
    }
  };

  const handleOpenDialog = () => {
    setOpenDialog(true);
  };

  const handleCloseDialog = () => {
    setOpenDialog(false);
    setNewSchedule({
      name: '',
      operation_type: 'bulk-publish',
      schedule_type: 'daily',
      timezone: 'UTC',
      cron_expression: '0 2 * * *',
    });
  };

  const handleCreateSchedule = async () => {
    if (!newSchedule.name) {
      setError('Schedule name is required');
      return;
    }
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/schedules`, {
        method: 'POST',
        headers: getApiHeaders(DEFAULT_TENANT_ID),
        body: JSON.stringify({
          name: newSchedule.name,
          operation_type: newSchedule.operation_type,
          schedule_type: newSchedule.schedule_type,
          timezone: newSchedule.timezone,
          cron_expression: newSchedule.schedule_type === 'cron' ? newSchedule.cron_expression : undefined,
          job_template: {},
          created_by: DEFAULT_TENANT_ID,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create schedule: ${response.statusText}`);
      }

      handleCloseDialog();
      await fetchSchedules();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create schedule');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleActive = async (id: string, is_active: boolean) => {
    try {
      const endpoint = is_active ? 'pause' : 'resume';
      const response = await fetch(`${API_BASE_URL}/schedules/${id}/${endpoint}`, {
        method: 'POST',
        headers: getApiHeaders(DEFAULT_TENANT_ID),
      });

      if (!response.ok) {
        throw new Error(`Failed to ${endpoint} schedule`);
      }

      await fetchSchedules();
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${is_active ? 'pause' : 'resume'} schedule`);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this schedule?')) return;
    
    try {
      const response = await fetch(`${API_BASE_URL}/schedules/${id}`, {
        method: 'DELETE',
        headers: getApiHeaders(DEFAULT_TENANT_ID),
      });

      if (!response.ok) {
        throw new Error(`Failed to delete schedule: ${response.statusText}`);
      }

      await fetchSchedules();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete schedule');
    }
  };

  const getScheduleTypeLabel = (type: string, cronExpr?: string): string => {
    if (type === 'cron') return `Cron: ${cronExpr}`;
    return type.charAt(0).toUpperCase() + type.slice(1);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Card>
        <CardHeader
          title="Job Scheduling"
          subheader="Schedule jobs to run at specific times or recurring intervals"
          action={
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleOpenDialog}
            >
              New Schedule
            </Button>
          }
        />
        <CardContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          
          {initialLoading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
              <CircularProgress />
            </Box>
          )}

          {!initialLoading && (
            <>
            <TableContainer component={Paper}>
            <Table>
              <TableHead sx={{ bgcolor: '#f3f4f6' }}>
                <TableRow>
                  <TableCell><strong>Name</strong></TableCell>
                  <TableCell><strong>Schedule Type</strong></TableCell>
                  <TableCell><strong>Timezone</strong></TableCell>
                  <TableCell align="center"><strong>Status</strong></TableCell>
                  <TableCell align="right"><strong>Runs</strong></TableCell>
                  <TableCell><strong>Next Run</strong></TableCell>
                  <TableCell align="center"><strong>Actions</strong></TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {schedules.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center" sx={{ py: 3 }}>
                      <Typography color="textSecondary">No schedules yet</Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  schedules.map((sched) => (
                    <TableRow key={sched.id} hover>
                      <TableCell>
                        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                          {sched.name}
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          {sched.operation_type}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={getScheduleTypeLabel(sched.schedule_type, sched.cron_expression)}
                          size="small"
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>{sched.timezone}</TableCell>
                      <TableCell align="center">
                        <FormControlLabel
                          control={
                            <Switch
                              checked={sched.is_active}
                              onChange={() => handleToggleActive(sched.id, sched.is_active)}
                              size="small"
                            />
                          }
                          label={sched.is_active ? 'Active' : 'Paused'}
                        />
                      </TableCell>
                      <TableCell align="right">
                        <Box sx={{ fontSize: '0.85rem' }}>
                          {sched.success_count}/{sched.run_count} successful
                        </Box>
                        {sched.failure_count > 0 && (
                          <Chip
                            label={`${sched.failure_count} failed`}
                            size="small"
                            color="error"
                            sx={{ mt: 0.5 }}
                          />
                        )}
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.85rem' }}>
                        {new Date(sched.next_run_at).toLocaleString()}
                      </TableCell>
                      <TableCell align="center">
                        <Button
                          size="small"
                          startIcon={sched.is_active ? <PauseIcon /> : <PlayArrowIcon />}
                          onClick={() => handleToggleActive(sched.id, sched.is_active)}
                          sx={{ mr: 1 }}
                        >
                          {sched.is_active ? 'Pause' : 'Resume'}
                        </Button>
                        <Button
                          size="small"
                          startIcon={<DeleteIcon />}
                          color="error"
                          onClick={() => handleDelete(sched.id)}
                        >
                          Delete
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>

          <Alert severity="info" sx={{ mt: 2 }}>
            💡 Schedules check for due jobs every minute. Supports timezone-aware scheduling with 5 schedule types.
          </Alert>
            </>
          )}
        </CardContent>
      </Card>

      {/* Create Schedule Dialog */}
      <Dialog open={openDialog} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Schedule</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <TextField
            label="Schedule Name"
            fullWidth
            value={newSchedule.name}
            onChange={(e) => setNewSchedule({ ...newSchedule, name: e.target.value })}
            placeholder="e.g., Daily Export Job"
            sx={{ mb: 2 }}
          />

          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Schedule Type</InputLabel>
            <Select
              value={newSchedule.schedule_type}
              onChange={(e) => setNewSchedule({ ...newSchedule, schedule_type: e.target.value as any })}
              label="Schedule Type"
            >
              <MenuItem value="once">Once - Run one time</MenuItem>
              <MenuItem value="daily">Daily - Every day</MenuItem>
              <MenuItem value="weekly">Weekly - Every week</MenuItem>
              <MenuItem value="monthly">Monthly - Every month</MenuItem>
              <MenuItem value="cron">Cron - Custom expression</MenuItem>
            </Select>
          </FormControl>

          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Operation Type</InputLabel>
            <Select
              value={newSchedule.operation_type}
              onChange={(e) => setNewSchedule({ ...newSchedule, operation_type: e.target.value })}
              label="Operation Type"
            >
              <MenuItem value="bulk-publish">Bulk Publish</MenuItem>
              <MenuItem value="bulk-create">Bulk Create</MenuItem>
              <MenuItem value="bulk-update">Bulk Update</MenuItem>
              <MenuItem value="export">Export</MenuItem>
            </Select>
          </FormControl>

          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel>Timezone</InputLabel>
            <Select
              value={newSchedule.timezone}
              onChange={(e) => setNewSchedule({ ...newSchedule, timezone: e.target.value })}
              label="Timezone"
            >
              {SCHEDULER_CONFIG.TIMEZONES.map((tz) => (
                <MenuItem key={tz} value={tz}>{tz}</MenuItem>
              ))}
            </Select>
          </FormControl>

          {newSchedule.schedule_type === 'cron' && (
            <TextField
              label="Cron Expression"
              fullWidth
              value={newSchedule.cron_expression}
              onChange={(e) => setNewSchedule({ ...newSchedule, cron_expression: e.target.value })}
              placeholder="e.g., 0 2 * * *"
              helperText="Format: second minute hour day month dayOfWeek"
            />
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button
            onClick={handleCreateSchedule}
            variant="contained"
            disabled={!newSchedule.name || loading}
          >
            {loading ? <CircularProgress size={24} /> : 'Create Schedule'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
