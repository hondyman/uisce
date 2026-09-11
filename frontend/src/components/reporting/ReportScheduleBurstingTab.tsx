/**
 * ReportScheduleBurstingTab
 *
 * Phase 4 frontend: async 202 contract for report execution.
 *
 * Backend contract (Phase 3, commit 81ff74506f):
 *   POST /api/v1/reports/{id}/schedules/{sid}/run
 *     → 202 Accepted  { status: "pending", execution_id, workflow_id }
 *     → 503 Service Unavailable { execution_id, error }
 *   GET  /api/v1/reports/executions/{id}
 *     → 200 { status, output_url, rows_processed, execution_time_ms, error_message, ... }
 *
 * Live-backend coverage: backend/internal/api/report_handlers_test.go
 *   TriggerScheduleRun_-_Success_returns_202_with_pending_and_execution_id,
 *   TriggerScheduleRun_-_DispatchError_returns_503_with_execution_id_and_error,
 *   GetExecution_* (all variants)
 *
 * This file: UI layer only. Network contract verified by Go tests above.
 */

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Calendar,
  Split,
  Bell,
  ShieldCheck,
  Play,
  ExternalLink,
  AlertTriangle,
  X,
} from 'lucide-react';
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
  CircularProgress,
  Alert,
} from '@mui/material';
import { ReportExecutionStatusChip } from './ReportExecutionStatusChip';
import {
  triggerScheduleRun,
  pollExecution,
  type ExecutionRecord,
} from '../../api/reportExecutionApi';
import { apiFetch } from '../../lib/apiClient';

interface ReportScheduleBurstingTabProps {
  reportId?: string;
  reportName?: string;
  tenantId?: string;
  onScheduleSaved?: () => void;
}

type ExecutionPhase =
  | 'idle'
  | 'dispatched'
  | 'polling'
  | 'completed'
  | 'failed'
  | 'dispatch_failed';

interface ExecutionSnapshot {
  phase: ExecutionPhase;
  executionId?: string;
  workflowId?: string;
  record?: ExecutionRecord;
  errorMessage?: string;
  dispatchError?: string;
}

export const ReportScheduleBurstingTab: React.FC<ReportScheduleBurstingTabProps> = ({
  reportId,
  reportName,
  tenantId: _tenantId,
  onScheduleSaved,
}) => {
  const [scheduleName, setScheduleName] = useState(
    reportName ? `${reportName} Schedule` : 'Daily Valuation Schedule'
  );
  const [cronExpression, setCronExpression] = useState('0 8 * * 1-5');
  const [region, setRegion] = useState('us-west');
  const [calendarCode, setCalendarCode] = useState('NYSE');
  const [unscheduledBehavior, setUnscheduledBehavior] = useState('RUN_PREVIOUS_BUS_DAY');
  const [burstDimension, setBurstDimension] = useState('client_id');
  const [exportFormat, setExportFormat] = useState<'PDF' | 'EXCEL' | 'BOTH'>('PDF');
  const [notifyInApp, setNotifyInApp] = useState(true);
  const [notifyEmail, setNotifyEmail] = useState(false);

  const [saving, setSaving] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [selectedScheduleId, setSelectedScheduleId] = useState<string | null>(null);

  const pollAbortRef = useRef<AbortController | null>(null);
  const [execution, setExecution] = useState<ExecutionSnapshot>({ phase: 'idle' });

  useEffect(() => {
    return () => {
      pollAbortRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (reportName) {
      setScheduleName(`${reportName} Schedule`);
    }
  }, [reportName]);

  const loadSchedules = useCallback(async () => {
    try {
      const endpoint = reportId
        ? `/api/v1/reports/${reportId}/schedules`
        : '/api/reports/schedules';
      const res = await apiFetch(endpoint);
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0 && !selectedScheduleId) {
          setSelectedScheduleId(data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed to load schedules:', err);
    }
  }, [reportId, selectedScheduleId]);

  useEffect(() => {
    loadSchedules();
  }, [loadSchedules]);

  const handleSaveSchedule = async () => {
    setSaving(true);
    setStatusMessage(null);
    try {
      const endpoint = reportId
        ? `/api/v1/reports/${reportId}/schedules`
        : '/api/reports/schedules';
      const res = await apiFetch(endpoint, {
        method: 'POST',
        body: JSON.stringify({
          schedule_name: scheduleName,
          cron_expression: cronExpression,
          region,
          calendar_code: calendarCode,
          unscheduled_behavior: unscheduledBehavior,
          business_day_offset:
            unscheduledBehavior === 'RUN_PREVIOUS_BUS_DAY'
              ? -1
              : unscheduledBehavior === 'RUN_NEXT_BUS_DAY'
              ? 1
              : 0,
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
        if (onScheduleSaved) {
          onScheduleSaved();
        }
      } else {
        setStatusMessage('Failed to save schedule.');
      }
    } catch (err: unknown) {
      setStatusMessage(`Error saving schedule: ${(err as Error).message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleTriggerRun = async () => {
    if (!selectedScheduleId || !reportId) return;

    // Cancel any in-flight poll
    pollAbortRef.current?.abort();
    pollAbortRef.current = new AbortController();

    setExecution({ phase: 'dispatched' });
    setStatusMessage(null);

    const result = await triggerScheduleRun(reportId, selectedScheduleId);

    if (result.kind === 'dispatch_failed') {
      setExecution({
        phase: 'dispatch_failed',
        executionId: result.execution_id,
        dispatchError: result.error,
      });
      return;
    }

    if (result.kind === 'error') {
      setExecution({
        phase: 'failed',
        errorMessage: result.message,
      });
      return;
    }

    // 202 Accepted — render pending immediately from the response body
    setExecution({
      phase: 'polling',
      executionId: result.execution_id,
      workflowId: result.workflow_id,
      record: { status: 'pending' } as ExecutionRecord,
    });

    // Start polling; AbortController cancels on dialog unmount or re-trigger
    pollAbortRef.current.signal.addEventListener('abort', () => {
      setExecution((prev) =>
        prev.executionId === result.execution_id ? { phase: 'idle' } : prev
      );
    });

    try {
      const terminal = await pollExecution(
        result.execution_id,
        pollAbortRef.current.signal,
        (record) => {
          setExecution((prev) =>
            prev.executionId === result.execution_id
              ? { ...prev, phase: 'polling', record }
              : prev
          );
        }
      );
      const phase: ExecutionPhase =
        terminal.status === 'completed'
          ? 'completed'
          : terminal.status === 'failed'
          ? 'failed'
          : 'idle';
      setExecution({ phase, executionId: result.execution_id, workflowId: result.workflow_id, record: terminal });
    } catch (err) {
      if ((err as Error).name === 'AbortError' || (err as DOMException)?.name === 'AbortError') {
        setExecution({ phase: 'idle' });
      } else {
        setExecution({
          phase: 'failed',
          executionId: result.execution_id,
          errorMessage: (err as Error).message,
        });
      }
    }
  };

  const isRunning =
    execution.phase === 'dispatched' ||
    execution.phase === 'polling';

  const formatMs = (ms?: number) =>
    ms == null ? null : `${(ms / 1000).toFixed(1)}s`;

  return (
    <Box
      sx={{
        p: 3,
        display: 'flex',
        flexDirection: 'column',
        gap: 3,
        bgcolor: '#071526',
        color: '#E2E8F0',
        borderRadius: 2,
        border: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid rgba(255,255,255,0.08)',
          pb: 2,
        }}
      >
        <Box>
          <Typography
            variant="subtitle1"
            fontWeight="700"
            sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#F8FAFC' }}
          >
            <Calendar size={18} color="#F5A623" /> Batch Schedule &amp; Client Bursting
          </Typography>
          <Typography variant="caption" sx={{ color: '#94A3B8' }}>
            Auto-generate and burst individual client PDF/Excel packages synchronized with
            exchange calendars.
          </Typography>
        </Box>
        <Chip
          size="small"
          label="Tenant-Isolated Mesh"
          sx={{
            bgcolor: 'rgba(16, 185, 129, 0.12)',
            color: '#34D399',
            border: '1px solid rgba(16, 185, 129, 0.3)',
            fontWeight: 700,
            fontSize: '0.7rem',
          }}
        />
      </Box>

      {/* Status messages */}
      {statusMessage && (
        <Paper
          sx={{
            p: 1.5,
            bgcolor: 'rgba(99, 102, 241, 0.1)',
            border: '1px solid rgba(99, 102, 241, 0.3)',
            borderRadius: 1.5,
          }}
        >
          <Typography variant="caption" sx={{ color: '#A5B4FC', fontWeight: 600 }}>
            {statusMessage}
          </Typography>
        </Paper>
      )}

      {/* Execution result banner */}
      {execution.phase === 'dispatch_failed' && (
        <Alert
          severity="error"
          icon={<AlertTriangle size={16} />}
          action={
            execution.executionId ? (
              <Button
                size="small"
                href={`/executions/${execution.executionId}`}
                sx={{ color: '#F87171', textTransform: 'none', fontSize: '0.7rem' }}
              >
                View
              </Button>
            ) : undefined
          }
          sx={{ bgcolor: 'rgba(248,113,113,0.08)', border: '1px solid rgba(248,113,113,0.3)' }}
        >
          <Typography variant="caption" sx={{ fontWeight: 600 }}>
            Dispatch failed
          </Typography>
          <Typography variant="caption" display="block" sx={{ color: '#FCA5A5' }}>
            {execution.dispatchError}
            {execution.executionId && ` — Execution ID: ${execution.executionId}`}
          </Typography>
        </Alert>
      )}

      {execution.phase === 'failed' && (execution.errorMessage || execution.record?.error_message) && (
        <Alert
          severity="error"
          icon={<X size={16} />}
          sx={{ bgcolor: 'rgba(248,113,113,0.08)', border: '1px solid rgba(248,113,113,0.3)' }}
        >
          <Typography variant="caption" sx={{ fontWeight: 600 }}>
            Execution failed
          </Typography>
          <Typography variant="caption" display="block" sx={{ color: '#FCA5A5' }}>
            {execution.errorMessage || execution.record?.error_message}
          </Typography>
        </Alert>
      )}

      {/* Execution result card (terminal state) */}
      {(execution.phase === 'completed' || execution.phase === 'polling' || execution.phase === 'failed') &&
        execution.record && (
          <Paper
            sx={{
              p: 2,
              bgcolor: 'rgba(15, 23, 42, 0.6)',
              border: '1px solid rgba(255,255,255,0.06)',
              borderRadius: 2,
            }}
          >
            <Box
              sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}
            >
              <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#F8FAFC' }}>
                Execution Result
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {execution.executionId && (
                  <Typography
                    variant="caption"
                    sx={{ color: '#94A3B8', fontFamily: 'monospace', fontSize: '0.65rem' }}
                  >
                    {execution.executionId}
                  </Typography>
                )}
                <ReportExecutionStatusChip status={execution.record.status} />
              </Box>
            </Box>

            <Grid container spacing={2}>
              {execution.record.output_url && (
                <Grid size={12}>
                  <Button
                    size="small"
                    variant="text"
                    href={execution.record.output_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    endIcon={<ExternalLink size={12} />}
                    sx={{
                      color: '#60A5FA',
                      textTransform: 'none',
                      fontSize: '0.72rem',
                      p: 0,
                    }}
                  >
                    Download output
                  </Button>
                  {execution.record.output_size_bytes != null && (
                    <Typography
                      variant="caption"
                      sx={{ color: '#94A3B8', ml: 1, fontSize: '0.65rem' }}
                    >
                      ({(execution.record.output_size_bytes / 1024).toFixed(1)} KB)
                    </Typography>
                  )}
                </Grid>
              )}
              {execution.record.rows_processed != null && (
                <Grid size={6}>
                  <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>
                    Rows processed
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#E2E8F0', fontWeight: 600 }}>
                    {execution.record.rows_processed.toLocaleString()}
                  </Typography>
                </Grid>
              )}
              {execution.record.execution_time_ms != null && (
                <Grid size={6}>
                  <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block' }}>
                    Execution time
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#E2E8F0', fontWeight: 600 }}>
                    {formatMs(execution.record.execution_time_ms)}
                  </Typography>
                </Grid>
              )}
              {execution.record.error_message && (
                <Grid size={12}>
                  <Typography variant="caption" sx={{ color: '#F87171', display: 'block' }}>
                    {execution.record.error_message}
                  </Typography>
                </Grid>
              )}
            </Grid>
          </Paper>
        )}

      {/* Timing & Calendar Form */}
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            fullWidth
            size="small"
            label="Schedule Name"
            value={scheduleName}
            onChange={(e) => setScheduleName(e.target.value)}
            sx={{ '& .MuiInputBase-input': { color: '#FFF', fontSize: '0.8rem' }, '& label': { color: '#94A3B8' } }}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TextField
            fullWidth
            size="small"
            label="Cron Expression"
            value={cronExpression}
            onChange={(e) => setCronExpression(e.target.value)}
            helperText="e.g. 0 8 * * 1-5 (Mon-Fri 08:00 AM)"
            sx={{
              '& .MuiInputBase-input': { color: '#FFF', fontSize: '0.8rem', fontFamily: 'monospace' },
              '& label': { color: '#94A3B8' },
            }}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
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
        <Grid size={{ xs: 12, sm: 4 }}>
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
        <Grid size={{ xs: 12, sm: 4 }}>
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
      <Paper
        sx={{
          p: 2.5,
          bgcolor: 'rgba(15, 23, 42, 0.6)',
          border: '1px solid rgba(255,255,255,0.06)',
          borderRadius: 2,
        }}
      >
        <Typography
          variant="subtitle2"
          fontWeight="700"
          sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, color: '#C084FC' }}
        >
          <Split size={16} /> Client Partitioning &amp; File Export
        </Typography>
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, sm: 6 }}>
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
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography
              variant="caption"
              sx={{ color: '#94A3B8', display: 'block', mb: 0.5, fontWeight: 600 }}
            >
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
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          p: 2,
          bgcolor: 'rgba(15, 23, 42, 0.4)',
          borderRadius: 2,
          border: '1px solid rgba(255,255,255,0.06)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography
            variant="caption"
            sx={{ display: 'flex', alignItems: 'center', gap: 0.5, fontWeight: 600, color: '#FCD34D' }}
          >
            <Bell size={14} /> Notification Options:
          </Typography>
          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={notifyInApp}
                onChange={(e) => setNotifyInApp(e.target.checked)}
              />
            }
            label={
              <Typography variant="caption" sx={{ color: '#E2E8F0' }}>
                In-App Notification Bell
              </Typography>
            }
          />
          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={notifyEmail}
                onChange={(e) => setNotifyEmail(e.target.checked)}
              />
            }
            label={
              <Typography variant="caption" sx={{ color: '#E2E8F0' }}>
                Email Pre-Signed Download URLs
              </Typography>
            }
          />
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          {selectedScheduleId && (
            <Button
              variant="outlined"
              size="small"
              onClick={handleTriggerRun}
              disabled={isRunning}
              startIcon={
                isRunning ? (
                  <CircularProgress size={14} sx={{ color: '#60A5FA' }} />
                ) : (
                  <Play size={14} />
                )
              }
              sx={{
                borderColor: '#6366F1',
                color: '#A5B4FC',
                textTransform: 'none',
                fontSize: '0.75rem',
                fontWeight: 700,
              }}
            >
              {isRunning ? 'Running...' : 'Run Now'}
            </Button>
          )}
          <Button
            variant="contained"
            size="small"
            onClick={handleSaveSchedule}
            disabled={saving}
            startIcon={
              saving ? (
                <CircularProgress size={14} sx={{ color: '#0F172A' }} />
              ) : (
                <ShieldCheck size={16} />
              )
            }
            sx={{
              bgcolor: '#F5A623',
              color: '#0F172A',
              textTransform: 'none',
              fontWeight: 800,
              fontSize: '0.75rem',
              '&:hover': { bgcolor: '#D97706' },
            }}
          >
            {saving ? 'Saving...' : 'Save Active Schedule'}
          </Button>
        </Box>
      </Box>
    </Box>
  );
};

export default ReportScheduleBurstingTab;
