import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
  Modal,
  TextField,
  Tabs,
  Tab,
  Badge,
  Avatar,
  Chip,
  Grid,
  Divider,
  Box,
  CircularProgress,
  Stepper,
  Step,
  StepLabel,
  Typography,
  Paper,
} from '@mui/material';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
} from '@mui/lab';
import {
  Add as PlusOutlined,
  Edit as EditOutlined,
  Delete as DeleteOutlined,
  History as HistoryOutlined,
  Replay as RollbackOutlined,
  Download as DownloadOutlined,
  Upload as UploadOutlined,
  PlayCircleOutline as _PlayCircleOutlined,
  // ClockCircleOutlined not available in current MUI icon set; removed to avoid build errors
  Person as UserOutlined,
  Article as _FileTextOutlined,
  Menu as _BarsOutlined,
  Group as _TeamOutlined,
  Difference as _DiffOutlined,
  LockOpen as _UnlockOutlined,
} from '@mui/icons-material';
import { useSnackbar } from 'notistack';
import { useNotification } from '../../hooks/useNotification';

import { devError } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';

// Use MUI Timeline from @mui/lab directly (types are available now that @mui/lab is installed)
  import TextPromptDialog from '../../components/TextPromptDialog';

// void Tree;
// void Collapse;
// void DatePicker;
// void Progress;

interface TimeoutTrigger {
  id?: string;
  workflow_name: string;
  step_name: string;
  due_hours: number;
  actions: TimeoutAction[];
  is_active: boolean;
  version?: number;
  status?: 'draft' | 'active' | 'deprecated';
  created_by?: string;
  created_at?: string;
  modified_by?: string;
  modified_at?: string;
  description?: string;
  tags?: string[];
  metadata?: Record<string, any>;
}

interface TimeoutAction {
  percent: number;
  type: 'escalate' | 'notify' | 'log' | 'cancel';
  target: string;
  message: string;
}

interface TriggerVersion {
  version: number;
  trigger: TimeoutTrigger;
  changes: string[];
  change_summary: string;
  timestamp: string;
  author: string;
  author_email: string;
  author_name: string;
}

interface ApprovalRequest {
  id: string;
  trigger_id: string;
  version: number;
  status: 'pending' | 'approved' | 'rejected';
  requested_by: string;
  requested_at: string;
  approvers: Array<{ name: string; status: string; timestamp?: string }>;
  rejection_reason?: string;
}

interface Comment {
  id: string;
  trigger_id: string;
  content: string;
  author_email: string;
  author_name: string;
  timestamp: string;
  mentioned_users?: string[];
}

interface AnalyticsData {
  trigger_id: string;
  total_invocations: number;
  successful_invocations: number;
  failed_invocations: number;
  success_rate: number;
  avg_execution_time_ms: number;
  min_execution_time_ms: number;
  max_execution_time_ms: number;
  last_30_days_invocations: number;
  last_30_days_success_rate: number;
  measured_at: string;
}

const WORKFLOW_STEPS = {
  HireEmployee: ['DataEntry', 'ManagerApproval', 'HRReview', 'Onboarding'],
  OrderApproval: ['DataEntry', 'CreditApproval', 'ExecutiveReview'],
  InvoiceProcessing: ['DataEntry', 'ApprovalQueue', 'PaymentApproval'],
};

const _ESCALATION_TARGETS = {
  escalate: ['hr_director', 'finance_director', 'accounting_manager', 'operations_lead'],
  notify: ['assignee', 'manager', 'hr', 'finance'],
  log: ['audit', 'compliance', 'system'],
  cancel: ['auto', 'manual_review'],
};

const WorkflowTimeoutTriggersPage: React.FC = () => {
  const { enqueueSnackbar } = useSnackbar();
  // const [_form] = Form.useForm(); // antd form instance
  const [triggers, setTriggers] = useState<TimeoutTrigger[]>([]);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<TimeoutTrigger | null>(null);
  const [selectedTrigger, setSelectedTrigger] = useState<TimeoutTrigger | null>(null);
  const [actions, setActions] = useState<TimeoutAction[]>([
    { percent: 80, type: 'notify', target: 'assignee', message: 'Approval overdue - notification sent' },
    { percent: 100, type: 'escalate', target: 'hr_director', message: 'Escalated to HR Director' },
  ]);
  
  // Versioning state
  const [versions, setVersions] = useState<TriggerVersion[]>([]);
  // version history UI toggle currently unused; prefix to silence lint
  const [_showVersionHistory, _setShowVersionHistory] = useState(false);
  
  // Collaboration state
  const [comments, setComments] = useState<Comment[]>([]);
  const [newComment, setNewComment] = useState('');
  const [approvals, setApprovals] = useState<ApprovalRequest[]>([]);
  const [analytics, setAnalytics] = useState<AnalyticsData | null>(null);
  const [detailTab, setDetailTab] = useState('overview');

  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectApprovalId, setRejectApprovalId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  // Fetch existing triggers
  useEffect(() => {
    fetchTriggers();
  }, []);

  // Fetch detailed data when trigger is selected
  useEffect(() => {
    if (selectedTrigger?.id) {
      fetchVersionHistory(selectedTrigger.id);
      fetchComments(selectedTrigger.id);
      fetchApprovals(selectedTrigger.id);
      fetchAnalytics(selectedTrigger.id);
    }
  }, [selectedTrigger]);

  const getTenantHeaders = () => {
    const tenantData = localStorage.getItem('selected_tenant');
    const datasourceData = localStorage.getItem('selected_datasource');
    
    if (!tenantData || !datasourceData) {
      enqueueSnackbar('Please select a tenant and datasource', { variant: 'error' });
      return null;
    }

    const tenant = JSON.parse(tenantData);
    const datasource = JSON.parse(datasourceData);

    return {
      'X-Tenant-ID': tenant.id,
      'X-Tenant-Datasource-ID': datasource.id,
      'X-Tenant-Region': getSelectedRegion(),
      'Content-Type': 'application/json',
    };
  };

  const fetchTriggers = async () => {
    const headers = getTenantHeaders();
    if (!headers) return;

    setLoading(true);
    try {
      const response = await fetch('/api/workflow-timeout-triggers', { headers });
      const data = await response.json();
      setTriggers(data || []);
    } catch (error) {
      enqueueSnackbar('Failed to fetch triggers', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const fetchVersionHistory = async (triggerId: string) => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(`/api/workflow-timeout-triggers/${triggerId}/versions`, { headers });
      const data = await response.json();
      setVersions(data || []);
    } catch (error) {
      devError('Failed to fetch versions:', error);
    }
  };

  const fetchComments = async (triggerId: string) => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(`/api/workflow-timeout-triggers/${triggerId}/comments`, { headers });
      const data = await response.json();
      setComments(data || []);
    } catch (error) {
      devError('Failed to fetch comments:', error);
    }
  };

  const fetchApprovals = async (triggerId: string) => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(`/api/workflow-timeout-triggers/${triggerId}/approvals`, { headers });
      const data = await response.json();
      setApprovals(data || []);
    } catch (error) {
      devError('Failed to fetch approvals:', error);
    }
  };

  const fetchAnalytics = async (triggerId: string) => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(`/api/workflow-timeout-triggers/${triggerId}/analytics`, { headers });
      const data = await response.json();
      setAnalytics(data);
    } catch (error) {
      devError('Failed to fetch analytics:', error);
    }
  };

  const handleSave = async () => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const trigger = {
        ...editing || {},
        actions,
      };

      const method = trigger.id ? 'PUT' : 'POST';
      const url = trigger.id 
        ? `/api/workflow-timeout-triggers/${trigger.id}`
        : '/api/workflow-timeout-triggers';

      const response = await fetch(url, {
        method,
        headers,
        body: JSON.stringify(trigger),
      });

      if (response.ok) {
        enqueueSnackbar(trigger.id ? 'Trigger updated' : 'Trigger created', { variant: 'success' });
        setEditing(null);
        setActions([]);
        fetchTriggers();
      } else {
        enqueueSnackbar('Failed to save trigger', { variant: 'error' });
      }
    } catch (error) {
      enqueueSnackbar('Failed to save trigger', { variant: 'error' });
    }
  };

  const handleDelete = async (triggerId: string) => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(`/api/workflow-timeout-triggers/${triggerId}`, {
        method: 'DELETE',
        headers,
      });

      if (response.ok) {
        enqueueSnackbar('Trigger deleted', { variant: 'success' });
        fetchTriggers();
      } else {
enqueueSnackbar('Failed to delete trigger', { variant: 'error' });
      }
    } catch (error) {
      enqueueSnackbar('Failed to delete trigger', { variant: 'error' });
    }
  };

  const handleAddComment = async () => {
    if (!selectedTrigger?.id || !newComment.trim()) return;

    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(`/api/workflow-timeout-triggers/${selectedTrigger.id}/comments`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ content: newComment }),
      });

      if (response.ok) {
        enqueueSnackbar('Comment added', { variant: 'success' });
        setNewComment('');
        fetchComments(selectedTrigger.id);
      }
    } catch (error) {
      enqueueSnackbar('Failed to add comment', { variant: 'error' });
    }
  };

  const handleRequestApproval = async () => {
    if (!selectedTrigger?.id) return;

    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(
        `/api/workflow-timeout-triggers/${selectedTrigger.id}/approvals/request`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify({ version: selectedTrigger.version }),
        }
      );

      if (response.ok) {
        enqueueSnackbar('Approval request sent', { variant: 'success' });
        fetchApprovals(selectedTrigger.id);
      }
    } catch (error) {
      enqueueSnackbar('Failed to request approval', { variant: 'error' });
    }
  };

  const handleRestoreVersion = async (version: number) => {
    if (!selectedTrigger?.id) return;

    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(
        `/api/workflow-timeout-triggers/${selectedTrigger.id}/versions/${version}/restore`,
        {
          method: 'POST',
          headers,
        }
      );

      if (response.ok) {
        enqueueSnackbar(`Restored to version ${version}`, { variant: 'success' });
        fetchTriggers();
        fetchVersionHistory(selectedTrigger.id);
      }
    } catch (error) {
      enqueueSnackbar('Failed to restore version', { variant: 'error' });
    }
  };

  const handleApproveChange = async (approvalId: string) => {
    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(
        `/api/workflow-timeout-triggers/approvals/${approvalId}/approve`,
        {
          method: 'POST',
          headers,
        }
      );

      if (response.ok) {
        enqueueSnackbar('Change approved', { variant: 'success' });
        if (selectedTrigger?.id) {
          fetchApprovals(selectedTrigger.id);
        }
      }
    } catch (error) {
      enqueueSnackbar('Failed to approve change', { variant: 'error' });
    }
  };

  const notification = useNotification();

  const handleRejectChange = async (approvalId: string) => {
    // Open the in-app text prompt dialog instead of using window.prompt
    setRejectDialogOpen(true);
    setRejectApprovalId(approvalId);
    setRejectReason('');
    return;
  };

  const handleRejectSubmit = async (reason: string) => {
    if (!rejectApprovalId) return;

    const headers = getTenantHeaders();
    if (!headers) return;

    try {
      const response = await fetch(
        `/api/workflow-timeout-triggers/approvals/${rejectApprovalId}/reject`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify({ reason }),
        }
      );

      if (response.ok) {
        notification.success('Change rejected');
        if (selectedTrigger?.id) {
          fetchApprovals(selectedTrigger.id);
        }
      }
    } catch (error) {
      notification.error('Failed to reject change');
    } finally {
      setRejectDialogOpen(false);
      setRejectApprovalId(null);
      setRejectReason('');
    }
  };

  

  

  return (
    <div >
      <Card>
        <CardHeader
          title="Workflow Timeout Triggers - Enterprise Edition"
          action={
            <Stack direction="row" spacing={2}>
              <Button
                variant="contained"
                startIcon={<PlusOutlined />}
                onClick={() => {
                  setEditing({} as TimeoutTrigger);
                  setActions([]);
                }}
              >
                New Trigger
              </Button>
              <Button variant="outlined" startIcon={<DownloadOutlined />}>
                Export
              </Button>
              <Button variant="outlined" startIcon={<UploadOutlined />}>
                Import
              </Button>
            </Stack>
          }
        />
        <CardContent>
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Workflow</TableCell>
                <TableCell>Step</TableCell>
                <TableCell>Due Hours</TableCell>
                <TableCell>Version</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={6} align="center">
                    <CircularProgress />
                  </TableCell>
                </TableRow>
              ) : (
                triggers.map((record) => (
                  <TableRow key={record.id}>
                    <TableCell>{record.workflow_name}</TableCell>
                    <TableCell>{record.step_name}</TableCell>
                    <TableCell>{record.due_hours}</TableCell>
                    <TableCell>
                      <Badge badgeContent={record.version} color="primary" />
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={record.status}
                        color={record.status === 'active' ? 'success' : 'error'}
                      />
                    </TableCell>
                    <TableCell>
                      <Stack direction="row" spacing={1}>
                        <Button
                          variant="contained"
                          size="small"
                          onClick={() => {
                            setSelectedTrigger(record);
                            setEditing(record);
                            setActions(record.actions || []);
                          }}
                        >
                          <EditOutlined /> Edit
                        </Button>
                        <Button
                          size="small"
                          onClick={() => setSelectedTrigger(record)}
                        >
                          <HistoryOutlined /> History
                        </Button>
                        <Button
                          color="error"
                          size="small"
                          onClick={() => handleDelete(record.id!)}
                        >
                          <DeleteOutlined /> Delete
                        </Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </CardContent>
    </Card>

      {editing && (
        <Modal
        open={!!editing}
        onClose={() => {
          setEditing(null);
          setActions([]);
        }}
      >
        <Box sx={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 800,
          bgcolor: 'background.paper',
          boxShadow: 24,
          p: 4,
        }}>
          <Typography variant="h6" component="h2">
            {editing.id ? 'Edit Trigger' : 'New Trigger'}
          </Typography>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <FormControl fullWidth>
              <InputLabel>Workflow Name</InputLabel>
              <Select
                value={editing.workflow_name}
                onChange={(e) =>
                  setEditing({ ...editing, workflow_name: e.target.value })
                }
              >
                {Object.keys(WORKFLOW_STEPS).map((w) => (
                  <MenuItem key={w} value={w}>{w}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth>
              <InputLabel>Step Name</InputLabel>
              <Select
                value={editing.step_name}
                onChange={(e) =>
                  setEditing({ ...editing, step_name: e.target.value })
                }
              >
                {editing.workflow_name
                  ? WORKFLOW_STEPS[editing.workflow_name as keyof typeof WORKFLOW_STEPS].map(
                      (s) => (
                        <MenuItem key={s} value={s}>{s}</MenuItem>
                      )
                    )
                  : []}
              </Select>
            </FormControl>
            <TextField
              label="Due Hours"
              type="number"
              value={editing.due_hours}
              onChange={(e) =>
                setEditing({
                  ...editing,
                  due_hours: parseInt(e.target.value) || 0,
                })
              }
              InputProps={{ inputProps: { min: 1, max: 999 } }}
            />
            <TextField
              label="Description"
              multiline
              rows={3}
              value={editing.description}
              onChange={(e) =>
                setEditing({ ...editing, description: e.target.value })
              }
              placeholder="Optional description"
            />
          </Stack>
          <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
            <Button variant="contained" onClick={handleSave}>Save</Button>
            <Button onClick={() => {
              setEditing(null);
              setActions([]);
            }}>Cancel</Button>
          </Stack>
        </Box>
      </Modal>
      )}

      {selectedTrigger && (
        <Modal
        open={!!selectedTrigger}
        onClose={() => setSelectedTrigger(null)}
      >
        <Box sx={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 900,
          bgcolor: 'background.paper',
          boxShadow: 24,
          p: 4,
        }}>
          <Typography variant="h6" component="h2">
            {selectedTrigger.workflow_name ? `${selectedTrigger.workflow_name} - ${selectedTrigger.step_name}`: ''}
          </Typography>
          <Tabs value={detailTab} onChange={(e, newValue) => setDetailTab(newValue)}>
            <Tab label="Overview" value="overview" />
            <Tab label="Version History" value="versions" />
            <Tab label="Approvals" value="approvals" />
            <Tab label="Comments" value="comments" />
            <Tab label="Analytics" value="analytics" />
          </Tabs>

          {detailTab === 'overview' && (
            <div>
              <Card>
                <CardHeader title="Trigger Details" />
                <CardContent>
                  {selectedTrigger && (
                    <Stack spacing={1}>
                      <Typography><strong>Workflow:</strong> {selectedTrigger.workflow_name}</Typography>
                      <Typography><strong>Step:</strong> {selectedTrigger.step_name}</Typography>
                      <Typography><strong>Due Hours:</strong> {selectedTrigger.due_hours}</Typography>
                      <Typography><strong>Status:</strong> <Chip label={selectedTrigger.status} color="primary" /></Typography>
                      <Typography><strong>Version:</strong> v{selectedTrigger.version}</Typography>
                      <Typography><strong>Created By:</strong> {selectedTrigger.created_by}</Typography>
                      <Typography><strong>Created At:</strong> {new Date(selectedTrigger.created_at || '').toLocaleString()}</Typography>
                      {selectedTrigger.description && (
                        <Typography><strong>Description:</strong> {selectedTrigger.description}</Typography>
                      )}
                    </Stack>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {detailTab === 'versions' && (
            <div>
              <Card>
                <CardHeader title={`Version History (${versions.length} versions)`} />
                <CardContent>
                  {versions.length > 0 ? (
                    <Timeline>
                      {versions.map((v, idx) => (
                        <TimelineItem
                          key={v.version}
                        >
                          <TimelineSeparator>
                            <TimelineDot color={idx === 0 ? 'success' : 'primary'} />
                            {idx < versions.length - 1 && <TimelineConnector />}
                          </TimelineSeparator>
                          <TimelineContent>
                            <Card>
                              <CardHeader
                                title={
                                  <Typography variant="h6">
                                    <strong>Version {v.version}</strong>
                                    {idx === 0 && <Chip label="CURRENT" color="success" style={{ marginLeft: '8px' }} />}
                                  </Typography>
                                }
                                action={
                                  idx !== 0 && (
                                    <Button
                                      variant="contained"
                                      size="small"
                                      onClick={() => handleRestoreVersion(v.version)}
                                    >
                                      <RollbackOutlined /> Restore
                                    </Button>
                                  )
                                }
                              />
                              <CardContent>
                                <Stack direction="column" spacing={1}>
                                  <Typography><strong>Author:</strong> {v.author_name} ({v.author_email})</Typography>
                                  <Typography><strong>Date:</strong> {new Date(v.timestamp).toLocaleString()}</Typography>
                                  {v.changes.length > 0 && (
                                    <div>
                                      <Typography><strong>Changes:</strong></Typography>
                                      <ul>
                                        {v.changes.map((change, i) => (
                                          <li key={i}>{change}</li>
                                        ))}
                                      </ul>
                                    </div>
                                  )}
                                </Stack>
                              </CardContent>
                            </Card>
                          </TimelineContent>
                        </TimelineItem>
                      ))}
                      </Timeline>
                  ) : (
                    <Box>No version history</Box>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {detailTab === 'approvals' && (
            <div>
              <Card>
                <CardHeader title="Approval Requests" />
                <CardContent>
                  <Stack direction="column" spacing={2} style={{ width: '100%', marginBottom: '16px' }}>
                    <Button 
                      variant="contained"
                      onClick={handleRequestApproval}
                    >
                      Request Approval
                    </Button>
                  </Stack>

                  {approvals.length > 0 ? (
                    approvals.map((approval) => (
                      <Card key={approval.id} style={{ marginBottom: '16px' }}>
                        <CardContent>
                          <Grid container spacing={2}>
                            <Grid item xs={12} sm={6}>
                              <Typography variant="h6">Status</Typography>
                              <Typography variant="body1" color={approval.status === 'pending' ? 'orange' : 'green'}>{approval.status}</Typography>
                            </Grid>
                            <Grid item xs={12} sm={6}>
                              <Typography variant="h6">Requested By</Typography>
                              <Typography variant="body1">{approval.requested_by}</Typography>
                            </Grid>
                          </Grid>

                          <Divider />

                          <div>
                            <Typography><strong>Approvers:</strong></Typography>
                            <Stepper>
                              {approval.approvers.map((approver) => (
                                <Step key={approver.name}>
                                  <StepLabel>{approver.name}</StepLabel>
                                </Step>
                              ))}
                            </Stepper>
                          </div>

                          {approval.status === 'pending' && (
                            <Stack direction="row" spacing={2} style={{ marginTop: '16px' }}>
                              <Button 
                                variant="contained"
                                onClick={() => handleApproveChange(approval.id)}
                              >
                                Approve
                              </Button>
                              <Button 
                                color="error"
                                onClick={() => handleRejectChange(approval.id)}
                              >
                                Reject
                              </Button>
                            </Stack>
                          )}
                        </CardContent>
                      </Card>
                    ))
                  ) : (
                    <Box>No approval requests</Box>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {detailTab === 'comments' && (
            <div>
              <Card>
                <CardHeader title="Collaboration" />
                <CardContent>
                  <Stack direction="column" spacing={2} style={{ width: '100%' }}>
                    <TextField
                      multiline
                      rows={4}
                      placeholder="Add a comment..."
                      value={newComment}
                      onChange={(e) => setNewComment(e.target.value)}
                    />
                    <Button 
                      variant="contained"
                      onClick={handleAddComment}
                    >
                      Post Comment
                    </Button>
                  </Stack>

                  <Divider />

                  {comments.length > 0 ? (
                    comments.map((comment) => (
                      <Card key={comment.id}>
                        <CardContent>
                          <Stack direction="column" spacing={1} style={{ width: '100%' }}>
                            <Stack direction="row" spacing={1} alignItems="center">
                              <Avatar><UserOutlined /></Avatar>
                              <Typography variant="body1"><strong>{comment.author_name}</strong></Typography>
                              <Typography variant="body2" color="text.secondary">({comment.author_email})</Typography>
                            </Stack>
                            <Typography variant="body1">{comment.content}</Typography>
                            <Typography variant="caption" color="text.secondary">{new Date(comment.timestamp).toLocaleString()}</Typography>
                          </Stack>
                        </CardContent>
                      </Card>
                    ))
                  ) : (
                    <Box>No comments yet</Box>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {detailTab === 'analytics' && (
            <div>
              <Card>
                <CardHeader title="Performance Analytics" />
                <CardContent>
                  {analytics ? (
                    <Grid container spacing={2}>
                      <Grid item xs={6}>
                        <Typography variant="h6">Total Invocations</Typography>
                        <Typography variant="body1">{analytics.total_invocations}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="h6">Success Rate</Typography>
                        <Typography variant="body1">{analytics.success_rate.toFixed(1)}%</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="h6">Avg Execution Time</Typography>
                        <Typography variant="body1">{analytics.avg_execution_time_ms.toFixed(2)}ms</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="h6">Last 30 Days</Typography>
                        <Typography variant="body1">{analytics.last_30_days_invocations}</Typography>
                      </Grid>
                    </Grid>
                  ) : (
                    <Box>No analytics data yet</Box>
                  )}
                </CardContent>
              </Card>
            </div>
          )}
        </Box>
      </Modal>
      )}
    <TextPromptDialog
      open={rejectDialogOpen}
      title="Rejection reason"
      label="Reason"
      defaultValue={rejectReason}
      onClose={() => { setRejectDialogOpen(false); setRejectApprovalId(null); }}
      onSubmit={(v: string) => handleRejectSubmit(v)}
    />
    </div>
  );
};

export default WorkflowTimeoutTriggersPage;