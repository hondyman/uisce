import React, { useState } from 'react';
import { Trash2, AlertCircle } from 'lucide-react';
import type { ValidationRule } from '../types';

const RuleDependencyChain: React.FC<{
  rules: ValidationRule[];
  selectedRuleId: string;
  onUpdateDependencies: (ruleId: string, dependencies: string[]) => void;
}> = ({ rules, selectedRuleId, onUpdateDependencies }) => {
  const selectedRule = rules.find(r => r.id === selectedRuleId);
  const [dependencies, setDependencies] = useState<string[]>(selectedRule?.dependent_rule_ids || []);

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
            <strong>Rule Dependencies:</strong> Define which rules must pass before this rule executes. This creates a validation chain where dependent rules are evaluated first.
          </div>
        </div>
      </div>

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
        <div className="text-lg font-semibold text-blue-600">{selectedRule?.name || selectedRule?.rule_name}</div>
        <div className="text-sm text-gray-600 mt-1">{selectedRule?.description}</div>
        <div className="text-xs text-gray-500 mt-2">Entity: {selectedRule?.entity || selectedRule?.target_entity}</div>
      </div>

      <div>
        <label className="block text-sm font-semibold text-gray-700 mb-3">Rules that must pass first:</label>

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
                      <span className="w-8 h-8 bg-blue-600 text-white rounded-full flex items-center justify-center font-semibold text-sm">{index + 1}</span>
                      <div>
                        <div className="font-semibold text-gray-900">{depRule?.name || depRule?.rule_name}</div>
                        <div className="text-xs text-gray-500">{depRule?.entity || depRule?.target_entity}</div>
                      </div>
                    </div>
                    <button onClick={() => removeDependency(depId)} className="text-red-600 hover:bg-red-50 p-2 rounded" aria-label={`Remove dependency: ${depRule?.name || depRule?.rule_name}`} title={`Remove dependency: ${depRule?.name || depRule?.rule_name}`}>
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {availableRules.length > 0 && (
          <div className="mt-4">
            <select onChange={(e) => { if (e.target.value) { addDependency(e.target.value); e.target.value = ''; } }} className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500" aria-label="Add dependent rule">
              <option value="">+ Add dependent rule...</option>
              {availableRules.map(rule => (
                <option key={rule.id} value={rule.id}>{rule.name || rule.rule_name} ({rule.entity || rule.target_entity})</option>
              ))}
            </select>
          </div>
        )}

        {dependencies.length > 0 && (
          <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 mt-4">
            <h3 className="font-semibold text-gray-900 mb-4">Execution Order</h3>
            <div className="flex items-center gap-2 overflow-x-auto pb-2">
              {getExecutionOrder().map((rule, index) => (
                <React.Fragment key={rule.id}>
                  <div className={`flex-shrink-0 p-3 rounded-lg border-2 ${rule.id === selectedRuleId ? 'border-blue-500 bg-blue-50' : 'border-gray-300 bg-white'}`}>
                    <div className="flex items-center gap-2 mb-1">
                      <span className="w-6 h-6 bg-gray-700 text-white rounded-full flex items-center justify-center text-xs font-bold">{index + 1}</span>
                      <span className="font-semibold text-sm">{rule.name || rule.rule_name}</span>
                    </div>
                    <div className="text-xs text-gray-600">{rule.entity || rule.target_entity}</div>
                  </div>
                </React.Fragment>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default RuleDependencyChain;
