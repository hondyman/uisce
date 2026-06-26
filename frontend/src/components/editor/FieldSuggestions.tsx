// frontend/src/components/editor/FieldSuggestions.tsx
import React, { useState, useEffect } from 'react';
import styles from './FieldSuggestions.module.css';
import { devError } from '../../utils/devLogger';

interface FieldRec {
  fieldId: string;
  fieldLabel: string;
  usageScore: number;
  reason: string;
}

export const FieldSuggestions: React.FC<{
  primaryBO: string;
  tenantId: string;
  existingFieldIds: string[];
  onAddFields: (fieldIds: string[]) => void;
}> = ({ primaryBO, tenantId, existingFieldIds = [], onAddFields }) => {
  const [recommendations, setRecommendations] = useState<FieldRec[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    if (expanded && recommendations.length === 0) {
      fetchRecommendations();
    }
  }, [expanded]);

  const fetchRecommendations = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/ai/field-recommendations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          primaryBO,
          sectionContext: { type: 'fields' },
          existingFieldIds,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        setRecommendations(data.recommendations || []);
      }
    } catch (err) {
      devError('Error fetching recommendations:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    if (selected.size > 0) {
      onAddFields(Array.from(selected));
      setSelected(new Set());
      setExpanded(false);
    }
  };

  const toggleSelect = (fieldId: string) => {
    const newSelected = new Set(selected);
    if (newSelected.has(fieldId)) {
      newSelected.delete(fieldId);
    } else {
      newSelected.add(fieldId);
    }
    setSelected(newSelected);
  };

  if (recommendations.length === 0 && !expanded) {
    return (
      <button
        className={styles.triggerBtn}
        onClick={() => setExpanded(true)}
      >
        💡 Suggest Fields
      </button>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <span className={styles.title}>Suggested Fields</span>
        <button
          className={styles.closeBtn}
          onClick={() => {
            setExpanded(false);
            setSelected(new Set());
          }}
        >
          ✕
        </button>
      </div>

      {loading ? (
        <div className={styles.loading}>Loading recommendations…</div>
      ) : recommendations.length === 0 ? (
        <div className={styles.empty}>No field recommendations available.</div>
      ) : (
        <>
          <div className={styles.listContainer}>
            {recommendations.map((rec) => (
              <div key={rec.fieldId} className={styles.item}>
                <input
                  type="checkbox"
                  id={`field-${rec.fieldId}`}
                  checked={selected.has(rec.fieldId)}
                  onChange={() => toggleSelect(rec.fieldId)}
                  className={styles.checkbox}
                />
                <label htmlFor={`field-${rec.fieldId}`} className={styles.label}>
                  <div className={styles.fieldName}>{rec.fieldLabel}</div>
                  <div className={styles.score}>Score: {(rec.usageScore * 100).toFixed(0)}%</div>
                  <div className={styles.reason}>{rec.reason}</div>
                </label>
              </div>
            ))}
          </div>

          <div className={styles.actions}>
            <button
              className={styles.cancelBtn}
              onClick={() => {
                setExpanded(false);
                setSelected(new Set());
              }}
            >
              Cancel
            </button>
            <button
              className={styles.addBtn}
              disabled={selected.size === 0}
              onClick={handleAdd}
            >
              Add {selected.size > 0 ? `(${selected.size})` : ''} Field{selected.size !== 1 ? 's' : ''}
            </button>
          </div>
        </>
      )}
    </div>
  );
};
