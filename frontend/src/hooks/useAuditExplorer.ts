import { useState, useEffect, useCallback } from 'react';

export interface UseAuditExplorerOptions {
  tenantScope: string[];
  timeRange: string;
  artifactTypes: string[];
  statuses: string[];
  riskLevels: string[];
}

interface ListEventsRequest {
  tenantFilter: string[];
  from: string;
  to: string;
  artifactTypes?: string[];
  statuses?: string[];
  riskLevels?: string[];
  limit: number;
  offset: number;
}

export function useAuditExplorer(options: UseAuditExplorerOptions) {
  const [events, setEvents] = useState<any[]>([]);
  const [incidents, setIncidents] = useState<any[]>([]);
  const [complianceEvents, setComplianceEvents] = useState<any[]>([]);
  const [dashboardData] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Calculate time range
  const getTimeRange = useCallback((range: string) => {
    const now = new Date();
    const to = now.toISOString();
    let from: string;

    switch (range) {
      case '24h':
        from = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        break;
      case '7d':
        from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString();
        break;
      case '30d':
        from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString();
        break;
      default:
        from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString();
    }

    return { from, to };
  }, []);

  const fetchEvents = useCallback(async () => {
    if (options.tenantScope.length === 0) {
      setEvents([]);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const { from, to } = getTimeRange(options.timeRange);

      const request: ListEventsRequest = {
        tenantFilter: options.tenantScope,
        from,
        to,
        artifactTypes: options.artifactTypes.length > 0 ? options.artifactTypes : undefined,
        statuses: options.statuses.length > 0 ? options.statuses : undefined,
        riskLevels: options.riskLevels.length > 0 ? options.riskLevels : undefined,
        limit: 50,
        offset: 0,
      };

      const response = await fetch('/api/audit-explorer/events', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
      });

      if (response.ok) {
        const data = await response.json();
        setEvents(data.events || []);
      } else {
        setError('Failed to fetch audit events');
        setEvents([]);
      }
    } catch (err) {
      console.error('Error fetching events:', err);
      setError('Error fetching audit events');
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [options, getTimeRange]);

  const fetchIncidents = useCallback(async () => {
    if (options.tenantScope.length === 0) {
      setIncidents([]);
      return;
    }

    try {
      const { from, to } = getTimeRange(options.timeRange);

      const response = await fetch(
        `/api/audit-explorer/incidents?from=${from}&to=${to}&limit=50&offset=0`,
        {
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (response.ok) {
        const data = await response.json();
        setIncidents(data.incidents || []);
      } else {
        setIncidents([]);
      }
    } catch (err) {
      console.error('Error fetching incidents:', err);
      setIncidents([]);
    }
  }, [options, getTimeRange]);

  const fetchComplianceEvents = useCallback(async () => {
    if (options.tenantScope.length === 0) {
      setComplianceEvents([]);
      return;
    }

    try {
      const { from, to } = getTimeRange(options.timeRange);

      const response = await fetch(
        `/api/audit-explorer/compliance-events?from=${from}&to=${to}&limit=50&offset=0`,
        {
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (response.ok) {
        const data = await response.json();
        setComplianceEvents(data.events || []);
      } else {
        setComplianceEvents([]);
      }
    } catch (err) {
      console.error('Error fetching compliance events:', err);
      setComplianceEvents([]);
    }
  }, [options, getTimeRange]);

  const refetch = useCallback(() => {
    fetchEvents();
    fetchIncidents();
    fetchComplianceEvents();
  }, [fetchEvents, fetchIncidents, fetchComplianceEvents]);

  // Fetch on options change
  useEffect(() => {
    refetch();
  }, [options, refetch]);

  return {
    events,
    incidents,
    complianceEvents,
    dashboardData,
    loading,
    error,
    refetch,
  };
}

export default useAuditExplorer;
