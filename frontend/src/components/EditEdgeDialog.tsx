import React, { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormControlLabel,
  Checkbox,
  Box,
  Typography,
  CircularProgress,
  Alert
} from '@mui/material';
import { useUpdateTermEdge } from '../api/glossary';
import { useEdgeTypes } from '../api/edgeTypes';
import { useTenant } from '../contexts/TenantContext';
import { useNodeTypes } from '../api/nodeTypes';

interface EditEdgeDialogProps {
  open: boolean;
  onClose: () => void;
  edge: any; // The edge instance
  onEdgeUpdated?: () => void;
}

const EditEdgeDialog: React.FC<EditEdgeDialogProps> = ({
  open,
  onClose,
  edge,
  onEdgeUpdated
}) => {
  const { tenant } = useTenant();
  const [propertyValues, setPropertyValues] = useState<Record<string, any>>({});
  const [description, setDescription] = useState('');
  const [error, setError] = useState<string | null>(null);

  // APIs
  const { data: edgeTypes, isLoading: loadingEdgeTypes } = useEdgeTypes(tenant?.id || '');
  const { data: nodeTypes } = useNodeTypes(tenant?.id || '');
  const updateEdgeMutation = useUpdateTermEdge();

  // Determine the edge type definition for this edge
  const edgeTypeDefinition = useMemo(() => {
    if (!edge || !edgeTypes || !nodeTypes) return null;

    // We try to match by edge_type_id if available, or by predicate + node types
    if (edge.edge_type_id) {
      return edgeTypes.find(et => et.id === edge.edge_type_id);
    }

    // If no ID, try to match by predicate (relationship name)
    // Note: This is a best-effort matching if the edge object doesn't have the type ID
    const predicate = edge.edge_type_name || edge.relationship_type || edge.label;
    if (!predicate) return null;

    return edgeTypes.find(et => 
      (et.edge_type_name && et.edge_type_name.toLowerCase() === predicate.toLowerCase()) ||
      (et.edge_type_name && et.edge_type_name.toLowerCase() === predicate.toLowerCase()) ||
      et.id === edge.type // sometimes type is the ID
    );
  }, [edge, edgeTypes, nodeTypes]);

  // Initialize values when edge opens
  useEffect(() => {
    if (open && edge) {
      setDescription(edge.description || '');
      
      const props = edge.properties || {};
      const initialValues: Record<string, any> = { ...props };
      
      // If we found the definition, ensure defaults for missing props
      if (edgeTypeDefinition?.properties) {
        edgeTypeDefinition.properties.forEach(prop => {
          if (initialValues[prop.name] === undefined) {
             initialValues[prop.name] = prop.default_value ?? '';
          }
        });
      }
      
      setPropertyValues(initialValues);
      setError(null);
    }
  }, [open, edge, edgeTypeDefinition]);

  const handleUpdate = () => {
    if (!edge) return;

    updateEdgeMutation.mutate({
      id: edge.id,
      updates: {
        description: description,
        properties: propertyValues
      }
    }, {
      onSuccess: () => {
        if (onEdgeUpdated) onEdgeUpdated();
        onClose();
      },
      onError: (err: any) => {
        setError(err.message || 'Failed to update edge');
      }
    });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Edit Relationship</DialogTitle>
      <DialogContent>
        {loadingEdgeTypes ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
            <CircularProgress />
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, mt: 1 }}>
            {error && <Alert severity="error">{error}</Alert>}

            {/* Basic Info */}
            <TextField
              label="Relationship Type"
              value={edgeTypeDefinition?.edge_type_name || edge?.edge_type_name || edge?.relationship_type || 'Unknown'}
              disabled
              fullWidth
              size="small"
              helperText={edgeTypeDefinition?.description}
            />

            <TextField
              label="Description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              fullWidth
              multiline
              rows={2}
              placeholder="Describe this relationship..."
            />

            {/* Dynamic Properties */}
            {edgeTypeDefinition?.properties && edgeTypeDefinition.properties.length > 0 && (
              <Box>
                <Typography variant="subtitle2" gutterBottom>
                  Edge Properties
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  {edgeTypeDefinition.properties.map((prop) => {
                    const value = propertyValues[prop.name] ?? '';
                    
                    // Render based on input_type
                    if (prop.input_type === 'textarea' || prop.input_type === 'text') {
                      return (
                        <TextField
                          key={prop.name}
                          label={prop.label}
                          value={value}
                          onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.value }))}
                          fullWidth
                          size="small"
                          required={!prop.nullable}
                          multiline={prop.input_type === 'textarea'}
                          rows={prop.input_type === 'textarea' ? 3 : 1}
                          helperText={prop.format}
                        />
                      );
                    }
                    
                    if (prop.input_type === 'checkbox') {
                      return (
                        <FormControlLabel
                          key={prop.name}
                          control={
                            <Checkbox
                              checked={!!value}
                              onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.checked }))}
                            />
                          }
                          label={prop.label}
                        />
                      );
                    }
                    
                    if (prop.input_type === 'select' && prop.options) {
                      return (
                        <FormControl key={prop.name} fullWidth size="small" required={!prop.nullable}>
                          <InputLabel>{prop.label}</InputLabel>
                          <Select
                            value={value}
                            onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.value }))}
                            label={prop.label}
                          >
                            <MenuItem value=""><em>None</em></MenuItem>
                            {prop.options.map(opt => (
                              <MenuItem key={opt} value={opt}>{opt}</MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      );
                    }
                    
                    if (prop.input_type === 'number') {
                      return (
                        <TextField
                          key={prop.name}
                          label={prop.label}
                          type="number"
                          value={value}
                          onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: parseFloat(e.target.value) || 0 }))}
                          fullWidth
                          size="small"
                          required={!prop.nullable}
                          helperText={prop.format}
                        />
                      );
                    }
                    
                    // Default to text
                    return (
                      <TextField
                        key={prop.name}
                        label={prop.label}
                        value={value}
                        onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.value }))}
                        fullWidth
                        size="small"
                        required={!prop.nullable}
                        helperText={prop.format}
                      />
                    );
                  })}
                </Box>
              </Box>
            )}

            {!edgeTypeDefinition && edge && (
              <Alert severity="info">
                No advanced properties configuration found for this relationship type.
                You can still edit the description.
              </Alert>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button 
          variant="contained" 
          onClick={handleUpdate}
          disabled={updateEdgeMutation.isPending}
          startIcon={updateEdgeMutation.isPending && <CircularProgress size={20} />}
        >
          Save Changes
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default EditEdgeDialog;
