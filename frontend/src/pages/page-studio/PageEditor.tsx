import React, { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Box, Typography, Paper, Tabs, Tab, Button, Grid, IconButton, Tooltip, Divider, TextField, Snackbar, Alert, ToggleButton, ToggleButtonGroup, FormControlLabel, Switch } from '@mui/material';
import { DndContext, PointerSensor, useSensor, useSensors, closestCenter, pointerWithin, rectIntersection, type CollisionDetection } from '@dnd-kit/core';
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
import { CorePageDefinition, PageLayout, PageTab, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import { PageStudioApi } from '../../api/pageStudio';
import ComponentPalette from './ComponentPalette';
import ObjectPalette from './ObjectPalette';
import LayoutCanvas from './LayoutCanvas';
import PropertiesPanel from './PropertiesPanel';
import DataBindingsPanel from './DataBindingsPanel';
import PresentationEventsPanel from './PresentationEventsPanel';
import { PresentationProvider } from './PresentationRuntime';
import { mergeGeneratedSpecIntoDraft } from './generatePageDraft';
import { PagePerformanceDashboard } from './PagePerformanceDashboard';
import { AIDocumentationViewer } from './AIDocumentationViewer';
import { AITestGenerator } from './AITestGenerator';
import { Description as DocIcon, BugReport as TestIcon, Edit as EditIcon, FlashOn as EventsIcon, AutoAwesome as CopilotIcon } from '@mui/icons-material';
import DraftPreview from './DraftPreview';
import TemplatePickerDialog from './TemplatePickerDialog';
import { SelectionProvider } from './SelectionContext';
import PageArtboard from './PageArtboard';
import PageBody from './PageBody';
import { CANVAS_SIZES, canvasWidthPx, type CanvasSizeId } from './canvasSizes';

interface PageEditorProps {
    page: CorePageDefinition;
    onSave: (page: CorePageDefinition) => void;
}

const DEFAULT_TAB_ID = '__default__';

const EMPTY_PAGE_LAYOUT: PageLayout = { root: 'filter_root', nodes: { filter_root: { id: 'filter_root', type: 'Row', children: [] } } };

/**
 * A page saved through the old page_studio_handler.go default (fixed
 * server-side, see HANDOFF_REPORT_BUILDER_SPINE_PLAN.md "Order Detail
 * crash fix") can have `filterBar`/`layout: {}` on disk today - valid
 * JSON, but missing the `root`/`nodes` PageLayout requires, and `{}` is
 * truthy so a plain `value || <default>` fallback never catches it.
 * Validates shape, not just presence, so already-saved malformed values
 * heal on read without a data migration. LayoutCanvas.tsx's own boundary
 * guard is the last-resort backstop if a malformed value reaches it by
 * some other path.
 */
const safePageLayout = (layout: PageLayout | undefined | null): PageLayout =>
    layout && typeof layout.root === 'string' && layout.nodes ? layout : EMPTY_PAGE_LAYOUT;

/** Palette/slot drops use pointer/rect intersection; form-field reorder
 * (FormFieldsDesigner) uses closestCenter among form-field ids only so a
 * grip-drag does not snap to a neighboring widget slot. */
const collisionDetection: CollisionDetection = (args) => {
    const kind = args.active?.data?.current?.kind as string | undefined;
    if (kind === 'related-object' || kind === 'component') {
        const relevant = args.droppableContainers.filter((d) => !String(d.id).startsWith('form-field:'));
        const pointerHits = pointerWithin({ ...args, droppableContainers: relevant });
        if (pointerHits.length > 0) return pointerHits;
        return rectIntersection({ ...args, droppableContainers: relevant });
    }
    const pointerHits = pointerWithin(args);
    const formFieldHits = pointerHits.filter((c) => String(c.id).startsWith('form-field:'));
    if (formFieldHits.length > 0) {
        const formFieldContainers = args.droppableContainers.filter((d) => String(d.id).startsWith('form-field:'));
        return closestCenter({ ...args, droppableContainers: formFieldContainers });
    }
    return pointerHits.length > 0 ? pointerHits : rectIntersection(args);
};

const PageEditor: React.FC<PageEditorProps> = ({ page, onSave }) => {
    const [searchParams, setSearchParams] = useSearchParams();
    const aiDraft = searchParams.get('aiDraft') === '1';
    const [draft, setDraft] = useState<CorePageDefinition>(page);
    const [copilot, setCopilot] = useState('');
    const [copilotBusy, setCopilotBusy] = useState(false);
    const [copilotError, setCopilotError] = useState<string | null>(null);
    const [tab, setTab] = useState(0); // 0: Design, 1: Data, 2: Events, 3: Performance, 4: Docs, 5: Testing
    const [selectedId, setSelectedId] = useState<string | null>(null);
    // Design/Preview: the Preview button previously had no onClick handler
    // at all - clicking it did nothing. Preview renders the live in-memory
    // draft (DraftPreview) the same way a consumer would see it, without
    // requiring a save first.
    const [viewMode, setViewMode] = useState<'design' | 'preview'>('design');
    const [templatePickerOpen, setTemplatePickerOpen] = useState(false);
    const [saveNotice, setSaveNotice] = useState<{ severity: 'success' | 'error'; message: string } | null>(null);
    const [propertiesPanelOpen, setPropertiesPanelOpen] = useState(true);
    const [paletteOpen, setPaletteOpen] = useState(true);
    const [canvasSize, setCanvasSize] = useState<CanvasSizeId>('page');
    const [fitToWorkspace, setFitToWorkspace] = useState(false);
    const [viewportWidth, setViewportWidth] = useState(() => (typeof window === 'undefined' ? 1280 : window.innerWidth));
    useEffect(() => {
        const onResize = () => setViewportWidth(window.innerWidth);
        window.addEventListener('resize', onResize);
        return () => window.removeEventListener('resize', onResize);
    }, []);
    const artboardWidth = canvasWidthPx(canvasSize, viewportWidth);
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
        : [{ id: DEFAULT_TAB_ID, label: draft.name || 'Page 1', layout: safePageLayout(draft.layout) }];
    const [activeTabId, setActiveTabId] = useState(pageTabs[0].id);
    const activeTab = pageTabs.find((t) => t.id === activeTabId) || pageTabs[0];

    // Page-wide filter bar, rendered above the tab strip so a Slicer placed
    // here scopes every tab instead of just one - synthesized empty the
    // same way pageTabs falls back to draft.layout when no tabs exist yet.
    // safePageLayout, not `draft.filterBar || <default>`: a page saved
    // through the old page_studio_handler.go default (fixed server-side,
    // see HANDOFF_REPORT_BUILDER_SPINE_PLAN.md "Order Detail crash fix")
    // has `filterBar: {}` on disk today - valid JSON, but missing
    // root/nodes, and `{}` is truthy so `||` never falls back to the
    // synthesized default. Validating shape here heals those already-saved
    // rows on read without a data migration; LayoutCanvas.tsx's own
    // boundary guard is the last-resort backstop if a malformed value gets
    // through some other path.
    const filterBarLayout: PageLayout = safePageLayout(draft.filterBar);
    const handleFilterBarLayoutChange = (updater: (layout: PageLayout) => PageLayout) => {
        setDraft((prev) => ({
            ...prev,
            filterBar: updater(safePageLayout(prev.filterBar)),
        }));
    };

    const handleLayoutChange = (updater: (layout: PageLayout) => PageLayout) => {
        setDraft((prev) => {
            const tabs = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: safePageLayout(prev.layout) }];
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

    const handleDataSourcesChange = (updater: (dataSources: CorePageDefinition['dataSources']) => CorePageDefinition['dataSources']) => {
        setDraft((prev) => ({ ...prev, dataSources: updater(prev.dataSources || []) }));
    };

    const handleAddTab = () => setTemplatePickerOpen(true);

    const handlePickTabTemplate = ({ layout }: { layout: PageLayout }) => {
        setDraft((prev) => {
            const existing = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: safePageLayout(prev.layout) }];
            const tabs = [...existing, { id: `tab_${Math.random().toString(36).slice(2, 8)}`, label: `Tab ${existing.length + 1}`, layout }];
            setActiveTabId(tabs[tabs.length - 1].id);
            return { ...prev, tabs };
        });
    };

    const handleRenameTab = (tabId: string, label: string) => {
        setDraft((prev) => {
            const existing = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: safePageLayout(prev.layout) }];
            return { ...prev, tabs: existing.map((t) => (t.id === tabId ? { ...t, label } : t)) };
        });
    };

    const handleDeleteTab = (tabId: string) => {
        setDraft((prev) => {
            const existing = prev.tabs && prev.tabs.length > 0
                ? prev.tabs
                : [{ id: DEFAULT_TAB_ID, label: prev.name || 'Page 1', layout: safePageLayout(prev.layout) }];
            if (existing.length <= 1) return prev;
            const tabs = existing.filter((t) => t.id !== tabId);
            if (activeTabId === tabId) setActiveTabId(tabs[0].id);
            return { ...prev, tabs };
        });
    };

    const handleCopilot = async () => {
        const instruction = copilot.trim();
        if (!instruction || copilotBusy) return;
        const primary = (draft.dataSources || []).find((d) => d.type === 'business_object');
        const cfg = primary?.config as unknown as BusinessObjectDataSourceConfig | undefined;
        if (!cfg?.boId) {
            setCopilotError('Bind a primary Business Object first.');
            return;
        }
        setCopilotBusy(true);
        setCopilotError(null);
        try {
            const spec = await PageStudioApi.generateSpec(
                cfg.boId,
                cfg.boKey,
                cfg.displayName || cfg.boKey,
                instruction,
                draft.pageKind || 'dashboard',
            );
            const next = await mergeGeneratedSpecIntoDraft(draft, spec);
            setDraft(next);
            setCopilot('');
        } catch (err) {
            setCopilotError(err instanceof Error ? err.message : 'Copilot failed');
        } finally {
            setCopilotBusy(false);
        }
    };

    const handleSave = async () => {
        try {
            // A page opened from the sidebar list carries its real id;
            // a fresh "Create New Page" draft has none. Without this
            // branch every save POSTed as a create, so re-saving an
            // existing page hit the (tenant_id, slug) unique constraint
            // as a 409 instead of updating it.
            const saved = !draft.id
                ? await PageStudioApi.savePage(draft)
                : draft.isCore && draft.editable === false
                  ? await PageStudioApi.saveOverlay(draft.id, { components: draft.components, layout: draft.layout, tabs: draft.tabs })
                  : await PageStudioApi.updatePage(draft.id, draft);
            onSave(saved);
            setSaveNotice({ severity: 'success', message: draft.isCore && draft.editable === false ? 'Saved as tenant overlay (core unchanged)' : 'Saved' });
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
        <PresentationProvider rules={draft.presentationEvents || []}>
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            {/* Header / Actions */}
            <Paper elevation={0} sx={{ p: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid rgba(0,0,0,0.05)' }}>
                <Box>
                    <Typography variant="subtitle1" fontWeight="bold">{draft.name}</Typography>
                    <Typography variant="caption" color="textSecondary">{draft.slug} • v{draft.version}</Typography>
                </Box>
                <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
                    <ToggleButtonGroup
                        exclusive
                        size="small"
                        value={canvasSize}
                        onChange={(_, v: CanvasSizeId | null) => { if (v) setCanvasSize(v); }}
                    >
                        {CANVAS_SIZES.map((s) => (
                            <ToggleButton key={s.id} value={s.id} sx={{ px: 1, textTransform: 'none', fontSize: 12 }}>
                                <Tooltip title={s.hint}><span>{s.label}</span></Tooltip>
                            </ToggleButton>
                        ))}
                    </ToggleButtonGroup>
                    <FormControlLabel
                        sx={{ mr: 0, ml: 0.5 }}
                        control={<Switch size="small" checked={fitToWorkspace} onChange={(e) => setFitToWorkspace(e.target.checked)} />}
                        label={<Typography variant="caption">Fit</Typography>}
                    />
                    <Typography variant="caption" color="text.secondary" sx={{ minWidth: 44 }}>{artboardWidth}px</Typography>
                    <Button
                        size="small"
                        variant="outlined"
                        onClick={() => {
                            const next = !(paletteOpen && propertiesPanelOpen);
                            setPaletteOpen(next);
                            setPropertiesPanelOpen(next);
                        }}
                    >
                        {paletteOpen && propertiesPanelOpen ? 'Hide panels' : 'Show panels'}
                    </Button>
                    {viewMode === 'design' ? (
                        <Button variant="outlined" startIcon={<PreviewIcon />} size="small" onClick={() => setViewMode('preview')}>Preview</Button>
                    ) : (
                        <Button variant="contained" color="secondary" startIcon={<EditIcon />} size="small" onClick={() => setViewMode('design')}>Back to Design</Button>
                    )}
                    <Divider orientation="vertical" flexItem sx={{ mx: 1 }} />
                    <Button variant="contained" startIcon={<SaveIcon />} size="small" onClick={handleSave}>Save Changes</Button>
                </Box>
            </Paper>
            {draft.isCore && (
                <Alert severity={draft.editable === false ? 'info' : 'success'} sx={{ mx: 2, mt: 1 }}>
                    {draft.editable === false
                        ? 'This is a gold-copy core page. You can add tenant overlay widgets; core FIX commands cannot be edited here.'
                        : 'Gold-copy core page. Only gold-copy admins can change core. Tenants inherit and extend via overlay.'}
                </Alert>
            )}
            {aiDraft && (
                <Alert
                    severity="info"
                    onClose={() => {
                        searchParams.delete('aiDraft');
                        setSearchParams(searchParams, { replace: true });
                    }}
                    sx={{ borderRadius: 0 }}
                >
                    AI draft — refine here. Copilot follow-ups add widgets; they do not rebuild the page from scratch. Include / Place / Bind / Format still apply.
                </Alert>
            )}
            <Box sx={{ px: 1.5, py: 1, display: 'flex', gap: 1, alignItems: 'center', borderBottom: '1px solid', borderColor: 'divider' }}>
                <CopilotIcon fontSize="small" color="primary" />
                <TextField
                    size="small"
                    fullWidth
                    placeholder="Copilot: add Execution as a related table, add a Status slicer…"
                    value={copilot}
                    onChange={(e) => setCopilot(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') void handleCopilot(); }}
                    disabled={copilotBusy}
                />
                <Button size="small" variant="outlined" onClick={() => void handleCopilot()} disabled={copilotBusy || !copilot.trim()}>
                    {copilotBusy ? 'Working…' : 'Apply'}
                </Button>
            </Box>
            {copilotError && <Alert severity="warning" onClose={() => setCopilotError(null)} sx={{ borderRadius: 0 }}>{copilotError}</Alert>}

            {viewMode === 'preview' ? (
                <PageArtboard width={artboardWidth} fit={fitToWorkspace}>
                    <DraftPreview draft={draft} tenantId={draft.tenantId || 'default'} framed />
                </PageArtboard>
            ) : (
            <DndContext sensors={dndSensors} collisionDetection={collisionDetection}>
            <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
                {/* Left Panel: Palette */}
                <Box sx={{ display: 'flex', flexShrink: 0 }}>
                <Paper elevation={0} sx={{ width: paletteOpen ? 250 : 0, overflow: 'hidden', borderRight: '1px solid rgba(0,0,0,0.05)', display: 'flex', flexDirection: 'column', transition: 'width 0.2s ease' }}>
                    <Box sx={{ width: 250, height: '100%', display: 'flex', flexDirection: 'column' }}>
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 48 }}>
                            <Tab label="Design" icon={<DesignIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Data Binding" icon={<DataIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Events" icon={<EventsIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Performance" icon={<PerformanceIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Docs" icon={<DocIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                            <Tab label="Testing" icon={<TestIcon sx={{ fontSize: 18 }} />} iconPosition="start" />
                        </Tabs>
                    </Box>
                    <Box sx={{ flex: 1, overflowY: 'auto' }}>
                        {tab === 0 && (
                            <>
                                <ObjectPalette draft={draft} tenantId={draft.tenantId || 'default'} />
                                <ComponentPalette />
                            </>
                        )}
                        {tab === 1 && (
                            <DataBindingsPanel
                                draft={draft}
                                setDraft={setDraft}
                                tenantId={draft.tenantId || 'default'}
                            />
                        )}
                        {tab === 2 && (
                            <PresentationEventsPanel draft={draft} setDraft={setDraft} />
                        )}
                        {tab === 3 && (
                            <PagePerformanceDashboard pageId={draft.id!} />
                        )}
                        {tab === 4 && (
                            <AIDocumentationViewer page={draft} />
                        )}
                        {tab === 5 && (
                            <AITestGenerator page={draft} />
                        )}
                    </Box>
                    </Box>
                </Paper>
                <Box
                    sx={{
                        display: 'flex', alignItems: 'flex-start', pt: 1.5,
                        borderRight: '1px solid rgba(0,0,0,0.05)', bgcolor: 'background.paper',
                    }}
                >
                    <Tooltip title={paletteOpen ? 'Collapse palette' : 'Expand palette'}>
                        <IconButton size="small" onClick={() => setPaletteOpen((v) => !v)}>
                            {paletteOpen ? <ChevronLeftIcon fontSize="small" /> : <ChevronRightIcon fontSize="small" />}
                        </IconButton>
                    </Tooltip>
                </Box>
                </Box>

                {/* Main: Page tabs + Layout Canvas */}
                <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
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
                    <PageArtboard width={artboardWidth} fit={fitToWorkspace}>
                        <PageBody name={draft.name} slug={draft.slug}>
                        <Box sx={{ mb: 2 }}>
                            <LayoutCanvas
                                layout={filterBarLayout}
                                onLayoutChange={handleFilterBarLayoutChange}
                                components={draft.components}
                                onComponentsChange={handleComponentsChange}
                                dataSources={draft.dataSources}
                                onDataSourcesChange={handleDataSourcesChange}
                                tenantId={draft.tenantId || 'default'}
                                selectedId={selectedId}
                                onSelect={setSelectedId}
                            />
                        </Box>
                        <LayoutCanvas
                            layout={activeTab.layout}
                            onLayoutChange={handleLayoutChange}
                            components={draft.components}
                            onComponentsChange={handleComponentsChange}
                            dataSources={draft.dataSources}
                            onDataSourcesChange={handleDataSourcesChange}
                            tenantId={draft.tenantId || 'default'}
                            selectedId={selectedId}
                            onSelect={setSelectedId}
                        />
                        </PageBody>
                    </PageArtboard>
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
        </PresentationProvider>
        </SelectionProvider>
    );
};

export default PageEditor;
