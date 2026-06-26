import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Typography,
  Checkbox,
  FormControlLabel,
  Autocomplete,
} from '@mui/material';
import { Lock as LockIcon } from '@mui/icons-material';
import JsonMonacoEditor from './editors/JsonMonacoEditor';
import ArrayChips from './editors/ArrayChips';
import { CatalogNode, useBusinessTerms } from '../api/glossary';
import { formatProperties, validateProperty, getJsonSchemaForProperty } from '../utils/propertyHelpers';
import PropertyEditor from './properties/PropertyEditor';
import { useNodeTypes } from '../api/nodeTypes';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';

interface TermFormProps {
  open: boolean;
  onClose: () => void;
  onSave: (term: Partial<CatalogNode>) => Promise<void> | void;
  term?: CatalogNode | null;
  termType: 'business_term' | 'semantic_term';
  loading?: boolean;
  // When true, the term type (business_term | semantic_term) is set externally
  // and should not be editable in this form. Useful when creating from a
  // specific tab so users don't need to pick the node type.
  disableTypeSelection?: boolean;
}

const TermForm: React.FC<TermFormProps> = ({
  open,
  onClose,
  onSave,
  term,
  termType,
  loading = false,
  disableTypeSelection = false,
}) => {
  devDebug('[TermForm] Rendering with props:', { open, term, termType, loading });
  const { tenant } = useTenant();
  // `useNodeTypes` expects a string tenant id; guard against undefined by providing empty string fallback.
  const { data: nodeTypes } = useNodeTypes(tenant?.id || '');
  const { data: businessTerms = [] } = useBusinessTerms();

  const termNodeType = useMemo(() => {
    if (!nodeTypes) return null;
    return (nodeTypes as any[]).find(nt => nt.catalog_type_name === termType);
  }, [nodeTypes, termType]);

  const [formData, setFormData] = useState<{
    node_name: string;
    description: string;
    catalog_type: 'business_term' | 'semantic_term';
    properties: Record<string, any>;
    parent_id?: string | null;
  }>({
    node_name: '',
    description: '',
    catalog_type: termType,
    properties: {},
    parent_id: null,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
      if (term) {
      // Normalize properties to always be an object (not array)
      let normalizedProperties: Record<string, any> = {};
      if (typeof term.properties === 'string') {
        // If it's a JSON string, parse it
        try {
          normalizedProperties = JSON.parse(term.properties as string);
        } catch (e) {
          devWarn('[TermForm] Failed to parse properties as JSON string:', e);
          normalizedProperties = {};
        }
      } else if (Array.isArray(term.properties)) {
        // If it's an array (shouldn't happen, but handle it), convert to empty object
        devWarn('[TermForm] Properties came back as array, converting to object');
        normalizedProperties = {};
      } else if (typeof term.properties === 'object' && term.properties) {
        // It's already an object, use it
        normalizedProperties = term.properties as Record<string, any>;
      }

      setFormData({
        node_name: term.node_name || '',
        description: term.description || '',
        catalog_type: (term.catalog_type as 'business_term' | 'semantic_term') || termType,
        properties: normalizedProperties,
        parent_id: term.parent_id || null,
      });
    } else {
      const initialProperties: Record<string, any> = {};
      if (termNodeType?.properties) {
        (termNodeType.properties as any[]).forEach(prop => {
          if (prop.name === 'name' || prop.name === 'description') return;
          if (prop.default_value !== undefined) {
            initialProperties[prop.name] = prop.default_value;
          } else if (prop.data_type === 'boolean') {
            initialProperties[prop.name] = false;
          } else if (prop.validation?.multiple) {
            initialProperties[prop.name] = [];
          } else {
            initialProperties[prop.name] = '';
          }
        });
      }
      setFormData({
        node_name: '',
        description: '',
        catalog_type: termType,
        properties: initialProperties,
        parent_id: null,
      });
    }
    setErrors({});
  }, [term, termType, open, termNodeType]);

  const validateForm = () => {
    const newErrors: Record<string, string> = {};
    devDebug('[validateForm] Starting validation with formData:', formData);

    if (!formData.node_name.trim()) {
      newErrors.node_name = 'Name is required';
      devDebug('[validateForm] FAILED: node_name is empty');
    }

    if (formData.node_name.length > 255) {
      newErrors.node_name = 'Name must be less than 255 characters';
      devDebug('[validateForm] FAILED: node_name too long');
    }

    if (formData.description && formData.description.length > 1000) {
      newErrors.description = 'Description must be less than 1000 characters';
      devDebug('[validateForm] FAILED: description too long');
    }

    // Property-level validation
    devDebug('[validateForm] Checking properties. termNodeType:', termNodeType, 'properties count:', termNodeType?.properties?.length || 0);
    (termNodeType?.properties || []).forEach((prop: any) => {
      const k = prop.name;
      const v = formData.properties?.[k];
      const cascadeFrom = (prop as any).cascade_from;
      
      devDebug(`[validateForm] Checking property "${k}": value="${v}", nullable=${prop.nullable}, cascadeFrom=${cascadeFrom}`);
      
      // Skip validation for cascaded fields when parent is not selected
      // (they are legitimately disabled and should not block form submission)
      if (cascadeFrom) {
        const parentValue = formData.properties?.[cascadeFrom];
        if (!parentValue) {
          // Parent not selected, so this field is disabled - skip validation
          devDebug(`[validateForm] Skipping "${k}" - parent field not selected`);
          return;
        }
      }

      if (!prop.nullable && (v === undefined || v === null || v === '')) {
        // Skip validation for 'name' - it's handled as node_name at the top level
        if (k === 'name') {
          devDebug(`[validateForm] SKIPPING "${k}" - handled as top-level node_name field`);
          return;
        }
        // For other required metadata properties (type, semantic_type, etc):
        // Let them through without UI values - backend can validate and return errors
        // This allows users to save terms with just a name initially
        devDebug(`[validateForm] ⚠️ ALLOWING required field "${k}" without value - backend will validate`);
        return;
      }

      if ((prop.data_type === 'integer' || prop.data_type === 'float' || prop.input_type === 'number') && v !== undefined && v !== '') {
        const num = Number(v);
        if (Number.isNaN(num)) {
          newErrors[`properties.${k}`] = `${prop.label || k} must be a number`;
          return;
        }
        if (prop.validation?.min !== undefined && num < prop.validation.min) newErrors[`properties.${k}`] = `${prop.label || k} must be >= ${prop.validation.min}`;
        if (prop.validation?.max !== undefined && num > prop.validation.max) newErrors[`properties.${k}`] = `${prop.label || k} must be <= ${prop.validation.max}`;
      }

      if ((prop.input_type === 'text' || prop.data_type === 'string') && typeof v === 'string') {
        if (prop.validation?.minLength !== undefined && v.length < prop.validation.minLength) newErrors[`properties.${k}`] = `${prop.label || k} must be at least ${prop.validation.minLength} characters`;
        if (prop.validation?.maxLength !== undefined && v.length > prop.validation.maxLength) newErrors[`properties.${k}`] = `${prop.label || k} must be at most ${prop.validation.maxLength} characters`;
        if (prop.validation?.pattern) {
          try {
            const re = new RegExp(prop.validation.pattern);
            if (!re.test(v)) newErrors[`properties.${k}`] = `${prop.label || k} must match pattern`;
          } catch (e) {
            // ignore bad patterns
          }
        }
      }

      if (prop.validation?.multiple && Array.isArray(v)) {
        if (prop.validation.minLength !== undefined && v.length < prop.validation.minLength) newErrors[`properties.${k}`] = `${prop.label || k} must have at least ${prop.validation.minLength} items`;
        if (prop.validation.maxLength !== undefined && v.length > prop.validation.maxLength) newErrors[`properties.${k}`] = `${prop.label || k} must have at most ${prop.validation.maxLength} items`;
      }

      if (prop.input_type === 'json-editor' || prop.data_type === 'json') {
        const toCheck = v;
        if (typeof toCheck === 'string' && toCheck.trim() !== '') {
          try {
            JSON.parse(toCheck);
          } catch (e) {
            newErrors[`properties.${k}`] = `${prop.label || k} is not valid JSON`;
          }
        }
      }
    });

    devDebug('[validateForm] Final errors object:', newErrors, 'hasErrors:', Object.keys(newErrors).length > 0);
    setErrors(newErrors);
    const isValid = Object.keys(newErrors).length === 0;
    devDebug('[validateForm] Returning:', isValid);
    return isValid;
  };

  const handleSave = async () => {
    devDebug('[TermForm.handleSave] CALLED - Starting save process');
    devDebug('[TermForm.handleSave] CALLED - Starting save process'); // Extra logging
    
    if (!validateForm()) {
      devDebug('[TermForm.handleSave] Form validation FAILED - not saving');
      devDebug('[TermForm.handleSave] Form validation FAILED - errors:', errors);
      return;
    }

    devDebug('[TermForm.handleSave] Form validation PASSED');
    // Format properties using the node type configuration before saving
    // Prepare complex types (JSON parsing, numbers, arrays) according to metadata
    const prepared: Record<string, any> = { ...formData.properties };
    (termNodeType?.properties || []).forEach((prop: any) => {
      const key = prop.name;
      const raw = prepared[key];
      if (raw === undefined) return;

      if ((prop.data_type === 'integer' || prop.data_type === 'float' || prop.input_type === 'number') && typeof raw === 'string') {
        const num = Number(raw);
        prepared[key] = Number.isNaN(num) ? raw : num;
      }

      if ((prop.input_type === 'json-editor' || prop.data_type === 'json') && typeof raw === 'string') {
        try {
          prepared[key] = JSON.parse(raw);
        } catch (e) {
          // keep as string if parse fails; validation should have prevented save
        }
      }

      if (prop.validation?.multiple && !Array.isArray(raw)) {
        // normalize comma/newline-separated lists to arrays
        if (typeof raw === 'string') {
          prepared[key] = raw.split(/[\n,]+/).map((s: string) => s.trim()).filter(Boolean);
        } else {
          prepared[key] = Array.isArray(raw) ? raw : [raw];
        }
      }
    });

    const formattedProperties = formatProperties(termNodeType?.properties as any[], prepared || {});

    const termData: Partial<CatalogNode> = {
      node_name: formData.node_name.trim(),
      description: formData.description.trim() || undefined,
      catalog_type: formData.catalog_type,
      properties: formattedProperties,
      // Include parent_id for semantic terms (null if not set, or the selected ID)
      ...(formData.catalog_type === 'semantic_term' && { parent_id: formData.parent_id || null }),
    };

    devDebug('[TermForm.handleSave]', { node_name: termData.node_name, catalog_type: formData.catalog_type, parent_id: termData.parent_id, hasParent: !!termData.parent_id });
    devDebug('[TermForm.handleSave] About to call onSave with:', termData);
    devDebug('[TermForm.handleSave] Properties being sent:', JSON.stringify(formattedProperties, null, 2));
    devDebug('[TermForm.handleSave] Prepared properties (before formatting):', JSON.stringify(prepared, null, 2));

    try {
      devDebug('[TermForm.handleSave] Calling onSave...');
      devDebug('[TermForm.handleSave] Calling onSave with termData:', termData);
      await onSave(termData as Partial<CatalogNode>);
      devDebug('[TermForm.handleSave] Save successful');
      devDebug('[TermForm.handleSave] Save successful - closing modal');
      // Close modal after successful save
      handleClose();
    } catch (err: any) {
      devDebug('[TermForm.handleSave] Save FAILED:', err);
      console.error('[TermForm.handleSave] Save FAILED with error:', err);
      // Server returned structured validation errors - map them to field-level UI
      if (err && Array.isArray(err.validation_errors)) {
        const newErrors: Record<string, string> = {};
        (err.validation_errors as any[]).forEach((ve: any) => {
          // If the server returns a field such as `properties.data_type` map directly
          newErrors[ve.field] = ve.message;
        });
        setErrors(prev => ({ ...prev, ...newErrors }));
        return;
      }

      // Re-throw unexpected errors so that parent forms/UI can handle them
      throw err;
    }
  };

  const handleClose = () => {
    devDebug('[TermForm.handleClose] CALLED - closing modal');
    devDebug('[TermForm.handleClose] Closing form modal');
    setFormData({
      node_name: '',
      description: '',
      catalog_type: termType,
      properties: {},
      parent_id: null,
    });
    setErrors({});
    onClose();
  };

  const title = term ? `Edit ${termType === 'business_term' ? 'Business' : 'Semantic'} Term` : `Create New ${termType === 'business_term' ? 'Business' : 'Semantic'} Term`;

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 1 }}>
          <TextField
            fullWidth
            label="Name"
            value={formData.node_name}
            onChange={(e) => setFormData({ ...formData, node_name: e.target.value })}
            error={!!errors.node_name}
            helperText={errors.node_name}
            margin="normal"
            required
            autoFocus
          />

          <TextField
            fullWidth
            label="Description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            error={!!errors.description}
            helperText={errors.description}
            margin="normal"
            multiline
            rows={3}
          />

          <FormControl fullWidth margin="normal">
            {disableTypeSelection || !!term ? (
              // Show a clearer 'locked' indicator when the Type is fixed by
              // context. A Lock icon with a small label makes it obvious to the
              // user that this field is intentionally non-editable.
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 1 }} data-testid="type-locked">
                <LockIcon fontSize="small" color="disabled" />
                <Typography variant="subtitle1">
                  {formData.catalog_type === 'business_term' ? 'Business Term' : 'Semantic Term'} <Typography component="span" sx={{ color: 'text.secondary', ml: 1 }}>(locked)</Typography>
                </Typography>
              </Box>
            ) : (
              <>
                <InputLabel>Type</InputLabel>
                <Select
                  value={formData.catalog_type}
                  onChange={(e) => setFormData({ ...formData, catalog_type: e.target.value as 'business_term' | 'semantic_term' })}
                  label="Type"
                >
                  <MenuItem value="business_term">Business Term</MenuItem>
                  <MenuItem value="semantic_term">Semantic Term</MenuItem>
                </Select>
              </>
            )}
          </FormControl>

          {formData.catalog_type === 'semantic_term' && (() => {
            devDebug('[TermForm] Parent selector - formData.parent_id:', formData.parent_id, 'businessTerms count:', businessTerms?.length);
            const selectedParent = businessTerms && businessTerms.find(bt => bt.id === formData.parent_id) || null;
            return (
              <Autocomplete
                fullWidth
                options={businessTerms || []}
                getOptionLabel={(option) => option?.node_name || ''}
                value={selectedParent}
                onChange={(_, newValue) => {
                  devDebug('[TermForm] Parent changed to:', newValue?.id, '(name:', newValue?.node_name, ')');
                  setFormData({ ...formData, parent_id: newValue?.id || null });
                }}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Parent Business Term"
                    placeholder="Search and select a business term..."
                    margin="normal"
                  />
                )}
              />
            );
          })()}

          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              Properties
            </Typography>
            {termNodeType?.properties?.sort((a: any, b: any) => a.order - b.order).map((prop: any) => {
              const key = prop.name;
              const label = prop.label;

              // Name and description are handled by top-level fields
              if (key === 'name' || key === 'description') {
                devDebug(`[PropertyEditor] SKIPPING "${key}" - handled as top-level field`);
                return null;
              }

              devDebug(`[PropertyEditor] Rendering property "${key}": nullable=${prop.nullable}, has_ui=true`);

              const handlePropertyChange = (value: any) => {
                setFormData(prev => ({
                  ...prev,
                  properties: {
                    ...prev.properties,
                    [key]: value,
                  },
                }));
                // Don't validate on change - only validate on submit
                // This allows users to fill in cascaded fields without blocking the form
              };

              // Use a centralized property editor for all property rendering to
              // keep the form nicely separated and testable.
              return (
                <PropertyEditor
                  key={key}
                  property={prop}
                  value={formData.properties[key]}
                  allProperties={formData.properties}
                  onChange={(v) => handlePropertyChange(v)}
                  error={errors[`properties.${key}`]}
                />
              );
            })}
          </Box>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={loading}>
          Cancel
        </Button>
        <Button
          onClick={handleSave}
          variant="contained"
          disabled={loading || Object.keys(errors).length > 0}
        >
          {loading ? 'Saving...' : 'Save'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default TermForm;

              // The property rendering is handled by `PropertyEditor` above.