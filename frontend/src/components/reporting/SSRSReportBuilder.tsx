import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { useParams, useLocation } from 'react-router-dom';
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
import { QueryClient, QueryClientProvider, useMutation } from '@tanstack/react-query';
import useUndo from 'use-undo';
import { getCachedGoldCopyId } from '../../utils/goldCopy';
import { readCachedSelection } from '../../utils/tenantScope';
import { apiClient } from '../../utils/apiClient';
import { getSelectedRegion } from '../../lib/region';

// Modular components & utils
import ToolboxItem from './ToolboxItem';
import ReportCanvas from './ReportCanvas';
import PropertiesPanel from './PropertiesPanel';
import DataSourcesDialog from './DataSourcesDialog';
import ParametersDialog from './ParametersDialog';
import ReportParametersToolbar from './ReportParametersToolbar';
import { SlidersHorizontal, Copy } from 'lucide-react';
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
import { FilterGroup, buildSQL } from './FilterBuilderPanel';
import CodeIcon from '@mui/icons-material/Code';
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
import DynamicFormIcon from '@mui/icons-material/DynamicForm';
import FirstPageIcon from '@mui/icons-material/FirstPage';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import NavigateBeforeIcon from '@mui/icons-material/NavigateBefore';
import NavigateNextIcon from '@mui/icons-material/NavigateNext';
import LastPageIcon from '@mui/icons-material/LastPage';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { useTheme } from '@mui/material';

import { useTenant } from '../../contexts/TenantContext';
import BOFieldsPalette, { BOField, extractAllBOFields } from './BOFieldsPalette';
import FilterBuilderPanel from './FilterBuilderPanel';
import ReportScheduleBurstingTab from './ReportScheduleBurstingTab';
import { FieldDefinition } from '../ExpressionBuilder/AdvancedConditionBuilder';
import { dedupeFields } from '../../utils/dedupeFields';
import { useCreateReportTemplate, useUpdateReportTemplate, useReportTemplate } from '../../api/reporting';
import { buildSavePayload, BOBinding } from './builderSerialization';
import { deserializeFromBackend, needsMigration, migrateV1ToV2 } from './tableSerialization';
import { UnifiedBOPickerModal } from '../common/UnifiedBOPickerModal';
import { FormTemplateSpec } from './form/FormManagerTypes';
import { useFormSpec } from './form/useFormSpec';
import { ReportFormManager } from './form/ReportFormManager';
import { ReportFormRenderer } from './form/ReportFormRenderer';
import { FormFieldPalette } from './form/FormFieldPalette';
import { ToolboxFormBlock } from './ToolboxFormBlock';
import { AdvancedReportSection, defaultSectionLayout } from './sectionLayoutModel';

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
  const { reportId: urlReportId } = useParams<{ reportId?: string }>();
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const colors = {
    // Use theme palette background values instead of hardcoded hex
    bg: theme.palette.background.default,
    cardBg: theme.palette.background.paper,
    sidebarBg: theme.palette.background.paper,
    border: theme.palette.divider,
    text: theme.palette.text.primary,
    textMuted: theme.palette.text.secondary,
    primary: theme.palette.primary.main,
    // Conditional formatting colors - MUI palette.main which is always defined
    positive: theme.palette.success?.main ?? '#10B981',
    negative: theme.palette.error?.main ?? '#EF4444',
    positiveBg: isDark ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.08)',
    negativeBg: isDark ? 'rgba(239, 68, 68, 0.2)' : 'rgba(239, 68, 68, 0.08)',
  };

  // Helper to build headers with authentication
  const getAuthHeaders = useCallback((additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';

    let tenantId = tenant?.id;
    let datasourceId = datasource?.id || (datasource as any)?.alpha_tenant_instance_id;

    if (!tenantId || !datasourceId) {
      try {
        const cached = readCachedSelection();
        tenantId = tenantId || cached.tenant?.id || getCachedGoldCopyId() || '';
        datasourceId = datasourceId || cached.datasource?.id || cached.datasource?.alpha_tenant_instance_id || '';
      } catch (_) {
        tenantId = tenantId || getCachedGoldCopyId() || '';
      }
    }

    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId || '',
      'X-Tenant-Datasource-ID': datasourceId || '',
      'X-Tenant-Region': getSelectedRegion() || 'us-west',
      ...additionalHeaders,
    };
  }, [tenant, datasource]);

  const [elementsState, { set: setElements, undo, redo, canUndo, canRedo }] = useUndo<any[]>([]);
  const elements = Array.isArray(elementsState.present) ? elementsState.present : [];

  const [selectedElement, setSelectedElement] = useState<string | null>(null);
  const [selectedSection, setSelectedSection] = useState<string | null>(null);
  const [activeDragItem, setActiveDragItem] = useState<any>(null);
  const [activeTab, setActiveTab] = useState('design');
  const [sidebarTab, setSidebarTab] = useState<'fields' | 'toolbox'>('fields');
  const [drawerOpen, setDrawerOpen] = useState(true);
  const [paletteWidth, setPaletteWidth] = useState<number>(380);
  const { formSpec, setFormSpec, formRegistry } = useFormSpec();

  const [focusedFormSectionId, setFocusedFormSectionId] = useState<string | null>(null);

  const startPaletteResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = paletteWidth;

    const onMove = (ev: MouseEvent) => {
      const delta = ev.clientX - startX;
      const newWidth = Math.min(640, Math.max(280, startWidth + delta));
      setPaletteWidth(newWidth);
    };
    const onUp = () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    document.body.style.cursor = 'ew-resize';
    document.body.style.userSelect = 'none';
  }, [paletteWidth]);
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

  // Advanced section layout state (ReportSectionContainer wrappers)
  const [layoutSections, setLayoutSections] = useState<AdvancedReportSection[]>(() =>
    Object.values(REPORT_SECTIONS).map((sec) => defaultSectionLayout(sec as any))
  );

  const handleUpdateSectionLayout = useCallback((id: string, patch: Partial<AdvancedReportSection>) => {
    setLayoutSections((prev) => prev.map((sec) => (sec.id === id ? { ...sec, ...patch } : sec)));
  }, []);

  const handleAddSubSection = useCallback((_parentId: string) => {
    // Phase 2: create CUSTOM_CONTAINER split. No-op for Phase 1.
  }, []);

  // Groups, calculated fields, expressions, event scripts, export options
  const [groupDefinitions, setGroupDefinitions] = useState<any[]>([]);

  const [calculatedFields, setCalculatedFields] = useState<CalculatedFieldItem[]>([
    { id: 'calc_margin', name: 'GrossMargin', expression: '=Fields!Revenue - Fields!Cost', datasetId: datasets[0]?.id ?? 'ds1', format: 'Currency' },
  ]);

  // Expression library - uses MUI theme colors, re-evaluates when theme changes
  const [expressionLibrary, setExpressionLibrary] = useState<string[]>([
    `=IIF(Fields!Growth.Value < 0, "${colors.negative}", "${colors.positive}")`,
    '=Sum(Fields!Sales.Value, "SalesGroup")',
  ]);

  const [eventScripts, setEventScripts] = useState<EventScripts>(() => ({
    onRowRender: `// Theme-aware conditional formatting\nconst isDark = document.documentElement.classList.contains('dark') || window.matchMedia('(prefers-color-scheme: dark)').matches;\n\nif (row.Fields.Growth < 0) {\n  row.Style.Background = isDark ? "rgba(239, 68, 68, 0.2)" : "rgba(239, 68, 68, 0.1)";\n  row.Style.Color = isDark ? "#F87171" : "#B91C1C";\n} else {\n  row.Style.Background = isDark ? "rgba(16, 185, 129, 0.2)" : "rgba(16, 185, 129, 0.1)";\n  row.Style.Color = isDark ? "#34D399" : "#15803D";\n}`,
    onCellRender: '// add tooltip\ncell.Tooltip = "{Field}: {Value}";',
    onPageRender: '// watermark\npage.Watermark = "Internal";',
    onExport: '// append metadata\nexportContext.Metadata.author = user.name;',
  }));

  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    includePrintFriendly: true,
    includeDrillThrough: true,
    includeComments: false,
  });

  const [reportParameters, setReportParameters] = useState<ReportParameter[]>([
    { id: 'param_year', name: 'Year', type: 'number', prompt: 'Enter a Year', defaultValue: String(new Date().getFullYear()) },
  ]);
  const [runtimeParamValues, setRuntimeParamValues] = useState<Record<string, any>>({
    Year: String(new Date().getFullYear()),
  });

  // Report title (editable in top bar)
  const [reportTitle, setReportTitle] = useState('Untitled Report');
  const [editingTitle, setEditingTitle] = useState(false);

  const handleAddParameter = (param: Omit<ReportParameter, 'id'>) => {
    const newParam = { ...param, id: `param_${Date.now()}` };
    setReportParameters(prev => [...prev, newParam]);
    if (newParam.defaultValue !== undefined) {
      setRuntimeParamValues(prev => ({ ...prev, [newParam.name]: newParam.defaultValue }));
    }
  };
  const handleUpdateParameter = (updatedParam: ReportParameter) => {
    setReportParameters(prev => prev.map(p => p.id === updatedParam.id ? updatedParam : p));
  };
  const handleRemoveParameter = (paramId: string) => {
    setReportParameters(prev => prev.filter(p => p.id !== paramId));
  };

  const currentUserProfile = useMemo(() => {
    let tenantId = tenant?.id;
    try {
      const cached = readCachedSelection();
      tenantId = tenantId || cached.tenant?.id || getCachedGoldCopyId() || '';
    } catch (_) {}
    return {
      id: typeof localStorage !== 'undefined' ? localStorage.getItem('user_id') || 'usr-current' : 'usr-current',
      tenantId: tenantId || 'gold_copy',
      tenantCode: tenant?.name || 'acme',
      accountId: typeof localStorage !== 'undefined' ? localStorage.getItem('account_id') || 'acc-institutional-001' : 'acc-institutional-001',
      clientId: typeof localStorage !== 'undefined' ? localStorage.getItem('client_id') || 'client-001' : 'client-001',
      branchId: typeof localStorage !== 'undefined' ? localStorage.getItem('branch_id') || 'NYC-MAIN' : 'NYC-MAIN',
      region: getSelectedRegion() || 'us-west',
    };
  }, [tenant]);

  const handleRuntimeParamChange = (paramName: string, value: any) => {
    setRuntimeParamValues(prev => ({ ...prev, [paramName]: value }));
  };

  const [headerTokenInput, setHeaderTokenInput] = useState('');
  const [footerTokenInput, setFooterTokenInput] = useState('');

  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'info' | 'warning' | 'error' });
  const handleCloseSnackbar = () => setSnackbar(prev => ({ ...prev, open: false }));

  const location = useLocation();
  const [boPickerOpen, setBoPickerOpen] = useState(false);

  // Business Object states
  const [businessObjects, setBusinessObjects] = useState<any[]>([]);
  const [selectedBOId, setSelectedBOId] = useState<string>('');
  const [selectedBO, setSelectedBO] = useState<any | null>(null);
  const [bindings, setBindings] = useState<any[]>([]);
  const [selectedBindingId, setSelectedBindingId] = useState<string>('');
  const [relatedBOs, setRelatedBOs] = useState<any[]>([]);
  const [activeDatasets, setActiveDatasets] = useState<any[]>([...datasets]);
  const [selectedSubtypeKey, setSelectedSubtypeKey] = useState<string | null>(null);

  // Preview state
  const [previewData, setPreviewData] = useState<any[] | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewSQL, setPreviewSQL] = useState<string | null>(null);
  const [showQueryDetails, setShowQueryDetails] = useState(false);
  const [reportFilterGroups, setReportFilterGroups] = useState<FilterGroup[]>([]);
  const [previewPage, setPreviewPage] = useState(1);
  const previewPageSize = 10;

  const previewDataRecord: Record<string, any> = useMemo(() => {
    if (!previewData || previewData.length === 0) return {};
    return previewData[0] as Record<string, any>;
  }, [previewData]);

  // Render state
  const [renderResult, setRenderResult] = useState<any | null>(null);
  const [renderLoading, setRenderLoading] = useState(false);

  // Save mutations
  const createMutation = useCreateReportTemplate();
  const updateMutation = useUpdateReportTemplate();

  // Load existing report template
  const { data: loadedTemplate } = useReportTemplate(urlReportId);

  // Load + auto-migrate v1 → v2 on mount
  useEffect(() => {
    if (!loadedTemplate?.definition) {
      return;
    }

    try {
      const def = deserializeFromBackend(loadedTemplate.definition as Record<string, unknown>);
      const needsMigrate = needsMigration(def);
      console.log('[DEBUG] elements after deserialize:', def.elements.map(e => ({ id: e.id, section: e.section, type: e.type })));

      // Load parameters from definition or metadata
      const loadedParams = (loadedTemplate as any)?.definition?.parameters || (loadedTemplate as any)?.metadata?.parameters;
      if (Array.isArray(loadedParams) && loadedParams.length > 0) {
        setReportParameters(loadedParams);
        const initialVals: Record<string, any> = {};
        loadedParams.forEach((p: ReportParameter) => {
          if (p.defaultValue !== undefined) {
            initialVals[p.name] = p.defaultValue;
          }
        });
        setRuntimeParamValues(prev => ({ ...initialVals, ...prev }));
      }

      if (needsMigrate) {
        const migrated = migrateV1ToV2(def);
        setElements(migrated.elements);
        setReportTitle(migrated.reportTitle || loadedTemplate.name || 'Untitled Report');
        setSectionConfig((migrated as any).sectionConfig || {});
        if ((migrated as any).layoutSettings) {
          setLayoutSettingsState((migrated as any).layoutSettings);
        }
        setSnackbar({
          open: true,
          message: 'Report upgraded to v2 format',
          severity: 'info',
        });
        // Persist v2 layout back — preserve any existing sectionConfig/layoutSettings from metadata
        const existingSectionConfig = (loadedTemplate as any)?.metadata?.sectionConfig || {};
        const existingLayoutSettings = (loadedTemplate as any)?.metadata?.layoutSettings;
        const loadedFormSpecMigrated = (migrated as any)?.formSpec
          ?? (loadedTemplate as any)?.metadata?.formSpec
          ?? null;
        setFormSpec(loadedFormSpecMigrated);
        const loadedFormRegistryMigrated = (migrated as any)?.formRegistry
          ?? (loadedTemplate as any)?.metadata?.formRegistry
          ?? {};
        const v2Payload = buildSavePayload(
          {
            elements: migrated.elements,
            reportTitle: migrated.reportTitle || loadedTemplate.name || 'Untitled Report',
            sectionConfig: { ...existingSectionConfig, ...((migrated as any).sectionConfig || {}) },
            layoutSettings: existingLayoutSettings || (migrated as any).layoutSettings,
            parameters: reportParameters,
            formSpec: loadedFormSpecMigrated,
            formRegistry: loadedFormRegistryMigrated,
            layoutSections: Object.values(REPORT_SECTIONS).map((sec) => defaultSectionLayout(sec as any)),
          },
          null,
          urlReportId
        );
        updateMutation.mutate({ id: urlReportId!, payload: v2Payload as any });
      } else {
        setElements(def.elements);
        setReportTitle(def.reportTitle || loadedTemplate.name || 'Untitled Report');
        setSectionConfig((def as any).sectionConfig || {});
        if ((def as any).layoutSettings) {
          setLayoutSettingsState((def as any).layoutSettings);
        }
        const loadedFormSpecV2 = (def as any)?.formSpec
          ?? (loadedTemplate as any)?.metadata?.formSpec
          ?? null;
        setFormSpec(loadedFormSpecV2);
        const loadedLayoutSections = (def as any)?.layoutSections
          ?? (loadedTemplate as any)?.metadata?.layoutSections
          ?? null;
        if (Array.isArray(loadedLayoutSections) && loadedLayoutSections.length > 0) {
          setLayoutSections(loadedLayoutSections);
        }
      }
    } catch (err) {
      console.error('[SSRSReportBuilder] Failed to load report:', err);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadedTemplate]);

  // Guardrail: clear form section focus when switching away from Form tab
  useEffect(() => {
    if (activeTab !== 'form') setFocusedFormSectionId(null);
  }, [activeTab]);

  // Phase 2d: Auto-select BO when report is loaded with a bo_path binding
  useEffect(() => {
    if (!loadedTemplate || businessObjects.length === 0) return;
    if (selectedBOId) return; // already have a selection

    // Try to read bo_path from saved metadata (new format: metadata.data_bindings[0].bo_path)
    const boPath = (loadedTemplate as any)?.metadata?.data_bindings?.[0]?.bo_path;
    if (!boPath) return;

    // bo_path like 'oms.account/institutional' or 'altinv.alternative_investment/private_equity'
    // Resolve to parent BO key: strip the subtype suffix after the last '/'
    const parentKey = boPath.includes('/') ? boPath.substring(0, boPath.lastIndexOf('/')) : boPath;

    // Find matching parent BO in the list (match by key or technicalName)
    const match = businessObjects.find((b: any) =>
      b.key === parentKey ||
      b.technicalName === parentKey ||
      b.technical_name === parentKey
    );

    if (match) {
      setSelectedBOId(match.id);
    }
  }, [loadedTemplate, businessObjects, selectedBOId]);

    const isGoldCopyTenant = Boolean(
      tenant?.gold_copy === true || (tenant?.id && tenant.id === getCachedGoldCopyId())
    );
    const isCoreTemplate = Boolean(
      loadedTemplate && (
        (loadedTemplate as any).is_core === true ||
        (loadedTemplate as any).gold_copy === true ||
        (loadedTemplate as any).report_type === 'core' ||
        (loadedTemplate as any).tenant_id === getCachedGoldCopyId()
      )
    );
    const isReadOnlyCore = isCoreTemplate && !isGoldCopyTenant;

    const handleSaveReport = useCallback(async () => {
      if (isReadOnlyCore) {
        setSnackbar({
          open: true,
          message: 'Core templates cannot be overwritten directly by client tenants. Please click "Clone" to create your own customizable copy.',
          severity: 'warning',
        });
        return;
      }

      const savedBO: BOBinding | null = urlReportId
        ? ((loadedTemplate as any)?.metadata?.data_bindings?.[0]?.bo_path
            ? { qualifiedPath: (loadedTemplate as any).metadata.data_bindings[0].bo_path }
            : (selectedBO as BOBinding | null))
        : (selectedBO as BOBinding | null);

      const payload = buildSavePayload(
        {
          elements,
          reportTitle,
          sectionConfig,
          layoutSettings: layoutSettingsState,
          parameters: reportParameters,
        },
        savedBO,
        urlReportId
      );
      try {
        if (urlReportId) {
          await updateMutation.mutateAsync({ id: urlReportId, payload });
        } else {
          const result = await createMutation.mutateAsync(payload as any);
          const newId = (result as any)?.id;
          if (newId) {
            window.history.replaceState({}, '', `/reports/builder/${newId}`);
          }
        }
        setSnackbar({ open: true, message: `"${reportTitle}" saved successfully (including ${reportParameters.length} parameters)`, severity: 'success' });
      } catch (err) {
        setSnackbar({ open: true, message: `Failed to save: ${err instanceof Error ? err.message : 'Unknown error'}`, severity: 'error' });
      }
    }, [elements, reportTitle, sectionConfig, layoutSettingsState, reportParameters, selectedBO, urlReportId, isReadOnlyCore, createMutation, updateMutation]);

    const handleCloneReport = useCallback(async () => {
      try {
        const baseName = reportTitle.replace(/\s*\(Custom\s*Copy\)/i, '').replace(/\s*\(Core\)/i, '');
        const cloneTitle = `${baseName} (Custom Copy)`;
        const payload = buildSavePayload(
          {
            elements,
            reportTitle: cloneTitle,
            sectionConfig,
            layoutSettings: layoutSettingsState,
            parameters: reportParameters,
          },
          selectedBO as BOBinding | null,
          undefined
        );
        (payload as any).is_core = false;
        (payload as any).name = cloneTitle;
        (payload as any).report_key = `${(loadedTemplate as any)?.report_key || 'rep'}_custom_${Date.now()}`;

        const result = await createMutation.mutateAsync(payload as any);
        const newId = (result as any)?.id || (result as any)?.report_id;
        setSnackbar({
          open: true,
          message: `Report cloned successfully as "${cloneTitle}"! You can now customize parameters, filters, and layout for your tenant.`,
          severity: 'success',
        });
        if (newId) {
          setTimeout(() => {
            window.location.href = `/reports/${newId}/edit`;
          }, 800);
        }
      } catch (err) {
        setSnackbar({
          open: true,
          message: `Failed to clone report: ${err instanceof Error ? err.message : 'Unknown error'}`,
          severity: 'error',
        });
      }
    }, [elements, reportTitle, sectionConfig, layoutSettingsState, reportParameters, selectedBO, loadedTemplate, createMutation]);

  const handleRunReport = useCallback(async (paramOverrides?: Record<string, any>) => {
    if (!urlReportId && !loadedTemplate?.report_key) {
      setSnackbar({ open: true, message: 'Save the report first before running', severity: 'warning' });
      return;
    }
    const key = urlReportId ? undefined : (loadedTemplate as any)?.report_key;
    const idOrKey = key || urlReportId;
    if (!idOrKey) return;

    const effectiveParams = paramOverrides || runtimeParamValues;

    setRenderLoading(true);
    try {
      const headers = getAuthHeaders();
      let url = `/api/v1/reports/${idOrKey}/render`;
      if (key) url = `/api/v1/reports/by-key/${key}/render`;
      const res = await apiClient<any>(url, {
        method: 'POST',
        headers,
        body: JSON.stringify({ parameters: effectiveParams }),
      });
      setRenderResult(res);
      if (res.rows && Array.isArray(res.rows)) {
        setPreviewData(res.rows);
      }
      setActiveTab('preview');
      setSnackbar({ open: true, message: `Rendered ${res.rowCount ?? res.rows?.length ?? 0} rows with parameters`, severity: 'success' });
    } catch (err) {
      setSnackbar({ open: true, message: `Render failed: ${err instanceof Error ? err.message : 'Unknown'}`, severity: 'error' });
    } finally {
      setRenderLoading(false);
    }
  }, [urlReportId, loadedTemplate, getAuthHeaders, runtimeParamValues]);



  // Fetch Business Objects list on mount
  useEffect(() => {
    let isMounted = true;

    const loadBOs = async () => {
      try {
        const headers = getAuthHeaders();
        const data = await apiClient<any>('/api/business-objects?format=array', { headers });

        if (!isMounted) return;

        // Handle both array and object responses
        let list: any[] = [];
        if (Array.isArray(data)) {
          list = data;
        } else if (typeof data === 'object' && data !== null) {
          if (data.data && Array.isArray(data.data)) {
            list = data.data;
          } else if (data.results && Array.isArray(data.results)) {
            list = data.results;
          } else {
            // Treat as map keyed by ID or key
            list = Object.entries(data).map(([id, val]: [string, any]) => ({
              id: val?.id || id,
              ...(typeof val === 'object' && val !== null ? val : {}),
            }));
          }
        }

        const normalizedList = list.map((bo: any) => ({
          id: bo.id || bo.key,
          key: bo.key || bo.technicalName || bo.technical_name || bo.name || bo.id,
          name: bo.name || bo.displayName || bo.display_name || bo.key || bo.id,
          displayName: bo.displayName || bo.display_name || bo.name || bo.key || bo.id,
          technicalName: bo.technicalName || bo.technical_name || bo.key || bo.name,
          coreFields: bo.coreFields || bo.core_fields || [],
          customFields: bo.customFields || bo.custom_fields || [],
          fields: bo.fields || bo.entity_fields || bo.config?.entity_fields || bo.config?.fields || [],
          config: bo.config || {},
          isCore: bo.isCore ?? bo.is_core ?? true,
        }));

        setBusinessObjects(normalizedList);
        // Check if BO ID was provided in URL query string / location state
        const searchParams = new URLSearchParams(window.location.search);
        const urlBO = searchParams.get('bo');
        const urlBinding = searchParams.get('binding');
        if (urlBinding) {
          setSelectedBindingId(urlBinding);
        }

        if (normalizedList.length > 0) {
          setSelectedBOId(prev => {
            if (urlBO && normalizedList.some((b: any) => b.id === urlBO || b.key === urlBO)) {
              return urlBO;
            }
            if (!prev || !normalizedList.some((b: any) => b.id === prev || b.key === prev)) {
              return normalizedList[0].id;
            }
            return prev;
          });
        }
      } catch (err) {
        if (isMounted) {
          console.error('[ReportBuilder] Failed to load business objects:', err);
        }
      }
    };

    if (tenant?.id) {
      loadBOs();
    }

    return () => {
      isMounted = false;
    };
  }, [getAuthHeaders, tenant?.id]);

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
        const headers = getAuthHeaders();
        let data: any = null;
        try {
          data = await apiClient<any>(`/api/business-objects/${selectedBOId}/with_bindings`, { headers });
        } catch (e) {
          const res = await fetch(`/api/business-objects/${selectedBOId}/with_bindings`, { headers });
          if (res.ok) {
            data = await res.json();
          }
        }

        let boData: any = null;
        let bList: any[] = [];
        let rList: any[] = [];
        let extraFields: any[] = [];

        if (data && (data.bo || data.id)) {
          boData = data.bo || data;
          bList = Array.isArray(data.bindings) ? data.bindings : data.bindings?.data || [];
          rList = Array.isArray(data.related_bos) ? data.related_bos : data.relatedBOs || [];
          extraFields = Array.isArray(data.fields) ? data.fields : [];
        } else {
          // Fallback: fetch BO details separately
          try {
            boData = await apiClient<any>(`/api/business-objects/${selectedBOId}`, { headers });
          } catch (e) {
            const boRes = await fetch(`/api/business-objects/${selectedBOId}`, { headers });
            if (boRes.ok) {
              boData = await boRes.json();
            }
          }
        }

        if (!boData) {
          boData = businessObjects.find((b: any) => b.id === selectedBOId || b.key === selectedBOId);
        }

        if (boData) {
          const extractedFields = extractAllBOFields(boData, 'all', extraFields);
          const coreFields = extractedFields.filter(f => f.isCore);
          const customFields = extractedFields.filter(f => !f.isCore);

          const fullBO = {
            ...boData,
            displayName: boData.displayName || boData.display_name || boData.name || boData.key,
            coreFields: coreFields.length > 0 || customFields.length > 0 ? coreFields : extractedFields,
            customFields,
            allExtractedFields: extractedFields,
          };
          setSelectedBO(fullBO);

          const boFields = extractedFields.map((f: BOField) => ({
            name: f.name,
            type: f.dataType || f.type || 'string',
            label: f.label || f.name,
          }));

          const boDataset = {
            id: `ds_${fullBO.id || fullBO.key || selectedBOId}`,
            name: fullBO.displayName || fullBO.name || fullBO.key || 'Business Object',
            fields: boFields,
          };
          setActiveDatasets([boDataset, ...datasets]);

          setBindings(bList);
          if (bList.length > 0) setSelectedBindingId(bList[0].id || bList[0].binding_id || '');
          setRelatedBOs(rList);
        }
      } catch (err) {
        console.error('Failed to load BO details:', err);
      }
    };
    loadBODetails();
  }, [selectedBOId, getAuthHeaders, businessObjects]);

  // Run preview query
  const runPreviewQuery = useCallback(async () => {
    setPreviewLoading(true);
    try {
      const filterSql = buildSQL(reportFilterGroups);
      const whereClause = filterSql ? `\n${filterSql}` : '';

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
          const baseSql = result.annotation?.generatedSQL || `SELECT ${fields.slice(0, 8).join(', ')} FROM ${selectedBO.name}`;
          setPreviewSQL(whereClause ? `${baseSql.replace(/;?\s*$/, '')}${whereClause}\nLIMIT 25;` : (result.annotation?.generatedSQL || `${baseSql} LIMIT 25;`));
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
      setPreviewSQL(`-- Simulated Query for ${selectedBO?.displayName || selectedBO?.name || 'Entity'}\nSELECT ${sampleFields.join(', ')}\nFROM ${selectedBO?.name || 'entity_table'}${whereClause}\nLIMIT 15;`);
      setSnackbar({ open: true, message: `Showing 15 sample records for ${selectedBO?.displayName || selectedBO?.name || 'report'}`, severity: 'info' });
    } catch (err: any) {
      console.error('Preview error:', err);
      setSnackbar({ open: true, message: 'Using sample preview data', severity: 'warning' });
    } finally {
      setPreviewLoading(false);
    }
  }, [selectedBO, getAuthHeaders, reportFilterGroups]);

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

  const handleAddToolboxItem = (type: string, targetSection: string = REPORT_SECTIONS.BODY, payload?: Record<string, unknown>) => {
    const defaultSizes: Record<string, { width: number; height: number }> = {
      [ELEMENT_TYPES.TEXTBOX]: { width: 180, height: 40 },
      [ELEMENT_TYPES.TABLE]: { width: 560, height: 180 },
      [ELEMENT_TYPES.MATRIX]: { width: 480, height: 160 },
      [ELEMENT_TYPES.LIST]: { width: 340, height: 140 },
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
      position: { x: 30 + (elements.length % 5) * 20, y: 30 + (elements.length % 5) * 20 },
      size: defaultSizes[type] || { width: 160, height: 60 },
      properties: {
        name: `${type.charAt(0).toUpperCase() + type.slice(1)} 1`,
        fontSize: 12,
        textColor: isDark ? '#E2E8F0' : '#1E293B',
        columns: type === ELEMENT_TYPES.TABLE || type === ELEMENT_TYPES.MATRIX || type === ELEMENT_TYPES.LIST ? [] : undefined,
      },
    };

    setElements([...elements, newElement]);
    setSelectedElement(newElement.id);
    setSnackbar({ open: true, message: `Added ${type} to ${targetSection}`, severity: 'success' });
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

    if (active.data.current?.isToolboxItem || active.id?.toString().startsWith('toolbox-')) {
      const type = active.data.current?.type || active.id.toString().replace('toolbox-', '');
      const payload = active.data.current?.payload as Record<string, unknown> | undefined;

      // form-block / formReference: create formReference element directly
      if (type === 'formReference' || type === 'form-block') {
        const templateId = payload?.templateId as string | undefined;
        const newElement = {
          id: `formref_${Date.now()}`,
          type: 'formReference',
          section: targetSection,
          position: { x: 30, y: 30 },
          size: { width: 600, height: 300 },
          properties: {
            templateId: templateId
              ? { isExpression: false, value: templateId }
              : { isExpression: false, value: '' },
            containerStyle: {},
          },
        };
        setElements([...elements, newElement]);
        setSelectedElement(newElement.id);
        setSnackbar({ open: true, message: `Added Form Block to ${targetSection}`, severity: 'success' });
        return;
      }

      // form-block-multi: delegate to ReportSection's onFormBlockAdd (picker dialog)
      if (type === 'form-block-multi') {
        // onFormBlockAdd will be called by ReportSection's native onDrop handler
        // which reads dataTransfer and calls onFormBlockAdd(section, { mode, templateId })
        handleAddToolboxItem(type, targetSection, payload);
        return;
      }

      handleAddToolboxItem(type, targetSection, payload);
      return;
    }

    if (active.data.current?.isBOField && active.data.current?.field) {
      const bundle: BOField[] = active.data.current.selectedFields || [active.data.current.field];
      if (bundle.length > 1) {
        handleAddAllAsTable(bundle);
      } else if (bundle.length === 1) {
        handleAddFieldToCanvas(bundle[0]);
      }
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

  const handleSectionSelect = useCallback((sectionId: string) => {
    setSelectedSection(sectionId);
    setSelectedElement(null);
  }, []);

  const handleNavigateToFormTab = useCallback(
    (_templateId: string) => {
      setActiveTab('form');
    },
    []
  );

  const handleAddFormBlock = useCallback(
    (section: string, payload: { mode: string; templateId: string }) => {
      const newElement = {
        id: `formref_${Date.now()}`,
        type: 'formReference',
        section,
        position: { x: 30, y: 30 },
        size: { width: 600, height: 300 },
        properties: {
          templateId: { isExpression: false, value: payload.templateId },
          containerStyle: {},
        },
      };
      setElements([...elements, newElement]);
      setSelectedElement(newElement.id);
      setSelectedSection(null);
      setSnackbar({
        open: true,
        message: `Added Form Block to ${section}`,
        severity: 'success',
      });
    },
    [elements, setElements]
  );

  const handleSectionDeleted = useCallback(
    (snapshot: any) => {
      let undoRan = false;
      const undo = () => {
        if (undoRan) return;
        undoRan = true;
        if (formSpec) {
          setFormSpec({
            ...formSpec,
            sections: [...formSpec.sections, snapshot],
          });
        }
      };
      setSnackbar({
        open: true,
        message: `Section "${snapshot.title}" deleted`,
        severity: 'warning',
        action: {
          label: 'Undo',
          onClick: undo,
        },
      });
    },
    [formSpec, setFormSpec]
  );

  const selectedElementData = useMemo(() => elements.find((el: any) => el.id === selectedElement), [elements, selectedElement]);

  // Convert BO fields to FieldDefinition[] for Expression Builders (includes root + all subtypes)
  const availableFieldDefs: (FieldDefinition & { _scope?: 'root' | 'subtype'; _subtypeKey?: string })[] = useMemo(() => {
    if (!selectedBO) return [];
    const fields: any[] = [];
    const seen = new Set<string>();

    const registerField = (f: any, scope: 'root' | 'subtype', subtypeKey?: string) => {
      const key = f.key || f.technicalName || f.technical_name || f.name;
      if (!key || typeof key !== 'string') return;
      const scopedKey = `${scope}:${subtypeKey || 'root'}:${key}`;
      if (seen.has(scopedKey)) return;
      seen.add(scopedKey);

      const fieldName = f.label || f.displayName || f.businessName || f.name || key;
      const dataType = f.dataType || f.data_type || f.type || f.fieldType || f.field_type || 'string';
      const type = (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].includes(dataType.toLowerCase())
        ? 'number'
        : ['boolean', 'bool'].includes(dataType.toLowerCase())
        ? 'boolean'
        : ['date', 'time', 'timestamp', 'datetime'].includes(dataType.toLowerCase())
        ? 'date'
        : 'string') as any;

      fields.push({ name: key, label: fieldName, type, _scope: scope, _subtypeKey: subtypeKey });
    };

    (selectedBO.coreFields || selectedBO.core_fields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.customFields || selectedBO.custom_fields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.fields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.entity_fields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.config?.entity_fields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.config?.fields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.config?.inheritedFields || []).forEach((f: any) => registerField(f, 'root'));
    (selectedBO.config?.customFields || []).forEach((f: any) => registerField(f, 'root'));

    if (selectedBO.subtypes && typeof selectedBO.subtypes === 'object') {
      Object.entries(selectedBO.subtypes).forEach(([stKey, subtype]: [string, any]) => {
        (subtype.subtypeFields || []).forEach((f: any) => registerField(f, 'subtype', stKey));
      });
    }

    return fields;
  }, [selectedBO]);

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
  const handleRemoveExpression = (index: number) => setExpressionLibrary(prev => prev.filter((_, i) => i !== index));
  const handleEventScriptChange = (key: keyof EventScripts, value: string) => setEventScripts(prev => ({ ...prev, [key]: value }));
  const handleExportOptionToggle = (key: keyof ExportOptions, checked: boolean) => setExportOptions(prev => ({ ...prev, [key]: checked }));
  const handleExport = (key: string) => setSnackbar({ open: true, message: `Exporting report as ${exportFormatLabels[key as keyof ExportOptions]}...`, severity: 'info' });

  const handleLayoutSettingChange = (key: string, value: any) => setLayoutSettingsState((prev: any) => ({ ...prev, [key]: value }));
  const handleAddToken = (key: 'headerTokens' | 'footerTokens', token: any) => {
    if (!token) return;
    const current: any[] = layoutSettingsState[key] || [];
    const isObjectToken = typeof token === 'object' && token !== null && 'id' in token;
    if (isObjectToken) {
      if (!current.find((t: any) => t && t.id === token.id)) {
        setLayoutSettingsState((prev: any) => ({ ...prev, [key]: [...current, token] }));
      }
    } else {
      if (!current.find((t: any) => t && (typeof t === 'string' ? t === token : t.text === token))) {
        setLayoutSettingsState((prev: any) => ({ ...prev, [key]: [...current, { id: `tok_${Date.now()}`, text: token, mode: 'static' }] }));
      }
    }
  };
  const handleRemoveToken = (key: 'headerTokens' | 'footerTokens', token: any) => {
    if (!token) return;
    const current: any[] = layoutSettingsState[key] || [];
    const isObjectToken = typeof token === 'object' && token !== null && 'id' in token;
    if (isObjectToken) {
      setLayoutSettingsState((prev: any) => ({ ...prev, [key]: current.filter((t: any) => t && t.id !== token.id) }));
    } else {
      setLayoutSettingsState((prev: any) => ({ ...prev, [key]: current.filter((t: any) => t && (typeof t === 'string' ? t !== token : t.text !== token)) }));
    }
  };

  return (
    <DndContext onDragEnd={handleDragEnd} onDragStart={handleDragStart}>
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', bgcolor: colors.bg }}>

        {/* ══════════════════════════════════════════════════════════════════
            TOP BAR — ribbon toolbar + editable title + BO switcher
        ══════════════════════════════════════════════════════════════════ */}
        <TopAppBar sx={{ bgcolor: theme.palette.background.paper, color: theme.palette.text.primary, boxShadow: 'none', borderBottom: `1px solid ${theme.palette.divider}` }}>
          <Box sx={{ display: 'flex', alignItems: 'center', width: '100%', px: 1 }}>
            {/* Left: action icons */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.25, flex: '0 0 auto' }}>
              <Tooltip title={isReadOnlyCore ? "Core template (read-only). Click Clone to create a custom tenant copy." : "Save (Ctrl+S)"}>
                <span>
                  <IconButton size="small" onClick={handleSaveReport} disabled={createMutation.isPending || updateMutation.isPending || isReadOnlyCore}
                    sx={{ color: isReadOnlyCore ? 'rgba(255,255,255,0.3)' : 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                    <SaveIcon sx={{ fontSize: 19 }} />
                  </IconButton>
                </span>
              </Tooltip>

              <Tooltip title="Clone as Tenant Custom Report">
                <Button
                  size="small"
                  variant="contained"
                  onClick={handleCloneReport}
                  startIcon={<Copy size={13} />}
                  sx={{
                    bgcolor: '#0D9488',
                    color: '#FFF',
                    textTransform: 'none',
                    fontSize: '0.72rem',
                    fontWeight: 700,
                    height: 28,
                    borderRadius: 1.5,
                    px: 1.2,
                    mx: 0.5,
                    '&:hover': { bgcolor: '#0F766E' },
                  }}
                >
                  Clone
                </Button>
              </Tooltip>

              <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.1)', mx: 0.5 }} />
              <Tooltip title="Undo">
                <span>
                  <IconButton size="small" onClick={undo} disabled={!canUndo || isReadOnlyCore}
                    sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                    <UndoIcon sx={{ fontSize: 19 }} />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Redo">
                <span>
                  <IconButton size="small" onClick={redo} disabled={!canRedo || isReadOnlyCore}
                    sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                    <RedoIcon sx={{ fontSize: 19 }} />
                  </IconButton>
                </span>
              </Tooltip>
              <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.1)', mx: 0.5 }} />
              <Tooltip title="Preview">
                <IconButton size="small" onClick={() => setActiveTab('preview')}
                  sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                  <VisibilityIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Tooltip title="Run Report">
                <span>
                  <IconButton size="small" onClick={handleRunReport} disabled={renderLoading}
                    sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                    <PlayArrowIcon sx={{ fontSize: 19 }} />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Print">
                <IconButton size="small" onClick={() => window.print()}
                  sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                  <PrintIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Tooltip title="Export PDF">
                <IconButton size="small" onClick={() => generatePixelPerfectPDF(elements, layoutSettingsState)}
                  sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                  <DownloadIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
              <Divider orientation="vertical" flexItem sx={{ borderColor: 'rgba(255,255,255,0.1)', mx: 0.5 }} />
              <Tooltip title="Configure Report Parameters (Modal)">
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => setParametersOpen(true)}
                  startIcon={<SlidersHorizontal size={13} />}
                  sx={{
                    color: 'rgba(255,255,255,0.85)',
                    borderColor: 'rgba(255,255,255,0.2)',
                    textTransform: 'none',
                    fontSize: '0.72rem',
                    fontWeight: 700,
                    height: 28,
                    borderRadius: 1.5,
                    px: 1,
                    '&:hover': { color: '#FFF', borderColor: 'rgba(255,255,255,0.4)', bgcolor: 'rgba(255,255,255,0.08)' },
                  }}
                >
                  Parameters ({reportParameters.length})
                </Button>
              </Tooltip>
              <Tooltip title="Page Layout">
                <IconButton size="small" onClick={() => setLayoutDrawerOpen(true)}
                  sx={{ color: 'rgba(255,255,255,0.7)', '&:hover': { color: 'white', bgcolor: 'rgba(255,255,255,0.1)' } }}>
                  <DashboardIcon sx={{ fontSize: 19 }} />
                </IconButton>
              </Tooltip>
            </Box>

            {/* Center: editable report title & core template badge */}
            <Box sx={{ flex: 1, display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 1 }}>
              {isCoreTemplate && (
                <Chip
                  size="small"
                  label={isReadOnlyCore ? 'Core Template (Read-Only)' : 'Core Template (Master)'}
                  sx={{
                    height: 22,
                    fontSize: '0.65rem',
                    fontWeight: 800,
                    bgcolor: isReadOnlyCore ? 'rgba(245, 158, 11, 0.2)' : 'rgba(16, 185, 129, 0.2)',
                    color: isReadOnlyCore ? '#FBBF24' : '#34D399',
                    border: `1px solid ${isReadOnlyCore ? 'rgba(245, 158, 11, 0.4)' : 'rgba(16, 185, 129, 0.4)'}`,
                  }}
                />
              )}
              {editingTitle && !isReadOnlyCore ? (
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
                  onClick={() => !isReadOnlyCore && setEditingTitle(true)}
                  sx={{
                    display: 'flex', alignItems: 'center', gap: 0.5, cursor: isReadOnlyCore ? 'default' : 'text', px: 1, py: 0.5, borderRadius: 1,
                    '&:hover': isReadOnlyCore ? {} : { bgcolor: 'rgba(255,255,255,0.07)', '& .pencil': { opacity: 1 } },
                  }}
                >
                  <Typography sx={{ fontSize: '0.88rem', fontWeight: 700, color: '#FFF', letterSpacing: '-0.01em' }}>
                    {reportTitle}
                  </Typography>
                  {!isReadOnlyCore && <EditIcon className="pencil" sx={{ fontSize: 13, color: 'rgba(255,255,255,0.4)', opacity: 0, transition: 'opacity 0.15s' }} />}
                </Box>
              )}
            </Box>

            {/* Right: Immutable Business Object & Binding Badge */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: '0 0 auto' }}>
              {selectedBO ? (
                <Tooltip title="Business Object and Datasource Binding are immutable for this report">
                  <Chip
                    icon={<LockOutlinedIcon sx={{ fontSize: '13px !important', color: '#FFF !important' }} />}
                    label={`${selectedBO.displayName || selectedBO.name} • ${selectedBindingId || 'Default Binding'}`}
                    size="small"
                    sx={{
                      height: 28,
                      color: '#FFF',
                      bgcolor: 'rgba(255,255,255,0.12)',
                      fontSize: '0.75rem',
                      fontWeight: 700,
                      borderRadius: 1.5,
                      border: '1px solid rgba(255,255,255,0.2)',
                    }}
                  />
                </Tooltip>
              ) : (
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => setBoPickerOpen(true)}
                  sx={{
                    height: 28,
                    color: '#FFF',
                    borderColor: 'rgba(255,255,255,0.3)',
                    fontSize: '0.75rem',
                    textTransform: 'none',
                    fontWeight: 600,
                  }}
                >
                  Select Business Object
                </Button>
              )}
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
              width: paletteWidth, flexShrink: 0,
              '& .MuiDrawer-paper': {
                width: paletteWidth, boxSizing: 'border-box', position: 'relative',
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
                <Tab icon={<LayersIcon sx={{ fontSize: 15 }} />} iconPosition="start" label={`BO Fields (${availableFieldDefs.length})`} value="fields" />
                <Tab icon={<GridViewIcon sx={{ fontSize: 15 }} />} iconPosition="start" label="Widgets" value="toolbox" />
              </Tabs>
            </Box>
            <Box sx={{ flex: 1, p: sidebarTab === 'fields' ? 0 : 2, overflowY: 'auto' }}>
              {sidebarTab === 'fields' ? (
                <BOFieldsPalette
                  selectedBO={selectedBO}
                  relatedBOs={relatedBOs}
                  onAddFieldToCanvas={handleAddFieldToCanvas}
                  onAddAllAsTable={handleAddAllAsTable}
                  onResize={startPaletteResize}
                  mode={activeTab === 'form' ? 'form' : 'design'}
                />
              ) : (
                <>
                  {activeTab === 'form' && (
                    <FormFieldPalette />
                  )}
                  <Typography variant="subtitle2" fontWeight="700" sx={{ mb: 1.5, color: colors.text }}>
                    {activeTab === 'form' ? 'Form Blocks' : 'Report Items'}
                  </Typography>
                  {toolboxItems.map(item => (
                    <ToolboxItem
                      key={item.type}
                      type={item.type}
                      icon={item.icon}
                      label={item.label}
                      onAdd={handleAddToolboxItem}
                    />
                  ))}
                  <ToolboxFormBlock
                    formRegistry={formRegistry}
                    onAddItem={handleAddToolboxItem}
                  />
                </>
              )}
            </Box>
            {/* Data Source footer card */}
            <Box sx={{ p: 1.5, borderTop: `1px solid ${colors.border}`, bgcolor: isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)', flexShrink: 0 }}>
              <Typography variant="caption" fontWeight="700" sx={{ color: colors.textMuted, display: 'block', mb: 0.5, textTransform: 'uppercase', letterSpacing: '0.05em', fontSize: '0.6rem' }}>
                Data Source
              </Typography>
              {selectedBO ? (
                <Box>
                  <Typography variant="caption" fontWeight={600} sx={{ color: colors.text, display: 'block', fontSize: '0.72rem' }}>
                    {selectedBO.displayName || selectedBO.name}
                  </Typography>
                  <Typography variant="caption" sx={{ color: colors.textMuted, display: 'block', fontSize: '0.65rem', fontFamily: 'monospace' }}>
                    {selectedBO.key || selectedBO.technicalName || selectedBO.technical_name}
                  </Typography>
                  {selectedBO.subtypes && Object.keys(selectedBO.subtypes).length > 0 && (
                    <Typography variant="caption" sx={{ color: colors.primary, display: 'block', fontSize: '0.65rem', mt: 0.25 }}>
                      {Object.keys(selectedBO.subtypes).length} subtypes
                    </Typography>
                  )}
                  {selectedBO.coreFields?.length > 0 && (
                    <Typography variant="caption" sx={{ color: colors.textMuted, display: 'block', fontSize: '0.65rem' }}>
                      {selectedBO.coreFields.length} fields
                    </Typography>
                  )}
                </Box>
              ) : (
                <Typography variant="caption" sx={{ color: colors.textMuted, fontSize: '0.65rem', fontStyle: 'italic' }}>
                  No BO selected
                </Typography>
              )}
            </Box>
          </Drawer>

          {/* MAIN AREA */}
          <Box component="main" sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Tab bar */}
            <Box sx={{ borderBottom: `1px solid ${colors.border}`, bgcolor: colors.sidebarBg, flexShrink: 0 }}>
              <Tabs
                value={activeTab}
                onChange={(_, v) => {
                  setActiveTab(v);
                  if (v === 'filters') {
                    setSidebarTab('fields');
                    setDrawerOpen(true);
                  }
                }}
                sx={{ minHeight: 38, '& .MuiTab-root': { minHeight: 38, py: 0.75, fontWeight: 600, fontSize: '0.78rem', textTransform: 'none' } }}
              >
                <Tab label="Design" value="design" />
                <Tab label="Form" value="form" icon={<DynamicFormIcon sx={{ fontSize: 15 }} />} iconPosition="start" />
                <Tab label="Preview" value="preview" />
                <Tab label="Filters" value="filters" />
                <Tab label="Schedule & Bursting" value="schedule" />
                <Tab label="Settings" value="settings" />
              </Tabs>
            </Box>

            {/* ════ DESIGN TAB ════ */}
            {activeTab === 'design' && (
              <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                {/* Canvas top bar with SQL Inspection */}
                <Paper sx={{ p: 1, px: 2.5, borderBottom: `1px solid ${colors.border}`, bgcolor: colors.cardBg, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderRadius: 0, flexShrink: 0 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="caption" fontWeight="700" sx={{ color: colors.textMuted, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                      Design Canvas
                    </Typography>
                    {reportFilterGroups.length > 0 && (
                      <Chip
                        size="small"
                        label={`${reportFilterGroups.reduce((acc, g) => acc + g.conditions.filter(c => c.enabled).length, 0)} Active Filters`}
                        sx={{ height: 20, fontSize: '0.68rem', bgcolor: 'rgba(0, 212, 255, 0.1)', color: '#00D4FF', fontWeight: 700 }}
                      />
                    )}
                    <Chip
                      size="small"
                      icon={<SlidersHorizontal size={11} />}
                      label={`${reportParameters.length} Parameters`}
                      onClick={() => setParametersOpen(true)}
                      sx={{
                        height: 20,
                        fontSize: '0.68rem',
                        fontWeight: 700,
                        cursor: 'pointer',
                        bgcolor: isDark ? 'rgba(13, 148, 136, 0.15)' : 'rgba(13, 148, 136, 0.08)',
                        color: isDark ? '#2DD4BF' : '#0D9488',
                        border: '1px solid rgba(13, 148, 136, 0.3)',
                        '&:hover': { bgcolor: 'rgba(13, 148, 136, 0.25)' },
                      }}
                    />
                  </Box>
                  <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<CodeIcon sx={{ fontSize: 14 }} />}
                      onClick={() => setShowQueryDetails(!showQueryDetails)}
                      sx={{
                        borderColor: showQueryDetails ? colors.primary : colors.border,
                        color: showQueryDetails ? colors.primary : colors.text,
                        textTransform: 'none',
                        fontSize: '0.72rem',
                        height: 26,
                        py: 0,
                      }}
                    >
                      {showQueryDetails ? 'Hide SQL' : 'View SQL WHERE'}
                    </Button>
                  </Box>
                </Paper>

                {/* Core Template Read-Only Notice for Client Tenants */}
                {isReadOnlyCore && (
                  <Paper sx={{ p: 1.2, px: 2, m: 2, mb: 0, bgcolor: isDark ? 'rgba(245, 158, 11, 0.12)' : '#FEF3C7', border: '1px solid rgba(245, 158, 11, 0.35)', borderRadius: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
                    <Typography variant="caption" sx={{ color: isDark ? '#FCD34D' : '#92400E', fontWeight: 600, fontSize: '0.75rem' }}>
                      <strong>Core Report Template (Read-Only Design):</strong> Master templates cannot be overwritten. You can schedule and run this report for your tenant, or click <strong>Clone</strong> to create a customizable copy.
                    </Typography>
                    <Button
                      size="small"
                      variant="contained"
                      onClick={handleCloneReport}
                      startIcon={<Copy size={13} />}
                      sx={{ bgcolor: '#0D9488', color: '#FFF', fontSize: '0.7rem', fontWeight: 700, textTransform: 'none', py: 0.3, px: 1.2, borderRadius: 1, '&:hover': { bgcolor: '#0F766E' } }}
                    >
                      Clone Report
                    </Button>
                  </Paper>
                )}

                {/* Collapsible SQL preview on canvas */}
                {showQueryDetails && (
                  <Paper sx={{ p: 2, m: 2, bgcolor: isDark ? '#080B10' : '#F1F5F9', border: `1px solid ${colors.border}`, borderRadius: 2, flexShrink: 0 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                      <Typography variant="caption" sx={{ color: colors.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                        Compiled Semantic AST SQL (Including Filters &amp; WHERE Clause)
                      </Typography>
                      <Chip size="small" label="Live Pushdown" sx={{ height: 18, fontSize: '0.62rem', bgcolor: 'rgba(16, 185, 129, 0.15)', color: '#10B981', fontWeight: 700 }} />
                    </Box>
                    <Typography component="pre" sx={{ fontFamily: 'monospace', fontSize: 11, color: '#06B6D4', whiteSpace: 'pre-wrap', m: 0 }}>
                      {(() => {
                        const fields = (selectedBO?.coreFields || selectedBO?.customFields || [])
                          .map((f: any) => f.name || f.technicalName).slice(0, 8);
                        const selFields = fields.length > 0 ? fields.join(', ') : '*';
                        const boTable = selectedBO?.name || 'entity_table';
                        const filterSql = buildSQL(reportFilterGroups);
                        const whereClause = filterSql ? `\n${filterSql}` : '';
                        return `SELECT ${selFields}\nFROM ${boTable}${whereClause}\nLIMIT 25;`;
                      })()}
                    </Typography>
                  </Paper>
                )}

                <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>
                  {/* Canvas area */}
                  <Box sx={{ flex: 1, p: 3, overflowY: 'auto', bgcolor: colors.bg, display: 'flex', justifyContent: 'center' }}>
                    <ReportCanvas
                      elements={elements}
                      layoutSettings={layoutSettingsState}
                      selectedElement={selectedElement}
                      onElementUpdate={handleElementUpdate}
                      onElementDelete={handleElementDelete}
                      onElementSelect={(id: string) => { setSelectedElement(id); setSelectedSection(null); }}
                      onElementAdd={(newEl) => {
                        setElements([...elements, newEl]);
                        setSelectedElement(newEl.id);
                        setSelectedSection(null);
                        setSnackbar({ open: true, message: `Added "${newEl.properties?.name || newEl.type}" to canvas`, severity: 'success' });
                      }}
                      onElementDuplicate={handleElementDuplicate}
                      onLayoutSettingsChange={handleLayoutSettingChangeFromCanvas}
                      sectionConfig={sectionConfig}
                      onSectionConfigChange={handleSectionConfigChange}
                      selectedSection={selectedSection}
                      onSectionSelect={handleSectionSelect}
                      orientation={orientation}
                      isLivePreview={false}
                      availableFieldDefs={availableFieldDefs}
                      formRegistry={formRegistry}
                      onFormBlockAdd={handleAddFormBlock}
                      layoutSections={layoutSections}
                      onUpdateSectionLayout={handleUpdateSectionLayout}
                      onAddSubSection={handleAddSubSection}
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
                      availableFieldDefs={availableFieldDefs}
                      selectedSection={selectedSection}
                      sectionConfig={sectionConfig}
                      onSectionConfigChange={handleSectionConfigChange}
                      layoutSettings={layoutSettingsState}
                      onLayoutSettingsChange={handleLayoutSettingChangeFromCanvas}
                      formRegistry={formRegistry}
                    />
                  </Paper>
                </Box>
              </Box>
            )}

            {/* ════ FORM TAB ════ */}
            {activeTab === 'form' && (
              <Box sx={{ flex: 1, overflow: 'hidden' }}>
                <ReportFormManager
                  formSpec={formSpec}
                  onFormSpecChange={setFormSpec}
                  availableFields={availableFieldDefs.map((f: any) => ({
                    key: f.technicalName || f.name,
                    name: f.displayName || f.name,
                    type: f.dataType || 'string',
                  }))}
                  previewData={previewDataRecord}
                  focusedSectionId={focusedFormSectionId}
                  onFocusSection={setFocusedFormSectionId}
                  onSectionDeleted={handleSectionDeleted}
                />
              </Box>
            )}

            {/* ════ PREVIEW TAB ════ */}
            {activeTab === 'preview' && (
              <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', bgcolor: colors.bg }}>
                {/* SSRS-Style Runtime Parameter Toolbar */}
                <Box sx={{ p: 2, pb: 0, flexShrink: 0 }}>
                  <ReportParametersToolbar
                    parameters={reportParameters}
                    values={runtimeParamValues}
                    onChange={handleRuntimeParamChange}
                    onRun={(params) => handleRunReport(params)}
                    currentUserProfile={currentUserProfile}
                    loading={renderLoading}
                    reportId={urlReportId}
                    reportKey={(loadedTemplate as any)?.report_key || 'rep-custom-001'}
                  />
                </Box>

                {/* Control bar with First/Prev/Next/Last Page Navigation */}
                <Paper sx={{ p: 1.25, px: 3, borderBottom: `1px solid ${colors.border}`, bgcolor: colors.cardBg, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderRadius: 0, flexShrink: 0 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                    <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text }}>
                      {selectedBO?.displayName || selectedBO?.name || 'Document'} — Preview
                    </Typography>
                    {previewData && previewData.length > 0 && (
                      <Chip size="small" label={`${previewData.length} records`}
                        sx={{ height: 20, fontSize: '0.68rem', bgcolor: isDark ? 'rgba(16,185,129,0.15)' : 'rgba(16,185,129,0.08)', color: '#10B981', fontWeight: 700 }} />
                    )}
                    {renderResult && renderResult.rowCount !== undefined && (
                      <Chip size="small" label={`Rendered: ${renderResult.rowCount} rows`}
                        sx={{ height: 20, fontSize: '0.68rem', bgcolor: isDark ? 'rgba(59,130,246,0.15)' : 'rgba(59,130,246,0.08)', color: '#3B82F6', fontWeight: 700 }} />
                    )}

                    {/* SSRS-Style Page Navigation Toolbar */}
                    {previewData && previewData.length > 0 && (() => {
                      const totalPages = Math.max(1, Math.ceil(previewData.length / previewPageSize));
                      return (
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, ml: 2, bgcolor: isDark ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.04)', p: '2px 6px', borderRadius: 1.5, border: `1px solid ${colors.border}` }}>
                          <Tooltip title="First Page">
                            <span>
                              <IconButton
                                size="small"
                                disabled={previewPage <= 1}
                                onClick={() => setPreviewPage(1)}
                                sx={{ color: colors.text, p: 0.25 }}
                              >
                                <FirstPageIcon sx={{ fontSize: 18 }} />
                              </IconButton>
                            </span>
                          </Tooltip>

                          <Tooltip title="Previous Page">
                            <span>
                              <IconButton
                                size="small"
                                disabled={previewPage <= 1}
                                onClick={() => setPreviewPage(p => Math.max(1, p - 1))}
                                sx={{ color: colors.text, p: 0.25 }}
                              >
                                <NavigateBeforeIcon sx={{ fontSize: 18 }} />
                              </IconButton>
                            </span>
                          </Tooltip>

                          <Typography sx={{ fontSize: '0.72rem', fontWeight: 700, color: colors.text, px: 1, fontFamily: 'monospace' }}>
                            Page {previewPage} of {totalPages}
                          </Typography>

                          <Tooltip title="Next Page">
                            <span>
                              <IconButton
                                size="small"
                                disabled={previewPage >= totalPages}
                                onClick={() => setPreviewPage(p => Math.min(totalPages, p + 1))}
                                sx={{ color: colors.text, p: 0.25 }}
                              >
                                <NavigateNextIcon sx={{ fontSize: 18 }} />
                              </IconButton>
                            </span>
                          </Tooltip>

                          <Tooltip title="Last Page">
                            <span>
                              <IconButton
                                size="small"
                                disabled={previewPage >= totalPages}
                                onClick={() => setPreviewPage(totalPages)}
                                sx={{ color: colors.text, p: 0.25 }}
                              >
                                <LastPageIcon sx={{ fontSize: 18 }} />
                              </IconButton>
                            </span>
                          </Tooltip>
                        </Box>
                      );
                    })()}
                  </Box>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button variant="contained" size="small"
                      onClick={() => { setPreviewPage(1); runPreviewQuery(); }} disabled={previewLoading}
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

                {/* Form Layout Manager Render (when form spec is defined) */}
                {formSpec && (
                  <Box sx={{ p: 3, overflowY: 'auto', bgcolor: '#050D1A' }}>
                    <Typography variant="caption" sx={{ color: '#94A3B8', mb: 2, textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 700, display: 'block' }}>
                      Form Render
                    </Typography>
                    <ReportFormRenderer formSpec={formSpec} previewData={previewDataRecord} />
                    <Box sx={{ my: 3, borderTop: `1px solid ${colors.border}` }} />
                  </Box>
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
                    previewData={(() => {
                      if (!previewData || previewData.length === 0) return null;
                      const start = (previewPage - 1) * previewPageSize;
                      return previewData.slice(start, start + previewPageSize);
                    })()}
                    availableFieldDefs={availableFieldDefs}
                    layoutSections={layoutSections}
                    onUpdateSectionLayout={handleUpdateSectionLayout}
                    onAddSubSection={handleAddSubSection}
                  />
                </Box>
              </Box>
            )}

            {/* ════ DATA TAB ════ */}
            {/* ════ SCHEDULE & BURSTING TAB ════ */}
            {activeTab === 'schedule' && (
              <Box sx={{ p: 3, overflowY: 'auto', bgcolor: colors.bg }}>
                <ReportScheduleBurstingTab
                  reportId={urlReportId}
                  reportName={reportTitle}
                  tenantId={tenant?.id}
                />
              </Box>
            )}

            {/* ════ FILTERS TAB ════ */}
            {activeTab === 'filters' && (
              <Box sx={{ flexGrow: 1, display: 'flex', overflow: 'hidden' }}>
                <FilterBuilderPanel
                  selectedBO={selectedBO}
                  reportId={urlReportId}
                  parameters={reportParameters}
                  isReadOnly={isReadOnlyCore}
                  onClone={handleCloneReport}
                  onChange={setReportFilterGroups}
                  onParametersChange={setReportParameters}
                />
              </Box>
            )}

            {/* ════ SETTINGS TAB ════ */}
            {activeTab === 'settings' && (
              <Box sx={{ p: 3, overflowY: 'auto', bgcolor: colors.bg }}>
                <Typography variant="subtitle1" fontWeight="700" sx={{ mb: 2.5, color: colors.text }}>Report Settings</Typography>
                <Grid container spacing={3}>
                  {/* Branding Info */}
                  <Grid size={12}>
                    <Paper sx={{ p: 2.5, bgcolor: colors.cardBg, border: `1px solid ${colors.border}`, borderRadius: 2 }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text, mb: 1 }}>Project Branding</Typography>
                      <Typography variant="body2" sx={{ color: colors.textMuted }}>
                        The internal application name is <strong>Uuisce</strong> (pronounced "ish-ka"). This is the semantic layer project name. Ensure all generated UI components, app titles, exports, metadata, and documentation consistently use <strong>Uuisce</strong> instead of any placeholder, legacy derivation, or variant (e.g., not "ishka", "UISCE", or other forms).
                      </Typography>
                    </Paper>
                  </Grid>

                  {/* Groups */}
                  <Grid size={12}>
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
                  <Grid size={12}>
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
                      <ExpressionsEditor
                        expressionLibrary={expressionLibrary}
                        onExpressionChange={handleExpressionChange}
                        onAddExpression={handleAddExpression}
                        onRemoveExpression={handleRemoveExpression}
                      />
                    </Paper>
                  </Grid>

                  {/* Event Scripts + Export */}
                  <Grid size={12}>
                    <Paper sx={{ p: 2.5, bgcolor: colors.cardBg, border: `1px solid ${colors.border}`, borderRadius: 2 }}>
                      <Typography variant="subtitle2" fontWeight="700" sx={{ color: colors.text, mb: 2 }}>Event Scripts</Typography>
                      <EventScriptsEditor eventScripts={eventScripts} onEventScriptChange={handleEventScriptChange} />
                      <Divider sx={{ my: 2, borderColor: colors.border }}>Export Options</Divider>
                      <Grid container spacing={1.5}>
                        {(Object.keys(exportOptions) as Array<keyof ExportOptions>).map((key) => (
                          <Grid size={{ xs: 12, sm: 6, md: 4 }} key={String(key)}>
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
            <PageSettings pageSize={pageSize} orientation={orientation} onChangePageSize={(v) => setPageSize(v)} onChangeOrientation={(v) => setOrientation(v as 'Portrait' | 'Landscape')} />
            <Paper sx={{ p: 2 }}>
              <Typography variant="subtitle1" gutterBottom fontWeight="600">Pagination</Typography>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, sm: 6 }}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakBeforeGroup} onChange={(e) => handleLayoutSettingChange('pageBreakBeforeGroup', e.target.checked)} />} label="Break before group" /></Grid>
                <Grid size={{ xs: 12, sm: 6 }}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.pageBreakAfterGroup} onChange={(e) => handleLayoutSettingChange('pageBreakAfterGroup', e.target.checked)} />} label="Break after group" /></Grid>
                <Grid size={{ xs: 12, sm: 6 }}><FormControlLabel control={<Switch size="small" checked={layoutSettingsState.fixedPageSize} onChange={(e) => handleLayoutSettingChange('fixedPageSize', e.target.checked)} />} label="Fixed page size" /></Grid>
                <Grid size={{ xs: 6, sm: 4 }}><TextField fullWidth size="small" type="number" label="Columns" value={layoutSettingsState.columns} onChange={(e) => handleLayoutSettingChange('columns', Math.max(1, Number(e.target.value) || 1))} /></Grid>
                <Grid size={{ xs: 6, sm: 8 }}><TextField fullWidth size="small" type="number" label="Column Spacing" value={layoutSettingsState.columnSpacing} onChange={(e) => handleLayoutSettingChange('columnSpacing', Math.max(0, Number(e.target.value) || 0))} InputProps={{ endAdornment: <InputAdornment position="end">px</InputAdornment> }} /></Grid>
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

        <ParametersDialog
          open={parametersOpen}
          onClose={() => setParametersOpen(false)}
          parameters={reportParameters}
          onAdd={handleAddParameter}
          onUpdate={handleUpdateParameter}
          onDelete={handleRemoveParameter}
          isReadOnly={isReadOnlyCore}
          onClone={handleCloneReport}
        />
        <UnifiedBOPickerModal
          open={boPickerOpen}
          context="report"
          businessObjects={businessObjects}
          onClose={() => setBoPickerOpen(false)}
          onPick={(bo, bindingId, _selectedRelatedBOs, _bindingDetails, selectedSubtypeKey) => {
            setSelectedBOId(bo.id);
            if (bindingId) setSelectedBindingId(bindingId);
            setSelectedSubtypeKey(selectedSubtypeKey ?? null);
            setBoPickerOpen(false);
          }}
        />
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