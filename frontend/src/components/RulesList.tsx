/**
 * RulesList Component
 * Displays a filterable and searchable list of validation rules
 */

import React, { useMemo, useCallback, useState } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { AlertCircle } from 'lucide-react';
import RuleCard from './RuleCard';

interface RulesListProps {
  rules: any[];
  loading: boolean;
  onEdit: (rule: any) => void;
  onDelete: (ruleId: string) => void;
  onCreateNew: () => void;
  filterType?: string;
  searchTerm?: string;
  sortBy?: 'name' | 'type' | 'severity' | 'order';
}

/**
 * RulesList displays validation rules with optional filtering/sorting
 */
const RulesList: React.FC<RulesListProps> = ({
  rules,
  loading,
  onEdit,
  onDelete,
  onCreateNew,
  filterType,
  searchTerm = '',
  sortBy = 'name',
}) => {
  const [deletingRuleId, setDeletingRuleId] = useState<string | null>(null);

  // Memoize filtered and sorted rules
  const displayedRules = useMemo(() => {
    let filtered = [...rules];

    // Apply search filter
    if (searchTerm) {
      const lowerSearch = searchTerm.toLowerCase();
      filtered = filtered.filter(
        (rule) =>
          rule.name.toLowerCase().includes(lowerSearch) ||
          rule.description?.toLowerCase().includes(lowerSearch) ||
          rule.ruleType.toLowerCase().includes(lowerSearch)
      );
    }

    // Apply type filter
    if (filterType && filterType !== 'ALL') {
      filtered = filtered.filter((rule) => rule.ruleType === filterType);
    }

    // Apply sorting
    filtered.sort((a, b) => {
      switch (sortBy) {
        case 'type':
          return a.ruleType.localeCompare(b.ruleType);
        case 'severity':
          {
            const severityOrder: Record<string, number> = { BLOCK: 0, WARNING: 1, INFO: 2 };
            return (severityOrder[a.severity] || 99) - (severityOrder[b.severity] || 99);
          }
        case 'order':
          return (a.evaluationOrder || 0) - (b.evaluationOrder || 0);
        case 'name':
        default:
          return a.name.localeCompare(b.name);
      }
    });

    return filtered;
  }, [rules, searchTerm, filterType, sortBy]);

  const handleDelete = useCallback(
    async (ruleId: string) => {
      const confirm = useConfirm();
      const notification = useNotification();
      if (!(await confirm({ title: 'Delete rule', description: 'Are you sure you want to delete this rule?' }))) return;
      setDeletingRuleId(ruleId);
      try {
        await onDelete(ruleId);
        notification.success('Rule deleted');
      } finally {
        setDeletingRuleId(null);
      }
    },
    [onDelete]
  );

  if (loading) {
    return (
      <div className="text-center py-12">
        <div className="animate-spin inline-block w-8 h-8 border-4 border-gray-300 dark:border-gray-600 border-t-blue-600 rounded-full"></div>
        <p className="text-gray-600 dark:text-gray-400 mt-4">Loading rules...</p>
      </div>
    );
  }

  if (rules.length === 0) {
    return (
      <div className="text-center py-12 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg">
        <AlertCircle className="w-12 h-12 text-gray-400 dark:text-gray-500 mx-auto mb-4" />
        <p className="text-gray-600 dark:text-gray-400 mb-4">No validation rules yet</p>
        <button
          onClick={onCreateNew}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
        >
          Create First Rule
        </button>
      </div>
    );
  }

  if (displayedRules.length === 0) {
    return (
      <div className="text-center py-12 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg">
        <AlertCircle className="w-12 h-12 text-gray-400 dark:text-gray-500 mx-auto mb-4" />
        <p className="text-gray-600 dark:text-gray-400 mb-4">No rules match your filters</p>
        <p className="text-sm text-gray-500 dark:text-gray-500">
          {searchTerm && `Search: "${searchTerm}"`}
          {filterType && filterType !== 'ALL' && ` • Type: ${filterType}`}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="text-sm text-gray-600 dark:text-gray-400">
        Showing {displayedRules.length} of {rules.length} rules
      </div>

      <div className="grid gap-4">
        {displayedRules.map((rule) => (
          <RuleCard
            key={rule.id}
            rule={rule}
            onEdit={onEdit}
            onDelete={handleDelete}
            isDeleting={deletingRuleId === rule.id}
          />
        ))}
      </div>
    </div>
  );
};

export default RulesList;
