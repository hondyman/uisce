import React, { useState } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    TextField,
    MenuItem,
    Box,
    Typography,
    CircularProgress,
    Stack,
    Card,
    CardActionArea,
    CardContent,
    IconButton
} from '@mui/material';
import {
    AutoAwesome as AiIcon,
    Dashboard as DashboardIcon,
    List as ListIcon,
    Description as DetailIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { PageStudioApi } from '../../api/pageStudio';

interface AIGeneratorWizardProps {
    open: boolean;
    onClose: () => void;
    onGenerated: (data: any) => void;
    tenantId: string;
}

const INTENTS = [
    { id: 'dashboard', label: 'Dashboard', icon: <DashboardIcon color="primary" />, description: 'Overview with KPIs and charts' },
    { id: 'list', label: 'List View', icon: <ListIcon color="secondary" />, description: 'Searchable data table' },
    { id: 'detail', label: 'Detail Page', icon: <DetailIcon color="info" />, description: 'In-depth view of a single entity' },
];

export const AIGeneratorWizard: React.FC<AIGeneratorWizardProps> = ({ open, onClose, onGenerated, tenantId }) => {
    const [boName, setBoName] = useState('');
    const [intent, setIntent] = useState('dashboard');
    const [loading, setLoading] = useState(false);

    const handleGenerate = async () => {
        if (!boName) return;
        setLoading(true);
        try {
            const result = await PageStudioApi.generateLayout(boName, intent, tenantId);
            onGenerated(result);
            onClose();
        } catch (err) {
            console.error('AI Generation failed', err);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <AiIcon color="primary" />
                    AI Page Generator
                </Box>
                <IconButton onClick={onClose} size="small">
                    <CloseIcon />
                </IconButton>
            </DialogTitle>
            <DialogContent dividers>
                <Typography variant="body2" color="textSecondary" sx={{ mb: 3 }}>
                    Describe what you want to build and let AI suggest the optimal layout, components, and data bindings.
                </Typography>

                <Stack spacing={3}>
                    <TextField
                        fullWidth
                        label="Source Business Object"
                        placeholder="e.g. Positions, Trades, Accounts"
                        value={boName}
                        onChange={(e) => setBoName(e.target.value)}
                        variant="outlined"
                        helperText="AI will analyze the semantic metadata for this object."
                    />

                    <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 'bold' }}>Select Page Intent</Typography>
                        <Grid container spacing={2}>
                            {INTENTS.map((i) => (
                                <Grid item xs={4} key={i.id}>
                                    <Card 
                                        variant="outlined" 
                                        sx={{ 
                                            borderColor: intent === i.id ? 'primary.main' : 'divider',
                                            bgcolor: intent === i.id ? 'primary.light' : 'inherit',
                                            opacity: intent === i.id ? 1 : 0.8,
                                            '&:hover': { opacity: 1 }
                                        }}
                                    >
                                        <CardActionArea onClick={() => setIntent(i.id)} sx={{ p: 1, textAlign: 'center' }}>
                                            <Box sx={{ mb: 1 }}>{i.icon}</Box>
                                            <Typography variant="caption" fontWeight="bold" display="block">{i.label}</Typography>
                                        </CardActionArea>
                                    </Card>
                                </Grid>
                            ))}
                        </Grid>
                    </Box>
                </Stack>
            </DialogContent>
            <DialogActions sx={{ p: 2 }}>
                <Button onClick={onClose} color="inherit">Cancel</Button>
                <Button
                    variant="contained"
                    onClick={handleGenerate}
                    disabled={!boName || loading}
                    startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <AiIcon />}
                >
                    {loading ? 'Generating...' : 'Generate Page'}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

// Helper Grid internal mock since MUI Grid might need import
const Grid = ({ children, container, spacing, item, xs }: any) => (
    <Box sx={{ 
        display: container ? 'flex' : 'block', 
        flexWrap: container ? 'wrap' : 'nowrap',
        m: container ? -(spacing || 0) * 4 : 0,
        width: item ? `${(xs / 12) * 100}%` : 'auto',
        p: item ? (spacing || 0) * 4 : 0
    }}>
        {children}
    </Box>
);
