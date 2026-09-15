import React, { useState } from 'react';
import { Box, Typography, Paper, Tabs, Tab, Button, Grid, IconButton, Tooltip, Divider, TextField, Snackbar, Alert } from '@mui/material';
import { DndContext, PointerSensor, useSensor, useSensors } from '@dnd-kit/core';
import {
    Save as SaveIcon,
    PlayArrow as PreviewIcon,
    Palette as PaletteIcon,
    Settings as SettingsIcon,
    Delete as DeleteIcon,
    Dashboard as DesignIcon,
    Storage as DataIcon,
    Speed as PerformanceIcon,
    Add as AddTabIcon,
    Close as CloseTabIcon,
    ChevronLeft as ChevronLeftIcon,
    ChevronRight as ChevronRightIcon,
} from '@mui/icons-material';
import { CorePageDefinition, PageLayout, PageTab } from '../../types/pageStudio';
import { PageStudioApi } from '../../api/pageStudio';
import ComponentPalette from './ComponentPalette';
import LayoutCanvas from './LayoutCanvas';
import PropertiesPanel from './PropertiesPanel';
import DataBindingsPanel from './DataBindingsPanel';
import { PagePerformanceDashboard } from './PagePerformanceDashboard';
import { AIDocumentationViewer } from './AIDocumentationViewer';
import { AITestGenerator } from './AITestGenerator';
import { Description as DocIcon, BugReport as TestIcon, Edit as EditIcon } from '@mui/icons-material';
import DraftPreview from './DraftPreview';
import TemplatePickerDialog from './TemplatePickerDialog';
import { SelectionProvider } from './SelectionContext';

interface PageEditorProps {
    page: CorePageDefinition;
    onSave: (page: CorePageDefinition) => void;
}

const DEFAULT_TAB_ID = '__default__';

const PageEditor: React.FC<PageEditorProps> = ({ page, onSave }) => {
    const [draft, setDraft] = useState<CorePageDefinition>(page);
    const [tab, setTab] = useState(0); // 0: Design, 1: Data, 2: Performance, 3: Documentation, 4: Testing
    const [selectedId, setSelectedId] = useState<string | null>(null);
    // Design/Preview: the Preview button previously had no onClick handler
    // at all - clicking it did nothing. Preview renders the live in-memory
    // draft (DraftPreview) the same way a consumer would see it, without
    // requiring a save first.
    const [viewMode, setViewMode] = useState<'design' | 'preview'>('design');
    const [templatePickerOpen, setTemplatePickerOpen] = useState(false);
    const [saveNotice, setSaveNotice] = useState<{ severity: 'success' | 'error'; message: string } | null>(null);
    const [propertiesPanelOpen, setPropertiesPanelOpen] = useState(true);
    // A single DndContext for the whole design surface (ComponentPalette,
    // DataBindingsPanel, and both LayoutCanvas instances below) - each
    // LayoutCanvas registers its own onDragEnd via useDndMonitor and only
    // acts when the drop actually belongs to its own tree, so one shared
    // context is enough even with two canvases mounted at once.
    const dndSensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }));

    // Multi-tab pages: draft.tabs is the source of truth once set. A page
    // saved before tabs existed (or never given a second tab) has no
    // `tabs` field at all - synthesize a single implicit tab from
    // draft.layout so the rest of this component never has to special-
    // case "no tabs" vs. "one tab".
    const pageTabs: PageTab[] = draft.tabs && draft.tabs.length > 0
        ? draft.tabs
        : [{ id: DEFAULT_TAB_ID, label: draft.name || 'Page 1', layout: draft.layout }];
    const [activeTabId, setActiveTabId] = useState(pageTabs[0].id);
    const activeTab = pageTabs.find((t) => t.id === activeTabId) || pageTabs[0];

    // Page-wide filter bar, rendered above the tab strip so a Slicer placed
    // here scopes every tab instead of just one - synthesized empty the
    // same way pageTabs falls back to draft.layout when no tabs exist yet.
    const filterBarLayout: PageLayout = draft.filterBar || { root: 'filter_root', nodes: { filter_root: { id: 'filter_root', type: 'Row', children: [] } } };
    const handleFilterBarLayoutChange = (updater: (layout: PageLayout) => PageLayout) => {
        setDraft((prev) => ({
            ...prev,
            filterBar: updater(prev.filterBar || { root: 'filter_root', nodes: { filter_root: { id: 'filter_root', type: 'Row', children: [] } } }),
        }));
    };

    const handleLayoutChange = (updater: (layout: PageLayout) => PageLayout) => {
        setDraft((prev) => {
            const tabs = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: prev.layout }];
            const nextTabs = tabs.map((t) => (t.id === activeTab.id ? { ...t, layout: updater(t.layout) } : t));
            // Keep draft.layout mirroring the first tab for pages that
            // never explicitly opt into multi-tab (backward compat: an
            // older save path or export that only reads draft.layout
            // still sees the right content).
            return { ...prev, tabs: prev.tabs ? nextTabs : undefined, layout: prev.tabs ? prev.layout : nextTabs[0].layout };
        });
    };

    const handleComponentsChange = (updater: (components: CorePageDefinition['components']) => CorePageDefinition['components']) => {
        setDraft((prev) => ({ ...prev, components: updater(prev.components) }));
    };

    const handleAddTab = () => setTemplatePickerOpen(true);

    const handlePickTabTemplate = ({ layout }: { layout: PageLayout }) => {
        setDraft((prev) => {
            const existing = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: prev.layout }];
            const tabs = [...existing, { id: `tab_${Math.random().toString(36).slice(2, 8)}`, label: `Tab ${existing.length + 1}`, layout }];
            setActiveTabId(tabs[tabs.length - 1].id);
            return { ...prev, tabs };
        });
    };

    const handleRenameTab = (tabId: string, label: string) => {
        setDraft((prev) => {
            const existing = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: prev.layout }];
            return { ...prev, tabs: existing.map((t) => (t.id === tabId ? { ...t, label } : t)) };
        });
    };

    const handleDeleteTab = (tabId: string) => {
        setDraft((prev) => {
            const existing = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: prev.layout }];
            if (existing.length <= 1) return prev;
            const tabs = existing.filter((t) => t.id !== tabId);
            if (activeTabId === tabId) setActiveTabId(tabs[0].id);
            return { ...prev, tabs };
        });
    };

    const handleSave = async () => {
        try {
            // A page opened from the sidebar list carries its real id;
            // a fresh "Create New Page" draft has none. Without this
            // branch every save POSTed as a create, so re-saving an
            // existing page hit the (tenant_id, slug) unique constraint
            // as a 409 instead of updating it.
            const saved = draft.id
                ? await PageStudioApi.updatePage(draft.id, draft)
                : await PageStudioApi.savePage(draft);
            onSave(saved);
            setSaveNotice({ severity: 'success', message: 'Saved' });
        } catch (err) {
            console.error('Save failed', err);
            setSaveNotice({ severity: 'error', message: err instanceof Error ? err.message : 'Save failed' });
        }
    };

    return (
        // Keyed by page id so switching to a different page (not just a tab
        // within this one) starts with no record selected, instead of
        // inheriting the previous page's selection.
        <SelectionProvider key={draft.id || 'new'}>
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            {/* Header / Actions */}
            <Paper elevation={0} sx={{ p: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid rgba(0,0,0,0.05)' }}>
                <Box>
                    <Typography variant="subtitle1" fontWeight="bold">{draft.name}</Typography>
                    <Typography variant="caption" color="textSecondary">{draft.slug} • v{draft.version}</Typography>
                </Box>
                <Box sx={{ display: 'flex', gap: 1 }}>
                    {viewMode === 'design' ? (
                        <Button variant="outlined" startIcon={<PreviewIcon />} size="small" onClick={() => setViewMode('preview')}>Preview</Button>
                    ) : (
                        <Button variant="contained" color="secondary" startIcon={<EditIcon />} size="small" onClick={() => setViewMode('design')}>Back to Design</Button>
                    )}
                    <Divider orientation="vertical" flexItem sx={{ mx: 1 }} />
                    <Button variant="contained" startIcon={<SaveIcon />} size="small" onClick={handleSave}>Save Changes</Button>
                </Box>
            </Paper>

            {viewMode === 'preview' ? (
                <DraftPreview draft={draft} tenantId={draft.tenantId || 'default'} />
            ) : (
            <DndContext sensors={dndSensors}>
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
                        </Tabs>
                    </Box>
                    <Box sx={{ flex: 1, overflowY: 'auto' }}>
                        {tab === 0 && (
                            <ComponentPalette />
                        )}
                        {tab === 1 && (
                            <DataBindingsPanel
                                draft={draft}
                                setDraft={setDraft}
                                tenantId={draft.tenantId || 'default'}
                            />
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
                    </Box>
                </Paper>

                {/* Main: Page tabs + Layout Canvas */}
                <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                    <Box sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'action.hover', px: 2, py: 1 }}>
                        <Typography variant="caption" color="text.secondary" fontWeight="bold" sx={{ display: 'block', mb: 0.5 }}>
                            PAGE FILTERS <Typography component="span" variant="caption" color="text.secondary" fontWeight="normal">(shown above every tab, applies to all of them)</Typography>
                        </Typography>
                        <Box sx={{ maxHeight: 220, overflowY: 'auto', '& > div': { pb: 0 } }}>
                            <LayoutCanvas
                                layout={filterBarLayout}
                                onLayoutChange={handleFilterBarLayoutChange}
                                components={draft.components}
                                onComponentsChange={handleComponentsChange}
                                dataSources={draft.dataSources}
                                tenantId={draft.tenantId || 'default'}
                                selectedId={selectedId}
                                onSelect={setSelectedId}
                            />
                        </Box>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper', px: 1 }}>
                        <Tabs
                            value={activeTabId}
                            onChange={(_, v) => setActiveTabId(v)}
                            variant="scrollable"
                            scrollButtons="auto"
                            sx={{ flex: 1, minHeight: 40 }}
                        >
                            {pageTabs.map((t) => (
                                <Tab
                                    key={t.id}
                                    value={t.id}
                                    sx={{ minHeight: 40, textTransform: 'none' }}
                                    label={
                                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                            <TextField
                                                variant="standard"
                                                value={t.label}
                                                onClick={(e) => e.stopPropagation()}
                                                onChange={(e) => handleRenameTab(t.id, e.target.value)}
                                                InputProps={{ disableUnderline: true }}
                                                sx={{ width: Math.max(50, t.label.length * 8), '& input': { p: 0, fontSize: 13, cursor: 'text' } }}
                                            />
                                            {pageTabs.length > 1 && (
                                                <CloseTabIcon
                                                    sx={{ fontSize: 14, opacity: 0.5, '&:hover': { opacity: 1 } }}
                                                    onClick={(e) => { e.stopPropagation(); handleDeleteTab(t.id); }}
                                                />
                                            )}
                                        </Box>
                                    }
                                />
                            ))}
                        </Tabs>
                        <Tooltip title="Add tab">
                            <IconButton size="small" onClick={handleAddTab}><AddTabIcon fontSize="small" /></IconButton>
                        </Tooltip>
                    </Box>
                    <Box sx={{ flex: 1, p: 3, bgcolor: '#f1f5f9', overflowY: 'auto' }}>
                        <LayoutCanvas
                            layout={activeTab.layout}
                            onLayoutChange={handleLayoutChange}
                            components={draft.components}
                            onComponentsChange={handleComponentsChange}
                            dataSources={draft.dataSources}
                            tenantId={draft.tenantId || 'default'}
                            selectedId={selectedId}
                            onSelect={setSelectedId}
                        />
                    </Box>
                </Box>

                {/* Right Panel: Properties - collapsible so it doesn't permanently
                    eat canvas width once a widget is selected; slides in/out
                    rather than just disappearing so the collapse/expand is legible. */}
                <Box sx={{ display: 'flex', flexShrink: 0 }}>
                    <Box
                        sx={{
                            display: 'flex', alignItems: 'flex-start', pt: 1.5,
                            borderLeft: '1px solid rgba(0,0,0,0.05)', bgcolor: 'background.paper',
                        }}
                    >
                        <Tooltip title={propertiesPanelOpen ? 'Collapse properties panel' : 'Expand properties panel'}>
                            <IconButton size="small" onClick={() => setPropertiesPanelOpen((v) => !v)}>
                                {propertiesPanelOpen ? <ChevronRightIcon fontSize="small" /> : <ChevronLeftIcon fontSize="small" />}
                            </IconButton>
                        </Tooltip>
                    </Box>
                    <Paper
                        elevation={0}
                        sx={{
                            width: propertiesPanelOpen ? 320 : 0,
                            overflow: 'hidden',
                            transition: 'width 0.2s ease',
                        }}
                    >
                        <Box sx={{ width: 320, height: '100%' }}>
                            <PropertiesPanel
                                selectedId={selectedId}
                                onSelectComponent={setSelectedId}
                                draft={draft}
                                setDraft={setDraft}
                                layout={activeTab.layout}
                                onLayoutChange={handleLayoutChange}
                                tenantId={draft.tenantId || 'default'}
                            />
                        </Box>
                    </Paper>
                </Box>
            </Box>
            </DndContext>
            )}
            <TemplatePickerDialog
                open={templatePickerOpen}
                onClose={() => setTemplatePickerOpen(false)}
                onPick={handlePickTabTemplate}
            />
            <Snackbar
                open={!!saveNotice}
                autoHideDuration={3000}
                onClose={() => setSaveNotice(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
            >
                {saveNotice ? <Alert severity={saveNotice.severity} variant="filled">{saveNotice.message}</Alert> : undefined}
            </Snackbar>
        </Box>
        </SelectionProvider>
    );
};

export default PageEditor;
