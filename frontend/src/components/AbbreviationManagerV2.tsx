import React, { useState, useMemo, useCallback } from 'react';
import {
  useAbbreviations,
  useAbbreviationExpansion,
  useSemanticTermValidation,
} from '../utils/abbreviationApi';
import ProfessionalSearchInput from './ProfessionalSearchInput';
import { devDebug, devError } from '../utils/devLogger';
import { useConfirm } from './ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { useTenant } from '../contexts/TenantContext';
import {
  Box,
  Button,
  Card,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Drawer,
  FormHelperText,
  Grid,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableSortLabel,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  Add as AddIcon,
  GridView as CardViewIcon,
  Delete as DeleteIcon,
  Download as DownloadIcon,
  Edit as EditIcon,
  TableChart as TableViewIcon,
} from '@mui/icons-material';
import './AbbreviationManager.css';

type SortField = 'abbreviation' | 'full_word';
type SortOrder = 'asc' | 'desc';
type ViewMode = 'table' | 'card';
type FilterType = 'all' | 'core' | 'custom';

interface AbbreviationManagerProps {
  className?: string;
  tenantId?: string;
}

export const AbbreviationManager: React.FC<AbbreviationManagerProps> = ({
  className = '',
  tenantId: propsTenantId,
}) => {
  const { tenant } = useTenant();
  const tenantId = propsTenantId || tenant?.id;

  const {
    abbreviations,
    loading,
    error,
    loaded,
    fetchAbbreviations,
    addAbbreviation,
    updateAbbreviation,
    deleteAbbreviation,
    totalCount,
    hasMore,
    loadMore,
    searchAbbreviations,
  } = useAbbreviations(tenantId);
  const { expandColumn } = useAbbreviationExpansion();
  const { validateTerms } = useSemanticTermValidation();

  // State management
  const [newAbbreviation, setNewAbbreviation] = useState('');
  const [newFullWord, setNewFullWord] = useState('');
  const [newNotes, setNewNotes] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [testColumn, setTestColumn] = useState('');
  const [testTerms, setTestTerms] = useState('');
  const [expansionResult, setExpansionResult] = useState<any>(null);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [activeTab, setActiveTab] = useState<'list' | 'add' | 'test' | 'validate'>('list');
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [filterType, setFilterType] = useState<FilterType>('all');
  const [sortField, setSortField] = useState<SortField>('abbreviation');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');
  const [sidebarOpen, setSidebarOpen] = useState(true);

  // Edit state
  const [editingAbbreviation, setEditingAbbreviation] = useState<any>(null);
  const [showEditModal, setShowEditModal] = useState(false);

  // Calculate facet counts
  const facetCounts = useMemo(() => {
    if (!abbreviations) return { core: 0, custom: 0, all: 0 };
    const core = abbreviations.filter((a: any) => a.is_core).length;
    const custom = abbreviations.filter((a: any) => !a.is_core).length;
    return { core, custom, all: abbreviations.length };
  }, [abbreviations]);

  // Filter and sort abbreviations
  const filteredAndSorted = useMemo(() => {
    let filtered = abbreviations || [];

    // Apply type filter
    if (filterType === 'core') {
      filtered = filtered.filter((a: any) => a.is_core);
    } else if (filterType === 'custom') {
      filtered = filtered.filter((a: any) => !a.is_core);
    }

    // Sort
    const sorted = [...filtered].sort((a: any, b: any) => {
      let aVal = a[sortField];
      let bVal = b[sortField];

      if (typeof aVal === 'string') aVal = aVal.toLowerCase();
      if (typeof bVal === 'string') bVal = bVal.toLowerCase();

      if (aVal < bVal) return sortOrder === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });

    return sorted;
  }, [abbreviations, filterType, sortField, sortOrder]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  };

  // Debounced search
  React.useEffect(() => {
    const timer = setTimeout(() => {
      searchAbbreviations(searchQuery);
    }, 400);
    return () => clearTimeout(timer);
  }, [searchQuery, searchAbbreviations]);

  const handleTabChange = useCallback(
    (tabId: 'list' | 'add' | 'test' | 'validate') => {
      setActiveTab(tabId);
      if (tabId === 'list' && !loaded) {
        fetchAbbreviations();
      }
    },
    [loaded, fetchAbbreviations]
  );

  React.useEffect(() => {
    if (activeTab === 'list' && !loaded) {
      fetchAbbreviations();
    }
  }, [activeTab, loaded, fetchAbbreviations]);

  React.useEffect(() => {
    if (!loaded) fetchAbbreviations();
  }, []);

  const searchData = useMemo(
    () =>
      (abbreviations || []).map((abbrev) => ({
        id: abbrev.id,
        text: abbrev.abbreviation,
        subtext: abbrev.full_word,
      })),
    [abbreviations]
  );

  const handleAddAbbreviation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newAbbreviation.trim() || !newFullWord.trim()) return;

    const success = await addAbbreviation(
      newAbbreviation.toUpperCase().trim(),
      newFullWord.toUpperCase().trim(),
      newNotes.trim()
    );

    if (success) {
      setNewAbbreviation('');
      setNewFullWord('');
      setNewNotes('');
      setActiveTab('list');
    }
  };

  const handleTestExpansion = async () => {
    if (!testColumn.trim()) return;
    try {
      const result = await expandColumn(testColumn.trim());
      setExpansionResult(result);
    } catch (err) {
      devError('Failed to test expansion:', err);
    }
  };

  const handleValidateTerms = async () => {
    if (!testTerms.trim()) return;
    const terms = testTerms.split(',').map((t) => t.trim()).filter((t) => t);
    if (terms.length === 0) return;
    try {
      const result = await validateTerms(terms);
      setValidationResult(result);
    } catch (err) {
      devError('Failed to validate terms:', err);
    }
  };

  const handleEditAbbreviation = (abbrev: any) => {
    setEditingAbbreviation(abbrev);
    setNewAbbreviation(abbrev.abbreviation);
    setNewFullWord(abbrev.full_word);
    setNewNotes(abbrev.notes || '');
    setShowEditModal(true);
  };

  const handleUpdateAbbreviation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingAbbreviation || !newAbbreviation.trim() || !newFullWord.trim()) return;
    const success = await updateAbbreviation(
      editingAbbreviation.id,
      newAbbreviation.trim(),
      newFullWord.trim(),
      newNotes.trim()
    );
    if (success) {
      setShowEditModal(false);
      setEditingAbbreviation(null);
      setNewAbbreviation('');
      setNewFullWord('');
      setNewNotes('');
      setActiveTab('list');
    }
  };

  const confirm = useConfirm();
  const notification = useNotification();

  const handleDeleteAbbreviation = async (id: number) => {
    if (!(await confirm({ title: 'Delete abbreviation', description: 'Are you sure you want to delete this abbreviation?' })))
      return;
    try {
      await deleteAbbreviation(id);
      notification.success('Abbreviation deleted');
    } catch (err) {
      notification.error('Failed to delete abbreviation');
    }
  };

  const handleCloseEditModal = () => {
    setShowEditModal(false);
    setEditingAbbreviation(null);
    setNewAbbreviation('');
    setNewFullWord('');
    setNewNotes('');
  };

  if (error) {
    return (
      <Box sx={{ p: 4, bgcolor: 'error.50', border: '1px solid', borderColor: 'error.200', borderRadius: 1 }}>
        <Typography color="error">Error: {error}</Typography>
      </Box>
    );
  }

  // Render table view
  const renderTableView = () => (
    <Box sx={{ overflowX: 'auto' }}>
      <Table size="small">
        <TableHead sx={{ backgroundColor: '#fafafa' }}>
          <TableRow>
            <TableCell>
              <TableSortLabel
                active={sortField === 'abbreviation'}
                direction={sortField === 'abbreviation' ? sortOrder : 'asc'}
                onClick={() => handleSort('abbreviation')}
              >
                Abbreviation
              </TableSortLabel>
            </TableCell>
            <TableCell>
              <TableSortLabel active={sortField === 'full_word'} direction={sortField === 'full_word' ? sortOrder : 'asc'} onClick={() => handleSort('full_word')}>
                Full Word
              </TableSortLabel>
            </TableCell>
            <TableCell width={100}>Type</TableCell>
            <TableCell>Notes</TableCell>
            <TableCell width={200} align="right">
              Actions
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredAndSorted.map((abbrev: any) => (
            <TableRow key={abbrev.id} hover>
              <TableCell sx={{ fontWeight: 600, fontFamily: 'monospace', fontSize: '0.95rem' }}>
                {abbrev.abbreviation}
              </TableCell>
              <TableCell sx={{ color: 'text.secondary' }}>{abbrev.full_word}</TableCell>
              <TableCell>
                <Chip label={abbrev.is_core ? 'CORE' : 'CUSTOM'} size="small" color={abbrev.is_core ? 'primary' : 'default'} variant={abbrev.is_core ? 'filled' : 'outlined'} />
              </TableCell>
              <TableCell sx={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>{abbrev.notes || '—'}</TableCell>
              <TableCell align="right">
                <Tooltip title={abbrev.tenant_id === tenantId ? 'Edit' : 'Read Only'}>
                  <span>
                    <IconButton
                      size="small"
                      onClick={() => handleEditAbbreviation(abbrev)}
                      disabled={abbrev.tenant_id !== tenantId}
                      color="primary"
                    >
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
                <Tooltip title={abbrev.tenant_id === tenantId ? 'Delete' : 'Read Only'}>
                  <span>
                    <IconButton
                      size="small"
                      onClick={() => handleDeleteAbbreviation(abbrev.id)}
                      disabled={abbrev.tenant_id !== tenantId}
                      color="error"
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );

  // Render card view
  const renderCardView = () => (
    <Grid container spacing={2}>
      {filteredAndSorted.map((abbrev: any) => (
        <Grid item xs={12} sm={6} md={4} key={abbrev.id}>
          <Card sx={{ display: 'flex', flexDirection: 'column', height: '100%' }} elevation={2}>
            <Box sx={{ p: 2, pb: 1 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', mb: 1 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, fontFamily: 'monospace', fontSize: '1.1rem' }}>
                  {abbrev.abbreviation}
                </Typography>
                <Chip label={abbrev.is_core ? 'CORE' : 'CUSTOM'} size="small" color={abbrev.is_core ? 'primary' : 'default'} variant={abbrev.is_core ? 'filled' : 'outlined'} />
              </Box>
              <Typography variant="body2" sx={{ color: 'text.secondary', mb: 1 }}>
                {abbrev.full_word}
              </Typography>
              {abbrev.notes && <Typography variant="caption">{abbrev.notes}</Typography>}
            </Box>
            <Divider />
            <Box sx={{ p: 2, display: 'flex', gap: 0.5, justifyContent: 'flex-end' }}>
              <Tooltip title={abbrev.tenant_id === tenantId ? 'Edit' : 'Read Only'}>
                <span>
                  <IconButton
                    size="small"
                    onClick={() => handleEditAbbreviation(abbrev)}
                    disabled={abbrev.tenant_id !== tenantId}
                    color="primary"
                  >
                    <EditIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title={abbrev.tenant_id === tenantId ? 'Delete' : 'Read Only'}>
                <span>
                  <IconButton
                    size="small"
                    onClick={() => handleDeleteAbbreviation(abbrev.id)}
                    disabled={abbrev.tenant_id !== tenantId}
                    color="error"
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
            </Box>
          </Card>
        </Grid>
      ))}
    </Grid>
  );

  return (
    <Box sx={{ display: 'flex', height: '100%', bgcolor: '#fafafa' }}>
      {/* Sidebar with Facets */}
      <Drawer variant="permanent" sx={{ width: sidebarOpen ? 280 : 0, transition: 'all 0.3s', overflow: 'hidden' }} PaperProps={{ sx: { position: 'relative', width: 280, pt: 2 } }}>
        <Box sx={{ px: 2 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, textTransform: 'uppercase', fontSize: '0.75rem', color: 'text.secondary', mb: 2 }}>
            Type
          </Typography>
          <Stack spacing={1}>
            {[
              { key: 'all', label: 'All', count: facetCounts.all },
              { key: 'core', label: 'Core', count: facetCounts.core },
              { key: 'custom', label: 'Custom', count: facetCounts.custom },
            ].map((facet) => (
              <Button
                key={facet.key}
                variant={filterType === facet.key ? 'contained' : 'text'}
                fullWidth
                onClick={() => setFilterType(facet.key as FilterType)}
                sx={{
                  justifyContent: 'space-between',
                  textTransform: 'none',
                  fontWeight: filterType === facet.key ? 600 : 400,
                }}
              >
                <span>{facet.label}</span>
                <Chip label={facet.count} size="small" variant="outlined" sx={{ ml: 1 }} />
              </Button>
            ))}
          </Stack>
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 3 }}>
        {/* Header */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="h5" sx={{ fontWeight: 600, mb: 3 }}>
            Abbreviations
          </Typography>

          {/* Tab Navigation */}
          <Paper sx={{ display: 'flex', gap: 1, p: 1, mb: 3 }}>
            {[
              { id: 'list', label: loaded ? `List (${totalCount})` : 'List' },
              { id: 'add', label: 'Add New' },
              { id: 'test', label: 'Test Expansion' },
              { id: 'validate', label: 'Validate Terms' },
            ].map((tab) => (
              <Button
                key={tab.id}
                onClick={() => handleTabChange(tab.id as any)}
                variant={activeTab === tab.id ? 'contained' : 'outlined'}
                sx={{ textTransform: 'none' }}
              >
                {tab.label}
              </Button>
            ))}
          </Paper>
        </Box>

        {/* List Tab */}
        {activeTab === 'list' && (
          <>
            {/* Search and Controls */}
            <Paper sx={{ p: 2, mb: 3 }}>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', justifyContent: 'space-between' }}>
                <TextField
                  placeholder="Search abbreviations..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  size="small"
                  sx={{ flex: 1, maxWidth: 400 }}
                />
                <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                  <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                    {filteredAndSorted.length}
                  </Typography>
                  <ToggleButtonGroup value={viewMode} exclusive onChange={(e, newVal) => newVal && setViewMode(newVal)} size="small">
                    <ToggleButton value="table" title="Table view">
                      <TableViewIcon fontSize="small" />
                    </ToggleButton>
                    <ToggleButton value="card" title="Card view">
                      <CardViewIcon fontSize="small" />
                    </ToggleButton>
                  </ToggleButtonGroup>
                  <Tooltip title="Add new abbreviation">
                    <IconButton size="small" onClick={() => handleTabChange('add')} color="primary">
                      <AddIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Export">
                    <IconButton
                      size="small"
                      onClick={() => {
                        const format = window.confirm('Export as CSV? (Cancel for JSON)') ? 'csv' : 'json';
                        const url = `/api/abbreviations/export?format=${format}`;
                        window.location.href = url;
                      }}
                    >
                      <DownloadIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </Box>
              </Box>
            </Paper>

            {/* Content */}
            {loading && !abbreviations?.length ? (
              <CircularProgress sx={{ mx: 'auto' }} />
            ) : filteredAndSorted.length === 0 ? (
              <Box sx={{ textAlign: 'center', py: 6 }}>
                <Typography color="text.secondary">
                  {searchQuery ? 'No matching abbreviations' : 'No abbreviations found'}
                </Typography>
              </Box>
            ) : viewMode === 'table' ? (
              renderTableView()
            ) : (
              renderCardView()
            )}

            {/* Load More */}
            {hasMore && (
              <Box sx={{ display: 'flex', justifyContent: 'center', mt: 3 }}>
                <Button variant="outlined" onClick={loadMore} disabled={loading}>
                  {loading ? <CircularProgress size={20} /> : 'Load More'}
                </Button>
              </Box>
            )}
          </>
        )}

        {/* Add New Tab */}
        {activeTab === 'add' && (
          <Box>
            <Typography variant="h6" sx={{ mb: 3 }}>
              Add New Abbreviation
            </Typography>
            <Paper sx={{ p: 3, maxWidth: 600 }}>
              <Stack spacing={2}>
                <TextField
                  label="Abbreviation"
                  value={newAbbreviation}
                  onChange={(e) => setNewAbbreviation(e.target.value.toUpperCase())}
                  placeholder="e.g., TXN"
                  fullWidth
                  required
                />
                <TextField
                  label="Full Word"
                  value={newFullWord}
                  onChange={(e) => setNewFullWord(e.target.value.toUpperCase())}
                  placeholder="e.g., TRANSACTION"
                  fullWidth
                  required
                />
                <TextField
                  label="Notes"
                  value={newNotes}
                  onChange={(e) => setNewNotes(e.target.value)}
                  placeholder="Additional context..."
                  multiline
                  rows={3}
                  fullWidth
                />
                <Stack direction="row" spacing={2}>
                  <Button variant="contained" onClick={handleAddAbbreviation} disabled={!newAbbreviation.trim() || !newFullWord.trim()}>
                    Add Abbreviation
                  </Button>
                  <Button variant="outlined" onClick={() => { setNewAbbreviation(''); setNewFullWord(''); setNewNotes(''); }}>
                    Clear
                  </Button>
                </Stack>
              </Stack>
            </Paper>
          </Box>
        )}

        {/* Test Tab */}
        {activeTab === 'test' && (
          <Box>
            <Typography variant="h6" sx={{ mb: 3 }}>
              Test Abbreviation Expansion
            </Typography>
            <Paper sx={{ p: 3, maxWidth: 600 }}>
              <Stack spacing={2}>
                <TextField
                  label="Column Name"
                  value={testColumn}
                  onChange={(e) => setTestColumn(e.target.value)}
                  placeholder="e.g., CUST_ID"
                  fullWidth
                />
                <Button variant="contained" onClick={handleTestExpansion} disabled={!testColumn.trim()}>
                  Test Expansion
                </Button>
                {expansionResult && (
                  <Box sx={{ mt: 2, p: 2, bgcolor: 'info.50', borderRadius: 1 }}>
                    <Typography variant="subtitle2" sx={{ mb: 1 }}>
                      Results:
                    </Typography>
                    <Typography variant="body2">
                      <strong>Original:</strong> {expansionResult.column_name}
                    </Typography>
                    {expansionResult.variations && (
                      <Typography variant="body2">
                        <strong>Variations:</strong> {expansionResult.variations.join(', ')}
                      </Typography>
                    )}
                  </Box>
                )}
              </Stack>
            </Paper>
          </Box>
        )}

        {/* Validate Tab */}
        {activeTab === 'validate' && (
          <Box>
            <Typography variant="h6" sx={{ mb: 3 }}>
              Validate Semantic Terms
            </Typography>
            <Paper sx={{ p: 3, maxWidth: 600 }}>
              <Stack spacing={2}>
                <TextField
                  label="Semantic Terms"
                  value={testTerms}
                  onChange={(e) => setTestTerms(e.target.value)}
                  placeholder="e.g., CUSTOMER_ID, TXN_AMT"
                  multiline
                  rows={4}
                  fullWidth
                  helperText="Enter terms separated by commas"
                />
                <Button variant="contained" onClick={handleValidateTerms} disabled={!testTerms.trim()}>
                  Validate Terms
                </Button>
                {validationResult && (
                  <Box sx={{ mt: 2, p: 2, bgcolor: 'info.50', borderRadius: 1 }}>
                    <Typography variant="subtitle2" sx={{ mb: 2 }}>
                      Validation Results:
                    </Typography>
                    <Grid container spacing={2}>
                      <Grid item xs={6}>
                        <Box sx={{ p: 1, bgcolor: 'success.50', borderRadius: 1 }}>
                          <Typography variant="caption">Valid Terms</Typography>
                          <Typography variant="h6">{validationResult.valid_terms}</Typography>
                        </Box>
                      </Grid>
                      <Grid item xs={6}>
                        <Box sx={{ p: 1, bgcolor: 'error.50', borderRadius: 1 }}>
                          <Typography variant="caption">With Abbreviations</Typography>
                          <Typography variant="h6">{Object.keys(validationResult.violations || {}).length}</Typography>
                        </Box>
                      </Grid>
                    </Grid>
                  </Box>
                )}
              </Stack>
            </Paper>
          </Box>
        )}
      </Box>

      {/* Edit Modal */}
      <Dialog open={showEditModal} onClose={handleCloseEditModal} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Abbreviation</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <TextField
              label="Abbreviation"
              value={newAbbreviation}
              onChange={(e) => setNewAbbreviation(e.target.value.toUpperCase())}
              fullWidth
              required
            />
            <TextField
              label="Full Word"
              value={newFullWord}
              onChange={(e) => setNewFullWord(e.target.value.toUpperCase())}
              fullWidth
              required
            />
            <TextField label="Notes" value={newNotes} onChange={(e) => setNewNotes(e.target.value)} multiline rows={3} fullWidth />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseEditModal}>Cancel</Button>
          <Button variant="contained" onClick={handleUpdateAbbreviation}>
            Update
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default AbbreviationManager;
