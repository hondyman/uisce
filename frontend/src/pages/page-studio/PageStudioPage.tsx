import React, { useState, useEffect } from 'react';
import { Box, Button, Divider, IconButton, List, ListItemButton, ListItemIcon, ListItemText, Paper, Typography } from '@mui/material';
import { 
    Dashboard as PageIcon, 
    Add as AddIcon, 
    ArrowForwardIos as ChevronIcon,
} from '@mui/icons-material';
import { CorePageDefinition } from '../../types/pageStudio';
import { PageStudioApi } from '../../api/pageStudio';
import PageEditor from './PageEditor';
import { UnifiedBOPickerModal } from '../../components/common/UnifiedBOPickerModal';
import { extractAllBOFields } from '../../components/reporting/BOFieldsPalette';
import type { BusinessObjectSummary } from '../../features/data-explorer/types/dataExplorerTypes';

const PageStudioPage: React.FC = () => {
    const [pages, setPages] = useState<CorePageDefinition[]>([]);
    const [selectedPage, setSelectedPage] = useState<CorePageDefinition | null>(null);
    const [selectedBO, setSelectedBO] = useState<any>(null);
    const [selectedSubtypeKey, setSelectedSubtypeKey] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [boPickerOpen, setBoPickerOpen] = useState(false);

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

    const handleOpenBOPicker = () => {
        setBoPickerOpen(true);
    };

    const handleBOPicked = async (
        bo: BusinessObjectSummary,
        bindingId?: string,
        selectedRelatedBOs?: string[],
        bindingDetails?: any,
        subtypeKey?: string | null
    ) => {
        setBoPickerOpen(false);
        setSelectedSubtypeKey(subtypeKey || null);

        let initialBOObj: any = { ...bo, selectedSubtypeKey: subtypeKey || null };
        try {
            const res = await fetch(`/api/business-objects/${encodeURIComponent(bo.id)}/with_bindings`);
            if (res.ok) {
                const fullBO = await res.json();
                initialBOObj = {
                    ...bo,
                    ...fullBO,
                    selectedSubtypeKey: subtypeKey || null,
                };
            }
        } catch {}

        setSelectedBO(initialBOObj);

        const boKey = (bo.name || bo.id).toLowerCase();
        const initialFields = extractAllBOFields(initialBOObj, subtypeKey || 'all');

        const newPage: Partial<CorePageDefinition> = {
            name: `${bo.displayName || bo.name} Workspace`,
            slug: `${boKey}-workspace-${Date.now().toString(36).slice(-4)}`,
            env,
            layout: { 
                root: 'root', 
                nodes: { 
                    'root': { id: 'root', type: 'Row', children: [`form_${boKey}`] } 
                } 
            },
            components: {
                [`form_${boKey}`]: {
                    id: `form_${boKey}`,
                    type: 'BO_FORM',
                    props: {
                        boKey: boKey,
                        subtypeKey: subtypeKey || undefined,
                        bindingId: bindingId,
                        relatedBOs: selectedRelatedBOs || [],
                        fields: initialFields,
                        title: `${bo.displayName || bo.name} ${subtypeKey ? `(${subtypeKey})` : ''} Form`,
                    }
                }
            },
            dataBindings: { sources: {}, bindings: [] },
            visibility: { roles: ['advisor'] },
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
                    <IconButton onClick={handleOpenBOPicker} color="primary" size="small" title="Create New Page">
                        <AddIcon />
                    </IconButton>
                </Box>
                <Divider />
                <List sx={{ flex: 1, overflowY: 'auto' }}>
                    {pages.map((page) => (
                        <ListItemButton  
                            key={page.id} 
                            selected={selectedPage?.id === page.id}
                            onClick={() => {
                                setSelectedPage(page);
                                // If page has a BO component, attempt to load its BO definition
                                const firstBOComp = Object.values(page.components || {}).find((c) => c.props?.boKey);
                                if (firstBOComp?.props?.boKey) {
                                    fetch(`/api/business-objects/${encodeURIComponent(firstBOComp.props.boKey)}/with_bindings`)
                                        .then(r => r.ok ? r.json() : null)
                                        .then(fullBO => { if (fullBO) setSelectedBO(fullBO); })
                                        .catch(() => {});
                                }
                            }}
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
                        </ListItemButton>
                    ))}
                </List>
            </Paper>

            {/* Main Area */}
            <Box sx={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
                {selectedPage ? (
                    <PageEditor 
                        page={selectedPage} 
                        selectedBO={selectedBO}
                        selectedSubtypeKey={selectedSubtypeKey}
                        onSave={(updated: CorePageDefinition) => {
                            setSelectedPage(updated);
                            loadPages();
                        }}
                    />
                ) : (
                    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', opacity: 0.7 }}>
                        <PageIcon sx={{ fontSize: 64, mb: 2, color: '#94A3B8' }} />
                        <Typography variant="h6" fontWeight={600} color="#334155">Select a page or create a new one</Typography>
                        <Typography variant="body2" color="#64748B" sx={{ mb: 2 }}>Choose a Business Object to generate your layout with fields palette</Typography>
                        <Button startIcon={<AddIcon />} variant="contained" onClick={handleOpenBOPicker} sx={{ fontWeight: 700, textTransform: 'none' }}>
                            Create New Page
                        </Button>
                    </Box>
                )}
            </Box>

            <UnifiedBOPickerModal
                open={boPickerOpen}
                context="page"
                onClose={() => setBoPickerOpen(false)}
                onPick={handleBOPicked}
                onSelect={handleBOPicked}
            />
        </Box>
    );
};

export default PageStudioPage;
