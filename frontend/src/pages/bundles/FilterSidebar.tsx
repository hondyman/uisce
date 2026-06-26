/**
 * Filter Sidebar Component
 *
 * Sidebar containing all filter options
 */

import type React from 'react';
import { RULE_CATEGORIES } from './ruleCatalogConstants';
import { getSeverityIcon } from './ruleCatalogUtils';
import WEALTH_VALIDATION_RULES from '@/data/wealthValidationRules';
import styles from './RulesCatalog.module.css';

interface FilterSidebarProps {
  filters: {
    categories: string[];
    severities: string[];
    frequencies: string[];
    ruleTypes: string[];
  };
  uniqueValues: {
    uniqueSeverities: string[];
    uniqueFrequencies: string[];
    uniqueRuleTypes: string[];
  };
  onToggleCategory: (categoryId: string) => void;
  onToggleSeverity: (severity: string) => void;
  onToggleFrequency: (frequency: string) => void;
  onToggleRuleType: (ruleType: string) => void;
  onClearFilters: () => void;
}

export const FilterSidebar: React.FC<FilterSidebarProps> = ({
  filters,
  uniqueValues,
  onToggleCategory,
  onToggleSeverity,
  onToggleFrequency,
  onToggleRuleType,
  onClearFilters
}) => {
  return (
    <aside className={styles.sidebar}>
      <div className={styles.filterSection}>
        <h3>Categories</h3>
        <div className={styles.filterGroup}>
          {RULE_CATEGORIES.map(category => (
            <label key={category.id} className={styles.filterCheckbox}>
              <input
                type="checkbox"
                checked={filters.categories.includes(category.id)}
                onChange={() => onToggleCategory(category.id)}
              />
              <span className={`${styles.categoryBadge} ${styles[`category-${category.id.replace(/_/g, '-')}`] || ''}`}>
                {category.icon}
              </span>
              <span>{category.name}</span>
              <span className={styles.count}>
                ({category.ruleIds.length})
              </span>
            </label>
          ))}
        </div>
      </div>

      <div className={styles.filterSection}>
        <h3>Severity</h3>
        <div className={styles.filterGroup}>
          {uniqueValues.uniqueSeverities.map(severity => (
            <label key={severity} className={styles.filterCheckbox}>
              <input
                type="checkbox"
                checked={filters.severities.includes(severity)}
                onChange={() => onToggleSeverity(severity)}
              />
              <span
                className={styles.severityBadge}
                data-severity={severity}
              >
                {getSeverityIcon(severity)}
              </span>
              <span>{severity}</span>
              <span className={styles.count}>
                ({WEALTH_VALIDATION_RULES.filter(r => r.severity === severity).length})
              </span>
            </label>
          ))}
        </div>
      </div>

      <div className={styles.filterSection}>
        <h3>Frequency</h3>
        <div className={styles.filterGroup}>
          {uniqueValues.uniqueFrequencies.map(frequency => (
            <label key={frequency} className={styles.filterCheckbox}>
              <input
                type="checkbox"
                checked={filters.frequencies.includes(frequency)}
                onChange={() => onToggleFrequency(frequency)}
              />
              <span>{frequency}</span>
              <span className={styles.count}>
                ({WEALTH_VALIDATION_RULES.filter(r => r.frequency === frequency).length})
              </span>
            </label>
          ))}
        </div>
      </div>

      <div className={styles.filterSection}>
        <h3>Rule Type</h3>
        <div className={styles.filterGroup}>
          {uniqueValues.uniqueRuleTypes.map(ruleType => (
            <label key={ruleType} className={styles.filterCheckbox}>
              <input
                type="checkbox"
                checked={filters.ruleTypes.includes(ruleType)}
                onChange={() => onToggleRuleType(ruleType)}
              />
              <span>{ruleType.replace(/_/g, ' ')}</span>
              <span className={styles.count}>
                ({WEALTH_VALIDATION_RULES.filter(r => r.rule_type === ruleType).length})
              </span>
            </label>
          ))}
        </div>
      </div>

      <button
        className={styles.clearFilters}
        onClick={onClearFilters}
      >
        Clear All Filters
      </button>
    </aside>
  );
};