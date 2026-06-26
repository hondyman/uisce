import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Grid,
  Card,
  CardContent,
  Chip,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  LinearProgress,
  IconButton,
  Tooltip,
  Alert,
  Paper
} from '@mui/material';
import {
  Timeline as TimelineIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Refresh as RefreshIcon,
  TrendingUp as TrendingUpIcon,
} from '@mui/icons-material';

interface IncidentEvent {
  event_type: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  title: string;
  details: { message: string };
  occurred_at: string;
}

const OpsCockpit: React.FC = () => {
  const healthScore = 98;
  const [incidents, setIncidents] = useState<IncidentEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Simulated fetch for demo
    const timer = setTimeout(() => {
      setIncidents([
        {
          event_type: 'latency_anomaly',
          severity: 'warning',
          title: 'High Latency Detected',
          details: { message: 'Latency p95 reached 542ms in US-West' },
          occurred_at: new Date().toISOString()
        },
        {
          event_type: 'data_drift',
          severity: 'info',
          title: 'Semantic Mapping Updated',
          details: { message: 'NetAssetValue formula updated via AI suggestion' },
          occurred_at: new Date(Date.now() - 3600000).toISOString()
        }
      ]);
      setLoading(false);
    }, 1000);
    return () => clearTimeout(timer);
  }, []);

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'error';
      case 'error': return 'error';
      case 'warning': return 'warning';
      case 'info': return 'info';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#f8fafc', minHeight: '100vh' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700, color: '#1e293b' }}>
            Ops Cockpit
          </Typography>
          <Typography variant="body1" color="textSecondary">
            Real-time Operational Intelligence for Semantic Execution
          </Typography>
        </Box>
        <IconButton color="primary" onClick={() => setLoading(true)}>
          <RefreshIcon />
        </IconButton>
      </Box>

      <Grid container spacing={3}>
        {/* Health Score Card */}
        <Grid item xs={12} md={4}>
          <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0', height: '100%' }}>
            <CardContent sx={{ textAlign: 'center', py: 4 }}>
              <Typography variant="h6" gutterBottom color="textSecondary">Tenant Health Score</Typography>
              <Box sx={{ position: 'relative', display: 'inline-flex', my: 2 }}>
                <Typography variant="h1" sx={{ fontWeight: 800, color: healthScore > 90 ? '#10b981' : '#f59e0b' }}>
                  {healthScore}
                </Typography>
              </Box>
              <Box sx={{ px: 4 }}>
                <LinearProgress 
                  variant="determinate" 
                  value={healthScore} 
                  sx={{ height: 10, borderRadius: 5, bgcolor: '#e2e8f0', '& .MuiLinearProgress-bar': { bgcolor: '#10b981' } }}
                />
              </Box>
              <Typography variant="body2" sx={{ mt: 2 }} color="textSecondary">
                System is performing optimally. Availability 100%.
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Latency Metrics */}
        <Grid item xs={12} md={8}>
          <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0', height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center' }}>
                <TimelineIcon sx={{ mr: 1, color: '#6366f1' }} /> Latency Distribution (p95)
              </Typography>
              <Box sx={{ mt: 3, height: 200, display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', px: 2 }}>
                {[45, 52, 48, 60, 120, 85, 42, 38, 45, 50, 55, 48].map((val, i) => (
                  <Tooltip key={i} title={`Time: -${(11-i)*5}m, Latency: ${val}ms`}>
                    <Box 
                      sx={{ 
                        width: '7%', 
                        height: `${(val / 150) * 100}%`, 
                        bgcolor: val > 100 ? '#f59e0b' : '#6366f1',
                        borderRadius: '4px 4px 0 0',
                        transition: 'height 0.3s ease'
                      }} 
                    />
                  </Tooltip>
                ))}
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 1, px: 1 }}>
                <Typography variant="caption" color="textSecondary">-60m</Typography>
                <Typography variant="caption" color="textSecondary">Now</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Incident Timeline */}
        <Grid item xs={12}>
          <Typography variant="h5" sx={{ mt: 4, mb: 2, fontWeight: 700 }}>Incident Timeline & Events</Typography>
          <Paper elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0', overflow: 'hidden' }}>
            <List sx={{ p: 0 }}>
              {incidents.map((incident, index) => (
                <React.Fragment key={index}>
                  <ListItem sx={{ py: 2 }}>
                    <ListItemIcon>
                      {incident.severity === 'warning' ? <WarningIcon color="warning" /> : <CheckCircleIcon color="success" />}
                    </ListItemIcon>
                    <ListItemText 
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{incident.title}</Typography>
                          <Chip label={incident.event_type} size="small" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                        </Box>
                      }
                      secondary={incident.details.message}
                    />
                    <Box sx={{ textAlign: 'right' }}>
                      <Typography variant="caption" color="textSecondary" sx={{ display: 'block' }}>
                        {new Date(incident.occurred_at).toLocaleTimeString()}
                      </Typography>
                      <Chip 
                        label={incident.severity.toUpperCase()} 
                        color={getSeverityColor(incident.severity) as any} 
                        size="small" 
                        sx={{ mt: 0.5, fontWeight: 700, fontSize: '0.6rem' }} 
                      />
                    </Box>
                  </ListItem>
                  {index < incidents.length - 1 && <Divider />}
                </React.Fragment>
              ))}
              {incidents.length === 0 && !loading && (
                <Box sx={{ p: 4, textAlign: 'center' }}>
                  <Typography color="textSecondary">No incidents reported in the last 24 hours.</Typography>
                </Box>
              )}
            </List>
          </Paper>
        </Grid>
      </Grid>

      {/* Real-time Alerts */}
      <Box sx={{ position: 'fixed', bottom: 24, right: 24, width: 320 }}>
        <Alert severity="info" variant="filled" icon={<TrendingUpIcon />} sx={{ borderRadius: 2, boxShadow: 3 }}>
          System utilization is at 42%. Efficient scaling active.
        </Alert>
      </Box>
    </Box>
  );
};

export default OpsCockpit;
