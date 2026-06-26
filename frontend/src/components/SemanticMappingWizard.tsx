import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Stepper,
  Step,
  StepLabel,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Checkbox,
  Chip,
  CircularProgress,
  Alert,
  TextField,
  IconButton,
  LinearProgress,
} from '@mui/material';
import { gql, useMutation } from '@apollo/client';
import SearchIcon from '@mui/icons-material/Search';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import PendingIcon from '@mui/icons-material/Pending';
import EditIcon from '@mui/icons-material/Edit';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import { devDebug } from '../utils/devLogger';

const STEPS = ['Scan & Expand', 'Generate Mappings', 'Review Auto-Created', 'Approve Pending', 'Summary'];

interface GeneratedMapping {
  column_id: string;
  column_name: string;
  table_name?: string;
  expanded_column_name: string;
  suggested_semantic_term: string;
  suggested_business_term: string;
  semantic_type?: string; // dimension, measure, time_dimension
  confidence: number;
  reasoning: string;
  will_auto_create: boolean;
  needs_approval: boolean;
}

interface PendingMapping {
  id: string;
  column_name: string;
  expanded_column_name: string;
  suggested_semantic_term: string;
  suggested_business_term: string;
  confidence: number;
  reasoning: string;
}

interface SemanticMappingWizardProps {
  tenantId: string;
  datasourceId: string;
  onClose: () => void;
  onMappingsApplied?: () => void; // Callback to refresh sidebar/catalog after mappings are applied
}

const LOG_TERM_FEEDBACK = gql`
  mutation LogTermAISuggestionFeedback($input: LogTermFeedbackInput!) {
    logTermAISuggestionFeedback(input: $input)
  }
`;

export const SemanticMappingWizard: React.FC<SemanticMappingWizardProps> = ({
  tenantId,
  datasourceId,
  onClose,
  onMappingsApplied,
}) => {

  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [progressMessage, setProgressMessage] = useState<string>('');

  // Manual approval and rejection state
  const [manualApprovals, setManualApprovals] = useState<Set<string>>(new Set());
  const [rejectedMappings, setRejectedMappings] = useState<Set<string>>(new Set());

  // Simulated progress simulation for better UX
  const [progressSteps, setProgressSteps] = useState([
    { label: "Scanning database columns", status: 'waiting' },
    { label: "Identifying abbreviations", status: 'waiting' },
    { label: "Expanding table names", status: 'waiting' },
    { label: "Consulting AI knowledge base", status: 'waiting' },
    { label: "Generating semantic terms", status: 'waiting' },
    { label: "Calculating confidence scores", status: 'waiting' },
    { label: "Finalizing suggestions", status: 'waiting' }
  ]);

  // Feedback logging mutation
  const [logTermFeedback] = useMutation(LOG_TERM_FEEDBACK, {
    onError: (e) => console.error('Failed to log feedback:', e)
  });

  useEffect(() => {
    let interval: NodeJS.Timeout;
    if (loading) {
      let stepIndex = 0;
      
      // Reset steps
      setProgressSteps(prev => prev.map(s => ({ ...s, status: 'waiting' })));
      
      const updateSteps = () => {
        setProgressSteps(prev => prev.map((s, i) => {
          if (i < stepIndex) return { ...s, status: 'completed' };
          if (i === stepIndex) return { ...s, status: 'active' };
          return { ...s, status: 'waiting' };
        }));
        
        stepIndex++;
        if (stepIndex >= 7) { 
           // keep the last one active or completed until loading finishes
           // actually let's just loop the last few if it takes longer, or just hold at verifying
        }
      };

      updateSteps(); // Initial state
      interval = setInterval(() => {
        if (stepIndex < 7) {
            updateSteps();
        }
      }, 800); // 800ms per step for a ~5-6s total animation which feels "fast but real"
    }
    return () => clearInterval(interval);
  }, [loading]);
  
  // Step 1 & 2: Generated mappings
  const [mappings, setMappings] = useState<GeneratedMapping[]>([]);
  const [totalColumns, setTotalColumns] = useState(0);
  
  // Step 3: Auto-created results
  const [autoCreatedCount, setAutoCreatedCount] = useState(0);
  const [autoCreatedMappings, setAutoCreatedMappings] = useState<GeneratedMapping[]>([]);
  
  // Step 4: Pending approvals
  const [pendingMappings, setPendingMappings] = useState<PendingMapping[]>([]);
  const [selectedPending, setSelectedPending] = useState<Set<string>>(new Set());
  
  // Step 5: Summary
  const [summary, setSummary] = useState({
    auto_created: 0,
    pending_approval: 0,
    skipped: 0,
    errors: 0,
  });

  // Created mappings to display
  const [createdMappings, setCreatedMappings] = useState<any[]>([]);

  // Sort and search state for mappings table
  const [mappingsSortField, setMappingsSortField] = useState<'table' | 'column' | 'expanded' | 'term' | 'confidence'>('column');
  const [mappingsSortOrder, setMappingsSortOrder] = useState<'asc' | 'desc'>('asc');
  const [mappingsSearchTerm, setMappingsSearchTerm] = useState('');

  const handleGenerateMappings = async () => {
    if (!tenantId || !datasourceId) {
      setError("Missing tenant or datasource context. Please select a tenant and datasource.");
      return;
    }

    setLoading(true);
    setError(null);
    
    // Debug: log what we're sending
    devDebug('[Wizard] Sending request with:', { tenant_id: tenantId, tenant_instance_id: datasourceId });
    
    try {
      const response = await fetch('/api/semantic-mapping/wizard/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenant_id: tenantId, tenant_instance_id: datasourceId }),
      });
      
      if (!response.ok) {
        const text = await response.text();
        console.error('[Wizard] API error:', text);
        throw new Error(`Failed to generate mappings: ${text}`);
      }
      
      const data = await response.json();
      devDebug('[Wizard] Success! Received:', data);
      setMappings(data.mappings || []);
      setTotalColumns(data.total_columns || 0);
      setMappingsSearchTerm(''); // Reset search
      setMappingsSortField('column'); // Reset sort
      setActiveStep(1);
    } catch (err: any) {
      console.error('[Wizard] Error:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleRejectMapping = async (mapping: GeneratedMapping) => {
    try {
      // Call backend to ignore this column-term pair
      await fetch('/api/semantic-mappings/ignore', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenant_id: tenantId,
          tenant_instance_id: datasourceId,
          column_id: mapping.column_id,
          semantic_term: mapping.suggested_semantic_term,
          reason: 'user_rejected'
        })
      });
      
      // Remove from local state
      setRejectedMappings(prev => new Set(prev).add(mapping.column_id));
      
      // Log rejection feedback
      logTermFeedback({
        variables: {
          input: {
            tenantId,
            datasourceId,
            termId: "", // No term created yet
            nodeId: mapping.column_id, // Use column ID as related node
            suggestionId: mapping.column_id + "_suggestion", // Synthesis ID since we don't have explicit one
            action: 'rejected',
            reason: 'manual_rejection',
            features: {
                semantic_term: mapping.suggested_semantic_term,
                business_term: mapping.suggested_business_term,
                column_name: mapping.column_name,
                confidence: mapping.confidence
            }
          }
        }
      });
    } catch (error) {
      console.error('Failed to reject mapping:', error);
    }
  };

  const handleApplyMappings = async () => {
    setLoading(true);
    setError(null);
    
    // Separate mappings by decision
    const approvedMappings = mappings.filter(m => 
      m.confidence >= 0.85 || manualApprovals.has(m.column_id)
    );
    
    const rejectedMappingsList = mappings.filter(m => 
      rejectedMappings.has(m.column_id)
    );
    
    const noDecisionMappings = mappings.filter(m => 
      m.confidence < 0.85 && 
      !manualApprovals.has(m.column_id) && 
      !rejectedMappings.has(m.column_id)
    );
    
    // Adjust confidence for manually approved items
    const adjustedMappings = [
      ...approvedMappings.map(m => ({ ...m, confidence: 0.85 })),
      ...noDecisionMappings
    ];
    
    try {
      const response = await fetch('/api/semantic-mapping/wizard/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenant_id: tenantId,
          tenant_instance_id: datasourceId,
          auto_create_threshold: 0.85,
          approval_threshold: 0.60,
          mappings: adjustedMappings,
        }),
      });
      
      if (!response.ok) throw new Error('Failed to apply mappings');
      
      const data = await response.json();
      setSummary(data);
      setAutoCreatedCount(data.auto_created);
      setAutoCreatedMappings(adjustedMappings.filter(m => m.confidence >= 0.85));
      
      // Fetch created mappings to show results
      await fetchCreatedMappings();
      
      // Fetch pending approvals
      await fetchPendingApprovals();
      
      // Trigger sidebar refresh if callback provided
      if (onMappingsApplied) {
        onMappingsApplied();
      }
      
      setActiveStep(2);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const fetchPendingApprovals = async () => {
    try {
      const response = await fetch(
        `/api/semantic-mapping/wizard/pending?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`
      );
      
      if (!response.ok) throw new Error('Failed to fetch pending approvals');
      
      const data = await response.json();
      setPendingMappings(data.pending_approvals || []);
    } catch (err: any) {
      console.error('Failed to fetch pending approvals:', err);
    }
  };

  const fetchCreatedMappings = async () => {
    try {
      const response = await fetch(
        `/api/semantic-mapping/wizard/created?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&limit=20`
      );
      
      if (response.ok) {
        const data = await response.json();
        setCreatedMappings(data.mappings || []);
      }
    } catch (err: any) {
      console.error('Failed to fetch created mappings:', err);
    }
  };

  const handleApprovePending = async (mappingId: string, approved: boolean) => {
    try {
      const response = await fetch(`/api/semantic-mapping/wizard/approve/${mappingId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ approved, user_id: 'current-user' }),
      });
      
      if (!response.ok) throw new Error('Failed to approve/reject mapping');

      const data = await response.json();
      const termId = data.term_id || "";

      // Log feedback
      logTermFeedback({
        variables: {
          input: {
            tenantId,
            datasourceId,
            termId: termId, // Use returned term ID (empty if rejected)
             // Use mapping ID as weak node_id reference, though it's the PENDING mapping ID
            suggestionId: mappingId, 
            action: approved ? 'approved' : 'rejected',
            features: {
                mapping_id: mappingId,
                approved: approved
            }
          }
        }
      });
      
      // Remove from pending list
      setPendingMappings(prev => prev.filter(m => m.id !== mappingId));
      setSelectedPending(prev => {
        const newSet = new Set(prev);
        newSet.delete(mappingId);
        return newSet;
      });
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleBulkApprove = async () => {
    setLoading(true);
    for (const id of selectedPending) {
      await handleApprovePending(id, true);
    }
    setLoading(false);
    setActiveStep(4);
  };

  const getConfidenceBadge = (confidence: number) => {
    if (confidence >= 0.85) {
      return <Chip label={`${(confidence * 100).toFixed(0)}%`} color="success" size="small" />;
    } else if (confidence >= 0.60) {
      return <Chip label={`${(confidence * 100).toFixed(0)}%`} color="warning" size="small" />;
    } else {
      return <Chip label={`${(confidence * 100).toFixed(0)}%`} color="error" size="small" />;
    }
  };

  const renderScanStep = () => (
    <Box sx={{ textAlign: 'center', py: 4, maxWidth: 600, mx: 'auto' }}>
      <SearchIcon sx={{ 
        fontSize: 60, 
        color: loading ? 'primary.main' : 'text.secondary', 
        mb: 2,
        animation: loading ? 'spin 2s linear infinite' : 'none',
        '@keyframes spin': {
          '0%': { transform: 'rotate(0deg)' },
          '100%': { transform: 'rotate(360deg)' },
        },
      }} />
      <Typography variant="h6" gutterBottom>
        Scan Database & Generate Semantic Mappings
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        This will scan all unmapped columns, expand abbreviations, and use AI to suggest semantic terms.
      </Typography>
      
      <Box sx={{ width: '100%', mb: 3, minHeight: '60px', bgcolor: 'grey.50', p: 2, borderRadius: 1, border: '1px dashed', borderColor: 'grey.300' }}>
        {error ? (
          <Alert severity="error">{error}</Alert>
        ) : loading ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Typography variant="body2" fontWeight="bold" color="primary" sx={{ mb: 1 }}>
              Running Analysis...
            </Typography>
            {progressSteps.map((step, index) => (
              <Box key={index} sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                 {step.status === 'completed' && <CheckIcon color="success" fontSize="small" />}
                 {step.status === 'active' && <CircularProgress size={14} thickness={5} />}
                 {step.status === 'waiting' && <Box sx={{ width: 14, height: 14, borderRadius: '50%', border: '1px solid', borderColor: 'text.disabled' }} />}
                 
                 <Typography 
                   variant="caption" 
                   sx={{ 
                     color: step.status === 'active' ? 'text.primary' : step.status === 'completed' ? 'text.secondary' : 'text.disabled',
                     fontWeight: step.status === 'active' ? 600 : 400
                   }}
                 >
                   {step.label}
                 </Typography>
              </Box>
            ))}
          </Box>
        ) : (
          <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic', textAlign: 'center', py: 2 }}>
            Ready to scan. Click the button below to start.
          </Typography>
        )}
      </Box>

      <Button 
        variant="contained" 
        onClick={handleGenerateMappings} 
        disabled={loading}
        size="large"
      >
        {loading ? 'Scanning & Generating...' : 'Start Scan'}
      </Button>
    </Box>
  );

  const renderMappingsReview = () => {
    // Filter out rejected mappings
    const visibleMappings = mappings.filter(m => !rejectedMappings.has(m.column_id));
    
    // Apply search filter
    const searchLower = mappingsSearchTerm.toLowerCase();
    const searchedMappings = visibleMappings.filter(m =>
      m.table_name?.toLowerCase().includes(searchLower) ||
      m.column_name?.toLowerCase().includes(searchLower) ||
      m.expanded_column_name?.toLowerCase().includes(searchLower) ||
      m.suggested_semantic_term?.toLowerCase().includes(searchLower)
    );
    
    // Apply sorting
    const sortedMappings = [...searchedMappings].sort((a, b) => {
      let aVal: string | number = '';
      let bVal: string | number = '';
      
      switch (mappingsSortField) {
        case 'table':
          aVal = (a.table_name || '').toLowerCase();
          bVal = (b.table_name || '').toLowerCase();
          break;
        case 'column':
          aVal = (a.column_name || '').toLowerCase();
          bVal = (b.column_name || '').toLowerCase();
          break;
        case 'expanded':
          aVal = (a.expanded_column_name || '').toLowerCase();
          bVal = (b.expanded_column_name || '').toLowerCase();
          break;
        case 'term':
          aVal = (a.suggested_semantic_term || '').toLowerCase();
          bVal = (b.suggested_semantic_term || '').toLowerCase();
          break;
        case 'confidence':
          aVal = a.confidence || 0;
          bVal = b.confidence || 0;
          break;
      }
      
      if (typeof aVal === 'string' && typeof bVal === 'string') {
        return mappingsSortOrder === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      } else {
        return mappingsSortOrder === 'asc' ? (aVal > bVal ? 1 : -1) : (bVal > aVal ? 1 : -1);
      }
    });
    
    const handleSortClick = (field: 'table' | 'column' | 'expanded' | 'term' | 'confidence') => {
      if (mappingsSortField === field) {
        setMappingsSortOrder(mappingsSortOrder === 'asc' ? 'desc' : 'asc');
      } else {
        setMappingsSortField(field);
        setMappingsSortOrder('asc');
      }
    };
    
    return (
      <Box>
        <Typography variant="h6" gutterBottom>
          Generated Mappings ({sortedMappings.length} columns)
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Review the AI-generated semantic term suggestions. Click column headers to sort, use search to filter.
        </Typography>
        
        {/* Typeahead Search */}
        <TextField
          placeholder="Search by table, column, expanded name, or semantic term..."
          size="small"
          fullWidth
          value={mappingsSearchTerm}
          onChange={(e) => setMappingsSearchTerm(e.target.value)}
          sx={{ mb: 2 }}
          InputProps={{
            startAdornment: (
              <Box sx={{ mr: 1, display: 'flex', alignItems: 'center' }}>
                🔍
              </Box>
            ),
          }}
        />

        <TableContainer component={Paper} sx={{ maxHeight: 600 }}>
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow sx={{ backgroundColor: 'action.hover' }}>
                <TableCell 
                  onClick={() => handleSortClick('table')}
                  sx={{ cursor: 'pointer', fontWeight: 'bold', userSelect: 'none', '&:hover': { backgroundColor: 'action.selected' } }}
                >
                  Table {mappingsSortField === 'table' && (mappingsSortOrder === 'asc' ? '▲' : '▼')}
                </TableCell>
                <TableCell 
                  onClick={() => handleSortClick('column')}
                  sx={{ cursor: 'pointer', fontWeight: 'bold', userSelect: 'none', '&:hover': { backgroundColor: 'action.selected' } }}
                >
                  Column {mappingsSortField === 'column' && (mappingsSortOrder === 'asc' ? '▲' : '▼')}
                </TableCell>
                <TableCell 
                  onClick={() => handleSortClick('expanded')}
                  sx={{ cursor: 'pointer', fontWeight: 'bold', userSelect: 'none', '&:hover': { backgroundColor: 'action.selected' } }}
                >
                  Expanded {mappingsSortField === 'expanded' && (mappingsSortOrder === 'asc' ? '▲' : '▼')}
                </TableCell>
                <TableCell 
                  onClick={() => handleSortClick('term')}
                  sx={{ cursor: 'pointer', fontWeight: 'bold', userSelect: 'none', '&:hover': { backgroundColor: 'action.selected' } }}
                >
                  Semantic Term {mappingsSortField === 'term' && (mappingsSortOrder === 'asc' ? '▲' : '▼')}
                </TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Type</TableCell>
                <TableCell 
                  onClick={() => handleSortClick('confidence')}
                  sx={{ cursor: 'pointer', fontWeight: 'bold', userSelect: 'none', '&:hover': { backgroundColor: 'action.selected' } }}
                  align="center"
                >
                  Confidence {mappingsSortField === 'confidence' && (mappingsSortOrder === 'asc' ? '▲' : '▼')}
                </TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Decision</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {sortedMappings.map((mapping) => {
                // Determine current decision state
                let currentDecision = 'none';
                if (mapping.confidence >= 0.85) {
                  currentDecision = 'approve'; // Auto-approved
                } else if (manualApprovals.has(mapping.column_id)) {
                  currentDecision = 'approve';
                } else if (rejectedMappings.has(mapping.column_id)) {
                  currentDecision = 'reject';
                }
                
                return (
                  <TableRow key={mapping.column_id} hover>
                    <TableCell sx={{ fontWeight: 500, color: 'text.secondary' }}>
                      {mapping.table_name || 'N/A'}
                    </TableCell>
                    <TableCell>{mapping.column_name}</TableCell>
                    <TableCell>{mapping.expanded_column_name}</TableCell>
                    <TableCell>
                      <TextField
                        size="small"
                        variant="standard"
                        value={mapping.suggested_semantic_term}
                        onChange={(e) => {
                          const newValue = e.target.value.toUpperCase(); // Force uppercase
                          setMappings(prev => prev.map(m => 
                            m.column_id === mapping.column_id 
                              ? { ...m, suggested_semantic_term: newValue, confidence: 1.0 } 
                              : m
                          ));
                          // Auto-approve
                          setManualApprovals(prev => new Set(prev).add(mapping.column_id));
                          setRejectedMappings(prev => {
                            const next = new Set(prev);
                            next.delete(mapping.column_id);
                            return next;
                          });
                        }}
                        sx={{ minWidth: 200 }}
                      />
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={mapping.semantic_type || 'dimension'}
                        size="small"
                        color={mapping.semantic_type === 'measure' ? 'primary' : mapping.semantic_type === 'time_dimension' ? 'secondary' : 'default'}
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell align="center">{getConfidenceBadge(mapping.confidence)}</TableCell>
                    <TableCell>
                      {mapping.confidence >= 0.85 ? (
                        <Chip label="Auto-Approved" color="success" size="small" icon={<CheckCircleIcon />} />
                      ) : (
                        <Box sx={{ display: 'flex', gap: 0.5 }}>
                          <Button
                            size="small"
                            variant={currentDecision === 'approve' ? 'contained' : 'outlined'}
                            color="success"
                            onClick={() => {
                              const newApprovals = new Set(manualApprovals);
                              const newRejections = new Set(rejectedMappings);
                              newApprovals.add(mapping.column_id);
                              newRejections.delete(mapping.column_id);
                              setManualApprovals(newApprovals);
                              setRejectedMappings(newRejections);
                            }}
                            sx={{ minWidth: '70px' }}
                          >
                            Approve
                          </Button>
                          <Button
                            size="small"
                            variant={currentDecision === 'reject' ? 'contained' : 'outlined'}
                            color="error"
                            onClick={() => {
                              handleRejectMapping(mapping);
                              const newApprovals = new Set(manualApprovals);
                              newApprovals.delete(mapping.column_id);
                              setManualApprovals(newApprovals);
                            }}
                            sx={{ minWidth: '70px' }}
                          >
                            Reject
                          </Button>
                          <Button
                            size="small"
                            variant={currentDecision === 'none' ? 'contained' : 'outlined'}
                            color="inherit"
                            onClick={() => {
                              const newApprovals = new Set(manualApprovals);
                              const newRejections = new Set(rejectedMappings);
                              newApprovals.delete(mapping.column_id);
                              newRejections.delete(mapping.column_id);
                              setManualApprovals(newApprovals);
                              setRejectedMappings(newRejections);
                            }}
                            sx={{ minWidth: '70px' }}
                          >
                            None
                          </Button>
                        </Box>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
        
        {sortedMappings.length === 0 && mappingsSearchTerm && (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 2, textAlign: 'center' }}>
            No mappings match your search.
          </Typography>
        )}
        
        <Box sx={{ mt: 3, display: 'flex', justifyContent: 'space-between' }}>
          <Button onClick={() => setActiveStep(0)}>Back</Button>
          <Button 
            variant="contained" 
            onClick={handleApplyMappings}
            disabled={loading}
          >
            {loading ? 'Applying...' : 'Apply Mappings'}
          </Button>
        </Box>
      </Box>
    );
  };

  const renderAutoCreatedReview = () => (
    <Box>
      <Typography variant="h6" gutterBottom>
        Auto-Created Semantic Terms ({autoCreatedCount})
      </Typography>
      <Alert severity="success" sx={{ mb: 2 }}>
        {autoCreatedCount} semantic terms were automatically created with high confidence (≥85%).
      </Alert>
      
      <TableContainer component={Paper} sx={{ maxHeight: 300, mb: 3 }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Column</TableCell>
              <TableCell>Semantic Term</TableCell>
              <TableCell>Confidence</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {autoCreatedMappings.map((mapping) => (
              <TableRow key={mapping.column_id}>
                <TableCell>{mapping.column_name}</TableCell>
                <TableCell>{mapping.suggested_semantic_term}</TableCell>
                <TableCell>{getConfidenceBadge(mapping.confidence)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
        {pendingMappings.length > 0 ? (
          <Button variant="contained" onClick={() => setActiveStep(3)}>
            Review Pending Approvals ({pendingMappings.length})
          </Button>
        ) : (
          <Button variant="contained" color="success" onClick={() => setActiveStep(4)}>
            Finish & View Summary
          </Button>
        )}
      </Box>
    </Box>
  );

  const renderPendingApprovals = () => (
    <Box>
      <Typography variant="h6" gutterBottom>
        Pending Approvals ({pendingMappings.length})
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        These mappings have medium confidence (60-84%) and require your approval.
      </Typography>
      
      <TableContainer component={Paper} sx={{ maxHeight: 400 }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell padding="checkbox">
                <Checkbox
                  checked={selectedPending.size === pendingMappings.length && pendingMappings.length > 0}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedPending(new Set(pendingMappings.map(m => m.id)));
                    } else {
                      setSelectedPending(new Set());
                    }
                  }}
                />
              </TableCell>
              <TableCell>Column</TableCell>
              <TableCell>Semantic Term</TableCell>
              <TableCell>Confidence</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {pendingMappings.map((mapping) => (
              <TableRow key={mapping.id}>
                <TableCell padding="checkbox">
                  <Checkbox
                    checked={selectedPending.has(mapping.id)}
                    onChange={(e) => {
                      const newSet = new Set(selectedPending);
                      if (e.target.checked) {
                        newSet.add(mapping.id);
                      } else {
                        newSet.delete(mapping.id);
                      }
                      setSelectedPending(newSet);
                    }}
                  />
                </TableCell>
                <TableCell>{mapping.column_name}</TableCell>
                <TableCell>{mapping.suggested_semantic_term}</TableCell>
                <TableCell>{getConfidenceBadge(mapping.confidence)}</TableCell>
                <TableCell>
                  <IconButton 
                    size="small" 
                    color="success"
                    onClick={() => handleApprovePending(mapping.id, true)}
                  >
                    <CheckIcon />
                  </IconButton>
                  <IconButton 
                    size="small" 
                    color="error"
                    onClick={() => handleApprovePending(mapping.id, false)}
                  >
                    <CloseIcon />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      
      <Box sx={{ mt: 3, display: 'flex', justifyContent: 'space-between' }}>
        <Button onClick={() => setActiveStep(2)}>Back</Button>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button 
            onClick={onClose}
            color="secondary"
          >
            Save Pending & Close
          </Button>
          <Button 
            variant="contained" 
            onClick={handleBulkApprove}
            disabled={selectedPending.size === 0 || loading}
          >
            Approve Selected ({selectedPending.size})
          </Button>
        </Box>
      </Box>
    </Box>
  );

  const renderSummary = () => (
    <Box sx={{ textAlign: 'center', py: 4 }}>
      <CheckCircleIcon sx={{ fontSize: 60, color: 'success.main', mb: 2 }} />
      <Typography variant="h6" gutterBottom>
        Semantic Mapping Complete!
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        All changes have been saved. You can review pending items in the main grid.
      </Typography>
      
      <Box sx={{ mt: 3, display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2, maxWidth: 400, mx: 'auto' }}>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h4" color="success.main">{summary.auto_created}</Typography>
          <Typography variant="body2">Auto-Created</Typography>
        </Paper>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h4" color="warning.main">{summary.pending_approval}</Typography>
          <Typography variant="body2">Pending</Typography>
        </Paper>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h4" color="text.secondary">{summary.skipped}</Typography>
          <Typography variant="body2">Skipped</Typography>
        </Paper>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h4" color="error.main">{summary.errors}</Typography>
          <Typography variant="body2">Errors</Typography>
        </Paper>
      </Box>

      {createdMappings.length > 0 && (
        <Box sx={{ mt: 4, maxWidth: 800, mx: 'auto' }}>
          <Typography variant="h6" gutterBottom>Created Mappings (Latest {createdMappings.length})</Typography>
          <TableContainer component={Paper}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Column</TableCell>
                  <TableCell>Semantic Term</TableCell>
                  <TableCell>Business Term</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {createdMappings.map((m, idx) => (
                  <TableRow key={idx}>
                    <TableCell>{m.column_name}</TableCell>
                    <TableCell>{m.semantic_term}</TableCell>
                    <TableCell>{m.business_term || '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}
      
      <Button variant="contained" onClick={onClose} sx={{ mt: 4 }}>
        Close Wizard
      </Button>
    </Box>
  );

  const renderStepContent = () => {
    switch (activeStep) {
      case 0:
        return renderScanStep();
      case 1:
        return renderMappingsReview();
      case 2:
        return renderAutoCreatedReview();
      case 3:
        return renderPendingApprovals();
      case 4:
        return renderSummary();
      default:
        return null;
    }
  };

  return (
    <Box sx={{ width: '100%', p: 2, position: 'relative' }}>
      <IconButton 
        onClick={onClose} 
        sx={{ position: 'absolute', top: 0, right: 0, zIndex: 10 }}
        aria-label="close wizard"
      >
        <CloseIcon />
      </IconButton>

      <Stepper activeStep={activeStep} alternativeLabel sx={{ mb: 4, mt: 2 }}>
        {STEPS.map((label, index) => (
          <Step key={label} completed={index < activeStep}>
            <StepLabel 
              sx={{
                '& .MuiStepLabel-label': {
                  color: index === activeStep ? 'primary.main' : index < activeStep ? 'success.main' : 'text.secondary',
                  fontWeight: index === activeStep ? 'bold' : 'normal',
                },
                '& .MuiStepIcon-root': {
                  color: index < activeStep ? 'success.main' : undefined,
                  '&.Mui-active': {
                    color: 'primary.main',
                    animation: 'pulse 1.5s ease-in-out infinite',
                  },
                },
                '@keyframes pulse': {
                  '0%, 100%': { transform: 'scale(1)' },
                  '50%': { transform: 'scale(1.1)' },
                },
              }}
            >
              {label}
            </StepLabel>
          </Step>
        ))}
      </Stepper>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Paper sx={{ p: 3, minHeight: 400 }}>
        {renderStepContent()}
      </Paper>
    </Box>
  );
};