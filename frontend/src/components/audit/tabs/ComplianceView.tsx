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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Alert,
  Tooltip,
} from '@mui/material';
import {
  Info as InfoIcon,
  Help as HelpIcon,
  Lock as LockIcon,
  WarningAmber as WarningIcon,
} from '@mui/icons-material';

interface ComplianceEvent {
  id: string;
  tenantId: string;
  timestamp: string;
  violationType: string;
  severity: string;
  affectedRecords: number;
  status: string;
  remediedAt?: string;
  artifact?: {
    type: string;
    id: string;
  };
  narrative?: string;
  remediationPath?: string;
}

interface ComplianceViewProps {
  events: ComplianceEvent[];
  onExplain: (event: any) => void;
  userRole: string;
}

/**
 * ComplianceView: Compliance violation tracking and remediation
 * 
 * Supports:
 * - Violation type filtering (PII, data classification, access, retention)
 * - Severity levels
 * - Remediation tracking
 * - Audit trail for compliance teams
 */
export function ComplianceView({
  events,
  onExplain,
  userRole,
}: ComplianceViewProps) {
  const [selectedEvent, setSelectedEvent] = useState<ComplianceEvent | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);

  const handleShowDetails = (event: ComplianceEvent) => {
    setSelectedEvent(event);
    setDetailsOpen(true);
  };

  const formatTimestamp = (ts: string) => {
    const date = new Date(ts);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getSeverityColor = (severity: string) => {
    switch (severity?.toLowerCase()) {
      case 'critical':
        return 'error';
      case 'high':
        return 'warning';
      case 'medium':
        return 'info';
      case 'low':
      default:
        return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status?.toLowerCase()) {
      case 'remediated':
      case 'resolved':
        return 'success';
      case 'pending':
      case 'open':
        return 'warning';
      case 'escalated':
        return 'error';
      default:
        return 'default';
    }
  };

  const canAckowledge = ['global_admin', 'tenant_admin'].includes(userRole);

  if (events.length === 0) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="body2" sx={{ color: 'text.secondary' }}>
          No compliance events found for the selected period.
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
              <TableCell>Timestamp</TableCell>
              <TableCell>Violation Type</TableCell>
              <TableCell>Severity</TableCell>
              <TableCell>Affected Records</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Artifact</TableCell>
              <TableCell>Remediated</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {events.map((event) => (
              <TableRow key={event.id} hover>
                <TableCell>
                  <Typography variant="body2">
                    {formatTimestamp(event.timestamp)}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    {event.violationType.toLowerCase().includes('pii') && (
                      <LockIcon fontSize="small" />
                    )}
                    <Chip
                      label={event.violationType}
                      size="small"
                      variant="outlined"
                    />
                  </Stack>
                </TableCell>
                <TableCell>
                  <Chip
                    label={event.severity}
                    size="small"
                    color={getSeverityColor(event.severity)}
                  />
                </TableCell>
                <TableCell>
                  <Typography variant="body2">
                    {event.affectedRecords.toLocaleString()}
                  </Typography>
                </TableCell>
                <TableCell>
                  <Chip
                    label={event.status}
                    size="small"
                    color={getStatusColor(event.status)}
                  />
                </TableCell>
                <TableCell>
                  <Typography variant="caption">
                    {event.artifact?.type}: {event.artifact?.id}
                  </Typography>
                </TableCell>
                <TableCell>
                  {event.remediedAt ? (
                    <Typography variant="caption">
                      {formatTimestamp(event.remediedAt)}
                    </Typography>
                  ) : (
                    <Tooltip title="Not yet remediated">
                      <WarningIcon
                        fontSize="small"
                        sx={{ color: 'warning.main' }}
                      />
                    </Tooltip>
                  )}
                </TableCell>
                <TableCell align="right">
                  <Stack direction="row" spacing={1}>
                    <IconButton
                      size="small"
                      title="View details"
                      onClick={() => handleShowDetails(event)}
                    >
                      <InfoIcon />
                    </IconButton>
                    <IconButton
                      size="small"
                      title="Get explanation"
                      onClick={() => onExplain(event)}
                    >
                      <HelpIcon />
                    </IconButton>
                  </Stack>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Details Dialog */}
      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Compliance Violation Details</DialogTitle>
        <DialogContent>
          {selectedEvent && (
            <Stack spacing={2} sx={{ mt: 2 }}>
              {selectedEvent.narrative && (
                <Alert severity="info">{selectedEvent.narrative}</Alert>
              )}

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Violation Type
                </Typography>
                <Typography variant="body2" sx={{ mt: 0.5 }}>
                  {selectedEvent.violationType}
                </Typography>
              </Box>

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Severity
                </Typography>
                <Box sx={{ mt: 0.5 }}>
                  <Chip
                    label={selectedEvent.severity}
                    color={getSeverityColor(selectedEvent.severity)}
                    size="small"
                  />
                </Box>
              </Box>

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Affected Records
                </Typography>
                <Typography variant="body2" sx={{ mt: 0.5 }}>
                  {selectedEvent.affectedRecords.toLocaleString()}
                </Typography>
              </Box>

              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600 }}>
                  Status
                </Typography>
                <Box sx={{ mt: 0.5 }}>
                  <Chip
                    label={selectedEvent.status}
                    color={getStatusColor(selectedEvent.status)}
                    size="small"
                  />
                </Box>
              </Box>

              {selectedEvent.remediationPath && (
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 600 }}>
                    Remediation Path
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 0.5 }}>
                    {selectedEvent.remediationPath}
                  </Typography>
                </Box>
              )}

              {selectedEvent.artifact && (
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 600 }}>
                    Affected Artifact
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 0.5 }}>
                    {selectedEvent.artifact.type}: {selectedEvent.artifact.id}
                  </Typography>
                </Box>
              )}

              {selectedEvent.remediedAt && (
                <Alert severity="success">
                  Remediated at {formatTimestamp(selectedEvent.remediedAt)}
                </Alert>
              )}
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          {canAckowledge && !selectedEvent?.remediedAt && (
            <Button variant="contained" size="small">
              Mark as Resolved
            </Button>
          )}
          <Button onClick={() => setDetailsOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

export default ComplianceView;
