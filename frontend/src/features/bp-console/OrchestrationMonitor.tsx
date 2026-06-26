import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  Grid,
} from '@mui/material';
import PlayCircleOutlineIcon from '@mui/icons-material/PlayCircleOutline';
import PauseCircleOutlineIcon from '@mui/icons-material/PauseCircleOutline';
import StopCircleIcon from '@mui/icons-material/StopCircle';
import TimelineIcon from '@mui/icons-material/Timeline';
import SpeedIcon from '@mui/icons-material/Speed';

// Types
interface RunningInstance {
  id: string;
  workflowId: string;
  tenantId: string;
  startTime: string;
  stepName: string;
  stepType: string;
  progress: number; // 0-100
  slaDeadline: string;
  status: 'Running' | 'Paused';
}

const OrchestrationMonitor: React.FC = () => {
  const [instances, setInstances] = useState<RunningInstance[]>([]);
  const [loading, setLoading] = useState(false);

  // Poll for updates
  useEffect(() => {
    const fetchInstances = () => {
      // Mock data
      const now = Date.now();
      const mockInstances: RunningInstance[] = [
        {
          id: 'inst-run-001',
          workflowId: 'Model Change',
          tenantId: 'Acme Corp',
          startTime: new Date(now - 120000).toISOString(),
          stepName: 'Risk Analysis',
          stepType: 'LLM',
          progress: 35,
          slaDeadline: new Date(now + 3600000).toISOString(),
          status: 'Running',
        },
        {
          id: 'inst-run-002',
          workflowId: 'Onboarding',
          tenantId: 'Globex',
          startTime: new Date(now - 450000).toISOString(),
          stepName: 'Document Review',
          stepType: 'Human',
          progress: 60,
          slaDeadline: new Date(now + 7200000).toISOString(),
          status: 'Running',
        },
        {
          id: 'inst-run-003',
          workflowId: 'Trade Settlement',
          tenantId: 'Stark Ind',
          startTime: new Date(now - 20000).toISOString(),
          stepName: 'Validation',
          stepType: 'System',
          progress: 15,
          slaDeadline: new Date(now + 1800000).toISOString(),
          status: 'Running',
        },
        {
          id: 'inst-run-004',
          workflowId: 'Policy Update',
          tenantId: 'Umbrella',
          startTime: new Date(now - 800000).toISOString(),
          stepName: 'Legal Approval',
          stepType: 'Human',
          progress: 85,
          slaDeadline: new Date(now - 100000).toISOString(), // Breached
          status: 'Paused',
        },
      ];
      setInstances(mockInstances);
    };

    fetchInstances();
    const interval = setInterval(fetchInstances, 5000); // Poll every 5s
    return () => clearInterval(interval);
  }, []);

  const getSLAStatus = (deadline: string) => {
    const remaining = new Date(deadline).getTime() - Date.now();
    if (remaining < 0) return { label: 'Breached', color: 'error' as const };
    if (remaining < 3600000) return { label: '< 1h', color: 'warning' as const };
    return { label: 'On Track', color: 'success' as const };
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={600}>
          <SpeedIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Live Orchestration Monitor
        </Typography>
        <Stack direction="row" alignItems="center" spacing={1}>
          <Box sx={{ display: 'flex', alignItems: 'center', mr: 2 }}>
            <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: '#4caf50', mr: 1, animation: 'pulse 1.5s infinite' }} />
            <Typography variant="caption" color="text.secondary">Live Stream</Typography>
          </Box>
        </Stack>
      </Stack>

      <Grid container spacing={3}>
        {/* Left: Active Table */}
        <Grid item xs={12} md={7}>
          <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.50' }}>
                  <TableCell>Instance</TableCell>
                  <TableCell>Current Step</TableCell>
                  <TableCell>Progress</TableCell>
                  <TableCell>SLA</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {instances.map((inst) => {
                  const sla = getSLAStatus(inst.slaDeadline);
                  return (
                    <TableRow key={inst.id} hover>
                      <TableCell>
                        <Stack>
                          <Typography variant="subtitle2">{inst.workflowId}</Typography>
                          <Typography variant="caption" color="text.secondary">{inst.tenantId}</Typography>
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Chip label={inst.stepName} size="small" variant="outlined" />
                        <Typography variant="caption" display="block" color="text.secondary" sx={{ mt: 0.5 }}>
                          {inst.stepType}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <LinearProgress variant="determinate" value={inst.progress} sx={{ flex: 1, mr: 1, borderRadius: 2 }} />
                          <Typography variant="caption">{inst.progress}%</Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label={sla.label} color={sla.color} size="small" />
                      </TableCell>
                      <TableCell>
                        <Stack direction="row">
                          {inst.status === 'Running' ? (
                            <Tooltip title="Pause">
                              <IconButton size="small"><PauseCircleOutlineIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          ) : (
                            <Tooltip title="Resume">
                              <IconButton size="small"><PlayCircleOutlineIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          )}
                          <Tooltip title="Terminate">
                            <IconButton size="small" color="error"><StopCircleIcon fontSize="small" /></IconButton>
                          </Tooltip>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </Grid>

        {/* Right: Gantt / Timeline visualization */}
        <Grid item xs={12} md={5}>
          <Paper sx={{ p: 2, height: '100%' }}>
            <Typography variant="subtitle2" sx={{ mb: 2, display: 'flex', alignItems: 'center' }}>
              <TimelineIcon sx={{ mr: 1, fontSize: 20 }} />
              Active Timelines
            </Typography>
            
            <Stack spacing={3}>
              {instances.map((inst) => (
                <Box key={inst.id}>
                  <Stack direction="row" justifyContent="space-between" sx={{ mb: 0.5 }}>
                    <Typography variant="caption" fontWeight="bold">{inst.workflowId}</Typography>
                    <Typography variant="caption" color="text.secondary">{inst.progress}%</Typography>
                  </Stack>
                  {/* Timeline Bar */}
                   {/* Timeline Bar */}
                  <Box sx={{ height: 24, bgcolor: '#f5f5f5', borderRadius: 1, position: 'relative', overflow: 'hidden' }}>
                    {/* Completed Bar */}
                    <Box 
                      sx={{ 
                        position: 'absolute', 
                        left: 0, 
                        top: 0, 
                        bottom: 0, 
                        width: `${inst.progress}%`,
                        bgcolor: inst.status === 'Paused' ? '#ff9800' : '#2196f3',
                        transition: 'width 1s linear'
                      }} 
                    />
                    {/* Current Step Marker */}
                    <Box 
                      sx={{ 
                        position: 'absolute', 
                        left: `${inst.progress}%`, 
                        top: 0, 
                        bottom: 0,
                        width: 2,
                        bgcolor: '#0d47a1'
                      }} 
                    />
                  </Box>
                  <Typography variant="caption" color="text.secondary">Step: {inst.stepName}</Typography>
                </Box>
              ))}
            </Stack>

            <Box sx={{ mt: 3, p: 2, bgcolor: 'grey.50', borderRadius: 2 }}>
                <Typography variant="caption" color="text.secondary" display="block" gutterBottom>Legend</Typography>
                <Stack direction="row" spacing={2}>
                    <Stack direction="row" alignItems="center" spacing={0.5}>
                        <Box sx={{ width: 12, height: 12, bgcolor: '#2196f3', borderRadius: 0.5 }} />
                        <Typography variant="caption">Running</Typography>
                    </Stack>
                    <Stack direction="row" alignItems="center" spacing={0.5}>
                        <Box sx={{ width: 12, height: 12, bgcolor: '#ff9800', borderRadius: 0.5 }} />
                        <Typography variant="caption">Paused</Typography>
                    </Stack>
                </Stack>
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default OrchestrationMonitor;
