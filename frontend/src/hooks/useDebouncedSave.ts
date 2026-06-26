import { useCallback, useRef, useEffect, useState } from 'react';

/**
 * Hook for debouncing save operations
 * Reduces API calls by batching multiple changes into one request
 * 
 * @example
 * const { debouncedSave, isSaving, isUnsaved } = useDebouncedSave(async (data) => {
 *   await fetch('/api/rules', { method: 'POST', body: JSON.stringify(data) });
 * }, 1000);
 * 
 * // In onChange handler:
 * debouncedSave(updatedData);
 */

export interface UseDebouncedSaveOptions {
  delay?: number; // milliseconds to wait before saving (default: 1000ms)
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}

export function useDebouncedSave<T>(
  saveFunction: (data: T) => Promise<void>,
  delay: number = 1000,
  options?: Omit<UseDebouncedSaveOptions, 'delay'>
) {
  const [isSaving, setIsSaving] = useState(false);
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [lastSaveTime, setLastSaveTime] = useState<number | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const timeoutRef = useRef<NodeJS.Timeout>();
  const pendingDataRef = useRef<T | null>(null);
  const isSavingRef = useRef(false);

  // Perform the actual save
  const performSave = useCallback(
    async (data: T) => {
      if (isSavingRef.current) return;

      isSavingRef.current = true;
      setIsSaving(true);
      setError(null);

      try {
        await saveFunction(data);
        setIsUnsaved(false);
        setLastSaveTime(Date.now());
        options?.onSuccess?.();
      } catch (err) {
        const error = err instanceof Error ? err : new Error(String(err));
        setError(error);
        options?.onError?.(error);
      } finally {
        isSavingRef.current = false;
        setIsSaving(false);
      }
    },
    [saveFunction, options]
  );

  // Debounced save function
  const debouncedSave = useCallback(
    (data: T) => {
      // Mark as unsaved immediately
      setIsUnsaved(true);
      pendingDataRef.current = data;

      // Clear existing timeout
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      // Set new timeout to save after delay
      timeoutRef.current = setTimeout(() => {
        if (pendingDataRef.current) {
          performSave(pendingDataRef.current);
        }
      }, delay);
    },
    [delay, performSave]
  );

  // Force save immediately
  const forceSave = useCallback(async () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    if (pendingDataRef.current) {
      await performSave(pendingDataRef.current);
    }
  }, [performSave]);

  // Cancel pending save
  const cancelSave = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    setIsUnsaved(false);
    pendingDataRef.current = null;
  }, []);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return {
    debouncedSave,
    forceSave,
    cancelSave,
    isSaving,
    isUnsaved,
    error,
    lastSaveTime,
    pendingData: pendingDataRef.current,
  };
}
