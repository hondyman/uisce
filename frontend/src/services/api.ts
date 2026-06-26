// Assuming a generic api client is available. If not, this uses fetch.
const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081/api/v1';

export const calendarApi = {
  getAuthUrl: async (tenantId: string, userId: string) => {
    const res = await fetch(`${API_BASE}/sync/google/auth-url-pkce?tenant_id=${tenantId}&user_id=${userId}`);
    if (!res.ok) throw new Error('Failed to get auth URL');
    return res.json();
  },

  getMicrosoftAuthUrl: async (tenantId: string, userId: string) => {
    const res = await fetch(`${API_BASE}/sync/microsoft/auth-url-pkce?tenant_id=${tenantId}&user_id=${userId}`);
    if (!res.ok) throw new Error('Failed to get Microsoft auth URL');
    return res.json();
  },

  getCalendars: async (tenantId: string, userId: string) => {
    const res = await fetch(`${API_BASE}/sync/google/calendars?tenant_id=${tenantId}&user_id=${userId}`);
    if (!res.ok) throw new Error('Failed to get google calendars');
    return res.json(); // Stub endpoint, handled by Hasura in real app
  },

  getMicrosoftCalendars: async (tenantId: string, userId: string) => {
    const res = await fetch(`${API_BASE}/sync/microsoft/calendars?tenant_id=${tenantId}&user_id=${userId}`);
    if (!res.ok) throw new Error('Failed to get microsoft calendars');
    return res.json(); // Stub endpoint, handled by Hasura in real app
  },

  triggerSync: async (tenantId: string, userId: string, calendarId: string = 'primary') => {
    const res = await fetch(`${API_BASE}/sync/google/sync-all`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tenant_id: tenantId, user_id: userId, google_calendar_id: calendarId }),
    });
    if (!res.ok) throw new Error('Failed to trigger sync');
    return res.json();
  },

  triggerMicrosoftSync: async (tenantId: string, userId: string, calendarId: string = 'primary') => {
    const res = await fetch(`${API_BASE}/sync/microsoft/sync-all`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tenant_id: tenantId, user_id: userId, microsoft_calendar_id: calendarId }),
    });
    if (!res.ok) throw new Error('Failed to trigger Microsoft sync');
    return res.json();
  },

  getSyncStatus: async (syncId: string) => {
    const res = await fetch(`${API_BASE}/sync/status/${syncId}`);
    if (!res.ok) throw new Error('Failed to get sync status');
    return res.json();
  },

  getSyncedEvents: async (tenantId: string) => {
    const res = await fetch(`${API_BASE}/sync/google/events?tenant_id=${tenantId}`);
    if (!res.ok) throw new Error('Failed to fetch events');
    return res.json(); // Stub endpoint, handled by Hasura in real app
  },

  getSyncedMicrosoftEvents: async (tenantId: string) => {
    const res = await fetch(`${API_BASE}/sync/microsoft/events?tenant_id=${tenantId}`);
    if (!res.ok) throw new Error('Failed to fetch Microsoft events');
    return res.json(); // Stub endpoint, handled by Hasura in real app
  }
};
