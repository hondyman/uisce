import React from 'react';
import {
    Box,
    Typography,
    Paper,
    Stack,
    Alert,
    AlertTitle,
    Button,
    Chip,
    LinearProgress,
    Divider
} from '@mui/material';
import {
    Speed as PerformanceIcon,
    AutoAwesome as AiIcon,
    FlashOn as SuggestionIcon,
    QueryStats as QueryIcon,
    Storage as CacheIcon
} from '@mui/icons-material';
import { CorePageDefinition } from '../../types/pageStudio';

interface AIPerformanceAdvisorProps {
    page: CorePageDefinition;
}

const RECOMMENDATIONS = [
    {
        id: 'pre-agg',
        type: 'latency',
        title: 'Suggest Pre-aggregation',
        description: 'Selected BO has high cardinality. Moving KPI computation to a pre-agg will reduce render time by ~800ms.',
        impact: 'High',
        effort: 'Low',
        icon: <PerformanceIcon color="error" />
    },
    {
        id: 'graphql-merge',
        type: 'fanout',
        title: 'Merge API Calls',
        description: 'Page has 4 separate REST calls. Merging into a single GraphQL query reduces network overhead and improves mobile experience.',
        impact: 'Medium',
        effort: 'Medium',
        icon: <QueryIcon color="primary" />
    },
    {
        id: 'caching',
        type: 'cache',
        title: 'Enable Data Caching',
        description: 'Source data changes infrequently. Enabling a 5-minute TTL will improve repeated access performance and reduce database load.',
        impact: 'Medium',
        effort: 'Very Low',
        icon: <CacheIcon color="success" />
    }
];

export const AIPerformanceAdvisor: React.FC<AIPerformanceAdvisorProps> = ({ page }) => {
    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
                <AiIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6" fontWeight="bold">AI Performance Advisor</Typography>
            </Box>

            <Grid container spacing={3}>
                <Grid item xs={12} md={4}>
                    <Paper variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                        <Typography variant="overline" color="textSecondary">Est. Page Load</Typography>
                        <Typography variant="h4" fontWeight="bold" sx={{ color: 'warning.main' }}>1.2s</Typography>
                        <Typography variant="caption" sx={{ display: 'block', mb: 1 }}>Target: 1.0s</Typography>
                        <LinearProgress variant="determinate" value={80} color="warning" sx={{ height: 8, borderRadius: 4 }} />
                    </Paper>
                </Grid>
                <Grid item xs={12} md={4}>
                    <Paper variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                        <Typography variant="overline" color="textSecondary">API Fanout</Typography>
                        <Typography variant="h4" fontWeight="bold" sx={{ color: 'error.main' }}>4</Typography>
                        <Typography variant="caption" sx={{ display: 'block', mb: 1 }}>Target: &le; 2</Typography>
                        <LinearProgress variant="determinate" value={100} color="error" sx={{ height: 8, borderRadius: 4 }} />
                    </Paper>
                </Grid>
                <Grid item xs={12} md={4}>
                    <Paper variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                        <Typography variant="overline" color="textSecondary">Pre-agg Hits</Typography>
                        <Typography variant="h4" fontWeight="bold" sx={{ color: 'success.main' }}>95%</Typography>
                        <Typography variant="caption" sx={{ display: 'block', mb: 1 }}>Optimized</Typography>
                        <LinearProgress variant="determinate" value={95} color="success" sx={{ height: 8, borderRadius: 4 }} />
                    </Paper>
                </Grid>

                <Grid item xs={12}>
                    <Typography variant="subtitle2" sx={{ mt: 2, mb: 1, fontWeight: 'bold' }}>AI Recommendations</Typography>
                    <Stack spacing={2}>
                        {RECOMMENDATIONS.map((rec) => (
                            <Paper 
                                key={rec.id} 
                                variant="outlined" 
                                sx={{ 
                                    p: 2, 
                                    borderRadius: 3, 
                                    display: 'flex', 
                                    gap: 2,
                                    borderLeft: '4px solid',
                                    borderLeftColor: rec.impact === 'High' ? 'error.main' : 'primary.main',
                                    '&:hover': { bgcolor: 'rgba(0,0,0,0.01)' }
                                }}
                            >
                                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', width: 48, height: 48, borderRadius: 2, bgcolor: 'rgba(0,0,0,0.03)' }}>
                                    {rec.icon}
                                </Box>
                                <Box sx={{ flex: 1 }}>
                                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                                        <Typography variant="subtitle1" fontWeight="bold">{rec.title}</Typography>
                                        <Box sx={{ display: 'flex', gap: 1 }}>
                                            <Chip label={`Impact: ${rec.impact}`} size="small" color={rec.impact === 'High' ? 'error' : 'primary'} variant="outlined" />
                                            <Chip label={`Effort: ${rec.effort}`} size="small" variant="outlined" />
                                        </Box>
                                    </Box>
                                    <Typography variant="body2" color="textSecondary">{rec.description}</Typography>
                                    <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
                                        <Button variant="contained" size="small" startIcon={<FlashOnIcon />} sx={{ textTransform: 'none' }}>
                                            Apply Optimization
                                        </Button>
                                        <Button variant="outlined" size="small" sx={{ textTransform: 'none' }}>
                                            Learn More
                                        </Button>
                                    </Box>
                                </Box>
                            </Paper>
                        ))}
                    </Stack>
                </Grid>
            </Grid>
        </Box>
    );
};

// Internal Mock Grid
const Grid = ({ children, container, spacing, item, xs, md }: any) => (
    <Box sx={{ 
        display: container ? 'flex' : 'block', 
        flexWrap: container ? 'wrap' : 'nowrap',
        m: container ? -(spacing || 0) * 1 : 0,
        width: item ? (md ? `${(md / 12) * 100}%` : (xs ? `${(xs / 12) * 100}%` : '100%')) : 'auto',
        p: item ? (spacing || 0) * 1 : 0
    }}>
        {children}
    </Box>
);

const FlashOnIcon = () => <SuggestionIcon sx={{ fontSize: 16 }} />;
