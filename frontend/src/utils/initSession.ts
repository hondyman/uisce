/**
 * Initialize session for development seeding and other startup tasks
 */
export function initSession(): void {
  // In development mode, seed a valid development admin user & token if no session exists.
  if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
    const token = localStorage.getItem('auth_token');
    let isExpired = false;
    if (token && token.split('.').length === 3) {
      try {
        const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
        if (payload.exp && payload.exp * 1000 < Date.now() + 60000) {
          isExpired = true;
        }
      } catch (_) {
        isExpired = true;
      }
    }

    if (!token || token.split('.').length !== 3 || isExpired) {
      const devToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbi11c2VyIiwidXNlcl9pZCI6ImFkbWluLXVzZXIiLCJyb2xlcyI6WyJnbG9iYWxfYWRtaW4iLCJjb3JlX2FkbWluIiwiZGV2ZWxvcGVyIiwicGxhdGZvcm1fb3BlcmF0b3IiXSwidGVuYW50X2lkIjoiOTllOTllOTktOTllOS00OWU5LTg5ZTktOTllOTllOTllOTk5IiwidGVuYW50X2lkcyI6WyI5OWU5OWU5OS05OWU5LTQ5ZTktODllOS05OWU5OWU5OWU5OTkiXSwiaWF0IjoxNzkxNDI3OTYyLCJleHAiOjIxMDY3ODc5NjJ9.d2LpahW43DF-VJsaj0U8RkP1gdOJoJpEpl0uSPGDYYA';
      const devUser = {
        id: 'admin-user',
        email: 'admin@example.com',
        name: 'Admin User',
        role: 'global_admin',
        organization: 'uisce',
        permissions: ['*'],
        is_active: true,
        roles: ['global_admin', 'core_admin', 'developer', 'platform_operator'],
        is_core_admin: true,
        isCoreAdmin: true,
        is_admin: true,
        is_global_admin: true,
      };

      try {
        localStorage.setItem('auth_token', devToken);
        localStorage.setItem('auth_user', JSON.stringify(devUser));
        localStorage.setItem('auth_expires_at', (Date.now() + 365 * 24 * 3600 * 1000).toString());
      } catch (_) {}
    }

    // Clean up any stale/invalid placeholder scope entries from previous sessions
    try {
      const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
      
      const storedDs = localStorage.getItem('selected_datasource');
      if (storedDs) {
        const parsed = JSON.parse(storedDs);
        if (!parsed?.id || !UUID_REGEX.test(parsed.id) || parsed.id.startsWith('00000000') || parsed.id.startsWith('11111111')) {
          localStorage.removeItem('selected_datasource');
        }
      }

      const storedScope = localStorage.getItem('operating_scope');
      if (storedScope) {
        const parsed = JSON.parse(storedScope);
        if (parsed?.datasourceId && (!UUID_REGEX.test(parsed.datasourceId) || parsed.datasourceId.startsWith('00000000') || parsed.datasourceId.startsWith('11111111'))) {
          localStorage.removeItem('operating_scope');
        }
      }
    } catch (_) {}
  }

  console.log('Session initialized');
}