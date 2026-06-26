import React, { useState } from 'react';
import { Plus, Trash2, GitBranch, Users, Link as LinkIcon, AlertTriangle, ChevronDown, ChevronUp } from 'lucide-react';
import { Condition, ConditionBranch, ApprovalChain, NotificationConfig, BPStep } from './useBPBuilderAPI';

// ============================================================================
// ADVANCED CONDITION BUILDER
// ============================================================================

interface ConditionBuilderProps {
  condition: ConditionBranch | undefined;
  onChange: (condition: ConditionBranch | undefined) => void;
  availableFields: string[];
  availableSteps: BPStep[];
}

export const AdvancedConditionBuilder: React.FC<ConditionBuilderProps> = ({
  condition,
  onChange,
  availableFields,
  availableSteps
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const addCondition = () => {
    const newCondition: Condition = {
      field: '',
      operator: '==',
      value: ''
    };

    const updated: ConditionBranch = condition || {
      operator: 'AND',
      conditions: [],
      trueBranch: [],
      falseBranch: []
    };

    updated.conditions.push(newCondition);
    onChange(updated);
  };

  const removeCondition = (index: number) => {
    if (!condition) return;
    const updated = { ...condition };
    updated.conditions = updated.conditions.filter((_, i) => i !== index);
    onChange(updated);
  };

  const updateCondition = (index: number, field: keyof Condition, value: any) => {
    if (!condition) return;
    const updated = { ...condition };
    updated.conditions[index] = {
      ...updated.conditions[index],
      [field]: value
    };
    onChange(updated);
  };

  const operators = [
    { value: '==', label: 'Equals' },
    { value: '!=', label: 'Not Equals' },
    { value: '>', label: 'Greater Than' },
    { value: '<', label: 'Less Than' },
    { value: '>=', label: 'Greater or Equal' },
    { value: '<=', label: 'Less or Equal' },
    { value: 'in', label: 'In List' },
    { value: 'contains', label: 'Contains' },
    { value: 'startsWith', label: 'Starts With' },
    { value: 'endsWith', label: 'Ends With' }
  ];

  return (
    <div className="border rounded-lg p-4 bg-gray-50">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center justify-between w-full text-left font-semibold text-gray-700 mb-2"
      >
        <span className="flex items-center gap-2">
          <GitBranch size={16} />
          Advanced Conditional Logic
        </span>
        {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
      </button>

      {isExpanded && (
        <div className="space-y-4 mt-4">
          {/* Boolean Operator */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Boolean Operator</label>
            <select
              value={condition?.operator || 'AND'}
              onChange={(e) => onChange({
                ...condition!,
                operator: e.target.value as 'AND' | 'OR' | 'NOT'
              })}
              className="w-full px-3 py-2 border rounded-lg"
            >
              <option value="AND">AND (All must be true)</option>
              <option value="OR">OR (Any can be true)</option>
              <option value="NOT">NOT (Negate conditions)</option>
            </select>
          </div>

          {/* Conditions List */}
          <div>
            <div className="flex justify-between items-center mb-2">
              <label className="text-sm font-medium text-gray-700">Conditions</label>
              <button
                onClick={addCondition}
                className="flex items-center gap-1 px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600"
              >
                <Plus size={14} /> Add Condition
              </button>
            </div>

            <div className="space-y-2">
              {condition?.conditions?.map((cond, idx) => (
                <div key={idx} className="flex gap-2 items-center bg-white p-3 rounded border">
                  <select
                    value={cond.field}
                    onChange={(e) => updateCondition(idx, 'field', e.target.value)}
                    className="flex-1 px-2 py-1 border rounded text-sm"
                  >
                    <option value="">Select field...</option>
                    {availableFields.map(field => (
                      <option key={field} value={field}>{field}</option>
                    ))}
                  </select>

                  <select
                    value={cond.operator}
                    onChange={(e) => updateCondition(idx, 'operator', e.target.value)}
                    className="px-2 py-1 border rounded text-sm"
                  >
                    {operators.map(op => (
                      <option key={op.value} value={op.value}>{op.label}</option>
                    ))}
                  </select>

                  <input
                    type="text"
                    value={cond.value}
                    onChange={(e) => updateCondition(idx, 'value', e.target.value)}
                    className="flex-1 px-2 py-1 border rounded text-sm"
                    placeholder="Value..."
                  />

                  <button
                    onClick={() => removeCondition(idx)}
                    className="p-1 text-red-500 hover:bg-red-50 rounded"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Branch Configuration */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">True Branch (Steps)</label>
              <select
                multiple
                value={condition?.trueBranch || []}
                onChange={(e) => {
                  const selected = Array.from(e.target.selectedOptions).map(opt => opt.value);
                  onChange({ ...condition!, trueBranch: selected });
                }}
                className="w-full px-2 py-1 border rounded text-sm h-24"
              >
                {availableSteps.map(step => (
                  <option key={step.id} value={step.id}>{step.stepName}</option>
                ))}
              </select>
              <p className="text-xs text-gray-500 mt-1">Steps to execute if condition is true</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">False Branch (Steps)</label>
              <select
                multiple
                value={condition?.falseBranch || []}
                onChange={(e) => {
                  const selected = Array.from(e.target.selectedOptions).map(opt => opt.value);
                  onChange({ ...condition!, falseBranch: selected });
                }}
                className="w-full px-2 py-1 border rounded text-sm h-24"
              >
                {availableSteps.map(step => (
                  <option key={step.id} value={step.id}>{step.stepName}</option>
                ))}
              </select>
              <p className="text-xs text-gray-500 mt-1">Steps to execute if condition is false</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// APPROVAL CHAIN CONFIGURATOR
// ============================================================================

interface ApprovalChainConfigProps {
  approvalChain: ApprovalChain | undefined;
  onChange: (chain: ApprovalChain | undefined) => void;
  availableRoles: string[];
}

export const ApprovalChainConfig: React.FC<ApprovalChainConfigProps> = ({
  approvalChain,
  onChange,
  availableRoles
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const initChain = () => {
    if (!approvalChain) {
      onChange({
        type: 'role',
        approvalMode: 'all',
        roles: []
      });
    }
  };

  return (
    <div className="border rounded-lg p-4 bg-gray-50">
      <button
        onClick={() => {
          setIsExpanded(!isExpanded);
          if (!isExpanded) initChain();
        }}
        className="flex items-center justify-between w-full text-left font-semibold text-gray-700 mb-2"
      >
        <span className="flex items-center gap-2">
          <Users size={16} />
          Dynamic Approval Chain
        </span>
        {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
      </button>

      {isExpanded && approvalChain && (
        <div className="space-y-4 mt-4">
          {/* Approval Type */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Approval Type</label>
            <select
              value={approvalChain.type}
              onChange={(e) => onChange({
                ...approvalChain,
                type: e.target.value as ApprovalChain['type']
              })}
              className="w-full px-3 py-2 border rounded-lg"
            >
              <option value="role">Single Role</option>
              <option value="multi_role">Multiple Roles</option>
              <option value="org_hierarchy">Organization Hierarchy</option>
              <option value="custom">Custom Logic</option>
            </select>
          </div>

          {/* Org Hierarchy Levels */}
          {approvalChain.type === 'org_hierarchy' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Levels Up (e.g., 2 = manager's manager)
              </label>
              <input
                type="number"
                min="1"
                max="5"
                value={approvalChain.levels || 1}
                onChange={(e) => onChange({
                  ...approvalChain,
                  levels: parseInt(e.target.value)
                })}
                className="w-full px-3 py-2 border rounded-lg"
              />
            </div>
          )}

          {/* Multi-Role Selection */}
          {(approvalChain.type === 'role' || approvalChain.type === 'multi_role') && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Select Roles</label>
              <select
                multiple
                value={approvalChain.roles || []}
                onChange={(e) => {
                  const selected = Array.from(e.target.selectedOptions).map(opt => opt.value);
                  onChange({ ...approvalChain, roles: selected });
                }}
                className="w-full px-2 py-1 border rounded text-sm h-32"
              >
                {availableRoles.map(role => (
                  <option key={role} value={role}>{role}</option>
                ))}
              </select>
              <p className="text-xs text-gray-500 mt-1">Hold Ctrl/Cmd to select multiple</p>
            </div>
          )}

          {/* Approval Mode */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Approval Mode</label>
            <select
              value={approvalChain.approvalMode}
              onChange={(e) => onChange({
                ...approvalChain,
                approvalMode: e.target.value as ApprovalChain['approvalMode']
              })}
              className="w-full px-3 py-2 border rounded-lg"
            >
              <option value="all">All Must Approve</option>
              <option value="any">Any One Can Approve</option>
              <option value="majority">Majority Must Approve</option>
            </select>
          </div>

          {/* Escalation Path */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
              <AlertTriangle size={14} />
              Escalation Path (on timeout)
            </label>
            <select
              multiple
              value={approvalChain.escalationPath || []}
              onChange={(e) => {
                const selected = Array.from(e.target.selectedOptions).map(opt => opt.value);
                onChange({ ...approvalChain, escalationPath: selected });
              }}
              className="w-full px-2 py-1 border rounded text-sm h-24"
            >
              {availableRoles.map(role => (
                <option key={role} value={role}>{role}</option>
              ))}
            </select>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// STEP DEPENDENCIES MANAGER
// ============================================================================

interface StepDependenciesProps {
  dependsOn: string[];
  skipCondition: ConditionBranch | undefined;
  availableSteps: BPStep[];
  availableFields: string[];
  onDependsOnChange: (deps: string[]) => void;
  onSkipConditionChange: (condition: ConditionBranch | undefined) => void;
}

export const StepDependenciesManager: React.FC<StepDependenciesProps> = ({
  dependsOn,
  skipCondition,
  availableSteps,
  availableFields,
  onDependsOnChange,
  onSkipConditionChange
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div className="border rounded-lg p-4 bg-gray-50">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center justify-between w-full text-left font-semibold text-gray-700 mb-2"
      >
        <span className="flex items-center gap-2">
          <LinkIcon size={16} />
          Step Dependencies & Skip Logic
        </span>
        {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
      </button>

      {isExpanded && (
        <div className="space-y-4 mt-4">
          {/* Dependencies */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Depends On (must complete first)
            </label>
            <select
              multiple
              value={dependsOn}
              onChange={(e) => {
                const selected = Array.from(e.target.selectedOptions).map(opt => opt.value);
                onDependsOnChange(selected);
              }}
              className="w-full px-2 py-1 border rounded text-sm h-24"
            >
              {availableSteps.map(step => (
                <option key={step.id} value={step.id}>{step.stepName}</option>
              ))}
            </select>
            <p className="text-xs text-gray-500 mt-1">This step will wait for selected steps to complete</p>
          </div>

          {/* Skip Condition */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Skip Condition</label>
            <AdvancedConditionBuilder
              condition={skipCondition}
              onChange={onSkipConditionChange}
              availableFields={availableFields}
              availableSteps={availableSteps}
            />
            <p className="text-xs text-gray-500 mt-1">Skip this step if the condition evaluates to true</p>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// PARALLEL EXECUTION CONFIGURATOR
// ============================================================================

interface ParallelExecutionProps {
  executionMode: 'sequential' | 'parallel';
  parallelGroup?: string;
  waitForAll?: boolean;
  onExecutionModeChange: (mode: 'sequential' | 'parallel') => void;
  onParallelGroupChange: (group: string | undefined) => void;
  onWaitForAllChange: (wait: boolean) => void;
}

export const ParallelExecutionConfig: React.FC<ParallelExecutionProps> = ({
  executionMode,
  parallelGroup,
  waitForAll,
  onExecutionModeChange,
  onParallelGroupChange,
  onWaitForAllChange
}) => {
  return (
    <div className="border rounded-lg p-4 bg-blue-50">
      <h4 className="font-semibold text-gray-700 mb-3 flex items-center gap-2">
        <GitBranch size={16} />
        Parallel Execution
      </h4>

      <div className="space-y-3">
        {/* Execution Mode */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Execution Mode</label>
          <select
            value={executionMode}
            onChange={(e) => onExecutionModeChange(e.target.value as 'sequential' | 'parallel')}
            className="w-full px-3 py-2 border rounded-lg bg-white"
          >
            <option value="sequential">Sequential (one after another)</option>
            <option value="parallel">Parallel (run simultaneously)</option>
          </select>
        </div>

        {/* Parallel Group */}
        {executionMode === 'parallel' && (
          <>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Parallel Group</label>
              <input
                type="text"
                value={parallelGroup || ''}
                onChange={(e) => onParallelGroupChange(e.target.value || undefined)}
                className="w-full px-3 py-2 border rounded-lg"
                placeholder="e.g., approval-group-1"
              />
              <p className="text-xs text-gray-500 mt-1">
                Steps with the same group name execute in parallel
              </p>
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="waitForAll"
                checked={waitForAll || false}
                onChange={(e) => onWaitForAllChange(e.target.checked)}
                className="rounded"
              />
              <label htmlFor="waitForAll" className="text-sm text-gray-700">
                Wait for all steps in group to complete
              </label>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
