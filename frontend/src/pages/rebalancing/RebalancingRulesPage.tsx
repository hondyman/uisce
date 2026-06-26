import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Container,
  Typography,
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  Tooltip,
  CircularProgress,
  Alert,
  Tabs,
  Tab,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  ContentCopy as DuplicateIcon,
} from '@mui/icons-material';
import { RDLRuleBuilder, RuleDefinition } from '../../components/rebalancing';
import { useTenant as useTenantContext } from '../../contexts/TenantContext';

// =============================================================================
// API FUNCTIONS
// =============================================================================

const API_BASE = '/api/rdl';

async function fetchRules(tenantId: string, datasourceId: string): Promise<RuleDefinition[]> {
  const response = await fetch(
    `${API_BASE}/rules?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
    {
      headers: {
        'X-Tenant-ID': tenantId,
        'X-Tenant-Datasource-ID': datasourceId,
      },
    }
  );
  if (!response.ok) throw new Error('Failed to fetch rules');
  const data = await response.json();
  return data.rules || [];
}

async function createRule(rule: RuleDefinition, datasourceId: string): Promise<RuleDefinition> {
  const response = await fetch(`${API_BASE}/rules?tenant_instance_id=${datasourceId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': rule.tenant_id,
      'X-Tenant-Datasource-ID': datasourceId,
    },
    body: JSON.stringify(rule),
  });
  if (!response.ok) throw new Error('Failed to create rule');
  return response.json();
}

async function updateRule(rule: RuleDefinition, datasourceId: string): Promise<RuleDefinition> {
  const response = await fetch(`${API_BASE}/rules/${rule.rule_id}?tenant_instance_id=${datasourceId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': rule.tenant_id,
      'X-Tenant-Datasource-ID': datasourceId,
    },
    body: JSON.stringify(rule),
  });
  if (!response.ok) throw new Error('Failed to update rule');
  return response.json();
}

async function deleteRule(tenantId: string, ruleId: string, datasourceId: string): Promise<void> {
  const response = await fetch(
    `${API_BASE}/rules/${ruleId}?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
    {
      method: 'DELETE',
      headers: {
        'X-Tenant-ID': tenantId,
        'X-Tenant-Datasource-ID': datasourceId,
      },
    }
  );
  if (!response.ok) throw new Error('Failed to delete rule');
}

async function testRule(rule: RuleDefinition, datasourceId: string): Promise<{ passed: boolean; score?: number; message?: string }> {
  const response = await fetch(`${API_BASE}/validate?tenant_instance_id=${datasourceId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': rule.tenant_id,
      'X-Tenant-Datasource-ID': datasourceId,
    },
    body: JSON.stringify({
      expression: rule.expression,
      parameters: rule.parameters,
    }),
  });
  if (!response.ok) throw new Error('Failed to validate rule');
  const data = await response.json();
  return {
    passed: data.valid,
    message: data.valid ? 'Rule expression is valid' : (data.errors || []).join(', '),
  };
}

// =============================================================================
// RULE TYPE CONFIGS (for display)
// =============================================================================

const RULE_TYPE_ICONS: Record<string, string> = {
  tax_loss_harvesting: '💰',
  wash_sale: '🚫',
  cppi_floor: '🛡️',
  drift_trigger: '⚖️',
  sector_limit: '📊',
  concentration_limit: '🎯',
  esg_restriction: '🌱',
  cash_flow: '💵',
  custom: '⚙️',
};

const RULE_TYPE_LABELS: Record<string, string> = {
  tax_loss_harvesting: 'Tax-Loss Harvesting',
  wash_sale: 'Wash Sale',
  cppi_floor: 'CPPI Floor',
  drift_trigger: 'Drift Trigger',
  sector_limit: 'Sector Limit',
  concentration_limit: 'Concentration Limit',
  esg_restriction: 'ESG Restriction',
  cash_flow: 'Cash Flow',
  custom: 'Custom',
};

// =============================================================================
// MAIN COMPONENT
// =============================================================================

export const RebalancingRulesPage: React.FC = () => {
  const { tenant, datasource } = useTenantContext();
  
  const [rules, setRules] = useState<RuleDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState(0);
  
  // Dialog state
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<Partial<RuleDefinition> | null>(null);

  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';

  // Load rules
  const loadRules = useCallback(async () => {
    if (!tenantId || !datasourceId) return;
    
    setLoading(true);
    setError(null);
    try {
      const data = await fetchRules(tenantId, datasourceId);
      setRules(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load rules');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    loadRules();
  }, [loadRules]);

  // Handle create/update
  const handleSave = async (rule: RuleDefinition) => {
    try {
      if (editingRule?.rule_id) {
        await updateRule(rule, datasourceId);
      } else {
        await createRule(rule, datasourceId);
      }
      setDialogOpen(false);
      setEditingRule(null);
      await loadRules();
    } catch (err) {
      throw err;
    }
  };

  // Handle delete
  const handleDelete = async (ruleId: string) => {
    if (!confirm('Are you sure you want to delete this rule?')) return;
    
    try {
      await deleteRule(tenantId, ruleId, datasourceId);
      await loadRules();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete rule');
    }
  };

  // Handle test
  const handleTest = async (rule: RuleDefinition) => {
    return testRule(rule, datasourceId);
  };

  // Open create dialog
  const handleCreate = () => {
    setEditingRule({
      tenant_id: tenantId,
      type: 'tax_loss_harvesting',
      active: true,
    });
    setDialogOpen(true);
  };

  // Open edit dialog
  const handleEdit = (rule: RuleDefinition) => {
    setEditingRule(rule);
    setDialogOpen(true);
  };

  // Duplicate rule
  const handleDuplicate = (rule: RuleDefinition) => {
    setEditingRule({
      ...rule,
      rule_id: `${rule.rule_id}_COPY`,
      name: `${rule.name} (Copy)`,
    });
    setDialogOpen(true);
  };

  // Filter rules by tab
  const getFilteredRules = () => {
    if (activeTab === 0) return rules;
    if (activeTab === 1) return rules.filter(r => r.type === 'tax_loss_harvesting' || r.type === 'wash_sale');
    if (activeTab === 2) return rules.filter(r => r.type === 'cppi_floor' || r.type === 'drift_trigger');
    if (activeTab === 3) return rules.filter(r => r.type === 'sector_limit' || r.type === 'concentration_limit' || r.type === 'esg_restriction');
    return rules;
  };

  if (!tenantId || !datasourceId) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Alert severity="warning">
          Please select a tenant and datasource to view rebalancing rules.
        </Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 600 }}>
            ⚖️ Rebalancing Rules
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Configure metadata-driven rules for portfolio rebalancing, tax optimization, and risk management
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={loadRules}
            disabled={loading}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
          >
            Create Rule
          </Button>
        </Box>
      </Box>

      {/* Error */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Tabs */}
      <Paper sx={{ mb: 2 }}>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label={`All Rules (${rules.length})`} />
          <Tab label="Tax Rules" />
          <Tab label="Risk Rules" />
          <Tab label="Constraint Rules" />
        </Tabs>
      </Paper>

      {/* Rules Table */}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Type</TableCell>
                <TableCell>Rule ID</TableCell>
                <TableCell>Name</TableCell>
                <TableCell>Jurisdiction</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Version</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {getFilteredRules().length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                    <Typography variant="body2" color="text.secondary">
                      No rules found. Create your first rule to get started.
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                getFilteredRules().map((rule) => (
                  <TableRow key={rule.rule_id} hover>
                    <TableCell>
                      <Chip
                        label={`${RULE_TYPE_ICONS[rule.type] || '⚙️'} ${RULE_TYPE_LABELS[rule.type] || rule.type}`}
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                        {rule.rule_id}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight={500}>
                        {rule.name}
                      </Typography>
                      {rule.description && (
                        <Typography variant="caption" color="text.secondary" display="block">
                          {rule.description.substring(0, 60)}
                          {rule.description.length > 60 ? '...' : ''}
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      <Chip label={rule.jurisdiction || 'GLOBAL'} size="small" />
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={rule.active ? 'Active' : 'Inactive'}
                        color={rule.active ? 'success' : 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                        v{rule.version}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip title="Edit">
                        <IconButton size="small" onClick={() => handleEdit(rule)}>
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Duplicate">
                        <IconButton size="small" onClick={() => handleDuplicate(rule)}>
                          <DuplicateIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete">
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => handleDelete(rule.rule_id)}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Rule Builder Dialog */}
      <Dialog
        open={dialogOpen}
        onClose={() => {
          setDialogOpen(false);
          setEditingRule(null);
        }}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          {editingRule?.rule_id ? 'Edit Rule' : 'Create New Rule'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <RDLRuleBuilder
              tenantId={tenantId}
              datasourceId={datasourceId}
              initialRule={editingRule || undefined}
              onSave={handleSave}
              onTest={handleTest}
              onCancel={() => {
                setDialogOpen(false);
                setEditingRule(null);
              }}
            />
          </Box>
        </DialogContent>
      </Dialog>
    </Container>
  );
};

export default RebalancingRulesPage;
