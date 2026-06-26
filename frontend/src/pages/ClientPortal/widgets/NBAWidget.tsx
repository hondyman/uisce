import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Chip,
  LinearProgress,
  Alert,
  IconButton,
  Collapse,
} from '@mui/material';
import {
  TrendingUp as OpportunityIcon,
  Close as CloseIcon,
  ExpandMore as ExpandIcon,
  CheckCircle as CompleteIcon,
} from '@mui/icons-material';
import { WidgetProps } from '../Dashboard';

interface NBARecommendation {
  recommendation_id: string;
  action_type: string;
  title: string;
  description: string;
  ai_reasoning: string;
  expected_benefit: string;
  urgency: 'LOW' | 'MEDIUM' | 'HIGH';
  due_date?: string;
  estimated_value: number;
}

export const NBAWidget: React.FC<WidgetProps> = () => {
  const [recommendations, setRecommendations] = useState<NBARecommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  useEffect(() => {
    fetchRecommendations();
  }, []);

  const fetchRecommendations = async () => {
    try {
      const response = await fetch('/api/nba/recommendations?status=PENDING&limit=5');
      const data = await response.json();
      setRecommendations(data.recommendations || []);
    } catch (error) {
      console.error('Failed to fetch recommendations:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleScheduleMeeting = async (recommendationId: string) => {
    try {
      await fetch(`/api/nba/recommendations/${recommendationId}/schedule-meeting`, {
        method: 'POST',
      });
      // Optionally redirect to meeting scheduler
      window.location.href = '/meetings/schedule';
    } catch (error) {
      console.error('Failed to schedule meeting:', error);
    }
  };

  const handleDismiss = async (recommendationId: string) => {
    try {
      await fetch(`/api/nba/recommendations/${recommendationId}/dismiss`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dismissal_reason: 'NOT_INTERESTED' }),
      });
      setRecommendations(recommendations.filter(r => r.recommendation_id !== recommendationId));
    } catch (error) {
      console.error('Failed to dismiss recommendation:', error);
    }
  };

  const toggleExpand = (id: string) => {
    setExpanded(prev => ({ ...prev, [id]: !prev[id] }));
  };

  const getUrgencyColor = (urgency: string) => {
    switch (urgency) {
      case 'HIGH': return 'error';
      case 'MEDIUM': return 'warning';
      default: return 'info';
    }
  };

  if (loading) {
    return <LinearProgress />;
  }

  if (recommendations.length === 0) {
    return (
      <Box sx={{ textAlign: 'center', py: 4 }}>
        <CompleteIcon sx={{ fontSize: 48, color: 'success.main', mb: 2 }} />
        <Typography variant="h6">All Caught Up!</Typography>
        <Typography variant="body2" color="text.secondary">
          No new recommendations at this time. We'll notify you when there are new opportunities.
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, height: '100%', overflow: 'auto' }}>
      {recommendations.map((rec) => (
        <Card key={rec.recommendation_id} variant="outlined">
          <CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', mb: 1 }}>
              <Box sx={{ flex: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                  <OpportunityIcon color="primary" />
                  <Typography variant="subtitle1" component="div">
                    {rec.title}
                  </Typography>
                  <Chip 
                    label={rec.urgency} 
                    size="small" 
                    color={getUrgencyColor(rec.urgency)} 
                  />
                </Box>
                <Typography variant="body2" color="text.secondary" paragraph>
                  {rec.description}
                </Typography>
              </Box>
              <IconButton size="small" onClick={() => handleDismiss(rec.recommendation_id)}>
                <CloseIcon fontSize="small" />
              </IconButton>
            </Box>

            {rec.expected_benefit && (
              <Alert severity="success" sx={{ mb: 2 }}>
                <Typography variant="body2">
                  <strong>Potential Benefit:</strong> {rec.expected_benefit}
                </Typography>
                {rec.estimated_value > 0 && (
                  <Typography variant="caption">
                    Estimated Value: ${rec.estimated_value.toLocaleString()}
                  </Typography>
                )}
              </Alert>
            )}

            <Box sx={{ display: 'flex', gap: 1, mb: expanded[rec.recommendation_id] ? 2 : 0 }}>
              <Button
                variant="contained"
                size="small"
                onClick={() => handleScheduleMeeting(rec.recommendation_id)}
              >
                Schedule Meeting
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => toggleExpand(rec.recommendation_id)}
                endIcon={<ExpandIcon />}
              >
                Why?
              </Button>
            </Box>

            <Collapse in={expanded[rec.recommendation_id]}>
              <Box sx={{ mt: 2, p: 2, bgcolor: 'background.paper', borderRadius: 1, border: 1, borderColor: 'divider' }}>
                <Typography variant="subtitle2" gutterBottom>
                  AI Reasoning:
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {rec.ai_reasoning}
                </Typography>
              </Box>
            </Collapse>
          </CardContent>
        </Card>
      ))}
    </Box>
  );
};
