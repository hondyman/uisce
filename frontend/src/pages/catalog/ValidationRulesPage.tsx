import React, { useState, useMemo, useEffect } from 'react';
import {
  Box,
  Button,
  CircularProgress,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  Paper,
  Stack,
  TextField,
  Typography,
  Alert,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  SelectChangeEvent,
  FormControlLabel,
  Checkbox,
  Tab,
  Tabs,
  Snackbar,
  Autocomplete,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
// Removed unused icon imports (Delete, Edit, FileCopy, Done) to silence lint warnings
import SearchIcon from '@mui/icons-material/Search';
import CodeIcon from '@mui/icons-material/Code';
import BuildIcon from '@mui/icons-material/Build';
// (icons removed)
import ErrorIcon from '@mui/icons-material/Error';
import WarningIcon from '@mui/icons-material/Warning';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import { useTenant } from '../../contexts/TenantContext';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import ValidationRulesWithFacets from '../../components/ValidationRules/ValidationRulesWithFacets';
import { devError } from '../../utils/devLogger';
import type { ValidationRule as SharedValidationRule } from '../../components/validation/types';
import { fetchEntitySchema } from '../../api/entitySchema';

// Local narrowers to avoid `(rule as any)` casts in render/filter logic
const asRecord = (v: unknown): Record<string, unknown> => (v && typeof v === 'object' ? (v as Record<string, unknown>) : {});


const RULE_TYPES = [
  { value: 'field_format', label: 'Field Format', icon: '📝', description: 'Validate field values against regex patterns' },
  { value: 'cardinality', label: 'Cardinality', icon: '📊', description: 'Check value counts and thresholds' },
  { value: 'uniqueness', label: 'Uniqueness', icon: '🔑', description: 'Ensure values are unique in dataset' },
  { value: 'referential_integrity', label: 'Referential Integrity', icon: '🔗', description: 'Validate foreign key relationships' },
  { value: 'business_logic', label: 'Business Logic', icon: '⚙️', description: 'Apply custom business rules' },
];

const SEVERITY_OPTIONS = [
  { value: 'error', label: 'Error', color: 'error' },
  { value: 'warning', label: 'Warning', color: 'warning' },
  { value: 'info', label: 'Info', color: 'info' },
];

export const ValidationRulesPage: React.FC = () => {
  const { tenant, datasource, isSelected } = useTenant();
  
  const [entities, setEntities] = useState<string[]>([]);
  const [entitySchema, setEntitySchema] = useState<Record<string, unknown>>({});
  const [rules, setRules] = useState<SharedValidationRule[]>([]);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [filterRuleType, setFilterRuleType] = useState<string>('');
  const [filterSeverity, setFilterSeverity] = useState<string>('');
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<SharedValidationRule | null>(null);
  const [formTab, setFormTab] = useState<0 | 1>(0); // 0=builder, 1=json
  const [_copiedId, _setCopiedId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error' | 'warning' | 'info';
  }>({
    open: false,
    message: '',
    severity: 'success',
  });
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  // Fetch entities from backend
  const fetchEntities = async () => {
    if (!isSelected || !tenant?.id || !datasource?.id) {
      setEntities([]);
      setEntitySchema({});
      return;
    }

    try {
      const schema = await fetchEntitySchema(tenant.id, datasource.id);
      const entityNames = Object.keys(schema).sort();
      setEntities(entityNames);
      setEntitySchema(schema);
    } catch (error) {
      devError('Error fetching entities:', error);
      setEntities([]);
      setEntitySchema({});
    }
  };

  // Fetch rules from API
  const fetchRules = async () => {
    if (!isSelected || !tenant?.id || !datasource?.id) {
      setRules([]);
      return;
    }

    setLoading(true);
    try {
      const response = await fetch(
        `/api/validation-rules?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
        {
          headers: {
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch validation rules: ${response.statusText}`);
      }

      const data = await response.json();
      // Handle both old format (array) and new format (object with rules array)
      const rawArr: unknown[] = Array.isArray(data)
        ? (data as unknown[])
        : (data && typeof data === 'object' && Array.isArray((data as Record<string, unknown>)['rules'])
          ? ((data as Record<string, unknown>)['rules'] as unknown[])
          : []);
      const normalized = rawArr.map((r: unknown) => {
        const rr = (r && typeof r === 'object') ? (r as Record<string, unknown>) : {};
        return {
          id: String(rr['id'] ?? ''),
          name: (rr['rule_name'] ?? rr['name']) as string | undefined,
          rule_name: (rr['rule_name'] ?? rr['name']) as string | undefined,
          rule_type: String(rr['rule_type'] ?? ''),
          entity: (rr['entity'] ?? rr['target_entity']) as string | undefined,
          target_entity: (rr['target_entity'] ?? rr['entity']) as string | undefined,
          target_entities: Array.isArray(rr['target_entities']) ? (rr['target_entities'] as string[]) : undefined,
          sub_entity_type: String(rr['sub_entity_type'] ?? ''),
          severity: String(rr['severity'] ?? ''),
          description: String(rr['description'] ?? ''),
          is_active: Boolean(rr['is_active']),
          is_global: Boolean(rr['is_global']),
          is_core: Boolean(rr['is_core']),
          conditions: Array.isArray((rr['condition_json'] as Record<string, unknown>)?.['conditions']) ? (((rr['condition_json'] as Record<string, unknown>)['conditions']) as unknown[]) : (Array.isArray(rr['conditions']) ? (rr['conditions'] as unknown[]) : undefined),
          dependent_rule_ids: Array.isArray(rr['dependent_rule_ids']) ? (rr['dependent_rule_ids'] as string[]) : [],
          created_at: String(rr['created_at'] ?? ''),
          updated_at: String(rr['updated_at'] ?? ''),
        } as SharedValidationRule;
      });
      setRules(normalized);
    } catch (error) {
      devError('Error fetching validation rules:', error);
      setSnackbar({
        open: true,
        message: `Error loading validation rules: ${error instanceof Error ? error.message : 'Unknown error'}`,
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  };

  // Load rules when tenant/datasource selected
  useEffect(() => {
    fetchRules();
  }, [isSelected, tenant?.id, datasource?.id]);

  // Load entities when tenant/datasource selected
  useEffect(() => {
    fetchEntities();
  }, [isSelected, tenant?.id, datasource?.id]);

  // Validate form data
  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    if (!formData.rule_name.trim()) {
      errors.rule_name = 'Rule name is required';
    }

    if (!formData.target_entity.trim()) {
      errors.target_entity = 'Target entity is required';
    }

    if (formData.rule_type === 'field_format') {
      if (!formData.format_field.trim()) {
        errors.format_field = 'Field name is required';
      }
      if (!formData.format_pattern.trim()) {
        errors.format_pattern = 'Regex pattern is required';
      }
    } else if (formData.rule_type === 'cardinality') {
      if (!formData.cardinality_field.trim()) {
        errors.cardinality_field = 'Field name is required';
      }
      if (!formData.cardinality_value) {
        errors.cardinality_value = 'Threshold value is required';
      }
    } else if (formData.rule_type === 'uniqueness') {
      if (!formData.unique_field.trim()) {
        errors.unique_field = 'Field name is required';
      }
    } else if (formData.rule_type === 'referential_integrity') {
      if (!formData.ref_source_entity.trim()) {
        errors.ref_source_entity = 'Source entity is required';
      }
      if (!formData.ref_source_field.trim()) {
        errors.ref_source_field = 'Source field is required';
      }
      if (!formData.ref_target_entity.trim()) {
        errors.ref_target_entity = 'Target entity is required';
      }
      if (!formData.ref_target_field.trim()) {
        errors.ref_target_field = 'Target field is required';
      }
    } else if (formData.rule_type === 'business_logic') {
      if (!formData.logic_condition.trim()) {
        errors.logic_condition = 'JSON condition is required';
      } else {
        try {
          JSON.parse(formData.logic_condition);
        } catch (e) {
          errors.logic_condition = 'Invalid JSON format';
        }
      }
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Form data state
  const [formData, setFormData] = useState({
    rule_name: '',
    rule_type: 'business_logic' as SharedValidationRule['rule_type'],
    description: '',
    target_entity: '',
    target_entities: [] as string[], // New: Multi-entity support
  severity: 'error' as SharedValidationRule['severity'],
    is_active: true,
    field_path: [] as string[], // New: for hierarchical validation
    // Field Format fields
    format_pattern: '',
    format_field: '',
    // Cardinality fields
    cardinality_field: '',
    cardinality_operator: '>',
    cardinality_value: '',
    // Uniqueness fields
    unique_field: '',
    // Referential Integrity fields
    ref_source_entity: '',
    ref_source_field: '',
    ref_target_entity: '',
    ref_target_field: '',
    // Business Logic fields
    logic_condition: '',
  });

  const filteredRules = useMemo(() => {
    return rules.filter((rule) => {
      const rrec = asRecord(rule);
      const name = String(rule.rule_name ?? rrec.name ?? '').toLowerCase();
      const desc = String(rule.description ?? '').toLowerCase();
      const target = String(rule.target_entity ?? rrec.entity ?? '').toLowerCase();
        const matchesSearch = name.includes(searchQuery.toLowerCase()) || desc.includes(searchQuery.toLowerCase()) || target.includes(searchQuery.toLowerCase());

      const matchesType = !filterRuleType || rule.rule_type === filterRuleType;
      const matchesSeverity = !filterSeverity || rule.severity === filterSeverity;

      return matchesSearch && matchesType && matchesSeverity;
    });
  }, [rules, searchQuery, filterRuleType, filterSeverity]);

  const handleCreate = () => {
    setValidationErrors({});
    setEditingRule(null);
    setFormData({
      rule_name: '',
      rule_type: 'business_logic',
      description: '',
      target_entity: '',
      target_entities: [],
      severity: 'error',
      is_active: true,
      field_path: [],
      format_pattern: '',
      format_field: '',
      cardinality_field: '',
      cardinality_operator: '>',
      cardinality_value: '',
      unique_field: '',
      ref_source_entity: '',
      ref_source_field: '',
      ref_target_entity: '',
      ref_target_field: '',
      logic_condition: '',
    });
    setFormTab(0);
    setIsFormOpen(true);
  };

  const _handleEdit = (rule: SharedValidationRule) => {
    setValidationErrors({});
    setEditingRule(rule);
    const rrec = asRecord(rule);
    const rawJsonCandidate = rrec['condition_json'] ?? rrec['conditions'] ?? {};
    const json = (typeof rawJsonCandidate === 'object' && rawJsonCandidate !== null) ? (rawJsonCandidate as Record<string, unknown>) : {};
    setFormData({
      rule_name: String(rule.rule_name ?? rrec['name'] ?? ''),
      rule_type: rule.rule_type,
      description: rule.description || '',
      target_entity: String(rule.target_entity ?? rrec['entity'] ?? ''),
      target_entities: [], // New: Multi-entity support
      severity: rule.severity || 'error',
      is_active: !!rule.is_active,
      field_path: Array.isArray(json['field_path']) ? (json['field_path'] as string[]) : [],
      format_pattern: String(json['pattern'] ?? ''),
      format_field: String(json['field'] ?? ''),
      cardinality_field: String(json['field'] ?? ''),
      cardinality_operator: String(json['operator'] ?? '>'),
      cardinality_value: json['value'] != null ? String(json['value']) : '',
      unique_field: String(json['field'] ?? ''),
      ref_source_entity: String(json['source_entity'] ?? ''),
      ref_source_field: String(json['source_field'] ?? ''),
      ref_target_entity: String(json['target_entity'] ?? ''),
      ref_target_field: String(json['target_field'] ?? ''),
      logic_condition: JSON.stringify(json, null, 2),
    });
    setFormTab(0);
    setIsFormOpen(true);
  };

  const _handleDelete = async (id: string) => {
    if (!tenant?.id || !datasource?.id) {
      setSnackbar({
        open: true,
        message: 'Tenant scope not selected',
        severity: 'error',
      });
      return;
    }

    const confirm = useConfirm();
    const notification = useNotification();
    if (!(await confirm({ title: 'Delete validation rule', description: 'Are you sure you want to delete this validation rule? This action cannot be undone.' }))) {
      return;
    }

    try {
      const response = await fetch(
        `/api/validation-rules/${id}?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
        {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to delete rule: ${response.statusText}`);
      }

      setRules(rules.filter((r) => r.id !== id));
      setSnackbar({ open: true, message: 'Validation rule deleted successfully', severity: 'success' });
      notification.success('Validation rule deleted successfully');
    } catch (error) {
      devError('Error deleting rule:', error);
      setSnackbar({
        open: true,
        message: `Error deleting rule: ${error instanceof Error ? error.message : 'Unknown error'}`,
        severity: 'error',
      });
    }
  };

  const buildConditionJson = (): Record<string, any> => {
    switch (formData.rule_type) {
      case 'field_format':
        return { field_path: formData.field_path, pattern: formData.format_pattern };
      case 'cardinality':
        return {
          field_path: formData.field_path,
          operator: formData.cardinality_operator,
          value: isNaN(Number(formData.cardinality_value)) ? formData.cardinality_value : Number(formData.cardinality_value),
        };
      case 'uniqueness':
        return { field_path: formData.field_path, unique: true };
      case 'referential_integrity':
        return {
          source_entity: formData.ref_source_entity,
          source_field: formData.ref_source_field,
          target_entity: formData.ref_target_entity,
          target_field: formData.ref_target_field,
        };
      case 'business_logic':
        try {
          return JSON.parse(formData.logic_condition || '{}');
        } catch {
          return {};
        }
      default:
        return {};
    }
  };

  const handleSave = async () => {
    if (!validateForm()) {
      setSnackbar({
        open: true,
        message: 'Please fix the errors in the form',
        severity: 'error',
      });
      return;
    }

    if (!tenant?.id || !datasource?.id) {
      setSnackbar({
        open: true,
        message: 'Tenant scope not selected',
        severity: 'error',
      });
      return;
    }

    setSubmitting(true);
    const condition_json = buildConditionJson();
    const payload = {
      rule_name: formData.rule_name,
      rule_type: formData.rule_type,
      description: formData.description,
      target_entity: formData.target_entity,
      condition_json,
      severity: formData.severity,
      is_active: formData.is_active,
    };

    try {
      let response;

      if (editingRule) {
        // Update existing rule
        response = await fetch(
          `/api/validation-rules/${editingRule.id}?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
          {
            method: 'PATCH',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': tenant.id,
              'X-Tenant-Datasource-ID': datasource.id,
            },
            body: JSON.stringify(payload),
          }
        );

        if (!response.ok) {
          const error = await response.text();
          throw new Error(`Failed to update rule: ${error || response.statusText}`);
        }

        const updatedRaw = await response.json();
        const updatedRule: SharedValidationRule = {
          id: updatedRaw.id,
          name: updatedRaw.rule_name || updatedRaw.name || undefined,
          rule_name: updatedRaw.rule_name || updatedRaw.name || undefined,
          rule_type: updatedRaw.rule_type,
          entity: updatedRaw.entity || updatedRaw.target_entity || undefined,
          target_entity: updatedRaw.target_entity || updatedRaw.entity || undefined,
          target_entities: updatedRaw.target_entities,
          sub_entity_type: updatedRaw.sub_entity_type,
          severity: updatedRaw.severity,
          description: updatedRaw.description,
          is_active: updatedRaw.is_active,
          is_global: updatedRaw.is_global,
          is_core: updatedRaw.is_core,
          conditions: updatedRaw.condition_json?.conditions || updatedRaw.conditions || undefined,
          dependent_rule_ids: updatedRaw.dependent_rule_ids || [],
          created_at: updatedRaw.created_at,
          updated_at: updatedRaw.updated_at,
        } as SharedValidationRule;
        setRules(rules.map((r) => (r.id === editingRule.id ? updatedRule : r)));
        setSnackbar({
          open: true,
          message: 'Validation rule updated successfully',
          severity: 'success',
        });
      } else {
        // Create new rule
        response = await fetch(
          `/api/validation-rules?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Tenant-ID': tenant.id,
              'X-Tenant-Datasource-ID': datasource.id,
            },
            body: JSON.stringify(payload),
          }
        );

        if (!response.ok) {
          const error = await response.text();
          throw new Error(`Failed to create rule: ${error || response.statusText}`);
        }

        const newRaw = await response.json();
        const newRule: SharedValidationRule = {
          id: newRaw.id,
          name: newRaw.rule_name || newRaw.name || undefined,
          rule_name: newRaw.rule_name || newRaw.name || undefined,
          rule_type: newRaw.rule_type,
          entity: newRaw.entity || newRaw.target_entity || undefined,
          target_entity: newRaw.target_entity || newRaw.entity || undefined,
          target_entities: newRaw.target_entities,
          sub_entity_type: newRaw.sub_entity_type,
          severity: newRaw.severity,
          description: newRaw.description,
          is_active: newRaw.is_active,
          is_global: newRaw.is_global,
          is_core: newRaw.is_core,
          conditions: newRaw.condition_json?.conditions || newRaw.conditions || undefined,
          dependent_rule_ids: newRaw.dependent_rule_ids || [],
          created_at: newRaw.created_at,
          updated_at: newRaw.updated_at,
        } as SharedValidationRule;
        setRules([...rules, newRule]);
        setSnackbar({
          open: true,
          message: 'Validation rule created successfully',
          severity: 'success',
        });
      }

      setIsFormOpen(false);
    } catch (error) {
      devError('Error saving rule:', error);
      setSnackbar({
        open: true,
        message: `Error saving rule: ${error instanceof Error ? error.message : 'Unknown error'}`,
        severity: 'error',
      });
    } finally {
      setSubmitting(false);
    }
  };

  const _copyToClipboard = (rule: SharedValidationRule) => {
    const json = JSON.stringify(rule, null, 2);
    navigator.clipboard.writeText(json);
    _setCopiedId(rule.id ?? null);
    setTimeout(() => _setCopiedId(null), 2000);
  };

  const _getRuleTypeInfo = (type: string) => {
    return RULE_TYPES.find((t) => t.value === type);
  };

  const handleFormChange = (field: string, value: unknown) => {
    // Dynamic field updater: narrow to unknown and assign locally
    setFormData((prev) => ({ ...prev, [field]: value as any }));
  };

  return (
    <Container maxWidth="lg" sx={{ py: 3 }}>
      {/* Tenant Scope Alert */}
      {!isSelected && (
        <Alert severity="warning" sx={{ mb: 3 }} icon={<WarningIcon />}>
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            ⚠️ No Tenant Selected
          </Typography>
          <Typography variant="body2" sx={{ mt: 0.5 }}>
            Please select a tenant and datasource from the picker to create or manage validation rules.
          </Typography>
        </Alert>
      )}

      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 600 }}>
            ✓ Validation Rules
          </Typography>
          <Typography variant="body2" sx={{ color: 'text.secondary', mt: 0.5 }}>
            {isSelected
              ? `Define business logic and data quality rules for your entities (${tenant?.display_name})`
              : 'Define business logic and data quality rules for your entities'}
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleCreate}
          size="large"
          disabled={!isSelected}
        >
          New Rule
        </Button>
      </Stack>

      {/* Filter Section */}
      <Paper sx={{ p: 2, mb: 3, backgroundColor: '#f9fafb' }}>
        <Stack spacing={2}>
          <TextField
            fullWidth
            placeholder="Search rules by name, description, or entity..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            size="small"
            InputProps={{
              startAdornment: <SearchIcon sx={{ mr: 1, color: 'text.secondary' }} />,
            }}
            sx={{ backgroundColor: 'white' }}
          />
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth size="small">
                <InputLabel>Rule Type</InputLabel>
                <Select
                  value={filterRuleType}
                  onChange={(e: SelectChangeEvent) => setFilterRuleType(e.target.value)}
                  label="Rule Type"
                >
                  <MenuItem value="">All Types</MenuItem>
                  {RULE_TYPES.map((type) => (
                    <MenuItem key={type.value} value={type.value}>
                      {type.icon} {type.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth size="small">
                <InputLabel>Severity</InputLabel>
                <Select
                  value={filterSeverity}
                  onChange={(e: SelectChangeEvent) => setFilterSeverity(e.target.value)}
                  label="Severity"
                >
                  <MenuItem value="">All Severities</MenuItem>
                  {SEVERITY_OPTIONS.map((sev) => (
                    <MenuItem key={sev.value} value={sev.value}>
                      {sev.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </Stack>
      </Paper>

      {/* Rules Table */}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
          <Stack alignItems="center" spacing={2}>
            <CircularProgress />
            <Typography variant="body2" color="text.secondary">
              Loading validation rules...
            </Typography>
          </Stack>
        </Box>
      ) : !isSelected ? (
        <Alert severity="info">
          Select a tenant and datasource to view validation rules.
        </Alert>
      ) : filteredRules.length === 0 ? (
        <Alert severity="info">
          {rules.length === 0
            ? 'No validation rules yet. Click "New Rule" to create one!'
            : 'No rules match your filters.'}
        </Alert>
      ) : (
        // Use the richer rules component with facets and lazy loading
        <ValidationRulesWithFacets tenantId={tenant!.id} datasourceId={datasource!.id} entities={entities} entitySchema={entitySchema} />
      )}

      {/* Form Dialog - Workday Style */}
      <Dialog open={isFormOpen} onClose={() => setIsFormOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ fontWeight: 600 }}>
          {editingRule ? '✏️ Edit Validation Rule' : '➕ Create New Validation Rule'}
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <Tabs value={formTab} onChange={(_, v) => setFormTab(v as 0 | 1)} sx={{ mb: 3 }}>
            <Tab icon={<BuildIcon />} label="Rule Builder" iconPosition="start" />
            <Tab icon={<CodeIcon />} label="JSON Editor" iconPosition="start" />
          </Tabs>

          {formTab === 0 ? (
            // Builder Tab
            <Stack spacing={3}>
              <TextField
                fullWidth
                label="Rule Name *"
                value={formData.rule_name}
                onChange={(e) => {
                  handleFormChange('rule_name', e.target.value);
                  setValidationErrors((prev) => ({ ...prev, rule_name: '' }));
                }}
                placeholder="e.g., Order Total Must Be Positive"
                error={!!validationErrors.rule_name}
                helperText={validationErrors.rule_name}
              />

              <FormControl fullWidth error={!!validationErrors.rule_type}>
                <InputLabel>Rule Type *</InputLabel>
                <Select
                  value={formData.rule_type}
                  onChange={(e: SelectChangeEvent) => {
                    // SelectChangeEvent.target.value is a string; avoid broad `as any` casts
                    handleFormChange('rule_type', e.target.value as string);
                    setValidationErrors((prev) => ({ ...prev, rule_type: '' }));
                  }}
                  label="Rule Type *"
                >
                  {RULE_TYPES.map((type) => (
                    <MenuItem key={type.value} value={type.value}>
                      {type.icon} {type.label} - {type.description}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              {/* Target Entity with Typeahead Search */}
              <Autocomplete
                freeSolo
                options={['Customer', 'Employee', 'Supplier', 'Product', 'Order', 'OrderDetail', 'Department']}
                value={formData.target_entity}
                onChange={(event, newValue) => {
                  handleFormChange('target_entity', newValue || '');
                  setValidationErrors((prev) => ({ ...prev, target_entity: '' }));
                }}
                onInputChange={(event, newInputValue) => {
                  handleFormChange('target_entity', newInputValue);
                }}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Target Entity *"
                    placeholder="e.g., Order, Customer, Product"
                    error={!!validationErrors.target_entity}
                    helperText={validationErrors.target_entity || 'Primary entity this rule applies to'}
                  />
                )}
                filterOptions={(options, state) => {
                  return options.filter((option) =>
                    option.toLowerCase().includes(state.inputValue.toLowerCase())
                  );
                }}
              />

              {/* New: Multi-Entity Selector */}
              <Autocomplete
                multiple
                options={['Customer', 'Employee', 'Supplier', 'Product', 'Order', 'OrderDetail', 'global']}
                value={formData.target_entities || []}
                onChange={(event, newValue) => handleFormChange('target_entities', newValue)}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Apply to Entities (Optional)"
                    placeholder="Search & select entities or leave empty for single entity"
                    helperText="Select multiple entities to apply this rule across them (e.g., Customer + Employee for phone validation)"
                  />
                )}
                filterOptions={(options, state) => {
                  return options.filter((option) =>
                    option.toLowerCase().includes(state.inputValue.toLowerCase())
                  );
                }}
                size="small"
                sx={{ mb: 2 }}
              />

              <TextField
                fullWidth
                label="Description"
                value={formData.description}
                onChange={(e) => handleFormChange('description', e.target.value)}
                placeholder="Describe what this rule validates"
                multiline
                rows={2}
              />

              <Divider />

              {/* Dynamic fields based on rule type */}
              {formData.rule_type === 'field_format' && (
                <>
                  <Autocomplete
                    multiple
                    freeSolo
                    options={['line_items', 'products']}
                    value={formData.field_path}
                    onChange={(event, newValue) => {
                      handleFormChange('field_path', newValue);
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Field Path"
                        placeholder="order.line_items.qty"
                      />
                    )}
                  />
                  <TextField
                    fullWidth
                    label="Regex Pattern *"
                    value={formData.format_pattern}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      handleFormChange('format_pattern', e.target.value);
                      setValidationErrors((prev) => ({ ...prev, format_pattern: '' }));
                    }}
                    placeholder="e.g., ^[^@]+@[^@]+\\.[^@]+$"
                    sx={{ fontFamily: 'monospace' }}
                    error={!!validationErrors.format_pattern}
                    helperText={validationErrors.format_pattern}
                  />
                </>
              )}

              {formData.rule_type === 'cardinality' && (
                <>
                  <Autocomplete
                    multiple
                    freeSolo
                    options={['line_items', 'products']}
                    value={formData.field_path}
                    onChange={(event, newValue) => {
                      handleFormChange('field_path', newValue);
                    }}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Field Path"
                        placeholder="order.line_items.qty"
                      />
                    )}
                  />
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <FormControl fullWidth>
                        <InputLabel>Operator *</InputLabel>
                        <Select
                          value={formData.cardinality_operator}
                          onChange={(e: SelectChangeEvent) =>
                            handleFormChange('cardinality_operator', e.target.value)
                          }
                          label="Operator *"
                        >
                          <MenuItem value="&gt;">&gt;</MenuItem>
                          <MenuItem value="&lt;">&lt;</MenuItem>
                          <MenuItem value="&gt;=">&gt;=</MenuItem>
                          <MenuItem value="&lt;=">&lt;=</MenuItem>
                          <MenuItem value="==">==</MenuItem>
                          <MenuItem value="!=">!=</MenuItem>
                        </Select>
                      </FormControl>
                    </Grid>
                    <Grid item xs={6}>
                      <TextField
                        fullWidth
                        label="Threshold Value *"
                        type="number"
                        value={formData.cardinality_value}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          handleFormChange('cardinality_value', e.target.value);
                          setValidationErrors((prev) => ({ ...prev, cardinality_value: '' }));
                        }}
                        error={!!validationErrors.cardinality_value}
                        helperText={validationErrors.cardinality_value}
                      />
                    </Grid>
                  </Grid>
                </>
              )}

              {formData.rule_type === 'uniqueness' && (
                <Autocomplete
                  multiple
                  freeSolo
                  options={['line_items', 'products']}
                  value={formData.field_path}
                  onChange={(event, newValue) => {
                    handleFormChange('field_path', newValue);
                  }}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Field Path"
                      placeholder="order.line_items.qty"
                    />
                  )}
                />
              )}

              {formData.rule_type === 'referential_integrity' && (
                <>
                  <Alert severity="info" sx={{ mb: 2 }}>
                    📌 <strong>Foreign Key (FK) Validation:</strong> Verify that values in the source field match values in the target field of the target entity. Example: Order.customer_id must match a Customer.id.
                  </Alert>
                  
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <FormControl fullWidth error={!!validationErrors.ref_source_entity}>
                        <InputLabel>Source Entity *</InputLabel>
                        <Select
                          value={formData.ref_source_entity}
                          label="Source Entity *"
                          onChange={(e) => {
                            handleFormChange('ref_source_entity', e.target.value);
                            setValidationErrors((prev) => ({ ...prev, ref_source_entity: '' }));
                          }}
                        >
                          <MenuItem value="">-- Select Entity --</MenuItem>
                          {['Customer', 'Employee', 'Supplier', 'Order', 'OrderDetail', 'Product', 'Department'].map((entity) => (
                            <MenuItem key={entity} value={entity}>
                              {entity}
                            </MenuItem>
                          ))}
                        </Select>
                        {validationErrors.ref_source_entity && (
                          <Typography variant="caption" color="error">{validationErrors.ref_source_entity}</Typography>
                        )}
                      </FormControl>
                    </Grid>
                    <Grid item xs={6}>
                      <Autocomplete
                        freeSolo
                        options={['id', 'customer_id', 'employee_id', 'supplier_id', 'order_id', 'product_id', 'department_id', 'email', 'phone']}
                        value={formData.ref_source_field}
                        onChange={(event, newValue) => {
                          handleFormChange('ref_source_field', newValue || '');
                          setValidationErrors((prev) => ({ ...prev, ref_source_field: '' }));
                        }}
                        onInputChange={(event, newInputValue) => {
                          handleFormChange('ref_source_field', newInputValue);
                        }}
                        renderInput={(params) => (
                          <TextField
                            {...params}
                            label="Source Field *"
                            placeholder="e.g., customer_id"
                            error={!!validationErrors.ref_source_field}
                            helperText={validationErrors.ref_source_field || 'Field that contains the foreign key value'}
                          />
                        )}
                      />
                    </Grid>
                  </Grid>
                  
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <FormControl fullWidth error={!!validationErrors.ref_target_entity}>
                        <InputLabel>Target Entity *</InputLabel>
                        <Select
                          value={formData.ref_target_entity}
                          label="Target Entity *"
                          onChange={(e) => {
                            handleFormChange('ref_target_entity', e.target.value);
                            setValidationErrors((prev) => ({ ...prev, ref_target_entity: '' }));
                          }}
                        >
                          <MenuItem value="">-- Select Entity --</MenuItem>
                          {['Customer', 'Employee', 'Supplier', 'Order', 'OrderDetail', 'Product', 'Department'].map((entity) => (
                            <MenuItem key={entity} value={entity}>
                              {entity}
                            </MenuItem>
                          ))}
                        </Select>
                        {validationErrors.ref_target_entity && (
                          <Typography variant="caption" color="error">{validationErrors.ref_target_entity}</Typography>
                        )}
                      </FormControl>
                    </Grid>
                    <Grid item xs={6}>
                      <Autocomplete
                        freeSolo
                        options={['id', 'customer_id', 'employee_id', 'supplier_id', 'order_id', 'product_id', 'department_id', 'email', 'phone']}
                        value={formData.ref_target_field}
                        onChange={(event, newValue) => {
                          handleFormChange('ref_target_field', newValue || '');
                          setValidationErrors((prev) => ({ ...prev, ref_target_field: '' }));
                        }}
                        onInputChange={(event, newInputValue) => {
                          handleFormChange('ref_target_field', newInputValue);
                        }}
                        renderInput={(params) => (
                          <TextField
                            {...params}
                            label="Target Field *"
                            placeholder="e.g., id"
                            error={!!validationErrors.ref_target_field}
                            helperText={validationErrors.ref_target_field || 'Field that uniquely identifies records to match'}
                          />
                        )}
                      />
                    </Grid>
                  </Grid>
                  <Divider sx={{ my: 2 }} />
                </>
              )}

              {formData.rule_type === 'business_logic' && (
                <TextField
                  fullWidth
                  label="JSON Condition *"
                  value={formData.logic_condition}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    handleFormChange('logic_condition', e.target.value);
                    setValidationErrors((prev) => ({ ...prev, logic_condition: '' }));
                  }}
                  placeholder='{"field": "total", "operator": ">", "value": 0}'
                  multiline
                  rows={4}
                  sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}
                  error={!!validationErrors.logic_condition}
                  helperText={validationErrors.logic_condition}
                />
              )}

              <Divider />

              <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Severity *</InputLabel>
                    <Select
                      value={formData.severity}
                      onChange={(e: SelectChangeEvent) => handleFormChange('severity', e.target.value)}
                      label="Severity *"
                    >
                      {SEVERITY_OPTIONS.map((sev) => (
                        <MenuItem key={sev.value} value={sev.value}>
                          {sev.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={formData.is_active}
                        onChange={(e) => handleFormChange('is_active', e.target.checked)}
                      />
                    }
                    label="Active"
                  />
                </Grid>
              </Grid>
            </Stack>
          ) : (
            // JSON Tab
            <TextField
              fullWidth
              label="Complete Rule JSON"
              value={JSON.stringify(
                {
                  rule_name: formData.rule_name,
                  rule_type: formData.rule_type,
                  target_entity: formData.target_entity,
                  description: formData.description,
                  condition: buildConditionJson(),
                  severity: formData.severity,
                  is_active: formData.is_active,
                },
                null,
                2
              )}
              multiline
              rows={12}
              sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
              inputProps={{ readOnly: true }}
            />
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2, gap: 1 }}>
          <Button onClick={() => setIsFormOpen(false)} disabled={submitting}>
            Cancel
          </Button>
          <Button onClick={handleSave} variant="contained" disabled={submitting}>
            {submitting && <CircularProgress size={16} sx={{ mr: 1 }} />}
            {editingRule ? 'Update Rule' : 'Create Rule'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar Notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
          icon={
            snackbar.severity === 'success' ? (
              <CheckCircleIcon />
            ) : snackbar.severity === 'error' ? (
              <ErrorIcon />
            ) : (
              <WarningIcon />
            )
          }
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Container>
  );
};

export default ValidationRulesPage;
