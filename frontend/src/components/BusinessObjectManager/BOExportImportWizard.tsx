import React, { useState, useEffect } from 'react';
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
  Radio,
  RadioGroup,
  FormControlLabel,
  FormControl,
  FormLabel,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Checkbox,
  Alert,
  CircularProgress,
  Divider,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { Close, Download, Upload, CheckCircle, Warning, Error as ErrorIcon } from '@mui/icons-material';

// --- Types ---

type ImportMode = 'dry_run' | 'apply';
type ConflictStrategy = 'create' | 'replace' | 'merge';

interface BO {
  id: string;
  name: string;
  description: string;
}

interface ImportResult {
  mode: ImportMode;
  summary: {
    nodes_to_create: number;
    nodes_to_update: number;
    nodes_conflicting: number;
    edges_to_create: number;
    edges_to_update: number;
  };
  node_diffs: NodeDiff[];
  errors?: string[];
}

interface NodeDiff {
  node_type: string;
  node_name: string;
  status: 'missing' | 'exists_same' | 'exists_different' | 'conflict';
  diff?: any; 
  errors?: string[];
}

// --- Main Component ---

export const BOExportImportWizard: React.FC<{ open: boolean; onClose: () => void; onComplete?: (boId?: string) => void }> = ({
  open,
  onClose,
  onComplete,
}) => {
  const [activeStep, setActiveStep] = useState(0);
  const [operation, setOperation] = useState<'export' | 'import' | null>(null);
  
  // Export State
  const [selectedBOs, setSelectedBOs] = useState<string[]>([]);
  // const [exportFormat, setExportFormat] = useState<'json' | 'yaml'>('json'); // Default JSON for now
  
  // Import State
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [importMode, setImportMode] = useState<ImportMode>('dry_run');
  const [conflictStrategy, setConflictStrategy] = useState<ConflictStrategy>('create');
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const steps = ['Select Operation', 'Configure', 'Review & Validate', 'Execute'];

  const resetState = () => {
    setActiveStep(0);
    setOperation(null);
    setSelectedBOs([]);
    setUploadedFile(null);
    setImportResult(null);
    setLoading(false);
    setError(null);
  };

  const handleClose = () => {
    onClose();
    setTimeout(resetState, 300);
  };

  const handleNext = async () => {
    if (activeStep === 1 && operation === 'import') {
      // Run Validation / Dry Run
      await runImport('dry_run');
    } else if (activeStep === 2 && operation === 'export') {
        // Just move to execution step (which renders button) or execute immediately?
        // Let's execute on Step 3 -> 4 transition or have a button in Step 4.
        // UX: "Execute" step usually shows "Processing..." or "Ready to Execute".
    } else if (activeStep === 2 && operation === 'import') {
       // Moving to Execute step.
    }
    setActiveStep((prev) => prev + 1);
  };

  const handleBack = () => setActiveStep((prev) => prev - 1);

  const runImport = async (modeOverride?: ImportMode) => {
    if (!uploadedFile) return;
    setLoading(true);
    setError(null);

    const formData = new FormData();
    formData.append('file', uploadedFile);
    
    // Construct Request JSON (ImportRequest) wrapper is handled by backend reading body? 
    // Wait, backend expects JSON body `ImportRequest` which contains `Bundle`.
    // BUT frontend is uploading a FILE.
    // My backend `ImportBO` handler decodes JSON body `ImportRequest`.
    // It does NOT handle multipart/form-data.
    // I need to read the file in Frontend and send it as JSON payload.
    
    try {
        const fileContent = await uploadedFile.text();
        const bundle = JSON.parse(fileContent);

        const payload = {
            mode: modeOverride || importMode,
            conflict_strategy: conflictStrategy,
            bundle: bundle
        };

        const res = await fetch('/api/bo/import', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                // Auth headers assumed handled by proxy/interceptor
            },
            body: JSON.stringify(payload)
        });

        if (!res.ok) throw new Error(await res.text());

        const result: ImportResult = await res.json();
        setImportResult(result);
    } catch (e: any) {
        setError(e.message);
    } finally {
        setLoading(false);
    }
  };

  const executeExport = async () => {
    setLoading(true);
    try {
        const res = await fetch('/api/bo/export/multiple', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ bo_ids: selectedBOs })
        });
        
        if (!res.ok) throw new Error(await res.text());

        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `bo_export_${new Date().toISOString()}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        handleClose();
    } catch (e: any) {
        setError(e.message);
    } finally {
        setLoading(false);
    }
  };

  const executeImportApply = async () => {
    if (!importResult) return;
    // Apply changes
    // We re-run import with 'apply' mode
    await runImport('apply');
    // If successful, close
    if (!error) {
        if (onComplete) onComplete();
        handleClose();
    }
  };


  const isNextDisabled = () => {
    if (activeStep === 0 && !operation) return true;
    if (activeStep === 1 && operation === 'export' && selectedBOs.length === 0) return true;
    if (activeStep === 1 && operation === 'import' && !uploadedFile) return true;
    if (loading) return true;
    return false;
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        BO Export / Import Wizard
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

        {loading && (
            <Box display="flex" justifyContent="center" my={4}>
                <CircularProgress />
            </Box>
        )}

        {error && (
            <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>
        )}

        {!loading && (
            <>
                {/* Step 1: Select Operation */}
                {activeStep === 0 && (
                  <Step1SelectOperation operation={operation} setOperation={setOperation} />
                )}

                {/* Step 2: Configure */}
                {activeStep === 1 && operation === 'export' && (
                  <Step2Export selectedBOs={selectedBOs} setSelectedBOs={setSelectedBOs} />
                )}

                {activeStep === 1 && operation === 'import' && (
                  <Step2Import
                    uploadedFile={uploadedFile}
                    setUploadedFile={setUploadedFile}
                    importMode={importMode} // Actually unused in UI if we force dry-run first?
                    // Strategy is important
                    conflictStrategy={conflictStrategy}
                    setConflictStrategy={setConflictStrategy}
                  />
                )}

                {/* Step 3: Review & Validate */}
                {activeStep === 2 && (
                  <Step3ReviewValidate
                    operation={operation}
                    importResult={importResult}
                  />
                )}

                {/* Step 4: Execute */}
                {activeStep === 3 && (
                    <Box textAlign="center" py={4}>
                         <Typography variant="h6" gutterBottom>
                            Ready to {operation === 'export' ? 'Download' : 'Apply Changes'}
                        </Typography>
                        {operation === 'import' && (
                            <Typography color="textSecondary">
                                This will apply {importResult?.summary.nodes_to_create} additions and {importResult?.summary.nodes_to_update} updates.
                            </Typography>
                        )}
                        <Box mt={3}>
                            {operation === 'export' ? (
                                <Button variant="contained" color="primary" onClick={executeExport} startIcon={<Download />}>
                                    Download Export Bundle
                                </Button>
                            ) : (
                                <Button variant="contained" color="primary" onClick={executeImportApply} startIcon={<Upload />}>
                                    Confirm & Apply
                                </Button>
                            )}
                        </Box>
                    </Box>
                )}
            </>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={handleBack} disabled={activeStep === 0 || loading}>
          Back
        </Button>
        {activeStep < steps.length - 1 && (
            <Button
              onClick={handleNext}
              variant="contained"
              disabled={isNextDisabled()}
            >
              Next
            </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

// --- Sub-components ---

const Step1SelectOperation: React.FC<{
  operation: 'export' | 'import' | null;
  setOperation: (op: 'export' | 'import') => void;
}> = ({ operation, setOperation }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>What would you like to do?</Typography>
      <RadioGroup value={operation} onChange={(e) => setOperation(e.target.value as any)}>
        <FormControlLabel
          value="export"
          control={<Radio />}
          label={
            <Box>
              <Typography variant="subtitle1" fontWeight="bold">Export Business Object(s)</Typography>
              <Typography variant="body2" color="text.secondary">
                Export BO definitions, including terms and calculations, for migration or backup.
              </Typography>
            </Box>
          }
          sx={{ mb: 2, alignItems: 'flex-start' }}
        />
        <FormControlLabel
          value="import"
          control={<Radio />}
          label={
            <Box>
              <Typography variant="subtitle1" fontWeight="bold">Import Business Object(s)</Typography>
              <Typography variant="body2" color="text.secondary">
                Import BO definitions from a JSON bundle. Includes conflict detection and diff review.
              </Typography>
            </Box>
          }
          sx={{ mb: 2, alignItems: 'flex-start' }}
        />
      </RadioGroup>
    </Box>
  );
};

const Step2Export: React.FC<{
  selectedBOs: string[];
  setSelectedBOs: (ids: string[]) => void;
}> = ({ selectedBOs, setSelectedBOs }) => {
  const [bos, setBOs] = useState<BO[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/bo') // Assumption: endpoint exists to list BOs
      .then(res => res.json())
      .then(data => {
          // Normalize data
          const list = Array.isArray(data) ? data : (data.business_objects || []);
          setBOs(list);
      })
      .catch(err => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <CircularProgress size={24} />;

  const handleToggle = (id: string) => {
    if (selectedBOs.includes(id)) {
        setSelectedBOs(selectedBOs.filter(x => x !== id));
    } else {
        setSelectedBOs([...selectedBOs, id]);
    }
  };

  return (
    <Box>
       <Typography variant="h6" gutterBottom>Select Business Objects</Typography>
       <Paper variant="outlined" sx={{ maxHeight: 300, overflow: 'auto' }}>
           <List dense>
               {bos.map(bo => (
                   <ListItem key={bo.id} button onClick={() => handleToggle(bo.id)}>
                       <ListItemIcon>
                           <Checkbox checked={selectedBOs.includes(bo.id)} edge="start" />
                       </ListItemIcon>
                       <ListItemText primary={bo.name} secondary={bo.description} />
                   </ListItem>
               ))}
               {bos.length === 0 && <ListItem><ListItemText primary="No Business Objects found." /></ListItem>}
           </List>
       </Paper>
       <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
           Selected: {selectedBOs.length}
       </Typography>
    </Box>
  );
};

const Step2Import: React.FC<{
  uploadedFile: File | null;
  setUploadedFile: (file: File | null) => void;
  importMode: ImportMode;
  conflictStrategy: ConflictStrategy;
  setConflictStrategy: (s: ConflictStrategy) => void;
}> = ({ uploadedFile, setUploadedFile, conflictStrategy, setConflictStrategy }) => {
  return (
    <Box>
        <Box mb={4}>
            <Typography variant="h6" gutterBottom>1. Upload Bundle</Typography>
            <Button
                variant="outlined"
                component="label"
                startIcon={<Upload />}
            >
                Choose File
                <input
                    type="file"
                    hidden
                    accept=".json"
                    onChange={(e) => setUploadedFile(e.target.files?.[0] || null)}
                />
            </Button>
            {uploadedFile && (
                <Typography variant="body2" sx={{ mt: 1, ml: 1, display: 'inline-block' }}>
                    {uploadedFile.name} ({(uploadedFile.size / 1024).toFixed(1)} KB)
                </Typography>
            )}
        </Box>

        <Box>
            <Typography variant="h6" gutterBottom>2. Conflict Strategy</Typography>
            <FormControl component="fieldset">
                <RadioGroup value={conflictStrategy} onChange={(e) => setConflictStrategy(e.target.value as any)}>
                    <FormControlLabel 
                        value="create" label="Create New (Fail on conflict)" control={<Radio />} 
                    />
                     <FormControlLabel 
                        value="merge" label="Merge (Update existing, add new)" control={<Radio />} 
                    />
                     <FormControlLabel 
                        value="replace" label="Replace (Overwrite existing)" control={<Radio />} 
                    />
                </RadioGroup>
            </FormControl>
        </Box>
    </Box>
  );
};

const Step3ReviewValidate: React.FC<{
    operation: 'export' | 'import' | null;
    importResult: ImportResult | null;
}> = ({ operation, importResult }) => {
    if (operation === 'export') {
        return (
            <Alert severity="info">
                Export bundle prepared. No validation required for export. 
                Click Next to download.
            </Alert>
        );
    }

    if (!importResult) return null;

    const hasErrors = (importResult.errors?.length || 0) > 0;
    const hasConflicts = importResult.summary.nodes_conflicting > 0;

    return (
        <Box>
            <Box mb={2}>
                {hasErrors ? (
                    <Alert severity="error" icon={<ErrorIcon />}>
                        Validation Failed: {importResult.errors?.length} errors found.
                    </Alert>
                ) : hasConflicts ? (
                    <Alert severity="warning" icon={<Warning />}>
                        Conflicts Detected: {importResult.summary.nodes_conflicting} nodes have conflicts.
                    </Alert>
                ) : (
                    <Alert severity="success" icon={<CheckCircle />}>
                        Validation Passed. Ready to import.
                    </Alert>
                )}
            </Box>

            <Typography variant="subtitle2" gutterBottom>Changes Summary</Typography>
            <Box display="flex" gap={2} mb={3}>
                <Paper variant="outlined" sx={{ p: 2, flex: 1, textAlign: 'center' }}>
                    <Typography variant="h4" color="primary">{importResult.summary.nodes_to_create}</Typography>
                    <Typography variant="caption">New Nodes</Typography>
                </Paper>
                <Paper variant="outlined" sx={{ p: 2, flex: 1, textAlign: 'center' }}>
                    <Typography variant="h4" color="secondary">{importResult.summary.nodes_to_update}</Typography>
                    <Typography variant="caption">Updated Nodes</Typography>
                </Paper>
                <Paper variant="outlined" sx={{ p: 2, flex: 1, textAlign: 'center' }}>
                    <Typography variant="h4" color="error">{importResult.summary.nodes_conflicting}</Typography>
                    <Typography variant="caption">Conflicts</Typography>
                </Paper>
            </Box>

            <Typography variant="subtitle2" gutterBottom>Detailed Diff</Typography>
            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 300 }}>
                <Table size="small" stickyHeader>
                    <TableHead>
                        <TableRow>
                            <TableCell>Type</TableCell>
                            <TableCell>Name</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell>Details</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {importResult.node_diffs.map((diff, idx) => (
                            <TableRow key={idx}>
                                <TableCell>{diff.node_type}</TableCell>
                                <TableCell>{diff.node_name}</TableCell>
                                <TableCell>
                                    <StatusBadge status={diff.status} />
                                </TableCell>
                                <TableCell>
                                    {diff.status === 'exists_different' && diff.diff ? (
                                        <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                                            {/* Simplification: Just count changes */}
                                            {Object.keys(diff.diff.properties?.changed || {}).length} prop changes
                                        </Typography>
                                    ) : '-'}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
    let color = 'default';
    let label = status;

    switch (status) {
        case 'missing': color = 'success.main'; label = 'New'; break;
        case 'exists_same': color = 'text.secondary'; label = 'Unchanged'; break;
        case 'exists_different': color = 'warning.main'; label = 'Changed'; break;
        case 'conflict': color = 'error.main'; label = 'Conflict'; break;
    }

    return (
        <Typography variant="caption" sx={{ color, fontWeight: 'bold' }}>
            {label}
        </Typography>
    );
};
