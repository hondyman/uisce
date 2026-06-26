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
  Chip,
  Stack,
  Divider,
  IconButton,
  Paper,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import { useCreateEdgeType, useUpdateEdgeType, edgeTypesKeys } from '../../api/edgeTypes';
import { useQueryClient } from '@tanstack/react-query';
import { useNodeTypes } from '../../api/nodeTypes';
import type { EdgeType, EdgeProperty } from '../../types/edgeTypes';
import type { NodeType } from '../../types/nodeTypes';
import PropertySchemaEditor, { PropertyDef } from '../../components/properties/PropertySchemaEditor';
import { devDebug, devWarn, devError } from '../../utils/devLogger';

// Helper to convert EdgeProperty to PropertyDef for the editor
const edgePropertyToPropertyDef = (ep: EdgeProperty): PropertyDef => ({
  name: ep.name,
  label: ep.label,
  data_type: ep.data_type === 'integer' || ep.data_type === 'float' ? 'number' : 
             ep.data_type === 'text' || ep.data_type === 'json' ? 'string' : 
             ep.data_type,
  original_data_type: ep.data_type, // Preserve original
  input_type: ep.input_type === 'date-picker' ? 'date' : 
              ep.input_type === 'textarea' || ep.input_type === 'number' || ep.input_type === 'json-editor' ? 'text' :
              ep.input_type,
  original_input_type: ep.input_type, // Preserve original
  required: !ep.nullable,
  options: ep.options || [],
  syntax_language: ep.syntax_language || null,
  is_array: ep.is_array || false,
  lookup_node_type_id: ep.lookup_config?.node_type_id || null, // Map from lookup_config
});

// Helper to convert PropertyDef to EdgeProperty for the backend
const propertyDefToEdgeProperty = (pd: PropertyDef, order: number): EdgeProperty => {
  const mappedDataType = ((): EdgeProperty['data_type'] => {
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
        return pd.data_type as EdgeProperty['data_type'];
    }
  })();

  const finalDataType = pd.original_data_type && pd.original_data_type === mappedDataType
    ? pd.original_data_type
    : mappedDataType;

  const mappedInputType = ((): EdgeProperty['input_type'] => {
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
        return pd.input_type as EdgeProperty['input_type'];
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
    syntax_language: pd.syntax_language || undefined,
    order,
    is_array: pd.is_array || false,
    lookup_config: pd.lookup_node_type_id ? { node_type_id: pd.lookup_node_type_id } : undefined, // Map to lookup_config
  };
};


interface EdgeTypeFormModalProps {
  open: boolean;
  onClose: () => void;
  edgeType?: EdgeType | null;
  tenantId: string;
  onChange?: (updated: Partial<EdgeType>) => void;
}

export const EdgeTypeFormModal: React.FC<EdgeTypeFormModalProps> = ({
  open,
  onClose,
  edgeType,
  tenantId,
  onChange,
}) => {
  const isEditMode = !!edgeType;
  const { data: nodeTypes = [] } = useNodeTypes(tenantId);
  // Lazy-load professional search input for better bundle splitting
  const ProfessionalSearchInput = lazy(() => import('../../components/ProfessionalSearchInput'));
  const createEdgeType = useCreateEdgeType();
  const updateEdgeType = useUpdateEdgeType();
  const queryClient = useQueryClient();

  const [formData, setFormData] = useState(() => ({
    edge_type_name: edgeType?.edge_type_name || '',
    description: edgeType?.description || '',
    subject_node_type_id: edgeType?.subject_node_type_id || '',
    object_node_type_id: edgeType?.object_node_type_id || '',
    is_active: edgeType?.is_active ?? true,
  }));

  const [propertiesSchema, setPropertiesSchema] = useState<PropertyDef[]>([]);

  const [errors, setErrors] = useState<Record<string, string>>({});

  // Track which edgeType we've initialized properties for to prevent re-initialization
  const initializedEdgeTypeId = useRef<string | null>(null);

  useEffect(() => {
    if (edgeType) {
      setFormData({
        edge_type_name: edgeType.edge_type_name || '',
        description: edgeType.description || '',
        subject_node_type_id: edgeType.subject_node_type_id || '',
        object_node_type_id: edgeType.object_node_type_id || '',
        is_active: edgeType.is_active ?? true,
      });
      // Only initialize properties schema once per edgeType to prevent overwriting user changes
      if (open && initializedEdgeTypeId.current !== edgeType.id) {
        try {
          const props = (edgeType.properties || [])
            .sort((a, b) => a.order - b.order) // Preserve order from backend
            .map(edgePropertyToPropertyDef);
          setPropertiesSchema(props);
          initializedEdgeTypeId.current = edgeType.id;
        } catch (e) {
          setPropertiesSchema([]);
          initializedEdgeTypeId.current = edgeType.id;
        }
      }
    } else {
      setFormData({
        edge_type_name: '',
        description: '',
        subject_node_type_id: '',
        object_node_type_id: '',
        is_active: true,
      });
      // Only reset properties schema when opening for create mode
      if (open) {
        setPropertiesSchema([]);
        initializedEdgeTypeId.current = null;
      }
    }
    setErrors({});
  }, [edgeType, open]);

  // Effective values (local state is source-of-truth once initialized)
  const effectiveEdgeTypeName = formData.edge_type_name;
  const effectiveDescription = formData.description;
  const effectiveSubjectId = formData.subject_node_type_id;
  const effectiveObjectId = formData.object_node_type_id;
  const effectiveIsActive = formData.is_active;

  const validateForm = (): Record<string, string> => {
    const newErrors: Record<string, string> = {};
    // Use effective values so controlled (edit) mode validates correctly
    if (!effectiveEdgeTypeName.trim()) {
      newErrors.edge_type_name = 'Name is required';
    } else if (effectiveEdgeTypeName.length < 2) {
      newErrors.edge_type_name = 'Name must be at least 2 characters';
    } else if (!/^[a-z_][a-z0-9_]*$/.test(effectiveEdgeTypeName)) {
      newErrors.edge_type_name = 'Name must be lowercase with underscores (e.g., has_parent, contains)';
    }

    if (!effectiveDescription.trim()) {
      newErrors.description = 'Description is required';
    }

    if (!effectiveSubjectId) {
      newErrors.subject_node_type_id = 'Subject node type is required';
    }

    // Validate subject ID format (UUID) and that it exists in nodeTypes
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    if (effectiveSubjectId && !uuidRegex.test(effectiveSubjectId)) {
      newErrors.subject_node_type_id = 'Invalid subject node type id';
    } else if (effectiveSubjectId && !nodeTypes.find((nt) => nt.id === effectiveSubjectId)) {
      newErrors.subject_node_type_id = 'Subject node type is not recognized';
    }

    if (!effectiveObjectId) {
      newErrors.object_node_type_id = 'Object node type is required';
    }

    // Validate object ID format (UUID) and that it exists in nodeTypes
    if (effectiveObjectId && !uuidRegex.test(effectiveObjectId)) {
      newErrors.object_node_type_id = 'Invalid object node type id';
    } else if (effectiveObjectId && !nodeTypes.find((nt) => nt.id === effectiveObjectId)) {
      newErrors.object_node_type_id = 'Object node type is not recognized';
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
    const ENABLE_MODAL_DEBUG = true;
    if (ENABLE_MODAL_DEBUG) {
      devDebug('[EdgeTypeFormModal] submit debug', {
        effectiveEdgeTypeName,
        effectiveDescription,
        effectiveSubjectId,
        effectiveObjectId,
        effectiveIsActive,
        formData,
        propertiesSchema,
        edgeType,
      });
    }

    const validationErrors = validateForm();
    const isValid = Object.keys(validationErrors).length === 0;
    if (!isValid) {
      if (ENABLE_MODAL_DEBUG) {
        devDebug('[EdgeTypeFormModal] validation failed', { validationErrors });
      }
      return;
    }

    try {
      // map schema to edge property shape expected by backend using the helper
      const mappedProperties = (propertiesSchema || []).map(propertyDefToEdgeProperty);

      if (ENABLE_MODAL_DEBUG) {
        devDebug('[EdgeTypeFormModal] Payload:', {
          edge_type_name: effectiveEdgeTypeName,
          description: effectiveDescription,
          subject_node_type_id: effectiveSubjectId,
          object_node_type_id: effectiveObjectId,
          is_active: effectiveIsActive,
          properties: mappedProperties,
        });
      }

  let resp: any = null;
      if (isEditMode && edgeType) {
        resp = await updateEdgeType.mutateAsync({
          id: edgeType.id,
          tenantId,
          data: {
            edge_type_name: effectiveEdgeTypeName,
            description: effectiveDescription,
            subject_node_type_id: effectiveSubjectId,
            object_node_type_id: effectiveObjectId,
            is_active: effectiveIsActive,
            properties: mappedProperties,
          },
        });
      } else {
        resp = await createEdgeType.mutateAsync({
          edge_type_name: effectiveEdgeTypeName,
          description: effectiveDescription,
          subject_node_type_id: effectiveSubjectId,
          object_node_type_id: effectiveObjectId,
          is_active: effectiveIsActive,
          tenant_id: tenantId,
          properties: mappedProperties,
        });
      }

      try {
        if (resp && resp.tenant_id) {
          queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(resp.tenant_id) });
          queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(resp.id, resp.tenant_id) });
        } else {
          queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(tenantId) });
        }
      } catch (e) {
        devWarn('[EdgeTypeFormModal] Query invalidation error', e);
      }

      // Notify parent so it can update its editing state immediately
      if (onChange) {
        if (resp) {
          onChange(resp as Partial<EdgeType>);
        } else {
          onChange({ properties: mappedProperties });
        }
      }
      onClose();
    } catch (error) {
      devError('[EdgeTypeFormModal] Mutation error:', error);
      // If backend returned a 409 conflict for edge_type_name, show a field-specific error
      const status = (error as any)?.status;
      const msg = error instanceof Error ? error.message : 'Failed to save edge type';
      if (status === 409 || (msg && msg.toLowerCase().includes('edge_type_name already exists'))) {
        setErrors({ edge_type_name: 'An edge type with this name already exists for this tenant' });
      } else {
        setErrors({ submit: msg });
      }
    }
  };

  const subjectNodeType = nodeTypes.find((nt: NodeType) => nt.id === formData.subject_node_type_id);
  const objectNodeType = nodeTypes.find((nt: NodeType) => nt.id === formData.object_node_type_id);

  // Derive if the form is currently valid for enabling the submit button
  const isSubmitDisabled = createEdgeType.isPending || updateEdgeType.isPending || Object.keys(errors).length > 0;

  // Re-validate live as the user edits useful fields (to enable/disable submit & show inline errors)
  useEffect(() => {
    // run a validation and set errors on shallow changes
    validateForm();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [formData, propertiesSchema, nodeTypes]);

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
      <DialogTitle sx={{ 
        pb: 1, 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        bgcolor: 'primary.main',
        color: 'primary.contrastText',
      }}>
        <Typography variant="h6" component="div" sx={{ fontWeight: 600 }}>
          {isEditMode ? 'Edit Edge Type' : 'Create New Edge Type'}
        </Typography>
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
            Define a relationship between two node types. For example: "table" <strong>contains</strong> "column"
          </Alert>

          {errors.submit && (
            <Alert severity="error" sx={{ borderRadius: 1 }}>
              {errors.submit}
            </Alert>
          )}

          {/* Relationship Visualization */}
          {(formData.subject_node_type_id && formData.object_node_type_id) && (
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
                  label={subjectNodeType?.catalog_type_name || 'Subject'}
                  color="primary"
                  variant="outlined"
                  sx={{ fontWeight: 600, fontSize: '0.875rem', px: 1 }}
                />
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <ArrowForwardIcon color="action" />
                  <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 600 }}>
                    {formData.edge_type_name || 'edge_type_name'}
                  </Typography>
                  <ArrowForwardIcon color="action" />
                </Box>
                <Chip
                  label={objectNodeType?.catalog_type_name || 'Object'}
                  color="secondary"
                  variant="outlined"
                  sx={{ fontWeight: 600, fontSize: '0.875rem', px: 1 }}
                />
              </Stack>
            </Paper>
          )}

          {/* Name Field */}
          <TextField
            label="Name"
            fullWidth
            required
            value={effectiveEdgeTypeName}
            onChange={(e) => setFormData({ ...formData, edge_type_name: e.target.value })}
            error={!!errors.edge_type_name}
            helperText={
              errors.edge_type_name || 
              'The relationship name (e.g., contains, has_parent, references). Use lowercase with underscores.'
            }
            placeholder="contains"
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
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            error={!!errors.description}
            helperText={errors.description || 'Describe this relationship and when it should be used'}
            placeholder="Describes the containment relationship where a parent entity contains child entities"
            InputLabelProps={{ shrink: true }}
          />

          <Divider sx={{ my: 1 }} />

          <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: -1 }}>
            Relationship Direction
          </Typography>

          <Suspense
            fallback={
              <TextField
                label="Subject Node Type (From)"
                helperText="Loading search..."
                placeholder="Search node types"
                fullWidth
                disabled
              />
            }
          >
            <ProfessionalSearchInput
              placeholder="Search subject node types..."
              data={nodeTypes.map((nt) => ({ id: nt.id, text: nt.catalog_type_name, subtext: nt.description, payload: nt }))}
              initialSelected={nodeTypes.find((nt) => nt.id === formData.subject_node_type_id) ? { id: formData.subject_node_type_id, text: nodeTypes.find((nt) => nt.id === formData.subject_node_type_id)!.catalog_type_name, payload: nodeTypes.find((nt) => nt.id === formData.subject_node_type_id)! } : null}
              onSelect={(payload) => {
                if (!payload) {
                  setFormData({ ...formData, subject_node_type_id: '' });
                  return;
                }
                const nt = payload as any;
                setFormData({ ...formData, subject_node_type_id: String(nt.id ?? '') });
              }}
              debounceMs={200}
            />
          </Suspense>

          <Suspense
            fallback={
              <TextField
                label="Object Node Type (To)"
                helperText="Loading search..."
                placeholder="Search node types"
                fullWidth
                disabled
              />
            }
          >
            <ProfessionalSearchInput
              placeholder="Search object node types..."
              data={nodeTypes.map((nt) => ({ id: nt.id, text: nt.catalog_type_name, subtext: nt.description, payload: nt }))}
              initialSelected={nodeTypes.find((nt) => nt.id === formData.object_node_type_id) ? { id: formData.object_node_type_id, text: nodeTypes.find((nt) => nt.id === formData.object_node_type_id)!.catalog_type_name, payload: nodeTypes.find((nt) => nt.id === formData.object_node_type_id)! } : null}
              onSelect={(payload) => {
                if (!payload) {
                  setFormData({ ...formData, object_node_type_id: '' });
                  return;
                }
                const nt = payload as any;
                setFormData({ ...formData, object_node_type_id: String(nt.id ?? '') });
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
                onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                color="primary"
              />
            }
            label={
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>
                  Active
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Only active edge types can be used to create relationships
                </Typography>
              </Box>
            }
          />

          <Divider sx={{ my: 1 }} />

          <Typography variant="subtitle2" color="text.secondary" sx={{ fontWeight: 600, mb: -1 }}>
            Properties (Optional)
          </Typography>

          <PropertySchemaEditor value={propertiesSchema} onChange={(next) => setPropertiesSchema(next)} />
        </Stack>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, bgcolor: 'grey.50', gap: 1 }}>
        <Button onClick={onClose} variant="outlined" sx={{ textTransform: 'none', fontWeight: 600 }}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={isSubmitDisabled}
          sx={{ textTransform: 'none', fontWeight: 600, px: 3 }}
        >
          {createEdgeType.isPending || updateEdgeType.isPending
            ? 'Saving...'
            : isEditMode
            ? 'Update Edge Type'
            : 'Create Edge Type'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
