import React, { useState } from 'react';
import {
  Box,
  Container,
  Typography,
  Card,
  CardContent,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Tooltip,
  LinearProgress,
  Alert,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Stack,
  Tabs,
  Tab,
  Badge,
} from '@mui/material';
import {
  AutoFixHigh,
  CheckCircle,
  Cancel,
  Pending,
  TrendingUp,
  Speed,
  DeleteOutline,
  Refresh,
  PlayArrow,
} from '@mui/icons-material';
import {
  useASOSummary,
  useASOOptimizations,
  useASOPolicies,
  applyASOOptimization,
  approveASOOptimization,
  rejectASOOptimization,
  triggerASOEvaluation,
  ASOOptimization,
} from '../hooks/useASO';
import OptimizationActions from '../components/aso/OptimizationActions';

// ============================================================================
// Main Optimization Center Page
// ============================================================================

interface OptimizationCenterProps {
  scope?: 'global' | 'tenant';
  tenantId?: string;
}

export const OptimizationCenter: React.FC<OptimizationCenterProps> = ({ scope = 'global', tenantId }) => {
  const [selectedEnv, setSelectedEnv] = useState<string>('prod');
  const [tabValue, setTabValue] = useState(0);

  // If tenant scope, we might want to restrict env or pass tenantId to hooks
  const { summaries, loading: summaryLoading, refresh: refreshSummary } = useASOSummary();
  const { optimizations, loading: optLoading, refresh: refreshOpts } = useASOOptimizations({
    env: selectedEnv,
    limit: 50,
    tenantId: scope === 'tenant' ? tenantId : undefined, // Pass tenantId if scoped
  });
  const { policies, loading: policyLoading } = useASOPolicies(selectedEnv);

  const handleRefresh = () => {
    refreshSummary();
    refreshOpts();
  };

  const handleTriggerEvaluation = async () => {
    await triggerASOEvaluation(selectedEnv);
    refreshOpts();
  };

  const envSummary = summaries[selectedEnv];
  const pendingOpts = optimizations.filter(o => o.status === 'proposed');
  const appliedOpts = optimizations.filter(o => o.status === 'applied');

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={4}>
        <Stack direction="row" alignItems="center" spacing={2}>
          <AutoFixHigh sx={{ fontSize: 40, color: 'primary.main' }} />
          <Box>
            <Typography variant="h4" fontWeight="bold">
              Optimization Center
            </Typography>
            <Typography color="text.secondary">
              Autonomous Semantic Optimization (ASO)
            </Typography>
          </Box>
        </Stack>
        <Stack direction="row" spacing={2}>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Environment</InputLabel>
            <Select
              value={selectedEnv}
              label="Environment"
              onChange={(e) => setSelectedEnv(e.target.value)}
            >
              <MenuItem value="dev">Development</MenuItem>
              <MenuItem value="staging">Staging</MenuItem>
              <MenuItem value="prod">Production</MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="outlined"
            startIcon={<Refresh />}
            onClick={handleRefresh}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<PlayArrow />}
            onClick={handleTriggerEvaluation}
          >
            Run Evaluation
          </Button>
        </Stack>
      </Stack>

      {(summaryLoading || optLoading) && <LinearProgress sx={{ mb: 2 }} />}

      {/* Summary Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} md={3}>
          <SummaryCard
            title="ASO Status"
            value={envSummary?.policy_enabled ? 'Enabled' : 'Disabled'}
            subtitle={`Mode: ${envSummary?.policy_mode || 'N/A'}`}
            color={envSummary?.policy_enabled ? 'success' : 'default'}
            icon={<AutoFixHigh />}
          />
        </Grid>
        <Grid item xs={12} md={3}>
          <SummaryCard
            title="Pending Optimizations"
            value={envSummary?.optimizations_pending || 0}
            subtitle="Awaiting review"
            color="warning"
            icon={<Pending />}
          />
        </Grid>
        <Grid item xs={12} md={3}>
          <SummaryCard
            title="Applied (7d)"
            value={envSummary?.optimizations_applied_7d || 0}
            subtitle="Last 7 days"
            color="success"
            icon={<CheckCircle />}
          />
        </Grid>
        <Grid item xs={12} md={3}>
          <SummaryCard
            title="Hot Paths Detected"
            value={envSummary?.hot_paths_detected || 0}
            subtitle="Optimization candidates"
            color="info"
            icon={<TrendingUp />}
          />
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)}>
          <Tab
            label={
              <Badge badgeContent={pendingOpts.length} color="warning">
                Pending
              </Badge>
            }
          />
          <Tab label="Applied" />
          <Tab label="All Optimizations" />
          <Tab label="Policies" />
        </Tabs>
      </Paper>

      {/* Tab Content */}
      {tabValue === 0 && (
        <OptimizationTable
          optimizations={pendingOpts}
          onRefresh={refreshOpts}
          showActions
        />
      )}
      {tabValue === 1 && (
        <OptimizationTable optimizations={appliedOpts} onRefresh={refreshOpts} />
      )}
      {tabValue === 2 && (
        <OptimizationTable optimizations={optimizations} onRefresh={refreshOpts} />
      )}
      {tabValue === 3 && <PolicyTable policies={policies} />}
    </Container>
  );
};

// ============================================================================
// Summary Card Component
// ============================================================================

interface SummaryCardProps {
  title: string;
  value: string | number;
  subtitle: string;
  color: 'success' | 'warning' | 'error' | 'info' | 'default';
  icon: React.ReactNode;
}

const SummaryCard: React.FC<SummaryCardProps> = ({
  title,
  value,
  subtitle,
  color,
  icon,
}) => {
  const colorMap = {
    success: '#4caf50',
    warning: '#ff9800',
    error: '#f44336',
    info: '#2196f3',
    default: '#9e9e9e',
  };

  return (
    <Card>
      <CardContent>
        <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
          <Box>
            <Typography variant="caption" color="text.secondary">
              {title}
            </Typography>
            <Typography variant="h4" fontWeight="bold" sx={{ color: colorMap[color] }}>
              {value}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {subtitle}
            </Typography>
          </Box>
          <Box sx={{ color: colorMap[color], opacity: 0.3 }}>{icon}</Box>
        </Stack>
      </CardContent>
    </Card>
  );
};

// ============================================================================
// Optimization Table Component
// ============================================================================

interface OptimizationTableProps {
  optimizations: ASOOptimization[];
  onRefresh: () => void;
  showActions?: boolean;
}

const OptimizationTable: React.FC<OptimizationTableProps> = ({
  optimizations,
  onRefresh,
  showActions = false,
}) => {
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const handleApply = async (id: string) => {
    setActionLoading(id);
    try {
      await applyASOOptimization(id);
      onRefresh();
    } finally {
      setActionLoading(null);
    }
  };

  const handleApprove = async (id: string) => {
    setActionLoading(id);
    try {
      await approveASOOptimization(id);
      onRefresh();
    } finally {
      setActionLoading(null);
    }
  };

  const handleReject = async (id: string) => {
    setActionLoading(id);
    try {
      await rejectASOOptimization(id, 'Rejected by user');
      onRefresh();
    } finally {
      setActionLoading(null);
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'create_preagg':
        return <TrendingUp fontSize="small" color="primary" />;
      case 'tune_refresh':
        return <Speed fontSize="small" color="info" />;
      case 'retire_asset':
        return <DeleteOutline fontSize="small" color="error" />;
      default:
        return <AutoFixHigh fontSize="small" />;
    }
  };

  const getStatusChip = (status: string) => {
    const config = {
      proposed: { color: 'warning' as const, label: 'Proposed' },
      approved: { color: 'info' as const, label: 'Approved' },
      applied: { color: 'success' as const, label: 'Applied' },
      rejected: { color: 'error' as const, label: 'Rejected' },
      failed: { color: 'error' as const, label: 'Failed' },
      superseded: { color: 'default' as const, label: 'Superseded' },
    };
    const c = config[status as keyof typeof config] || { color: 'default' as const, label: status };
    return <Chip label={c.label} color={c.color} size="small" />;
  };

  if (optimizations.length === 0) {
    return (
      <Alert severity="info">
        No optimizations found. Run an evaluation to discover optimization opportunities.
      </Alert>
    );
  }

  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Type</TableCell>
            <TableCell>Target</TableCell>
            <TableCell>Reason</TableCell>
            <TableCell>Score</TableCell>
            <TableCell>Scope</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Created</TableCell>
            {showActions && <TableCell align="center">Actions</TableCell>}
          </TableRow>
        </TableHead>
        <TableBody>
          {optimizations.map((opt) => (
            <TableRow key={opt.id} hover>
              <TableCell>
                <Stack direction="row" spacing={1} alignItems="center">
                  {getTypeIcon(opt.optimization_type)}
                  <Typography variant="body2">
                    {opt.optimization_type.replace(/_/g, ' ')}
                  </Typography>
                </Stack>
              </TableCell>
              <TableCell>
                <Typography variant="body2" fontWeight="medium">
                  {opt.target_name}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {opt.target_type}
                </Typography>
              </TableCell>
              <TableCell>
                <Typography variant="body2" sx={{ maxWidth: 300 }} noWrap>
                  {opt.reason}
                </Typography>
              </TableCell>
              <TableCell>
                <Chip
                  label={opt.score.toFixed(2)}
                  size="small"
                  color={opt.score >= 0.8 ? 'success' : opt.score >= 0.5 ? 'warning' : 'default'}
                />
              </TableCell>
              <TableCell>
                <Chip label={opt.scope} size="small" variant="outlined" />
              </TableCell>
              <TableCell>{getStatusChip(opt.status)}</TableCell>
              <TableCell>
                <Typography variant="caption">
                  {new Date(opt.created_at).toLocaleDateString()}
                </Typography>
              </TableCell>
              {showActions && (
                <TableCell align="center">
                  <OptimizationActions
                    optimization={opt}
                    compact
                    onApply={() => handleApply(opt.id)}
                    onApprove={() => handleApprove(opt.id)}
                    onReject={async (reason) => {
                      // OptimizationActions handles the dialog, but we need to map the call
                      await rejectASOOptimization(opt.id, reason);
                      onRefresh();
                    }}
                  />
                </TableCell>
              )}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

// ============================================================================
// Policy Table Component
// ============================================================================

interface PolicyTableProps {
  policies: Array<{
    id: string;
    env: string;
    tenant_id?: string;
    enabled: boolean;
    mode: string;
    max_new_preaggs_per_day: number;
    max_changes_per_day: number;
  }>;
}

const PolicyTable: React.FC<PolicyTableProps> = ({ policies }) => {
  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Environment</TableCell>
            <TableCell>Scope</TableCell>
            <TableCell>Enabled</TableCell>
            <TableCell>Mode</TableCell>
            <TableCell>Max New Pre-Aggs/Day</TableCell>
            <TableCell>Max Changes/Day</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {policies.map((policy) => (
            <TableRow key={policy.id} hover>
              <TableCell>
                <Chip label={policy.env} size="small" />
              </TableCell>
              <TableCell>
                {policy.tenant_id ? (
                  <Chip label="Tenant Override" size="small" variant="outlined" />
                ) : (
                  <Chip label="Core" size="small" color="primary" />
                )}
              </TableCell>
              <TableCell>
                {policy.enabled ? (
                  <CheckCircle color="success" />
                ) : (
                  <Cancel color="disabled" />
                )}
              </TableCell>
              <TableCell>
                <Chip
                  label={policy.mode}
                  size="small"
                  color={
                    policy.mode === 'auto_apply'
                      ? 'success'
                      : policy.mode === 'auto_tune'
                      ? 'warning'
                      : 'default'
                  }
                />
              </TableCell>
              <TableCell>{policy.max_new_preaggs_per_day}</TableCell>
              <TableCell>{policy.max_changes_per_day}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default OptimizationCenter;
