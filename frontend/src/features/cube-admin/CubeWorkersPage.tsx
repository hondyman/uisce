import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Grid,
  Paper,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Slider,
  Switch,
  FormControlLabel,
  Alert,
  Tabs,
  Tab,
  Badge,
  CircularProgress,
  Divider,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  PlayArrow as PlayIcon,
  Stop as StopIcon,
  ScaleOutlined as ScaleIcon,
  Memory as MemoryIcon,
  Speed as SpeedIcon,
  Storage as StorageIcon,
  Schedule as ScheduleIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  HourglassEmpty as PendingIcon,
  Sync as RunningIcon,
  TrendingUp as TrendingUpIcon,
  Settings as SettingsIcon,
  Add as AddIcon,
  Visibility as ViewIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { Line, Doughnut, Bar } from 'react-chartjs-2';

interface WorkerPool {
  id: string;
  name: string;
  display_name: string;
  description: string;
  tier: string;
  min_workers: number;
  max_workers: number;
  current_workers: number;
  target_workers: number;
  memory_limit_mb: number;
  cpu_limit_cores: number;
  concurrent_jobs: number;
  queue_size: number;
  auto_scale_enabled: boolean;
  scale_up_threshold: number;
  scale_down_threshold: number;
  scale_cooldown_seconds: number;
  status: string;
  last_scale_at: string | null;
  health_check_at: string | null;
}

interface WorkerInstance {
  id: string;
  pool_id: string;
  instance_id: string;
  hostname: string;
  ip_address: string;
  status: string;
  current_job_id: string | null;
  jobs_completed: number;
  jobs_failed: number;
  memory_used_mb: number;
  cpu_used_percent: number;
  started_at: string;
  last_heartbeat_at: string;
}

interface PreAggJob {
  id: string;
  preagg_id: string;
  tenant_id: string;
  tenant_instance_id: string;
  job_type: string;
  partition_key: string;
  priority: number;
  status: string;
  progress_percent: number;
  current_step: string;
  scheduled_at: string;
  started_at: string | null;
  completed_at: string | null;
  rows_processed: number;
  bytes_written: number;
  duration_ms: number | null;
  retry_count: number;
  max_retries: number;
  error_message: string;
}

interface QueueStats {
  pending: number;
  queued: number;
  running: number;
  completed_1h: number;
  failed_1h: number;
  avg_duration_ms: number;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div hidden={value !== index} {...other}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

const CubeWorkersPage: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [pools, setPools] = useState<WorkerPool[]>([]);
  const [workers, setWorkers] = useState<WorkerInstance[]>([]);
  const [jobs, setJobs] = useState<PreAggJob[]>([]);
  const [queueStats, setQueueStats] = useState<QueueStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedPool, setSelectedPool] = useState<WorkerPool | null>(null);
  const [scaleDialogOpen, setScaleDialogOpen] = useState(false);
  const [targetWorkers, setTargetWorkers] = useState(1);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      // Fetch worker pools
      const poolsRes = await fetch('/api/cube/worker-pools');
      if (poolsRes.ok) {
        const poolsData = await poolsRes.json();
        setPools(poolsData || []);
      }

      // Fetch queue stats
      const statsRes = await fetch('/api/cube/jobs/stats');
      if (statsRes.ok) {
        const statsData = await statsRes.json();
        setQueueStats(statsData);
      }

      // Fetch recent jobs
      const jobsRes = await fetch('/api/cube/jobs?limit=50');
      if (jobsRes.ok) {
        const jobsData = await jobsRes.json();
        setJobs(jobsData || []);
      }
    } catch (err) {
      console.error('Failed to fetch worker data:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchPoolWorkers = async (poolId: string) => {
    const res = await fetch(`/api/cube/worker-pools/${poolId}/workers`);
    if (res.ok) {
      const data = await res.json();
      setWorkers(data || []);
    }
  };

  useEffect(() => {
    fetchData();
    let interval: NodeJS.Timeout | null = null;
    if (autoRefresh) {
      interval = setInterval(fetchData, 10000); // Refresh every 10 seconds
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [fetchData, autoRefresh]);

  useEffect(() => {
    if (selectedPool) {
      fetchPoolWorkers(selectedPool.id);
    }
  }, [selectedPool]);

  const handleScalePool = async () => {
    if (!selectedPool) return;
    try {
      await fetch(`/api/cube/worker-pools/${selectedPool.id}/scale`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target_workers: targetWorkers }),
      });
      setScaleDialogOpen(false);
      fetchData();
    } catch (err) {
      console.error('Failed to scale pool:', err);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
      case 'completed':
      case 'healthy':
        return 'success';
      case 'running':
      case 'processing':
        return 'info';
      case 'pending':
      case 'queued':
      case 'starting':
        return 'warning';
      case 'failed':
      case 'error':
      case 'unhealthy':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <SuccessIcon color="success" />;
      case 'running':
        return <RunningIcon color="info" />;
      case 'pending':
      case 'queued':
        return <PendingIcon color="warning" />;
      case 'failed':
        return <ErrorIcon color="error" />;
      default:
        return null;
    }
  };

  const formatDuration = (ms: number | null) => {
    if (!ms) return '-';
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 400 }}>
        <CircularProgress />
      </Box>
    );
  }

  const totalWorkers = pools.reduce((sum, p) => sum + p.current_workers, 0);
  const activeJobs = jobs.filter(j => j.status === 'running').length;

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Worker & Pre-Aggregation Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage refresh workers, job queues, and pre-aggregation builds
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
          <FormControlLabel
            control={
              <Switch
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                size="small"
              />
            }
            label="Auto-refresh"
          />
          <IconButton onClick={fetchData}>
            <RefreshIcon />
          </IconButton>
        </Box>
      </Box>

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <MemoryIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6">{pools.length}</Typography>
              </Box>
              <Typography color="text.secondary" variant="body2">
                Worker Pools
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <SpeedIcon color="success" sx={{ mr: 1 }} />
                <Typography variant="h6">{totalWorkers}</Typography>
              </Box>
              <Typography color="text.secondary" variant="body2">
                Active Workers
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <RunningIcon color="info" sx={{ mr: 1 }} />
                <Typography variant="h6">{activeJobs}</Typography>
              </Box>
              <Typography color="text.secondary" variant="body2">
                Running Jobs
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                <ScheduleIcon color="warning" sx={{ mr: 1 }} />
                <Typography variant="h6">{queueStats?.pending || 0}</Typography>
              </Box>
              <Typography color="text.secondary" variant="body2">
                Pending Jobs
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)}>
          <Tab
            label={
              <Badge badgeContent={pools.length} color="primary">
                Worker Pools
              </Badge>
            }
          />
          <Tab
            label={
              <Badge badgeContent={queueStats?.running || 0} color="info">
                Job Queue
              </Badge>
            }
          />
          <Tab label="Autoscaling" />
          <Tab label="Metrics" />
        </Tabs>
      </Paper>

      {/* Worker Pools Tab */}
      <TabPanel value={tabValue} index={0}>
        <Grid container spacing={3}>
          {pools.map((pool) => (
            <Grid item xs={12} md={6} key={pool.id}>
              <Card variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Box>
                      <Typography variant="h6">{pool.display_name}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {pool.description}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      <Chip label={pool.tier} size="small" color="primary" />
                      <Chip
                        label={pool.status}
                        size="small"
                        color={getStatusColor(pool.status) as any}
                      />
                    </Box>
                  </Box>

                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <Typography variant="caption" color="text.secondary">
                        Workers
                      </Typography>
                      <Typography variant="h5">
                        {pool.current_workers} / {pool.max_workers}
                      </Typography>
                      <LinearProgress
                        variant="determinate"
                        value={(pool.current_workers / pool.max_workers) * 100}
                        sx={{ mt: 1 }}
                      />
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="caption" color="text.secondary">
                        Queue Size
                      </Typography>
                      <Typography variant="h5">{pool.queue_size}</Typography>
                      <LinearProgress
                        variant="determinate"
                        value={Math.min((pool.queue_size / 100) * 100, 100)}
                        color="warning"
                        sx={{ mt: 1 }}
                      />
                    </Grid>
                  </Grid>

                  <Divider sx={{ my: 2 }} />

                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Box>
                      <Typography variant="caption" color="text.secondary">
                        Resources: {pool.memory_limit_mb}MB / {pool.cpu_limit_cores} cores
                      </Typography>
                      <br />
                      <Typography variant="caption" color="text.secondary">
                        Concurrent Jobs: {pool.concurrent_jobs}
                      </Typography>
                    </Box>
                    {pool.auto_scale_enabled && (
                      <Chip
                        icon={<TrendingUpIcon />}
                        label="Auto-scale"
                        size="small"
                        color="success"
                        variant="outlined"
                      />
                    )}
                  </Box>
                </CardContent>
                <CardActions>
                  <Button
                    size="small"
                    startIcon={<ScaleIcon />}
                    onClick={() => {
                      setSelectedPool(pool);
                      setTargetWorkers(pool.target_workers);
                      setScaleDialogOpen(true);
                    }}
                  >
                    Scale
                  </Button>
                  <Button
                    size="small"
                    startIcon={<ViewIcon />}
                    onClick={() => setSelectedPool(pool)}
                  >
                    View Workers
                  </Button>
                  <Button size="small" startIcon={<SettingsIcon />}>
                    Configure
                  </Button>
                </CardActions>
              </Card>
            </Grid>
          ))}

          {pools.length === 0 && (
            <Grid item xs={12}>
              <Alert severity="info">
                No worker pools configured. Create a pool to start managing pre-aggregation workers.
              </Alert>
            </Grid>
          )}
        </Grid>

        {/* Worker Instances Table */}
        {selectedPool && (
          <Paper sx={{ mt: 3, p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Workers in {selectedPool.display_name}
            </Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Instance</TableCell>
                    <TableCell>Hostname</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>CPU %</TableCell>
                    <TableCell>Memory</TableCell>
                    <TableCell>Jobs Done</TableCell>
                    <TableCell>Last Heartbeat</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {workers.map((worker) => (
                    <TableRow key={worker.id}>
                      <TableCell>
                        <Typography variant="body2" fontFamily="monospace">
                          {worker.instance_id.slice(0, 12)}
                        </Typography>
                      </TableCell>
                      <TableCell>{worker.hostname}</TableCell>
                      <TableCell>
                        <Chip
                          label={worker.status}
                          size="small"
                          color={getStatusColor(worker.status) as any}
                        />
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <LinearProgress
                            variant="determinate"
                            value={worker.cpu_used_percent}
                            sx={{ width: 50 }}
                          />
                          <Typography variant="caption">
                            {worker.cpu_used_percent.toFixed(1)}%
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>{worker.memory_used_mb} MB</TableCell>
                      <TableCell>
                        {worker.jobs_completed}
                        {worker.jobs_failed > 0 && (
                          <Typography component="span" color="error" variant="caption">
                            {' '}/ {worker.jobs_failed} failed
                          </Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        {new Date(worker.last_heartbeat_at).toLocaleTimeString()}
                      </TableCell>
                      <TableCell>
                        <Tooltip title="Stop Worker">
                          <IconButton size="small" color="error">
                            <StopIcon />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                  {workers.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={8} align="center">
                        <Typography color="text.secondary">No workers running</Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        )}
      </TabPanel>

      {/* Job Queue Tab */}
      <TabPanel value={tabValue} index={1}>
        {/* Queue Stats */}
        {queueStats && (
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={2}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h4" color="warning.main">
                  {queueStats.pending}
                </Typography>
                <Typography variant="caption">Pending</Typography>
              </Paper>
            </Grid>
            <Grid item xs={2}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h4" color="info.main">
                  {queueStats.running}
                </Typography>
                <Typography variant="caption">Running</Typography>
              </Paper>
            </Grid>
            <Grid item xs={2}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h4" color="success.main">
                  {queueStats.completed_1h}
                </Typography>
                <Typography variant="caption">Completed (1h)</Typography>
              </Paper>
            </Grid>
            <Grid item xs={2}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h4" color="error.main">
                  {queueStats.failed_1h}
                </Typography>
                <Typography variant="caption">Failed (1h)</Typography>
              </Paper>
            </Grid>
            <Grid item xs={4}>
              <Paper sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="h4">
                  {formatDuration(queueStats.avg_duration_ms)}
                </Typography>
                <Typography variant="caption">Avg Duration (1h)</Typography>
              </Paper>
            </Grid>
          </Grid>
        )}

        {/* Jobs Table */}
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Status</TableCell>
                <TableCell>Job Type</TableCell>
                <TableCell>Partition</TableCell>
                <TableCell>Priority</TableCell>
                <TableCell>Progress</TableCell>
                <TableCell>Rows</TableCell>
                <TableCell>Duration</TableCell>
                <TableCell>Retries</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {jobs.map((job) => (
                <TableRow key={job.id}>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {getStatusIcon(job.status)}
                      <Chip
                        label={job.status}
                        size="small"
                        color={getStatusColor(job.status) as any}
                      />
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip label={job.job_type} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" fontFamily="monospace">
                      {job.partition_key || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={`P${job.priority}`}
                      size="small"
                      color={job.priority > 50 ? 'error' : job.priority > 25 ? 'warning' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 100 }}>
                      <LinearProgress
                        variant="determinate"
                        value={job.progress_percent}
                        sx={{ flexGrow: 1 }}
                      />
                      <Typography variant="caption">{job.progress_percent}%</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>{job.rows_processed.toLocaleString()}</TableCell>
                  <TableCell>{formatDuration(job.duration_ms)}</TableCell>
                  <TableCell>
                    {job.retry_count > 0 ? (
                      <Typography color={job.retry_count >= job.max_retries ? 'error' : 'warning'}>
                        {job.retry_count}/{job.max_retries}
                      </Typography>
                    ) : (
                      '-'
                    )}
                  </TableCell>
                  <TableCell>
                    {job.status === 'running' && (
                      <Tooltip title="Cancel Job">
                        <IconButton size="small" color="warning">
                          <StopIcon />
                        </IconButton>
                      </Tooltip>
                    )}
                    {job.status === 'failed' && (
                      <Tooltip title="Retry Job">
                        <IconButton size="small" color="primary">
                          <RefreshIcon />
                        </IconButton>
                      </Tooltip>
                    )}
                    {job.error_message && (
                      <Tooltip title={job.error_message}>
                        <IconButton size="small" color="error">
                          <ErrorIcon />
                        </IconButton>
                      </Tooltip>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {jobs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={9} align="center">
                    <Typography color="text.secondary" sx={{ py: 3 }}>
                      No jobs in queue
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </TabPanel>

      {/* Autoscaling Tab */}
      <TabPanel value={tabValue} index={2}>
        <Alert severity="info" sx={{ mb: 3 }}>
          Configure autoscaling rules to automatically adjust worker capacity based on queue depth and resource utilization.
        </Alert>
        
        <Grid container spacing={3}>
          {pools.map((pool) => (
            <Grid item xs={12} md={6} key={pool.id}>
              <Paper sx={{ p: 3 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="h6">{pool.display_name}</Typography>
                  <Switch
                    checked={pool.auto_scale_enabled}
                    color="success"
                  />
                </Box>

                <Box sx={{ mb: 3 }}>
                  <Typography variant="subtitle2" gutterBottom>
                    Worker Range
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Min: {pool.min_workers} | Max: {pool.max_workers}
                  </Typography>
                  <Slider
                    value={[pool.min_workers, pool.max_workers]}
                    min={1}
                    max={20}
                    marks
                    disabled
                    sx={{ mt: 1 }}
                  />
                </Box>

                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Typography variant="subtitle2">Scale Up Threshold</Typography>
                    <Typography variant="h5" color="error.main">
                      {(pool.scale_up_threshold * 100).toFixed(0)}%
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Queue utilization
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="subtitle2">Scale Down Threshold</Typography>
                    <Typography variant="h5" color="success.main">
                      {(pool.scale_down_threshold * 100).toFixed(0)}%
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Queue utilization
                    </Typography>
                  </Grid>
                </Grid>

                <Divider sx={{ my: 2 }} />

                <Typography variant="body2" color="text.secondary">
                  Cooldown Period: {pool.scale_cooldown_seconds}s
                </Typography>
                {pool.last_scale_at && (
                  <Typography variant="body2" color="text.secondary">
                    Last Scaled: {new Date(pool.last_scale_at).toLocaleString()}
                  </Typography>
                )}
              </Paper>
            </Grid>
          ))}
        </Grid>
      </TabPanel>

      {/* Metrics Tab */}
      <TabPanel value={tabValue} index={3}>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Job Throughput (24h)
              </Typography>
              <Box sx={{ height: 300 }}>
                <Typography color="text.secondary" sx={{ textAlign: 'center', pt: 10 }}>
                  Chart will display job completion rate over time
                </Typography>
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Job Duration Distribution
              </Typography>
              <Box sx={{ height: 300 }}>
                <Typography color="text.secondary" sx={{ textAlign: 'center', pt: 10 }}>
                  Chart will display job duration histogram
                </Typography>
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Worker Utilization
              </Typography>
              <Box sx={{ height: 300 }}>
                <Typography color="text.secondary" sx={{ textAlign: 'center', pt: 10 }}>
                  Chart will display CPU/memory usage by pool
                </Typography>
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Queue Depth Over Time
              </Typography>
              <Box sx={{ height: 300 }}>
                <Typography color="text.secondary" sx={{ textAlign: 'center', pt: 10 }}>
                  Chart will display pending jobs over time
                </Typography>
              </Box>
            </Paper>
          </Grid>
        </Grid>
      </TabPanel>

      {/* Scale Dialog */}
      <Dialog open={scaleDialogOpen} onClose={() => setScaleDialogOpen(false)}>
        <DialogTitle>Scale Worker Pool</DialogTitle>
        <DialogContent>
          {selectedPool && (
            <Box sx={{ pt: 2, minWidth: 400 }}>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                Adjust the target number of workers for {selectedPool.display_name}
              </Typography>
              <Box sx={{ mt: 3 }}>
                <Typography gutterBottom>
                  Target Workers: {targetWorkers}
                </Typography>
                <Slider
                  value={targetWorkers}
                  onChange={(_, v) => setTargetWorkers(v as number)}
                  min={selectedPool.min_workers}
                  max={selectedPool.max_workers}
                  marks
                  valueLabelDisplay="auto"
                />
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="caption">Min: {selectedPool.min_workers}</Typography>
                  <Typography variant="caption">Max: {selectedPool.max_workers}</Typography>
                </Box>
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setScaleDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleScalePool} variant="contained" color="primary">
            Scale
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default CubeWorkersPage;
