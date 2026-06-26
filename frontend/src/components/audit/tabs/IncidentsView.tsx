import React, { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Box,
  IconButton,
  Typography,
  Stack,
  Grid,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  Info as InfoIcon,
  Help as HelpIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
} from '@mui/icons-material';

interface IncidentCluster {
  id: string;
  timeWindow: {
    start: string;
    end: string;
  };
  affectedTenants: string[];
  affectedJobs: string[];
  affectedDAGs: string[];
  failureCount: number;
  aiRootCause?: string;
  blastRadius: string;
  sloImpact: {
    failed_slas: number;
    estimated_impact: number;
  };
}

interface IncidentsViewProps {
  incidents: IncidentCluster[];
  onExplain: (event: any) => void;
  userRole: string;
}

/**
 * IncidentsView: Grouped failure incidents with AI root cause analysis
 * 
 * Supports:
 * - Incident clustering and grouping
 * - AI root cause explanations
 * - Blast radius visualization
 * - SLO impact assessment
 * - Cross-tenant incident correlation
 */
export function IncidentsView({
  incidents,
  onExplain,
}: IncidentsViewProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [selectedIncident, setSelectedIncident] = useState<IncidentCluster | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);

  const toggleRow = (incidentId: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(incidentId)) {
      newExpanded.delete(incidentId);
    } else {
      newExpanded.add(incidentId);
    }
    setExpandedRows(newExpanded);
  };

  const handleShowDetails = (incident: IncidentCluster) => {
    setSelectedIncident(incident);
    setDetailsOpen(true);
  };

  const formatTime = (ts: string) => {
    return new Date(ts).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (incidents.length === 0) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="body2" sx={{ color: 'text.secondary' }}>
          No incidents found for the selected period.
        </Typography>
      </Box>
    );
  }

  return (
    <>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell width="50px" />
              <TableCell>Time Window</TableCell>
              <TableCell>Failures</TableCell>
              <TableCell>Affected Resources</TableCell>
              <TableCell>SLO Impact</TableCell>
              <TableCell>Root Cause</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {incidents.map((incident) => (
              <React.Fragment key={incident.id}>
                {/* Main Row */}
                <TableRow hover>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => toggleRow(incident.id)}
                    >
                      {expandedRows.has(incident.id) ? (
                        <ExpandLessIcon />
                      ) : (
                        <ExpandMoreIcon />
                      )}
                    </IconButton>
                  </TableCell>
                  <TableCell>
                    <Stack spacing={0.5}>
                      <Typography variant="body2">
                        {formatTime(incident.timeWindow.start)}
                      </Typography>
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        {formatTime(incident.timeWindow.end)}
                      </Typography>
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={incident.failureCount}
                      color="error"
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Stack spacing={0.5}>
                      <Typography variant="caption">
                        {incident.affectedJobs.length} jobs,{' '}
                        {incident.affectedDAGs.length} DAGs
                      </Typography>
                      {incident.affectedTenants.length > 1 && (
                        <Chip
                          label={`${incident.affectedTenants.length} tenants`}
                          size="small"
                          color="warning"
                        />
                      )}
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">
                      {incident.sloImpact.failed_slas} SLAs failed
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ maxWidth: 200 }}>
                      {incident.aiRootCause
                        ? incident.aiRootCause.substring(0, 80) + '...'
                        : 'Analyzing...'}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1}>
                      <IconButton
                        size="small"
                        title="View details"
                        onClick={() => handleShowDetails(incident)}
                      >
                        <InfoIcon />
                      </IconButton>
                      <IconButton
                        size="small"
                        title="Get explanation"
                        onClick={() => onExplain(incident)}
                      >
                        <HelpIcon />
                      </IconButton>
                    </Stack>
                  </TableCell>
                </TableRow>

                {/* Expandable Details */}
                {expandedRows.has(incident.id) && (
                  <TableRow sx={{ backgroundColor: '#fafafa' }}>
                    <TableCell colSpan={7}>
                      <Box sx={{ p: 2 }}>
                        <Grid container spacing={2}>
                          <Grid item xs={12} sm={6}>
                            <Box>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Blast Radius
                              </Typography>
                              <Chip
                                label={incident.blastRadius}
                                size="small"
                                sx={{ mt: 1 }}
                              />
                            </Box>
                          </Grid>
                          <Grid item xs={12} sm={6}>
                            <Box>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Estimated Impact
                              </Typography>
                              <Typography variant="body2" sx={{ mt: 1 }}>
                                {(incident.sloImpact.estimated_impact * 100).toFixed(1)}%
                              </Typography>
                            </Box>
                          </Grid>
                          {incident.aiRootCause && (
                            <Grid item xs={12}>
                              <Box>
                                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                  AI Root Cause Analysis
                                </Typography>
                                <Typography variant="body2" sx={{ mt: 1 }}>
                                  {incident.aiRootCause}
                                </Typography>
                              </Box>
                            </Grid>
                          )}
                        </Grid>
                      </Box>
                    </TableCell>
                  </TableRow>
                )}
              </React.Fragment>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Incident Details Dialog */}
      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Incident Details</DialogTitle>
        <DialogContent>
          {selectedIncident && (
            <Stack spacing={2} sx={{ mt: 2 }}>
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Time Window
                </Typography>
                <Typography variant="body2">
                  {formatTime(selectedIncident.timeWindow.start)} →{' '}
                  {formatTime(selectedIncident.timeWindow.end)}
                </Typography>
              </Box>

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Affected Jobs
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 1, flexWrap: 'wrap' }}>
                  {selectedIncident.affectedJobs.map((jobId) => (
                    <Chip key={jobId} label={jobId} size="small" />
                  ))}
                </Stack>
              </Box>

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Affected DAGs
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 1, flexWrap: 'wrap' }}>
                  {selectedIncident.affectedDAGs.map((dagId) => (
                    <Chip key={dagId} label={dagId} size="small" />
                  ))}
                </Stack>
              </Box>

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  SLO Impact
                </Typography>
                <Typography variant="body2" sx={{ mt: 1 }}>
                  {selectedIncident.sloImpact.failed_slas} SLAs failed,{' '}
                  {(selectedIncident.sloImpact.estimated_impact * 100).toFixed(1)}%
                  impact
                </Typography>
              </Box>
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

export default IncidentsView;
