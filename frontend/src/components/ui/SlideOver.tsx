import React, { useRef } from 'react';
import ReactDOM from 'react-dom';
import styles from './SlideOver.module.css';
import { useDialog } from '../../hooks/useDialog';

type SlideOverProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  side?: 'right' | 'left';
  width?: number;
  modal?: boolean;
};

export const SlideOver: React.FC<SlideOverProps> = ({ open, onClose, title, children, side: _side = 'right', width = 520, modal = true }) => {
  const panelRef = useRef<HTMLDivElement | null>(null);

  useDialog({ open, onClose, containerRef: panelRef });

  // map numeric width to one of preset CSS classes (avoid inline styles)
  const widthClass = width <= 360 ? styles.wSm : width <= 520 ? styles.wMd : styles.wLg;

  if (!open) return null;

  return ReactDOM.createPortal(
    <div className={styles.overlay} role="presentation">
      {modal && <div className={styles.backdrop} onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }} />}
      <div
        role="dialog"
        aria-modal={modal}
        aria-labelledby="slideover-title"
        ref={panelRef}
        tabIndex={-1}
        className={`${styles.panel} ${styles.slideIn} ${widthClass}`}
      >
        <div className={styles.header}>
          <h2 id="slideover-title" className={styles.title}>{title}</h2>
          <button onClick={onClose} aria-label="Close panel" className={styles.closeBtn}>✕</button>
        </div>
        <div className={styles.body}>{children}</div>
        <div className={styles.footer}><button onClick={onClose}>Done</button></div>
      </div>
    </div>,
    document.body
  );
};

export default SlideOver;
