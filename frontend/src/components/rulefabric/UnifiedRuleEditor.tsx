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
import { AdvancedConditionBuilder, ConditionGroup } from '../ExpressionBuilder/AdvancedConditionBuilder';

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

const CATEGORY_CONFIG: Record<RuleCategory, {
  icon: React.ReactNode;
  color: string;
  bgColor: string;
  label: string;
  description: string;
  defaultContext: RuleContext;
  allowedContexts: RuleContext[];
  actionTypes: Array<{ value: string; label: string; description: string }>;
  presetTemplates: Array<{ name: string; description: string; template: Partial<Rule> }>;
}> = {
  data_quality: {
    icon: <Database size={20} />,
    color: 'text-blue-600',
    bgColor: 'bg-blue-50',
    label: 'Data Quality',
    description: 'Validate data integrity, completeness, and accuracy',
    defaultContext: 'data_record',
    allowedContexts: ['data_record', 'mdm_group', 'system_job'],
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
    color: 'text-purple-600',
    bgColor: 'bg-purple-50',
    label: 'Compliance',
    description: 'Enforce regulatory and policy requirements',
    defaultContext: 'trade_event',
    allowedContexts: ['trade_event', 'portfolio', 'client_profile'],
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
    color: 'text-green-600',
    bgColor: 'bg-green-50',
    label: 'Master Data',
    description: 'Manage data matching, merging, and survivorship',
    defaultContext: 'mdm_group',
    allowedContexts: ['mdm_group', 'data_record'],
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
    color: 'text-red-600',
    bgColor: 'bg-red-50',
    label: 'Wash Trade',
    description: 'Detect and prevent wash trading patterns',
    defaultContext: 'trade_event',
    allowedContexts: ['trade_event', 'portfolio'],
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
    color: 'text-teal-600',
    bgColor: 'bg-teal-50',
    label: 'Values/ESG',
    description: 'Enforce values-based and ESG investment policies',
    defaultContext: 'portfolio',
    allowedContexts: ['portfolio', 'trade_event', 'client_profile'],
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
    color: 'text-orange-600',
    bgColor: 'bg-orange-50',
    label: 'Workflow',
    description: 'Define business process rules and routing',
    defaultContext: 'system_job',
    allowedContexts: ['system_job', 'data_record', 'trade_event'],
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
    color: 'text-gray-600',
    bgColor: 'bg-gray-50',
    label: 'Custom',
    description: 'Define custom business rules',
    defaultContext: 'data_record',
    allowedContexts: ['data_record', 'trade_event', 'portfolio', 'client_profile', 'mdm_group', 'system_job'],
    actionTypes: [
      { value: 'custom_action', label: 'Custom Action', description: 'Execute custom action handler' },
      { value: 'webhook', label: 'Webhook', description: 'Call external webhook' },
      { value: 'log_event', label: 'Log Event', description: 'Log custom event' },
      { value: 'send_notification', label: 'Send Notification', description: 'Send custom notification' }
    ],
    presetTemplates: []
  }
};

const SEVERITY_CONFIG: Record<RuleSeverity, { icon: React.ReactNode; color: string; bgColor: string; label: string }> = {
  error: { icon: <XCircle size={16} />, color: 'text-red-600', bgColor: 'bg-red-100', label: 'Error' },
  warning: { icon: <AlertTriangle size={16} />, color: 'text-yellow-600', bgColor: 'bg-yellow-100', label: 'Warning' },
  info: { icon: <Eye size={16} />, color: 'text-blue-600', bgColor: 'bg-blue-100', label: 'Info' },
  hard_block: { icon: <Lock size={16} />, color: 'text-red-700', bgColor: 'bg-red-200', label: 'Hard Block' },
  soft_block: { icon: <Unlock size={16} />, color: 'text-orange-600', bgColor: 'bg-orange-100', label: 'Soft Block' }
};

const STATUS_CONFIG: Record<RuleStatus, { color: string; bgColor: string; label: string }> = {
  draft: { color: 'text-gray-600', bgColor: 'bg-gray-100', label: 'Draft' },
  awaiting_approval: { color: 'text-yellow-600', bgColor: 'bg-yellow-100', label: 'Awaiting Approval' },
  active: { color: 'text-green-600', bgColor: 'bg-green-100', label: 'Active' },
  inactive: { color: 'text-gray-500', bgColor: 'bg-gray-100', label: 'Inactive' },
  deprecated: { color: 'text-red-500', bgColor: 'bg-red-100', label: 'Deprecated' }
};

// ============================================================================
// Helper Functions
// ============================================================================

const generateId = () => `rule_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

const createEmptyRule = (tenantId: string, datasourceId: string, category: RuleCategory = 'data_quality'): Rule => ({
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
});

// ============================================================================
// Sub-Components
// ============================================================================

interface CategorySelectorProps {
  value: RuleCategory;
  onChange: (category: RuleCategory) => void;
  disabled?: boolean;
}

const CategorySelector: React.FC<CategorySelectorProps> = ({ value, onChange, disabled }) => (
  <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-2">
    {(Object.keys(CATEGORY_CONFIG) as RuleCategory[]).map(cat => {
      const config = CATEGORY_CONFIG[cat];
      const isSelected = value === cat;
      return (
        <button
          key={cat}
          type="button"
          disabled={disabled}
          onClick={() => onChange(cat)}
          className={`p-3 rounded-lg border-2 transition-all ${
            isSelected
              ? `${config.bgColor} border-current ${config.color}`
              : 'border-gray-200 hover:border-gray-300 bg-white'
          } ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
        >
          <div className={`flex flex-col items-center gap-1 ${isSelected ? config.color : 'text-gray-600'}`}>
            {config.icon}
            <span className="text-xs font-medium">{config.label}</span>
          </div>
        </button>
      );
    })}
  </div>
);

interface ActionEditorProps {
  actions: RuleAction[];
  category: RuleCategory;
  onChange: (actions: RuleAction[]) => void;
}

const ActionEditor: React.FC<ActionEditorProps> = ({ actions, category, onChange }) => {
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
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <label className="text-sm font-semibold text-gray-700">Actions</label>
        <button
          type="button"
          onClick={addAction}
          className="flex items-center gap-1 text-sm text-blue-600 hover:text-blue-700"
        >
          <Plus size={14} /> Add Action
        </button>
      </div>

      {actions.length === 0 ? (
        <div className="text-center py-6 border-2 border-dashed border-gray-300 rounded-lg">
          <p className="text-gray-500 text-sm">No actions configured</p>
          <p className="text-gray-400 text-xs mt-1">Add actions to define what happens when the rule triggers</p>
        </div>
      ) : (
        <div className="space-y-2">
          {actions.map((action, index) => (
            <div key={index} className="flex items-center gap-2 p-3 bg-gray-50 rounded-lg border">
              <span className="w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-xs font-semibold">
                {index + 1}
              </span>
              <select
                value={action.action_type}
                onChange={(e) => updateAction(index, { action_type: e.target.value })}
                aria-label={`Action type for step ${index + 1}`}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm"
              >
                {actionTypes.map(at => (
                  <option key={at.value} value={at.value}>{at.label}</option>
                ))}
              </select>
              <button
                type="button"
                onClick={() => removeAction(index)}
                aria-label={`Remove action at step ${index + 1}`}
                className="p-2 text-red-600 hover:bg-red-50 rounded"
              >
                <Trash2 size={16} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

interface ExecutionPolicyEditorProps {
  policies: ExecutionPolicy[];
  onChange: (policies: ExecutionPolicy[]) => void;
}

const ExecutionPolicyEditor: React.FC<ExecutionPolicyEditorProps> = ({ policies, onChange }) => {
  const [expanded, setExpanded] = useState(false);

  const updatePolicy = (channel: ExecutionChannel, updates: Partial<ExecutionPolicy>) => {
    const newPolicies = policies.map(p =>
      p.channel === channel ? { ...p, ...updates } : p
    );
    onChange(newPolicies);
  };

  return (
    <div className="border rounded-lg">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between p-3 hover:bg-gray-50"
      >
        <div className="flex items-center gap-2">
          <Settings size={16} className="text-gray-500" />
          <span className="text-sm font-semibold text-gray-700">Execution Policies</span>
        </div>
        {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </button>

      {expanded && (
        <div className="p-3 border-t space-y-3">
          {policies.map(policy => (
            <div key={policy.channel} className="flex items-center gap-4 p-2 bg-gray-50 rounded">
              <label className="flex items-center gap-2 min-w-[120px]">
                <input
                  type="checkbox"
                  checked={policy.is_enabled}
                  onChange={(e) => updatePolicy(policy.channel, { is_enabled: e.target.checked })}
                  className="rounded"
                />
                <span className="text-sm font-medium capitalize">{policy.channel}</span>
              </label>
              {policy.is_enabled && (
                <>
                  <div className="flex items-center gap-1">
                    <Clock size={14} className="text-gray-400" />
                    <input
                      type="number"
                      value={policy.timeout_seconds}
                      onChange={(e) => updatePolicy(policy.channel, { timeout_seconds: parseInt(e.target.value) })}
                      className="w-16 px-2 py-1 border rounded text-sm"
                      title="Timeout (seconds)"
                    />
                    <span className="text-xs text-gray-500">sec</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Users size={14} className="text-gray-400" />
                    <input
                      type="number"
                      value={policy.max_concurrent}
                      onChange={(e) => updatePolicy(policy.channel, { max_concurrent: parseInt(e.target.value) })}
                      className="w-16 px-2 py-1 border rounded text-sm"
                      title="Max concurrent"
                    />
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

interface GovernanceBarProps {
  rule: Rule;
  onSubmitForApproval: () => void;
  onPromote: (env: string) => void;
  isSaving: boolean;
}

const GovernanceBar: React.FC<GovernanceBarProps> = ({ rule, onSubmitForApproval, onPromote, isSaving }) => {
  const statusConfig = STATUS_CONFIG[rule.status];

  return (
    <div className="flex items-center justify-between p-3 bg-gray-50 border-b">
      <div className="flex items-center gap-4">
        <div className={`px-3 py-1 rounded-full text-xs font-semibold ${statusConfig.bgColor} ${statusConfig.color}`}>
          {statusConfig.label}
        </div>
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <History size={14} />
          <span>v{rule.version}</span>
        </div>
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <span className={`px-2 py-0.5 rounded text-xs ${
            rule.environment === 'prod' ? 'bg-green-100 text-green-700' :
            rule.environment === 'test' ? 'bg-yellow-100 text-yellow-700' :
            'bg-gray-100 text-gray-700'
          }`}>
            {rule.environment.toUpperCase()}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        {rule.status === 'draft' && (
          <button
            type="button"
            onClick={onSubmitForApproval}
            disabled={isSaving}
            className="flex items-center gap-2 px-3 py-1.5 bg-yellow-500 text-white rounded-lg text-sm hover:bg-yellow-600 disabled:opacity-50"
          >
            <Send size={14} />
            Submit for Approval
          </button>
        )}
        {rule.status === 'active' && rule.environment !== 'prod' && (
          <button
            type="button"
            onClick={() => onPromote(rule.environment === 'dev' ? 'test' : 'prod')}
            disabled={isSaving}
            className="flex items-center gap-2 px-3 py-1.5 bg-purple-500 text-white rounded-lg text-sm hover:bg-purple-600 disabled:opacity-50"
          >
            <GitBranch size={14} />
            Promote to {rule.environment === 'dev' ? 'Test' : 'Prod'}
          </button>
        )}
      </div>
    </div>
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

  // Update context when category changes
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
    <div className="bg-white rounded-lg shadow-lg border overflow-hidden">
      {/* Governance Bar */}
      <GovernanceBar
        rule={rule}
        onSubmitForApproval={handleSubmitForApproval}
        onPromote={handlePromote}
        isSaving={isSaving}
      />

      {/* Header */}
      <div className="p-6 border-b">
        <div className="flex items-start gap-4">
          <div className={`p-3 rounded-lg ${categoryConfig.bgColor}`}>
            <span className={categoryConfig.color}>{categoryConfig.icon}</span>
          </div>
          <div className="flex-1 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1">Rule Name</label>
                <input
                  type="text"
                  value={rule.name}
                  onChange={(e) => updateRule({ name: e.target.value })}
                  placeholder="e.g., check_required_fields"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1">Display Name</label>
                <input
                  type="text"
                  value={rule.display_name}
                  onChange={(e) => updateRule({ display_name: e.target.value })}
                  placeholder="e.g., Required Fields Validation"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1">Description</label>
              <textarea
                value={rule.description}
                onChange={(e) => updateRule({ description: e.target.value })}
                placeholder="Describe what this rule validates..."
                className="w-full px-3 py-2 border border-gray-300 rounded-lg resize-none"
                rows={2}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Category Selector */}
      <div className="p-4 border-b bg-gray-50">
        <label className="block text-sm font-semibold text-gray-700 mb-2">Rule Category</label>
        <CategorySelector
          value={rule.category}
          onChange={(cat) => updateRule({ category: cat, context: CATEGORY_CONFIG[cat].defaultContext })}
          disabled={!!initialRule}
        />
        <p className="text-xs text-gray-500 mt-2">{categoryConfig.description}</p>

        {/* Quick Templates */}
        {categoryConfig.presetTemplates.length > 0 && (
          <div className="mt-3">
            <label className="block text-xs font-semibold text-gray-600 mb-1">Quick Start Templates:</label>
            <div className="flex flex-wrap gap-2">
              {categoryConfig.presetTemplates.map((tpl, idx) => (
                <button
                  key={idx}
                  type="button"
                  onClick={() => applyTemplate(tpl.template)}
                  className="px-2 py-1 text-xs bg-white border rounded hover:bg-gray-100"
                  title={tpl.description}
                >
                  <Copy size={12} className="inline mr-1" />
                  {tpl.name}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Entity & Context Selection */}
      <div className="p-4 border-b grid grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-semibold text-gray-700 mb-1">Target Entity</label>
          <select
            value={rule.target_entity}
            onChange={(e) => updateRule({ target_entity: e.target.value })}
            aria-label="Target Entity"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg"
          >
            <option value="">Select entity...</option>
            {availableEntities.map(ent => (
              <option key={ent.name} value={ent.name}>{ent.name}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-semibold text-gray-700 mb-1">Context</label>
          <select
            value={rule.context}
            onChange={(e) => updateRule({ context: e.target.value as RuleContext })}
            aria-label="Context"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg"
          >
            {categoryConfig.allowedContexts.map(ctx => (
              <option key={ctx} value={ctx}>{ctx.replace('_', ' ')}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-semibold text-gray-700 mb-1">Severity</label>
          <select
            value={rule.severity}
            onChange={(e) => updateRule({ severity: e.target.value as RuleSeverity })}
            aria-label="Severity"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg"
          >
            {(Object.keys(SEVERITY_CONFIG) as RuleSeverity[]).map(sev => (
              <option key={sev} value={sev}>{SEVERITY_CONFIG[sev].label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b">
        <div className="flex">
          {[
            { key: 'logic', label: 'Logic', icon: <FileText size={16} /> },
            { key: 'actions', label: 'Actions', icon: <Zap size={16} /> },
            { key: 'dependencies', label: 'Dependencies', icon: <GitBranch size={16} /> },
            { key: 'settings', label: 'Settings', icon: <Settings size={16} /> }
          ].map(tab => (
            <button
              key={tab.key}
              type="button"
              onClick={() => setActiveTab(tab.key as typeof activeTab)}
              className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.key
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      <div className="p-6">
        {activeTab === 'logic' && (
          <div className="space-y-4">
            {rule.logic.map((logic, idx) => (
              <div key={idx} className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-4">
                  <h4 className="font-semibold text-gray-900">Logic Block {idx + 1}</h4>
                  <select
                    value={logic.logic_type}
                    onChange={(e) => updateLogic(idx, { logic_type: e.target.value as RuleLogic['logic_type'] })}
                    aria-label={`Logic type for block ${idx + 1}`}
                    className="px-2 py-1 border rounded text-sm"
                  >
                    <option value="condition">Visual Condition</option>
                    <option value="expression">CEL Expression</option>
                    <option value="script">Script</option>
                  </select>
                </div>

                {logic.logic_type === 'condition' && selectedEntity && (
                  <AdvancedConditionBuilder
                    value={logic.condition_tree || { id: generateId(), type: 'group', operator: 'AND', conditions: [] }}
                    onChange={(tree) => updateLogic(idx, { condition_tree: tree })}
                    availableFields={selectedEntity.fields}
                    entityName={rule.target_entity}
                  />
                )}

                {logic.logic_type === 'expression' && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">CEL Expression</label>
                    <textarea
                      value={logic.cel_expression || ''}
                      onChange={(e) => updateLogic(idx, { cel_expression: e.target.value })}
                      placeholder="record.amount > 0 && record.status != 'cancelled'"
                      className="w-full px-3 py-2 border rounded-lg font-mono text-sm"
                      rows={4}
                    />
                  </div>
                )}

                {logic.logic_type === 'script' && (
                  <div className="space-y-2">
                    <select
                      value={logic.script_language || 'javascript'}
                      onChange={(e) => updateLogic(idx, { script_language: e.target.value })}
                      aria-label={`Script language for block ${idx + 1}`}
                      className="px-2 py-1 border rounded text-sm"
                    >
                      <option value="javascript">JavaScript</option>
                      <option value="python">Python</option>
                    </select>
                    <textarea
                      value={logic.script_content || ''}
                      onChange={(e) => updateLogic(idx, { script_content: e.target.value })}
                      placeholder="// Custom validation logic..."
                      className="w-full px-3 py-2 border rounded-lg font-mono text-sm"
                      rows={8}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {activeTab === 'actions' && (
          <ActionEditor
            actions={rule.actions}
            category={rule.category}
            onChange={(actions) => updateRule({ actions })}
          />
        )}

        {activeTab === 'dependencies' && (
          <div className="space-y-4">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <p className="text-sm text-blue-800">
                <strong>Rule Dependencies:</strong> Define which rules must pass before this rule executes.
                This creates a validation chain where dependent rules are evaluated first.
              </p>
            </div>
            <div>
              <label htmlFor={`dependent-rules-${rule.id}`} className="block text-sm font-semibold text-gray-700 mb-2">Dependent Rules</label>
              <select
                id={`dependent-rules-${rule.id}`}
                multiple
                value={rule.dependent_rule_ids}
                onChange={(e) => updateRule({
                  
                  dependent_rule_ids: Array.from(e.target.selectedOptions, opt => opt.value)
                })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg h-40"
              >
                {availableRules
                  .filter(r => r.id !== rule.id && r.category === rule.category)
                  .map(r => (
                    <option key={r.id} value={r.id}>{r.display_name || r.name}</option>
                  ))}
              </select>
              <p className="text-xs text-gray-500 mt-1">Hold Ctrl/Cmd to select multiple rules</p>
            </div>
          </div>
        )}

        {activeTab === 'settings' && (
          <div className="space-y-6">
            <ExecutionPolicyEditor
              policies={rule.execution_policies}
              onChange={(policies) => updateRule({ execution_policies: policies })}
            />

            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">Tags</label>
              <input
                type="text"
                value={rule.tags.join(', ')}
                onChange={(e) => updateRule({ tags: e.target.value.split(',').map(t => t.trim()).filter(Boolean) })}
                placeholder="tag1, tag2, tag3"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1">Effective From</label>
                <input
                  type="datetime-local"
                  value={rule.effective_from || ''}
                  onChange={(e) => updateRule({ effective_from: e.target.value })}
                  aria-label="Effective From"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1">Effective To</label>
                <input
                  type="datetime-local"
                  value={rule.effective_to || ''}
                  onChange={(e) => updateRule({ effective_to: e.target.value })}
                  aria-label="Effective To"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg"
                />
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Errors */}
      {errors.length > 0 && (
        <div className="mx-6 mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
          {errors.map((err, idx) => (
            <p key={idx} className="text-sm text-red-700">{err}</p>
          ))}
        </div>
      )}

      {/* Test Results */}
      {testResult && (
        <div className={`mx-6 mb-4 p-4 rounded-lg border ${
          testResult.success ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'
        }`}>
          <div className="flex items-center gap-2 mb-2">
            {testResult.success ? (
              <CheckCircle size={20} className="text-green-600" />
            ) : (
              <XCircle size={20} className="text-red-600" />
            )}
            <span className={`font-semibold ${testResult.success ? 'text-green-700' : 'text-red-700'}`}>
              Test {testResult.success ? 'Passed' : 'Failed'}
            </span>
          </div>
          <div className="text-sm text-gray-600">
            {testResult.passed} passed, {testResult.failed} failed
          </div>
          {testResult.sample_results.slice(0, 3).map((sr, idx) => (
            <div key={idx} className="text-xs text-gray-500 mt-1">
              {sr.record_id}: {sr.message}
            </div>
          ))}
        </div>
      )}

      {/* Footer Actions */}
      <div className="p-4 border-t bg-gray-50 flex items-center justify-between">
        <button
          type="button"
          onClick={handleTest}
          disabled={isTesting || !rule.target_entity}
          className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-100 disabled:opacity-50"
        >
          <Play size={16} />
          {isTesting ? 'Testing...' : 'Test Rule'}
        </button>

        <button
          type="button"
          onClick={handleSave}
          disabled={isSaving || !rule.name}
          className="flex items-center gap-2 px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
        >
          <Save size={16} />
          {isSaving ? 'Saving...' : 'Save Rule'}
        </button>
      </div>
    </div>
  );
};

export default UnifiedRuleEditor;
