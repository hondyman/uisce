/**
 * FieldSelectionWizard.tsx
 * A wizard-style field selection component with visual semantic term information
 * - Only shows available fields (not already mapped)
 * - Requires semantic terms for all fields
 * - Multi-step wizard experience with field previews
 */

import React, { useState, useMemo, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Stack,
  Box,
  Chip,
  Card,
  CardContent,
  Alert,
  CircularProgress,
  Typography,
  Checkbox,
  Grid,
  Paper,
  Tooltip,
  InputAdornment,
  Tab,
  Tabs,
} from '@mui/material';
import {
  Add as AddIcon,
  Check as CheckIcon,
  Search as SearchIcon,
  Info as InfoIcon,
} from '@mui/icons-material';
import { useNotification } from '../../hooks/useNotification';
import { useEnhancedSemanticTerms, EnhancedSemanticTerm } from '../../hooks/useEnhancedSemanticTerms';
import { useTenant } from '../../contexts/TenantContext';
import { DataTypeChip } from './DataTypeChip';
import { TermInfoTooltip } from './TermInfoTooltip';

interface FieldSelectionWizardProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectFields: (fields: EnhancedSemanticTerm[]) => Promise<void> | void; // Allow async
  selectedDriverTable?: {
    node_id: string;
    node_name: string;
    qualified_path: string;
  } | null;
  existingFields?: any[];
  loading?: boolean;
}

export const FieldSelectionWizard: React.FC<FieldSelectionWizardProps> = ({
  isOpen,
  onClose,
  onSelectFields,
  selectedDriverTable,
  existingFields = [],
  loading: externalLoading = false,
}) => {
  const notification = useNotification();
  const { datasource } = useTenant();
  const { semanticTerms, loading: semanticLoading } = useEnhancedSemanticTerms(datasource?.id || '');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedFields, setSelectedFields] = useState<EnhancedSemanticTerm[]>([]);
  const [filterType, setFilterType] = useState<'all' | 'numeric' | 'string' | 'boolean'>('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [internalLoading, setInternalLoading] = useState(false);

  const [edgeFilteredTerms, setEdgeFilteredTerms] = useState<EnhancedSemanticTerm[]>([]);
  const [fetchingTerms, setFetchingTerms] = useState(false);
  const loading = semanticLoading || externalLoading || internalLoading || fetchingTerms;

  // Get already-mapped semantic term IDs
  const mappedSemanticTermIds = useMemo(() => {
    return new Set(
      existingFields
        .map((f: any) => {
           const id = f.semanticTermId || f.semantic_term_id || f.id;
           return id ? String(id) : null;
        })
        .filter(Boolean)
    );
  }, [existingFields]);

  // Fetch semantic terms using edge-based filtering from backend API
  useEffect(() => {
    const fetchTerms = async () => {
      if (!selectedDriverTable || !datasource?.id) {
        setEdgeFilteredTerms([]);
        return;
      }

      setFetchingTerms(true);
      try {
        const { catalogApi } = await import('../../api/catalogApi');
        const terms = await catalogApi.getSemanticTermsByTable(
          selectedDriverTable.node_id,
          datasource.id
        );
        
        // Convert to EnhancedSemanticTerm format
        const enhancedTerms: EnhancedSemanticTerm[] = terms.map((term: any) => ({
          id: term.id,
          node_name: term.node_name,
          qualified_path: term.qualified_path || '',
          description: term.description || '',
          properties: term.properties || {},
          businessName: term.node_name,
          technicalName: term.properties?.technical_name || term.node_name,
          dataType: term.properties?.data_type,
        }));
        
        setEdgeFilteredTerms(enhancedTerms);
      } catch (error) {
        console.error('Failed to fetch semantic terms by table:', error);
        notification.error('Failed to load semantic terms');
        setEdgeFilteredTerms([]);
      } finally {
        setFetchingTerms(false);
      }
    };

    fetchTerms();
  }, [selectedDriverTable, datasource, notification]);

  // Filter to exclude already-mapped fields
  const availableFields = useMemo(() => {
    return edgeFilteredTerms.filter((term) => !mappedSemanticTermIds.has(String(term.id)));
  }, [edgeFilteredTerms, mappedSemanticTermIds]);

  // Apply search and type filters
  const filteredFields = useMemo(() => {
    let result = [...availableFields];

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter((term) =>
        term.node_name.toLowerCase().includes(query) ||
        term.qualified_path.toLowerCase().includes(query)
      );
    }

    // Apply type filter
    if (filterType !== 'all') {
      result = result.filter((term) => {
        const dataType = (term.dataType || '').toLowerCase();
        switch (filterType) {
          case 'numeric':
            return /int|double|float|decimal|number/i.test(dataType);
          case 'string':
            return /string|varchar|text|char/i.test(dataType);
          case 'boolean':
            return /bool|boolean|bit/i.test(dataType);
          default:
            return true;
        }
      });
    }

    return result;
  }, [availableFields, searchQuery, filterType]);

  // Group fields by category (based on path)
  const groupedFields = useMemo(() => {
    const groups: Record<string, EnhancedSemanticTerm[]> = {};

    filteredFields.forEach((field) => {
      const parts = field.qualified_path.split('.');
      const category = parts.length > 1 ? parts[parts.length - 2] : 'Other';

      if (!groups[category]) {
        groups[category] = [];
      }
      groups[category].push(field);
    });

    return groups;
  }, [filteredFields]);

  const handleToggleField = (field: EnhancedSemanticTerm) => {
    const isSelected = selectedFields.some((f) => f.id === field.id);
    if (isSelected) {
      setSelectedFields(selectedFields.filter((f) => f.id !== field.id));
    } else {
      setSelectedFields([...selectedFields, field]);
    }
  };

  const handleConfirm = async () => {
    if (selectedFields.length === 0) {
      notification.warning('Please select at least one field');
      return;
    }
    
    setInternalLoading(true);
    try {
      await onSelectFields(selectedFields);
      // Only clear and close on success
      setSelectedFields([]);
      setSearchQuery('');
      setFilterType('all');
      onClose();
    } catch (error) {
      // Error handling is likely done in parent, but we catch here to stop loading state
      console.error('Error confirming fields:', error);
    } finally {
      setInternalLoading(false);
    }
  };

  const handleClose = () => {
    setSelectedFields([]);
    setSearchQuery('');
    setFilterType('all');
    onClose();
  };

  if (!selectedDriverTable) {
    return (
      <Dialog open={isOpen} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 600 }}>Select Fields</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Alert severity="info">
            Please select a driver table first to see available fields for mapping.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Close</Button>
        </DialogActions>
      </Dialog>
    );
  }

  return (
    <Dialog
      open={isOpen}
      onClose={handleClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          boxShadow: 3,
          maxHeight: '90vh',
        },
      }}
    >
      <DialogTitle sx={{ fontWeight: 600, fontSize: '1.25rem', pb: 1 }}>
        🎯 Select Fields to Map
      </DialogTitle>

      <DialogContent sx={{ pt: 2 }}>
        <Stack spacing={3}>
          {/* Table Info */}
          <Card variant="outlined" sx={{ bgcolor: 'info.light', border: '2px solid', borderColor: 'info.main' }}>
            <CardContent sx={{ py: 1.5 }}>
              <Stack direction="row" spacing={2} alignItems="center">
                <InfoIcon sx={{ color: 'info.main' }} />
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    {selectedDriverTable.node_name}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {selectedDriverTable.qualified_path}
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>

          {/* Search and Filter Bar */}
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="flex-start">
            <TextField
              placeholder="Search fields..."
              size="small"
              fullWidth
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              disabled={loading}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" />
                  </InputAdornment>
                ),
              }}
            />

            {/* Type Filter Chips */}
            <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap' }}>
              {(['all', 'numeric', 'string', 'boolean'] as const).map((type) => (
                <Chip
                  key={type}
                  label={type === 'all' ? 'All Types' : type.charAt(0).toUpperCase() + type.slice(1)}
                  onClick={() => setFilterType(type)}
                  variant={filterType === type ? 'filled' : 'outlined'}
                  size="small"
                  color={filterType === type ? 'primary' : 'default'}
                  icon={type === 'all' ? undefined : filterType === type ? <CheckIcon /> : undefined}
                />
              ))}
            </Stack>
          </Stack>

          {/* View Mode Toggle */}
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="caption" color="text.secondary">
              {selectedFields.length > 0 ? (
                <span>
                  <strong>{selectedFields.length}</strong> field{selectedFields.length !== 1 ? 's' : ''} selected
                </span>
              ) : (
                <span>
                  {filteredFields.length} available field{filteredFields.length !== 1 ? 's' : ''}
                </span>
              )}
            </Typography>
            <Tabs
              value={viewMode}
              onChange={(_, value) => setViewMode(value)}
              sx={{ minHeight: 'auto' }}
            >
              <Tab label="Grid" value="grid" sx={{ minHeight: 'auto', py: 0.5 }} />
              <Tab label="List" value="list" sx={{ minHeight: 'auto', py: 0.5 }} />
            </Tabs>
          </Box>

          {/* Fields Content */}
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress />
            </Box>
          ) : filteredFields.length === 0 ? (
            <Alert severity="info">
              {fetchingTerms || semanticLoading ? (
                'Loading fields...'
              ) : edgeFilteredTerms.length === 0 ? (
                'No semantic terms were found for the selected driver table. Verify the driver table or check semantic mappings.'
              ) : availableFields.length === 0 ? (
                'All available fields are already mapped.'
              ) : (
                'No fields match your search criteria.'
              )}
            </Alert>
          ) : viewMode === 'grid' ? (
            <Grid container spacing={2}>
              {filteredFields.map((field) => (
                <Grid item xs={12} sm={6} key={field.id}>
                  <Paper
                    onClick={() => handleToggleField(field)}
                    sx={{
                      p: 2,
                      cursor: 'pointer',
                      border: '2px solid',
                      borderColor: selectedFields.some((f) => f.id === field.id)
                        ? 'primary.main'
                        : 'divider',
                      backgroundColor: selectedFields.some((f) => f.id === field.id)
                        ? 'action.selected'
                        : 'background.paper',
                      transition: 'all 0.2s',
                      '&:hover': {
                        borderColor: 'primary.main',
                        boxShadow: 2,
                      },
                    }}
                  >
                    <Stack spacing={1}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                        <Box sx={{ flex: 1 }}>
                          <Typography variant="body2" sx={{ fontWeight: 600 }}>
                            {field.node_name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                            {field.qualified_path}
                          </Typography>
                        </Box>
                        <Checkbox
                          checked={selectedFields.some((f) => f.id === field.id)}
                          onChange={() => handleToggleField(field)}
                          onClick={(e) => e.stopPropagation()}
                          size="small"
                        />
                      </Box>

                      <Stack direction="row" spacing={1} sx={{ mt: 1, flexWrap: 'wrap', gap: 1 }}>
                        <Tooltip title={`Data Type: ${field.dataType || 'Unknown'}`}>
                           <Box>
                             <DataTypeChip type={field.dataType} />
                           </Box>
                        </Tooltip>
                      </Stack>
                    </Stack>
                  </Paper>
                </Grid>
              ))}
            </Grid>
          ) : (
            <Stack spacing={1}>
              {Object.entries(groupedFields).map(([category, fields]) => (
                <Box key={category}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600, color: 'primary.main', mb: 1 }}>
                    {category}
                  </Typography>
                  <Stack spacing={1} sx={{ ml: 2 }}>
                    {fields.map((field) => (
                      <Paper
                        key={field.id}
                        onClick={() => handleToggleField(field)}
                        sx={{
                          p: 1.5,
                          cursor: 'pointer',
                          border: '1px solid',
                          borderColor: selectedFields.some((f) => f.id === field.id)
                            ? 'primary.main'
                            : 'divider',
                          backgroundColor: selectedFields.some((f) => f.id === field.id)
                            ? 'action.selected'
                            : 'background.paper',
                          transition: 'all 0.2s',
                          '&:hover': {
                            borderColor: 'primary.main',
                            boxShadow: 1,
                          },
                        }}
                      >
                        <Stack direction="row" spacing={2} alignItems="center" justifyContent="space-between">
                            <Box sx={{ flex: 1 }}>
                              <Stack direction="row" alignItems="center" spacing={1} mb={0.5}>
                                <Typography variant="body2" sx={{ fontWeight: 600 }}>
                                  {field.node_name}
                                </Typography>
                                <TermInfoTooltip term={{
                                    node_name: field.node_name,
                                    description: field.description,
                                    qualified_path: field.qualified_path,
                                    properties: field.properties
                                }} />
                              </Stack>

                              <Stack direction="row" spacing={1} alignItems="center">
                                <DataTypeChip type={field.dataType} sx={{ height: 20, fontSize: '0.7rem' }} />
                                <Typography variant="caption" color="text.secondary" noWrap sx={{ maxWidth: 200 }}>
                                  {field.qualified_path}
                                </Typography>
                              </Stack>
                            </Box>
                          <Checkbox
                            checked={selectedFields.some((f) => f.id === field.id)}
                            onChange={() => handleToggleField(field)}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </Stack>
                      </Paper>
                    ))}
                  </Stack>
                </Box>
              ))}
            </Stack>
          )}

          {/* Selected Fields Summary */}
          {selectedFields.length > 0 && (
            <Card variant="outlined" sx={{ bgcolor: 'success.light', border: '2px solid', borderColor: 'success.main' }}>
              <CardContent sx={{ py: 1.5 }}>
                <Stack spacing={1}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <CheckIcon sx={{ color: 'success.main' }} />
                    <Typography variant="body2" sx={{ fontWeight: 600 }}>
                      {selectedFields.length} field{selectedFields.length !== 1 ? 's' : ''} selected
                    </Typography>
                  </Box>
                  <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', gap: 1 }}>
                    {selectedFields.map((field) => (
                      <Chip
                        key={field.id}
                        label={field.node_name}
                        size="small"
                        color="success"
                        variant="filled"
                        onDelete={() => handleToggleField(field)}
                      />
                    ))}
                  </Stack>
                </Stack>
              </CardContent>
            </Card>
          )}
        </Stack>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, gap: 1 }}>
        <Button onClick={handleClose}>Cancel</Button>
        <Button
          onClick={handleConfirm}
          variant="contained"
          startIcon={<AddIcon />}
          disabled={selectedFields.length === 0 || loading}
          sx={{ minWidth: 120 }}
        >
          Add {selectedFields.length > 0 ? `(${selectedFields.length})` : ''}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default FieldSelectionWizard;
