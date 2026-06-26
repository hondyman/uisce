import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, CircularProgress, Grid, Divider, Button, Chip, LinearProgress } from '@mui/material';
import { Download as DownloadIcon, Speed as SpeedIcon, GraphicEq as MetricsIcon, CheckCircle as HealthyIcon } from '@mui/icons-material';
import { ApiStudioApi } from '../../api/apiStudio';
import { APIEndpoint } from '../../types/apiStudio';

interface APIPreviewProps {
    endpoint: APIEndpoint;
}

const APIPreview: React.FC<APIPreviewProps> = ({ endpoint }) => {
    const [data, setData] = useState<any>(null);
    const [spec, setSpec] = useState<any>(null);
    const [metrics, setMetrics] = useState<any>(null);
    const [loading, setLoading] = useState(false);

    // Mock tenant/env
    const env = 'production';
    const tenantId = 'default';

    useEffect(() => {
        if (endpoint.path) {
            loadPreview();
        }
    }, [endpoint]);

    const loadPreview = async () => {
        setLoading(true);
        try {
            const [sampleData, openApi, perfMetrics] = await Promise.all([
                ApiStudioApi.previewEndpoint(endpoint.path, endpoint.method, env, tenantId, {}),
                ApiStudioApi.getOpenApiSpec(env, tenantId),
                ApiStudioApi.getEndpointMetrics(endpoint.id)
            ]);
            setData(sampleData);
            setSpec(openApi);
            setMetrics(perfMetrics);
        } catch (err) {
            console.error('Preview failed', err);
        } finally {
            setLoading(false);
        }
    };

    const handleDownloadSDK = () => {
        const url = ApiStudioApi.getSdkURL('typescript', env, tenantId);
        window.open(url, '_blank');
    };

    if (loading) return (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 5 }}>
            <CircularProgress />
        </Box>
    );

    return (
        <Grid container spacing={3}>
            {/* Performance Cockpit */}
            <Grid item xs={12}>
                <Paper elevation={0} sx={{ p: 2, borderRadius: 3, background: 'rgba(255, 255, 255, 0.4)', border: '1px solid rgba(0,0,0,0.05)' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                        <SpeedIcon sx={{ mr: 1, color: 'primary.main' }} />
                        <Typography variant="subtitle1" fontWeight="bold">Performance Cockpit</Typography>
                        <Chip label="Healthy" size="small" color="success" icon={<HealthyIcon />} sx={{ ml: 2 }} />
                    </Box>
                    <Grid container spacing={2}>
                        <Grid item xs={6} md={2}>
                            <Typography variant="caption" color="textSecondary">p95 Latency</Typography>
                            <Typography variant="h6">{metrics?.p95}ms</Typography>
                            <LinearProgress variant="determinate" value={60} color="success" sx={{ height: 4, borderRadius: 2 }} />
                        </Grid>
                        <Grid item xs={6} md={2}>
                            <Typography variant="caption" color="textSecondary">Throughput (QPS)</Typography>
                            <Typography variant="h6">{metrics?.qps}</Typography>
                        </Grid>
                        <Grid item xs={6} md={2}>
                            <Typography variant="caption" color="textSecondary">Cache Hit</Typography>
                            <Typography variant="h6">{Math.round(metrics?.cacheHitRate * 100)}%</Typography>
                        </Grid>
                        <Grid item xs={6} md={2}>
                            <Typography variant="caption" color="textSecondary">Pre-agg Hit</Typography>
                            <Typography variant="h6">{Math.round(metrics?.preaggHitRate * 100)}%</Typography>
                        </Grid>
                        <Grid item xs={12} md={4} sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
                           <Button variant="outlined" startIcon={<DownloadIcon />} onClick={handleDownloadSDK}>
                               Download TypeScript SDK
                           </Button>
                        </Grid>
                    </Grid>
                </Paper>
            </Grid>

            {/* Data Response */}
            <Grid item xs={12} md={6}>
                <Typography variant="subtitle2" gutterBottom color="primary">Sample Data Response</Typography>
                <Paper sx={{ p: 2, bgcolor: '#1e293b', color: '#f8fafc', borderRadius: 2, maxHeight: 400, overflowY: 'auto' }}>
                    <pre style={{ margin: 0, fontSize: '0.8rem' }}>
                        {JSON.stringify(data, null, 2)}
                    </pre>
                </Paper>
            </Grid>

            {/* OpenAPI / GQL Spec */}
            <Grid item xs={12} md={6}>
                <Typography variant="subtitle2" gutterBottom color="secondary">Developer Documentation</Typography>
                <Paper sx={{ p: 2, bgcolor: '#0f172a', color: '#e2e8f0', borderRadius: 2, maxHeight: 400, overflowY: 'auto' }}>
                    <pre style={{ margin: 0, fontSize: '0.8rem' }}>
                        {JSON.stringify(spec, null, 2)}
                    </pre>
                </Paper>

                {endpoint.type === 'graphql' && (
                    <Box sx={{ mt: 2 }}>
                        <Typography variant="subtitle2" gutterBottom color="success.main">GraphQL Query Snippet</Typography>
                        <Paper sx={{ p: 2, bgcolor: '#064e3b', color: '#ecfdf5', borderRadius: 2 }}>
                            <pre style={{ margin: 0, fontSize: '0.8rem' }}>
{`query {
  ${endpoint.name}(limit: 10) {
    ${(Array.isArray(endpoint.fields) ? endpoint.fields : []).join('\n    ')}
  }
}`}
                            </pre>
                        </Paper>
                    </Box>
                )}
            </Grid>
        </Grid>
    );
};

export default APIPreview;
