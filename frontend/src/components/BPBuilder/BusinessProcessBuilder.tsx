import React, { useState, useCallback, useMemo } from 'react';
import { devDebug, devError } from '../../utils/devLogger';
import {
  Plus, Trash2, Clock, User, CheckCircle, AlertTriangle, FileText, Send, GitBranch,
  Settings, Play, Save, ChevronUp, ChevronDown, Eye, EyeOff, Copy, Download, Upload as _Upload,
  AlertCircle as _AlertCircle, TrendingUp, Zap
} from 'lucide-react';
import { useNotification } from '../../hooks/useNotification';

// Types
interface BPStep {
  id: string;
  stepOrder: number;
  stepType: 'data_entry' | 'validate' | 'approve' | 'notify' | 'integrate' | 'condition';
  stepName: string;
  durationHours: number;
  assigneeRole?: string;
  assigneeUser?: string;
  validationRules?: string[];
  notificationTemplate?: string;
  conditionLogic?: ConditionBranch;
  description?: string;
  status?: 'pending' | 'active' | 'completed' | 'failed';
  escalationThresholdHours?: number;
}

interface ConditionBranch {
  condition: string;
  trueStepId?: string;
  falseStepId?: string;
}

interface BusinessProcess {
  id: string;
  processName: string;
  entity: string;
  description: string;
  steps: BPStep[];
  isActive: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt?: string;
  version: number;
  tags?: string[];
}

// Step Type Configurations
const STEP_TYPES = [
  {
    type: 'data_entry',
    label: 'Data Entry',
    icon: FileText,
    color: 'blue',
    description: 'Collect information from user',
    badge: 'INPUT'
  },
  {
    type: 'validate',
    label: 'Validation',
    icon: CheckCircle,
    color: 'green',
    description: 'Run validation rules',
    badge: 'VALIDATE'
  },
  {
    type: 'approve',
    label: 'Approval',
    icon: User,
    color: 'purple',
    description: 'Require approval from user/role',
    badge: 'APPROVE'
  },
  {
    type: 'notify',
    label: 'Notification',
    icon: Send,
    color: 'orange',
    description: 'Send email/SMS notification',
    badge: 'NOTIFY'
  },
  {
    type: 'integrate',
    label: 'Integration',
    icon: Settings,
    color: 'indigo',
    description: 'Call external API/system',
    badge: 'INTEGRATE'
  },
  {
    type: 'condition',
    label: 'Conditional Branch',
    icon: GitBranch,
    color: 'yellow',
    description: 'Branch based on conditions',
    badge: 'BRANCH'
  }
];

const AVAILABLE_ROLES = [
  'Manager',
  'HR Admin',
  'Department Head',
  'Finance Approver',
  'System Admin',
  'Compliance Officer',
  'Process Owner',
  'Data Steward'
];

const AVAILABLE_ENTITIES = [
  'Employee',
  'Order',
  'Invoice',
  'Request',
  'User',
  'Account',
  'Contract',
  'Expense',
  'Leave Request',
  'Promotion'
];

// Step Configuration Component
const StepConfigurator: React.FC<{
  step: BPStep;
  onUpdate: (step: BPStep) => void;
  onDelete: () => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  availableRules: string[];
  canMoveUp?: boolean;
  canMoveDown?: boolean;
}> = ({ step, onUpdate, onDelete, onMoveUp, onMoveDown, availableRules, canMoveUp, canMoveDown }) => {
  const stepConfig = STEP_TYPES.find(t => t.type === step.stepType);
  const Icon = stepConfig?.icon || FileText;
  const colorMap = {
    blue: { bg: 'bg-blue-100', text: 'text-blue-600', border: 'border-blue-300', hover: 'hover:border-blue-400' },
    green: { bg: 'bg-green-100', text: 'text-green-600', border: 'border-green-300', hover: 'hover:border-green-400' },
    purple: { bg: 'bg-purple-100', text: 'text-purple-600', border: 'border-purple-300', hover: 'hover:border-purple-400' },
    orange: { bg: 'bg-orange-100', text: 'text-orange-600', border: 'border-orange-300', hover: 'hover:border-orange-400' },
    indigo: { bg: 'bg-indigo-100', text: 'text-indigo-600', border: 'border-indigo-300', hover: 'hover:border-indigo-400' },
    yellow: { bg: 'bg-yellow-100', text: 'text-yellow-600', border: 'border-yellow-300', hover: 'hover:border-yellow-400' }
  };
  const colors = colorMap[stepConfig?.color as keyof typeof colorMap] || colorMap.blue;

  return (
    <div className={`bg-white border-2 ${colors.border} rounded-lg p-6 ${colors.hover} transition-all shadow-md`}>
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3 flex-1">
          <div className={`w-12 h-12 ${colors.bg} rounded-lg flex items-center justify-center flex-shrink-0`}>
            <Icon className={colors.text} size={24} />
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1">
              <span className="w-8 h-8 bg-gray-700 text-white rounded-full flex items-center justify-center text-sm font-bold flex-shrink-0">
                {step.stepOrder}
              </span>
              <input
                type="text"
                value={step.stepName}
                onChange={(e) => onUpdate({ ...step, stepName: e.target.value })}
                className="text-lg font-semibold text-gray-900 border-b-2 border-transparent hover:border-gray-300 focus:border-blue-500 focus:outline-none px-2 flex-1"
                placeholder="Step name..."
              />
              <span className={`px-2 py-1 ${colors.bg} ${colors.text} text-xs font-bold rounded`}>
                {stepConfig?.badge}
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-1">{stepConfig?.description}</p>
          </div>
        </div>

        <div className="flex items-center gap-1 ml-4">
          {onMoveUp && (
            <button
              onClick={onMoveUp}
              disabled={!canMoveUp}
              className="text-gray-600 hover:bg-gray-100 p-2 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title="Move step up"
            >
              <ChevronUp size={20} />
            </button>
          )}
          {onMoveDown && (
            <button
              onClick={onMoveDown}
              disabled={!canMoveDown}
              className="text-gray-600 hover:bg-gray-100 p-2 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title="Move step down"
            >
              <ChevronDown size={20} />
            </button>
          )}
          <button
            onClick={onDelete}
            className="text-red-600 hover:bg-red-50 p-2 rounded transition-colors"
            title="Delete step"
          >
            <Trash2 size={20} />
          </button>
        </div>
      </div>

      {/* Step-specific configuration */}
      <div className="space-y-4 mt-6 border-t border-gray-200 pt-4">
        {/* Common: Duration */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2">
              <Clock className="inline mr-1" size={16} />
              Duration (hours)
            </label>
            <input
              type="number"
              value={step.durationHours}
              onChange={(e) => onUpdate({ ...step, durationHours: parseInt(e.target.value) || 0 })}
              title="Duration in hours"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              min="0"
            />
          </div>

          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2">
              <AlertTriangle className="inline mr-1" size={16} />
              Escalation Threshold (hours)
            </label>
            <input
              type="number"
              value={step.escalationThresholdHours || step.durationHours}
              onChange={(e) => onUpdate({ ...step, escalationThresholdHours: parseInt(e.target.value) || 0 })}
              title="Escalation threshold in hours"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              min="0"
              placeholder="Auto-escalate after X hours"
            />
          </div>
        </div>

        {/* Description */}
        <div>
          <label className="block text-sm font-semibold text-gray-700 mb-2">
            Description
          </label>
          <textarea
            value={step.description || ''}
            onChange={(e) => onUpdate({ ...step, description: e.target.value })}
              title="Step description"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
            placeholder="What happens in this step..."
            rows={2}
          />
        </div>

        {/* Validation Step - Select Rules */}
        {step.stepType === 'validate' && (
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <CheckCircle size={16} />
              Validation Rules to Execute
            </label>
            <div className="space-y-2 max-h-48 overflow-y-auto border border-gray-300 rounded-lg p-3 bg-gray-50">
              {availableRules.length === 0 ? (
                <p className="text-sm text-gray-500 italic">No validation rules available</p>
              ) : (
                availableRules.map((rule) => (
                  <label key={rule} className="flex items-center gap-2 cursor-pointer hover:bg-white p-2 rounded">
                    <input
                      type="checkbox"
                      checked={step.validationRules?.includes(rule) || false}
                      onChange={(e) => {
                        const rules = step.validationRules || [];
                        const updated = e.target.checked
                          ? [...rules, rule]
                          : rules.filter(r => r !== rule);
                        onUpdate({ ...step, validationRules: updated });
                      }}
                      className="w-4 h-4 text-blue-600 rounded"
                    />
                    <span className="text-sm text-gray-700">{rule}</span>
                  </label>
                ))
              )}
            </div>
            {(!step.validationRules || step.validationRules.length === 0) && (
              <p className="text-xs text-orange-600 mt-2">⚠️ No validation rules selected</p>
            )}
          </div>
        )}

        {/* Approval Step - Assignee */}
        {step.stepType === 'approve' && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">
                <User className="inline mr-1" size={16} />
                Assignee Role
              </label>
              <select
                value={step.assigneeRole || ''}
                onChange={(e) => onUpdate({ ...step, assigneeRole: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                title="Select assignee role"
              >
                <option value="">Select role...</option>
                {AVAILABLE_ROLES.map(role => (
                  <option key={role} value={role}>{role}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">
                Or Specific User
              </label>
              <input
                type="email"
                value={step.assigneeUser || ''}
                onChange={(e) => onUpdate({ ...step, assigneeUser: e.target.value })}
                title="Specific user email"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                placeholder="user@example.com"
              />
            </div>
          </div>
        )}

        {/* Notification Step - Template */}
        {step.stepType === 'notify' && (
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <Send size={16} />
              Notification Template
            </label>
            <textarea
              value={step.notificationTemplate || ''}
              onChange={(e) => onUpdate({ ...step, notificationTemplate: e.target.value })}
              rows={4}
              title="Notification template"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              placeholder={'Subject: {{subject}}\nBody: Hi {{name}}, your request has been {{status}}.'}
            />
            <p className="text-xs text-gray-500 mt-1">
              💡 Use &#123;&#123;variable&#125;&#125; syntax for dynamic values from the entity
            </p>
          </div>
        )}

        {/* Conditional Branch */}
        {step.stepType === 'condition' && (
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
                <GitBranch size={16} />
                Condition Logic
              </label>
              <input
                type="text"
                value={step.conditionLogic?.condition || ''}
                onChange={(e) => onUpdate({
                  ...step,
                  conditionLogic: { ...step.conditionLogic!, condition: e.target.value }
                })}
                title="Condition logic"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                placeholder="e.g., amount > 10000 OR vip_status = true"
              />
              <p className="text-xs text-gray-500 mt-1">
                💡 Example: salary {'>'} 100000 AND department = 'Engineering'
              </p>
            </div>
          </div>
        )}

        {/* Integration Step */}
        {step.stepType === 'integrate' && (
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <Settings size={16} />
              API Endpoint
            </label>
            <input
              type="url"
              title="API endpoint URL"
              placeholder="https://api.example.com/webhook"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
            />
            <p className="text-xs text-gray-500 mt-1">
              💡 POST request with entity data as payload
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

// Main BP Builder Component
export const BusinessProcessBuilder: React.FC = () => {
  const notification = useNotification();
  const [process, setProcess] = useState<BusinessProcess>({
    id: 'bp_new',
    processName: 'New Business Process',
    entity: 'Employee',
    description: '',
    steps: [],
    isActive: false,
    createdBy: 'Current User',
    createdAt: new Date().toISOString(),
    version: 1,
    tags: []
  });

  const [showPreview, setShowPreview] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'success' | 'error'>('idle');

  // Mock validation rules from all validators
  const availableRules = useMemo(() => [
    'Email Format Validation',
    'Age Verification (18+)',
    'Salary Range Check',
    'Duplicate Email Check',
    'Required Fields Validation',
    'Date Range Validation',
    'Cross-Entity Validation',
    'Business Rule Engine Check',
    'Data Quality Check',
    'Compliance Validation'
  ], []);

  const addStep = useCallback((stepType: string) => {
    const newStep: BPStep = {
      id: `step_${Date.now()}`,
      stepOrder: process.steps.length + 1,
      stepType: stepType as any,
      stepName: `${STEP_TYPES.find(t => t.type === stepType)?.label} Step`,
      durationHours: stepType === 'approve' ? 48 : 24,
      escalationThresholdHours: stepType === 'approve' ? 72 : 48,
      validationRules: [],
      description: ''
    };

    setProcess({
      ...process,
      steps: [...process.steps, newStep]
    });
  }, [process]);

  const updateStep = useCallback((stepId: string, updatedStep: BPStep) => {
    setProcess({
      ...process,
      steps: process.steps.map((s: BPStep) => s.id === stepId ? updatedStep : s)
    });
  }, [process]);

  const deleteStep = useCallback((stepId: string) => {
    const filtered = process.steps.filter((s: BPStep) => s.id !== stepId);
    const reordered = filtered.map((step: BPStep, idx: number) => ({ ...step, stepOrder: idx + 1 }));
    setProcess({ ...process, steps: reordered });
  }, [process]);

  const moveStep = useCallback((stepId: string, direction: 'up' | 'down') => {
    const index = process.steps.findIndex((s: BPStep) => s.id === stepId);
    if (
      (direction === 'up' && index === 0) ||
      (direction === 'down' && index === process.steps.length - 1)
    ) {
      return;
    }

    const newSteps = [...process.steps];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    [newSteps[index], newSteps[targetIndex]] = [newSteps[targetIndex], newSteps[index]];

    const reordered = newSteps.map((step: BPStep, idx: number) => ({ ...step, stepOrder: idx + 1 }));
    setProcess({ ...process, steps: reordered });
  }, [process]);

  const saveBP = async () => {
    if (process.steps.length === 0) {
      notification.error('Please add at least one step to save the process');
      return;
    }

    setIsSaving(true);
    setSaveStatus('idle');

    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1500));
  devDebug('Saving BP:', process);
      setSaveStatus('success');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (error) {
      devError('Error saving BP:', error);
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } finally {
      setIsSaving(false);
    }
  };

  const simulateBP = () => {
    if (process.steps.length === 0) {
      notification.error('Please add at least one step to simulate the process');
      return;
    }
    notification.info('🎬 Simulating BP execution...\n\nThis would:\n1. Start a Temporal workflow\n2. Execute each step in sequence\n3. Show real-time progress\n4. Handle escalations on timeout\n\nProcess: ' + process.processName + '\nSteps: ' + process.steps.length);
  };

  const exportBP = () => {
    const json = JSON.stringify(process, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${process.processName.toLowerCase().replace(/\s+/g, '_')}_bp_v${process.version}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const duplicateBP = () => {
    const newProcess = {
      ...process,
      id: `bp_${Date.now()}`,
      processName: `${process.processName} (Copy)`,
      version: 1,
      createdAt: new Date().toISOString()
    };
  devDebug('Duplicated process:', newProcess);
    notification.success('Process duplicated! (In production, would open new editor with copy)');
  };

  const totalDuration = process.steps.reduce((sum: number, step: BPStep) => sum + step.durationHours, 0);
  const totalEscalation = process.steps.reduce((sum: number, step: BPStep) => sum + (step.escalationThresholdHours || step.durationHours), 0);
  const hasValidationSteps = process.steps.some((s: BPStep) => s.stepType === 'validate');
  const hasApprovalSteps = process.steps.some((s: BPStep) => s.stepType === 'approve');

  const _getStepTypeIcon = (type: string) => {
    const stepType = STEP_TYPES.find(t => t.type === type);
    return stepType?.icon || FileText;
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <div className="bg-gradient-to-r from-blue-600 via-blue-700 to-purple-700 rounded-lg shadow-xl p-8 text-white">
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <Zap className="animate-pulse" size={28} />
                <h1 className="text-3xl font-bold">Business Process Builder</h1>
              </div>
              <p className="text-blue-100 text-lg">
                Create production-grade automated workflows with validation, approvals, and integrations
              </p>
            </div>
            <div className="flex gap-2 flex-wrap justify-end">
              <button
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="px-4 py-2 bg-white/20 text-white rounded-lg font-semibold hover:bg-white/30 transition-colors flex items-center gap-2 border border-white/30"
              >
                {showAdvanced ? <EyeOff size={18} /> : <Eye size={18} />}
                {showAdvanced ? 'Hide' : 'Show'} Advanced
              </button>
              <button
                onClick={() => setShowPreview(!showPreview)}
                className="px-4 py-2 bg-white/20 text-white rounded-lg font-semibold hover:bg-white/30 transition-colors flex items-center gap-2 border border-white/30"
              >
                <FileText size={18} />
                {showPreview ? 'Hide' : 'Show'} JSON
              </button>
              <button
                onClick={exportBP}
                className="px-4 py-2 bg-white/20 text-white rounded-lg font-semibold hover:bg-white/30 transition-colors flex items-center gap-2 border border-white/30"
              >
                <Download size={18} />
                Export
              </button>
              <button
                onClick={duplicateBP}
                className="px-4 py-2 bg-white/20 text-white rounded-lg font-semibold hover:bg-white/30 transition-colors flex items-center gap-2 border border-white/30"
              >
                <Copy size={18} />
                Duplicate
              </button>
            </div>
          </div>
        </div>

        {/* Save Status */}
        {saveStatus === 'success' && (
          <div className="bg-green-50 border-l-4 border-green-500 p-4 rounded flex items-center gap-3">
            <CheckCircle className="text-green-600" size={24} />
            <div>
              <h3 className="font-semibold text-green-900">Process Saved Successfully</h3>
              <p className="text-sm text-green-800">Your business process has been saved and is ready to deploy.</p>
            </div>
          </div>
        )}

        {/* Process Info */}
        <div className="bg-white rounded-lg shadow-md p-6 border border-gray-200">
          <h2 className="text-xl font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Settings size={20} />
            Process Configuration
          </h2>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">
                Process Name *
              </label>
              <input
                type="text"
                value={process.processName}
                onChange={(e) => setProcess({ ...process, processName: e.target.value })}
                title="Business process name"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="e.g., Hire Employee Process"
              />
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">
                Target Entity *
              </label>
              <select
                value={process.entity}
                onChange={(e) => setProcess({ ...process, entity: e.target.value })}
                title="Target entity type"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                {AVAILABLE_ENTITIES.map(entity => (
                  <option key={entity} value={entity}>{entity}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">
                Status
              </label>
              <label className="flex items-center gap-3 p-3 border border-gray-300 rounded-lg cursor-pointer hover:bg-gray-50 bg-white">
                <input
                  type="checkbox"
                  checked={process.isActive}
                  title="Toggle process active status"
                  onChange={(e) => setProcess({ ...process, isActive: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded"
                />
                <span className="text-sm font-medium text-gray-700">
                  {process.isActive ? '✅ Active' : '⏸️ Inactive'}
                </span>
              </label>
            </div>
          </div>
          <div className="mt-4">
            <label className="block text-sm font-semibold text-gray-700 mb-2">
              Description
            </label>
            <textarea
              value={process.description}
              onChange={(e) => setProcess({ ...process, description: e.target.value })}
              rows={3}
              title="Process description"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Describe what this business process does, when it's triggered, and what outcomes are expected..."
            />
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-4 gap-4">
          <div className="bg-white rounded-lg shadow p-4 border-l-4 border-blue-600">
            <div className="text-3xl font-bold text-blue-600">{process.steps.length}</div>
            <div className="text-sm text-gray-600 mt-1">Total Steps</div>
            <div className="text-xs text-gray-500 mt-2">Workflow stages</div>
          </div>
          <div className="bg-white rounded-lg shadow p-4 border-l-4 border-purple-600">
            <div className="text-3xl font-bold text-purple-600">{totalDuration}h</div>
            <div className="text-sm text-gray-600 mt-1">Total Duration</div>
            <div className="text-xs text-gray-500 mt-2">End-to-end time</div>
          </div>
          <div className="bg-white rounded-lg shadow p-4 border-l-4 border-orange-600">
            <div className="text-3xl font-bold text-orange-600">{totalEscalation}h</div>
            <div className="text-sm text-gray-600 mt-1">Escalation Time</div>
            <div className="text-xs text-gray-500 mt-2">Before auto-escalation</div>
          </div>
          <div className="bg-white rounded-lg shadow p-4 border-l-4 border-green-600">
            <div className="text-3xl font-bold text-green-600">
              {hasValidationSteps ? '✓' : '○'} {hasApprovalSteps ? '✓' : '○'}
            </div>
            <div className="text-sm text-gray-600 mt-1">Validation & Approval</div>
            <div className="text-xs text-gray-500 mt-2">Business controls</div>
          </div>
        </div>

        {/* Add Step Palette */}
        <div className="bg-white rounded-lg shadow-md p-6 border border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Plus size={20} className="text-blue-600" />
            Add Workflow Step
          </h3>
          <div className="grid grid-cols-3 gap-3">
            {STEP_TYPES.map((stepType) => {
              const Icon = stepType.icon;
              return (
                <button
                  key={stepType.type}
                  onClick={() => addStep(stepType.type)}
                  className="p-4 border-2 border-gray-300 rounded-lg hover:border-blue-400 hover:bg-blue-50 transition-all text-left group shadow-sm hover:shadow-md"
                >
                  <div className="flex items-center gap-3 mb-2">
                    <div className="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center group-hover:bg-blue-100 transition-colors">
                      <Icon className="text-gray-600 group-hover:text-blue-600" size={20} />
                    </div>
                    <span className="font-semibold text-gray-900 flex-1">{stepType.label}</span>
                  </div>
                  <p className="text-xs text-gray-600">{stepType.description}</p>
                </button>
              );
            })}
          </div>
        </div>

        {/* Steps List */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
            <TrendingUp size={20} className="text-purple-600" />
            Process Workflow ({process.steps.length} steps)
          </h3>

          {process.steps.length === 0 ? (
            <div className="bg-white rounded-lg shadow p-12 text-center border border-gray-200 border-dashed">
              <div className="text-gray-300 mb-4">
                <Plus size={64} className="mx-auto opacity-30" />
              </div>
              <p className="text-gray-600 text-lg font-semibold mb-2">No workflow steps yet</p>
              <p className="text-gray-500 text-sm">
                Select a step type above to start building your business process. You can add multiple steps, define dependencies, and configure validation rules.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {process.steps.map((step: BPStep, index: number) => (
                <div key={step.id}>
                  {index > 0 && (
                    <div className="flex justify-center py-2">
                      <div className="w-1 h-4 bg-gradient-to-b from-gray-300 to-gray-200"></div>
                    </div>
                  )}
                  <StepConfigurator
                    step={step}
                    onUpdate={(updated) => updateStep(step.id, updated)}
                    onDelete={() => deleteStep(step.id)}
                    onMoveUp={() => moveStep(step.id, 'up')}
                    onMoveDown={() => moveStep(step.id, 'down')}
                    availableRules={availableRules}
                    canMoveUp={index > 0}
                    canMoveDown={index < process.steps.length - 1}
                  />
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Advanced Settings */}
        {showAdvanced && (
          <div className="bg-white rounded-lg shadow-md p-6 border border-gray-200 border-dashed">
            <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <Settings size={20} className="text-indigo-600" />
              Advanced Settings
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Process Version
                </label>
                <input
                  type="number"
                  value={process.version}
                  onChange={(e) => setProcess({ ...process, version: parseInt(e.target.value) || 1 })}
                  title="Process version"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  min="1"
                />
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Tags (comma-separated)
                </label>
                <input
                  type="text"
                  value={process.tags?.join(', ') || ''}
                  onChange={(e) => setProcess({ ...process, tags: e.target.value.split(',').map((t: string) => t.trim()) })}
                  title="Process tags (comma-separated)"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., hr, onboarding, critical"
                />
              </div>
            </div>
          </div>
        )}

        {/* JSON Preview */}
        {showPreview && (
          <div className="bg-white rounded-lg shadow-md p-6 border border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <FileText size={20} />
              JSON Configuration (Read-only)
            </h3>
            <pre className="bg-gray-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm font-mono max-h-96 overflow-y-auto">
              {JSON.stringify(process, null, 2)}
            </pre>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex gap-4 sticky bottom-6">
          <button
            onClick={simulateBP}
            disabled={process.steps.length === 0}
            className="flex-1 px-6 py-3 bg-green-600 text-white rounded-lg font-semibold hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2 shadow-lg"
          >
            <Play size={20} />
            Simulate Workflow
          </button>
          <button
            onClick={saveBP}
            disabled={isSaving || process.steps.length === 0}
            className="flex-1 px-6 py-3 bg-blue-600 text-white rounded-lg font-semibold hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2 shadow-lg"
          >
            {isSaving ? (
              <>
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
                Saving...
              </>
            ) : (
              <>
                <Save size={20} />
                Save Process
              </>
            )}
          </button>
        </div>

        {/* Workday Feature Comparison */}
        <div className="bg-gradient-to-r from-blue-50 to-purple-50 rounded-lg border-2 border-blue-300 shadow-md p-6 mt-8">
          <h3 className="font-semibold text-gray-900 mb-4 text-lg flex items-center gap-2">
            <CheckCircle className="text-green-600" size={24} />
            Enterprise-Grade Features
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[
              { icon: '🔄', label: 'Drag & Drop', desc: 'Reorder steps easily' },
              { icon: '✓', label: 'Multi-Rule Validation', desc: 'Complex rule chains' },
              { icon: '👤', label: 'Role-Based Approvals', desc: 'Dynamic assignments' },
              { icon: '⏰', label: 'Escalation SLAs', desc: 'Auto-escalate timeout' },
              { icon: '🔀', label: 'Conditional Branching', desc: 'Smart routing' },
              { icon: '📧', label: 'Notifications', desc: 'Email/SMS alerts' },
              { icon: '🔌', label: 'API Integration', desc: 'Webhook support' },
              { icon: '⚡', label: 'Temporal Workflows', desc: 'Production orchestration' }
            ].map((feature, idx) => (
              <div key={idx} className="flex items-start gap-3 bg-white p-3 rounded-lg">
                <div className="text-2xl flex-shrink-0">{feature.icon}</div>
                <div>
                  <div className="font-semibold text-sm text-gray-900">{feature.label}</div>
                  <div className="text-xs text-gray-600">{feature.desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default BusinessProcessBuilder;
