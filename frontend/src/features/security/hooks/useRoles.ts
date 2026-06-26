import { useState, useCallback } from 'react';
import { Role } from '../types/security';
import { fetchAPI } from '../../../api';

export const useRoles = () => {
    const [roles, setRoles] = useState<Role[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchRoles = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            // Assuming GET /api/roles returns Role[] directly or { roles: Role[] }
            const response = await fetchAPI<any>('/roles');
            // Adjust based on actual API response structure
            const data = response.permissions ? response.permissions : response;
            // The current backend /api/roles handler returns a list of roles.
            // Let's assume generic api client returns the JSON payload.
            // If the backend returns `null` or empty for no roles, handle it.
            setRoles(Array.isArray(response) ? response : (response.roles || []));
        } catch (err: any) {
            setError(err.message || 'Failed to fetch roles');
        } finally {
            setLoading(false);
        }
    }, []);

    const createRole = useCallback(async (role: Partial<Role>) => {
        setLoading(true);
        try {
            await fetchAPI('/roles', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(role),
            });
            await fetchRoles(); // Refresh list
        } catch (err: any) {
            setError(err.message || 'Failed to create role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchRoles]);

    const updateRole = useCallback(async (roleId: string, updates: Partial<Role>) => {
        setLoading(true);
        try {
            await fetchAPI(`/roles/${roleId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(updates),
            });
            await fetchRoles();
        } catch (err: any) {
            setError(err.message || 'Failed to update role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchRoles]);

    const deleteRole = useCallback(async (roleId: string) => {
        setLoading(true);
        try {
            await fetchAPI(`/roles/${roleId}`, { method: 'DELETE' });
            await fetchRoles();
        } catch (err: any) {
            setError(err.message || 'Failed to delete role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchRoles]);

    return {
        roles,
        loading,
        error,
        fetchRoles,
        createRole,
        updateRole,
        deleteRole,
    };
};
