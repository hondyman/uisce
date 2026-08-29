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
            // /rbac/roles returns the caller's tenant roles UNIONed with every
            // gold-copy template role, so inherited roles show up here too.
            const response = await fetchAPI<any>('/rbac/roles');
            setRoles(Array.isArray(response) ? response : (response.roles || response.data || []));
        } catch (err: any) {
            setError(err.message || 'Failed to fetch roles');
        } finally {
            setLoading(false);
        }
    }, []);

    const createRole = useCallback(async (role: Partial<Role>) => {
        setLoading(true);
        try {
            await fetchAPI('/rbac/roles', {
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
            await fetchAPI(`/rbac/roles/${roleId}`, {
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
            await fetchAPI(`/rbac/roles/${roleId}`, { method: 'DELETE' });
            await fetchRoles();
        } catch (err: any) {
            setError(err.message || 'Failed to delete role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchRoles]);

    // Clones a gold-copy template role into the caller's own tenant, linked
    // via parent_role_id. No permissions are copied — they resolve through
    // the inheritance chain (see GET /rbac/roles/{id}/effective-permissions),
    // so the clone starts out identical to its source and stays in sync with
    // any core-level change until the tenant explicitly overrides something.
    const cloneRole = useCallback(async (sourceRoleId: string, roleKey?: string, roleName?: string) => {
        setLoading(true);
        try {
            await fetchAPI(`/rbac/roles/${sourceRoleId}/clone`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ role_key: roleKey, role_name: roleName }),
            });
            await fetchRoles();
        } catch (err: any) {
            setError(err.message || 'Failed to clone role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchRoles]);

    const fetchEffectivePermissions = useCallback(async (roleId: string) => {
        return fetchAPI<{ role_id: string; permissions: any[] }>(`/rbac/roles/${roleId}/effective-permissions`);
    }, []);

    return {
        roles,
        loading,
        error,
        fetchRoles,
        createRole,
        updateRole,
        deleteRole,
        cloneRole,
        fetchEffectivePermissions,
    };
};
