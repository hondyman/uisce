import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormHelperText,
  Box,
  Typography,
  IconButton,
  Stack,
  Alert,
  Chip,
  Switch,
  FormControlLabel,
  CircularProgress,
  Paper,
  LinearProgress,
  Tab,
  Tabs,
  Divider,
  alpha,
  useTheme,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import CodeIcon from '@mui/icons-material/Code';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import ErrorIcon from '@mui/icons-material/Error';
import WarningIcon from '@mui/icons-material/Warning';
import InfoIcon from '@mui/icons-material/Info';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SaveIcon from '@mui/icons-material/Save';
import EditIcon from '@mui/icons-material/Edit';
import ChatBubbleOutlineIcon from '@mui/icons-material/ChatBubbleOutline';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import RuleIcon from '@mui/icons-material/Rule';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import type { ValidationRule as SharedValidationRule } from '../../components/validation/types';
import { ValidationRuleScriptEditor } from './ValidationRuleScriptEditor';
import ValidationRuleSimulator from './ValidationRuleSimulator';

export interface FieldTypeInfo {
  type: string;
  isNullable?: boolean;
  enumValues?: string[];
}

interface Condition {
  field: string;
  fieldType?: string;
  operator: string;
  value: string;
  fieldLabel?: string;
}

interface ValidationRuleCreatorProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (rule: SharedValidationRule) => void | Promise<void>;
  tenantId: string;
  datasourceId: string;
  availableEntities: string[];
  entitySchema?: Record<string, unknown>;
  fieldMetadata?: Record<string, FieldTypeInfo>;
  editingRule?: SharedValidationRule | null;
  initialRule?: SharedValidationRule | null;
  defaultTargetEntity?: string;
  displayMode?: 'modal' | 'inline' | 'drawer' | string;
  initialScope?: { subtype?: string; field?: string };
  subtypes?: Record<string, any>;
  coreFields?: any[];
  customFields?: any[];
}

const SEVERITY_LEVELS = [
  { value: 'error', label: 'Error', icon: ErrorIcon, color: '#dc2626', bgColor: '#fef2f2', borderColor: '#fecaca', description: 'Blocks the action completely. User cannot proceed.' },
  { value: 'warning', label: 'Warning', icon: WarningIcon, color: '#f59e0b', bgColor: '#fffbeb', borderColor: '#fed7aa', description: 'Alerts the user but allows them to proceed.' },
  { value: 'info', label: 'Info', icon: InfoIcon, color: '#3b82f6', bgColor: '#eff6ff', borderColor: '#bfdbfe', description: 'Silent logging for audit. No user interruption.' },
];

const OPERATORS = [
  { value: 'equals', label: 'Equals' },
  { value: 'not_equals', label: 'Not Equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'starts_with', label: 'Starts With' },
  { value: 'ends_with', label: 'Ends With' },
  { value: 'greater_than', label: 'Greater Than' },
  { value: 'less_than', label: 'Less Than' },
  { value: 'is_empty', label: 'Is Empty' },
  { value: 'is_not_empty', label: 'Is Not Empty' }
];

export const ValidationRuleCreator = ({
  isOpen,
  onClose,
  onSave,
  tenantId,
  datasourceId,
  availableEntities,
  entitySchema = {},
  editingRule,
  defaultTargetEntity,
  initialScope,
  subtypes = {},
  coreFields = [],
  customFields = [],
}: ValidationRuleCreatorProps) => {
  const theme = useTheme();
  const isEditMode = !!editingRule;
  
  // Wizard step state
  const [currentStep, setCurrentStep] = useState(0);
  const [logicTab, setLogicTab] = useState(0); // 0 = Script, 1 = Conditions
  
  const [formData, setFormData] = useState({
    rule_name: '',
    rule_type: 'cue' as 'cue' | 'business_logic',
    target_entity: '',
    sub_entity_type: initialScope?.subtype || '',
    severity: 'error' as 'error' | 'warning' | 'info',
    description: '',
    error_message: '',
    is_global: false,
    is_active: true,
    match_type: 'ALL' as 'ALL' | 'ANY',
    conditions: [] as Condition[],
    script_content: `// Validation Rule (CUE)
// The input data is available as 'record'.
// Unification failure = validation error.

record: {
    // Add your constraints here
    // fieldName: > 0
    // status: "active"
}`
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [schemaContext, setSchemaContext] = useState<string>('');

  // Fetch CUE schema for IntelliSense
  useEffect(() => {
    const fetchSchema = async () => {
      let boId = '';
      if (formData.target_entity && entitySchema && (entitySchema as any)[formData.target_entity]) {
          boId = (entitySchema as any)[formData.target_entity].id;
      }
      
      if (!boId && formData.target_entity) {
          boId = formData.target_entity;
      }

      if (tenantId && boId) {
        try {
          const res = await fetch(`/api/validation-rules/schema?tenant_id=${tenantId}&bo_id=${boId}&locale=en`);
          if (res.ok) {
            const data = await res.json();
            setSchemaContext(data.schema || '');
          } else {
            console.warn("Failed to fetch CUE schema");
            setSchemaContext('');
          }
        } catch (e) {
          console.error(e);
          setSchemaContext('');
        }
      } else {
        setSchemaContext('');
      }
    };
    
    if (formData.target_entity) {
        fetchSchema();
    }
  }, [formData.target_entity, tenantId, entitySchema]);

  // Convert conditions to CUE code
  const generateCueFromConditions = (): string => {
    if (!formData.conditions || formData.conditions.length === 0) {
      return `// Validation Rule: ${formData.rule_name}
// ${formData.description}

record: {}`;
    }

    const operators: Record<string, { format: (field: string, value: string) => string }> = {
      equals: {
        format: (field, value) => `${field}: "${value}"`
      },
      not_equals: {
        format: (field, value) => `${field}: !="${value}"`
      },
      contains: {
        format: (field, value) => `${field}: =~"${value}"` // Regex-ish
      },
      starts_with: {
        format: (field, value) => `${field}: =~"^${value}"`
      },
      ends_with: {
        format: (field, value) => `${field}: =~"${value}$"`
      },
      greater_than: {
        format: (field, value) => `${field}: >${value}`
      },
      less_than: {
        format: (field, value) => `${field}: <${value}`
      },
      is_empty: {
        format: (field) => `${field}: ""` // Approximate
      },
      is_not_empty: {
        format: (field) => `${field}: !=""`
      }
    };

    // Build condition checks
    // CUE unifies everything, effectively AND.
    // OR is harder in CUE without disjunctions which operate on same field structure.
    // For now, let's assume AND. TODO: Handle OR/ANY match type.
    const conditionChecks = formData.conditions.map(cond => {
      const opConfig = operators[cond.operator];
      if (!opConfig) return '';
      return opConfig.format(cond.field, cond.value);
    });

    const body = conditionChecks.filter(Boolean).join('\n    ');

    return `// Validation Rule: ${formData.rule_name}
// ${formData.description}

record: {
    ${body}
}`;
  };

  // Get fields for entity
  const getFieldsForEntity = (entityName: string): Array<{ key: string; name: string; type: string; businessName: string }> => {
    const entity = (entitySchema && (entitySchema as Record<string, unknown>)[entityName]) as Record<string, unknown> | undefined;
    if (!entity || !Array.isArray(entity['entity_fields'])) {
      return [];
    }
    return (entity['entity_fields'] as unknown) as Array<{ key: string; name: string; type: string; businessName: string }>;
  };

  const getAllAvailableFields = (): Array<{ key: string; name: string; type: string; businessName: string }> => {
    // If a subtype is selected, return only the fields for that subtype
    if (formData.sub_entity_type && subtypes && subtypes[formData.sub_entity_type]) {
      const subtype = subtypes[formData.sub_entity_type];
      const subtypeFields = subtype.subtypeFields || subtype.fields || [];
      return subtypeFields.map((f: any) => ({
        key: f.key || f.name || '',
        name: f.name || '',
        type: f.type || '',
        businessName: f.displayName || f.name || '',
      }));
    }
    
    // Otherwise, use the business object's core and custom fields
    const allFields = [...(coreFields || []), ...(customFields || [])];
    return allFields.map((f: any) => ({
      key: f.key || f.name || '',
      name: f.name || '',
      type: f.type || '',
      businessName: f.displayName || f.name || '',
    }));
  };

  // Initialize form data
  useEffect(() => {
    if (isEditMode && editingRule && isOpen) {
      const editingAny = editingRule as any;
      let conditions = [];
      let matchType = 'ALL';
      
      if (editingAny.condition_json) {
        const cj = typeof editingAny.condition_json === 'string' 
          ? JSON.parse(editingAny.condition_json) 
          : editingAny.condition_json;
        conditions = cj.conditions || [];
        matchType = cj.match_type || 'ALL';
      }

      setFormData({
        rule_name: editingRule.rule_name || editingAny.name || '',
        rule_type: (editingRule.rule_type === 'starlark' ? 'cue' : (editingRule.rule_type as 'cue' | 'business_logic')) || 'cue',
        target_entity: editingRule.target_entity || editingAny.entity || '',
        sub_entity_type: editingRule.sub_entity_type || '',
        severity: editingRule.severity || 'error',
        description: editingRule.description || '',
        error_message: editingRule.error_message || '',
        is_global: false,
        is_active: editingRule.is_active ?? true,
        match_type: matchType as 'ALL' | 'ANY',
        conditions: conditions,
        script_content: (editingRule as any).script_content || ''
      });
      setLogicTab(editingRule.rule_type === 'starlark' || editingRule.rule_type === 'cue' ? 0 : 1);
    } else if (defaultTargetEntity && isOpen && !isEditMode) {
      setFormData(prev => ({ 
        ...prev, 
        target_entity: defaultTargetEntity,
        sub_entity_type: initialScope?.subtype || '',
      }));
    }
  }, [isEditMode, editingRule, isOpen, defaultTargetEntity, initialScope]);

  // Reset step when dialog opens
  useEffect(() => {
    if (isOpen) {
      setCurrentStep(0);
    }
  }, [isOpen]);

  const validateStep = (step: number): boolean => {
    const newErrors: Record<string, string> = {};
    
    if (step === 0) {
      if (!formData.rule_name.trim()) newErrors.rule_name = 'Reference ID is required';
      if (!formData.description.trim()) newErrors.description = 'Description is required';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleNext = () => {
    if (validateStep(currentStep)) {
      setCurrentStep(prev => Math.min(prev + 1, 2));
    }
  };

  const handleBack = () => {
    setCurrentStep(prev => Math.max(prev - 1, 0));
  };

  const handleSubmit = async () => {
    setLoading(true);

    try {
      console.log('[ValidationRuleCreator] handleSubmit started');
      console.log('[ValidationRuleCreator] formData.rule_type:', formData.rule_type);
      console.log('[ValidationRuleCreator] formData.conditions:', formData.conditions);
      
      // Always use starlark as the rule_type and always generate script_content
      let scriptContent = '';
      let conditionJson = null;
      
      // Check if we have conditions from the builder (even if rule_type is 'cue')
      const hasConditions = formData.conditions && formData.conditions.length > 0;
      
      console.log('[ValidationRuleCreator] hasConditions:', hasConditions);
      
      if (hasConditions) {
        // Generate CUE from conditions builder
        console.log('[ValidationRuleCreator] Generating CUE from conditions builder');
        scriptContent = generateCueFromConditions();
        console.log('[ValidationRuleCreator] Generated CUE:', scriptContent);
        
        conditionJson = {
          match_type: formData.match_type,
          conditions: formData.conditions,
        };
        console.log('[ValidationRuleCreator] conditionJson:', conditionJson);
      } else if (formData.rule_type === 'cue' && formData.script_content) {
        // Use manually entered CUE script
        console.log('[ValidationRuleCreator] Using manually entered CUE script');
        scriptContent = formData.script_content;
      } else {
        console.log('[ValidationRuleCreator] No conditions and no script_content');
      }

      const ruleData = {
        id: isEditMode && editingRule ? editingRule.id : undefined,
        rule_name: formData.rule_name.trim(),
        rule_type: 'cue', // Default to cue for script/conditions
        target_entity: defaultTargetEntity || formData.target_entity,
        sub_entity_type: formData.sub_entity_type,
        description: formData.description,
        error_message: formData.error_message,
        severity: formData.severity,
        is_global: formData.is_global,
        is_active: formData.is_active,
        condition_json: conditionJson, // Store conditions if from conditions builder for editing
        script_content: scriptContent,
      };
      
      console.log('[ValidationRuleCreator] Final ruleData being saved:', ruleData);
      console.log('[ValidationRuleCreator] script_content length:', scriptContent.length);

      await onSave(ruleData);
      handleClose();
    } catch (err) {
      console.error('[ValidationRuleCreator] Error in handleSubmit:', err);
      setErrors({ submit: err instanceof Error ? err.message : 'Unknown error occurred' });
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setFormData({
      rule_name: '',
      rule_type: 'cue',
      target_entity: '',
      sub_entity_type: '',
      severity: 'error',
      description: '',
      error_message: '',
      is_global: false,
      is_active: true,
      match_type: 'ALL',
      conditions: [],
      script_content: ''
    });
    setErrors({});
    setCurrentStep(0);
    onClose();
  };

  const addCondition = () => {
    setFormData({
      ...formData,
      conditions: [...formData.conditions, { field: '', operator: 'equals', value: '' }]
    });
  };

  const removeCondition = (index: number) => {
    setFormData({
      ...formData,
      conditions: formData.conditions.filter((_, i) => i !== index)
    });
  };

  const updateCondition = (index: number, field: keyof Condition, value: string) => {
    const newConditions = [...formData.conditions];
    newConditions[index] = { ...newConditions[index], [field]: value };
    if (field === 'operator' && (value === 'is_empty' || value === 'is_not_empty')) {
      newConditions[index].value = '';
    }
    setFormData({ ...formData, conditions: newConditions });
  };

  const handleLogicTabChange = (_: React.SyntheticEvent, newValue: number) => {
    setLogicTab(newValue);
    setFormData(prev => ({ ...prev, rule_type: newValue === 0 ? 'cue' : 'business_logic' }));
  };

  const progressPercent = ((currentStep + 1) / 3) * 100;
  const stepLabels = ['Basic Info', 'Logic & Severity', 'Review & Save'];

  // ============ STEP 1: Basic Info ============
  const renderStep1 = () => (
    <Stack spacing={4}>
      <TextField
        label="Reference ID"
        placeholder="e.g., VR-2023-001"
        value={formData.rule_name}
        onChange={(e) => setFormData({ ...formData, rule_name: e.target.value })}
        error={!!errors.rule_name}
        helperText={errors.rule_name || 'Unique identifier for this rule'}
        fullWidth
        required
      />

      <TextField
        label="Description"
        placeholder="Describe the purpose and logic of this validation rule for documentation..."
        value={formData.description}
        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
        error={!!errors.description}
        helperText={errors.description}
        fullWidth
        multiline
        rows={3}
        required
      />

      <TextField
        label="Validation Message"
        placeholder="e.g., The invoice date cannot be in the future."
        value={formData.error_message}
        onChange={(e) => setFormData({ ...formData, error_message: e.target.value })}
        helperText="Displayed to end-users when validation fails"
        fullWidth
        InputProps={{
          startAdornment: <ChatBubbleOutlineIcon sx={{ mr: 1, color: 'text.secondary' }} />
        }}
      />

      {/* Tip */}
      <Box sx={{ display: 'flex', gap: 1.5, color: 'text.secondary', px: 1 }}>
        <LightbulbIcon sx={{ fontSize: 20, mt: 0.25 }} />
        <Typography variant="body2">
          Tip: A clear Reference ID helps your team find rules quickly later. Try using a prefix like 'INV-' for invoice rules.
        </Typography>
      </Box>
    </Stack>
  );

  // ============ STEP 2: Logic & Severity ============
  const renderStep2 = () => (
    <Stack spacing={4}>
      {/* Script / Conditions Tabs */}
      <Paper variant="outlined" sx={{ borderRadius: 2 }}>
        <Tabs
          value={logicTab}
          onChange={handleLogicTabChange}
          sx={{ borderBottom: 1, borderColor: 'divider', px: 2 }}
        >
          <Tab icon={<CodeIcon />} iconPosition="start" label="Script (CUE)" sx={{ textTransform: 'none', fontWeight: 600 }} />
          <Tab 
            icon={<AccountTreeIcon />} 
            iconPosition="start" 
            label="Conditions Builder" 
            disabled={isEditMode && editingRule && !(editingRule as any).condition_json && logicTab === 0}
            title={isEditMode && editingRule && !(editingRule as any).condition_json ? "Not available for rules created via raw script" : ""}
            sx={{ textTransform: 'none', fontWeight: 600 }} 
          />
          <Tab 
            icon={<PlayArrowIcon />} 
            iconPosition="start" 
            label="Simulator (Preview)" 
            sx={{ textTransform: 'none', fontWeight: 600 }} 
          />
        </Tabs>

        <Box sx={{ p: 3 }}>
          {logicTab === 2 ? (
            <ValidationRuleSimulator scriptContent={formData.script_content} />
          ) : logicTab === 0 ? (
            <Stack spacing={2}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="subtitle2" fontWeight={600}>CUE Script Editor</Typography>
                <Stack direction="row" spacing={1}>
                  <Button size="small" variant="text" startIcon={<InfoIcon />}>Documentation</Button>
                </Stack>
              </Box>


              <Paper 
                variant="outlined" 
                sx={{ 
                  borderRadius: 2, 
                  overflow: 'hidden',
                  bgcolor: '#1e1e1e',
                }}
              >
                <Box sx={{ px: 2, py: 1, bgcolor: '#2d2d2d', borderBottom: '1px solid #404040', display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Chip label="main.cue" size="small" sx={{ bgcolor: '#404040', color: 'white', fontSize: '0.75rem' }} />
                  <Box sx={{ flex: 1 }} />
                  <Typography variant="caption" sx={{ color: '#888' }}>UTF-8</Typography>
                </Box>
                <ValidationRuleScriptEditor
                  value={formData.script_content}
                  onChange={(val) => setFormData({ ...formData, script_content: val || '' })}
                  height="280px"
                  schemaContext={schemaContext}
                />
              </Paper>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <InfoIcon sx={{ fontSize: 14 }} /> Press Ctrl+Space for autocomplete suggestions.
                {schemaContext && <span style={{ marginLeft: 10 }}>• Schema Loaded</span>}
              </Typography>
            </Stack>
          ) : (
            <Stack spacing={3}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
                <Typography variant="body2" fontWeight={600}>Execute if</Typography>
                <FormControl size="small" sx={{ minWidth: 80 }}>
                  <Select
                    value={formData.match_type}
                    onChange={(e) => setFormData({ ...formData, match_type: e.target.value as 'ALL' | 'ANY' })}
                  >
                    <MenuItem value="ALL">ALL</MenuItem>
                    <MenuItem value="ANY">ANY</MenuItem>
                  </Select>
                </FormControl>
                <Typography variant="body2" fontWeight={600}>conditions are met:</Typography>
              </Box>

              <Stack spacing={2}>
                {formData.conditions.map((condition, index) => (
                  <Paper key={index} variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="flex-start">
                      {index > 0 && (
                        <Chip 
                          label={formData.match_type === 'ALL' ? 'AND' : 'OR'} 
                          size="small" 
                          color="primary" 
                          variant="outlined"
                        />
                      )}
                      <FormControl size="small" sx={{ minWidth: 150 }}>
                        <InputLabel>Field</InputLabel>
                        <Select
                          value={condition.field}
                          label="Field"
                          onChange={(e) => updateCondition(index, 'field', e.target.value)}
                        >
                          {getAllAvailableFields().map(f => (
                            <MenuItem key={f.key} value={f.key}>{f.businessName || f.name}</MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                      <FormControl size="small" sx={{ minWidth: 130 }}>
                        <InputLabel>Operator</InputLabel>
                        <Select
                          value={condition.operator}
                          label="Operator"
                          onChange={(e) => updateCondition(index, 'operator', e.target.value)}
                        >
                          {OPERATORS.map(op => (
                            <MenuItem key={op.value} value={op.value}>{op.label}</MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                      {condition.operator !== 'is_empty' && condition.operator !== 'is_not_empty' && (
                        <TextField
                          size="small"
                          label="Value"
                          value={condition.value}
                          onChange={(e) => updateCondition(index, 'value', e.target.value)}
                          sx={{ flex: 1 }}
                        />
                      )}
                      <IconButton onClick={() => removeCondition(index)} color="error" size="small">
                        <DeleteIcon />
                      </IconButton>
                    </Stack>
                  </Paper>
                ))}
                
                <Button startIcon={<AddIcon />} onClick={addCondition} variant="outlined" sx={{ alignSelf: 'flex-start' }}>
                  Add Condition
                </Button>

                {/* Live Preview */}
                <Box sx={{ mt: 2, pt: 2, borderTop: 1, borderColor: 'divider' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>
                      Generated CUE Script
                  </Typography>
                  <Paper sx={{ bgcolor: '#1e293b', p: 2, borderRadius: 2, fontFamily: 'monospace', fontSize: '0.8rem', color: '#94a3b8', maxHeight: 200, overflow: 'auto' }}>
                      <pre style={{ margin: '0px' }}>{generateCueFromConditions()}</pre>
                  </Paper>
                </Box>
              </Stack>
            </Stack>
          )}
        </Box>
      </Paper>

      {/* Severity & Status Grid */}
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={3}>
        {/* Severity Section */}
        <Paper variant="outlined" sx={{ flex: 2, p: 3, borderRadius: 2 }}>
          <Typography variant="subtitle1" fontWeight={700} gutterBottom>Severity Level</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Determine the impact when this rule is triggered.
          </Typography>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
            {SEVERITY_LEVELS.map((level) => {
              const Icon = level.icon;
              const isSelected = formData.severity === level.value;
              return (
                <Paper
                  key={level.value}
                  onClick={() => setFormData({ ...formData, severity: level.value as any })}
                  sx={{
                    flex: 1,
                    p: 2,
                    cursor: 'pointer',
                    borderRadius: 2,
                    border: '2px solid',
                    borderColor: isSelected ? level.color : 'divider',
                    bgcolor: isSelected ? level.bgColor : 'transparent',
                    transition: 'all 0.2s ease',
                    '&:hover': {
                      borderColor: level.color,
                      bgcolor: alpha(level.color, 0.05),
                    },
                  }}
                >
                  <Stack spacing={1}>
                    <Stack direction="row" spacing={1} alignItems="center" sx={{ color: level.color }}>
                      <Icon />
                      <Typography variant="body2" fontWeight={700}>{level.label}</Typography>
                    </Stack>
                    <Typography variant="caption" color="text.secondary">{level.description}</Typography>
                  </Stack>
                </Paper>
              );
            })}
          </Stack>
        </Paper>

        {/* Status Section */}
        <Paper variant="outlined" sx={{ flex: 1, p: 3, borderRadius: 2 }}>
          <Typography variant="subtitle1" fontWeight={700} gutterBottom>Rule Status</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Enable or disable this rule immediately.
          </Typography>
          <Paper variant="outlined" sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderRadius: 2, bgcolor: 'action.hover' }}>
            <Typography variant="body2" fontWeight={600}>Active</Typography>
            <Switch
              checked={formData.is_active}
              onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
              color="primary"
            />
          </Paper>
        </Paper>
      </Stack>
    </Stack>
  );

  // ============ STEP 3: Review & Save ============
  const renderStep3 = () => {
    const severityConfig = SEVERITY_LEVELS.find(s => s.value === formData.severity);
    
    return (
      <Stack spacing={0}>
        {errors.submit && (
          <Alert severity="error" sx={{ mb: 3 }}>{errors.submit}</Alert>
        )}

        {/* Basic Information */}
        <Paper variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden' }}>
          <Box sx={{ p: 3, borderBottom: 1, borderColor: 'divider' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="subtitle1" fontWeight={700}>Basic Information</Typography>
              <Button size="small" startIcon={<EditIcon />} onClick={() => setCurrentStep(0)}>Edit</Button>
            </Box>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 3 }}>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1 }}>Rule Name</Typography>
                <Typography variant="body2" fontWeight={500}>{formData.rule_name || '—'}</Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1 }}>Target Object</Typography>
                <Typography variant="body2" fontWeight={500} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <RuleIcon sx={{ fontSize: 16, color: 'text.secondary' }} /> {formData.target_entity || '—'}
                </Typography>
              </Box>
              <Box sx={{ gridColumn: { sm: '1 / -1' } }}>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1 }}>Description</Typography>
                <Typography variant="body2">{formData.description || '—'}</Typography>
              </Box>
            </Box>
          </Box>

          {/* Logic Configuration */}
          <Box sx={{ p: 3, borderBottom: 1, borderColor: 'divider', bgcolor: 'action.hover' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="subtitle1" fontWeight={700}>Logic Configuration</Typography>
              <Button size="small" startIcon={<EditIcon />} onClick={() => setCurrentStep(1)}>Edit</Button>
            </Box>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 3 }}>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1 }}>Logic Type</Typography>
                <Typography variant="body2" fontWeight={500}>
                  {formData.rule_type === 'cue' ? 'CUE Script' : 'Conditions Builder'}
                </Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1 }}>Error Message</Typography>
                <Typography variant="body2" fontWeight={500} color="error.main" sx={{ fontStyle: 'italic' }}>
                  "{formData.error_message || 'Not specified'}"
                </Typography>
              </Box>



              <Box sx={{ gridColumn: '1 / -1' }}>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>Script Preview</Typography>
                <Paper sx={{ bgcolor: '#1e293b', p: 2, borderRadius: 2, fontFamily: 'monospace', fontSize: '0.8rem', color: '#94a3b8', maxHeight: 120, overflow: 'auto' }}>
                  <pre style={{ margin: '0px' }}>
                    {formData.rule_type === 'cue' ? formData.script_content : generateCueFromConditions()}
                  </pre>
                </Paper>
              </Box>
            </Box>
          </Box>

          {/* Settings */}
          <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="subtitle1" fontWeight={700}>Settings</Typography>
              <Button size="small" startIcon={<EditIcon />} onClick={() => setCurrentStep(1)}>Edit</Button>
            </Box>
            <Stack direction="row" spacing={4}>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>Severity</Typography>
                {severityConfig && (
                  <Chip
                    icon={<severityConfig.icon />}
                    label={severityConfig.label}
                    size="small"
                    sx={{
                      bgcolor: severityConfig.bgColor,
                      color: severityConfig.color,
                      border: `1px solid ${severityConfig.borderColor}`,
                      fontWeight: 600,
                    }}
                  />
                )}
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 1, mb: 1, display: 'block' }}>Status</Typography>
                <Chip
                  icon={formData.is_active ? <CheckCircleIcon /> : undefined}
                  label={formData.is_active ? 'Active' : 'Inactive'}
                  size="small"
                  color={formData.is_active ? 'success' : 'default'}
                  sx={{ fontWeight: 600 }}
                />
              </Box>
            </Stack>
          </Box>
        </Paper>
      </Stack>
    );
  };

  return (
    <Dialog 
      open={isOpen} 
      onClose={handleClose} 
      maxWidth="md" 
      fullWidth
      PaperProps={{
        sx: { 
          maxHeight: '90vh',
          borderRadius: 3,
        }
      }}
    >
      {/* Progress Header */}
      <DialogTitle sx={{ pb: 1 }}>
        <Stack spacing={2}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h5" fontWeight={700}>
                {isEditMode ? 'Edit Validation Rule' : 'Create Validation Rule'}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {currentStep === 0 && 'Define the core identification and scope for this rule.'}
                {currentStep === 1 && 'Configure how the validation rule behaves and its impact.'}
                {currentStep === 2 && 'Please verify the details below before saving.'}
              </Typography>
            </Box>
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="body2" color="primary" fontWeight={600}>
                Step {currentStep + 1} of 3
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {stepLabels[currentStep]}
              </Typography>
              <IconButton onClick={handleClose} size="small" sx={{ ml: 1 }}>
                <CloseIcon />
              </IconButton>
            </Stack>
          </Stack>
          <LinearProgress 
            variant="determinate" 
            value={progressPercent} 
            sx={{ 
              height: 8, 
              borderRadius: 4,
              bgcolor: 'action.hover',
              '& .MuiLinearProgress-bar': {
                borderRadius: 4,
              }
            }} 
          />
        </Stack>
      </DialogTitle>
        
      <DialogContent sx={{ pt: 3 }}>
        {currentStep === 0 && renderStep1()}
        {currentStep === 1 && renderStep2()}
        {currentStep === 2 && renderStep3()}
      </DialogContent>
      
      <DialogActions sx={{ px: 3, py: 2.5, borderTop: 1, borderColor: 'divider', bgcolor: 'action.hover', justifyContent: 'space-between' }}>
        <Button onClick={handleClose} color="inherit">
          Cancel
        </Button>
        <Stack direction="row" spacing={2}>
          {currentStep > 0 && (
            <Button startIcon={<ArrowBackIcon />} onClick={handleBack} variant="outlined">
              Back
            </Button>
          )}
          {currentStep < 2 ? (
            <Button 
              endIcon={<ArrowForwardIcon />} 
              onClick={handleNext} 
              variant="contained"
              sx={{ px: 3 }}
            >
              Next Step
            </Button>
          ) : (
            <Button
              startIcon={loading ? <CircularProgress size={18} color="inherit" /> : <SaveIcon />}
              onClick={handleSubmit}
              variant="contained"
              disabled={loading}
              sx={{ px: 4 }}
            >
              {loading ? 'Saving...' : 'Save Rule'}
            </Button>
          )}
        </Stack>
      </DialogActions>
    </Dialog>
  );
};

export default ValidationRuleCreator;
