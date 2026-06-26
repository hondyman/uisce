import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  InputAdornment,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  LinearProgress,
  Collapse,
  Stack,
  Divider,
  Alert,
  Card,
  CardContent,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import PauseIcon from '@mui/icons-material/Pause';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import HourglassEmptyIcon from '@mui/icons-material/HourglassEmpty';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import PsychologyIcon from '@mui/icons-material/Psychology';
import PersonIcon from '@mui/icons-material/Person';

// Types
interface WorkflowInstance {
  id: string;
  workflowId: string;
  runId: string;
  processType: string;
  status: 'running' | 'completed' | 'failed' | 'pending' | 'cancelled';
  startTime: string;
  endTime?: string;
  currentStep?: string;
  progress: number;
  assignees: string[];
  metadata: Record<string, any>;
}

interface StepExecution {
  stepId: string;
  stepType: string;
  stepName: string;
  status: 'completed' | 'running' | 'pending' | 'failed';
  startTime?: string;
  endTime?: string;
  assignee?: string;
  action?: string;
  llmInvoked: boolean;
  routing?: {
    type: string;
    resolvedTo: string[];
  };
}

// Status chip colors
const statusColors: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
  completed: 'success',
  running: 'info',
  pending: 'warning',
  failed: 'error',
  cancelled: 'default',
};

// Status icons
const StatusIcon: React.FC<{ status: string }> = ({ status }) => {
  switch (status) {
    case 'completed':
      return <CheckCircleIcon sx={{ color: '#10b981' }} />;
    case 'running':
      return <PlayArrowIcon sx={{ color: '#3b82f6' }} />;
    case 'pending':
      return <HourglassEmptyIcon sx={{ color: '#f59e0b' }} />;
    case 'failed':
      return <ErrorIcon sx={{ color: '#ef4444' }} />;
    default:
      return <PauseIcon sx={{ color: '#6b7280' }} />;
  }
};

// Main Component
const InstanceExplorer: React.FC = () => {
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedInstance, setExpandedInstance] = useState<string | null>(null);
  const [stepExecutions, setStepExecutions] = useState<Record<string, StepExecution[]>>({});

  // Fetch instances
  const fetchInstances = async () => {
    setLoading(true);
    try {
      // TODO: Replace with actual API call
      // const response = await fetch('/api/bp/instances');
      // const data = await response.json();
      
      // Mock data for now
      const mockInstances: WorkflowInstance[] = [
        {
          id: 'inst-001',
          workflowId: 'wf-model-change-001',
          runId: 'run-abc123',
          processType: 'Model Change Approval',
          status: 'running',
          startTime: new Date(Date.now() - 3600000).toISOString(),
          currentStep: 'Advisor Review',
          progress: 60,
          assignees: ['john.advisor@firm.com'],
          metadata: { clientId: 'client-123', portfolioId: 'port-456' },
        },
        {
          id: 'inst-002',
          workflowId: 'wf-onboarding-002',
          runId: 'run-def456',
          processType: 'Client Onboarding',
          status: 'pending',
          startTime: new Date(Date.now() - 7200000).toISOString(),
          currentStep: 'Document Upload',
          progress: 25,
          assignees: ['jane.ops@firm.com'],
          metadata: { clientName: 'Acme Corp' },
        },
        {
          id: 'inst-003',
          workflowId: 'wf-rebalance-003',
          runId: 'run-ghi789',
          processType: 'Portfolio Rebalance',
          status: 'completed',
          startTime: new Date(Date.now() - 86400000).toISOString(),
          endTime: new Date(Date.now() - 82800000).toISOString(),
          progress: 100,
          assignees: [],
          metadata: { portfolioValue: 1500000 },
        },
      ];
      
      setInstances(mockInstances);
    } catch (error) {
      console.error('Failed to fetch instances:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch step executions for an instance
  const fetchStepExecutions = async (instanceId: string) => {
    // TODO: Replace with actual API call
    const mockSteps: StepExecution[] = [
      {
        stepId: 'step-1',
        stepType: 'Interpretation',
        stepName: 'Parse Request',
        status: 'completed',
        startTime: new Date(Date.now() - 3500000).toISOString(),
        endTime: new Date(Date.now() - 3400000).toISOString(),
        llmInvoked: true,
      },
      {
        stepId: 'step-2',
        stepType: 'Classification',
        stepName: 'Risk Classification',
        status: 'completed',
        startTime: new Date(Date.now() - 3400000).toISOString(),
        endTime: new Date(Date.now() - 3300000).toISOString(),
        llmInvoked: true,
      },
      {
        stepId: 'step-3',
        stepType: 'Review',
        stepName: 'Advisor Review',
        status: 'running',
        startTime: new Date(Date.now() - 3300000).toISOString(),
        assignee: 'john.advisor@firm.com',
        llmInvoked: false,
        routing: {
          type: 'DynamicRole',
          resolvedTo: ['john.advisor@firm.com'],
        },
      },
      {
        stepId: 'step-4',
        stepType: 'Approval',
        stepName: 'Compliance Approval',
        status: 'pending',
        llmInvoked: false,
        routing: {
          type: 'StaticGroup',
          resolvedTo: ['compliance_team'],
        },
      },
    ];
    
    setStepExecutions(prev => ({ ...prev, [instanceId]: mockSteps }));
  };

  useEffect(() => {
    fetchInstances();
  }, []);

  const handleExpandInstance = (instanceId: string) => {
    if (expandedInstance === instanceId) {
      setExpandedInstance(null);
    } else {
      setExpandedInstance(instanceId);
      if (!stepExecutions[instanceId]) {
        fetchStepExecutions(instanceId);
      }
    }
  };

  const filteredInstances = instances.filter(inst =>
    inst.processType.toLowerCase().includes(searchQuery.toLowerCase()) ||
    inst.workflowId.toLowerCase().includes(searchQuery.toLowerCase()) ||
    inst.status.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const formatDuration = (start: string, end?: string) => {
    const startTime = new Date(start);
    const endTime = end ? new Date(end) : new Date();
    const diff = endTime.getTime() - startTime.getTime();
    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);
    return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={600}>
          <AccountTreeIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Instance Explorer
        </Typography>
        <Stack direction="row" spacing={2}>
          <TextField
            size="small"
            placeholder="Search instances..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
            sx={{ width: 300 }}
          />
          <Tooltip title="Refresh">
            <IconButton onClick={fetchInstances}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {loading && <LinearProgress sx={{ mb: 2 }} />}

      {/* Instances Table */}
      <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
        <Table>
          <TableHead>
            <TableRow sx={{ bgcolor: 'grey.50' }}>
              <TableCell width={50}></TableCell>
              <TableCell>Process</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Current Step</TableCell>
              <TableCell>Progress</TableCell>
              <TableCell>Duration</TableCell>
              <TableCell>Assignees</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredInstances.map((instance) => (
              <React.Fragment key={instance.id}>
                <TableRow
                  hover
                  sx={{ cursor: 'pointer' }}
                  onClick={() => handleExpandInstance(instance.id)}
                >
                  <TableCell>
                    <IconButton size="small">
                      {expandedInstance === instance.id ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    </IconButton>
                  </TableCell>
                  <TableCell>
                    <Typography fontWeight={500}>{instance.processType}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {instance.workflowId}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      icon={<StatusIcon status={instance.status} />}
                      label={instance.status}
                      size="small"
                      color={statusColors[instance.status]}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{instance.currentStep || '-'}</TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LinearProgress
                        variant="determinate"
                        value={instance.progress}
                        sx={{ width: 80, height: 8, borderRadius: 4 }}
                      />
                      <Typography variant="caption">{instance.progress}%</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>{formatDuration(instance.startTime, instance.endTime)}</TableCell>
                  <TableCell>
                    {instance.assignees.length > 0 ? (
                      instance.assignees.map((a, i) => (
                        <Chip key={i} label={a} size="small" sx={{ mr: 0.5 }} />
                      ))
                    ) : (
                      <Typography variant="caption" color="text.secondary">-</Typography>
                    )}
                  </TableCell>
                </TableRow>

                {/* Expanded Step Details */}
                <TableRow>
                  <TableCell colSpan={7} sx={{ p: 0, border: 0 }}>
                    <Collapse in={expandedInstance === instance.id}>
                      <Box sx={{ p: 3, bgcolor: 'grey.50' }}>
                        <Typography variant="subtitle2" sx={{ mb: 2 }}>
                          Step Execution Timeline
                        </Typography>
                        <Stack spacing={2}>
                          {(stepExecutions[instance.id] || []).map((step, idx) => (
                            <Card key={step.stepId} variant="outlined">
                              <CardContent sx={{ py: 1.5 }}>
                                <Stack direction="row" alignItems="center" spacing={2}>
                                  <Box sx={{ 
                                    width: 32, 
                                    height: 32, 
                                    borderRadius: '50%', 
                                    bgcolor: step.status === 'completed' ? '#dcfce7' : 
                                             step.status === 'running' ? '#dbeafe' : '#f3f4f6',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center'
                                  }}>
                                    <StatusIcon status={step.status} />
                                  </Box>
                                  
                                  <Box sx={{ flex: 1 }}>
                                    <Typography fontWeight={500}>{step.stepName}</Typography>
                                    <Stack direction="row" spacing={1} alignItems="center">
                                      <Chip label={step.stepType} size="small" variant="outlined" />
                                      {step.llmInvoked && (
                                        <Chip 
                                          icon={<PsychologyIcon />} 
                                          label="LLM" 
                                          size="small" 
                                          color="secondary"
                                        />
                                      )}
                                      {step.assignee && (
                                        <Chip 
                                          icon={<PersonIcon />} 
                                          label={step.assignee} 
                                          size="small" 
                                        />
                                      )}
                                    </Stack>
                                  </Box>
                                  
                                  {step.routing && (
                                    <Box>
                                      <Typography variant="caption" color="text.secondary">
                                        Routing: {step.routing.type}
                                      </Typography>
                                    </Box>
                                  )}
                                  
                                  {step.startTime && (
                                    <Typography variant="caption" color="text.secondary">
                                      {formatDuration(step.startTime, step.endTime)}
                                    </Typography>
                                  )}
                                </Stack>
                              </CardContent>
                            </Card>
                          ))}
                        </Stack>
                      </Box>
                    </Collapse>
                  </TableCell>
                </TableRow>
              </React.Fragment>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {filteredInstances.length === 0 && !loading && (
        <Alert severity="info" sx={{ mt: 2 }}>
          No workflow instances found matching your search.
        </Alert>
      )}
    </Box>
  );
};

export default InstanceExplorer;
