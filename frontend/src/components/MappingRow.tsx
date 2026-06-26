import { useState, useEffect } from 'react';
import { Check, X, Database, Tag, ArrowRight } from 'lucide-react';
import {
  Grid, Card, CardContent, Button, Box, Typography, Chip, LinearProgress,
  Stack, IconButton, Autocomplete, TextField, Tooltip, Link
} from '@mui/material';
import { getMappingUniqueId } from '../utils/mappingId';

interface MappingRowProps {
  mapping: any;
  idx: number;
  editingMapping: any | null;
  savedRows: Set<string>;
  compactRows: boolean;
  keyboardExpanded: boolean;
  setKeyboardExpanded: (expanded: boolean) => void;
  toggleMapping: (id: string) => void;
  openEditing: (mapping: any) => void;
  confirmEditing: (id: string, term?: string) => void;
  cancelEditing: (id: string) => void;
  searchSemanticTerms: (query: string) => Promise<any[]>;
  selectSemanticTerm: (term: any, mappingId: string) => void;
  handleCreateAndSelectTerm: (mappingId: string, termName: string) => Promise<any>;
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

const getConfidenceColorMUI = (conf: number): 'success' | 'info' | 'warning' | 'error' => {
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
  const [localResults, setLocalResults] = useState<any[]>([]);
  const uniqueId = getMappingUniqueId(mapping);
  const editingUniqueId = props.editingMapping ? getMappingUniqueId(props.editingMapping) : null;
  // Local narrow for chip color to avoid scattered `as any` casts
  const confColor = getConfidenceColorMUI(mapping.confidence) as 'success' | 'info' | 'warning' | 'error';

  useEffect(() => {
    if (editingUniqueId !== uniqueId) {
      setLocalSearchTerm(mapping.semantic_term || '');
      setLocalResults([]);
    }
  }, [editingUniqueId, mapping.semantic_term, uniqueId]);

  return (
    <Card
      className={`mapping-card ${mapping.edge_exists ? 'mapped' : ''} ${mapping.override ? 'override' : ''}`}
      sx={{
        borderRadius: 2, border: mapping.selected ? '2px solid' : '1px solid',
        borderColor: mapping.selected ? 'primary.main' : 'divider', transition: 'all 0.2s ease',
        '&:hover': { transform: 'translateY(-2px)', boxShadow: 4 }
      }}
      elevation={mapping.selected ? 3 : 1}
      onClick={(e) => {
        const el = (e.target as HTMLElement);
        if (el.closest('button, input, a, .MuiAutocomplete-root, .MuiAutocomplete-input')) return;
        props.toggleMapping(uniqueId);
      }}
    >
      <CardContent>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Box sx={{ pt: 0.5 }}>
            <input id={`select-${uniqueId}`} type="checkbox" aria-label={`Select mapping ${idx + 1}`} checked={mapping.selected || false} onChange={(e) => { e.stopPropagation(); props.toggleMapping(uniqueId); }} className={`select-checkbox ${mapping.edge_exists ? 'disabled' : ''}`} disabled={mapping.edge_exists} />
            <label htmlFor={`select-${uniqueId}`} className="visually-hidden">Select mapping</label>
          </Box>
          <Box sx={{ flex: 1 }}>
            <Grid container spacing={2} alignItems="center">
              <Grid item xs={12} md={5}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                  <Database width={14} height={14} style={{ color: '#666' }} />
                  <Typography variant="caption" color="text.secondary" fontWeight={600}>Database Column</Typography>
                </Box>
                <Box className="mono-box" sx={{ bgcolor: 'grey.50', border: '1px solid', borderColor: 'grey.200' }}>
                  {mapping.database_column.schema && (<span className="db-schema">{mapping.database_column.schema}.</span>)}
                  <span className="db-table">{mapping.database_column.table}</span>
                  <span className="db-dot">.</span>
                  <span className="db-col">{mapping.database_column.column}</span>
                </Box>
                {mapping.database_column.data_type && (<Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>Type: <strong className="type-pill">{mapping.database_column.data_type}</strong></Typography>)}
              </Grid>
              <Grid item xs={12} md={1} sx={{ textAlign: 'center' }}><ArrowRight width={24} height={24} style={{ color: '#999' }} /></Grid>
              <Grid item xs={12} md={6}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                  <Tag width={14} height={14} style={{ color: '#666' }} />
                  <Typography variant="caption" color="text.secondary" fontWeight={600}>Semantic Term</Typography>
                  {mapping.is_new_term && (<Chip label="NEW" size="small" color="success" sx={{ height: 18, fontSize: '10px' }} />)}
                </Box>
                {editingUniqueId === uniqueId ? (
                  <Box sx={{ position: 'relative' }}>
                    <Autocomplete freeSolo options={localResults || []} getOptionLabel={(option: any) => typeof option === 'string' ? option : (option.term_name || '')} filterOptions={(x) => x} inputValue={localSearchTerm}
                      onInputChange={async (_e, value) => {
                        setLocalSearchTerm(value || '');
                        if ((value || '').length >= 2) { const res = await props.searchSemanticTerms(value || ''); setLocalResults(res || []); } else setLocalResults([]);
                      }}
                      onChange={async (_e, value) => {
                        if (!value) return;
                        if (typeof value === 'string') setLocalSearchTerm(value);
                        else { props.selectSemanticTerm(value, uniqueId); setLocalSearchTerm(value.term_name || ''); }
                      }}
                      renderInput={(params) => (<TextField {...params} placeholder="Search semantic terms..." className="search-input" onKeyDown={async (e) => { if (e.key === 'Enter') await props.confirmEditing(uniqueId, localSearchTerm); if (e.key === 'Escape') props.cancelEditing(uniqueId); }} autoFocus />)}
                    />
                    {localSearchTerm.length >= 2 && !mapping.edge_exists && (!localResults || localResults.length === 0) && (
                      <Box sx={{ mt: 1 }}><Button onClick={async () => await props.handleCreateAndSelectTerm(uniqueId, localSearchTerm)} variant="outlined" size="small" fullWidth className="create-new-btn">➕ Create New: "{localSearchTerm.toUpperCase()}"</Button></Box>
                    )}
                    <Box sx={{ position: 'absolute', right: 4, top: 4, display: 'flex', gap: 1 }}>
                      <IconButton onClick={async () => await props.confirmEditing(uniqueId, localSearchTerm)} size="small" aria-label={`Save mapping ${mapping.database_column.column}`}><Check width={16} height={16} /></IconButton>
                      <IconButton onClick={() => props.cancelEditing(uniqueId)} size="small" aria-label={`Cancel editing ${mapping.database_column.column}`}><X width={16} height={16} /></IconButton>
                    </Box>
                  </Box>
                ) : (
                  <Button onClick={(e) => { e.stopPropagation(); props.openEditing(mapping); }} variant="outlined" fullWidth sx={{ justifyContent: 'flex-start', fontFamily: 'monospace', fontSize: '13px', bgcolor: 'primary.main', borderColor: 'primary.dark', color: 'white', fontWeight: 600, textTransform: 'none', '&:hover': { bgcolor: 'primary.dark', color: 'white' }, '&.Mui-disabled': { color: 'white', bgcolor: 'primary.main', borderColor: 'primary.dark', opacity: 1 } }} disabled={mapping.edge_exists}>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                      <span>{mapping.semantic_term}</span>
                      {props.savedRows.has(uniqueId) && (<Chip label="Saved" size="small" color="success" sx={{ ml: 1, height: 22 }} />)}
                      {(mapping.semantic_term_id || mapping.edge_exists) && (
                        <IconButton size="small" onClick={(e) => { e.stopPropagation(); props.openLineageModal(mapping); }} aria-label={`Open lineage for ${mapping.semantic_term || uniqueId}`} title="Open lineage">
                          <span className="lineage-icon">🔗</span>
                        </IconButton>
                      )}
                    </Box>
                  </Button>
                )}
              </Grid>
            </Grid>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, mt: 1 }}>
              {props.compactRows ? (
                <Tooltip title={(<Box sx={{ maxWidth: 360 }}><Typography variant="subtitle2" sx={{ fontWeight: 700 }}>Why this score</Typography><Typography variant="body2" sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }}>{mapping.match_reason}</Typography><Box sx={{ mt: 1 }}><Link href={`/#/mappings/${uniqueId}`} underline="hover">Open mapping details</Link></Box></Box>)} arrow>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }} tabIndex={0} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); props.setKeyboardExpanded(!props.keyboardExpanded); } }}>
                    <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ whiteSpace: 'nowrap' }}>Confidence:</Typography>
                    <Typography variant="body2" fontWeight={700} sx={{ fontSize: '13px', whiteSpace: 'nowrap' }}>{getConfidenceIcon(mapping.confidence)} {(mapping.confidence * 100).toFixed(1)}%</Typography>
                    <Chip label={getConfidenceLabel(mapping.confidence)} size="small" color={confColor} sx={{ height: 20, fontSize: '11px', fontWeight: 600 }} />
                    <Box sx={{ width: 160, ml: 1 }}><LinearProgress variant="determinate" value={mapping.confidence * 100} sx={{ height: 6, borderRadius: 3, bgcolor: 'grey.200', '& .MuiLinearProgress-bar': { bgcolor: mapping.confidence >= 0.75 ? '#4caf50' : mapping.confidence >= 0.6 ? '#2196f3' : mapping.confidence >= 0.4 ? '#ff9800' : '#f44336', transition: 'transform 0.4s ease' } }} /></Box>
                    {mapping.match_reason && (<Typography variant="caption" color="text.secondary" sx={{ ml: 1 }}>💡</Typography>)}
                  </Box>
                </Tooltip>
              ) : (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ whiteSpace: 'nowrap' }}>Confidence:</Typography>
                  <Typography variant="body2" fontWeight={700} sx={{ fontSize: '13px', whiteSpace: 'nowrap' }}>{getConfidenceIcon(mapping.confidence)} {(mapping.confidence * 100).toFixed(1)}%</Typography>
                  <Chip label={getConfidenceLabel(mapping.confidence)} size="small" color={confColor} sx={{ height: 20, fontSize: '11px', fontWeight: 600 }} />
                  <Box sx={{ width: 160, ml: 1 }}><LinearProgress variant="determinate" value={mapping.confidence * 100} sx={{ height: 6, borderRadius: 3, bgcolor: 'grey.200', '& .MuiLinearProgress-bar': { bgcolor: mapping.confidence >= 0.75 ? '#4caf50' : mapping.confidence >= 0.6 ? '#2196f3' : mapping.confidence >= 0.4 ? '#ff9800' : '#f44336', transition: 'transform 0.4s ease' } }} /></Box>
                </Box>
              )}
              <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <input type="checkbox" checked={!!mapping.override} onChange={(e) => { e.stopPropagation(); props.setOverride(uniqueId, e.target.checked); }} id={`override-${uniqueId}`} aria-label={`Override mapping ${idx + 1}`} />
                  <Typography variant="caption" component="label" htmlFor={`override-${uniqueId}`} sx={{ cursor: 'pointer' }}>Override</Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <input type="checkbox" checked={!!mapping.ignored} onChange={(e) => { e.stopPropagation(); props.setIgnored(uniqueId, e.target.checked); }} id={`ignore-${uniqueId}`} aria-label={`Ignore mapping ${idx + 1}`} />
                  <Typography variant="caption" component="label" htmlFor={`ignore-${uniqueId}`} sx={{ cursor: 'pointer' }}>Ignore</Typography>
                </Box>
                <Stack direction="row" spacing={1} alignItems="center">
                  {mapping.override && (<Chip label="Override" size="small" color="warning" icon={<span>🔧</span>} sx={{ height: 22, fontSize: '11px', fontWeight: 600 }} />)}
                  {mapping.edge_exists && (<Chip label="Mapped" size="small" color="success" icon={<span>✅</span>} className="mapped-chip" sx={{ height: 22, fontSize: '11px', fontWeight: 600 }} title="This mapping already exists in the database" />)}
                  {mapping.edge_exists && (<Button size="small" variant="outlined" color="warning" onClick={() => props.openReplaceConfirm(idx)} aria-label={`Replace mapping for ${mapping.database_column.column}`} title="Replace this mapping (will cascade deletions to linked items)" startIcon={<span>🔄</span>}>Replace</Button>)}
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