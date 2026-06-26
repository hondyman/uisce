import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, List, ListItem, ListItemText, ListItemIcon, Button, Divider, Chip, IconButton } from '@mui/material';
import { 
    Dashboard as PageIcon, 
    Add as AddIcon, 
    Layers as OverlayIcon, 
    Speed as PerformanceIcon, 
    History as HistoryIcon,
    ArrowForwardIos as ChevronIcon,
    Save as SaveIcon, 
    PlayArrow as PreviewIcon, 
    AccountTree as LineageIcon, 
    Rule as ReviewIcon,
    Palette as PaletteIcon,
    Settings as SettingsIcon,
    Delete as DeleteIcon,
    LinkOff as UnbindIcon,
    Api as ApiIcon,
    Storage as GqlIcon,
    AutoAwesome as AiIcon
} from '@mui/icons-material';
import { CorePageDefinition, ComponentDefinition, DataSourceDefinition } from '../../types/pageStudio';
import { PageStudioApi } from '../../api/pageStudio';
import PageEditor from './PageEditor';
import { AIGeneratorWizard } from './AIGeneratorWizard';
import { TenantUpgradeAssistant } from './TenantUpgradeAssistant';
import { TenantBrandingEditor } from './TenantBrandingEditor';
import { Upgrade as UpgradeIcon } from '@mui/icons-material';

const PageStudioPage: React.FC = () => {
    const [pages, setPages] = useState<CorePageDefinition[]>([]);
    const [selectedPage, setSelectedPage] = useState<CorePageDefinition | null>(null);
    const [loading, setLoading] = useState(true);
    const [wizardOpen, setWizardOpen] = useState(false);
    const [view, setView] = useState<'editor' | 'upgrades' | 'branding'>('editor');

    const env = 'production'; // Mock

    useEffect(() => {
        loadPages();
    }, []);

    const loadPages = async () => {
        try {
            const data = await PageStudioApi.listPages(env);
            setPages(data);
        } catch (err) {
            console.error('Failed to load pages', err);
        } finally {
            setLoading(false);
        }
    };

    const handleCreatePage = () => {
        const newPage: Partial<CorePageDefinition> = {
            name: 'New Page',
            slug: 'new-page',
            env,
            layout: { root: 'root', nodes: { 'root': { id: 'root', type: 'Row', children: [] } } },
            components: {},
            dataBindings: { sources: {}, bindings: [] },
            visibility: { roles: ['advisor'] },
            version: 1,
        };
        setSelectedPage(newPage as CorePageDefinition);
    };

    const handleAIGenerated = (result: any) => {
        const newPage: Partial<CorePageDefinition> = {
            name: 'Generated Page',
            slug: 'generated-' + Math.random().toString(36).substring(7),
            env,
            layout: result.layout,
            components: result.components,
            dataBindings: result.dataBindings,
            version: 1,
        };
        setSelectedPage(newPage as CorePageDefinition);
    };

    return (
        <Box sx={{ display: 'flex', height: '100vh', bgcolor: '#f8fafc' }}>
            {/* Sidebar */}
            <Paper elevation={0} sx={{ width: 300, borderRight: '1px solid rgba(0,0,0,0.05)', display: 'flex', flexDirection: 'column' }}>
                <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="h6" fontWeight="bold">Page Studio</Typography>
                    <IconButton onClick={handleCreatePage} color="primary" size="small">
                        <AddIcon />
                    </IconButton>
                </Box>
                <Divider />
                <List sx={{ flex: 1, overflowY: 'auto' }}>
                    {pages.map((page) => (
                        <ListItem 
                            key={page.id} 
                            button 
                            selected={selectedPage?.id === page.id}
                            onClick={() => setSelectedPage(page)}
                            sx={{ 
                                mb: 0.5, 
                                mx: 1, 
                                borderRadius: 2,
                                '&.Mui-selected': { bgcolor: 'primary.light', color: 'primary.contrastText' }
                            }}
                        >
                            <ListItemIcon sx={{ minWidth: 40, color: 'inherit' }}>
                                <PageIcon />
                            </ListItemIcon>
                            <ListItemText 
                                primary={page.name} 
                                secondary={page.slug} 
                                primaryTypographyProps={{ variant: 'body2', fontWeight: 600 }}
                                secondaryTypographyProps={{ variant: 'caption', color: 'inherit', sx: { opacity: 0.7 } }}
                            />
                            <ChevronIcon sx={{ fontSize: 14, opacity: 0.5 }} />
                        </ListItem>
                    ))}
                </List>
                <Box sx={{ p: 2 }}>
                    <Button 
                        fullWidth 
                        variant="contained" 
                        color="primary"
                        startIcon={<AiIcon />} 
                        size="small" 
                        sx={{ mb: 1, background: 'linear-gradient(45deg, #3b82f6 30%, #6366f1 90%)' }}
                        onClick={() => setWizardOpen(true)}
                    >
                        Generate with AI
                    </Button>
                    <Button 
                        fullWidth 
                        variant="outlined" 
                        startIcon={<UpgradeIcon />} 
                        size="small"
                        onClick={() => setView('upgrades')}
                        sx={{ mb: 1, textTransform: 'none', borderStyle: 'dashed' }}
                    >
                        Upgrade Assistant
                    </Button>
                    <Button 
                        fullWidth 
                        variant="outlined" 
                        startIcon={<PaletteIcon />} 
                        size="small"
                        onClick={() => setView('branding')}
                        sx={{ mb: 1 }}
                    >
                        Tenant Branding
                    </Button>
                    <Button 
                        fullWidth 
                        variant="outlined" 
                        startIcon={<OverlayIcon />} 
                        size="small"
                        onClick={() => setView('editor')}
                    >
                        Page Editor
                    </Button>
                </Box>
            </Paper>

            {/* Main Area */}
            <Box sx={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
                {view === 'upgrades' ? (
                    <TenantUpgradeAssistant />
                ) : view === 'branding' ? (
                    <TenantBrandingEditor />
                ) : selectedPage ? (
                    <PageEditor 
                        page={selectedPage} 
                        onSave={(updated: CorePageDefinition) => {
                            setSelectedPage(updated);
                            loadPages();
                        }}
                    />
                ) : (
                    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', opacity: 0.5 }}>
                        <PageIcon sx={{ fontSize: 64, mb: 2 }} />
                        <Typography variant="h6">Select a page to start building</Typography>
                        <Button startIcon={<AddIcon />} variant="contained" sx={{ mt: 2 }} onClick={handleCreatePage}>
                            Create New Page
                        </Button>
                    </Box>
                )}
            </Box>

            <AIGeneratorWizard 
                open={wizardOpen} 
                onClose={() => setWizardOpen(false)} 
                onGenerated={handleAIGenerated}
                tenantId="default"
            />
        </Box>
    );
};

export default PageStudioPage;
