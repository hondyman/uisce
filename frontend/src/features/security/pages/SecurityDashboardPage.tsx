import React, { useEffect } from 'react';
import { Box, Typography, Grid, Paper, CircularProgress, Button } from '@mui/material';
import { 
    Security as SecurityIcon, 
    Group as GroupIcon, 
    VpnKey as RoleIcon, 
    NotificationsActive as AlertIcon, 
    Sync as SyncIcon,
    Refresh as RefreshIcon
} from '@mui/icons-material';
import { useSecurityStats } from '../hooks/useSecurityStats';
import { format } from 'date-fns';

export const SecurityDashboardPage: React.FC = () => {
    const { stats, loading, error, fetchStats } = useSecurityStats();

    useEffect(() => {
        fetchStats();
    }, [fetchStats]);

    if (loading && !stats) {
        return <Box sx={{ display: 'flex', justifyContent: 'center', p: 5 }}><CircularProgress /></Box>;
    }

    if (error) {
        return (
             <Box sx={{ p: 3 }}>
                <Typography color="error">{error}</Typography>
                <Button onClick={() => fetchStats()}>Retry</Button>
            </Box>
        );
    }

    if (!stats) return null;

    const MetricCard = ({ title, value, icon, color, subtext }: any) => (
        <Paper sx={{ p: 3, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
                <Typography variant="body2" color="textSecondary">{title}</Typography>
                <Typography variant="h4" sx={{ fontWeight: 'bold', my: 1 }}>{value}</Typography>
                {subtext && <Typography variant="caption" color="textSecondary">{subtext}</Typography>}
            </Box>
            <Box sx={{ p: 1.5, borderRadius: 2, bgcolor: `${color}.light`, color: `${color}.main` }}>
                {icon}
            </Box>
        </Paper>
    );

    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                <Typography variant="h4">Security Dashboard</Typography>
                <Button startIcon={<RefreshIcon />} onClick={() => fetchStats()}>Refresh</Button>
            </Box>

            <Grid container spacing={3}>
                <Grid item xs={12} sm={6} md={3}>
                    <MetricCard 
                        title="Total Users" 
                        value={stats.total_users} 
                        icon={<GroupIcon fontSize="large" />} 
                        color="primary" 
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <MetricCard 
                        title="Active Sessions" 
                        value={stats.active_sessions} 
                        icon={<SecurityIcon fontSize="large" />} 
                        color="success" 
                        subtext="Last 24 hours"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <MetricCard 
                        title="Active Roles" 
                        value={stats.active_roles} 
                        icon={<RoleIcon fontSize="large" />} 
                        color="info" 
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <MetricCard 
                        title="Recent Alerts" 
                        value={stats.recent_alerts} 
                        icon={<AlertIcon fontSize="large" />} 
                        color="warning" 
                        subtext="Last 24 hours"
                    />
                </Grid>
                
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                         <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                            <SyncIcon color={stats.sync_status === 'healthy' ? 'success' : 'error'} />
                            <Typography variant="h6">Sync Status</Typography>
                        </Box>
                        <Typography variant="body1">
                            System Health:  
                            <Box component="span" sx={{ 
                                fontWeight: 'bold', 
                                color: stats.sync_status === 'healthy' ? 'success.main' : 'error.main', 
                                ml: 1 
                            }}>
                                {stats.sync_status.toUpperCase()}
                            </Box>
                        </Typography>
                        <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
                            Last Successful Sync: {stats.last_sync_time ? format(new Date(stats.last_sync_time), 'yyyy-MM-dd HH:mm:ss') : 'N/A'}
                        </Typography>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};
