import React, { useState, useEffect } from 'react';
import { Box, Typography, Button, List, ListItem, ListItemText, ListItemSecondaryAction, IconButton, Paper, Divider, Chip, TextField, Select, MenuItem, FormControl, InputLabel } from '@mui/material';
import { Add as AddIcon, Edit as EditIcon, PlayArrow as PlayIcon, Code as CodeIcon, Storage as StorageIcon, CloudQueue as ApiIcon } from '@mui/icons-material';
import { ApiStudioApi } from '../../api/apiStudio';
import { APIEndpoint } from '../../types/apiStudio';
import axios from 'axios';
import { apiClient } from '../../utils/apiClient';
import APIEndpointEditor from './APIEndpointEditor';
import APIPreview from './APIPreview';
import NLDesignInterface from './NLDesignInterface';
import { AutoAwesome as AIIcon } from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';

const APIStudioPage: React.FC = () => {
    const { tenant } = useTenant();
    const [bos, setBos] = useState<any[]>([]);
    const [endpoints, setEndpoints] = useState<APIEndpoint[]>([]);
    const [selectedBO, setSelectedBO] = useState<any | null>(null);
    const [editingEndpoint, setEditingEndpoint] = useState<Partial<APIEndpoint> | null>(null);
    const [loading, setLoading] = useState(true);
    const [showAI, setShowAI] = useState(false);

    // Mock env for now, should come from context
    const env = 'production';
    const tenantId = tenant?.id || 'default';

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setLoading(true);
        try {
            const [boList, epList] = await Promise.all([
                apiClient<any[]>('/business-objects'),
                ApiStudioApi.listEndpoints(env, tenantId)
            ]);
            setBos(Array.isArray(boList) ? boList : []);
            setEndpoints(Array.isArray(epList) ? epList : []);
        } catch (err) {
            console.error('Failed to load API Studio data', err);
        } finally {
            setLoading(false);
        }
    };

    const handleNewEndpoint = (bo: any) => {
        setEditingEndpoint({
            env,
            tenant_id: tenantId,
            bo_name: bo.name,
            name: `${bo.name} API`,
            path: `/api/v1/${bo.name.toLowerCase()}`,
            method: 'GET',
            type: 'rest',
            fields: [],
            filters: {},
            pagination: { type: 'offset', default_limit: 100 }
        });
    };

    const handleSave = async (ep: Partial<APIEndpoint>) => {
        try {
            await ApiStudioApi.saveEndpoint(ep);
            loadData();
            setEditingEndpoint(null);
        } catch (err) {
            alert('Failed to save endpoint');
        }
    };

    return (
        <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', background: 'linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)', p: 3 }}>
            {/* Sidebar */}
            <Paper elevation={0} sx={{ width: 300, mr: 3, borderRadius: 4, background: 'rgba(255, 255, 255, 0.7)', backdropFilter: 'blur(10px)', border: '1px solid rgba(255, 255, 255, 0.3)', display: 'flex', flexDirection: 'column' }}>
                <Box sx={{ p: 2 }}>
                    <Typography variant="h6" fontWeight="bold">API Studio</Typography>
                    <Typography variant="caption" color="textSecondary">Self-Service Governed APIs</Typography>
                </Box>
                <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <Button 
                        variant={showAI ? "contained" : "outlined"} 
                        fullWidth 
                        startIcon={<AIIcon />} 
                        onClick={() => {
                            setShowAI(!showAI);
                            setEditingEndpoint(null);
                        }}
                        sx={{ borderRadius: 3 }}
                    >
                        AI Design
                    </Button>
                </Box>
                <Divider />
                
                <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
                    {!showAI ? (
                        <>
                            <Typography variant="overline" color="primary" fontWeight="bold">Business Objects</Typography>
                            <List dense>
                                {bos.map(bo => (
                                    <ListItem key={bo.id} button onClick={() => setSelectedBO(bo)}>
                                        <StorageIcon fontSize="small" sx={{ mr: 1, color: 'primary.main' }} />
                                        <ListItemText primary={bo.name} />
                                        <ListItemSecondaryAction>
                                            <IconButton size="small" onClick={() => handleNewEndpoint(bo)}>
                                                <AddIcon fontSize="small" />
                                            </IconButton>
                                        </ListItemSecondaryAction>
                                    </ListItem>
                                ))}
                            </List>

                            <Divider sx={{ my: 2 }} />
                            <Typography variant="overline" color="secondary" fontWeight="bold">Existing Endpoints</Typography>
                            <List dense>
                                {endpoints.map(ep => (
                                    <ListItem key={ep.id} button onClick={() => setEditingEndpoint(ep)}>
                                        <ApiIcon fontSize="small" sx={{ mr: 1, color: 'secondary.main' }} />
                                        <ListItemText primary={ep.name} secondary={ep.path} />
                                        <ListItemSecondaryAction>
                                            <IconButton size="small" onClick={() => setEditingEndpoint(ep)}>
                                                <EditIcon fontSize="small" />
                                            </IconButton>
                                        </ListItemSecondaryAction>
                                    </ListItem>
                                ))}
                            </List>
                        </>
                    ) : (
                        <Typography variant="body2" color="textSecondary" sx={{ px: 1 }}>
                            The AI Design Assistant is active. Describe your requirements in the chat to generate a proposal.
                        </Typography>
                    )}
                </Box>
            </Paper>

            {/* Main Content */}
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
                {editingEndpoint ? (
                    <APIEndpointEditor 
                        endpoint={editingEndpoint} 
                        bo={bos.find(b => b.name === editingEndpoint.bo_name)} 
                        onSave={handleSave}
                        onCancel={() => setEditingEndpoint(null)}
                    />
                ) : showAI ? (
                    <NLDesignInterface 
                        tenantId={tenantId}
                        onProposalGenerated={(proposal) => setEditingEndpoint(proposal)}
                    />
                ) : (
                    <Paper sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', borderRadius: 4, background: 'rgba(255, 255, 255, 0.5)', backdropFilter: 'blur(5px)' }}>
                        <Box sx={{ textAlign: 'center' }}>
                            <CodeIcon sx={{ fontSize: 60, color: 'text.disabled', mb: 2 }} />
                            <Typography variant="h5" color="textSecondary">Select a Business Object or Endpoint to begin</Typography>
                            <Typography variant="body2" color="textSecondary">Define governed REST/GraphQL surfaces over your semantic layer</Typography>
                        </Box>
                    </Paper>
                )}
            </Box>
        </Box>
    );
};

export default APIStudioPage;
