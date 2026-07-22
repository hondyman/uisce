import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Typography,
  Alert,
  CircularProgress,
} from '@mui/material';
import { useNodeTypes } from '../api/nodeTypes';
import { useUpdateTerm } from '../api/glossary';
import DynamicPropertyForm, { PropertyMetadata } from './DynamicPropertyForm';

interface EditSemanticTermDialogProps {
  open: boolean;
  onClose: () => void;
  term: {
    id: string;
    name: string;
    description?: string;
    properties?: any;
    tenant_id?: string;
    tenant_datasource_id?: string;
    catalog_type_name?: string;
  };
  onSave?: () => void;
}

const EditSemanticTermDialog: React.FC<EditSemanticTermDialogProps> = ({
  open,
  onClose,
  term,
  onSave
}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [properties, setProperties] = useState<Record<string, any>>({});
  
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const updateTermMutation = useUpdateTerm();

  // Fetch node types to get the schema for the catalog node type
  // Fallback to term's tenant_id if available, otherwise default
  const { data: nodeTypes, isLoading: isLoadingTypes } = useNodeTypes(term?.tenant_id || '');

  const isSemantic = term?.catalog_type_name === 'semantic_term';

  const termType = useMemo(() => {
    if (!nodeTypes || !term?.catalog_type_name) return null;
    const targetName = String(term.catalog_type_name).toLowerCase();
    return (nodeTypes as any[]).find((nt) => {
      const ntName = String(nt.catalog_type_name || '').toLowerCase();
      return ntName === targetName || ntName.replace('_', ' ') === targetName.replace('_', ' ');
    });
  }, [nodeTypes, term?.catalog_type_name]);

  const propertyMetadata = useMemo((): PropertyMetadata[] => {
    const base = (termType?.properties || []) as PropertyMetadata[];
    if (!isSemantic) {
      return [...base].sort((a, b) => (a.order ?? 999) - (b.order ?? 999));
    }
    
    const existingNames = new Set(base.map(p => p.name));

    const coreProperties: PropertyMetadata[] = [
        { name: 'semantic_type', label: 'Semantic Type', data_type: 'string', input_type: 'text', order: 5, description: 'dimension, measure, or time_dimension' },
        { 
          name: 'role', 
          label: 'Field Role', 
          data_type: 'string', 
          input_type: 'select', 
          order: 5.5, 
          description: 'Special role for temporal or analytical logic',
          options: [
            { label: 'Dimension (Default)', value: 'DIMENSION' },
            { label: 'Measure (Numeric)', value: 'MEASURE' },
            { label: 'Validity Start (Effective Start)', value: 'VALIDITY_START' },
            { label: 'Validity End (Effective End)', value: 'VALIDITY_END' },
            { label: 'Event Date (Single Timeline)', value: 'EVENT_DATE' },
            { label: 'Partition Key (ID Field)', value: 'PARTITION_KEY' }
          ]
        },
        { name: 'data_type', label: 'Data Type', data_type: 'string', input_type: 'text', order: 6 },
        { name: 'is_effective_dated', label: 'Effective Dated', data_type: 'boolean', input_type: 'checkbox', order: 7, description: 'Inject temporal filters (valid_from/to) when querying' },
        { name: 'sql', label: 'SQL', data_type: 'string', input_type: 'textarea', order: 10, placeholder: '${CUBE}.column_name', description: 'SQL definition using Cube.js syntax' },
        { name: 'case', label: 'Case (SQL Case)', data_type: 'string', input_type: 'textarea', order: 11, description: 'Optional CASE statement logic' },
    ];

    const missing = coreProperties.filter(p => !existingNames.has(p.name));
    return [...base, ...missing].sort((a, b) => (a.order ?? 999) - (b.order ?? 999));
  }, [termType, isSemantic]);

  useEffect(() => {
    if (open && term) {
      setName(term.name || '');
      setDescription(term.description || '');
      
      // Initialize properties
      let initialProps: Record<string, any> = {};
      
      // If properties is string (JSON), parse it
      if (typeof term.properties === 'string') {
          try {
             initialProps = JSON.parse(term.properties);
          } catch (e) {
             initialProps = {};
          }
      } else if (typeof term.properties === 'object') {
          initialProps = { ...term.properties };
      }

      // Ensure defaults for metadata fields if missing
      propertyMetadata.forEach(p => {
         if (p.data_type === 'boolean' || p.input_type === 'checkbox') {
             if (initialProps[p.name] === undefined) initialProps[p.name] = false;
         }
      });

      setProperties(initialProps);
      setError(null);
    }
  }, [open, term, propertyMetadata]);

  const handlePropertyChange = (field: string, value: any) => {
    setProperties(prev => ({ ...prev, [field]: value }));
  };

  const handleSave = async () => {
    setError(null);
    setSaving(true);

    try {
      // Update term using the unified React Query mutation hook
      await updateTermMutation.mutateAsync({
        id: term.id,
        updates: {
          node_name: name,
          description: description,
          properties: properties,
          catalog_type: term.catalog_type_name,
        },
      });

      // Trigger refresh before closing to ensure UI updates
      if (onSave) {
        onSave();
      }
      
      // Small delay to allow parent to refresh before closing
      setTimeout(() => {
        onClose();
      }, 100);
    } catch (err: any) {
      setError(err.message || 'Failed to save changes');
    } finally {
      setSaving(false);
    }
  };

  const title = isSemantic ? 'Edit Semantic Term' : 'Edit Business Term';
  const nameHelper = isSemantic ? 'Unique identifier for the dimension' : 'Name of the business term';

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {isLoadingTypes ? (
           <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
             <CircularProgress />
           </Box>
         ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            {/* Basic Info */}
            <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 1 }}>
              Basic Information
            </Typography>
            
            <TextField
              label="Name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              fullWidth
              required
              helperText={nameHelper}
            />

            <TextField
              label="Description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              fullWidth
              multiline
              rows={2}
              helperText="Documentation description"
            />

            {/* Dynamic Properties */}
            {propertyMetadata.length > 0 && (
                <>
                    <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 2 }}>
                    Properties
                    </Typography>
                    
                    <DynamicPropertyForm 
                        properties={propertyMetadata}
                        values={properties}
                        onChange={handlePropertyChange}
                    />
                </>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={saving}>
          {saving ? 'Saving...' : 'Save Changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default EditSemanticTermDialog;
