import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    Box,
    Typography,
    Container,
    Paper,
    Grid,
    CircularProgress,
    Stack,
    Button,
    Chip,
    Divider,
    IconButton,
    Alert,
    FormControlLabel,
    Switch,
    TextField,
    MenuItem,
} from '@mui/material';
import { MappingModal } from './components/MappingModal';
import { catalogApi, BusinessTermCompliance } from '../../api/catalogApi';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SaveIcon from '@mui/icons-material/Save';
import LinkIcon from '@mui/icons-material/Link';
import DeleteIcon from '@mui/icons-material/Delete';

export const BusinessTermDetailPage: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const [term, setTerm] = useState<BusinessTermCompliance | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);
    
    // Modal state
    const [mappingModalOpen, setMappingModalOpen] = useState(false);

    // Form state
    const [formData, setFormData] = useState({
        piiFlag: false,
        residency: '',
        sensitivity: '',
    });

    const fetchTerm = async () => {
        if (!id) return;
        setLoading(true);
        try {
            const data = await catalogApi.getBusinessTerm(id);
            setTerm(data);
            setFormData({
                piiFlag: data.piiFlag,
                residency: data.residency || 'USA',
                sensitivity: data.sensitivity || 'INTERNAL',
            });
            setError(null);
        } catch (err) {
            console.error(err);
            setError('Failed to load business term details.');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchTerm();
    }, [id]);

    const handleSave = async () => {
        if (!id) return;
        setSaving(true);
        try {
            await catalogApi.updateCompliance(id, formData);
            // Refresh to confirm content
            await fetchTerm();
        } catch (err) {
            setError('Failed to save changes.');
        } finally {
            setSaving(false);
        }
    };

    const handleRemoveMapping = async (semId: string) => {
        if (!id) return;
        if (!window.confirm('Are you sure you want to remove this mapping?')) return;
        
        try {
            await catalogApi.removeMapping(id, semId);
            fetchTerm();
        } catch (err) {
            setError('Failed to remove mapping.');
        }
    };

    if (loading) {
        return (
            <Box display="flex" justifyContent="center" height="50vh" alignItems="center">
                <CircularProgress />
            </Box>
        );
    }

    if (!term) {
        return (
            <Container maxWidth="lg" sx={{ mt: 4 }}>
                <Alert severity="error">Business term not found</Alert>
                <Button startIcon={<ArrowBackIcon />} onClick={() => navigate(-1)} sx={{ mt: 2 }}>
                    Go Back
                </Button>
            </Container>
        );
    }

    return (
        <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
            <Button
                startIcon={<ArrowBackIcon />}
                onClick={() => navigate('/core/glossary')} 
                sx={{ mb: 2 }}
                color="inherit"
            >
                Back to Glossary
            </Button>

            {error && (
                <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
                    {error}
                </Alert>
            )}

            <Stack direction="row" justifyContent="space-between" alignItems="flex-start" mb={3}>
                <Box>
                    <Typography variant="h4" fontWeight="bold" gutterBottom>
                        {term.name}
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        {term.description}
                    </Typography>
                </Box>
                <Button
                    variant="contained"
                    startIcon={<SaveIcon />}
                    onClick={handleSave}
                    disabled={saving}
                >
                    {saving ? 'Saving...' : 'Save Changes'}
                </Button>
            </Stack>

            <Grid container spacing={3}>
                {/* Left Column: Compliance Controls */}
                <Grid item xs={12} md={7}>
                    <Paper sx={{ p: 3, mb: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            Compliance & Governance
                        </Typography>
                        <Divider sx={{ mb: 3 }} />
                        
                        <Stack spacing={3}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={formData.piiFlag}
                                        onChange={(e) => setFormData({ ...formData, piiFlag: e.target.checked })}
                                        color="error" // Red for PII
                                    />
                                }
                                label={
                                    <Box>
                                        <Typography variant="subtitle1" fontWeight="medium">
                                            Contains PII
                                        </Typography>
                                        <Typography variant="caption" color="text.secondary">
                                            Does this term represent Personally Identifiable Information?
                                        </Typography>
                                    </Box>
                                }
                            />

                            <TextField
                                select
                                label="Data Residency"
                                value={formData.residency}
                                onChange={(e) => setFormData({ ...formData, residency: e.target.value })}
                                fullWidth
                                helperText="Region where this data must reside"
                            >
                                <MenuItem value="USA">USA (United States)</MenuItem>
                                <MenuItem value="EU">EU (European Union)</MenuItem>
                                <MenuItem value="APAC">APAC (Asia Pacific)</MenuItem>
                                <MenuItem value="GLOBAL">Global (No Restriction)</MenuItem>
                            </TextField>

                            <TextField
                                select
                                label="Data Sensitivity"
                                value={formData.sensitivity}
                                onChange={(e) => setFormData({ ...formData, sensitivity: e.target.value })}
                                fullWidth
                                helperText="Classification level for access control"
                            >
                                <MenuItem value="PUBLIC">Public</MenuItem>
                                <MenuItem value="INTERNAL">Internal</MenuItem>
                                <MenuItem value="CONFIDENTIAL">Confidential</MenuItem>
                                <MenuItem value="RESTRICTED">Restricted</MenuItem>
                            </TextField>
                        </Stack>
                    </Paper>
                </Grid>

                {/* Right Column: Semantic Mappings */}
                <Grid item xs={12} md={5}>
                    <Paper sx={{ p: 3 }}>
                        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                            <Typography variant="h6">
                                Semantic Mappings
                            </Typography>
                            <Button
                                startIcon={<LinkIcon />}
                                size="small"
                                onClick={() => setMappingModalOpen(true)}
                            >
                                Add Mapping
                            </Button>
                        </Stack>
                        <Divider sx={{ mb: 2 }} />

                        {term.semanticTerms.length === 0 ? (
                            <Typography variant="body2" color="text.secondary" align="center" py={4}>
                                No semantic terms mapped yet.
                            </Typography>
                        ) : (
                            <Stack spacing={1}>
                                {term.semanticTerms.map((sem) => (
                                    <Paper
                                        key={sem.id}
                                        variant="outlined"
                                        sx={{ p: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                                    >
                                        <Typography variant="body2" fontWeight="medium">
                                            {sem.name}
                                        </Typography>
                                        <IconButton
                                            size="small"
                                            color="error"
                                            onClick={() => handleRemoveMapping(sem.id)}
                                        >
                                            <DeleteIcon fontSize="small" />
                                        </IconButton>
                                    </Paper>
                                ))}
                            </Stack>
                        )}
                    </Paper>
                </Grid>
            </Grid>

            <MappingModal 
                open={mappingModalOpen} 
                onClose={() => setMappingModalOpen(false)}
                onAdd={async (semIds) => {
                     if (id) {
                        await catalogApi.addMappings(id, { semanticTermIds: semIds });
                        await fetchTerm();
                     }
                }}
            />
        </Container>
    );
};
