import React, { createContext, useContext, useState, ReactNode } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@mui/material';

interface ConfirmOptions {
  title?: string;
  description?: string;
  confirmText?: string;
  cancelText?: string;
}

type ConfirmFn = (opts: ConfirmOptions) => Promise<boolean>;

const ConfirmContext = createContext<ConfirmFn | null>(null);

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [opts, setOpts] = useState<ConfirmOptions>({});
  const [resolver, setResolver] = useState<(value: boolean) => void>(() => () => {});

  const showConfirm: ConfirmFn = (o) => {
    setOpts({ ...o });
    setOpen(true);
    return new Promise((resolve) => {
      setResolver(() => resolve);
    });
  };

  const accept = () => { setOpen(false); resolver(true); };
  const cancel = () => { setOpen(false); resolver(false); };

  return (
    <ConfirmContext.Provider value={showConfirm}>
      {children}
      <Dialog open={open} onClose={cancel} aria-labelledby="confirm-title">
        <DialogTitle id="confirm-title">{opts.title || 'Confirm'}</DialogTitle>
        <DialogContent>{opts.description || 'Are you sure?'}</DialogContent>
        <DialogActions>
          <Button onClick={cancel}>{opts.cancelText || 'Cancel'}</Button>
          <Button onClick={accept} color="error" variant="contained">{opts.confirmText || 'Confirm'}</Button>
        </DialogActions>
      </Dialog>
    </ConfirmContext.Provider>
  );
}

export function useConfirm() {
  const ctx = useContext(ConfirmContext);
  if (!ctx) throw new Error('useConfirm must be used inside ConfirmProvider');
  return ctx;
}
