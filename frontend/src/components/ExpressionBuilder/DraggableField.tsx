import type { FC } from 'react';
import { useDraggable } from '@dnd-kit/core';
import styles from './ExpressionBuilder.module.css';

type Props = { id: string; data: { field: string; type?: string } };

const DraggableField: FC<Props> = ({ id, data }) => {
  const { attributes, listeners, setNodeRef, transform: _transform } = useDraggable({ id, data: { field: data.field } });

  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className={styles.fieldItem}
      tabIndex={0}
      role="button"
      aria-label={`Drag field ${data.field}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
        }
      }}
    >
      {data.field}
    </div>
  );
};

export default DraggableField;
