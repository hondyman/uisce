import React, { useState } from 'react';
import { Search, AlertCircle, CheckCircle, AlertTriangle, Link2 } from 'lucide-react';
import { Link } from 'react-router-dom';
import { devLog } from '../../utils/devLogger';

interface ValidationRule {
  id: string;
  name?: string;
  rule_name?: string;
  entity?: string;
  target_entity?: string;
  description?: string;
  severity?: 'error' | 'warning' | 'info';
  dependent_rule_ids?: string[];
}

interface AdvancedRuleConfigurationProps {
  rules?: ValidationRule[];
  onRulesUpdate?: (rules: ValidationRule[] | any[]) => void;
  onCrossEntitySave?: (condition: any) => void;
  selectedEntity?: string;
}

const AdvancedRuleConfiguration: React.FC<
  AdvancedRuleConfigurationProps
> = ({
  rules: propRules,
  onRulesUpdate: _onRulesUpdate,
  onCrossEntitySave: _onCrossEntitySave,
  selectedEntity: _selectedEntity,
}) => {
  const [activeTab, setActiveTab] = useState<number>(0);
  const [searchTerm, setSearchTerm] = useState('');

  // Use prop rules if provided, otherwise use default hardcoded rules for standalone mode
  const rules = propRules || [
    {
      id: 'rule_1',
      name: 'Age Verification',
      entity: 'Employee',
      description: 'Employee must be at least 18 years old',
      severity: 'error',
    },
    {
      id: 'rule_2',
      name: 'Status Check',
      entity: 'Employee',
      description: 'Employee status must be Active or On Leave',
      severity: 'warning',
    },
    {
      id: 'rule_3',
      name: 'Salary Range Validation',
      entity: 'Employee',
      description: 'Employee salary must be within position salary range',
      severity: 'error',
      dependent_rule_ids: ['rule_1', 'rule_2'],
    },
    {
      id: 'rule_4',
      name: 'Manager Assignment',
      entity: 'Employee',
      description: 'Employee must have an assigned manager if not executive level',
      severity: 'warning',
    },
  ];

  // Normalize rule names (handle both 'name' and 'rule_name' fields)
  const normalizedRules = rules.map(r => ({
    ...r,
    displayName: r.name || r.rule_name || 'Unnamed Rule'
  }));

  const filteredRules = normalizedRules.filter((rule: ValidationRule & { displayName: string }) => {
    const term = searchTerm.toLowerCase();
    const name = (rule.displayName ?? '').toLowerCase();
    const desc = (rule.description ?? '').toLowerCase();
    return name.includes(term) || desc.includes(term);
  });

  devLog('AdvancedRuleConfiguration: Rendering rules', { count: normalizedRules.length, filtered: filteredRules.length, selectedEntity: _selectedEntity });

  const getSeverityInfo = (severity?: string) => {
    const severityType = severity || 'info';
    const severities: Record<string, { bg: string; text: string; border: string; icon: React.ReactNode }> = {
      error: {
        bg: 'bg-red-50 dark:bg-red-950/20',
        text: 'text-red-900 dark:text-red-200',
        border: 'border-red-200 dark:border-red-800',
        icon: <AlertCircle size={16} className="text-red-600 dark:text-red-400" />,
      },
      warning: {
        bg: 'bg-amber-50 dark:bg-amber-950/20',
        text: 'text-amber-900 dark:text-amber-200',
        border: 'border-amber-200 dark:border-amber-800',
        icon: <AlertTriangle size={16} className="text-amber-600 dark:text-amber-400" />,
      },
      info: {
        bg: 'bg-blue-50 dark:bg-blue-950/20',
        text: 'text-blue-900 dark:text-blue-200',
        border: 'border-blue-200 dark:border-blue-800',
        icon: <CheckCircle size={16} className="text-blue-600 dark:text-blue-400" />,
      },
    };
    return severities[severityType] || severities.info;
  };

  const hasDependencies = (rule: ValidationRule) => {
    return rule.dependent_rule_ids && rule.dependent_rule_ids.length > 0;
  };

  const tabConfig = [
    { id: 0, label: '📋 Rules Overview', key: 'overview' },
  ];

  return (
    <div className="flex flex-col gap-6">
      {/* Search Bar */}
      <div className="flex items-center gap-2 px-4 py-3 bg-slate-100 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
        <Search size={20} className="text-slate-400 dark:text-slate-500 flex-shrink-0" />
        <input
          type="text"
          placeholder="Search rules by name or description..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="flex-1 bg-transparent text-slate-900 dark:text-slate-100 placeholder-slate-500 dark:placeholder-slate-400 outline-none"
        />
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-2 border-b border-slate-200 dark:border-slate-700">
        {tabConfig.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.id)}
            className={`
              px-4 py-3 font-medium text-sm transition-all duration-200 relative
              ${
                activeTab === tab.id
                  ? 'text-blue-600 dark:text-blue-400'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
              }
            `}
          >
            {tab.label}
            {activeTab === tab.id && (
              <div className="absolute bottom-0 left-0 right-0 h-1 bg-blue-600 dark:bg-blue-400 rounded-t-full"></div>
            )}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div>
        {/* Rules Overview Tab */}
        {activeTab === 0 && (
          <div>
            {filteredRules.length === 0 ? (
              <div className="p-8 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg text-center">
                <AlertCircle className="w-12 h-12 mx-auto mb-4 text-blue-400 opacity-50" />
                <h3 className="font-semibold text-blue-950 dark:text-blue-200 mb-1">No rules found</h3>
                <p className="text-blue-900/70 dark:text-blue-300/70 text-sm">
                  {searchTerm
                    ? 'Try a different search term'
                    : 'No validation rules are currently associated with this entity'}
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
                      <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-slate-100">Rule Name</th>
                      <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-slate-100">Entity</th>
                      <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-slate-100">Description</th>
                      <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-slate-100">Severity</th>
                      <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-slate-100">Dependencies</th>
                      <th className="px-4 py-3 text-left text-sm font-semibold text-slate-900 dark:text-slate-100">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredRules.map((rule: any) => {
                      const severity = rule.severity || 'info';
                      const severityInfo = getSeverityInfo(severity);
                      return (
                        <tr
                          key={rule.id}
                          className="border-b border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                        >
                          <td className="px-4 py-3 text-sm font-medium text-slate-900 dark:text-slate-100">{rule.displayName}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className="inline-flex items-center px-2.5 py-1.5 rounded-full text-xs font-medium bg-slate-200 dark:bg-slate-700 text-slate-800 dark:text-slate-200">
                              {rule.entity || rule.target_entity || 'N/A'}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-400">{rule.description || '—'}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-xs font-medium border ${severityInfo.bg} ${severityInfo.text} ${severityInfo.border} border`}>
                              {severityInfo.icon}
                              {severity.toUpperCase()}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`font-medium ${hasDependencies(rule) ? 'text-blue-600 dark:text-blue-400' : 'text-slate-500 dark:text-slate-400'}`}>
                              {hasDependencies(rule) ? `${rule.dependent_rule_ids?.length || 0} dependencies` : 'None'}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm">
                            <Link
                              to={`/core/validation-rules?ruleId=${encodeURIComponent(rule.id)}`}
                              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-950/20 transition-colors font-medium text-xs"
                            >
                              <Link2 size={16} />
                              <span>Go to Rule</span>
                            </Link>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default AdvancedRuleConfiguration;
