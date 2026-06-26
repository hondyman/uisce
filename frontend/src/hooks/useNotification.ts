/**
 * Notification Hook
 * Replaces antd's message API with notistack (already in dependencies)
 */

import { useMemo } from 'react';
import { useSnackbar } from 'notistack';

export const useNotification = () => {
  const { enqueueSnackbar, closeSnackbar } = useSnackbar();

  return useMemo(() => ({
    success: (msg: string, options?: { duration?: number; autoHideDuration?: number }) => {
      enqueueSnackbar(msg, {
        variant: 'success',
        autoHideDuration: options?.autoHideDuration || 3000,
      });
    },
    error: (msg: string, options?: { duration?: number; autoHideDuration?: number }) => {
      enqueueSnackbar(msg, {
        variant: 'error',
        autoHideDuration: options?.autoHideDuration || 4000,
      });
    },
    info: (msg: string, options?: { duration?: number; autoHideDuration?: number }) => {
      enqueueSnackbar(msg, {
        variant: 'info',
        autoHideDuration: options?.autoHideDuration || 3000,
      });
    },
    warning: (msg: string, options?: { duration?: number; autoHideDuration?: number }) => {
      enqueueSnackbar(msg, {
        variant: 'warning',
        autoHideDuration: options?.autoHideDuration || 4000,
      });
    },
    loading: (msg: string) => {
      return enqueueSnackbar(msg, {
        variant: 'info',
        persist: true,
      });
    },
    close: (key?: string | number) => {
      closeSnackbar(key);
    },
  }), [enqueueSnackbar, closeSnackbar]);
};
