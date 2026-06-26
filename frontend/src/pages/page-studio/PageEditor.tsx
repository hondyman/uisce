import React, { useState } from 'react';
import { Box, Typography, Paper, Tabs, Tab, Button, Grid, IconButton, Tooltip, Divider } from '@mui/material';
import { 
    Save as SaveIcon, 
    PlayArrow as PreviewIcon, 
    AccountTree as LineageIcon, 
    Rule as ReviewIcon,
    Palette as PaletteIcon,
    Settings as SettingsIcon,
    Delete as DeleteIcon,
    Dashboard as DesignIcon,
    Storage as DataIcon,
    Speed as PerformanceIcon
} from '@mui/icons-material';
import { CorePageDefinition } from '../../types/pageStudio';
import { PageStudioApi } from '../../api/pageStudio';
import ComponentPalette from './ComponentPalette';
import LayoutCanvas from './LayoutCanvas';
import PropertiesPanel from './PropertiesPanel';
import { PagePerformanceDashboard } from './PagePerformanceDashboard';
import { AIDocumentationViewer } from './AIDocumentationViewer';
import { AITestGenerator } from './AITestGenerator';
import { Description as DocIcon, BugReport as TestIcon } from '@mui/icons-material';

interface PageEditorProps {
    page: CorePageDefinition;
    onSave: (page: CorePageDefinition) => void;
}

const PageEditor: React.FC<PageEditorProps> = ({ page, onSave }) => {
    const [draft, setDraft] = useState<CorePageDefinition>(page);
    const [tab, setTab] = useState(0); // 0: Design, 1: Data, 2: Performance, 3: Documentation, 4: Testing, 5: Review
    const [selectedId, setSelectedId] = useState<string | null>(null);

    const handleSave = async () => {
        try {
            const saved = await PageStudioApi.savePage(draft);
            onSave(saved);
        } catch (err) {
            console.error('Save failed', err);
        }
    };

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            {/* Header / Actions */}
            <Paper elevation={0} sx={{ p: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid rgba(0,0,0,0.05)' }}>
                <Box>
                    <Typography variant="subtitle1" fontWeight="bold">{draft.name}</Typography>
                    <Typography variant="caption" color="textSecondary">{draft.slug} • v{draft.version}</Typography>
                </Box>
                <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button variant="outlined" startIcon={<PreviewIcon />} size="small">Preview</Button>
                    <Button variant="outlined" startIcon={<LineageIcon />} size="small">Lineage</Button>
                    <Button variant="outlined" startIcon={<ReviewIcon />} size="small" color="secondary">Review</Button>
                    <Divider orientation="vertical" flexItem sx={{ mx: 1 }} />
                    <Button variant="contained" startIcon={<SaveIcon />} size="small" onClick={handleSave}>Save Changes</Button>
                </Box>
            </Paper>

            <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
                {/* Left Panel: Palette */}
                <Paper elevation={0} sx={{ width: 250, borderRight: '1px solid rgba(0,0,0,0.05)', display: 'flex', flexDirection: 'column' }}>
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 48 }}>
                            <Tab label="Design" icon={<DesignIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Data Binding" icon={<DataIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Performance" icon={<PerformanceIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Docs" icon={<DocIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Testing" icon={<TestIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Review" icon={<ReviewIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                        </Tabs>
                    </Box>
                    <Box sx={{ flex: 1, overflowY: 'auto' }}>
                        {tab === 0 && (
                            <ComponentPalette />
                        )}
                        {tab === 1 && (
                            <Box sx={{ p: 2 }}>
                                <Typography variant="caption" color="textSecondary">Data Bindings coming soon...</Typography>
                            </Box>
                        )}
                        {tab === 2 && (
                            <PagePerformanceDashboard pageId={draft.id!} />
                        )}
                        {tab === 3 && (
                            <AIDocumentationViewer page={draft} />
                        )}
                        {tab === 4 && (
                            <AITestGenerator page={draft} />
                        )}
                        {tab === 5 && (
                            <Box sx={{ p: 2 }}>
                                <Typography variant="caption" color="textSecondary">Review coming soon...</Typography>
                            </Box>
                        )}
                    </Box>
                </Paper>

                {/* Main: Layout Canvas */}
                <Box sx={{ flex: 1, p: 3, bgcolor: '#f1f5f9', overflowY: 'auto' }}>
                    <LayoutCanvas 
                        draft={draft} 
                        setDraft={setDraft} 
                        selectedId={selectedId} 
                        onSelect={setSelectedId} 
                    />
                </Box>

                {/* Right Panel: Properties */}
                <Paper elevation={0} sx={{ width: 300, borderLeft: '1px solid rgba(0,0,0,0.05)', overflowY: 'auto' }}>
                    <PropertiesPanel 
                        selectedId={selectedId} 
                        draft={draft} 
                        setDraft={setDraft}
                        tenantId={draft.tenantId || 'default'}
                    />
                </Paper>
            </Box>
        </Box>
    );
};

export default PageEditor;
