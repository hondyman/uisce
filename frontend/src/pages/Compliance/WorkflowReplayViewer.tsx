import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent,
  Card,
  CardContent,
  Chip,
  Alert,
  Grid,
  Divider,
} from '@mui/material';
import {
  PlayArrow as PlayIcon,
  Search as SearchIcon,
  CheckCircle as CompleteIcon,
  Schedule as ScheduleIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';

interface WorkflowEvent {
  event_id: number;
  event_type: string;
  timestamp: string;
  attributes: Record<string, any>;
  actor_id?: string;
  decision_made?: string;
}

interface WorkflowExecution {
  workflow_id: string;
  run_id: string;
  workflow_type: string;
  start_time: string;
  close_time?: string;
  status: string;
  execution_time_ms: number;
  events: WorkflowEvent[];
  inputs: Record<string, any>;
  result?: Record<string, any>;
  ai_model_versions: string[];
  policy_versions: string[];
}

export const WorkflowReplayViewer: React.FC = () => {
  const [workflowId, setWorkflowId] = useState('');
  const [execution, setExecution] = useState<WorkflowExecution | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleReplay = async () => {
    if (!workflowId) return;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/workflows/${workflowId}/replay`);
      if (!response.ok) throw new Error('Failed to replay workflow');

      const data = await response.json();
      setExecution(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const getEventIcon = (eventType: string) => {
    if (eventType.includes('Completed')) return <CompleteIcon color="success" />;
    if (eventType.includes('Failed')) return <ErrorIcon color="error" />;
    if (eventType.includes('Scheduled')) return <ScheduleIcon color="primary" />;
    return <PlayIcon />;
  };

  const getStatusColor = (status: string): 'success' | 'error' | 'warning' | 'info' => {
    if (status === 'COMPLETED') return 'success';
    if (status === 'FAILED') return 'error';
    if (status === 'RUNNING') return 'info';
    return 'warning';
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Workflow Replay & Forensic Analysis
      </Typography>

      <Alert severity="info" sx={{ mb: 3 }}>
        Replay any workflow execution for regulatory compliance (SEC Rule 204-2, FINRA).
        View complete event history, AI decisions, and policy versions used.
      </Alert>

      {/* Search */}
      <Paper elevation={2} sx={{ p: 3, mb: 3 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={8}>
            <TextField
              fullWidth
              label="Workflow ID"
              value={workflowId}
              onChange={(e) => setWorkflowId(e.target.value)}
              placeholder="Enter workflow ID to replay"
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <Button
              fullWidth
              variant="contained"
              size="large"
              startIcon={<SearchIcon />}
              onClick={handleReplay}
              disabled={!workflowId || loading}
            >
              {loading ? 'Replaying...' : 'Replay Workflow'}
            </Button>
          </Grid>
        </Grid>
      </Paper>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {execution && (
        <>
          {/* Execution Summary */}
          <Paper elevation={2} sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" gutterBottom>
              Execution Summary
            </Typography>

            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <Typography variant="body2" color="text.secondary">
                  Workflow ID
                </Typography>
                <Typography variant="body1" fontWeight="medium">
                  {execution.workflow_id}
                </Typography>
              </Grid>

              <Grid item xs={12} md={6}>
                <Typography variant="body2" color="text.secondary">
                  Run ID
                </Typography>
                <Typography variant="body1" fontWeight="medium">
                  {execution.run_id}
                </Typography>
              </Grid>

              <Grid item xs={12} md={4}>
                <Typography variant="body2" color="text.secondary">
                  Workflow Type
                </Typography>
                <Typography variant="body1">{execution.workflow_type}</Typography>
              </Grid>

              <Grid item xs={12} md={4}>
                <Typography variant="body2" color="text.secondary">
                  Status
                </Typography>
                <Chip label={execution.status} color={getStatusColor(execution.status)} />
              </Grid>

              <Grid item xs={12} md={4}>
                <Typography variant="body2" color="text.secondary">
                  Execution Time
                </Typography>
                <Typography variant="body1">
                  {(execution.execution_time_ms / 1000).toFixed(2)}s
                </Typography>
              </Grid>

              <Grid item xs={12} md={6}>
                <Typography variant="body2" color="text.secondary">
                  Start Time
                </Typography>
                <Typography variant="body1">
                  {format(new Date(execution.start_time), 'PPpp')}
                </Typography>
              </Grid>

              {execution.close_time && (
                <Grid item xs={12} md={6}>
                  <Typography variant="body2" color="text.secondary">
                    Close Time
                  </Typography>
                  <Typography variant="body1">
                    {format(new Date(execution.close_time), 'PPpp')}
                  </Typography>
                </Grid>
              )}

              {execution.ai_model_versions.length > 0 && (
                <Grid item xs={12}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    AI Model Versions Used
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    {execution.ai_model_versions.map((version, i) => (
                      <Chip key={i} label={version} size="small" variant="outlined" />
                    ))}
                  </Box>
                </Grid>
              )}

              {execution.policy_versions.length > 0 && (
                <Grid item xs={12}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    Policy Versions Used
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    {execution.policy_versions.map((version, i) => (
                      <Chip key={i} label={version} size="small" variant="outlined" color="primary" />
                    ))}
                  </Box>
                </Grid>
              )}
            </Grid>
          </Paper>

          {/* Event Timeline */}
          <Paper elevation={2} sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Event History ({execution.events.length} events)
            </Typography>

            <Timeline position="alternate">
              {execution.events.map((event, index) => (
                <TimelineItem key={event.event_id}>
                  <TimelineOppositeContent color="text.secondary">
                    <Typography variant="caption">
                      {format(new Date(event.timestamp), 'HH:mm:ss.SSS')}
                    </Typography>
                    {event.actor_id && (
                      <Typography variant="caption" display="block">
                        Actor: {event.actor_id}
                      </Typography>
                    )}
                  </TimelineOppositeContent>

                  <TimelineSeparator>
                    <TimelineDot color={event.event_type.includes('Failed') ? 'error' : 'primary'}>
                      {getEventIcon(event.event_type)}
                    </TimelineDot>
                    {index < execution.events.length - 1 && <TimelineConnector />}
                  </TimelineSeparator>

                  <TimelineContent>
                    <Card variant="outlined">
                      <CardContent>
                        <Typography variant="subtitle2" gutterBottom>
                          {event.event_type}
                        </Typography>
                        <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                          Event ID: {event.event_id}
                        </Typography>

                        {Object.keys(event.attributes).length > 0 && (
                          <Box sx={{ mt: 1 }}>
                            <Divider sx={{ my: 1 }} />
                            <Typography variant="caption" color="text.secondary">
                              Attributes:
                            </Typography>
                            <Box component="pre" sx={{ fontSize: '0.75rem', overflowX: 'auto', mt: 0.5 }}>
                              {JSON.stringify(event.attributes, null, 2)}
                            </Box>
                          </Box>
                        )}

                        {event.decision_made && (
                          <Alert severity="info" sx={{ mt: 1 }}>
                            <Typography variant="caption">
                              <strong>AI Decision:</strong> {event.decision_made}
                            </Typography>
                          </Alert>
                        )}
                      </CardContent>
                    </Card>
                  </TimelineContent>
                </TimelineItem>
              ))}
            </Timeline>
          </Paper>
        </>
      )}
    </Box>
  );
};
