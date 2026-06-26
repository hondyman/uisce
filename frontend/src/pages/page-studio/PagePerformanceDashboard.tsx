import React from 'react';
import { 
    Box, 
    Typography, 
    Paper, 
    Grid, 
    Stack, 
    Chip, 
    Divider, 
    LinearProgress,
    Button,
    Alert,
    AlertTitle
} from '@mui/material';
import { 
    Speed as PerformanceIcon, 
    Timeline as TrafficIcon, 
    ErrorOutline as ErrorIcon,
    Bolt as AsoIcon,
    AutoAwesome as AiIcon,
    CheckCircle as SuccessIcon,
    Warning as WarningIcon
} from '@mui/icons-material';

interface PagePerformanceDashboardProps {
    pageId: string;
}

export const PagePerformanceDashboard: React.FC<PagePerformanceDashboardProps> = ({ pageId }) => {
    // Mock data - in real app would come from usePageMetrics or an API
    const metrics = {
        p95Latency: 580,
        sloTarget: 500,
        requestsPerSecond: 125,
        errorRate: 0.05, 
        preAggHitRate: 0.65,
        cacheHitRate: 0.42,
    };

    const isViolating = metrics.p95Latency > metrics.sloTarget;

    return (
        <Box sx={{ p: 4, maxWidth: 1000, mx: 'auto' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    <PerformanceIcon color="primary" sx={{ mr: 1, fontSize: 32 }} />
                    <Typography variant="h5" fontWeight="bold">Page Performance Intelligence</Typography>
                </Box>
                <Chip 
                    icon={isViolating ? <WarningIcon /> : <SuccessIcon />} 
                    label={isViolating ? 'SLO Violation Detected' : 'Healthy Performance'} 
                    color={isViolating ? 'error' : 'success'} 
                    sx={{ fontWeight: 'bold' }}
                />
            </Box>

            {isViolating && (
                <Alert severity="error" sx={{ mb: 4, borderRadius: 2 }}>
                    <AlertTitle>Performance Critical</AlertTitle>
                    This page is exceeding its 500ms p95 latency SLO. 65% of queries are hitting raw storage.
                    <Button variant="text" color="inherit" size="small" sx={{ ml: 2, fontWeight: 'bold' }}>
                        Apply ASO Recommendations
                    </Button>
                </Alert>
            )}

            <Grid container spacing={3}>
                <Grid item xs={12} md={4}>
                    <Paper variant="outlined" sx={{ p: 3, borderRadius: 3 }}>
                        <Typography variant="subtitle2" color="textSecondary" gutterBottom>Response Time (p95)</Typography>
                        <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1 }}>
                            <Typography variant="h4" fontWeight="bold">{metrics.p95Latency}ms</Typography>
                            <Typography variant="caption" color="error.main" fontWeight="bold">+16% from target</Typography>
                        </Box>
                        <LinearProgress 
                            variant="determinate" 
                            value={Math.min(100, (metrics.p95Latency / metrics.sloTarget) * 100)} 
                            color={isViolating ? 'error' : 'primary'}
                            sx={{ mt: 2, height: 6, borderRadius: 3 }}
                        />
                    </Paper>
                </Grid>

                <Grid item xs={12} md={4}>
                    <Paper variant="outlined" sx={{ p: 3, borderRadius: 3 }}>
                        <Typography variant="subtitle2" color="textSecondary" gutterBottom>Pre-agg Hit Rate</Typography>
                        <Typography variant="h4" fontWeight="bold">{Math.round(metrics.preAggHitRate * 100)}%</Typography>
                        <LinearProgress 
                            variant="determinate" 
                            value={metrics.preAggHitRate * 100} 
                            color={metrics.preAggHitRate > 0.8 ? 'success' : 'warning'}
                            sx={{ mt: 2, height: 6, borderRadius: 3 }}
                        />
                    </Paper>
                </Grid>

                <Grid item xs={12} md={4}>
                    <Paper variant="outlined" sx={{ p: 3, borderRadius: 3 }}>
                        <Typography variant="subtitle2" color="textSecondary" gutterBottom>Throughput (RPS)</Typography>
                        <Typography variant="h4" fontWeight="bold">{metrics.requestsPerSecond}</Typography>
                        <Typography variant="caption" color="textSecondary">Steady traffic flow</Typography>
                        <Box sx={{ mt: 2, height: 6, bgcolor: 'grey.100', borderRadius: 3, overflow: 'hidden' }}>
                            <Box sx={{ width: '70%', height: '100%', bgcolor: 'primary.light' }} />
                        </Box>
                    </Paper>
                </Grid>

                <Grid item xs={12}>
                    <Paper variant="outlined" sx={{ p: 0, borderRadius: 3, overflow: 'hidden' }}>
                        <Box sx={{ p: 2, bgcolor: 'rgba(99, 102, 241, 0.05)', borderBottom: '1px solid', borderColor: 'divider', display: 'flex', alignItems: 'center' }}>
                            <AiIcon color="primary" sx={{ mr: 1 }} />
                            <Typography variant="subtitle2" fontWeight="bold">AI Performance Advisor</Typography>
                        </Box>
                        <Stack divider={<Divider />}>
                            <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <Box>
                                    <Typography variant="body2" fontWeight="bold">Create Pre-aggregation for 'Positions'</Typography>
                                    <Typography variant="caption" color="textSecondary">Estimated to reduce page latency by 320ms.</Typography>
                                </Box>
                                <Button size="small" variant="contained" startIcon={<AsoIcon />}>Tune ASO</Button>
                            </Box>
                            <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <Box>
                                    <Typography variant="body2" fontWeight="bold">Enable Component-level Redis Caching</Typography>
                                    <Typography variant="caption" color="textSecondary">Predicted cache hit rate: 85% for this workload.</Typography>
                                </Box>
                                <Button size="small" variant="outlined">Configure</Button>
                            </Box>
                        </Stack>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};
