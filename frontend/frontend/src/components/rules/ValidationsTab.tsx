import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  InputAdornment,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  Stack,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  Divider,
  useTheme
} from '@mui/material';
import {
  Add as PlusIcon,
  Code as CodeBracketIcon,
  FileUpload as ArrowUpOnSquareIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as XCircleIcon,
  History as HistoryIcon,
  Timeline as TimelineIcon,
  Difference as DifferenceIcon,
  Search as SearchIcon,
  FilterList as FilterListIcon,
  VerifiedUser as ShieldCheckIcon,
  Warning as WarningIcon,
  Info as InfoIcon,
  Error as ErrorIcon,
  Visibility as VisibilityIcon
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext'; // Ensure this path is correct
import { rulesApi, createBusinessObjectRule } from '../../services/rulesApi';
import { ValidationRule, Rule } from '../../types/rules';
import { PreviewSQLModal } from './PreviewSQLModal';
import { CreateRuleModal } from './CreateRuleModal';
import { RuleDiffViewer } from './RuleDiffViewer';
import { RuleHistory } from './RuleHistory';
import { RuleLineagePanel } from './RuleLineagePanel';
import PromotionImpactModal from './PromotionImpactModal';
import PromotionImpactModal from './PromotionImpactModal';

export const ValidationsTab: React.FC = () => {
  const { id: boId } = useParams<{ id: string }>();
  const { tenant, datasource } = useTenant();
  const theme = useTheme();

  // State
  const [rules, setRules] = useState<ValidationRule[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Preview Modal State
  const [isPreviewOpen, setIsPreviewOpen] = useState(false);
  const [previewSql, setPreviewSql] = useState<string | undefined>();
  const [previewError, setPreviewError] = useState<string | undefined>();
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);

  // Promotion State
  const [promotingRuleId, setPromotingRuleId] = useState<string | null>(null);
  const [promotionImpact, setPromotionImpact] = useState<any | null>(null);
  const [promotionOpen, setPromotionOpen] = useState(false);

  // Creation State
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);

  // Filter State
  const [searchQuery, setSearchQuery] = useState('');
  const [scopeFilter, setScopeFilter] = useState<string>('all');
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  // Side Panels
  const [diffRuleId, setDiffRuleId] = useState<string | null>(null);
  const [historyRuleId, setHistoryRuleId] = useState<string | null>(null);
  const [lineageRuleId, setLineageRuleId] = useState<string | null>(null);

  // Computed
  const filteredRules = (rules || []).filter(rule => {
    const searchLower = searchQuery.toLowerCase();
    const matchesSearch = rule.name.toLowerCase().includes(searchLower) || 
                          (rule.expression || '').toLowerCase().includes(searchLower);
    const matchesScope = scopeFilter === 'all' || rule.scope === scopeFilter;
    const matchesSeverity = severityFilter === 'all' || rule.severity === severityFilter;
    const matchesStatus = statusFilter === 'all' || rule.promotionStatus === statusFilter;
    return matchesSearch && matchesScope && matchesSeverity && matchesStatus;
  });

  useEffect(() => {
    if (boId) {
      loadRules(boId);
    }
  }, [boId]);

  // Polling for transient statuses
  useEffect(() => {
    if (!rules || rules.length === 0 || !boId) return;
    const transientStatuses = ['pending_approval', 'promoting'];
    const hasTransient = rules.some(r => transientStatuses.includes(r.promotionStatus || ''));
    
    if (hasTransient) {
      const interval = setInterval(() => {
        loadRules(boId);
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [rules, boId]);

  const loadRules = async (id: string) => {
    try {
      setIsLoading(true);
      const data = await rulesApi.fetchBOValidations(id, tenant?.id, datasource?.id);
      // Deduplicate rules by ID to prevent React key warnings
      const uniqueRules = Array.from(new Map((data || []).map((r: any) => [r.id, r])).values());
      setRules(uniqueRules);
      setError(null);
    } catch (err) {
      console.error('Failed to load rules:', err);
      setError('Failed to load validation rules.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateRule = async (rule: Partial<Rule>) => {
    if (!boId) return;
    setIsCreating(true);
    try {
      await createBusinessObjectRule(boId, rule, tenant?.id, datasource?.id);
      setIsCreateModalOpen(false);
      loadRules(boId);
    } catch (err) {
      console.error('Failed to create rule:', err);
      alert('Failed to create rule.'); // Consider replacing with Snackbar
    } finally {
      setIsCreating(false);
    }
  };

  const handlePreview = async (ruleId: string) => {
    if (!boId) return;
    setIsPreviewOpen(true);
    setPreviewSql(undefined);
    setPreviewError(undefined);
    setIsPreviewLoading(true);

    try {
      const result = await rulesApi.fetchValidationPreviewSQL(boId, ruleId, tenant?.id, datasource?.id);
      setPreviewSql(result.generated_sql);
    } catch (err) {
      console.error('Preview failed:', err);
      setPreviewError('Failed to generate SQL preview.');
    } finally {
      setIsPreviewLoading(false);
    }
  };

  const handlePromote = async (ruleId: string) => {
    if (!boId) return;

    try {
      const impact = await rulesApi.fetchPromotionImpact(ruleId, tenant?.id, datasource?.id);
      setPromotionImpact(impact);
      setPromotionOpen(true);
      // store id for confirmation
      setPromotingRuleId(ruleId);
    } catch (err) {
      console.error('Failed to fetch promotion impact:', err);
      alert('Failed to fetch promotion impact summary.');
    }
  };

  const handleConfirmPromote = async () => {
    if (!boId || !promotingRuleId) return;
    setPromotionOpen(false);
    setPromotingRuleId(promotingRuleId);
    try {
      await rulesApi.promoteValidationRuleToCore(boId, promotingRuleId, tenant?.id, datasource?.id);
      loadRules(boId); // Refresh to see status change
    } catch (err) {
      console.error('Promotion failed:', err);
      alert('Failed to start promotion workflow.');
    } finally {
      setPromotingRuleId(null);
    }
  };


  const handleApprove = async (ruleId: string) => {
    if (!boId) return;
    const comment = window.prompt('Optional approval comment:');
    if (comment === null) return;

    try {
      await rulesApi.approveRule(boId, ruleId, comment || undefined, tenant?.id, datasource?.id);
      loadRules(boId);
    } catch (err) {
      console.error('Approval failed:', err);
      alert('Failed to submit approval.');
    }
  };

  const handleDeny = async (ruleId: string) => {
    if (!boId) return;
    const comment = window.prompt('Reason for denial (required):');
    if (!comment) return;

    try {
      await rulesApi.denyRule(boId, ruleId, comment, tenant?.id, datasource?.id);
      loadRules(boId);
    } catch (err) {
      console.error('Denial failed:', err);
      alert('Failed to submit denial.');
    }
  };

  // Render Helpers
  const getSeverityChip = (severity: string) => {
    const props: any = { size: 'small', label: severity, variant: 'outlined' };
    switch (severity?.toLowerCase()) {
      case 'error':
        props.color = 'error';
        props.icon = <ErrorIcon fontSize="small" />;
        break;
      case 'warning':
        props.color = 'warning';
        props.icon = <WarningIcon fontSize="small" />;
        break;
      case 'info':
        props.color = 'info';
        props.icon = <InfoIcon fontSize="small" />;
        break;
      default:
        props.color = 'default';
    }
    return <Chip {...props} sx={{ textTransform: 'capitalize' }} />;
  };

  const getScopeChip = (scope: string) => {
    let color: 'default' | 'primary' | 'secondary' | 'success' = 'default';
    let label = scope;

    switch (scope?.toLowerCase()) {
      case 'inherited':
        color = 'primary';
        label = 'Inherited';
        break;
      case 'override':
        color = 'secondary';
        label = 'Override';
        break;
      case 'local':
        color = 'success';
        label = 'Local';
        break;
    }
    return <Chip label={label} color={color} size="small" variant="filled" />;
  };

  const getStatusChip = (status?: string) => {
    if (!status || status === 'none' || status === 'promoted') return null;
    let color: 'default' | 'warning' | 'info' | 'error' = 'default';
    let label = status;

    switch (status) {
      case 'pending_approval':
        color = 'warning';
        label = 'Pending Approval';
        break;
      case 'promoting':
        color = 'info';
        label = 'Promoting...';
        break;
      case 'failed':
        color = 'error';
        label = 'Failed';
        break;
    }
    return <Chip label={label} color={color} size="small" sx={{ ml: 1 }} />;
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Promotion Impact Modal */}
      <PromotionImpactModal open={promotionOpen} impact={promotionImpact} onClose={() => setPromotionOpen(false)} onConfirm={handleConfirmPromote} />

      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h5" fontWeight="bold" gutterBottom>
            Validation Rules
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage data quality rules, including global policies and local constraints.
          </Typography>
        </Box>
        <Button
          variant="contained"
          color="primary"
          startIcon={<PlusIcon />}
          onClick={() => setIsCreateModalOpen(true)}
          sx={{ textTransform: 'none', fontWeight: 600 }}
        >
          Add Rule
        </Button>
      </Stack>

      {/* Filters */}
      <Card sx={{ mb: 3, boxShadow: 1 }}>
        <CardContent sx={{ py: 2 }}>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              fullWidth
              size="small"
              placeholder="Search by name or expression..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon color="action" />
                  </InputAdornment>
                ),
              }}
            />
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel>Scope</InputLabel>
              <Select
                value={scopeFilter}
                label="Scope"
                onChange={(e) => setScopeFilter(e.target.value)}
              >
                <MenuItem value="all">All Scopes</MenuItem>
                <MenuItem value="local">Local</MenuItem>
                <MenuItem value="inherited">Inherited</MenuItem>
                <MenuItem value="override">Override</MenuItem>
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel>Severity</InputLabel>
              <Select
                value={severityFilter}
                label="Severity"
                onChange={(e) => setSeverityFilter(e.target.value)}
              >
                <MenuItem value="all">All Severities</MenuItem>
                <MenuItem value="error">Error</MenuItem>
                <MenuItem value="warning">Warning</MenuItem>
                <MenuItem value="info">Info</MenuItem>
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel>Status</InputLabel>
              <Select
                value={statusFilter}
                label="Status"
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <MenuItem value="all">All Statuses</MenuItem>
                <MenuItem value="none">Local Only</MenuItem>
                <MenuItem value="pending_approval">Pending Approval</MenuItem>
                <MenuItem value="promoting">Promoting</MenuItem>
                <MenuItem value="failed">Failed</MenuItem>
              </Select>
            </FormControl>
          </Stack>
        </CardContent>
      </Card>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {/* Rules Table */}
      <TableContainer component={Paper} elevation={2} sx={{ borderRadius: 2 }}>
        <Table sx={{ minWidth: 650 }} aria-label="validation rules">
          <TableHead sx={{ bgcolor: theme.palette.grey[50] }}>
            <TableRow>
              <TableCell sx={{ fontWeight: 600 }}>Name</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Expression</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Scope</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Severity</TableCell>
              <TableCell align="right" sx={{ fontWeight: 600 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredRules.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center" sx={{ py: 8 }}>
                   <Typography color="text.secondary">
                     No validation rules found matching your criteria.
                   </Typography>
                </TableCell>
              </TableRow>
            ) : filteredRules.map((rule) => (
              <TableRow
                key={rule.id}
                sx={{ '&:last-child td, &:last-child th': { border: 0 }, '&:hover': { bgcolor: theme.palette.action.hover } }}
              >
                <TableCell component="th" scope="row">
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <Typography variant="subtitle2" fontWeight={600}>
                       {rule.name}
                    </Typography>
                    {rule.source === 'semantic_term' && (
                       <Tooltip title="Derived from Semantic Term">
                         <ShieldCheckIcon fontSize="small" color="primary" sx={{ opacity: 0.7 }} />
                       </Tooltip>
                    )}
                    {getStatusChip(rule.promotionStatus)}
                  </Stack>
                </TableCell>
                <TableCell>
                  <Tooltip title={rule.expression || "No expression"}>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', maxWidth: 300 }} noWrap>
                      {rule.expression || <span style={{ fontStyle: 'italic', color: 'gray' }}>No expression</span>}
                    </Typography>
                  </Tooltip>
                </TableCell>
                <TableCell>{getScopeChip(rule.scope)}</TableCell>
                <TableCell>{getSeverityChip(rule.severity)}</TableCell>
                <TableCell align="right">
                  <Stack direction="row" justifyContent="flex-end" spacing={1}>
                    <Tooltip title="Preview SQL">
                      <IconButton size="small" onClick={() => handlePreview(rule.id)}>
                        <CodeBracketIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="View Diff">
                      <IconButton size="small" onClick={() => setDiffRuleId(rule.id)}>
                        <DifferenceIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="History">
                      <IconButton size="small" onClick={() => setHistoryRuleId(rule.id)}>
                        <HistoryIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Lineage">
                      <IconButton size="small" onClick={() => setLineageRuleId(rule.id)}>
                        <TimelineIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>

                    {rule.promotionStatus === 'pending_approval' && (
                      <>
                        <Tooltip title="Approve">
                          <IconButton size="small" color="success" onClick={() => handleApprove(rule.id)}>
                            <CheckCircleIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Deny">
                          <IconButton size="small" color="error" onClick={() => handleDeny(rule.id)}>
                            <XCircleIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </>
                    )}

                    {rule.scope === 'local' && (!rule.promotionStatus || rule.promotionStatus === 'none' || rule.promotionStatus === 'failed') && (
                       <Tooltip title="Promote to Core">
                         <IconButton 
                           size="small" 
                           color="primary" 
                           onClick={() => handlePromote(rule.id)}
                           disabled={promotingRuleId === rule.id}
                         >
                           {promotingRuleId === rule.id ? <CircularProgress size={20} /> : <ArrowUpOnSquareIcon fontSize="small" />}
                         </IconButton>
                       </Tooltip>
                    )}
                  </Stack>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Modals */}
      <PreviewSQLModal
        isOpen={isPreviewOpen}
        onClose={() => setIsPreviewOpen(false)}
        isLoading={isPreviewLoading}
        sql={previewSql}
        error={previewError}
      />
      <CreateRuleModal 
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSubmit={handleCreateRule}
        isLoading={isCreating}
        boId={boId || ''}
      />
      <RuleDiffViewer 
        isOpen={!!diffRuleId}
        onClose={() => setDiffRuleId(null)}
        boId={boId || ''}
        ruleId={diffRuleId || ''}
      />
      <RuleHistory
        isOpen={!!historyRuleId}
        onClose={() => setHistoryRuleId(null)}
        boId={boId || ''}
        ruleId={historyRuleId || ''}
      />
      <RuleLineagePanel
        isOpen={!!lineageRuleId}
        onClose={() => setLineageRuleId(null)}
        boId={boId || ''}
        ruleId={lineageRuleId || ''}
        ruleName={rules.find(r => r.id === lineageRuleId)?.name || ''}
      />
    </Box>
  );
};
