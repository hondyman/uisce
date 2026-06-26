import type { FC, ReactNode } from 'react';
import { Modal } from './Modal';
import styles from './Modal.module.css';

type ConfirmModalProps = {
  open: boolean;
  title?: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => Promise<void> | void;
  onClose: () => void;
};

const ConfirmModal: FC<ConfirmModalProps> = ({ open, title = 'Confirm', message, confirmLabel = 'Confirm', cancelLabel = 'Cancel', onConfirm, onClose }) => {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      <div>
        <div className={styles.body}>{message}</div>
        <div className={styles.footer}>
          <button onClick={onClose} className={styles.btnClose}>{cancelLabel}</button>
          <button onClick={async () => { await onConfirm(); }} className={styles.btnClose}>{confirmLabel}</button>
        </div>
      </div>
    </Modal>
  );
};

export default ConfirmModal;
