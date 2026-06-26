import { useState, useCallback } from 'react';

/**
 * Hook for optimistic updates
 * Updates UI immediately, reverts on failure
 * Provides instant feedback to users
 * 
 * @example
 * const { items, addItemOptimistic, removeItemOptimistic, loading, error } = useOptimisticUpdate(
 *   initialRules,
 *   async (item) => await fetch('/api/rules', { method: 'POST', body: JSON.stringify(item) })
 * );
 * 
 * // UI updates immediately, reverts if API fails
 * await addItemOptimistic(newRule);
 */

export interface UseOptimisticUpdateOptions<T> {
  onSuccess?: (item: T, operation: 'add' | 'update' | 'remove') => void;
  onError?: (error: Error, operation: 'add' | 'update' | 'remove') => void;
}

export function useOptimisticUpdate<T extends { id: string }>(
  initialItems: T[],
  saveToServer: (item: T, operation: 'add' | 'update' | 'remove') => Promise<void>,
  options?: UseOptimisticUpdateOptions<T>
) {
  const [items, setItems] = useState(initialItems);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(false);
  const [optimisticIds, setOptimisticIds] = useState<Set<string>>(new Set());

  // Optimistic add
  const addItemOptimistic = useCallback(
    async (newItem: T) => {
      const previousItems = items;
      const tempId = `optimistic-${Date.now()}`;
      const itemToAdd = { ...newItem, id: tempId };

      // Optimistic update (immediate)
      setItems((prev) => [...prev, itemToAdd]);
      setOptimisticIds((prev) => new Set([...prev, tempId]));
      setError(null);
      setLoading(true);

      try {
        // Try to save to server
        await saveToServer(newItem, 'add');
        
        // Update with real ID if server returned different one
        setItems((prev) =>
          prev.map((item) => (item.id === tempId ? { ...newItem } : item))
        );
        setOptimisticIds((prev) => {
          const next = new Set(prev);
          next.delete(tempId);
          return next;
        });
        
        options?.onSuccess?.(newItem, 'add');
      } catch (err) {
        // Revert on failure
        setItems(previousItems);
        setOptimisticIds((prev) => {
          const next = new Set(prev);
          next.delete(tempId);
          return next;
        });

        const error = err instanceof Error ? err : new Error(String(err));
        setError(error);
        options?.onError?.(error, 'add');
      } finally {
        setLoading(false);
      }
    },
    [items, saveToServer, options]
  );

  // Optimistic update
  const updateItemOptimistic = useCallback(
    async (updatedItem: T) => {
      const previousItems = items;

      // Optimistic update (immediate)
      setItems((prev) =>
        prev.map((item) => (item.id === updatedItem.id ? updatedItem : item))
      );
      setOptimisticIds((prev) => new Set([...prev, updatedItem.id]));
      setError(null);
      setLoading(true);

      try {
        // Try to save to server
        await saveToServer(updatedItem, 'update');
        
        setOptimisticIds((prev) => {
          const next = new Set(prev);
          next.delete(updatedItem.id);
          return next;
        });
        
        options?.onSuccess?.(updatedItem, 'update');
      } catch (err) {
        // Revert on failure
        setItems(previousItems);
        setOptimisticIds((prev) => {
          const next = new Set(prev);
          next.delete(updatedItem.id);
          return next;
        });

        const error = err instanceof Error ? err : new Error(String(err));
        setError(error);
        options?.onError?.(error, 'update');
      } finally {
        setLoading(false);
      }
    },
    [items, saveToServer, options]
  );

  // Optimistic remove
  const removeItemOptimistic = useCallback(
    async (itemId: string) => {
      const previousItems = items;
      const itemToRemove = items.find((item) => item.id === itemId);

      if (!itemToRemove) return;

      // Optimistic remove (immediate)
      setItems((prev) => prev.filter((item) => item.id !== itemId));
      setOptimisticIds((prev) => new Set([...prev, itemId]));
      setError(null);
      setLoading(true);

      try {
        // Try to save to server
        await saveToServer(itemToRemove, 'remove');
        
        setOptimisticIds((prev) => {
          const next = new Set(prev);
          next.delete(itemId);
          return next;
        });
        
        options?.onSuccess?.(itemToRemove, 'remove');
      } catch (err) {
        // Revert on failure
        setItems(previousItems);
        setOptimisticIds((prev) => {
          const next = new Set(prev);
          next.delete(itemId);
          return next;
        });

        const error = err instanceof Error ? err : new Error(String(err));
        setError(error);
        options?.onError?.(error, 'remove');
      } finally {
        setLoading(false);
      }
    },
    [items, saveToServer, options]
  );

  // Check if item is being optimistically updated
  const isOptimistic = useCallback(
    (itemId: string) => optimisticIds.has(itemId),
    [optimisticIds]
  );

  return {
    items,
    addItemOptimistic,
    updateItemOptimistic,
    removeItemOptimistic,
    isOptimistic,
    loading,
    error,
    optimisticIds,
  };
}
