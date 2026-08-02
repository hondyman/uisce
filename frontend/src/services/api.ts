const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081/api/v1';

export type CalendarProvider = 'google' | 'microsoft' | 'apple';

function getHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  const tenantId = localStorage.getItem('tenant_id');
  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId;
  }
  return headers;
}

export const calendarApi = {
  getAuthUrl: async (tenantId: string, userId: string, provider: CalendarProvider = 'google') => {
    const res = await fetch(`${API_BASE}/sync/${provider}/auth-url-pkce?user_id=${userId}`, {
      headers: { ...getHeaders(), 'X-User-ID': userId }
    });
    if (!res.ok) throw new Error('Failed to get auth URL');
    return res.json();
  },

  getMicrosoftAuthUrl: async (tenantId: string, userId: string) => {
    return calendarApi.getAuthUrl(tenantId, userId, 'microsoft');
  },

  getCalendars: async (tenantId: string, userId: string, provider: CalendarProvider = 'google') => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/calendars?provider=${provider}`, {
      headers: { ...getHeaders(), 'X-User-ID': userId }
    });
    if (!res.ok) throw new Error(`Failed to get ${provider} calendars`);
    return res.json();
  },

  getMicrosoftCalendars: async (tenantId: string, userId: string) => {
    return calendarApi.getCalendars(tenantId, userId, 'microsoft');
  },

  triggerSync: async (
    tenantId: string,
    userId: string,
    provider: CalendarProvider,
    externalCalendarId: string,
    internalCalendarId?: string
  ) => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/sync`, {
      method: 'POST',
      headers: { ...getHeaders(), 'X-User-ID': userId },
      body: JSON.stringify({
        user_id: userId,
        provider,
        external_calendar_id: externalCalendarId,
        internal_calendar_id: internalCalendarId,
      }),
    });
    if (!res.ok) throw new Error('Failed to trigger sync');
    return res.json();
  },

  triggerMicrosoftSync: async (tenantId: string, userId: string, calendarId: string = 'primary') => {
    return calendarApi.triggerSync(tenantId, userId, 'microsoft', calendarId);
  },

  getSyncStatus: async (syncId: string) => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/status/${syncId}`, {
      headers: getHeaders()
    });
    if (!res.ok) throw new Error('Failed to get sync status');
    return res.json();
  },

  getSyncedEvents: async (tenantId: string, userId: string, provider?: CalendarProvider) => {
    const url = new URL(`${API_BASE}/v1/sync/calendars/events`);
    if (provider) url.searchParams.set('provider', provider);
    const res = await fetch(url.toString(), {
      headers: { ...getHeaders(), 'X-User-ID': userId }
    });
    if (!res.ok) throw new Error('Failed to fetch events');
    return res.json();
  },

  getSyncedMicrosoftEvents: async (tenantId: string, userId: string) => {
    return calendarApi.getSyncedEvents(tenantId, userId, 'microsoft');
  },

  listActiveSyncs: async (userId: string) => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/active?user_id=${userId}`, {
      headers: getHeaders()
    });
    if (!res.ok) throw new Error('Failed to fetch active syncs');
    return res.json();
  },

  cancelSync: async (syncId: string) => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/cancel/${syncId}`, {
      method: 'POST',
      headers: getHeaders()
    });
    if (!res.ok) throw new Error('Failed to cancel sync');
    return res.json();
  },
};
