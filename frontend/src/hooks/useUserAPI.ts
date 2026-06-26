import { useCallback } from 'react';
import { getEnv } from '../utils/getEnv';
import { useAuthFetch } from '../utils/authFetch';

export interface UserPreferences {
  language?: string;
}

export const useUserAPI = () => {
  const { authFetch } = useAuthFetch();
  const API_BASE_URL = getEnv('', 'VITE_API_BASE_URL', 'http://localhost:29080') as string;

  const getUserPreferences = useCallback(async (userId: string): Promise<UserPreferences> => {
    const resp = await authFetch<UserPreferences>(`${API_BASE_URL}/api/users/${userId}/preferences`, { method: 'GET' });
    if (!resp.ok) throw new Error(`Failed to fetch preferences: ${resp.status}`);
    return resp.data as UserPreferences;
  }, [authFetch]);

  const updateUserPreferences = useCallback(async (userId: string, prefs: UserPreferences): Promise<UserPreferences> => {
    const resp = await authFetch<UserPreferences>(`${API_BASE_URL}/api/users/${userId}/preferences`, { method: 'PUT', json: prefs });
    if (!resp.ok) throw new Error(`Failed to update preferences: ${resp.status}`);
    return resp.data as UserPreferences;
  }, [authFetch]);

  return { getUserPreferences, updateUserPreferences };
};

export default useUserAPI;
