import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Chip,
  Tooltip,
  CircularProgress,
  Alert,
  TextField,
  InputAdornment,
  FormControl,
  Select,
  MenuItem,
  Checkbox,
  Button,
  Stack,
} from '@mui/material';
import {
  Edit as EditIcon,
  Search as SearchIcon,
  Visibility as VisibleIcon,
  VisibilityOff as HiddenIcon,
  Functions as CalculationIcon,
  CheckCircle as RequiredIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';
import { TermMetadataEditorDrawer } from './TermMetadataEditorDrawer';
import { SemanticTermPhysicalMappingEditor } from './SemanticTermPhysicalMappingEditor';
import { Link as LinkIcon, Info as InfoIcon } from '@mui/icons-material';

// ============================================================================
// Types
// ============================================================================

interface TermWithMetadata {
  term_id: string;
  term_name: string;
  term_title?: string;
  source_column?: string;
  data_type?: string;
  is_calculation?: boolean;
  metadata_id?: string;
  display_name?: string;
  description?: string;
  group_name?: string;
  required: boolean;
  visible: boolean;
  format?: string;
  precision?: number;
  aggregation?: string;
  sort_order?: number;
  inferred_type?: string;
  is_aggregate?: boolean;
}

interface BOTermsTabProps {
  boId: string;
  terms?: TermWithMetadata[];
  onDelete?: (term: TermWithMetadata) => void;
}

// ============================================================================
// Component
// ============================================================================

export const BOTermsTab: React.FC<BOTermsTabProps> = ({ boId, terms: propsTerms, onDelete }) => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [terms, setTerms] = useState<TermWithMetadata[]>([]);

  const [searchQuery, setSearchQuery] = useState('');
  const [groupFilter, setGroupFilter] = useState<string>('all');
  const [showHidden, setShowHidden] = useState(false);

  const [selectedTerm, setSelectedTerm] = useState<TermWithMetadata | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  
  const [mappingTerm, setMappingTerm] = useState<TermWithMetadata | null>(null);
  const [mappingDrawerOpen, setMappingDrawerOpen] = useState(false);

  // Load terms
  const loadTerms = async () => {
    if (propsTerms && propsTerms.length > 0) {
      setTerms(propsTerms);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/bo/${boId}/terms`, {
        headers: {
          'X-Tenant-ID': tenantId,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to load terms');
      }

      const data = await response.json();
      setTerms(data.terms || []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (propsTerms) {
      setTerms(propsTerms);
      setLoading(false);
    } else if (boId && tenantId) {
      loadTerms();
    }
  }, [boId, tenantId, propsTerms]);

  // Get unique groups for filter
  const groups = useMemo(() => {
    const groupSet = new Set<string>();
    terms.forEach((t) => {
      if (t.group_name) groupSet.add(t.group_name);
    });
    return Array.from(groupSet).sort();
  }, [terms]);

  // Filtered terms
  const filteredTerms = useMemo(() => {
    return terms.filter((term) => {
      // Search filter
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        const matchesSearch =
          term.term_name.toLowerCase().includes(q) ||
          term.display_name?.toLowerCase().includes(q) ||
          term.source_column?.toLowerCase().includes(q) ||
          term.group_name?.toLowerCase().includes(q);
        if (!matchesSearch) return false;
      }

      // Group filter
      if (groupFilter !== 'all') {
        if (groupFilter === 'ungrouped') {
          if (term.group_name) return false;
        } else {
          if (term.group_name !== groupFilter) return false;
        }
      }

      // Visibility filter
      if (!showHidden && !term.visible) return false;

      return true;
    });
  }, [terms, searchQuery, groupFilter, showHidden]);

  // Group terms by group_name for display
  const groupedTerms = useMemo(() => {
    const groups: Record<string, TermWithMetadata[]> = {};
    filteredTerms.forEach((term) => {
      const group = term.group_name || 'Ungrouped';
      if (!groups[group]) groups[group] = [];
      groups[group].push(term);
    });

    // Sort terms within each group by sort_order
    Object.keys(groups).forEach((key) => {
      groups[key].sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0));
    });

    return groups;
  }, [filteredTerms]);

  const handleEditTerm = (term: TermWithMetadata) => {
    setSelectedTerm(term);
    setEditorOpen(true);
  };

  const handleEditorClose = () => {
    setEditorOpen(false);
    setSelectedTerm(null);
  };

  const handleEditorSave = () => {
    loadTerms(); // Refresh list
  };

  const handleOpenMappings = (term: TermWithMetadata) => {
    setMappingTerm(term);
    setMappingDrawerOpen(true);
  };

  const formatDisplay = (format?: string) => {
    switch (format) {
      case 'currency':
        return '💵';
      case 'percent':
        return '%';
      case 'number':
        return '#';
      case 'date':
        return '📅';
      case 'integer':
        return '123';
      case 'boolean':
        return '✓/✗';
      default:
        return 'Aa';
    }
  };

  const aggregationLabel = (agg?: string) => {
    switch (agg) {
      case 'sum':
        return 'Σ';
      case 'avg':
        return 'x̄';
      case 'min':
        return 'min';
      case 'max':
        return 'max';
      case 'count':
        return '#';
      default:
        return null;
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
        <Button size="small" onClick={loadTerms} sx={{ ml: 2 }}>
          Retry
        </Button>
      </Alert>
    );
  }

  return (
    <Box>
      {/* Toolbar */}
      <Box sx={{ display: 'flex', gap: 2, mb: 2, flexWrap: 'wrap', alignItems: 'center' }}>
        <TextField
          size="small"
          placeholder="Search terms..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
          sx={{ minWidth: 200 }}
        />

        <FormControl size="small" sx={{ minWidth: 150 }}>
          <Select
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
            displayEmpty
            startAdornment={<FilterIcon fontSize="small" sx={{ mr: 1 }} />}
          >
            <MenuItem value="all">All Groups</MenuItem>
            <MenuItem value="ungrouped">Ungrouped</MenuItem>
            {groups.map((g) => (
              <MenuItem key={g} value={g}>
                {g}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <Tooltip title="Show hidden fields">
          <Checkbox
            checked={showHidden}
            onChange={(e) => setShowHidden(e.target.checked)}
            icon={<VisibleIcon />}
            checkedIcon={<HiddenIcon />}
          />
        </Tooltip>

        <Box sx={{ flex: 1 }} />

        <Button size="small" startIcon={<RefreshIcon />} onClick={loadTerms}>
          Refresh
        </Button>

        <Typography variant="body2" color="text.secondary">
          {filteredTerms.length} of {terms.length} terms
        </Typography>
      </Box>

      {/* Terms Table */}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={{ fontWeight: 600 }}>Display Name</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Source / Type</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Group</TableCell>
              <TableCell sx={{ fontWeight: 600 }} align="center">
                Format
              </TableCell>
              <TableCell sx={{ fontWeight: 600 }} align="center">
                Agg
              </TableCell>
              <TableCell sx={{ fontWeight: 600 }} align="center">
                Flags
              </TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {Object.entries(groupedTerms).map(([groupName, groupTerms]) => (
              <React.Fragment key={groupName}>
                {/* Group Header Row (subtle) */}
                {groups.length > 0 && (
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell colSpan={7}>
                      <Typography variant="caption" sx={{ fontWeight: 600 }}>
                        {groupName} ({groupTerms.length})
                      </Typography>
                    </TableCell>
                  </TableRow>
                )}

                {groupTerms.map((term) => (
                  <TableRow
                    key={term.term_id}
                    sx={{
                      opacity: term.visible ? 1 : 0.5,
                      '&:hover': { bgcolor: 'action.hover' },
                    }}
                  >
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="body2" fontWeight={500}>
                            {term.display_name || term.term_name}
                        </Typography>
                        {term.is_calculation && (
                          <Tooltip title="Calculated Field">
                            <CalculationIcon fontSize="small" color="info" />
                          </Tooltip>
                        )}
                        <Tooltip
                            title={
                              <Box sx={{ p: 0.5 }}>
                                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>
                                  {term.term_name}
                                </Typography>
                                <Typography variant="body2" sx={{ fontSize: '0.75rem', mb: 1, opacity: 0.9 }}>
                                  {term.description || "No description available."}
                                </Typography>
                                <Stack spacing={0.5}>
                                  {term.source_column && (
                                    <Typography variant="caption" display="block" sx={{ opacity: 0.8 }}>
                                      <strong>Source:</strong> {term.source_column}
                                    </Typography>
                                  )}
                                </Stack>
                              </Box>
                            }
                            arrow
                            placement="top"
                          >
                            <InfoIcon 
                              fontSize="small" 
                              sx={{ fontSize: '1rem', color: 'text.disabled', cursor: 'help', '&:hover': { color: 'primary.main' } }} 
                            />
                        </Tooltip>
                      </Box>
                      {term.description && (
                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                          {term.description.length > 60
                            ? term.description.substring(0, 60) + '...'
                            : term.description}
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                        {term.data_type && (
                            <Chip 
                                label={term.data_type} 
                                size="small" 
                                sx={{ 
                                  height: 22, 
                                  fontSize: '0.7rem',
                                  fontWeight: 500,
                                  border: '1px solid',
                                  ...(() => {
                                    const type = (term.data_type || 'string').toLowerCase();
                                    if (/int|double|float|decimal|number/.test(type)) return { bgcolor: 'primary.50', color: 'primary.700', borderColor: 'primary.200' };
                                    if (/bool|boolean/.test(type)) return { bgcolor: 'secondary.50', color: 'secondary.700', borderColor: 'secondary.200' };
                                    if (/date|time/.test(type)) return { bgcolor: 'warning.50', color: 'warning.800', borderColor: 'warning.200' };
                                    return { bgcolor: 'success.50', color: 'success.700', borderColor: 'success.200' };
                                  })()
                                }}
                            />
                        )}
                        {term.inferred_type && term.is_calculation && (
                             <Tooltip title="Inferred Type">
                                <Chip label={term.inferred_type} size="small" variant="outlined" color="info" />
                             </Tooltip>
                        )}
                      </Box>
                      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace' }}>
                         {term.source_column}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {term.group_name ? (
                        <Chip label={term.group_name} size="small" />
                      ) : (
                        <Typography variant="caption" color="text.secondary">
                          —
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell align="center">
                      <Tooltip title={term.format || 'String'}>
                        <Chip label={formatDisplay(term.format)} size="small" variant="outlined" />
                      </Tooltip>
                    </TableCell>
                    <TableCell align="center">
                      <Box sx={{ display: 'flex', justifyContent: 'center', gap: 0.5 }}>
                      {term.aggregation && term.aggregation !== 'none' ? (
                        <Tooltip title={term.aggregation}>
                          <Chip label={aggregationLabel(term.aggregation)} size="small" color="primary" />
                        </Tooltip>
                      ) : (
                        <Typography variant="caption" color="text.secondary">
                          —
                        </Typography>
                      )}
                      {term.is_aggregate && (
                           <Tooltip title="Expression is aggregated">
                                <Chip label="AGG" size="small" color="secondary" variant="outlined"/>
                           </Tooltip>
                      )}
                      </Box>
                    </TableCell>
                    <TableCell align="center">
                      <Box sx={{ display: 'flex', justifyContent: 'center', gap: 0.5 }}>
                        {term.required && (
                          <Tooltip title="Required">
                            <RequiredIcon fontSize="small" color="error" />
                          </Tooltip>
                        )}
                        {!term.visible && (
                          <Tooltip title="Hidden">
                            <HiddenIcon fontSize="small" color="disabled" />
                          </Tooltip>
                        )}
                      </Box>
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip title="Edit Metadata">
                        <IconButton size="small" onClick={() => handleEditTerm(term)}>
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Physical Mappings">
                        <IconButton size="small" onClick={() => handleOpenMappings(term)}>
                          <LinkIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete Term">
                        <IconButton size="small" color="error" onClick={() => {
                          if (onDelete) {
                            onDelete(term);
                          } else {
                            console.warn('Delete not implemented for this view');
                          }
                        }}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </React.Fragment>
            ))}

            {filteredTerms.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    {terms.length === 0
                      ? 'No terms associated with this Business Object yet.'
                      : 'No terms match your filter criteria.'}
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Metadata Editor Drawer */}
      <TermMetadataEditorDrawer
        open={editorOpen}
        onClose={handleEditorClose}
        boId={boId}
        term={selectedTerm}
        existingGroups={groups}
        onSave={handleEditorSave}
      />

      {/* Physical Mapping Editor Drawer */}
        <Box sx={{
          position: 'fixed',
          right: mappingDrawerOpen ? 0 : '-100%',
          top: 0,
          height: '100%',
          width: 600,
          bgcolor: 'background.paper',
          boxShadow: 24,
          transition: 'right 0.3s',
          zIndex: 1300,
          display: 'flex',
          flexDirection: 'column',
        }}>
          {mappingDrawerOpen && mappingTerm && (
            <Box sx={{ p: 3, height: '100%', overflowY: 'auto' }}>
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
                <Button onClick={() => setMappingDrawerOpen(false)}>Close</Button>
              </Box>
              <SemanticTermPhysicalMappingEditor 
                termId={mappingTerm.term_id} 
                termName={mappingTerm.term_name} 
              />
            </Box>
          )}
        </Box>
    </Box>
  );
};

export default BOTermsTab;
