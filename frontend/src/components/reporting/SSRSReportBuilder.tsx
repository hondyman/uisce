import React, { useState, useCallback, useMemo } from 'react';
import { DndContext, DragOverlay, useDraggable as _useDraggable, useDroppable as _useDroppable } from '@dnd-kit/core';
import { Box, Drawer, Typography, Tabs, Tab, Paper, Grid, LinearProgress, Pagination, Snackbar, Alert, Divider, Chip, InputAdornment, FormControlLabel, Switch, Card, TextField, Button, Tooltip, IconButton } from '@mui/material';
import { QueryClient, QueryClientProvider, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import useUndo from 'use-undo';

// Modular components & utils
import ToolboxItem from './ToolboxItem';
import ReportCanvas from './ReportCanvas';
import PropertiesPanel from './PropertiesPanel';
import DataSourcesDialog from './DataSourcesDialog';
import ParametersDialog from './ParametersDialog';
import TopAppBar from './TopAppBar';
import PageSettings from './PageSettings';
import { ELEMENT_TYPES, dataSources as staticDataSources, datasets, generatePixelPerfectPDF, dynamicTokens as _dynamicTokens, sanitizeInput, exportFormatLabels, exportOptionDescriptions, EventScripts, ExportOptions } from './reportingUtils';
import GroupsEditor from './GroupsEditor';
import CalculatedFieldsEditor from './CalculatedFieldsEditor';
import ExpressionsEditor from './ExpressionsEditor';
import EventScriptsEditor from './EventScriptsEditor';
import { Database, Table as TableIcon, BarChart3, Type, Image, FileText, Square, Minus, Gauge, Activity, Grid3X3, List as ListIcon, Plus, Save, Download, Printer, Eye, Undo2, Redo2, Settings, LayoutDashboard } from 'lucide-react';

import axios from 'axios';

type ReportParameter = {
  id: string;
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean';
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
};

// API helpers (kept lightweight here)
const API_BASE_URL = 'http://localhost:9088/api/v1';
const api = axios.create({ baseURL: API_BASE_URL, headers: { 'Content-Type': 'application/json' } });
const fetchReportTemplates = async () => { const token = localStorage.getItem('token'); const response = await api.get('/reports', { headers: { Authorization: `Bearer ${token}` } }); return response.data; };
const createReportTemplate = async (template: any) => { const token = localStorage.getItem('token'); const response = await api.post('/reports', template, { headers: { Authorization: `Bearer ${token}` } }); return response.data; };
// const runReport = async (templateId: string) => { const token = localStorage.getItem('token'); const response = await api.post(`/reports/${templateId}/run`, {}, { headers: { Authorization: `Bearer ${token}` } }); return response.data; };
const fetchDataSources = async () => { const token = localStorage.getItem('token'); const response = await api.get('/data-sources', { headers: { Authorization: `Bearer ${token}` } }); return response.data; };
// const createDataSource = async (dataSource: any) => { const token = localStorage.getItem('token'); const response = await api.post('/data-sources', dataSource, { headers: { Authorization: `Bearer ${token}` } }); return response.data; };

const SSRSReportBuilderContent: React.FC = () => {
  const queryClient = useQueryClient();

  const [elementsState, { set: setElements, undo, redo, canUndo, canRedo }] = useUndo<any[]>([]);
  const elements = elementsState.present;

  const [selectedElement, setSelectedElement] = useState<string | null>(null);
  const [activeDragItem, setActiveDragItem] = useState<any>(null);
  const [activeTab, setActiveTab] = useState('design');
  const [designTab, setDesignTab] = useState(0);
  const [drawerOpen] = useState(true);
  const [dataSourcesOpen, setDataSourcesOpen] = useState(false);
  const [parametersOpen, setParametersOpen] = useState(false);
  const [layoutDrawerOpen, setLayoutDrawerOpen] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState('A4');
  const [orientation, setOrientation] = useState('Portrait');
  const [layoutSettings] = useState<any>({
    pageBreakBeforeGroup: false,
    pageBreakAfterGroup: true,
    pageBreakBetweenRegions: true,
    fixedPageSize: true,
    columns: 1,
    columnSpacing: 24,
    headerTokens: ['Page {PageNumber} of {TotalPages}', 'User: {UserName}'],
    footerTokens: ['Generated: {ExecutionTime}', 'Confidential'],
    includeExecutionTime: true,
    includeUserName: true,
  });

  // Groups, calculated fields, expressions, event scripts, export options
  const [groupDefinitions, setGroupDefinitions] = useState<any[]>([
    {
      id: 'grp_region',
      name: 'Region Group',
      expression: '=Fields!Region',
      aggregates: [
        { id: 'agg_region_sales', field: 'sales', function: 'SUM', scope: 'Group', displayName: 'Regional Sales' },
      ],
      pageBreakAfter: true,
    },
  ]);

  const [calculatedFields, setCalculatedFields] = useState<any[]>([
    { id: 'calc_margin', name: 'GrossMargin', expression: '=Fields!Revenue - Fields!Cost', datasetId: datasets[0]?.id ?? 'ds1', format: 'Currency' },
  ]);

  const [expressionLibrary, setExpressionLibrary] = useState<string[]>([
    '=IIF(Fields!Growth.Value < 0, "#DC2626", "#16A34A")',
    '=Sum(Fields!Sales.Value, "Region Group")',
  ]);

  const [eventScripts, setEventScripts] = useState<EventScripts>({
    onRowRender: '// format negative growth rows\nif (row.Fields.Growth < 0) { row.Style.Background = "#FEF2F2"; }',
    onCellRender: '// add tooltip\ncell.Tooltip = "{Region}: {Value}";',
    onPageRender: '// watermark\npage.Watermark = "Internal";',
    onExport: '// append metadata\nexportContext.Metadata.author = user.name;',
  });

  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    includePrintFriendly: true,
    includeDrillThrough: true,
    includeComments: false,
  });

  const [reportParameters, setReportParameters] = useState<ReportParameter[]>([
    { id: 'param_region', name: 'Region', type: 'string', prompt: 'Select a Region', defaultValue: 'North America' },
    { id: 'param_year', name: 'Year', type: 'number', prompt: 'Enter a Year', defaultValue: String(new Date().getFullYear()) },
  ]);

  const handleAddParameter = (param: Omit<ReportParameter, 'id'>) => setReportParameters(prev => [...prev, { ...param, id: `param_${Date.now()}` }]);
  const handleUpdateParameter = (updatedParam: ReportParameter) => setReportParameters(prev => prev.map(p => p.id === updatedParam.id ? updatedParam : p));
  const handleRemoveParameter = (paramId: string) => setReportParameters(prev => prev.filter(p => p.id !== paramId));


  const [headerTokenInput, setHeaderTokenInput] = useState('');
  const [footerTokenInput, setFooterTokenInput] = useState('');

  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'info' | 'warning' | 'error' });
  
  // Preview data state
  const [previewData, setPreviewData] = useState<any[] | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewSQL, setPreviewSQL] = useState<string>('');
  const [previewError, setPreviewError] = useState<string | null>(null);

  // API Queries and Mutations
  const { isLoading: templatesLoading } = useQuery({ queryKey: ['reportTemplates'], queryFn: fetchReportTemplates });
  const { data: fetchedDataSources, isLoading: dataSourcesLoading } = useQuery({ queryKey: ['dataSources'], queryFn: fetchDataSources });

  const createTemplateMutation = useMutation({ mutationFn: createReportTemplate, onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reportTemplates'] }) });
  // Mutations intentionally kept for future use; comment out unused ones to avoid lint errors.
  // const createDataSourceMutation = useMutation({ mutationFn: createDataSource, onSuccess: () => queryClient.invalidateQueries({ queryKey: ['dataSources'] }) });
  // const runReportMutation = useMutation({ mutationFn: runReport });

  // Layout helpers (kept for later; currently unused and commented to avoid lint warnings)
  const [layoutSettingsState, setLayoutSettingsState] = useState(layoutSettings);

  const handleLayoutSettingChange = <K extends keyof typeof layoutSettingsState>(key: K, value: any) => {
    setLayoutSettingsState((prev: any) => ({ ...prev, [key]: value }));
  };

  const handleAddToken = (target: 'headerTokens' | 'footerTokens', token: string) => {
    const sanitizedToken = sanitizeInput(token);
    setLayoutSettingsState((prev: any) => {
      if (prev[target].includes(sanitizedToken)) return prev;
      return { ...prev, [target]: [...prev[target], sanitizedToken] };
    });
  };

  const handleRemoveToken = (target: 'headerTokens' | 'footerTokens', token: string) => {
    setLayoutSettingsState((prev: any) => ({ ...prev, [target]: prev[target].filter((item: string) => item !== token) }));
  };

  const handleAddGroup = () => {
    const nextIndex = groupDefinitions.length + 1;
    setGroupDefinitions((prev) => ([...prev, { id: `grp_${nextIndex}`, name: `Group ${nextIndex}`, expression: '=Fields!Category', aggregates: [] }]));
  };

  const handleGroupChange = (groupId: string, key: string, value: any) => {
    setGroupDefinitions((prev) => prev.map((g) => (g.id === groupId ? { ...g, [key]: value } : g)));
  };

  const handleAddAggregate = (groupId: string) => {
    const aggregate = { id: `${groupId}_agg_${Date.now()}`, field: 'sales', function: 'SUM', scope: 'Group', displayName: 'Aggregate' };
    setGroupDefinitions((prev) => prev.map((g) => (g.id === groupId ? { ...g, aggregates: [...g.aggregates, aggregate] } : g)));
  };

  const handleAggregateChange = (groupId: string, aggregateId: string, key: string, value: any) => {
    setGroupDefinitions((prev) => prev.map((g) => {
      if (g.id !== groupId) return g;
      return { ...g, aggregates: g.aggregates.map((agg: any) => (agg.id === aggregateId ? { ...agg, [key]: value } : agg)) };
    }));
  };

  const handleRemoveAggregate = (groupId: string, aggregateId: string) => {
    setGroupDefinitions((prev) => prev.map((g) => (g.id === groupId ? { ...g, aggregates: g.aggregates.filter((a: any) => a.id !== aggregateId) } : g)));
  };

  const handleAddCalculatedField = () => {
    setCalculatedFields((prev) => ([...prev, { id: `calc_${Date.now()}`, name: 'NewField', expression: '=Fields!Value * 0.1', datasetId: datasets[0]?.id ?? 'ds1' }]));
  };

  const handleCalculatedFieldChange = (fieldId: string, key: string, value: any) => {
    setCalculatedFields((prev) => prev.map((f) => (f.id === fieldId ? { ...f, [key]: value } : f)));
  };

  const handleExpressionChange = (index: number, value: string) => setExpressionLibrary((prev) => prev.map((e, i) => (i === index ? value : e)));
  const handleAddExpression = () => setExpressionLibrary((prev) => ([...prev, '=Fields!Amount * 1.0']));

  const handleEventScriptChange = (key: keyof typeof eventScripts, value: string) => {
    const sanitizedValue = sanitizeInput(value);
    if (sanitizedValue.includes('alert') || sanitizedValue.includes('eval')) {
      setSnackbar({ open: true, message: 'Invalid script content detected', severity: 'error' });
      return;
    }
    setEventScripts((prev: any) => ({ ...prev, [key]: sanitizedValue }));
  };

  const handleExportOptionToggle = (key: keyof typeof exportOptions, checked: boolean) => setExportOptions((prev: any) => ({ ...prev, [key]: checked }));

  const handleExport = (format: keyof ExportOptions) => {
    const humanLabel = exportFormatLabels[format as keyof ExportOptions];
    setSnackbar({ open: true, message: `Queued ${humanLabel} export with ${groupDefinitions.length} grouping level(s).`, severity: 'success' });
    if (format === 'includePrintFriendly') generatePixelPerfectPDF(elements, layoutSettingsState);
  };

  const handleCloseSnackbar = () => setSnackbar((s) => ({ ...s, open: false }));

  const handleElementDrop = useCallback((item: any, section: any, position: { x: number; y: number }) => {
    const newElement = {
      id: `element_${Date.now()}`,
      type: item.type,
      section,
      position,
      size: { width: 200, height: 100 },
      properties: {
        name: `${item.type}_${Date.now()}`,
        ...(item.type === ELEMENT_TYPES.TEXTBOX ? { text: 'Sample Text', fontSize: 12 } : {}),
        ...(item.type === ELEMENT_TYPES.TABLE ? { columns: ['Column 1', 'Column 2'], previewRows: 3 } : {}),
      },
    };

    setElements([...elements, newElement]);
    // keep a lightweight server call to illustrate intent
    createTemplateMutation.mutate({ name: newElement.properties.name, query: '', description: '', schedule: '' });
  }, [createTemplateMutation, setElements, elements]);

  const handleDragEnd = useCallback((event: any) => {
    const { active, over } = event;
    setActiveDragItem(null);
    if (over && active.data.current) {
      const item = active.data.current;
      const section = over.id;
      // For simplicity, place at a default position
      const position = { x: 10, y: 10 };
      handleElementDrop(item, section, position);
    }
  }, [handleElementDrop]);

  const handleDragStart = useCallback((event: any) => {
    setActiveDragItem(event.active.data.current);
  }, []);

  const handleElementUpdate = useCallback((id: string, updates: Partial<any>) => {
    const updated = elements.map((el: any) => el.id === id ? { ...el, ...updates } : el);
    setElements(updated);
  }, [setElements, elements]);
  const handleElementDelete = useCallback((id: string) => {
    const updated = elements.filter((el: any) => el.id !== id);
    setElements(updated);
  }, [setElements, elements]);

  const selectedElementData = useMemo(() => elements.find((el: any) => el.id === selectedElement), [elements, selectedElement]);

  const toolboxItems = [
    { type: ELEMENT_TYPES.TEXTBOX, icon: <Type size={16} />, label: 'Text Box' },
    { type: ELEMENT_TYPES.TABLE, icon: <TableIcon size={16} />, label: 'Table' },
    { type: ELEMENT_TYPES.MATRIX, icon: <Grid3X3 size={16} />, label: 'Matrix' },
    { type: ELEMENT_TYPES.LIST, icon: <ListIcon size={16} />, label: 'List' },
    { type: ELEMENT_TYPES.CHART, icon: <BarChart3 size={16} />, label: 'Chart' },
    { type: ELEMENT_TYPES.IMAGE, icon: <Image size={16} />, label: 'Image' },
    { type: ELEMENT_TYPES.SUBREPORT, icon: <FileText size={16} />, label: 'Subreport' },
    { type: ELEMENT_TYPES.RECTANGLE, icon: <Square size={16} />, label: 'Rectangle' },
    { type: ELEMENT_TYPES.LINE, icon: <Minus size={16} />, label: 'Line' },
    { type: ELEMENT_TYPES.GAUGE, icon: <Gauge size={16} />, label: 'Gauge / KPI' },
    { type: ELEMENT_TYPES.SPARKLINE, icon: <Activity size={16} />, label: 'Sparkline' }
  ];

  if (templatesLoading || dataSourcesLoading) return <LinearProgress />;

  return (
    <DndContext onDragEnd={handleDragEnd} onDragStart={handleDragStart}>
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <TopAppBar>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2 }}>
            <Tooltip title="Save Report">
              <IconButton color="inherit" onClick={() => { /* noop for now */ }}>
                <Save />
              </IconButton>
            </Tooltip>
            <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.2)' }} />
            <Tooltip title="Undo">
              <span>
                <IconButton color="inherit" onClick={undo} disabled={!canUndo}>
                  <Undo2 />
                </IconButton>
              </span>
            </Tooltip>
            <Tooltip title="Redo">
              <span>
                <IconButton color="inherit" onClick={redo} disabled={!canRedo}>
                  <Redo2 />
                </IconButton>
              </span>
            </Tooltip>
            <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.2)' }} />
            <Tooltip title="Preview">
              <IconButton color="inherit" onClick={() => setActiveTab('preview')}>
                <Eye />
              </IconButton>
            </Tooltip>
            <Tooltip title="Print">
              <IconButton color="inherit" onClick={() => { /* noop for now */ }}>
                <Printer />
              </IconButton>
            </Tooltip>
            <Tooltip title="Export to PDF">
              <IconButton color="inherit" onClick={() => generatePixelPerfectPDF(elements, layoutSettingsState)}>
                <Download />
              </IconButton>
            </Tooltip>
            <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.2)' }} />
            <Tooltip title="Data Sources">
              <IconButton color="inherit" onClick={() => setDataSourcesOpen(true)}>
                <Database />
              </IconButton>
            </Tooltip>
            <Tooltip title="Parameters">
              <IconButton color="inherit" onClick={() => setParametersOpen(true)}>
                <Settings />
              </IconButton>
            </Tooltip>
            <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.2)' }} />
            <Tooltip title="Layout & Page Settings">
              <IconButton color="inherit" onClick={() => setLayoutDrawerOpen(true)}>
                <LayoutDashboard />
              </IconButton>
            </Tooltip>
          </Box>
        </TopAppBar>

        <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>
          <Drawer variant="persistent" open={drawerOpen} sx={{ width: 240, flexShrink: 0, '& .MuiDrawer-paper': { width: 240, boxSizing: 'border-box', position: 'relative' } }}>
            <Box sx={{ p: 2, overflowY: 'auto' }}>
              <Typography variant="h6" gutterBottom>Report Items</Typography>
              {toolboxItems.map(item => <ToolboxItem key={item.type} type={item.type} icon={item.icon} label={item.label} />)}
            </Box>
          </Drawer>
          <Box component="main" sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <Box sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: '#f5f5f5' }}>
            <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
              <Tab label="Design" value="design" />
              <Tab label="Preview" value="preview" />
              <Tab label="Data" value="data" />
            </Tabs>
          </Box>

          {activeTab === 'design' && (
            <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>
              <Box sx={{ width: 220, borderRight: 1, borderColor: 'divider', flexShrink: 0 }}>
                <Tabs orientation="vertical" value={designTab} onChange={(_, v) => setDesignTab(v)} sx={{ '& .MuiTab-root': { alignItems: 'flex-start' } }}>
                  <Tab label="Canvas" />                  
                  <Tab label="Grouping" />
                  <Tab label="Data Logic" />
                  <Tab label="Export & Events" />
                </Tabs>
              </Box>
              <Box sx={{ flex: 1, p: 2, overflowY: 'auto' }}>
                {designTab === 0 && (
                  <ReportCanvas elements={elements} layoutSettings={layoutSettingsState} selectedElement={selectedElement} onElementUpdate={handleElementUpdate} onElementDelete={handleElementDelete} onElementSelect={setSelectedElement} orientation={orientation} />
                )}
                {designTab === 1 && (
                  <Paper sx={{ p: 2 }}>
                    <GroupsEditor
                      groupDefinitions={groupDefinitions}
                      onAddGroup={handleAddGroup}
                      onRemoveGroup={(groupId) => setGroupDefinitions((prev) => prev.filter((c) => c.id !== groupId))}
                      onGroupChange={handleGroupChange}
                      onAddAggregate={handleAddAggregate}
                      onAggregateChange={handleAggregateChange}
                      onRemoveAggregate={handleRemoveAggregate}
                    />
                  </Paper>
                )}
                {designTab === 2 && (
                  <Paper sx={{ p: 2 }}>
                    <CalculatedFieldsEditor
                      calculatedFields={calculatedFields}
                      datasets={datasets as unknown as any[]}
                      onAddCalculatedField={handleAddCalculatedField}
                      onCalculatedFieldChange={handleCalculatedFieldChange}
                      onRemoveCalculatedField={(fieldId) => setCalculatedFields((prev) => prev.filter((c) => c.id !== fieldId))}
                    />
                    <Divider sx={{ my: 2 }} />
                    <ExpressionsEditor expressionLibrary={expressionLibrary} onExpressionChange={handleExpressionChange} onAddExpression={handleAddExpression} />
                  </Paper>
                )}
                {designTab === 3 && (
                  <Paper sx={{ p: 2 }}>
                    <EventScriptsEditor eventScripts={eventScripts} onEventScriptChange={handleEventScriptChange} />
                    <Divider sx={{ my: 2 }}>Export Options</Divider>
                    <Grid container spacing={1.5}>
                      {(Object.keys(exportOptions) as Array<keyof ExportOptions>).map((key) => (
                        <Grid item xs={12} sm={6} md={4} key={`export_option_${String(key)}`}>
                          <Card variant="outlined" sx={{ p: 1.5, height: '100%' }}>
                            <FormControlLabel control={<Switch size="small" checked={exportOptions[key]} onChange={(e) => handleExportOptionToggle(key, e.target.checked)} />} label={exportFormatLabels[key as keyof ExportOptions]} />
                            <Typography variant="caption" color="text.secondary">{exportOptionDescriptions[key as keyof ExportOptions]}</Typography>
                          </Card>
                        </Grid>
                      ))}
                    </Grid>
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 2 }}>
                      {(Object.keys(exportOptions) as Array<keyof ExportOptions>).map((key) => (
                        <Button key={`export_button_${String(key)}`} variant="contained" size="small" disabled={!exportOptions[key]} onClick={() => handleExport(key)}>{exportFormatLabels[key as keyof ExportOptions]}</Button>
                      ))}
                    </Box>
                  </Paper>
                )}
              </Box>
              <Paper sx={{ width: 300, flexShrink: 0, overflowY: 'auto' }}>
                <PropertiesPanel selectedElement={selectedElementData ?? null} onElementUpdate={handleElementUpdate} groupDefinitions={[]} />
              </Paper>
            </Box>
          )}

          {activeTab === 'preview' && (
            <Box sx={{ p: 2 }}>
              <Paper sx={{ p: 2, mb: 2 }}>
                <Grid container spacing={2} alignItems="center" justifyContent="space-between">
                  <Grid item>
                    <Typography variant="h6">Report Preview</Typography>
                  </Grid>
                  <Grid item sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                    <Button
                      variant="contained"
                      size="small"
                      startIcon={previewLoading ? undefined : <Eye />}
                      onClick={async () => {
                        setPreviewLoading(true);
                        setPreviewError(null);
                        try {
                          // Try to fetch from API first
                          const response = await fetch('/api/semantic/query', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default-tenant' },
                            body: JSON.stringify({
                              measures: ['orders.count', 'orders.total_amount'],
                              dimensions: ['orders.status', 'orders.ship_country'],
                              limit: 20,
                            }),
                          });
                          if (response.ok) {
                            const result = await response.json();
                            setPreviewData(result.data || []);
                            setPreviewSQL(result.annotation?.generatedSQL || 'SELECT * FROM orders LIMIT 20');
                          } else {
                            throw new Error('API not available');
                          }
                        } catch (err) {
                          // Generate sample data for demo
                          const sampleData = Array.from({ length: 10 }, (_, i) => ({
                            status: ['Completed', 'Pending', 'Shipped'][i % 3],
                            ship_country: ['USA', 'UK', 'Germany', 'France', 'Canada'][i % 5],
                            count: Math.floor(Math.random() * 100) + 10,
                            total_amount: (Math.random() * 10000 + 1000).toFixed(2),
                          }));
                          setPreviewData(sampleData);
                          setPreviewSQL('SELECT status, ship_country, COUNT(*) as count, SUM(amount) as total_amount\nFROM orders\nGROUP BY status, ship_country\nLIMIT 20;');
                        } finally {
                          setPreviewLoading(false);
                        }
                      }}
                      disabled={previewLoading}
                    >
                      {previewLoading ? 'Loading...' : 'Run Preview'}
                    </Button>
                    <Pagination count={10} page={currentPage} onChange={(_, page) => setCurrentPage(page)} size="small" />
                  </Grid>
                </Grid>
              </Paper>
              
              {/* SQL Preview */}
              {previewSQL && (
                <Paper sx={{ p: 2, mb: 2, bgcolor: '#1e293b' }}>
                  <Typography variant="caption" sx={{ color: '#94a3b8', display: 'block', mb: 1 }}>Generated SQL</Typography>
                  <Typography component="pre" sx={{ fontFamily: 'monospace', fontSize: 12, color: '#e2e8f0', whiteSpace: 'pre-wrap', m: 0 }}>
                    {previewSQL}
                  </Typography>
                </Paper>
              )}
              
              {/* Data Results Table */}
              {previewData && previewData.length > 0 && (
                <Paper sx={{ mb: 2 }}>
                  <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
                    <Typography variant="subtitle2">{previewData.length} rows returned</Typography>
                  </Box>
                  <Box sx={{ maxHeight: 300, overflow: 'auto' }}>
                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
                      <thead>
                        <tr style={{ backgroundColor: '#f8fafc' }}>
                          {Object.keys(previewData[0]).map(key => (
                            <th key={key} style={{ padding: '8px 12px', textAlign: 'left', borderBottom: '1px solid #e2e8f0', fontWeight: 600 }}>
                              {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {previewData.map((row, idx) => (
                          <tr key={idx} style={{ borderBottom: '1px solid #e2e8f0' }}>
                            {Object.values(row).map((value: any, cellIdx) => (
                              <td key={cellIdx} style={{ padding: '8px 12px' }}>
                                {typeof value === 'number' ? value.toLocaleString() : String(value ?? '-')}
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </Box>
                </Paper>
              )}
              
              {previewError && (
                <Paper sx={{ p: 2, mb: 2, bgcolor: '#fef2f2', border: '1px solid #fecaca' }}>
                  <Typography color="error">{previewError}</Typography>
                </Paper>
              )}
              
              {/* Report Layout Preview */}
              <Paper sx={{ width: orientation === 'Portrait' ? 794 : 1123, minHeight: 600, mx: 'auto', p: 3, bgcolor: '#ffffff', boxShadow: 3 }}>
                <ReportCanvas elements={elements} layoutSettings={layoutSettings} selectedElement={null} onElementUpdate={() => { }} onElementDelete={() => { }} onElementSelect={() => { }} orientation={orientation} />
              </Paper>
            </Box>
          )}

          {activeTab === 'data' && ( // Adjusted for new flex layout
            <Box sx={{ p: 2, overflowY: 'auto' }}>
              <Typography variant="h6" gutterBottom>Data Sources & Datasets</Typography>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Paper sx={{ p: 2 }}>
                    <Typography variant="h6" gutterBottom>Data Sources</Typography>
                    {(fetchedDataSources || staticDataSources).map((ds: any) => (<Box key={ds.id} sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}><Database size={20} />{ds.name}</Box>))}
                    <Button variant="outlined" startIcon={<Plus />} onClick={() => setDataSourcesOpen(true)}>Add Data Source</Button>
                  </Paper>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Paper sx={{ p: 2 }}>
                    <Typography variant="h6" gutterBottom>Datasets</Typography>
                    {datasets.map(ds => (<Box key={ds.id} sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}><TableIcon size={20} />{ds.name}</Box>))}
                    <Button variant="outlined" startIcon={<Plus />}>Add Dataset</Button>
                  </Paper>
                </Grid>
              </Grid>
            </Box>
          )}
          </Box>
        </Box>

        <Drawer anchor="right" open={layoutDrawerOpen} onClose={() => setLayoutDrawerOpen(false)}>
          <Box sx={{ width: 350, p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="h6">Layout & Page Settings</Typography>
            <PageSettings pageSize={pageSize} orientation={orientation} onChangePageSize={(v) => setPageSize(v)} onChangeOrientation={(v) => setOrientation(v)} />
            <Paper sx={{ p: 2, mt: 2 }}>
              <Typography variant="subtitle1" gutterBottom>Layout & Pagination</Typography>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakBeforeGroup} onChange={(e) => handleLayoutSettingChange('pageBreakBeforeGroup', e.target.checked)} />} label="Page break before group" /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakAfterGroup} onChange={(e) => handleLayoutSettingChange('pageBreakAfterGroup', e.target.checked)} />} label="Page break after group" /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakBetweenRegions} onChange={(e) => handleLayoutSettingChange('pageBreakBetweenRegions', e.target.checked)} />} label="Page break between regions" /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.fixedPageSize} onChange={(e) => handleLayoutSettingChange('fixedPageSize', e.target.checked)} />} label="Fixed page size" /></Grid>
                <Grid item xs={6} sm={4}><TextField fullWidth size="small" type="number" label="Columns" value={layoutSettingsState.columns} onChange={(e) => handleLayoutSettingChange('columns', Math.max(1, Number(e.target.value) || 1))} /></Grid>
                <Grid item xs={6} sm={8}><TextField fullWidth size="small" type="number" label="Column Spacing" value={layoutSettingsState.columnSpacing} onChange={(e) => handleLayoutSettingChange('columnSpacing', Math.max(0, Number(e.target.value) || 0))} InputProps={{ endAdornment: <InputAdornment position="end">px</InputAdornment> }} /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.includeExecutionTime} onChange={(e) => handleLayoutSettingChange('includeExecutionTime', e.target.checked)} />} label="Include execution time" /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.includeUserName} onChange={(e) => handleLayoutSettingChange('includeUserName', e.target.checked)} />} label="Include user name" /></Grid>
              </Grid>
              <Divider sx={{ my: 2 }}>Header Tokens</Divider>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 1 }}>
                {layoutSettingsState.headerTokens.map((token: string) => (<Chip key={`header_${token}`} size="small" label={token} onDelete={() => handleRemoveToken('headerTokens', token)} color="primary" variant="outlined" />))}
              </Box>
              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                <TextField size="small" label="Add header token" value={headerTokenInput} onChange={(e) => setHeaderTokenInput(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter' && headerTokenInput.trim()) { handleAddToken('headerTokens', headerTokenInput.trim()); setHeaderTokenInput(''); } }} />
                <Button variant="contained" size="small" onClick={() => { if (headerTokenInput.trim()) { handleAddToken('headerTokens', headerTokenInput.trim()); setHeaderTokenInput(''); } }}>Add</Button>
              </Box>
              <Divider sx={{ my: 2 }}>Footer Tokens</Divider>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 1 }}>
                {layoutSettingsState.footerTokens.map((token: string) => (<Chip key={`footer_${token}`} size="small" label={token} onDelete={() => handleRemoveToken('footerTokens', token)} color="secondary" variant="outlined" />))}
              </Box>
              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                <TextField size="small" label="Add footer token" value={footerTokenInput} onChange={(e) => setFooterTokenInput(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter' && footerTokenInput.trim()) { handleAddToken('footerTokens', footerTokenInput.trim()); setFooterTokenInput(''); } }} />
                <Button variant="contained" size="small" onClick={() => { if (footerTokenInput.trim()) { handleAddToken('footerTokens', footerTokenInput.trim()); setFooterTokenInput(''); } }}>Add</Button>
              </Box>
            </Paper>
          </Box>
        </Drawer>

        <DataSourcesDialog open={dataSourcesOpen} onClose={() => setDataSourcesOpen(false)} />
        <ParametersDialog
          open={parametersOpen}
          onClose={() => setParametersOpen(false)}
          parameters={reportParameters}
          onAdd={handleAddParameter}
          onUpdate={handleUpdateParameter}
          onDelete={handleRemoveParameter} />
        <Snackbar open={snackbar.open} autoHideDuration={4000} onClose={handleCloseSnackbar} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}>
          <Alert onClose={handleCloseSnackbar} severity={snackbar.severity} sx={{ width: '100%' }}>{snackbar.message}</Alert>
        </Snackbar>
        <DragOverlay>
          {activeDragItem ? (
            <Box sx={{ p: 2, bgcolor: 'background.paper', border: '1px solid', borderColor: 'primary.main', borderRadius: 1 }}>
              <Typography variant="body2">{activeDragItem.type}</Typography>
            </Box>
          ) : null}
        </DragOverlay>
      </Box>
    </DndContext>
  );
};

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

const SSRSReportBuilder: React.FC = () => (
  <QueryClientProvider client={queryClient}>
    <SSRSReportBuilderContent />
  </QueryClientProvider>
);

export default SSRSReportBuilder;