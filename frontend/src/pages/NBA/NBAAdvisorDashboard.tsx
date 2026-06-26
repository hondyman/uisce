import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Tabs,
  Tab,
  Badge,
  IconButton,
  Divider,
  LinearProgress,
} from '@mui/material';
import {
  TrendingUp as TrendingUpIcon,
  AttachMoney as MoneyIcon,
  Schedule as ScheduleIcon,
  CheckCircle as SuccessIcon,
  Close as CloseIcon,
  Phone as PhoneIcon,
  Email as EmailIcon,
  VideoCall as VideoIcon,
  Person as PersonIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';
import { devDebug } from '../../utils/devLogger';

interface NBARecommendation {
  recommendation_id: string;
  client_id: string;
  client_name?: string;
  action_id: string;
  action_name: string;
  action_code: string;
  action_category: string;
  confidence_score: number;
  urgency_score: number;
  expected_value: number;
  success_probability: number;
  overall_score: number;
  reasoning: string;
  supporting_data: any;
  recommended_at: string;
  template_content?: {
    email_subject?: string;
    email_body?: string;
    call_script?: string;
    meeting_agenda?: string;
  };
  estimated_duration_minutes?: number;
  default_channel?: string;
}

interface NBAStats {
  pending_count: number;
  critical_count: number;
  total_potential_revenue: number;
  avg_success_rate: number;
  completed_today: number;
}

export const NBAAdvisorDashboard: React.FC = () => {
  const [recommendations, setRecommendations] = useState<NBARecommendation[]>([]);
  const [stats, setStats] = useState<NBAStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'critical' | 'high_value'>('all');
  const [selectedAction, setSelectedAction] = useState<NBARecommendation | null>(null);
  const [dismissDialogOpen, setDismissDialogOpen] = useState(false);
  const [dismissReason, setDismissReason] = useState('');
  const [ws, setWs] = useState<WebSocket | null>(null);

  // WebSocket connection for real-time updates
  useEffect(() => {
    const advisorId = localStorage.getItem('advisor_id'); // Get from auth context
    const wsUrl = `ws://localhost:8080/api/nba/stream?advisor_id=${advisorId}`;
    
    const websocket = new WebSocket(wsUrl);
    
    websocket.onopen = () => {
      devDebug('NBA WebSocket connected');
    };
    
    websocket.onmessage = (event) => {
      const newRecommendation = JSON.parse(event.data);
      setRecommendations(prev => [newRecommendation, ...prev].slice(0, 100)); // Keep top 100
    };
    
    websocket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
    
    websocket.onclose = () => {
      devDebug('NBA WebSocket disconnected');
      // Attempt reconnect after 5 seconds
      setTimeout(() => {
        setWs(websocket);
      }, 5000);
    };
    
    setWs(websocket);
    
    return () => {
      websocket.close();
    };
  }, []);

  // Fetch initial data
  useEffect(() => {
    fetchRecommendations();
    fetchStats();
  }, []);

  const fetchRecommendations = async () => {
    try {
      const advisorId = localStorage.getItem('advisor_id');
      const response = await fetch(`/api/nba/recommendations?advisor_id=${advisorId}&status=PENDING&limit=50`);
      const data = await response.json();
      setRecommendations(data.recommendations || []);
    } catch (error) {
      console.error('Failed to fetch recommendations:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const advisorId = localStorage.getItem('advisor_id');
      const response = await fetch(`/api/nba/stats?advisor_id=${advisorId}`);
      const data = await response.json();
      setStats(data);
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  };

  const handleExecute = async (recommendation: NBARecommendation) => {
    setSelectedAction(recommendation);
  };

  const handleDismiss = async (recommendationId: string, reason: string) => {
    try {
      await fetch(`/api/nba/recommendations/${recommendationId}/dismiss`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason, notes: dismissReason })
      });
      
      // Remove from list
      setRecommendations(prev => prev.filter(r => r.recommendation_id !== recommendationId));
      setDismissDialogOpen(false);
      setDismissReason('');
      
      // Refresh stats
      fetchStats();
    } catch (error) {
      console.error('Failed to dismiss recommendation:', error);
    }
  };

  const handleCompleteAction = async () => {
    if (!selectedAction) return;
    
    try {
      await fetch(`/api/nba/recommendations/${selectedAction.recommendation_id}/execute`, {
        method: 'POST'
      });
      
      // Remove from list
      setRecommendations(prev => prev.filter(r => r.recommendation_id !== selectedAction.recommendation_id));
      setSelectedAction(null);
      
      // Refresh stats
      fetchStats();
    } catch (error) {
      console.error('Failed to execute recommendation:', error);
    }
  };

  const filteredRecommendations = recommendations.filter(r => {
    if (filter === 'critical') return r.urgency_score > 0.8;
    if (filter === 'high_value') return r.expected_value > 5000;
    return true;
  });

  const getUrgencyColor = (score: number): 'error' | 'warning' | 'info' => {
    if (score > 0.8) return 'error';
    if (score > 0.5) return 'warning';
    return 'info';
  };

  const getChannelIcon = (channel?: string) => {
    switch (channel) {
      case 'PHONE': return <PhoneIcon />;
      case 'EMAIL': return <EmailIcon />;
      case 'VIDEO_CALL': return <VideoIcon />;
      default: return <PersonIcon />;
    }
  };

  if (loading) {
    return <LinearProgress />;
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" gutterBottom>
          AI-Powered Next Best Actions
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Real-time recommendations powered by machine learning
        </Typography>
      </Box>

      {/* Stats Cards */}
      {stats && (
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <ScheduleIcon color="primary" sx={{ mr: 1 }} />
                  <Typography variant="h6">{stats.pending_count}</Typography>
                </Box>
                <Typography variant="body2" color="text.secondary">
                  Pending Actions
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <TrendingUpIcon color="error" sx={{ mr: 1 }} />
                  <Typography variant="h6">{stats.critical_count}</Typography>
                </Box>
                <Typography variant="body2" color="text.secondary">
                  Critical (Urgent)
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <MoneyIcon color="success" sx={{ mr: 1 }} />
                  <Typography variant="h6">
                    ${stats.total_potential_revenue.toLocaleString()}
                  </Typography>
                </Box>
                <Typography variant="body2" color="text.secondary">
                  Potential Revenue
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <SuccessIcon color="info" sx={{ mr: 1 }} />
                  <Typography variant="h6">
                    {(stats.avg_success_rate * 100).toFixed(0)}%
                  </Typography>
                </Box>
                <Typography variant="body2" color="text.secondary">
                  Avg Success Rate
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Filter Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={filter} onChange={(_, v) => setFilter(v)}>
          <Tab 
            label={
              <Badge badgeContent={recommendations.length} color="primary">
                All
              </Badge>
            } 
            value="all" 
          />
          <Tab 
            label={
              <Badge badgeContent={recommendations.filter(r => r.urgency_score > 0.8).length} color="error">
                Critical
              </Badge>
            } 
            value="critical" 
          />
          <Tab 
            label={
              <Badge badgeContent={recommendations.filter(r => r.expected_value > 5000).length} color="success">
                High Value ($5K+)
              </Badge>
            } 
            value="high_value" 
          />
        </Tabs>
      </Paper>

      {/* Recommendations List */}
      <Grid container spacing={2}>
        {filteredRecommendations.map((rec) => (
          <Grid item xs={12} key={rec.recommendation_id}>
            <Card variant="outlined" sx={{ '&:hover': { boxShadow: 3 } }}>
              <CardContent>
                <Grid container spacing={2}>
                  {/* Left: Client Info & Action */}
                  <Grid item xs={12} md={8}>
                    <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                      <Typography variant="h6" sx={{ mr: 2 }}>
                        {rec.client_name || `Client ${rec.client_id.substring(0, 8)}`}
                      </Typography>
                      <Chip 
                        label={`Urgency: ${(rec.urgency_score * 100).toFixed(0)}%`}
                        color={getUrgencyColor(rec.urgency_score)}
                        size="small"
                        sx={{ mr: 1 }}
                      />
                      <Chip 
                        label={`Confidence: ${(rec.confidence_score * 100).toFixed(0)}%`}
                        color="primary"
                        variant="outlined"
                        size="small"
                      />
                    </Box>

                    <Typography variant="subtitle1" color="primary" gutterBottom>
                      {rec.action_name}
                    </Typography>

                    <Alert severity="info" sx={{ mb: 2 }}>
                      <Typography variant="body2">
                        <strong>AI Reasoning:</strong> {rec.reasoning}
                      </Typography>
                    </Alert>

                    <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                      <Chip 
                        icon={getChannelIcon(rec.default_channel)}
                        label={rec.default_channel || 'PHONE'}
                        size="small"
                      />
                      <Chip 
                        icon={<ScheduleIcon />}
                        label={`${rec.estimated_duration_minutes || 30} min`}
                        size="small"
                      />
                      <Chip 
                        icon={<MoneyIcon />}
                        label={`$${rec.expected_value.toLocaleString()} value`}
                        size="small"
                        color="success"
                      />
                      <Chip 
                        icon={<SuccessIcon />}
                        label={`${(rec.success_probability * 100).toFixed(0)}% success rate`}
                        size="small"
                      />
                    </Box>
                  </Grid>

                  {/* Right: Actions */}
                  <Grid item xs={12} md={4}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, height: '100%', justifyContent: 'center' }}>
                      <Button
                        variant="contained"
                        color="primary"
                        fullWidth
                        onClick={() => handleExecute(rec)}
                      >
                        Execute Action
                      </Button>
                      <Button
                        variant="outlined"
                        color="secondary"
                        fullWidth
                        onClick={() => {
                          setSelectedAction(rec);
                          setDismissDialogOpen(true);
                        }}
                      >
                        Dismiss
                      </Button>
                      <Typography variant="caption" color="text.secondary" align="center">
                        Recommended {format(new Date(rec.recommended_at), 'PP')}
                      </Typography>
                    </Box>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          </Grid>
        ))}

        {filteredRecommendations.length === 0 && (
          <Grid item xs={12}>
            <Alert severity="success">
              <Typography variant="h6">All caught up!</Typography>
              <Typography variant="body2">
                No pending recommendations at this time. New AI-powered actions will appear here automatically.
              </Typography>
            </Alert>
          </Grid>
        )}
      </Grid>

      {/* Action Execution Modal */}
      <Dialog 
        open={!!selectedAction && !dismissDialogOpen} 
        onClose={() => setSelectedAction(null)}
        maxWidth="md"
        fullWidth
      >
        {selectedAction && (
          <>
            <DialogTitle>
              Execute: {selectedAction.action_name}
              <IconButton
                onClick={() => setSelectedAction(null)}
                sx={{ position: 'absolute', right: 8, top: 8 }}
              >
                <CloseIcon />
              </IconButton>
            </DialogTitle>
            <DialogContent dividers>
              <Typography variant="subtitle2" gutterBottom>
                Client: {selectedAction.client_name || selectedAction.client_id}
              </Typography>

              <Divider sx={{ my: 2 }} />

              {selectedAction.template_content?.email_subject && (
                <Box sx={{ mb: 3 }}>
                  <Typography variant="subtitle1" gutterBottom>
                    📧 Email Template
                  </Typography>
                  <TextField
                    label="Subject"
                    fullWidth
                    defaultValue={selectedAction.template_content.email_subject}
                    sx={{ mb: 2 }}
                  />
                  <TextField
                    label="Body"
                    fullWidth
                    multiline
                    rows={6}
                    defaultValue={selectedAction.template_content.email_body}
                  />
                </Box>
              )}

              {selectedAction.template_content?.call_script && (
                <Box sx={{ mb: 3 }}>
                  <Typography variant="subtitle1" gutterBottom>
                    📞 Call Script
                  </Typography>
                  <Paper sx={{ p: 2, bgcolor: 'grey.50' }}>
                    <Typography variant="body2" sx={{ whiteSpace: 'pre-line' }}>
                      {selectedAction.template_content.call_script}
                    </Typography>
                  </Paper>
                </Box>
              )}

              {selectedAction.template_content?.meeting_agenda && (
                <Box>
                  <Typography variant="subtitle1" gutterBottom>
                    📅 Meeting Agenda
                  </Typography>
                  <Paper sx={{ p: 2, bgcolor: 'grey.50' }}>
                    <Typography variant="body2" sx={{ whiteSpace: 'pre-line' }}>
                      {selectedAction.template_content.meeting_agenda}
                    </Typography>
                  </Paper>
                </Box>
              )}
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setSelectedAction(null)}>
                Cancel
              </Button>
              <Button onClick={handleCompleteAction} variant="contained" color="primary">
                Mark as Complete
              </Button>
            </DialogActions>
          </>
        )}
      </Dialog>

      {/* Dismiss Dialog */}
      <Dialog open={dismissDialogOpen} onClose={() => setDismissDialogOpen(false)}>
        <DialogTitle>Dismiss Recommendation</DialogTitle>
        <DialogContent>
          <Typography variant="body2" gutterBottom>
            Why are you dismissing this recommendation?
          </Typography>
          <TextField
            label="Reason (optional)"
            fullWidth
            multiline
            rows={3}
            value={dismissReason}
            onChange={(e) => setDismissReason(e.target.value)}
            sx={{ mt: 2 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDismissDialogOpen(false)}>
            Cancel
          </Button>
          <Button 
            onClick={() => selectedAction && handleDismiss(selectedAction.recommendation_id, 'NOT_RELEVANT')}
            color="secondary"
          >
            Not Relevant
          </Button>
          <Button 
            onClick={() => selectedAction && handleDismiss(selectedAction.recommendation_id, 'ALREADY_DONE')}
            color="secondary"
          >
            Already Done
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};