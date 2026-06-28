/**
 * ImpersonationContext
 *
 * Manages the Global Admin → Tenant Impersonation lifecycle:
 *   1. Calls POST /api/admin/impersonate to obtain a scoped context token
 *   2. Stores the scoped token and swaps it into all outgoing API calls
 *   3. Maintains a live countdown and auto-exits on expiry
 *   4. Calls DELETE /api/admin/impersonate/:sessionId on manual exit
 *
 * Design contract:
 *   - When impersonating, ALL API requests carry the scoped token
 *   - The scoped token contains a concrete tenant_id — downstream RLS / ABAC
 *     runs identically to a regular tenant-scoped request
 *   - On exit, the original admin token is restored
 */

import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { useAuth } from './AuthContext';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ImpersonationMode = 'read_only' | 'break_glass';

export interface ImpersonationSession {
  sessionId: string;
  targetTenantId: string;
  targetTenantName: string;
  adminUserId: string;
  mode: ImpersonationMode;
  reason: string;
  ticketReference: string;
  expiresAt: Date;
  /** Countdown seconds remaining — updates every second */
  secondsRemaining: number;
}

export interface AssumeContextParams {
  targetTenantId: string;
  targetTenantName: string;
  reason: string;
  ticketReference: string;
  mode: ImpersonationMode;
  durationMinutes: number;
}

interface ImpersonationContextType {
  /** True while an impersonation session is active */
  isImpersonating: boolean;

  /** Current session metadata (null when not impersonating) */
  session: ImpersonationSession | null;

  /** The scoped context token returned by the backend */
  impersonationToken: string | null;

  /** True while the assume/exit API call is in flight */
  isLoading: boolean;

  /** Start an impersonation session */
  assumeTenantContext: (params: AssumeContextParams) => Promise<void>;

  /** End the active impersonation session */
  exitImpersonation: () => Promise<void>;
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

const ImpersonationContext = createContext<ImpersonationContextType | undefined>(undefined);

const IMPERSONATION_SESSION_KEY = 'uisce_impersonation_session';
const IMPERSONATION_TOKEN_KEY = 'uisce_impersonation_token';

const API_BASE = import.meta.env.VITE_API_URL ?? '/api';

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

export const ImpersonationProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const { token: adminToken, user } = useAuth();
  const [session, setSession] = useState<ImpersonationSession | null>(null);
  const [impersonationToken, setImpersonationToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Rehydrate session from localStorage on mount (survives page refresh)
  useEffect(() => {
    try {
      const raw = localStorage.getItem(IMPERSONATION_SESSION_KEY);
      const tok = localStorage.getItem(IMPERSONATION_TOKEN_KEY);
      if (raw && tok) {
        const parsed = JSON.parse(raw) as ImpersonationSession;
        const expiresAt = new Date(parsed.expiresAt);
        if (expiresAt > new Date()) {
          setSession({ ...parsed, expiresAt });
          setImpersonationToken(tok);
        } else {
          // Session expired while the page was closed — clean up silently
          clearPersistedSession();
        }
      }
    } catch {
      clearPersistedSession();
    }
  }, []);

  // Live countdown ticker
  useEffect(() => {
    if (!session) {
      if (countdownRef.current) clearInterval(countdownRef.current);
      return;
    }

    countdownRef.current = setInterval(() => {
      const remaining = Math.max(
        0,
        Math.round((session.expiresAt.getTime() - Date.now()) / 1000),
      );

      if (remaining === 0) {
        // Auto-exit on expiry
        void exitImpersonation();
        return;
      }

      setSession((prev) => (prev ? { ...prev, secondsRemaining: remaining } : null));
    }, 1000);

    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.sessionId]);

  // ---------------------------------------------------------------------------
  // assumeTenantContext
  // ---------------------------------------------------------------------------

  const assumeTenantContext = useCallback(
    async (params: AssumeContextParams): Promise<void> => {
      if (!adminToken) throw new Error('Not authenticated');

      setIsLoading(true);
      try {
        const resp = await fetch(`${API_BASE}/admin/impersonate`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${adminToken}`,
          },
          body: JSON.stringify({
            target_tenant_id: params.targetTenantId,
            reason: params.reason,
            ticket_reference: params.ticketReference,
            mode: params.mode,
            duration_minutes: params.durationMinutes,
          }),
        });

        if (!resp.ok) {
          const err = await resp.json().catch(() => ({ error: resp.statusText }));
          throw new Error(err.error ?? `Impersonation request failed (${resp.status})`);
        }

        const data = await resp.json() as {
          access_token: string;
          token_type: string;
          expires_at: string;
          session_id: string;
          tenant_id: string;
          mode: ImpersonationMode;
        };

        const expiresAt = new Date(data.expires_at);
        const newSession: ImpersonationSession = {
          sessionId: data.session_id,
          targetTenantId: data.tenant_id,
          targetTenantName: params.targetTenantName,
          adminUserId: user?.id ?? '',
          mode: data.mode,
          reason: params.reason,
          ticketReference: params.ticketReference,
          expiresAt,
          secondsRemaining: Math.round((expiresAt.getTime() - Date.now()) / 1000),
        };

        setSession(newSession);
        setImpersonationToken(data.access_token);
        persistSession(newSession, data.access_token);
      } finally {
        setIsLoading(false);
      }
    },
    [adminToken, user],
  );

  // ---------------------------------------------------------------------------
  // exitImpersonation
  // ---------------------------------------------------------------------------

  const exitImpersonation = useCallback(async (): Promise<void> => {
    if (!session) return;

    setIsLoading(true);
    try {
      // Best-effort server-side END audit record; don't block local cleanup on failure
      await fetch(
        `${API_BASE}/admin/impersonate/${session.sessionId}?tenant_id=${session.targetTenantId}`,
        {
          method: 'DELETE',
          headers: {
            Authorization: `Bearer ${adminToken}`,
          },
        },
      ).catch(() => {
        // Network failure — audit will rely on expiry detection server-side
        console.error('[ImpersonationContext] Failed to notify server of session end');
      });
    } finally {
      setSession(null);
      setImpersonationToken(null);
      clearPersistedSession();
      setIsLoading(false);
    }
  }, [session, adminToken]);

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  const persistSession = (s: ImpersonationSession, tok: string) => {
    localStorage.setItem(IMPERSONATION_SESSION_KEY, JSON.stringify(s));
    localStorage.setItem(IMPERSONATION_TOKEN_KEY, tok);
  };

  const clearPersistedSession = () => {
    localStorage.removeItem(IMPERSONATION_SESSION_KEY);
    localStorage.removeItem(IMPERSONATION_TOKEN_KEY);
  };

  // ---------------------------------------------------------------------------
  // Value
  // ---------------------------------------------------------------------------

  const value: ImpersonationContextType = {
    isImpersonating: !!session,
    session,
    impersonationToken,
    isLoading,
    assumeTenantContext,
    exitImpersonation,
  };

  return (
    <ImpersonationContext.Provider value={value}>
      {children}
    </ImpersonationContext.Provider>
  );
};

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export const useImpersonation = (): ImpersonationContextType => {
  const ctx = useContext(ImpersonationContext);
  if (!ctx) throw new Error('useImpersonation must be used within ImpersonationProvider');
  return ctx;
};

/**
 * Returns the token that should be sent in Authorization headers.
 * During an active impersonation session this is the scoped context token;
 * otherwise it's the primary admin token.
 */
export const useActiveToken = (): string | null => {
  const { impersonationToken, isImpersonating } = useImpersonation();
  const { token: adminToken } = useAuth();
  return isImpersonating ? impersonationToken : adminToken;
};
