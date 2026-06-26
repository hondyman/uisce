import React, { useState, useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  CardActionArea,
  Chip,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tabs,
  Tab,
  Stack,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Divider,
  Alert,
  CircularProgress,
  Tooltip,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import {
  Description as ReportIcon,
  PlayArrow as RunIcon,
  Schedule as ScheduleIcon,
  Download as DownloadIcon,
  Visibility as ViewIcon,
  Settings as SettingsIcon,
  History as HistoryIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
  Add as AddIcon,
  PieChart as ChartIcon,
  TableChart as TableIcon,
  Assessment as AssessmentIcon,
} from '@mui/icons-material';

// ============================================================================
// Types
// ============================================================================

interface ReportBusinessObject {
  id: string;
  key: string;
  name: string;
  display_name: string;
  description: string;
  category: string;
  report_type: 'standard' | 'composite' | 'scheduled';
  output_formats: string[];
  is_system: boolean;
}

interface ReportParameter {
  name: string;
  label: string;
  type: 'string' | 'number' | 'date' | 'daterange' | 'select' | 'multiselect';
  required: boolean;
  default_value?: any;
  options?: { label: string; value: any }[];
}

interface ReportExecution {
  id: string;
  report_key: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
  parameters: Record<string, any>;
  output_format: string;
  output_url?: string;
  row_count?: number;
  generation_ms?: number;
  started_at: string;
  completed_at?: string;
  error?: string;
}

// ============================================================================
// Report Category Icons
// ============================================================================

const getCategoryIcon = (category: string) => {
  switch (category) {
    case 'portfolio':
      return <ChartIcon />;
    case 'performance':
      return <AssessmentIcon />;
    case 'activity':
    case 'billing':
      return <TableIcon />;
    default:
      return <ReportIcon />;
  }
};

const getCategoryColor = (category: string) => {
  switch (category) {
    case 'portfolio':
      return '#2196f3';
    case 'performance':
      return '#4caf50';
    case 'activity':
      return '#ff9800';
    case 'billing':
      return '#9c27b0';
    case 'tax':
      return '#f44336';
    default:
      return '#607d8b';
  }
};

// ============================================================================
// Report Card Component
// ============================================================================

interface ReportCardProps {
  report: ReportBusinessObject;
  onRun: (report: ReportBusinessObject) => void;
  onSchedule?: (report: ReportBusinessObject) => void;
}

const ReportCard: React.FC<ReportCardProps> = ({ report, onRun, onSchedule }) => (
  <Card
    sx={{
      height: '100%',
      transition: 'transform 0.2s, box-shadow 0.2s',
      '&:hover': {
        transform: 'translateY(-2px)',
        boxShadow: 4,
      },
    }}
  >
    <CardContent>
      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
        <Box
          sx={{
            p: 1,
            borderRadius: 1,
            bgcolor: `${getCategoryColor(report.category)}15`,
            color: getCategoryColor(report.category),
          }}
        >
          {getCategoryIcon(report.category)}
        </Box>
        <Box flex={1}>
          <Typography variant="subtitle1" fontWeight="bold">
            {report.display_name}
          </Typography>
          <Stack direction="row" spacing={0.5}>
            <Chip
              label={report.category}
              size="small"
              sx={{
                bgcolor: `${getCategoryColor(report.category)}20`,
                color: getCategoryColor(report.category),
              }}
            />
            {report.is_system && (
              <Chip label="System" size="small" color="info" variant="outlined" />
            )}
          </Stack>
        </Box>
      </Stack>

      <Typography variant="body2" color="text.secondary" sx={{ mb: 2, minHeight: 40 }}>
        {report.description}
      </Typography>

      <Stack direction="row" spacing={1}>
        <Button
          variant="contained"
          size="small"
          startIcon={<RunIcon />}
          onClick={() => onRun(report)}
          sx={{ flex: 1 }}
        >
          Run
        </Button>
        {onSchedule && (
          <Tooltip title="Schedule">
            <IconButton size="small" onClick={() => onSchedule(report)}>
              <ScheduleIcon />
            </IconButton>
          </Tooltip>
        )}
      </Stack>

      <Stack direction="row" spacing={0.5} sx={{ mt: 1 }}>
        {report.output_formats.map((format) => (
          <Chip
            key={format}
            label={format.toUpperCase()}
            size="small"
            variant="outlined"
            sx={{ fontSize: '0.65rem', height: 20 }}
          />
        ))}
      </Stack>
    </CardContent>
  </Card>
);

// ============================================================================
// Parameter Input Component
// ============================================================================

interface ParameterInputProps {
  parameter: ReportParameter;
  value: any;
  onChange: (value: any) => void;
}

const ParameterInput: React.FC<ParameterInputProps> = ({ parameter, value, onChange }) => {
  switch (parameter.type) {
    case 'date':
      return (
        <TextField
          fullWidth
          type="date"
          label={parameter.label}
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          required={parameter.required}
          InputLabelProps={{ shrink: true }}
        />
      );
    case 'select':
      return (
        <FormControl fullWidth>
          <InputLabel>{parameter.label}</InputLabel>
          <Select
            value={value || ''}
            label={parameter.label}
            onChange={(e) => onChange(e.target.value)}
          >
            {parameter.options?.map((opt) => (
              <MenuItem key={opt.value} value={opt.value}>
                {opt.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      );
    case 'number':
      return (
        <TextField
          fullWidth
          type="number"
          label={parameter.label}
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          required={parameter.required}
        />
      );
    default:
      return (
        <TextField
          fullWidth
          label={parameter.label}
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          required={parameter.required}
        />
      );
  }
};

// ============================================================================
// Run Report Dialog
// ============================================================================

interface RunReportDialogProps {
  open: boolean;
  report: ReportBusinessObject | null;
  parameters: ReportParameter[];
  onClose: () => void;
  onRun: (params: Record<string, any>, format: string) => Promise<void>;
}

const RunReportDialog: React.FC<RunReportDialogProps> = ({
  open,
  report,
  parameters,
  onClose,
  onRun,
}) => {
  const [paramValues, setParamValues] = useState<Record<string, any>>({});
  const [outputFormat, setOutputFormat] = useState('html');
  const [loading, setLoading] = useState(false);

  const handleRun = async () => {
    setLoading(true);
    try {
      await onRun(paramValues, outputFormat);
      onClose();
    } finally {
      setLoading(false);
    }
  };

  if (!report) return null;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        <Stack direction="row" spacing={1} alignItems="center">
          {getCategoryIcon(report.category)}
          <span>Run {report.display_name}</span>
        </Stack>
      </DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {report.description}
        </Typography>

        {parameters.length > 0 && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              Parameters
            </Typography>
            <Grid container spacing={2}>
              {parameters.map((param) => (
                <Grid item xs={12} sm={6} key={param.name}>
                  <ParameterInput
                    parameter={param}
                    value={paramValues[param.name]}
                    onChange={(v) =>
                      setParamValues({ ...paramValues, [param.name]: v })
                    }
                  />
                </Grid>
              ))}
            </Grid>
          </Box>
        )}

        <FormControl fullWidth sx={{ mt: 2 }}>
          <InputLabel>Output Format</InputLabel>
          <Select
            value={outputFormat}
            label="Output Format"
            onChange={(e) => setOutputFormat(e.target.value)}
          >
            {report.output_formats.map((format) => (
              <MenuItem key={format} value={format}>
                {format.toUpperCase()}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleRun}
          disabled={loading}
          startIcon={loading ? <CircularProgress size={20} /> : <RunIcon />}
        >
          Run Report
        </Button>
      </DialogActions>
    </Dialog>
  );
};

// ============================================================================
// Execution History Component
// ============================================================================

interface ExecutionHistoryProps {
  executions: ReportExecution[];
  onView: (execution: ReportExecution) => void;
  onDownload: (execution: ReportExecution) => void;
}

const ExecutionHistory: React.FC<ExecutionHistoryProps> = ({
  executions,
  onView,
  onDownload,
}) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'running':
        return 'info';
      case 'failed':
        return 'error';
      default:
        return 'default';
    }
  };

  return (
    <TableContainer component={Paper}>
      <Table size="small">
        <TableHead>
          <TableRow sx={{ bgcolor: 'grey.50' }}>
            <TableCell>Report</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Format</TableCell>
            <TableCell align="right">Rows</TableCell>
            <TableCell align="right">Time</TableCell>
            <TableCell>Started</TableCell>
            <TableCell align="center">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {executions.map((exec) => (
            <TableRow key={exec.id} hover>
              <TableCell>{exec.report_key}</TableCell>
              <TableCell>
                <Chip
                  label={exec.status}
                  size="small"
                  color={getStatusColor(exec.status) as any}
                />
              </TableCell>
              <TableCell>{exec.output_format.toUpperCase()}</TableCell>
              <TableCell align="right">{exec.row_count?.toLocaleString() || '-'}</TableCell>
              <TableCell align="right">
                {exec.generation_ms ? `${(exec.generation_ms / 1000).toFixed(2)}s` : '-'}
              </TableCell>
              <TableCell>{new Date(exec.started_at).toLocaleString()}</TableCell>
              <TableCell align="center">
                <Stack direction="row" spacing={0.5} justifyContent="center">
                  {exec.status === 'completed' && (
                    <>
                      <IconButton size="small" onClick={() => onView(exec)}>
                        <ViewIcon fontSize="small" />
                      </IconButton>
                      <IconButton size="small" onClick={() => onDownload(exec)}>
                        <DownloadIcon fontSize="small" />
                      </IconButton>
                    </>
                  )}
                </Stack>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

// ============================================================================
// Main Report Builder Component
// ============================================================================

interface ReportBuilderProps {
  reports: ReportBusinessObject[];
  executions: ReportExecution[];
  onRunReport: (
    reportKey: string,
    params: Record<string, any>,
    format: string
  ) => Promise<void>;
  onViewExecution: (execution: ReportExecution) => void;
  onDownloadExecution: (execution: ReportExecution) => void;
  onRefresh: () => void;
}

export const ReportBuilder: React.FC<ReportBuilderProps> = ({
  reports,
  executions,
  onRunReport,
  onViewExecution,
  onDownloadExecution,
  onRefresh,
}) => {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [runDialogOpen, setRunDialogOpen] = useState(false);
  const [selectedReport, setSelectedReport] = useState<ReportBusinessObject | null>(null);

  const categories = useMemo(() => {
    const cats = [...new Set(reports.map((r) => r.category))];
    return ['all', ...cats];
  }, [reports]);

  const filteredReports = useMemo(() => {
    if (selectedCategory === 'all') return reports;
    return reports.filter((r) => r.category === selectedCategory);
  }, [reports, selectedCategory]);

  const handleRunClick = (report: ReportBusinessObject) => {
    setSelectedReport(report);
    setRunDialogOpen(true);
  };

  const handleRunReport = async (params: Record<string, any>, format: string) => {
    if (!selectedReport) return;
    await onRunReport(selectedReport.key, params, format);
  };

  return (
    <Box>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight="bold">
            Report Builder
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Generate and schedule business reports with semantic layer integration
          </Typography>
        </Box>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined" startIcon={<RefreshIcon />} onClick={onRefresh}>
            Refresh
          </Button>
        </Stack>
      </Stack>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label="Available Reports" icon={<ReportIcon />} iconPosition="start" />
          <Tab label="Execution History" icon={<HistoryIcon />} iconPosition="start" />
        </Tabs>
      </Paper>

      {/* Available Reports Tab */}
      {activeTab === 0 && (
        <>
          {/* Category Filter */}
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            {categories.map((cat) => (
              <Chip
                key={cat}
                label={cat === 'all' ? 'All Reports' : cat}
                onClick={() => setSelectedCategory(cat)}
                color={selectedCategory === cat ? 'primary' : 'default'}
                variant={selectedCategory === cat ? 'filled' : 'outlined'}
                sx={{ textTransform: 'capitalize' }}
              />
            ))}
          </Stack>

          {/* Report Grid */}
          <Grid container spacing={2}>
            {filteredReports.map((report) => (
              <Grid item xs={12} sm={6} md={4} key={report.id}>
                <ReportCard report={report} onRun={handleRunClick} />
              </Grid>
            ))}
          </Grid>

          {filteredReports.length === 0 && (
            <Alert severity="info" sx={{ mt: 2 }}>
              No reports available in this category.
            </Alert>
          )}
        </>
      )}

      {/* Execution History Tab */}
      {activeTab === 1 && (
        <ExecutionHistory
          executions={executions}
          onView={onViewExecution}
          onDownload={onDownloadExecution}
        />
      )}

      {/* Run Report Dialog */}
      <RunReportDialog
        open={runDialogOpen}
        report={selectedReport}
        parameters={[]} // Would be fetched from report definition
        onClose={() => setRunDialogOpen(false)}
        onRun={handleRunReport}
      />
    </Box>
  );
};

export default ReportBuilder;
