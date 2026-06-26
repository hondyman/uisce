import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Card,
  CardContent,
  Tab,
  Tabs,
  LinearProgress,
  Tooltip,
  IconButton,
  Chip,
  Grid,
  Divider,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import AssessmentIcon from '@mui/icons-material/Assessment';
import WarningIcon from '@mui/icons-material/Warning';
import GridViewIcon from '@mui/icons-material/GridView';

// Types
interface QueueStat {
  id: string;
  xKey: string; // e.g., Step Type
  yKey: string; // e.g., Tenant or Advisor
  count: number;
  slaBreached: number;
}

// Intensity color scale
const getIntentistyColor = (count: number, max: number) => {
  const intensity = Math.min(count / max, 1);
  if (intensity < 0.2) return '#e0f2f1'; // Very light teal
  if (intensity < 0.4) return '#b2dfdb';
  if (intensity < 0.6) return '#80cbc4';
  if (intensity < 0.8) return '#4db6ac';
  return '#009688'; // Dark teal
};

const WorkQueueHeatmap: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [viewMode, setViewMode] = useState<'step_type' | 'queue'>('step_type');
  const [stats, setStats] = useState<QueueStat[]>([]);

  const fetchStats = async () => {
    setLoading(true);
    // Mock data
    setTimeout(() => {
      const mockStats: QueueStat[] = [];
      const tenants = ['Acme Corp', 'Globex', 'Soylent', 'Umbrella', 'Stark Ind'];
      const steps = ['Interpretation', 'Review', 'Approval', 'Drafting', 'Recommendation'];
      
      tenants.forEach(tenant => {
        steps.forEach(step => {
          // Generate random realistic distribution
          const count = Math.floor(Math.random() * 50);
          mockStats.push({
            id: `${tenant}-${step}`,
            xKey: step,
            yKey: tenant,
            count: count,
            slaBreached: Math.floor(Math.random() * (count * 0.2)),
          });
        });
      });
      
      setStats(mockStats);
      setLoading(false);
    }, 800);
  };

  useEffect(() => {
    fetchStats();
  }, [viewMode]);

  // Unique keys
  const xKeys = Array.from(new Set(stats.map(s => s.xKey))).sort();
  const yKeys = Array.from(new Set(stats.map(s => s.yKey))).sort();
  const maxCount = Math.max(...stats.map(s => s.count), 1);

  const getCellStat = (x: string, y: string) => stats.find(s => s.xKey === x && s.yKey === y);

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={600}>
          <AssessmentIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Work Queue Heatmap
        </Typography>
        <Stack direction="row" spacing={2} alignItems="center">
          <Tabs 
            value={viewMode} 
            onChange={(_, v) => setViewMode(v)} 
            sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36, py: 0 } }}
          >
            <Tab label="By Step Type" value="step_type" />
            <Tab label="By Queue" value="queue" />
          </Tabs>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchStats}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>

      {loading && <LinearProgress sx={{ mb: 2 }} />}

      {/* Summary Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent sx={{ pb: 1 }}>
              <Typography color="text.secondary" variant="caption">Total Pending</Typography>
              <Typography variant="h4">{stats.reduce((a, b) => a + b.count, 0)}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent sx={{ pb: 1 }}>
              <Typography color="text.secondary" variant="caption">SLA Breached</Typography>
              <Typography variant="h4" color="error.main">
                {stats.reduce((a, b) => a + b.slaBreached, 0)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent sx={{ pb: 1 }}>
              <Typography color="text.secondary" variant="caption">Busiest Tenant</Typography>
              <Typography variant="h5" noWrap>
                {/* Find tenant with max total items */}
                {yKeys.reduce((a, b) => {
                   const countA = stats.filter(s => s.yKey === a).reduce((acc, curr) => acc + curr.count, 0);
                   const countB = stats.filter(s => s.yKey === b).reduce((acc, curr) => acc + curr.count, 0);
                   return countA > countB ? a : b;
                }, '')}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
           <Card>
            <CardContent sx={{ pb: 1 }}>
              <Typography color="text.secondary" variant="caption">Backend Load</Typography>
              <Stack direction="row" alignItems="center" spacing={1}>
                <Chip size="small" label="Healthy" color="success" />
                <Typography variant="caption" color="text.secondary">88ms avg</Typography>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Heatmap Grid */}
      <Paper sx={{ p: 2, overflowX: 'auto' }}>
        <Box sx={{ display: 'grid', gridTemplateColumns: `150px repeat(${xKeys.length}, 1fr)`, gap: 1 }}>
          {/* Header Row */}
          <Box sx={{ display: 'flex', alignItems: 'flex-end', pb: 1 }}>
            <GridViewIcon color="action" />
          </Box>
          {xKeys.map(x => (
            <Box key={x} sx={{ textAlign: 'center', pb: 1, fontWeight: 500 }}>
              <Typography variant="caption" fontWeight="bold">{x}</Typography>
            </Box>
          ))}

          {/* Rows */}
          {yKeys.map(y => (
            <React.Fragment key={y}>
              {/* Row Label */}
              <Box sx={{ display: 'flex', alignItems: 'center', fontWeight: 500 }}>
                <Typography variant="body2" noWrap>{y}</Typography>
              </Box>
              
              {/* Cells */}
              {xKeys.map(x => {
                const stat = getCellStat(x, y);
                const count = stat?.count || 0;
                const breached = stat?.slaBreached || 0;
                
                return (
                  <Tooltip 
                    key={`${y}-${x}`}
                    title={
                      <Box sx={{ textAlign: 'center' }}>
                        <Typography variant="subtitle2">{y} • {x}</Typography>
                        <Divider sx={{ my: 0.5, bgcolor: 'rgba(255,255,255,0.2)' }} />
                        <Typography variant="body2">Pending: {count}</Typography>
                        {breached > 0 && <Typography variant="caption" color="#ff8a80">Breached: {breached}</Typography>}
                      </Box>
                    }
                  >
                    <Box
                      sx={{
                        height: 48,
                        bgcolor: count > 0 ? getIntentistyColor(count, maxCount) : '#f5f5f5',
                        borderRadius: 1,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        cursor: 'pointer',
                        position: 'relative',
                        transition: 'all 0.2s',
                        '&:hover': {
                          transform: 'scale(1.05)',
                          boxShadow: 2,
                          zIndex: 1
                        }
                      }}
                    >
                      {count > 0 && (
                        <Typography 
                          variant="body2" 
                          fontWeight={600} 
                          sx={{ color: count > maxCount * 0.5 ? '#fff' : 'text.primary' }}
                        >
                          {count}
                        </Typography>
                      )}
                      {breached > 0 && (
                        <Box sx={{ position: 'absolute', top: -4, right: -4 }}>
                          <WarningIcon sx={{ fontSize: 16, color: '#ef5350', bgcolor: '#fff', borderRadius: '50%' }} />
                        </Box>
                      )}
                    </Box>
                  </Tooltip>
                );
              })}
            </React.Fragment>
          ))}
        </Box>
      </Paper>
    </Box>
  );
};

export default WorkQueueHeatmap;
