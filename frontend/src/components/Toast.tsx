import React, { createContext, useContext, useState, useCallback } from 'react';
import styles from '../pages/DynamicUIGeneratorPage.module.css';

type Toast = { id: string; message: string; type?: 'info' | 'success' | 'error' };

const ToastContext = createContext<{ showToast: (msg: string, type?: Toast['type']) => void } | null>(null);

export const useToast = () => {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
};

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback((message: string, type: Toast['type'] = 'info') => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    setToasts((t) => [...t, { id, message, type }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 3500);
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div className={styles.toastContainer} aria-live="polite">
        {toasts.map((t) => (
          <div key={t.id} className={`${styles.toast} ${t.type === 'error' ? styles.toastError : ''}`}>
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
};

export default ToastProvider;
