import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { DndContext, DragOverlay, useDraggable as _useDraggable, useDroppable as _useDroppable } from '@dnd-kit/core';
import {
  Box,
  Drawer,
  Typography,
  Tabs,
  Tab,
  Paper,
  Grid,
  Snackbar,
  Alert,
  Divider,
  Chip,
  InputAdornment,
  FormControlLabel,
  Switch,
  Card,
  TextField,
  Button,
  Tooltip,
  IconButton,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Stack
} from '@mui/material';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import useUndo from 'use-undo';
import { getCachedGoldCopyId } from '../../utils/goldCopy';
import { apiClient } from '../../utils/apiClient';

// Modular components & utils
import ToolboxItem from './ToolboxItem';
import ReportCanvas from './ReportCanvas';
import PropertiesPanel from './PropertiesPanel';
import DataSourcesDialog from './DataSourcesDialog';
import ParametersDialog from './ParametersDialog';
import TopAppBar from './TopAppBar';
import PageSettings from './PageSettings';
import {
  ELEMENT_TYPES,
  REPORT_SECTIONS,
  datasets,
  generatePixelPerfectPDF,
  sanitizeInput,
  exportFormatLabels,
  exportOptionDescriptions,
  EventScripts,
  ExportOptions
} from './reportingUtils';
import GroupsEditor from './GroupsEditor';
import CalculatedFieldsEditor, { CalculatedFieldItem } from './CalculatedFieldsEditor';
import ExpressionsEditor from './ExpressionsEditor';
import EventScriptsEditor from './EventScriptsEditor';
import StorageIcon from '@mui/icons-material/Storage';
import TableChartIcon from '@mui/icons-material/TableChart';
import BarChartIcon from '@mui/icons-material/BarChart';
import TextFieldsIcon from '@mui/icons-material/TextFields';
import ImageIcon from '@mui/icons-material/Image';
import DescriptionIcon from '@mui/icons-material/Description';
import SquareIcon from '@mui/icons-material/Square';
import RemoveIcon from '@mui/icons-material/Remove';
import SpeedIcon from '@mui/icons-material/Speed';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import GridViewIcon from '@mui/icons-material/GridView';
import ListAltIcon from '@mui/icons-material/ListAlt';
import SaveIcon from '@mui/icons-material/Save';
import DownloadIcon from '@mui/icons-material/Download';
import PrintIcon from '@mui/icons-material/Print';
import VisibilityIcon from '@mui/icons-material/Visibility';
import UndoIcon from '@mui/icons-material/Undo';
import RedoIcon from '@mui/icons-material/Redo';
import SettingsIcon from '@mui/icons-material/Settings';
import DashboardIcon from '@mui/icons-material/Dashboard';
import LayersIcon from '@mui/icons-material/Layers';
import EditIcon from '@mui/icons-material/Edit';
import { useTheme } from '@mui/material';

import { useTenant } from '../../contexts/TenantContext';
import BOFieldsPalette, { BOField } from './BOFieldsPalette';
import ReportScheduleBurstingTab from './ReportScheduleBurstingTab';
import { FieldDefinition } from '../ExpressionBuilder/AdvancedConditionBuilder';
import { dedupeFields } from '../../utils/dedupeFields';

type ReportParameter = {
  id: string;
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean';
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
};

const SSRSReportBuilderContent: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const colors = {
    bg: isDark ? '#0A0C12' : '#F8FAFC',
    cardBg: isDark ? '#13161E' : '#FFFFFF',
    sidebarBg: isDark ? '#0F1117' : '#FFFFFF',
    border: isDark ? 'rgba(255, 255, 255, 0.08)' : 'rgba(0, 0, 0, 0.08)',
    text: isDark ? '#E2E8F0' : '#1E293B',
    textMuted: isDark ? '#94A3B8' : '#64748B',
    primary: '#6366F1',
  };

  const [elementsState, { set: setElements, undo, redo, canUndo, canRedo }] = useUndo<any[]>([]);
  const elements = elementsState.present;

  const [selectedElement, setSelectedElement] = useState<string | null>(null);
  const [activeDragItem, setActiveDragItem] = useState<any>(null);
  const [activeTab, setActiveTab] = useState('design');
  const [sidebarTab, setSidebarTab] = useState<'fields' | 'toolbox'>('fields');
  const [drawerOpen] = useState(true);
  const [dataSourcesOpen, setDataSourcesOpen] = useState(false);
  const [parametersOpen, setParametersOpen] = useState(false);
  const [layoutDrawerOpen, setLayoutDrawerOpen] = useState(false);
  const [pageSize, setPageSize] = useState('A4');
  const [orientation, setOrientation] = useState<'Portrait' | 'Landscape'>('Portrait');
  
  const [layoutSettingsState, setLayoutSettingsState] = useState<any>({
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

  // Section configuration state (hidden, visibilityCondition, backgroundColor)
  const [sectionConfig, setSectionConfig] = useState<Record<string, any>>({});

  const handleSectionConfigChange = (section: string, update: Partial<any>) => {
    setSectionConfig((prev) => ({
      ...prev,
      [section]: { ...(prev[section] || {}), ...update },
    }));
  };

  // Groups, calculated fields, expressions, event scripts, export options
  const [groupDefinitions, setGroupDefinitions] = useState<any[]>([]);

  const [calculatedFields, setCalculatedFields] = useState<CalculatedFieldItem[]>([
    { id: 'calc_margin', name: 'GrossMargin', expression: '=Fields!Revenue - Fields!Cost', datasetId: datasets[0]?.id ?? 'ds1', format: 'Currency' },
  ]);

  const [expressionLibrary, setExpressionLibrary] = useState<string[]>([
    '=IIF(Fields!Growth.Value < 0, "#DC2626", "#16A34A")',
    '=Sum(Fields!Sales.Value, "SalesGroup")',
  ]);

  const [eventScripts, setEventScripts] = useState<EventScripts>({
    onRowRender: '// format negative growth rows\nif (row.Fields.Growth < 0) { row.Style.Background = "#FEF2F2"; }',
    onCellRender: '// add tooltip\ncell.Tooltip = "{Field}: {Value}";',
    onPageRender: '// watermark\npage.Watermark = "Internal";',
    onExport: '// append metadata\nexportContext.Metadata.author = user.name;',
  });

  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    includePrintFriendly: true,
    includeDrillThrough: true,
    includeComments: false,
  });

  const [reportParameters, setReportParameters] = useState<ReportParameter[]>([
    { id: 'param_year', name: 'Year', type: 'number', prompt: 'Enter a Year', defaultValue: String(new Date().getFullYear()) },
  ]);

  // Report title (editable in top bar)
  const [reportTitle, setReportTitle] = useState('Untitled Report');
  const [editingTitle, setEditingTitle] = useState(false);

  const handleAddParameter = (param: Omit<ReportParameter, 'id'>) => setReportParameters(prev => [...prev, { ...param, id: `param_${Date.now()}` }]);
  const handleUpdateParameter = (updatedParam: ReportParameter) => setReportParameters(prev => prev.map(p => p.id === updatedParam.id ? updatedParam : p));
  const handleRemoveParameter = (paramId: string) => setReportParameters(prev => prev.filter(p => p.id !== paramId));

  const [headerTokenInput, setHeaderTokenInput] = useState('');
  const [footerTokenInput, setFooterTokenInput] = useState('');

  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'info' | 'warning' | 'error' });
  const handleCloseSnackbar = () => setSnackbar(prev => ({ ...prev, open: false }));

  // Business Object states
  const [businessObjects, setBusinessObjects] = useState<any[]>([]);
  const [selectedBOId, setSelectedBOId] = useState<string>('');
  const [selectedBO, setSelectedBO] = useState<any | null>(null);
  const [bindings, setBindings] = useState<any[]>([]);
  const [selectedBindingId, setSelectedBindingId] = useState<string>('');
  const [relatedBOs, setRelatedBOs] = useState<any[]>([]);
  const [activeDatasets, setActiveDatasets] = useState<any[]>([...datasets]);

  // Preview state
  const [previewData, setPreviewData] = useState<any[] | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewSQL, setPreviewSQL] = useState<string | null>(null);
  const [showQueryDetails, setShowQueryDetails] = useState(false);

  // Helper to build headers with authentication
  const getAuthHeaders = useCallback((additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';
    
    const tenantId = tenant?.id ?? getCachedGoldCopyId() ?? '';
    const datasourceId = datasource?.id || 'b7879e02-7e4c-44c9-bade-2b10aab2d3c0';

    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      'X-Tenant-Region': 'us-east-1',
      ...additionalHeaders,
    };
  }, [tenant, datasource]);

  // Fetch Business Objects list on mount
  useEffect(() => {
    const loadBOs = async () => {
      try {
        const data = await apiClient<any>('/api/business-objects?format=array', {
          headers: getAuthHeaders(),
        });
        const list = Array.isArray(data) ? data : (typeof data === 'object' && data !== null ? Object.values(data) : []);
        setBusinessObjects(list);
        if (list.length > 0 && !selectedBOId) {
          setSelectedBOId((list[0] as any).id);
        }
      } catch (err) {
        console.error('Failed to load business objects:', err);
      }
    };
    loadBOs();
  }, [getAuthHeaders, selectedBOId]);

  // Fetch detailed BO metadata when selectedBOId changes
  useEffect(() => {
    if (!selectedBOId) {
      setSelectedBO(null);
      setBindings([]);
      setRelatedBOs([]);
      return;
    }
    const loadBODetails = async () => {
      try {
        const res = await fetch(`/api/business-objects/${selectedBOId}`, {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const bo = await res.json();
          setSelectedBO(bo);

          // Build synthetic dataset from BO fields
          const boFields = [
            ...(bo.coreFields || []),
            ...(bo.customFields || []),
          ].map((f: any) => ({
            name: f.name || f.technicalName,
            type: f.dataType || 'string',
          }));

          const boDataset = {
            id: `ds_${bo.id}`,
            name: bo.displayName || bo.name,
            fields: boFields,
          };
          setActiveDatasets([boDataset, ...datasets]);
        }

        // Fetch bindings
        const bRes = await fetch(`/api/business-objects/${selectedBOId}/bindings`, {
          headers: getAuthHeaders(),
        });
        if (bRes.ok) {
          const bData = await bRes.json();
          const bList = Array.isArray(bData) ? bData : bData?.data || [];
          setBindings(bList);
          if (bList.length > 0) setSelectedBindingId(bList[0].id);
        }

        // Fetch relationships
        const rRes = await fetch(`/api/business-objects/${selectedBOId}/relationships`, {
          headers: getAuthHeaders(),
        });
        if (rRes.ok) {
          const rData = await rRes.json();
          setRelatedBOs(Array.isArray(rData) ? rData : rData?.data || []);
        }
      } catch (err) {
        console.error('Failed to load BO details:', err);
      }
    };
    loadBODetails();
  }, [selectedBOId, getAuthHeaders]);

  // Run preview query
  const runPreviewQuery = useCallback(async () => {
    setPreviewLoading(true);
    try {
      if (selectedBO) {
        const fields = [
          ...(selectedBO.coreFields || []),
          ...(selectedBO.customFields || []),
        ].map((f: any) => f.name || f.technicalName);

        const response = await fetch('/api/semantic/query', {
          method: 'POST',
          headers: getAuthHeaders(),
          body: JSON.stringify({
            businessObject: selectedBO.name,
            fields: fields.slice(0, 8),
            limit: 25,
          }),
        });

        if (response.ok) {
          const result = await response.json();
          setPreviewData(result.data || []);
          setPreviewSQL(result.annotation?.generatedSQL || `SELECT ${fields.slice(0, 8).join(', ')} FROM ${selectedBO.name} LIMIT 25;`);
          setSnackbar({ open: true, message: `Loaded ${result.data?.length || 0} preview records for ${selectedBO.displayName || selectedBO.name}`, severity: 'success' });
          return;
        }
      }

      // Synthetic sample data fallback
      const sampleFields = selectedBO
        ? dedupeFields([...(selectedBO.coreFields || []), ...(selectedBO.customFields || [])]).map((f: any) => f.name || f.technicalName)
        : ['id', 'name', 'status', 'amount', 'created_at', 'region'];

      const sampleData = Array.from({ length: 15 }, (_, i) => {
        const row: Record<string, any> = {};
        sampleFields.forEach((field: string) => {
          if (field.toLowerCase().includes('id')) row[field] = `ID-${1000 + i}`;
          else if (field.toLowerCase().includes('name')) row[field] = `Sample ${selectedBO?.name || 'Entity'} ${i + 1}`;
          else if (field.toLowerCase().includes('status')) row[field] = ['Active', 'Pending', 'Closed', 'Draft'][i % 4];
          else if (field.toLowerCase().includes('amount') || field.toLowerCase().includes('price') || field.toLowerCase().includes('value') || field.toLowerCase().includes('cost')) row[field] = parseFloat((Math.random() * 8500 + 500).toFixed(2));
          else if (field.toLowerCase().includes('date') || field.toLowerCase().includes('at')) row[field] = new Date(Date.now() - i * 86400000 * 3).toISOString().split('T')[0];
          else row[field] = `Val_${i + 1}`;
        });
        return row;
      });

      setPreviewData(sampleData);
      setPreviewSQL(`-- Simulated Query for ${selectedBO?.displayName || selectedBO?.name || 'Entity'}\nSELECT ${sampleFields.join(', ')}\nFROM ${selectedBO?.name || 'entity_table'}\nLIMIT 15;`);
      setSnackbar({ open: true, message: `Showing 15 sample records for ${selectedBO?.displayName || selectedBO?.name || 'report'}`, severity: 'info' });
    } catch (err: any) {
      console.error('Preview error:', err);
      setSnackbar({ open: true, message: 'Using sample preview data', severity: 'warning' });
    } finally {
      setPreviewLoading(false);
    }
  }, [selectedBO, getAuthHeaders]);

  // Auto-run preview when switching to preview tab
  useEffect(() => {
    if (activeTab === 'preview' && !previewData) {
      runPreviewQuery();
    }
  }, [activeTab, previewData, runPreviewQuery]);

  // Add individual BO field to Canvas
  const handleAddFieldToCanvas = (field: BOField) => {
    const newElement = {
      id: `field_${field.name}_${Date.now()}`,
      type: ELEMENT_TYPES.TEXTBOX,
      section: REPORT_SECTIONS.BODY,
      position: { x: 30 + (elements.length % 5) * 20, y: 30 + (elements.length % 5) * 30 },
      size: { width: 220, height: 44 },
      properties: {
        text: `[${field.label || field.name}]`,
        valueExpression: `[${selectedBO?.name || 'Entity'}.${field.name}]`,
        fieldName: field.name,
        name: field.label || field.name,
        fontSize: 12,
        fontWeight: 500,
        textColor: isDark ? '#E2E8F0' : '#1E293B',
      },
    };
    setElements([...elements, newElement]);
    setSelectedElement(newElement.id);
    setSnackbar({ open: true, message: `Added "${field.label || field.name}" to canvas`, severity: 'success' });
  };

  // Add all BO fields as a Table
  const handleAddAllAsTable = (fields: BOField[]) => {
    const tableColumns = fields.map(f => f.name);
    const newTable = {
      id: `table_bo_${Date.now()}`,
      type: ELEMENT_TYPES.TABLE,
      section: REPORT_SECTIONS.BODY,
      position: { x: 30, y: 30 },
      size: { width: Math.min(700, Math.max(400, tableColumns.length * 110)), height: 220 },
      properties: {
        name: `${selectedBO?.displayName || selectedBO?.name || 'BO'} Data Table`,
        columns: tableColumns,
        fontSize: 11,
        showGridLines: true,
        alternatingRowColors: true,
      },
    };
    setElements([...elements, newTable]);
    setSelectedElement(newTable.id);
    setSnackbar({ open: true, message: `Created Table with ${fields.length} columns from ${selectedBO?.displayName || 'BO'}`, severity: 'success' });
  };

  const handleDragStart = (event: any) => {
    setActiveDragItem(event.active.data.current);
  };

  const handleDragEnd = (event: any) => {
    const { active, over } = event;
    setActiveDragItem(null);
    if (!over) return;

    const rawTargetSection = over.id as string;
    const targetSection = Object.values(REPORT_SECTIONS).includes(rawTargetSection as any)
      ? rawTargetSection
      : REPORT_SECTIONS.BODY;

    if (active.data.current?.isToolboxItem) {
      const type = active.data.current.type;
      const defaultSizes: Record<string, { width: number; height: number }> = {
        [ELEMENT_TYPES.TEXTBOX]: { width: 180, height: 40 },
        [ELEMENT_TYPES.TABLE]: { width: 520, height: 180 },
        [ELEMENT_TYPES.MATRIX]: { width: 440, height: 160 },
        [ELEMENT_TYPES.LIST]: { width: 320, height: 140 },
        [ELEMENT_TYPES.CHART]: { width: 340, height: 200 },
        [ELEMENT_TYPES.IMAGE]: { width: 140, height: 100 },
        [ELEMENT_TYPES.SUBREPORT]: { width: 280, height: 70 },
        [ELEMENT_TYPES.RECTANGLE]: { width: 180, height: 90 },
        [ELEMENT_TYPES.LINE]: { width: 240, height: 10 },
        [ELEMENT_TYPES.GAUGE]: { width: 160, height: 120 },
        [ELEMENT_TYPES.SPARKLINE]: { width: 180, height: 50 },
      };

      const newElement = {
        id: `${type}_${Date.now()}`,
        type,
        section: targetSection,
        position: { x: 30, y: 30 },
        size: defaultSizes[type] || { width: 160, height: 60 },
        properties: {
          name: `${type.charAt(0).toUpperCase() + type.slice(1)} 1`,
          fontSize: 12,
          textColor: isDark ? '#E2E8F0' : '#1E293B',
          columns: type === ELEMENT_TYPES.TABLE
            ? (selectedBO?.coreFields?.slice(0, 5).map((f: any) => f.name) || ['id', 'name', 'status', 'amount'])
            : undefined,
        },
      };

      setElements([...elements, newElement]);
      setSelectedElement(newElement.id);
      return;
    }

    if (active.data.current?.isBOField) {
      const field: BOField = active.data.current.field;
      handleAddFieldToCanvas(field);
    }
  };

  const handleElementUpdate = useCallback((id: string, updates: Partial<any>) => {
    const updated = elements.map((el: any) => (el.id === id ? { ...el, ...updates } : el));
    setElements(updated);
  }, [setElements, elements]);

  const handleElementDelete = useCallback((id: string) => {
    const updated = elements.filter((el: any) => el.id !== id);
    setElements(updated);
  }, [setElements, elements]);

  const handleElementDuplicate = useCallback((id: string) => {
    const source = elements.find((el: any) => el.id === id);
    if (!source) return;
    const copy = {
      ...source,
      id: `${source.type}_copy_${Date.now()}`,
      position: { x: source.position.x + 20, y: source.position.y + 20 },
    };
    setElements([...elements, copy]);
    setSnackbar({ open: true, message: `Duplicated element`, severity: 'success' });
  }, [elements, setElements]);

  const handleLayoutSettingChangeFromCanvas = useCallback((key: string, value: any) => {
    setLayoutSettingsState((prev: any) => ({ ...prev, [key]: value }));
  }, []);

  const selectedElementData = useMemo(() => elements.find((el: any) => el.id === selectedElement), [elements, selectedElement]);

  // Convert BO fields to FieldDefinition[] for Expression Builders
  const availableFieldDefs: FieldDefinition[] = useMemo(() => [
    ...(selectedBO?.coreFields || []),
    ...(selectedBO?.customFields || []),
  ].map((f: any) => ({
    name: f.name || f.technicalName || '',
    label: f.displayName || f.name || '',
    type: (['number', 'integer', 'float', 'decimal', 'currency'].includes((f.dataType || '').toLowerCase())
      ? 'number'
      : ['boolean', 'bool'].includes((f.dataType || '').toLowerCase())
      ? 'boolean'
      : ['date', 'timestamp', 'datetime'].includes((f.dataType || '').toLowerCase())
      ? 'date'
      : 'string') as any,
  })), [selectedBO]);

  const toolboxItems = [
    { type: ELEMENT_TYPES.TEXTBOX, icon: <TextFieldsIcon sx={{ fontSize: 16 }} />, label: 'Text Box' },
    { type: ELEMENT_TYPES.TABLE, icon: <TableChartIcon sx={{ fontSize: 16 }} />, label: 'Table' },
    { type: ELEMENT_TYPES.MATRIX, icon: <GridViewIcon sx={{ fontSize: 16 }} />, label: 'Matrix' },
    { type: ELEMENT_TYPES.LIST, icon: <ListAltIcon sx={{ fontSize: 16 }} />, label: 'List' },
    { type: ELEMENT_TYPES.CHART, icon: <BarChartIcon sx={{ fontSize: 16 }} />, label: 'Chart' },
    { type: ELEMENT_TYPES.IMAGE, icon: <ImageIcon sx={{ fontSize: 16 }} />, label: 'Image' },
    { type: ELEMENT_TYPES.SUBREPORT, icon: <DescriptionIcon sx={{ fontSize: 16 }} />, label: 'Subreport' },
    { type: ELEMENT_TYPES.RECTANGLE, icon: <SquareIcon sx={{ fontSize: 16 }} />, label: 'Rectangle' },
    { type: ELEMENT_TYPES.LINE, icon: <RemoveIcon sx={{ fontSize: 16 }} />, label: 'Line' },
    { type: ELEMENT_TYPES.GAUGE, icon: <SpeedIcon sx={{ fontSize: 16 }} />, label: 'Gauge / KPI' },
    { type: ELEMENT_TYPES.SPARKLINE, icon: <TrendingUpIcon sx={{ fontSize: 16 }} />, label: 'Sparkline' }
  ];

  // Calculated fields handlers
  const handleAddCalculatedField = (newField?: CalculatedFieldItem) => {
    const item = newField || {
      id: `calc_${Date.now()}`,
      name: `CalcField_${calculatedFields.length + 1}`,
      expression: '',
      datasetId: activeDatasets[0]?.id || 'ds1',
      format: 'Auto',
    };
    setCalculatedFields(prev => [...prev, item]);
  };

  const handleCalculatedFieldChange = (fieldId: string, key: string, value: any) => {
    setCalculatedFields(prev => prev.map(f => f.id === fieldId ? { ...f, [key]: value } : f));
  };

  // Group handlers
  const handleAddGroup = () => setGroupDefinitions(prev => [...prev, { id: `grp_${Date.now()}`, name: 'New Group', expression: '', aggregates: [], pageBreakAfter: false }]);
  const handleGroupChange = (groupId: string, key: string, value: any) => setGroupDefinitions(prev => prev.map(g => g.id === groupId ? { ...g, [key]: value } : g));
  const handleAddAggregate = (groupId: string) => setGroupDefinitions(prev => prev.map(g => g.id === groupId ? { ...g, aggregates: [...(g.aggregates || []), { id: `agg_${Date.now()}`, field: '', function: 'SUM', scope: 'Group', displayName: 'Total' }] } : g));
  const handleAggregateChange = (groupId: string, aggId: string, key: string, value: any) => setGroupDefinitions(prev => prev.map(g => g.id === groupId ? { ...g, aggregates: g.aggregates.map((a: any) => a.id === aggId ? { ...a, [key]: value } : a) } : g));
  const handleRemoveAggregate = (groupId: string, aggId: string) => setGroupDefinitions(prev => prev.map(g => g.id === groupId ? { ...g, aggregates: g.aggregates.filter((a: any) => a.id !== aggId) } : g));

  const handleExpressionChange = (index: number, value: string) => setExpressionLibrary(prev => { const next = [...prev]; next[index] = value; return next; });
  const handleAddExpression = () => setExpressionLibrary(prev => [...prev, '=Fields!Amount.Value * 1.1']);
  const handleEventScriptChange = (key: keyof EventScripts, value: string) => setEventScripts(prev => ({ ...prev, [key]: value }));
  const handleExportOptionToggle = (key: keyof ExportOptions, checked: boolean) => setExportOptions(prev => ({ ...prev, [key]: checked }));
  const handleExport = (key: string) => setSnackbar({ open: true, message: `Exporting report as ${exportFormatLabels[key as keyof ExportOptions]}...`, severity: 'info' });

  const handleLayoutSettingChange = (key: string, value: any) => setLayoutSettingsState((prev: any) => ({ ...prev, [key]: value }));
  const handleAddToken = (key: 'headerTokens' | 'footerTokens', token: string) => {
    const current = layoutSettingsState[key] || [];
    if (!current.includes(token)) setLayoutSettingsState((prev: any) => ({ ...prev, [key]: [...current, token] }));
  };
  const handleRemoveToken = (key: 'headerTokens' | 'footerTokens', token: string) => {
    const current = layoutSettingsState[key] || [];
    setLayoutSettingsState((prev: any) => ({ ...prev, [key]: current.filter((t: string) => t !== token) }));
  };

  return (
    <DndContext onDragEnd={handleDragEnd} onDragStart={handleDragStart}>
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', bgcolor: colors.bg }}>

        {/* ══════════════════════════════════════════════════════════════════
            TOP BAR — ribbon toolbar + editable title + BO switcher
        ══════════════════════════════════════════════════════════════════ */}
        <TopAppBar sx={{ bgcolor: isDark ? '#0D1117' : '#1E293B', color: '#FFFFFF', boxShadow: 'none', borderBottom: `1px solid rgba(255,255,255,0.06)` }}>
          <Box sx={{ display: 'flex', alignItems: 'center', width: '100%', px: 1 }}>
            {/* Left: action icons */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.25, flex: '0 0 auto' }}>
              <Tooltip title="Save (Ctrl+S)">
                <IconButton color="inherit" size="small" onClick={() => setSnackbar({ open: true, message: `"${reportTitle}" saved`, severity: 'success' })}>
                  <SaveIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.1)', mx: 0.5 }} />
              <Tooltip title="Undo">
                <span>
                  <IconButton color="inherit" size="small" onClick={undo} disabled={!canUndo}>
                    <UndoIcon sx={{ fontSize: 19 }} />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Redo">
                <span>
                  <IconButton color="inherit" size="small" onClick={redo} disabled={!canRedo}>
                    <RedoIcon sx={{ fontSize: 19 }} />
                  </IconButton>
                </span>
              </Tooltip>
              <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.1)', mx: 0.5 }} />
              <Tooltip title="Preview">
                <IconButton color="inherit" size="small" onClick={() => setActiveTab('preview')}>
                  <VisibilityIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Tooltip title="Print">
                <IconButton color="inherit" size="small" onClick={() => window.print()}>
                  <PrintIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Tooltip title="Export PDF">
                <IconButton color="inherit" size="small" onClick={() => generatePixelPerfectPDF(elements, layoutSettingsState)}>
                  <DownloadIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.1)', mx: 0.5 }} />
              <Tooltip title="Parameters">
                <IconButton color="inherit" size="small" onClick={() => setParametersOpen(true)}>
                  <SettingsIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Tooltip title="Page Layout">
                <IconButton color="inherit" size="small" onClick={() => setLayoutDrawerOpen(true)}>
                  <DashboardIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
            </Box>

            {/* Center: editable report title */}
            <Box sx={{ flex: 1, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
              {editingTitle ? (
                <TextField
                  autoFocus
                  value={reportTitle}
                  onChange={(e) => setReportTitle(e.target.value)}
                  onBlur={() => setEditingTitle(false)}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === 'Escape') setEditingTitle(false); }}
                  size="small"
                  sx={{
                    '& .MuiInputBase-root': { color: '#FFF', fontSize: '0.88rem', fontWeight: 700, bgcolor: 'rgba(255,255,255,0.1)', height: 30 },
                    '& .MuiOutlinedInput-notchedOutline': { borderColor: 'rgba(255,255,255,0.3)' },
                    width: 280,
                  }}
                />
              ) : (
                <Box
                  onClick={() => setEditingTitle(true)}
                  sx={{
                    display: 'flex', alignItems: 'center', gap: 0.5, cursor: 'text', px: 1, py: 0.5, borderRadius: 1,
                    '&:hover': { bgcolor: 'rgba(255,255,255,0.07)', '& .pencil': { opacity: 1 } },
                  }}
                >
                  <Typography sx={{ fontSize: '0.88rem', fontWeight: 700, color: '#FFF', letterSpacing: '-0.01em' }}>
                    {reportTitle}
                  </Typography>
                  <EditIcon className="pencil" sx={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', opacity: 0, transition: 'opacity 0.15s' }} />
                </Box>
              )}
            </Box>

            {/* Right: BO switcher */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: '0 0 auto' }}>
              <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.55)', fontWeight: 600, whiteSpace: 'nowrap', fontSize: '0.72rem' }}>
                Business Object
              </Typography>
              <FormControl size="small" sx={{ minWidth: 190 }}>
                <Select
                  value={selectedBOId}
                  displayEmpty
                  onChange={(e) => setSelectedBOId(e.target.value as string)}
                  sx={{
                    height: 28, color: '#FFF', bgcolor: 'rgba(255,255,255,0.09)', fontSize: '0.75rem', fontWeight: 600,
                    borderRadius: 1.5, '& .MuiSvgIcon-root': { color: '#FFF' },
                    '& fieldset': { borderColor: 'rgba(255,255,255,0.18)' },
                    '&:hover fieldset': { borderColor: 'rgba(255,255,255,0.35)' },
                  }}
                >
                  {businessObjects.length === 0 && <MenuItem value="">Loading…</MenuItem>}
                  {businessObjects.map((bo: any) => (
                    <MenuItem key={bo.id} value={bo.id}>{bo.displayName || bo.name} ({bo.key})</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>
          </Box>
        </TopAppBar>

        {/* ══════════════════════════════════════════════════════════════════
            BODY: Left sidebar + main area (tabs + content)
        ══════════════════════════════════════════════════════════════════ */}
        <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>

          {/* LEFT SIDEBAR */}
          <Drawer
            variant="persistent"
            open={drawerOpen}
            sx={{
              width: 280, flexShrink: 0,
              '& .MuiDrawer-paper': {
                width: 280, boxSizing: 'border-box', position: 'relative',
                bgcolor: colors.sidebarBg, color: colors.text, borderColor: colors.border,
                display: 'flex', flexDirection: 'column', overflow: 'hidden',
              },
            }}
          >
            <Box sx={{ borderBottom: `1px solid ${colors.border}`, bgcolor: colors.sidebarBg, px: 1, pt: 0.75 }}>
              <Tabs
                value={sidebarTab}
                onChange={(_, v) => setSidebarTab(v)}
                variant="fullWidth"
                sx={{ minHeight: 34, '& .MuiTab-root': { minHeight: 34, py: 0.5, fontSize: '0.72rem', fontWeight: 700, textTransform: 'none' } }}
              >
                <Tab icon={<LayersIcon sx={{ fontSize: 15 }} />} iconPosition="start"
                  label={`BO Fields (${(selectedBO?.coreFields?.length || 0) + (selectedBO?.customFields?.length || 0)})`}
                  value="fields" />
                <Tab icon={<GridViewIcon sx={{ fontSize: 15 }} />} iconPosition="start" label="Widgets" value="toolbox" />
              </Tabs>
            </Box>
            <Box sx={{ flex: 1, p: sidebarTab === 'fields' ? 0 : 2, overflowY: 'auto' }}>
              {sidebarTab === 'fields' ? (
                <BOFieldsPalette selectedBO={selectedBO} relatedBOs={relatedBOs} onAddFieldToCanvas={handleAddFieldToCanvas} onAddAllAsTable={handleAddAllAsTable} />
              ) : (
                <>
                  <Typography variant="subtitle2" fontWeight="700" sx={{ mb: 1.5, color: colors.text }}>Report Items</Typography>
                  {toolboxItems.map(item => <ToolboxItem key={item.type} type={item.type} icon={item.icon} label={item.label} />)}
                </>
              )}
            </Box>
          </Drawer>

          {/* MAIN AREA */}
          <Box component="main" sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Tab bar */}
            <Box sx={{ borderBottom: `1px solid ${colors.border}`, bgcolor: colors.sidebarBg, flexShrink: 0 }}>
              <Tabs value={activeTab} onChange={(_, v) => { setActiveTab(v); }} sx={{ minHeight: 38, '& .MuiTab-root': { minHeight: 38, py: 0.75, fontWeight: 600, fontSize: '0.78rem', textTransform: 'none' } }}>
                <Tab label="Design" value="design" />
                <Tab label="Preview" value="preview" />
                <Tab label="Data" value="data" />
                <Tab label="Schedule & Bursting" value="schedule" />
                <Tab label="Settings" value="settings" />
              </Tabs>
            </Box>

            {/* ════ DESIGN TAB ════ */}
            {activeTab === 'design' && (
              <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>
                {/* Canvas area */}
                <Box sx={{ flex: 1, p: 3, overflowY: 'auto', bgcolor: colors.bg, display: 'flex', justifyContent: 'center' }}>
                  <ReportCanvas
                    elements={elements}
                    layoutSettings={layoutSettingsState}
                    selectedElement={selectedElement}
                    onElementUpdate={handleElementUpdate}
                    onElementDelete={handleElementDelete}
                    onElementSelect={setSelectedElement}
                    onElementDuplicate={handleElementDuplicate}
                    onLayoutSettingsChange={handleLayoutSettingChangeFromCanvas}
                    sectionConfig={sectionConfig}
                    onSectionConfigChange={handleSectionConfigChange}
                    orientation={orientation}
                    isLivePreview={false}
                    availableFieldDefs={availableFieldDefs}
                  />
                </Box>

                {/* Right properties panel */}
                <Paper sx={{ width: 310, flexShrink: 0, overflowY: 'auto', bgcolor: colors.sidebarBg, borderLeft: `1px solid ${colors.border}`, borderRadius: 0 }}>
                  <PropertiesPanel
                    selectedElement={selectedElementData ?? null}
                    onElementUpdate={handleElementUpdate}
                    groupDefinitions={groupDefinitions}
                    selectedBO={selectedBO}
                    businessObjects={businessObjects}
                    activeDatasets={activeDatasets}
                  />
                </Paper>
              </Box>
            )}

            {/* ════ PREVIEW TAB ════ */}
            {activeTab === 'preview' && (
              <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', bgcolor: colors.bg }}>
                {/* Control bar */}
                <Paper sx={{ p: 1.25, px: 3, borderBottom: `1px solid ${colors.border}`, bgcolor: colors.cardBg, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderRadius: 0, flexShrink: 0 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                    <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text }}>
                      {selectedBO?.displayName || selectedBO?.name || 'Document'} — Preview
                    </Typography>
                    {previewData && previewData.length > 0 && (
                      <Chip size="small" label={`${previewData.length} records`}
                        sx={{ height: 20, fontSize: '0.68rem', bgcolor: isDark ? 'rgba(16,185,129,0.15)' : 'rgba(16,185,129,0.08)', color: '#10B981', fontWeight: 700 }} />
                    )}
                  </Box>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button variant="contained" size="small"
                      onClick={runPreviewQuery} disabled={previewLoading}
                      sx={{ bgcolor: colors.primary, color: '#FFF', textTransform: 'none', fontSize: '0.73rem', fontWeight: 700, height: 28 }}>
                      {previewLoading ? 'Loading…' : 'Refresh Preview'}
                    </Button>
                    <Button variant="outlined" size="small"
                      onClick={() => generatePixelPerfectPDF(elements, layoutSettingsState)}
                      sx={{ borderColor: colors.border, color: colors.text, textTransform: 'none', fontSize: '0.73rem', height: 28 }}>
                      Export PDF
                    </Button>
                    <Button variant="outlined" size="small"
                      onClick={() => setShowQueryDetails(!showQueryDetails)}
                      sx={{ borderColor: showQueryDetails ? colors.primary : colors.border, color: showQueryDetails ? colors.primary : colors.text, textTransform: 'none', fontSize: '0.73rem', height: 28 }}>
                      {showQueryDetails ? 'Hide SQL' : 'Inspect SQL'}
                    </Button>
                  </Box>
                </Paper>

                {/* Collapsible SQL inspector */}
                {showQueryDetails && (
                  <Paper sx={{ p: 2, m: 2, bgcolor: isDark ? '#080B10' : '#F1F5F9', border: `1px solid ${colors.border}`, borderRadius: 2, flexShrink: 0 }}>
                    <Typography variant="caption" sx={{ color: colors.textMuted, display: 'block', mb: 0.5, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                      Generated SQL
                    </Typography>
                    <Typography component="pre" sx={{ fontFamily: 'monospace', fontSize: 11, color: '#06B6D4', whiteSpace: 'pre-wrap', m: 0 }}>
                      {previewSQL || `SELECT * FROM ${selectedBO?.key || 'entity'} LIMIT 25;`}
                    </Typography>
                  </Paper>
                )}

                {/* Centered document */}
                <Box sx={{ flex: 1, p: 3, overflowY: 'auto', display: 'flex', justifyContent: 'center' }}>
                  <ReportCanvas
                    elements={elements}
                    layoutSettings={layoutSettingsState}
                    selectedElement={null}
                    onElementUpdate={() => {}}
                    onElementDelete={() => {}}
                    onElementSelect={() => {}}
                    sectionConfig={sectionConfig}
                    orientation={orientation}
                    isLivePreview={true}
                    previewData={previewData}
                    availableFieldDefs={availableFieldDefs}
                  />
                </Box>
              </Box>
            )}

            {/* ════ DATA TAB ════ */}
            {activeTab === 'data' && (
              <Box sx={{ p: 3, overflowY: 'auto', bgcolor: colors.bg }}>
                <Typography variant="subtitle1" fontWeight="700" sx={{ mb: 2.5, color: colors.text }}>Business Object Data Source</Typography>
                <Grid container spacing={3}>
                  <Grid item xs={12} md={7}>
                    <Paper sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2.5, bgcolor: colors.cardBg, border: `1px solid ${colors.border}` }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text }}>Primary Business Object</Typography>
                      <FormControl fullWidth size="small">
                        <InputLabel id="bo-select-label">Business Object</InputLabel>
                        <Select labelId="bo-select-label" value={selectedBOId} label="Business Object" onChange={(e) => setSelectedBOId(e.target.value as string)}>
                          <MenuItem value=""><em>None</em></MenuItem>
                          {businessObjects.map((bo: any) => (
                            <MenuItem key={bo.id} value={bo.id}>{bo.displayName || bo.name} ({bo.key})</MenuItem>
                          ))}
                        </Select>
                      </FormControl>

                      {selectedBOId && (
                        <FormControl fullWidth size="small">
                          <InputLabel id="binding-select-label">Active Binding</InputLabel>
                          <Select labelId="binding-select-label" value={selectedBindingId} label="Active Binding" onChange={(e) => setSelectedBindingId(e.target.value as string)}>
                            {bindings.length > 0
                              ? bindings.map((b: any) => <MenuItem key={b.id} value={b.id}>{b.name || `Binding: ${b.datasource_id || b.datasourceId}`} ({b.binding_type || b.bindingType || 'physical'})</MenuItem>)
                              : <MenuItem value="" disabled>No bindings defined</MenuItem>}
                          </Select>
                        </FormControl>
                      )}
                    </Paper>
                  </Grid>

                  <Grid item xs={12} md={5}>
                    <Paper sx={{ p: 3, height: '100%', bgcolor: colors.cardBg, border: `1px solid ${colors.border}` }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text }} gutterBottom>Related Business Objects</Typography>
                      {relatedBOs.length > 0 ? (
                        <Stack spacing={1} sx={{ mt: 1 }}>
                          {relatedBOs.map((rel: any, i: number) => (
                            <Box key={i} sx={{ p: 1.5, border: `1px solid ${colors.border}`, borderRadius: 1.5, bgcolor: colors.sidebarBg }}>
                              <Typography variant="subtitle2" sx={{ fontWeight: 700, color: colors.text }}>{rel.relatedObjectName}</Typography>
                              <Typography variant="caption" sx={{ color: colors.textMuted }}>{rel.relationshipType} · {rel.description}</Typography>
                            </Box>
                          ))}
                        </Stack>
                      ) : (
                        <Typography variant="body2" sx={{ color: colors.textMuted, mt: 1 }}>
                          {selectedBOId ? 'No related objects defined.' : 'Select a Business Object above.'}
                        </Typography>
                      )}
                    </Paper>
                  </Grid>

                  <Grid item xs={12}>
                    <Paper sx={{ p: 3, bgcolor: colors.cardBg, border: `1px solid ${colors.border}` }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text }} gutterBottom>Available Fields</Typography>
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mt: 1 }}>
                        {activeDatasets.flatMap(ds => ds.fields || []).map((f: any, idx: number) => (
                          <Chip key={`${f.name}_${idx}`} label={`${f.name} (${f.type})`} size="small"
                            sx={{ bgcolor: isDark ? 'rgba(99,102,241,0.12)' : 'rgba(99,102,241,0.07)', color: colors.primary, borderColor: colors.border, fontSize: '0.7rem' }} variant="outlined" />
                        ))}
                      </Box>
                    </Paper>
                  </Grid>
                </Grid>
              </Box>
            )}

            {/* ════ SCHEDULE & BURSTING TAB ════ */}
            {activeTab === 'schedule' && (
              <Box sx={{ p: 3, overflowY: 'auto', bgcolor: colors.bg }}>
                <ReportScheduleBurstingTab />
              </Box>
            )}

            {/* ════ SETTINGS TAB ════ */}
            {activeTab === 'settings' && (
              <Box sx={{ p: 3, overflowY: 'auto', bgcolor: colors.bg }}>
                <Typography variant="subtitle1" fontWeight="700" sx={{ mb: 2.5, color: colors.text }}>Report Settings</Typography>
                <Grid container spacing={3}>
                  {/* Groups */}
                  <Grid item xs={12}>
                    <Paper sx={{ p: 2.5, bgcolor: colors.cardBg, border: `1px solid ${colors.border}`, borderRadius: 2 }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text, mb: 2 }}>Grouping & Aggregation</Typography>
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
                  </Grid>

                  {/* Calculated Fields & Expressions */}
                  <Grid item xs={12}>
                    <Paper sx={{ p: 2.5, bgcolor: colors.cardBg, border: `1px solid ${colors.border}`, borderRadius: 2 }}>
                      <CalculatedFieldsEditor
                        calculatedFields={calculatedFields}
                        datasets={activeDatasets as unknown as any[]}
                        onAddCalculatedField={handleAddCalculatedField}
                        onCalculatedFieldChange={handleCalculatedFieldChange}
                        onRemoveCalculatedField={(fieldId) => setCalculatedFields((prev) => prev.filter((c) => c.id !== fieldId))}
                        boName={selectedBO?.name || 'BusinessObject'}
                      />
                      <Divider sx={{ my: 2, borderColor: colors.border }} />
                      <ExpressionsEditor expressionLibrary={expressionLibrary} onExpressionChange={handleExpressionChange} onAddExpression={handleAddExpression} />
                    </Paper>
                  </Grid>

                  {/* Event Scripts + Export */}
                  <Grid item xs={12}>
                    <Paper sx={{ p: 2.5, bgcolor: colors.cardBg, border: `1px solid ${colors.border}`, borderRadius: 2 }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text, mb: 2 }}>Event Scripts</Typography>
                      <EventScriptsEditor eventScripts={eventScripts} onEventScriptChange={handleEventScriptChange} />
                      <Divider sx={{ my: 2, borderColor: colors.border }}>Export Options</Divider>
                      <Grid container spacing={1.5}>
                        {(Object.keys(exportOptions) as Array<keyof ExportOptions>).map((key) => (
                          <Grid item xs={12} sm={6} md={4} key={String(key)}>
                            <Card variant="outlined" sx={{ p: 1.5, bgcolor: colors.sidebarBg, borderColor: colors.border }}>
                              <FormControlLabel control={<Switch size="small" checked={exportOptions[key]} onChange={(e) => handleExportOptionToggle(key, e.target.checked)} />} label={exportFormatLabels[key as keyof ExportOptions]} />
                              <Typography variant="caption" sx={{ color: colors.textMuted, display: 'block' }}>{exportOptionDescriptions[key as keyof ExportOptions]}</Typography>
                            </Card>
                          </Grid>
                        ))}
                      </Grid>
                      <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 2 }}>
                        {(Object.keys(exportOptions) as Array<keyof ExportOptions>).map((key) => (
                          <Button key={`exp_${String(key)}`} variant="contained" size="small" disabled={!exportOptions[key]} onClick={() => handleExport(key)} sx={{ bgcolor: colors.primary }}>
                            {exportFormatLabels[key as keyof ExportOptions]}
                          </Button>
                        ))}
                      </Box>
                    </Paper>
                  </Grid>
                </Grid>
              </Box>
            )}
          </Box>
        </Box>

        {/* ════ Layout Drawer (right side) ════ */}
        <Drawer anchor="right" open={layoutDrawerOpen} onClose={() => setLayoutDrawerOpen(false)}>
          <Box sx={{ width: 340, p: 2.5, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="h6" fontWeight="700">Page Layout</Typography>
            <PageSettings pageSize={pageSize} orientation={orientation} onChangePageSize={(v) => setPageSize(v)} onChangeOrientation={(v) => setOrientation(v)} />
            <Paper sx={{ p: 2 }}>
              <Typography variant="subtitle1" gutterBottom fontWeight="600">Pagination</Typography>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakBeforeGroup} onChange={(e) => handleLayoutSettingChange('pageBreakBeforeGroup', e.target.checked)} />} label="Break before group" /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakAfterGroup} onChange={(e) => handleLayoutSettingChange('pageBreakAfterGroup', e.target.checked)} />} label="Break after group" /></Grid>
                <Grid item xs={12} sm={6}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.fixedPageSize} onChange={(e) => handleLayoutSettingChange('fixedPageSize', e.target.checked)} />} label="Fixed page size" /></Grid>
                <Grid item xs={6} sm={4}><TextField fullWidth size="small" type="number" label="Columns" value={layoutSettingsState.columns} onChange={(e) => handleLayoutSettingChange('columns', Math.max(1, Number(e.target.value) || 1))} /></Grid>
                <Grid item xs={6} sm={8}><TextField fullWidth size="small" type="number" label="Column Spacing" value={layoutSettingsState.columnSpacing} onChange={(e) => handleLayoutSettingChange('columnSpacing', Math.max(0, Number(e.target.value) || 0))} InputProps={{ endAdornment: <InputAdornment position="end">px</InputAdornment> }} /></Grid>
              </Grid>
              <Divider sx={{ my: 2 }}>Header Tokens</Divider>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 1 }}>
                {layoutSettingsState.headerTokens.map((token: string) => (
                  <Chip key={`h_${token}`} size="small" label={token} onDelete={() => handleRemoveToken('headerTokens', token)} color="primary" variant="outlined" />
                ))}
              </Box>
              <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
                <TextField size="small" label="Add header token" value={headerTokenInput} onChange={(e) => setHeaderTokenInput(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter' && headerTokenInput.trim()) { handleAddToken('headerTokens', headerTokenInput.trim()); setHeaderTokenInput(''); }}} sx={{ flex: 1 }} />
                <Button variant="contained" size="small" onClick={() => { if (headerTokenInput.trim()) { handleAddToken('headerTokens', headerTokenInput.trim()); setHeaderTokenInput(''); }}}>Add</Button>
              </Box>
              <Divider sx={{ my: 2 }}>Footer Tokens</Divider>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 1 }}>
                {layoutSettingsState.footerTokens.map((token: string) => (
                  <Chip key={`f_${token}`} size="small" label={token} onDelete={() => handleRemoveToken('footerTokens', token)} color="secondary" variant="outlined" />
                ))}
              </Box>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <TextField size="small" label="Add footer token" value={footerTokenInput} onChange={(e) => setFooterTokenInput(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter' && footerTokenInput.trim()) { handleAddToken('footerTokens', footerTokenInput.trim()); setFooterTokenInput(''); }}} sx={{ flex: 1 }} />
                <Button variant="contained" size="small" onClick={() => { if (footerTokenInput.trim()) { handleAddToken('footerTokens', footerTokenInput.trim()); setFooterTokenInput(''); }}}>Add</Button>
              </Box>
            </Paper>
          </Box>
        </Drawer>

        <DataSourcesDialog open={dataSourcesOpen} onClose={() => setDataSourcesOpen(false)} />
        <ParametersDialog open={parametersOpen} onClose={() => setParametersOpen(false)} parameters={reportParameters} onAdd={handleAddParameter} onUpdate={handleUpdateParameter} onDelete={handleRemoveParameter} />
        <Snackbar open={snackbar.open} autoHideDuration={4000} onClose={handleCloseSnackbar} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}>
          <Alert onClose={handleCloseSnackbar} severity={snackbar.severity} sx={{ width: '100%' }}>{snackbar.message}</Alert>
        </Snackbar>
        <DragOverlay>
          {activeDragItem ? (
            <Box sx={{ p: 2, bgcolor: 'background.paper', border: '2px solid', borderColor: 'primary.main', borderRadius: 1.5, boxShadow: '0 8px 24px rgba(0,0,0,0.2)' }}>
              <Typography variant="body2" fontWeight="600">{activeDragItem.label || activeDragItem.type}</Typography>
            </Box>
          ) : null}
        </DragOverlay>
      </Box>
    </DndContext>
  );
};

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
});

const SSRSReportBuilder: React.FC = () => (
  <QueryClientProvider client={queryClient}>
    <SSRSReportBuilderContent />
  </QueryClientProvider>
);

export default SSRSReportBuilder;