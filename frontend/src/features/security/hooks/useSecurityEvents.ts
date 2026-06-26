import { useState, useCallback } from 'react';
import { fetchAPI } from '../../../api';
import { AuditEvent } from '../types/security';

export const useSecurityEvents = () => {
    const [events, setEvents] = useState<AuditEvent[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchEvents = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            // GET /api/audit/events
            const response = await fetchAPI<AuditEvent[]>('/audit/events');
            setEvents(response || []);
        } catch (err: any) {
            setError(err.message || 'Failed to fetch audit events');
        } finally {
            setLoading(false);
        }
    }, []);

    return {
        events,
        loading,
        error,
        fetchEvents,
    };
};
