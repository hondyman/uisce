import React, { useEffect as _useEffect, useRef } from 'react';
import ReactDOM from 'react-dom';
import styles from './SlideOver.module.css'; // Reusing styles for consistency
import { useDialog } from '../../hooks/useDialog';

type ModalProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  initialFocusRef?: React.RefObject<HTMLElement>;
};

export const Modal: React.FC<ModalProps> = ({ open, onClose, title, children, initialFocusRef }) => {
  const panelRef = useRef<HTMLDivElement | null>(null);

  useDialog({ open, onClose, initialFocusRef, containerRef: panelRef });

  if (!open) return null;

  return ReactDOM.createPortal(
    <div className={styles.overlay} role="presentation">
      <div className={styles.backdrop} onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }} />
      <div role="dialog" aria-modal="true" aria-labelledby="modal-title" ref={panelRef} tabIndex={-1} className={styles.panel} style={{ width: 520, margin: 'auto' }}>
        <div className={styles.header}>
          <h2 id="modal-title" className={styles.title}>{title}</h2>
          <button onClick={onClose} aria-label="Close panel" className={styles.closeBtn}>✕</button>
        </div>
        <div className={styles.body}>{children}</div>
        <div className={styles.footer}><button onClick={onClose}>Done</button></div>
      </div>
    </div>,
    document.body
  );
};