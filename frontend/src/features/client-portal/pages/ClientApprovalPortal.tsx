import React, { useState } from 'react';
import { 
  Box, Typography, Paper, Button, Container, Stack, Divider, Alert 
} from '@mui/material';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import CancelOutlinedIcon from '@mui/icons-material/CancelOutlined';

export const ClientApprovalPortal: React.FC = () => {
    const [status, setStatus] = useState<'pending' | 'approved' | 'rejected'>('pending');

    const handleAction = (outcome: 'approved' | 'rejected') => {
        // In real app, calls API to signal Temporal S5 step
        setStatus(outcome);
    };

    if (status !== 'pending') {
        return (
            <Container maxWidth="sm" sx={{ mt: 8, textAlign: 'center' }}>
                <Paper sx={{ p: 6 }}>
                    {status === 'approved' ? (
                         <CheckCircleOutlineIcon color="success" sx={{ fontSize: 80, mb: 2 }} />
                    ) : (
                         <CancelOutlinedIcon color="error" sx={{ fontSize: 80, mb: 2 }} />
                    )}
                    <Typography variant="h4" gutterBottom>
                        {status === 'approved' ? 'Change Approved' : 'Change Declined'}
                    </Typography>
                    <Typography color="text.secondary">
                        Thank you. Your advisor has been notified and will proceed accordingly.
                    </Typography>
                </Paper>
            </Container>
        );
    }

    return (
        <Container maxWidth="md" sx={{ mt: 4 }}>
            {/* Header / Brand */}
            <Box mb={4}>
                <Typography variant="h5" fontWeight="bold" color="primary">WealthCo Portal</Typography>
            </Box>

            <Typography variant="h4" gutterBottom>Review Portfolio Update</Typography>
            <Alert severity="info" sx={{ mb: 3 }}>
                Action Required: Please review the proposed changes to your portfolio.
            </Alert>

            <Paper sx={{ p: 4, mb: 3 }}>
                <Typography variant="h6" gutterBottom>Summary of Changes</Typography>
                <Typography paragraph>
                    Dear Jane, based on our discussion, I recommend shifting 20% of your portfolio to the 
                    Growth strategy to capture recent market trends while maintaining your core preservation goals.
                </Typography>
                
                <Divider sx={{ my: 2 }} />
                
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={4} sx={{ mt: 2 }}>
                    <Box>
                        <Typography variant="caption" color="text.secondary">FROM MODEL</Typography>
                        <Typography variant="subtitle1" fontWeight="bold">Balanced 2025</Typography>
                    </Box>
                    <Box>
                        <Typography variant="caption" color="text.secondary">TO MODEL</Typography>
                        <Typography variant="subtitle1" fontWeight="bold">Growth 2025 (Aggressive)</Typography>
                    </Box>
                    <Box>
                        <Typography variant="caption" color="text.secondary">EST. RISK CHANGE</Typography>
                        <Typography variant="subtitle1" fontWeight="bold" color="error">+0.8%</Typography>
                    </Box>
                </Stack>
            </Paper>

            <Stack direction="row" spacing={2} justifyContent="flex-end">
                <Button 
                    variant="outlined" 
                    color="error" 
                    size="large"
                    startIcon={<CancelOutlinedIcon />}
                    onClick={() => handleAction('rejected')}
                >
                    Decline Change
                </Button>
                <Button 
                    variant="contained" 
                    color="success" 
                    size="large"
                    startIcon={<CheckCircleOutlineIcon />}
                    onClick={() => handleAction('approved')}
                >
                    Approve Change
                </Button>
            </Stack>
        </Container>
    );
};
