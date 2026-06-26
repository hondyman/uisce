import { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';

import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Typography,
  Switch,
  Box,
  Button,
  CircularProgress,
  AppBar,
  Toolbar,
  Container,
  Grid,
  Card,
  CardContent,
  CardActions,
  Chip,
  TextField,
  Paper,
  Stack,
  InputAdornment,
  IconButton,
  Avatar,
  Divider,
  DialogContentText,
  useTheme,
  Alert,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Search as SearchIcon,
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
} from '@mui/icons-material';
import ValidationRuleScriptEditor from '../components/ValidationRules/ValidationRuleScriptEditor';
import { ValidationRuleCreator } from '../components/ValidationRules/ValidationRuleCreator';
import { EditBusinessObjectModal } from '../components/BusinessObjectManager/EditBusinessObjectModal';
import { BusinessObjectWizard } from '../components/BusinessObjectManager/BusinessObjectWizard';
import { GlobalBOSearch } from '../components/Search/GlobalBOSearch';
import { useTenant } from '../contexts/TenantContext';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { devDebug } from '../utils/devLogger';

import { getSelectedRegion } from '../lib/region';

interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
  description?: string;
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
  
  const [viewMode, setViewMode] = useState<'card' | 'table'>('card');
  const [businessObjectsSearch, setBusinessObjectsSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'draft'>('all');

  // Validation Rules State
  const [selectedFieldForValidation, _setSelectedFieldForValidation] = useState<any>(null);
  const [fieldValidationModalOpen, setFieldValidationModalOpen] = useState(false);
  const [validationRuleCreatorOpen, setValidationRuleCreatorOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<any>(null);
  const [viewingRule, setViewingRule] = useState<any>(null);
  const [availableEntitiesMemo, _setAvailableEntitiesMemo] = useState<any[]>([]);
  const [entitySchemaMemo, _setEntitySchemaMemo] = useState<any>(null);
  const [selectedObject, _setSelectedObject] = useState<BusinessObject | null>(null);

  // Edit Business Object Modal State
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingObject, setEditingObject] = useState<BusinessObject | null>(null);
  
  // Wizard State
  const [wizardOpen, setWizardOpen] = useState(false);

  // Helper to build headers with authentication
  const getAuthHeaders = (additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';
    
    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      'X-Tenant-Region': getSelectedRegion(),
      ...additionalHeaders,
    };
  };

  const getValidationRulesForField = (_fieldKey: string) => {
    // Placeholder for fetching validation rules filtering by field
    return [];
  };

  const handleSaveValidationRule = (rule: any) => {
     devDebug('Saved rule:', rule);
     setValidationRuleCreatorOpen(false);
  };

  const _handleEditValidationRule = (rule: any) => {
      setEditingRule(rule);
      setValidationRuleCreatorOpen(true);
  };

  // Filtered business objects based on search and status
  const filteredBusinessObjects = useMemo(() => {
    let filtered = businessObjects;
    
    // Search Filter
    if (businessObjectsSearch.trim()) {
      const searchTerm = businessObjectsSearch.toLowerCase();
      filtered = filtered.filter(obj => 
        obj.name.toLowerCase().includes(searchTerm) ||
        obj.display_name.toLowerCase().includes(searchTerm) ||
        obj.description?.toLowerCase().includes(searchTerm)
      );
    }

    // Status Filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter(obj => 
        statusFilter === 'active' ? !!obj.is_active : !obj.is_active
      );
    }
    
    // Sort by name
    return filtered.sort((a, b) => a.display_name.localeCompare(b.display_name));
  }, [businessObjects, businessObjectsSearch, statusFilter]);

  const fetchBusinessObjects = async () => {
    if (!tenantId || !datasourceId) {
      setBusinessObjects([]);
      setError('Please select a tenant and datasource');
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
          enable_history: obj.enableHistory || false,
          updated_at: obj.updated_at,
          parent_id: (obj.parentId && typeof obj.parentId === 'object' && 'Valid' in obj.parentId) 
            ? (obj.parentId.Valid ? obj.parentId.String : null)
            : (obj.parentId || null),
        };
      }).filter(obj => !obj.parent_id); // Filter out subtypes - only show parent business objects

      // Fallback: if primary endpoint returned empty, try the designer list endpoint
      if (objectsArray.length === 0) {
        const resp2 = await fetch('/api/business-objects/list', {
          headers: getAuthHeaders(),
        });
        if (resp2.ok) {
          const items = await resp2.json(); // array of { id, name, display_name, description, fields, icon, config }
          objectsArray = (Array.isArray(items) ? items : []).map((item: any) => ({
            id: item.id,
            name: item.name || item.display_name || item.id,
            display_name: item.display_name || item.name || item.id,
            description: item.description,
            config: item.config || {},
            subtypes: {},
            is_active: (item.config?.is_active !== false),
            enable_history: false,
            updated_at: undefined,
            parent_id: null,
          }));
        }
      }

      setBusinessObjects(objectsArray);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch business objects';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (tenantId && datasourceId) {
      fetchBusinessObjects();
    }
  }, [tenantId, datasourceId]);

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
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3} justifyContent="flex-end" alignItems={{ xs: 'stretch', sm: 'center' }} sx={{ mb: 4 }}>
            <Button 
              variant="contained" 
              color="primary"
              startIcon={<AddIcon />}
              onClick={handleCreateObject}
              size="large"
              sx={{ fontWeight: 600, textTransform: 'none' }}
            >
              Create Object
            </Button>
          </Stack>

          {/* Controls Toolbar */}
          <Paper 
            elevation={0}
            sx={{ 
              p: 2.5, 
              mb: 4,
              border: '1px solid',
              borderColor: 'divider',
              borderRadius: 2,
            }}
          >
            <Stack direction={{ xs: 'column', lg: 'row' }} spacing={2} alignItems={{ lg: 'center' }}>
              
              {/* Semantic Search */}
              <Box sx={{ flex: 1 }}>
                <GlobalBOSearch onResultClick={(boId) => navigate(`/bo/${boId}`)} />
              </Box>

              {/* Filters */}
              <Stack direction="row" spacing={1} alignItems="center" sx={{ overflow: 'auto', pb: { xs: 1, lg: 0 }, minWidth: 'fit-content' }}>
                <Typography variant="caption" sx={{ fontWeight: 600, whiteSpace: 'nowrap', color: 'text.secondary' }}>
                  Status:
                </Typography>
                
                <Chip
                  label="All"
                  variant={statusFilter === 'all' ? 'filled' : 'outlined'}
                  color={statusFilter === 'all' ? 'primary' : 'default'}
                  onClick={() => setStatusFilter('all')}
                  size="small"
                />
                
                <Chip
                  label="Active"
                  variant={statusFilter === 'active' ? 'filled' : 'outlined'}
                  color={statusFilter === 'active' ? 'primary' : 'default'}
                  onClick={() => setStatusFilter('active')}
                  size="small"
                />
                
                <Chip
                  label="Draft"
                  variant={statusFilter === 'draft' ? 'filled' : 'outlined'}
                  color={statusFilter === 'draft' ? 'primary' : 'default'}
                  onClick={() => setStatusFilter('draft')}
                  size="small"
                />

                <Divider orientation="vertical" flexItem sx={{ mx: 0.5 }} />

                {/* View Toggle */}
                <Stack direction="row" spacing={0.5} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 0.5 }}>
                  <IconButton 
                    size="small"
                    onClick={() => setViewMode('card')}
                    sx={{ 
                      color: viewMode === 'card' ? 'primary.main' : 'action.disabled',
                      backgroundColor: viewMode === 'card' ? 'action.selected' : 'transparent',
                    }}
                    title="Card View"
                  >
                    <ViewWeekIcon fontSize="small" />
                  </IconButton>
                  <IconButton 
                    size="small"
                    onClick={() => setViewMode('table')}
                    sx={{ 
                      color: viewMode === 'table' ? 'primary.main' : 'action.disabled',
                      backgroundColor: viewMode === 'table' ? 'action.selected' : 'transparent',
                    }}
                    title="Table View"
                  >
                    <ViewAgendaIcon fontSize="small" />
                  </IconButton>
                </Stack>

                <IconButton 
                  size="small"
                  onClick={fetchBusinessObjects}
                  disabled={loading}
                  sx={{ display: { xs: 'none', sm: 'flex' } }}
                  title="Refresh"
                >
                  <RefreshIcon sx={{ animation: loading ? 'spin 1s linear infinite' : 'none', '@keyframes spin': { '0%': { transform: 'rotate(0deg)' }, '100%': { transform: 'rotate(360deg)' } } }} />
                </IconButton>
              </Stack>
            </Stack>
          </Paper>

          {/* Error Alert */}
          {error && (
            <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 3 }}>
              {error}
            </Alert>
          )}

          {/* Loading State */}
          {loading && filteredBusinessObjects.length === 0 ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 10 }}>
              <CircularProgress />
            </Box>
          ) : filteredBusinessObjects.length === 0 ? (
            /* Empty State */
            <Paper 
              elevation={0}
              sx={{ 
                p: 6, 
                textAlign: 'center',
                border: '2px dashed',
                borderColor: 'divider',
                borderRadius: 2,
              }}
            >
              <Avatar 
                sx={{ 
                  width: 64, 
                  height: 64, 
                  mx: 'auto', 
                  mb: 2,
                  bgcolor: 'action.hover',
                  color: 'text.secondary',
                }}
              >
                📊
              </Avatar>
              <Typography variant="h6" sx={{ fontWeight: 700, mb: 1 }}>
                No Business Objects Found
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 3, maxWidth: 400, mx: 'auto' }}>
                {businessObjectsSearch 
                  ? `No matches for "${businessObjectsSearch}"` 
                  : 'Get started by creating your first business object.'}
              </Typography>
              {!businessObjectsSearch && (
                <Button 
                  variant="contained" 
                  color="primary"
                  startIcon={<AddIcon />}
                  onClick={handleCreateObject}
                >
                  Create Object
                </Button>
              )}
            </Paper>
          ) : (
            viewMode === 'card' ? (
              /* Card Grid Layout */
              <Grid container spacing={3}>
                {filteredBusinessObjects.map((object) => (
                  <Grid item xs={12} sm={6} md={4} lg={3} key={object.id}>
                    <Card
                      onClick={() => handleViewDetails(object)}
                      sx={{
                        cursor: 'pointer',
                        height: '100%',
                        display: 'flex',
                        flexDirection: 'column',
                        transition: 'all 0.3s ease',
                        border: '1px solid',
                        borderColor: 'divider',
                        '&:hover': {
                          transform: 'translateY(-4px)',
                          boxShadow: theme.shadows[6],
                          borderColor: 'primary.main',
                        },
                      }}
                    >
                      <CardContent sx={{ flex: 1 }}>
                        <Stack direction="row" justifyContent="space-between" alignItems="flex-start" spacing={1} sx={{ mb: 2 }}>
                          <Stack direction="column" spacing={1} sx={{ flex: 1 }}>
                            <Typography variant="h6" sx={{ fontWeight: 700 }}>
                              {object.display_name}
                            </Typography>
                            <Chip 
                              label={object.is_active ? 'Active' : 'Draft'}
                              size="small"
                              color={object.is_active ? 'success' : 'warning'}
                              variant="filled"
                              sx={{ width: 'fit-content' }}
                            />
                            {object.enable_history && (
                              <Chip 
                                label="Historical"
                                size="small"
                                color="info"
                                variant="outlined"
                                sx={{ width: 'fit-content' }}
                              />
                            )}
                          </Stack>
                        </Stack>

                        <Typography 
                          variant="body2" 
                          color="text.secondary" 
                          sx={{ 
                            mb: 2,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            display: '-webkit-box',
                            WebkitLineClamp: 2,
                            WebkitBoxOrient: 'vertical',
                            minHeight: 40,
                          }}
                        >
                          {object.description || 'No description provided.'}
                        </Typography>
                      </CardContent>

                      <Divider />

                      <CardActions sx={{ justifyContent: 'space-between', py: 1.5 }}>
                        <Stack direction="row" spacing={3} sx={{ fontSize: '0.75rem' }}>
                          <Stack 
                            direction="row" 
                            spacing={0.5} 
                            alignItems="center"
                            sx={{
                              cursor: 'default',
                              padding: '4px 8px',
                              borderRadius: 1,
                              transition: 'all 0.2s ease',
                              '&:hover': { 
                                bgcolor: 'action.hover',
                                '& .MuiSvgIcon-root': {
                                  color: 'primary.main',
                                }
                              }
                            }}
                            title="Number of Fields"
                          >
                            <SchemaIcon sx={{ fontSize: '1rem', transition: 'color 0.2s ease' }} />
                            <Typography variant="caption">
                              {object.config?.fields?.length || 0}
                            </Typography>
                          </Stack>
                          <Stack 
                            direction="row" 
                            spacing={0.5} 
                            alignItems="center"
                            sx={{
                              cursor: 'default',
                              padding: '4px 8px',
                              borderRadius: 1,
                              transition: 'all 0.2s ease',
                              '&:hover': { 
                                bgcolor: 'action.hover',
                                '& .MuiSvgIcon-root': {
                                  color: 'primary.main',
                                }
                              }
                            }}
                            title="Number of Subtypes"
                          >
                            <CategoryIcon sx={{ fontSize: '1rem', transition: 'color 0.2s ease' }} />
                            <Typography variant="caption">
                              {Object.keys(object.subtypes || {}).length}
                            </Typography>
                          </Stack>
                        </Stack>
                        <Stack direction="row" spacing={1} sx={{ ml: 'auto', alignItems: 'center' }}>
                          <Switch
                            size="small"
                            checked={object.is_active}
                            onChange={(e) => handleToggleStatus(object, e.target.checked)}
                            onClick={(e) => e.stopPropagation()}
                          />
                          <Tooltip title="Edit">
                            <IconButton
                              size="small"
                              onClick={(e) => handleEditObject(object, e)}
                              sx={{ '&:hover': { color: 'primary.main' } }}
                              className="card-action-btn"
                            >
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Delete">
                            <IconButton
                              size="small"
                              onClick={(e) => handleDeleteObject(object, e)}
                              sx={{ '&:hover': { color: 'error.main' } }}
                              className="card-action-btn"
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </CardActions>

                    </Card>
                  </Grid>
                ))}

                {/* Create New Card */}
                <Grid item xs={12} sm={6} md={4} lg={3}>
                  <Card
                    onClick={handleCreateObject}
                    sx={{
                      cursor: 'pointer',
                      height: '100%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      border: '2px dashed',
                      borderColor: 'divider',
                      bgcolor: 'action.hover',
                      transition: 'all 0.3s ease',
                      '&:hover': {
                        borderColor: 'primary.main',
                        bgcolor: 'primary.main',
                        '& svg': { color: 'primary.contrastText' },
                        '& .MuiTypography-root': { color: 'primary.contrastText' },
                      },
                    }}
                  >
                    <Stack direction="column" alignItems="center" spacing={1}>
                      <Avatar 
                        sx={{ 
                          bgcolor: 'transparent',
                          color: 'text.secondary',
                        }}
                      >
                        <AddIcon sx={{ fontSize: 32 }} />
                      </Avatar>
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>
                        New Object
                      </Typography>
                    </Stack>
                  </Card>
                </Grid>
              </Grid>
            ) : (
              /* Table View */
              <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
                <Box sx={{ overflowX: 'auto' }}>
                  <Box component="table" sx={{ width: '100%', borderCollapse: 'collapse' }}>
                    <Box component="thead">
                      <Box component="tr" sx={{ backgroundColor: theme.palette.action.hover, borderBottom: `1px solid ${theme.palette.divider}` }}>
                        <Box component="th" sx={{ padding: '16px', textAlign: 'left', fontWeight: 600, color: theme.palette.text.secondary }}>Name</Box>
                        <Box component="th" sx={{ padding: '16px', textAlign: 'left', fontWeight: 600, color: theme.palette.text.secondary }}>Description</Box>
                        <Box component="th" sx={{ padding: '16px', textAlign: 'center', fontWeight: 600, color: theme.palette.text.secondary }}>Status</Box>
                        <Box component="th" sx={{ padding: '16px', textAlign: 'center', fontWeight: 600, color: theme.palette.text.secondary }}>Fields</Box>
                        <Box component="th" sx={{ padding: '16px', textAlign: 'center', fontWeight: 600, color: theme.palette.text.secondary }}>Subtypes</Box>
                        <Box component="th" sx={{ padding: '16px', textAlign: 'right', fontWeight: 600, color: theme.palette.text.secondary }}>Actions</Box>
                      </Box>
                    </Box>
                    <Box component="tbody">
                      {filteredBusinessObjects.map((object, idx) => (
                        <Box 
                          component="tr"
                          key={object.id} 
                          onClick={() => handleViewDetails(object)}
                          sx={{ 
                            borderBottom: `1px solid ${theme.palette.divider}`,
                            backgroundColor: idx % 2 === 0 ? 'transparent' : theme.palette.action.hover,
                            cursor: 'pointer',
                            transition: 'background-color 0.2s ease',
                          }}
                          onMouseEnter={(e) => {
                            (e.currentTarget as HTMLElement).style.backgroundColor = theme.palette.action.selected;
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLElement).style.backgroundColor = idx % 2 === 0 ? 'transparent' : theme.palette.action.hover;
                          }}
                        >
                          <Box component="td" sx={{ padding: '16px', fontWeight: 600, color: theme.palette.text.primary }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              {object.display_name}
                              {object.enable_history && (
                                <Tooltip title="Effective Dated / History Enabled">
                                  <Chip label="H" size="small" color="info" variant="outlined" sx={{ height: 16, fontSize: '0.6rem' }} />
                                </Tooltip>
                              )}
                            </Box>
                          </Box>
                          <Box component="td" sx={{ padding: '16px', color: theme.palette.text.secondary, maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {object.description || '—'}
                          </Box>
                          <Box component="td" sx={{ padding: '16px', textAlign: 'center' }}>
                            <Chip 
                              label={object.is_active ? 'Active' : 'Draft'}
                              size="small"
                              color={object.is_active ? 'success' : 'warning'}
                              variant="filled"
                            />
                          </Box>
                          <Box component="td" sx={{ padding: '16px', textAlign: 'center', color: theme.palette.text.secondary }}>
                            <Stack 
                              direction="row" 
                              spacing={0.5} 
                              alignItems="center"
                              justifyContent="center"
                              sx={{
                                padding: '4px 8px',
                                borderRadius: 1,
                                transition: 'all 0.2s ease',
                                cursor: 'default',
                                '&:hover': { 
                                  bgcolor: 'action.hover',
                                  '& .MuiSvgIcon-root': {
                                    color: 'primary.main',
                                  }
                                }
                              }}
                            >
                              <SchemaIcon sx={{ fontSize: '1rem', transition: 'color 0.2s ease' }} />
                              <span>{object.config?.fields?.length || 0}</span>
                            </Stack>
                          </Box>
                          <Box component="td" sx={{ padding: '16px', textAlign: 'center', color: theme.palette.text.secondary }}>
                            <Stack 
                              direction="row" 
                              spacing={0.5} 
                              alignItems="center"
                              justifyContent="center"
                              sx={{
                                padding: '4px 8px',
                                borderRadius: 1,
                                transition: 'all 0.2s ease',
                                cursor: 'default',
                                '&:hover': { 
                                  bgcolor: 'action.hover',
                                  '& .MuiSvgIcon-root': {
                                    color: 'primary.main',
                                  }
                                }
                              }}
                            >
                              <CategoryIcon sx={{ fontSize: '1rem', transition: 'color 0.2s ease' }} />
                              <span>{Object.keys(object.subtypes || {}).length}</span>
                            </Stack>
                          </Box>
                          <Box component="td" sx={{ padding: '16px', textAlign: 'right' }}>
                            <Stack 
                              direction="row" 
                              spacing={1} 
                              justifyContent="flex-end"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <Tooltip title="Edit">
                                <IconButton
                                  size="small"
                                  onClick={(e) => handleEditObject(object, e)}
                                  sx={{ '&:hover': { color: 'primary.main' } }}
                                >
                                  <EditIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                              <Tooltip title="Delete">
                                <IconButton
                                  size="small"
                                  onClick={(e) => handleDeleteObject(object, e)}
                                  sx={{ '&:hover': { color: 'error.main' } }}
                                >
                                  <DeleteIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                            </Stack>
                          </Box>
                        </Box>
                      ))}
                    </Box>
                  </Box>
                </Box>
                {filteredBusinessObjects.length === 0 && (
                  <Box sx={{ textAlign: 'center', py: 4 }}>
                    <Typography variant="body2" color="text.secondary">
                      No business objects to display
                    </Typography>
                  </Box>
                )}
              </Paper>
            )
          )}
          
        </Container>
      </Box>

      {/* Validation Rule Creator Modal */}
      {validationRuleCreatorOpen && (
        <ValidationRuleCreator
          isOpen={validationRuleCreatorOpen}
          onClose={() => {
            setValidationRuleCreatorOpen(false);
            setEditingRule(null);
          }}
          onSave={handleSaveValidationRule}
          tenantId={tenantId}
          datasourceId={datasourceId}
          availableEntities={availableEntitiesMemo}
          defaultTargetEntity={selectedObject?.name || ''}
          entitySchema={entitySchemaMemo}
          editingRule={editingRule as any}
        />
      )}

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

      {/* Code Viewer Modal */}
      {viewingRule && (
        <Dialog
          open={!!viewingRule}
          onClose={() => setViewingRule(null)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle sx={{ fontWeight: 600 }}>
            Rule Logic: {viewingRule.rule_name}
            <IconButton
              onClick={() => setViewingRule(null)}
              sx={{ position: 'absolute', right: 8, top: 8 }}
            >
              <CloseIcon />
            </IconButton>
          </DialogTitle>
          <DialogContent>
            <Box sx={{ height: '400px', mt: 2, border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
              <ValidationRuleScriptEditor
                value={
                  viewingRule.rule_type === 'starlark' 
                    ? (viewingRule as any).script_content || '# No script content'
                    : JSON.stringify(viewingRule.condition_json || {}, null, 2)
                }
                onChange={() => {}}
                language={viewingRule.rule_type === 'starlark' ? 'python' : 'json'}
                theme="vs-dark"
              />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setViewingRule(null)}>Close</Button>
          </DialogActions>
        </Dialog>
      )}

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
      <BusinessObjectWizard
        open={wizardOpen}
        onClose={() => setWizardOpen(false)}
        onSave={handleWizardSave}
      />
    </Box>
  );
}
