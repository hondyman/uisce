import { useState, useEffect, useMemo, useCallback, useRef, Fragment } from 'react';
import { RefreshCw, Database, ScanSearch, BarChart3, Tag, Briefcase } from 'lucide-react';
import { Box, Typography, Alert, Card, Button, Tabs, Tab, Dialog } from '@mui/material';
import { useLineageData } from '../services/lineageService';
import { useScope } from '../contexts/ScopeContext';
import { useTenant } from '../contexts/TenantContext';
import { hasTenantScope } from '../utils/tenantScope';
import './SemanticMapper.css';
import { useSemanticMapper } from './semantic-mapper/useSemanticMapper';
import type { Mapping, SemanticTerm } from './semantic-mapper/types';
import { getMappingUniqueId } from '../utils/mappingId';
import { SemanticMapperHeader } from './semantic-mapper/SemanticMapperHeader';
import { MappingList } from './semantic-mapper/MappingList';
import { ConfirmationDialogs } from './semantic-mapper/ConfirmationDialogs';
import { LineageModal } from './semantic-mapper/LineageModal';
import { BusinessTermMapper } from './semantic-mapper/BusinessTermMapper';
import DbScanner from './DbScanner';
import ProfilerPage from './ProfilerPage';
import DatabaseTreePanel from './DatabaseTreePanel';
import { SemanticMappingWizard } from './SemanticMappingWizard';
import { ProfessionalSearchInput, SearchSuggestion } from './common/ProfessionalSearchInput';
import { Wand2 } from 'lucide-react';
import { devDebug, devError } from '../utils/devLogger';


export default function SemanticMapper({ onOpenWizard }: { onOpenWizard?: () => void }) {
  const {
    mappings, setMappings, loading, toast, setToast,
    loadMappings, searchSemanticTerms, createNewSemanticTerm,
    createEdges, replaceMapping, persistIgnores
  } = useSemanticMapper();

  const getUniqueId = (m: Mapping) => getMappingUniqueId(m);

  const [globalSearchTerm, setGlobalSearchTerm] = useState('');
  const [savedRows, setSavedRows] = useState<Set<string>>(new Set());
  const [selectedMappings, setSelectedMappings] = useState<Set<string>>(new Set());
  const [sortBy, setSortBy] = useState<'confidence' | 'name' | 'none'>('confidence');
  const [mappedFilter, setMappedFilter] = useState<Set<'all' | 'mapped' | 'unmapped' | 'selected' | 'highConfidence' | 'pending'>>(new Set(['all']));
  const [currentMatchIndex, setCurrentMatchIndex] = useState<number>(0);
  const [suggestions, setSuggestions] = useState<any[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const updateMappings = useCallback((updater: (prev: Mapping[]) => Mapping[]) => {
    setMappings((prev: Mapping[]) => {
      const next = updater(prev);
      setSelectedMappings(new Set(next.filter((m: Mapping) => m.selected).map((m: Mapping) => getUniqueId(m))));
      return next;
    });
  }, [setMappings, setSelectedMappings, getUniqueId]);

  // Debounced search for semantic terms with useRef to avoid recreating
  const searchTimeoutRef = useRef<NodeJS.Timeout>();
  
  const debouncedSearch = useCallback((query: string) => {
    // Clear previous timeout
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }
    
    if (query.trim().length < 2) {
      setSuggestions([]);
      setShowSuggestions(false);
      return;
    }
    
    searchTimeoutRef.current = setTimeout(async () => {
      try {
        const results = await searchSemanticTerms(query);
        const formattedSuggestions = results.map((term: any) => ({
          id: term.node_id || term.term_name,
          title: term.term_name,
          subtitle: term.description || 'Semantic Term',
          type: 'semantic-term'
        }));
        setSuggestions(formattedSuggestions);
        setShowSuggestions(true);
        setHighlightedIndex(-1);
      } catch (error) {
        devError('Error searching semantic terms:', error);
        setSuggestions([]);
        setShowSuggestions(false);
      }
    }, 300);
  }, [searchSemanticTerms]);

  const handleSuggestionSelect = (suggestion: any) => {
    // When a semantic term is selected, set it as the search term
    setGlobalSearchTerm(suggestion.title);
    setShowSuggestions(false);
    setCurrentMatchIndex(0);
  };

  const handleSearchFocus = () => {
    if (suggestions.length > 0 && globalSearchTerm.trim().length >= 2) {
      setShowSuggestions(true);
    }
  };

  const handleSearchBlur = () => {
    // Delay hiding suggestions to allow clicks
    setTimeout(() => setShowSuggestions(false), 200);
  };

  const handleNavigateMatch = (direction: 1 | -1) => {
    const totalMatches = filteredMappings.length;
    if (totalMatches === 0) return;
    
    const newIndex = direction === 1 
      ? Math.min(currentMatchIndex + 1, totalMatches - 1)
      : Math.max(currentMatchIndex - 1, 0);
    
    setCurrentMatchIndex(newIndex);
  };

  const handleSearchChange = useCallback((newTerm: string) => {
    setGlobalSearchTerm(newTerm);
    setCurrentMatchIndex(0);
    debouncedSearch(newTerm);
  }, [debouncedSearch]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, []);
  const [compactRows] = useState<boolean>(() => {
    try { const v = localStorage.getItem('semantic_mapper_compact_rows'); return v === null ? true : v === '1' || v === 'true'; } catch (e) { return true }
  });

  const [wizardOpen, setWizardOpen] = useState(false);
  const [lineageModalOpen, setLineageModalOpen] = useState(false);
  const [lineageSelectedAsset, setLineageSelectedAsset] = useState<any | null>(null);
  const [lineageDatasourceId, setLineageDatasourceId] = useState<string | null>(null);
  const lineageData = useLineageData(lineageDatasourceId || '');

  const TabPanel = (props: { children?: React.ReactNode; index: number; value: number }) => {
    const { children, value, index, ...other } = props;
    return (
      <div
        role="tabpanel"
        hidden={value !== index}
        id={`semantic-mapper-tabpanel-${index}`}
        aria-labelledby={`semantic-mapper-tab-${index}`}
        {...other}
      >
        {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
      </div>
    );
  };

  const [activeTab, setActiveTab] = useState(0);
  // profile preselection state (set by DbScanner)
  const [profileSchema, setProfileSchema] = useState<string | null>(null);
  const [profileTable, setProfileTable] = useState<string | null>(null);
  const [profileTables, setProfileTables] = useState<string[] | null>(null);
  const scope = useScope();
  const { tenant, datasource } = useTenant();
  const runScanRef = (window as any).runScanRef || { current: null };

  const toggleMapping = (mappingId: string) => {
    updateMappings((prev: Mapping[]) => prev.map((m: Mapping) => {
      if (getUniqueId(m) === mappingId) {
        return { ...m, selected: !m.selected };
      }
      return m;
    }));
  };

  const confirmEditing = async (id: string, _typedTerm?: string) => {
    // This is no longer needed - we handle everything through selectSemanticTerm and handleCreateAndSelectTerm
    setSavedRows(s => { const n = new Set(s); n.add(id); setTimeout(() => { setSavedRows(prev => { const r = new Set(prev); r.delete(id); return r; }); }, 3000); return n; });
  };

  const setOverride = (id: string, val: boolean) => {
    updateMappings((prev: Mapping[]) => prev.map((m: Mapping) => {
      if (getUniqueId(m) === id) {
        return { ...m, override: val, selected: val ? true : m.selected };
      }
      return m;
    }));
  };

  const setIgnored = (id: string, val: boolean) => {
    updateMappings((prev: Mapping[]) => prev.map((m: Mapping) => {
      if (getUniqueId(m) === id) {
        return { ...m, ignored: val, selected: val ? false : m.selected };
      }
      return m;
    }));
  };

  const filteredMappings = useMemo(() => {
    const searchTerm = globalSearchTerm.toLowerCase().trim();
    
    return mappings.filter((m: Mapping) => {
      // Skip ignored mappings
      if (m.ignored) return false;
      
      // Apply filters
      if (!mappedFilter.has('all')) {
        // If we have specific filters, check them
        // Note: These can be additive or exclusive. 
        // Logic: if any single positive filter is active, we check if it matches.
        // If multiple are active, usually we interpret as OR or AND. 
        // Based on typical UI, if 'Pending' and 'High Confidence' are clicked, it might mean "Pending AND High Conf" or "Pending OR High Conf".
        // Given the simplistic set, let's assume if specific filters are on, at least one must match (OR logic) or all must match (AND).
        // Let's go with behavior:
        // If 'mapped' is on -> must be mapped
        // If 'unmapped' is on -> must be unmapped
        // If 'pending' is on -> must be pending
        
        // But commonly, "Mapped" vs "Unmapped" are mutually exclusive.
        // "High Confidence" is a subset.
        
        // Revised Logic:
        // If 'mapped' is active, include mapped.
        // If 'unmapped' is active, include unmapped.
        // If 'pending' is active, include pending.
        // If 'highConfidence' is active, include high confidence.
        // If 'selected' is active, include selected.
        
        // We act as an OR filter if multiple are selected? 
        // Or is it a replacement? "Clicking High Confidence shows ONLY High Confidence".
        // The user said "make these aggregates filters". Usually clicking a stat card filters to THAT category.
        // So likely we treat the Set as having usually one active filter, or we allow multiple.
        // Implementation: Check if ANY active filter matches.
        
        const activeFilters = Array.from(mappedFilter);
        const matchesAny = activeFilters.some(f => {
          if (f === 'mapped') return m.edge_exists;
          if (f === 'unmapped') return !m.edge_exists;
          if (f === 'pending') return m.is_pending;
          if (f === 'highConfidence') return m.confidence >= 0.75;
          if (f === 'selected') return m.selected;
          return false;
        });
        
        if (!matchesAny) return false;
      }
      
      // Apply scope filters (sidebar selection)
      if (scope.schemaNames.length > 0 && !scope.schemaNames.includes(m.database_column.schema || '')) return false;
      if (scope.tableNames.length > 0 && !scope.tableNames.includes(m.database_column.table || '')) return false;
      if (scope.columnNames.length > 0 && !scope.columnNames.includes(m.database_column.column || '')) return false;
      
      // Apply search term filter
      if (searchTerm) {
        const col = m.database_column;
        const semanticTerm = m.semantic_term || '';
        const matchReason = m.match_reason || '';
        
        const searchableText = [
          col.schema || '',
          col.table || '',
          col.column || '',
          semanticTerm,
          matchReason,
          col.data_type || ''
        ].join(' ').toLowerCase();
        
        // Check if search term matches any part of the searchable text
        const matches = searchTerm.split(' ').every(term => 
          term.length === 0 || searchableText.includes(term)
        );
        
        if (!matches) return false;
      }
      
      return true;
    }).sort((a: any, b: any) => {
      // Pending items float to top if sorting by confidence
      if (a.is_pending && !b.is_pending) return -1;
      
      const aT = a as Mapping; const bT = b as Mapping;
      if (sortBy === 'confidence') return bT.confidence - aT.confidence;
      if (sortBy === 'name') return aT.database_column.column.localeCompare(bT.database_column.column);
      return 0;
    });
  }, [mappings, globalSearchTerm, mappedFilter, scope, sortBy]);

  const mappingCounts = useMemo(() => {
    const counts = { all: 0, mapped: 0, unmapped: 0, pending: 0 };
    mappings.forEach((m: Mapping) => {
      if (m.ignored) return;
      counts.all++;
      if (m.edge_exists) {
        counts.mapped++;
      } else {
        counts.unmapped++;
      }
      if (m.is_pending) counts.pending++;
    });
    return counts;
  }, [mappings]);

  const highConfidenceCount = useMemo(() => filteredMappings.filter(m => m.confidence >= 0.75).length, [filteredMappings]);
  
  const averageConfidence = useMemo(() => {
    if (filteredMappings.length === 0) return 0;
    return filteredMappings.reduce((s, m) => s + (m.confidence || 0), 0) / filteredMappings.length;
  }, [filteredMappings]);

  const sortedMappings = filteredMappings; // Renaming for clarity as we already sorted in filteredMappings


  const [confirmOpen, setConfirmOpen] = useState(false);
  const [replaceConfirmOpen, setReplaceConfirmOpen] = useState(false);
  const [replaceIndex, setReplaceIndex] = useState<number | null>(null);
  const openConfirm = () => { if (selectedMappings.size > 0) setConfirmOpen(true); };
  const closeConfirm = () => setConfirmOpen(false);
  const confirmCreate = async () => { 
    closeConfirm(); 
    const selected = mappings.filter((m: any) => selectedMappings.has(getUniqueId(m)) && !m.ignored && !m.edge_exists);
    devDebug('[SemanticMapper] Creating edges for mappings:', selected.map(m => ({
      column: m.database_column.column,
      semantic_term: m.semantic_term,
      semantic_term_id: m.semantic_term_id,
      is_new_term: m.is_new_term,
      override: m.override,
      edge_exists: m.edge_exists,
      has_tenant_id: !!m.database_column.tenant_id,
      has_tenant_instance_id: !!m.database_column.tenant_tenant_instance_id,
      full_db_column: m.database_column
    })));
    
    if (selected.length === 0) {
      setToast({ type: 'error', message: 'No mappings selected for edge creation. Make sure mappings are checked and have semantic terms assigned.' });
      return;
    }
    
    await createEdges(selected); 
    setSelectedMappings(new Set()); 
  };
  const openReplaceConfirm = (idx: number) => { setReplaceIndex(idx); setReplaceConfirmOpen(true); };
  const closeReplaceConfirm = () => { setReplaceIndex(null); setReplaceConfirmOpen(false); };
  const confirmReplace = async () => { if (replaceIndex !== null) await replaceMapping(mappings[replaceIndex]); closeReplaceConfirm(); };

  useEffect(() => { try { localStorage.setItem('semantic_mapper_compact_rows', compactRows ? '1' : '0'); } catch (e) {} }, [compactRows]);

  return (
    <Fragment>
    <Box sx={{ minHeight: '100vh', bgcolor: 'grey.50', p: 3 }}>
        <Box sx={{ maxWidth: '1600px', margin: '0 auto' }}>
        <Box className="semantic-topbar">
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', maxWidth: '1600px', margin: '0 auto', px: 2 }}>
            <Typography className="title">Semantic Mapper</Typography>
            {activeTab === 0 && (
              <Button variant="contained" onClick={() => { if (runScanRef.current) runScanRef.current(); }} data-testid="top-scan-button">Scan</Button>
            )}
          </Box>
        </Box>
        {!hasTenantScope() && (
          <Alert severity="warning" sx={{ mb: 3 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>Tenant selection required</Typography>
            <Typography variant="body2" sx={{ mt: 0.5 }}>Please select a tenant and datasource using the selector in the navigation to enable catalog filtering and searches. Without tenant scope, API requests for schemas, tables, and columns will be blocked.</Typography>
          </Alert>
        )}

        {mappings.filter((m: any) => m.override && m.selected && m.semantic_term_id && !m.edge_exists).length > 0 && (
          <Alert severity="success" sx={{ mb: 3 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>✅ Override Ready to Apply</Typography>
            <Typography variant="body2" sx={{ mt: 0.5 }}>
              {mappings.filter((m: any) => m.override && m.selected && m.semantic_term_id && !m.edge_exists).length} override mapping{mappings.filter((m: any) => m.override && m.selected && m.semantic_term_id && !m.edge_exists).length > 1 ? 's are' : ' is'} selected and ready.
              Click "Create Edges" to persist {mappings.filter((m: any) => m.override && m.selected && m.semantic_term_id && !m.edge_exists).length > 1 ? 'them' : 'it'} to the knowledge graph.
            </Typography>
          </Alert>
        )}

        {mappings.filter((m: any) => m.override && !m.semantic_term_id).length > 0 && (
          <Alert severity="info" sx={{ mb: 3 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>ℹ️ Override in Progress</Typography>
            <Typography variant="body2" sx={{ mt: 0.5 }}>
              {mappings.filter((m: any) => m.override && !m.semantic_term_id).length} mapping{mappings.filter((m: any) => m.override && !m.semantic_term_id).length > 1 ? 's are' : ' is'} in override mode.
              Type a semantic term name or select from suggestions, then create or apply the term.
            </Typography>
          </Alert>
        )}

        {mappings.filter((m: any) => m.override).length > 0 && (
          <Alert severity="warning" sx={{ mb: 3 }}>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>Override Mode Active</Typography>
            <Typography variant="body2" sx={{ mt: 0.5 }}>
              {mappings.filter((m: any) => m.override).length} mapping{mappings.filter((m: any) => m.override).length > 1 ? 's' : ''} currently in override mode.
              These mappings will use custom semantic terms instead of automated suggestions. Review and confirm changes before creating edges.
            </Typography>
          </Alert>
        )}

      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        {/* Global Header */}
        <Box sx={{ 
          p: 2, 
          pb: 0,
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          borderBottom: 1, 
          borderColor: 'divider',
          bgcolor: 'background.paper',
          gap: 4
        }}>
          {/* Tabs */}
          <Tabs 
            value={activeTab} 
            onChange={(_, newValue) => setActiveTab(newValue)} 
            aria-label="semantic mapper tabs"
            sx={{
              '& .MuiTab-root': {
                textTransform: 'none',
                fontWeight: 600,
                fontSize: '0.95rem',
                minHeight: 48,
                px: 3,
                color: '#64748b',
                '&.Mui-selected': {
                  color: '#2563eb',
                },
              },
              '& .MuiTabs-indicator': {
                height: 3,
                borderRadius: '3px 3px 0 0',
                backgroundColor: '#2563eb',
              }
            }}
          >
            <Tab 
              label="Scan"
              icon={<ScanSearch width={18} height={18} />}
              iconPosition="start"
            />
            <Tab 
              label="Profile"
              icon={<BarChart3 width={18} height={18} />}
              iconPosition="start"
            />
            <Tab 
              label="Semantics"
              icon={<Tag width={18} height={18} />}
              iconPosition="start"
            />
            <Tab 
              label="Business"
              icon={<Briefcase width={18} height={18} />}
              iconPosition="start"
            />
          </Tabs>

          {/* Right Side: Search & Wizard */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1, justifyContent: 'flex-end', mb: 1 }}>
            <Box sx={{ width: '100%', maxWidth: 400 }}>
              <ProfessionalSearchInput
                value={globalSearchTerm}
                onChange={setGlobalSearchTerm}
                onClear={() => setGlobalSearchTerm('')}
                placeholder={
                  activeTab === 0 ? "Search schemas, tables, columns..." :
                  activeTab === 2 ? "Search mappings, columns, semantic terms..." :
                  activeTab === 3 ? "Search business terms..." :
                  "Search..."
                }
                mode="suggestions"
                suggestions={activeTab === 2 ? suggestions : []}
                showSuggestions={activeTab === 2 && showSuggestions}
                highlightedIndex={activeTab === 2 ? highlightedIndex : -1}
                onSuggestionSelect={handleSuggestionSelect}
                onHighlightChange={setHighlightedIndex}
                onFocus={handleSearchFocus}
                onBlur={handleSearchBlur}
                inputRef={searchInputRef}
                size="sm"
                variant="enhanced"
                navigationEnabled={activeTab === 2}
                currentMatch={currentMatchIndex}
                totalMatches={activeTab === 2 ? filteredMappings.length : undefined}
                onNavigateMatch={handleNavigateMatch}
              />
            </Box>

            {onOpenWizard && (
              <Button
                onClick={onOpenWizard}
                variant="contained"
                size="small"
                startIcon={<Wand2 width={16} height={16} />}
                sx={{
                  borderRadius: 2,
                  textTransform: 'none',
                  whiteSpace: 'nowrap',
                  fontWeight: 600,
                  height: 40,
                  background: 'linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%)',
                  boxShadow: '0 4px 12px rgba(139, 92, 246, 0.3)',
                  '&:hover': {
                    background: 'linear-gradient(135deg, #7c3aed 0%, #6d28d9 100%)',
                    boxShadow: '0 6px 16px rgba(139, 92, 246, 0.4)',
                  }
                }}
              >
                Open AI Wizard
              </Button>
            )}
          </Box>
        </Box>

        {/* Tab Content Panels */}
        <Box sx={{ flex: 1, overflow: 'hidden', p: 3 }}>
        <TabPanel value={activeTab} index={0}>
          <DbScanner
            refreshMappings={loadMappings}
            registerRunScan={(fn: any) => { runScanRef.current = fn; (window as any).runScanRef = runScanRef }}
            onProfile={(schemaName: string | null, tableName: string | null, tableNames?: string[]) => {
              setProfileSchema(schemaName);
              setProfileTable(tableName);
              setProfileTables(tableNames ?? null);
              setActiveTab(1); // switch to Profile tab
            }}
            searchTerm={globalSearchTerm}
          />
        </TabPanel>
        <TabPanel value={activeTab} index={1}>
          <ProfilerPage preselectedSchema={profileSchema} preselectedTable={profileTable} preselectedTables={profileTables} />
        </TabPanel>
        <TabPanel value={activeTab} index={2}>
          <SemanticMapperHeader
            loading={loading}
            loadMappings={loadMappings}
            filteredMappingsCount={filteredMappings.length}
            selectedMappingsCount={selectedMappings.size}
            highConfidenceCount={highConfidenceCount}
            pendingCount={mappingCounts.pending}
            averageScore={averageConfidence * 100}
            hasScopeFilter={scope.schemaNames.length > 0 || scope.tableNames.length > 0 || scope.columnNames.length > 0}
            sortBy={sortBy}
            setSortBy={setSortBy}
            openConfirm={openConfirm}
            mappedFilter={mappedFilter}
            setMappedFilter={setMappedFilter}
            mappingCounts={mappingCounts}
          />

          <Box sx={{ display: 'flex', gap: 3, alignItems: 'flex-start' }}>
            <DatabaseTreePanel
              title="Database Browser"
              subtitle="Matches by schema and table"
              showSchemaSelection={false}
              showTableSelection={false}
              showColumnSelection={false}
              showColumns={false}
              initialSelectedSchemas={scope.schemaNames}
              initialSelectedTables={scope.tableNames}
              initialSelectedColumns={scope.columnNames}
              mappedFilter={mappedFilter}
              setMappedFilter={setMappedFilter}
              mappingCounts={mappingCounts}
              mappings={mappings}
            />

            <Box sx={{ flex: 1, minHeight: '70vh', maxHeight: '80vh', overflow: 'auto' }}>
              {loading && mappings.length === 0 ? (
                <Card sx={{ p: 6, textAlign: 'center', borderRadius: 2 }} elevation={2}>
                  <RefreshCw className="animate-spin" width={48} height={48} style={{ margin: '0 auto 16px', color: '#1976d2' }} />
                  <Typography variant="h6" color="text.secondary">Loading mappings...</Typography>
                </Card>
              ) : (
                <MappingList
                  loading={loading}
                  mappings={filteredMappings}
                  savedRows={savedRows}
                  compactRows={compactRows}
                  keyboardExpanded={false}
                  setKeyboardExpanded={() => {}}
                  toggleMapping={toggleMapping}
                  confirmEditing={confirmEditing}
                  searchSemanticTerms={searchSemanticTerms}
                  selectSemanticTerm={(term: SemanticTerm, id: string) => {
                    updateMappings((prev: Mapping[]) => prev.map((m: Mapping) => {
                      if (getUniqueId(m) === id) {
                        // Preserve the entire database_column object with tenant info
                        return { 
                          ...m,
                          database_column: { ...m.database_column }, // Preserve tenant_id and tenant_tenant_instance_id
                          semantic_term: term.term_name, 
                          semantic_term_id: term.node_id, 
                          is_new_term: false, 
                          confidence: 1.0, 
                          override: true, 
                          selected: true,
                          edge_exists: false // Will be updated after creating edge
                        };
                      }
                      return m;
                    }));
                    setToast({ type: 'success', message: `Applied semantic term "${term.term_name}". Click "Create Edges" to persist the mapping.` });
                  }}
                  handleCreateAndSelectTerm={async (id: string, name: string) => { 
                    const t = await createNewSemanticTerm(name); 
                    if (t) { 
                      updateMappings((prev: Mapping[]) => prev.map((m: Mapping) => {
                        if (getUniqueId(m) === id) {
                          // Preserve the entire database_column object with tenant info
                          return { 
                            ...m,
                            database_column: { ...m.database_column }, // Preserve tenant_id and tenant_tenant_instance_id
                            semantic_term: t.term_name || name.toUpperCase(), 
                            semantic_term_id: t.node_id || '', 
                            is_new_term: true, 
                            confidence: 1.0, 
                            override: true, 
                            selected: true,
                            edge_exists: false // Mark as not yet persisted
                          };
                        }
                        return m;
                      })); 
                      setToast({ type: 'success', message: `Created semantic term "${t.term_name}". Click "Create Edges" to persist the mapping.` });
                    } 
                    return t; 
                  }}
                  setOverride={setOverride}
                  setIgnored={(id: string, val: boolean) => { setIgnored(id, val); if (val) { const mapping = mappings.find((m: any) => getUniqueId(m) === id); if (mapping) persistIgnores([mapping]); } }}
                  openReplaceConfirm={openReplaceConfirm}
                  openLineageModal={(mapping: Mapping) => { 
                    const asset = { 
                      type: 'table', 
                      id: `${mapping.database_column.schema || 'public'}.${mapping.database_column.table}`, 
                      name: `${mapping.database_column.schema || 'public'}.${mapping.database_column.table}`, 
                      qualifiedPath: `${mapping.database_column.schema || 'public'}.${mapping.database_column.table}`, 
                      isCore: false 
                    }; 
                    setLineageSelectedAsset(asset); 
                    setLineageDatasourceId(mapping.database_column?.tenant_tenant_instance_id || ''); 
                    setLineageModalOpen(true); 
                  }}
                />
              )}
            </Box>
          </Box>
        </TabPanel>
        <TabPanel value={activeTab} index={3}>
          <BusinessTermMapper />
        </TabPanel>
      </Box> {/* Close Content Panels */}
      </Box> {/* Close Global Wrapper */}
    </Box> {/* Close outer Box */}
    </Box> {/* Close maxWidth Box */}

    {toast && (
      <Box className={`toast ${toast?.type === 'success' ? 'toast-success' : 'toast-error'}`} role="status" aria-live="polite">
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography variant="body2" fontWeight={600}>{toast?.message}</Typography>
          <Button onClick={() => setToast(null)} size="small" className="toast-dismiss">Dismiss</Button>
        </Box>
      </Box>
    )}

    {/* Confirmation Dialogs */}
    <ConfirmationDialogs
      confirmOpen={confirmOpen}
      closeConfirm={closeConfirm}
      confirmCreate={confirmCreate}
      replaceConfirmOpen={replaceConfirmOpen}
      closeReplaceConfirm={closeReplaceConfirm}
      confirmReplace={confirmReplace}
      selectedCount={selectedMappings.size}
    />
    
    {/* Lineage Modal */}
    <LineageModal
      open={lineageModalOpen}
      onClose={() => setLineageModalOpen(false)}
      selectedAsset={lineageSelectedAsset}
      lineageData={lineageData}
      loading={false}
    />

    {/* AI Wizard */}
    <Dialog
      open={wizardOpen}
      onClose={() => setWizardOpen(false)}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: { height: '90vh', display: 'flex', flexDirection: 'column' }
      }}
    >
      <SemanticMappingWizard 
        tenantId={tenant?.id || ''} 
        datasourceId={datasource?.id || ''} 
        onClose={() => { 
            setWizardOpen(false); 
            loadMappings(); 
            if (activeTab !== 2) setActiveTab(2);
        }}
        onMappingsApplied={() => {
            // Refresh mappings to show newly created semantic terms in sidebar
            loadMappings();
        }}
      />
    </Dialog>
    </Fragment>
  );
}
