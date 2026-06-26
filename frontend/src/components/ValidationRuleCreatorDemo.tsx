/**
 * Demo Component: ValidationRuleCreator with Advanced Expression Builder
 * 
 * This demo showcases the enhanced condition-building experience with:
 * - Type-aware operator filtering (e.g., string fields show text operators, numbers show comparison)
 * - Smart value visibility (operators like "is_empty" hide the value field)
 * - Advanced Looker-style filter expressions
 * - Relative date expressions (last 7 days, this month, etc.)
 * - Pattern matching with wildcards
 * - Numeric intervals and logical operators
 * - Field type auto-detection hints
 * - Better visual feedback and guidance
 */

import React, { useState } from 'react';
import { ValidationRuleCreator } from './ValidationRules/ValidationRuleCreator';
import type { ValidationRule, Condition } from './validation/types';
import type { FieldTypeInfo } from './ValidationRules/ValidationRuleCreator';

interface ComponentProps {
  fieldMetadata?: Record<string, FieldTypeInfo>;
}

// Example field metadata that would normally come from your backend/schema
const EXAMPLE_FIELD_METADATA: Record<string, FieldTypeInfo> = {
  employee_id: {
    type: 'string',
    isNullable: false,
  },
  salary: {
    type: 'number',
    isNullable: false,
  },
  hire_date: {
    type: 'date',
    isNullable: true,
  },
  department: {
    type: 'enum',
    enumValues: ['HR', 'Engineering', 'Sales', 'Finance'],
    isNullable: false,
  },
  is_active: {
    type: 'boolean',
    isNullable: false,
  },
  email: {
    type: 'string',
    isNullable: true,
  },
  years_experience: {
    type: 'number',
    isNullable: true,
  },
  join_date: {
    type: 'date',
    isNullable: false,
  },
};

export const ValidationRuleCreatorDemo: React.FC<ComponentProps> = ({ fieldMetadata = EXAMPLE_FIELD_METADATA }) => {
  const [isOpen, setIsOpen] = useState(true);
  const [rules, setRules] = useState<ValidationRule[]>([]);
  const [editingRule, setEditingRule] = useState<ValidationRule | null>(null);

  const handleSave = (rule: ValidationRule) => {
    if (editingRule) {
      // Update existing rule
      setRules((prev) =>
        prev.map((r) => (r.id === editingRule.id ? rule : r))
      );
      setEditingRule(null);
    } else {
      // Add new rule
      setRules((prev) => [rule, ...prev]);
    }
    setIsOpen(false);
  };

  const handleEdit = (rule: ValidationRule) => {
    setEditingRule(rule);
    setIsOpen(true);
  };

  const handleClose = () => {
    setIsOpen(false);
    setEditingRule(null);
  };

  return (
    <div className="p-6 bg-gray-100 min-h-screen">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="mb-6 flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Validation Rules</h1>
            <p className="text-gray-600 text-sm mt-1">
              Create and manage validation rules with smart, type-aware conditions
            </p>
          </div>
          <button
            onClick={() => {
              setEditingRule(null);
              setIsOpen(true);
            }}
            className="px-4 py-2 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 transition"
          >
            + New Rule
          </button>
        </div>

        {/* Info Panel */}
        <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <h3 className="font-semibold text-blue-900 mb-2">✨ Advanced Condition Builder Features</h3>
          <div className="grid md:grid-cols-2 gap-4 text-sm text-blue-800">
            <div>
              <p className="font-medium mb-1">🎯 Basic Features:</p>
              <ul className="space-y-1">
                <li>✓ Type-aware operator filtering</li>
                <li>✓ Smart value field visibility</li>
                <li>✓ Field type hints and guidance</li>
              </ul>
            </div>
            <div>
              <p className="font-medium mb-1">⚡ Advanced Features:</p>
              <ul className="space-y-1">
                <li>✓ Looker-style filter expressions</li>
                <li>✓ Relative dates (last 7 days, this month, etc.)</li>
                <li>✓ Pattern matching with wildcards</li>
                <li>✓ Numeric intervals & logical operators</li>
              </ul>
            </div>
          </div>
        </div>

        {/* Example Expressions Panel */}
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg">
          <h3 className="font-semibold text-green-900 mb-3">💡 Example Expressions (Try These!)</h3>
          <div className="grid md:grid-cols-3 gap-4 text-sm">
            <div className="bg-white p-3 rounded border border-green-200">
              <p className="font-medium text-green-900">String Patterns:</p>
              <ul className="mt-2 space-y-1 text-gray-700 font-mono text-xs">
                <li>%employee% (contains)</li>
                <li>emp% (starts with)</li>
                <li>-test (NOT test)</li>
              </ul>
            </div>
            <div className="bg-white p-3 rounded border border-green-200">
              <p className="font-medium text-green-900">Numeric Ranges:</p>
              <ul className="mt-2 space-y-1 text-gray-700 font-mono text-xs">
                <li>[50000, 100000] (inclusive)</li>
                <li>&gt;=5 AND &lt;=10 (AND logic)</li>
                <li>NOT 5 (any except 5)</li>
              </ul>
            </div>
            <div className="bg-white p-3 rounded border border-green-200">
              <p className="font-medium text-green-900">Relative Dates:</p>
              <ul className="mt-2 space-y-1 text-gray-700 font-mono text-xs">
                <li>last 7 days</li>
                <li>this month</li>
                <li>3 days ago</li>
              </ul>
            </div>
          </div>
        </div>

        {/* Rules List */}
        {rules.length === 0 ? (
          <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
            <p className="text-gray-500">No validation rules created yet. Click "New Rule" to get started.</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {rules.map((rule) => (
              <div key={rule.id} className="p-4 bg-white border rounded-lg hover:shadow-md transition">
                <div className="flex justify-between items-start mb-3">
                  <div className="flex-1">
                    <h3 className="font-semibold text-gray-900">{rule.rule_name}</h3>
                    <p className="text-sm text-gray-600 mt-1">{rule.description}</p>
                  </div>
                  <div className={`px-2 py-1 rounded text-xs font-medium ${
                    rule.severity === 'error' ? 'bg-red-100 text-red-800' :
                    rule.severity === 'warning' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-blue-100 text-blue-800'
                  }`}>
                    {rule.severity}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-2 mb-3 text-sm">
                  <div>
                    <span className="text-gray-600">Type:</span>
                    <div className="font-medium">{rule.rule_type}</div>
                  </div>
                  <div>
                    <span className="text-gray-600">Entity:</span>
                    <div className="font-medium">{rule.target_entity}</div>
                  </div>
                </div>

                {rule.conditions && rule.conditions.length > 0 && (
                  <div className="mb-3 p-2 bg-gray-50 rounded text-sm">
                    <p className="text-gray-600 font-medium mb-2">Conditions:</p>
                    {rule.conditions.map((cond: Condition, i: number) => (
                      <div key={i} className="text-xs text-gray-700 ml-2">
                        • <strong>{cond.field}</strong> {cond.operator} {cond.value && `"${cond.value}"`}
                      </div>
                    ))}
                  </div>
                )}

                <div className="flex gap-2 justify-end pt-2 border-t">
                  <button
                    onClick={() => handleEdit(rule)}
                    className="px-3 py-1 text-sm text-blue-600 hover:bg-blue-50 rounded transition"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => setRules((prev) => prev.filter((r) => r.id !== rule.id))}
                    className="px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded transition"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Modal */}
        {isOpen && (
          <ValidationRuleCreator
            isOpen={isOpen}
            onClose={handleClose}
            onSave={handleSave}
            // This demo runs without tenant/datasource scope: provide empty values
            tenantId={''}
            datasourceId={''}
            availableEntities={['Employee', 'Department', 'Position', 'Organization']}
            displayMode="modal"
            initialRule={editingRule}
            fieldMetadata={fieldMetadata}
          />
        )}
      </div>
    </div>
  );
};

export default ValidationRuleCreatorDemo;
