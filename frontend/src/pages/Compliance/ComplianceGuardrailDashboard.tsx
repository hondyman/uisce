import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  LinearProgress,
  Alert,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  Shield as ShieldIcon,
  Warning as WarningIcon,
  CheckCircle as CheckIcon,
  Block as BlockIcon,
} from '@mui/icons-material';

interface GuardrailStats {
  total_checks_today: number;
  violations_today: number;
  blocked_outputs: number;
  pii_redactions: number;
  approval_rate: number;
}

interface PolicyViolation {
  policy_id: string;
  policy_name: string;
  severity: string;
  description: string;
  timestamp: string;
  user_id: string;
}

export const ComplianceGuardrailDashboard: React.FC = () => {
  const [stats, setStats] = useState<GuardrailStats | null>(null);
  const [recentViolations, setRecentViolations] = useState<PolicyViolation[]>([]);
  const [selectedViolation, setSelectedViolation] = useState<PolicyViolation | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      const [statsRes, violationsRes] = await Promise.all([
        fetch('/api/guardrails/stats?tenant_id=current'),
        fetch('/api/audit/events?action=GUARDRAIL_CHECK&limit=20'),
      ]);

      const statsData = await statsRes.json();
      const violationsData = await violationsRes.json();

      setStats(statsData);
      setRecentViolations(violationsData.events || []);
    } catch (error) {
      console.error('Failed to load guardrail data:', error);
    } finally {
      setLoading(false);
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'info' => {
    switch (severity) {
      case 'CRITICAL':
        return 'error';
      case 'HIGH':
      case 'MEDIUM':
        return 'warning';
      default:
        return 'info';
    }
  };

  if (loading || !stats) {
    return <LinearProgress />;
  }

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <ShieldIcon fontSize="large" color="primary" />
        AI Guardrails & Compliance Dashboard
      </Typography>

      <Alert severity="info" sx={{ mb: 3 }}>
        Real-time monitoring of AI output filtering and regulatory compliance checks
      </Alert>

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} md={3}>
          <Card elevation={3}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <ShieldIcon color="primary" />
                <Typography color="text.secondary" variant="body2">
                  Total Checks Today
                </Typography>
              </Box>
              <Typography variant="h3">{stats.total_checks_today}</Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card elevation={3}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <CheckIcon color="success" />
                <Typography color="text.secondary" variant="body2">
                  Approval Rate
                </Typography>
              </Box>
              <Typography variant="h3" color="success.main">
                {(stats.approval_rate * 100).toFixed(1)}%
              </Typography>
              <LinearProgress
                variant="determinate"
                value={stats.approval_rate * 100}
                color="success"
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card elevation={3}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <WarningIcon color="warning" />
                <Typography color="text.secondary" variant="body2">
                  Violations Detected
                </Typography>
              </Box>
              <Typography variant="h3" color="warning.main">
                {stats.violations_today}
              </Typography>
              <Typography variant="caption">PII Redactions: {stats.pii_redactions}</Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={3}>
          <Card elevation={3}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <BlockIcon color="error" />
                <Typography color="text.secondary" variant="body2">
                  Blocked Outputs
                </Typography>
              </Box>
              <Typography variant="h3" color="error.main">
                {stats.blocked_outputs}
              </Typography>
              <Typography variant="caption">Critical violations</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Recent Violations Table */}
      <Paper elevation={2}>
        <Box sx={{ p: 2, bgcolor: 'grey.100' }}>
          <Typography variant="h6">Recent Policy Violations</Typography>
        </Box>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell><strong>Timestamp</strong></TableCell>
                <TableCell><strong>Policy</strong></TableCell>
                <TableCell><strong>Severity</strong></TableCell>
                <TableCell><strong>Description</strong></TableCell>
                <TableCell><strong>User</strong></TableCell>
                <TableCell><strong>Actions</strong></TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {recentViolations.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} align="center">
                    <Box sx={{ py: 4 }}>
                      <CheckIcon sx={{ fontSize: 48, color: 'success.main', mb: 1 }} />
                      <Typography color="text.secondary">
                        No violations detected today. System operating normally.
                      </Typography>
                    </Box>
                  </TableCell>
                </TableRow>
              ) : (
                recentViolations.map((violation, index) => (
                  <TableRow key={index} hover>
                    <TableCell>
                      {new Date(violation.timestamp).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight="medium">
                        {violation.policy_name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {violation.policy_id}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={violation.severity}
                        color={getSeverityColor(violation.severity)}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ maxWidth: 300 }}>
                        {violation.description}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption">{violation.user_id}</Typography>
                    </TableCell>
                    <TableCell>
                      <Button
                        size="small"
                        onClick={() => setSelectedViolation(violation)}
                      >
                        View Details
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Violation Details Dialog */}
      <Dialog
        open={selectedViolation !== null}
        onClose={() => setSelectedViolation(null)}
        maxWidth="md"
        fullWidth
      >
        {selectedViolation && (
          <>
            <DialogTitle>
              Violation Details: {selectedViolation.policy_name}
            </DialogTitle>
            <DialogContent dividers>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Policy ID
                  </Typography>
                  <Typography variant="body1">{selectedViolation.policy_id}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Severity
                  </Typography>
                  <Chip
                    label={selectedViolation.severity}
                    color={getSeverityColor(selectedViolation.severity)}
                  />
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Description
                  </Typography>
                  <Typography variant="body1">{selectedViolation.description}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Timestamp
                  </Typography>
                  <Typography variant="body1">
                    {new Date(selectedViolation.timestamp).toLocaleString()}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="text.secondary">
                    User
                  </Typography>
                  <Typography variant="body1">{selectedViolation.user_id}</Typography>
                </Grid>
              </Grid>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setSelectedViolation(null)}>Close</Button>
              <Button variant="outlined">Export Audit Log</Button>
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  );
};
