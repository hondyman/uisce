import type { FC } from 'react';
import { Modal } from './Modal';
import styles from './ErrorSummary.module.css';

type FieldError = { fieldId: string; label: string; message: string };

export const ErrorSummary: FC<{
  open: boolean;
  onClose: () => void;
  title?: string;
  errors: FieldError[];
  onJumpToField: (fieldId: string) => void;
}> = ({ open, onClose, title = 'Please fix the following', errors, onJumpToField }) => {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      <ul className={styles.list}>
        {errors.map(err => (
          <li key={err.fieldId} className={styles.item}>
            <button onClick={() => { onClose(); onJumpToField(err.fieldId); }} className={styles.linkBtn}>
              {err.label}: {err.message}
            </button>
          </li>
        ))}
      </ul>
    </Modal>
  );
};

export default ErrorSummary;
