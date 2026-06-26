import React, { useState, useEffect, useCallback as _useCallback, useMemo as _useMemo, useRef as _useRef } from 'react';
import {
  Plus, Trash2, Clock, User, CheckCircle, AlertTriangle, FileText, Send, GitBranch,
  Settings, Play, Save, ChevronUp, ChevronDown, Eye, EyeOff, Copy, Download, Upload,
  AlertCircle, TrendingUp, Zap, ArrowRight, X, Maximize2, Minimize2, Loader, Check,
  Layers, Grid3X3, List, Workflow, Command, Lock, Unlock, Share2, Archive, Terminal,
  Code, Lightbulb, Sliders, Database, BarChart3, Activity, BookOpen, Info, HelpCircle,
  Calendar, Flag, Tag as TagIcon, Users, Briefcase, Shield, Wand2, Package
} from 'lucide-react';
import { useNotification } from '../../hooks/useNotification';
import { useCreateBusinessProcess, useUpdateBusinessProcess, useDeleteBusinessProcess, 
         usePublishBusinessProcess, useSimulateBusinessProcess, useDuplicateBusinessProcess,
         useFetchBusinessProcess as _useFetchBusinessProcess, BPStep, BusinessProcess } from './useBPBuilderAPI';
import { useTenant } from '../../contexts/TenantContext';
import { 
  AdvancedConditionBuilder, 
  ApprovalChainConfig, 
  StepDependenciesManager, 
  ParallelExecutionConfig 
} from './AdvancedStepComponents';
import { NaturalLanguageBuilder } from './NaturalLanguageBuilder';
import { ProcessAnalyticsDashboard } from './ProcessAnalyticsDashboard';
import { ProcessMonitorDashboard } from './ProcessMonitorDashboard';
import { ProcessOptimizationDashboard } from './ProcessOptimizationDashboard';
import { IntegrationMarketplaceBrowser } from './IntegrationMarketplaceBrowser';
import { ProcessTemplatesLibrary } from './ProcessTemplatesLibrary';

// ============================================================================
// TYPES & CONSTANTS
// ============================================================================

type ViewMode = 'canvas' | 'list' | 'grid' | 'json' | 'timeline' | 'analytics' | 'monitor' | 'optimize' | 'integrations' | 'templates';
type StepType = BPStep['stepType'];

const STEP_TYPES = [
  {
    type: 'data_entry' as StepType,
    label: 'Data Entry',
    icon: FileText,
    color: 'from-blue-500 to-blue-600',
    bgColor: 'bg-blue-50',
    borderColor: 'border-blue-300',
    badgeColor: 'bg-blue-100 text-blue-700',
    description: 'Collect information from user'
  },
  {
    type: 'validate' as StepType,
    label: 'Validation',
    icon: CheckCircle,
    color: 'from-green-500 to-green-600',
    bgColor: 'bg-green-50',
    borderColor: 'border-green-300',
    badgeColor: 'bg-green-100 text-green-700',
    description: 'Run validation rules'
  },
  {
    type: 'approve' as StepType,
    label: 'Approval',
    icon: User,
    color: 'from-purple-500 to-purple-600',
    bgColor: 'bg-purple-50',
    borderColor: 'border-purple-300',
    badgeColor: 'bg-purple-100 text-purple-700',
    description: 'Require approval from user/role'
  },
  {
    type: 'notify' as StepType,
    label: 'Notification',
    icon: Send,
    color: 'from-orange-500 to-orange-600',
    bgColor: 'bg-orange-50',
    borderColor: 'border-orange-300',
    badgeColor: 'bg-orange-100 text-orange-700',
    description: 'Send email/SMS notification'
  },
  {
    type: 'integrate' as StepType,
    label: 'Integration',
    icon: Settings,
    color: 'from-indigo-500 to-indigo-600',
    bgColor: 'bg-indigo-50',
    borderColor: 'border-indigo-300',
    badgeColor: 'bg-indigo-100 text-indigo-700',
    description: 'Call external API/system'
  },
  {
    type: 'condition' as StepType,
    label: 'Conditional Branch',
    icon: GitBranch,
    color: 'from-yellow-500 to-yellow-600',
    bgColor: 'bg-yellow-50',
    borderColor: 'border-yellow-300',
    badgeColor: 'bg-yellow-100 text-yellow-700',
    description: 'Branch based on conditions'
  }
];

const AVAILABLE_ROLES = ['Manager', 'HR Admin', 'Department Head', 'Director', 'Executive', 'Analyst'];
const AVAILABLE_ENTITIES = ['Employee', 'Order', 'Invoice', 'Project', 'Contract', 'Asset'];

// ============================================================================
// COMPONENTS
// ============================================================================

// Toast Notification Component
const Toast: React.FC<{ message: string; type: 'success' | 'error' | 'info'; onClose: () => void }> = 
  ({ message, type, onClose }) => {
    useEffect(() => {
      const timer = setTimeout(onClose, 3000);
      return () => clearTimeout(timer);
    }, [onClose]);

    const bgColor = type === 'success' ? 'bg-green-500' : type === 'error' ? 'bg-red-500' : 'bg-blue-500';
    
    return (
      <div className={`${bgColor} text-white px-6 py-3 rounded-lg shadow-lg animate-fade-in-out flex items-center gap-3`}>
        {type === 'success' && <Check size={20} />}
        {type === 'error' && <AlertCircle size={20} />}
        {type === 'info' && <Info size={20} />}
        <span>{message}</span>
      </div>
    );
  };

// Some icon imports are intentionally kept for future/conditional rendering or
// design variants. Reference them here to avoid noisy eslint unused-var warnings
// (the runtime bundle is tree-shaken as usual).
void ChevronUp;
void ChevronDown;
void Eye;
void EyeOff;
void Copy;
void TrendingUp;
void Zap;
void List;
void Command;
void Lock;
void Unlock;
void Share2;
void Archive;
void Terminal;
void Lightbulb;
void Sliders;
void Database;
void BookOpen;
void HelpCircle;
void Calendar;
void Flag;
void TagIcon;
void Users;
void Briefcase;
void Shield;

// Step Editor Modal
interface StepEditorProps {
  step?: BPStep;
  onSave: (step: BPStep) => void;
  onCancel: () => void;
  stepOrder: number;
}

const StepEditor: React.FC<StepEditorProps> = ({ step, onSave, onCancel, stepOrder }) => {
  const notification = useNotification();
  const [formData, setFormData] = useState<BPStep>(
    step || {
      id: `step-${Date.now()}`,
      stepOrder,
      stepType: 'data_entry',
      stepName: '',
      durationHours: 1,
      description: '',
      executionMode: 'sequential',
      dependsOn: [],
    }
  );

  const selectedStepType = STEP_TYPES.find(t => t.type === formData.stepType);

  // Available fields for condition builder
  const availableFields = ['amount', 'department', 'status', 'priority', 'user_role', 'entity_type'];

  const handleSave = () => {
    if (!formData.stepName.trim()) {
      notification.error('Step name is required');
      return;
    }
    onSave(formData);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className={`bg-gradient-to-r ${selectedStepType?.color} text-white px-6 py-4 flex items-center justify-between`}>
          <div className="flex items-center gap-3">
            {selectedStepType && <selectedStepType.icon size={24} />}
            <div>
              <h3 className="text-xl font-bold">
                {step ? 'Edit Step' : 'New Step'} - {selectedStepType?.label}
              </h3>
              <p className="text-sm opacity-90">{selectedStepType?.description}</p>
            </div>
          </div>
          <button 
            onClick={onCancel} 
            className="hover:bg-white hover:bg-opacity-20 p-2 rounded"
            title="Close step editor"
            aria-label="Close step editor"
          >
            <X size={24} />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Step Type Selection */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-3">Step Type</label>
            <div className="grid grid-cols-3 gap-2">
              {STEP_TYPES.map(st => (
                <button
                  key={st.type}
                  onClick={() => setFormData({ ...formData, stepType: st.type })}
                  className={`p-3 rounded-lg border-2 transition-all text-center ${
                    formData.stepType === st.type
                      ? `border-${st.color.split('-')[1]}-600 ${st.bgColor}`
                      : 'border-gray-300 hover:border-gray-400'
                  }`}
                >
                  <st.icon size={20} className="mx-auto mb-1" />
                  <div className="text-xs font-semibold">{st.label}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Step Name */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2">Step Name *</label>
            <input
              type="text"
              value={formData.stepName}
              onChange={(e) => setFormData({ ...formData, stepName: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="e.g., Verify Employee Information"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-2">Description</label>
            <textarea
              value={formData.description || ''}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              rows={3}
              placeholder="Add details about this step..."
            />
          </div>

          {/* Duration & Escalation */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
                <Clock size={16} /> Duration (hours)
              </label>
              <input
                type="number"
                min="0.5"
                step="0.5"
                value={formData.durationHours}
                onChange={(e) => setFormData({ ...formData, durationHours: parseFloat(e.target.value) })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                title="Step duration in hours"
                placeholder="1.0"
              />
            </div>
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
                <AlertTriangle size={16} /> Escalation Threshold (hours)
              </label>
              <input
                type="number"
                min="0.5"
                step="0.5"
                value={formData.escalationThresholdHours || 0}
                onChange={(e) => setFormData({ ...formData, escalationThresholdHours: parseFloat(e.target.value) })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                title="Escalation threshold in hours"
                placeholder="2.0"
              />
            </div>
          </div>

          {/* Role Assignment (for approve/notify steps) */}
          {(formData.stepType === 'approve' || formData.stepType === 'notify') && (
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
                <User size={16} /> Assign Role
              </label>
              <select
                value={formData.assigneeRole || ''}
                onChange={(e) => setFormData({ ...formData, assigneeRole: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                title="Select assignee role"
                aria-label="Select assignee role"
              >
                <option value="">Select a role...</option>
                {AVAILABLE_ROLES.map(role => (
                  <option key={role} value={role}>{role}</option>
                ))}
              </select>
            </div>
          )}

          {/* Validation Rules (for validate steps) */}
          {formData.stepType === 'validate' && (
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
                <CheckCircle size={16} /> Validation Rules
              </label>
              <div className="space-y-2">
                <input
                  type="text"
                  placeholder="e.g., Email format, Non-negative salary"
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  title="Add validation rule"
                  onKeyPress={(e) => {
                    if (e.key === 'Enter' && (e.target as HTMLInputElement).value) {
                      const rule = (e.target as HTMLInputElement).value;
                      setFormData({
                        ...formData,
                        validationRules: [...(formData.validationRules || []), rule]
                      });
                      (e.target as HTMLInputElement).value = '';
                    }
                  }}
                />
                <div className="flex flex-wrap gap-2">
                  {(formData.validationRules || []).map((rule, idx) => (
                    <div key={idx} className="bg-green-100 text-green-700 px-3 py-1 rounded-full text-sm flex items-center gap-2">
                      {rule}
                      <button
                        onClick={() => setFormData({
                          ...formData,
                          validationRules: formData.validationRules?.filter((_, i) => i !== idx)
                        })}
                        className="hover:text-red-600"
                        title="Remove validation rule"
                        aria-label="Remove validation rule"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* ========== ADVANCED FEATURES ========== */}
          
          {/* Parallel Execution */}
          <ParallelExecutionConfig
            executionMode={formData.executionMode}
            parallelGroup={formData.parallelGroup}
            waitForAll={formData.waitForAll}
            onExecutionModeChange={(mode) => setFormData({ ...formData, executionMode: mode })}
            onParallelGroupChange={(group) => setFormData({ ...formData, parallelGroup: group })}
            onWaitForAllChange={(wait) => setFormData({ ...formData, waitForAll: wait })}
          />

          {/* Advanced Conditional Logic (for condition step types) */}
          {formData.stepType === 'condition' && (
            <AdvancedConditionBuilder
              condition={formData.conditionLogic}
              onChange={(condition) => setFormData({ ...formData, conditionLogic: condition })}
              availableFields={availableFields}
              availableSteps={[]} // Will be populated from parent component
            />
          )}

          {/* Approval Chain (for approve step types) */}
          {formData.stepType === 'approve' && (
            <ApprovalChainConfig
              approvalChain={formData.approvalChain}
              onChange={(chain) => setFormData({ ...formData, approvalChain: chain })}
              availableRoles={AVAILABLE_ROLES}
            />
          )}

          {/* Step Dependencies */}
          <StepDependenciesManager
            dependsOn={formData.dependsOn || []}
            skipCondition={formData.skipCondition}
            availableSteps={[]} // Will be populated from parent component
            availableFields={availableFields}
            onDependsOnChange={(deps) => setFormData({ ...formData, dependsOn: deps })}
            onSkipConditionChange={(condition) => setFormData({ ...formData, skipCondition: condition })}
          />
        </div>

        {/* Footer */}
        <div className="bg-gray-50 px-6 py-4 flex justify-end gap-3 border-t">
          <button
            onClick={onCancel}
            className="px-6 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 font-semibold"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold flex items-center gap-2"
          >
            <Check size={18} />
            Save Step
          </button>
        </div>
      </div>
    </div>
  );
};

// Canvas View - Visual Workflow Designer
const CanvasView: React.FC<{
  steps: BPStep[];
  onEditStep: (step: BPStep) => void;
  onDeleteStep: (stepId: string) => void;
  onAddStep: () => void;
  onReorderSteps: (steps: BPStep[]) => void;
}> = ({ steps, onEditStep, onDeleteStep, onAddStep, onReorderSteps }) => {
  const [draggedStep, setDraggedStep] = useState<string | null>(null);

  const moveStep = (fromIdx: number, toIdx: number) => {
    const newSteps = [...steps];
    const [movedStep] = newSteps.splice(fromIdx, 1);
    newSteps.splice(toIdx, 0, movedStep);
    newSteps.forEach((step, idx) => step.stepOrder = idx + 1);
    onReorderSteps(newSteps);
  };

  return (
    <div className="space-y-4">
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <Workflow className="text-blue-600 flex-shrink-0 mt-0.5" size={20} />
          <div>
            <h4 className="font-semibold text-gray-900">Visual Workflow</h4>
            <p className="text-sm text-gray-600">Drag steps to reorder. Click to edit. Connections show workflow flow.</p>
          </div>
        </div>
      </div>

      <div className="space-y-3">
        {steps.map((step, idx) => {
          const stepType = STEP_TYPES.find(t => t.type === step.stepType)!;
          const IconComponent = stepType.icon;

          return (
            <React.Fragment key={step.id}>
              <div
                draggable
                onDragStart={() => setDraggedStep(step.id)}
                onDragOver={(e) => e.preventDefault()}
                onDrop={() => {
                  const draggedIdx = steps.findIndex(s => s.id === draggedStep);
                  if (draggedIdx !== undefined) {
                    moveStep(draggedIdx, idx);
                  }
                }}
                onDragEnd={() => setDraggedStep(null)}
                className={`${stepType.bgColor} border-2 ${stepType.borderColor} rounded-lg p-4 cursor-move transition-all ${
                  draggedStep === step.id ? 'opacity-50' : ''
                } hover:shadow-md`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-4 flex-1">
                    <div className={`p-3 bg-gradient-to-br ${stepType.color} text-white rounded-lg flex-shrink-0`}>
                      <IconComponent size={20} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <h4 className="font-semibold text-gray-900">{step.stepName}</h4>
                        <span className={`${stepType.badgeColor} px-2 py-1 rounded text-xs font-semibold whitespace-nowrap`}>
                          Step {step.stepOrder}
                        </span>
                      </div>
                      <p className="text-sm text-gray-600 mt-1">{step.description}</p>
                      <div className="flex items-center gap-4 mt-3 text-sm text-gray-600">
                        {step.durationHours && (
                          <div className="flex items-center gap-1">
                            <Clock size={14} />
                            {step.durationHours}h
                          </div>
                        )}
                        {step.escalationThresholdHours && (
                          <div className="flex items-center gap-1 text-red-600">
                            <AlertTriangle size={14} />
                            Escalate: {step.escalationThresholdHours}h
                          </div>
                        )}
                        {step.assigneeRole && (
                          <div className="flex items-center gap-1">
                            <User size={14} />
                            {step.assigneeRole}
                          </div>
                        )}
                        {(step.validationRules?.length ?? 0) > 0 && (
                          <div className="flex items-center gap-1">
                            <CheckCircle size={14} />
                            {step.validationRules?.length} rules
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <button
                      onClick={() => onEditStep(step)}
                      className="p-2 hover:bg-white hover:bg-opacity-50 rounded-lg transition-all"
                      title="Edit step"
                      aria-label="Edit step"
                    >
                      <Settings size={18} className="text-gray-600" />
                    </button>
                    <button
                      onClick={() => onDeleteStep(step.id)}
                      className="p-2 hover:bg-red-100 rounded-lg transition-all"
                      title="Delete step"
                      aria-label="Delete step"
                    >
                      <Trash2 size={18} className="text-red-600" />
                    </button>
                  </div>
                </div>
              </div>

              {idx < steps.length - 1 && (
                <div className="flex justify-center">
                  <ArrowRight className="text-gray-400 rotate-90" size={20} />
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>

      <button
        onClick={onAddStep}
        className="w-full p-4 border-2 border-dashed border-blue-300 rounded-lg hover:bg-blue-50 transition-all flex items-center justify-center gap-2 text-blue-600 font-semibold"
      >
        <Plus size={20} />
        Add Step
      </button>
    </div>
  );
};

// Main BP Builder Component
export const BusinessProcessBuilderEnhanced: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [viewMode, setViewMode] = useState<ViewMode>('canvas');
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [editingStepIndex, setEditingStepIndex] = useState<number | null>(null);
  const [isMaximized, setIsMaximized] = useState(false);
  const [showNLBuilder, setShowNLBuilder] = useState(false);
  const scopeMissing = !tenant?.id || !datasource?.id;

  // API Hooks
  const createBPMutation = useCreateBusinessProcess();
  const updateBPMutation = useUpdateBusinessProcess();
  const _deleteBPMutation = useDeleteBusinessProcess();
  const publishBPMutation = usePublishBusinessProcess();
  const simulateBPMutation = useSimulateBusinessProcess();
  const _duplicateBPMutation = useDuplicateBusinessProcess();

  // Process State
  const [currentProcess, setCurrentProcess] = useState<BusinessProcess>({
    id: '',
    processName: '',
    entity: AVAILABLE_ENTITIES[0],
    description: '',
    steps: [],
    isActive: false,
    createdBy: 'bp_builder_agent',
    createdAt: new Date().toISOString(),
    version: 1,
    tags: [],
  });

  const handleAddStep = () => {
    setEditingStepIndex(currentProcess.steps.length);
  };

  const handleEditStep = (idx: number) => {
    setEditingStepIndex(idx);
  };

  const handleSaveStep = (step: BPStep) => {
    const updatedSteps = [...currentProcess.steps];
    if (editingStepIndex !== null) {
      updatedSteps[editingStepIndex] = { ...step, stepOrder: editingStepIndex + 1 };
    }
    setCurrentProcess({ ...currentProcess, steps: updatedSteps });
    setEditingStepIndex(null);
    setToast({ message: 'Step saved successfully', type: 'success' });
  };

  const handleDeleteStep = (stepId: string) => {
    const updatedSteps = currentProcess.steps.filter(s => s.id !== stepId);
    updatedSteps.forEach((step, idx) => step.stepOrder = idx + 1);
    setCurrentProcess({ ...currentProcess, steps: updatedSteps });
    setToast({ message: 'Step deleted', type: 'info' });
  };

  const handleReorderSteps = (reorderedSteps: BPStep[]) => {
    setCurrentProcess({ ...currentProcess, steps: reorderedSteps });
  };

  const handleNLGenerated = (process: BusinessProcess) => {
    setCurrentProcess({
      ...process,
      id: '',
      createdBy: 'nl_builder',
      createdAt: new Date().toISOString(),
      version: 1,
    });
    setShowNLBuilder(false);
    setToast({ message: 'Process generated! Review and save when ready.', type: 'success' });
  };

  const handleSaveProcess = async () => {
    if (scopeMissing) {
      setToast({ message: 'Select a tenant and datasource before saving', type: 'error' });
      return;
    }
    if (!currentProcess.processName.trim()) {
      setToast({ message: 'Process name is required', type: 'error' });
      return;
    }
    if (currentProcess.steps.length === 0) {
      setToast({ message: 'Add at least one step', type: 'error' });
      return;
    }

    setIsLoading(true);
    try {
      const isNewProcess = !currentProcess.id;
      if (isNewProcess) {
        const { id: _id, createdAt: _createdAt, updatedAt: _updatedAt, version: _version, ...createPayload } = currentProcess;
        const result = await createBPMutation.mutateAsync(
          createPayload as Omit<BusinessProcess, 'id' | 'createdAt' | 'updatedAt' | 'version'>
        );
        setCurrentProcess(result!);
      } else {
        const result = await updateBPMutation.mutateAsync(currentProcess);
        setCurrentProcess(result!);
      }
      setToast({ message: 'Process saved successfully', type: 'success' });
    } catch (error) {
      setToast({ message: error instanceof Error ? error.message : 'Failed to save', type: 'error' });
    } finally {
      setIsLoading(false);
    }
  };

  const handlePublish = async () => {
    if (scopeMissing) {
      setToast({ message: 'Select a tenant and datasource before publishing', type: 'error' });
      return;
    }
    if (!currentProcess.id) {
      setToast({ message: 'Save process first before publishing', type: 'error' });
      return;
    }

    setIsLoading(true);
    try {
      const result = await publishBPMutation.mutateAsync(currentProcess.id);
      setCurrentProcess(result!);
      setToast({ message: 'Process published successfully', type: 'success' });
    } catch (error) {
      setToast({ message: error instanceof Error ? error.message : 'Failed to publish', type: 'error' });
    } finally {
      setIsLoading(false);
    }
  };

  const handleSimulate = async () => {
    if (scopeMissing) {
      setToast({ message: 'Select a tenant and datasource before running simulations', type: 'error' });
      return;
    }
    if (!currentProcess.id) {
      setToast({ message: 'Save the process before running a simulation', type: 'error' });
      return;
    }
    setIsLoading(true);
    try {
      const result = await simulateBPMutation.mutateAsync({
        processId: currentProcess.id,
        testData: {},
      });
      setToast({ message: `Simulation completed: ${result.message || 'Success'}`, type: 'success' });
    } catch (error) {
      setToast({ message: error instanceof Error ? error.message : 'Simulation failed', type: 'error' });
    } finally {
      setIsLoading(false);
    }
  };

  const handleExport = () => {
    const dataStr = JSON.stringify(currentProcess, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${currentProcess.processName}-v${currentProcess.version}.json`;
    link.click();
    setToast({ message: 'Process exported', type: 'success' });
  };

  const containerClass = isMaximized ? 'fixed inset-0 z-50 bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50' : 'w-full';

  const headerLarge = "bg-gradient-to-r from-indigo-600 via-blue-600 to-purple-600 text-white px-8 py-6 shadow-2xl border-b border-white/10";
  const headerCompact = "bg-transparent text-gray-900 px-4 py-3 border-b border-gray-100";

  if (scopeMissing) {
    return (
      <div className="p-10">
        <div className="max-w-2xl mx-auto bg-white border border-amber-200 rounded-2xl shadow-sm p-10 text-center">
          <div className="text-4xl mb-4">🔐</div>
          <h2 className="text-2xl font-semibold mb-2">Select a tenant + datasource</h2>
          <p className="text-gray-600 mb-6">
            The Business Process Builder calls tenant-scoped APIs. Use the Fabric Builder scope picker in the shell
            to choose a tenant, product, and datasource, or seed&nbsp;`localStorage` using the values from
            <code className="mx-1 bg-gray-100 px-1 rounded">TenantContext</code> before returning.
          </p>
          <p className="text-sm text-gray-500">
            Tip: in headless sessions you can run <code className="bg-gray-100 px-1 rounded">localStorage.setItem('selected_tenant', ...)</code>
            then reload to unlock the designer.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={`flex flex-col ${containerClass}`}>
      {/* Header - large when maximized, compact when embedded inside app shell */}
      <div className={isMaximized ? headerLarge : headerCompact}>
        <div className={` ${isMaximized ? 'max-w-screen-2xl mx-auto flex items-center justify-between' : 'flex items-center justify-between w-full'}`}>
          <div className={`flex items-center gap-4 ${isMaximized ? '' : 'w-full'}`}>
            <div className={`${isMaximized ? 'bg-white/20 backdrop-blur-sm p-3 rounded-xl' : 'p-2 rounded-md'}`}>
              <Workflow size={isMaximized ? 32 : 20} className={isMaximized ? 'drop-shadow-lg' : ''} />
            </div>
            <div className={isMaximized ? '' : 'flex-1'}>
              <h1 className={`${isMaximized ? 'text-3xl font-bold tracking-tight' : 'text-lg font-semibold'}`}>Business Process Builder</h1>
              {isMaximized && <p className="text-sm text-blue-100 mt-1">Visual workflow designer for enterprise automation</p>}
            </div>
          </div>
          <div className="flex items-center gap-3">
            {isMaximized ? (
              <>
                <div className="bg-white/10 backdrop-blur-sm px-4 py-2 rounded-lg border border-white/20">
                  <span className="text-xs text-blue-100">Process Steps: </span>
                  <span className="text-lg font-bold">{currentProcess.steps.length}</span>
                </div>
                <button
                  onClick={() => setIsMaximized(!isMaximized)}
                  className="hover:bg-white/20 p-2 rounded-lg transition-all backdrop-blur-sm"
                  title={isMaximized ? 'Restore window' : 'Maximize window'}
                  aria-label={isMaximized ? 'Restore window' : 'Maximize window'}
                >
                  {isMaximized ? <Minimize2 size={22} /> : <Maximize2 size={22} />}
                </button>
              </>
            ) : (
              <button
                onClick={() => setIsMaximized(true)}
                className="px-3 py-1 border border-gray-200 rounded-md text-sm hover:bg-gray-50"
                title="Open in full screen"
                aria-label="Open in full screen"
              >
                Open
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex h-full overflow-hidden max-w-screen-2xl mx-auto w-full">
        {/* Left Panel - Config with Modern Card Design */}
        <div className="w-96 border-r border-gray-200 overflow-y-auto p-6 space-y-5 bg-white/60 backdrop-blur-sm">
          {/* Process Configuration Card */}
          <div className="bg-white rounded-xl shadow-lg border border-gray-200 p-6 space-y-5">
            <div className="flex items-center gap-3 mb-4">
              <div className="bg-gradient-to-br from-blue-500 to-indigo-600 p-2 rounded-lg">
                <Settings size={20} className="text-white" />
              </div>
              <h2 className="text-lg font-bold text-gray-900">Process Configuration</h2>
            </div>

            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">
                Process Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={currentProcess.processName}
                onChange={(e) => setCurrentProcess({ ...currentProcess, processName: e.target.value })}
                className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all"
                placeholder="e.g., Employee Onboarding"
              />
            </div>

            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">Entity Type</label>
              <select
                value={currentProcess.entity}
                onChange={(e) => setCurrentProcess({ ...currentProcess, entity: e.target.value })}
                className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all bg-white"
                title="Select entity type"
                aria-label="Select entity type"
              >
                {AVAILABLE_ENTITIES.map(entity => (
                  <option key={entity} value={entity}>{entity}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-2">Description</label>
              <textarea
                value={currentProcess.description}
                onChange={(e) => setCurrentProcess({ ...currentProcess, description: e.target.value })}
                className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all resize-none"
                rows={3}
                placeholder="Describe the process purpose and scope..."
              />
            </div>
          </div>

          {/* Stats Card */}
          <div className="bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl shadow-lg p-6 text-white">
            <h3 className="text-sm font-semibold mb-4 opacity-90">Process Metrics</h3>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Layers size={18} className="opacity-80" />
                  <span className="text-sm">Total Steps</span>
                </div>
                <span className="text-2xl font-bold">{currentProcess.steps.length}</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Clock size={18} className="opacity-80" />
                  <span className="text-sm">Duration</span>
                </div>
                <span className="text-2xl font-bold">{currentProcess.steps.reduce((sum, s) => sum + s.durationHours, 0)}h</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Activity size={18} className="opacity-80" />
                  <span className="text-sm">Status</span>
                </div>
                <span className={`px-3 py-1 rounded-full text-xs font-bold ${
                  currentProcess.isActive 
                    ? 'bg-green-400 text-green-900' 
                    : 'bg-white/20 text-white'
                }`}>
                  {currentProcess.isActive ? '● Published' : '○ Draft'}
                </span>
              </div>
            </div>
          </div>

          {/* AI Builder Button */}
          <button
            onClick={() => setShowNLBuilder(true)}
            className="w-full px-6 py-4 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-xl hover:from-purple-700 hover:to-indigo-700 font-semibold text-base flex items-center justify-center gap-3 transition-all shadow-lg hover:shadow-xl transform hover:scale-[1.02]"
          >
            <Wand2 size={20} />
            Create with AI
          </button>

          {/* Analytics Button */}
          <button
            onClick={() => setViewMode('analytics')}
            className="w-full px-6 py-4 bg-gradient-to-r from-blue-600 to-cyan-600 text-white rounded-xl hover:from-blue-700 hover:to-cyan-700 font-semibold text-base flex items-center justify-center gap-3 transition-all shadow-lg hover:shadow-xl transform hover:scale-[1.02]"
          >
            <BarChart3 size={20} />
            View Analytics
          </button>

          {/* Live Monitor Button */}
          <button
            onClick={() => setViewMode('monitor')}
            className="w-full px-6 py-4 bg-gradient-to-r from-green-600 to-emerald-600 text-white rounded-xl hover:from-green-700 hover:to-emerald-700 font-semibold text-base flex items-center justify-center gap-3 transition-all shadow-lg hover:shadow-xl transform hover:scale-[1.02]"
          >
            <Activity size={20} />
            Live Monitor
          </button>

          {/* AI Optimize Button */}
          <button
            onClick={() => setViewMode('optimize')}
            className="w-full px-6 py-4 bg-gradient-to-r from-purple-600 to-pink-600 text-white rounded-xl hover:from-purple-700 hover:to-pink-700 font-semibold text-base flex items-center justify-center gap-3 transition-all shadow-lg hover:shadow-xl transform hover:scale-[1.02]"
          >
            <Zap size={20} />
            AI Optimize
          </button>

          {/* Integrations Button */}
          <button
            onClick={() => setViewMode('integrations')}
            className="w-full px-6 py-4 bg-gradient-to-r from-green-600 to-teal-600 text-white rounded-xl hover:from-green-700 hover:to-teal-700 font-semibold text-base flex items-center justify-center gap-3 transition-all shadow-lg hover:shadow-xl transform hover:scale-[1.02]"
          >
            <Package size={20} />
            Integrations
          </button>

          {/* Templates Button */}
          <button
            onClick={() => setViewMode('templates')}
            className="w-full px-6 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 font-semibold text-base flex items-center justify-center gap-3 transition-all shadow-lg hover:shadow-xl transform hover:scale-[1.02]"
          >
            <Layers size={20} />
            Templates
          </button>

          {/* Action Buttons */}
          <div className="space-y-3 pt-2">
            <button
              onClick={handleSaveProcess}
              disabled={isLoading}
              className="w-full px-4 py-3.5 bg-gradient-to-r from-blue-600 to-blue-700 text-white rounded-xl hover:from-blue-700 hover:to-blue-800 font-semibold flex items-center justify-center gap-2 disabled:opacity-50 shadow-lg hover:shadow-xl transition-all transform hover:scale-[1.02]"
            >
              {isLoading ? <Loader size={20} className="animate-spin" /> : <Save size={20} />}
              Save Process
            </button>
            <button
              onClick={handlePublish}
              disabled={isLoading || currentProcess.steps.length === 0}
              className="w-full px-4 py-3.5 bg-gradient-to-r from-green-600 to-emerald-600 text-white rounded-xl hover:from-green-700 hover:to-emerald-700 font-semibold flex items-center justify-center gap-2 disabled:opacity-50 shadow-lg hover:shadow-xl transition-all transform hover:scale-[1.02]"
            >
              {isLoading ? <Loader size={20} className="animate-spin" /> : <Upload size={20} />}
              Publish
            </button>
            <button
              onClick={handleSimulate}
              disabled={isLoading || currentProcess.steps.length === 0}
              className="w-full px-4 py-3.5 bg-gradient-to-r from-purple-600 to-pink-600 text-white rounded-xl hover:from-purple-700 hover:to-pink-700 font-semibold flex items-center justify-center gap-2 disabled:opacity-50 shadow-lg hover:shadow-xl transition-all transform hover:scale-[1.02]"
            >
              {isLoading ? <Loader size={20} className="animate-spin" /> : <Play size={20} />}
              Simulate
            </button>
            <button
              onClick={handleExport}
              className="w-full px-4 py-3.5 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 font-semibold flex items-center justify-center gap-2 shadow-lg hover:shadow-xl transition-all transform hover:scale-[1.02]"
            >
              <Download size={20} />
              Export
            </button>
          </div>
        </div>

        {/* Right Panel - Canvas */}
        <div className="flex-1 overflow-y-auto p-8 bg-white/40">
          {/* View Mode Selector */}
          <div className="flex gap-3 mb-8">
            {[
              { mode: 'canvas' as ViewMode, icon: Grid3X3, label: 'Canvas' },
              { mode: 'timeline' as ViewMode, icon: Activity, label: 'Timeline' },
              { mode: 'json' as ViewMode, icon: Code, label: 'JSON' },
            ].map(({ mode, icon: Icon, label }) => (
              <button
                key={mode}
                onClick={() => setViewMode(mode)}
                className={`px-6 py-3 rounded-xl font-semibold flex items-center gap-2 transition-all shadow-sm ${
                  viewMode === mode
                    ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white shadow-lg scale-105'
                    : 'bg-white text-gray-700 hover:bg-gray-50 border-2 border-gray-200'
                }`}
              >
                <Icon size={20} />
                {label}
              </button>
            ))}
          </div>

          {/* View Content */}
          {viewMode === 'canvas' && (
            <CanvasView
              steps={currentProcess.steps}
              onEditStep={(step) => {
                const idx = currentProcess.steps.findIndex((s) => s.id === step.id);
                if (idx >= 0) {
                  handleEditStep(idx);
                }
              }}
              onDeleteStep={handleDeleteStep}
              onAddStep={handleAddStep}
              onReorderSteps={handleReorderSteps}
            />
          )}

          {viewMode === 'timeline' && (
            <div className="space-y-4">
              <div className="relative">
                {currentProcess.steps.map((step, idx) => {
                  const stepType = STEP_TYPES.find(t => t.type === step.stepType)!;
                  const accumulatedTime = currentProcess.steps.slice(0, idx).reduce((sum, s) => sum + s.durationHours, 0);

                  return (
                    <div key={step.id} className="flex gap-4 mb-6">
                      <div className="w-24 text-right">
                        <div className="text-sm font-bold text-gray-900">{accumulatedTime.toFixed(1)}h</div>
                        <div className="text-xs text-gray-500">+{step.durationHours}h</div>
                      </div>
                      <div className="relative flex-1">
                        <div className={`absolute -left-12 top-2 w-6 h-6 ${stepType.bgColor} border-2 ${stepType.borderColor} rounded-full`} />
                        {idx < currentProcess.steps.length - 1 && (
                          <div className="absolute -left-10 top-8 w-0.5 h-20 bg-gray-300" />
                        )}
                        <div className={`${stepType.bgColor} border-2 ${stepType.borderColor} rounded-lg p-4`}>
                          <h4 className="font-semibold text-gray-900">{step.stepName}</h4>
                          <p className="text-sm text-gray-600">{step.description}</p>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {viewMode === 'json' && (
            <div className="bg-gray-900 text-gray-100 rounded-lg p-4 font-mono text-sm overflow-x-auto">
              <pre>{JSON.stringify(currentProcess, null, 2)}</pre>
            </div>
          )}

          {viewMode === 'analytics' && tenant && datasource && (
            <ProcessAnalyticsDashboard
              tenant={tenant}
              datasource={datasource}
            />
          )}

          {viewMode === 'monitor' && tenant && datasource && (
            <ProcessMonitorDashboard
              tenant={tenant}
              datasource={datasource}
            />
          )}

          {viewMode === 'optimize' && tenant && datasource && (
            <ProcessOptimizationDashboard
              tenant={tenant}
              datasource={datasource}
            />
          )}

          {viewMode === 'integrations' && tenant && datasource && (
            <IntegrationMarketplaceBrowser
              tenant={tenant}
              datasource={datasource}
            />
          )}

          {viewMode === 'templates' && tenant && datasource && (
            <ProcessTemplatesLibrary
              tenant={tenant}
              datasource={datasource}
              onTemplateCloned={(processId) => {
                // Refresh or navigate to cloned process
                setViewMode('canvas');
                showNotification('Template cloned successfully!', 'success');
              }}
            />
          )}
        </div>
      </div>

      {/* Step Editor Modal */}
      {editingStepIndex !== null && (
        <StepEditor
          step={currentProcess.steps[editingStepIndex]}
          onSave={handleSaveStep}
          onCancel={() => setEditingStepIndex(null)}
          stepOrder={editingStepIndex + 1}
        />
      )}

      {/* Natural Language Builder Modal */}
      {showNLBuilder && (
        <NaturalLanguageBuilder
          onProcessGenerated={handleNLGenerated}
          onCancel={() => setShowNLBuilder(false)}
          tenant={tenant!}
          datasource={datasource!}
        />
      )}

      {/* Toast Notifications */}
      {toast && (
        <div className="fixed bottom-6 right-6 z-50">
          <Toast
            message={toast.message}
            type={toast.type}
            onClose={() => setToast(null)}
          />
        </div>
      )}
    </div>
  );
};

export default BusinessProcessBuilderEnhanced;
