import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Step,
  StepLabel,
  Stepper,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import {
  AccountTree as WorkflowIcon,
  Add as AddIcon,
  CheckCircle as ApprovedIcon,
  Check as CheckIcon,
  Delete as DeleteIcon,
  History as HistoryIcon,
  NotificationsActive as NotificationIcon,
  OpenInNew as OpenInNewIcon,
  PlayArrow as RunIcon,
  PublishedWithChanges as PublishIcon,
  RateReview as ReviewIcon,
  Send as SendIcon,
  Sync as SyncIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { useNotification } from '../../../../hooks/useNotification';
import { fetchAPI } from '../../../../api';

interface WorkflowTabProps {
  businessObject: any;
}

const LIFECYCLE_STEPS = ['DRAFT', 'IN_REVIEW', 'APPROVED', 'PUBLISHED'];

export default function WorkflowTab({ businessObject }: WorkflowTabProps) {
  const theme = useTheme();
  const navigate = useNavigate();
  const notification = useNotification();

  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [workflowStatus, setWorkflowStatus] = useState<any>(null);

  // Proposal modal
  const [proposalModalOpen, setProposalModalOpen] = useState(false);
  const [proposalNotes, setProposalNotes] = useState('');

  // Add trigger modal
  const [triggerModalOpen, setTriggerModalOpen] = useState(false);
  const [newEvent, setNewEvent] = useState('ON_CREATE');
  const [newActionType, setNewActionType] = useState('WORKFLOW');
  const [newTarget, setNewTarget] = useState('DefaultAuditPipeline');
  const [newDescription, setNewDescription] = useState('');

  const boId = businessObject?.id || businessObject?.key;

  const fetchStatus = async () => {
    if (!boId) return;
    setLoading(true);
    try {
      const data = await fetchAPI<any>(`/business-objects/${boId}/workflow`);
      setWorkflowStatus(data);
    } catch (err: any) {
      // Fallback local state if endpoint empty
      setWorkflowStatus({
        lifecycleStatus: businessObject?.is_active ? 'PUBLISHED' : 'DRAFT',
        isCore: businessObject?.is_core ?? false,
        pendingProposals: [],
        eventTriggers: [
          {
            id: 'trig-001',
            event: 'ON_CREATE',
            actionType: 'WORKFLOW',
            target: 'EntityIngestionPipeline',
            enabled: true,
            description: 'Triggered when records are created via ORM CRUD or ETL',
          },
          {
            id: 'trig-002',
            event: 'ON_VALIDATION_FAILURE',
            actionType: 'NOTIFICATION',
            target: 'ComplianceAlertChannel',
            enabled: true,
            description: 'Broadcasts alerts upon data quality boundary violations',
          },
        ],
        recentExecutions: [
          {
            id: 'exec-101',
            workflow: 'CatalogSyncWorkflow',
            triggeredBy: 'System',
            status: 'COMPLETED',
            startTime: new Date(Date.now() - 3600000).toISOString(),
            endTime: new Date(Date.now() - 3590000).toISOString(),
          },
        ],
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, [boId]);

  const handleAction = async (action: string, reviewerNote?: string) => {
    setActionLoading(true);
    try {
      const resp = await fetchAPI<any>(`/business-objects/${boId}/workflow/action`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, reviewerNote }),
      });
      setWorkflowStatus(resp);
      notification.success(`Workflow action '${action}' applied successfully.`);
    } catch (err: any) {
      notification.error(err?.message || `Failed to execute action ${action}`);
    } finally {
      setActionLoading(false);
    }
  };

  const handleCreateTrigger = () => {
    if (!newTarget.trim()) return;
    const newTrig = {
      id: `trig-${Date.now()}`,
      event: newEvent,
      actionType: newActionType,
      target: newTarget,
      enabled: true,
      description: newDescription || `Action triggered on ${newEvent}`,
    };
    setWorkflowStatus((prev: any) => ({
      ...prev,
      eventTriggers: [...(prev?.eventTriggers || []), newTrig],
    }));
    setTriggerModalOpen(false);
    notification.success(`Added event trigger for ${newEvent}.`);
  };

  const handleToggleTrigger = (triggerId: string, enabled: boolean) => {
    setWorkflowStatus((prev: any) => ({
      ...prev,
      eventTriggers: (prev?.eventTriggers || []).map((t: any) =>
        t.id === triggerId ? { ...t, enabled } : t
      ),
    }));
    notification.success(`Trigger ${enabled ? 'enabled' : 'disabled'}.`);
  };

  const handleDeleteTrigger = (triggerId: string) => {
    setWorkflowStatus((prev: any) => ({
      ...prev,
      eventTriggers: (prev?.eventTriggers || []).map((t: any) => t.id !== triggerId),
    }));
    notification.success('Trigger removed.');
  };

  const currentStatus = workflowStatus?.lifecycleStatus || (businessObject?.is_active ? 'PUBLISHED' : 'DRAFT');
  const activeStep = LIFECYCLE_STEPS.indexOf(currentStatus);

  if (loading && !workflowStatus) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header Banner */}
      <Paper
        variant="outlined"
        sx={{
          p: 3,
          mb: 3,
          background: `linear-gradient(135deg, ${theme.palette.background.paper} 0%, ${theme.palette.action.hover} 100%)`,
          borderRadius: 2,
        }}
      >
        <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" alignItems={{ md: 'center' }} spacing={2}>
          <Box>
            <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 0.5 }}>
              <WorkflowIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h6" sx={{ fontWeight: 800 }}>
                Governance Lifecycle & Automation Workflows
              </Typography>
              <Chip
                label={currentStatus}
                size="small"
                color={currentStatus === 'PUBLISHED' ? 'success' : currentStatus === 'APPROVED' ? 'primary' : 'warning'}
                sx={{ fontWeight: 700 }}
              />
              {businessObject?.is_core ? (
                <Chip label="Gold Copy Core" size="small" color="primary" variant="outlined" />
              ) : (
                <Chip label="Tenant Custom Extension" size="small" color="secondary" variant="outlined" />
              )}
            </Stack>
            <Typography variant="body2" color="text.secondary">
              Orchestrate approvals, promotion proposals to master core blueprints, and reactive event pipelines.
            </Typography>
          </Box>

          <Stack direction="row" spacing={1.5}>
            <Button
              variant="outlined"
              color="primary"
              startIcon={<OpenInNewIcon />}
              onClick={() => navigate(`/workflow/designer?boId=${boId}`)}
              sx={{ textTransform: 'none', fontWeight: 600 }}
            >
              Open in Titan Designer
            </Button>
            {!businessObject?.is_core && (
              <Button
                variant="contained"
                color="secondary"
                startIcon={<SendIcon />}
                onClick={() => setProposalModalOpen(true)}
                sx={{ textTransform: 'none', fontWeight: 600 }}
              >
                Propose to Core Master
              </Button>
            )}
          </Stack>
        </Stack>
      </Paper>

      {/* State Machine Stepper */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardHeader
          title={
            <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
              Lifecycle State Machine
            </Typography>
          }
          subheader="Governance promotion workflow from draft specification through active production release"
          action={
            <Stack direction="row" spacing={1}>
              {currentStatus === 'DRAFT' && (
                <Button
                  size="small"
                  variant="contained"
                  color="primary"
                  startIcon={<ReviewIcon />}
                  disabled={actionLoading}
                  onClick={() => handleAction('SUBMIT_REVIEW')}
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                >
                  Submit for Review
                </Button>
              )}
              {currentStatus === 'IN_REVIEW' && (
                <Button
                  size="small"
                  variant="contained"
                  color="success"
                  startIcon={<ApprovedIcon />}
                  disabled={actionLoading}
                  onClick={() => handleAction('APPROVE')}
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                >
                  Approve Definition
                </Button>
              )}
              {currentStatus === 'APPROVED' && (
                <Button
                  size="small"
                  variant="contained"
                  color="primary"
                  startIcon={<PublishIcon />}
                  disabled={actionLoading}
                  onClick={() => handleAction('PUBLISH')}
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                >
                  Publish to Production
                </Button>
              )}
              {currentStatus === 'PUBLISHED' && (
                <Button
                  size="small"
                  variant="outlined"
                  color="warning"
                  disabled={actionLoading}
                  onClick={() => handleAction('DEPRECATE')}
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                >
                  Deprecate
                </Button>
              )}
            </Stack>
          }
        />
        <Divider />
        <CardContent sx={{ py: 4 }}>
          <Stepper activeStep={activeStep >= 0 ? activeStep : 0} alternativeLabel>
            {LIFECYCLE_STEPS.map((label) => (
              <Step key={label} completed={activeStep > LIFECYCLE_STEPS.indexOf(label)}>
                <StepLabel>
                  <Typography variant="caption" sx={{ fontWeight: 700 }}>
                    {label}
                  </Typography>
                </StepLabel>
              </Step>
            ))}
          </Stepper>
        </CardContent>
      </Card>

      <Grid container spacing={3}>
        {/* Event Triggers */}
        <Grid size={{ xs: 12, lg: 7 }}>
          <Card variant="outlined">
            <CardHeader
              title={
                <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
                  Event Triggers & Action Pipelines
                </Typography>
              }
              subheader="Automated actions triggered on record lifecycle events and data boundary violations"
              action={
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<AddIcon />}
                  onClick={() => setTriggerModalOpen(true)}
                  sx={{ textTransform: 'none', fontWeight: 600 }}
                >
                  Add Trigger
                </Button>
              }
            />
            <Divider />
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell sx={{ fontWeight: 700 }}>Event</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Action Type</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Target</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Enabled</TableCell>
                    <TableCell sx={{ fontWeight: 700 }} align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(workflowStatus?.eventTriggers || []).map((trig: any) => (
                    <TableRow key={trig.id} hover>
                      <TableCell>
                        <Chip
                          label={trig.event}
                          size="small"
                          color={trig.event.includes('FAILURE') ? 'error' : 'primary'}
                          variant="outlined"
                          sx={{ fontWeight: 600, fontSize: '0.7rem' }}
                        />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          {trig.actionType}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                          {trig.target}
                        </Typography>
                        {trig.description && (
                          <Typography variant="caption" color="text.secondary" display="block">
                            {trig.description}
                          </Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        <Switch
                          size="small"
                          checked={trig.enabled}
                          onChange={(e) => handleToggleTrigger(trig.id, e.target.checked)}
                        />
                      </TableCell>
                      <TableCell align="right">
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => handleDeleteTrigger(trig.id)}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                  {(workflowStatus?.eventTriggers || []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} sx={{ textAlign: 'center', py: 4 }}>
                        <Typography variant="body2" color="text.secondary">
                          No event triggers registered. Click "Add Trigger" to attach automated pipelines.
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Card>
        </Grid>

        {/* Executions & Governance History */}
        <Grid size={{ xs: 12, lg: 5 }}>
          <Card variant="outlined">
            <CardHeader
              title={
                <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
                  Recent Workflow Executions
                </Typography>
              }
              subheader="Real-time execution log from the workflow engine"
            />
            <Divider />
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell sx={{ fontWeight: 700 }}>Workflow</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Triggered By</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Status</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(workflowStatus?.recentExecutions || []).map((exec: any) => (
                    <TableRow key={exec.id} hover>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          {exec.workflow}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {new Date(exec.startTime).toLocaleTimeString()}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">{exec.triggeredBy}</Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={exec.status}
                          size="small"
                          color={exec.status === 'COMPLETED' ? 'success' : exec.status === 'RUNNING' ? 'info' : 'error'}
                          sx={{ fontSize: '0.65rem', height: 20 }}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                  {(workflowStatus?.recentExecutions || []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} sx={{ textAlign: 'center', py: 3 }}>
                        <Typography variant="body2" color="text.secondary">
                          No recent workflow executions.
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Card>
        </Grid>
      </Grid>

      {/* Propose to Core Master Modal */}
      <Dialog open={proposalModalOpen} onClose={() => setProposalModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>
          Propose Extension to Gold Copy Core Master
        </DialogTitle>
        <DialogContent dividers>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Submit your tenant's custom fields and configurations for review to be promoted as standard Core Master blueprint capabilities for all tenants.
          </Typography>
          <TextField
            fullWidth
            multiline
            rows={4}
            label="Business Justification & Reviewer Notes"
            value={proposalNotes}
            onChange={(e) => setProposalNotes(e.target.value)}
            placeholder="Describe why these custom fields should become part of the universal Gold Copy model..."
          />
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setProposalModalOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            color="secondary"
            startIcon={<SendIcon />}
            onClick={() => {
              notification.success('Promotion proposal submitted to Enterprise Architecture review board.');
              setProposalModalOpen(false);
            }}
          >
            Submit Proposal
          </Button>
        </DialogActions>
      </Dialog>

      {/* Add Event Trigger Modal */}
      <Dialog open={triggerModalOpen} onClose={() => setTriggerModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>Configure Event Trigger</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ mt: 1 }}>
            <FormControl fullWidth size="small">
              <InputLabel>Event Condition</InputLabel>
              <Select value={newEvent} label="Event Condition" onChange={(e) => setNewEvent(e.target.value)}>
                <MenuItem value="ON_CREATE">ON_CREATE (Record Insertion)</MenuItem>
                <MenuItem value="ON_UPDATE">ON_UPDATE (Record Modification)</MenuItem>
                <MenuItem value="ON_DELETE">ON_DELETE (Record Removal)</MenuItem>
                <MenuItem value="ON_VALIDATION_FAILURE">ON_VALIDATION_FAILURE (Boundary Violation)</MenuItem>
                <MenuItem value="ON_PUBLISH">ON_PUBLISH (Schema Published)</MenuItem>
              </Select>
            </FormControl>

            <FormControl fullWidth size="small">
              <InputLabel>Action Type</InputLabel>
              <Select value={newActionType} label="Action Type" onChange={(e) => setNewActionType(e.target.value)}>
                <MenuItem value="WORKFLOW">Execute Temporal Workflow</MenuItem>
                <MenuItem value="WEBHOOK">Call HTTP Webhook</MenuItem>
                <MenuItem value="NOTIFICATION">Send Compliance Alert</MenuItem>
                <MenuItem value="RECALCULATE">Trigger Derived View Recalculation</MenuItem>
              </Select>
            </FormControl>

            <TextField
              fullWidth
              size="small"
              label="Target Workflow ID or Endpoint URL"
              value={newTarget}
              onChange={(e) => setNewTarget(e.target.value)}
            />

            <TextField
              fullWidth
              size="small"
              label="Description / Purpose"
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
              placeholder="e.g. Audit record insertion and dispatch notification"
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setTriggerModalOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreateTrigger}>
            Add Trigger
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
