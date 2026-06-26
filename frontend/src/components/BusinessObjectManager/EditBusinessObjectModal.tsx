/**
 * EditBusinessObjectModal.tsx
 * Modal for editing Business Object definitions with driver table selection
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Stack,
  FormControlLabel,
  Checkbox,
  Typography,
  Autocomplete,
  Box,
  Chip,
  Card,
  CardContent,
  Alert,
  CircularProgress,
  Tooltip,
  InputAdornment,
  Paper,
} from '@mui/material';
import { useTenant } from '../../contexts/TenantContext';
import { useNotification } from '../../hooks/useNotification';
import { devError } from '../../utils/devLogger';
import { useEnhancedSemanticTerms, semanticTermToField, EnhancedSemanticTerm } from '../../hooks/useEnhancedSemanticTerms';
import { FieldSelectionWizard } from './FieldSelectionWizard';
import { useBORelationships } from '../../hooks/useBORelationships';
import { SemanticMappingTab } from './SemanticMappingTab';
import { Tabs, Tab } from '@mui/material';

interface BusinessObjectData {
  id?: string;
  bo_def_id?: string;
  name: string;
  display_name: string;
  description?: string;
  driver_table_id?: string | null;
  driver_table_name?: string;
  status: 'draft' | 'active' | 'deprecated';
  enable_history?: boolean;
  history_mode?: 'EXPLICIT_RANGE' | 'EVENT_LOG';
  config?: {
    is_active?: boolean;
    [key: string]: any;
  };
}

interface CatalogNode {
  node_id: string;
  qualified_path: string;
  node_name: string;
  node_type: string;
}

interface EditBusinessObjectModalProps {
  isOpen: boolean;
  object?: BusinessObjectData | null;
  onClose: () => void;
  onSave: (object: BusinessObjectData) => Promise<void>;
}

import { getSelectedRegion } from '../../lib/region';

// ... (existing imports)

export const EditBusinessObjectModal: React.FC<EditBusinessObjectModalProps> = ({
  isOpen,
  object,
  onClose,
  onSave,
}) => {
  const { tenant, datasource } = useTenant();
  const notification = useNotification();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';
  
  const [formData, setFormData] = useState<BusinessObjectData>({
    name: '',
    display_name: '',
    description: '',
    status: 'draft',
    enable_history: false,
    history_mode: 'EXPLICIT_RANGE',
    driver_table_id: null,
    driver_table_name: '',
    config: { is_active: true },
  });

  const [catalogNodes, setCatalogNodes] = useState<CatalogNode[]>([]);
  const [loadingCatalog, setLoadingCatalog] = useState(false);
  const { semanticTerms, loading: semanticLoading, error: semanticError } = useEnhancedSemanticTerms(datasourceId);
  const [selectedSemanticTerms, setSelectedSemanticTerms] = useState<EnhancedSemanticTerm[]>([]);
  const [saving, setSaving] = useState(false);
  const [searchCatalog, setSearchCatalog] = useState('');
  const [fieldWizardOpen, setFieldWizardOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  const { data: relationships, loading: relLoading } = useBORelationships(object?.id || object?.bo_def_id);

  const isEditMode = !!object?.id || !!object?.bo_def_id;

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

  // Load catalog nodes for driver table selection
  useEffect(() => {
    if (!isOpen || !tenantId || !datasourceId) return;

    const loadCatalogNodes = async () => {
      try {
        setLoadingCatalog(true);
        const response = await fetch(
          `/api/catalog/nodes?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&type=table`,
          {
            headers: getAuthHeaders(),
          }
        );

        if (!response.ok) {
          throw new Error('Failed to load catalog nodes');
        }

        const data = await response.json();
        setCatalogNodes(Array.isArray(data) ? data : (data?.nodes || []));
      } catch (err) {
        devError('Failed to load catalog nodes:', err);
        notification.error('Failed to load available tables');
      } finally {
        setLoadingCatalog(false);
      }
    };

    loadCatalogNodes();
  }, [isOpen, tenantId, datasourceId]);

  // Initialize form with object data when modal opens
  useEffect(() => {
    if (object && isOpen) {
      setFormData({
        id: object.id,
        bo_def_id: object.bo_def_id,
        name: object.name || '',
        display_name: object.display_name || object.name || '',
        description: object.description || '',
        status: object.status || 'draft',
        enable_history: object.enable_history || false,
        history_mode: (object as any).history_mode || 'EXPLICIT_RANGE',
        driver_table_id: object.driver_table_id || null,
        driver_table_name: object.driver_table_name || '',
        config: object.config || { is_active: true },
      });
    } else if (isOpen) {
      // Reset for new object
      setFormData({
        name: '',
        display_name: '',
        description: '',
        status: 'draft',
        enable_history: false,
        history_mode: 'EXPLICIT_RANGE',
        driver_table_id: null,
        driver_table_name: '',
        config: { is_active: true },
      });
      setSelectedSemanticTerms([]);
    }
  }, [object, isOpen]);

  // Preload semantic term selections from existing config after semantic terms are loaded
  useEffect(() => {
    if (object && isOpen && semanticTerms && semanticTerms.length > 0) {
      const existingFields = object.config?.fields || [];
      const semanticFieldIds = existingFields
        .map((f: any) => f.semanticTermId || f.semantic_term_id)
        .filter(Boolean);

      if (semanticFieldIds.length > 0) {
        const matchedTerms: EnhancedSemanticTerm[] = [];
        semanticFieldIds.forEach((id: string) => {
          const match = semanticTerms.find((t) => t.id === id);
          if (match) matchedTerms.push(match);
        });
        if (matchedTerms.length > 0) {
          setSelectedSemanticTerms(matchedTerms);
        }
      }
    }
  }, [object?.bo_def_id, isOpen, semanticTerms]);

  const selectedDriverTable = useMemo(() => {
    if (!formData.driver_table_id && !formData.driver_table_name) return null;
    
    // First try to match by ID if available
    if (formData.driver_table_id) {
      const byId = catalogNodes.find(n => n.node_id === formData.driver_table_id);
      if (byId) return byId;
    }
    
    // Fallback to match by qualified_path (driver_table_name)
    if (formData.driver_table_name) {
      const byPath = catalogNodes.find(n => n.qualified_path === formData.driver_table_name);
      if (byPath) return byPath;
    }
    
    return null;
  }, [formData.driver_table_id, formData.driver_table_name, catalogNodes]);

  // Sort and search filtered semantic terms
  const sortedAndSearchedSemanticTerms = useMemo(() => {
    let result = [...(selectedSemanticTerms || [])];
    // Simple sort by name for the selected fields display
    result.sort((a, b) => a.node_name.localeCompare(b.node_name));
    return result;
  }, [selectedSemanticTerms]);

  // Filter semantic terms to the selected driver table's columns
  useEffect(() => {
    if (!selectedDriverTable || !semanticTerms || semanticTerms.length === 0) {
      return;
    }

    // Drop any selected terms that are from a different driver table
    const normalizedTable = selectedDriverTable.qualified_path?.toLowerCase?.() || '';
    const tableName = selectedDriverTable.node_name?.toLowerCase?.() || '';

    setSelectedSemanticTerms((prev) =>
      prev.filter((term) => {
        const path = term.qualified_path?.toLowerCase?.() || '';
        return path.includes(normalizedTable) || path.includes(tableName);
      })
    );
  }, [selectedDriverTable, semanticTerms]);

  const handleDriverTableSelect = (node: CatalogNode | null) => {
    if (node) {
      setFormData({
        ...formData,
        driver_table_id: node.node_id,
        driver_table_name: node.qualified_path,
      });
    } else {
      setFormData({
        ...formData,
        driver_table_id: null,
        driver_table_name: '',
      });
      setSelectedSemanticTerms([]); // Clear selected fields if driver table is cleared
    }
  };

  const handleFieldsSelected = (fields: EnhancedSemanticTerm[]) => {
    // Process new fields to set default display names from title_short
    const processedFields = fields.map(field => {
      // Create a modified copy of the field with title_short as the default name
      // This ensures the initial "name" and "displayName" of the field come from title_short
      if (field.title_short) {
        // We attach title_short to the term so it can be used by semanticTermToField
        const termWithTitle = { ...field };
        // We can't directly modify read-only properties of EnhancedSemanticTerm effectively here 
        // without type assertions, but the key is how semanticTermToField uses it.
        // Or we can pre-process them here before adding to selectedSemanticTerms.
        
        // Actually, let's just use the field directly but ensure when we map to backend format
        // in handleSave or render in the list, we prioritize title_short if available.
        return termWithTitle;
      }
      return field;
    });

    // Instead of just mapping, let's explicitly set the name/displayName properties for the UI
    const fieldsWithDefaults = fields.map(f => ({
      ...f,
      // If title_short exists, use it as the default name/display name override
      // We store this override on the object which will be used by semanticTermToField
      overrideName: f.title_short || f.node_name,
      overrideDisplayName: f.title_short || f.node_name
    }));

    setSelectedSemanticTerms([...selectedSemanticTerms, ...fieldsWithDefaults]);
    setFieldWizardOpen(false);
  };

  const handleRemoveField = (fieldId: string) => {
    setSelectedSemanticTerms(selectedSemanticTerms.filter((f) => f.id !== fieldId));
  };

  const handleUpdateFieldMapping = (index: number, updates: any) => {
    // This is a bit tricky because we're using selectedSemanticTerms which are EnhancedSemanticTerm[]
    // But we need to update properties that aren't native to EnhancedSemanticTerm if we want to preview them
    // Actually, we should probably update the BO data's config.fields if we want to persist mappings.
    // However, selectedSemanticTerms is the primary source of truth during editing.
    
    // We'll update the selectedSemanticTerms by adding/updating to their local properties or config
    setSelectedSemanticTerms(prev => {
      const updated = [...prev];
      const term = updated[index];
      if (term) {
        // We'll store role and semanticTermId in an 'override' metadata for now
        // or just add them to the term object directly
        updated[index] = {
          ...term,
          ...updates
        };
      }
      return updated;
    });
  };

  const handleSave = async () => {
    try {
      // Validation
      if (!formData.name.trim()) {
        notification.error('Object name is required');
        return;
      }

      if (!formData.display_name.trim()) {
        notification.error('Display name is required');
        return;
      }

      if (!formData.driver_table_id) {
        notification.warning('Selecting a driver table is recommended for better integration');
      }

      setSaving(true);

      // Prepare payload
      const semanticFields = selectedSemanticTerms.map((term, idx) => {
        const field = semanticTermToField(term, idx);
        
        // specific overrides from our local state (handleFieldsSelected)
        const nameOverride = (term as any).overrideName;
        const displayNameOverride = (term as any).overrideDisplayName;

        // Include any overrides from SemanticMappingTab
        return {
          ...field,
          name: nameOverride || field.name,
          displayName: displayNameOverride || field.businessName, // Fallback to businessName if field.displayName missing
          businessName: displayNameOverride || field.businessName,
          role: (term as any).role || field.role || 'DIMENSION',
          semanticTermId: (term as any).semanticTermId || field.semanticTermId,
        };
      });
      const payload = {
        ...formData,
        display_name: formData.display_name || formData.name,
        config: {
          ...(formData.config || {}),
          fields: semanticFields,
        },
      };

      await onSave(payload);

      notification.success(
        isEditMode 
          ? `"${formData.display_name}" updated successfully` 
          : `"${formData.display_name}" created successfully`
      );
      onClose();
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to save business object';
      devError('Error saving business object:', err);
      notification.error(errorMsg);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog 
      open={isOpen} 
      onClose={onClose} 
      maxWidth="sm" 
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          boxShadow: 3,
        },
      }}
    >
      <DialogTitle sx={{ fontWeight: 600, fontSize: '1.25rem' }}>
        {isEditMode ? '✏️ Edit Business Object' : '➕ Create Business Object'}
      </DialogTitle>

      <DialogContent sx={{ pt: 2, minHeight: 400 }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
          <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
            <Tab label="General" />
            <Tab label="Fields & Mappings" disabled={!isEditMode && selectedSemanticTerms.length === 0} />
          </Tabs>
        </Box>

        <Stack spacing={3} sx={{ display: activeTab === 0 ? 'flex' : 'none' }}>
          {/* Basic Information Section */}
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2, color: 'primary.main' }}>
              📋 Basic Information
            </Typography>
            
            <Stack spacing={2}>
              <TextField
                label="Object Name"
                placeholder="e.g., Customer, Portfolio, IPS"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                fullWidth
                size="small"
                helperText="Stable key used in code (e.g., 'customer')"
              />

              <TextField
                label="Display Name"
                placeholder="e.g., Customer Profile"
                value={formData.display_name}
                onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
                fullWidth
                size="small"
                helperText="Human-readable name shown in UI"
              />

              <TextField
                label="Description"
                placeholder="What does this business object represent?"
                value={formData.description || ''}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                fullWidth
                multiline
                rows={3}
                size="small"
              />
            </Stack>
          </Box>

          {/* Driver Table Selection */}
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2, color: 'primary.main' }}>
              🗂️ Driver Table (Source)
            </Typography>

            <Card variant="outlined" sx={{ mb: 2, bgcolor: 'background.paper' }}>
              <CardContent>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                  The primary table that defines this business object
                </Typography>

                {!isEditMode && (
                  <Autocomplete
                    options={catalogNodes}
                    getOptionLabel={(option) => option.qualified_path}
                    value={selectedDriverTable}
                    onChange={(_, node) => handleDriverTableSelect(node)}
                    loading={loadingCatalog}
                    inputValue={searchCatalog}
                    onInputChange={(_, value) => setSearchCatalog(value)}
                    isOptionEqualToValue={(option, value) => option.node_id === value?.node_id}
                    size="small"
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        placeholder="Search tables..."
                        variant="outlined"
                        size="small"
                        InputProps={{
                          ...params.InputProps,
                          endAdornment: (
                            <>
                              {loadingCatalog ? <CircularProgress color="inherit" size={20} /> : null}
                              {params.InputProps.endAdornment}
                            </>
                          ),
                        }}
                      />
                    )}
                    renderOption={(props, option) => {
                      const { key, ...otherProps } = props;
                      return (
                        <li key={key} {...otherProps}>
                          <Stack spacing={0.5}>
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>
                              {option.node_name}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {option.qualified_path}
                            </Typography>
                          </Stack>
                        </li>
                      );
                    }}
                    noOptionsText={loadingCatalog ? 'Loading...' : 'No tables found'}
                  />
                )}

                {selectedDriverTable && (
                  <Box sx={{ mt: 1.5 }}>
                    {isEditMode ? (
                      <Stack spacing={0.5} sx={{ p: 1, bgcolor: 'action.disabledBackground', borderRadius: 1 }}>
                         <Typography variant="body2" sx={{ fontWeight: 600 }}>
                            {selectedDriverTable.node_name}
                         </Typography>
                         <Typography variant="caption" color="text.secondary">
                            {selectedDriverTable.qualified_path}
                         </Typography>
                         <Typography variant="caption" color="warning.main" sx={{ fontStyle: 'italic' }}>
                            Driver table cannot be changed after creation.
                         </Typography>
                      </Stack>
                    ) : (
                      <Chip
                        label={selectedDriverTable.qualified_path}
                        onDelete={() => handleDriverTableSelect(null)}
                        variant="outlined"
                        size="small"
                      />
                    )}
                  </Box>
                )}
              </CardContent>
            </Card>
          </Box>

          {/* Configuration */}
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2, color: 'primary.main' }}>
              ⚙️ Configuration
            </Typography>

            <FormControlLabel
              control={
                <Checkbox
                  checked={formData.config?.is_active ?? true}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      config: { ...formData.config, is_active: e.target.checked },
                    })
                  }
                />
              }
              label="Enable this business object"
            />

             <FormControlLabel
               control={
                 <Checkbox
                   checked={formData.enable_history ?? false}
                   onChange={(e) =>
                     setFormData({
                       ...formData,
                       enable_history: e.target.checked,
                     })
                   }
                 />
               }
               label="Enable Effective Dating (Temporal Queries)"
             />

             {formData.enable_history && (
               <TextField
                 select
                 fullWidth
                 size="small"
                 label="History Mode"
                 value={formData.history_mode || 'EXPLICIT_RANGE'}
                 onChange={(e) => setFormData({ ...formData, history_mode: e.target.value as any })}
                 SelectProps={{ native: true }}
                 sx={{ mt: 2 }}
               >
                 <option value="EXPLICIT_RANGE">Explicit Range (Start & End Columns)</option>
                 <option value="EVENT_LOG">Event Log (Lead Window Function)</option>
               </TextField>
             )}
          </Box>
        </Stack>

        <Box sx={{ display: activeTab === 1 ? 'block' : 'none' }}>
          <SemanticMappingTab 
            fields={selectedSemanticTerms.map((term, idx) => ({
              ...semanticTermToField(term, idx),
              role: (term as any).role,
              semanticTermId: (term as any).semanticTermId || term.id,
              qualified_path: term.qualified_path,
              description: term.description,
              source_column: term.properties?.sql
            }))}
            availableTerms={relationships.availableTerms}
            onUpdateField={handleUpdateFieldMapping}
            onAddField={() => setFieldWizardOpen(true)}
            onRemoveField={(idx) => {
               const term = selectedSemanticTerms[idx];
               if (term) handleRemoveField(term.id);
            }}
          />
        </Box>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, gap: 1 }}>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button
          onClick={handleSave}
          variant="contained"
          disabled={saving || !formData.name.trim() || !formData.display_name.trim()}
          sx={{ minWidth: 120 }}
        >
          {saving ? <CircularProgress size={20} sx={{ mr: 1 }} /> : null}
          {isEditMode ? '✓ Update' : '✓ Create'}
        </Button>
      </DialogActions>

      {/* Field Selection Wizard Modal */}
      <FieldSelectionWizard
        isOpen={fieldWizardOpen}
        onClose={() => setFieldWizardOpen(false)}
        onSelectFields={handleFieldsSelected}
        selectedDriverTable={selectedDriverTable}
        existingFields={selectedSemanticTerms}
        loading={semanticLoading}
      />
    </Dialog>
  );
};

export default EditBusinessObjectModal;

