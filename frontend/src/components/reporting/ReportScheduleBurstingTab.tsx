import React, { useState, useEffect } from 'react';
import { Calendar, Clock, Globe, Split, Bell, ShieldCheck, Play } from 'lucide-react';
import {
  Box,
  Typography,
  Grid,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  Switch,
  FormControlLabel,
  Paper,
  Chip,
  Table,
  TableHead,
  TableRow,
  CircularProgress
} from '@mui/material';
import ReportBurstTelemetryHUD from './ReportBurstTelemetryHUD';

interface ReportScheduleBurstingTabProps {
  reportId?: string;
  tenantId?: string;
}

export const ReportScheduleBurstingTab: React.FC<ReportScheduleBurstingTabProps> = ({
  reportId,
  tenantId,
}) => {
  const [scheduleName, setScheduleName] = useState('Daily Institutional Client Valuation');
  const [cronExpression, setCronExpression] = useState('0 8 * * 1-5'); // Mon-Fri 08:00
  const [region, setRegion] = useState('us-west');
  const [calendarCode, setCalendarCode] = useState('NYSE');
  const [unscheduledBehavior, setUnscheduledBehavior] = useState('RUN_PREVIOUS_BUS_DAY');
  const [burstDimension, setBurstDimension] = useState('client_id');
  const [exportFormat, setExportFormat] = useState<'PDF' | 'EXCEL' | 'BOTH'>('PDF');
  const [notifyInApp, setNotifyInApp] = useState(true);
  const [notifyEmail, setNotifyEmail] = useState(true);

  const [saving, setSaving] = useState(false);
  const [triggering, setTriggering] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [schedulesList, setSchedulesList] = useState<any[]>([]);
  const [batchesList, setBatchesList] = useState<any[]>([]);
  const [selectedScheduleId, setSelectedScheduleId] = useState<string | null>(null);

  // Load existing schedules
  const loadSchedules = async () => {
    try {
      const res = await fetch('/api/reports/schedules');
      if (res.ok) {
        const data = await res.json();
        setSchedulesList(Array.isArray(data) ? data : []);
        if (data.length > 0 && !selectedScheduleId) {
          setSelectedScheduleId(data[0].id);
          loadBatches(data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed to load schedules:', err);
    }
  };

  const loadBatches = async (scheduleId: string) => {
    try {
      const res = await fetch(`/api/reports/schedules/${scheduleId}/batches`);
      if (res.ok) {
        const data = await res.json();
        setBatchesList(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error('Failed to load batches:', err);
    }
  };

  useEffect(() => {
    loadSchedules();
  }, []);

  const handleSaveSchedule = async () => {
    setSaving(true);
    setStatusMessage(null);
    try {
      const res = await fetch('/api/reports/schedules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          schedule_name: scheduleName,
          cron_expression: cronExpression,
          region,
          calendar_code: calendarCode,
          unscheduled_behavior: unscheduledBehavior,
          business_day_offset: unscheduledBehavior === 'RUN_PREVIOUS_BUS_DAY' ? -1 : unscheduledBehavior === 'RUN_NEXT_BUS_DAY' ? 1 : 0,
          burst_dimension: burstDimension,
          export_format: exportFormat,
          notify_in_app: notifyInApp,
          notify_email: notifyEmail,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        setStatusMessage('Schedule registered and activated successfully!');
        loadSchedules();
        if (data.id) {
          setSelectedScheduleId(data.id);
        }
      } else {
        setStatusMessage('Failed to save schedule.');
      }
    } catch (err: any) {
      setStatusMessage(`Error saving schedule: ${err.message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleTriggerRun = async () => {
    if (!selectedScheduleId) return;
    setTriggering(true);
    setStatusMessage(null);
    try {
      const res = await fetch(`/api/reports/schedules/${selectedScheduleId}/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (res.ok) {
        const data = await res.json();
        setStatusMessage(`Burst batch started successfully! Batch ID: ${data.batch_id || data.id}`);
        loadBatches(selectedScheduleId);
      } else {
        setStatusMessage('Failed to trigger burst batch.');
      }
    } catch (err: any) {
      setStatusMessage(`Error triggering batch: ${err.message}`);
    } finally {
      setTriggering(false);
    }
  };

  return (
    <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 3, bgcolor: '#071526', color: '#E2E8F0', borderRadius: 2, border: '1px solid rgba(255,255,255,0.08)' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.08)', pb: 2 }}>
        <Box>
          <Typography variant="subtitle1" fontWeight="700" sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#F8FAFC' }}>
            <Calendar size={18} color="#F5A623" /> Batch Schedule &amp; Client Bursting
          </Typography>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>
            Auto-generate and burst individual client PDF/Excel packages synchronized with exchange calendars.
          </Typography>
        </Box>
        <Chip
          size="small"
          label="Tenant-Isolated Mesh"
          sx={{ bgcolor: 'rgba(16, 185, 129, 0.12)', color: '#34D399', border: '1px solid rgba(16, 185, 129, 0.3)', fontWeight: 700, fontSize: '0.7rem' }}
        />
      </Box>

      {statusMessage && (
        <Paper sx={{ p: 1.5, bgcolor: 'rgba(99, 102, 241, 0.1)', border: '1px solid rgba(99, 102, 241, 0.3)', borderRadius: 1.5 }}>
          <Typography variant="caption" sx={{ color: '#A5B4FC', fontWeight: 600 }}>{statusMessage}</Typography>
        </Paper>
      )}

      {/* Timing & Calendar Form */}
      <Grid container spacing={2}>
        <Grid item xs={12} sm={6}>
          <TextField
            fullWidth
            size="small"
            label="Schedule Name"
            value={scheduleName}
            onChange={(e) => setScheduleName(e.target.value)}
            sx={{ '& .MuiInputBase-input': { color: '#FFF', fontSize: '0.8rem' }, '& label': { color: '#94A3B8' } }}
          />
        </Grid>
        <Grid item xs={12} sm={6}>
          <TextField
            fullWidth
            size="small"
            label="Cron Expression"
            value={cronExpression}
            onChange={(e) => setCronExpression(e.target.value)}
            helperText="e.g. 0 8 * * 1-5 (Mon-Fri 08:00 AM)"
            sx={{ '& .MuiInputBase-input': { color: '#FFF', fontSize: '0.8rem', fontFamily: 'monospace' }, '& label': { color: '#94A3B8' } }}
          />
        </Grid>
        <Grid item xs={12} sm={4}>
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: '#94A3B8' }}>Execution Region</InputLabel>
            <Select
              value={region}
              label="Execution Region"
              onChange={(e) => setRegion(e.target.value)}
              sx={{ color: '#FFF', '& .MuiSvgIcon-root': { color: '#FFF' } }}
            >
              <MenuItem value="us-west">US West (Oregon)</MenuItem>
              <MenuItem value="us-east">US East (N. Virginia)</MenuItem>
              <MenuItem value="eu-west">EU West (Ireland)</MenuItem>
            </Select>
          </FormControl>
        </Grid>
        <Grid item xs={12} sm={4}>
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: '#94A3B8' }}>Exchange Master Calendar</InputLabel>
            <Select
              value={calendarCode}
              label="Exchange Master Calendar"
              onChange={(e) => setCalendarCode(e.target.value)}
              sx={{ color: '#FFF', '& .MuiSvgIcon-root': { color: '#FFF' } }}
            >
              <MenuItem value="NYSE">NYSE (New York Stock Exchange)</MenuItem>
              <MenuItem value="LSE">LSE (London Stock Exchange)</MenuItem>
              <MenuItem value="TARGET2">TARGET2 (European Central Bank)</MenuItem>
            </Select>
          </FormControl>
        </Grid>
        <Grid item xs={12} sm={4}>
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: '#94A3B8' }}>Holiday / Non-Trading Action</InputLabel>
            <Select
              value={unscheduledBehavior}
              label="Holiday / Non-Trading Action"
              onChange={(e) => setUnscheduledBehavior(e.target.value)}
              sx={{ color: '#FFF', '& .MuiSvgIcon-root': { color: '#FFF' } }}
            >
              <MenuItem value="RUN_PREVIOUS_BUS_DAY">Run Previous Business Day (T-1)</MenuItem>
              <MenuItem value="RUN_NEXT_BUS_DAY">Run Next Business Day (T+1)</MenuItem>
              <MenuItem value="SKIP">Skip Execution</MenuItem>
              <MenuItem value="WARN_HALT">Halt &amp; Alert Compliance</MenuItem>
            </Select>
          </FormControl>
        </Grid>
      </Grid>

      {/* Bursting & Slicing Dimension */}
      <Paper sx={{ p: 2.5, bgcolor: 'rgba(15, 23, 42, 0.6)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 2 }}>
        <Typography variant="subtitle2" fontWeight="700" sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, color: '#C084FC' }}>
          <Split size={16} /> Client Partitioning &amp; File Export
        </Typography>
        <Grid container spacing={2}>
          <Grid item xs={12} sm={6}>
            <FormControl fullWidth size="small">
              <InputLabel sx={{ color: '#94A3B8' }}>Bursting Slicing Field</InputLabel>
              <Select
                value={burstDimension}
                label="Bursting Slicing Field"
                onChange={(e) => setBurstDimension(e.target.value)}
                sx={{ color: '#FFF', '& .MuiSvgIcon-root': { color: '#FFF' } }}
              >
                <MenuItem value="client_id">Client Identifier (client_id)</MenuItem>
                <MenuItem value="account_id">Custodial Account Code (account_id)</MenuItem>
                <MenuItem value="portfolio_id">Portfolio Identifier (portfolio_id)</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 0.5, fontWeight: 600 }}>
              Export File Format
            </Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              {(['PDF', 'EXCEL', 'BOTH'] as const).map((fmt) => (
                <Button
                  key={fmt}
                  variant={exportFormat === fmt ? 'contained' : 'outlined'}
                  size="small"
                  onClick={() => setExportFormat(fmt)}
                  sx={{
                    flex: 1,
                    textTransform: 'none',
                    fontSize: '0.75rem',
                    fontWeight: 700,
                    bgcolor: exportFormat === fmt ? '#F5A623' : 'transparent',
                    color: exportFormat === fmt ? '#0F172A' : '#94A3B8',
                    borderColor: 'rgba(255,255,255,0.15)',
                    '&:hover': {
                      bgcolor: exportFormat === fmt ? '#D97706' : 'rgba(255,255,255,0.05)',
                    },
                  }}
                >
                  {fmt}
                </Button>
              ))}
            </Box>
          </Grid>
        </Grid>
      </Paper>

      {/* Notifications & Actions */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', p: 2, bgcolor: 'rgba(15, 23, 42, 0.4)', borderRadius: 2, border: '1px solid rgba(255,255,255,0.06)' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography variant="caption" sx={{ display: 'flex', alignItems: 'center', gap: 0.5, fontWeight: 600, color: '#FCD34D' }}>
            <Bell size={14} /> Notification Options:
          </Typography>
          <FormControlLabel
            control={<Switch size="small" checked={notifyInApp} onChange={(e) => setNotifyInApp(e.target.checked)} />}
            label={<Typography variant="caption" sx={{ color: '#E2E8F0' }}>In-App Notification Bell</Typography>}
          />
          <FormControlLabel
            control={<Switch size="small" checked={notifyEmail} onChange={(e) => setNotifyEmail(e.target.checked)} />}
            label={<Typography variant="caption" sx={{ color: '#E2E8F0' }}>Email Pre-Signed Download URLs</Typography>}
          />
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          {selectedScheduleId && (
            <Button
              variant="outlined"
              size="small"
              onClick={handleTriggerRun}
              disabled={triggering}
              startIcon={triggering ? <CircularProgress size={14} /> : <Play size={14} />}
              sx={{ borderColor: '#6366F1', color: '#A5B4FC', textTransform: 'none', fontSize: '0.75rem', fontWeight: 700 }}
            >
              {triggering ? 'Bursting...' : 'Run Burst Now'}
            </Button>
          )}
          <Button
            variant="contained"
            size="small"
            onClick={handleSaveSchedule}
            disabled={saving}
            startIcon={saving ? <CircularProgress size={14} /> : <ShieldCheck size={16} />}
            sx={{ bgcolor: '#F5A623', color: '#0F172A', textTransform: 'none', fontWeight: 800, fontSize: '0.75rem', '&:hover': { bgcolor: '#D97706' } }}
          >
            {saving ? 'Saving...' : 'Save Active Schedule'}
          </Button>
        </Box>
      </Box>

      {/* Live Telemetry HUD */}
      {selectedScheduleId && batchesList.length > 0 && (
        <ReportBurstTelemetryHUD batchId={batchesList[0].id} tenantId={tenantId} />
      )}

      {/* Execution Batches Table */}
      {batchesList.length > 0 && (
        <Paper sx={{ p: 2, bgcolor: 'rgba(15, 23, 42, 0.6)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 2 }}>
          <Typography variant="subtitle2" fontWeight="700" sx={{ mb: 1.5, color: '#F8FAFC' }}>
            Recent Burst Batches &amp; Artifact Ledger
          </Typography>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Batch ID</TableCell>
                <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Effective Date</TableCell>
                <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Total Slices</TableCell>
                <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Rendered</TableCell>
                <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Status</TableCell>
                <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Timestamp</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {batchesList.map((batch) => (
                <TableRow key={batch.id}>
                  <TableCell sx={{ color: '#E2E8F0', fontSize: '0.7rem', fontFamily: 'monospace' }}>{batch.id.slice(0, 8)}...</TableCell>
                  <TableCell sx={{ color: '#E2E8F0', fontSize: '0.7rem' }}>{batch.effective_date}</TableCell>
                  <TableCell sx={{ color: '#E2E8F0', fontSize: '0.7rem' }}>{batch.total_clients}</TableCell>
                  <TableCell sx={{ color: '#34D399', fontSize: '0.7rem' }}>{batch.successful_renders} / {batch.total_clients}</TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      label={batch.status}
                      color={batch.status === 'COMPLETED' ? 'success' : batch.status === 'PARTIAL' ? 'warning' : 'info'}
                      sx={{ height: 18, fontSize: '0.62rem', fontWeight: 700 }}
                    />
                  </TableCell>
                  <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem' }}>{new Date(batch.started_at).toLocaleTimeString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}
    </Box>
  );
};

export default ReportScheduleBurstingTab;
