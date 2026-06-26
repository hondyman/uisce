import React, { useState, useCallback, useEffect } from 'react';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { Link as RouterLink } from 'react-router-dom';
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
  ToggleButton, 
  ToggleButtonGroup,
  Stack 
} from '@mui/material';
import ValidationRuleScriptEditor from '../ValidationRules/ValidationRuleScriptEditor';
import apiClient from '../../utils/apiClient';
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
import RuleTester from './RuleTester';
import CoreRuleSummary from '../rules/CoreRuleSummary';
import BusinessObjectTree from '../bo/BusinessObjectTree';
import { RuleTemplate, ValidationRule as TemplateValidationRule } from '../../data/ruleTemplates';
import { rulesApi } from '../../services/rulesApi';
import { Rule as ApiRule, ValidationRule as ApiValidationRule } from '../../types/rules';
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
  rule_type: 'sql' | 'dsl';
}

interface ValidationRuleEditorProps {
  contextEntity?: string;
  contextEntityId?: string;
  contextField?: string;
  contextType?: 'bo' | 'term' | 'field';
  onRuleSaved?: () => void;
}

const ValidationRuleEditor: React.FC<ValidationRuleEditorProps> = ({ 
  contextEntity, 
  contextEntityId,
  contextField,
  contextType,
  onRuleSaved
}) => {
  const confirm = useConfirm();
  const notification = useNotification();
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingFullRule, setEditingFullRule] = useState<ApiRule | null>(null);
  const [coreRule, setCoreRule] = useState<ApiRule | null>(null);
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
    rule_type: 'sql',
  });
  const [editorMode, setEditorMode] = useState<'visual' | 'code'>('visual');
  const [selectedTemplate, setSelectedTemplate] = useState<RuleTemplate | null>(null);
  // value names prefixed with '_' because only setters are used in this component
  const [_showLivePreview, setShowLivePreview] = useState(false);
  const [_showImpactAnalysis, setShowImpactAnalysis] = useState(false);
  const [_generatedSampleData, setGeneratedSampleData] = useState<Record<string, unknown>[]>([]);
  const [showFieldSelector, setShowFieldSelector] = useState(false);
  const [_conflictCheckResults, setConflictCheckResults] = useState<unknown>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMsg, setSnackbarMsg] = useState('');

  // Highlighted fields (full paths like "entity.field.path") used to visually emphasize fields in selectors
  const [highlightedFields, setHighlightedFields] = useState<Set<string>>(new Set());

  // Substitution state: track previous condition to allow undo
  const [previousCondition, setPreviousCondition] = useState<string | null>(null);
  const [substituted, setSubstituted] = useState(false);
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

      let fetchedRules: (ApiRule | ApiValidationRule)[] = [];

      if (contextType === 'term' && contextEntityId) {
         // Term Context: Fetch rules linked to this semantic term
         fetchedRules = await rulesApi.fetchRulesBySemanticTerm(contextEntityId);
      } else if (contextType === 'bo' && contextEntityId) {
         // BO Context: Fetch resolved validation rules (includes inherited)
         // Note: returns ValidationRule[] which needs mapping to Rule
         fetchedRules = await rulesApi.fetchBOValidations(contextEntityId, tenantId, datasourceId);
      } else {
         // Global Context: Fetch all rules (legacy/generic)
         fetchedRules = await rulesApi.getRules();
      }

      // Map to local Rule interface
      const mappedRules: Rule[] = fetchedRules.map(r => {
          // Check if it's ApiValidationRule (from fetchBOValidations)
          const isValidationRule = (obj: any): obj is ApiValidationRule => {
              return 'source' in obj && 'scope' in obj; 
          };

          if (isValidationRule(r)) {
              return {
                  id: r.id,
                  name: r.name,
                  rule_name: r.name,
                  bp_name: r.source === 'bo' ? contextEntity : (r.source === 'semantic_term' ? 'Semantic Term' : ''),
                  target_entity: r.source === 'bo' ? contextEntityId : undefined, 
                  step_name: '', // Field path logic complex for resolved rules
                  field_name: '',
                  condition_json: r.expression, // Expression string as condition_json
                  action_on_success: '',
                  action_on_failure: '',
                  priority: 0,
                  enabled: true, // simplified
                  is_active: true,
                  is_core: r.scope === 'inherited',
                  created_at: undefined, // missing in simplified view
                  updated_at: undefined
              };
          } else {
              // It's ApiRule (full rule object)
              const ar = r as ApiRule;
              return {
                  id: ar.id,
                  name: ar.name,
                  rule_name: ar.name,
                  bp_name: ar.target_entity || '',
                  target_entity: ar.target_entity || '',
                  step_name: ar.field_path?.[0] || '', // Mapping simplified
                  field_name: ar.field_path?.[0] || '',
                  condition_json: ar.script_content || ar.condition_json || '{}',
                  action_on_success: '',
                  action_on_failure: '',
                  priority: ar.evaluation_order || 0,
                  enabled: ar.is_active,
                  is_active: ar.is_active,
                  is_core: ar.is_core,
                  created_at: ar.created_at,
                  updated_at: ar.updated_at
              };
          }
      });

      setRules(mappedRules);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch rules');
    } finally {
      setLoading(false);
    }
  }, [contextEntity, contextField, contextType, contextEntityId]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  // Fetch Metadata for ASL Autocomplete
  useEffect(() => {
      const fetchMetadata = async () => {
          try {
              // 1. Fetch Semantic Terms
              const termsRes = await apiClient('/api/semantic-terms');
              if (termsRes.ok) {
                  const data = await termsRes.json();
                  if (data.data) {
                    (window as any).semanticTerms = data.data;
                  }
              }

              // 2. Fetch BO Fields (if context is BO)
              if (contextEntityId && contextType === 'bo') {
                   const boRes = await apiClient(`/api/business-objects/${contextEntityId}`);
                   if (boRes.ok) {
                       const data = await boRes.json();
                       if (data.fields) {
                           // Transform to expected format { path: string, type: string }
                           (window as any).boFields = data.fields.map((f: any) => ({
                               path: f.name,
                               type: f.data_type
                           }));
                       }
                   }
              }
          } catch (e) {
              console.warn("Failed to fetch metadata for ASL autocomplete", e);
          }
      };
      
      fetchMetadata();
  }, [contextEntityId, contextType]);





  const handleOpenDialog = async (rule?: Rule) => {
    setError(null);
    if (rule) {
      setEditingId(rule.id);
      
      // Default to what we have
      let ruleToEdit = rule;

      // If we are in BO context, the rule object might be simplified (missing priority, full expression, etc)
      // So we fetch the full details
      if ((contextType === 'bo' || contextType === 'term') && rule.id) {
          try {
              const fullRule = await rulesApi.getRule(rule.id);
              setEditingFullRule(fullRule);

              // If this is an override, try to fetch the referenced core rule for context
              if (fullRule?.core_rule_id) {
                try {
                  const core = await rulesApi.getRule(fullRule.core_rule_id);
                  setCoreRule(core);
                } catch (e) {
                  console.warn('Failed to fetch core rule for context', e);
                  setCoreRule(null);
                }
              } else {
                setCoreRule(null);
              }

              // Map full ApiRule to local Rule
              ruleToEdit = {
                  id: fullRule.id,
                  name: fullRule.name,
                  rule_name: fullRule.name,
                  bp_name: fullRule.target_entity || '',
                  target_entity: fullRule.target_entity || '',
                  step_name: fullRule.field_path?.[0] || '',
                  field_name: fullRule.field_path?.[0] || '',
                  condition_json: fullRule.script_content || fullRule.condition_json || '{}',
                  action_on_success: '',
                  action_on_failure: '',
                  priority: fullRule.evaluation_order || 0,
                  enabled: fullRule.is_active,
                  is_active: fullRule.is_active,
                  is_core: fullRule.is_core,
                  created_at: fullRule.created_at,
                  updated_at: fullRule.updated_at
              };
          } catch (e) {
              console.error("Failed to fetch full rule details", e);
              setEditingFullRule(null);
              setCoreRule(null);
              // Fallback to existing partial rule
          }
      } else {
        setEditingFullRule(null);
      }

      // Convert condition_json to string if it's an object
      let conditionStr = '{}';
      if (ruleToEdit.condition_json) {
        if (typeof ruleToEdit.condition_json === 'string') {
          conditionStr = ruleToEdit.condition_json;
        } else if (typeof ruleToEdit.condition_json === 'object') {
          conditionStr = JSON.stringify(ruleToEdit.condition_json);
        }
      }
      // Map API fields to form fields (API uses target_entity/field_name, form uses bp_name/step_name)
      const bpName = ruleToEdit.bp_name || ruleToEdit.target_entity || '';
      const stepName = ruleToEdit.step_name || ruleToEdit.field_name || '';
      // API returns is_active, form uses enabled
      const isEnabled = ruleToEdit.is_active !== undefined ? ruleToEdit.is_active : (ruleToEdit.enabled !== undefined ? ruleToEdit.enabled : true);
      
      setFormData({
        name: ruleToEdit.name || ruleToEdit.rule_name || '',
        bp_name: bpName,
        step_name: stepName,
        condition_json: conditionStr,
        action_on_success: ruleToEdit.action_on_success || '',
        action_on_failure: ruleToEdit.action_on_failure || '',
        priority: ruleToEdit.priority || 0,
        enabled: isEnabled,
        rule_type: (ruleToEdit as any).rule_type || 'sql',
      });
      // Set editor mode based on rule content or type
      if ((ruleToEdit as any).rule_type === 'dsl' || (ruleToEdit as any).rule_type === 'asl') {
          setEditorMode('code');
      } else {
          setEditorMode('visual');
      }
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
        rule_type: 'sql',
      });
      setEditorMode('visual');
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
    setEditingFullRule(null);
    setDialogTab(0);
    setSelectedTemplate(null);
  };

  const handleOverrideClick = async () => {
    if (!editingId) return;
    const { tenantId, datasourceId } = getTenantContext();
    try {
      const newRule = await rulesApi.overrideRule(editingId, tenantId, datasourceId);
      // Open newly created override for editing
      handleCloseDialog();
      // Small delay to ensure state reset
      setTimeout(() => handleOpenDialog({ id: newRule.id } as any), 50);
    } catch (err) {
      notification?.({ message: (err as Error).message, severity: 'error' });
    }
  };

  const handleViewCoreRuleClick = async () => {
    if (!editingFullRule?.core_rule_id) return;
    // Close current dialog and open the core rule
    handleCloseDialog();
    // Small delay to ensure state reset
    setTimeout(() => handleOpenDialog({ id: editingFullRule.core_rule_id } as any), 50);
  };

  const handleRevertToCoreClick = async () => {
    if (!editingId || !editingFullRule?.is_override || !editingFullRule.core_rule_id) return;

    const confirmed = await confirm({
      title: 'Revert override',
      description: 'This will delete the tenant override and revert to the core rule. Do you want to continue?'
    });

    if (!confirmed) return;

    try {
      // Delete the override (same endpoint as delete)
      const { tenantId, datasourceId } = getTenantContext();
      const res = await fetch(
        `/api/validation-rules/${editingId}?tenant_id=${tenantId}&datasource_id=${datasourceId}`,
        {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Instance-ID': datasourceId,
          },
        }
      );

      if (!res.ok) throw new Error('Failed to delete override');

      // Re-open the core rule for editing if available
      handleCloseDialog();
      setTimeout(() => handleOpenDialog({ id: editingFullRule.core_rule_id } as any), 200);
      notification.success('Override deleted — reverted to core rule');
      await fetchRules();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revert to core');
    }
  };

  const handleTemplateSelected = (template: RuleTemplate, rule: Partial<TemplateValidationRule>) => {
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

    // Replace placeholders in template condition with selected step name (if provided)
    let processedCondition: string | object = rule.rule_condition || {};
    if (typeof processedCondition === 'string' && stepName) {
      processedCondition = processedCondition.replace(/\bfield\b/g, stepName);
      processedCondition = processedCondition.replace(/\bfield_value\b/g, stepName);
      processedCondition = processedCondition.replace(/\$\{FIELD_PATH\}/g, stepName);
    }

    setFormData({
      name: ruleName,
      bp_name: bpName,
      step_name: stepName,
      condition_json: typeof processedCondition === 'string'
        ? JSON.stringify(processedCondition)
        : JSON.stringify(processedCondition || {}),
      action_on_success: rule.severity === 'error' ? 'notify:admin' : '',
      action_on_failure: '',
      priority: 50,
      enabled: rule.is_enabled ?? true,
      rule_type: 'sql',
    });
    setEditorMode('visual');
    setDialogTab(1); // Move to form tab
  };

  const handleSaveRule = async () => {
    console.log('handleSaveRule called');
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
        target_entity_id: contextEntityId,
        target_entity_type: contextType === 'term' ? 'semantic_term' : 'business_object',
        
        // Scope and inheritance
        scope: contextType === 'term' ? ['global'] : ['local'],
        inherit_mode: 'custom',

        // Map step_name to field parameters or description if needed, 
        // but core field is target_entity. Storing field info in parameters for now
        parameters: {
            field_name: formData.step_name
        },

        description: `Validation for ${formData.step_name} on ${formData.bp_name}`,
        
        // Backend expects map, not string
        condition_json: editorMode === 'visual' ? conditionMap : {},
        script_content: editorMode === 'code' ? formData.condition_json : undefined,
        
        // Mapping enabled to is_active
        is_active: formData.enabled,
        
        // Mapping priority/actions to appropriate fields or parameters
        severity: formData.action_on_success === 'notify:admin' ? 'error' : 'warning',
        
        rule_type: editorMode === 'code' ? 'dsl' : 'sql'
      };

      console.log('Saving rule with payload:', payload);

      console.log('Saving rule with payload:', payload);
      console.log('Enabled status:', formData.enabled);

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Instance-ID': datasourceId,
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
      await new Promise(resolve => setTimeout(resolve, 300));
      await fetchRules();
      
      if (onRuleSaved) {
        onRuleSaved();
      }

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
            'X-Tenant-Instance-ID': datasourceId,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to delete rule: ${response.statusText}`);
      }

      await fetchRules();
      notification.success('Rule deleted');

      // If we deleted a tenant override, reopen the core rule for context
      if (editingFullRule?.is_override && editingFullRule.core_rule_id) {
        handleCloseDialog();
        setTimeout(() => handleOpenDialog({ id: editingFullRule.core_rule_id } as any), 200);
      }
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
        // If existing condition contains the placeholder 'field', replace it with the newly selected field
        try {
          // Parse stored condition value (it may be a JSON string of a string or object)
          const raw = next.condition_json || '';
          let parsed: any;
          try {
            parsed = JSON.parse(typeof raw === 'string' ? raw : JSON.stringify(raw));
          } catch (e) {
            parsed = raw;
          }

          const origString = typeof raw === 'string' ? raw : JSON.stringify(raw);
          const fieldPattern = /\bfield\b|\bfield_value\b|\$\{FIELD_PATH\}/;
          if (typeof parsed === 'string' && fieldPattern.test(parsed) && next.step_name) {
            const replacedStr = parsed
              .replace(/\bfield\b/g, next.step_name)
              .replace(/\bfield_value\b/g, next.step_name)
              .replace(/\$\{FIELD_PATH\}/g, next.step_name);
            next.condition_json = JSON.stringify(replacedStr);
            // If substitution occurred, remember previous state for undo
            if (origString !== next.condition_json) {
              setPreviousCondition(origString);
              setSubstituted(true);
            }
          } else if (parsed && typeof parsed === 'object') {
            // Replace inside JSON string representation if present
            const s = JSON.stringify(parsed);
            if (fieldPattern.test(s) && next.step_name) {
              const replaced = s
                .replace(/\bfield\b/g, next.step_name)
                .replace(/\bfield_value\b/g, next.step_name)
                .replace(/\$\{FIELD_PATH\}/g, next.step_name);
              try {
                const reparsed = JSON.parse(replaced);
                next.condition_json = JSON.stringify(reparsed);
              } catch (e) {
                // fallback to string
                next.condition_json = JSON.stringify(replaced);
              }
              if (origString !== next.condition_json) {
                setPreviousCondition(origString);
                setSubstituted(true);
              }
            }
          }
        } catch (e) {
          // ignore substitution errors
        }
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
      rule_type: (c.rule_type as any) || 'sql'
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

      // Query existing rules for the same entity/field
      const response = await fetch(
        `/api/validation-rules?tenant_id=${tenantId}&datasource_id=${datasourceId}&entity=${formData.bp_name}&field=${formData.step_name}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Instance-ID': datasourceId,
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
        <Tabs value={dialogTab} onChange={(_, val) => setDialogTab(val)} variant="fullWidth">
            {!editingId && <Tab label="📋 Templates" />}
            <Tab label="⚙️ Configure" disabled={!editingId && dialogTab === 0} />
            <Tab label="▶️ Test" disabled={!editingId && dialogTab === 0} />
            <Tab label="📊 Impact" disabled={!editingId && dialogTab === 0} />
        </Tabs>

        <DialogContent sx={{ pt: 2 }}>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          {/* Governance banners: core / override */}
          {editingFullRule?.is_core && (
            <Alert severity="info" sx={{ mb: 2 }}>
              This rule is a <strong>core (gold copy)</strong> and changes here will not affect tenant overrides. You may create a tenant override to customize behavior for your tenant.
              {editingFullRule?.can_override && (
                <Button size="small" sx={{ ml: 2 }} onClick={handleOverrideClick}>Create Tenant Override</Button>
              )}
            </Alert>
          )}

          {editingFullRule?.is_override && (
            <>
            <Alert severity="info" sx={{ mb: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                This rule is a <strong>tenant override</strong> of a core rule.
                {editingFullRule?.core_rule_id && (
                  <Typography variant="caption" display="block">Overrides core rule: {editingFullRule.core_rule_id}</Typography>
                )}
              </Box>
              <Box sx={{ display: 'flex', gap: 1 }}>
                {editingFullRule?.core_rule_id && (
                  <Button size="small" variant="outlined" onClick={() => handleViewCoreRuleClick()}>View Core Rule</Button>
                )}
                {editingFullRule?.core_rule_id && (
                  <Button size="small" component={RouterLink} to={`/governance/rules/${editingFullRule.id}/diff`} sx={{ ml: 1 }}>
                    View Diff
                  </Button>
                )}
                <Button size="small" color="warning" onClick={() => handleRevertToCoreClick()}>Revert to Core</Button>
              </Box>
            </Alert>
            {coreRule && (
              <Box sx={{ mb: 2 }}>
                <CoreRuleSummary core={coreRule} />
              </Box>
            )}
            </>
          )}
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
                  selectedField={formData.step_name}
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
                    targetEntity: r.bp_name || r.target_entity || '',
                    fieldName: r.step_name || r.field_name || '',
                  }))}
                  onRuleCloned={handleRuleCloned}
                  newRuleData={{
                    condition: formData.condition_json,
                    targetEntity: formData.bp_name || '',
                    fieldName: formData.step_name || '',
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
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    Condition
                    </Typography>
                    <ToggleButtonGroup
                        value={editorMode}
                        exclusive
                        onChange={(_, val) => val && setEditorMode(val)}
                        size="small"
                        aria-label="editor mode"
                    >
                        <ToggleButton value="visual" aria-label="visual builder">
                            Visual Builder
                        </ToggleButton>
                        <ToggleButton value="code" aria-label="asl code">
                            ASL Code
                        </ToggleButton>
                    </ToggleButtonGroup>
                </Stack>
                {editorMode === 'visual' ? (
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
                ) : (
                    <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1 }}>
                        <ValidationRuleScriptEditor
                            value={formData.condition_json}
                            onChange={(val) => handleFormChange('condition_json', val || '')}
                            language="asl"
                            height="400px"
                            theme="vs-dark"
                            highlight={formData.step_name}
                        />
                    </Box>
                )}
                <Box sx={{ mt: 1, display: 'flex', gap: 1 }}>
                  <Button 
                    variant="text" 
                    size="small" 
                    onClick={() => handleFormChange('condition_json', editorMode === 'visual' ? '{}' : '')}>
                    Reset Conditions
                  </Button>
                  {previousCondition && substituted && (
                    <Button
                      variant="text"
                      size="small"
                      onClick={() => {
                        // Undo the last substitution
                        handleFormChange('condition_json', previousCondition);
                        setPreviousCondition(null);
                        setSubstituted(false);
                      }}
                    >
                      Undo Substitution
                    </Button>
                  )}
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
          {dialogTab === 2 && (
            <Box>
              <Alert severity="info" sx={{ mb: 2 }}>
                Execute the ASL rule against sample data using the target engine (SQL/WASM) for trustworthy results.
              </Alert>
              
              <RuleTester
                dsl={formData.condition_json}
                tenantId={getTenantContext().tenantId || ''}
                boId={contextEntityId || formData.bp_name}
                onTestComplete={(results) => {
                  devDebug('ASL Test complete:', results);
                  const refs = (results && (results as any).referenced_fields) || [];
                  const setPaths = new Set<string>(refs.map((f: any) => `${f.business_object_id || contextEntityId || formData.bp_name}.${(f.field_path || []).join('.')}`));
                  setHighlightedFields(setPaths);
                  setSnackbarMsg(`Detected ${setPaths.size} referenced field(s)`);
                  setSnackbarOpen(true);
                }}
              />

              <Box sx={{ mt: 4 }}>
                <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                  Advanced: Generate Sample Data
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
                  onDataGenerated={(data: any) => {
                    setGeneratedSampleData(data);
                  }}
                />
              </Box>
            </Box>
          )}

          {/* Tab 3: Impact Analysis */}
              {dialogTab === 3 && (
            <Box>
              <Alert severity="info" sx={{ mb: 2 }}>
                Understand how many records will be affected before deploying this rule.
              </Alert>
              <ImpactAnalysis
                    ruleID={editingId || undefined}
                    rule={{
                      target_entity: formData.bp_name,
                      field_name: formData.step_name,
                      rule_condition: formData.condition_json,
                      severity: 'error',
                    }}
                tenantId={localStorage.getItem('selected_tenant') ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id : ''}
                datasourceId={localStorage.getItem('selected_datasource') ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id : ''}
                onFieldsDetected={(fields) => setHighlightedFields(new Set(fields))}
              />

              {highlightedFields.size > 0 && (
                <Box sx={{ mt: 2 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>Detected Fields</Typography>
                  <BusinessObjectTree fields={Array.from(highlightedFields).map(f => ({ fullPath: f, label: f.split('.').slice(-1)[0] }))} highlightedFields={highlightedFields} />
                </Box>
              )}
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
          {editingId && !editingFullRule?.is_override && (
            <Button
              color="error"
              startIcon={<DeleteIcon />}
              onClick={() => handleDeleteRule(editingId)}
              disabled={!editingFullRule?.can_delete}
            >
              Delete
            </Button>
          )}

          {editingFullRule?.can_override && (
            <Button variant="outlined" onClick={handleOverrideClick} sx={{ mr: 1 }}>
              Override Rule
            </Button>
          )}

          <Button onClick={handleSaveRule} variant="contained" disabled={!!editingId && !editingFullRule?.can_edit}>
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
            highlightedFields={highlightedFields}
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
