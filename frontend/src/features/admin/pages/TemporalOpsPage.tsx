import React, { Suspense, lazy, useState, useEffect, useCallback, useMemo } from 'react';
import {
  Box,
  Dialog,
  CircularProgress,
  Container,
  Typography,
  Paper,
  Grid,
  TextField,
  InputAdornment,
  Button,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  Alert,
  Stack,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  PlayCircleOutline as PlayIcon,
  HelpOutline as QueryIcon,
  Send as SignalIcon,
  Cancel as TerminateIcon,
  InfoOutlined as InfoIcon,
} from '@mui/icons-material';
import { formatDistanceToNow } from 'date-fns';
import { devLog } from '../../../utils/devLogger';

const TemporalOpsPanel = lazy(() => import('../components/TemporalOpsPanel'));

interface WorkflowExecution {
  id: string;
  workflow_id: string;
  workflow_name: string;
  status: 'running' | 'completed' | 'failed' | 'timed_out' | 'terminated';
  start_time: string;
  end_time?: string;
  duration_ms?: number;
}

// Mock hook for fetching Temporal data
const useTemporal = () => {
  const [workflows, setWorkflows] = useState<WorkflowExecution[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkflows = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Mock API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      const mockData: WorkflowExecution[] = [
        { id: 'wf-1', workflow_id: 'onboarding-process-123', workflow_name: 'EmployeeOnboarding', status: 'running', start_time: new Date(Date.now() - 1000 * 60 * 5).toISOString(), duration_ms: 300000 },
        { id: 'wf-2', workflow_id: 'invoice-processing-456', workflow_name: 'InvoiceProcessing', status: 'completed', start_time: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(), end_time: new Date(Date.now() - 1000 * 60 * 60 * 1).toISOString(), duration_ms: 3600000 },
        { id: 'wf-3', workflow_id: 'onboarding-process-124', workflow_name: 'EmployeeOnboarding', status: 'failed', start_time: new Date(Date.now() - 1000 * 60 * 30).toISOString(), end_time: new Date(Date.now() - 1000 * 60 * 25).toISOString(), duration_ms: 300000 },
        { id: 'wf-4', workflow_id: 'order-approval-789', workflow_name: 'OrderApproval', status: 'timed_out', start_time: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(), end_time: new Date(Date.now() - 1000 * 60 * 60 * 12).toISOString(), duration_ms: 43200000 },
      ];
      setWorkflows(mockData);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch workflows');
    } finally {
      setLoading(false);
    }
  }, []);

  return { workflows, loading, error, fetchWorkflows };
};

const getStatusChip = (status: WorkflowExecution['status']) => {
  const colorMap: Record<typeof status, 'success' | 'info' | 'error' | 'warning' | 'default'> = {
    running: 'info',
    completed: 'success',
    failed: 'error',
    timed_out: 'warning',
    terminated: 'default',
  };
  return <Chip label={status} color={colorMap[status]} size="small" />;
};

const TemporalOpsPage: React.FC = () => {
  // keep imported icon referenced to silence no-unused-vars when icon is not rendered yet
  void PlayIcon;
  const { workflows, loading, error, fetchWorkflows } = useTemporal();
  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState(0);
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowExecution | null>(null);
  const [actionDialog, setActionDialog] = useState<{
    open: boolean;
    action: 'query' | 'signal' | 'terminate' | null;
    workflow: WorkflowExecution | null;
  }>({ open: false, action: null, workflow: null });

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  const filteredWorkflows = useMemo(() => {
    // When a workflow is selected, we want to keep it visible in the list,
    // even if the search term would otherwise filter it out.
    if (selectedWorkflow && searchTerm) {
      return workflows.filter(wf => wf.id === selectedWorkflow.id);
    }

    const query = searchTerm.toLowerCase();
    return workflows.filter(wf =>
      wf.workflow_id.toLowerCase().includes(query) ||
      wf.workflow_name.toLowerCase().includes(query)
    );
  }, [workflows, searchTerm]);

  const handleActionClick = (action: 'query' | 'signal' | 'terminate', workflow: WorkflowExecution) => {
    setActionDialog({ open: true, action, workflow });
  };

  const handleCloseActionDialog = () => {
    setActionDialog({ open: false, action: null, workflow: null });
  };

  const handleConfirmAction = async () => {
    if (!actionDialog.action || !actionDialog.workflow) return;
    // Mock action
    devLog(`Performing ${actionDialog.action} on ${actionDialog.workflow.workflow_id}`);
    await new Promise(resolve => setTimeout(resolve, 500));
    handleCloseActionDialog();
    fetchWorkflows(); // Refresh list after action
  };

  const renderWorkflowTable = (data: WorkflowExecution[]) => (
    <TableContainer component={Paper} variant="outlined">
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Workflow ID</TableCell>
            <TableCell>Name</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Start Time</TableCell>
            <TableCell>Duration</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {data.map((wf) => (
            <TableRow
              key={wf.id}
              hover
              onClick={() => setSelectedWorkflow(wf)}
              sx={{ cursor: 'pointer', '&.Mui-selected': { bgcolor: 'action.selected' } }}
              selected={selectedWorkflow?.id === wf.id}
            >
              <TableCell>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>{wf.workflow_id}</Typography>
              </TableCell>
              <TableCell>{wf.workflow_name}</TableCell>
              <TableCell><Box sx={{ display: 'flex' }}>{getStatusChip(wf.status)}</Box></TableCell>
                <TableCell>
                <Tooltip title={new Date(wf.start_time).toLocaleString()}><span>{formatDistanceToNow(new Date(wf.start_time), { addSuffix: true })}</span></Tooltip>
              </TableCell>
              <TableCell>{wf.duration_ms ? `${(wf.duration_ms / 1000).toFixed(2)}s` : '-'}</TableCell>
              <TableCell align="right">
                <Tooltip title="Query Workflow">
                  <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleActionClick('query', wf); }}>
                    <QueryIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Signal Workflow">
                  <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleActionClick('signal', wf); }}>
                    <SignalIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Terminate Workflow">
                    <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleActionClick('terminate', wf); }}>
                    <TerminateIcon fontSize="small" color="error" />
                  </IconButton>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );

  return (
    <Container maxWidth={false} sx={{ mt: 4, mb: 4, height: 'calc(100vh - 120px)', display: 'flex', flexDirection: 'column' }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Temporal Operations
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Troubleshoot Temporal workflows, inspect history, and run supported runbook actions.
          </Typography>
        </Box>
        <Button variant="outlined" startIcon={<RefreshIcon />} onClick={fetchWorkflows} disabled={loading}>
          Refresh
        </Button>
      </Stack>

      <Paper sx={{ p: 2, mb: 2 }}>
        <TextField
          fullWidth
          size="small"
          placeholder="Search by Workflow ID or Name..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Paper>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={2} sx={{ flex: 1, overflow: 'hidden' }}>
        <Grid item xs={12} md={7} sx={{ display: 'flex', flexDirection: 'column' }}>
          <Paper variant="outlined">
            <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
              <Tab label="Running" />
              <Tab label="History" />
            </Tabs>
            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress /></Box>
            ) : (
              <>
                {activeTab === 0 && renderWorkflowTable(filteredWorkflows.filter(wf => wf.status === 'running'))}
                {activeTab === 1 && renderWorkflowTable(filteredWorkflows.filter(wf => wf.status !== 'running'))}
              </>
            )}
          </Paper>
        </Grid>

        <Grid item xs={12} md={5} sx={{ display: 'flex', flexDirection: 'column' }}>
          <Paper variant="outlined" sx={{ flex: 1, position: 'sticky', top: '80px' }}>
            {selectedWorkflow ? (
              <Suspense fallback={<Box sx={{ p: 4, textAlign: 'center' }}><CircularProgress /></Box>}>
                {React.createElement(TemporalOpsPanel as any, {
                  workflowId: selectedWorkflow.workflow_id,
                  runId: selectedWorkflow.id,
                  onClose: () => setSelectedWorkflow(null)
                })}
              </Suspense>
            ) : (
              <Box sx={{ p: 4, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', textAlign: 'center', color: 'text.secondary' }}>
                <InfoIcon sx={{ fontSize: 48, mb: 2 }} />
                <Typography variant="h6" gutterBottom>
                  No Workflow Selected
                </Typography>
                <Typography variant="body1">
                  Select a workflow from the list on the left to view its details and perform actions.
                </Typography>
              </Box>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Dialog open={actionDialog.open} onClose={handleCloseActionDialog}>
        <DialogTitle>Confirm Action: {actionDialog.action?.toUpperCase()}</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to <strong>{actionDialog.action}</strong> the workflow?
          </Typography>
          <Typography variant="body2" sx={{ fontFamily: 'monospace', mt: 2, p: 1, bgcolor: 'action.hover', borderRadius: 1 }}>
            {actionDialog.workflow?.workflow_id}
          </Typography>
          {actionDialog.action === 'signal' && (
            <Stack spacing={2} sx={{ mt: 2 }}>
              <TextField label="Signal Name" variant="outlined" size="small" fullWidth />
              <TextField label="Payload (JSON)" variant="outlined" size="small" fullWidth multiline rows={4} />
            </Stack>
          )}
          {actionDialog.action === 'terminate' && (
            <TextField label="Reason" variant="outlined" size="small" fullWidth sx={{ mt: 2 }} />
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseActionDialog}>Cancel</Button>
          <Button
            onClick={handleConfirmAction}
            variant="contained"
            color={actionDialog.action === 'terminate' ? 'error' : 'primary'}
          >
            Confirm
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};

export default TemporalOpsPage;
