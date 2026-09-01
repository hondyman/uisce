/**
 * useSavedExplorerQueries — list/save/delete named explorer queries.
 *
 * Phase 1 uses localStorage as a backing store so the UI is demoable
 * without the saved-query table. The backend endpoint is still called;
 * if unavailable, we fall back silently to local persistence.
 */

import { useCallback, useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  deleteExplorerQuery,
  fetchSavedExplorerQueries,
  saveExplorerQuery,
} from '../services/dataExplorerApi';
import type { SavedExplorerQuery } from '../types/dataExplorerTypes';

const QUERY_KEY = ['data-explorer', 'saved'] as const;

export function useSavedExplorerQueries() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: QUERY_KEY,
    queryFn: fetchSavedExplorerQueries,
    staleTime: 30_000,
  });

  const save = useMutation({
    mutationFn: (input: Omit<SavedExplorerQuery, 'id' | 'createdAt' | 'updatedAt'>) =>
      saveExplorerQuery(input),
    onSuccess: (record) => {
      queryClient.setQueryData<SavedExplorerQuery[]>(QUERY_KEY, (prev) => {
        const list = prev ?? [];
        const idx = list.findIndex((q) => q.id === record.id);
        if (idx >= 0) {
          const next = [...list];
          next[idx] = record;
          return next;
        }
        return [record, ...list];
      });
    },
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteExplorerQuery(id),
    onSuccess: (_void, id) => {
      queryClient.setQueryData<SavedExplorerQuery[]>(QUERY_KEY, (prev) =>
        (prev ?? []).filter((q) => q.id !== id)
      );
    },
  });

  const sortedRecords = useMemo(() => {
    const list = query.data ?? [];
    return [...list].sort((a, b) => {
      const aTime = a.updatedAt ?? a.createdAt ?? '';
      const bTime = b.updatedAt ?? b.createdAt ?? '';
      return bTime.localeCompare(aTime);
    });
  }, [query.data]);

  const refresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: QUERY_KEY });
  }, [queryClient]);

  return {
    records: sortedRecords,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    save: save.mutateAsync,
    saveStatus: save.status,
    remove: remove.mutateAsync,
    refresh,
  };
}
