import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  Chip,
  Button,
  TextField,
  InputAdornment,
  CircularProgress,
  Divider,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Alert,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  IconButton,
  Tooltip,
  Autocomplete,
  FormControlLabel,
  Switch,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import RefreshIcon from '@mui/icons-material/Refresh';
import LanguageIcon from '@mui/icons-material/Language';
import SettingsIcon from '@mui/icons-material/Settings';
import KeyIcon from '@mui/icons-material/Key';
import LinkIcon from '@mui/icons-material/Link';
import LinkOffIcon from '@mui/icons-material/LinkOff';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ViewModuleIcon from '@mui/icons-material/ViewModule';
import ViewListIcon from '@mui/icons-material/ViewList';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTenant } from '../../../contexts/TenantContext';
import apiClient from '../../../utils/apiClient';

interface ApiDatasource {
  id: string;
  node_name: string;
  qualified_path: string;
  description: string;
  config: any;
  properties: any;
  tenant_id: string;
  is_active: boolean;
  tenant_base_url?: string;
  tenant_auth_type?: string;
}

interface ApiEndpoint {
  id: string;
  node_name: string;
  qualified_path: string;
  description: string;
  config: any;
  properties: any;
  parent_id: string;
  resource_name: string;
  datasource_id: string;
  datasource_name: string;
  fields_count: number;
  semantic_terms_count: number;
}

interface ApiField {
  id: string;
  node_name: string;
  qualified_path: string;
  description: string;
  config: any;
  properties: any;
  mapped_semantic_term_name?: string;
  mapped_semantic_term_id?: string;
  mapped_semantic_term_desc?: string;
  edge_id?: string;
}

interface SemanticTermOption {
  id: string;
  node_name: string;
  description: string;
  data_type: string;
}

interface LineageData {
  nodes: Array<{
    id: string;
    data: {
      label: string;
      type: string;
      icon?: string;
      category?: string;
      method?: string;
      dataType?: string;
      service?: string;
    };
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
    label?: string;
  }>;
}

export const ApiInventoryPage: React.FC = () => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [datasources, setDatasources] = useState<ApiDatasource[]>([]);
  const [endpoints, setEndpoints] = useState<ApiEndpoint[]>([]);
  const [loading, setLoading] = useState(true);

  // Facet Filters
  const [search, setSearch] = useState('');
  const [selectedVendor, setSelectedVendor] = useState<string>('ALL');
  const [selectedMethod, setSelectedMethod] = useState<string>('ALL');
  const [selectedResource, setSelectedResource] = useState<string>('ALL');
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('grid');

  // Pagination
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(12);

  // Drilldown Detail State
  const selectedEndpointId = searchParams.get('id');
  const [endpointDetail, setEndpointDetail] = useState<any>(null);
  const [fields, setFields] = useState<ApiField[]>([]);
  const [lineage, setLineage] = useState<LineageData | null>(null);
  const [recentCalls, setRecentCalls] = useState<any[]>([]);
  const [recentCallsLoading, setRecentCallsLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  // Semantic Terms for Mapping
  const [availableTerms, setAvailableTerms] = useState<SemanticTermOption[]>([]);
  const [termsLoading, setTermsLoading] = useState(false);

  // Map Term Modal State
  const [isMapModalOpen, setIsMapModalOpen] = useState(false);
  const [selectedFieldForMap, setSelectedFieldForMap] = useState<ApiField | null>(null);
  const [chosenTerm, setChosenTerm] = useState<SemanticTermOption | null>(null);
  const [isSavingMapping, setIsSavingMapping] = useState(false);

  // Add Field Modal State
  const [isAddFieldOpen, setIsAddFieldOpen] = useState(false);
  const [newFieldName, setNewFieldName] = useState('');
  const [newFieldDataType, setNewFieldDataType] = useState('varchar');
  const [newFieldJsonPath, setNewFieldJsonPath] = useState('');
  const [newFieldIsPk, setNewFieldIsPk] = useState(false);
  const [newFieldDesc, setNewFieldDesc] = useState('');
  const [newFieldTerm, setNewFieldTerm] = useState<SemanticTermOption | null>(null);
  const [isSavingField, setIsSavingField] = useState(false);

  // Live Test Execution State
  const [isExecuting, setIsExecuting] = useState(false);
  const [execResult, setExecResult] = useState<any>(null);
  const [testPathParams, setTestPathParams] = useState<Record<string, string>>({});
  const [testQueryParams, setTestQueryParams] = useState<Record<string, string>>({});
  const [testBody, setTestBody] = useState<string>('{}');

  // Connection Config Modal
  const [isConfigOpen, setIsConfigOpen] = useState(false);
  const [configBaseUrl, setConfigBaseUrl] = useState('');
  const [configAuthType, setConfigAuthType] = useState('oauth2_bearer');
  const [configToken, setConfigToken] = useState('');
  const [configUsername, setConfigUsername] = useState('');
  const [configPassword, setConfigPassword] = useState('');
  const [configApiKey, setConfigApiKey] = useState('');
  const [configApiKeyHeader, setConfigApiKeyHeader] = useState('X-API-Key');
  // OAuth 2.0 fields (used when auth_type === 'oauth2_bearer')
  const [configClientId, setConfigClientId] = useState('');
  const [configClientSecret, setConfigClientSecret] = useState('');
  const [configRefreshToken, setConfigRefreshToken] = useState('');
  const [configTokenUrl, setConfigTokenUrl] = useState('');
  const [configScopes, setConfigScopes] = useState('');
  const [oauthConfigured, setOauthConfigured] = useState<{ has_client_secret: boolean; has_refresh_token: boolean } | null>(null);
  const [isSavingConfig, setIsSavingConfig] = useState(false);
  const [configStatus, setConfigStatus] = useState<{ success: boolean; message: string } | null>(null);

  // Add API Datasource modal (OpenAPI ingester)
  const [isAddApiOpen, setIsAddApiOpen] = useState(false);
  const [addApiName, setAddApiName] = useState('');
  const [addApiUrl, setAddApiUrl] = useState('');
  const [addApiSource, setAddApiSource] = useState<'url' | 'paste'>('url');
  const [addApiSpec, setAddApiSpec] = useState('');
  const [isIngesting, setIsIngesting] = useState(false);
  const [addApiStatus, setAddApiStatus] = useState<{ success: boolean; message: string; details?: any } | null>(null);

  // Fetch Summary Data
  const fetchData = async () => {
    setLoading(true);
    try {
      const [dsRes, epRes] = await Promise.all([
        apiClient<any>(`/api/api-dispatcher/datasources?tenant_id=${tenantId}`),
        apiClient<any>(`/api/api-dispatcher/endpoints?tenant_id=${tenantId}`),
      ]);
      setDatasources(dsRes?.data || []);
      setEndpoints(epRes?.data || []);
    } catch (err) {
      console.error('Failed to load API catalog:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [tenantId]);

  // Fetch Available Semantic Terms
  const fetchSemanticTerms = async (q = '') => {
    setTermsLoading(true);
    try {
      const res = await apiClient<any>(`/api/api-dispatcher/semantic-terms?tenant_id=${tenantId}&q=${encodeURIComponent(q)}`);
      setAvailableTerms(res?.data || []);
    } catch (err) {
      console.error('Failed to load semantic terms:', err);
    } finally {
      setTermsLoading(false);
    }
  };

  useEffect(() => {
    fetchSemanticTerms();
  }, [tenantId]);

  // Fetch Endpoint Detail when drilldown ID is set
  const reloadEndpointDetail = async (endpointId: string) => {
    setDetailLoading(true);
    try {
      const [detailRes, fieldsRes, lineageRes] = await Promise.all([
        apiClient<any>(`/api/api-dispatcher/endpoints/${endpointId}?tenant_id=${tenantId}`),
        apiClient<any>(`/api/api-dispatcher/fields?endpoint_id=${endpointId}`),
        apiClient<any>(`/api/api-dispatcher/lineage?endpoint_id=${endpointId}`),
      ]);

      const detail = detailRes?.data;
      setEndpointDetail(detail);
      setFields(fieldsRes?.data || []);
      setLineage(lineageRes || null);

      if (detail?.config?.method === 'POST' || detail?.config?.method === 'PUT') {
        const sampleBody: Record<string, any> = {};
        (fieldsRes?.data || []).forEach((f: ApiField) => {
          sampleBody[f.node_name] = f.properties?.data_type === 'numeric' ? 1000 : 'Sample ' + f.node_name;
        });
        setTestBody(JSON.stringify(sampleBody, null, 2));
      }
    } catch (err) {
      console.error('Failed to load endpoint detail:', err);
    } finally {
      setDetailLoading(false);
    }
  };

  const reloadAudit = async (endpointId: string) => {
    setRecentCallsLoading(true);
    try {
      const res = await apiClient<any>(`/api/api-dispatcher/audit?tenant_id=${tenantId}&endpoint_id=${endpointId}&limit=50`);
      setRecentCalls(res?.data || []);
    } catch (err) {
      console.error('Failed to load audit:', err);
      setRecentCalls([]);
    } finally {
      setRecentCallsLoading(false);
    }
  };

  useEffect(() => {
    if (!selectedEndpointId) {
      setEndpointDetail(null);
      setFields([]);
      setLineage(null);
      setRecentCalls([]);
      return;
    }
    reloadEndpointDetail(selectedEndpointId);
    reloadAudit(selectedEndpointId);
  }, [selectedEndpointId, tenantId]);

  useEffect(() => {
    // When the user lands on the Recent Calls tab, refresh in case the
    // user just executed a request from the Live Test Runner.
    if (activeTab === 4 && selectedEndpointId) {
      reloadAudit(selectedEndpointId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, selectedEndpointId]);

  // After a successful execute, refresh the audit list so the new row appears.
  useEffect(() => {
    if (execResult?.success && selectedEndpointId) {
      // small delay so the fire-and-forget writer has time to land
      const t = setTimeout(() => reloadAudit(selectedEndpointId), 350);
      return () => clearTimeout(t);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [execResult?.duration_ms]);

  // Open Map Term Modal
  const handleOpenMapModal = (field: ApiField) => {
    setSelectedFieldForMap(field);
    const existing = availableTerms.find((t) => t.id === field.mapped_semantic_term_id || t.node_name === field.mapped_semantic_term_name);
    setChosenTerm(existing || (field.mapped_semantic_term_name ? { id: field.mapped_semantic_term_id || '', node_name: field.mapped_semantic_term_name, description: '', data_type: '' } : null));
    setIsMapModalOpen(true);
  };

  // Save Mapping (or Unmap)
  const handleSaveMapping = async () => {
    if (!selectedFieldForMap || !selectedEndpointId) return;
    setIsSavingMapping(true);
    try {
      await apiClient<any>('/api/api-dispatcher/map-term', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          field_id: selectedFieldForMap.id,
          semantic_term_id: chosenTerm ? chosenTerm.id : '',
          tenant_id: tenantId,
        }),
      });
      setIsMapModalOpen(false);
      await reloadEndpointDetail(selectedEndpointId);
      await fetchData();
    } catch (err) {
      console.error('Failed to save semantic mapping:', err);
    } finally {
      setIsSavingMapping(false);
    }
  };

  // Direct Unmap Quick Action
  const handleUnmapField = async (field: ApiField) => {
    if (!selectedEndpointId) return;
    try {
      await apiClient<any>('/api/api-dispatcher/map-term', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          field_id: field.id,
          semantic_term_id: '',
          tenant_id: tenantId,
        }),
      });
      await reloadEndpointDetail(selectedEndpointId);
      await fetchData();
    } catch (err) {
      console.error('Failed to unmap field:', err);
    }
  };

  // Open Add Field Modal
  const handleOpenAddField = () => {
    setNewFieldName('');
    setNewFieldDataType('varchar');
    setNewFieldJsonPath('');
    setNewFieldIsPk(false);
    setNewFieldDesc('');
    setNewFieldTerm(null);
    setIsAddFieldOpen(true);
  };

  // Save New Field
  const handleSaveNewField = async () => {
    if (!newFieldName || !selectedEndpointId) return;
    setIsSavingField(true);
    try {
      await apiClient<any>('/api/api-dispatcher/fields', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          endpoint_id: selectedEndpointId,
          node_name: newFieldName,
          data_type: newFieldDataType,
          json_path: newFieldJsonPath || `$.${newFieldName}`,
          is_primary_key: newFieldIsPk,
          description: newFieldDesc,
          semantic_term_id: newFieldTerm ? newFieldTerm.id : '',
          tenant_id: tenantId,
        }),
      });
      setIsAddFieldOpen(false);
      await reloadEndpointDetail(selectedEndpointId);
      await fetchData();
    } catch (err) {
      console.error('Failed to create field:', err);
    } finally {
      setIsSavingField(false);
    }
  };

  // Delete Field
  const handleDeleteField = async (fieldId: string) => {
    if (!selectedEndpointId) return;
    try {
      await apiClient<any>(`/api/api-dispatcher/fields/${fieldId}`, {
        method: 'DELETE',
      });
      await reloadEndpointDetail(selectedEndpointId);
      await fetchData();
    } catch (err) {
      console.error('Failed to delete field:', err);
    }
  };

  // Facet Options
  const vendors = useMemo(() => {
    const set = new Set<string>();
    endpoints.forEach((ep) => {
      if (ep.datasource_name) set.add(ep.datasource_name);
    });
    return Array.from(set);
  }, [endpoints]);

  const resources = useMemo(() => {
    const set = new Set<string>();
    endpoints.forEach((ep) => {
      if (ep.resource_name) set.add(ep.resource_name);
    });
    return Array.from(set);
  }, [endpoints]);

  // Filtered Endpoints
  const filteredEndpoints = useMemo(() => {
    return endpoints.filter((ep) => {
      const method = (ep.config?.method || (ep.node_name.startsWith('POST') ? 'POST' : 'GET')).toUpperCase();
      const matchesSearch =
        ep.node_name.toLowerCase().includes(search.toLowerCase()) ||
        ep.description.toLowerCase().includes(search.toLowerCase()) ||
        (ep.config?.path_template || '').toLowerCase().includes(search.toLowerCase()) ||
        ep.datasource_name.toLowerCase().includes(search.toLowerCase());

      const matchesVendor = selectedVendor === 'ALL' || ep.datasource_name === selectedVendor;
      const matchesMethod = selectedMethod === 'ALL' || method === selectedMethod;
      const matchesResource = selectedResource === 'ALL' || ep.resource_name === selectedResource;

      return matchesSearch && matchesVendor && matchesMethod && matchesResource;
    });
  }, [endpoints, search, selectedVendor, selectedMethod, selectedResource]);

  const totalTermsCount = useMemo(() => {
    return endpoints.reduce((acc, ep) => acc + (ep.semantic_terms_count || 0), 0);
  }, [endpoints]);

  const configuredCount = useMemo(() => {
    return datasources.filter((d) => !!d.tenant_base_url).length;
  }, [datasources]);

  // Execute Endpoint Test
  const handleExecute = async () => {
    if (!endpointDetail) return;
    setIsExecuting(true);
    setExecResult(null);
    try {
      let parsedBody: any = undefined;
      if (endpointDetail.config?.method === 'POST' || endpointDetail.config?.method === 'PUT') {
        try {
          parsedBody = JSON.parse(testBody);
        } catch {
          // ignore
        }
      }

      const res = await apiClient<any>('/api/api-dispatcher/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          endpoint_node_id: endpointDetail.id,
          tenant_id: tenantId,
          path_params: testPathParams,
          query_params: testQueryParams,
          body: parsedBody,
        }),
      });
      setExecResult(res);
    } catch (err: any) {
      setExecResult({
        success: false,
        error: err.message || 'Execution error',
      });
    } finally {
      setIsExecuting(false);
    }
  };

  // Open Config Modal
  const handleOpenConfig = async (dsId?: string) => {
    const targetDsId = dsId || endpointDetail?.datasource_id || datasources[0]?.id;
    const targetDs = datasources.find((d) => d.id === targetDsId);
    if (!targetDs) return;

    setConfigStatus(null);
    setOauthConfigured(null);
    // Reset OAuth fields each open — secrets are write-only and not echoed back.
    setConfigClientId('');
    setConfigClientSecret('');
    setConfigRefreshToken('');
    setConfigTokenUrl('');
    setConfigScopes('');
    try {
      const res = await apiClient<any>(`/api/api-dispatcher/connections?tenant_id=${tenantId}&api_datasource_id=${targetDs.id}`);
      if (res?.data) {
        const data = res.data;
        setConfigBaseUrl(data.base_url || '');
        setConfigAuthType(data.auth_type || 'oauth2_bearer');
        setConfigToken(data.auth_config?.token || '');
        setConfigUsername(data.auth_config?.username || '');
        setConfigPassword(data.auth_config?.password || '');
        setConfigApiKey(data.auth_config?.api_key || '');
        setConfigApiKeyHeader(data.auth_config?.header_name || 'X-API-Key');
        setConfigClientId(data.oauth?.client_id || '');
        setConfigTokenUrl(data.oauth?.token_url || '');
        setConfigScopes(data.oauth?.scopes || '');
        setOauthConfigured({
          has_client_secret: !!data.oauth?.has_client_secret,
          has_refresh_token: !!data.oauth?.has_refresh_token,
        });
      } else {
        setConfigBaseUrl(targetDs.config?.default_base_url || '');
        setConfigAuthType(targetDs.config?.default_auth || 'oauth2_bearer');
        setConfigToken('');
        setConfigUsername('');
        setConfigPassword('');
        setConfigApiKey('');
        setConfigApiKeyHeader('X-API-Key');
      }
    } catch {
      setConfigBaseUrl(targetDs.config?.default_base_url || '');
    }
    setIsConfigOpen(true);
  };

  const handleSaveConfig = async () => {
    const targetDsId = endpointDetail?.datasource_id || datasources[0]?.id;
    if (!targetDsId || !tenantId) return;
    setIsSavingConfig(true);
    setConfigStatus(null);
    try {
      const authConfig: Record<string, any> = {};
      if (configAuthType === 'oauth2_bearer') authConfig.token = configToken;
      else if (configAuthType === 'basic_auth') {
        authConfig.username = configUsername;
        authConfig.password = configPassword;
      } else if (configAuthType === 'api_key') {
        authConfig.api_key = configApiKey;
        authConfig.header_name = configApiKeyHeader;
      }

      await apiClient<any>('/api/api-dispatcher/connections', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenant_id: tenantId,
          api_datasource_id: targetDsId,
          base_url: configBaseUrl,
          auth_type: configAuthType,
          auth_config: authConfig,
          // OAuth refresh-flow fields. The backend only updates the encrypted
          // columns when these strings are non-empty, so leaving them blank
          // preserves any existing secrets.
          oauth_client_id: configClientId,
          oauth_client_secret: configClientSecret,
          oauth_refresh_token: configRefreshToken,
          oauth_token_url: configTokenUrl,
          oauth_scopes: configScopes,
          is_active: true,
        }),
      });

      setConfigStatus({ success: true, message: 'Instance credentials saved successfully!' });
      await fetchData();
      setTimeout(() => setIsConfigOpen(false), 800);
    } catch (err: any) {
      setConfigStatus({ success: false, message: err.message || 'Failed to save configuration' });
    } finally {
      setIsSavingConfig(false);
    }
  };

  // Ingest an OpenAPI 3.0 spec from a URL or pasted JSON.
  const handleIngestOpenAPI = async () => {
    setIsIngesting(true);
    setAddApiStatus(null);
    try {
      let body: Record<string, any> = { name: addApiName };
      if (addApiSource === 'url') {
        if (!addApiUrl.trim()) {
          setAddApiStatus({ success: false, message: 'URL is required' });
          setIsIngesting(false);
          return;
        }
        body.url = addApiUrl.trim();
      } else {
        if (!addApiSpec.trim()) {
          setAddApiStatus({ success: false, message: 'Spec JSON is required' });
          setIsIngesting(false);
          return;
        }
        try {
          body.spec = JSON.parse(addApiSpec);
        } catch (e: any) {
          setAddApiStatus({ success: false, message: `Invalid JSON: ${e.message}` });
          setIsIngesting(false);
          return;
        }
      }

      const res = await apiClient<any>('/api/api-dispatcher/ingest-openapi', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = res?.data;
      setAddApiStatus({
        success: true,
        message: data
          ? `Imported ${data.endpoints_created} endpoints across ${data.resources_created} resources (${data.fields_created} fields).`
          : 'Ingestion complete',
        details: data,
      });
      await fetchData();
      setAddApiName('');
      setAddApiUrl('');
      setAddApiSpec('');
      setTimeout(() => setIsAddApiOpen(false), 1500);
    } catch (err: any) {
      setAddApiStatus({ success: false, message: err.message || 'Ingestion failed' });
    } finally {
      setIsIngesting(false);
    }
  };

  const getMethodColor = (method: string) => {
    switch ((method || 'GET').toUpperCase()) {
      case 'GET':
        return '#10B981';
      case 'POST':
        return '#F59E0B';
      case 'PUT':
      case 'PATCH':
        return '#6366F1';
      case 'DELETE':
        return '#EF4444';
      default:
        return '#60A5FA';
    }
  };

  // ═══════════════════════════════════════════════════════════════════════════
  // VIEW: DRILLDOWN DETAIL PAGE
  // ═══════════════════════════════════════════════════════════════════════════
  if (selectedEndpointId) {
    if (detailLoading || !endpointDetail) {
      return (
        <Box sx={{ p: 4, display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', bgcolor: '#0A0C12' }}>
          <CircularProgress />
        </Box>
      );
    }

    const method = (endpointDetail.config?.method || 'GET').toUpperCase();
    const methodColor = getMethodColor(method);

    return (
      <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column', gap: 2.5, bgcolor: '#0A0C12', color: '#F3F4F6', overflowY: 'auto' }}>
        {/* Breadcrumb & Navigation */}
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Button
            startIcon={<ArrowBackIcon />}
            onClick={() => setSearchParams({})}
            sx={{ color: '#9CA3AF', textTransform: 'none', '&:hover': { color: '#fff' } }}
          >
            Back to API Catalog
          </Button>
          <Box sx={{ display: 'flex', gap: 1.5 }}>
            <Button
              variant="outlined"
              startIcon={<SettingsIcon />}
              onClick={() => handleOpenConfig(endpointDetail.datasource_id)}
              sx={{ borderColor: 'rgba(255,255,255,0.2)', color: '#E5E7EB', textTransform: 'none' }}
            >
              Configure Instance Credentials
            </Button>
            <Button
              variant="contained"
              startIcon={isExecuting ? <CircularProgress size={16} color="inherit" /> : <PlayArrowIcon />}
              disabled={isExecuting}
              onClick={() => {
                setActiveTab(3); // Switch to Test Runner Tab
                handleExecute();
              }}
              sx={{ bgcolor: '#6366F1', '&:hover': { bgcolor: '#4F46E5' }, textTransform: 'none', fontWeight: 600 }}
            >
              Execute Endpoint
            </Button>
          </Box>
        </Box>

        {/* Hero Header */}
        <Paper sx={{ p: 3, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 2 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
                <Chip
                  label={method}
                  sx={{
                    fontWeight: 800,
                    fontSize: 13,
                    bgcolor: `${methodColor}22`,
                    color: methodColor,
                    border: `1px solid ${methodColor}55`,
                    height: 28,
                  }}
                />
                <Typography variant="h5" sx={{ fontWeight: 700, color: '#fff', fontFamily: 'monospace' }}>
                  {endpointDetail.node_name}
                </Typography>
                <Chip
                  label={endpointDetail.datasource_name}
                  sx={{ bgcolor: 'rgba(99,102,241,0.15)', color: '#818CF8', fontWeight: 600 }}
                />
                {endpointDetail.resource_name && (
                  <Chip
                    label={`Resource: ${endpointDetail.resource_name}`}
                    sx={{ bgcolor: 'rgba(255,255,255,0.05)', color: '#D1D5DB' }}
                  />
                )}
                {endpointDetail.tenant_base_url ? (
                  <Chip
                    icon={<CheckCircleIcon style={{ fontSize: 14, color: '#10B981' }} />}
                    label="Configured Instance"
                    sx={{ bgcolor: 'rgba(16,185,129,0.15)', color: '#10B981' }}
                  />
                ) : (
                  <Chip label="Gold Copy Default" sx={{ bgcolor: 'rgba(59,130,246,0.15)', color: '#60A5FA' }} />
                )}
              </Box>

              <Typography variant="body2" sx={{ color: '#9CA3AF', mt: 0.5 }}>
                {endpointDetail.description || 'No description provided.'}
              </Typography>

              {/* Endpoint Path Banner */}
              <Box sx={{ mt: 1, p: 1.5, bgcolor: '#0D0F17', borderRadius: 1.5, border: '1px solid rgba(255,255,255,0.06)', fontFamily: 'monospace', fontSize: 13, display: 'flex', alignItems: 'center', gap: 1 }}>
                <span style={{ color: methodColor, fontWeight: 700 }}>{method}</span>
                <span style={{ color: '#60A5FA' }}>
                  {(endpointDetail.tenant_base_url || endpointDetail.datasource_config?.default_base_url || '') +
                    (endpointDetail.config?.path_template || endpointDetail.qualified_path)}
                </span>
              </Box>
            </Box>
          </Box>
        </Paper>

        {/* Tab Navigation */}
        <Box sx={{ borderBottom: 1, borderColor: 'rgba(255,255,255,0.08)' }}>
          <Tabs
            value={activeTab}
            onChange={(_, val) => setActiveTab(val)}
            sx={{
              '& .MuiTab-root': { color: '#9CA3AF', textTransform: 'none', fontWeight: 600, fontSize: 14 },
              '& .Mui-selected': { color: '#6366F1' },
              '& .MuiTabs-indicator': { backgroundColor: '#6366F1' },
            }}
          >
            <Tab label="📋 Overview & Contract" />
            <Tab label={`🔹 Schema & Payload Fields (${fields.length})`} />
            <Tab label="🌐 End-to-End Lineage" />
            <Tab label="▶️ Live Test Runner & Response" />
            <Tab label={`📜 Recent Calls (${recentCalls.length})`} />
          </Tabs>
        </Box>

        {/* TAB 0: OVERVIEW & CONTRACT */}
        {activeTab === 0 && (
          <Grid container spacing={3}>
            <Grid item xs={12} md={7}>
              <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                  API Contract & Specifications
                </Typography>
                <Divider sx={{ borderColor: 'rgba(255,255,255,0.08)' }} />
                <Box sx={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: 1.5, fontSize: 13 }}>
                  <Typography sx={{ color: '#9CA3AF' }}>HTTP Method:</Typography>
                  <Typography sx={{ fontFamily: 'monospace', fontWeight: 700, color: methodColor }}>{method}</Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Path Template:</Typography>
                  <Typography sx={{ fontFamily: 'monospace', color: '#60A5FA' }}>{endpointDetail.config?.path_template || endpointDetail.qualified_path}</Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Response Root:</Typography>
                  <Typography sx={{ fontFamily: 'monospace', color: '#F59E0B' }}>{endpointDetail.config?.response_root || 'Root Object ($)'}</Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Qualified Path:</Typography>
                  <Typography sx={{ fontFamily: 'monospace', color: '#9CA3AF' }}>{endpointDetail.qualified_path}</Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Parent Resource:</Typography>
                  <Typography sx={{ color: '#E5E7EB' }}>{endpointDetail.resource_name || 'None'}</Typography>
                </Box>
              </Paper>
            </Grid>

            <Grid item xs={12} md={5}>
              <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                  Datasource & Connection
                </Typography>
                <Divider sx={{ borderColor: 'rgba(255,255,255,0.08)' }} />
                <Box sx={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: 1.5, fontSize: 13 }}>
                  <Typography sx={{ color: '#9CA3AF' }}>Service Name:</Typography>
                  <Typography sx={{ fontWeight: 600, color: '#fff' }}>{endpointDetail.datasource_name}</Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Service Type:</Typography>
                  <Typography sx={{ color: '#E5E7EB' }}>{endpointDetail.datasource_config?.service_type || 'REST API'}</Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Active Instance:</Typography>
                  <Typography sx={{ fontFamily: 'monospace', color: '#10B981' }}>
                    {endpointDetail.tenant_base_url || endpointDetail.datasource_config?.default_base_url || 'Not configured'}
                  </Typography>

                  <Typography sx={{ color: '#9CA3AF' }}>Auth Type:</Typography>
                  <Typography sx={{ color: '#E5E7EB' }}>{endpointDetail.tenant_auth_type || endpointDetail.datasource_config?.default_auth || 'oauth2_bearer'}</Typography>
                </Box>
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<SettingsIcon />}
                  onClick={() => handleOpenConfig(endpointDetail.datasource_id)}
                  sx={{ mt: 1, borderColor: 'rgba(255,255,255,0.15)', color: '#D1D5DB', textTransform: 'none' }}
                >
                  Edit Instance Credentials
                </Button>
              </Paper>
            </Grid>
          </Grid>
        )}

        {/* TAB 1: SCHEMA & PAYLOAD FIELDS */}
        {activeTab === 1 && (
          <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
              <Box>
                <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                  Payload Attributes &amp; Schema ({fields.length})
                </Typography>
                <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                  Map request/response fields to authoritative Semantic Terms in your catalog to enable data federation and lineage.
                </Typography>
              </Box>
              <Button
                variant="contained"
                size="small"
                startIcon={<AddIcon />}
                onClick={handleOpenAddField}
                sx={{ bgcolor: '#10B981', '&:hover': { bgcolor: '#059669' }, textTransform: 'none', fontWeight: 600 }}
              >
                Add Payload Field
              </Button>
            </Box>

            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ '& th': { color: '#9CA3AF', borderColor: 'rgba(255,255,255,0.08)', fontWeight: 600 } }}>
                    <TableCell>Field Name</TableCell>
                    <TableCell>Data Type</TableCell>
                    <TableCell>JSON Path</TableCell>
                    <TableCell>Key Type</TableCell>
                    <TableCell>Mapped Semantic Term</TableCell>
                    <TableCell>Description</TableCell>
                    <TableCell align="right">Mapping Action</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {fields.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} sx={{ textAlign: 'center', py: 4, color: '#6B7280' }}>
                        No payload fields registered for this endpoint. Click "Add Payload Field" to define one.
                      </TableCell>
                    </TableRow>
                  ) : (
                    fields.map((f) => (
                      <TableRow key={f.id} sx={{ '& td': { borderColor: 'rgba(255,255,255,0.05)', color: '#E5E7EB' } }}>
                        <TableCell sx={{ fontFamily: 'monospace', fontWeight: 600, color: '#60A5FA' }}>
                          🔹 {f.node_name}
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={f.properties?.data_type || 'varchar'}
                            size="small"
                            sx={{ height: 20, fontSize: 10, bgcolor: 'rgba(99,102,241,0.15)', color: '#818CF8' }}
                          />
                        </TableCell>
                        <TableCell sx={{ fontFamily: 'monospace', color: '#9CA3AF' }}>
                          {f.config?.json_path || `$.${f.node_name}`}
                        </TableCell>
                        <TableCell>
                          {f.properties?.is_primary_key ? (
                            <Chip label="PRIMARY KEY" size="small" sx={{ height: 18, fontSize: 9, bgcolor: 'rgba(245,158,11,0.2)', color: '#F59E0B' }} />
                          ) : (
                            <span style={{ color: '#6B7280', fontSize: 11 }}>Attribute</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {f.mapped_semantic_term_name ? (
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Chip
                                icon={<LinkIcon style={{ fontSize: 12, color: '#10B981' }} />}
                                label={`🧠 ${f.mapped_semantic_term_name}`}
                                size="small"
                                onClick={() => navigate(`/core/glossary?id=${f.mapped_semantic_term_id}`)}
                                sx={{
                                  height: 24,
                                  fontSize: 11,
                                  fontWeight: 600,
                                  bgcolor: 'rgba(16,185,129,0.15)',
                                  color: '#10B981',
                                  cursor: 'pointer',
                                  border: '1px solid rgba(16,185,129,0.3)',
                                  '&:hover': { bgcolor: 'rgba(16,185,129,0.25)' },
                                }}
                              />
                              <Tooltip title="Unlink Semantic Term">
                                <IconButton size="small" onClick={() => handleUnmapField(f)} sx={{ color: '#EF4444', p: 0.5 }}>
                                  <LinkOffIcon style={{ fontSize: 14 }} />
                                </IconButton>
                              </Tooltip>
                            </Box>
                          ) : (
                            <span style={{ color: '#6B7280', fontSize: 11, fontStyle: 'italic' }}>Unmapped</span>
                          )}
                        </TableCell>
                        <TableCell sx={{ color: '#9CA3AF', fontSize: 12 }}>
                          {f.description || '-'}
                        </TableCell>
                        <TableCell align="right">
                          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 1 }}>
                            <Button
                              size="small"
                              variant="outlined"
                              startIcon={<LinkIcon />}
                              onClick={() => handleOpenMapModal(f)}
                              sx={{
                                borderColor: f.mapped_semantic_term_name ? 'rgba(99,102,241,0.4)' : 'rgba(16,185,129,0.4)',
                                color: f.mapped_semantic_term_name ? '#A5B4FC' : '#10B981',
                                textTransform: 'none',
                                fontSize: 11,
                                py: 0.2,
                              }}
                            >
                              {f.mapped_semantic_term_name ? 'Change Mapping' : 'Map Semantic Term'}
                            </Button>
                            <Tooltip title="Delete Field">
                              <IconButton size="small" onClick={() => handleDeleteField(f.id)} sx={{ color: '#6B7280', '&:hover': { color: '#EF4444' } }}>
                                <DeleteOutlineIcon style={{ fontSize: 16 }} />
                              </IconButton>
                            </Tooltip>
                          </Box>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        )}

        {/* TAB 2: END-TO-END LINEAGE */}
        {activeTab === 2 && (
          <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Box sx={{ mb: 2 }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                End-to-End Semantic &amp; Technical Lineage
              </Typography>
              <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                Upstream Business Terms ➔ Semantic Terms ➔ API Endpoint ➔ Payload Attributes
              </Typography>
            </Box>

            {lineage && lineage.nodes?.length > 0 ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-around', p: 3, bgcolor: '#0D0F17', borderRadius: 2, border: '1px solid rgba(255,255,255,0.06)', minHeight: 220 }}>
                  {/* Business Terms Column */}
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, minWidth: 200 }}>
                    <Typography variant="caption" sx={{ color: '#818CF8', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
                      💼 Business Glossary
                    </Typography>
                    {lineage.nodes.filter(n => n.data.type === 'business_term').length === 0 ? (
                      <Typography variant="caption" sx={{ color: '#6B7280', fontStyle: 'italic' }}>None linked</Typography>
                    ) : (
                      lineage.nodes.filter(n => n.data.type === 'business_term').map(n => (
                        <Box key={n.id} sx={{ p: 1.5, bgcolor: 'rgba(99,102,241,0.12)', border: '1px solid #6366F144', borderRadius: 1.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 600, color: '#A5B4FC' }}>
                            💼 {n.data.label}
                          </Typography>
                        </Box>
                      ))
                    )}
                  </Box>

                  <Typography sx={{ color: '#4B5563', fontSize: 20, mt: 4 }}>➔</Typography>

                  {/* Semantic Terms Column */}
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, minWidth: 200 }}>
                    <Typography variant="caption" sx={{ color: '#10B981', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
                      🧠 Semantic Terms
                    </Typography>
                    {lineage.nodes.filter(n => n.data.type === 'semantic_term').length === 0 ? (
                      <Typography variant="caption" sx={{ color: '#6B7280', fontStyle: 'italic' }}>None linked</Typography>
                    ) : (
                      lineage.nodes.filter(n => n.data.type === 'semantic_term').map(n => (
                        <Box key={n.id} sx={{ p: 1.5, bgcolor: 'rgba(16,185,129,0.12)', border: '1px solid #10B98144', borderRadius: 1.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 600, color: '#6EE7B7' }}>
                            🧠 {n.data.label}
                          </Typography>
                        </Box>
                      ))
                    )}
                  </Box>

                  <Typography sx={{ color: '#4B5563', fontSize: 20, mt: 4 }}>➔</Typography>

                  {/* Endpoint Center */}
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, minWidth: 220 }}>
                    <Typography variant="caption" sx={{ color: '#F59E0B', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
                      ⚡ API Endpoint
                    </Typography>
                    <Box sx={{ p: 1.5, bgcolor: 'rgba(245,158,11,0.12)', border: '1px solid #F59E0B44', borderRadius: 1.5 }}>
                      <Typography variant="body2" sx={{ fontWeight: 700, color: '#FCD34D', fontFamily: 'monospace' }}>
                        ⚡ {endpointDetail.node_name}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#9CA3AF', display: 'block' }}>
                        {endpointDetail.datasource_name}
                      </Typography>
                    </Box>
                  </Box>

                  <Typography sx={{ color: '#4B5563', fontSize: 20, mt: 4 }}>➔</Typography>

                  {/* Payload Fields Column */}
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 200 }}>
                    <Typography variant="caption" sx={{ color: '#60A5FA', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 1 }}>
                      🔹 Payload Attributes
                    </Typography>
                    {lineage.nodes.filter(n => n.data.type === 'api_field').map(n => (
                      <Box key={n.id} sx={{ p: 1, bgcolor: 'rgba(59,130,246,0.12)', border: '1px solid #3B82F633', borderRadius: 1 }}>
                        <Typography variant="caption" sx={{ fontWeight: 600, color: '#93C5FD', fontFamily: 'monospace' }}>
                          🔹 {n.data.label}
                        </Typography>
                      </Box>
                    ))}
                  </Box>
                </Box>
              </Box>
            ) : (
              <Typography sx={{ color: '#6B7280', fontStyle: 'italic', p: 3 }}>
                No active lineage edges found for this endpoint. Map fields in the "Schema &amp; Payload Fields" tab to view lineage.
              </Typography>
            )}
          </Paper>
        )}

        {/* TAB 3: LIVE TEST RUNNER & RESPONSE */}
        {activeTab === 3 && (
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                    Request Parameters &amp; Body
                  </Typography>
                  <Button
                    variant="contained"
                    startIcon={isExecuting ? <CircularProgress size={16} color="inherit" /> : <PlayArrowIcon />}
                    disabled={isExecuting}
                    onClick={handleExecute}
                    sx={{ bgcolor: '#6366F1', '&:hover': { bgcolor: '#4F46E5' }, textTransform: 'none', fontWeight: 600 }}
                  >
                    {isExecuting ? 'Sending...' : 'Send Request'}
                  </Button>
                </Box>

                <Divider sx={{ borderColor: 'rgba(255,255,255,0.08)' }} />

                {(method === 'POST' || method === 'PUT') && (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <Typography variant="caption" sx={{ color: '#9CA3AF', fontWeight: 600 }}>
                      JSON Request Body:
                    </Typography>
                    <TextField
                      multiline
                      rows={8}
                      value={testBody}
                      onChange={(e) => setTestBody(e.target.value)}
                      fullWidth
                      sx={{
                        bgcolor: '#0D0F17',
                        borderRadius: 1,
                        '& textarea': { color: '#10B981', fontFamily: 'monospace', fontSize: 12 },
                      }}
                    />
                  </Box>
                )}

                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Typography variant="caption" sx={{ color: '#9CA3AF', fontWeight: 600 }}>
                    Target Full URL:
                  </Typography>
                  <Box sx={{ p: 1.5, bgcolor: '#0D0F17', borderRadius: 1, border: '1px solid rgba(255,255,255,0.06)', fontFamily: 'monospace', fontSize: 12, color: '#60A5FA' }}>
                    {(endpointDetail.tenant_base_url || endpointDetail.datasource_config?.default_base_url || '') +
                      (endpointDetail.config?.path_template || endpointDetail.qualified_path)}
                  </Box>
                </Box>
              </Paper>
            </Grid>

            <Grid item xs={12} md={6}>
              <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                    Response Explorer
                  </Typography>
                  {execResult && (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Chip
                        label={execResult.status_code ? `${execResult.status_code} ${execResult.success ? 'OK' : 'Error'}` : execResult.success ? '200 OK' : 'Failed'}
                        size="small"
                        sx={{
                          fontWeight: 700,
                          bgcolor: execResult.success ? 'rgba(16,185,129,0.2)' : 'rgba(239,68,68,0.2)',
                          color: execResult.success ? '#10B981' : '#EF4444',
                        }}
                      />
                      <Chip
                        label={`${execResult.duration_ms || 0}ms`}
                        size="small"
                        sx={{ bgcolor: 'rgba(255,255,255,0.08)', color: '#9CA3AF' }}
                      />
                    </Box>
                  )}
                </Box>

                <Divider sx={{ borderColor: 'rgba(255,255,255,0.08)' }} />

                {execResult ? (
                  <Box sx={{ p: 2, bgcolor: '#0D0F17', borderRadius: 1.5, border: `1px solid ${execResult.success ? '#10B98144' : '#EF444444'}`, maxHeight: 420, overflowY: 'auto' }}>
                    <pre style={{ margin: 0, fontSize: 12, fontFamily: 'monospace', color: execResult.success ? '#10B981' : '#EF4444' }}>
                      {JSON.stringify(execResult.records || execResult.raw_response || execResult, null, 2)}
                    </pre>
                  </Box>
                ) : (
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 260, color: '#6B7280' }}>
                    Click "Send Request" to execute call against the tenant API
                  </Box>
                )}
              </Paper>
            </Grid>
          </Grid>
        )}

        {/* TAB 4: RECENT CALLS (AUDIT) */}
        {activeTab === 4 && (
          <Paper sx={{ p: 2.5, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
              <Box>
                <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
                  Recent Audit Trail
                </Typography>
                <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                  Persisted fire-and-forget record of every dispatch this endpoint has made.
                </Typography>
              </Box>
              <Button
                size="small"
                variant="outlined"
                onClick={() => reloadAudit(selectedEndpointId)}
                sx={{ borderColor: 'rgba(255,255,255,0.15)', color: '#D1D5DB', textTransform: 'none' }}
              >
                Refresh
              </Button>
            </Box>

            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ '& th': { color: '#9CA3AF', borderColor: 'rgba(255,255,255,0.08)', fontWeight: 600 } }}>
                    <TableCell>When</TableCell>
                    <TableCell>Method</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Latency</TableCell>
                    <TableCell>Records</TableCell>
                    <TableCell>Error</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {recentCallsLoading ? (
                    <TableRow><TableCell colSpan={6} sx={{ textAlign: 'center', py: 3, color: '#9CA3AF' }}>Loading…</TableCell></TableRow>
                  ) : recentCalls.length === 0 ? (
                    <TableRow><TableCell colSpan={6} sx={{ textAlign: 'center', py: 3, color: '#6B7280' }}>No audit entries yet — click "Send Request" on the Live Test Runner tab to record one.</TableCell></TableRow>
                  ) : (
                    recentCalls.map((row) => (
                      <TableRow key={row.id} sx={{ '& td': { borderColor: 'rgba(255,255,255,0.05)', color: '#E5E7EB' } }}>
                        <TableCell sx={{ fontFamily: 'monospace', fontSize: 11, color: '#9CA3AF' }}>
                          {new Date(row.created_at).toLocaleString()}
                        </TableCell>
                        <TableCell>
                          <Chip label={row.method} size="small" sx={{ height: 18, fontSize: 10, fontWeight: 700, bgcolor: `${getMethodColor(row.method)}22`, color: getMethodColor(row.method), border: `1px solid ${getMethodColor(row.method)}44` }} />
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={row.status_code ? String(row.status_code) : '—'}
                            size="small"
                            sx={{
                              height: 18,
                              fontSize: 10,
                              fontWeight: 700,
                              bgcolor: row.success ? 'rgba(16,185,129,0.15)' : 'rgba(239,68,68,0.15)',
                              color: row.success ? '#10B981' : '#EF4444',
                            }}
                          />
                        </TableCell>
                        <TableCell sx={{ fontFamily: 'monospace', color: '#60A5FA' }}>{row.duration_ms}ms</TableCell>
                        <TableCell sx={{ color: '#9CA3AF' }}>{row.record_count ?? 0}</TableCell>
                        <TableCell sx={{ color: '#EF4444', maxWidth: 320, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={row.error}>
                          {row.error || ''}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        )}

        {/* MAP SEMANTIC TERM MODAL */}
        <Dialog
          open={isMapModalOpen}
          onClose={() => setIsMapModalOpen(false)}
          maxWidth="sm"
          fullWidth
          PaperProps={{
            sx: { bgcolor: '#13161E', color: '#fff', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 2 },
          }}
        >
          <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <LinkIcon sx={{ color: '#10B981' }} />
            Map Field <strong>"{selectedFieldForMap?.node_name}"</strong> to Semantic Term
          </DialogTitle>
          <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '10px !important' }}>
            <Typography variant="body2" sx={{ color: '#9CA3AF' }}>
              Select a governed Semantic Term from your catalog to map this API field attribute.
            </Typography>

            <Autocomplete
              options={availableTerms}
              getOptionLabel={(opt) => `${opt.node_name} (${opt.data_type || 'varchar'})`}
              value={chosenTerm}
              onChange={(_, val) => setChosenTerm(val)}
              loading={termsLoading}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Search Semantic Terms"
                  placeholder="e.g. Customer ID, Revenue, Incident State..."
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
              )}
              renderOption={(props, opt) => (
                <li {...props} style={{ backgroundColor: '#13161E', color: '#fff', borderBottom: '1px solid rgba(255,255,255,0.05)', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', padding: 8 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%', justifyContent: 'space-between' }}>
                    <Typography variant="body2" sx={{ fontWeight: 600, color: '#10B981' }}>
                      🧠 {opt.node_name}
                    </Typography>
                    <Chip label={opt.data_type || 'varchar'} size="small" sx={{ height: 18, fontSize: 9, bgcolor: 'rgba(99,102,241,0.15)', color: '#818CF8' }} />
                  </Box>
                  {opt.description && (
                    <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                      {opt.description}
                    </Typography>
                  )}
                </li>
              )}
            />

            {chosenTerm && (
              <Box sx={{ p: 1.5, bgcolor: '#0D0F17', borderRadius: 1, border: '1px solid #10B98133' }}>
                <Typography variant="caption" sx={{ color: '#9CA3AF', display: 'block' }}>Selected Semantic Concept:</Typography>
                <Typography variant="subtitle2" sx={{ color: '#10B981', fontWeight: 700 }}>
                  🧠 {chosenTerm.node_name}
                </Typography>
                <Typography variant="caption" sx={{ color: '#D1D5DB' }}>
                  {chosenTerm.description || 'Governed business concept definition'}
                </Typography>
              </Box>
            )}
          </DialogContent>
          <DialogActions sx={{ p: 2 }}>
            <Button onClick={() => setIsMapModalOpen(false)} sx={{ color: '#9CA3AF' }}>
              Cancel
            </Button>
            {selectedFieldForMap?.mapped_semantic_term_name && (
              <Button onClick={() => { setChosenTerm(null); handleSaveMapping(); }} sx={{ color: '#EF4444' }}>
                Remove Mapping
              </Button>
            )}
            <Button
              variant="contained"
              disabled={isSavingMapping}
              onClick={handleSaveMapping}
              sx={{ bgcolor: '#10B981', '&:hover': { bgcolor: '#059669' } }}
            >
              {isSavingMapping ? 'Saving...' : 'Save Mapping'}
            </Button>
          </DialogActions>
        </Dialog>

        {/* ADD PAYLOAD FIELD MODAL */}
        <Dialog
          open={isAddFieldOpen}
          onClose={() => setIsAddFieldOpen(false)}
          maxWidth="sm"
          fullWidth
          PaperProps={{
            sx: { bgcolor: '#13161E', color: '#fff', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 2 },
          }}
        >
          <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <AddIcon sx={{ color: '#10B981' }} />
            Add Payload / Response Field
          </DialogTitle>
          <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '10px !important' }}>
            <TextField
              label="Field Name"
              placeholder="e.g. AccountId, Status, Email, TotalAmount"
              value={newFieldName}
              onChange={(e) => {
                setNewFieldName(e.target.value);
                if (!newFieldJsonPath) setNewFieldJsonPath(`$.${e.target.value}`);
              }}
              fullWidth
              size="small"
              sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
            />

            <Box sx={{ display: 'flex', gap: 2 }}>
              <FormControl fullWidth size="small" sx={{ bgcolor: '#1E2130', borderRadius: 1 }}>
                <InputLabel sx={{ color: '#9CA3AF' }}>Data Type</InputLabel>
                <Select
                  value={newFieldDataType}
                  label="Data Type"
                  onChange={(e) => setNewFieldDataType(e.target.value)}
                  sx={{ color: '#fff' }}
                >
                  <MenuItem value="varchar">varchar (String)</MenuItem>
                  <MenuItem value="numeric">numeric (Decimal / Float)</MenuItem>
                  <MenuItem value="integer">integer (Number)</MenuItem>
                  <MenuItem value="boolean">boolean</MenuItem>
                  <MenuItem value="timestamp">timestamp (DateTime)</MenuItem>
                  <MenuItem value="uuid">uuid</MenuItem>
                  <MenuItem value="jsonb">jsonb (Object / Array)</MenuItem>
                </Select>
              </FormControl>

              <TextField
                label="JSON Path"
                placeholder="$.FieldName"
                value={newFieldJsonPath}
                onChange={(e) => setNewFieldJsonPath(e.target.value)}
                fullWidth
                size="small"
                sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
              />
            </Box>

            <FormControlLabel
              control={<Switch checked={newFieldIsPk} onChange={(e) => setNewFieldIsPk(e.target.checked)} color="warning" />}
              label={<Typography variant="body2" sx={{ color: '#D1D5DB' }}>Is Primary Key / Identifier</Typography>}
            />

            <TextField
              label="Description"
              placeholder="Field description or contract notes..."
              value={newFieldDesc}
              onChange={(e) => setNewFieldDesc(e.target.value)}
              fullWidth
              multiline
              rows={2}
              size="small"
              sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
            />

            <Autocomplete
              options={availableTerms}
              getOptionLabel={(opt) => `${opt.node_name} (${opt.data_type || 'varchar'})`}
              value={newFieldTerm}
              onChange={(_, val) => setNewFieldTerm(val)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Map to Semantic Term (Optional)"
                  placeholder="Select term..."
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
              )}
            />
          </DialogContent>
          <DialogActions sx={{ p: 2 }}>
            <Button onClick={() => setIsAddFieldOpen(false)} sx={{ color: '#9CA3AF' }}>
              Cancel
            </Button>
            <Button
              variant="contained"
              disabled={isSavingField || !newFieldName}
              onClick={handleSaveNewField}
              sx={{ bgcolor: '#10B981', '&:hover': { bgcolor: '#059669' } }}
            >
              {isSavingField ? 'Creating...' : 'Create Field'}
            </Button>
          </DialogActions>
        </Dialog>

        {/* CONFIGURE INSTANCE CREDENTIALS MODAL */}
        <Dialog
          open={isConfigOpen}
          onClose={() => setIsConfigOpen(false)}
          maxWidth="sm"
          fullWidth
          PaperProps={{
            sx: { bgcolor: '#13161E', color: '#fff', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 2 },
          }}
        >
          <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <KeyIcon sx={{ color: '#F59E0B' }} />
            Configure Instance Credentials
          </DialogTitle>
          <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '10px !important' }}>
            <Typography variant="body2" sx={{ color: '#9CA3AF' }}>
              Override the gold-copy default base URL and credentials for this tenant.
              Secrets are encrypted at rest with the platform's AES-256-GCM key.
            </Typography>

            <TextField
              label="Base URL"
              placeholder="https://acme.my.salesforce.com"
              value={configBaseUrl}
              onChange={(e) => setConfigBaseUrl(e.target.value)}
              fullWidth
              size="small"
              sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
            />

            <FormControl fullWidth size="small" sx={{ bgcolor: '#1E2130', borderRadius: 1 }}>
              <InputLabel sx={{ color: '#9CA3AF' }}>Authentication Type</InputLabel>
              <Select
                value={configAuthType}
                label="Authentication Type"
                onChange={(e) => setConfigAuthType(e.target.value as string)}
                sx={{ color: '#fff' }}
              >
                <MenuItem value="oauth2_bearer">OAuth 2.0 Bearer (with refresh)</MenuItem>
                <MenuItem value="basic_auth">HTTP Basic Auth</MenuItem>
                <MenuItem value="api_key">API Key (custom header)</MenuItem>
                <MenuItem value="none">No Auth</MenuItem>
              </Select>
            </FormControl>

            {configAuthType === 'oauth2_bearer' && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, p: 1.5, bgcolor: '#0D0F17', borderRadius: 1, border: '1px solid rgba(255,255,255,0.06)' }}>
                <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                  OAuth 2.0 client credentials. Leave secret fields blank to keep the previously saved value.
                </Typography>
                <TextField
                  label="Client ID"
                  placeholder="3MVG9LH2gXQ5nG2..."
                  value={configClientId}
                  onChange={(e) => setConfigClientId(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <TextField
                  label="Client Secret (leave blank to keep existing)"
                  type="password"
                  autoComplete="new-password"
                  placeholder={oauthConfigured?.has_client_secret ? '•••••••• (saved)' : 'Enter client secret'}
                  value={configClientSecret}
                  onChange={(e) => setConfigClientSecret(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <TextField
                  label="Refresh Token (leave blank to keep existing)"
                  type="password"
                  autoComplete="new-password"
                  placeholder={oauthConfigured?.has_refresh_token ? '•••••••• (saved)' : '5Aep861...'}
                  value={configRefreshToken}
                  onChange={(e) => setConfigRefreshToken(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <TextField
                  label="Token URL"
                  placeholder="https://login.salesforce.com/services/oauth2/token"
                  value={configTokenUrl}
                  onChange={(e) => setConfigTokenUrl(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <TextField
                  label="Scopes (space-separated, optional)"
                  placeholder="api refresh_token"
                  value={configScopes}
                  onChange={(e) => setConfigScopes(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <Typography variant="caption" sx={{ color: '#F59E0B' }}>
                  Static fallback bearer (used when refresh fails or no refresh_token is set):
                </Typography>
                <TextField
                  label="Static Bearer Token (optional)"
                  type="password"
                  autoComplete="new-password"
                  placeholder="ya29..."
                  value={configToken}
                  onChange={(e) => setConfigToken(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
              </Box>
            )}

            {configAuthType === 'basic_auth' && (
              <Box sx={{ display: 'flex', gap: 1.5 }}>
                <TextField
                  label="Username"
                  value={configUsername}
                  onChange={(e) => setConfigUsername(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <TextField
                  label="Password"
                  type="password"
                  autoComplete="new-password"
                  value={configPassword}
                  onChange={(e) => setConfigPassword(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
              </Box>
            )}

            {configAuthType === 'api_key' && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <TextField
                  label="Header Name"
                  placeholder="X-API-Key"
                  value={configApiKeyHeader}
                  onChange={(e) => setConfigApiKeyHeader(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
                <TextField
                  label="API Key"
                  type="password"
                  autoComplete="new-password"
                  value={configApiKey}
                  onChange={(e) => setConfigApiKey(e.target.value)}
                  fullWidth
                  size="small"
                  sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
                />
              </Box>
            )}

            {configStatus && (
              <Alert
                severity={configStatus.success ? 'success' : 'error'}
                sx={{ bgcolor: configStatus.success ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)', color: configStatus.success ? '#10B981' : '#EF4444' }}
              >
                {configStatus.message}
              </Alert>
            )}
          </DialogContent>
          <DialogActions sx={{ p: 2 }}>
            <Button onClick={() => setIsConfigOpen(false)} sx={{ color: '#9CA3AF' }}>
              Cancel
            </Button>
            <Button
              variant="contained"
              disabled={isSavingConfig || !configBaseUrl}
              onClick={handleSaveConfig}
              sx={{ bgcolor: '#6366F1', '&:hover': { bgcolor: '#4F46E5' } }}
            >
              {isSavingConfig ? 'Saving...' : 'Save Credentials'}
            </Button>
          </DialogActions>
        </Dialog>
      </Box>
    );
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // VIEW: SUMMARY LEVEL CATALOG WITH FACETED FILTERING
  // ═══════════════════════════════════════════════════════════════════════════
  const paginatedEndpoints = filteredEndpoints.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage);

  return (
    <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column', gap: 3, bgcolor: '#0A0C12', color: '#F3F4F6', overflowY: 'auto' }}>
      {/* Top Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.08)', pb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <LanguageIcon sx={{ fontSize: 36, color: '#6366F1' }} />
          <Box>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#fff' }}>
              API Inventory &amp; Service Catalog
            </Typography>
            <Typography variant="body2" sx={{ color: '#9CA3AF' }}>
              Catalog and federate API endpoints, payload schemas, and semantic terms for tenant:{' '}
              <strong style={{ color: '#60A5FA' }}>{tenant?.name || 'Gold Copy / Core'}</strong>
            </Typography>
          </Box>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Button
            variant="contained"
            startIcon={<SettingsIcon />}
            onClick={() => handleOpenConfig()}
            sx={{ bgcolor: '#3B82F6', '&:hover': { bgcolor: '#2563EB' }, textTransform: 'none', fontWeight: 600 }}
          >
            Configure Credentials
          </Button>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchData}
            sx={{ borderColor: 'rgba(255,255,255,0.15)', color: '#D1D5DB' }}
          >
            Refresh
          </Button>
        </Box>
      </Box>

      {/* Summary KPI Cards */}
      <Grid container spacing={2}>
        <Grid item xs={12} sm={6} md={3}>
          <Paper sx={{ p: 2, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Typography variant="caption" sx={{ color: '#9CA3AF', textTransform: 'uppercase', letterSpacing: 1 }}>
              🌐 API Services
            </Typography>
            <Typography variant="h4" sx={{ fontWeight: 800, color: '#fff', mt: 0.5 }}>
              {datasources.length}
            </Typography>
          </Paper>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Paper sx={{ p: 2, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Typography variant="caption" sx={{ color: '#9CA3AF', textTransform: 'uppercase', letterSpacing: 1 }}>
              ⚡ Total Endpoints
            </Typography>
            <Typography variant="h4" sx={{ fontWeight: 800, color: '#6366F1', mt: 0.5 }}>
              {endpoints.length}
            </Typography>
          </Paper>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Paper sx={{ p: 2, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Typography variant="caption" sx={{ color: '#9CA3AF', textTransform: 'uppercase', letterSpacing: 1 }}>
              🧠 Mapped Semantic Concepts
            </Typography>
            <Typography variant="h4" sx={{ fontWeight: 800, color: '#10B981', mt: 0.5 }}>
              {totalTermsCount}
            </Typography>
          </Paper>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Paper sx={{ p: 2, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
            <Typography variant="caption" sx={{ color: '#9CA3AF', textTransform: 'uppercase', letterSpacing: 1 }}>
              🔒 Configured Connections
            </Typography>
            <Typography variant="h4" sx={{ fontWeight: 800, color: '#F59E0B', mt: 0.5 }}>
              {configuredCount} / {datasources.length}
            </Typography>
          </Paper>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => { setAddApiStatus(null); setIsAddApiOpen(true); }}
            sx={{
              bgcolor: '#6366F1',
              '&:hover': { bgcolor: '#4F46E5' },
              textTransform: 'none',
              fontWeight: 600,
              height: '100%',
              minHeight: 88,
              borderRadius: 2,
            }}
          >
            Add API (OpenAPI 3.0)
          </Button>
        </Grid>
      </Grid>

      {/* Facet Filter & Search Bar */}
      <Paper sx={{ p: 2, bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
          <TextField
            size="small"
            placeholder="Search API endpoints, paths, descriptions..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(0);
            }}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon sx={{ color: '#6B7280' }} />
                </InputAdornment>
              ),
            }}
            sx={{ flex: 1, minWidth: 280, bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
          />

          <FormControl size="small" sx={{ minWidth: 160, bgcolor: '#1E2130', borderRadius: 1 }}>
            <InputLabel sx={{ color: '#9CA3AF' }}>Vendor / Service</InputLabel>
            <Select
              value={selectedVendor}
              label="Vendor / Service"
              onChange={(e) => {
                setSelectedVendor(e.target.value);
                setPage(0);
              }}
              sx={{ color: '#fff' }}
            >
              <MenuItem value="ALL">All Vendors ({endpoints.length})</MenuItem>
              {vendors.map((v) => (
                <MenuItem key={v} value={v}>
                  {v}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <FormControl size="small" sx={{ minWidth: 140, bgcolor: '#1E2130', borderRadius: 1 }}>
            <InputLabel sx={{ color: '#9CA3AF' }}>HTTP Method</InputLabel>
            <Select
              value={selectedMethod}
              label="HTTP Method"
              onChange={(e) => {
                setSelectedMethod(e.target.value);
                setPage(0);
              }}
              sx={{ color: '#fff' }}
            >
              <MenuItem value="ALL">All Actions</MenuItem>
              <MenuItem value="GET">GET (Read)</MenuItem>
              <MenuItem value="POST">POST (Create)</MenuItem>
              <MenuItem value="PUT">PUT (Update)</MenuItem>
              <MenuItem value="DELETE">DELETE (Remove)</MenuItem>
            </Select>
          </FormControl>

          {resources.length > 0 && (
            <FormControl size="small" sx={{ minWidth: 150, bgcolor: '#1E2130', borderRadius: 1 }}>
              <InputLabel sx={{ color: '#9CA3AF' }}>Resource</InputLabel>
              <Select
                value={selectedResource}
                label="Resource"
                onChange={(e) => {
                  setSelectedResource(e.target.value);
                  setPage(0);
                }}
                sx={{ color: '#fff' }}
              >
                <MenuItem value="ALL">All Resources</MenuItem>
                {resources.map((r) => (
                  <MenuItem key={r} value={r}>
                    {r}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}

          <Box sx={{ display: 'flex', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 1 }}>
            <IconButton
              size="small"
              onClick={() => setViewMode('grid')}
              sx={{ color: viewMode === 'grid' ? '#6366F1' : '#6B7280' }}
            >
              <ViewModuleIcon />
            </IconButton>
            <IconButton
              size="small"
              onClick={() => setViewMode('table')}
              sx={{ color: viewMode === 'table' ? '#6366F1' : '#6B7280' }}
            >
              <ViewListIcon />
            </IconButton>
          </Box>
        </Box>
      </Paper>

      {/* Grid View */}
      {viewMode === 'grid' && (
        <Grid container spacing={2}>
          {paginatedEndpoints.map((ep) => {
            const method = (ep.config?.method || (ep.node_name.startsWith('POST') ? 'POST' : 'GET')).toUpperCase();
            const methodColor = getMethodColor(method);

            return (
              <Grid item xs={12} sm={6} md={4} key={ep.id}>
                <Card
                  onClick={() => setSearchParams({ id: ep.id })}
                  sx={{
                    bgcolor: '#13161E',
                    border: '1px solid rgba(255,255,255,0.08)',
                    borderRadius: 2,
                    cursor: 'pointer',
                    transition: 'all 0.2s ease',
                    height: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'space-between',
                    '&:hover': {
                      borderColor: '#6366F1',
                      transform: 'translateY(-2px)',
                      boxShadow: '0 8px 24px rgba(99,102,241,0.15)',
                    },
                  }}
                >
                  <CardContent sx={{ p: 2 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip
                          label={method}
                          size="small"
                          sx={{
                            fontWeight: 800,
                            fontSize: 10,
                            bgcolor: `${methodColor}22`,
                            color: methodColor,
                            border: `1px solid ${methodColor}55`,
                            height: 22,
                          }}
                        />
                        <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#fff', fontFamily: 'monospace' }}>
                          {ep.node_name}
                        </Typography>
                      </Box>
                      <Chip
                        label={ep.datasource_name}
                        size="small"
                        sx={{ height: 20, fontSize: 10, bgcolor: 'rgba(99,102,241,0.15)', color: '#818CF8' }}
                      />
                    </Box>

                    <Typography variant="body2" sx={{ color: '#9CA3AF', fontSize: 12, mb: 2, minHeight: 36 }}>
                      {ep.description || 'No description available'}
                    </Typography>

                    <Box sx={{ p: 1, bgcolor: '#0D0F17', borderRadius: 1, border: '1px solid rgba(255,255,255,0.05)', fontFamily: 'monospace', fontSize: 11, color: '#60A5FA', mb: 1.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {ep.config?.path_template || ep.qualified_path}
                    </Box>

                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pt: 1, borderTop: '1px solid rgba(255,255,255,0.05)' }}>
                      <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                        🔹 {ep.fields_count} Fields
                      </Typography>
                      {ep.semantic_terms_count > 0 ? (
                        <Chip
                          label={`🧠 ${ep.semantic_terms_count} Semantic Terms`}
                          size="small"
                          sx={{ height: 20, fontSize: 10, bgcolor: 'rgba(16,185,129,0.15)', color: '#10B981' }}
                        />
                      ) : (
                        <Typography variant="caption" sx={{ color: '#6B7280' }}>
                          Unmapped
                        </Typography>
                      )}
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            );
          })}
        </Grid>
      )}

      {/* Table View */}
      {viewMode === 'table' && (
        <Paper sx={{ bgcolor: '#13161E', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ '& th': { color: '#9CA3AF', borderColor: 'rgba(255,255,255,0.08)', fontWeight: 600 } }}>
                  <TableCell>Method</TableCell>
                  <TableCell>Endpoint Name</TableCell>
                  <TableCell>Path Template</TableCell>
                  <TableCell>Service / Vendor</TableCell>
                  <TableCell>Resource</TableCell>
                  <TableCell>Fields</TableCell>
                  <TableCell>Semantic Terms</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedEndpoints.map((ep) => {
                  const method = (ep.config?.method || (ep.node_name.startsWith('POST') ? 'POST' : 'GET')).toUpperCase();
                  const methodColor = getMethodColor(method);

                  return (
                    <TableRow
                      key={ep.id}
                      hover
                      onClick={() => setSearchParams({ id: ep.id })}
                      sx={{ cursor: 'pointer', '& td': { borderColor: 'rgba(255,255,255,0.05)', color: '#E5E7EB' } }}
                    >
                      <TableCell>
                        <Chip
                          label={method}
                          size="small"
                          sx={{
                            fontWeight: 800,
                            fontSize: 10,
                            bgcolor: `${methodColor}22`,
                            color: methodColor,
                            border: `1px solid ${methodColor}55`,
                            height: 20,
                          }}
                        />
                      </TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontWeight: 600, color: '#fff' }}>
                        {ep.node_name}
                      </TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', color: '#60A5FA' }}>
                        {ep.config?.path_template || ep.qualified_path}
                      </TableCell>
                      <TableCell>{ep.datasource_name}</TableCell>
                      <TableCell>{ep.resource_name || '-'}</TableCell>
                      <TableCell>🔹 {ep.fields_count}</TableCell>
                      <TableCell>
                        {ep.semantic_terms_count > 0 ? (
                          <Chip label={`🧠 ${ep.semantic_terms_count}`} size="small" sx={{ height: 20, fontSize: 10, bgcolor: 'rgba(16,185,129,0.15)', color: '#10B981' }} />
                        ) : (
                          '-'
                        )}
                      </TableCell>
                      <TableCell align="right">
                        <Button size="small" sx={{ color: '#6366F1', textTransform: 'none' }}>
                          Inspect →
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}

      {/* Pagination */}
      <TablePagination
        component="div"
        count={filteredEndpoints.length}
        page={page}
        onPageChange={(_, newPage) => setPage(newPage)}
        rowsPerPage={rowsPerPage}
        onRowsPerPageChange={(e) => {
          setRowsPerPage(parseInt(e.target.value, 10));
          setPage(0);
        }}
        rowsPerPageOptions={[12, 24, 48, 96]}
        sx={{ color: '#9CA3AF' }}
      />

      {/* ADD API (OPENAPI 3.0 INGESTER) MODAL */}
      <Dialog
        open={isAddApiOpen}
        onClose={() => setIsAddApiOpen(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          sx: { bgcolor: '#13161E', color: '#fff', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 2 },
        }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AddIcon sx={{ color: '#6366F1' }} />
          Add API Datasource from OpenAPI 3.0
        </DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '10px !important' }}>
          <Typography variant="body2" sx={{ color: '#9CA3AF' }}>
            Import a public OpenAPI 3.0 specification. The parser will create
            one API Datasource, one Resource per tag/path-prefix, one Endpoint
            per (path, method), and one Api Field per response/request body
            property. Re-ingesting the same spec updates in place.
          </Typography>

          <TextField
            label="Display Name (optional)"
            placeholder="Inferred from spec.info.title when blank"
            value={addApiName}
            onChange={(e) => setAddApiName(e.target.value)}
            fullWidth
            size="small"
            sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
          />

          <FormControl fullWidth size="small" sx={{ bgcolor: '#1E2130', borderRadius: 1 }}>
            <InputLabel sx={{ color: '#9CA3AF' }}>Source</InputLabel>
            <Select
              value={addApiSource}
              label="Source"
              onChange={(e) => setAddApiSource(e.target.value as 'url' | 'paste')}
              sx={{ color: '#fff' }}
            >
              <MenuItem value="url">Fetch from URL</MenuItem>
              <MenuItem value="paste">Paste JSON</MenuItem>
            </Select>
          </FormControl>

          {addApiSource === 'url' ? (
            <TextField
              label="OpenAPI 3.0 Spec URL"
              placeholder="https://api.example.com/openapi.json"
              value={addApiUrl}
              onChange={(e) => setAddApiUrl(e.target.value)}
              fullWidth
              size="small"
              sx={{ bgcolor: '#1E2130', borderRadius: 1, input: { color: '#fff' } }}
            />
          ) : (
            <TextField
              label="OpenAPI 3.0 JSON"
              placeholder='{ "openapi": "3.0.0", ... }'
              value={addApiSpec}
              onChange={(e) => setAddApiSpec(e.target.value)}
              fullWidth
              multiline
              minRows={8}
              size="small"
              sx={{
                bgcolor: '#1E2130',
                borderRadius: 1,
                '& textarea': { color: '#fff', fontFamily: 'monospace', fontSize: 12 },
              }}
            />
          )}

          {addApiStatus && (
            <Alert
              severity={addApiStatus.success ? 'success' : 'error'}
              sx={{
                bgcolor: addApiStatus.success ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)',
                color: addApiStatus.success ? '#10B981' : '#EF4444',
              }}
            >
              {addApiStatus.message}
            </Alert>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setIsAddApiOpen(false)} sx={{ color: '#9CA3AF' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            disabled={isIngesting}
            onClick={handleIngestOpenAPI}
            sx={{ bgcolor: '#6366F1', '&:hover': { bgcolor: '#4F46E5' } }}
          >
            {isIngesting ? 'Importing…' : 'Import Spec'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ApiInventoryPage;
