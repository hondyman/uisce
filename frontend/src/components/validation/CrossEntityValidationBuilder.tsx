import React, { useState, useCallback as _useCallback } from 'react';
import { ChevronRight, Plus as _Plus, Trash2, AlertCircle, Link2, ExternalLink } from 'lucide-react';
import type { ValidationRule } from './types';

interface EntityPath {
  segments: Array<{
    entity: string;
    field: string;
    relationship: string;
  }>;
  displayPath: string;
}

interface CrossEntityCondition {
  sourcePath: EntityPath;
  operator: string;
  targetPath: EntityPath;
}

// Mock data for entity relationships
const ENTITY_RELATIONSHIPS = {
  Employee: [
    { field: 'department_id', targetEntity: 'Department', relationship: 'many-to-one' },
    { field: 'manager_id', targetEntity: 'Employee', relationship: 'many-to-one' },
    { field: 'position_id', targetEntity: 'Position', relationship: 'many-to-one' },
    { field: 'location_id', targetEntity: 'Location', relationship: 'many-to-one' }
  ],
  Department: [
    { field: 'location_id', targetEntity: 'Location', relationship: 'many-to-one' },
    { field: 'cost_center_id', targetEntity: 'Cost Center', relationship: 'many-to-one' },
    { field: 'parent_department_id', targetEntity: 'Department', relationship: 'many-to-one' }
  ],
  Position: [
    { field: 'department_id', targetEntity: 'Department', relationship: 'many-to-one' },
    { field: 'job_family_id', targetEntity: 'Job Family', relationship: 'many-to-one' }
  ],
  Location: [
    { field: 'country_id', targetEntity: 'Country', relationship: 'many-to-one' }
  ]
};

const ENTITY_FIELDS = {
  Employee: [
    { name: 'employee_id', type: 'string', label: 'Employee ID' },
    { name: 'first_name', type: 'string', label: 'First Name' },
    { name: 'last_name', type: 'string', label: 'Last Name' },
    { name: 'salary', type: 'number', label: 'Salary' },
    { name: 'hire_date', type: 'date', label: 'Hire Date' },
    { name: 'status', type: 'string', label: 'Status' },
    { name: 'age', type: 'number', label: 'Age' }
  ],
  Department: [
    { name: 'department_name', type: 'string', label: 'Department Name' },
    { name: 'budget', type: 'number', label: 'Budget' },
    { name: 'head_count', type: 'number', label: 'Head Count' }
  ],
  Position: [
    { name: 'position_title', type: 'string', label: 'Position Title' },
    { name: 'min_salary', type: 'number', label: 'Minimum Salary' },
    { name: 'max_salary', type: 'number', label: 'Maximum Salary' },
    { name: 'job_level', type: 'number', label: 'Job Level' }
  ],
  Location: [
    { name: 'location_name', type: 'string', label: 'Location Name' },
    { name: 'city', type: 'string', label: 'City' },
    { name: 'country', type: 'string', label: 'Country' }
  ]
};

// Rule Dependency Chain Component
const RuleDependencyChain: React.FC<{
  rules: ValidationRule[];
  selectedRuleId: string;
  onUpdateDependencies: (ruleId: string, dependencies: string[]) => void;
}> = ({ rules, selectedRuleId, onUpdateDependencies }) => {
  const selectedRule = rules.find(r => r.id === selectedRuleId);
  const [dependencies, setDependencies] = useState<string[]>(
    selectedRule?.dependent_rule_ids || []
  );

  const availableRules = rules.filter(r => {
    if (!r.id) return false;
    return r.id !== selectedRuleId && !dependencies.includes(r.id);
  });

  const addDependency = (ruleId: string) => {
    const newDeps = [...dependencies, ruleId];
    setDependencies(newDeps);
    onUpdateDependencies(selectedRuleId, newDeps);
  };

  const removeDependency = (ruleId: string) => {
    const newDeps = dependencies.filter(id => id !== ruleId);
    setDependencies(newDeps);
    onUpdateDependencies(selectedRuleId, newDeps);
  };

  const getExecutionOrder = () => {
    const dependentRules = dependencies.map(id => rules.find(r => r.id === id)!);
    return [...dependentRules, selectedRule!];
  };

  return (
    <div className="space-y-6">
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="text-blue-600 flex-shrink-0 mt-0.5" size={20} />
          <div className="text-sm text-blue-800">
            <strong>Rule Dependencies:</strong> Define which rules must pass before this rule executes.
            This creates a validation chain where dependent rules are evaluated first.
          </div>
        </div>
      </div>

      {/* Current Rule */}
      <div className="bg-white border-2 border-blue-500 rounded-lg p-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="font-semibold text-gray-900">Current Rule</h3>
          <span className={`px-3 py-1 rounded text-xs font-semibold ${
            selectedRule?.severity === 'error' ? 'bg-red-100 text-red-700' :
            selectedRule?.severity === 'warning' ? 'bg-yellow-100 text-yellow-700' :
            'bg-blue-100 text-blue-700'
          }`}>
            {selectedRule?.severity}
          </span>
        </div>
        <div className="text-lg font-semibold text-blue-600">{selectedRule?.name}</div>
        <div className="text-sm text-gray-600 mt-1">{selectedRule?.description}</div>
        <div className="text-xs text-gray-500 mt-2">Entity: {selectedRule?.entity}</div>
      </div>

      {/* Dependencies */}
      <div>
        <label className="block text-sm font-semibold text-gray-700 mb-3">
          Rules that must pass first:
        </label>

        {dependencies.length === 0 ? (
          <div className="text-center py-8 border-2 border-dashed border-gray-300 rounded-lg bg-gray-50">
            <p className="text-gray-500 text-sm">No dependencies configured</p>
            <p className="text-gray-400 text-xs mt-1">This rule will execute independently</p>
          </div>
        ) : (
          <div className="space-y-2">
            {dependencies.map((depId, index) => {
              const depRule = rules.find(r => r.id === depId);
              return (
                <div key={depId} className="bg-white border border-gray-300 rounded-lg p-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="w-8 h-8 bg-blue-600 text-white rounded-full flex items-center justify-center font-semibold text-sm">
                        {index + 1}
                      </span>
                      <div>
                        <div className="font-semibold text-gray-900">{depRule?.name}</div>
                        <div className="text-xs text-gray-500">{depRule?.entity}</div>
                      </div>
                    </div>
                    <button
                      onClick={() => removeDependency(depId)}
                      className="text-red-600 hover:bg-red-50 p-2 rounded"
                      title={`Remove ${depRule?.name} dependency`}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Add Dependency */}
        {availableRules.length > 0 && (
          <div className="mt-4">
            <select
              onChange={(e) => {
                if (e.target.value) {
                  // @ts-ignore
                  addDependency(e.target.value);
                  e.target.value = '';
                }
              }}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              aria-label="Add dependent rule"
              title="Add dependent rule"
            >
              <option value="">+ Add dependent rule...</option>
              {availableRules.map(rule => (
                <option key={rule.id} value={rule.id}>
                  {rule.name} ({rule.entity})
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {/* Execution Order Visualization */}
      {dependencies.length > 0 && (
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
          <h3 className="font-semibold text-gray-900 mb-4">Execution Order</h3>
          <div className="flex items-center gap-2 overflow-x-auto pb-2">
            {getExecutionOrder().map((rule, index) => (
              <React.Fragment key={rule.id}>
                <div className={`flex-shrink-0 p-3 rounded-lg border-2 ${
                  rule.id === selectedRuleId
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-300 bg-white'
                }`}>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="w-6 h-6 bg-gray-700 text-white rounded-full flex items-center justify-center text-xs font-bold">
                      {index + 1}
                    </span>
                    <span className="font-semibold text-sm">{rule.name}</span>
                  </div>
                  <div className="text-xs text-gray-600">{rule.entity}</div>
                </div>
                {index < getExecutionOrder().length - 1 && (
                  <ChevronRight className="flex-shrink-0 text-gray-400" size={24} />
                )}
              </React.Fragment>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

// Entity Path Picker Component
const EntityPathPicker: React.FC<{
  startEntity: string;
  value: EntityPath | null;
  onChange: (path: EntityPath) => void;
  label: string;
}> = ({ startEntity, value, onChange, label }) => {
  const [currentEntity, setCurrentEntity] = useState(startEntity);
  const [pathSegments, setPathSegments] = useState<EntityPath['segments']>(
    value?.segments || []
  );
  const [isOpen, setIsOpen] = useState(false);

  const relationships = ENTITY_RELATIONSHIPS[currentEntity as keyof typeof ENTITY_RELATIONSHIPS] || [];
  const fields = ENTITY_FIELDS[currentEntity as keyof typeof ENTITY_FIELDS] || [];

  const addSegment = (field: string, targetEntity: string, relationship: string) => {
    const newSegment = {
      entity: currentEntity,
      field,
      relationship,
    };
    const newSegments = [...pathSegments, newSegment];
    setPathSegments(newSegments);
    setCurrentEntity(targetEntity);
  };

  const selectField = (fieldName: string) => {
    const displayPath = [...pathSegments.map(s => s.entity), currentEntity]
      .join(' → ') + '.' + fieldName;
    
    onChange({
      segments: pathSegments,
      displayPath: displayPath
    });
    setIsOpen(false);
  };

  const reset = () => {
    setPathSegments([]);
    setCurrentEntity(startEntity);
  };

  const currentPath = [...pathSegments.map(s => s.entity), currentEntity].join(' → ');

  return (
    <div className="space-y-2">
      <label className="block text-sm font-semibold text-gray-700">
        {label}
      </label>

      {/* Display Selected Path */}
      <div
        onClick={() => setIsOpen(!isOpen)}
        className="w-full px-4 py-3 border-2 border-gray-300 rounded-lg cursor-pointer hover:border-blue-400 bg-white"
      >
        {value ? (
          <div className="flex items-center justify-between">
            <span className="font-mono text-sm text-blue-600">{value.displayPath}</span>
            <ExternalLink size={16} className="text-gray-400" />
          </div>
        ) : (
          <div className="text-gray-400 text-sm">Click to select a field path...</div>
        )}
      </div>

      {/* Path Builder Modal */}
      {isOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-2xl w-full max-w-3xl max-h-[80vh] flex flex-col">
            <div className="bg-gradient-to-r from-purple-600 to-purple-700 text-white px-6 py-4 rounded-t-lg">
              <h3 className="text-xl font-semibold">Select Field Path</h3>
              <p className="text-purple-100 text-sm mt-1">Navigate through related entities to select a field</p>
            </div>

            <div className="p-6 flex-1 overflow-y-auto">
              {/* Current Path Display */}
              <div className="bg-purple-50 border border-purple-200 rounded-lg p-3 mb-4">
                <div className="text-xs font-semibold text-purple-700 mb-1">CURRENT PATH</div>
                <div className="font-mono text-sm text-purple-900">{currentPath}</div>
                {pathSegments.length > 0 && (
                  <button
                    onClick={reset}
                    className="mt-2 text-xs text-purple-600 hover:text-purple-700 underline"
                  >
                    Reset to {startEntity}
                  </button>
                )}
              </div>

              {/* Relationships - Navigate to related entities */}
              {relationships.length > 0 && (
                <div className="mb-6">
                  <h4 className="font-semibold text-gray-900 mb-3 flex items-center gap-2">
                    <Link2 size={18} className="text-purple-600" />
                    Related Entities
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                    {relationships.map((rel) => (
                      <button
                        key={rel.field}
                        onClick={() => addSegment(rel.field, rel.targetEntity, rel.relationship)}
                        className="text-left p-3 border-2 border-purple-200 rounded-lg hover:border-purple-400 hover:bg-purple-50 transition-all"
                      >
                        <div className="font-semibold text-gray-900">{rel.targetEntity}</div>
                        <div className="text-xs text-gray-600 mt-1">via {rel.field}</div>
                        <div className="text-xs text-purple-600 mt-1">{rel.relationship}</div>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Fields - Select final field */}
              <div>
                <h4 className="font-semibold text-gray-900 mb-3">Select Field from {currentEntity}</h4>
                <div className="grid grid-cols-2 gap-2">
                  {fields.map((field) => (
                    <button
                      key={field.name}
                      onClick={() => selectField(field.name)}
                      className="text-left p-3 border border-gray-300 rounded-lg hover:border-blue-400 hover:bg-blue-50 transition-all"
                    >
                      <div className="font-semibold text-gray-900 text-sm">{field.label}</div>
                      <div className="text-xs text-gray-500 mt-1">{field.name}</div>
                      <span className={`inline-block px-2 py-0.5 text-xs rounded mt-2 ${
                        field.type === 'string' ? 'bg-blue-100 text-blue-700' :
                        field.type === 'number' ? 'bg-green-100 text-green-700' :
                        'bg-purple-100 text-purple-700'
                      }`}>
                        {field.type}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <div className="bg-gray-50 px-6 py-4 border-t rounded-b-lg flex justify-end">
              <button
                onClick={() => setIsOpen(false)}
                className="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// Cross-Entity Validation Builder
export const CrossEntityValidationBuilder: React.FC<{
  sourceEntity: string;
  onSave: (condition: CrossEntityCondition) => void;
}> = ({ sourceEntity, onSave }) => {
  const [sourcePath, setSourcePath] = useState<EntityPath | null>(null);
  const [operator, setOperator] = useState('equals');
  const [targetPath, setTargetPath] = useState<EntityPath | null>(null);

  const operators = [
    { value: 'equals', label: 'Equals (=)' },
    { value: 'not_equals', label: 'Not Equals (≠)' },
    { value: 'greater_than', label: 'Greater Than (>)' },
    { value: 'less_than', label: 'Less Than (<)' },
    { value: 'greater_equal', label: 'Greater or Equal (≥)' },
    { value: 'less_equal', label: 'Less or Equal (≤)' }
  ];

  const handleSave = () => {
    if (sourcePath && targetPath) {
      onSave({ sourcePath, operator, targetPath });
    }
  };

  const isValid = sourcePath && targetPath;

  return (
    <div className="space-y-6">
      <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="text-purple-600 flex-shrink-0 mt-0.5" size={20} />
          <div className="text-sm text-purple-800">
            <strong>Cross-Entity Validation:</strong> Compare fields across related entities.
            Example: Validate that "Employee → Position → min_salary" is less than "Employee → salary"
          </div>
        </div>
      </div>

      {/* Source Path */}
      <EntityPathPicker
        startEntity={sourceEntity}
        value={sourcePath}
        onChange={setSourcePath}
        label="Source Field Path"
      />

      {/* Operator */}
      <div>
        <label className="block text-sm font-semibold text-gray-700 mb-2">
          Comparison Operator
        </label>
        <div className="grid grid-cols-3 gap-2">
          {operators.map((op) => (
            <button
              key={op.value}
              onClick={() => setOperator(op.value)}
              className={`p-3 rounded-lg border-2 text-sm font-semibold transition-all ${
                operator === op.value
                  ? 'border-purple-600 bg-purple-50 text-purple-700'
                  : 'border-gray-300 hover:border-gray-400'
              }`}
            >
              {op.label}
            </button>
          ))}
        </div>
      </div>

      {/* Target Path */}
      <EntityPathPicker
        startEntity={sourceEntity}
        value={targetPath}
        onChange={setTargetPath}
        label="Target Field Path"
      />

      {/* Visual Representation */}
      {sourcePath && targetPath && (
        <div className="bg-gradient-to-r from-purple-50 to-blue-50 border-2 border-purple-300 rounded-lg p-4">
          <h4 className="font-semibold text-gray-900 mb-3">Validation Rule Preview</h4>
          <div className="flex items-center gap-3 text-sm">
            <div className="flex-1 bg-white p-3 rounded border border-purple-300">
              <div className="text-xs text-purple-600 font-semibold mb-1">SOURCE</div>
              <div className="font-mono text-purple-900">{sourcePath.displayPath}</div>
            </div>
            <div className="px-4 py-2 bg-purple-600 text-white rounded-full font-bold">
              {operators.find(op => op.value === operator)?.label}
            </div>
            <div className="flex-1 bg-white p-3 rounded border border-blue-300">
              <div className="text-xs text-blue-600 font-semibold mb-1">TARGET</div>
              <div className="font-mono text-blue-900">{targetPath.displayPath}</div>
            </div>
          </div>
        </div>
      )}

      {/* Save Button */}
      <button
        onClick={handleSave}
        disabled={!isValid}
        className={`w-full py-3 rounded-lg font-semibold transition-all ${
          isValid
            ? 'bg-purple-600 text-white hover:bg-purple-700'
            : 'bg-gray-300 text-gray-500 cursor-not-allowed'
        }`}
      >
        {isValid ? 'Add Cross-Entity Validation' : 'Select both paths to continue'}
      </button>
    </div>
  );
};

// Export Rule Dependency Chain for external use
export { RuleDependencyChain };

// Export types
export type { ValidationRule, EntityPath, CrossEntityCondition };

// Demo
const Demo = () => {
  const [activeTab, setActiveTab] = useState<'dependency' | 'cross-entity'>('dependency');
  
  const [rules, setRules] = useState<ValidationRule[]>([
    {
      id: 'rule_1',
      name: 'Age Verification',
      entity: 'Employee',
      description: 'Employee must be at least 18 years old',
      severity: 'error'
    },
    {
      id: 'rule_2',
      name: 'Status Check',
      entity: 'Employee',
      description: 'Employee status must be Active or On Leave',
      severity: 'warning'
    },
    {
      id: 'rule_3',
      name: 'Salary Range Validation',
      entity: 'Employee',
      description: 'Employee salary must be within position salary range',
      severity: 'error',
      dependent_rule_ids: ['rule_1', 'rule_2']
    },
    {
      id: 'rule_4',
      name: 'Manager Assignment',
      entity: 'Employee',
      description: 'Employee must have an assigned manager if not executive level',
      severity: 'warning'
    }
  ]);

  const [selectedRuleId, setSelectedRuleId] = useState('rule_3');
  const [crossEntityConditions, setCrossEntityConditions] = useState<CrossEntityCondition[]>([]);

  const handleUpdateDependencies = (ruleId: string, dependencies: string[]) => {
    setRules(rules.map(rule =>
      rule.id === ruleId
        ? { ...rule, dependent_rule_ids: dependencies }
        : rule
    ));
  };

  const handleAddCrossEntity = (condition: CrossEntityCondition) => {
    setCrossEntityConditions([...crossEntityConditions, condition]);
  };

  return (
    <div className="min-h-screen bg-gray-100 p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            Advanced Rule Configuration vvvv
          </h1>
          <p className="text-gray-600">
            Configure rule dependencies and cross-entity validations for complex business logic
          </p>
        </div>

        {/* Tabs */}
        <div className="bg-white rounded-lg shadow">
          <div className="border-b border-gray-200">
            <div className="flex">
              <button
                onClick={() => setActiveTab('dependency')}
                className={`px-6 py-3 font-semibold border-b-2 transition-colors ${
                  activeTab === 'dependency'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                Rule Dependencies
              </button>
              <button
                onClick={() => setActiveTab('cross-entity')}
                className={`px-6 py-3 font-semibold border-b-2 transition-colors ${
                  activeTab === 'cross-entity'
                    ? 'border-purple-600 text-purple-600'
                    : 'border-transparent text-gray-600 hover:text-gray-900'
                }`}
              >
                Cross-Entity Validation
              </button>
            </div>
          </div>

          <div className="p-6">
            {activeTab === 'dependency' && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2" htmlFor="rule-selector">
                    Select Rule to Configure
                  </label>
                  <select
                    id="rule-selector"
                    value={selectedRuleId}
                    onChange={(e) => setSelectedRuleId(e.target.value)}
                    className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  >
                    {rules.map(rule => (
                      <option key={rule.id} value={rule.id}>
                        {rule.name} - {rule.entity}
                      </option>
                    ))}
                  </select>
                </div>

                <RuleDependencyChain
                  rules={rules}
                  selectedRuleId={selectedRuleId}
                  onUpdateDependencies={handleUpdateDependencies}
                />
              </div>
            )}

            {activeTab === 'cross-entity' && (
              <div className="space-y-6">
                <CrossEntityValidationBuilder
                  sourceEntity="Employee"
                  onSave={handleAddCrossEntity}
                />

                {/* Display saved cross-entity conditions */}
                {crossEntityConditions.length > 0 && (
                  <div className="mt-6">
                    <h3 className="font-semibold text-gray-900 mb-3">
                      Saved Cross-Entity Validations ({crossEntityConditions.length})
                    </h3>
                    <div className="space-y-3">
                      {crossEntityConditions.map((condition, index) => (
                        <div key={index} className="bg-white border border-gray-300 rounded-lg p-4">
                          <div className="flex items-center gap-3 text-sm">
                            <div className="flex-1 font-mono text-purple-700">
                              {condition.sourcePath.displayPath}
                            </div>
                            <div className="px-3 py-1 bg-purple-600 text-white rounded font-semibold text-xs">
                              {condition.operator}
                            </div>
                            <div className="flex-1 font-mono text-blue-700">
                              {condition.targetPath.displayPath}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Demo;
