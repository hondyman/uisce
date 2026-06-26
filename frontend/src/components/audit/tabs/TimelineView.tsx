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
} from '@mui/material';
import {
  KeyboardArrowDown as KeyboardArrowDownIcon,
  KeyboardArrowUp as KeyboardArrowUpIcon,
  Help as HelpIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';

interface AuditEvent {
  id: string;
  type: string;
  tenantId: string;
  timestamp: string;
  status: string;
  riskLevel: string;
  artifactId?: string;
  artifactType?: string;
  actor?: string;
  message?: string;
  semanticContext?: any;
  complianceContext?: any;
  aiNarrative?: string;
}

interface TimelineViewProps {
  events: AuditEvent[];
  onExplain: (event: AuditEvent) => void;
  userRole: string;
}

/**
 * TimelineView: Unified timeline showing all audit events
 * 
 * Supports:
 * - Event type icons and colors
 * - Expandable rows showing context details
 * - AI explain button per event
 * - Tenant badge and timestamp formatting
 * - Risk level color coding
 */
export function TimelineView({
  events,
  onExplain,
}: TimelineViewProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const toggleRow = (eventId: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(eventId)) {
      newExpanded.delete(eventId);
    } else {
      newExpanded.add(eventId);
    }
    setExpandedRows(newExpanded);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
      case 'approved':
        return <CheckCircleIcon sx={{ color: 'success.main' }} />;
      case 'failed':
      case 'rejected':
        return <ErrorIcon sx={{ color: 'error.main' }} />;
      case 'pending':
      case 'in_progress':
        return <WarningIcon sx={{ color: 'warning.main' }} />;
      default:
        return null;
    }
  };

  const getRiskColor = (riskLevel: string) => {
    switch (riskLevel?.toLowerCase()) {
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      case 'low':
      default:
        return 'success';
    }
  };

  const formatTimestamp = (ts: string) => {
    const date = new Date(ts);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  const getEventTypeLabel = (type: string): string => {
    const labels: Record<string, string> = {
      job_run: 'Job Run',
      dag_run: 'DAG Run',
      changeset: 'Change',
      semantic_snapshot: 'Semantic',
      compliance_violation: 'Compliance',
      orchestration_event: 'Event',
    };
    return labels[type] || type;
  };

  if (events.length === 0) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="body2" sx={{ color: 'text.secondary' }}>
          No events found for the selected filters.
        </Typography>
      </Box>
    );
  }

  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
            <TableCell width="50px" />
            <TableCell>Timestamp</TableCell>
            <TableCell>Type</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Risk</TableCell>
            <TableCell>Artifact</TableCell>
            <TableCell>Actor</TableCell>
            <TableCell width="100px" align="right">
              Actions
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {events.map((event) => (
            <React.Fragment key={event.id}>
              {/* Main Row */}
              <TableRow
                hover
                sx={{
                  backgroundColor: expandedRows.has(event.id) ? '#fafafa' : 'inherit',
                  cursor: 'pointer',
                }}
              >
                <TableCell>
                  <IconButton
                    size="small"
                    onClick={() => toggleRow(event.id)}
                  >
                    {expandedRows.has(event.id) ? (
                      <KeyboardArrowUpIcon />
                    ) : (
                      <KeyboardArrowDownIcon />
                    )}
                  </IconButton>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {formatTimestamp(event.timestamp)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    {getStatusIcon(event.status)}
                    <Chip
                      label={getEventTypeLabel(event.type)}
                      size="small"
                      variant="outlined"
                    />
                  </Stack>
                </TableCell>
                <TableCell>
                  <Chip
                    label={event.status}
                    size="small"
                    variant="filled"
                    color={event.status === 'success' ? 'success' : 'default'}
                  />
                </TableCell>
                <TableCell>
                  <Chip
                    label={event.riskLevel?.toUpperCase()}
                    size="small"
                    color={getRiskColor(event.riskLevel)}
                  />
                </TableCell>
                <TableCell>
                  <Typography variant="caption">
                    {event.artifactId}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="caption">
                    {event.actor || '—'}
                  </Typography>
                </TableCell>
                <TableCell align="right">
                  <IconButton
                    size="small"
                    title="Explain this event"
                    onClick={() => onExplain(event)}
                  >
                    <HelpIcon />
                  </IconButton>
                </TableCell>
              </TableRow>

              {/* Expandable Details Row */}
              {expandedRows.has(event.id) && (
                <TableRow sx={{ backgroundColor: '#fafafa' }}>
                  <TableCell colSpan={8}>
                    <Box sx={{ p: 2 }}>
                      <Grid container spacing={2}>
                        {/* Message */}
                        {event.message && (
                          <Grid item xs={12}>
                            <Box>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Message
                              </Typography>
                              <Typography variant="body2" sx={{ mt: 0.5 }}>
                                {event.message}
                              </Typography>
                            </Box>
                          </Grid>
                        )}

                        {/* Semantic Context */}
                        {event.semanticContext && (
                          <Grid item xs={12} sm={6}>
                            <Box>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Semantic Impact
                              </Typography>
                              <Box
                                sx={{
                                  mt: 1,
                                  p: 1,
                                  backgroundColor: '#f5f5f5',
                                  borderRadius: 1,
                                  fontFamily: 'monospace',
                                  fontSize: '0.75rem',
                                  '& pre': { margin: 0, overflow: 'auto' },
                                }}
                              >
                                <pre>
                                  {JSON.stringify(event.semanticContext, null, 2)}
                                </pre>
                              </Box>
                            </Box>
                          </Grid>
                        )}

                        {/* Compliance Context */}
                        {event.complianceContext && (
                          <Grid item xs={12} sm={6}>
                            <Box>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                Compliance Impact
                              </Typography>
                              <Box
                                sx={{
                                  mt: 1,
                                  p: 1,
                                  backgroundColor: '#f5f5f5',
                                  borderRadius: 1,
                                  fontFamily: 'monospace',
                                  fontSize: '0.75rem',
                                  '& pre': { margin: 0, overflow: 'auto' },
                                }}
                              >
                                <pre>
                                  {JSON.stringify(event.complianceContext, null, 2)}
                                </pre>
                              </Box>
                            </Box>
                          </Grid>
                        )}

                        {/* AI Narrative */}
                        {event.aiNarrative && (
                          <Grid item xs={12}>
                            <Box>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                AI Summary
                              </Typography>
                              <Typography variant="body2" sx={{ mt: 0.5 }}>
                                {event.aiNarrative}
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
  );
}

export default TimelineView;
