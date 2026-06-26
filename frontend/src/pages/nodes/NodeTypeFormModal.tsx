import React, { useState, useEffect, lazy, Suspense, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControlLabel,
  Switch,
  Box,
  Typography,
  Alert,
  Stack,
  Divider,
  IconButton,
  Paper,
  Chip,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import CategoryIcon from '@mui/icons-material/Category';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import { useCreateNodeType, useUpdateNodeType, nodeTypesKeys } from '../../api/nodeTypes';
import { useQueryClient } from '@tanstack/react-query';
import { Snackbar } from '@mui/material';
import type { NodeType } from '../../types/nodeTypes';
import PropertySchemaEditor, { PropertyDef } from '../../components/properties/PropertySchemaEditor';
import type { NodeProperty } from '../../types/nodeTypes';
import { devDebug, devWarn, devError } from '../../utils/devLogger';

// Helper to convert NodeProperty to PropertyDef for the editor
const nodePropertyToPropertyDef = (np: NodeProperty): PropertyDef => ({
  name: np.name,
  label: np.label,
  data_type: np.data_type === 'integer' || np.data_type === 'float' ? 'number' : 
             np.data_type === 'text' || np.data_type === 'json' ? 'string' : 
             np.data_type,
  original_data_type: np.data_type, // Preserve original
  input_type: np.input_type === 'date-picker' ? 'date' : 
              np.input_type === 'textarea' || np.input_type === 'number' || np.input_type === 'json-editor' ? 'text' :
              np.input_type,
  original_input_type: np.input_type, // Preserve original
  required: !np.nullable,
  options: np.options || [],
  lookup: (np as any).lookup_id || null,
  cascade_from: (np as any).cascade_from || null,
  syntax_language: np.syntax_language || null,
});

// Helper to convert PropertyDef to NodeProperty for the backend
const propertyDefToNodeProperty = (pd: PropertyDef, order: number): NodeProperty => {
  // Map editor-friendly types back to backend data_type
  const mappedDataType = ((): NodeProperty['data_type'] => {
    switch (pd.data_type) {
      case 'number':
        return 'float';
      case 'string':
        return 'text';
      case 'boolean':
        return 'boolean';
      case 'date':
        return 'date';
      default:
        return pd.data_type as NodeProperty['data_type'];
    }
  })();

  // Prefer the user's selection (mappedDataType) unless it exactly matches the preserved original
  const finalDataType = pd.original_data_type && pd.original_data_type === mappedDataType
    ? pd.original_data_type
    : mappedDataType;

  // Map input types
  const mappedInputType = ((): NodeProperty['input_type'] => {
    switch (pd.input_type) {
      case 'date':
        return 'date-picker';
      case 'text':
        return 'text';
      case 'select':
        return 'select';
      case 'checkbox':
        return 'checkbox';
      default:
        return pd.input_type as NodeProperty['input_type'];
    }
  })();

  const finalInputType = pd.original_input_type && pd.original_input_type === mappedInputType
    ? pd.original_input_type
    : mappedInputType;

  return {
    name: pd.name,
    label: pd.label || pd.name,
    data_type: finalDataType,
    nullable: !pd.required,
    default_value: undefined,
    input_type: finalInputType,
    options: pd.options || [],
    lookup_id: (pd as any).lookup || undefined,
    cascade_from: (pd as any).cascade_from || undefined,
    syntax_language: pd.syntax_language || undefined,
    order,
  };
};


interface NodeTypeFormModalProps {
  open: boolean;
  onClose: () => void;
  nodeType?: NodeType | null;
  tenantId: string;
  allNodeTypes: NodeType[];
  onChange?: (updated: Partial<NodeType>) => void;
}

export const NodeTypeFormModal: React.FC<NodeTypeFormModalProps> = ({
  open,
  onClose,
  nodeType,
  tenantId,
  allNodeTypes,
  onChange,
}) => {
  const isEditMode = !!nodeType;
  const createNodeType = useCreateNodeType();
  const updateNodeType = useUpdateNodeType();
  const queryClient = useQueryClient();

  const [formData, setFormData] = useState({
    catalog_type_name: '',
    description: '',
    parent_type_id: '',
    is_active: true,
  });
  const [propertiesSchema, setPropertiesSchema] = useState<PropertyDef[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [snackOpen, setSnackOpen] = useState(false);
  const [snackMessage, setSnackMessage] = useState('');
    // For optimistic updates rollback
    let previousList: NodeType[] | undefined;

  // Track which nodeType we've initialized properties for to prevent re-initialization
  const initializedNodeTypeId = useRef<string | null>(null);

  // Filter out current node to prevent circular parent references
  const parentOptions = allNodeTypes.filter((nt) => nt.id !== nodeType?.id);

  // Lazy-load professional search input to reduce initial bundle size
  const ProfessionalSearchInput = lazy(() => import('../../components/ProfessionalSearchInput'));

  useEffect(() => {
    if (nodeType) {
      // Always initialize local state from nodeType for consistency
      setFormData({
        catalog_type_name: nodeType.catalog_type_name || '',
        description: nodeType.description || '',
        parent_type_id: nodeType.parent_type_id || '',
        is_active: nodeType.is_active ?? true,
      });
      // Only initialize properties schema once per nodeType to prevent overwriting user changes
      if (open && initializedNodeTypeId.current !== nodeType.id) {
        try {
          const props = (nodeType.properties || [])
            .sort((a, b) => a.order - b.order) // Preserve order from backend
            .map(nodePropertyToPropertyDef);
          setPropertiesSchema(props);
          initializedNodeTypeId.current = nodeType.id;
        } catch (e) {
          setPropertiesSchema([]);
          initializedNodeTypeId.current = nodeType.id;
        }
      }
    } else {
      setFormData({
        catalog_type_name: '',
        description: '',
        parent_type_id: '',
        is_active: true,
      });
      // Only reset properties schema when opening for create mode
      if (open) {
        setPropertiesSchema([]);
        initializedNodeTypeId.current = null;
      }
    }
    setErrors({});
  }, [nodeType, open]);

  // Helper to update either local state (create mode) or notify parent (edit mode)
  const updateField = (field: string, value: any) => {
    setFormData((s) => ({ ...s, [field]: value }));
    if (nodeType && onChange) {
      onChange({ [field]: value });
    }
  };

  // Effective values: always use local formData state for consistency
  const effectiveCatalogTypeName = formData.catalog_type_name;
  const effectiveDescription = formData.description;
  const effectiveParentTypeId = formData.parent_type_id;
  const effectiveIsActive = formData.is_active;

  const validateForm = (): Record<string, string> => {
    const newErrors: Record<string, string> = {};
    // Use effective values so validation works in controlled (edit) mode.
    if (!effectiveCatalogTypeName.trim()) {
      newErrors.catalog_type_name = 'Type name is required';
    } else if (effectiveCatalogTypeName.length < 2) {
      newErrors.catalog_type_name = 'Type name must be at least 2 characters';
    } else if (!/^[a-z_][a-z0-9_]*$/.test(effectiveCatalogTypeName)) {
      newErrors.catalog_type_name = 'Type name must be lowercase with underscores (e.g., business_term, semantic_column)';
    }

    if (!effectiveDescription.trim()) {
      newErrors.description = 'Description is required';
    } else if (effectiveDescription.length < 10) {
      newErrors.description = 'Description must be at least 10 characters';
    }

    // Validate properties
    const propertyNames = new Set<string>();
    propertiesSchema.forEach((prop, index) => {
      if (!prop.name.trim()) {
        newErrors[`properties.${index}.name`] = 'Property name is required';
      } else if (!/^[a-z_][a-z0-9_]*$/.test(prop.name)) {
        newErrors[`properties.${index}.name`] = 'Property name must be lowercase with underscores';
      } else if (propertyNames.has(prop.name)) {
        newErrors[`properties.${index}.name`] = 'Property names must be unique';
      } else {
        propertyNames.add(prop.name);
      }
      if (!prop.label?.trim()) {
        newErrors[`properties.${index}.label`] = 'Property label is required';
      }
    });

    setErrors(newErrors);
    return newErrors;
  };

  const handleSubmit = async () => {
    // Debug: capture effective values & local state to help diagnose validation issues
    // Toggle detailed logging via ENABLE_MODAL_DEBUG for noisy output control
    const ENABLE_MODAL_DEBUG = true;
    if (ENABLE_MODAL_DEBUG) {
      devDebug('[NodeTypeFormModal] submit debug', {
        effectiveCatalogTypeName,
        effectiveDescription,
        effectiveParentTypeId,
        effectiveIsActive,
        formData,
        propertiesSchema,
        nodeType,
      });
    }

    const validationErrors = validateForm();
    const isValid = Object.keys(validationErrors).length === 0;
    if (!isValid) {
      if (ENABLE_MODAL_DEBUG) {
        devDebug('[NodeTypeFormModal] validation failed', { validationErrors });
      }
      return;
    }

    try {
      // map local PropertyDef to the project's NodeProperty shape
      const mappedProperties: NodeProperty[] = (propertiesSchema || []).map(propertyDefToNodeProperty);

      if (ENABLE_MODAL_DEBUG) {
        devDebug('[NodeTypeFormModal] Payload:', {
          catalog_type_name: effectiveCatalogTypeName,
          description: effectiveDescription,
          parent_type_id: effectiveParentTypeId || null,
          is_active: effectiveIsActive,
          config: nodeType?.config || {},
          properties: mappedProperties,
        });
      }

      const payload = {
        catalog_type_name: effectiveCatalogTypeName,
        description: effectiveDescription,
        parent_type_id: effectiveParentTypeId || null,
        is_active: effectiveIsActive,
        // keep existing config separate from properties
        config: nodeType?.config || {},
        // store properties in top-level `properties` JSONB field
        properties: mappedProperties,
      };

      // Diagnostic: ensure mutation functions exist and log them
      if (ENABLE_MODAL_DEBUG) {
        devDebug('[NodeTypeFormModal] Mutations', {
          createNodeType: !!createNodeType?.mutateAsync,
          updateNodeType: !!updateNodeType?.mutateAsync,
        });
      }

      // Optimistic update: snapshot current list data
      const listKey = nodeTypesKeys.list(tenantId);
      previousList = queryClient.getQueryData(listKey) as NodeType[] | undefined;
      // Build optimistic object
      const optimisticNode: Partial<NodeType> = {
        catalog_type_name: effectiveCatalogTypeName,
        description: effectiveDescription,
        parent_type_id: effectiveParentTypeId || null,
        is_active: effectiveIsActive,
        properties: mappedProperties,
      };

      // Apply optimistic update to list cache
      if (previousList) {
        queryClient.setQueryData(listKey, previousList.map((nt) => (nt.id === nodeType?.id ? { ...nt, ...optimisticNode } : nt)));
      }

      let resp = null as any;
      if (isEditMode && nodeType) {
        if (!updateNodeType || typeof updateNodeType.mutateAsync !== 'function') {
          devError('[NodeTypeFormModal] updateNodeType.mutateAsync is not available', updateNodeType);
          throw new Error('Update mutation not available');
        }

        // Call the mutation and log response for debugging
        // This should produce an XHR/fetch entry in the Network tab
        devDebug('[NodeTypeFormModal] Calling updateNodeType.mutateAsync', { id: nodeType.id, tenantId, data: payload });
        resp = await updateNodeType.mutateAsync({ id: nodeType.id, tenantId, data: payload });
        if (ENABLE_MODAL_DEBUG) {
          devDebug('[NodeTypeFormModal] updateNodeType response', resp);
        }
      } else {
        if (!createNodeType || typeof createNodeType.mutateAsync !== 'function') {
          devError('[NodeTypeFormModal] createNodeType.mutateAsync is not available', createNodeType);
          throw new Error('Create mutation not available');
        }

        devDebug('[NodeTypeFormModal] Calling createNodeType.mutateAsync', { payload: { ...payload, tenant_id: tenantId } });
        resp = await createNodeType.mutateAsync({ ...payload, tenant_id: tenantId });
        if (ENABLE_MODAL_DEBUG) {
          devDebug('[NodeTypeFormModal] createNodeType response', resp);
        }
      }

      // Ensure UI updates immediately: invalidate relevant queries and notify parent
      try {
        if (resp && resp.tenant_id) {
          queryClient.invalidateQueries({ queryKey: nodeTypesKeys.list(resp.tenant_id) });
          queryClient.invalidateQueries({ queryKey: nodeTypesKeys.detail(resp.id, resp.tenant_id) });
        } else {
          // Fallback: use local tenantId to invalidate list
          queryClient.invalidateQueries({ queryKey: nodeTypesKeys.list(tenantId) });
        }
      } catch (e) {
        devWarn('[NodeTypeFormModal] Query invalidation error', e);
      }

      // Notify parent so it can update its editing state immediately
      if (onChange) {
        if (resp) {
          onChange(resp as Partial<NodeType>);
        } else {
          // If server returned no body, send a minimal partial update so parent can refresh
          onChange({ properties: mappedProperties });
        }
      }

      // Show toast
      setSnackMessage(isEditMode ? 'Node type updated' : 'Node type created');
      setSnackOpen(true);
      onClose();
    } catch (error) {
      devError('[NodeTypeFormModal] Mutation error:', error);
      setErrors({
        submit: error instanceof Error ? error.message : 'Failed to save node type',
      });
      // rollback optimistic update
      try {
        const listKey = nodeTypesKeys.list(tenantId);
        if (previousList) {
          queryClient.setQueryData(listKey, previousList);
        }
      } catch (e) {
        devWarn('[NodeTypeFormModal] rollback failed', e);
      }
    }
  };

  const selectedParent = parentOptions.find((nt) => nt.id === formData.parent_type_id);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          boxShadow: 24,
        },
      }}
    >
      <DialogTitle
        sx={{
          pb: 1,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
        }}
      >
        <Stack direction="row" spacing={1.5} alignItems="center">
          <CategoryIcon />
          <Typography variant="h6" component="div" sx={{ fontWeight: 600 }}>
            {isEditMode ? 'Edit Node Type' : 'Create New Node Type'}
          </Typography>
        </Stack>
        <IconButton
          edge="end"
          color="inherit"
          onClick={onClose}
          aria-label="close"
          sx={{ '&:hover': { bgcolor: 'primary.dark' } }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ pt: 3, pb: 2 }}>
        <Stack spacing={3}>
          {/* Info Alert */}
          <Alert severity="info" icon={<InfoOutlinedIcon />} sx={{ borderRadius: 1 }}>
            Define an entity type in your business glossary. For example: <strong>business_term</strong>, <strong>semantic_column</strong>, <strong>table</strong>
          </Alert>

          {errors.submit && (
            <Alert severity="error" sx={{ borderRadius: 1 }}>
              {errors.submit}
            </Alert>
          )}

          {/* Hierarchy Visualization */}
          {formData.parent_type_id && selectedParent && (
            <Paper
              elevation={0}
              sx={{
                p: 3,
                bgcolor: 'grey.50',
                border: 1,
                borderColor: 'divider',
                borderRadius: 2,
              }}
            >
              <Stack direction="row" spacing={2} alignItems="center" justifyContent="center">
                <Chip
                  icon={<AccountTreeIcon />}
                  label={selectedParent.catalog_type_name}
                  color="secondary"
                  variant="outlined"
                  sx={{ fontWeight: 600, fontSize: '0.875rem', px: 1 }}
                />
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                  contains
                </Typography>
                <Chip
                  icon={<CategoryIcon />}
                  label={formData.catalog_type_name || 'new type'}
                  color="primary"
                  variant="outlined"
                  sx={{ fontWeight: 600, fontSize: '0.875rem', px: 1 }}
                />
              </Stack>
              <Typography variant="caption" color="text.secondary" textAlign="center" display="block" mt={1}>
                Hierarchy: {selectedParent.catalog_type_name} → {formData.catalog_type_name || 'new type'}
              </Typography>
            </Paper>
          )}

          {/* Type Name Field */}
          <TextField
            label="Type Name"
            fullWidth
            required
            value={effectiveCatalogTypeName}
            onChange={(e) => updateField('catalog_type_name', e.target.value)}
            error={!!errors.catalog_type_name}
            helperText={
              errors.catalog_type_name ||
              'The internal name for this type (e.g., business_term, semantic_column). Use lowercase with underscores.'
            }
            placeholder="business_term"
            disabled={isEditMode}
            InputLabelProps={{ shrink: true }}
          />

          {/* Description Field */}
          <TextField
            label="Description"
            fullWidth
            required
            multiline
            rows={3}
            value={effectiveDescription}
            onChange={(e) => updateField('description', e.target.value)}
            error={!!errors.description}
            helperText={errors.description || 'Describe what this node type represents in your business glossary'}
            placeholder="Represents a business term used in the organization's vocabulary"
            InputLabelProps={{ shrink: true }}
          />

          <Divider sx={{ my: 1 }} />

          <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: -1 }}>
            Hierarchy (Optional)
          </Typography>

          {/* Parent Type Selection (professional search) */}
          <Suspense
            fallback={
              <TextField
                label="Parent Node Type"
                helperText="Loading search..."
                placeholder="Search parent types"
                fullWidth
                disabled
              />
            }
          >
            <ProfessionalSearchInput
              placeholder="Search parent types..."
              data={parentOptions.map((opt) => ({ id: opt.id, text: opt.catalog_type_name, subtext: opt.description, payload: opt }))}
              initialSelected={parentOptions.find((nt) => nt.id === effectiveParentTypeId) ? { id: String(effectiveParentTypeId), text: parentOptions.find((nt) => nt.id === effectiveParentTypeId)!.catalog_type_name, payload: parentOptions.find((nt) => nt.id === effectiveParentTypeId)! } : null}
              onSelect={(payload) => {
                if (!payload) {
                  updateField('parent_type_id', '');
                  return;
                }
                const nt = payload as any;
                updateField('parent_type_id', String(nt.id ?? ''));
              }}
              debounceMs={200}
            />
          </Suspense>

          <Divider sx={{ my: 1 }} />

          {/* Active Switch */}
          <FormControlLabel
            control={
              <Switch
                checked={effectiveIsActive}
                onChange={(e) => updateField('is_active', e.target.checked)}
                color="primary"
              />
            }
            label={
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>
                  Active
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Only active node types can be used to create glossary entries
                </Typography>
              </Box>
            }
          />
          <Divider sx={{ my: 1 }} />

          <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: -1 }}>
            Properties (Optional)
          </Typography>

          <PropertySchemaEditor 
            value={propertiesSchema} 
            onChange={(next) => {
              setPropertiesSchema(next);
              if (nodeType && onChange) {
                // notify parent to update properties (map PropertyDef -> NodeProperty)
                const mappedProperties = (next || []).map(propertyDefToNodeProperty);
                onChange({ properties: mappedProperties });
              }
            }} 
          />
        </Stack>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, bgcolor: 'grey.50', gap: 1 }}>
        <Button onClick={onClose} variant="outlined" sx={{ textTransform: 'none', fontWeight: 600 }}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={createNodeType.isPending || updateNodeType.isPending}
          sx={{ textTransform: 'none', fontWeight: 600, px: 3 }}
        >
          {createNodeType.isPending || updateNodeType.isPending
            ? 'Saving...'
            : isEditMode
            ? 'Update Node Type'
            : 'Create Node Type'}
        </Button>
      </DialogActions>
      <Snackbar open={snackOpen} autoHideDuration={4000} onClose={() => setSnackOpen(false)} message={snackMessage} />
    </Dialog>
  );
};
