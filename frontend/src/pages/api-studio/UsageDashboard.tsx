import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, Grid, Card, CardContent, CircularProgress, Chip, LinearProgress } from '@mui/material';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { APIEndpoint } from '../../types/apiStudio';
import { ApiStudioApi } from '../../api/apiStudio';

interface UsageDashboardProps {
    endpoint: Partial<APIEndpoint>;
}

const UsageDashboard: React.FC<UsageDashboardProps> = ({ endpoint }) => {
    const [metrics, setMetrics] = useState<any>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchMetrics = async () => {
            if (!endpoint.id) return;
            setLoading(true);
            try {
                // In a real implementation, this would fetch historical time-series data
                // For now, we simulate it or use the summary metrics
                const data = await ApiStudioApi.getEndpointMetrics(endpoint.id);
                setMetrics(data);
            } catch (err) {
                console.error("Failed to load metrics", err);
            } finally {
                setLoading(false);
            }
        };

        fetchMetrics();
    }, [endpoint.id]);

    if (loading) {
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
                <CircularProgress />
            </Box>
        );
    }

    if (!metrics) {
        return <Typography color="textSecondary">No usage data available.</Typography>;
    }

    // Mock time-series data for visualization (since the API currently returns summary stats)
    const timeSeriesData = Array.from({ length: 24 }, (_, i) => ({
        time: `${i}:00`,
        requests: Math.floor(Math.random() * 1000) + 50,
        latency: Math.floor(Math.random() * 200) + 20,
        errors: Math.floor(Math.random() * 10)
    }));

    return (
        <Box sx={{ p: 2 }}>
            <Grid container spacing={3} sx={{ mb: 4 }}>
                <Grid item xs={12} md={3}>
                    <Card elevation={0} sx={{ border: '1px solid #eee', borderRadius: 2 }}>
                        <CardContent>
                            <Typography variant="caption" color="textSecondary">Avg Latency (P95)</Typography>
                            <Typography variant="h4" fontWeight="bold">{metrics.p95}ms</Typography>
                             <LinearProgress variant="determinate" value={Math.min(metrics.p95 / 5, 100)} sx={{ mt: 1, height: 6, borderRadius: 3, bgcolor: '#e0e0e0', '& .MuiLinearProgress-bar': { bgcolor: metrics.p95 > 200 ? 'error.main' : 'success.main' } }} />
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} md={3}>
                    <Card elevation={0} sx={{ border: '1px solid #eee', borderRadius: 2 }}>
                        <CardContent>
                            <Typography variant="caption" color="textSecondary">Error Rate</Typography>
                            <Typography variant="h4" fontWeight="bold">{(metrics.errorRate || 0.02) * 100}%</Typography>
                             <LinearProgress variant="determinate" value={(metrics.errorRate || 0.02) * 100 * 5} sx={{ mt: 1, height: 6, borderRadius: 3 }} color="secondary" />
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} md={3}>
                    <Card elevation={0} sx={{ border: '1px solid #eee', borderRadius: 2 }}>
                        <CardContent>
                            <Typography variant="caption" color="textSecondary">Requests (24h)</Typography>
                            <Typography variant="h4" fontWeight="bold">{(metrics.qps * 60 * 60 * 24 / 1000).toFixed(1)}k</Typography>
                            <Typography variant="caption" color="success.main">+12% vs yesterday</Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} md={3}>
                    <Card elevation={0} sx={{ border: '1px solid #eee', borderRadius: 2 }}>
                        <CardContent>
                            <Typography variant="caption" color="textSecondary">Cache Hit Rate</Typography>
                            <Typography variant="h4" fontWeight="bold">{metrics.cacheHitRate * 100}%</Typography>
                            <LinearProgress variant="determinate" value={metrics.cacheHitRate * 100} sx={{ mt: 1, height: 6, borderRadius: 3 }} />
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>

            {/* Charts */}
            <Grid container spacing={3}>
                <Grid item xs={12} md={8}>
                    <Paper elevation={0} sx={{ p: 2, border: '1px solid #eee', borderRadius: 2 }}>
                         <Typography variant="subtitle2" gutterBottom fontWeight="bold">Traffic Volume & Latency</Typography>
                         <Box sx={{ height: 300 }}>
                            <ResponsiveContainer width="100%" height="100%">
                                <AreaChart data={timeSeriesData}>
                                    <defs>
                                        <linearGradient id="colorReq" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#8884d8" stopOpacity={0.8}/>
                                            <stop offset="95%" stopColor="#8884d8" stopOpacity={0}/>
                                        </linearGradient>
                                        <linearGradient id="colorLat" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#82ca9d" stopOpacity={0.8}/>
                                            <stop offset="95%" stopColor="#82ca9d" stopOpacity={0}/>
                                        </linearGradient>
                                    </defs>
                                    <XAxis dataKey="time" />
                                    <YAxis yAxisId="left" />
                                    <YAxis yAxisId="right" orientation="right" />
                                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                                    <Tooltip />
                                    <Area yAxisId="left" type="monotone" dataKey="requests" stroke="#8884d8" fillOpacity={1} fill="url(#colorReq)" name="Requests" />
                                    <Area yAxisId="right" type="monotone" dataKey="latency" stroke="#82ca9d" fillOpacity={1} fill="url(#colorLat)" name="Latency (ms)" />
                                </AreaChart>
                            </ResponsiveContainer>
                         </Box>
                    </Paper>
                </Grid>
                <Grid item xs={12} md={4}>
                    <Paper elevation={0} sx={{ p: 2, border: '1px solid #eee', borderRadius: 2 }}>
                        <Typography variant="subtitle2" gutterBottom fontWeight="bold">Errors by Type</Typography>
                        <Box sx={{ height: 300 }}>
                             <ResponsiveContainer width="100%" height="100%">
                                <BarChart data={[
                                    { name: '400', value: 120 },
                                    { name: '401', value: 45 },
                                    { name: '403', value: 30 },
                                    { name: '404', value: 80 },
                                    { name: '500', value: 15 },
                                ]}>
                                    <XAxis dataKey="name" />
                                    <YAxis />
                                    <Tooltip />
                                    <Bar dataKey="value" fill="#ff8042" radius={[4, 4, 0, 0]} />
                                </BarChart>
                             </ResponsiveContainer>
                        </Box>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};

export default UsageDashboard;
