import React, { useState, useEffect } from 'react';
import {
    Box,
    Typography,
    Paper,
    Stack,
    Button,
    Chip,
    Divider,
    Alert,
    List,
    ListItem,
    ListItemText,
    ListItemIcon,
    Collapse,
    CircularProgress
} from '@mui/material';
import {
    AutoAwesome as AiIcon,
    Upgrade as UpgradeIcon,
    Warning as ConflictIcon,
    CheckCircle as SuccessIcon,
    ExpandMore as ExpandMoreIcon,
    ExpandLess as ExpandLessIcon,
    History as HistoryIcon,
    Info as InfoIcon
} from '@mui/icons-material';
import { PageStudioApi } from '../../api/pageStudio';
import { UpgradeImpact, ConflictItem, ChangeItem } from '../../types/pageStudio';

export const TenantUpgradeAssistant: React.FC = () => {
    const [impacts, setImpacts] = useState<UpgradeImpact[]>([]);
    const [loading, setLoading] = useState(true);
    const [expanded, setExpanded] = useState<string | null>(null);
    const [decisions, setDecisions] = useState<Record<string, Record<string, string>>>({});

    const tenantId = 'tenant-123'; // Mock

    useEffect(() => {
        loadImpacts();
    }, []);

    const loadImpacts = async () => {
        try {
            const data = await PageStudioApi.getUpgradeImpacts(tenantId);
            setImpacts(data);
        } catch (err) {
            console.error('Failed to load upgrade impacts', err);
        } finally {
            setLoading(false);
        }
    };

    const handleDecision = (impactId: string, conflictId: string, decision: string) => {
        setDecisions(prev => ({
            ...prev,
            [impactId]: {
                ...(prev[impactId] || {}),
                [conflictId]: decision
            }
        }));
    };

    const handleFinalize = async (impact: UpgradeImpact) => {
        try {
            await PageStudioApi.applyUpgradeDecision(impact.id, decisions[impact.id]);
            loadImpacts();
        } catch (err) {
            console.error('Failed to finalize upgrade', err);
        }
    };

    if (loading) return <Box sx={{ p: 4, textAlign: 'center' }}><CircularProgress /></Box>;

    if (impacts.length === 0) {
        return (
            <Box sx={{ p: 4, textAlign: 'center' }}>
                <SuccessIcon color="success" sx={{ fontSize: 64, mb: 2 }} />
                <Typography variant="h6">All clear! No pending upgrades.</Typography>
                <Button variant="outlined" startIcon={<HistoryIcon />} sx={{ mt: 2 }}>View Upgrade History</Button>
            </Box>
        );
    }

    return (
        <Box sx={{ p: 3, maxWidth: 900, mx: 'auto' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
                <UpgradeIcon color="primary" sx={{ mr: 1, fontSize: 32 }} />
                <Typography variant="h5" fontWeight="bold">Tenant Upgrade Assistant</Typography>
                <Chip 
                    label={`${impacts.length} Upgrades Pending`} 
                    color="primary" 
                    size="small" 
                    sx={{ ml: 2, fontWeight: 'bold' }} 
                />
            </Box>

            <Stack spacing={3}>
                {impacts.map((impact) => (
                    <Paper key={impact.id} variant="outlined" sx={{ borderRadius: 3, overflow: 'hidden' }}>
                        <Box sx={{ p: 3, bgcolor: '#f8fafc', borderBottom: '1px solid', borderColor: 'divider' }}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                                <Box>
                                    <Typography variant="h6" fontWeight="bold">Page Upgrade: v{impact.coreOldVersion} &rarr; v{impact.coreNewVersion}</Typography>
                                    <Typography variant="body2" color="textSecondary">{impact.summary}</Typography>
                                </Box>
                                <Button 
                                    variant="contained" 
                                    color="success" 
                                    startIcon={<SuccessIcon />}
                                    disabled={Object.keys(decisions[impact.id] || {}).length < impact.conflicts.length}
                                    onClick={() => handleFinalize(impact)}
                                >
                                    Finalize Upgrades
                                </Button>
                            </Box>
                        </Box>

                        <List sx={{ p: 0 }}>
                            {/* Conflicts section */}
                            {impact.conflicts.length > 0 && (
                                <>
                                    <ListItem sx={{ bgcolor: 'error.lighter', py: 1 }}>
                                        <ListItemIcon><ConflictIcon color="error" fontSize="small" /></ListItemIcon>
                                        <ListItemText 
                                            primary={<Typography variant="subtitle2" color="error.dark" fontWeight="bold">Conflicts ({impact.conflicts.length})</Typography>} 
                                        />
                                    </ListItem>
                                    {impact.conflicts.map((conflict, idx) => {
                                        const cId = `conflict-${idx}`;
                                        const decision = decisions[impact.id]?.[cId] || 'keep-tenant';
                                        return (
                                            <Box key={cId} sx={{ p: 3, borderBottom: '1px solid', borderColor: 'divider' }}>
                                                <Typography variant="subtitle2" gutterBottom>
                                                    {conflict.type === 'componentProp' ? `Prop Change: ${conflict.propName}` : `Component: ${conflict.componentId}`}
                                                </Typography>
                                                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 2, mb: 2 }}>
                                                    <Paper variant="outlined" sx={{ p: 1, bgcolor: '#fff' }}>
                                                        <Typography variant="caption" fontWeight="bold">Core Before</Typography>
                                                        <pre style={{ fontSize: '10px', margin: 0 }}>{JSON.stringify(conflict.coreBefore, null, 2)}</pre>
                                                    </Paper>
                                                    <Paper variant="outlined" sx={{ p: 1, bgcolor: 'primary.lighter' }}>
                                                        <Typography variant="caption" fontWeight="bold">Core After</Typography>
                                                        <pre style={{ fontSize: '10px', margin: 0 }}>{JSON.stringify(conflict.coreAfter, null, 2)}</pre>
                                                    </Paper>
                                                    <Paper variant="outlined" sx={{ p: 1, bgcolor: 'warning.lighter' }}>
                                                        <Typography variant="caption" fontWeight="bold">Your Override</Typography>
                                                        <pre style={{ fontSize: '10px', margin: 0 }}>{JSON.stringify(conflict.tenantOverride, null, 2)}</pre>
                                                    </Paper>
                                                </Box>
                                                <Stack direction="row" spacing={1}>
                                                    <Button 
                                                        size="small" 
                                                        variant={decision === 'keep-tenant' ? 'contained' : 'outlined'}
                                                        onClick={() => handleDecision(impact.id, cId, 'keep-tenant')}
                                                    >
                                                        Keep My Override
                                                    </Button>
                                                    <Button 
                                                        size="small" 
                                                        variant={decision === 'adopt-core' ? 'contained' : 'outlined'}
                                                        color="primary"
                                                        onClick={() => handleDecision(impact.id, cId, 'adopt-core')}
                                                    >
                                                        Adopt Core Change
                                                    </Button>
                                                </Stack>
                                            </Box>
                                        );
                                    })}
                                </>
                            )}

                            {/* Inherited section */}
                            {impact.inheritedChanges.length > 0 && (
                                <>
                                    <ListItem 
                                        button 
                                        onClick={() => setExpanded(expanded === impact.id ? null : impact.id)}
                                        sx={{ bgcolor: 'success.lighter', py: 1 }}
                                    >
                                        <ListItemIcon><InfoIcon color="success" fontSize="small" /></ListItemIcon>
                                        <ListItemText 
                                            primary={<Typography variant="subtitle2" color="success.dark" fontWeight="bold">Safe Inherited Changes ({impact.inheritedChanges.length})</Typography>} 
                                        />
                                        {expanded === impact.id ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                                    </ListItem>
                                    <Collapse in={expanded === impact.id}>
                                        <Box sx={{ p: 2 }}>
                                            {impact.inheritedChanges.map((change, idx) => (
                                                <Typography key={idx} variant="caption" display="block" color="textSecondary">
                                                    &bull; {change.type}: {change.componentId || 'Layout'} updated from core.
                                                </Typography>
                                            ))}
                                        </Box>
                                    </Collapse>
                                </>
                            )}
                        </List>
                    </Paper>
                ))}
            </Stack>
        </Box>
    );
};
