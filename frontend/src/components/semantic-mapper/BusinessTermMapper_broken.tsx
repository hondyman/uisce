import { useState, useEffect, useMemo, useCallback } from 'react';
import { devWarn, devError } from '../../utils/devLogger';
import { 
  Box, Typography, Card, Autocomplete, TextField, Button, Chip, Alert as _Alert, 
  List, ListItem, ListItemText, Divider, Grid, Paper, CircularProgress,
  FormControl, InputLabel, Select, MenuItem, Checkbox, FormControlLabel as _FormControlLabel,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow
} from '@mui/material';
import { useSemanticMapper } from './useSemanticMapper';
import type { SemanticTerm, Mapping as _Mapping } from './types';

interface BusinessTermSuggestion {
  business_term_id: string;
  business_term_name: string;
  description: string;
  categories: string[];
  confidence: number;
  semantic_term_id: string;
  semantic_term_name: string;
  database_column: string;
}

interface BusinessTermMapping {
  semantic_term: SemanticTerm;
  selected_business_term: SemanticTerm | null;
  suggestions: BusinessTermSuggestion[];
  override: boolean;
  edge_exists: boolean;
}

export function BusinessTermMapper() {
  const {
    mappings: _mappings,
    loadSemanticTerms,
    loadBusinessTerms,
    toast: _toast,
    setToast,
  } = useSemanticMapper();

  const [semanticTerms, setSemanticTerms] = useState<SemanticTerm[]>([]);
  const [businessTerms, setBusinessTerms] = useState<SemanticTerm[]>([]);
  const [businessTermMappings, setBusinessTermMappings] = useState<Record<string, BusinessTermMapping>>({});
  const [suggestions, setSuggestions] = useState<BusinessTermSuggestion[]>([]);
  const [loadingSuggestions, setLoadingSuggestions] = useState(false);
  const [selectedSuggestions, setSelectedSuggestions] = useState<Set<string>>(new Set());
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'mapped' | 'unmapped'>('all');
  const [loading, setLoading] = useState(true);

  // Load initial data
  useEffect(() => {
    const initializeData = async () => {
      try {
        setLoading(true);
        const [semanticData, businessData] = await Promise.all([
          loadSemanticTerms(),
          loadBusinessTerms()
        ]);
        setSemanticTerms(semanticData);
        setBusinessTerms(businessData);
        
        // Initialize mappings for semantic terms that have database columns
        const initialMappings: Record<string, BusinessTermMapping> = {};
        semanticData.forEach(term => {
          if (term.node_id) {
            initialMappings[term.node_id] = {
              semantic_term: term,
              selected_business_term: null,
              suggestions: [],
              override: false,
              edge_exists: false
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
  }, []); // Empty dependency array to prevent infinite loop

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
          edge_type_id: '3be9d6ae-1598-4628-a3dd-b606921a9193',
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
    
    if (businessTerm?.node_id) {
      try {
        await createBusinessTermEdgeWithCorrectType(semanticTermId, businessTerm.node_id);
        
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
        setToast({ type: 'error', message: 'Failed to create business term mapping' });
      }
    }
  }, [businessTermMappings, setToast]);

  const generateSuggestions = async () => {
    if (semanticTerms.length === 0) {
      setToast({ type: 'error', message: 'No semantic terms available to generate suggestions' });
      return;
    }

    setLoadingSuggestions(true);
    try {
      // Get mapped database columns from semantic terms
      const semanticTermsWithColumns = semanticTerms.filter(st => st.node_id);
      
      const suggestions: BusinessTermSuggestion[] = [];
      
      for (const semanticTerm of semanticTermsWithColumns) {
        try {
          const response = await fetch(`/api/semantic-terms/${semanticTerm.node_id}/suggest-business-terms`, {
            method: 'GET',
            credentials: 'include',
          });

          if (response.ok) {
            const termSuggestions = await response.json();
            suggestions.push(...termSuggestions.map((s: any) => ({
              business_term_id: s.business_term_id || s.node_id,
              business_term_name: s.business_term_name || s.term_name,
              description: s.description || 'No description available',
              categories: s.categories || [],
              confidence: s.confidence || 0.5,
              semantic_term_id: semanticTerm.node_id,
              semantic_term_name: semanticTerm.term_name,
              database_column: semanticTerm.term_name // Use term name as placeholder
            })));
          }
          } catch (error) {
          devWarn(`Failed to get suggestions for ${semanticTerm.term_name}:`, error);
        }
      }

      setSuggestions(suggestions);
      setToast({ type: 'success', message: `Generated ${suggestions.length} business term suggestions` });
    } catch (error) {
      devError('Error generating suggestions:', error);
      setToast({ type: 'error', message: 'Failed to generate business term suggestions' });
    } finally {
      setLoadingSuggestions(false);
    }
  };



  const toggleSuggestionSelection = (suggestionId: string) => {
    setSelectedSuggestions(prev => {
      const newSet = new Set(prev);
      if (newSet.has(suggestionId)) {
        newSet.delete(suggestionId);
      } else {
        newSet.add(suggestionId);
      }
      return newSet;
    });
  };

  const applySelectedSuggestions = async () => {
    const selectedSuggestionsList = suggestions.filter((_, index) => 
      selectedSuggestions.has(`suggestion-${index}`)
    );

    for (const suggestion of selectedSuggestionsList) {
      try {
        await createBusinessTermEdgeWithCorrectType(
          suggestion.semantic_term_id,
          suggestion.business_term_id
        );
        
        // Update mapping to reflect edge creation
        setBusinessTermMappings(prev => ({
          ...prev,
          [suggestion.semantic_term_id]: {
            ...prev[suggestion.semantic_term_id],
            edge_exists: true,
            selected_business_term: businessTerms.find(bt => bt.node_id === suggestion.business_term_id) || null
          }
        }));
      } catch (error) {
        devError(`Failed to apply suggestion for ${suggestion.semantic_term_name}:`, error);
      }
    }

    setToast({ type: 'success', message: `Applied ${selectedSuggestionsList.length} business term mappings` });
    setSelectedSuggestions(new Set());
  };

  const filteredMappings = useMemo(() => {
    const mappingsList = Object.values(businessTermMappings);
    
    return mappingsList.filter(mapping => {
      // Filter by search term
      if (searchTerm) {
        const searchLower = searchTerm.toLowerCase();
        const matchesTerm = mapping.semantic_term.term_name.toLowerCase().includes(searchLower);
        const matchesBusinessTerm = mapping.selected_business_term?.term_name.toLowerCase().includes(searchLower);
        if (!matchesTerm && !matchesBusinessTerm) return false;
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

      {/* Statistics */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={4}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="primary">{mappingCounts.total}</Typography>
            <Typography variant="body2" color="text.secondary">Total Semantic Terms</Typography>
          </Paper>
        </Grid>
        <Grid item xs={4}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="success.main">{mappingCounts.mapped}</Typography>
            <Typography variant="body2" color="text.secondary">Mapped to Business Terms</Typography>
          </Paper>
        </Grid>
        <Grid item xs={4}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h4" color="warning.main">{mappingCounts.unmapped}</Typography>
            <Typography variant="body2" color="text.secondary">Unmapped</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Generate Suggestions Section */}
      <Card sx={{ p: 2, mb: 3 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          AI-Powered Business Term Suggestions
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Generate intelligent business term suggestions based on your semantic terms and database columns using matching algorithms.
        </Typography>
        <Button
          variant="contained"
          onClick={generateSuggestions}
          disabled={loadingSuggestions || semanticTerms.length === 0}
          sx={{ mb: 2 }}
        >
          {loadingSuggestions ? 'Generating...' : 'Generate AI Suggestions'}
        </Button>

        {suggestions.length > 0 && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle1" sx={{ mb: 1 }}>
              Suggested Business Terms ({suggestions.length})
            </Typography>
            <List sx={{ maxHeight: 300, overflow: 'auto', border: 1, borderColor: 'divider', borderRadius: 1 }}>
              {suggestions.map((suggestion, index) => {
                const suggestionId = `suggestion-${index}`;
                const isSelected = selectedSuggestions.has(suggestionId);
                return (
                  <Box key={suggestionId}>
                    <ListItem
                      sx={{
                        cursor: 'pointer',
                        bgcolor: isSelected ? 'action.selected' : 'inherit',
                        '&:hover': { bgcolor: 'action.hover' }
                      }}
                      onClick={() => toggleSuggestionSelection(suggestionId)}
                    >
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Checkbox checked={isSelected} />
                            <Typography variant="subtitle2">{suggestion.business_term_name}</Typography>
                            <Chip
                              label={`${(suggestion.confidence * 100).toFixed(0)}%`}
                              size="small"
                              color={suggestion.confidence > 0.8 ? 'success' : suggestion.confidence > 0.6 ? 'warning' : 'default'}
                            />
                          </Box>
                        }
                        secondary={
                          <Box>
                            <Typography variant="body2" color="text.secondary">
                              {suggestion.description}
                            </Typography>
                            <Box sx={{ mt: 1, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                              {suggestion.categories.map((category, catIndex) => (
                                <Chip
                                  key={catIndex}
                                  label={category}
                                  size="small"
                                  variant="outlined"
                                />
                              ))}
                            </Box>
                            <Typography variant="caption" color="text.secondary">
                              For: {suggestion.semantic_term_name} ({suggestion.database_column})
                            </Typography>
                          </Box>
                        }
                      />
                    </ListItem>
                    {index < suggestions.length - 1 && <Divider />}
                  </Box>
                );
              })}
            </List>
            {selectedSuggestions.size > 0 && (
              <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={applySelectedSuggestions}
                >
                  Apply Selected ({selectedSuggestions.size})
                </Button>
                <Button
                  variant="outlined"
                  onClick={() => setSelectedSuggestions(new Set())}
                >
                  Clear Selection
                </Button>
              </Box>
            )}
          </Box>
        )}
      </Card>

      {/* Filters and Search */}
      <Card sx={{ p: 2, mb: 3 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={6}>
            <TextField
              fullWidth
              label="Search semantic terms or business terms"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              variant="outlined"
            />
          </Grid>
          <Grid item xs={3}>
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
        </Grid>
      </Card>

      {/* Manual Mapping Table */}
      <Card sx={{ p: 2 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Manual Business Term Mapping ({filteredMappings.length})
        </Typography>
        <TableContainer sx={{ maxHeight: 600, overflow: 'auto' }}>
          <Table stickyHeader>
            <TableHead>
              <TableRow>
                <TableCell>Semantic Term</TableCell>
                <TableCell>Business Term</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredMappings.map(mapping => {
                const semanticTermId = mapping.semantic_term.node_id as string;
                return (
                  <TableRow key={semanticTermId}>
                    <TableCell>
                      <Typography variant="subtitle2">{mapping.semantic_term.term_name}</Typography>
                    </TableCell>
                    <TableCell>
                      <Autocomplete
                        sx={{ minWidth: 300 }}
                        options={businessTerms}
                        getOptionLabel={(option) => option.term_name}
                        value={mapping.selected_business_term}
                        onChange={(_, value) => handleSelectBusinessTerm(semanticTermId, value)}
                        renderInput={(params) => (
                          <TextField
                            {...params}
                            label="Select Business Term"
                            variant="outlined"
                            size="small"
                          />
                        )}
                        renderOption={(props, option) => (
                          <li {...props}>
                            <Box>
                              <Typography variant="subtitle2">{option.term_name}</Typography>
                              <Typography variant="caption" color="text.secondary">
                                {option.data_type || 'No description'}
                              </Typography>
                            </Box>
                          </li>
                        )}
                      />
                    </TableCell>
                    <TableCell>
                      {mapping.edge_exists ? (
                        <Chip label="Mapped" color="success" size="small" />
                      ) : mapping.selected_business_term ? (
                        <Chip label="Ready" color="warning" size="small" />
                      ) : (
                        <Chip label="Unmapped" color="default" size="small" />
                      )}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="contained"
                        size="small"
                        onClick={() => handleSave(semanticTermId)}
                        disabled={!mapping.selected_business_term || mapping.edge_exists}
                      >
                        {mapping.edge_exists ? 'Mapped' : 'Save Mapping'}
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>
    </Box>
  );
}
