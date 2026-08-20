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
  getAuthUrl: async (provider: CalendarProvider = 'google') => {
    const res = await fetch(`${API_BASE}/sync/${provider}/auth-url-pkce`, {
      headers: getHeaders()
    });
    if (!res.ok) throw new Error('Failed to get auth URL');
    return res.json();
  },

  getMicrosoftAuthUrl: async () => {
    return calendarApi.getAuthUrl('microsoft');
  },

  getCalendars: async (provider: CalendarProvider = 'google') => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/calendars?provider=${provider}`, {
      headers: getHeaders()
    });
    if (!res.ok) throw new Error(`Failed to get ${provider} calendars`);
    return res.json();
  },

  getMicrosoftCalendars: async () => {
    return calendarApi.getCalendars('microsoft');
  },

  triggerSync: async (
    provider: CalendarProvider,
    externalCalendarId: string,
    internalCalendarId?: string
  ) => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/sync`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({
        provider,
        external_calendar_id: externalCalendarId,
        internal_calendar_id: internalCalendarId,
      }),
    });
    if (!res.ok) throw new Error('Failed to trigger sync');
    return res.json();
  },

  triggerMicrosoftSync: async (calendarId: string = 'primary') => {
    return calendarApi.triggerSync('microsoft', calendarId);
  },

  getSyncStatus: async (syncId: string) => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/status/${syncId}`, {
      headers: getHeaders()
    });
    if (!res.ok) throw new Error('Failed to get sync status');
    return res.json();
  },

  getSyncedEvents: async (provider?: CalendarProvider) => {
    const url = new URL(`${API_BASE}/v1/sync/calendars/events`);
    if (provider) url.searchParams.set('provider', provider);
    const res = await fetch(url.toString(), {
      headers: getHeaders()
    });
    if (!res.ok) throw new Error('Failed to fetch events');
    return res.json();
  },

  getSyncedMicrosoftEvents: async () => {
    return calendarApi.getSyncedEvents('microsoft');
  },

  listActiveSyncs: async () => {
    const res = await fetch(`${API_BASE}/v1/sync/calendars/active`, {
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
