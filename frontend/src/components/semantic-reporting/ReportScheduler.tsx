/**
 * ReportScheduler Component
 *
 * Manages report scheduling with cron expressions, delivery options,
 * and parameter presets. Supports email delivery, file export, and
 * webhook notifications.
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  IconButton,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormControlLabel,
  Switch,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Card,
  CardContent,
  CardActions,
  Grid,
  Divider,
  Alert,
  Tooltip,
  List,
  ListItem,
  ListItemText,
  CircularProgress,
  Autocomplete,
  InputAdornment,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import {
  Add,
  Edit,
  Delete,
  Schedule,
  PlayArrow,
  Pause,
  Email,
  CloudUpload,
  Webhook,
  History,
  CheckCircle,
  Refresh,
  ContentCopy,
} from '@mui/icons-material';
import { ReportSchedule, ReportDefinition, DeliveryChannel } from '../../api/semanticReporting';
import {
  useReportSchedules,
  useReportSchedule,
  useCreateReportSchedule,
  useUpdateReportSchedule,
  useDeleteReportSchedule,
  useReportDefinitions,
} from '../../hooks/useSemanticReporting';
import { devLog } from '../../utils/logger';

// ============================================================================
// Types
// ============================================================================

interface ReportSchedulerProps {
  reportId?: string;
  onScheduleCreated?: (schedule: ReportSchedule) => void;
}

interface ScheduleFormData {
  reportDefinitionId: string;
  scheduleName: string;
  cronExpression: string;
  timezone: string;
  outputFormats: string[];
  parametersTemplate: Record<string, unknown>;
  deliveryChannels: DeliveryChannel[];
  isActive: boolean;
}

interface CronPreset {
  label: string;
  expression: string;
  description: string;
}

// ============================================================================
// Constants
// ============================================================================

const CRON_PRESETS: CronPreset[] = [
  { label: 'Every hour', expression: '0 * * * *', description: 'At minute 0 of every hour' },
  { label: 'Daily at 8am', expression: '0 8 * * *', description: 'Every day at 8:00 AM' },
  { label: 'Daily at midnight', expression: '0 0 * * *', description: 'Every day at midnight' },
  { label: 'Weekly on Monday', expression: '0 8 * * 1', description: 'Every Monday at 8:00 AM' },
  { label: 'Monthly 1st', expression: '0 8 1 * *', description: '1st of every month at 8:00 AM' },
  { label: 'Quarterly', expression: '0 8 1 1,4,7,10 *', description: 'Jan, Apr, Jul, Oct 1st at 8:00 AM' },
  { label: 'Every 15 min', expression: '*/15 * * * *', description: 'Every 15 minutes' },
  { label: 'Business hours', expression: '0 9-17 * * 1-5', description: 'Every hour 9-5 Mon-Fri' },
];

const TIMEZONES = [
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'America/Phoenix',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Asia/Singapore',
  'Australia/Sydney',
  'Pacific/Auckland',
  'UTC',
];

const OUTPUT_FORMATS = [
  { value: 'pdf', label: 'PDF Document' },
  { value: 'xlsx', label: 'Excel Spreadsheet' },
  { value: 'csv', label: 'CSV File' },
  { value: 'html', label: 'HTML Report' },
  { value: 'json', label: 'JSON Data' },
];

const DEFAULT_FORM_DATA: ScheduleFormData = {
  reportDefinitionId: '',
  scheduleName: '',
  cronExpression: '0 8 * * *',
  timezone: 'America/New_York',
  outputFormats: ['pdf'],
  parametersTemplate: {},
  deliveryChannels: [],
  isActive: true,
};

// ============================================================================
// Helper Functions
// ============================================================================

function parseCronExpression(cron: string): string {
  const preset = CRON_PRESETS.find((p) => p.expression === cron);
  if (preset) return preset.description;

  const parts = cron.split(' ');
  if (parts.length !== 5) return 'Invalid expression';

  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts;
  const descriptions: string[] = [];

  if (minute === '*') descriptions.push('every minute');
  else if (minute.startsWith('*/')) descriptions.push(`every ${minute.slice(2)} minutes`);
  else descriptions.push(`at minute ${minute}`);

  if (hour === '*') descriptions.push('of every hour');
  else if (hour.startsWith('*/')) descriptions.push(`every ${hour.slice(2)} hours`);
  else descriptions.push(`at ${hour}:00`);

  if (dayOfMonth !== '*') descriptions.push(`on day ${dayOfMonth}`);
  if (month !== '*') descriptions.push(`in month ${month}`);
  if (dayOfWeek !== '*') {
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    descriptions.push(`on ${days[parseInt(dayOfWeek)] || dayOfWeek}`);
  }

  return descriptions.join(' ');
}

function getNextRunTime(cronExpression: string, _timezone: string): Date | null {
  // Simple next run calculation - in production, use a proper cron parser
  try {
    const now = new Date();
    const parts = cronExpression.split(' ');
    if (parts.length !== 5) return null;

    const [minute, hour] = parts;
    const next = new Date(now);

    if (minute !== '*' && !minute.startsWith('*/')) {
      next.setMinutes(parseInt(minute));
    }
    if (hour !== '*' && !hour.startsWith('*/')) {
      next.setHours(parseInt(hour));
    }

    if (next <= now) {
      next.setDate(next.getDate() + 1);
    }

    return next;
  } catch {
    return null;
  }
}

// ============================================================================
// Sub-Components
// ============================================================================

interface CronBuilderProps {
  value: string;
  onChange: (value: string) => void;
}

const CronBuilder: React.FC<CronBuilderProps> = ({ value, onChange }) => {
  const [activeTab, setActiveTab] = useState(0);
  const [customCron, setCustomCron] = useState(value);

  return (
    <Box>
      <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ mb: 2 }}>
        <Tab label="Presets" />
        <Tab label="Custom" />
      </Tabs>

      {activeTab === 0 && (
        <Grid container spacing={1}>
          {CRON_PRESETS.map((preset) => (
            <Grid item xs={12} sm={6} md={4} key={preset.expression}>
              <Card
                variant={value === preset.expression ? 'elevation' : 'outlined'}
                sx={{
                  cursor: 'pointer',
                  bgcolor: value === preset.expression ? 'primary.light' : 'background.paper',
                  '&:hover': { bgcolor: 'action.hover' },
                }}
                onClick={() => onChange(preset.expression)}
              >
                <CardContent sx={{ py: 1, '&:last-child': { pb: 1 } }}>
                  <Typography variant="subtitle2">{preset.label}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {preset.description}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {activeTab === 1 && (
        <Box>
          <TextField
            fullWidth
            label="Cron Expression"
            value={customCron}
            onChange={(e) => setCustomCron(e.target.value)}
            placeholder="* * * * *"
            helperText="Format: minute hour day-of-month month day-of-week"
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <Button size="small" onClick={() => onChange(customCron)}>
                    Apply
                  </Button>
                </InputAdornment>
              ),
            }}
          />
          <Box sx={{ mt: 2 }}>
            <Typography variant="caption" color="text.secondary">
              Preview: {parseCronExpression(customCron)}
            </Typography>
          </Box>
        </Box>
      )}
    </Box>
  );
};

interface DeliveryChannelEditorProps {
  channels: DeliveryChannel[];
  onChange: (channels: DeliveryChannel[]) => void;
}

const DeliveryChannelEditor: React.FC<DeliveryChannelEditorProps> = ({ channels, onChange }) => {
  const [emailInput, setEmailInput] = useState('');
  const [channelType, setChannelType] = useState<DeliveryChannel['type']>('email');

  const handleAddChannel = () => {
    const newChannel: DeliveryChannel = {
      type: channelType,
      config: {},
    };

    switch (channelType) {
      case 'email':
        newChannel.config = {
          recipients: [],
          subject: '',
          body: '',
          attachReport: true,
        };
        break;
      case 'webhook':
        newChannel.config = {
          url: '',
          method: 'POST',
          headers: {},
          includeReport: true,
        };
        break;
      case 'file_share':
        newChannel.config = {
          path: '/reports',
          filename: '{{report_name}}_{{date}}.{{format}}',
          overwrite: false,
        };
        break;
    }

    onChange([...channels, newChannel]);
  };

  const handleRemoveChannel = (index: number) => {
    const updated = [...channels];
    updated.splice(index, 1);
    onChange(updated);
  };

  const handleUpdateChannel = (index: number, config: Record<string, unknown>) => {
    const updated = [...channels];
    updated[index] = { ...updated[index], config };
    onChange(updated);
  };

  const handleAddEmail = (channelIndex: number) => {
    if (emailInput) {
      const channel = channels[channelIndex];
      const recipients = [...((channel.config.recipients as string[]) || []), emailInput];
      handleUpdateChannel(channelIndex, { ...channel.config, recipients });
      setEmailInput('');
    }
  };

  const handleRemoveEmail = (channelIndex: number, email: string) => {
    const channel = channels[channelIndex];
    const recipients = ((channel.config.recipients as string[]) || []).filter((e) => e !== email);
    handleUpdateChannel(channelIndex, { ...channel.config, recipients });
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>Channel Type</InputLabel>
          <Select
            value={channelType}
            label="Channel Type"
            onChange={(e) => setChannelType(e.target.value as DeliveryChannel['type'])}
          >
            <MenuItem value="email">
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Email fontSize="small" /> Email
              </Box>
            </MenuItem>
            <MenuItem value="webhook">
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Webhook fontSize="small" /> Webhook
              </Box>
            </MenuItem>
            <MenuItem value="file_share">
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <CloudUpload fontSize="small" /> File Export
              </Box>
            </MenuItem>
          </Select>
        </FormControl>
        <Button variant="outlined" startIcon={<Add />} onClick={handleAddChannel}>
          Add Channel
        </Button>
      </Box>

      {channels.length === 0 && (
        <Alert severity="info" sx={{ mb: 2 }}>
          No delivery channels configured. Reports will be generated but not delivered.
        </Alert>
      )}

      {channels.map((channel, index) => (
        <Paper key={index} variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="subtitle2">
              {channel.type === 'email' && <><Email fontSize="small" sx={{ mr: 1, verticalAlign: 'middle' }} /> Email Delivery</>}
              {channel.type === 'webhook' && <><Webhook fontSize="small" sx={{ mr: 1, verticalAlign: 'middle' }} /> Webhook</>}
              {channel.type === 'file_share' && <><CloudUpload fontSize="small" sx={{ mr: 1, verticalAlign: 'middle' }} /> File Export</>}
              {channel.type === 'slack' && 'Slack'}
              {channel.type === 'teams' && 'Microsoft Teams'}
            </Typography>
            <IconButton size="small" color="error" onClick={() => handleRemoveChannel(index)}>
              <Delete fontSize="small" />
            </IconButton>
          </Box>

          {channel.type === 'email' && (
            <Box>
              <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                <TextField
                  fullWidth
                  size="small"
                  label="Add recipient"
                  type="email"
                  value={emailInput}
                  onChange={(e) => setEmailInput(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && handleAddEmail(index)}
                />
                <Button variant="outlined" onClick={() => handleAddEmail(index)}>
                  Add
                </Button>
              </Box>

              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 2 }}>
                {((channel.config.recipients as string[]) || []).map((email) => (
                  <Chip
                    key={email}
                    label={email}
                    size="small"
                    onDelete={() => handleRemoveEmail(index, email)}
                  />
                ))}
              </Box>

              <TextField
                fullWidth
                label="Subject"
                value={channel.config.subject || ''}
                onChange={(e) =>
                  handleUpdateChannel(index, { ...channel.config, subject: e.target.value })
                }
                placeholder="Report: {{report_name}} - {{date}}"
                sx={{ mb: 2 }}
              />

              <TextField
                fullWidth
                multiline
                rows={3}
                label="Email Body"
                value={channel.config.body || ''}
                onChange={(e) =>
                  handleUpdateChannel(index, { ...channel.config, body: e.target.value })
                }
                placeholder="Please find attached the scheduled report..."
                sx={{ mb: 2 }}
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={(channel.config.attachReport as boolean) ?? true}
                    onChange={(e) =>
                      handleUpdateChannel(index, { ...channel.config, attachReport: e.target.checked })
                    }
                  />
                }
                label="Attach report to email"
              />
            </Box>
          )}

          {channel.type === 'webhook' && (
            <Box>
              <TextField
                fullWidth
                label="Webhook URL"
                value={channel.config.url || ''}
                onChange={(e) =>
                  handleUpdateChannel(index, { ...channel.config, url: e.target.value })
                }
                placeholder="https://api.example.com/reports/webhook"
                sx={{ mb: 2 }}
              />

              <FormControl fullWidth sx={{ mb: 2 }}>
                <InputLabel>HTTP Method</InputLabel>
                <Select
                  value={channel.config.method || 'POST'}
                  label="HTTP Method"
                  onChange={(e) =>
                    handleUpdateChannel(index, { ...channel.config, method: e.target.value })
                  }
                >
                  <MenuItem value="POST">POST</MenuItem>
                  <MenuItem value="PUT">PUT</MenuItem>
                </Select>
              </FormControl>

              <FormControlLabel
                control={
                  <Switch
                    checked={(channel.config.includeReport as boolean) ?? true}
                    onChange={(e) =>
                      handleUpdateChannel(index, { ...channel.config, includeReport: e.target.checked })
                    }
                  />
                }
                label="Include report data in payload"
              />
            </Box>
          )}

          {channel.type === 'file_share' && (
            <Box>
              <TextField
                fullWidth
                label="Export Path"
                value={channel.config.path || ''}
                onChange={(e) =>
                  handleUpdateChannel(index, { ...channel.config, path: e.target.value })
                }
                sx={{ mb: 2 }}
              />

              <TextField
                fullWidth
                label="Filename Pattern"
                value={channel.config.filename || ''}
                onChange={(e) =>
                  handleUpdateChannel(index, { ...channel.config, filename: e.target.value })
                }
                helperText="Variables: {{report_name}}, {{date}}, {{time}}, {{format}}"
                sx={{ mb: 2 }}
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={(channel.config.overwrite as boolean) ?? false}
                    onChange={(e) =>
                      handleUpdateChannel(index, { ...channel.config, overwrite: e.target.checked })
                    }
                  />
                }
                label="Overwrite existing files"
              />
            </Box>
          )}
        </Paper>
      ))}
    </Box>
  );
};

// ============================================================================
// Main Component
// ============================================================================

const ReportScheduler: React.FC<ReportSchedulerProps> = ({ reportId, onScheduleCreated }) => {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<ReportSchedule | null>(null);
  const [formData, setFormData] = useState<ScheduleFormData>(DEFAULT_FORM_DATA);
  const [historyDialogOpen, setHistoryDialogOpen] = useState(false);
  const [selectedScheduleId, setSelectedScheduleId] = useState<string | null>(null);

  // Queries
  const { data: schedules, isLoading, refetch } = useReportSchedules();
  const { data: definitions } = useReportDefinitions();
  const { data: selectedSchedule } = useReportSchedule(selectedScheduleId || undefined);

  // Mutations
  const createSchedule = useCreateReportSchedule();
  const updateSchedule = useUpdateReportSchedule();
  const deleteSchedule = useDeleteReportSchedule();

  // Filter schedules by reportId if provided
  const filteredSchedules = reportId
    ? schedules?.filter((s: ReportSchedule) => s.report_definition_id === reportId)
    : schedules;

  // Initialize form with reportId if provided
  useEffect(() => {
    if (reportId && !formData.reportDefinitionId) {
      setFormData((prev) => ({ ...prev, reportDefinitionId: reportId }));
    }
  }, [reportId, formData.reportDefinitionId]);

  const handleOpenDialog = useCallback((schedule?: ReportSchedule) => {
    if (schedule) {
      setEditingSchedule(schedule);
      setFormData({
        reportDefinitionId: schedule.report_definition_id,
        scheduleName: schedule.schedule_name,
        cronExpression: schedule.cron_expression,
        timezone: schedule.timezone,
        outputFormats: schedule.output_formats || ['pdf'],
        parametersTemplate: schedule.parameters_template || {},
        deliveryChannels: schedule.delivery_channels || [],
        isActive: schedule.is_active,
      });
    } else {
      setEditingSchedule(null);
      setFormData({ ...DEFAULT_FORM_DATA, reportDefinitionId: reportId || '' });
    }
    setDialogOpen(true);
  }, [reportId]);

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingSchedule(null);
    setFormData(DEFAULT_FORM_DATA);
  };

  const handleSave = async () => {
    try {
      const scheduleData = {
        report_definition_id: formData.reportDefinitionId,
        schedule_name: formData.scheduleName,
        cron_expression: formData.cronExpression,
        timezone: formData.timezone,
        output_formats: formData.outputFormats,
        parameters_template: formData.parametersTemplate,
        delivery_channels: formData.deliveryChannels,
        is_active: formData.isActive,
      };

      if (editingSchedule) {
        await updateSchedule.mutateAsync({
          id: editingSchedule.id,
          updates: scheduleData,
        });
      } else {
        const created = await createSchedule.mutateAsync(scheduleData);
        onScheduleCreated?.(created);
      }

      handleCloseDialog();
      refetch();
    } catch (error) {
      devLog('Failed to save schedule:', error);
    }
  };

  const handleDelete = async (scheduleId: string) => {
    if (window.confirm('Are you sure you want to delete this schedule?')) {
      try {
        await deleteSchedule.mutateAsync(scheduleId);
        refetch();
      } catch (error) {
        devLog('Failed to delete schedule:', error);
      }
    }
  };

  const handleToggleActive = async (schedule: ReportSchedule) => {
    try {
      await updateSchedule.mutateAsync({
        id: schedule.id,
        updates: { is_active: !schedule.is_active },
      });
      refetch();
    } catch (error) {
      devLog('Failed to toggle schedule:', error);
    }
  };

  const handleViewHistory = (scheduleId: string) => {
    setSelectedScheduleId(scheduleId);
    setHistoryDialogOpen(true);
  };

  const getReportName = (definitionId: string): string => {
    const def = definitions?.find((d: ReportDefinition) => d.id === definitionId);
    return def?.display_name || 'Unknown Report';
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h5" component="h1">
          <Schedule sx={{ mr: 1, verticalAlign: 'middle' }} />
          Report Schedules
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button startIcon={<Refresh />} onClick={() => refetch()}>
            Refresh
          </Button>
          <Button variant="contained" startIcon={<Add />} onClick={() => handleOpenDialog()}>
            New Schedule
          </Button>
        </Box>
      </Box>

      {/* Schedule List */}
      {!filteredSchedules?.length ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Schedule sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" gutterBottom>
            No Schedules Configured
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 2 }}>
            Create a schedule to automatically run reports on a recurring basis.
          </Typography>
          <Button variant="contained" startIcon={<Add />} onClick={() => handleOpenDialog()}>
            Create First Schedule
          </Button>
        </Paper>
      ) : (
        <Grid container spacing={2}>
          {filteredSchedules.map((schedule: ReportSchedule) => {
            const nextRun = schedule.next_run_at
              ? new Date(schedule.next_run_at)
              : getNextRunTime(schedule.cron_expression, schedule.timezone);

            return (
              <Grid item xs={12} md={6} lg={4} key={schedule.id}>
                <Card>
                  <CardContent>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <Box>
                        <Typography variant="h6" noWrap>
                          {schedule.schedule_name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" noWrap>
                          {getReportName(schedule.report_definition_id)}
                        </Typography>
                      </Box>
                      <Chip
                        size="small"
                        label={schedule.is_active ? 'Active' : 'Paused'}
                        color={schedule.is_active ? 'success' : 'default'}
                        icon={schedule.is_active ? <CheckCircle /> : <Pause />}
                      />
                    </Box>

                    <Divider sx={{ my: 2 }} />

                    <List dense disablePadding>
                      <ListItem disableGutters>
                        <ListItemText
                          primary="Schedule"
                          secondary={parseCronExpression(schedule.cron_expression)}
                        />
                      </ListItem>
                      <ListItem disableGutters>
                        <ListItemText
                          primary="Next Run"
                          secondary={nextRun ? nextRun.toLocaleString() : 'Not scheduled'}
                        />
                      </ListItem>
                      <ListItem disableGutters>
                        <ListItemText
                          primary="Last Run"
                          secondary={
                            schedule.last_run_at
                              ? new Date(schedule.last_run_at).toLocaleString()
                              : 'Never'
                          }
                        />
                      </ListItem>
                      <ListItem disableGutters>
                        <ListItemText
                          primary="Formats"
                          secondary={schedule.output_formats.map((f) => f.toUpperCase()).join(', ')}
                        />
                      </ListItem>
                    </List>
                  </CardContent>

                  <CardActions sx={{ justifyContent: 'space-between' }}>
                    <Box>
                      <Tooltip title={schedule.is_active ? 'Pause' : 'Activate'}>
                        <IconButton size="small" onClick={() => handleToggleActive(schedule)}>
                          {schedule.is_active ? <Pause /> : <PlayArrow />}
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="View History">
                        <IconButton size="small" onClick={() => handleViewHistory(schedule.id)}>
                          <History />
                        </IconButton>
                      </Tooltip>
                    </Box>
                    <Box>
                      <Tooltip title="Edit">
                        <IconButton size="small" onClick={() => handleOpenDialog(schedule)}>
                          <Edit />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete">
                        <IconButton size="small" color="error" onClick={() => handleDelete(schedule.id)}>
                          <Delete />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  </CardActions>
                </Card>
              </Grid>
            );
          })}
        </Grid>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>{editingSchedule ? 'Edit Schedule' : 'Create New Schedule'}</DialogTitle>
        <DialogContent dividers>
          <Grid container spacing={3}>
            {/* Basic Info */}
            <Grid item xs={12}>
              <Typography variant="subtitle2" gutterBottom>
                Basic Information
              </Typography>
            </Grid>

            {!reportId && (
              <Grid item xs={12}>
                <Autocomplete
                  options={definitions || []}
                  getOptionLabel={(option: ReportDefinition) => option.display_name}
                  value={definitions?.find((d: ReportDefinition) => d.id === formData.reportDefinitionId) || null}
                  onChange={(_, value) =>
                    setFormData((prev) => ({
                      ...prev,
                      reportDefinitionId: value?.id || '',
                    }))
                  }
                  renderInput={(params) => (
                    <TextField {...params} label="Report" required />
                  )}
                />
              </Grid>
            )}

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Schedule Name"
                value={formData.scheduleName}
                onChange={(e) => setFormData((prev) => ({ ...prev, scheduleName: e.target.value }))}
                required
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Output Formats</InputLabel>
                <Select
                  multiple
                  value={formData.outputFormats}
                  label="Output Formats"
                  onChange={(e) => setFormData((prev) => ({
                    ...prev,
                    outputFormats: e.target.value as string[],
                  }))}
                  renderValue={(selected) => (
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                      {(selected as string[]).map((value) => (
                        <Chip key={value} label={value.toUpperCase()} size="small" />
                      ))}
                    </Box>
                  )}
                >
                  {OUTPUT_FORMATS.map((format) => (
                    <MenuItem key={format.value} value={format.value}>
                      {format.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            {/* Schedule */}
            <Grid item xs={12}>
              <Divider sx={{ my: 2 }} />
              <Typography variant="subtitle2" gutterBottom>
                Schedule Configuration
              </Typography>
            </Grid>

            <Grid item xs={12}>
              <CronBuilder
                value={formData.cronExpression}
                onChange={(value) => setFormData((prev) => ({ ...prev, cronExpression: value }))}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Timezone</InputLabel>
                <Select
                  value={formData.timezone}
                  label="Timezone"
                  onChange={(e) => setFormData((prev) => ({ ...prev, timezone: e.target.value }))}
                >
                  {TIMEZONES.map((tz) => (
                    <MenuItem key={tz} value={tz}>
                      {tz}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} md={6}>
              <Alert severity="info" sx={{ height: '100%' }}>
                <Typography variant="body2">
                  Next run: {getNextRunTime(formData.cronExpression, formData.timezone)?.toLocaleString() || 'N/A'}
                </Typography>
              </Alert>
            </Grid>

            {/* Delivery Channels */}
            <Grid item xs={12}>
              <Divider sx={{ my: 2 }} />
              <Typography variant="subtitle2" gutterBottom>
                Delivery Options
              </Typography>
            </Grid>

            <Grid item xs={12}>
              <DeliveryChannelEditor
                channels={formData.deliveryChannels}
                onChange={(channels) => setFormData((prev) => ({ ...prev, deliveryChannels: channels }))}
              />
            </Grid>

            {/* Enable/Disable */}
            <Grid item xs={12}>
              <Divider sx={{ my: 2 }} />
              <FormControlLabel
                control={
                  <Switch
                    checked={formData.isActive}
                    onChange={(e) => setFormData((prev) => ({ ...prev, isActive: e.target.checked }))}
                  />
                }
                label="Enable this schedule"
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={!formData.scheduleName || !formData.reportDefinitionId || createSchedule.isPending || updateSchedule.isPending}
          >
            {editingSchedule ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* History Dialog */}
      <Dialog open={historyDialogOpen} onClose={() => setHistoryDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>
          Schedule History
          {selectedSchedule && ` - ${selectedSchedule.schedule_name}`}
        </DialogTitle>
        <DialogContent dividers>
          {selectedSchedule?.last_run_at ? (
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Run Time</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Next Run</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  <TableRow>
                    <TableCell>{new Date(selectedSchedule.last_run_at).toLocaleString()}</TableCell>
                    <TableCell>
                      <Chip size="small" label="Completed" color="success" />
                    </TableCell>
                    <TableCell>
                      {selectedSchedule.next_run_at
                        ? new Date(selectedSchedule.next_run_at).toLocaleString()
                        : 'N/A'}
                    </TableCell>
                    <TableCell>
                      <Tooltip title="Copy ID">
                        <IconButton
                          size="small"
                          onClick={() => navigator.clipboard.writeText(selectedSchedule.id)}
                        >
                          <ContentCopy fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TableContainer>
          ) : (
            <Box sx={{ textAlign: 'center', py: 4 }}>
              <History sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
              <Typography color="text.secondary">No run history available</Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setHistoryDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ReportScheduler;