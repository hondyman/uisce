import { useState, useCallback } from 'react';
import { fetchAPI } from '../../../api';
import { User, Role } from '../types/security';

export const useUsers = () => {
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchUsers = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            // GET /api/users
            const response = await fetchAPI<User[]>('/users');
            setUsers(response || []);
        } catch (err: any) {
            setError(err.message || 'Failed to fetch users');
        } finally {
            setLoading(false);
        }
    }, []);

    const fetchUserRoles = useCallback(async (userId: string): Promise<Role[]> => {
        setLoading(true);
        try {
            // GET /api/users/{user_id}/roles
            const response = await fetchAPI<Role[]>(`/users/${userId}/roles`);
            return response || [];
        } catch (err: any) {
            setError(err.message || 'Failed to fetch user roles');
            return [];
        } finally {
            setLoading(false);
        }
    }, []);

    const assignRole = useCallback(async (userId: string, roleId: string) => {
        setLoading(true);
        try {
            // Backend endpoint: POST /api/users/{user_id}/roles
            // Body: { role_id: roleId }
            await fetchAPI(`/users/${userId}/roles`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ role_id: roleId }),
            });
            await fetchUsers();
        } catch (err: any) {
            setError(err.message || 'Failed to assign role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchUsers]);

    const revokeRole = useCallback(async (userId: string, roleId: string) => {
        setLoading(true);
        try {
            // Backend endpoint: DELETE /api/users/{user_id}/roles/{role_id}
            await fetchAPI(`/users/${userId}/roles/${roleId}`, {
                method: 'DELETE',
            });
            await fetchUsers();
        } catch (err: any) {
            setError(err.message || 'Failed to revoke role');
            throw err;
        } finally {
            setLoading(false);
        }
    }, [fetchUsers]);

    return {
        users,
        loading,
        error,
        fetchUsers,
        fetchUserRoles,
        assignRole,
        revokeRole,
    };
};
