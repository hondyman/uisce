import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box, Paper, Typography, Button, Tabs, Tab,
  Grid, IconButton, Chip, Stack, Alert, Tooltip,
  Card, CardContent, CardActionArea, CardActions,
  TextField, InputAdornment, CircularProgress, Divider
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import SaveIcon from '@mui/icons-material/Save';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import EditIcon from '@mui/icons-material/Edit';
import DashboardCustomizeIcon from '@mui/icons-material/DashboardCustomize';
import LayersIcon from '@mui/icons-material/Layers';
import RefreshIcon from '@mui/icons-material/Refresh';
import AddIcon from '@mui/icons-material/Add';
import SearchIcon from '@mui/icons-material/Search';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import WebAssetIcon from '@mui/icons-material/WebAsset';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';

import { PageLayoutSpec, PageWidgetDef } from '../components/pagedesigner/PageDesignerTypes';
import { PageGroupSpec } from '../components/pagedesigner/AutoPageTypes';
import { PageEventBusProvider, usePageEventBus } from '../components/pagedesigner/PageEventBusContext';
import { DynamicBOFormWidget } from '../components/pagedesigner/widgets/DynamicBOFormWidget';
import { DynamicAnalyticsChartWidget } from '../components/pagedesigner/widgets/DynamicAnalyticsChartWidget';
import { InfiniteScrollBOGridWidget } from '../components/pagedesigner/widgets/InfiniteScrollBOGridWidget';
import { InteractiveBOGridWidget } from '../components/pagedesigner/widgets/InteractiveBOGridWidget';
import { AutoPageGeneratorModal } from '../components/pagedesigner/AutoPageGeneratorModal';
import { UnifiedBOPickerModal } from '../components/common/UnifiedBOPickerModal';
import type { BusinessObjectSummary } from '../features/data-explorer/types/dataExplorerTypes';

const DEFAULT_PAGE_SPEC: PageLayoutSpec = {
  pageKey: 'account_master_hub',
  title: 'Institutional Account & Allocation Hub',
  description: 'Reactive cross-filtering application mesh',
  isGoldCopy: true,
  declaredParameters: [
    { key: 'selected_account_id', displayName: 'Selected Account ID', dataType: 'string', defaultValue: 'ACC-901' },
    { key: 'filter_region', displayName: 'Filter Region', dataType: 'string', defaultValue: 'EMEA' },
    { key: 'selected_child_id', displayName: 'Selected Child Entity', dataType: 'string' },
  ],
  sections: [
    {
      id: 'sec_top',
      flow: 'ROW',
      widgets: [
        {
          id: 'w_chart_analytics',
          type: 'QUERY_VISUALIZATION',
          title: 'AUM Distribution by Asset Class',
          boKey: 'account',
          gridSpan: { xs: 12, md: 6, lg: 6 },
          subscribedParams: ['filter_region'],
          events: {
            onChartSelect: [
              { targetChannel: 'filter_region', sourcePropertyKey: 'name', actionType: 'SET_PARAMETER' },
            ],
          },
        },
        {
          id: 'w_crud_form',
          type: 'BO_FORM',
          title: 'Account Detail & Mandate Form',
          boKey: 'account',
          gridSpan: { xs: 12, md: 6, lg: 6 },
          subscribedParams: ['selected_account_id'],
          entitlements: { allowCreate: true, allowUpdate: true, allowDelete: false },
          events: {
            onFormSubmit: [
              { targetChannel: 'selected_account_id', sourcePropertyKey: 'id', actionType: 'SET_PARAMETER' },
            ],
          },
        },
      ],
    },
    {
      id: 'sec_bottom',
      flow: 'COLUMN',
      widgets: [
        {
          id: 'w_grid_positions',
          type: 'BO_GRID',
          title: 'Associated Positions & Live Orders',
          boKey: 'position',
          gridSpan: { xs: 12, md: 12, lg: 12 },
          subscribedParams: ['selected_account_id'],
          events: {
            onRowSelect: [
              { targetChannel: 'selected_child_id', sourcePropertyKey: 'id', actionType: 'SET_PARAMETER' },
            ],
            onRowDoubleClick: [
              { targetChannel: 'active_modal_record_id', sourcePropertyKey: 'id', actionType: 'LAUNCH_MODAL_FORM' },
            ],
          },
        },
      ],
    },
  ],
};

const PageCanvasRenderer: React.FC<{
  layout: PageLayoutSpec;
  activeTabIdx: number;
  pageGroup?: PageGroupSpec | null;
}> = ({ layout, activeTabIdx, pageGroup }) => {
  const { parameters } = usePageEventBus();

  const renderWidget = (widget: PageWidgetDef) => {
    switch (widget.type) {
      case 'BO_FORM':
        return <DynamicBOFormWidget widget={widget} />;
      case 'QUERY_VISUALIZATION':
        return <DynamicAnalyticsChartWidget widget={widget} />;
      case 'BO_GRID':
        return <InteractiveBOGridWidget widget={widget} />;
      default:
        return (
          <Paper sx={{ p: 2, bgcolor: '#071526', color: '#94A3B8', border: '1px solid #1E293B' }}>
            <Typography variant="subtitle2">{widget.title} ({widget.type})</Typography>
          </Paper>
        );
    }
  };

  const sectionsToRender = pageGroup
    ? (pageGroup.tabs[activeTabIdx]?.sections || [])
    : layout.sections;

  return (
    <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
      {/* Active Parameters Banner */}
      <Paper sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid #1E293B', display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
        <Typography variant="caption" sx={{ color: '#00D4FF', fontWeight: 700, textTransform: 'uppercase', mr: 1 }}>
          Event Bus Parameters:
        </Typography>
        {Object.entries(parameters).length === 0 ? (
          <Typography variant="caption" sx={{ color: '#64748B' }}>None active</Typography>
        ) : (
          Object.entries(parameters).map(([k, v]) => (
            <Chip
              key={k}
              size="small"
              label={`${k} = "${v}"`}
              sx={{ bgcolor: '#0B1E36', color: '#38BDF8', border: '1px solid #0284C7', fontSize: 11 }}
            />
          ))
        )}
      </Paper>

      {/* Grid Sections */}
      {sectionsToRender.map((section) => (
        <Box key={section.id} sx={{ mb: 1 }}>
          {section.header?.show && (
            <Typography variant="subtitle2" sx={{ color: '#94A3B8', mb: 1, fontWeight: 600 }}>
              {section.header.title}
            </Typography>
          )}
          <Grid container spacing={2}>
            {section.widgets.map((widget) => (
              <Grid
                size={{
                  xs: widget.gridSpan.xs,
                  sm: widget.gridSpan.sm || widget.gridSpan.xs,
                  md: widget.gridSpan.md,
                  lg: widget.gridSpan.lg,
                }}
                key={widget.id}
              >
                {renderWidget(widget)}
              </Grid>
            ))}
          </Grid>
        </Box>
      ))}
    </Box>
  );
};

export const PageDesignerPage: React.FC = () => {
  const { pageKey } = useParams<{ pageKey?: string }>();
  const navigate = useNavigate();

  // Pages List State for Summary View
  const [pagesList, setPagesList] = useState<any[]>([]);
  const [loadingList, setLoadingList] = useState<boolean>(true);
  const [searchQuery, setSearchQuery] = useState<string>('');

  // Active Editor State
  const [layout, setLayout] = useState<PageLayoutSpec>(DEFAULT_PAGE_SPEC);
  const [pageGroup, setPageGroup] = useState<PageGroupSpec | null>(null);
  const [activeTab, setActiveTab] = useState(0);
  const [mode, setMode] = useState<'preview' | 'edit'>('preview');
  
  // Modals
  const [boPickerOpen, setBoPickerOpen] = useState(false);
  const [generatorOpen, setGeneratorOpen] = useState(false);
  const [selectedBOKKey, setSelectedBOKKey] = useState<string>('account');

  const [saving, setSaving] = useState(false);
  const [alertMsg, setAlertMsg] = useState<string | null>(null);

  // Fetch list of pages for summary view
  const fetchPagesList = () => {
    setLoadingList(true);
    fetch('/api/v1/page-designer/pages')
      .then((res) => (res.ok ? res.json() : []))
      .then((data) => {
        setPagesList(Array.isArray(data) ? data : []);
      })
      .catch(() => {
        setPagesList([]);
      })
      .finally(() => {
        setLoadingList(false);
      });
  };

  useEffect(() => {
    fetchPagesList();
  }, []);

  // When a pageKey is selected, load page details
  useEffect(() => {
    if (!pageKey) return;
    fetch(`/api/v1/page-designer/pages/${pageKey}`)
      .then(async (res) => {
        if (!res.ok) throw new Error('Page not found');
        return res.json();
      })
      .then((data) => {
        if (data.layout_spec && data.layout_spec.sections) {
          setLayout({
            pageKey: data.page_key,
            title: data.title,
            description: data.description,
            isGoldCopy: data.is_gold_copy,
            ...data.layout_spec,
          });
        }
      })
      .catch(() => {
        // Fallback default
      });
  }, [pageKey]);

  const handleSavePage = async () => {
    setSaving(true);
    setAlertMsg(null);
    try {
      const res = await fetch('/api/v1/page-designer/pages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          page_key: layout.pageKey,
          title: layout.title,
          description: layout.description,
          is_gold_copy: layout.isGoldCopy || false,
          layout_spec: {
            declaredParameters: layout.declaredParameters,
            sections: layout.sections,
          },
        }),
      });

      if (!res.ok) throw new Error(await res.text());
      setAlertMsg('Page blueprint committed successfully (Rule 1 & Rule 7).');
      fetchPagesList();
    } catch (err: any) {
      setAlertMsg('Failed to save blueprint: ' + (err.message || ''));
    } finally {
      setSaving(false);
    }
  };

  // Called when user picks a Business Object in the UnifiedBOPickerModal
  const handleBOPick = (bo: BusinessObjectSummary, bindingId?: string, selectedRelatedBOs?: string[], bindingDetails?: any, selectedSubtypeKey?: string | null) => {
    setBoPickerOpen(false);
    const boKeyName = bo.name || bo.id;
    setSelectedBOKKey(boKeyName);
    // Automatically transition to the AutoPageGenerator topology configurator
    setGeneratorOpen(true);
  };

  const handleAutoGenerated = (generatedGroup: PageGroupSpec) => {
    setPageGroup(generatedGroup);
    setActiveTab(0);
    setAlertMsg(`Self-assembled PageGroup "${generatedGroup.title}" generated successfully with ${generatedGroup.tabs.length} tabs.`);
  };

  const filteredPages = pagesList.filter((p) => {
    const q = searchQuery.toLowerCase();
    return (
      (p.title || '').toLowerCase().includes(q) ||
      (p.page_key || '').toLowerCase().includes(q) ||
      (p.description || '').toLowerCase().includes(q)
    );
  });

  const initialParams = (layout.declaredParameters || []).reduce((acc, p) => {
    if (p.defaultValue !== undefined) acc[p.key] = p.defaultValue;
    return acc;
  }, {} as Record<string, any>);

  // Render Summary / Catalog View when no pageKey is in URL
  if (!pageKey) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: '#030B15', color: '#F8FAFC' }}>
        {/* Header */}
        <Paper
          elevation={0}
          sx={{
            p: 2.5,
            bgcolor: '#071526',
            borderBottom: '1px solid #1E293B',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Stack direction="row" spacing={2} alignItems="center">
            <DashboardCustomizeIcon sx={{ color: '#00D4FF', fontSize: 32 }} />
            <Box>
              <Typography variant="h6" fontWeight={700} color="#F8FAFC">
                Application Page Studio & Designer
              </Typography>
              <Typography variant="body2" sx={{ color: '#94A3B8' }}>
                Declarative metadata blueprints, self-assembling metapages & reactive event mesh
              </Typography>
            </Box>
          </Stack>

          <Stack direction="row" spacing={1.5}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<RefreshIcon />}
              onClick={fetchPagesList}
              sx={{ color: '#94A3B8', borderColor: '#1E293B', textTransform: 'none' }}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              size="medium"
              startIcon={<AddIcon />}
              onClick={() => setBoPickerOpen(true)}
              sx={{
                bgcolor: '#0284C7',
                fontWeight: 700,
                textTransform: 'none',
                px: 2.5,
                background: 'linear-gradient(135deg, #0284C7 0%, #2563EB 100%)',
                boxShadow: '0 4px 12px rgba(37,99,235,0.3)',
              }}
            >
              Create New Page
            </Button>
          </Stack>
        </Paper>

        {/* Search & Filter Bar */}
        <Box sx={{ p: 3, pb: 1, maxWidth: 1400, mx: 'auto', width: '100%' }}>
          <Stack direction="row" spacing={2} alignItems="center" justifyContent="space-between" sx={{ mb: 3 }}>
            <TextField
              size="small"
              placeholder="Search application pages by title, key, or entity..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon sx={{ color: '#64748B', fontSize: 20 }} />
                  </InputAdornment>
                ),
              }}
              sx={{
                width: 420,
                bgcolor: '#071526',
                borderRadius: 1,
                input: { color: '#F8FAFC', fontSize: 13 },
                '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1E293B' },
              }}
            />
            <Typography variant="body2" sx={{ color: '#64748B' }}>
              Showing {filteredPages.length} active page blueprints
            </Typography>
          </Stack>

          {/* Cards Grid */}
          {loadingList ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 10 }}>
              <CircularProgress sx={{ color: '#00D4FF' }} />
            </Box>
          ) : filteredPages.length === 0 ? (
            <Paper
              sx={{
                p: 6,
                textAlign: 'center',
                bgcolor: '#071526',
                border: '1px dashed #1E293B',
                borderRadius: 2,
                mt: 2,
              }}
            >
              <WebAssetIcon sx={{ fontSize: 56, color: '#334155', mb: 2 }} />
              <Typography variant="h6" fontWeight={600} color="#E2E8F0" gutterBottom>
                No Page Blueprints Found
              </Typography>
              <Typography variant="body2" color="#64748B" sx={{ mb: 3, maxWidth: 460, mx: 'auto' }}>
                Create your first application page by selecting a core Business Object. The Self-Assembling Metapage Engine will discover subtypes and wire reactive CRUD forms and virtual grids automatically.
              </Typography>
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                onClick={() => setBoPickerOpen(true)}
                sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 700 }}
              >
                Create New Page
              </Button>
            </Paper>
          ) : (
            <Grid container spacing={2.5}>
              {filteredPages.map((page) => (
                <Grid size={{ xs: 12, sm: 6, md: 4 }} key={page.page_key}>
                  <Card
                    sx={{
                      bgcolor: '#071526',
                      border: '1px solid #1E293B',
                      borderRadius: 2,
                      height: '100%',
                      display: 'flex',
                      flexDirection: 'column',
                      transition: 'all 0.2s ease',
                      '&:hover': {
                        borderColor: '#00D4FF',
                        transform: 'translateY(-2px)',
                        boxShadow: '0 8px 24px rgba(0, 212, 255, 0.12)',
                      },
                    }}
                  >
                    <CardActionArea
                      onClick={() => navigate(`/page-designer/${page.page_key}`)}
                      sx={{ flex: 1, p: 2.5, alignItems: 'flex-start' }}
                    >
                      <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1.5 }}>
                        <WebAssetIcon sx={{ color: '#00D4FF', fontSize: 24 }} />
                        <Box sx={{ flex: 1 }}>
                          <Typography variant="subtitle1" fontWeight={700} color="#F8FAFC" noWrap>
                            {page.title}
                          </Typography>
                          <Typography variant="caption" sx={{ color: '#64748B', fontFamily: 'monospace' }}>
                            {page.page_key}
                          </Typography>
                        </Box>
                        {page.is_gold_copy && (
                          <Chip
                            size="small"
                            label="GOLD COPY"
                            sx={{ height: 18, fontSize: 9, bgcolor: '#D97706', color: '#FFF', fontWeight: 700 }}
                          />
                        )}
                      </Stack>

                      <Typography variant="body2" sx={{ color: '#94A3B8', fontSize: 12, minHeight: 36, mb: 2 }}>
                        {page.description || 'Configured dynamic reactive widget canvas'}
                      </Typography>

                      <Divider sx={{ borderColor: '#1E293B', mb: 1.5 }} />

                      <Stack direction="row" spacing={1} justifyContent="space-between" alignItems="center">
                        <Typography variant="caption" sx={{ color: '#64748B' }}>
                          Updated {new Date(page.updated_at || page.created_at).toLocaleDateString()}
                        </Typography>
                        <Chip
                          size="small"
                          label={`${page.layout_spec?.sections?.length || 0} Sections`}
                          sx={{ height: 20, fontSize: 10, bgcolor: '#0B1E36', color: '#38BDF8', border: '1px solid #1E293B' }}
                        />
                      </Stack>
                    </CardActionArea>
                  </Card>
                </Grid>
              ))}
            </Grid>
          )}
        </Box>

        {/* Reusable Unified Business Object Selection Modal */}
        <UnifiedBOPickerModal
          open={boPickerOpen}
          context="page"
          onClose={() => setBoPickerOpen(false)}
          onPick={handleBOPick}
          onSelect={handleBOPick}
        />

        {/* Auto-Page Topology & Entitlement Configurator */}
        <AutoPageGeneratorModal
          open={generatorOpen}
          boKey={selectedBOKKey}
          onClose={() => setGeneratorOpen(false)}
          onGenerated={(generatedGroup) => {
            handleAutoGenerated(generatedGroup);
            // Navigate to newly created draft
            navigate(`/page-designer/${generatedGroup.pageGroupId}`);
          }}
        />
      </Box>
    );
  }

  // Render Detail / Canvas Studio View when pageKey is present
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: '#030B15', color: '#F8FAFC' }}>
      {/* Top Header Bar */}
      <Paper
        elevation={0}
        sx={{
          p: 1.5,
          bgcolor: '#071526',
          borderBottom: '1px solid #1E293B',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Stack direction="row" spacing={1.5} alignItems="center">
          <Tooltip title="Back to Pages Directory">
            <IconButton
              size="small"
              onClick={() => navigate('/page-designer')}
              sx={{ color: '#94A3B8', '&:hover': { color: '#F8FAFC' } }}
            >
              <ArrowBackIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <DashboardCustomizeIcon sx={{ color: '#00D4FF' }} />
          <Box>
            <Typography variant="subtitle1" fontWeight={700} color="#F8FAFC">
              {pageGroup ? pageGroup.title : layout.title}
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Page Designer & Metapage Mesh • {layout.pageKey} {layout.isGoldCopy && <Chip size="small" label="GOLD COPY" sx={{ height: 16, fontSize: 9, bgcolor: '#D97706', color: '#FFF' }} />}
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={1} alignItems="center">
          <Button
            variant="outlined"
            size="small"
            startIcon={<AutoAwesomeIcon />}
            onClick={() => setBoPickerOpen(true)}
            sx={{ color: '#00D4FF', borderColor: '#0284C7', textTransform: 'none' }}
          >
            Switch / Generate BO Mesh
          </Button>

          <Button
            variant="outlined"
            size="small"
            startIcon={mode === 'preview' ? <EditIcon /> : <PlayArrowIcon />}
            onClick={() => setMode(mode === 'preview' ? 'edit' : 'preview')}
            sx={{ color: '#38BDF8', borderColor: '#1E293B', textTransform: 'none' }}
          >
            {mode === 'preview' ? 'Design Mode' : 'Live Preview'}
          </Button>

          <Button
            variant="contained"
            size="small"
            startIcon={<SaveIcon />}
            onClick={handleSavePage}
            disabled={saving}
            sx={{ bgcolor: '#0284C7', textTransform: 'none' }}
          >
            {saving ? 'Saving...' : 'Save Blueprint'}
          </Button>
        </Stack>
      </Paper>

      {alertMsg && (
        <Alert
          severity={alertMsg.includes('Failed') ? 'error' : 'success'}
          onClose={() => setAlertMsg(null)}
          sx={{ mx: 2, mt: 1, py: 0.5, fontSize: 12 }}
        >
          {alertMsg}
        </Alert>
      )}

      {/* Tabs if PageGroup is active */}
      {pageGroup && pageGroup.tabs.length > 1 && (
        <Box sx={{ px: 2, bgcolor: '#071526', borderBottom: '1px solid #1E293B' }}>
          <Tabs
            value={activeTab}
            onChange={(_, v) => setActiveTab(v)}
            sx={{
              minHeight: 40,
              '& .MuiTab-root': { color: '#94A3B8', fontSize: 12, minHeight: 40, textTransform: 'none' },
              '& .Mui-selected': { color: '#00D4FF', fontWeight: 700 },
            }}
          >
            {pageGroup.tabs.map((tab, idx) => (
              <Tab key={tab.tabId} label={tab.tabTitle} />
            ))}
          </Tabs>
        </Box>
      )}

      {/* Reactive Metapage Canvas */}
      <Box sx={{ flex: 1, overflowY: 'auto' }}>
        <PageEventBusProvider initialParams={initialParams}>
          <PageCanvasRenderer
            layout={layout}
            activeTabIdx={activeTab}
            pageGroup={pageGroup}
          />
        </PageEventBusProvider>
      </Box>

      {/* Reusable Unified Business Object Selection Modal */}
      <UnifiedBOPickerModal
        open={boPickerOpen}
        context="page"
        onClose={() => setBoPickerOpen(false)}
        onPick={handleBOPick}
        onSelect={handleBOPick}
      />

      {/* Auto-Page Topology & Entitlement Configurator */}
      <AutoPageGeneratorModal
        open={generatorOpen}
        boKey={selectedBOKKey}
        onClose={() => setGeneratorOpen(false)}
        onGenerated={handleAutoGenerated}
      />
    </Box>
  );
};

export default PageDesignerPage;

