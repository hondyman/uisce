import React, { useState, useMemo, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Typography,
  Box,
  CircularProgress,
  Alert,
  TextField,
  Autocomplete,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Chip,
  FormControlLabel,
  Checkbox
} from '@mui/material';
import {
  AddLink as AddLinkIcon,
  Description as TermIcon,
  TableChart as TableIcon,
  ViewColumn as ColumnIcon,
  Label as TagIcon
} from '@mui/icons-material';
import { useEdgeTypes } from '../api/edgeTypes';
import { useNodeTypes } from '../api/nodeTypes';
import { useCreateTermEdge } from '../api/glossary';
import { useTenant } from '../contexts/TenantContext';
import { LineageService } from '../services/lineageService';

interface AddEdgeDialogProps {
  open: boolean;
  onClose: () => void;
  sourceNodeId: string;
  sourceNodeType: string; // 'business_term', 'semantic_term', 'column', 'table' etc.
  onEdgeAdded?: () => void;
}

const AddEdgeDialog: React.FC<AddEdgeDialogProps> = ({
  open,
  onClose,
  sourceNodeId,
  sourceNodeType,
  onEdgeAdded
}) => {
  const { tenant, datasource } = useTenant();
  const [selectedEdgeTypeId, setSelectedEdgeTypeId] = useState<string>('');
  const [targetSearchQuery, setTargetSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [selectedTargetNode, setSelectedTargetNode] = useState<any | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [propertyValues, setPropertyValues] = useState<Record<string, any>>({});

  // APIs
  const { data: edgeTypes, isLoading: loadingEdgeTypes } = useEdgeTypes(tenant?.id || '');
  const { data: nodeTypes } = useNodeTypes(tenant?.id || '');
  const createEdgeMutation = useCreateTermEdge();
  const lineageService = useMemo(() => new LineageService(), []);

  // Helper to map string type to ID
  const getNodeTypeId = (typeName: string) => {
    return nodeTypes?.find(nt => nt.catalog_type_name === typeName)?.id;
  };

  const getNodeTypeName = (typeId: string) => {
    return nodeTypes?.find(nt => nt.id === typeId)?.catalog_type_name;
  };

  // Filter valid edge types for this source
  const validEdgeTypes = useMemo(() => {
    if (!edgeTypes || !nodeTypes) return [];
    
    const sourceTypeId = getNodeTypeId(sourceNodeType);
    if (!sourceTypeId) return [];

    return (edgeTypes || []).filter(et => {
      if (!et.is_active) return false;
      return et.subject_node_type_id === sourceTypeId || et.object_node_type_id === sourceTypeId;
    });
  }, [edgeTypes, nodeTypes, sourceNodeType]);

  const selectedEdgeType = useMemo(() => 
    edgeTypes?.find(et => et.id === selectedEdgeTypeId),
    [edgeTypes, selectedEdgeTypeId]
  );

  useEffect(() => {
    if (selectedEdgeType?.properties) {
      const initialValues: Record<string, any> = {};
      selectedEdgeType.properties.forEach(prop => {
        initialValues[prop.name] = prop.default_value ?? '';
      });
      setPropertyValues(initialValues);
    } else {
      setPropertyValues({});
    }
  }, [selectedEdgeType]);

  const targetNodeTypeName = useMemo(() => {
    if (!selectedEdgeType) return null;
    const sourceTypeId = getNodeTypeId(sourceNodeType);
    if (!sourceTypeId) return null;
    
    if (sourceTypeId === selectedEdgeType.subject_node_type_id) {
      return getNodeTypeName(selectedEdgeType.object_node_type_id);
    } else if (sourceTypeId === selectedEdgeType.object_node_type_id) {
      return getNodeTypeName(selectedEdgeType.subject_node_type_id);
    }
    return null;
  }, [selectedEdgeType, sourceNodeType, nodeTypes]);

  const isReversedEdge = useMemo(() => {
    if (!selectedEdgeType) return false;
    const sourceTypeId = getNodeTypeId(sourceNodeType);
    return sourceTypeId === selectedEdgeType.object_node_type_id;
  }, [selectedEdgeType, sourceNodeType, nodeTypes]);

  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);

    if (!targetSearchQuery.trim() || !targetNodeTypeName) {
      setSearchResults([]);
      return;
    }

    debounceTimerRef.current = setTimeout(async () => {
      setIsSearching(true);
      setError(null);

      try {
        const isBusinessTerm = targetNodeTypeName.toLowerCase() === 'business_term';
        const isSemanticTerm = targetNodeTypeName.toLowerCase() === 'semantic_term';
        
        if (isBusinessTerm) {
          const results = await lineageService.searchBusinessTerms({
            query: targetSearchQuery,
            limit: 20
          });
          setSearchResults(results.business_terms || []);
        } else if (isSemanticTerm) {
          const response = await fetch('/api/semantic-terms/search', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': tenant?.id || '',
              'X-Tenant-Datasource-ID': datasource?.id || '',
            },
            body: JSON.stringify({ query: targetSearchQuery, limit: 20 })
          });
          if (response.ok) {
            const data = await response.json();
            setSearchResults(data.semantic_terms || []);
          }
        } else {
          const params = new URLSearchParams({ q: targetSearchQuery, type: targetNodeTypeName, limit: '20' });
          const url = `/api/catalog/nodes?${params.toString()}`;
          const results = await fetch(url, {
             headers: {
               'X-Tenant-ID': tenant?.id || '',
               'X-Tenant-Datasource-ID': datasource?.id || '',
               'Content-Type': 'application/json'
             }
          }).then(res => res.ok ? res.json() : []);
          setSearchResults(results || []);
        }
      } catch (e) {
        console.error('Search failed', e);
        setError('Search failed. Please try again.');
        setSearchResults([]);
      } finally {
        setIsSearching(false);
      }
    }, 300);

    return () => {
      if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    };
  }, [targetSearchQuery, targetNodeTypeName, tenant?.id, lineageService, datasource?.id]);

  const handleCreate = () => {
    if (!selectedEdgeTypeId || !selectedTargetNode) return;
    const isReversed = isReversedEdge;
    
    createEdgeMutation.mutate({
      subject_node_id: isReversed ? selectedTargetNode.id : sourceNodeId,
      object_node_id: isReversed ? sourceNodeId : selectedTargetNode.id,
      edge_type_id: selectedEdgeTypeId,
      properties: propertyValues
    }, {
      onSuccess: () => {
        if (onEdgeAdded) onEdgeAdded();
        handleClose();
      },
      onError: (err: any) => {
        setError(err.message || 'Failed to create edge');
      }
    });
  };

  const handleClose = () => {
    setSelectedEdgeTypeId('');
    setTargetSearchQuery('');
    setSearchResults([]);
    setSelectedTargetNode(null);
    setError(null);
    onClose();
  };

  const getIconForType = (type: string) => {
    const lowerType = type.toLowerCase();
    if (lowerType.includes('table')) return <TableIcon fontSize="small" />;
    if (lowerType.includes('column')) return <ColumnIcon fontSize="small" />;
    if (lowerType.includes('term')) return <TermIcon fontSize="small" />;
    return <TermIcon fontSize="small" />;
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Add Relationship</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          
          <FormControl fullWidth size="small">
            <InputLabel>Relationship Type</InputLabel>
            <Select
              value={selectedEdgeTypeId}
              label="Relationship Type"
              onChange={(e) => {
                setSelectedEdgeTypeId(e.target.value);
                setSearchResults([]);
                setSelectedTargetNode(null);
              }}
            >
              {validEdgeTypes.map(et => {
                const sourceTypeId = getNodeTypeId(sourceNodeType);
                const isReversed = sourceTypeId === et.object_node_type_id;
                const targetTypeId = isReversed ? et.subject_node_type_id : et.object_node_type_id;
                const targetTypeName = getNodeTypeName(targetTypeId);
                const arrow = isReversed ? '←' : '→';
                return (
                  <MenuItem key={et.id} value={et.id}>
                    {et.edge_type_name} {arrow} {targetTypeName}
                  </MenuItem>
                );
              })}
            </Select>
          </FormControl>

          {selectedEdgeTypeId && targetNodeTypeName && (
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Select Target {targetNodeTypeName.replace('_', ' ')}
              </Typography>
              
              <Autocomplete
                fullWidth
                size="small"
                open={open && targetSearchQuery.length > 0 && !selectedTargetNode}
                options={searchResults}
                loading={isSearching}
                getOptionLabel={(option) => option.node_name || option.name || ''}
                filterOptions={(x) => x}
                onInputChange={(_, newInputValue) => setTargetSearchQuery(newInputValue)}
                onChange={(_, newValue) => setSelectedTargetNode(newValue)}
                value={selectedTargetNode}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    placeholder={`Search ${targetNodeTypeName}s...`}
                    InputProps={{
                      ...params.InputProps,
                      endAdornment: (
                        <>
                          {isSearching ? <CircularProgress color="inherit" size={20} /> : null}
                          {params.InputProps.endAdornment}
                        </>
                      ),
                    }}
                  />
                )}
                renderOption={(props, option) => {
                  const { key, ...optionProps } = props;
                  return (
                    <li key={key} {...optionProps}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          {getIconForType(targetNodeTypeName || '')}
                          <Typography variant="body2" fontWeight={500}>
                            {option.node_name || option.name}
                          </Typography>
                        </Box>
                        {option.qualified_path && (
                          <Typography variant="caption" color="primary">
                            {option.qualified_path}
                          </Typography>
                        )}
                        {option.description && (
                          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {option.description}
                          </Typography>
                        )}
                      </Box>
                    </li>
                  );
                }}
                noOptionsText={targetSearchQuery && !isSearching ? "No results found" : "Type to search..."}
              />

              {selectedTargetNode && (
                <Box sx={{ mt: 2 }}>
                  <Alert severity="info" action={<Button color="inherit" size="small" onClick={() => setSelectedTargetNode(null)}>Change</Button>}>
                    <Box>
                      <Typography variant="body2">Selected: <strong>{selectedTargetNode.node_name || selectedTargetNode.name}</strong></Typography>
                      {selectedTargetNode.qualified_path && <Typography variant="caption" color="primary">{selectedTargetNode.qualified_path}</Typography>}
                    </Box>
                  </Alert>
                </Box>
              )}
            </Box>
          )}

          {selectedEdgeType?.properties && selectedEdgeType.properties.length > 0 && (
            <Box sx={{ mt: 3 }}>
              <Typography variant="subtitle2" gutterBottom>Edge Properties</Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {selectedEdgeType.properties.map((prop) => {
                  const value = propertyValues[prop.name] ?? '';
                  if (prop.input_type === 'textarea' || prop.input_type === 'text') {
                    return <TextField key={prop.name} label={prop.label} value={value} onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.value }))} fullWidth size="small" required={!prop.nullable} multiline={prop.input_type === 'textarea'} rows={prop.input_type === 'textarea' ? 3 : 1} helperText={prop.format} />;
                  }
                  if (prop.input_type === 'checkbox') {
                    return <FormControlLabel key={prop.name} control={<Checkbox checked={!!value} onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.checked }))} />} label={prop.label} />;
                  }
                  if (prop.input_type === 'select' && prop.options) {
                    return <FormControl key={prop.name} fullWidth size="small" required={!prop.nullable}><InputLabel>{prop.label}</InputLabel><Select value={value} onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.value }))} label={prop.label}><MenuItem value=""><em>None</em></MenuItem>{prop.options.map(opt => (<MenuItem key={opt} value={opt}>{opt}</MenuItem>))}</Select></FormControl>;
                  }
                  return <TextField key={prop.name} label={prop.label} value={value} onChange={(e) => setPropertyValues(prev => ({ ...prev, [prop.name]: e.target.value }))} fullWidth size="small" required={!prop.nullable} helperText={prop.format} />;
                })}
              </Box>
            </Box>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>Cancel</Button>
        <Button variant="contained" onClick={handleCreate} disabled={!selectedEdgeTypeId || !selectedTargetNode || createEdgeMutation.isPending} startIcon={<AddLinkIcon />}>
          Add Relationship
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddEdgeDialog;
