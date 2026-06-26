import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Box, CircularProgress, Typography, Alert, Paper } from '@mui/material';
import { PageStudioApi } from '../../api/pageStudio';
import { EffectivePageDefinition, CorePageDefinition, PageOverlay } from '../../types/pageStudio';
import { mergePage } from '../../utils/pageMerge';
import RuntimeRenderer from './RuntimeRenderer';
import { AppThemeProvider } from '../../runtime/AppThemeProvider';
import { ThemeDefinition } from '../../types/pageStudio';

const RuntimePage: React.FC = () => {
    const { slug } = useParams<{ slug: string }>();
    const [page, setPage] = useState<EffectivePageDefinition | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const env = 'production'; // Mock
    const tenantId = 'default'; // Mock

    const coreTheme: ThemeDefinition = {
        id: 'gold-copy',
        name: 'Semlayer Gold',
        tokens: {
            colors: {
                primary: '#3b82f6',
                secondary: '#6366f1',
                background: '#f8fafc',
            },
            typography: {
                fontFamily: '"Inter", "Roboto", "Arial", sans-serif',
            },
            spacing: { unit: 8 },
            borderRadius: 12,
        }
    };

    useEffect(() => {
        if (slug) {
            loadPageFlow();
        }
    }, [slug]);

    const loadPageFlow = async () => {
        setLoading(true);
        setError(null);
        try {
            // 1. Fetch Core Page
            const core: CorePageDefinition = await PageStudioApi.getPageBySlug(slug!, env);
            
            // 2. Fetch Overlay (if exists)
            let overlay: PageOverlay | undefined;
            try {
                overlay = await PageStudioApi.getOverlay(core.id, tenantId, env);
            } catch (err) {
                console.log('No overlay found for this tenant/page');
            }

            // 3. Merge
            const effective = mergePage(core, overlay);
            setPage(effective);
        } catch (err: any) {
            console.error('Failed to load page', err);
            setError(err.response?.data || 'Failed to load page');
        } finally {
            setLoading(false);
        }
    };

    if (loading) return (
        <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
            <CircularProgress size={60} />
            <Typography variant="body2" sx={{ mt: 2, opacity: 0.6 }}>Loading semantic experience...</Typography>
        </Box>
    );

    if (error) return (
        <Box sx={{ p: 4 }}>
            <Alert severity="error">
                <Typography variant="subtitle2">Error Loading Page</Typography>
                <Typography variant="body2">{error}</Typography>
            </Alert>
        </Box>
    );

    if (!page) return null;

    return (
        <AppThemeProvider coreTheme={coreTheme}>
            <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column', bgcolor: 'background.default' }}>
                {/* Minimal Runtime Header */}
                <Paper elevation={0} sx={{ p: 1.5, borderBottom: '1px solid rgba(0,0,0,0.05)', display: 'flex', alignItems: 'center' }}>
                    <Typography variant="subtitle2" fontWeight="bold" sx={{ color: 'primary.main' }}>
                        {page.name}
                    </Typography>
                    <Box sx={{ flex: 1 }} />
                    <Typography variant="caption" color="textSecondary">
                        Tenant: {tenantId} | Env: {env}
                    </Typography>
                </Paper>

                <Box sx={{ flex: 1, overflow: 'hidden' }}>
                    <RuntimeRenderer page={page} tenantId={tenantId} />
                </Box>
            </Box>
        </AppThemeProvider>
    );
};

export default RuntimePage;
