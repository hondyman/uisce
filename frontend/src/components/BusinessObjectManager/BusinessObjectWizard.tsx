import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Box,
  Button,
  Stepper,
  Step,
  StepLabel,
  TextField,
  Typography,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  ListItemSecondaryAction,
  Checkbox,
  Chip,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Autocomplete,
  IconButton,
  Collapse,
  Radio,
  RadioGroup,
  FormControlLabel,
  Divider,
  Tooltip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  InputAdornment,
  CircularProgress,
  Grid,
  Card,
  CardContent,
  Badge,
} from '@mui/material';
import {
  Close as CloseIcon,
  TableChart as TableIcon,
  AccountTree as TermIcon,
  Link as LinkIcon,
  Check as CheckIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Delete as DeleteIcon,
  Add as AddIcon,
  Search as SearchIcon,
  Storage as StorageIcon,
  Category as CategoryIcon,
  ArrowForward as ArrowForwardIcon,
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';
import { useNotification } from '../../hooks/useNotification';
import resolveApiUrl from '../../utils/resolveApiUrl';

import { getSelectedRegion } from '../../lib/region';

// ============================================================================
// Types
// ============================================================================

interface CatalogNode {
  id: string;
  node_name: string;
  description?: string;
  qualified_path?: string;
}

interface WizardSemanticTerm {
  termId: string;
  termName: string;
  displayName: string;
  columnId: string;
  columnName: string;
  dataType?: string;
  description?: string;
  selected?: boolean;
}

interface WizardRelatedTable {
  tableId: string;
  tableName: string;
  fkName: string;
  existingBOId?: string;
  existingBOName?: string;
  semanticTerms?: WizardSemanticTerm[];
  linkType: 'include_terms' | 'link_bo' | 'create_new' | 'ignore';
  expanded?: boolean;
  selectedTerms?: string[];
}

interface WizardContext {
  drivingTable: {
    id: string;
    name: string;
    qualifiedPath: string;
    columnCount: number;
    termCount: number;
    relatedCount: number;
  };
  semanticTerms: WizardSemanticTerm[];
  relatedTables: WizardRelatedTable[];
}

interface BusinessObjectWizardProps {
  open: boolean;
  onClose: () => void;
  onSave?: (boId: string) => void;
  existingBO?: any; // For editing existing BOs
}

const WIZARD_STEPS = [
  { label: 'Select Driving Table', icon: <TableIcon /> },
  { label: 'Semantic Terms', icon: <TermIcon /> },
  { label: 'Related Objects', icon: <LinkIcon /> },
  { label: 'Review & Save', icon: <CheckIcon /> },
];

// ============================================================================
// Main Component
// ============================================================================

export const BusinessObjectWizard: React.FC<BusinessObjectWizardProps> = ({
  open,
  onClose,
  onSave,
  existingBO,
}) => {
  const { tenant, datasource } = useTenant();
  const notification = useNotification();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';

  // Wizard state
  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<{code: string; field: string; message: string}[]>([]);

  // Step 1: Driving Table
  const [tables, setTables] = useState<CatalogNode[]>([]);
  const [selectedTable, setSelectedTable] = useState<CatalogNode | null>(null);
  const [tableSearchInput, setTableSearchInput] = useState('');

  // Step 2: Semantic Terms
  const [wizardContext, setWizardContext] = useState<WizardContext | null>(null);
  const [selectedTermIds, setSelectedTermIds] = useState<Set<string>>(new Set());
  const [relatedTables, setRelatedTables] = useState<WizardRelatedTable[]>([]);
  const [termSearchQuery, setTermSearchQuery] = useState('');

  // Step 4: BO Metadata
  const [boKey, setBoKey] = useState('');
  const [boName, setBoName] = useState('');
  const [boDisplayName, setBoDisplayName] = useState('');
  const [boDescription, setBoDescription] = useState('');

  // Helper to build headers with authentication
  const getAuthHeaders = (additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';
    
    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      'X-Tenant-Region': getSelectedRegion(),
      ...additionalHeaders,
    };
  };

  // Load tables on mount
  useEffect(() => {
    if (open && tenantId && datasourceId) {
      loadTables();
    }
  }, [open, tenantId, datasourceId]);

  // Load context when table is selected
  useEffect(() => {
    if (selectedTable) {
      loadWizardContext(selectedTable.id);
    }
  }, [selectedTable]);

  // ============================================================================
  // API Calls
  // ============================================================================

  const loadTables = async (search?: string) => {
    try {
      const url = resolveApiUrl(`/api/catalog/nodes?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&type=table${search ? `&q=${encodeURIComponent(search)}` : ''}`);
      const response = await fetch(url, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setTables(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error('Failed to load tables:', err);
    }
  };

  const loadWizardContext = async (tableId: string) => {
    setLoading(true);
    setError(null);
    try {
      const url = resolveApiUrl(`/api/bo-wizard/context/${tableId}`);
      const response = await fetch(url, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        throw new Error('Failed to load wizard context');
      }
      const context: WizardContext = await response.json();
      setWizardContext(context);
      
      // Initialize related tables with expansion state (handle null/undefined)
      setRelatedTables(
        (context.relatedTables || []).map((t) => ({
          ...t,
          expanded: false,
          selectedTerms: [],
        }))
      );

      // Auto-select only terms already mapped (t.selected === true)
      setSelectedTermIds(new Set((context.semanticTerms || []).filter((t) => t.selected).map((t) => t.termId)));
      
      // Pre-fill BO metadata from table name
      const tableName = context.drivingTable.name;
      setBoKey(tableName.toLowerCase().replace(/[^a-z0-9]/g, '_'));
      setBoName(tableName);
      setBoDisplayName(tableName.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' '));
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const saveBusinessObject = async () => {
    setLoading(true);
    setError(null);
    try {
      // Pre-create existence check to avoid unique constraint violations
      // (only for UPDATE scenario; new BOs should skip this)
      const normalizedKey = (boKey || boName || '').toLowerCase().replace(/[^a-z0-9_]/g, '').trim();
      if (normalizedKey && existingBO?.id) {
        // Only check existence if we're updating an existing BO
        const existsUrl = resolveApiUrl(`/api/business-objects/${encodeURIComponent(normalizedKey)}`);
        const existsResp = await fetch(existsUrl, {
          headers: getAuthHeaders(),
        });
        if (existsResp.ok) {
          const existing = await existsResp.json();
          // Inform user and short-circuit creation; return existing
          notification.warning(`Business Object "${existing.displayName || normalizedKey}" already exists. Opening existing.`);
          setValidationErrors([]);
          onSave?.(existing.id || normalizedKey);
          onClose();
          return;
        }
      }

      // Collect linked BOs
      const linkedBOs = relatedTables
        .filter((t) => t.linkType === 'link_bo' && t.existingBOId)
        .map((t) => ({ bo_id: t.existingBOId!, relationship_type: 'RELATES_TO' }));

      // Collect included terms from related tables
      const includedTermsFromTables = relatedTables
        .filter((t) => t.linkType === 'include_terms' && t.selectedTerms && t.selectedTerms.length > 0)
        .map((t) => ({ table_id: t.tableId, term_ids: t.selectedTerms! }));

      const payload = {
        bo_key: normalizedKey || boKey,
        name: boName,
        display_name: boDisplayName,
        description: boDescription || undefined,
        driver_table_id: selectedTable!.id,
        selected_terms: Array.from(selectedTermIds),
        linked_bos: linkedBOs,
        included_terms_from_tables: includedTermsFromTables,
      };

      const saveUrl = resolveApiUrl('/api/bo-wizard/save');
      const response = await fetch(saveUrl, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        // Try to parse validation errors
        const contentType = response.headers.get('content-type');
        if (contentType?.includes('application/json')) {
          const validationResult = await response.json();
          if (validationResult.errors && Array.isArray(validationResult.errors)) {
            setValidationErrors(validationResult.errors);
            setError(validationResult.errors.map((e: any) => e.message).join('. '));
            return;
          }
        }
        const errData = await response.text();
        throw new Error(errData || 'Failed to save business object');
      }

      setValidationErrors([]);
      const result = await response.json();
      onSave?.(result.id);
      onClose();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // ============================================================================
  // Navigation
  // ============================================================================

  const handleNext = () => {
    if (activeStep === WIZARD_STEPS.length - 1) {
      saveBusinessObject();
    } else {
      setActiveStep((prev) => prev + 1);
    }
  };

  const handleBack = () => {
    setActiveStep((prev) => prev - 1);
  };

  const canProceed = useCallback(() => {
    switch (activeStep) {
      case 0:
        return selectedTable !== null;
      case 1:
        return selectedTermIds.size > 0;
      case 2:
        return true; // Related objects are optional
      case 3:
        return boKey.trim() !== '' && boName.trim() !== '';
      default:
        return false;
    }
  }, [activeStep, selectedTable, selectedTermIds, boKey, boName]);

  // ============================================================================
  // Term Selection Handlers
  // ============================================================================

  // Track terms that were originally mapped and user removed; prevent re-adding
  const [blockedTermIds, setBlockedTermIds] = useState<Set<string>>(new Set());

  // Related-table term locks for include_terms mode
  const [blockedRelatedTerms, setBlockedRelatedTerms] = useState<Record<string, Set<string>>>(() => ({}));

  const toggleTerm = (term: WizardSemanticTerm) => {
    const termId = term.termId;
    setSelectedTermIds((prev) => {
      const next = new Set(prev);
      const wasOriginallyMapped = !!term.selected;

      if (next.has(termId)) {
        // Removing selection is always allowed
        next.delete(termId);
        // If originally mapped, mark as blocked from re-adding in this session
        if (wasOriginallyMapped) {
          setBlockedTermIds((bPrev) => new Set(bPrev).add(termId));
          notification.info(`Removed already-mapped term: ${term.displayName}`);
        }
      } else {
        // Adding selection
        if (wasOriginallyMapped) {
          // Cannot select already-mapped terms for addition
          setBlockedTermIds((bPrev) => new Set(bPrev).add(termId));
          notification.warning(`"${term.displayName}" is already mapped and cannot be added. You may remove it.`);
          return prev; // keep unchanged
        }
        if (blockedTermIds.has(termId)) {
          // Prevent re-adding after removal of originally mapped term
          notification.warning(`"${term.displayName}" was removed and cannot be re-added in this session.`);
          return prev;
        }
        next.add(termId);
      }
      return next;
    });
  };

  const toggleRelatedTableTerm = (tableId: string, termId: string) => {
    setRelatedTables((prev) => {
      return prev.map((t) => {
        if (t.tableId !== tableId) return t;

        const wasOriginallyMapped = !!t.semanticTerms?.find((st) => st.termId === termId)?.selected;
        const blockedForTable = blockedRelatedTerms[tableId] || new Set<string>();

        // If blocked (removed originally mapped), prevent re-add
        if (blockedForTable.has(termId)) {
          notification.warning(`"${termId}" was removed and cannot be re-added in this session.`);
          return t;
        }

        // Toggle selection list
        const selectedTerms = t.selectedTerms || [];
        const isSelected = selectedTerms.includes(termId);
        if (isSelected) {
          // Removing is allowed
          const next = selectedTerms.filter((id) => id !== termId);
          // If originally mapped, lock from re-adding
          if (wasOriginallyMapped) {
            const updatedBlocked = new Set(blockedForTable);
            updatedBlocked.add(termId);
            setBlockedRelatedTerms((prevBlocked) => ({ ...prevBlocked, [tableId]: updatedBlocked }));
            notification.info(`Removed already-mapped term from related table: ${termId}`);
          }
          return { ...t, selectedTerms: next };
        }

        // Adding
        if (wasOriginallyMapped) {
          notification.warning(`"${termId}" is already mapped and cannot be added. You may remove it.`);
          return t;
        }

        return { ...t, selectedTerms: [...selectedTerms, termId] };
      });
    });
  };

  const selectAllTerms = () => {
    if (wizardContext && wizardContext.semanticTerms) {
      setSelectedTermIds(new Set(wizardContext.semanticTerms.map((t) => t.termId)));
    }
  };

  const deselectAllTerms = () => {
    setSelectedTermIds(new Set());
  };

  // ============================================================================
  // Related Table Handlers
  // ============================================================================

  const toggleRelatedTableExpanded = (tableId: string) => {
    setRelatedTables((prev) =>
      prev.map((t) => (t.tableId === tableId ? { ...t, expanded: !t.expanded } : t))
    );
  };

  const setRelatedTableLinkType = (tableId: string, linkType: WizardRelatedTable['linkType']) => {
    setRelatedTables((prev) =>
      prev.map((t) => (t.tableId === tableId ? { ...t, linkType } : t))
    );
  };

  // ============================================================================
  // Render Steps
  // ============================================================================

  const renderStep1 = () => (
    <Box sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 500 }}>
        Select the primary table that drives this business object
      </Typography>

      <Autocomplete
        options={tables}
        getOptionLabel={(option) => option.node_name}
        value={selectedTable}
        isOptionEqualToValue={(option, value) => option.id === value.id}
        onChange={(_, value) => setSelectedTable(value)}
        inputValue={tableSearchInput}
        onInputChange={(_, value) => {
          setTableSearchInput(value);
          loadTables(value);
        }}
        renderInput={(params) => (
          <TextField
            {...params}
            label="Driving Table"
            placeholder="Search tables..."
            fullWidth
            sx={{ mt: 2 }}
          />
        )}
        renderOption={(props, option) => (
          <li {...props} key={option.id}>
            <ListItemIcon>
              <StorageIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText
              primary={option.node_name}
              secondary={option.qualified_path || option.description}
            />
          </li>
        )}
      />

      {selectedTable && wizardContext && (
        <Card sx={{ mt: 3 }} variant="outlined">
          <CardContent>
            <Typography variant="h6" gutterBottom>
              <StorageIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
              {wizardContext.drivingTable.name}
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={4}>
                <Typography variant="body2" color="text.secondary">
                  Columns
                </Typography>
                <Typography variant="h5">{wizardContext.drivingTable.columnCount}</Typography>
              </Grid>
              <Grid item xs={4}>
                <Typography variant="body2" color="text.secondary">
                  Semantic Terms
                </Typography>
                <Typography variant="h5" color="primary">
                  {wizardContext.drivingTable.termCount}
                </Typography>
              </Grid>
              <Grid item xs={4}>
                <Typography variant="body2" color="text.secondary">
                  Related Tables
                </Typography>
                <Typography variant="h5" color="secondary">
                  {wizardContext.drivingTable.relatedCount}
                </Typography>
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      )}
    </Box>
  );

  // Filter semantic terms based on search query
  const filteredTerms = useMemo(() => {
    if (!wizardContext?.semanticTerms) return [];
    if (!termSearchQuery.trim()) return wizardContext.semanticTerms;
    
    const query = termSearchQuery.toLowerCase();
    return wizardContext.semanticTerms.filter(term =>
      term.displayName.toLowerCase().includes(query) ||
      term.columnName.toLowerCase().includes(query) ||
      term.dataType?.toLowerCase().includes(query) ||
      term.description?.toLowerCase().includes(query)
    );
  }, [wizardContext?.semanticTerms, termSearchQuery]);

  const renderStep2 = () => (
    <Box sx={{ p: 2 }}>
      <Typography variant="subtitle1" sx={{ fontWeight: 500, mb: 2 }}>
        Select semantic terms to include in this business object
      </Typography>

      {/* Search and action buttons */}
      <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
        <TextField
          placeholder="Search semantic terms..."
          value={termSearchQuery}
          onChange={(e) => setTermSearchQuery(e.target.value)}
          size="small"
          sx={{ flexGrow: 1 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
        <Button size="small" variant="outlined" onClick={selectAllTerms}>
          Select All
        </Button>
        <Button size="small" variant="outlined" onClick={deselectAllTerms}>
          Deselect All
        </Button>
      </Box>

      {/* Table with semantic terms */}
      {wizardContext && wizardContext.semanticTerms && wizardContext.semanticTerms.length > 0 && (
        <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 500 }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox">
                  <Checkbox
                    checked={filteredTerms.length > 0 && filteredTerms.every(t => selectedTermIds.has(t.termId))}
                    indeterminate={
                      filteredTerms.some(t => selectedTermIds.has(t.termId)) &&
                      !filteredTerms.every(t => selectedTermIds.has(t.termId))
                    }
                    onChange={() => {
                      if (filteredTerms.every(t => selectedTermIds.has(t.termId))) {
                        // Deselect all filtered
                        const newSelected = new Set(selectedTermIds);
                        filteredTerms.forEach(t => newSelected.delete(t.termId));
                        setSelectedTermIds(newSelected);
                      } else {
                        // Select all filtered
                        const newSelected = new Set(selectedTermIds);
                        filteredTerms.forEach(t => newSelected.add(t.termId));
                        setSelectedTermIds(newSelected);
                      }
                    }}
                  />
                </TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Semantic Term</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Column</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Data Type</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Description</TableCell>
                <TableCell sx={{ fontWeight: 600, width: 140 }}>State</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredTerms.map((term) => (
                <TableRow
                  key={term.termId}
                  hover
                  onClick={() => toggleTerm(term)}
                  sx={{
                    cursor: blockedTermIds.has(term.termId) ? 'not-allowed' : 'pointer',
                    backgroundColor: selectedTermIds.has(term.termId) ? 'action.selected' : 'inherit',
                    opacity: blockedTermIds.has(term.termId) ? 0.6 : 1,
                  }}
                >
                  <TableCell padding="checkbox">
                    <Checkbox checked={selectedTermIds.has(term.termId)} disabled={blockedTermIds.has(term.termId)} />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>
                      {term.displayName}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {term.columnName}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {term.dataType ? (
                      <Chip label={term.dataType} size="small" variant="outlined" />
                    ) : (
                      <Typography variant="body2" color="text.disabled">-</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 300 }}>
                      {term.description || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {term.selected && (
                      <Chip label="Already mapped" size="small" color="info" variant="outlined" sx={{ mr: 1 }} />
                    )}
                    {blockedTermIds.has(term.termId) && (
                      <Chip label="Removed" size="small" color="warning" variant="outlined" />
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* No terms message */}
      {wizardContext && wizardContext.semanticTerms && wizardContext.semanticTerms.length === 0 && (
        <Alert severity="info" sx={{ mt: 2 }}>
          No semantic terms have been mapped to this table's columns yet. You can still create the
          business object and add terms later.
        </Alert>
      )}

      {/* No search results message */}
      {wizardContext && wizardContext.semanticTerms && wizardContext.semanticTerms.length > 0 && filteredTerms.length === 0 && (
        <Alert severity="info" sx={{ mt: 2 }}>
          No semantic terms match your search query.
        </Alert>
      )}

      {/* Selection summary */}
      <Box sx={{ mt: 2, display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
        <Chip
          label={`${selectedTermIds.size} of ${wizardContext?.semanticTerms?.length || 0} terms selected`}
          color="primary"
          variant="outlined"
        />
        {termSearchQuery && (
          <Chip
            label={`${filteredTerms.length} matching search`}
            size="small"
            variant="outlined"
          />
        )}
        <Chip label="Already mapped" size="small" color="info" variant="outlined" />
        <Chip label="Removed (locked)" size="small" color="warning" variant="outlined" />
        <Typography variant="caption" color="text.secondary" sx={{ ml: 1 }}>
          Already mapped terms can be removed but not re-added in this session.
        </Typography>
      </Box>
    </Box>
  );

  const renderStep3 = () => (
    <Box sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 500 }}>
        Configure relationships with related tables
      </Typography>

      {relatedTables.length === 0 ? (
        <Alert severity="info" sx={{ mt: 2 }}>
          No related tables found via foreign key relationships.
        </Alert>
      ) : (
        <>
          {/* Governance warning for tables with existing BOs */}
          {relatedTables.filter((t) => t.existingBOId && t.linkType !== 'link_bo').length > 0 && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              <strong>BO Linking Recommended:</strong> Some related tables already have Business Objects defined. 
              Linking to existing BOs maintains semantic consistency. "Include Terms" may create duplicate semantic definitions.
            </Alert>
          )}
          <List>
          {relatedTables.map((table) => (
            <Paper key={table.tableId} variant="outlined" sx={{ mb: 2 }}>
              <ListItem
                button
                onClick={() => toggleRelatedTableExpanded(table.tableId)}
              >
                <ListItemIcon>
                  <StorageIcon />
                </ListItemIcon>
                <ListItemText
                  primary={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {table.tableName}
                      {table.existingBOName && (
                        <Chip
                          label={`BO: ${table.existingBOName}`}
                          size="small"
                          color="success"
                          variant="outlined"
                          sx={{ height: 20, fontSize: '0.75rem' }}
                        />
                      )}
                    </Box>
                  }
                  secondary={`FK: ${table.fkName}`}
                />
                <ListItemSecondaryAction>
                  <IconButton onClick={() => toggleRelatedTableExpanded(table.tableId)}>
                    {table.expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                  </IconButton>
                </ListItemSecondaryAction>
              </ListItem>

              <Collapse in={table.expanded}>
                <Box sx={{ p: 2, bgcolor: 'background.default' }}>
                  <Typography variant="body2" sx={{ mb: 2 }}>
                    How should this related table be handled?
                  </Typography>

                  <Grid container spacing={1}>
                    {table.existingBOId && (
                      <Grid item>
                        <Chip
                          label="Link to Business Object"
                          onClick={() => setRelatedTableLinkType(table.tableId, 'link_bo')}
                          color={table.linkType === 'link_bo' ? 'primary' : 'default'}
                          variant={table.linkType === 'link_bo' ? 'filled' : 'outlined'}
                          icon={<LinkIcon />}
                        />
                      </Grid>
                    )}
                    <Grid item>
                      <Chip
                        label="Include Terms"
                        onClick={() => setRelatedTableLinkType(table.tableId, 'include_terms')}
                        color={table.linkType === 'include_terms' ? 'primary' : 'default'}
                        variant={table.linkType === 'include_terms' ? 'filled' : 'outlined'}
                        icon={<TermIcon />}
                      />
                    </Grid>
                    <Grid item>
                      <Chip
                        label="Ignore"
                        onClick={() => setRelatedTableLinkType(table.tableId, 'ignore')}
                        color={table.linkType === 'ignore' ? 'default' : 'default'}
                        variant={table.linkType === 'ignore' ? 'filled' : 'outlined'}
                      />
                    </Grid>
                  </Grid>

                  {table.linkType === 'include_terms' && table.semanticTerms && (
                    <Box sx={{ mt: 2 }}>
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                        Select terms to include ({table.semanticTerms.length} available):
                      </Typography>
                      <List dense sx={{ maxHeight: 200, overflow: 'auto' }}>
                        {table.semanticTerms.map((term) => (
                          <ListItem
                            key={term.termId}
                            button
                            onClick={() => toggleRelatedTableTerm(table.tableId, term.termId)}
                            dense
                          >
                            <ListItemIcon>
                              <Checkbox
                                checked={table.selectedTerms?.includes(term.termId) || false}
                                size="small"
                                disabled={(blockedRelatedTerms[table.tableId]?.has(term.termId)) || false}
                              />
                            </ListItemIcon>
                            <ListItemText
                              primary={
                                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                  {term.displayName}
                                  {term.selected && (
                                    <Chip label="Already mapped" size="small" color="info" variant="outlined" />
                                  )}
                                  {blockedRelatedTerms[table.tableId]?.has(term.termId) && (
                                    <Chip label="Removed" size="small" color="warning" variant="outlined" />
                                  )}
                                </Box>
                              }
                              secondary={term.columnName}
                            />
                          </ListItem>
                        ))}
                      </List>
                      <Typography variant="caption" color="text.secondary">
                        Already mapped terms can be removed but not re-added in this session.
                      </Typography>
                    </Box>
                  )}

                  {table.linkType === 'link_bo' && table.existingBOName && (
                    <Alert severity="success" sx={{ mt: 2 }}>
                      Will create a relationship to "{table.existingBOName}" business object.
                    </Alert>
                  )}
                </Box>
              </Collapse>
            </Paper>
          ))}
          </List>
        </>
      )}
    </Box>
  );

  const renderStep4 = () => {
    const linkedBOCount = relatedTables.filter(
      (t) => t.linkType === 'link_bo' && t.existingBOId
    ).length;
    const includedTermCount = relatedTables.reduce(
      (sum, t) =>
        sum + (t.linkType === 'include_terms' ? t.selectedTerms?.length || 0 : 0),
      0
    );

    return (
      <Box sx={{ p: 2 }}>
        <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 500 }}>
          Review and complete your business object
        </Typography>

        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Typography variant="subtitle2" gutterBottom>
              Business Object Details
            </Typography>
            <TextField
              label="Key"
              value={boKey}
              onChange={(e) => setBoKey(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
              fullWidth
              margin="dense"
              helperText="Unique identifier (lowercase, no spaces)"
              required
            />
            <TextField
              label="Name"
              value={boName}
              onChange={(e) => setBoName(e.target.value)}
              fullWidth
              margin="dense"
              required
            />
            <TextField
              label="Display Name"
              value={boDisplayName}
              onChange={(e) => setBoDisplayName(e.target.value)}
              fullWidth
              margin="dense"
            />
            <TextField
              label="Description"
              value={boDescription}
              onChange={(e) => setBoDescription(e.target.value)}
              fullWidth
              margin="dense"
              multiline
              rows={3}
            />
          </Grid>

          <Grid item xs={12} md={6}>
            <Typography variant="subtitle2" gutterBottom>
              Summary
            </Typography>

            <Card variant="outlined">
              <CardContent>
                <List dense>
                  <ListItem>
                    <ListItemIcon>
                      <StorageIcon />
                    </ListItemIcon>
                    <ListItemText
                      primary="Driving Table"
                      secondary={selectedTable?.node_name || 'Not selected'}
                    />
                  </ListItem>
                  <Divider />
                  <ListItem>
                    <ListItemIcon>
                      <Badge badgeContent={selectedTermIds.size} color="primary">
                        <TermIcon />
                      </Badge>
                    </ListItemIcon>
                    <ListItemText
                      primary="Semantic Terms"
                      secondary={`${selectedTermIds.size} from driving table`}
                    />
                  </ListItem>
                  {includedTermCount > 0 && (
                    <ListItem>
                      <ListItemIcon>
                        <CategoryIcon />
                      </ListItemIcon>
                      <ListItemText
                        primary="Terms from Related Tables"
                        secondary={`${includedTermCount} additional terms`}
                      />
                    </ListItem>
                  )}
                  {linkedBOCount > 0 && (
                    <ListItem>
                      <ListItemIcon>
                        <LinkIcon />
                      </ListItemIcon>
                      <ListItemText
                        primary="Linked Business Objects"
                        secondary={`${linkedBOCount} relationships`}
                      />
                    </ListItem>
                  )}
                </List>
              </CardContent>
            </Card>

            <Alert severity="info" sx={{ mt: 2 }}>
              Edge relationships will be processed asynchronously after saving.
            </Alert>
          </Grid>
        </Grid>
      </Box>
    );
  };

  const renderStepContent = () => {
    switch (activeStep) {
      case 0:
        return renderStep1();
      case 1:
        return renderStep2();
      case 2:
        return renderStep3();
      case 3:
        return renderStep4();
      default:
        return null;
    }
  };

  // ============================================================================
  // Main Render
  // ============================================================================

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6" component="div">
          {existingBO ? 'Edit Business Object' : 'Create Business Object'}
        </Typography>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <Box sx={{ px: 3 }}>
        <Stepper activeStep={activeStep} alternativeLabel>
          {WIZARD_STEPS.map((step, index) => (
            <Step key={step.label}>
              <StepLabel>{step.label}</StepLabel>
            </Step>
          ))}
        </Stepper>
      </Box>

      <DialogContent sx={{ minHeight: 400 }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!loading && renderStepContent()}
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose}>Cancel</Button>
        <Box sx={{ flex: 1 }} />
        <Button onClick={handleBack} disabled={activeStep === 0}>
          Back
        </Button>
        <Button
          variant="contained"
          onClick={handleNext}
          disabled={!canProceed() || loading}
          endIcon={activeStep === WIZARD_STEPS.length - 1 ? <CheckIcon /> : <ArrowForwardIcon />}
        >
          {activeStep === WIZARD_STEPS.length - 1 ? 'Create Business Object' : 'Next'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BusinessObjectWizard;
