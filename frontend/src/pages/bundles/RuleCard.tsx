/**
 * Rule Card Component
 *
 * Individual rule card for the grid view
 */

import type React from 'react';
import { RuleCatalogItem } from './ruleCatalogConstants';
import { getSeverityIcon } from './ruleCatalogUtils';
import styles from './RulesCatalog.module.css';

interface RuleCardProps {
  item: RuleCatalogItem;
  isSelected: boolean;
  isSaved: boolean;
  onSelect: (ruleId: string) => void;
  onToggleSaved: (ruleId: string) => void;
}

export const RuleCard: React.FC<RuleCardProps> = ({
  item,
  isSelected,
  isSaved,
  onSelect,
  onToggleSaved
}) => {
  return (
    <div
      className={`${styles.card} ${isSelected ? styles.selected : ''}`}
      onClick={() => onSelect(item.rule.id)}
    >
      <div className={styles.cardHeader}>
        <div className={styles.cardTitle}>
          <h3>{item.rule.name}</h3>
          {item.rule.isCore && <span className={styles.coreBadge}>CORE</span>}
        </div>
        <button
          className={`${styles.saveButton} ${isSaved ? styles.saved : ''}`}
          onClick={(e) => {
            e.stopPropagation();
            onToggleSaved(item.rule.id);
          }}
          title={isSaved ? 'Remove from favorites' : 'Add to favorites'}
        >
          {isSaved ? '★' : '☆'}
        </button>
      </div>

      <p className={styles.description}>{item.rule.description}</p>

      <div className={styles.badges}>
        <span
          className={styles.badge}
          data-severity={item.rule.severity}
        >
          {getSeverityIcon(item.rule.severity)} {item.rule.severity}
        </span>
        {item.categories.map(cat => (
          <span
            key={cat.id}
            className={`${styles.badge} ${styles[`category-${cat.id.replace(/_/g, '-')}`] || ''}`}
          >
            {cat.icon} {cat.name.split(' & ')[0]}
          </span>
        ))}
      </div>

      <div className={styles.meta}>
        <span title="Frequency">⏱️ {item.rule.frequency}</span>
        <span title="Evaluation Order">#{item.rule.evaluationOrder}</span>
        <span title="Rule Type">⚙️ {item.rule.rule_type.replace(/_/g, ' ')}</span>
      </div>

      {item.parameters.length > 0 && (
        <div className={styles.parameters}>
          <strong>{item.parameters.length}</strong> parameters
        </div>
      )}
    </div>
  );
};