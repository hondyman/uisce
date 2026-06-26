import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  Button,
  Card,
  CardContent,
  Chip,
  Grid,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Divider,
  Alert,
  CircularProgress,
  Tooltip,
  Stack,
} from '@mui/material';
import {
  PlayArrow as StartIcon,
  CheckCircle as CompleteIcon,
  Cancel as RejectIcon,
  History as HistoryIcon,
  Refresh as RefreshIcon,
  Person as PersonIcon,
  Schedule as ScheduleIcon,
  ArrowForward as AdvanceIcon,
  Info as InfoIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';

interface ProcessStep {
  id: string;
  key: string;
  name: string;
  display_name: string;
  step_type: string;
  sequence: number;
  config: any;
  is_required: boolean;
}

interface ProcessInstance {
  id: string;
  process_id: string;
  entity_type: string;
  entity_id: string;
  current_step_id: string | null;
  status: 'pending' | 'in_progress' | 'completed' | 'rejected' | 'cancelled';
  started_at: string;
  completed_at: string | null;
  created_by: string;
  process?: {
    key: string;
    name: string;
    display_name: string;
  };
}

interface StepHistoryItem {
  id: string;
  step_id: string;
  action: string;
  actor: string;
  comments: string;
  created_at: string;
  step?: {
    name: string;
    display_name: string;
  };
}

interface BusinessProcess {
  id: string;
  key: string;
  name: string;
  display_name: string;
  description: string;
  category: string;
  is_system: boolean;
}

interface BusinessProcessViewerProps {
  instance: ProcessInstance;
  steps: ProcessStep[];
  history: StepHistoryItem[];
  onAdvance: (action: string, comments: string) => Promise<void>;
  onRefresh: () => void;
}

const getStepIcon = (stepType: string) => {
  switch (stepType) {
    case 'initiate':
      return '🚀';
    case 'validate':
      return '✅';
    case 'approve':
      return '👤';
    case 'generate':
      return '📄';
    case 'notify':
      return '📧';
    case 'complete':
      return '🎉';
    case 'integration':
      return '🔗';
    case 'calculation':
      return '🔢';
    default:
      return '📋';
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case 'completed':
      return 'success';
    case 'in_progress':
      return 'primary';
    case 'rejected':
      return 'error';
    case 'cancelled':
      return 'default';
    default:
      return 'warning';
  }
};

export const BusinessProcessViewer: React.FC<BusinessProcessViewerProps> = ({
  instance,
  steps,
  history,
  onAdvance,
  onRefresh,
}) => {
  const [advanceDialogOpen, setAdvanceDialogOpen] = useState(false);
  const [selectedAction, setSelectedAction] = useState<string>('');
  const [comments, setComments] = useState('');
  const [loading, setLoading] = useState(false);

  const currentStepIndex = steps.findIndex(s => s.id === instance.current_step_id);
  const currentStep = steps[currentStepIndex];

  const handleAdvance = async () => {
    setLoading(true);
    try {
      await onAdvance(selectedAction, comments);
      setAdvanceDialogOpen(false);
      setComments('');
      onRefresh();
    } finally {
      setLoading(false);
    }
  };

  const isStepCompleted = (stepIndex: number) => {
    if (instance.status === 'completed') return true;
    return stepIndex < currentStepIndex;
  };

  return (
    <Box>
      {/* Header */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Box>
            <Typography variant="h6">
              {instance.process?.display_name || 'Business Process'}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Started {new Date(instance.started_at).toLocaleString()} by {instance.created_by}
            </Typography>
          </Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <Chip
              label={instance.status.toUpperCase()}
              color={getStatusColor(instance.status) as any}
              size="small"
            />
            <IconButton onClick={onRefresh} size="small">
              <RefreshIcon />
            </IconButton>
          </Stack>
        </Stack>
      </Paper>

      {/* Steps */}
      <Grid container spacing={2}>
        <Grid item xs={12} md={8}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Process Steps
            </Typography>
            <Stepper orientation="vertical" activeStep={currentStepIndex}>
              {steps.map((step, index) => (
                <Step key={step.id} completed={isStepCompleted(index)}>
                  <StepLabel
                    icon={
                      <Box
                        sx={{
                          width: 32,
                          height: 32,
                          borderRadius: '50%',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          bgcolor: isStepCompleted(index)
                            ? 'success.main'
                            : index === currentStepIndex
                            ? 'primary.main'
                            : 'grey.300',
                          color: 'white',
                          fontSize: '14px',
                        }}
                      >
                        {getStepIcon(step.step_type)}
                      </Box>
                    }
                  >
                    <Typography variant="subtitle2">{step.display_name}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {step.step_type.replace('_', ' ').toUpperCase()}
                      {step.is_required && (
                        <Chip label="Required" size="small" sx={{ ml: 1 }} />
                      )}
                    </Typography>
                  </StepLabel>
                  <StepContent>
                    {index === currentStepIndex && instance.status === 'in_progress' && (
                      <Box sx={{ mt: 1 }}>
                        <Alert severity="info" sx={{ mb: 2 }}>
                          This step is currently active. Take action to proceed.
                        </Alert>
                        <Stack direction="row" spacing={1}>
                          <Button
                            variant="contained"
                            color="success"
                            size="small"
                            startIcon={<CompleteIcon />}
                            onClick={() => {
                              setSelectedAction('approved');
                              setAdvanceDialogOpen(true);
                            }}
                          >
                            Approve
                          </Button>
                          <Button
                            variant="outlined"
                            color="error"
                            size="small"
                            startIcon={<RejectIcon />}
                            onClick={() => {
                              setSelectedAction('rejected');
                              setAdvanceDialogOpen(true);
                            }}
                          >
                            Reject
                          </Button>
                          {!step.is_required && (
                            <Button
                              variant="text"
                              size="small"
                              onClick={() => {
                                setSelectedAction('skipped');
                                setAdvanceDialogOpen(true);
                              }}
                            >
                              Skip
                            </Button>
                          )}
                        </Stack>
                      </Box>
                    )}
                  </StepContent>
                </Step>
              ))}
            </Stepper>
          </Paper>
        </Grid>

        {/* History */}
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              <HistoryIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
              Activity History
            </Typography>
            <List dense>
              {history.length === 0 ? (
                <ListItem>
                  <ListItemText secondary="No activity yet" />
                </ListItem>
              ) : (
                history.map((item, index) => (
                  <React.Fragment key={item.id}>
                    {index > 0 && <Divider />}
                    <ListItem>
                      <ListItemIcon sx={{ minWidth: 36 }}>
                        {item.action === 'approved' || item.action === 'completed' ? (
                          <CompleteIcon color="success" fontSize="small" />
                        ) : item.action === 'rejected' ? (
                          <RejectIcon color="error" fontSize="small" />
                        ) : (
                          <InfoIcon color="primary" fontSize="small" />
                        )}
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Typography variant="body2">
                            <strong>{item.step?.display_name || 'Step'}</strong>
                            {' - '}
                            {item.action}
                          </Typography>
                        }
                        secondary={
                          <>
                            by {item.actor} •{' '}
                            {new Date(item.created_at).toLocaleString()}
                            {item.comments && (
                              <Typography variant="caption" display="block" color="text.secondary">
                                "{item.comments}"
                              </Typography>
                            )}
                          </>
                        }
                      />
                    </ListItem>
                  </React.Fragment>
                ))
              )}
            </List>
          </Paper>
        </Grid>
      </Grid>

      {/* Advance Dialog */}
      <Dialog open={advanceDialogOpen} onClose={() => setAdvanceDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {selectedAction === 'approved' && 'Approve Step'}
          {selectedAction === 'rejected' && 'Reject Step'}
          {selectedAction === 'skipped' && 'Skip Step'}
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" gutterBottom>
            {selectedAction === 'approved' && 'Approve this step to proceed to the next one.'}
            {selectedAction === 'rejected' && 'Rejecting will stop this process.'}
            {selectedAction === 'skipped' && 'Skip this optional step.'}
          </Typography>
          <TextField
            fullWidth
            multiline
            rows={3}
            label="Comments (optional)"
            value={comments}
            onChange={(e) => setComments(e.target.value)}
            sx={{ mt: 2 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAdvanceDialogOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleAdvance}
            disabled={loading}
            color={selectedAction === 'rejected' ? 'error' : 'primary'}
          >
            {loading ? <CircularProgress size={20} /> : 'Confirm'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

// ============================================================================
// Process Selector Component
// ============================================================================

interface ProcessSelectorProps {
  processes: BusinessProcess[];
  onSelect: (process: BusinessProcess) => void;
  onStart: (processKey: string, entityType: string, entityId: string) => Promise<void>;
}

export const ProcessSelector: React.FC<ProcessSelectorProps> = ({
  processes,
  onSelect,
  onStart,
}) => {
  const [startDialogOpen, setStartDialogOpen] = useState(false);
  const [selectedProcess, setSelectedProcess] = useState<BusinessProcess | null>(null);
  const [entityType, setEntityType] = useState('');
  const [entityId, setEntityId] = useState('');
  const [loading, setLoading] = useState(false);

  const handleStartClick = (process: BusinessProcess) => {
    setSelectedProcess(process);
    setStartDialogOpen(true);
  };

  const handleStart = async () => {
    if (!selectedProcess) return;
    setLoading(true);
    try {
      await onStart(selectedProcess.key, entityType, entityId);
      setStartDialogOpen(false);
      setEntityType('');
      setEntityId('');
    } finally {
      setLoading(false);
    }
  };

  const groupedProcesses = processes.reduce((acc, p) => {
    const category = p.category || 'Other';
    if (!acc[category]) acc[category] = [];
    acc[category].push(p);
    return acc;
  }, {} as Record<string, BusinessProcess[]>);

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Available Business Processes
      </Typography>
      {Object.entries(groupedProcesses).map(([category, procs]) => (
        <Box key={category} sx={{ mb: 3 }}>
          <Typography variant="subtitle2" color="text.secondary" gutterBottom>
            {category.replace('_', ' ').toUpperCase()}
          </Typography>
          <Grid container spacing={2}>
            {procs.map((process) => (
              <Grid item xs={12} sm={6} md={4} key={process.id}>
                <Card
                  sx={{
                    cursor: 'pointer',
                    '&:hover': { boxShadow: 4 },
                    transition: 'box-shadow 0.2s',
                  }}
                  onClick={() => onSelect(process)}
                >
                  <CardContent>
                    <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                      <Box>
                        <Typography variant="subtitle1" fontWeight="bold">
                          {process.display_name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {process.description}
                        </Typography>
                      </Box>
                      {process.is_system && (
                        <Chip label="System" size="small" color="info" />
                      )}
                    </Stack>
                    <Button
                      variant="outlined"
                      size="small"
                      startIcon={<StartIcon />}
                      sx={{ mt: 2 }}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleStartClick(process);
                      }}
                    >
                      Start
                    </Button>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        </Box>
      ))}

      {/* Start Dialog */}
      <Dialog open={startDialogOpen} onClose={() => setStartDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          Start {selectedProcess?.display_name}
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            {selectedProcess?.description}
          </Typography>
          <TextField
            fullWidth
            label="Entity Type"
            value={entityType}
            onChange={(e) => setEntityType(e.target.value)}
            placeholder="e.g., client, portfolio, account"
            sx={{ mt: 2 }}
          />
          <TextField
            fullWidth
            label="Entity ID"
            value={entityId}
            onChange={(e) => setEntityId(e.target.value)}
            placeholder="UUID of the entity"
            sx={{ mt: 2 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setStartDialogOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleStart}
            disabled={loading || !entityType || !entityId}
            startIcon={loading ? <CircularProgress size={20} /> : <StartIcon />}
          >
            Start Process
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

// ============================================================================
// Process Instances List
// ============================================================================

interface ProcessInstancesListProps {
  instances: ProcessInstance[];
  onSelect: (instance: ProcessInstance) => void;
}

export const ProcessInstancesList: React.FC<ProcessInstancesListProps> = ({
  instances,
  onSelect,
}) => {
  if (instances.length === 0) {
    return (
      <Alert severity="info">
        No process instances found. Start a new process to see it here.
      </Alert>
    );
  }

  return (
    <List>
      {instances.map((instance, index) => (
        <React.Fragment key={instance.id}>
          {index > 0 && <Divider />}
          <ListItem
            button
            onClick={() => onSelect(instance)}
            sx={{ '&:hover': { bgcolor: 'action.hover' } }}
          >
            <ListItemIcon>
              {instance.status === 'completed' ? (
                <CompleteIcon color="success" />
              ) : instance.status === 'rejected' ? (
                <RejectIcon color="error" />
              ) : (
                <ScheduleIcon color="primary" />
              )}
            </ListItemIcon>
            <ListItemText
              primary={instance.process?.display_name || 'Process'}
              secondary={
                <>
                  Started {new Date(instance.started_at).toLocaleString()}
                  {instance.completed_at && (
                    <> • Completed {new Date(instance.completed_at).toLocaleString()}</>
                  )}
                </>
              }
            />
            <Chip
              label={instance.status}
              color={getStatusColor(instance.status) as any}
              size="small"
            />
          </ListItem>
        </React.Fragment>
      ))}
    </List>
  );
};

export default BusinessProcessViewer;
