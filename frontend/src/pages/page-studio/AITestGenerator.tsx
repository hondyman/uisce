import React from 'react';
import {
    Box,
    Typography,
    Paper,
    Stack,
    Switch,
    FormControlLabel,
    Button,
    Chip,
    Divider,
    Alert
} from '@mui/material';
import {
    AutoAwesome as AiIcon,
    BugReport as TestIcon,
    Speed as PerformanceIcon,
    Lock as PrivacyIcon,
    Link as BindingIcon
} from '@mui/icons-material';
import { CorePageDefinition } from '../../types/pageStudio';

interface AITestGeneratorProps {
    page: CorePageDefinition;
}

const SUGGESTED_TESTS = [
    {
        id: 'load-test',
        type: 'performance',
        title: 'SLO Load Test',
        description: 'Verify page loads in < 1s with 50 concurrent users.',
        category: 'Stability',
        icon: <PerformanceIcon fontSize="small" color="primary" />
    },
    {
        id: 'pii-test',
        type: 'privacy',
        title: 'PII Leakage Scan',
        description: 'Ensure no unmasked tax identifiers or personal emails are rendered.',
        category: 'Compliance',
        icon: <PrivacyIcon fontSize="small" color="error" />
    },
    {
        id: 'binding-test',
        type: 'binding',
        title: 'Data Binding Integrity',
        description: 'Verify all table columns map to active BO fields.',
        category: 'Functional',
        icon: <BindingIcon fontSize="small" color="success" />
    }
];

export const AITestGenerator: React.FC<AITestGeneratorProps> = ({ page }) => {
    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
                <AiIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6" fontWeight="bold">AI Test Generator</Typography>
            </Box>

            <Alert severity="info" sx={{ mb: 3 }}>
                AI has analyzed your layout and data bindings. The following tests are recommended for the CRS.
            </Alert>

            <Stack spacing={3}>
                {SUGGESTED_TESTS.map((test) => (
                    <Paper 
                        key={test.id} 
                        variant="outlined" 
                        sx={{ p: 2, borderRadius: 2, '&:hover': { bgcolor: 'rgba(0,0,0,0.01)' } }}
                    >
                        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 1 }}>
                            <Box sx={{ display: 'flex', alignItems: 'center' }}>
                                <Box sx={{ mr: 1.5, display: 'flex' }}>{test.icon}</Box>
                                <Box>
                                    <Typography variant="subtitle2" fontWeight="bold">{test.title}</Typography>
                                    <Chip label={test.category} size="small" variant="outlined" sx={{ height: 20, fontSize: '10px' }} />
                                </Box>
                            </Box>
                            <FormControlLabel 
                                control={<Switch defaultChecked size="small" />} 
                                label={<Typography variant="caption">Enabled</Typography>}
                                labelPlacement="start"
                            />
                        </Box>
                        <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>{test.description}</Typography>
                        <Box sx={{ display: 'flex', gap: 1 }}>
                            <Button variant="outlined" size="small" startIcon={<TestIcon />} sx={{ textTransform: 'none' }}>
                                Run Now
                            </Button>
                            <Button variant="outlined" size="small" sx={{ textTransform: 'none' }}>
                                View Code
                            </Button>
                        </Box>
                    </Paper>
                ))}
            </Stack>

            <Divider sx={{ my: 4 }} />

            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="caption" color="textSecondary">
                    Tests will be executed automatically in the review workflow.
                </Typography>
                <Button variant="contained" startIcon={<TestIcon />} color="primary">
                    Sync to ChangeSet
                </Button>
            </Box>
        </Box>
    );
};
