import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Container,
  Grid,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Tooltip,
  Alert,
  Checkbox,
  Menu,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Security as SecurityIcon,
  Group as GroupIcon,
  Business as BusinessIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
  GetApp as ExportIcon,
  Publish as ImportIcon,
  Help as HelpIcon,
  MoreVert as MoreIcon,
} from '@mui/icons-material';
import { accessRulesApi, AccessRule, RuleStatus } from '../../../api/accessRules';
import { BulkActionsMenu } from '../components/BulkActionsMenu';
import { ImportWizard } from '../components/ImportWizard';
import { HelpDrawer } from '../components/HelpDrawer';

const statuses: RuleStatus[] = ['DRAFT', 'REVIEW', 'APPROVED', 'DEPRECATED'];

const statusColors: Record<RuleStatus, 'default' | 'warning' | 'info' | 'success' | 'error'> = {
  DRAFT: 'default',
  REVIEW: 'warning',
  APPROVED: 'success',
  DEPRECATED: 'error',
};

export const AccessRulesDashboard: React.FC = () => {
  const navigate = useNavigate();
  const [rules, setRules] = useState<AccessRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Filters
  const [boFilter, setBoFilter] = useState('');
  const [groupFilter, setGroupFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<RuleStatus | ''>('');

  // Bulk operations
  const [selectedRules, setSelectedRules] = useState<Set<string>>(new Set());
  const [bulkMenuAnchor, setBulkMenuAnchor] = useState<HTMLElement | null>(null);

  // Dialogs
  const [importOpen, setImportOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);

  const loadRules = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await accessRulesApi.list({
        businessObjectId: boFilter || undefined,
        groupDn: groupFilter || undefined,
        status: statusFilter || undefined,
      });
      setRules(data);
    } catch (e: any) {
      setError(e?.message || 'Failed to load access rules');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadRules();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleClearFilters = () => {
    setBoFilter('');
    setGroupFilter('');
    setStatusFilter('');
    void loadRules();
  };

  // Metrics
  const totalRules = rules.length;
  const activeRules = rules.filter(r => r.status === 'APPROVED').length;
  const pendingReview = rules.filter(r => r.status === 'REVIEW').length;
  const draftRules = rules.filter(r => r.status === 'DRAFT').length;

  const filteredRules = rules;

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 4 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700, display: 'flex', alignItems: 'center', gap: 1 }}>
            <SecurityIcon fontSize="large" color="primary" />
            Access Control
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            Manage data access rules for teams and user groups
          </Typography>
        </Box>
        <Button
          variant="contained"
          size="large"
          startIcon={<AddIcon />}
          onClick={() => navigate('/security/access-rules/wizard')}
        >
          Create New Rule
        </Button>
      </Stack>

      {/* Metrics Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Total Rules
                  </Typography>
                  <Typography variant="h4" sx={{ fontWeight: 700, mt: 1 }}>
                    {totalRules}
                  </Typography>
                </Box>
                <SecurityIcon sx={{ fontSize: 48, color: 'primary.main', opacity: 0.3 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Active
                  </Typography>
                  <Typography variant="h4" sx={{ fontWeight: 700, mt: 1, color: 'success.main' }}>
                    {activeRules}
                  </Typography>
                </Box>
                <SecurityIcon sx={{ fontSize: 48, color: 'success.main', opacity: 0.3 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Pending Review
                  </Typography>
                  <Typography variant="h4" sx={{ fontWeight: 700, mt: 1, color: 'warning.main' }}>
                    {pendingReview}
                  </Typography>
                </Box>
                <SecurityIcon sx={{ fontSize: 48, color: 'warning.main', opacity: 0.3 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Drafts
                  </Typography>
                  <Typography variant="h4" sx={{ fontWeight: 700, mt: 1 }}>
                    {draftRules}
                  </Typography>
                </Box>
                <SecurityIcon sx={{ fontSize: 48, color: 'text.disabled', opacity: 0.3 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Filters */}
      <Paper elevation={1} sx={{ p: 3, mb: 3 }}>
        <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 2 }}>
          <FilterIcon color="action" />
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            Filters
          </Typography>
        </Stack>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              size="small"
              label="Data Type"
              placeholder="e.g., Portfolio, Client"
              value={boFilter}
              onChange={(e) => setBoFilter(e.target.value)}
              InputProps={{
                startAdornment: <BusinessIcon sx={{ mr: 1, color: 'action.active' }} />,
              }}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              size="small"
              label="Team/User Group"
              placeholder="e.g., Finance Team"
              value={groupFilter}
              onChange={(e) => setGroupFilter(e.target.value)}
              InputProps={{
                startAdornment: <GroupIcon sx={{ mr: 1, color: 'action.active' }} />,
              }}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControl fullWidth size="small">
              <InputLabel>Status</InputLabel>
              <Select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as RuleStatus | '')}
                label="Status"
              >
                <MenuItem value="">All</MenuItem>
                {statuses.map((s) => (
                  <MenuItem key={s} value={s}>
                    {s}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={3}>
            <Stack direction="row" spacing={1}>
              <Button
                variant="contained"
                onClick={() => void loadRules()}
                disabled={loading}
                fullWidth
              >
                Apply
              </Button>
              <Button variant="outlined" onClick={handleClearFilters} fullWidth>
                Clear
              </Button>
              <IconButton onClick={() => void loadRules()} disabled={loading}>
                <RefreshIcon />
              </IconButton>
            </Stack>
          </Grid>
        </Grid>
      </Paper>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Rules Table */}
      <Paper elevation={2}>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: 'grey.50' }}>
                <TableCell sx={{ fontWeight: 700 }}>Data Type</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Team/User Group</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Access Level</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Row Filters</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Field Masks</TableCell>
                <TableCell sx={{ fontWeight: 700 }} align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading && (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">Loading rules...</Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading && filteredRules.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      No access rules found. Create your first rule to get started.
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {!loading &&
                filteredRules.map((rule) => (
                  <TableRow key={rule.ruleId} hover>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {rule.businessObjectId}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Tooltip title={rule.groupDn}>
                        <Typography variant="body2" noWrap sx={{ maxWidth: 200 }}>
                          {rule.groupDn}
                        </Typography>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={rule.accessLevel}
                        size="small"
                        color={rule.accessLevel === 'WRITE' ? 'primary' : 'default'}
                      />
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={rule.status}
                        size="small"
                        color={statusColors[rule.status]}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {rule.rowFilterDsl ? '✓ Active' : '—'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {rule.columnMasks?.length || 0} fields
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip title="Edit Rule">
                        <IconButton
                          size="small"
                          color="primary"
                          onClick={() => navigate(`/security/access-rules/${rule.ruleId}`)}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Container>
  );
};

export default AccessRulesDashboard;
