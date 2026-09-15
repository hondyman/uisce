/**
 * StepPalette.tsx
 * Draggable palette of step types that can be dropped onto the canvas
 */

import type React from 'react';
import { useDraggable } from '@dnd-kit/core';
import { useStepTypes } from './useBPDesignerAPI';
import styles from './BPDesigner.module.css';

interface StepPaletteProps {
  onDragStart?: (e: React.DragEvent, stepType: any) => void;
}

interface DraggableStepItemProps {
  stepType: any;
}

const DraggableStepItem: React.FC<DraggableStepItemProps> = ({ stepType }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `step-palette-${stepType.id}`,
    data: { stepType },
  });

  return (
    <div
      ref={setNodeRef}
      className={styles.stepItem}
      title={stepType.description}
      style={{ opacity: isDragging ? 0.4 : 1 }}
      {...attributes}
      {...listeners}
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
  );
};

export const StepPalette: React.FC<StepPaletteProps> = () => {
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
            <DraggableStepItem key={stepType.id} stepType={stepType} />
          ))}
        </div>
      </div>
    </aside>
  );
};
