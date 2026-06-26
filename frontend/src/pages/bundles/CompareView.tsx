/**
 * Compare View Component
 *
 * Side-by-side comparison of selected rules
 */

import type React from 'react';
import { RuleCatalogItem } from './ruleCatalogConstants';
import styles from './RulesCatalog.module.css';

interface CompareViewProps {
  selectedItems: RuleCatalogItem[];
}

export const CompareView: React.FC<CompareViewProps> = ({ selectedItems }) => {
  if (selectedItems.length < 2) {
    return null;
  }

  const asRecord = (v: unknown): Record<string, unknown> => (v && typeof v === 'object') ? (v as Record<string, unknown>) : {};

  return (
    <div className={styles.compareView}>
      <div className={styles.compareTable}>
        <div className={styles.compareHeader}>
          <div className={styles.compareColumn}>Property</div>
          {selectedItems.map(item => (
            <div key={item.rule.id} className={styles.compareColumn}>
              {item.rule.name}
            </div>
          ))}
        </div>

        {['severity', 'frequency', 'evaluationOrder', 'rule_type', 'scope'].map(prop => (
          <div key={prop} className={styles.compareRow}>
            <div className={styles.compareColumn}>{prop.replace(/_/g, ' ')}</div>
            {selectedItems.map(item => (
              <div key={item.rule.id} className={styles.compareColumn}>
                {String(asRecord(item.rule)[prop])}
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
};