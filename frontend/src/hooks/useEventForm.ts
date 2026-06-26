import { useState, useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { InternalEvent } from '../types/calendar';

const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081/api/v1';

export const useEventForm = (tenantId: string) => {
    const queryClient = useQueryClient();
    const [error, setError] = useState<string | null>(null);

    const createMutation = useMutation({
        mutationFn: async (event: Omit<InternalEvent, 'id' | 'tenant_id'>) => {
            const res = await fetch(`${API_BASE}/events`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ...event, tenant_id: tenantId }),
            });
            if (!res.ok) throw new Error('Failed to create event');
            return res.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['syncedEvents'] });
        },
        onError: (err: Error) => setError(err.message),
    });

    const updateMutation = useMutation({
        mutationFn: async (event: InternalEvent) => {
            const res = await fetch(`${API_BASE}/events/${event.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(event),
            });
            if (!res.ok) throw new Error('Failed to update event');
            return res.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['syncedEvents'] });
        },
        onError: (err: Error) => setError(err.message),
    });

    const deleteMutation = useMutation({
        mutationFn: async (eventId: string) => {
            const res = await fetch(`${API_BASE}/events/${eventId}`, {
                method: 'DELETE',
            });
            if (!res.ok) throw new Error('Failed to delete event');
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['syncedEvents'] });
        },
        onError: (err: Error) => setError(err.message),
    });

    const handleCreate = useCallback((event: Omit<InternalEvent, 'id' | 'tenant_id'>) => {
        createMutation.mutate(event);
    }, [createMutation]);

    const handleUpdate = useCallback((event: InternalEvent) => {
        updateMutation.mutate(event);
    }, [updateMutation]);

    const handleDelete = useCallback((eventId: string) => {
        deleteMutation.mutate(eventId);
    }, [deleteMutation]);

    return {
        handleCreate,
        handleUpdate,
        handleDelete,
        isCreating: createMutation.isPending,
        isUpdating: updateMutation.isPending,
        isDeleting: deleteMutation.isPending,
        error,
        clearError: () => setError(null)
    };
};
