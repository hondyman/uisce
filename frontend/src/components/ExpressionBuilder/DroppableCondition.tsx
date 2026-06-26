import type React from 'react';
import { useDroppable } from '@dnd-kit/core';
import OperatorSelector from './OperatorSelector';
import ValueInput from './ValueInput';
import { Select, MenuItem } from '@mui/material';
import ActionButton from '../ui/ActionButton';
import styles from './ExpressionBuilder.module.css';

type Props = {
  id: string;
  condition: any;
  index: number;
  onUpdate: (id: number, updates: any) => void;
  onLogicUpdate: (id: number, logic: string) => void;
};

const DroppableCondition: React.FC<Props> = ({ id, condition, index, onUpdate, onLogicUpdate }) => {
  const { isOver, setNodeRef } = useDroppable({ id });

  const removeCondition = () => {
    // parent updates conditions by setting field to null or empty — parent will implement full remove
    onUpdate(condition.id, { _remove: true });
  };

  return (
    <div ref={setNodeRef} className={`${styles.conditionBox} ${isOver ? styles.conditionOver : ''}`} role="group" tabIndex={0} onKeyDown={(e) => { if ((e.key === 'Delete' || e.key === 'Backspace')) removeCondition(); }}>
      {index > 0 && (
        <Select
          value={condition.logic}
          onChange={(e) => onLogicUpdate(condition.id, e.target.value)}
          sx={{ width: 80 }}
          size="small"
        >
          <MenuItem value="and">AND</MenuItem>
          <MenuItem value="or">OR</MenuItem>
        </Select>
      )}

      <span className={styles.fieldLabel} style={{ fontWeight: condition.field ? 'bold' : 'normal' }}>
        {condition.field || 'Drop field...'}
      </span>

      <OperatorSelector value={condition.operator} onChange={op => onUpdate(condition.id, { operator: op })} />

      <ValueInput value={condition.value} field={condition.field} onChange={val => onUpdate(condition.id, { value: val })} />

      <ActionButton size="sm" variant="danger" iconName="close" iconOnly onClick={removeCondition} className={styles.removeBtn} aria-label="Remove condition" />
    </div>
  );
};

export default DroppableCondition;
