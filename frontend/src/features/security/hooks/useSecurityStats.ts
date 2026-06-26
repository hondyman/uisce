import { useState, useCallback } from 'react';
import { fetchAPI } from '../../../api';
import { SecurityStats } from '../types/security';

export const useSecurityStats = () => {
    const [stats, setStats] = useState<SecurityStats | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchStats = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            // GET /api/audit/stats
            const response = await fetchAPI<SecurityStats>('/audit/stats');
            setStats(response);
        } catch (err: any) {
            setError(err.message || 'Failed to fetch security stats');
        } finally {
            setLoading(false);
        }
    }, []);

    return {
        stats,
        loading,
        error,
        fetchStats,
    };
};
