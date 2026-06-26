import { useCallback, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081/api/v1';

export interface SyncConflict {
    id: string;
    tenant_id: string;
    internal_event_id: string;
    google_event_id: string;
    conflict_type: string;
    severity: string;
    status: string;
    resolution_strategy: string | null;
    created_at: string;
    updated_at: string;
    internal_event_data?: any;
    google_event_data?: any;
}

export const useConflictResolution = (tenantId: string) => {
    const queryClient = useQueryClient();
    const [error, setError] = useState<string | null>(null);

    const { data: conflicts, isLoading } = useQuery<SyncConflict[]>({
        queryKey: ['syncConflicts', tenantId],
        queryFn: async () => {
            const res = await fetch(`${API_BASE}/sync/conflicts?tenant_id=${tenantId}&status=pending`);
            if (!res.ok) throw new Error('Failed to fetch conflicts');
            return res.json();
        },
        enabled: !!tenantId,
    });

    const { data: stats } = useQuery<any>({
        queryKey: ['syncConflictStats', tenantId],
        queryFn: async () => {
            const res = await fetch(`${API_BASE}/sync/conflicts/stats?tenant_id=${tenantId}`);
            if (!res.ok) throw new Error('Failed to fetch conflict stats');
            return res.json();
        },
        enabled: !!tenantId,
        refetchInterval: 10000,
    });

    const resolveMutation = useMutation({
        mutationFn: async ({ conflictId, strategy }: { conflictId: string, strategy: string }) => {
            const res = await fetch(`${API_BASE}/sync/conflicts/${conflictId}/resolve`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ strategy }),
            });
            if (!res.ok) throw new Error('Failed to resolve conflict');
            return res.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['syncConflicts'] });
            queryClient.invalidateQueries({ queryKey: ['syncConflictStats'] });
            queryClient.invalidateQueries({ queryKey: ['syncedEvents'] });
        },
        onError: (err: Error) => setError(err.message),
    });

    const autoResolveMutation = useMutation({
        mutationFn: async (severityLevels: string[]) => {
            const res = await fetch(`${API_BASE}/sync/conflicts/auto-resolve`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ tenant_id: tenantId, severity: severityLevels }),
            });
            if (!res.ok) throw new Error('Failed to auto-resolve conflicts');
            return res.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['syncConflicts'] });
            queryClient.invalidateQueries({ queryKey: ['syncConflictStats'] });
            queryClient.invalidateQueries({ queryKey: ['syncedEvents'] });
        },
        onError: (err: Error) => setError(err.message),
    });

    const handleResolve = useCallback((conflictId: string, strategy: string) => {
        resolveMutation.mutate({ conflictId, strategy });
    }, [resolveMutation]);

    const handleAutoResolve = useCallback((severityLevels: string[] = ['info', 'warning']) => {
        autoResolveMutation.mutate(severityLevels);
    }, [autoResolveMutation]);

    return {
        conflicts,
        isLoading,
        stats,
        handleResolve,
        handleAutoResolve,
        isResolving: resolveMutation.isPending,
        isAutoResolving: autoResolveMutation.isPending,
        error,
        clearError: () => setError(null),
    };
};
