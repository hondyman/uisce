import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Alert,
  CircularProgress,
  Button,
  Tabs,
  Tab,
  LinearProgress,
} from '@mui/material';
import {
  CheckCircle,
  Error,
  Warning,
  Download,
  Assessment,
} from '@mui/icons-material';
import { EvidenceBundleAPI, ComplianceReport } from '../../api/evidenceBundle';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => {
  return (
    <div role="tabpanel" hidden={value !== index}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
};

export const ComplianceReportPage: React.FC = () => {
  const { bundleId } = useParams<{ bundleId: string }>();
  const [report, setReport] = useState<ComplianceReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentTab, setCurrentTab] = useState(0);

  useEffect(() => {
    if (bundleId) {
      loadReport();
    }
  }, [bundleId]);

  const loadReport = async () => {
    try {
      setLoading(true);
      const data = await EvidenceBundleAPI.getComplianceReport(bundleId!);
      setReport(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load compliance report');
    } finally {
      setLoading(false);
    }
  };

  const handleDownload = async () => {
    try {
      await EvidenceBundleAPI.downloadComplianceReport(bundleId!);
    } catch (err: any) {
      alert('Failed to download report');
    }
  };

  const getRiskColor = (risk: string): 'error' | 'warning' | 'success' => {
    switch (risk) {
      case 'HIGH':
        return 'error';
      case 'MEDIUM':
        return 'warning';
      case 'LOW':
        return 'success';
      default:
        return 'success';
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'success' => {
    switch (severity) {
      case 'BREAKING':
        return 'error';
      case 'ADDITIVE':
        return 'success';
      case 'SAFE':
        return 'success';
      default:
        return 'warning';
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error || !report) {
    return (
      <Alert severity="error" sx={{ m: 3 }}>
        {error || 'Compliance report not found'}
      </Alert>
    );
  }

  const { executive_summary, change_inventory, test_summary, approval_chain, deployment_log } = report;

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Compliance Report
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Generated: {new Date(report.generated_at).toLocaleString()}
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<Download />}
          onClick={handleDownload}
        >
          Download Report
        </Button>
      </Box>

      {/* Executive Summary */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box display="flex" alignItems="center" mb={2}>
            <Assessment sx={{ mr: 1 }} />
            <Typography variant="h5">Executive Summary</Typography>
          </Box>

          <Grid container spacing={3}>
            <Grid item xs={12} md={6} lg={3}>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50', textAlign: 'center' }}>
                <Typography variant="h3" color="primary">
                  {executive_summary.breaking_changes}
                </Typography>
                <Typography variant="subtitle2" color="text.secondary">
                  Breaking Changes
                </Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} md={6} lg={3}>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50', textAlign: 'center' }}>
                <Typography variant="h3" color="success.main">
                  {executive_summary.additive_changes}
                </Typography>
                <Typography variant="subtitle2" color="text.secondary">
                  Additive Changes
                </Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} md={6} lg={3}>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50', textAlign: 'center' }}>
                <Typography variant="h3" color="info.main">
                  {(executive_summary.test_pass_rate * 100).toFixed(1)}%
                </Typography>
                <Typography variant="subtitle2" color="text.secondary">
                  Test Pass Rate
                </Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} md={6} lg={3}>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50', textAlign: 'center' }}>
                <Chip
                  label={executive_summary.risk_level}
                  color={getRiskColor(executive_summary.risk_level)}
                  size="large"
                  sx={{ fontSize: '1.25rem', fontWeight: 'bold' }}
                />
                <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 1 }}>
                  Risk Level
                </Typography>
              </Paper>
            </Grid>
          </Grid>

          {executive_summary.deployment_success ? (
            <Alert severity="success" icon={<CheckCircle />} sx={{ mt: 2 }}>
              Deployment completed successfully
            </Alert>
          ) : (
            <Alert severity="error" icon={<Error />} sx={{ mt: 2 }}>
              Deployment failed or incomplete
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Tabs for detailed sections */}
      <Card>
        <Tabs value={currentTab} onChange={(_, v) => setCurrentTab(v)}>
          <Tab label={`Change Inventory (${change_inventory.length})`} />
          <Tab label="Test Results" />
          <Tab label={`Approvals (${approval_chain.length})`} />
          <Tab label="Deployment Log" />
        </Tabs>

        <CardContent>
          {/* Change Inventory Tab */}
          <TabPanel value={currentTab} index={0}>
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Path</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Severity</TableCell>
                    <TableCell>Impact</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {change_inventory.map((change, idx) => (
                    <TableRow key={idx}>
                      <TableCell>
                        <Typography variant="body2" fontFamily="monospace" fontSize="0.875rem">
                          {change.path}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={change.type} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={change.severity}
                          size="small"
                          color={getSeverityColor(change.severity)}
                        />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">{change.impact}</Typography>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </TabPanel>

          {/* Test Results Tab */}
          <TabPanel value={currentTab} index={1}>
            <Grid container spacing={2} sx={{ mb: 3 }}>
              <Grid item xs={6} md={3}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                  <Typography variant="h4">{test_summary.total_tests}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Total Tests
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={6} md={3}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'success.50' }}>
                  <Typography variant="h4" color="success.main">
                    {test_summary.passed_tests}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Passed
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={6} md={3}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'error.50' }}>
                  <Typography variant="h4" color="error.main">
                    {test_summary.failed_tests}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Failed
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={6} md={3}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                  <Typography variant="h4">{test_summary.coverage.toFixed(1)}%</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Coverage
                  </Typography>
                </Paper>
              </Grid>
            </Grid>

            <Box sx={{ mb: 2 }}>
              <Typography variant="subtitle2" gutterBottom>
                Test Pass Rate
              </Typography>
              <LinearProgress
                variant="determinate"
                value={(test_summary.passed_tests / test_summary.total_tests) * 100}
                color={test_summary.failed_tests === 0 ? 'success' : 'warning'}
                sx={{ height: 10, borderRadius: 1 }}
              />
            </Box>

            {test_summary.failed_test_details && test_summary.failed_test_details.length > 0 && (
              <Box>
                <Typography variant="h6" gutterBottom>
                  Failed Tests
                </Typography>
                {test_summary.failed_test_details.map((test, idx) => (
                  <Alert key={idx} severity="error" sx={{ mb: 1 }}>
                    <Typography variant="subtitle2">{test.test_name}</Typography>
                    <Typography variant="body2">{test.error_message}</Typography>
                    {test.related_diff && (
                      <Typography variant="caption" color="text.secondary">
                        Related to: {test.related_diff}
                      </Typography>
                    )}
                  </Alert>
                ))}
              </Box>
            )}
          </TabPanel>

          {/* Approvals Tab */}
          <TabPanel value={currentTab} index={2}>
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Approver</TableCell>
                    <TableCell>Decision</TableCell>
                    <TableCell>Justification</TableCell>
                    <TableCell>Timestamp</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {approval_chain.map((approval, idx) => (
                    <TableRow key={idx}>
                      <TableCell>{approval.approver_id}</TableCell>
                      <TableCell>
                        <Chip
                          label={approval.decision}
                          size="small"
                          color={approval.decision === 'approved' ? 'success' : 'error'}
                          icon={approval.decision === 'approved' ? <CheckCircle /> : <Error />}
                        />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">{approval.justification}</Typography>
                      </TableCell>
                      <TableCell>
                        {new Date(approval.decided_at).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                  {approval_chain.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} align="center">
                        <Typography variant="body2" color="text.secondary">
                          No approvals required or recorded
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </TabPanel>

          {/* Deployment Log Tab */}
          <TabPanel value={currentTab} index={3}>
            <Grid container spacing={2}>
              <Grid item xs={12} md={4}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                  <Typography variant="h4">{deployment_log.target_tenants.length}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Target Tenants
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} md={4}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'success.50' }}>
                  <Typography variant="h4" color="success.main">
                    {deployment_log.successful_deploys}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Successful
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} md={4}>
                <Paper elevation={0} sx={{ p: 2, bgcolor: 'error.50' }}>
                  <Typography variant="h4" color="error.main">
                    {deployment_log.failed_deploys}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Failed
                  </Typography>
                </Paper>
              </Grid>
            </Grid>

            <Box sx={{ mt: 3 }}>
              <Typography variant="subtitle2" gutterBottom>
                Started: {new Date(deployment_log.started_at).toLocaleString()}
              </Typography>
              {deployment_log.completed_at && (
                <Typography variant="subtitle2" gutterBottom>
                  Completed: {new Date(deployment_log.completed_at).toLocaleString()}
                </Typography>
              )}
            </Box>

            {deployment_log.rollback_events && deployment_log.rollback_events.length > 0 && (
              <Box sx={{ mt: 3 }}>
                <Alert severity="warning" icon={<Warning />}>
                  <Typography variant="subtitle2" gutterBottom>
                    Rollback Events ({deployment_log.rollback_events.length})
                  </Typography>
                  {deployment_log.rollback_events.map((event, idx) => (
                    <Box key={idx} sx={{ mt: 1 }}>
                      <Typography variant="body2">
                        <strong>{event.tenant_id}</strong>: {event.reason}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {new Date(event.timestamp).toLocaleString()} by {event.actor_id}
                      </Typography>
                    </Box>
                  ))}
                </Alert>
              </Box>
            )}
          </TabPanel>
        </CardContent>
      </Card>
    </Box>
  );
};
