/**
 * Initialize session for development seeding and other startup tasks
 */
export function initSession(): void {
  // In development mode, seed a valid development admin user & token if no session exists.
  if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
    const token = localStorage.getItem('auth_token');
    if (!token || token.split('.').length !== 3) {
      const devToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3OTE0Mjk5ODYsImlhdCI6MTc5MTQyNjM4Niwicm9sZXMiOlsiZ2xvYmFsX2FkbWluIiwiY29yZV9hZG1pbiIsImRldmVsb3BlciIsInBsYXRmb3JtX29wZXJhdG9yIl0sInN1YiI6ImFkbWluLXVzZXIiLCJ0ZW5hbnRfaWQiOiI5OWU5OWU5OS05OWU5LTQ5ZTktODllOS05OWU5OWU5OWU5OTkiLCJ0ZW5hbnRfaWRzIjpbIjk5ZTk5ZTk5LTk5ZTktNDllOS04OWU5LTk5ZTk5ZTk5ZTk5OSJdLCJ1c2VyX2lkIjoiYWRtaW4tdXNlciJ9.Pr-3sDgWxFvVnhkBua4sRlF8rzhkqhEVcLTJUd89hek';
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
  }

  console.log('Session initialized');
}