// Centralized authenticated fetch wrapper
// Automatically injects Authorization header, X-User-ID, handles token refresh once, JSON parsing,
// and globally redirects to /login on 401s.

import { useAuth } from '../contexts/AuthContext';
import { useLocation } from 'react-router-dom';
import useBlockableNavigate from '../components/RouteBlocker/useBlockableNavigate';
import { useToast } from '../hooks/use-toast';

export interface AuthFetchOptions extends RequestInit {
  json?: any; // auto JSON body
  retry?: boolean; // internal flag to avoid infinite loops
}

export interface AuthFetchResponse<T=any> {
  ok: boolean;
  status: number;
  data: T | null;
  error?: string;
  response: Response;
}

// simple module-level guard to avoid spamming redirects on mass 401s
let isRedirectingToLogin = false;

// Hook returning a typed fetch wrapper
export function useAuthFetch() {
  const { getValidToken, user, logout } = useAuth();
  const navigate = useBlockableNavigate();
  const location = useLocation();
  const toast = useToast();

  const authFetch = async <T=any>(url: string, options: AuthFetchOptions = {}): Promise<AuthFetchResponse<T>> => {
    // Be defensive: tests may mock useAuth incompletely
    const token = typeof getValidToken === 'function' ? await getValidToken() : undefined;
    const headers: Record<string,string> = {
      'Accept': 'application/json',
      ...(options.headers as Record<string,string> || {})
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (user?.id) headers['X-User-ID'] = user.id;

    let body = options.body;
    if (options.json !== undefined) {
      headers['Content-Type'] = 'application/json';
      body = JSON.stringify(options.json);
    }

  const resp = await fetch(url, { credentials: 'include', ...options, headers, body });

    // If unauthorized: try a single refresh-and-retry; if still 401, logout and redirect
    if (resp.status === 401) {
      if (!options.retry) {
        const refreshed = await getValidToken();
        if (refreshed) {
          const retryResp = await authFetch<T>(url, { ...options, retry: true });
          if (!retryResp.ok && retryResp.status === 401) {
            if (!isRedirectingToLogin) {
              isRedirectingToLogin = true;
              await logout();
              try { toast.toast({ title: 'Session expired', description: 'Please sign in again to continue.', variant: 'destructive' }); } catch {}
              try { void navigate('/login', { replace: true, state: { from: location } }); } catch { if (typeof window !== 'undefined') window.location.href = '/login'; }
              // release the guard after a tick to avoid blocking future legitimate redirects
              setTimeout(() => { isRedirectingToLogin = false; }, 500);
            }
          }
          return retryResp;
        }
      }
      if (!isRedirectingToLogin) {
        isRedirectingToLogin = true;
        await logout();
        try { toast.toast({ title: 'Session expired', description: 'Please sign in again to continue.', variant: 'destructive' }); } catch {}
  try { void navigate('/login', { replace: true, state: { from: location } }); } catch { if (typeof window !== 'undefined') window.location.href = '/login'; }
        setTimeout(() => { isRedirectingToLogin = false; }, 500);
      }
      return { ok: false, status: 401, data: null, error: 'Unauthorized', response: resp };
    }

    // Treat 304 Not Modified as a successful response (no body) so callers
    // don't interpret it as an error. Many consumers expect cached responses
    // and will handle null data appropriately.
    if (resp.status === 304) {
      return { ok: true, status: 304, data: null as any, response: resp };
    }

    let data: any = null;
    // Safely detect JSON without assuming headers shape (tests often mock fetch)
    let isJson = false;
    try {
      const ct = (resp as any)?.headers && typeof (resp as any).headers.get === 'function'
        ? (resp as any).headers.get('Content-Type')
        : ((resp as any).headers?.['Content-Type'] || (resp as any).headers?.['content-type'] || '');
      if (typeof ct === 'string' && ct.toLowerCase().includes('application/json')) isJson = true;
    } catch { /* ignore */ }
    if (isJson || typeof (resp as any)?.json === 'function') {
      try { data = await (resp as any).json(); } catch { /* ignore */ }
    }

    if (!resp.ok) {
      return { ok: false, status: resp.status, data: data, error: (data && (data.error || data.message)) || resp.statusText, response: resp };
    }

    return { ok: true, status: resp.status, data, response: resp };
  };

  return { authFetch };
}
