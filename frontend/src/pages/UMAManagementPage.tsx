import React, { useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  Box,
  Container,
  Card,
  CardContent,
  Tabs,
  Tab,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Alert,
  CircularProgress,
  Chip,
  Grid,
  Typography,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Download as DownloadIcon,
  Share as ShareIcon,
  History as HistoryIcon,
  Assessment as AssessmentIcon,
} from '@mui/icons-material';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';
import UMABuilder from './UMABuilder';

interface WorkflowStatus {
  id: string;
  state: 'RUNNING' | 'COMPLETED' | 'FAILED';
  startTime: string;
  closeTime?: string;
  memo: Record<string, any>;
  lastResult?: any;
  reason?: string;
}

interface AuditLog {
  id: string;
  action: string;
  userId: string;
  timestamp: string;
  details: Record<string, any>;
  status: 'success' | 'failed';
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`tabpanel-${index}`}
      aria-labelledby={`tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

/**
 * UMA Management Page
 * 
 * Complete page for managing a single UMA account with:
 * - Visual portfolio builder
 * - Real-time rebalance status
 * - Approval workflow management
 * - Audit trail
 * - Historical rebalances
 */
export const UMAManagementPage: React.FC = () => {
  const { umaId } = useParams<{ umaId: string }>();

  const [tabValue, setTabValue] = useState(0);
  const [currentWorkflow, setCurrentWorkflow] = useState<string | null>(null);
  const [workflowDialogOpen, setWorkflowDialogOpen] = useState(false);
  const [shareDialogOpen, setShareDialogOpen] = useState(false);
  const [shareEmail, setShareEmail] = useState('');

  // Fetch UMA account basic info
  const { tenant, datasource } = useTenant();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';

  const { data: umaAccount, isLoading: umaLoading } = useQuery({
    queryKey: ['uma', umaId, tenantId, datasourceId],
    queryFn: async () => {
      if (!tenantId || !datasourceId) return null;
      const response = await fetch(
        `/api/uma/${umaId}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );
      return response.json();
    },
    enabled: !!umaId && !!tenantId && !!datasourceId,
  });

  // Fetch workflow status
  const { data: workflowStatus, refetch: refetchWorkflow } = useQuery<WorkflowStatus | undefined>({
    queryKey: ['workflow', currentWorkflow, tenantId, datasourceId],
    queryFn: async () => {
      if (!tenantId || !datasourceId) throw new Error('Tenant context missing');
      const response = await fetch(
        `/api/uma/rebalance/${currentWorkflow}/status`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );
      return response.json() as Promise<WorkflowStatus>;
    },
    enabled: !!currentWorkflow && !!tenantId && !!datasourceId,
    // Avoid referencing `workflowStatus` inside this initializer to prevent a TS block-scoped
    // declaration ordering error. We default to no automatic polling here; can reintroduce
    // dynamic polling via effect if needed.
    refetchInterval: false,
  });

  // Fetch audit logs
  const { data: auditLogs } = useQuery({
    queryKey: ['uma-audit', umaId, tenantId, datasourceId],
    queryFn: async () => {
      if (!tenantId || !datasourceId) return [];
      const response = await fetch(
        `/api/uma/${umaId}/audit`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );
      return response.json() as Promise<AuditLog[]>;
    },
    enabled: !!umaId && !!tenantId && !!datasourceId,
  });

  // Fetch rebalance history
  const { data: rebalanceHistory } = useQuery({
    queryKey: ['uma-history', umaId, tenantId, datasourceId],
    queryFn: async () => {
      if (!tenantId || !datasourceId) return [];
      const response = await fetch(
        `/api/uma/${umaId}/rebalance/history`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );
      return response.json() as Promise<any[]>;
    },
    enabled: !!umaId && !!tenantId && !!datasourceId,
  });

  // Share UMA mutation
  const shareMutation = useMutation({
    mutationFn: async () => {
      if (!tenantId || !datasourceId) throw new Error('Tenant context missing');
      const response = await fetch(
        `/api/uma/${umaId}/share`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify({
            recipientEmail: shareEmail,
            permissions: ['read'],
          }),
        }
      );
      return response.json();
    },
    onSuccess: () => {
      setShareDialogOpen(false);
      setShareEmail('');
    },
  });

  // Download report mutation
  const downloadMutation = useMutation({
    mutationFn: async () => {
      if (!tenantId || !datasourceId) throw new Error('Tenant context missing');
      const response = await fetch(
        `/api/uma/${umaId}/report`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `UMA-${umaId}-Report.pdf`;
      a.click();
    },
  });

  const handleRebalanceTriggered = (workflowId: string) => {
    setCurrentWorkflow(workflowId);
    setWorkflowDialogOpen(true);
  };

  const handleDownloadReport = () => {
    downloadMutation.mutate();
  };

  const handleSharePortfolio = () => {
    shareMutation.mutate();
  };

  // Narrow return type to valid Alert severity values to avoid `as any` casts at call sites
  const getWorkflowStatusColor = (status: string): 'info' | 'success' | 'error' | 'warning' => {
    switch (status) {
      case 'RUNNING':
        return 'info';
      case 'COMPLETED':
        return 'success';
      case 'FAILED':
        return 'error';
      default:
        return 'warning';
    }
  };

  if (umaLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 600 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!umaAccount) {
    return (
      <Container>
        <Alert severity="error">
          UMA Account not found. Please select a valid account.
        </Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Page Header */}
      <Box sx={{ mb: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
          <Box>
            <Typography variant="h4" gutterBottom>
              {umaAccount.name}
            </Typography>
            <Typography variant="subtitle1" color="textSecondary">
              Account ID: {umaId}
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={() => refetchWorkflow()}
            >
              Refresh
            </Button>
            <Button
              variant="outlined"
              startIcon={<DownloadIcon />}
              onClick={handleDownloadReport}
              disabled={downloadMutation.isPending}
            >
              Report
            </Button>
            <Button
              variant="outlined"
              startIcon={<ShareIcon />}
              onClick={() => setShareDialogOpen(true)}
            >
              Share
            </Button>
          </Box>
        </Box>

        {/* Account Summary */}
        <Grid container spacing={2}>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  AUM
                </Typography>
                <Typography variant="h6">
                  ${(umaAccount.aum / 1000000).toFixed(2)}M
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Status
                </Typography>
                <Chip label={umaAccount.status} color="primary" />
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Sleeves
                </Typography>
                <Typography variant="h6">
                  {umaAccount.sleeves?.length || 0}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Last Rebalanced
                </Typography>
                <Typography variant="body2">
                  {umaAccount.lastRebalanced
                    ? new Date(umaAccount.lastRebalanced).toLocaleDateString()
                    : 'Never'}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Box>

      {/* Workflow Status Alert */}
      {workflowStatus && (
        <Alert
          severity={getWorkflowStatusColor(workflowStatus.state) as any}
          sx={{ mb: 3 }}
          onClose={() => setCurrentWorkflow(null)}
        >
          <strong>Workflow {workflowStatus.state}</strong>
          {workflowStatus.state === 'RUNNING' && ' - Processing...'}
          {workflowStatus.state === 'FAILED' && ` - ${workflowStatus.reason}`}
          <Box sx={{ mt: 1 }}>
            Started: {new Date(workflowStatus.startTime).toLocaleString()}
          </Box>
        </Alert>
      )}

      {/* Tabs */}
      <Card>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={tabValue}
            onChange={(e, newValue) => setTabValue(newValue)}
            aria-label="UMA management tabs"
          >
            <Tab label="Portfolio Builder" icon={<AssessmentIcon />} iconPosition="start" />
            <Tab label="Rebalance History" icon={<HistoryIcon />} iconPosition="start" />
            <Tab label="Audit Trail" />
          </Tabs>
        </Box>

        <TabPanel value={tabValue} index={0}>
          <UMABuilder
            umaId={umaId}
            onRebalanceTriggered={handleRebalanceTriggered}
          />
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <Box sx={{ p: 3 }}>
            {rebalanceHistory && rebalanceHistory.length > 0 ? (
              <Box>
                {rebalanceHistory.map((history) => (
                  <Card key={history.id} sx={{ mb: 2 }}>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
                        <Box>
                          <Typography variant="subtitle1">
                            Rebalance {new Date(history.createdAt).toLocaleDateString()}
                          </Typography>
                          <Typography variant="body2" color="textSecondary">
                            {history.tradeCount} trades | Tax harvested: ${history.taxHarvested?.toFixed(2)}
                          </Typography>
                        </Box>
                        <Chip label={history.approvalStatus} />
                      </Box>
                    </CardContent>
                  </Card>
                ))}
              </Box>
            ) : (
              <Typography color="textSecondary">No rebalances yet</Typography>
            )}
          </Box>
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <Box sx={{ p: 3 }}>
            {auditLogs && auditLogs.length > 0 ? (
              <Box>
                {auditLogs.map((log) => (
                  <Card key={log.id} sx={{ mb: 1 }}>
                    <CardContent sx={{ pb: 1 }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
                        <Box>
                          <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                            {log.action}
                          </Typography>
                          <Typography variant="caption" color="textSecondary">
                            {log.userId} | {new Date(log.timestamp).toLocaleString()}
                          </Typography>
                        </Box>
                        <Chip
                          label={log.status}
                          color={log.status === 'success' ? 'success' : 'error'}
                          size="small"
                        />
                      </Box>
                    </CardContent>
                  </Card>
                ))}
              </Box>
            ) : (
              <Typography color="textSecondary">No audit logs</Typography>
            )}
          </Box>
        </TabPanel>
      </Card>

      {/* Workflow Status Dialog */}
      <Dialog
        open={workflowDialogOpen}
        onClose={() => setWorkflowDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Rebalance Workflow Status</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {workflowStatus ? (
            <Box>
              <Typography variant="body2" gutterBottom>
                <strong>Status:</strong> {workflowStatus.state}
              </Typography>
              <Typography variant="body2" gutterBottom>
                <strong>Workflow ID:</strong> {workflowStatus.id}
              </Typography>
              <Typography variant="body2" gutterBottom>
                <strong>Started:</strong> {new Date(workflowStatus.startTime).toLocaleString()}
              </Typography>
              {workflowStatus.closeTime && (
                <Typography variant="body2" gutterBottom>
                  <strong>Completed:</strong> {new Date(workflowStatus.closeTime).toLocaleString()}
                </Typography>
              )}
              {workflowStatus.state === 'RUNNING' && (
                <Box sx={{ mt: 2, display: 'flex', justifyContent: 'center' }}>
                  <CircularProgress />
                </Box>
              )}
            </Box>
          ) : (
            <CircularProgress />
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setWorkflowDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Share Dialog */}
      <Dialog open={shareDialogOpen} onClose={() => setShareDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Share Portfolio</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <TextField
            fullWidth
            label="Email Address"
            type="email"
            value={shareEmail}
            onChange={(e) => setShareEmail(e.target.value)}
            placeholder="recipient@example.com"
          />
          <Typography variant="caption" color="textSecondary" sx={{ mt: 1, display: 'block' }}>
            They will receive read-only access to this portfolio
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShareDialogOpen(false)}>Cancel</Button>
          <Button
            onClick={handleSharePortfolio}
            variant="contained"
            disabled={!shareEmail || shareMutation.isPending}
          >
            Share
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};

export default UMAManagementPage;
