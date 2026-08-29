import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  Chip,
  CircularProgress,
  Alert,
  Paper,
  Dialog,
  DialogTitle,
  DialogContent,
  IconButton,
  Grid,
} from '@mui/material';
import {
  Warning,
  TrendingUp,
  Storage,
  CheckCircle,
  Refresh,
  Close,
} from '@mui/icons-material';

interface AuditExplorerProps {
  tenantId: string;
  tenantName: string;
}

interface JobRun {
  run_id: string;
  job_id: string;
  tenant_id: string;
  start_ts: string;
  end_ts: string;
  status: string;
  error_message?: string;
  semantic_context?: any;
  compliance_context?: any;
  slo_context?: any;
  ai_narrative?: any;
}

interface ComplianceViolation {
  violation_id: string;
  tenant_id: string;
  violated_at: string;
  remediated_at?: string;
  violation_type: string;
  severity: string;
  pii_exposed: boolean;
  affected_records: number;
  narrative: string;
}

interface ChangeSet {
  changeset_id: string;
  type: string;
  actor: string;
  created_at: string;
  status: string;
  semantic_impact?: any;
  compliance_impact?: any;
  ai_summary?: any;
  ai_risk?: any;
}

type TabType = 'jobs' | 'violations' | 'changesets' | 'dashboards';

export const AuditExplorer: React.FC<AuditExplorerProps> = ({ tenantId, tenantName }) => {
  const [activeTab, setActiveTab] = useState<TabType>('jobs');
  const [jobRuns, setJobRuns] = useState<JobRun[]>([]);
  const [violations, setViolations] = useState<ComplianceViolation[]>([]);
  const [changeSets, setChangeSets] = useState<ChangeSet[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedRecord, setSelectedRecord] = useState<any | null>(null);
  
  // Filters
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [dateFilter, setDateFilter] = useState<string>('7d');

  useEffect(() => {
    loadData();
  }, [activeTab, tenantId, statusFilter, dateFilter]);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const headers = {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json',
      };

      switch (activeTab) {
        case 'jobs':
          const jobsResponse = await fetch(
            `/api/audit/job-runs?status=${statusFilter}&limit=100`,
            { headers }
          );
          const jobsData = await jobsResponse.json();
          setJobRuns(jobsData.data || []);
          break;

        case 'violations':
          const violationsResponse = await fetch(
            `/api/audit/violations?limit=100`,
            { headers }
          );
          const violationsData = await violationsResponse.json();
          setViolations(violationsData.data || []);
          break;

        case 'changesets':
          const changesetsResponse = await fetch(
            `/api/audit/changesets?limit=100`,
            { headers }
          );
          const changesetsData = await changesetsResponse.json();
          setChangeSets(changesetsData.data || []);
          break;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const explainWithAI = async (recordType: string, recordId: string) => {
    try {
      const response = await fetch(`/api/audit/ai-narratives`, {
        method: 'POST',
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ record_type: recordType, record_id: recordId }),
      });
      
      const data = await response.json();
      setSelectedRecord({ ...selectedRecord, aiNarrative: data });
    } catch (err) {
      console.error('Failed to generate AI narrative:', err);
    }
  };

  const getStatusColor = (status: string): 'success' | 'error' | 'warning' | 'info' | 'default' => {
    switch (status.toUpperCase()) {
      case 'SUCCESS': return 'success';
      case 'FAILED': return 'error';
      case 'COMPLIANCE_BLOCK': return 'warning';
      case 'RUNNING': return 'info';
      default: return 'default';
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'info' | 'success' | 'default' => {
    switch (severity.toUpperCase()) {
      case 'CRITICAL': return 'error';
      case 'HIGH': return 'warning';
      case 'MEDIUM': return 'info';
      case 'LOW': return 'success';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', p: 3 }}>
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Box>
              <Typography variant="h5" fontWeight="bold">
                Audit Explorer
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Tenant: {tenantName} ({tenantId})
              </Typography>
            </Box>
            <Button variant="contained" startIcon={<Refresh />} onClick={loadData}>
              Refresh
            </Button>
          </Box>

          <Tabs
            value={activeTab}
            onChange={(_, newValue) => setActiveTab(newValue)}
            sx={{ borderBottom: 1, borderColor: 'divider' }}
          >
            <Tab icon={<Storage />} iconPosition="start" label="Job Runs" value="jobs" />
            <Tab icon={<Warning />} iconPosition="start" label="Compliance" value="violations" />
            <Tab icon={<TrendingUp />} iconPosition="start" label="Governance" value="changesets" />
            <Tab icon={<CheckCircle />} iconPosition="start" label="Dashboards" value="dashboards" />
          </Tabs>
        </CardContent>
      </Card>

      <Box sx={{ maxWidth: 1400, mx: 'auto' }}>
        <Card sx={{ mb: 2 }}>
          <CardContent>
            <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
              {activeTab === 'jobs' && (
                <FormControl size="small" sx={{ minWidth: 150 }}>
                  <InputLabel>Status</InputLabel>
                  <Select
                    value={statusFilter}
                    label="Status"
                    onChange={(e) => setStatusFilter(e.target.value)}
                  >
                    <MenuItem value="">All Statuses</MenuItem>
                    <MenuItem value="SUCCESS">Success</MenuItem>
                    <MenuItem value="FAILED">Failed</MenuItem>
                    <MenuItem value="COMPLIANCE_BLOCK">Compliance Block</MenuItem>
                  </Select>
                </FormControl>
              )}
              <FormControl size="small" sx={{ minWidth: 150 }}>
                <InputLabel>Date Range</InputLabel>
                <Select
                  value={dateFilter}
                  label="Date Range"
                  onChange={(e) => setDateFilter(e.target.value)}
                >
                  <MenuItem value="1d">Last 24 Hours</MenuItem>
                  <MenuItem value="7d">Last 7 Days</MenuItem>
                  <MenuItem value="30d">Last 30 Days</MenuItem>
                  <MenuItem value="90d">Last 90 Days</MenuItem>
                </Select>
              </FormControl>
            </Box>
          </CardContent>
        </Card>

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 256 }}>
            <CircularProgress />
          </Box>
        ) : error ? (
          <Alert severity="error">{error}</Alert>
        ) : (
          <>
            {activeTab === 'jobs' && (
              <JobRunsTable
                jobRuns={jobRuns}
                onExplain={(runId) => explainWithAI('JOB_RUN', runId)}
                onSelect={setSelectedRecord}
                getStatusColor={getStatusColor}
              />
            )}

            {activeTab === 'violations' && (
              <ViolationsTable
                violations={violations}
                onSelect={setSelectedRecord}
                getSeverityColor={getSeverityColor}
              />
            )}

            {activeTab === 'changesets' && (
              <ChangeSetsTable
                changeSets={changeSets}
                onExplain={(csId) => explainWithAI('CHANGESET', csId)}
                onSelect={setSelectedRecord}
                getStatusColor={getStatusColor}
              />
            )}

            {activeTab === 'dashboards' && (
              <DashboardsView tenantId={tenantId} />
            )}
          </>
        )}
      </Box>

      <DetailPanel
        open={!!selectedRecord}
        record={selectedRecord}
        onClose={() => setSelectedRecord(null)}
      />
    </Box>
  );
};

const JobRunsTable: React.FC<{
  jobRuns: JobRun[];
  onExplain: (runId: string) => void;
  onSelect: (record: any) => void;
  getStatusColor: (status: string) => 'success' | 'error' | 'warning' | 'info' | 'default';
}> = ({ jobRuns, onExplain, onSelect, getStatusColor }) => {
  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Job ID</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Duration</TableCell>
            <TableCell>Start Time</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {jobRuns.map((run) => (
            <TableRow key={run.run_id} hover onClick={() => onSelect(run)} sx={{ cursor: 'pointer' }}>
              <TableCell>{run.job_id}</TableCell>
              <TableCell>
                <Chip label={run.status} color={getStatusColor(run.status)} size="small" />
              </TableCell>
              <TableCell>{calculateDuration(run.start_ts, run.end_ts)}</TableCell>
              <TableCell>{formatTimestamp(run.start_ts)}</TableCell>
              <TableCell align="right">
                <Button size="small" onClick={() => onExplain(run.run_id)} sx={{ mr: 1 }}>
                  Explain with AI
                </Button>
                <Button size="small" onClick={() => onSelect(run)}>
                  View Details
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const ViolationsTable: React.FC<{
  violations: ComplianceViolation[];
  onSelect: (record: any) => void;
  getSeverityColor: (severity: string) => 'error' | 'warning' | 'info' | 'success' | 'default';
}> = ({ violations, onSelect, getSeverityColor }) => {
  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Type</TableCell>
            <TableCell>Severity</TableCell>
            <TableCell>PII Exposed</TableCell>
            <TableCell>Records Affected</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Violated At</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {violations.map((violation) => (
            <TableRow
              key={violation.violation_id}
              hover
              onClick={() => onSelect(violation)}
              sx={{ cursor: 'pointer' }}
            >
              <TableCell>{violation.violation_type}</TableCell>
              <TableCell>
                <Chip label={violation.severity} color={getSeverityColor(violation.severity)} size="small" />
              </TableCell>
              <TableCell>
                {violation.pii_exposed ? (
                  <Typography color="error" fontWeight="bold">YES</Typography>
                ) : (
                  <Typography color="success.main">No</Typography>
                )}
              </TableCell>
              <TableCell>{violation.affected_records.toLocaleString()}</TableCell>
              <TableCell>
                {violation.remediated_at ? (
                  <Typography color="success.main">Remediated</Typography>
                ) : (
                  <Typography color="warning.main" fontWeight="bold">Open</Typography>
                )}
              </TableCell>
              <TableCell>{formatTimestamp(violation.violated_at)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const ChangeSetsTable: React.FC<{
  changeSets: ChangeSet[];
  onExplain: (csId: string) => void;
  onSelect: (record: any) => void;
  getStatusColor: (status: string) => 'success' | 'error' | 'warning' | 'info' | 'default';
}> = ({ changeSets, onExplain, onSelect, getStatusColor }) => {
  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Type</TableCell>
            <TableCell>Actor</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Created</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {changeSets.map((cs) => (
            <TableRow key={cs.changeset_id} hover onClick={() => onSelect(cs)} sx={{ cursor: 'pointer' }}>
              <TableCell>{cs.type}</TableCell>
              <TableCell>{cs.actor}</TableCell>
              <TableCell>
                <Chip label={cs.status} color={getStatusColor(cs.status)} size="small" />
              </TableCell>
              <TableCell>{formatTimestamp(cs.created_at)}</TableCell>
              <TableCell align="right">
                <Button size="small" onClick={() => onExplain(cs.changeset_id)} sx={{ mr: 1 }}>
                  Explain with AI
                </Button>
                <Button size="small" onClick={() => onSelect(cs)}>
                  View Details
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const DashboardsView: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [sloData, setSloData] = useState<any[]>([]);
  const [complianceData, setComplianceData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDashboardData();
  }, [tenantId]);

  const loadDashboardData = async () => {
    try {
      const headers = { 'X-Tenant-ID': tenantId };
      
      const [sloResponse, complianceResponse] = await Promise.all([
        fetch('/api/audit/dashboard/slo', { headers }),
        fetch('/api/audit/dashboard/compliance', { headers }),
      ]);

      const slo = await sloResponse.json();
      const compliance = await complianceResponse.json();

      setSloData(slo.data || []);
      setComplianceData(compliance.data || []);
    } catch (err) {
      console.error('Failed to load dashboard data:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <Typography textAlign="center" py={4}>Loading dashboards...</Typography>;
  }

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>SLO Performance</Typography>
            <Box>
              {sloData.slice(0, 7).map((day, idx) => (
                <Box key={idx} sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    {new Date(day.run_date).toLocaleDateString()}
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 2 }}>
                    <Typography variant="body2">
                      {((day.successful_runs / day.total_runs) * 100).toFixed(1)}% success
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {day.total_runs} runs
                    </Typography>
                  </Box>
                </Box>
              ))}
            </Box>
          </CardContent>
        </Card>
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>Compliance Status</Typography>
            <Box>
              {complianceData.slice(0, 7).map((day, idx) => (
                <Box key={idx} sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    {new Date(day.violation_date).toLocaleDateString()}
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 2 }}>
                    <Typography
                      variant="body2"
                      fontWeight="bold"
                      color={day.violation_count > 0 ? 'error' : 'success.main'}
                    >
                      {day.violation_count} violations
                    </Typography>
                    {day.pii_exposure_count > 0 && (
                      <Typography variant="body2" color="error" fontWeight="bold">
                        {day.pii_exposure_count} PII
                      </Typography>
                    )}
                  </Box>
                </Box>
              ))}
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  );
};

const DetailPanel: React.FC<{
  open: boolean;
  record: any;
  onClose: () => void;
}> = ({ open, record, onClose }) => {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6">Record Details</Typography>
        <IconButton onClick={onClose}>
          <Close />
        </IconButton>
      </DialogTitle>
      <DialogContent>
        <Paper sx={{ p: 2, bgcolor: 'background.default' }}>
          <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {JSON.stringify(record, null, 2)}
          </pre>
        </Paper>
      </DialogContent>
    </Dialog>
  );
};

// Utility functions
const calculateDuration = (start: string, end: string): string => {
  const duration = new Date(end).getTime() - new Date(start).getTime();
  const seconds = Math.floor(duration / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
};

const formatTimestamp = (ts: string): string => {
  return new Date(ts).toLocaleString();
};

export default AuditExplorer;
