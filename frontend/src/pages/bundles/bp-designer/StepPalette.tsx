/**
 * StepPalette.tsx
 * Draggable palette of step types that can be dropped onto the canvas
 */

import type React from 'react';
import { useStepTypes } from './useBPDesignerAPI';
import styles from './BPDesigner.module.css';

interface StepPaletteProps {
  onDragStart: (e: React.DragEvent, stepType: any) => void;
}

export const StepPalette: React.FC<StepPaletteProps> = ({ onDragStart }) => {
  const { data: stepTypes = [], isLoading, error } = useStepTypes();

  if (isLoading) {
    return (
      <aside className={styles.sidebar}>
        <div className={styles.loading}>Loading step types...</div>
      </aside>
    );
  }

  if (error) {
    return (
      <aside className={styles.sidebar}>
        <div className={styles.error}>Failed to load step types</div>
      </aside>
    );
  }

  return (
    <aside className={styles.sidebar}>
      <div className={styles.sidebarContent}>
        <div className={styles.sidebarHeader}>
          <h2>Step Palette</h2>
          <p>Drag steps onto the canvas</p>
        </div>

        <div className={styles.stepList}>
          {stepTypes.map((stepType: any) => (
            <div
              key={stepType.id}
              className={styles.stepItem}
              draggable
              onDragStart={(e) => onDragStart(e, stepType)}
              title={stepType.description}
            >
              {stepType.icon_svg ? (
                <div
                  className={styles.stepIcon}
                  dangerouslySetInnerHTML={{ __html: stepType.icon_svg }}
                />
              ) : (
                <span className={styles.stepIconPlaceholder}>📦</span>
              )}
              <p className={styles.stepLabel}>{stepType.label}</p>
            </div>
          ))}
        </div>
      </div>
    </aside>
  );
};
