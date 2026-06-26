import React, { useState, useEffect } from 'react';
import { Box, Typography, TextField, MenuItem, Divider, Switch, FormControlLabel, Button, Chip, Dialog, DialogTitle, DialogContent, List, ListItem, ListItemText, ListItemIcon, Paper, IconButton } from '@mui/material';
import { 
    Link as BindIcon, 
    Visibility as ViewIcon, 
    Settings as ConfigIcon, 
    LinkOff as UnbindIcon,
    Api as ApiIcon,
    Storage as GqlIcon,
    AutoAwesome as AiIcon
} from '@mui/icons-material';
import { CorePageDefinition, ComponentDefinition, DataSourceDefinition } from '../../types/pageStudio';
import { ApiStudioApi } from '../../api/apiStudio';
import { APIEndpoint } from '../../types/apiStudio';

interface PropertiesPanelProps {
    selectedId: string | null;
    draft: CorePageDefinition;
    setDraft: (d: CorePageDefinition) => void;
    tenantId: string;
}

const PropertiesPanel: React.FC<PropertiesPanelProps> = ({ selectedId, draft, setDraft, tenantId }) => {
    const [isBindingOpen, setIsBindingOpen] = useState(false);
    const [endpoints, setEndpoints] = useState<APIEndpoint[]>([]);

    useEffect(() => {
        if (isBindingOpen) {
            loadEndpoints();
        }
    }, [isBindingOpen]);

    const loadEndpoints = async () => {
        try {
            const env = 'production';
            const data = await ApiStudioApi.listEndpoints(env, tenantId);
            setEndpoints(data);
        } catch (err) {
            console.error('Failed to load endpoints', err);
        }
    };

    if (!selectedId) return (
        <Box sx={{ p: 4, textAlign: 'center', opacity: 0.5 }}>
            <ConfigIcon sx={{ fontSize: 40, mb: 2 }} />
            <Typography variant="body2">Select an element to edit properties</Typography>
        </Box>
    );

    const component = draft.components[selectedId];
    const node = draft.layout.nodes[selectedId];

    const handlePropChange = (key: string, value: any) => {
        const newDraft = { ...draft };
        if (component) {
            newDraft.components[selectedId].props[key] = value;
        } else if (node) {
            newDraft.layout.nodes[selectedId].props = { ...(newDraft.layout.nodes[selectedId].props || {}), [key]: value };
        }
        setDraft(newDraft);
    };

    const handleBindSource = (ep: APIEndpoint) => {
        const newDraft = { ...draft };
        const sourceId = `${ep.name}_source`;
        
        // Add to sources
        newDraft.dataBindings.sources[sourceId] = {
            id: sourceId,
            type: ep.type as 'rest' | 'graphql',
            endpointId: ep.id,
            query: ep.type === 'graphql' ? ep.name : undefined,
            args: {}
        };

        // Bind to component (assuming primary prop like 'rows' or 'data')
        const propName = component?.type === 'Table' ? 'rows' : 'data';
        newDraft.dataBindings.bindings = [
            ...newDraft.dataBindings.bindings.filter(b => b.componentId !== selectedId || b.prop !== propName),
            { componentId: selectedId, prop: propName, sourceId, path: '$' }
        ];

        setDraft(newDraft);
        setIsBindingOpen(false);
    };

    const handleAutoBind = async () => {
        // AI Heuristic: Find first endpoint that matches component name or type
        if (endpoints.length === 0) await loadEndpoints();
        const ep = endpoints[0]; // Simple mock
        if (ep) handleBindSource(ep);
    };

    const activeBinding = draft.dataBindings.bindings.find(b => b.componentId === selectedId);
    const activeSource = activeBinding ? draft.dataBindings.sources[activeBinding.sourceId] : null;

    return (
        <Box sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <ConfigIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="subtitle1" fontWeight="bold">Properties</Typography>
                <Chip label={selectedId} size="small" variant="outlined" sx={{ ml: 'auto', fontSize: '10px' }} />
            </Box>
            <Divider sx={{ mb: 2 }} />

            {component && (
                <>
                    <Typography variant="overline" color="textSecondary">Configuration</Typography>
                    <TextField 
                        fullWidth 
                        label="Component ID" 
                        variant="outlined" 
                        size="small" 
                        value={component.id} 
                        sx={{ mt: 1, mb: 2 }} 
                    />
                    
                    {component.type === 'Table' && (
                        <>
                            <TextField 
                                select 
                                fullWidth 
                                label="Page Size" 
                                size="small" 
                                value={component.props.pageSize || 10} 
                                onChange={(e) => handlePropChange('pageSize', e.target.value)}
                                sx={{ mb: 2 }}
                            >
                                <MenuItem value={10}>10 items</MenuItem>
                                <MenuItem value={25}>25 items</MenuItem>
                                <MenuItem value={50}>50 items</MenuItem>
                            </TextField>
                        </>
                    )}

                    <Divider sx={{ my: 2 }} />
                    <Typography variant="overline" color="textSecondary">Data Binding</Typography>
                    
                    {activeSource ? (
                        <Paper sx={{ p: 2, bgcolor: 'primary.main', color: 'white', borderRadius: 2, mb: 2 }}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                                    {activeSource.type === 'graphql' ? <GqlIcon sx={{ mr: 1, fontSize: 20 }} /> : <ApiIcon sx={{ mr: 1, fontSize: 20 }} />}
                                    <Box>
                                        <Typography variant="body2" fontWeight="bold">{activeSource.id}</Typography>
                                        <Typography variant="caption" sx={{ opacity: 0.8 }}>Bound to {activeBinding?.prop}</Typography>
                                    </Box>
                                </Box>
                                <IconButton size="small" sx={{ color: 'white' }} onClick={() => {
                                    const newDraft = { ...draft };
                                    newDraft.dataBindings.bindings = newDraft.dataBindings.bindings.filter(b => b.componentId !== selectedId);
                                    setDraft(newDraft);
                                }}>
                                    <UnbindIcon fontSize="small" />
                                </IconButton>
                            </Box>
                        </Paper>
                    ) : (
                        <Button 
                            fullWidth 
                            variant="outlined" 
                            startIcon={<BindIcon />} 
                            size="small"
                            onClick={() => setIsBindingOpen(true)}
                            sx={{ mt: 1, mb: 2, textTransform: 'none' }}
                        >
                            Connect Data Source
                        </Button>
                    )}

                    <Button
                        fullWidth
                        variant="contained"
                        startIcon={<AiIcon />}
                        size="small"
                        onClick={handleAutoBind}
                        sx={{ 
                            mt: 1, 
                            textTransform: 'none', 
                            background: 'rgba(59, 130, 246, 0.1)',
                            color: 'primary.main',
                            boxShadow: 'none',
                            border: '1px dashed',
                            borderColor: 'primary.main',
                            '&:hover': {
                                background: 'rgba(59, 130, 246, 0.2)',
                                boxShadow: 'none',
                            }
                        }}
                    >
                        Auto-bind with AI
                    </Button>
                </>
            )}

            <Dialog open={isBindingOpen} onClose={() => setIsBindingOpen(false)} maxWidth="sm" fullWidth>
                <DialogTitle>Select Data Source</DialogTitle>
                <DialogContent>
                    <List>
                        {endpoints.map((ep) => (
                            <ListItem button key={ep.id} onClick={() => handleBindSource(ep)}>
                                <ListItemIcon>
                                    {ep.type === 'graphql' ? <GqlIcon color="secondary" /> : <ApiIcon color="primary" />}
                                </ListItemIcon>
                                <ListItemText 
                                    primary={ep.name} 
                                    secondary={`${ep.method} ${ep.path}`} 
                                />
                                <Chip label={ep.type} size="small" variant="outlined" />
                            </ListItem>
                        ))}
                    </List>
                </DialogContent>
            </Dialog>
        </Box>
    );
};

export default PropertiesPanel;
