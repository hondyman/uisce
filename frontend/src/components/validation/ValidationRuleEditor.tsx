import React, { useState, useCallback, useEffect } from 'react';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { getSelectedRegion } from '../../lib/region';
import {
  Box,
  Button,
  Card,
  CardContent as _CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Chip,
  IconButton,
  Alert,
  Tabs,
  Tab,
  Snackbar,
} from '@mui/material';
import { devDebug } from '../../utils/devLogger';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import _PlayArrowIcon from '@mui/icons-material/PlayArrow';
import _AnalyticsIcon from '@mui/icons-material/Analytics';
import _ConditionBuilder from './ConditionBuilder';
import ExpressionBuilder from '../../components/ExpressionBuilder/ExpressionBuilder';
import RuleTemplatesSelector from './RuleTemplatesSelector';
import LivePreview from './LivePreview';
import ImpactAnalysis from './ImpactAnalysis';
import AdvancedFieldSelector from './AdvancedFieldSelector';
import RuleCloneAndConflict from './RuleCloneAndConflict';
import SampleDataGenerator from './SampleDataGenerator';
import { RuleTemplate, ValidationRule } from '../../data/ruleTemplates';

interface Rule {
  id: string;
  name: string;
  rule_name?: string;
  bp_name?: string;
  target_entity?: string;
  step_name?: string;
  field_name?: string;
  condition_json: string | object;
  action_on_success?: string;
  action_on_failure?: string;
  priority?: number;
  enabled?: boolean;
  is_active?: boolean;
  is_core?: boolean;
  created_at?: string;
  updated_at?: string;
}

interface RuleFormData {
  name: string;
  bp_name: string;
  step_name: string;
  condition_json: string;
  action_on_success: string;
  action_on_failure: string;
  priority: number;
  enabled: boolean;
}

interface ValidationRuleEditorProps {
  contextEntity?: string;
  contextField?: string;
}

const ValidationRuleEditor: React.FC<ValidationRuleEditorProps> = ({ 
  contextEntity, 
  contextField 
}) => {
  const confirm = useConfirm();
  const notification = useNotification();
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [dialogTab, setDialogTab] = useState(0); // 0: Templates, 1: Form, 2: Preview, 3: Impact
  const [formData, setFormData] = useState<RuleFormData>({
    name: '',
    bp_name: contextEntity || '',
    step_name: contextField || '',
    condition_json: '{}',
    action_on_success: '',
    action_on_failure: '',
    priority: 0,
    enabled: true,
  });
  const [selectedTemplate, setSelectedTemplate] = useState<RuleTemplate | null>(null);
  // value names prefixed with '_' because only setters are used in this component
  const [_showLivePreview, setShowLivePreview] = useState(false);
  const [_showImpactAnalysis, setShowImpactAnalysis] = useState(false);
  const [_generatedSampleData, setGeneratedSampleData] = useState<Record<string, unknown>[]>([]);
  const [showFieldSelector, setShowFieldSelector] = useState(false);
  const [_conflictCheckResults, setConflictCheckResults] = useState<unknown>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMsg, setSnackbarMsg] = useState('');
  // ... state ...

  // Update formData when context props change
  useEffect(() => {
    setFormData(prev => ({
      ...prev,
      bp_name: contextEntity || prev.bp_name,
      step_name: contextField || prev.step_name
    }));
  }, [contextEntity, contextField]);

  const getTenantContext = () => {
    const tenantData = localStorage.getItem('selected_tenant')
      ? JSON.parse(localStorage.getItem('selected_tenant') || '{}')
      : {};
    const tenantId = tenantData.id || null;
    const datasourceId = localStorage.getItem('selected_datasource')
      ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id
      : null;
    const isGoldCopy = tenantData.gold_copy === true;
    return { tenantId, datasourceId, isGoldCopy };
  };

  const fetchRules = useCallback(async () => {
    try {
      setLoading(true);
      const { tenantId, datasourceId } = getTenantContext();

      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }

      let url = `/api/validation-rules?tenant_id=${tenantId}&datasource_id=${datasourceId}`;
      if (contextEntity) url += `&entity=${encodeURIComponent(contextEntity)}`;
      if (contextField) url += `&field=${encodeURIComponent(contextField)}`;

      // Get region from local storage
      const region = getSelectedRegion();

      const response = await fetch(url, {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
            ...(region ? { 'X-Tenant-Region': region } : {}),
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch rules: ${response.statusText}`);
      }

      const data = await response.json();
      // Handle both old format (array) and new format (object with rules array)
      const rulesArray = Array.isArray(data) ? data : (data.rules || []);
      setRules(rulesArray);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch rules');
    } finally {
      setLoading(false);
    }
  }, [contextEntity, contextField]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const handleOpenDialog = (rule?: Rule) => {
    if (rule) {
      setEditingId(rule.id);
      // Convert condition_json to string if it's an object
      let conditionStr = '{}';
      if (rule.condition_json) {
        if (typeof rule.condition_json === 'string') {
          conditionStr = rule.condition_json;
        } else if (typeof rule.condition_json === 'object') {
          conditionStr = JSON.stringify(rule.condition_json);
        }
      }
      // Map API fields to form fields (API uses target_entity/field_name, form uses bp_name/step_name)
      const bpName = rule.bp_name || rule.target_entity || '';
      const stepName = rule.step_name || rule.field_name || '';
      // API returns is_active, form uses enabled
      const isEnabled = rule.is_active !== undefined ? rule.is_active : (rule.enabled !== undefined ? rule.enabled : true);
      
      setFormData({
        name: rule.name || rule.rule_name || '',
        bp_name: bpName,
        step_name: stepName,
        condition_json: conditionStr,
        action_on_success: rule.action_on_success || '',
        action_on_failure: rule.action_on_failure || '',
        priority: rule.priority || 0,
        enabled: isEnabled,
      });
      setDialogTab(1); // Skip templates when editing
    } else {
      setEditingId(null);
      setFormData({
        name: '',
        bp_name: contextEntity || '',
        step_name: contextField || '',
        condition_json: '{}',
        action_on_success: '',
        action_on_failure: '',
        priority: 0,
        enabled: true,
      });
      setDialogTab(0); // Start with templates for new rules
    }
    setSelectedTemplate(null);
    setShowLivePreview(false);
    setShowImpactAnalysis(false);
    setOpenDialog(true);
  };

  const handleCloseDialog = () => {
    setOpenDialog(false);
    setEditingId(null);
    setDialogTab(0);
    setSelectedTemplate(null);
  };

  const handleTemplateSelected = (template: RuleTemplate, rule: Partial<ValidationRule>) => {
    setSelectedTemplate(template);
    
    // Preserve context if template doesn't specify target/field
    const bpName = rule.target_entity || formData.bp_name;
    const stepName = rule.field_name || formData.step_name;

    // Auto-generate name with context to avoid collisions: "[Field] Rule Name"
    let ruleName = rule.name || '';
    if (stepName && ruleName) {
        ruleName = `[${stepName}] ${ruleName}`;
    } else if (bpName && ruleName) {
        ruleName = `[${bpName}] ${ruleName}`;
    }

    setFormData({
      name: ruleName,
      bp_name: bpName, 
      step_name: stepName, 
      condition_json: typeof rule.rule_condition === 'string' 
        ? JSON.stringify(rule.rule_condition) 
        : JSON.stringify(rule.rule_condition || {}),
      action_on_success: rule.severity === 'error' ? 'notify:admin' : '',
      action_on_failure: '',
      priority: 50,
      enabled: rule.is_enabled ?? true,
    });
    setDialogTab(1); // Move to form tab
  };

  const handleSaveRule = async () => {
    try {
      // Validate required fields
      if (!formData.name || !formData.name.trim()) {
        setError('Rule name is required');
        return;
      }
      if (!formData.bp_name || !formData.bp_name.trim()) {
        setError('Business Process / Entity is required');
        return;
      }

      const { tenantId, datasourceId } = getTenantContext();

      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }

      // Get region from local storage
      const region = getSelectedRegion();

      const method = editingId ? 'PATCH' : 'POST';
      const url = editingId
        ? `/api/validation-rules/${editingId}?tenant_id=${tenantId}&datasource_id=${datasourceId}`
        : `/api/validation-rules?tenant_id=${tenantId}&datasource_id=${datasourceId}`;

      let conditionMap = {};
      try {
        // Handle both string JSON and already-parsed objects
        if (typeof formData.condition_json === 'string') {
          conditionMap = JSON.parse(formData.condition_json || '{}');
        } else if (typeof formData.condition_json === 'object' && formData.condition_json !== null) {
          conditionMap = formData.condition_json;
        } else {
          conditionMap = {};
        }
      } catch (e) {
        console.error('Invalid JSON for condition', e);
        setError('Invalid JSON in condition field');
        return;
      }

      const payload = {
        // Backend expects 'rule_name' or 'name' - providing both/standard
        name: formData.name,
        rule_name: formData.name,
        
        // Backend expects 'target_entity'
        target_entity: formData.bp_name,
        
        // Map step_name to field parameters or description if needed, 
        // but core field is target_entity. Storing field info in parameters for now
        parameters: {
            field_name: formData.step_name
        },

        description: `Validation for ${formData.step_name} on ${formData.bp_name}`,
        
        // Backend expects map, not string
        condition_json: conditionMap,
        
        // Mapping enabled to is_active
        is_active: formData.enabled,
        
        // Mapping priority/actions to appropriate fields or parameters
        severity: formData.action_on_success === 'notify:admin' ? 'error' : 'warning',
        
        rule_type: 'business_logic' // Default type
      };

      console.log('Saving rule with payload:', payload);
      console.log('Enabled status:', formData.enabled);

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
          ...(region ? { 'X-Tenant-Region': region } : {}),
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        let errorMessage = `Failed to save rule: ${response.statusText}`;
        try {
          const errorData = await response.json();
          if (errorData.details) {
            errorMessage = errorData.details;
          } else if (errorData.message) {
            errorMessage = errorData.message;
          }
        } catch (e) {
            // Use fallback
        }
        throw new Error(errorMessage);
      }

      handleCloseDialog();
      // Add a small delay to ensure backend has processed the change
      await new Promise(resolve => setTimeout(resolve, 100));
      await fetchRules();
      const actionType = editingId ? 'updated' : 'created';
      console.log(`Rule ${actionType} successfully. New enabled state:`, formData.enabled);
      notification.success(`Rule ${actionType} successfully!`);
      setEditingId(null); // Clear editing state
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save rule');
    }
  };

  const handleDeleteRule = async (ruleId: string) => {
    if (!(await confirm({ title: 'Delete rule', description: 'Are you sure you want to delete this rule?' }))) {
      return;
    }

    try {
      const { tenantId, datasourceId } = getTenantContext();
      
      // Get region from local storage
      const region = getSelectedRegion();

      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }

      const response = await fetch(
        `/api/validation-rules/${ruleId}?tenant_id=${tenantId}&datasource_id=${datasourceId}`,
        {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
            ...(region ? { 'X-Tenant-Region': region } : {}),
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to delete rule: ${response.statusText}`);
      }

      await fetchRules();
      notification.success('Rule deleted');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete rule');
    }
  };

  const handleFormChange = (field: keyof RuleFormData, value: unknown) => {
    // Narrow unknown -> expected types for specific fields at the boundary
    setFormData((prev) => {
      const next: RuleFormData = { ...prev };

      if (field === 'priority') {
        next.priority = typeof value === 'number' ? value : parseInt(String(value)) || 0;
      } else if (field === 'enabled') {
        next.enabled = typeof value === 'boolean' ? value : String(value) === 'enabled' || String(value) === 'true';
      } else if (field === 'name') {
        next.name = String(value ?? '');
      } else if (field === 'bp_name') {
        next.bp_name = String(value ?? '');
      } else if (field === 'step_name') {
        next.step_name = String(value ?? '');
      } else if (field === 'condition_json') {
        next.condition_json = String(value ?? '{}');
      } else if (field === 'action_on_success') {
        next.action_on_success = String(value ?? '');
      } else if (field === 'action_on_failure') {
        next.action_on_failure = String(value ?? '');
      }

      return next;
    });
  };

  const _handleFieldSelected = (fieldPath: string) => {
    // Set the field/attribute using dot notation
    handleFormChange('step_name', fieldPath);
    setShowFieldSelector(false);
  };

  const handleRuleCloned = (clonedRule: unknown) => {
    // Populate form with cloned rule data (defensive narrowing)
    const c = (clonedRule as Record<string, unknown>) || {}
    const name = typeof c.name === 'string' ? `${c.name} (Copy)` : `${formData.name} (Copy)`
    const bp = typeof c.target_entity === 'string' ? c.target_entity : formData.bp_name
    const step = typeof c.field_name === 'string' ? c.field_name : formData.step_name
    const conditionJson = typeof c.condition_json === 'string' ? c.condition_json : JSON.stringify(c.condition_json || {})
    const actionSuccess = typeof c.action_on_success === 'string' ? c.action_on_success : ''
    const actionFailure = typeof c.action_on_failure === 'string' ? c.action_on_failure : ''
    const priority = typeof c.priority === 'number' ? c.priority : 50

    setFormData({
      name,
      bp_name: bp as string,
      step_name: step as string,
      condition_json: conditionJson,
      action_on_success: actionSuccess,
      action_on_failure: actionFailure,
      priority,
      enabled: true,
    });
    setSelectedTemplate(null);
    setDialogTab(1); // Move to configure tab
  };

  const _handleSampleDataGenerated = (data: Record<string, unknown>[]) => {
    // Store generated data and switch to test tab
    setGeneratedSampleData(data);
    setDialogTab(2);
  };

  const _handleConflictCheckRequested = async () => {
    // Check for conflicts with existing rules before saving
    try {
      const { tenantId, datasourceId } = getTenantContext();
      
      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }
      
      // Get region from local storage
      const region = getSelectedRegion();

      // Query existing rules for the same entity/field
      const response = await fetch(
        `/api/validation-rules?tenant_id=${tenantId}&datasource_id=${datasourceId}&entity=${formData.bp_name}&field=${formData.step_name}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
            ...(region ? { 'X-Tenant-Region': region } : {}),
          },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to check conflicts');
      }

      const existingRules = await response.json();
      const conflicts = Array.isArray(existingRules) ? existingRules : (existingRules.rules || []);
      
      setConflictCheckResults({
        existingRules: conflicts,
        timestamp: new Date().toISOString(),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to check conflicts');
    }
  };

  return (
    <Box sx={{ p: '20px' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h6">Validation Rules</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => handleOpenDialog()}>
          Add Rule
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <TableContainer component={Card} sx={{ mt: '20px' }}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell>Name</TableCell>
              <TableCell>BP / Step</TableCell>
              <TableCell>Priority</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rules.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} sx={{ textAlign: 'center', py: 4 }}>
                  {loading ? 'Loading...' : 'No rules found'}
                </TableCell>
              </TableRow>
            ) : (
              rules.map((rule) => {
                const { isGoldCopy } = getTenantContext();
                const isCore = isGoldCopy; // If tenant is gold_copy, all rules are core
                return (
                <TableRow key={rule.id}>
                  <TableCell>{rule.name || rule.rule_name}</TableCell>
                  <TableCell>
                    {(rule.bp_name || rule.target_entity || 'N/A')} / {(rule.step_name || rule.field_name || 'N/A')}
                  </TableCell>
                  <TableCell>{rule.priority || 0}</TableCell>
                  <TableCell>
                    <Chip
                      label={isCore ? 'Core' : 'Custom'}
                      color={isCore ? 'info' : 'success'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      sx={{ mr: '8px' }}
                      label={(rule.is_active !== undefined ? rule.is_active : rule.enabled) ? 'Enabled' : 'Disabled'}
                      color={(rule.is_active !== undefined ? rule.is_active : rule.enabled) ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: '8px' }}>
                      <IconButton
                        size="small"
                        onClick={() => handleOpenDialog(rule)}
                        title="Edit"
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => handleDeleteRule(rule.id)}
                        title="Delete"
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={openDialog} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle sx={{ borderBottom: 1, borderColor: 'divider' }}>
          {editingId ? 'Edit Rule' : 'Create New Rule'}
        </DialogTitle>

        {/* Tabs for multi-step workflow */}
        {!editingId && (
          <Tabs value={dialogTab} onChange={(_, val) => setDialogTab(val)}>
            <Tab label="📋 Templates" />
            <Tab label="⚙️ Configure" disabled={dialogTab === 0} />
            <Tab label="▶️ Test" disabled={dialogTab === 0} />
            <Tab label="📊 Impact" disabled={dialogTab === 0} />
          </Tabs>
        )}

        <DialogContent sx={{ pt: 2 }}>
          {/* Tab 0: Rule Templates & Cloning */}
          {dialogTab === 0 && !editingId && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
              <Box>
                <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600 }}>
                  Start from Template
                </Typography>
                <RuleTemplatesSelector
                  onTemplateSelected={handleTemplateSelected}
                  targetEntity={formData.bp_name}
                />
              </Box>

              <Box sx={{ borderTop: 1, borderColor: 'divider', pt: 2 }}>
                <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600 }}>
                  OR Clone Existing Rule
                </Typography>
                <RuleCloneAndConflict
                  existingRules={rules.map(r => ({
                    id: r.id,
                    name: r.name,
                    description: '',
                    condition: r.condition_json,
                    severity: 'error' as const,
                    targetEntity: r.bp_name,
                    fieldName: r.step_name,
                  }))}
                  onRuleCloned={handleRuleCloned}
                  newRuleData={{
                    condition: formData.condition_json,
                    targetEntity: formData.bp_name,
                    fieldName: formData.step_name,
                  }}
                />
              </Box>
            </Box>
          )}

          {/* Tab 1: Configuration Form */}
          {(dialogTab === 1 || editingId) && (
            <Grid container spacing={2}>
              {selectedTemplate && !editingId && (
                <Grid item xs={12}>
                  <Alert severity="info">
                    Using template: <strong>{selectedTemplate.name}</strong> - {selectedTemplate.description}
                  </Alert>
                </Grid>
              )}

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Rule Name"
                  value={formData.name}
                  onChange={(e) => handleFormChange('name', e.target.value)}
                  placeholder="e.g., Age Must Be 18 or Older"
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Business Process / Entity"
                  value={formData.bp_name}
                  onChange={(e) => handleFormChange('bp_name', e.target.value)}
                  placeholder="e.g., Customer"
                  disabled={!!contextEntity}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <Box sx={{ display: 'flex', gap: 1 }}>
                  <TextField
                    fullWidth
                    label="Field / Attribute"
                    value={formData.step_name}
                    onChange={(e) => handleFormChange('step_name', e.target.value)}
                    placeholder="e.g., email (use dot notation for related fields)"
                    helperText="Or use Advanced Selector for related fields"
                    disabled={!!contextField}
                  />
                  <Button
                    variant="outlined"
                    onClick={() => setShowFieldSelector(true)}
                    sx={{ whiteSpace: 'nowrap' }}
                    title="Browse entities and relationships"
                  >
                    Browse
                  </Button>
                </Box>
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Priority"
                  value={formData.priority}
                  onChange={(e) => handleFormChange('priority', parseInt(e.target.value))}
                  inputProps={{ min: 0, max: 100 }}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth>
                  <InputLabel>Status</InputLabel>
                  <Select
                    value={formData.enabled ? 'enabled' : 'disabled'}
                    label="Status"
                    onChange={(e) => handleFormChange('enabled', e.target.value === 'enabled')}
                  >
                    <MenuItem value="enabled">Enabled</MenuItem>
                    <MenuItem value="disabled">Disabled</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
                  Condition (Visual Builder)
                </Typography>
                <ExpressionBuilder
                  ruleName={formData.name}
                  targetEntity={formData.bp_name}
                  autosave={!!editingId}
                  ruleId={editingId || undefined}
                  onDraftCreated={(id, name) => {
                    setEditingId(id);
                    if (name) handleFormChange('name', name);
                    setSnackbarMsg(`Draft created: ${name || id}`);
                    setSnackbarOpen(true);
                  }}
                  onSave={(cj) => {
                    handleFormChange('condition_json', JSON.stringify(cj));
                  }}
                  onChange={(cj) => {
                    handleFormChange('condition_json', JSON.stringify(cj));
                  }}
                />
                <Box sx={{ mt: 1, display: 'flex', gap: 1 }}>
                  <Button 
                    variant="text" 
                    size="small" 
                    onClick={() => handleFormChange('condition_json', '{}')}>
                    Reset Conditions
                  </Button>
                </Box>
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Action on Success"
                  value={formData.action_on_success}
                  onChange={(e) => handleFormChange('action_on_success', e.target.value)}
                  placeholder="e.g., route:approval.queue"
                  helperText="Format: route:queue_name, notify:email, or webhook:url"
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Action on Failure"
                  value={formData.action_on_failure}
                  onChange={(e) => handleFormChange('action_on_failure', e.target.value)}
                  placeholder="e.g., notify:admin@company.com"
                  helperText="Format: route:queue_name, notify:email, or webhook:url"
                />
              </Grid>
            </Grid>
          )}

          {/* Tab 2: Live Preview / Testing */}
          {dialogTab === 2 && !editingId && (
            <Box>
              <Alert severity="info" sx={{ mb: 2 }}>
                Generate test data and preview rule execution before deploying.
              </Alert>
              
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                  Step 1: Generate Sample Data
                </Typography>
                <SampleDataGenerator
                  entity={formData.bp_name || 'Entity'}
                  fields={[
                    {
                      name: formData.step_name || 'field',
                      dataType: 'string',
                      format: 'email',
                    }
                  ]}
                  onDataGenerated={(data) => {
                    setGeneratedSampleData(data);
                  }}
                />
              </Box>

              <Box>
                <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                  Step 2: Test Rule with Generated Data
                </Typography>
                  <LivePreview
                  rule={{
                    target_entity: formData.bp_name,
                    field_name: formData.step_name,
                    rule_condition: formData.condition_json,
                    severity: 'error',
                  }}
                  onTestResults={(results) => {
                    devDebug('Test results:', results);
                  }}
                />
              </Box>
            </Box>
          )}

          {/* Tab 3: Impact Analysis */}
              {dialogTab === 3 && !editingId && (
            <Box>
              <Alert severity="info" sx={{ mb: 2 }}>
                Understand how many records will be affected before deploying this rule.
              </Alert>
              <ImpactAnalysis
                    rule={{
                      target_entity: formData.bp_name,
                      field_name: formData.step_name,
                      rule_condition: formData.condition_json,
                      severity: 'error',
                    }}
                tenantId={localStorage.getItem('selected_tenant') ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id : ''}
                datasourceId={localStorage.getItem('selected_datasource') ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id : ''}
              />
            </Box>
          )}
        </DialogContent>

        <DialogActions sx={{ p: 2, borderTop: 1, borderColor: 'divider' }}>
          {!editingId && dialogTab > 0 && (
            <Button onClick={() => setDialogTab(dialogTab - 1)}>Back</Button>
          )}

          {!editingId && dialogTab < 3 && (
            <Button
              onClick={() => setDialogTab(dialogTab + 1)}
              variant="outlined"
            >
              Next
            </Button>
          )}

          <Box sx={{ flex: 1 }} />

          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSaveRule} variant="contained">
            {editingId ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Advanced Field Selector Dialog */}
      <Dialog open={showFieldSelector} onClose={() => setShowFieldSelector(false)} maxWidth="md" fullWidth>
        <DialogTitle>Select Field with Entity Navigation</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <AdvancedFieldSelector
            onFieldSelected={(fieldPath) => {
              handleFormChange('step_name', fieldPath);
              setShowFieldSelector(false);
            }}
            entities={[
              {
                name: formData.bp_name || 'Entity',
                displayName: formData.bp_name || 'Entity',
                fields: [
                  { name: formData.step_name || 'field', dataType: 'string', nullable: false }
                ],
                relationships: [],
              }
            ]}
            currentEntity={formData.bp_name}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowFieldSelector(false)}>Close</Button>
        </DialogActions>
      </Dialog>
      
      {/* Snackbar for feedback messages */}
      <Snackbar 
        open={snackbarOpen} 
        autoHideDuration={3000} 
        onClose={() => setSnackbarOpen(false)} 
        message={snackbarMsg} 
      />
    </Box>
  );
};

export default ValidationRuleEditor;
