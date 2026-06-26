import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  CircularProgress,
  Typography,
} from '@mui/material';
import { useNodeTypes } from '../api/nodeTypes';
import DynamicPropertyForm, { PropertyMetadata } from './DynamicPropertyForm';

interface AddSemanticTermDialogProps {
  open: boolean;
  onClose: () => void;
  tenantId?: string;
  datasourceId: string;
}

const AddSemanticTermDialog: React.FC<AddSemanticTermDialogProps> = ({
  open,
  onClose,
  tenantId = 'default',
  datasourceId,
}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [properties, setProperties] = useState<Record<string, any>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Fetch node types to get the schema for semantic_term
  const { data: nodeTypes, isLoading: isLoadingTypes } = useNodeTypes(tenantId || '');

  const semanticTermType = useMemo(() => {
    if (!nodeTypes) return null;
    return (nodeTypes as any[]).find((nt) => {
      const ntName = String(nt.catalog_type_name || '').toLowerCase();
      return ntName === 'semantic_term' || ntName === 'semantic term';
    });
  }, [nodeTypes]);

  const propertyMetadata = useMemo((): PropertyMetadata[] => {
    return semanticTermType?.properties || [];
  }, [semanticTermType]);

  // Reset form when opening
  useEffect(() => {
    if (open) {
      setName('');
      setDescription('');
      // Initialize properties with defaults if any
      const defaults: Record<string, any> = {};
      propertyMetadata.forEach(p => {
        if (p.data_type === 'boolean' || p.input_type === 'checkbox') {
           defaults[p.name] = false;
        }
      });
      setProperties(defaults);
    }
  }, [open, propertyMetadata]);

  const handlePropertyChange = (field: string, value: any) => {
    setProperties(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleSubmit = async () => {
    if (!name.trim()) return;

    setIsSubmitting(true);
    try {
      const response = await fetch('/api/glossary/terms', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          node_name: name,
          description: description,
          catalog_type: 'semantic_term',
          tenant_datasource_id: datasourceId,
          properties: properties
        }),
      });

      if (response.ok) {
        onClose(); // Parent should refresh
      } else {
        alert('Failed to create semantic term');
      }
    } catch (e) {
      console.error(e);
      alert('Error creating semantic term');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Add Semantic Term</DialogTitle>
      <DialogContent>
        {isLoadingTypes ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
            <TextField
              label="Term Name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              fullWidth
              required
              autoFocus
            />
            <TextField
              label="Description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              fullWidth
              multiline
              rows={2}
            />

            {propertyMetadata.length > 0 && (
                <>
                    <Typography variant="subtitle2" sx={{ mt: 1 }}>Properties</Typography>
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
        <Button onClick={onClose}>Cancel</Button>
        <Button 
            onClick={handleSubmit} 
            variant="contained" 
            disabled={!name.trim() || isSubmitting}
        >
            {isSubmitting ? 'Creating...' : 'Create'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddSemanticTermDialog;
