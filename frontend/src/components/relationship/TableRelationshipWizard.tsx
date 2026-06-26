import React, { useState, useCallback, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Stepper,
  Step,
  StepLabel,
  Typography,
  Box,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Checkbox,
  Chip,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Alert,
  CircularProgress,
  LinearProgress,
  Tooltip,
  Stack,
} from '@mui/material';
import {
  Close,
  Check,
  Edit,
  Cancel,
  Link as LinkIcon,
  LinkOff,
  Search,
  AutoFixHigh,
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';

// Types
interface RelationshipCandidate {
  left_table_id: string;
  left_table_name: string;
  left_column: string;
  right_table_id: string;
  right_table_name: string;
  right_column: string;
  cardinality: string;
  confidence: number;
  join_condition: string;
  join_type: string;
  origin: string;
  lookup_candidate: boolean;
  profile?: {
    left_distinct: number;
    right_distinct: number;
    left_row_count: number;
    right_row_count: number;
    join_selectivity: number;
    left_unique: boolean;
    right_unique: boolean;
  };
  match_reasons?: string[];
}

interface TableInfo {
  id: string;
  name: string;
  schema?: string;
  row_count?: number;
}

interface TableRelationshipWizardProps {
  open: boolean;
  onClose: () => void;
  onComplete?: () => void;
  initialTables?: TableInfo[];
}

export const TableRelationshipWizard: React.FC<TableRelationshipWizardProps> = ({
  open,
  onClose,
  onComplete,
  initialTables = [],
}) => {
  const { tenant, datasource } = useTenant();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';

  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Step 1: Table selection
  const [availableTables, setAvailableTables] = useState<TableInfo[]>(initialTables);
  const [selectedTableIds, setSelectedTableIds] = useState<Set<string>>(new Set());
  const [tableSearch, setTableSearch] = useState('');

  // Step 2: Candidates
  const [candidates, setCandidates] = useState<RelationshipCandidate[]>([]);
  const [acceptedCandidates, setAcceptedCandidates] = useState<Set<number>>(new Set());
  const [rejectedCandidates, setRejectedCandidates] = useState<Set<number>>(new Set());

  // Step 3: Edit
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editForm, setEditForm] = useState<Partial<RelationshipCandidate>>({});

  // Step 4: Results
  const [createdCount, setCreatedCount] = useState(0);

  const steps = ['Select Tables', 'Review Candidates', 'Edit & Confirm', 'Create Relationships'];

  useEffect(() => {
    if (open && availableTables.length === 0) {
      fetchTables();
    }
  }, [open]);

  const fetchTables = async () => {
    try {
      setLoading(true);
      const res = await fetch('/api/catalog/tables', {
        headers: {
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
      });
      if (!res.ok) throw new Error('Failed to fetch tables');
      const data = await res.json();
      setAvailableTables(Array.isArray(data) ? data : data.tables || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tables');
    } finally {
      setLoading(false);
    }
  };

  const discoverRelationships = async () => {
    if (selectedTableIds.size === 0) return;

    try {
      setLoading(true);
      setError(null);

      const res = await fetch('/api/relationships/infer', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify({
          table_ids: Array.from(selectedTableIds),
        }),
      });

      if (!res.ok) throw new Error('Failed to discover relationships');

      const data = await res.json();
      setCandidates(data.candidates || []);
      setAcceptedCandidates(new Set());
      setRejectedCandidates(new Set());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Discovery failed');
    } finally {
      setLoading(false);
    }
  };

  const createRelationships = async () => {
    const toCreate = candidates.filter((_, idx) => acceptedCandidates.has(idx));
    if (toCreate.length === 0) return;

    try {
      setLoading(true);
      let created = 0;

      for (const candidate of toCreate) {
        const res = await fetch('/api/relationships/physical', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify({
            source_table_id: candidate.left_table_id,
            target_table_id: candidate.right_table_id,
            join_condition: candidate.join_condition,
            join_type: candidate.join_type,
            cardinality: candidate.cardinality,
            confidence: candidate.confidence,
            origin: candidate.origin,
            lookup_candidate: candidate.lookup_candidate,
          }),
        });

        if (res.ok) created++;
      }

      setCreatedCount(created);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create relationships');
    } finally {
      setLoading(false);
    }
  };

  const handleNext = async () => {
    if (activeStep === 0) {
      await discoverRelationships();
    } else if (activeStep === 2) {
      await createRelationships();
    }
    setActiveStep((prev) => prev + 1);
  };

  const handleBack = () => setActiveStep((prev) => prev - 1);

  const handleClose = () => {
    onClose();
    if (createdCount > 0 && onComplete) {
      onComplete();
    }
    // Reset state
    setTimeout(() => {
      setActiveStep(0);
      setCandidates([]);
      setAcceptedCandidates(new Set());
      setRejectedCandidates(new Set());
      setCreatedCount(0);
      setError(null);
    }, 300);
  };

  const toggleTableSelection = (id: string) => {
    const newSet = new Set(selectedTableIds);
    if (newSet.has(id)) {
      newSet.delete(id);
    } else {
      newSet.add(id);
    }
    setSelectedTableIds(newSet);
  };

  const toggleCandidate = (idx: number, accept: boolean) => {
    if (accept) {
      const newAccepted = new Set(acceptedCandidates);
      if (newAccepted.has(idx)) {
        newAccepted.delete(idx);
      } else {
        newAccepted.add(idx);
        rejectedCandidates.delete(idx);
        setRejectedCandidates(new Set(rejectedCandidates));
      }
      setAcceptedCandidates(newAccepted);
    } else {
      const newRejected = new Set(rejectedCandidates);
      if (newRejected.has(idx)) {
        newRejected.delete(idx);
      } else {
        newRejected.add(idx);
        acceptedCandidates.delete(idx);
        setAcceptedCandidates(new Set(acceptedCandidates));
      }
      setRejectedCandidates(newRejected);
    }
  };

  const startEdit = (idx: number) => {
    setEditingIndex(idx);
    setEditForm({ ...candidates[idx] });
  };

  const saveEdit = () => {
    if (editingIndex !== null) {
      const updated = [...candidates];
      updated[editingIndex] = { ...candidates[editingIndex], ...editForm };
      setCandidates(updated);
      setEditingIndex(null);
    }
  };

  const filteredTables = availableTables.filter((t) =>
    t.name.toLowerCase().includes(tableSearch.toLowerCase())
  );

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return 'success';
    if (confidence >= 0.6) return 'warning';
    return 'error';
  };

  const isNextDisabled = () => {
    if (activeStep === 0) return selectedTableIds.size < 2;
    if (activeStep === 2) return acceptedCandidates.size === 0;
    return loading;
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="lg" fullWidth>
      <DialogTitle>
        <Stack direction="row" alignItems="center" spacing={1}>
          <AutoFixHigh color="primary" />
          <Typography variant="h6">Table Relationship Wizard</Typography>
        </Stack>
        <IconButton onClick={handleClose} sx={{ position: 'absolute', right: 8, top: 8 }}>
          <Close />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers>
        <Stepper activeStep={activeStep} sx={{ mb: 4 }}>
          {steps.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {loading && <LinearProgress sx={{ mb: 2 }} />}
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {/* Step 1: Select Tables */}
        {activeStep === 0 && (
          <Box>
            <Typography variant="subtitle1" gutterBottom>
              Select tables to analyze for relationships
            </Typography>
            <TextField
              fullWidth
              size="small"
              placeholder="Search tables..."
              value={tableSearch}
              onChange={(e) => setTableSearch(e.target.value)}
              InputProps={{ startAdornment: <Search sx={{ mr: 1, color: 'text.secondary' }} /> }}
              sx={{ mb: 2 }}
            />
            <Paper variant="outlined" sx={{ maxHeight: 400, overflow: 'auto' }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell padding="checkbox"></TableCell>
                    <TableCell>Table Name</TableCell>
                    <TableCell>Schema</TableCell>
                    <TableCell align="right">Row Count</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredTables.map((table) => (
                    <TableRow
                      key={table.id}
                      hover
                      onClick={() => toggleTableSelection(table.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell padding="checkbox">
                        <Checkbox checked={selectedTableIds.has(table.id)} />
                      </TableCell>
                      <TableCell>{table.name}</TableCell>
                      <TableCell>{table.schema || '-'}</TableCell>
                      <TableCell align="right">
                        {table.row_count?.toLocaleString() || '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Paper>
            <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
              Selected: {selectedTableIds.size} tables
            </Typography>
          </Box>
        )}

        {/* Step 2: Review Candidates */}
        {activeStep === 1 && (
          <Box>
            <Typography variant="subtitle1" gutterBottom>
              Review discovered relationship candidates ({candidates.length} found)
            </Typography>
            {candidates.length === 0 ? (
              <Alert severity="info">
                No relationship candidates found. Try selecting more tables or tables with matching column names.
              </Alert>
            ) : (
              <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 400 }}>
                <Table size="small" stickyHeader>
                  <TableHead>
                    <TableRow>
                      <TableCell>Left Table</TableCell>
                      <TableCell>Right Table</TableCell>
                      <TableCell>Join Condition</TableCell>
                      <TableCell>Cardinality</TableCell>
                      <TableCell>Confidence</TableCell>
                      <TableCell>Lookup</TableCell>
                      <TableCell align="center">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {candidates.map((candidate, idx) => (
                      <TableRow
                        key={idx}
                        sx={{
                          bgcolor: acceptedCandidates.has(idx)
                            ? 'success.light'
                            : rejectedCandidates.has(idx)
                            ? 'error.light'
                            : 'inherit',
                          opacity: rejectedCandidates.has(idx) ? 0.5 : 1,
                        }}
                      >
                        <TableCell>{candidate.left_table_name}</TableCell>
                        <TableCell>{candidate.right_table_name}</TableCell>
                        <TableCell>
                          <Typography variant="caption" fontFamily="monospace">
                            {candidate.join_condition}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip label={candidate.cardinality} size="small" variant="outlined" />
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={`${Math.round(candidate.confidence * 100)}%`}
                            size="small"
                            color={getConfidenceColor(candidate.confidence)}
                          />
                        </TableCell>
                        <TableCell>
                          {candidate.lookup_candidate && (
                            <Chip label="Lookup" size="small" color="info" variant="outlined" />
                          )}
                        </TableCell>
                        <TableCell align="center">
                          <Stack direction="row" spacing={0.5} justifyContent="center">
                            <Tooltip title="Accept">
                              <IconButton
                                size="small"
                                color={acceptedCandidates.has(idx) ? 'success' : 'default'}
                                onClick={() => toggleCandidate(idx, true)}
                              >
                                <Check />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Reject">
                              <IconButton
                                size="small"
                                color={rejectedCandidates.has(idx) ? 'error' : 'default'}
                                onClick={() => toggleCandidate(idx, false)}
                              >
                                <Cancel />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Edit">
                              <IconButton size="small" onClick={() => startEdit(idx)}>
                                <Edit />
                              </IconButton>
                            </Tooltip>
                          </Stack>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
            <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
              Accepted: {acceptedCandidates.size} | Rejected: {rejectedCandidates.size}
            </Typography>
          </Box>
        )}

        {/* Step 3: Edit & Confirm */}
        {activeStep === 2 && (
          <Box>
            <Typography variant="subtitle1" gutterBottom>
              Review accepted relationships before creating
            </Typography>
            {acceptedCandidates.size === 0 ? (
              <Alert severity="warning">
                No relationships accepted. Go back and accept at least one candidate.
              </Alert>
            ) : (
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Left Table → Right Table</TableCell>
                      <TableCell>Join Condition</TableCell>
                      <TableCell>Cardinality</TableCell>
                      <TableCell>Type</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {candidates
                      .filter((_, idx) => acceptedCandidates.has(idx))
                      .map((c, i) => (
                        <TableRow key={i}>
                          <TableCell>
                            {c.left_table_name} → {c.right_table_name}
                          </TableCell>
                          <TableCell>
                            <Typography variant="caption" fontFamily="monospace">
                              {c.join_condition}
                            </Typography>
                          </TableCell>
                          <TableCell>{c.cardinality}</TableCell>
                          <TableCell>
                            {c.lookup_candidate ? 'Lookup' : 'Regular'}
                          </TableCell>
                        </TableRow>
                      ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Box>
        )}

        {/* Step 4: Results */}
        {activeStep === 3 && (
          <Box textAlign="center" py={4}>
            {loading ? (
              <>
                <CircularProgress />
                <Typography sx={{ mt: 2 }}>Creating relationships...</Typography>
              </>
            ) : (
              <>
                <LinkIcon sx={{ fontSize: 64, color: 'success.main', mb: 2 }} />
                <Typography variant="h5" gutterBottom>
                  {createdCount} Relationship{createdCount !== 1 ? 's' : ''} Created
                </Typography>
                <Typography color="text.secondary">
                  Physical table relationships have been saved. These will be available for BO inheritance.
                </Typography>
              </>
            )}
          </Box>
        )}

        {/* Edit Dialog */}
        <Dialog open={editingIndex !== null} onClose={() => setEditingIndex(null)} maxWidth="sm" fullWidth>
          <DialogTitle>Edit Relationship</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 2 }}>
              <TextField
                fullWidth
                label="Join Condition"
                value={editForm.join_condition || ''}
                onChange={(e) => setEditForm({ ...editForm, join_condition: e.target.value })}
              />
              <FormControl fullWidth>
                <InputLabel>Join Type</InputLabel>
                <Select
                  value={editForm.join_type || 'left'}
                  label="Join Type"
                  onChange={(e) => setEditForm({ ...editForm, join_type: e.target.value })}
                >
                  <MenuItem value="inner">Inner</MenuItem>
                  <MenuItem value="left">Left</MenuItem>
                  <MenuItem value="right">Right</MenuItem>
                  <MenuItem value="full">Full</MenuItem>
                </Select>
              </FormControl>
              <FormControl fullWidth>
                <InputLabel>Cardinality</InputLabel>
                <Select
                  value={editForm.cardinality || 'unknown'}
                  label="Cardinality"
                  onChange={(e) => setEditForm({ ...editForm, cardinality: e.target.value })}
                >
                  <MenuItem value="1:1">1:1 (One-to-One)</MenuItem>
                  <MenuItem value="1:M">1:M (One-to-Many)</MenuItem>
                  <MenuItem value="M:1">M:1 (Many-to-One)</MenuItem>
                  <MenuItem value="M:M">M:M (Many-to-Many)</MenuItem>
                  <MenuItem value="unknown">Unknown</MenuItem>
                </Select>
              </FormControl>
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setEditingIndex(null)}>Cancel</Button>
            <Button variant="contained" onClick={saveEdit}>Save</Button>
          </DialogActions>
        </Dialog>
      </DialogContent>

      <DialogActions>
        <Button onClick={handleBack} disabled={activeStep === 0 || loading}>
          Back
        </Button>
        {activeStep < steps.length - 1 ? (
          <Button
            variant="contained"
            onClick={handleNext}
            disabled={isNextDisabled()}
          >
            {activeStep === 2 ? 'Create Relationships' : 'Next'}
          </Button>
        ) : (
          <Button variant="contained" onClick={handleClose}>
            Done
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

export default TableRelationshipWizard;
