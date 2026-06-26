import React, { useEffect, useState } from 'react';
import {
    Box,
    Typography,
    Button,
    Container,
    Paper,
    CircularProgress,
    Stack,
    Alert,
} from '@mui/material';
import { AutoAwesome as AutoIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import { catalogApi, AIBusinessTermDraft } from '../../api/catalogApi';
import { SuggestionsTable } from './components/SuggestionsTable';
import { BusinessTermReviewDrawer } from './components/BusinessTermReviewDrawer';

export const AIBusinessTermSuggestionsPage: React.FC = () => {
    const [loading, setLoading] = useState(false);
    const [suggestions, setSuggestions] = useState<AIBusinessTermDraft[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [selectedSuggestion, setSelectedSuggestion] = useState<AIBusinessTermDraft | null>(null);
    const [drawerOpen, setDrawerOpen] = useState(false);

    const fetchSuggestions = async () => {
        setLoading(true);
        try {
            const data = await catalogApi.listSuggestions();
            setSuggestions(data);
            setError(null);
        } catch (err) {
            setError('Failed to load suggestions. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchSuggestions();
    }, []);

    const handleGenerate = async () => {
        setLoading(true);
        try {
            // In a real app, this might open a dialog to select tables
            // For MVP, we'll trigger generation for a default set or all
            await catalogApi.generateSuggestion({ tableNames: ['all'] }); // Simplified
            await fetchSuggestions();
        } catch (err) {
            setError('Failed to generate suggestions.');
            setLoading(false);
        }
    };

    const handleView = (suggestion: AIBusinessTermDraft) => {
        setSelectedSuggestion(suggestion);
        setDrawerOpen(true);
    };

    const handleApprove = async (id: string) => {
        try {
            await catalogApi.approveSuggestion(id);
            setDrawerOpen(false);
            fetchSuggestions(); // Refresh list to show status change
        } catch (err) {
            setError('Failed to approve suggestion.');
        }
    };

    const handleReject = async (id: string, reason: string) => {
        try {
            await catalogApi.rejectSuggestion(id, reason);
            setDrawerOpen(false);
            fetchSuggestions();
        } catch (err) {
            setError('Failed to reject suggestion.');
        }
    };

    return (
        <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" mb={4}>
                <Box>
                    <Typography variant="h4" component="h1" gutterBottom fontWeight="bold">
                        AI Business Term Suggestions
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        Review and approve business terms generated from your technical metadata.
                    </Typography>
                </Box>
                <Stack direction="row" spacing={2}>
                    <Button
                        variant="outlined"
                        startIcon={<RefreshIcon />}
                        onClick={fetchSuggestions}
                        disabled={loading}
                    >
                        Refresh
                    </Button>
                    <Button
                        variant="contained"
                        startIcon={<AutoIcon />}
                        onClick={handleGenerate}
                        disabled={loading}
                        sx={{
                            background: 'linear-gradient(45deg, #2196F3 30%, #21CBF3 90%)',
                            color: 'white',
                        }}
                    >
                        Generate New
                    </Button>
                </Stack>
            </Stack>

            {error && (
                <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
                    {error}
                </Alert>
            )}

            {loading && suggestions.length === 0 ? (
                <Box display="flex" justifyContent="center" p={8}>
                    <CircularProgress />
                </Box>
            ) : (
                <SuggestionsTable
                    suggestions={suggestions}
                    onView={handleView}
                    onApprove={handleApprove}
                    onReject={(id) => {
                        const suggestion = suggestions.find((s) => s.id === id);
                        if (suggestion) {
                            handleView(suggestion);
                        }
                    }}
                />
            )}

            <BusinessTermReviewDrawer
                open={drawerOpen}
                suggestion={selectedSuggestion}
                onClose={() => setDrawerOpen(false)}
                onApprove={handleApprove}
                onReject={handleReject}
            />
        </Container>
    );
};
