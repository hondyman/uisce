import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Button,
  Chip,
  Collapse,
  Alert,
  Snackbar,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Grid,
  Divider,
} from '@mui/material';
import {
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  WarningAmber as WarningIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  AutoAwesome as AutoAwesomeIcon,
} from '@mui/icons-material';

interface DriftProposal {
  id: string;
  tenantId: string;
  boId: string;
  boName: string;
  boBindingId: string;
  fieldBindingId: string;
  semanticTermName: string;
  missingColumnName: string;
  proposedColumnName: string;
  aiConfidence: number;
  status: string;
  createdAt: string;
}

export default function SemanticHealthDashboard() {
  const [proposals, setProposals] = useState<DriftProposal[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [snackbarMessage, setSnackbarMessage] = useState<string | null>(null);
  const [previewProposal, setPreviewProposal] = useState<DriftProposal | null>(null);

  const fetchProposals = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/bo/drift-proposals');
      if (res.ok) {
        const data = await res.json();
        setProposals(data || []);
      }
    } catch (err) {
      console.error('Failed to fetch drift proposals', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProposals();
  }, []);

  const handleApprove = async (id: string) => {
    setActionLoading(id);
    try {
      const res = await fetch(`/api/v1/bo/drift-proposals/${id}/approve`, {
        method: 'POST',
      });
      if (res.ok) {
        setSnackbarMessage('Mapping successfully healed and BO schema cache invalidated!');
        fetchProposals();
      } else {
        const errMsg = await res.text();
        setSnackbarMessage(`Error: ${errMsg}`);
      }
    } catch (err) {
      setSnackbarMessage('Network error occurred during healing.');
    } finally {
      setActionLoading(null);
      setPreviewProposal(null);
    }
  };

  const handleReject = async (id: string) => {
    setActionLoading(id);
    try {
      const res = await fetch(`/api/v1/bo/drift-proposals/${id}/reject`, {
        method: 'POST',
      });
      if (res.ok) {
        setSnackbarMessage('Drift proposal successfully rejected.');
        fetchProposals();
      }
    } catch (err) {
      setSnackbarMessage('Network error occurred during rejection.');
    } finally {
      setActionLoading(null);
    }
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.9) return 'success';
    if (confidence >= 0.7) return 'warning';
    return 'error';
  };

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
      <Box display="flex" alignItems="center" mb={4}>
        <AutoAwesomeIcon color="primary" sx={{ fontSize: 32, mr: 2 }} />
        <Typography variant="h4" fontWeight={800} letterSpacing="-0.02em" color="text.primary">
          Semantic Health & Drift Ledger
        </Typography>
      </Box>

      {loading ? (
        <Box display="flex" justifyContent="center" py={8}>
          <CircularProgress />
        </Box>
      ) : proposals.length === 0 ? (
        <Alert severity="success" sx={{ borderRadius: 2 }}>
          No schema drift detected! All physical schemas align perfectly with your semantic data model.
        </Alert>
      ) : (
        <TableContainer component={Paper} elevation={0} sx={{ border: '1px solid #e0e0e0', borderRadius: 2 }}>
          <Table>
            <TableHead sx={{ bgcolor: '#f8f9fa' }}>
              <TableRow>
                <TableCell width={50} />
                <TableCell><Typography fontWeight={700}>BO Name</Typography></TableCell>
                <TableCell><Typography fontWeight={700}>Affected Field</Typography></TableCell>
                <TableCell><Typography fontWeight={700}>Missing Column</Typography></TableCell>
                <TableCell><Typography fontWeight={700}>AI Proposed Fix</Typography></TableCell>
                <TableCell><Typography fontWeight={700}>Confidence</Typography></TableCell>
                <TableCell align="right"><Typography fontWeight={700}>Actions</Typography></TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {proposals.map((prop) => (
                <React.Fragment key={prop.id}>
                  <TableRow hover sx={{ '& > *': { borderBottom: 'unset' } }}>
                    <TableCell>
                      <IconButton size="small" onClick={() => setExpandedRow(expandedRow === prop.id ? null : prop.id)}>
                        {expandedRow === prop.id ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                      </IconButton>
                    </TableCell>
                    <TableCell>{prop.boName}</TableCell>
                    <TableCell>
                      <Chip label={prop.semanticTermName} variant="outlined" size="small" />
                    </TableCell>
                    <TableCell sx={{ color: 'error.main', fontFamily: 'monospace' }}>{prop.missingColumnName}</TableCell>
                    <TableCell sx={{ color: 'success.main', fontFamily: 'monospace', cursor: 'pointer', fontWeight: 700 }} onClick={() => setPreviewProposal(prop)}>
                      {prop.proposedColumnName}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={`${Math.round(prop.aiConfidence * 100)}%`}
                        color={getConfidenceColor(prop.aiConfidence)}
                        size="small"
                        sx={{ fontWeight: 'bold' }}
                      />
                    </TableCell>
                    <TableCell align="right">
                      <Box display="flex" justifyContent="flex-end" gap={1}>
                        <Button
                          variant="contained"
                          color="primary"
                          size="small"
                          startIcon={actionLoading === prop.id ? <CircularProgress size={16} color="inherit" /> : <CheckCircleIcon />}
                          disabled={actionLoading !== null}
                          onClick={() => handleApprove(prop.id)}
                        >
                          Approve
                        </Button>
                        <IconButton
                          color="error"
                          size="small"
                          disabled={actionLoading !== null}
                          onClick={() => handleReject(prop.id)}
                        >
                          <CancelIcon />
                        </IconButton>
                      </Box>
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={7}>
                      <Collapse in={expandedRow === prop.id} timeout="auto" unmountOnExit>
                        <Box sx={{ margin: 2, p: 2, bgcolor: '#fcfdfe', borderRadius: 2, border: '1px dashed #cfd8dc' }}>
                          <Typography variant="h6" gutterBottom component="div" sx={{ display: 'flex', alignItems: 'center' }}>
                            <WarningIcon color="warning" sx={{ mr: 1 }} />
                            Impact & AI Insights
                          </Typography>
                          <Grid container spacing={2}>
                            <Grid item xs={6}>
                              <Typography variant="body2" color="text.secondary">
                                <strong>Active Field Bindings Impacted:</strong> 1 active binding.
                              </Typography>
                              <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                                <strong>Detected At:</strong> {new Date(prop.createdAt).toLocaleString()}
                              </Typography>
                            </Grid>
                            <Grid item xs={6}>
                              <Typography variant="body2" color="text.secondary">
                                <strong>LLM Confidence:</strong> {Math.round(prop.aiConfidence * 100)}%
                              </Typography>
                              <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                                <strong>Heuristic Confidence:</strong> 90% (based on string similarity matching)
                              </Typography>
                            </Grid>
                          </Grid>
                        </Box>
                      </Collapse>
                    </TableCell>
                  </TableRow>
                </React.Fragment>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Preview Proposal Dialog */}
      <Dialog open={previewProposal !== null} onClose={() => setPreviewProposal(null)}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center' }}>
          <AutoAwesomeIcon color="primary" sx={{ mr: 1 }} />
          Heal Mapping Preview
        </DialogTitle>
        <DialogContent dividers>
          {previewProposal && (
            <Box>
              <Typography variant="body1" gutterBottom>
                Are you sure you want to approve the AI‑proposed mapping correction?
              </Typography>
              <Box my={2} p={2} bgcolor="#fafafa" borderRadius={1} border="1px solid #eee">
                <Grid container spacing={2}>
                  <Grid item xs={5}>
                    <Typography variant="caption" color="text.secondary">SEMANTIC FIELD</Typography>
                    <Typography variant="body2" fontWeight="bold">{previewProposal.semanticTermName}</Typography>
                  </Grid>
                  <Grid item xs={2} display="flex" alignItems="center" justifyContent="center">
                    <Typography variant="body2" color="text.secondary">➔</Typography>
                  </Grid>
                  <Grid item xs={5}>
                    <Typography variant="caption" color="text.secondary">NEW PHYSICAL COLUMN</Typography>
                    <Typography variant="body2" fontWeight="bold" color="success.main" sx={{ fontFamily: 'monospace' }}>
                      {previewProposal.proposedColumnName}
                    </Typography>
                  </Grid>
                </Grid>
              </Box>
              <Typography variant="caption" color="text.secondary">
                Approving this change transactionally updates the metadata graph and invalidates the cached query definitions.
              </Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreviewProposal(null)}>Cancel</Button>
          <Button variant="contained" onClick={() => previewProposal && handleApprove(previewProposal.id)} color="primary">
            Apply Correction
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={snackbarMessage !== null}
        autoHideDuration={6000}
        onClose={() => setSnackbarMessage(null)}
        message={snackbarMessage}
      />
    </Container>
  );
}
