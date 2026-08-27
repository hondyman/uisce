import { useState, useCallback } from 'react';
import { fetchAPI } from '../../../api';

export interface ShareableUser {
  id: string;
  name: string;
  email: string;
  role: string;
  organization: string;
  access_path: 'direct' | 'entitlement';
  is_active: boolean;
  tenant_id?: string;
}

export const useShareableUsers = (tenantId?: string) => {
  const [users, setUsers] = useState<ShareableUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    try {
      const response = await fetchAPI<ShareableUser[]>(
        `/api/v1/users/shareable?tenant_id=${encodeURIComponent(tenantId)}`
      );
      setUsers(response || []);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch shareable users');
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  return { users, loading, error, fetchUsers };
};
