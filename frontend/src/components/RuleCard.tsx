/**
 * RuleCard Component
 * Displays a single validation rule with actions
 * Reusable across different rule management interfaces
 */

import React, { useMemo, useCallback } from 'react';
import { Edit2, Trash2, Eye, EyeOff } from 'lucide-react';
import {
  getRuleTypeBadgeColorClasses,
  getSeverityBadgeColorClasses,
  getStatusBadgeColorClasses,
  formatAccountTypes,
} from '../lib/ruleUtils';

interface RuleCardProps {
  rule: any;
  onEdit: (rule: any) => void;
  onDelete: (ruleId: string) => void;
  isDeleting?: boolean;
}

/**
 * RuleCard displays a single validation rule
 * Use React.memo for performance optimization
 */
const RuleCard: React.FC<RuleCardProps> = React.memo(
  ({ rule, onEdit, onDelete, isDeleting = false }) => {
    const handleEditClick = useCallback(() => {
      onEdit(rule);
    }, [rule, onEdit]);

    const handleDeleteClick = useCallback(() => {
      onDelete(rule.id);
    }, [rule.id, onDelete]);

    // Memoize badge classes to avoid recalculation
    const ruleTypeBadgeClass = useMemo(
      () => getRuleTypeBadgeColorClasses(rule.ruleType),
      [rule.ruleType]
    );

    const severityBadgeClass = useMemo(
      () => getSeverityBadgeColorClasses(rule.severity),
      [rule.severity]
    );

    const statusBadgeClass = useMemo(
      () => getStatusBadgeColorClasses(rule.isActive),
      [rule.isActive]
    );

    const accountTypesDisplay = useMemo(
      () => formatAccountTypes(rule.accountTypes || rule.scope),
      [rule.accountTypes, rule.scope]
    );

    return (
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:shadow-md transition-shadow">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            {/* Rule Title and Badges */}
            <div className="flex items-center gap-3 flex-wrap">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{rule.name}</h3>

              <span className={`px-3 py-1 rounded-full text-sm font-medium ${ruleTypeBadgeClass}`}>
                {rule.ruleType}
              </span>

              <span className={`px-3 py-1 rounded-full text-sm font-medium ${severityBadgeClass}`}>
                {rule.severity}
              </span>

              <span className={`flex items-center gap-1 px-3 py-1 rounded-full text-sm ${statusBadgeClass}`}>
                {rule.isActive ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                {rule.isActive ? 'Active' : 'Inactive'}
              </span>
            </div>

            {/* Description */}
            {rule.description && (
              <p className="text-gray-600 dark:text-gray-400 mt-2 text-sm">{rule.description}</p>
            )}

            {/* Metadata */}
            <div className="flex items-center gap-4 mt-3 text-sm text-gray-500 dark:text-gray-400">
              <span className="truncate">Account Types: {accountTypesDisplay}</span>
              <span>Order: {rule.evaluationOrder}</span>
              {rule.allowOverride && (
                <span className="text-blue-600 dark:text-blue-400 font-medium">✓ Overridable</span>
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-2 ml-4 flex-shrink-0">
            <button
              onClick={handleEditClick}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              aria-label="Edit rule"
              title="Edit rule"
              disabled={isDeleting}
            >
              <Edit2 className="w-5 h-5 text-gray-600 dark:text-gray-400" />
            </button>
            <button
              onClick={handleDeleteClick}
              className="p-2 hover:bg-red-100 dark:hover:bg-red-900/20 rounded-lg transition-colors disabled:opacity-50"
              aria-label="Delete rule"
              title="Delete rule"
              disabled={isDeleting}
            >
              <Trash2 className="w-5 h-5 text-red-600 dark:text-red-400" />
            </button>
          </div>
        </div>
      </div>
    );
  },
  (prevProps, nextProps) => {
    // Custom comparison for memo
    // Re-render only if rule data changed or handlers changed
    return (
      JSON.stringify(prevProps.rule) === JSON.stringify(nextProps.rule) &&
      prevProps.isDeleting === nextProps.isDeleting
    );
  }
);

RuleCard.displayName = 'RuleCard';

export default RuleCard;
