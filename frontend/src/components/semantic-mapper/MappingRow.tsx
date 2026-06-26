import { useState, useEffect } from 'react';
import { devDebug } from '../../utils/devLogger';
import { Database, Tag, ArrowRight, Edit3, Link2, AlertCircle } from 'lucide-react';
import {
  Grid, Card, CardContent, Button, Box, Typography, Chip, LinearProgress,
  Stack, Autocomplete, TextField, Tooltip, IconButton
} from '@mui/material';

import type { Mapping, SemanticTerm } from './types';
import { getMappingUniqueId } from '../../utils/mappingId';

interface MappingRowProps {
  mapping: Mapping;
  idx: number;
  savedRows: Set<string>;
  compactRows: boolean;
  keyboardExpanded: boolean;
  setKeyboardExpanded: (expanded: boolean) => void;
  toggleMapping: (id: string) => void;
  confirmEditing: (id: string, term?: string) => void;
  searchSemanticTerms: (query: string) => Promise<SemanticTerm[]>;
  selectSemanticTerm: (term: SemanticTerm, mappingId: string) => void;
  handleCreateAndSelectTerm: (mappingId: string, termName: string) => Promise<SemanticTerm | null>;
  setOverride: (id: string, value: boolean) => void;
  setIgnored: (id: string, value: boolean) => void;
  openReplaceConfirm: (index: number) => void;
  openLineageModal: (mapping: any) => void;
}

const getConfidenceLabel = (conf: number) => {
  if (conf >= 0.9) return 'Excellent';
  if (conf >= 0.75) return 'Good';
  if (conf >= 0.6) return 'Fair';
  if (conf >= 0.4) return 'Low';
  return 'Poor';
};

const getConfidenceColorMUI = (conf: number) => {
  if (conf >= 0.75) return 'success';
  if (conf >= 0.6) return 'info';
  if (conf >= 0.4) return 'warning';
  return 'error';
};

const getConfidenceIcon = (conf: number) => {
  if (conf >= 0.75) return '🎯';
  if (conf >= 0.6) return '✓';
  if (conf >= 0.4) return '⚠️';
  return '❌';
};

export function MappingRow({ mapping, idx, ...props }: MappingRowProps) {
  const [localSearchTerm, setLocalSearchTerm] = useState(mapping.semantic_term || '');
  const [localResults, setLocalResults] = useState<SemanticTerm[]>([]);
  const [localOverride, setLocalOverride] = useState(mapping.override || false);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [termExistsInSearch, setTermExistsInSearch] = useState(false);

  // Mark whether this mapping was generated client-side (no backend id)
  const isGenerated = Boolean(!mapping.id || String(mapping.id).startsWith('generated-') || !mapping.database_column.node_id);

  // Helper function to get unique identifier for this mapping
  const getUniqueId = () => getMappingUniqueId(mapping);

  // Sync local state when mapping changes to prevent stale state
  useEffect(() => {
    setLocalSearchTerm(mapping.semantic_term || '');
    setLocalOverride(mapping.override || false);
    setHasUnsavedChanges(false);
    setTermExistsInSearch(false);
  }, [mapping.id, mapping.semantic_term, mapping.override]);

  // When user types, mark as unsaved and check if term exists
  useEffect(() => {
    const trimmed = localSearchTerm.trim().toUpperCase();
    const currentTerm = (mapping.semantic_term || '').trim().toUpperCase();
    
    if (localOverride && trimmed && trimmed !== currentTerm) {
      setHasUnsavedChanges(true);
      
      // Check if the typed term exists in search results
      const exists = localResults.some((r: SemanticTerm) => 
        r.term_name.toUpperCase() === trimmed
      );
      setTermExistsInSearch(exists);
    } else {
      setHasUnsavedChanges(false);
      setTermExistsInSearch(false);
    }
  }, [localSearchTerm, mapping.semantic_term, localOverride, localResults]);

  const handleBlur = async () => {
    if (!hasUnsavedChanges) return;
    
    const trimmed = localSearchTerm.trim();
    if (!trimmed) return;
    
    // If term exists in search, apply it
    if (termExistsInSearch) {
      const term = localResults.find((r: SemanticTerm) => 
        r.term_name.toUpperCase() === trimmed.toUpperCase()
      );
      if (term) {
        props.selectSemanticTerm(term, getUniqueId());
        setHasUnsavedChanges(false);
        return;
      }
    }
    
    // Otherwise just update the local state (user can click "Create New" button)
    setHasUnsavedChanges(true);
  };

  return (
    <Card
      className={`mapping-card ${mapping.edge_exists ? 'mapped' : ''} ${localOverride ? 'override' : ''}`}
      sx={{
        borderRadius: 2, 
        border: mapping.selected ? '2px solid' : '1px solid',
        borderColor: mapping.selected ? 'primary.main' : 'divider', 
        transition: 'all 0.2s ease',
        '&:hover': { transform: 'translateY(-1px)', boxShadow: 3 },
        mb: 1
      }}
      elevation={mapping.selected ? 2 : 1}
      onClick={(e) => {
        const el = (e.target as HTMLElement);
        if (el.closest('button, input, a, .MuiAutocomplete-root, .MuiAutocomplete-input, .MuiIconButton-root')) return;
        props.toggleMapping(getUniqueId());
      }}
    >
      <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Box sx={{ pt: 0.5 }}>
            <input id={`select-${getUniqueId()}`} type="checkbox" aria-label={`Select mapping ${idx + 1}`} checked={mapping.selected || false} onChange={(e) => { e.stopPropagation(); props.toggleMapping(getUniqueId()); }} className={`select-checkbox ${mapping.edge_exists ? 'disabled' : ''}`} disabled={mapping.edge_exists} />
            <label htmlFor={`select-${getUniqueId()}`} className="visually-hidden">Select mapping</label>
          </Box>
          <Box sx={{ flex: 1 }}>
            <Grid container spacing={1.5} alignItems="center">
              <Grid item xs={12} md={5}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                  <Database width={12} height={12} style={{ color: '#666' }} />
                  <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ fontSize: '11px' }}>Database Column</Typography>
                </Box>
                <Box sx={{ 
                  bgcolor: 'grey.50', 
                  border: '1px solid', 
                  borderColor: 'grey.200', 
                  p: 0.75, 
                  borderRadius: 1, 
                  fontFamily: 'monospace', 
                  fontSize: '12px' 
                }}>
                  {mapping.database_column.schema && (
                    <Box component="span" sx={{ color: 'grey.600' }}>{mapping.database_column.schema}.</Box>
                  )}
                  <Box component="span" sx={{ fontWeight: 600 }}>{mapping.database_column.table}</Box>
                  <Box component="span" sx={{ color: 'grey.600' }}>.</Box>
                  <Box component="span" sx={{ color: 'primary.main', fontWeight: 600 }}>{mapping.database_column.column}</Box>
                </Box>
                <Box sx={{ mt: 0.5, display: 'flex', gap: 1, alignItems: 'center' }}>
                  <Tooltip title={isGenerated ? 'This mapping was generated client-side. It will be persisted when you create an edge or save changes.' : 'This mapping has been persisted in the knowledge graph.'}>
                    <span>
                      <Chip label={isGenerated ? 'Generated ID' : 'Persisted'} size="small" color={isGenerated ? 'warning' : 'success'} sx={{ height: 20, fontSize: '10px', fontWeight: 600 }} />
                    </span>
                  </Tooltip>
                  {mapping.database_column.node_id && (
                    <Typography variant="caption" color="text.secondary" sx={{ fontSize: '10px' }}>Column node id: {mapping.database_column.node_id}</Typography>
                  )}
                </Box>
                {mapping.database_column.data_type && (
                  <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block', fontSize: '10px' }}>
                    <Chip label={mapping.database_column.data_type} size="small" sx={{ height: 16, fontSize: '9px' }} />
                  </Typography>
                )}
              </Grid>
              <Grid item xs={12} md={1} sx={{ textAlign: 'center' }}><ArrowRight width={24} height={24} style={{ color: '#999' }} /></Grid>
              <Grid item xs={12} md={6}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                  <Tag width={12} height={12} style={{ color: '#666' }} />
                  <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ fontSize: '11px' }}>Semantic Term</Typography>
                  {mapping.is_new_term && (<Chip label="NEW" size="small" color="success" sx={{ height: 16, fontSize: '9px', fontWeight: 600 }} />)}
                </Box>
                {localOverride ? (
                  <Box sx={{ position: 'relative' }}>
                    <Autocomplete 
                      freeSolo
                      disableClearable
                      blurOnSelect={false}
                      clearOnBlur={false}
                      options={localResults || []} 
                      getOptionLabel={(option: any) => typeof option === 'string' ? option : (option.term_name || '')} 
                      filterOptions={(x) => x} 
                      inputValue={localSearchTerm}
                      onInputChange={async (_e, value, reason) => {
                        // Prevent blur from clearing the input
                        if (reason === 'reset') return;
                        setLocalSearchTerm(value || '');
                        if ((value || '').length >= 2) { 
                          const res = await props.searchSemanticTerms(value || ''); 
                          setLocalResults(res || []); 
                        } else {
                          setLocalResults([]);
                        }
                      }}
                      onChange={async (_e, value) => {
                        if (!value) return;
                        if (typeof value === 'string') {
                          setLocalSearchTerm(value);
                        } else { 
                          // User selected an existing term from dropdown
                          props.selectSemanticTerm(value as SemanticTerm, getUniqueId()); 
                          setLocalSearchTerm((value as SemanticTerm).term_name || '');
                          setHasUnsavedChanges(false);
                        }
                      }}
                      onBlur={handleBlur}
                      renderInput={(params) => (
                        <TextField
                          {...params}
                          placeholder="Search or type a semantic term..."
                          className="search-input"
                          onKeyDown={async (e) => {
                            if (e.key === 'Enter' && hasUnsavedChanges) {
                              const trimmed = localSearchTerm.trim();
                              if (termExistsInSearch) {
                                // Find and select the existing term
                                const term = localResults.find((r: SemanticTerm) => 
                                  r.term_name.toUpperCase() === trimmed.toUpperCase()
                                );
                                if (term) {
                                  props.selectSemanticTerm(term, getUniqueId());
                                  setHasUnsavedChanges(false);
                                }
                              }
                            }
                          }}
                          autoFocus
                          inputProps={{
                            ...params.inputProps,
                            autoComplete: 'off'
                          }}
                        />
                      )}
                    />
                    <Box sx={{ mt: 1 }}>
                      {hasUnsavedChanges ? (
                        <Box sx={{ p: 1.5, bgcolor: 'warning.50', borderRadius: 1, border: '1px solid', borderColor: 'warning.200' }}>
                          <Typography variant="caption" color="warning.dark" sx={{ display: 'block', fontWeight: 600, mb: 1 }}>
                            ⚠️ Unsaved Override: "{localSearchTerm.toUpperCase()}"
                          </Typography>
                          {termExistsInSearch ? (
                            <Box sx={{ display: 'flex', gap: 1, flexDirection: 'column' }}>
                              <Typography variant="caption" color="text.secondary" sx={{ fontSize: '11px' }}>
                                This term exists! Click below to apply it.
                              </Typography>
                              <Button
                                onClick={() => {
                                  const term = localResults.find((r: SemanticTerm) => 
                                    r.term_name.toUpperCase() === localSearchTerm.trim().toUpperCase()
                                  );
                                  if (term) {
                                    props.selectSemanticTerm(term, getUniqueId());
                                    setHasUnsavedChanges(false);
                                  }
                                }}
                                variant="contained"
                                size="small"
                                fullWidth
                                color="success"
                              >
                                ✓ Apply Existing Term
                              </Button>
                            </Box>
                          ) : (
                            <Box sx={{ display: 'flex', gap: 1, flexDirection: 'column' }}>
                              <Typography variant="caption" color="text.secondary" sx={{ fontSize: '11px' }}>
                                This term doesn't exist yet. Create it first.
                              </Typography>
                              <Button
                                onClick={async () => {
                                  await props.handleCreateAndSelectTerm(getUniqueId(), localSearchTerm.trim());
                                  setHasUnsavedChanges(false);
                                }}
                                variant="contained"
                                size="small"
                                fullWidth
                                color="primary"
                              >
                                ➕ Create & Apply New Term
                              </Button>
                            </Box>
                          )}
                        </Box>
                      ) : (
                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', fontSize: '11px' }}>
                          💡 Search for an existing term or type a new name to override
                        </Typography>
                      )}
                    </Box>
                  </Box>
                ) : (
                  <Box 
                    sx={{ 
                      bgcolor: 'primary.main', 
                      border: '1px solid', 
                      borderColor: 'primary.dark', 
                      p: 1, 
                      borderRadius: 1, 
                      fontFamily: 'monospace', 
                      fontSize: '13px', 
                      color: 'white',
                      fontWeight: 600,
                      textAlign: 'center'
                    }}
                  >
                    {mapping.semantic_term || 'No term assigned'}
                  </Box>
                )}
              </Grid>
            </Grid>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, mt: 0.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: 1 }}>
                <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ whiteSpace: 'nowrap', fontSize: '11px' }}>
                  {getConfidenceIcon(mapping.confidence)} {(mapping.confidence * 100).toFixed(0)}%
                </Typography>
                <Chip 
                  label={getConfidenceLabel(mapping.confidence)} 
                  size="small" 
                  color={getConfidenceColorMUI(mapping.confidence) as any} 
                  sx={{ height: 18, fontSize: '10px', fontWeight: 600 }} 
                />
                <Box sx={{ width: 100, ml: 0.5 }}>
                    <LinearProgress 
                    variant="determinate" 
                    value={mapping.confidence * 100} 
                    sx={{ 
                      height: 4, 
                      borderRadius: 2, 
                      '& .MuiLinearProgress-bar': { 
                        bgcolor: mapping.confidence >= 0.75 ? '#4caf50' : mapping.confidence >= 0.6 ? '#2196f3' : mapping.confidence >= 0.4 ? '#ff9800' : '#f44336',
                        borderRadius: 2
                      } 
                    }} 
                  />
                </Box>
                {mapping.match_reason && (
                  <Tooltip title={mapping.match_reason} arrow>
                    <Typography variant="caption" color="text.secondary" sx={{ cursor: 'help' }}>💡</Typography>
                  </Tooltip>
                )}
              </Box>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
                {localOverride && mapping.semantic_term_id && !mapping.edge_exists && (
                  <Chip 
                    label="Ready to Create Edge" 
                    size="small" 
                    color="success"
                    icon={<Tag size={14} />}
                    sx={{ 
                      height: 20, 
                      fontSize: '10px', 
                      fontWeight: 600,
                      animation: 'pulse 2s ease-in-out infinite',
                      '@keyframes pulse': {
                        '0%, 100%': { opacity: 1 },
                        '50%': { opacity: 0.7 }
                      }
                    }} 
                  />
                )}
                <Stack direction="row" spacing={0.5} alignItems="center">
                  <Tooltip title={localOverride ? "Override enabled - Click to disable" : "Enable override to edit semantic term"} arrow>
                    <IconButton 
                      size="small" 
                      onClick={(e) => { 
                        e.preventDefault();
                        e.stopPropagation(); 
                        const newOverrideState = !localOverride;
                        const uniqueId = getUniqueId();
                        devDebug(`Setting override for mapping with uniqueId: ${uniqueId}`);
                        setLocalOverride(newOverrideState);
                        props.setOverride(uniqueId, newOverrideState); 
                      }} 
                      sx={{ 
                        color: localOverride ? 'warning.main' : 'grey.400',
                        '&:hover': { 
                          backgroundColor: localOverride ? 'warning.50' : 'grey.50',
                          color: localOverride ? 'warning.dark' : 'grey.600'
                        }
                      }}
                    >
                      <Edit3 size={16} />
                    </IconButton>
                  </Tooltip>
                  
                  <Tooltip title={mapping.ignored ? "Currently ignored - Click to unignore" : "Ignore this mapping"} arrow>
                    <IconButton 
                      size="small" 
                      onClick={(e) => { 
                        e.preventDefault();
                        e.stopPropagation(); 
                        props.setIgnored(getUniqueId(), !mapping.ignored); 
                      }} 
                      sx={{ 
                        color: mapping.ignored ? 'error.main' : 'grey.400',
                        '&:hover': { 
                          backgroundColor: mapping.ignored ? 'error.50' : 'grey.50',
                          color: mapping.ignored ? 'error.dark' : 'grey.600'
                        }
                      }}
                    >
                      <AlertCircle size={16} />
                    </IconButton>
                  </Tooltip>

                  {mapping.edge_exists && (
                    <Tooltip title="Mapping exists in database" arrow>
                      <Chip 
                        icon={<Link2 size={14} />} 
                        label="Mapped" 
                        size="small" 
                        color="success" 
                        sx={{ height: 20, fontSize: '10px', fontWeight: 600 }} 
                      />
                    </Tooltip>
                  )}
                  {mapping.is_pending && !mapping.edge_exists && (
                    <Tooltip title="Pending Approval: This mapping was suggested by AI but not yet approved." arrow>
                       <Chip 
                         label="Pending Review" 
                         size="small" 
                         color="warning" 
                         sx={{ height: 20, fontSize: '10px', fontWeight: 600 }} 
                       />
                    </Tooltip>
                  )}
                </Stack>
              </Box>
            </Box>
            {props.compactRows && props.keyboardExpanded && mapping.match_reason && (
              <Box sx={{ mt: 1, p: 1, bgcolor: 'grey.50', borderRadius: 1, border: '1px solid', borderColor: 'grey.200' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5 }}>
                  <span className="info-icon">💡</span>
                  <span><strong>Why this score:</strong> {mapping.match_reason}</span>
                </Typography>
              </Box>
            )}
            {!props.compactRows && mapping.match_reason && (
              <Box sx={{ mt: 1, p: 1, bgcolor: 'grey.50', borderRadius: 1, border: '1px solid', borderColor: 'grey.200' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5 }}>
                  <span className="info-icon">💡</span>
                  <span><strong>Why this score:</strong> {mapping.match_reason}</span>
                </Typography>
              </Box>
            )}
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
}
