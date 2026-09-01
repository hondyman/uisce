/**
 * UnifiedRuleEditor.tsx
 * 
 * A comprehensive Rule Fabric editor that supports all rule categories:
 * - Data Quality (DQ)
 * - Compliance
 * - MDM (Master Data Management)
 * - Wash Trade Detection
 * - Values/ESG
 * - Workflow
 * - Custom
 * 
 * Wraps existing AdvancedConditionBuilder, EntityPathPicker, and RuleDependencyChain
 * with category-specific presets, action templates, and governance workflow UI.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Checkbox from '@mui/material/Checkbox';
import {
  Save,
  Play,
  Send,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Settings,
  Eye,
  GitBranch,
  Shield,
  Database,
  RefreshCw,
  TrendingUp,
  Zap,
  Clock,
  Users,
  FileText,
  ChevronDown,
  ChevronRight,
  Plus,
  Trash2,
  Copy,
  History,
  Lock,
  Unlock
} from 'lucide-react';
import { AdvancedConditionBuilder } from '../AdvancedConditionBuilder';
import { ConditionGroup } from '../ExpressionBuilder/AdvancedConditionBuilder';

// ============================================================================
// Types
// ============================================================================

export type RuleCategory = 
  | 'data_quality'
  | 'compliance'
  | 'mdm'
  | 'wash_trade'
  | 'values'
  | 'workflow'
  | 'custom';

export type RuleContext =
  | 'data_record'
  | 'trade_event'
  | 'portfolio'
  | 'client_profile'
  | 'mdm_group'
  | 'system_job';

export type RuleSeverity = 'error' | 'warning' | 'info' | 'hard_block' | 'soft_block';

export type RuleStatus = 'draft' | 'awaiting_approval' | 'active' | 'inactive' | 'deprecated';

export type ExecutionChannel = 'batch' | 'realtime' | 'api' | 'workflow' | 'scheduler';

export interface RuleAction {
  action_type: string;
  action_config: Record<string, unknown>;
  execution_order: number;
}

export interface RuleLogic {
  id?: string;
  logic_type: 'condition' | 'expression' | 'script' | 'ml_model';
  condition_tree?: ConditionGroup;
  cel_expression?: string;
  script_language?: string;
  script_content?: string;
  ml_model_id?: string;
  evaluation_order: number;
  is_active: boolean;
}

export interface ExecutionPolicy {
  channel: ExecutionChannel;
  is_enabled: boolean;
  max_concurrent: number;
  timeout_seconds: number;
  retry_count: number;
  batch_size?: number;
  schedule_cron?: string;
}

export interface Rule {
  id?: string;
  tenant_id: string;
  tenant_instance_id: string;
  name: string;
  display_name: string;
  description: string;
  category: RuleCategory;
  context: RuleContext;
  target_entity: string;
  severity: RuleSeverity;
  status: RuleStatus;
  version: number;
  environment: 'dev' | 'test' | 'prod';
  logic: RuleLogic[];
  actions: RuleAction[];
  execution_policies: ExecutionPolicy[];
  dependent_rule_ids: string[];
  tags: string[];
  metadata: Record<string, unknown>;
  effective_from?: string;
  effective_to?: string;
  created_by?: string;
  updated_by?: string;
  created_at?: string;
  updated_at?: string;
}

export interface UnifiedRuleEditorProps {
  rule?: Rule;
  tenantId: string;
  datasourceId: string;
  availableEntities: Array<{ name: string; fields: Array<{ name: string; type: string; label: string }> }>;
  availableRules: Rule[];
  onSave: (rule: Rule) => Promise<void>;
  onTest: (rule: Rule) => Promise<TestResult>;
  onSubmitForApproval?: (rule: Rule) => Promise<void>;
  onPromote?: (rule: Rule, targetEnv: string) => Promise<void>;
}

export interface TestResult {
  success: boolean;
  passed: number;
  failed: number;
  errors: string[];
  sample_results: Array<{
    record_id: string;
    passed: boolean;
    message: string;
  }>;
}

// ============================================================================
// Category Configurations
// ============================================================================

const getCategoryConfig = (isDark: boolean) => ({
  data_quality: {
    icon: <Database size={20} />,
    color: '#2563eb',
    bgColor: isDark ? '#1e3a5f' : '#eff6ff',
    label: 'Data Quality',
    description: 'Validate data integrity, completeness, and accuracy',
    defaultContext: 'data_record' as RuleContext,
    allowedContexts: ['data_record', 'mdm_group', 'system_job'] as RuleContext[],
    actionTypes: [
      { value: 'reject_row', label: 'Reject Row', description: 'Reject the entire row from processing' },
      { value: 'quarantine_row', label: 'Quarantine Row', description: 'Move row to quarantine for review' },
      { value: 'flag_warning', label: 'Flag Warning', description: 'Add warning flag but continue processing' },
      { value: 'auto_correct', label: 'Auto-Correct', description: 'Apply automatic correction rules' },
      { value: 'send_alert', label: 'Send Alert', description: 'Send notification to data steward' }
    ],
    presetTemplates: [
      {
        name: 'Required Field Check',
        description: 'Ensure required fields are not null or empty',
        template: {
          severity: 'error',
          actions: [{ action_type: 'reject_row', action_config: { reason: 'Missing required field' }, execution_order: 1 }]
        }
      },
      {
        name: 'Range Validation',
        description: 'Validate numeric values are within acceptable range',
        template: {
          severity: 'warning',
          actions: [{ action_type: 'flag_warning', action_config: { flag_type: 'out_of_range' }, execution_order: 1 }]
        }
      },
      {
        name: 'Format Validation',
        description: 'Validate string formats (email, phone, SSN, etc.)',
        template: {
          severity: 'error',
          actions: [{ action_type: 'quarantine_row', action_config: { queue: 'format_errors' }, execution_order: 1 }]
        }
      }
    ]
  },
  compliance: {
    icon: <Shield size={20} />,
    color: '#9333ea',
    bgColor: isDark ? '#2e1065' : '#faf5ff',
    label: 'Compliance',
    description: 'Enforce regulatory and policy requirements',
    defaultContext: 'trade_event' as RuleContext,
    allowedContexts: ['trade_event', 'portfolio', 'client_profile'] as RuleContext[],
    actionTypes: [
      { value: 'block_trade', label: 'Block Trade', description: 'Prevent trade execution' },
      { value: 'require_approval', label: 'Require Approval', description: 'Route to compliance officer' },
      { value: 'log_breach', label: 'Log Breach', description: 'Record compliance breach event' },
      { value: 'notify_officer', label: 'Notify Officer', description: 'Send alert to compliance team' },
      { value: 'escalate', label: 'Escalate', description: 'Escalate to senior management' }
    ],
    presetTemplates: [
      {
        name: 'Position Limit Check',
        description: 'Ensure trades do not exceed position limits',
        template: {
          severity: 'hard_block',
          actions: [
            { action_type: 'block_trade', action_config: { reason: 'Position limit exceeded' }, execution_order: 1 },
            { action_type: 'notify_officer', action_config: { priority: 'high' }, execution_order: 2 }
          ]
        }
      },
      {
        name: 'Restricted List Check',
        description: 'Block trades in restricted securities',
        template: {
          severity: 'hard_block',
          actions: [{ action_type: 'block_trade', action_config: { reason: 'Security on restricted list' }, execution_order: 1 }]
        }
      },
      {
        name: 'Pre-Trade Approval',
        description: 'Require approval for large trades',
        template: {
          severity: 'soft_block',
          actions: [{ action_type: 'require_approval', action_config: { approver_role: 'compliance_officer' }, execution_order: 1 }]
        }
      }
    ]
  },
  mdm: {
    icon: <RefreshCw size={20} />,
    color: '#16a34a',
    bgColor: isDark ? '#14532d' : '#f0fdf4',
    label: 'Master Data',
    description: 'Manage data matching, merging, and survivorship',
    defaultContext: 'mdm_group' as RuleContext,
    allowedContexts: ['mdm_group', 'data_record'] as RuleContext[],
    actionTypes: [
      { value: 'auto_merge', label: 'Auto-Merge', description: 'Automatically merge matched records' },
      { value: 'flag_duplicate', label: 'Flag Duplicate', description: 'Mark as potential duplicate' },
      { value: 'assign_golden', label: 'Assign Golden', description: 'Set as golden/master record' },
      { value: 'queue_steward', label: 'Queue for Steward', description: 'Add to data steward work queue' },
      { value: 'apply_survivorship', label: 'Apply Survivorship', description: 'Apply survivorship rules' }
    ],
    presetTemplates: [
      {
        name: 'Exact Match',
        description: 'Identify exact duplicates based on key fields',
        template: {
          severity: 'warning',
          actions: [{ action_type: 'flag_duplicate', action_config: { confidence: 'high' }, execution_order: 1 }]
        }
      },
      {
        name: 'Fuzzy Match',
        description: 'Identify potential matches using fuzzy logic',
        template: {
          severity: 'info',
          actions: [{ action_type: 'queue_steward', action_config: { reason: 'Fuzzy match review' }, execution_order: 1 }]
        }
      }
    ]
  },
  wash_trade: {
    icon: <AlertTriangle size={20} />,
    color: '#dc2626',
    bgColor: isDark ? '#450a0a' : '#fef2f2',
    label: 'Wash Trade',
    description: 'Detect and prevent wash trading patterns',
    defaultContext: 'trade_event' as RuleContext,
    allowedContexts: ['trade_event', 'portfolio'] as RuleContext[],
    actionTypes: [
      { value: 'cancel_trade', label: 'Cancel Trade', description: 'Cancel the suspicious trade' },
      { value: 'flag_pattern', label: 'Flag Pattern', description: 'Flag wash trade pattern' },
      { value: 'alert_surveillance', label: 'Alert Surveillance', description: 'Send to surveillance team' },
      { value: 'block_account', label: 'Block Account', description: 'Temporarily block account' },
      { value: 'generate_sar', label: 'Generate SAR', description: 'Generate suspicious activity report' }
    ],
    presetTemplates: [
      {
        name: 'Self-Trade Detection',
        description: 'Detect trades between same beneficial owner',
        template: {
          severity: 'hard_block',
          actions: [
            { action_type: 'cancel_trade', action_config: {}, execution_order: 1 },
            { action_type: 'alert_surveillance', action_config: { priority: 'critical' }, execution_order: 2 }
          ]
        }
      },
      {
        name: 'Circular Trade Pattern',
        description: 'Detect circular trading patterns',
        template: {
          severity: 'hard_block',
          actions: [{ action_type: 'flag_pattern', action_config: { pattern_type: 'circular' }, execution_order: 1 }]
        }
      }
    ]
  },
  values: {
    icon: <TrendingUp size={20} />,
    color: '#0d9488',
    bgColor: isDark ? '#134e4a' : '#f0fdfa',
    label: 'Values/ESG',
    description: 'Enforce values-based and ESG investment policies',
    defaultContext: 'portfolio' as RuleContext,
    allowedContexts: ['portfolio', 'trade_event', 'client_profile'] as RuleContext[],
    actionTypes: [
      { value: 'exclude_security', label: 'Exclude Security', description: 'Exclude from investment universe' },
      { value: 'apply_tilt', label: 'Apply Tilt', description: 'Apply ESG tilt to allocation' },
      { value: 'require_disclosure', label: 'Require Disclosure', description: 'Require ESG disclosure' },
      { value: 'flag_controversy', label: 'Flag Controversy', description: 'Flag for controversy review' },
      { value: 'update_score', label: 'Update Score', description: 'Update ESG score' }
    ],
    presetTemplates: [
      {
        name: 'ESG Exclusion',
        description: 'Exclude securities below ESG threshold',
        template: {
          severity: 'hard_block',
          actions: [{ action_type: 'exclude_security', action_config: { reason: 'Below ESG threshold' }, execution_order: 1 }]
        }
      },
      {
        name: 'Carbon Footprint Limit',
        description: 'Enforce portfolio carbon footprint limits',
        template: {
          severity: 'warning',
          actions: [{ action_type: 'apply_tilt', action_config: { tilt_type: 'low_carbon' }, execution_order: 1 }]
        }
      }
    ]
  },
  workflow: {
    icon: <GitBranch size={20} />,
    color: '#ea580c',
    bgColor: isDark ? '#7c2d12' : '#fff7ed',
    label: 'Workflow',
    description: 'Define business process rules and routing',
    defaultContext: 'system_job' as RuleContext,
    allowedContexts: ['system_job', 'data_record', 'trade_event'] as RuleContext[],
    actionTypes: [
      { value: 'route_task', label: 'Route Task', description: 'Route to specific queue or user' },
      { value: 'trigger_workflow', label: 'Trigger Workflow', description: 'Start a workflow process' },
      { value: 'set_priority', label: 'Set Priority', description: 'Set task priority level' },
      { value: 'assign_owner', label: 'Assign Owner', description: 'Assign to specific owner' },
      { value: 'schedule_action', label: 'Schedule Action', description: 'Schedule future action' }
    ],
    presetTemplates: [
      {
        name: 'Exception Routing',
        description: 'Route exceptions to appropriate handler',
        template: {
          severity: 'info',
          actions: [{ action_type: 'route_task', action_config: { queue: 'exceptions' }, execution_order: 1 }]
        }
      }
    ]
  },
  custom: {
    icon: <Zap size={20} />,
    color: '#4b5563',
    bgColor: isDark ? '#1f2937' : '#f9fafb',
    label: 'Custom',
    description: 'Define custom business rules',
    defaultContext: 'data_record' as RuleContext,
    allowedContexts: ['data_record', 'trade_event', 'portfolio', 'client_profile', 'mdm_group', 'system_job'] as RuleContext[],
    actionTypes: [
      { value: 'custom_action', label: 'Custom Action', description: 'Execute custom action handler' },
      { value: 'webhook', label: 'Webhook', description: 'Call external webhook' },
      { value: 'log_event', label: 'Log Event', description: 'Log custom event' },
      { value: 'send_notification', label: 'Send Notification', description: 'Send custom notification' }
    ],
    presetTemplates: []
  }
});

const SEVERITY_CONFIG: Record<RuleSeverity, { icon: React.ReactNode; color: string; bgColor: string; label: string }> = {
  error: { icon: <XCircle size={16} />, color: '#dc2626', bgColor: '#fee2e2', label: 'Error' },
  warning: { icon: <AlertTriangle size={16} />, color: '#ca8a04', bgColor: '#fef9c3', label: 'Warning' },
  info: { icon: <Eye size={16} />, color: '#2563eb', bgColor: '#dbeafe', label: 'Info' },
  hard_block: { icon: <Lock size={16} />, color: '#991b1b', bgColor: '#fecaca', label: 'Hard Block' },
  soft_block: { icon: <Unlock size={16} />, color: '#ea580c', bgColor: '#ffedd5', label: 'Soft Block' }
};

const STATUS_CONFIG: Record<RuleStatus, { color: string; bgColor: string; label: string }> = {
  draft: { color: '#4b5563', bgColor: '#f3f4f6', label: 'Draft' },
  awaiting_approval: { color: '#ca8a04', bgColor: '#fef9c3', label: 'Awaiting Approval' },
  active: { color: '#16a34a', bgColor: '#dcfce7', label: 'Active' },
  inactive: { color: '#6b7280', bgColor: '#f3f4f6', label: 'Inactive' },
  deprecated: { color: '#dc2626', bgColor: '#fee2e2', label: 'Deprecated' }
};

// ============================================================================
// Helper Functions
// ============================================================================

const generateId = () => `rule_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

const createEmptyRule = (tenantId: string, datasourceId: string, category: RuleCategory = 'data_quality'): Rule => {
  const CATEGORY_CONFIG = getCategoryConfig(false);
  return {
    tenant_id: tenantId,
    tenant_instance_id: datasourceId,
    name: '',
    display_name: '',
    description: '',
    category,
    context: CATEGORY_CONFIG[category].defaultContext,
    target_entity: '',
    severity: 'warning',
    status: 'draft',
    version: 1,
    environment: 'dev',
    logic: [{
      logic_type: 'condition',
      condition_tree: {
        id: generateId(),
        type: 'group',
        operator: 'AND',
        conditions: []
      },
      evaluation_order: 1,
      is_active: true
    }],
    actions: [],
    execution_policies: [
      { channel: 'batch', is_enabled: true, max_concurrent: 10, timeout_seconds: 300, retry_count: 3 },
      { channel: 'realtime', is_enabled: false, max_concurrent: 100, timeout_seconds: 5, retry_count: 1 }
    ],
    dependent_rule_ids: [],
    tags: [],
    metadata: {}
  };
};

// ============================================================================
// Sub-Components
// ============================================================================

interface CategorySelectorProps {
  value: RuleCategory;
  onChange: (category: RuleCategory) => void;
  disabled?: boolean;
}

const CategorySelector: React.FC<CategorySelectorProps> = ({ value, onChange, disabled }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const CATEGORY_CONFIG = getCategoryConfig(isDark);
  const categories = Object.keys(CATEGORY_CONFIG) as RuleCategory[];

  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)', lg: 'repeat(7, 1fr)' }, gap: 1 }}>
      {categories.map(cat => {
        const config = CATEGORY_CONFIG[cat];
        const isSelected = value === cat;
        return (
          <Button
            key={cat}
            variant={isSelected ? 'contained' : 'outlined'}
            disabled={disabled}
            onClick={() => onChange(cat)}
            sx={{
              p: 1.5,
              display: 'flex',
              flexDirection: 'column',
              gap: 0.5,
              borderWidth: 2,
              borderColor: isSelected ? config.color : 'divider',
              bgcolor: isSelected ? config.bgColor : 'transparent',
              color: isSelected ? config.color : 'text.secondary',
              '&:hover': {
                borderColor: config.color,
                bgcolor: config.bgColor,
              },
              ...(disabled && { opacity: 0.5, cursor: 'not-allowed' }),
            }}
          >
            <Box sx={{ color: isSelected ? config.color : 'inherit' }}>{config.icon}</Box>
            <Typography variant="caption" sx={{ fontWeight: 500 }}>{config.label}</Typography>
          </Button>
        );
      })}
    </Box>
  );
};

interface ActionEditorProps {
  actions: RuleAction[];
  category: RuleCategory;
  onChange: (actions: RuleAction[]) => void;
}

const ActionEditor: React.FC<ActionEditorProps> = ({ actions, category, onChange }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const CATEGORY_CONFIG = getCategoryConfig(isDark);
  const actionTypes = CATEGORY_CONFIG[category].actionTypes;

  const addAction = () => {
    onChange([
      ...actions,
      {
        action_type: actionTypes[0]?.value || 'custom_action',
        action_config: {},
        execution_order: actions.length + 1
      }
    ]);
  };

  const updateAction = (index: number, updates: Partial<RuleAction>) => {
    const newActions = [...actions];
    newActions[index] = { ...newActions[index], ...updates };
    onChange(newActions);
  };

  const removeAction = (index: number) => {
    onChange(actions.filter((_, i) => i !== index));
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>Actions</Typography>
        <Button
          size="small"
          startIcon={<Plus size={14} />}
          onClick={addAction}
          sx={{ color: '#2563eb' }}
        >
          Add Action
        </Button>
      </Box>

      {actions.length === 0 ? (
        <Paper variant="outlined" sx={{ p: 3, textAlign: 'center', borderStyle: 'dashed' }}>
          <Typography variant="body2" color="text.secondary">No actions configured</Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
            Add actions to define what happens when the rule triggers
          </Typography>
        </Paper>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {actions.map((action, index) => (
            <Paper
              key={index}
              variant="outlined"
              sx={{ p: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}
            >
              <Box
                sx={{
                  width: 24,
                  height: 24,
                  bgcolor: '#2563eb',
                  color: '#fff',
                  borderRadius: '50%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '0.75rem',
                  fontWeight: 600,
                  flexShrink: 0,
                }}
              >
                {index + 1}
              </Box>
              <Select
                value={action.action_type}
                onChange={(e) => updateAction(index, { action_type: e.target.value })}
                size="small"
                sx={{ flex: 1, minWidth: 150 }}
              >
                {actionTypes.map(at => (
                  <MenuItem key={at.value} value={at.value}>{at.label}</MenuItem>
                ))}
              </Select>
              <IconButton
                size="small"
                onClick={() => removeAction(index)}
                sx={{ color: '#dc2626', '&:hover': { bgcolor: '#fee2e2' } }}
              >
                <Trash2 size={16} />
              </IconButton>
            </Paper>
          ))}
        </Box>
      )}
    </Box>
  );
};

interface ExecutionPolicyEditorProps {
  policies: ExecutionPolicy[];
  onChange: (policies: ExecutionPolicy[]) => void;
}

const ExecutionPolicyEditor: React.FC<ExecutionPolicyEditorProps> = ({ policies, onChange }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const [expanded, setExpanded] = useState(false);

  const updatePolicy = (channel: ExecutionChannel, updates: Partial<ExecutionPolicy>) => {
    const newPolicies = policies.map(p =>
      p.channel === channel ? { ...p, ...updates } : p
    );
    onChange(newPolicies);
  };

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
      <Button
        fullWidth
        onClick={() => setExpanded(!expanded)}
        sx={{
          justifyContent: 'space-between',
          p: 1.5,
          '&:hover': { bgcolor: isDark ? '#374151' : '#f9fafb' },
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Settings size={16} sx={{ color: isDark ? '#9ca3af' : '#6b7280' }} />
          <Typography variant="body2" sx={{ fontWeight: 600 }}>Execution Policies</Typography>
        </Box>
        {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </Button>

      {expanded && (
        <Box sx={{ p: 1.5, borderTop: 1, borderColor: 'divider' }}>
          {policies.map(policy => (
            <Box
              key={policy.channel}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 2,
                p: 1,
                mb: 1,
                bgcolor: isDark ? '#1f2937' : '#f9fafb',
                borderRadius: 1,
              }}
            >
              <FormControlLabel
                control={
                  <Checkbox
                    checked={policy.is_enabled}
                    onChange={(e) => updatePolicy(policy.channel, { is_enabled: e.target.checked })}
                    size="small"
                  />
                }
                label={<Typography variant="body2" sx={{ fontWeight: 500, textTransform: 'capitalize' }}>{policy.channel}</Typography>}
                sx={{ minWidth: 100 }}
              />
              {policy.is_enabled && (
                <>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Clock size={14} sx={{ color: isDark ? '#9ca3af' : '#9ca3af' }} />
                    <TextField
                      type="number"
                      size="small"
                      value={policy.timeout_seconds}
                      onChange={(e) => updatePolicy(policy.channel, { timeout_seconds: parseInt(e.target.value) })}
                      sx={{ width: 60 }}
                    />
                    <Typography variant="caption" sx={{ color: isDark ? '#9ca3af' : '#6b7280' }}>sec</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Users size={14} sx={{ color: isDark ? '#9ca3af' : '#9ca3af' }} />
                    <TextField
                      type="number"
                      size="small"
                      value={policy.max_concurrent}
                      onChange={(e) => updatePolicy(policy.channel, { max_concurrent: parseInt(e.target.value) })}
                      sx={{ width: 60 }}
                    />
                  </Box>
                </>
              )}
            </Box>
          ))}
        </Box>
      )}
    </Paper>
  );
};

interface GovernanceBarProps {
  rule: Rule;
  onSubmitForApproval: () => void;
  onPromote: (env: string) => void;
  isSaving: boolean;
}

const GovernanceBar: React.FC<GovernanceBarProps> = ({ rule, onSubmitForApproval, onPromote, isSaving }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const statusConfig = STATUS_CONFIG[rule.status];

  const getEnvColor = () => {
    if (rule.environment === 'prod') return { bg: '#dcfce7', color: '#166534' };
    if (rule.environment === 'test') return { bg: '#fef9c3', color: '#854d0e' };
    return { bg: isDark ? '#374151' : '#f3f4f6', color: isDark ? '#d1d5db' : '#374151' };
  };
  const envColors = getEnvColor();

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        p: 1.5,
        bgcolor: isDark ? '#1f2937' : '#f9fafb',
        borderBottom: 1,
        borderColor: 'divider',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        <Chip
          label={statusConfig.label}
          size="small"
          sx={{ bgcolor: statusConfig.bgColor, color: statusConfig.color, fontWeight: 600 }}
        />
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: isDark ? '#9ca3af' : '#6b7280' }}>
          <History size={14} />
          <Typography variant="body2">v{rule.version}</Typography>
        </Box>
        <Chip
          label={rule.environment.toUpperCase()}
          size="small"
          sx={{ bgcolor: envColors.bg, color: envColors.color }}
        />
      </Box>

      <Box sx={{ display: 'flex', gap: 1 }}>
        {rule.status === 'draft' && (
          <Button
            size="small"
            variant="contained"
            onClick={onSubmitForApproval}
            disabled={isSaving}
            startIcon={<Send size={14} />}
            sx={{ bgcolor: '#eab308', '&:hover': { bgcolor: '#ca8a04' }, textTransform: 'none' }}
          >
            Submit for Approval
          </Button>
        )}
        {rule.status === 'active' && rule.environment !== 'prod' && (
          <Button
            size="small"
            variant="contained"
            onClick={() => onPromote(rule.environment === 'dev' ? 'test' : 'prod')}
            disabled={isSaving}
            startIcon={<GitBranch size={14} />}
            sx={{ bgcolor: '#9333ea', '&:hover': { bgcolor: '#7e22ce' }, textTransform: 'none' }}
          >
            Promote to {rule.environment === 'dev' ? 'Test' : 'Prod'}
          </Button>
        )}
      </Box>
    </Box>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const UnifiedRuleEditor: React.FC<UnifiedRuleEditorProps> = ({
  rule: initialRule,
  tenantId,
  datasourceId,
  availableEntities,
  availableRules,
  onSave,
  onTest,
  onSubmitForApproval,
  onPromote
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const CATEGORY_CONFIG = getCategoryConfig(isDark);

  const [rule, setRule] = useState<Rule>(
    initialRule || createEmptyRule(tenantId, datasourceId)
  );
  const [activeTab, setActiveTab] = useState<'logic' | 'actions' | 'dependencies' | 'settings'>('logic');
  const [isSaving, setIsSaving] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [errors, setErrors] = useState<string[]>([]);

  const categoryConfig = CATEGORY_CONFIG[rule.category];
  const selectedEntity = availableEntities.find(e => e.name === rule.target_entity);

  useEffect(() => {
    if (!categoryConfig.allowedContexts.includes(rule.context)) {
      setRule(prev => ({ ...prev, context: categoryConfig.defaultContext }));
    }
  }, [rule.category, rule.context, categoryConfig]);

  const updateRule = useCallback((updates: Partial<Rule>) => {
    setRule(prev => ({ ...prev, ...updates }));
  }, []);

  const updateLogic = useCallback((logicIndex: number, updates: Partial<RuleLogic>) => {
    setRule(prev => ({
      ...prev,
      logic: prev.logic.map((l, i) => i === logicIndex ? { ...l, ...updates } : l)
    }));
  }, []);

  const handleSave = async () => {
    setIsSaving(true);
    setErrors([]);
    try {
      await onSave(rule);
    } catch (err) {
      setErrors([err instanceof Error ? err.message : 'Failed to save rule']);
    } finally {
      setIsSaving(false);
    }
  };

  const handleTest = async () => {
    setIsTesting(true);
    setTestResult(null);
    try {
      const result = await onTest(rule);
      setTestResult(result);
    } catch (err) {
      setErrors([err instanceof Error ? err.message : 'Failed to test rule']);
    } finally {
      setIsTesting(false);
    }
  };

  const handleSubmitForApproval = async () => {
    if (onSubmitForApproval) {
      setIsSaving(true);
      try {
        await onSubmitForApproval(rule);
        updateRule({ status: 'awaiting_approval' });
      } catch (err) {
        setErrors([err instanceof Error ? err.message : 'Failed to submit for approval']);
      } finally {
        setIsSaving(false);
      }
    }
  };

  const handlePromote = async (targetEnv: string) => {
    if (onPromote) {
      setIsSaving(true);
      try {
        await onPromote(rule, targetEnv);
      } catch (err) {
        setErrors([err instanceof Error ? err.message : 'Failed to promote rule']);
      } finally {
        setIsSaving(false);
      }
    }
  };

  const applyTemplate = (template: Partial<Rule>) => {
    setRule(prev => ({
      ...prev,
      ...template,
      logic: template.logic || prev.logic,
      actions: template.actions || prev.actions
    }));
  };

  return (
    <Paper sx={{ overflow: 'hidden', border: 1, borderColor: 'divider' }}>
      <GovernanceBar
        rule={rule}
        onSubmitForApproval={handleSubmitForApproval}
        onPromote={handlePromote}
        isSaving={isSaving}
      />

      <Box sx={{ p: 3, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Paper sx={{ p: 1.5, bgcolor: categoryConfig.bgColor, borderRadius: 1, alignSelf: 'flex-start' }}>
            <Box sx={{ color: categoryConfig.color }}>{categoryConfig.icon}</Box>
          </Paper>
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>Rule Name</Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={rule.name}
                  onChange={(e) => updateRule({ name: e.target.value })}
                  placeholder="e.g., check_required_fields"
                />
              </Box>
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>Display Name</Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={rule.display_name}
                  onChange={(e) => updateRule({ display_name: e.target.value })}
                  placeholder="e.g., Required Fields Validation"
                />
              </Box>
            </Box>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>Description</Typography>
              <TextField
                fullWidth
                size="small"
                multiline
                rows={2}
                value={rule.description}
                onChange={(e) => updateRule({ description: e.target.value })}
                placeholder="Describe what this rule validates..."
              />
            </Box>
          </Box>
        </Box>
      </Box>

      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider', bgcolor: isDark ? '#1f2937' : '#f9fafb' }}>
        <Typography variant="caption" sx={{ fontWeight: 600, mb: 1, display: 'block' }}>Rule Category</Typography>
        <CategorySelector
          value={rule.category}
          onChange={(cat) => updateRule({ category: cat, context: CATEGORY_CONFIG[cat].defaultContext })}
          disabled={!!initialRule}
        />
        <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>{categoryConfig.description}</Typography>

        {categoryConfig.presetTemplates.length > 0 && (
          <Box sx={{ mt: 1.5 }}>
            <Typography variant="caption" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>Quick Start Templates:</Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
              {categoryConfig.presetTemplates.map((tpl, idx) => (
                <Button
                  key={idx}
                  size="small"
                  variant="outlined"
                  onClick={() => applyTemplate(tpl.template)}
                  startIcon={<Copy size={12} />}
                  sx={{ fontSize: '0.75rem' }}
                >
                  {tpl.name}
                </Button>
              ))}
            </Box>
          </Box>
        )}
      </Box>

      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider', display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2 }}>
        <FormControl fullWidth size="small">
          <InputLabel>Target Entity</InputLabel>
          <Select
            value={rule.target_entity}
            label="Target Entity"
            onChange={(e) => updateRule({ target_entity: e.target.value })}
          >
            <MenuItem value="">Select entity...</MenuItem>
            {availableEntities.map(ent => (
              <MenuItem key={ent.name} value={ent.name}>{ent.name}</MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl fullWidth size="small">
          <InputLabel>Context</InputLabel>
          <Select
            value={rule.context}
            label="Context"
            onChange={(e) => updateRule({ context: e.target.value as RuleContext })}
          >
            {categoryConfig.allowedContexts.map(ctx => (
              <MenuItem key={ctx} value={ctx}>{ctx.replace('_', ' ')}</MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl fullWidth size="small">
          <InputLabel>Severity</InputLabel>
          <Select
            value={rule.severity}
            label="Severity"
            onChange={(e) => updateRule({ severity: e.target.value as RuleSeverity })}
          >
            {(Object.keys(SEVERITY_CONFIG) as RuleSeverity[]).map(sev => (
              <MenuItem key={sev} value={sev}>{SEVERITY_CONFIG[sev].label}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs
          value={activeTab}
          onChange={(_, v) => setActiveTab(v as typeof activeTab)}
          sx={{
            '& .MuiTab-root': {
              textTransform: 'none',
              minHeight: 48,
            },
          }}
        >
          <Tab value="logic" icon={<FileText size={16} />} label="Logic" iconPosition="start" />
          <Tab value="actions" icon={<Zap size={16} />} label="Actions" iconPosition="start" />
          <Tab value="dependencies" icon={<GitBranch size={16} />} label="Dependencies" iconPosition="start" />
          <Tab value="settings" icon={<Settings size={16} />} label="Settings" iconPosition="start" />
        </Tabs>
      </Box>

      <Box sx={{ p: 3 }}>
        {activeTab === 'logic' && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {rule.logic.map((logic, idx) => (
              <Paper key={idx} variant="outlined" sx={{ p: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>Logic Block {idx + 1}</Typography>
                  <FormControl size="small" sx={{ minWidth: 150 }}>
                    <Select
                      value={logic.logic_type}
                      onChange={(e) => updateLogic(idx, { logic_type: e.target.value as RuleLogic['logic_type'] })}
                    >
                      <MenuItem value="condition">Visual Condition</MenuItem>
                      <MenuItem value="expression">CEL Expression</MenuItem>
                      <MenuItem value="script">Script</MenuItem>
                    </Select>
                  </FormControl>
                </Box>

                {logic.logic_type === 'condition' && selectedEntity && React.createElement(
                  AdvancedConditionBuilder as any,
                  {
                    value: logic.condition_tree || { id: generateId(), type: 'group', operator: 'AND', conditions: [] },
                    onChange: (tree: any) => updateLogic(idx, { condition_tree: tree }),
                    availableFields: selectedEntity.fields,
                    entityName: rule.target_entity,
                  }
                )}

                {logic.logic_type === 'expression' && (
                  <Box>
                    <Typography variant="caption" sx={{ fontWeight: 500, mb: 0.5, display: 'block' }}>CEL Expression</Typography>
                    <TextField
                      fullWidth
                      multiline
                      rows={4}
                      value={logic.cel_expression || ''}
                      onChange={(e) => updateLogic(idx, { cel_expression: e.target.value })}
                      placeholder="record.amount > 0 && record.status != 'cancelled'"
                      sx={{ '& .MuiInputBase-input': { fontFamily: 'monospace', fontSize: '0.875rem' } }}
                    />
                  </Box>
                )}

                {logic.logic_type === 'script' && (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <FormControl size="small" sx={{ minWidth: 120 }}>
                      <Select
                        value={logic.script_language || 'javascript'}
                        onChange={(e) => updateLogic(idx, { script_language: e.target.value })}
                      >
                        <MenuItem value="javascript">JavaScript</MenuItem>
                        <MenuItem value="python">Python</MenuItem>
                      </Select>
                    </FormControl>
                    <TextField
                      fullWidth
                      multiline
                      rows={8}
                      value={logic.script_content || ''}
                      onChange={(e) => updateLogic(idx, { script_content: e.target.value })}
                      placeholder="// Custom validation logic..."
                      sx={{ '& .MuiInputBase-input': { fontFamily: 'monospace', fontSize: '0.875rem' } }}
                    />
                  </Box>
                )}
              </Paper>
            ))}
          </Box>
        )}

        {activeTab === 'actions' && (
          <ActionEditor
            actions={rule.actions}
            category={rule.category}
            onChange={(actions) => updateRule({ actions })}
          />
        )}

        {activeTab === 'dependencies' && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Paper sx={{ p: 2, bgcolor: isDark ? '#1e3a5f' : '#eff6ff', borderColor: '#3b82f6' }} variant="outlined">
              <Typography variant="body2">
                <strong>Rule Dependencies:</strong> Define which rules must pass before this rule executes.
                This creates a validation chain where dependent rules are evaluated first.
              </Typography>
            </Paper>
            <FormControl fullWidth>
              <InputLabel>Dependent Rules</InputLabel>
              <Select
                multiple
                value={rule.dependent_rule_ids}
                onChange={(e) => updateRule({
                  dependent_rule_ids: typeof e.target.value === 'string' ? e.target.value.split(',') : e.target.value
                }}
                label="Dependent Rules"
                sx={{ minHeight: 100 }}
              >
                {availableRules
                  .filter(r => r.id !== rule.id && r.category === rule.category)
                  .map(r => (
                    <MenuItem key={r.id} value={r.id}>{r.display_name || r.name}</MenuItem>
                  ))}
              </Select>
              <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5 }}>Hold Ctrl/Cmd to select multiple rules</Typography>
            </FormControl>
          </Box>
        )}

        {activeTab === 'settings' && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            <ExecutionPolicyEditor
              policies={rule.execution_policies}
              onChange={(policies) => updateRule({ execution_policies: policies })}
            />

            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, mb: 1, display: 'block' }}>Tags</Typography>
              <TextField
                fullWidth
                size="small"
                value={rule.tags.join(', ')}
                onChange={(e) => updateRule({ tags: e.target.value.split(',').map(t => t.trim()).filter(Boolean) })}
                placeholder="tag1, tag2, tag3"
              />
            </Box>

            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>Effective From</Typography>
                <TextField
                  fullWidth
                  type="datetime-local"
                  size="small"
                  value={rule.effective_from || ''}
                  onChange={(e) => updateRule({ effective_from: e.target.value })}
                  InputLabelProps={{ shrink: true }}
                />
              </Box>
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>Effective To</Typography>
                <TextField
                  fullWidth
                  type="datetime-local"
                  size="small"
                  value={rule.effective_to || ''}
                  onChange={(e) => updateRule({ effective_to: e.target.value })}
                  InputLabelProps={{ shrink: true }}
                />
              </Box>
            </Box>
          </Box>
        )}
      </Box>

      {errors.length > 0 && (
        <Box sx={{ mx: 3, mb: 2, p: 2, bgcolor: isDark ? '#450a0a' : '#fef2f2', border: 1, borderColor: 'error.main', borderRadius: 1 }}>
          {errors.map((err, idx) => (
            <Typography key={idx} variant="body2" sx={{ color: 'error.main' }}>{err}</Typography>
          ))}
        </Box>
      )}

      {testResult && (
        <Box sx={{ mx: 3, mb: 2, p: 2, borderRadius: 1, border: 1, borderColor: testResult.success ? 'success.main' : 'error.main', bgcolor: testResult.success ? isDark ? '#14532d' : '#f0fdf4' : isDark ? '#450a0a' : '#fef2f2' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            {testResult.success ? (
              <CheckCircle size={20} sx={{ color: '#16a34a' }} />
            ) : (
              <XCircle size={20} sx={{ color: '#dc2626' }} />
            )}
            <Typography variant="subtitle2" sx={{ fontWeight: 600, color: testResult.success ? '#166534' : '#991b1b' }}>
              Test {testResult.success ? 'Passed' : 'Failed'}
            </Typography>
          </Box>
          <Typography variant="body2" color="text.secondary">
            {testResult.passed} passed, {testResult.failed} failed
          </Typography>
          {testResult.sample_results.slice(0, 3).map((sr, idx) => (
            <Typography key={idx} variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
              {sr.record_id}: {sr.message}
            </Typography>
          ))}
        </Box>
      )}

      <Box sx={{ p: 2, borderTop: 1, borderColor: 'divider', display: 'flex', justifyContent: 'space-between', bgcolor: isDark ? '#1f2937' : '#f9fafb' }}>
        <Button
          variant="outlined"
          onClick={handleTest}
          disabled={isTesting || !rule.target_entity}
          startIcon={<Play size={16} />}
        >
          {isTesting ? 'Testing...' : 'Test Rule'}
        </Button>

        <Button
          variant="contained"
          onClick={handleSave}
          disabled={isSaving || !rule.name}
          startIcon={<Save size={16} />}
        >
          {isSaving ? 'Saving...' : 'Save Rule'}
        </Button>
      </Box>
    </Paper>
  );
};

export default UnifiedRuleEditor;
