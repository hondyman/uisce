import { useState, useEffect, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Typography,
  Box,
  Button,
  AppBar,
  Toolbar,
  Container,
  Grid,
  Chip,
  Paper,
  Stack,
  IconButton,
  Avatar,
  useTheme,
  Menu,
  MenuItem,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import {
  Add as AddIcon,
  Schema as SchemaIcon,
  Category as CategoryIcon,
  Info as InfoIcon,
  Help as HelpIcon,
  Refresh as RefreshIcon,
  Close as CloseIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ViewWeek as ViewWeekIcon,
  ViewAgenda as ViewAgendaIcon,
  Storage as StorageIcon,
  PlayArrow as RunIcon,
  AutoAwesome as AIIcon,
  Search as SearchIcon,
} from '@mui/icons-material';

import { EditBusinessObjectModal } from '../components/BusinessObjectManager/EditBusinessObjectModal';
import BusinessObjectBindingWizard from '../components/BusinessObjectManager/BusinessObjectBindingWizard';
import BOAIAssistantModal from '../components/BusinessObjectManager/BOAIAssistantModal';
import CatalogList from '../components/common/CatalogList';
import { filterBusinessObjectsBySearch } from '../utils/businessObjectSearch';

import { useTenant } from '../contexts/TenantContext';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';

import { getSelectedRegion } from '../lib/region';

interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  is_core?: boolean;
  driver_table_name?: string;
  category?: string;
  history_mode?: string;
  config?: {
    is_active?: boolean;
    fields?: Array<{
      key: string;
      name: string;
      displayName?: string;
      technicalName?: string;
      type: string;
      isCore?: boolean;
    }>;
  };
  is_active?: boolean;
  enable_history?: boolean;
  status?: 'draft' | 'active' | 'deprecated';
  updated_at?: string;
  subtypes?: Record<
    string,
    {
      id: string;
      key: string;
      name: string;
      display_name: string;
      technical_name: string;
      description?: string;
      is_core: boolean;
      config?: {
        inheritedFields?: any[];
        customFields?: any[];
      };
    }
  >;
}

export default function BusinessObjectsPage() {
  const { tenant, datasource } = useTenant();
  const confirm = useConfirm();
  const notification = useNotification();
  const navigate = useNavigate();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || datasource?.alpha_tenant_instance_id || '';
  
  const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const [actionsMenu, setActionsMenu] = useState<{ el: HTMLElement; object: BusinessObject } | null>(null);
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'draft'>('all');
  const [scopeFilter, setScopeFilter] = useState<'all' | 'core' | 'custom'>('all');

  // Validation Rules State
  const [selectedFieldForValidation, _setSelectedFieldForValidation] = useState<any>(null);
  const [fieldValidationModalOpen, setFieldValidationModalOpen] = useState(false);


  // Edit Business Object Modal State
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingObject, setEditingObject] = useState<BusinessObject | null>(null);
  
  // Wizard State (?new=1, e.g. from the /business-objects/new redirect, opens it)
  const [searchParams, setSearchParams] = useSearchParams();
  const [wizardOpen, setWizardOpen] = useState(() => searchParams.get('new') === '1');
  useEffect(() => {
    if (searchParams.has('new')) {
      const next = new URLSearchParams(searchParams);
      next.delete('new');
      setSearchParams(next, { replace: true });
    }
  }, [searchParams, setSearchParams]);
  const [aiModalOpen, setAiModalOpen] = useState(false);


  // Helper to build headers with authentication
  const getAuthHeaders = (additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Region': getSelectedRegion(),
      ...additionalHeaders,
    };
    if (authHeader) {
      headers['Authorization'] = authHeader;
    }
    return headers;
  };

  const getValidationRulesForField = (_fieldKey: string) => {
    // Placeholder for fetching validation rules filtering by field
    return [];
  };

  const sortedBusinessObjects = useMemo(
    () => [...businessObjects].sort((a, b) => (a.display_name ?? a.name ?? '').localeCompare(b.display_name ?? b.name ?? '')),
    [businessObjects],
  );

  // Executive KPI summary metrics
  const metrics = useMemo(() => {
    const total = businessObjects.length;
    const coreCount = businessObjects.filter(b => b.is_core).length;
    const customCount = total - coreCount;
    const activeCount = businessObjects.filter(b => b.is_active).length;
    const boundTables = businessObjects.filter(b => !!b.driver_table_name).length;
    return { total, coreCount, customCount, activeCount, boundTables };
  }, [businessObjects]);

  const fetchBusinessObjects = async () => {
    if (!tenantId) {
      setBusinessObjects([]);
      setError('Please select a tenant');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/business-objects', {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to fetch business objects');
      }

      const data = await response.json();
      // Handle both array response (from main endpoint) and object response (legacy format)
      const dataArray = Array.isArray(data) ? data : Object.entries(data).map(([id, obj]: [string, any]) => ({ ...obj, id }));
      
      let objectsArray = dataArray.map((obj: any) => {
        const id = obj.id;
        const config = obj.config || {};
        
        // Normalize fields logic (simplified for list view)
        const normalizedConfig = {
          ...config,
          fields: (config.entity_fields || []).map((field: any) => ({
             key: field.key,
             name: field.name,
             displayName: field.businessName || field.displayName || field.name,
             technicalName: field.technicalName || field.name,
             type: field.type,
             isCore: field.isCore,
          })),
        };

        const processedSubtypes: Record<string, any> = {};
        if (obj.subtypes) {
            Object.entries(obj.subtypes).forEach(([stId, st]: [string, any]) => {
                processedSubtypes[stId] = { ...st, is_core: st.config?.isCore ?? false };
            });
        }

        return {
          id: id,
          name: obj.name || obj.technical_name || id,
          display_name: obj.display_name || obj.name || obj.technical_name || id,
          description: obj.description,
          config: normalizedConfig,
          subtypes: processedSubtypes,
          is_active: normalizedConfig.is_active !== false,
          is_core: obj.is_core ?? obj.isCore ?? false,
          driver_table_name: obj.driver_table_name || obj.driverTableName || '',
          category: obj.category || 'General',
          history_mode: obj.history_mode || obj.historyMode || '',
          enable_history: obj.enableHistory || obj.enable_history || false,
          updated_at: obj.updated_at,
          parent_id: (obj.parentId && typeof obj.parentId === 'object' && 'Valid' in obj.parentId) 
            ? (obj.parentId.Valid ? obj.parentId.String : null)
            : (obj.parentId || null),
        };
      }).filter(obj => !obj.parent_id); // Filter out subtypes - only show parent business objects

      setBusinessObjects(objectsArray);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch business objects';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  };


  useEffect(() => {
    if (tenantId) {
      fetchBusinessObjects();
    }
  }, [tenantId]);

  const _handleToggleObjectStatus = (object: BusinessObject) => {
    // Optimistic Update
    const newStatus = !object.is_active;
    const updatedObject = { ...object, is_active: newStatus, config: { ...object.config, is_active: newStatus } };
    setBusinessObjects(prev => prev.map(obj => obj.id === object.id ? updatedObject : obj));
    
    // In a real scenario, fire API call here.
    notification.success(`Business Object "${object.display_name}" is now ${newStatus ? 'Active' : 'Draft'}`);
  };

  const handleCreateObject = () => {
    setWizardOpen(true);
  };
  
  const handleWizardSave = (boId: string) => {
    notification.success('Business Object created successfully!');
    fetchBusinessObjects(); // Refresh the list
    // Optionally navigate to the new object
    navigate(`/business-objects/${boId}`);
  };

  const handleViewDetails = (object: BusinessObject) => {
    navigate(`/business-objects/${object.id}`, { state: { object } });
  };

  const theme = useTheme();
  // const _isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const handleEditObject = (object: BusinessObject, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation();
    }
    setEditingObject(object);
    setEditModalOpen(true);
  };

  const handleDeleteObject = async (object: BusinessObject, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation();
    }
    
    const confirmed = await confirm({
      title: 'Delete Business Object',
      description: `Are you sure you want to delete "${object.display_name}"? This action cannot be undone.`,
      confirmText: 'Delete',
      cancelText: 'Cancel',
    });

    if (!confirmed) return;

    try {
      const response = await fetch(`/api/business-objects/${object.id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to delete business object');
      }

      setBusinessObjects(prev => prev.filter(obj => obj.id !== object.id));
      notification.success(`"${object.display_name}" deleted successfully`);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to delete business object';
      notification.error(errorMsg);
    }
  };

  const handleToggleStatus = (object: BusinessObject, isActive: boolean) => {
    // Update the local state
    setBusinessObjects(prev =>
      prev.map(obj =>
        obj.id === object.id ? { ...obj, is_active: isActive } : obj
      )
    );
    
    // In a real scenario, fire API call here.
    notification.success(`Business Object "${object.display_name}" is now ${isActive ? 'Active' : 'Draft'}`);
  };

  const handleSaveBusinessObject = async (objectData: any) => {
    try {
      const isEditMode = !!editingObject?.id;
      const method = isEditMode ? 'PATCH' : 'POST';
      const url = isEditMode 
        ? `/api/business-objects/${editingObject?.id}`
        : '/api/business-objects';

      // Before creating, check for existing by key/technical name to avoid unique constraint
      if (!isEditMode) {
        const rawKey: string = (objectData?.technical_name
          || objectData?.technicalName
          || objectData?.key
          || objectData?.name) || '';
        const normalizedKey = (rawKey || '').trim();
        if (normalizedKey) {
          const existsResp = await fetch(`/api/business-objects/${encodeURIComponent(normalizedKey)}`, {
            headers: getAuthHeaders(),
          });
          if (existsResp.ok) {
            const existing = await existsResp.json();
            notification.warning(`Business Object "${existing.displayName || normalizedKey}" already exists. Opening existing.`);
            setEditModalOpen(false);
            setEditingObject(null);
            // Ensure list shows it
            await fetchBusinessObjects();
            navigate(`/business-objects/${existing.id || normalizedKey}`);
            return;
          }
        }
      }

      const response = await fetch(url, {
        method,
        headers: getAuthHeaders(),
        body: JSON.stringify(objectData),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || 'Failed to save business object');
      }

      const savedObject = await response.json();

      if (isEditMode) {
        // Update existing
        setBusinessObjects(prev =>
          prev.map(obj =>
            obj.id === editingObject?.id ? { ...obj, ...savedObject } : obj
          )
        );
      } else {
        // Add new
        setBusinessObjects(prev => [...prev, savedObject]);
      }

      setEditModalOpen(false);
      setEditingObject(null);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to save business object';
      throw new Error(errorMsg);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: 'background.default' }}>
      
      {/* Navigation AppBar */}
      <AppBar 
        position="sticky" 
        elevation={0}
        sx={{ 
          bgcolor: 'background.paper', 
          color: 'text.primary',
          borderBottom: '1px solid',
          borderBottomColor: 'divider',
        }}
      >
        <Toolbar>
          <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: 1 }}>
            <Avatar
              sx={{
                width: 40,
                height: 40,
                bgcolor: 'primary.main',
                color: 'primary.contrastText',
              }}
            >
              📦
            </Avatar>
            <Typography variant="h6" component="div" sx={{ fontWeight: 700 }}>
              Business Object Manager
            </Typography>
          </Stack>

          <Stack direction="row" spacing={2} alignItems="center" sx={{ display: { xs: 'none', md: 'flex' } }}>
            <Button 
              startIcon={<HelpIcon />}
              sx={{ textTransform: 'none', color: 'text.secondary' }}
            >
              Help
            </Button>
          </Stack>

          <Avatar sx={{ ml: 2 }} />
        </Toolbar>
      </AppBar>

      {/* Main Content */}
      <Box component="main" sx={{ flex: 1, py: 4, px: { xs: 2, sm: 4, md: 6, lg: 10 } }}>
        <Container maxWidth="xl">
          
          {/* Action Bar */}
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3} justifyContent="space-between" alignItems={{ xs: 'stretch', sm: 'center' }} sx={{ mb: 3 }}>
            <Box>
              <Typography variant="h5" sx={{ fontWeight: 800 }}>
                Business Object Studio
              </Typography>
              <Typography variant="body2" color="text.secondary">
                The semantic centerpiece bridging physical ORM schemas, multi-tenant Workday delta models, and real-time query engines.
              </Typography>
            </Box>
            <Stack direction="row" spacing={1.5}>
              <Button
                variant="outlined"
                color="secondary"
                startIcon={<AIIcon />}
                onClick={() => setAiModalOpen(true)}
                sx={{ fontWeight: 700, textTransform: 'none' }}
              >
                AI Blueprint
              </Button>
              <Button
                variant="outlined"
                startIcon={<RunIcon />}
                onClick={() => navigate('/query-builder')}
                sx={{ fontWeight: 600, textTransform: 'none' }}
              >
                Global Query Builder
              </Button>
              <Button 
                variant="contained" 
                color="primary"
                startIcon={<AddIcon />}
                onClick={handleCreateObject}
                sx={{ fontWeight: 600, textTransform: 'none' }}
              >
                Create Business Object
              </Button>
            </Stack>

          </Stack>

          {/* Executive Metrics Bar */}
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid size={{ xs: 6, sm: 4, md: 2.4 }}>
              <Paper
                variant="outlined"
                sx={{
                  p: 1.75,
                  textAlign: 'center',
                  cursor: 'pointer',
                  borderColor: scopeFilter === 'all' ? 'primary.main' : 'divider',
                  bgcolor: scopeFilter === 'all' ? 'action.selected' : 'background.paper',
                  transition: 'all 0.2s ease',
                }}
                onClick={() => setScopeFilter('all')}
              >
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                  Total Entities
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5 }}>
                  {metrics.total}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Across all scopes
                </Typography>
              </Paper>
            </Grid>

            <Grid size={{ xs: 6, sm: 4, md: 2.4 }}>
              <Paper
                variant="outlined"
                sx={{
                  p: 1.75,
                  textAlign: 'center',
                  cursor: 'pointer',
                  borderColor: scopeFilter === 'core' ? 'primary.main' : 'divider',
                  bgcolor: scopeFilter === 'core' ? 'action.selected' : 'background.paper',
                  transition: 'all 0.2s ease',
                }}
                onClick={() => setScopeFilter('core')}
              >
                <Typography variant="caption" color="primary.main" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                  Core Master (Gold)
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5, color: 'primary.main' }}>
                  {metrics.coreCount}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Gold Copy Standard
                </Typography>
              </Paper>
            </Grid>

            <Grid size={{ xs: 6, sm: 4, md: 2.4 }}>
              <Paper
                variant="outlined"
                sx={{
                  p: 1.75,
                  textAlign: 'center',
                  cursor: 'pointer',
                  borderColor: scopeFilter === 'custom' ? 'secondary.main' : 'divider',
                  bgcolor: scopeFilter === 'custom' ? 'action.selected' : 'background.paper',
                  transition: 'all 0.2s ease',
                }}
                onClick={() => setScopeFilter('custom')}
              >
                <Typography variant="caption" color="secondary.main" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                  Tenant Custom
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5, color: 'secondary.main' }}>
                  {metrics.customCount}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Client Delta Extensions
                </Typography>
              </Paper>
            </Grid>

            <Grid size={{ xs: 6, sm: 4, md: 2.4 }}>
              <Paper variant="outlined" sx={{ p: 1.75, textAlign: 'center' }}>
                <Typography variant="caption" color="success.main" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                  Active & Published
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5, color: 'success.main' }}>
                  {metrics.activeCount}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Production ready
                </Typography>
              </Paper>
            </Grid>

            <Grid size={{ xs: 6, sm: 4, md: 2.4 }}>
              <Paper variant="outlined" sx={{ p: 1.75, textAlign: 'center' }}>
                <Typography variant="caption" color="info.main" sx={{ fontWeight: 700, textTransform: 'uppercase' }}>
                  ORM Driver Tables
                </Typography>
                <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5, color: 'info.main' }}>
                  {metrics.boundTables}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Physical source links
                </Typography>
              </Paper>
            </Grid>
          </Grid>

          <CatalogList<BusinessObject>
            items={sortedBusinessObjects}
            getId={(o) => o.id}
            getTitle={(o) => o.display_name ?? o.name}
            getSubtitle={(o) => o.driver_table_name || undefined}
            getDescription={(o) => o.description || 'No description provided.'}
            isCore={(o) => !!o.is_core}
            matchesSearch={(o, q) => filterBusinessObjectsBySearch([o], q).length > 0}
            filter={(o) => statusFilter === 'all' || (statusFilter === 'active' ? !!o.is_active : !o.is_active)}
            scope={scopeFilter}
            onScopeChange={setScopeFilter}
            storageKey="business-objects-view"
            searchPlaceholder="Search business objects by name, description or driving table (e.g. mdm party)"
            loading={loading}
            error={error}
            onDismissError={() => setError(null)}
            emptyMessage="No business objects yet. Create your first one or select a driving table."
            noMatchMessage="No business objects match your search."
            onOpen={handleViewDetails}
            onActions={(o, el) => setActionsMenu({ el, object: o })}
            createLabel="Create Business Object"
            onCreate={handleCreateObject}
            toolbarExtra={(
              <Stack direction="row" spacing={1} alignItems="center">
                <ToggleButtonGroup exclusive size="small" value={statusFilter} onChange={(_, v) => v && setStatusFilter(v)}>
                  <ToggleButton value="all" sx={{ textTransform: 'none', px: 2 }}>Any status</ToggleButton>
                  <ToggleButton value="active" sx={{ textTransform: 'none', px: 2 }}>Active</ToggleButton>
                  <ToggleButton value="draft" sx={{ textTransform: 'none', px: 2 }}>Draft</ToggleButton>
                </ToggleButtonGroup>
                <IconButton size="small" onClick={fetchBusinessObjects} disabled={loading} title="Refresh">
                  <RefreshIcon />
                </IconButton>
              </Stack>
            )}
            renderTileMeta={(o) => (
              <Stack direction="row" spacing={0.5} useFlexGap flexWrap="wrap" alignItems="center">
                <Chip label={o.is_active ? 'Active' : 'Draft'} size="small" color={o.is_active ? 'success' : 'warning'} />
                {o.enable_history && <Chip label="Historical" size="small" color="info" variant="outlined" />}
                <Chip icon={<SchemaIcon />} label={`${o.config?.fields?.length || 0} fields`} size="small" variant="outlined" />
                <Chip icon={<CategoryIcon />} label={`${Object.keys(o.subtypes || {}).length} subtypes`} size="small" variant="outlined" />
              </Stack>
            )}
            columns={[
              { key: 'driver', header: 'Driver Table', render: (o) => <span style={{ fontFamily: 'monospace' }}>{o.driver_table_name || '—'}</span> },
              { key: 'description', header: 'Description', render: (o) => o.description || '—' },
              { key: 'status', header: 'Status', render: (o) => <Chip label={o.is_active ? 'Active' : 'Draft'} size="small" color={o.is_active ? 'success' : 'warning'} /> },
              { key: 'fields', header: 'Fields', render: (o) => o.config?.fields?.length || 0 },
              { key: 'subtypes', header: 'Subtypes', render: (o) => Object.keys(o.subtypes || {}).length },
            ]}
          />

          <Menu anchorEl={actionsMenu?.el} open={!!actionsMenu} onClose={() => setActionsMenu(null)}>
            <MenuItem onClick={() => { const o = actionsMenu!.object; setActionsMenu(null); handleViewDetails(o); }}>
              <RunIcon fontSize="small" sx={{ mr: 1 }} /> Live Query Explorer
            </MenuItem>
            <MenuItem onClick={() => { const o = actionsMenu!.object; setActionsMenu(null); handleEditObject(o); }}>
              <EditIcon fontSize="small" sx={{ mr: 1 }} /> Edit
            </MenuItem>
            <MenuItem sx={{ color: 'error.main' }} onClick={() => { const o = actionsMenu!.object; setActionsMenu(null); handleDeleteObject(o); }}>
              <DeleteIcon fontSize="small" sx={{ mr: 1 }} /> Delete
            </MenuItem>
          </Menu>
        </Container>
      </Box>


      {/* Edit Business Object Modal */}
      <EditBusinessObjectModal
        isOpen={editModalOpen}
        object={editingObject as any}
        onClose={() => {
          setEditModalOpen(false);
          setEditingObject(null);
        }}
        onSave={handleSaveBusinessObject}
      />

      {/* Field Validation Rules Modal */}
      {fieldValidationModalOpen && selectedFieldForValidation && (
        <Dialog
          open={fieldValidationModalOpen}
          onClose={() => setFieldValidationModalOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle sx={{ fontWeight: 600 }}>
            Validation Rules: {selectedFieldForValidation.displayName || selectedFieldForValidation.name}
            <IconButton
              onClick={() => setFieldValidationModalOpen(false)}
              sx={{ position: 'absolute', right: 8, top: 8 }}
            >
              <CloseIcon />
            </IconButton>
          </DialogTitle>
          <DialogContent>
            {(() => {
              const fieldRules = getValidationRulesForField(selectedFieldForValidation.key);
              return fieldRules.length > 0 ? (
                <Box sx={{ mt: 2, overflowX: 'auto' }}>
                  {/* Rules table would go here */}
                  <Typography variant="body2" color="text.secondary">
                    Rules list placeholder
                  </Typography>
                </Box>
              ) : (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <InfoIcon sx={{ fontSize: 40, color: 'text.secondary', mb: 1 }} />
                  <Typography variant="body2" color="text.secondary">
                    No validation rules found for this field.
                  </Typography>
                </Box>
              );
            })()}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setFieldValidationModalOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>
      )}
      
      {/* Business Object Creation Wizard */}
      <BusinessObjectBindingWizard
        open={wizardOpen}
        onClose={() => setWizardOpen(false)}
        onSave={handleWizardSave}
      />

      {/* AI Semantic Blueprint Modal */}
      <BOAIAssistantModal
        open={aiModalOpen}
        onClose={() => setAiModalOpen(false)}
        onCreated={fetchBusinessObjects}
      />
    </Box>
  );
}

