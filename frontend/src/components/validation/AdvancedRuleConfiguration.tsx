import React, { useState } from 'react';
import { Trash2 } from 'lucide-react';
import './AdvancedRuleConfiguration.css';

interface ValidationRule {
  id: string;
  name: string;
  entity: string;
  description: string;
  severity: 'error' | 'warning' | 'info';
  dependent_rule_ids?: string[];
}

interface AdvancedRuleConfigurationProps {
  rules?: ValidationRule[];
  onRulesUpdate?: (rules: ValidationRule[]) => void;
  onCrossEntitySave?: (condition: any) => void;
  selectedEntity?: string;
}

const AdvancedRuleConfiguration: React.FC<AdvancedRuleConfigurationProps> = ({ 
  rules: propRules, 
  onRulesUpdate: _onRulesUpdate, 
  onCrossEntitySave: _onCrossEntitySave, 
  selectedEntity: _selectedEntity 
}) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'dependency' | 'cross-entity'>('overview');
  const [searchTerm, setSearchTerm] = useState('');

  // Use prop rules if provided, otherwise use default hardcoded rules for standalone mode
  const rules = propRules || [
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
  ];

  const filteredRules = rules.filter((rule: ValidationRule) => {
    const term = searchTerm.toLowerCase();
    const name = (rule.name ?? '').toLowerCase();
    const desc = (rule.description ?? '').toLowerCase();
    return name.includes(term) || desc.includes(term);
  });

  const getSeverityStyles = (severity: string) => {
    const styles = {
      error: 'bg-gradient-to-r from-red-500/30 to-red-600/30 dark:from-red-600/20 dark:to-red-700/20 text-red-700 dark:text-red-300 border border-red-500/50 dark:border-red-600/40 shadow-sm shadow-red-500/20 dark:shadow-red-700/10 backdrop-blur-sm',
      warning: 'bg-gradient-to-r from-orange-500/30 to-amber-600/30 dark:from-orange-600/20 dark:to-amber-700/20 text-orange-700 dark:text-amber-300 border border-orange-500/50 dark:border-amber-600/40 shadow-sm shadow-orange-500/20 dark:shadow-amber-700/10 backdrop-blur-sm',
      info: 'bg-gradient-to-r from-sky-500/30 to-blue-600/30 dark:from-sky-600/20 dark:to-blue-700/20 text-sky-700 dark:text-blue-300 border border-sky-500/50 dark:border-blue-600/40 shadow-sm shadow-sky-500/20 dark:shadow-blue-700/10 backdrop-blur-sm'
    };
    return styles[severity as keyof typeof styles] || styles.info;
  };

  const hasDependencies = (rule: ValidationRule) => {
    return rule.dependent_rule_ids && rule.dependent_rule_ids.length > 0 ? 'Yes' : 'No';
  };

  return (
    <div className="advanced-rule-config relative flex h-auto w-full flex-col bg-gradient-to-br from-slate-50 via-blue-50/30 to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950 rounded-lg">
      {/* PostCSS config enabled - glassmorphism styles active */}
      <div className="layout-container flex h-full grow flex-col">
        {/* Main Content */}
        <main className="flex flex-1 justify-center">
          <div className="layout-content-container flex flex-col w-full flex-1 gap-5">
            {/* Tabs - Compact for tab environment */}
            <div className="border-b border-slate-200/80 dark:border-slate-800/80">
              <div className="flex gap-1 sm:gap-3">
                {[
                  { id: 'overview', label: 'Rules Overview' },
                  { id: 'dependency', label: 'Dependencies' },
                  { id: 'cross-entity', label: 'Cross-Entity Validation' }
                ].map(tab => (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id as any)}
                    className={`relative flex flex-col items-center justify-center pb-3 pt-2 px-3 sm:px-5 transition-all duration-300 ${
                      activeTab === tab.id
                        ? 'text-blue-600 dark:text-blue-400'
                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                    }`}
                  >
                    <p className={`text-xs sm:text-sm font-bold tracking-wide transition-all duration-300 ${activeTab === tab.id ? 'scale-105' : ''}`}>
                      {tab.label}
                    </p>
                    {activeTab === tab.id && (
                      <div className="absolute bottom-0 left-0 right-0 h-1 bg-gradient-to-r from-blue-500 via-blue-600 to-blue-500 rounded-t-full shadow-lg shadow-blue-500/50 dark:shadow-blue-400/50 animate-in slide-in-from-bottom-1 duration-300" />
                    )}
                  </button>
                ))}
              </div>
            </div>

            {/* Tab Content */}
            {activeTab === 'overview' && (
              <div className="flex flex-col gap-6 animate-in fade-in duration-500">
                {/* Search and Filters */}
                <div className="flex flex-col sm:flex-row gap-4 items-center">
                  <div className="w-full sm:w-auto sm:flex-1">
                    <label className="flex flex-col h-12 w-full">
                      <div className="flex w-full flex-1 items-stretch rounded-xl h-full shadow-md backdrop-blur-sm transition-shadow duration-200">
                        <div className="text-slate-300 dark:text-slate-400 flex bg-white/10 dark:bg-white/5 items-center justify-center pl-4 rounded-l-xl border border-r-0 border-white/20 dark:border-white/10">
                          <span className="text-lg">🔍</span>
                        </div>
                        <input
                          type="text"
                          className="flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-r-xl text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500/50 dark:focus:ring-blue-400/50 border border-l-0 border-white/20 dark:border-white/10 bg-white/10 dark:bg-white/5 h-full placeholder:text-slate-500 dark:placeholder:text-slate-400 px-4 text-sm font-medium transition-all duration-200"
                          placeholder="Search rules by name or description..."
                          value={searchTerm}
                          onChange={(e) => setSearchTerm(e.target.value)}
                        />
                      </div>
                    </label>
                  </div>
                  <div className="flex gap-3 flex-wrap">
                    <button className="flex h-11 shrink-0 items-center justify-center gap-x-2 rounded-xl bg-white/10 dark:bg-white/5 border border-white/20 dark:border-white/10 px-5 hover:bg-white/20 dark:hover:bg-white/10 transition-all duration-200 shadow-sm hover:shadow-md text-slate-700 dark:text-slate-300">
                      <p className="text-sm font-semibold">Filter by Severity</p>
                      <span>▼</span>
                    </button>
                    <button className="flex h-11 shrink-0 items-center justify-center gap-x-2 rounded-xl bg-white/10 dark:bg-white/5 border border-white/20 dark:border-white/10 px-5 hover:bg-white/20 dark:hover:bg-white/10 transition-all duration-200 shadow-sm hover:shadow-md text-slate-700 dark:text-slate-300">
                      <p className="text-sm font-semibold">Filter by Entity</p>
                      <span>▼</span>
                    </button>
                  </div>
                </div>

                {/* Rules Table */}
                <div className="overflow-hidden bg-white/5 dark:bg-white/5 backdrop-blur-md rounded-2xl border border-white/20 dark:border-white/10 shadow-2xl">
                  {filteredRules.length === 0 ? (
                    <div className="text-center py-20 px-6">
                      <div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-gradient-to-br from-blue-400/30 to-blue-600/30 dark:from-blue-500/20 dark:to-blue-600/20 mb-6 shadow-lg backdrop-blur-sm">
                        <span className="text-5xl">🔍</span>
                      </div>
                      <p className="text-lg font-bold text-slate-800 dark:text-slate-200 mb-2">No rules found</p>
                      <p className="text-sm text-slate-600 dark:text-slate-400">
                        Try adjusting your filters or creating a new rule.
                      </p>
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full text-left text-sm">
                        <thead className="border-b border-white/20 dark:border-white/10 bg-gradient-to-r from-blue-400/10 via-blue-500/10 to-blue-400/10 dark:from-blue-500/5 dark:via-blue-600/5 dark:to-blue-500/5 backdrop-blur-sm">
                          <tr>
                            <th className="px-6 py-4 font-bold text-xs uppercase tracking-wider text-slate-700 dark:text-slate-300" scope="col">
                              Rule Name
                            </th>
                            <th className="px-6 py-4 font-bold text-xs uppercase tracking-wider text-slate-700 dark:text-slate-300" scope="col">
                              Entity
                            </th>
                            <th className="px-6 py-4 font-bold text-xs uppercase tracking-wider text-slate-700 dark:text-slate-300" scope="col">
                              Description
                            </th>
                            <th className="px-6 py-4 font-bold text-xs uppercase tracking-wider text-slate-700 dark:text-slate-300" scope="col">
                              Severity
                            </th>
                            <th className="px-6 py-4 font-bold text-xs uppercase tracking-wider text-center text-slate-700 dark:text-slate-300" scope="col">
                              Dependencies
                            </th>
                            <th className="px-6 py-4 font-bold text-xs uppercase tracking-wider text-right text-slate-600 dark:text-slate-400" scope="col">
                              Actions
                            </th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-white/10 dark:divide-white/5">
                          {filteredRules.map((rule: ValidationRule) => (
                            <tr 
                              key={rule.id} 
                              className="group hover:bg-gradient-to-r hover:from-blue-400/10 hover:to-transparent dark:hover:from-blue-500/10 dark:hover:to-transparent transition-all duration-200 backdrop-blur-sm"
                            >
                              <td className="px-6 py-5 font-semibold text-slate-900 dark:text-slate-100 whitespace-nowrap">
                                <div className="flex items-center gap-3">
                                  <div className="w-2 h-2 rounded-full bg-gradient-to-r from-blue-400 to-blue-500 shadow-lg shadow-blue-400/50 group-hover:scale-125 transition-transform duration-200" />
                                  {rule.name}
                                </div>
                              </td>
                              <td className="px-6 py-5">
                                <span className="inline-flex items-center px-3 py-1 rounded-lg text-xs font-semibold bg-blue-400/20 dark:bg-blue-500/20 text-slate-800 dark:text-slate-200 border border-blue-400/30 dark:border-blue-500/30 backdrop-blur-sm">
                                  {rule.entity}
                                </span>
                              </td>
                              <td className="px-6 py-5 text-slate-700 dark:text-slate-300 max-w-md">
                                <p className="line-clamp-2">{rule.description}</p>
                              </td>
                              <td className="px-6 py-5">
                                <span
                                  className={`inline-flex items-center px-3 py-1.5 rounded-lg text-xs font-bold shadow-sm ${getSeverityStyles(
                                    rule.severity
                                  )}`}
                                >
                                  {(rule.severity ?? 'info').charAt(0).toUpperCase() +
                                    (rule.severity ?? 'info').slice(1)}
                                </span>
                              </td>
                              <td className="px-6 py-5 text-center">
                                <span className={`inline-flex items-center justify-center w-8 h-8 rounded-lg text-xs font-bold ${
                                  hasDependencies(rule) === 'Yes' 
                                    ? 'bg-blue-400/30 dark:bg-blue-500/30 text-slate-900 dark:text-slate-100 border border-blue-400/50 dark:border-blue-500/50 backdrop-blur-sm' 
                                    : 'bg-slate-600/20 dark:bg-slate-500/20 text-slate-500 dark:text-slate-400 border border-slate-600/30 dark:border-slate-500/20 backdrop-blur-sm'
                                }`}>
                                  {hasDependencies(rule) === 'Yes' ? '✓' : '—'}
                                </span>
                              </td>
                              <td className="px-6 py-5 text-right">
                                <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                                  <button 
                                    aria-label={`Edit rule ${rule.name}`} 
                                    className="p-2 rounded-lg hover:bg-blue-400/30 dark:hover:bg-blue-500/30 text-slate-500 hover:text-blue-600 dark:hover:text-blue-300 transition-all duration-200 hover:scale-110 backdrop-blur-sm"
                                  >
                                    <span className="text-base">✎</span>
                                  </button>
                                  <button 
                                    aria-label={`Delete rule ${rule.name}`} 
                                    className="p-2 rounded-lg hover:bg-red-400/30 dark:hover:bg-red-500/30 text-slate-500 hover:text-red-600 dark:hover:text-red-300 transition-all duration-200 hover:scale-110 backdrop-blur-sm"
                                  >
                                    <Trash2 size={16} />
                                  </button>
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Dependencies Tab */}
            {activeTab === 'dependency' && (
              <div className="animate-in fade-in duration-500">
                <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xl shadow-slate-200/50 dark:shadow-slate-950/50 overflow-hidden">
                  <div className="p-12">
                    <div className="text-center max-w-md mx-auto">
                      <div className="inline-flex items-center justify-center w-24 h-24 rounded-2xl bg-gradient-to-br from-blue-500 via-blue-600 to-purple-600 dark:from-blue-600 dark:via-blue-700 dark:to-purple-700 mb-8 shadow-2xl shadow-blue-500/30 dark:shadow-blue-600/30 animate-pulse">
                        <span className="text-6xl">🔗</span>
                      </div>
                      <h3 className="text-2xl font-black text-slate-900 dark:text-slate-100 mb-3 bg-gradient-to-r from-slate-900 via-slate-800 to-slate-900 dark:from-white dark:via-slate-100 dark:to-white bg-clip-text text-transparent">
                        Dependency Chain
                      </h3>
                      <p className="text-slate-600 dark:text-slate-400 leading-relaxed">
                        Visualize and manage complex rule dependencies with an intuitive graph interface
                      </p>
                      <button className="mt-8 px-6 py-3 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white rounded-xl font-semibold shadow-lg shadow-blue-500/30 dark:shadow-blue-600/30 transition-all duration-200 hover:scale-105">
                        Explore Dependencies
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Cross-Entity Validation Tab */}
            {activeTab === 'cross-entity' && (
              <div className="animate-in fade-in duration-500">
                <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-xl shadow-slate-200/50 dark:shadow-slate-950/50 overflow-hidden">
                  <div className="p-12">
                    <div className="text-center max-w-md mx-auto">
                      <div className="inline-flex items-center justify-center w-24 h-24 rounded-2xl bg-gradient-to-br from-purple-500 via-purple-600 to-pink-600 dark:from-purple-600 dark:via-purple-700 dark:to-pink-700 mb-8 shadow-2xl shadow-purple-500/30 dark:shadow-purple-600/30 animate-pulse">
                        <span className="text-6xl">↔️</span>
                      </div>
                      <h3 className="text-2xl font-black text-slate-900 dark:text-slate-100 mb-3 bg-gradient-to-r from-slate-900 via-slate-800 to-slate-900 dark:from-white dark:via-slate-100 dark:to-white bg-clip-text text-transparent">
                        Cross-Entity Validation
                      </h3>
                      <p className="text-slate-600 dark:text-slate-400 leading-relaxed">
                        Create powerful validation rules that span multiple entities and ensure data consistency
                      </p>
                      <button className="mt-8 px-6 py-3 bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-700 hover:to-pink-700 text-white rounded-xl font-semibold shadow-lg shadow-purple-500/30 dark:shadow-purple-600/30 transition-all duration-200 hover:scale-105">
                        Create Validation
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </main>
      </div>
    </div>
  );
};

export default AdvancedRuleConfiguration;