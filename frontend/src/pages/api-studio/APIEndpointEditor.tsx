import React, { useState, useEffect } from 'react';
import { Box, Typography, Button, TextField, Select, MenuItem, FormControl, InputLabel, Checkbox, FormControlLabel, FormGroup, Grid, Paper, Divider, Tabs, Tab, Chip } from '@mui/material';
import { Save as SaveIcon, Close as CloseIcon, Visibility as PreviewIcon, Assessment as AnalyticsIcon, Warning as WarningIcon, DeleteForever as RetireIcon, AutoAwesome as AIIcon } from '@mui/icons-material';
import { APIEndpoint } from '../../types/apiStudio';
import APIPreview from './APIPreview';
import UsageDashboard from './UsageDashboard';
import { ApiStudioApi } from '../../api/apiStudio';

interface APIEndpointEditorProps {
    endpoint: Partial<APIEndpoint>;
    bo?: any;
    onSave: (ep: Partial<APIEndpoint>) => void;
    onCancel: () => void;
}

const APIEndpointEditor: React.FC<APIEndpointEditorProps> = ({ endpoint, bo, onSave, onCancel }) => {
    const [form, setForm] = useState<Partial<APIEndpoint>>(endpoint);
    const [selectedFields, setSelectedFields] = useState<string[]>([]);
    const [activeTab, setActiveTab] = useState(0);

    useEffect(() => {
        setForm(endpoint);
        if (endpoint.fields && typeof endpoint.fields === 'string') {
            try {
                setSelectedFields(JSON.parse(endpoint.fields));
            } catch {
                setSelectedFields([]);
            }
        } else if (Array.isArray(endpoint.fields)) {
            setSelectedFields(endpoint.fields);
        }
    }, [endpoint]);

    const handleToggleField = (field: string) => {
        const next = selectedFields.includes(field)
            ? selectedFields.filter(f => f !== field)
            : [...selectedFields, field];
        setSelectedFields(next);
        setForm({ ...form, fields: next as any });
    };

    const handleSave = () => {
        onSave({ ...form, fields: selectedFields as any });
    };

    const handleDeprecate = async () => {
        if (!form.id || !window.confirm("Are you sure you want to deprecate this endpoint? Consumers will receive warning headers.")) return;
        try {
            await ApiStudioApi.deprecateEndpoint(form.id);
            setForm({ ...form, status: 'deprecated', is_active: true });
        } catch (e) { console.error(e); }
    };

    const handleRetire = async () => {
        if (!form.id || !window.confirm("Are you sure you want to RETIRE this endpoint? It will be blocked for all consumers.")) return;
        try {
            await ApiStudioApi.retireEndpoint(form.id);
            setForm({ ...form, status: 'retired', is_active: false });
        } catch (e) { console.error(e); }
    };

    return (
        <Paper elevation={0} sx={{ flex: 1, p: 3, borderRadius: 4, background: 'rgba(255, 255, 255, 0.8)', backdropFilter: 'blur(10px)', border: '1px solid rgba(255, 255, 255, 0.3)', display: 'flex', flexDirection: 'column' }}>
            {!form.id && (
                <Box sx={{ bgcolor: 'info.light', p: 1.5, mb: 2, borderRadius: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                    <AIIcon color="info" />
                    <Typography variant="body2" color="info.dark" fontWeight="500">
                        AI Design Proposal: Please review the configuration below. You can refine any field before saving as a permanent asset.
                    </Typography>
                </Box>
            )}
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                <Box>
                    <Typography variant="h5" fontWeight="bold">Configuring Endpoint: {form.name}</Typography>
                    <Box sx={{ display: 'flex', gap: 1, mt: 0.5 }}>
                        <Typography variant="caption" color="textSecondary">{form.id}</Typography>
                         {form.status && (
                            <Chip 
                                label={form.status.toUpperCase()} 
                                color={form.status === 'active' ? 'success' : form.status === 'deprecated' ? 'warning' : 'error'} 
                                size="small" 
                            />
                        )}
                    </Box>
                </Box>
                <Box>
                    <Button startIcon={<CloseIcon />} onClick={onCancel} sx={{ mr: 1 }}>Cancel</Button>
                    <Button variant="contained" startIcon={<SaveIcon />} onClick={handleSave}>Save Asset</Button>
                </Box>
            </Box>

            <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ mb: 2 }}>
                <Tab label="Configuration" />
                <Tab label="Data Preview & Spec" />
                <Tab label="Usage & Governance" icon={<AnalyticsIcon fontSize="small" />} iconPosition="start" />
            </Tabs>

            <Box sx={{ flex: 1, overflowY: 'auto' }}>
                {activeTab === 0 && (
                    <Grid container spacing={4}>
                        <Grid item xs={12} md={6}>
                            <Typography variant="subtitle2" gutterBottom>Basic Information</Typography>
                            <TextField 
                                fullWidth label="Endpoint Name" variant="outlined" sx={{ mb: 2 }}
                                value={form.name} onChange={e => setForm({ ...form, name: e.target.value })}
                            />
                            <TextField 
                                fullWidth label="Relative Path" variant="outlined" sx={{ mb: 2 }}
                                value={form.path} onChange={e => setForm({ ...form, path: e.target.value })}
                                helperText="Example: /api/v1/positions"
                            />
                            
                            <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
                                <FormControl fullWidth>
                                    <InputLabel>Method</InputLabel>
                                    <Select 
                                        value={form.method} label="Method"
                                        onChange={e => setForm({ ...form, method: e.target.value as any })}
                                    >
                                        <MenuItem value="GET">GET (Query)</MenuItem>
                                        <MenuItem value="POST">POST (Mutation/Legacy)</MenuItem>
                                    </Select>
                                </FormControl>
                                <FormControl fullWidth>
                                    <InputLabel>Type</InputLabel>
                                    <Select 
                                        value={form.type} label="Type"
                                        onChange={e => setForm({ ...form, type: e.target.value as any })}
                                    >
                                        <MenuItem value="rest">REST API</MenuItem>
                                        <MenuItem value="graphql">GraphQL Field</MenuItem>
                                    </Select>
                                </FormControl>
                            </Box>
                        </Grid>

                        <Grid item xs={12} md={6}>
                            <Typography variant="subtitle2" gutterBottom>Field Selection ({bo?.name})</Typography>
                            <Box sx={{ p: 2, border: '1px solid rgba(0,0,0,0.1)', borderRadius: 2, maxHeight: 300, overflowY: 'auto' }}>
                                <FormGroup>
                                    {bo?.fields?.map((f: any) => (
                                        <FormControlLabel 
                                            key={f.key}
                                            control={<Checkbox checked={selectedFields.includes(f.key)} onChange={() => handleToggleField(f.key)} />} 
                                            label={`${f.displayName || f.name} (${f.key})`} 
                                        />
                                    ))}
                                </FormGroup>
                            </Box>
                        </Grid>
                    </Grid>
                )}
                
                {activeTab === 1 && (
                    <APIPreview endpoint={form as any} />
                )}

                {activeTab === 2 && (
                    <Box>
                        <UsageDashboard endpoint={form} />
                        <Divider sx={{ my: 4 }} />
                        <Box sx={{ p: 2, border: '1px solid #ff9800', borderRadius: 2, bgcolor: '#fff3e0' }}>
                            <Typography variant="h6" gutterBottom color="warning.dark" sx={{ display: 'flex', alignItems: 'center' }}>
                                <WarningIcon sx={{ mr: 1 }} /> Lifecycle Management
                            </Typography>
                            <Typography variant="body2" paragraph>
                                Manage the lifecycle state of this API endpoint. Changing state affects all consumers.
                            </Typography>
                            <Box sx={{ display: 'flex', gap: 2 }}>
                                <Button 
                                    variant="outlined" 
                                    color="warning" 
                                    onClick={handleDeprecate}
                                    disabled={form.status !== 'active'}
                                >
                                    Deprecate Endpoint
                                </Button>
                                <Button 
                                    variant="contained" 
                                    color="error" 
                                    startIcon={<RetireIcon />}
                                    onClick={handleRetire}
                                    disabled={form.status === 'retired'}
                                >
                                    Retire Endpoint
                                </Button>
                            </Box>
                        </Box>
                    </Box>
                )}
            </Box>
        </Paper>
    );
};

export default APIEndpointEditor;
