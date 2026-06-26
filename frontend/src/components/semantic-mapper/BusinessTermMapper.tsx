import { useState, useEffect, useMemo, useCallback } from 'react';
import { 
  Box, Typography, Card, Autocomplete, TextField, Button, Chip, 
  Grid, Paper, CircularProgress,
  FormControl, InputLabel, Select, MenuItem, Collapse, IconButton,
  Dialog, DialogTitle, DialogContent, DialogActions
} from '@mui/material';
import { 
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Edit as EditIcon,
  Add as AddIcon,
  Check as CheckIcon,
  Close as CloseIcon,
  AutoAwesome as AutoAwesomeIcon,
  AutoFixHigh
} from '@mui/icons-material';
import { useSemanticMapper } from './useSemanticMapper';
import type { SemanticTerm } from './types';
import { devDebug, devError } from '../../utils/devLogger';

// Helper function to convert UPPER_CASE_WITH_UNDERSCORES to Title Case
function formatBusinessTermName(name: string): string {
  if (!name) return '';
  
  // Replace underscores with spaces and convert to lowercase
  const withSpaces = name.replace(/_/g, ' ').toLowerCase();
  
  // Capitalize first letter of each word
  return withSpaces
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

interface BusinessTermSuggestion {
  semantic_term_id: string;
  business_term_id?: string;
  business_term_name: string;
  confidence: number;
  reason: string;
  description?: string;
}

interface BusinessTermMapping {
  semantic_term: SemanticTerm;
  selected_business_term: SemanticTerm | null;
  override: boolean;
  edge_exists: boolean;
}

interface EnhancedMappingRowProps {
  mapping: BusinessTermMapping;
  businessTerms: SemanticTerm[];
  suggestions: BusinessTermSuggestion[]; // Suggestions for this specific term
  onSelectBusinessTerm: (semanticTermId: string, businessTerm: SemanticTerm | null) => void;
  // allow onSave to return a Promise so callers can be async
  onSave: (semanticTermId: string) => void | Promise<void>;
  onUpdateBusinessTerm: (termNodeId: string, updates: Record<string, any>) => Promise<boolean>;
  onCreateBusinessTerm: (termName: string, category: string, description: string) => Promise<SemanticTerm | null>;
  onAcceptSuggestion: (suggestion: BusinessTermSuggestion) => Promise<void>;
  onRejectSuggestion: (suggestion: BusinessTermSuggestion) => Promise<void>;
  isLast: boolean;
}

function EnhancedMappingRow({ 
  mapping, 
  businessTerms, 
  suggestions,
  onSelectBusinessTerm, 
  onSave, 
  onUpdateBusinessTerm,
  onCreateBusinessTerm,
  onAcceptSuggestion,
  onRejectSuggestion,
  isLast 
}: EnhancedMappingRowProps) {
  const [expanded, setExpanded] = useState(false);
  const [showCustomEntry, setShowCustomEntry] = useState(false);
  const [customTermName, setCustomTermName] = useState('');
  const [customCategory, setCustomCategory] = useState('');
  const [customDescription, setCustomDescription] = useState('');
  const [editMode, setEditMode] = useState(false);
  const [editingTerm, setEditingTerm] = useState<SemanticTerm | null>(null);
  const [saving, setSaving] = useState(false);
  
  const semanticTermId = mapping.semantic_term.node_id as string;
  
  const handleCreateCustomTerm = useCallback(async () => {
    if (!customTermName.trim()) return;
    
    try {
      setSaving(true);
      
      // Format the name as Title Case
      const formattedName = formatBusinessTermName(customTermName);
      
  devDebug('[handleCreateCustomTerm] Creating business term:', {
        formattedName,
        category: customCategory || 'General',
        description: customDescription || `Custom business term: ${formattedName}`
      });
      
      // Call backend to create the business term
      const newTerm = await onCreateBusinessTerm(
        formattedName,
        customCategory || 'General',
        customDescription || `Custom business term: ${formattedName}`
      );
      
      if (newTerm) {
  devDebug('[handleCreateCustomTerm] Business term created:', newTerm);
        
        // Select the newly created term
        onSelectBusinessTerm(semanticTermId, newTerm);
        
        // Clear form
        setCustomTermName('');
        setCustomCategory('');
        setCustomDescription('');
        setShowCustomEntry(false);
      }
    } catch (error) {
      devError('[handleCreateCustomTerm] Failed to create business term:', error);
    } finally {
      setSaving(false);
    }
  }, [customTermName, customCategory, customDescription, onSelectBusinessTerm, onCreateBusinessTerm, semanticTermId]);
  
  const handleEditBusinessTerm = useCallback((businessTerm: SemanticTerm) => {
    setEditingTerm(businessTerm);
    setEditMode(true);
  }, []);
  
  const handleUpdateTerm = useCallback(async () => {
    if (!editingTerm?.node_id) return;
    
    const updates = {
      term_name: editingTerm.term_name,
      description: customDescription || 'Updated business term',
      category: customCategory || 'General'
    };
    
    const success = await onUpdateBusinessTerm(editingTerm.node_id, updates);
    if (success) {
      setEditMode(false);
      setEditingTerm(null);
    }
  }, [editingTerm, customDescription, customCategory, onUpdateBusinessTerm]);
  
  return (
    <Box sx={{ 
      borderBottom: isLast ? 0 : 1, 
      borderColor: 'divider',
      '&:hover': { bgcolor: 'action.hover' }
    }}>
      {/* Main Row */}
      <Box 
        sx={{ 
          p: 2, 
          cursor: mapping.edge_exists ? 'default' : 'pointer',
          display: 'flex', 
          alignItems: 'center',
          justifyContent: 'space-between'
        }}
        onClick={mapping.edge_exists ? undefined : () => setExpanded(!expanded)}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 200 }}>
            <Typography variant="subtitle2">
              {mapping.semantic_term.term_name}
            </Typography>
          </Box>
          
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flex: 1 }}>
            {mapping.selected_business_term ? (
              <>
                <Chip 
                  label={mapping.selected_business_term.term_name} 
                  color="primary" 
                  size="small"
                  onDelete={() => onSelectBusinessTerm(semanticTermId, null)}
                />
                <IconButton 
                  size="small" 
                  onClick={(e) => {
                    e.stopPropagation();
                    handleEditBusinessTerm(mapping.selected_business_term!);
                  }}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
              </>
            ) : (
              <Typography variant="body2" color="text.secondary">
                No business term selected
              </Typography>
            )}
          </Box>
          
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {suggestions && suggestions.length > 0 && !mapping.edge_exists && (
              <Chip
                label={`Suggestions (${suggestions.length})`}
                size="small"
                color="info"
                onClick={(e) => {
                  e.stopPropagation();
                  setExpanded(true);
                }}
                sx={{ cursor: 'pointer' }}
              />
            )}
            
            {mapping.edge_exists ? (
              <Chip label="Mapped" color="success" size="small" />
            ) : mapping.selected_business_term ? (
              <Chip label="Ready" color="warning" size="small" />
            ) : (
              <Chip label="Unmapped" color="default" size="small" />
            )}
          </Box>
        </Box>
        
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {mapping.selected_business_term && !mapping.edge_exists && (
            <Button
              variant="contained"
              size="small"
              disabled={saving}
              onClick={async (e) => {
                e.stopPropagation();
                try {
                  setSaving(true);
                  // Support both sync and async onSave implementations
                  const maybePromise: any = onSave(semanticTermId as string);
                  if (maybePromise && typeof maybePromise.then === 'function') {
                    await maybePromise;
                  }
                } catch (err) {
                  devError('Save mapping failed:', err);
                } finally {
                  setSaving(false);
                }
              }}
            >
              {saving ? <CircularProgress size={18} color="inherit" /> : 'Save Mapping'}
            </Button>
          )}
          {!mapping.edge_exists && (expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />)}
        </Box>
      </Box>
      
      {/* Expanded Content */}
      <Collapse in={expanded}>
        <Box sx={{ p: 2, pt: 0, bgcolor: 'background.default' }}>
          
          {/* AI Suggestions Section (Inline) - shown when suggestions exist */}
          {!mapping.edge_exists && suggestions.length > 0 && (
            <Box sx={{ mb: 2, p: 2, bgcolor: 'action.hover', borderRadius: 1 }}>
              <Typography variant="subtitle2" sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <AutoAwesomeIcon fontSize="small" color="primary" />
                AI Suggestions
              </Typography>

              <Grid container spacing={1}>
                {suggestions.map((suggestion, idx) => (
                  <Grid item xs={12} sm={4} key={`${suggestion.business_term_name}-${idx}`}>
                    <Paper sx={{ p: 1.5, border: 1, borderColor: 'divider' }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', mb: 0.5 }}>
                        <Typography variant="body2" fontWeight="bold">
                          {formatBusinessTermName(suggestion.business_term_name)}
                        </Typography>
                        <Chip
                          label={`${(suggestion.confidence * 100).toFixed(0)}%`}
                          size="small"
                          color={suggestion.confidence > 0.8 ? 'success' : suggestion.confidence > 0.6 ? 'info' : 'default'}
                        />
                      </Box>
                      <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block', minHeight: '32px' }}>
                        {suggestion.reason}
                      </Typography>
                      <Box sx={{ display: 'flex', gap: 0.5 }}>
                        <Button
                          size="small"
                          variant="contained"
                          color="success"
                          startIcon={<CheckIcon />}
                          onClick={(e) => {
                            e.stopPropagation();
                            onAcceptSuggestion(suggestion);
                          }}
                          fullWidth
                        >
                          Accept
                        </Button>
                        <Button
                          size="small"
                          variant="outlined"
                          color="error"
                          startIcon={<CloseIcon />}
                          onClick={(e) => {
                            e.stopPropagation();
                            onRejectSuggestion(suggestion);
                          }}
                          fullWidth
                        >
                          Reject
                        </Button>
                      </Box>
                    </Paper>
                  </Grid>
                ))}
              </Grid>
            </Box>
          )}
          
          {/* Manual Selection and Custom Entry - only show if not already mapped */}
          {!mapping.edge_exists && (
            <>
              {/* Manual Selection */}
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  Select Existing Business Term
                </Typography>
                <Autocomplete
                  options={businessTerms}
                  getOptionLabel={(option) => option.term_name}
                  value={mapping.selected_business_term}
                  onChange={(_, value) => onSelectBusinessTerm(semanticTermId, value)}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Choose from existing business terms"
                      variant="outlined"
                      size="small"
                      fullWidth
                    />
                  )}
                  renderOption={(props, option) => {
                    const { key, ...otherProps } = props;
                    return (
                      <li key={key} {...otherProps}>
                        <Box>
                          <Typography variant="body2">{option.term_name}</Typography>
                          <Typography variant="caption" color="text.secondary">
                            {option.data_type || 'No description'}
                          </Typography>
                        </Box>
                      </li>
                    );
                  }}
                />
              </Box>
              
              {/* Custom Entry */}
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="subtitle2">
                    Create New Business Term
                  </Typography>
                  <Button
                    size="small"
                    startIcon={<AddIcon />}
                    onClick={() => setShowCustomEntry(!showCustomEntry)}
                  >
                    {showCustomEntry ? 'Cancel' : 'Create Custom'}
                  </Button>
                </Box>
                
                <Collapse in={showCustomEntry}>
                  <Grid container spacing={2} sx={{ mt: 1 }}>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        label="Business Term Name"
                        value={customTermName}
                        onChange={(e) => setCustomTermName(e.target.value)}
                        size="small"
                        fullWidth
                        required
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        label="Category"
                        value={customCategory}
                        onChange={(e) => setCustomCategory(e.target.value)}
                        size="small"
                        fullWidth
                      />
                    </Grid>
                    <Grid item xs={12}>
                      <TextField
                        label="Description"
                        value={customDescription}
                        onChange={(e) => setCustomDescription(e.target.value)}
                        size="small"
                        fullWidth
                        multiline
                        rows={2}
                      />
                    </Grid>
                    <Grid item xs={12}>
                      <Button
                        variant="contained"
                        color="primary"
                        onClick={handleCreateCustomTerm}
                        disabled={!customTermName.trim()}
                      >
                        Create & Map
                      </Button>
                    </Grid>
                  </Grid>
                </Collapse>
              </Box>
            </>
          )}
        </Box>
      </Collapse>
      
      {/* Edit Dialog */}
      <Dialog open={editMode} onClose={() => setEditMode(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Business Term</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12}>
              <TextField
                label="Term Name"
                value={editingTerm?.term_name || ''}
                onChange={(e) => setEditingTerm(prev => prev ? {...prev, term_name: e.target.value} : null)}
                fullWidth
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                label="Category"
                value={customCategory}
                onChange={(e) => setCustomCategory(e.target.value)}
                fullWidth
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                label="Description"
                value={customDescription}
                onChange={(e) => setCustomDescription(e.target.value)}
                fullWidth
                multiline
                rows={3}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditMode(false)}>Cancel</Button>
          <Button onClick={handleUpdateTerm} variant="contained">
            Update
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export function BusinessTermMapper({ searchTerm }: { searchTerm?: string }) {
  const {
    loadSemanticTerms,
    loadBusinessTerms,
    setToast,
    updateBusinessTerm,
    recordSuggestionFeedback
  } = useSemanticMapper();

  const [businessTerms, setBusinessTerms] = useState<SemanticTerm[]>([]);
  const [businessTermMappings, setBusinessTermMappings] = useState<Record<string, BusinessTermMapping>>({});
  // const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'mapped' | 'unmapped'>('all');
  const [loading, setLoading] = useState(true);
  const [allSuggestions, setAllSuggestions] = useState<Record<string, BusinessTermSuggestion[]>>({});
  const [generatingSuggestions, setGeneratingSuggestions] = useState(false);

  // Load initial data
  useEffect(() => {
    const initializeData = async () => {
      try {
        setLoading(true);
    devDebug('[initializeData] Loading semantic and business terms...');
        
        const [semanticData, businessData] = await Promise.all([
          loadSemanticTerms(),
          loadBusinessTerms()
        ]);
        setBusinessTerms(businessData);
        
  devDebug(`[initializeData] Loaded ${semanticData.length} semantic terms, ${businessData.length} business terms`);
        
        // Fetch existing edges to populate mappings (use relative URL for tenant scope)
        const existingEdgesResponse = await fetch(`/api/business-term-edges`, {
          method: 'GET',
          credentials: 'include'
        });
        
        let existingEdges: any[] = [];
        if (existingEdgesResponse.ok) {
          const edgesData = await existingEdgesResponse.json();
          existingEdges = Array.isArray(edgesData) ? edgesData : [];
          devDebug(`[initializeData] Loaded ${existingEdges.length} existing edges`);
        } else {
          const errorText = await existingEdgesResponse.text();
          devError('[initializeData] Failed to load edges:', existingEdgesResponse.status, errorText);
        }
        
        // Create a map of semantic_term_id -> business_term_id from existing edges
        const edgeMap = new Map<string, string>();
        existingEdges.forEach(edge => {
          // Edge structure: { source_node_id: business_term_id, target_node_id: semantic_term_id }
          if (edge.target_node_id && edge.source_node_id) {
            edgeMap.set(edge.target_node_id, edge.source_node_id);
          }
        });
        
  devDebug(`[initializeData] Created edge map with ${edgeMap.size} entries`);
        
        // Initialize mappings for semantic terms that have database columns
        const initialMappings: Record<string, BusinessTermMapping> = {};
        semanticData.forEach(term => {
          if (term.node_id) {
            const businessTermId = edgeMap.get(term.node_id);
            const businessTerm = businessTermId 
              ? businessData.find(bt => bt.node_id === businessTermId) 
              : null;
            
            initialMappings[term.node_id] = {
              semantic_term: term,
              selected_business_term: businessTerm || null,
              override: false,
              edge_exists: !!businessTerm
            };
          }
        });
        setBusinessTermMappings(initialMappings);
      } catch (error) {
  devError('Failed to load initial data:', error);
        setToast({ type: 'error', message: 'Failed to load business terms data' });
      } finally {
        setLoading(false);
      }
    };

    initializeData();
  }, []); // Empty dependency array - only run on mount

  const handleSelectBusinessTerm = useCallback((semanticTermId: string, businessTerm: SemanticTerm | null) => {
    setBusinessTermMappings(prev => ({
      ...prev,
      [semanticTermId]: {
        ...prev[semanticTermId],
        selected_business_term: businessTerm,
        override: true
      }
    }));
  }, []);

  const createBusinessTermEdgeWithCorrectType = async (semanticTermId: string, businessTermId: string) => {
    try {
      const response = await fetch('/api/business-term-edges', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          subject_node_id: businessTermId, // Business term as subject
          object_node_id: semanticTermId,  // Semantic term as object
          edge_type_id: '3be9d6ae-1598-4628-a3dd-b606921a9193', // Correct UUID for business term to semantic term mapping
          relationship_type: 'business_term_to_semantic_term'
        })
      });

      if (!response.ok) {
        throw new Error('Failed to create business term edge');
      }

      return await response.json();
    } catch (error) {
      devError('Failed to create business term edge:', error);
      throw error;
    }
  };

  const handleSave = useCallback(async (semanticTermId: string) => {
    const mapping = businessTermMappings[semanticTermId];
    const businessTerm = mapping?.selected_business_term;
    
    if (!businessTerm?.node_id) {
      setToast({ type: 'error', message: 'No business term selected' });
      return;
    }
    
    try {
  devDebug('[handleSave] Creating edge:', {
        semanticTermId,
        businessTermId: businessTerm.node_id,
        businessTermName: businessTerm.term_name
      });
      
      const result = await createBusinessTermEdgeWithCorrectType(semanticTermId, businessTerm.node_id);
      
  devDebug('[handleSave] Edge created successfully:', result);
      
      // Update mapping to reflect edge creation
      setBusinessTermMappings(prev => ({
        ...prev,
        [semanticTermId]: {
          ...prev[semanticTermId],
          edge_exists: true
        }
      }));
      
      setToast({ type: 'success', message: `Created business term mapping for "${businessTerm.term_name}"` });
    } catch (error) {
  devError('[handleSave] Failed to create edge:', error);
      setToast({ type: 'error', message: `Failed to create business term mapping: ${error instanceof Error ? error.message : 'Unknown error'}` });
      throw error; // Re-throw so the Save button can handle it
    }
  }, [businessTermMappings, setToast]);

  // Generate suggestions for all unmapped terms in batch
  const handleGenerateAllSuggestions = useCallback(async () => {
    setGeneratingSuggestions(true);
    
    try {
      // Find all unmapped semantic terms
      const unmappedTerms = Object.values(businessTermMappings).filter(
        mapping => !mapping.edge_exists && !mapping.selected_business_term
      );
      
      if (unmappedTerms.length === 0) {
        setToast({ type: 'info', message: 'All terms are already mapped' });
        setGeneratingSuggestions(false);
        return;
      }
      
      setToast({ type: 'info', message: `Generating suggestions for ${unmappedTerms.length} terms...` });
      
      // Fetch suggestions for all unmapped terms in parallel
      const suggestionPromises = unmappedTerms.map(async (mapping) => {
        const semanticTermId = mapping.semantic_term.node_id;
        if (!semanticTermId) return null;
        
        try {
          // Use relative URL to go through tenant scope
          const response = await fetch(
            `/api/semantic-terms/${semanticTermId}/suggest-business-terms`,
            {
              method: 'GET',
              headers: { 'Content-Type': 'application/json' },
              credentials: 'include'
            }
          );
          
          if (response.ok) {
            const suggestions = await response.json();
            if (Array.isArray(suggestions)) {
              devDebug(`[handleGenerateAllSuggestions] Got ${suggestions.length} suggestions for ${semanticTermId}`);
              return {
                semanticTermId,
                suggestions: suggestions.map((s: any) => ({
                  semantic_term_id: semanticTermId,
                  business_term_id: s.business_term_id,
                  business_term_name: s.term_name,
                  confidence: s.confidence,
                  reason: s.reason,
                  description: s.description
                }))
              };
            }
          } else {
            const errorText = await response.text();
            devError(`[handleGenerateAllSuggestions] Failed for ${semanticTermId}:`, response.status, errorText);
          }
        } catch (error) {
          devError(`Failed to get suggestions for ${semanticTermId}:`, error);
        }
        return null;
      });
      
      const results = await Promise.all(suggestionPromises);
      
      // Build the suggestions map
      const newSuggestions: Record<string, BusinessTermSuggestion[]> = {};
      let totalCount = 0;
      
      results.forEach(result => {
        if (result && result.suggestions.length > 0) {
          newSuggestions[result.semanticTermId] = result.suggestions;
          totalCount += result.suggestions.length;
        }
      });
      
      setAllSuggestions(newSuggestions);
      setToast({ 
        type: 'success', 
        message: `Generated ${totalCount} suggestions for ${Object.keys(newSuggestions).length} terms` 
      });
    } catch (error) {
  devError('Failed to generate suggestions:', error);
      setToast({ type: 'error', message: 'Failed to generate suggestions' });
    } finally {
      setGeneratingSuggestions(false);
    }
  }, [businessTermMappings, setToast]);

  const handleAcceptSuggestion = useCallback(async (suggestion: BusinessTermSuggestion) => {
    try {
      // Format the business term name as Title Case
      const formattedName = formatBusinessTermName(suggestion.business_term_name);
      
      // Check if business term exists, create if not
      let businessTermId = suggestion.business_term_id;
      
      if (!businessTermId) {
    devDebug('[handleAcceptSuggestion] Creating new business term:', formattedName);
        
        // Create new business term (use relative URL to go through tenant scope)
        const createResponse = await fetch('/api/business-terms', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({
            term_name: formattedName,
            properties: { 
              description: suggestion.description || `Auto-created from suggestion`,
              category: 'General'
            }
          })
        });
        
        if (!createResponse.ok) {
          const errorText = await createResponse.text();
          devError('[handleAcceptSuggestion] Failed to create business term:', createResponse.status, errorText);
          setToast({ type: 'error', message: `Failed to create business term: ${errorText}` });
          return;
        }
        
        const created = await createResponse.json();
        businessTermId = created.node_id;
        
  devDebug('[handleAcceptSuggestion] Business term created:', created);
        
        // Add to business terms list
        setBusinessTerms(prev => [...prev, {
          node_id: businessTermId,
          term_name: created.term_name,
          data_type: 'business_term',
          qualified_path: created.qualified_path || `/business_terms/${created.term_name}`
        }]);
      }
      
  devDebug('[handleAcceptSuggestion] Creating edge:', {
        businessTermId,
        semanticTermId: suggestion.semantic_term_id
      });
      
      // Create the edge automatically (use relative URL for tenant scope)
      const edgeResponse = await fetch('/api/business-term-edges', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          subject_node_id: businessTermId, // business term as subject
          object_node_id: suggestion.semantic_term_id, // semantic term as object
          edge_type_id: '3be9d6ae-1598-4628-a3dd-b606921a9193',
          relationship_type: 'business_term_to_semantic_term'
        })
      });
      
        if (!edgeResponse.ok) {
        const errorText = await edgeResponse.text();
        devError('[handleAcceptSuggestion] Failed to create edge:', edgeResponse.status, errorText);
        setToast({ type: 'error', message: `Failed to create edge: ${errorText}` });
        return;
      }
      
      const edgeResult = await edgeResponse.json();
  devDebug('[handleAcceptSuggestion] Edge created successfully:', edgeResult);
      
      // Record accept feedback
      await recordSuggestionFeedback(
        suggestion.semantic_term_id,
        suggestion.business_term_name,
        'accept',
        businessTermId,
        suggestion.confidence,
        'User accepted suggestion'
      );
      
      // Record reject feedback for all other suggestions for this term
      const otherSuggestions = allSuggestions[suggestion.semantic_term_id] || [];
      await Promise.all(
        otherSuggestions
          .filter((s: BusinessTermSuggestion) => s.business_term_name !== suggestion.business_term_name)
          .map((s: BusinessTermSuggestion) => 
            recordSuggestionFeedback(
              s.semantic_term_id,
              s.business_term_name,
              'reject',
              s.business_term_id,
              s.confidence,
              'Auto-rejected when another suggestion was accepted'
            )
          )
      );
      
      // Update the mapping to show as mapped
      setBusinessTermMappings(prev => ({
        ...prev,
        [suggestion.semantic_term_id]: {
          ...prev[suggestion.semantic_term_id],
          selected_business_term: {
            node_id: businessTermId,
            term_name: formattedName,
            data_type: 'business_term',
            qualified_path: `/business_terms/${formattedName.replace(/ /g, '_')}`
          },
          edge_exists: true,
          override: false
        }
      }));
      
      // Remove all suggestions for this term from global state
      setAllSuggestions(prev => {
        const newSuggestions = { ...prev };
        delete newSuggestions[suggestion.semantic_term_id];
        return newSuggestions;
      });
      
      setToast({ type: 'success', message: `Mapped ${formattedName} and saved automatically` });
    } catch (error) {
  devError('Failed to accept suggestion:', error);
      setToast({ type: 'error', message: 'Failed to accept suggestion' });
    }
  }, [allSuggestions, recordSuggestionFeedback, setToast, setBusinessTerms]);

  const handleRejectSuggestion = useCallback(async (suggestion: BusinessTermSuggestion) => {
    try {
  devDebug('[handleRejectSuggestion] Rejecting:', {
        semantic_term_id: suggestion.semantic_term_id,
        business_term_name: suggestion.business_term_name,
        business_term_id: suggestion.business_term_id
      });
      
      // Record the feedback - this ensures it won't appear in future suggestions
      await recordSuggestionFeedback(
        suggestion.semantic_term_id,
        suggestion.business_term_name,
        'reject',
        suggestion.business_term_id,
        suggestion.confidence,
        'User rejected suggestion'
      );
      
  devDebug('[handleRejectSuggestion] Feedback recorded successfully');
      
      // Remove this specific suggestion from the global suggestions state
      setAllSuggestions(prev => {
        const termSuggestions = prev[suggestion.semantic_term_id] || [];
        const filtered = termSuggestions.filter(
          (s: BusinessTermSuggestion) => s.business_term_name !== suggestion.business_term_name
        );
        
  devDebug(`[handleRejectSuggestion] Removed suggestion, remaining: ${filtered.length}`);
        
        return {
          ...prev,
          [suggestion.semantic_term_id]: filtered
        };
      });
      
      setToast({ 
        type: 'success', 
        message: `Rejected: ${formatBusinessTermName(suggestion.business_term_name)} - This suggestion won't appear again` 
      });
    } catch (error) {
  devError('[handleRejectSuggestion] Failed to record feedback:', error);
      setToast({ type: 'error', message: 'Failed to record rejection. Please try again.' });
    }
  }, [recordSuggestionFeedback, setToast]);

  const handleCreateBusinessTerm = useCallback(async (termName: string, category: string, description: string): Promise<SemanticTerm | null> => {
    try {
  devDebug('[handleCreateBusinessTerm] Creating:', { termName, category, description });
      
      // Use relative URL to go through tenant scope
      const response = await fetch('/api/business-terms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          term_name: termName, // Use title case instead of uppercase
          properties: { 
            description,
            category
          }
        })
      });
      
      if (!response.ok) {
        const errorText = await response.text();
          devError('[handleCreateBusinessTerm] Failed:', response.status, errorText);
        setToast({ type: 'error', message: `Failed to create business term: ${errorText}` });
        return null;
      }
      
      const newTerm = await response.json();
  devDebug('[handleCreateBusinessTerm] Created successfully:', newTerm);
      
      // Add to the business terms list
      setBusinessTerms(prev => [...prev, {
        node_id: newTerm.node_id,
        term_name: newTerm.term_name,
        data_type: 'business_term',
        qualified_path: newTerm.qualified_path || `/business_terms/${newTerm.term_name}`
      }]);
      
      setToast({ type: 'success', message: `Created business term: ${termName}` });
      
      return {
        node_id: newTerm.node_id,
        term_name: newTerm.term_name,
        data_type: 'business_term',
        qualified_path: newTerm.qualified_path || `/business_terms/${newTerm.term_name}`
      };
    } catch (error) {
  devError('[handleCreateBusinessTerm] Error:', error);
      setToast({ type: 'error', message: `Failed to create business term: ${error instanceof Error ? error.message : 'Unknown error'}` });
      return null;
    }
  }, [setToast, setBusinessTerms]);

  const filteredMappings = useMemo(() => {
    const mappingsList = Object.values(businessTermMappings);
    
    return mappingsList.filter(mapping => {
      // Filter by search term
      if (searchTerm) {
        const searchLower = searchTerm.toLowerCase();
        const matchesTerm = mapping.semantic_term.term_name.toLowerCase().includes(searchLower);
        const matchesBusinessTerm = mapping.selected_business_term?.term_name.toLowerCase().includes(searchLower) || false;
        const matchesDefinition = mapping.semantic_term.description?.toLowerCase().includes(searchLower) || false;
        
        if (!matchesTerm && !matchesBusinessTerm && !matchesDefinition) return false;
      }
      
      // Filter by mapping status
      if (filterStatus === 'mapped' && !mapping.edge_exists) return false;
      if (filterStatus === 'unmapped' && mapping.edge_exists) return false;
      
      return true;
    });
  }, [businessTermMappings, searchTerm, filterStatus]);

  const mappingCounts = useMemo(() => {
    const mappingsList = Object.values(businessTermMappings);
    return {
      total: mappingsList.length,
      mapped: mappingsList.filter(m => m.edge_exists).length,
      unmapped: mappingsList.filter(m => !m.edge_exists).length
    };
  }, [businessTermMappings]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Business Term Mapper
      </Typography>

      {/* Statistics - Horizontal Layout */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
        <Paper 
          sx={{ 
            p: 2, 
            flex: '1 1 150px',
            minWidth: '150px',
            textAlign: 'center', 
            cursor: 'pointer',
            '&:hover': { bgcolor: 'action.hover' },
            transition: 'all 0.2s'
          }}
          onClick={() => setFilterStatus('all')}
        >
          <Typography variant="h5" color="primary" sx={{ fontWeight: 700 }}>{mappingCounts.total}</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>Total Semantic Terms</Typography>
        </Paper>
        <Paper 
          sx={{ 
            p: 2, 
            flex: '1 1 150px',
            minWidth: '150px',
            textAlign: 'center', 
            cursor: 'pointer',
            '&:hover': { bgcolor: 'action.hover' },
            transition: 'all 0.2s'
          }}
          onClick={() => setFilterStatus('mapped')}
        >
          <Typography variant="h5" color="success.main" sx={{ fontWeight: 700 }}>{mappingCounts.mapped}</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>Mapped to Business Terms</Typography>
        </Paper>
        <Paper 
          sx={{ 
            p: 2, 
            flex: '1 1 150px',
            minWidth: '150px',
            textAlign: 'center', 
            cursor: 'pointer',
            '&:hover': { bgcolor: 'action.hover' },
            transition: 'all 0.2s'
          }}
          onClick={() => setFilterStatus('unmapped')}
        >
          <Typography variant="h5" color="warning.main" sx={{ fontWeight: 700 }}>{mappingCounts.unmapped}</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>Unmapped</Typography>
        </Paper>
      </Box>

      {/* Filters and Search */}
      <Card sx={{ p: 2, mb: 3 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel>Filter Status</InputLabel>
              <Select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value as 'all' | 'mapped' | 'unmapped')}
                label="Filter Status"
              >
                <MenuItem value="all">All</MenuItem>
                <MenuItem value="mapped">Mapped</MenuItem>
                <MenuItem value="unmapped">Unmapped</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={3}>
            <Button
              fullWidth
              variant="contained"
              color="primary"
              startIcon={generatingSuggestions ? <CircularProgress size={20} /> : <AutoFixHigh />}
              onClick={handleGenerateAllSuggestions}
              disabled={generatingSuggestions || mappingCounts.unmapped === 0}
              sx={{ height: '56px' }}
            >
              {generatingSuggestions ? 'Generating...' : 'Generate Suggestions'}
            </Button>
          </Grid>
        </Grid>
      </Card>

      {/* Enhanced Mapping Interface */}
      <Card sx={{ p: 2 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Enhanced Business Term Mapping ({filteredMappings.length})
        </Typography>
        
        <Box sx={{ mb: 2, display: 'flex', gap: 1, alignItems: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            Click on any row to expand and see mapping options. Use inline suggestions or create custom business terms.
          </Typography>
        </Box>
        
        <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
          {filteredMappings.map((mapping, index) => (
            <EnhancedMappingRow
              key={mapping.semantic_term.node_id}
              mapping={mapping}
              businessTerms={businessTerms}
              onSelectBusinessTerm={handleSelectBusinessTerm}
              onSave={handleSave}
              onUpdateBusinessTerm={updateBusinessTerm}
              onCreateBusinessTerm={handleCreateBusinessTerm}
              onAcceptSuggestion={handleAcceptSuggestion}
              onRejectSuggestion={handleRejectSuggestion}
              suggestions={allSuggestions[mapping.semantic_term.node_id || ''] || []}
              isLast={index === filteredMappings.length - 1}
            />
          ))}
        </Box>
      </Card>
    </Box>
  );
}