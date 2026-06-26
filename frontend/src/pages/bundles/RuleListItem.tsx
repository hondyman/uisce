/**
 * Rule List Item Component
 *
 * Individual rule item for the list view
 */

import type React from 'react';
import { RuleCatalogItem } from './ruleCatalogConstants';
import { getSeverityIcon } from './ruleCatalogUtils';
import styles from './RulesCatalog.module.css';

interface RuleListItemProps {
  item: RuleCatalogItem;
  isSelected: boolean;
  isSaved: boolean;
  onSelect: (ruleId: string) => void;
  onToggleSaved: (ruleId: string) => void;
}

export const RuleListItem: React.FC<RuleListItemProps> = ({
  item,
  isSelected,
  isSaved,
  onSelect,
  onToggleSaved
}) => {
  return (
    <div
      className={`${styles.listItem} ${isSelected ? styles.selected : ''}`}
      onClick={() => onSelect(item.rule.id)}
    >
      <label className={styles.checkboxLabel}>
        <input
          type="checkbox"
          checked={isSelected}
          onChange={(e) => {
            e.stopPropagation();
            onSelect(item.rule.id);
          }}
          className={styles.checkbox}
        />
        <span className={styles.visuallyHidden}>Select {item.rule.name}</span>
      </label>

      <div className={styles.itemContent}>
        <div className={styles.itemHeader}>
          <h4>{item.rule.name}</h4>
          {item.rule.isCore && <span className={styles.coreBadge}>CORE</span>}
        </div>
        <p className={styles.itemDescription}>{item.rule.description}</p>
        <div className={styles.itemFooter}>
          <span
            className={styles.severityLabel}
            data-severity={item.rule.severity}
          >
            {getSeverityIcon(item.rule.severity)} {item.rule.severity}
          </span>
          <span className={styles.frequencyLabel}>📅 {item.rule.frequency}</span>
          <span className={styles.typeLabel}>#{item.rule.evaluationOrder}</span>
          {item.categories.map(cat => (
            <span
              key={cat.id}
              className={`${styles.categoryLabel} ${styles[`category-${cat.id.replace(/_/g, '-')}`] || ''}`}
            >
              {cat.icon} {cat.name}
            </span>
          ))}
        </div>
      </div>

      <button
        className={`${styles.saveButton} ${isSaved ? styles.saved : ''}`}
        onClick={(e) => {
          e.stopPropagation();
          onToggleSaved(item.rule.id);
        }}
      >
        {isSaved ? '★' : '☆'}
      </button>
    </div>
  );
};