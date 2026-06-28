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

// Centralized constants shared by both the client and server (server is authoritative).
// If you change these, update backend/internal/security/impersonation.go to match.
export const MIN_REASON_LENGTH = 10;
export const TICKET_REQUIRED_FOR_BREAK_GLASS = true;
export const DEFAULT_SESSION_DURATION_MINUTES = 30;
export const MAX_SESSION_DURATION_MINUTES = 120;

/**
 * ImpersonationScopeKind matches the backend CHECK constraint in
 * platform_admin_audit.scope_kind. Adding a new kind here requires a corresponding
 * server-side update.
 */
export type ImpersonationScopeKind = 'tenant' | 'instance' | 'product' | 'datasource';

// String constants matching the ImpersonationScopeKind union. Re-exported here so
// components can compare against typed values without sprinkling string literals.
export const ScopeTenant = 'tenant' as const;
export const ScopeInstance = 'instance' as const;
export const ScopeProduct = 'product' as const;
export const ScopeDatasource = 'datasource' as const;

/**
 * ImpersonationScope describes the optional narrowing of impersonation to a specific
 * resource within the target tenant. When scope_kind = 'tenant' (the default), the
 * admin has full tenant access. For tighter scopes, scope_id identifies the resource.
 */
export interface ImpersonationScope {
  kind: ImpersonationScopeKind;
  id: string;
}

/**
 * ImpersonationSession is the resolved session record kept in memory and localStorage.
 * Includes the audit-mode banner fields and an optional scope narrowing.
 */
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
  /** Optional scope narrowing (defaults to { kind: 'tenant', id: targetTenantId }) */
  scope: ImpersonationScope;
}

export interface AssumeContextParams {
  targetTenantId: string;
  targetTenantName: string;
  reason: string;
  ticketReference: string;
  mode: ImpersonationMode;
  durationMinutes: number;
  /** Optional scope narrowing; defaults to tenant-wide. */
  scope?: ImpersonationScope;
}

/**
 * RecentSession is a lightweight record of a recent impersonation, used for the
 * "recent sessions" strip in the picker.
 */
export interface RecentSession {
  tenantId: string;
  tenantName: string;
  lastUsedAt: number; // unix epoch ms
  mode: ImpersonationMode;
}

/**
 * ActiveImpersonationSession mirrors the backend's GET /admin/impersonate/sessions/active
 * response. Returned only by listActiveSessions().
 */
export interface ActiveImpersonationSession {
  session_id: string;
  admin_user_id: string;
  admin_email: string;
  target_tenant_id: string;
  mode: string;
  scope_kind?: string;
  scope_id?: string;
  reason: string;
  started_at: string;
  expires_at: string;
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

  /** Last 5 tenants the admin impersonated, newest first. */
  recentSessions: RecentSession[];

  /** Start an impersonation session */
  assumeTenantContext: (params: AssumeContextParams) => Promise<void>;

  /** End the active impersonation session */
  exitImpersonation: () => Promise<void>;

  /** Clear the recent-sessions cache (UI hook for the "clear history" button) */
  clearRecentSessions: () => void;

  /**
   * Fetch the recent-sessions list from the backend (which derives it from the
   * audit log) and merge it with the localStorage cache. Use on first mount so a
   * new browser still sees recently impersonated tenants.
   */
  refreshRecentSessionsFromServer: () => Promise<void>;

  /**
   * Fetch the list of currently-active impersonation sessions for this admin.
   * Returns sessions where an audit START row exists but no END row, AND
   * expires_at is still in the future.
   */
  listActiveSessions: () => Promise<ActiveImpersonationSession[]>;
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

const ImpersonationContext = createContext<ImpersonationContextType | undefined>(undefined);

const IMPERSONATION_SESSION_KEY = 'uisce_impersonation_session';
const IMPERSONATION_TOKEN_KEY = 'uisce_impersonation_token';
const IMPERSONATION_RECENT_KEY = 'uisce_impersonation_recent';

const API_BASE = import.meta.env.VITE_API_URL ?? '/api';

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

export const ImpersonationProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const { token: adminToken, user } = useAuth();
  const [session, setSession] = useState<ImpersonationSession | null>(null);
  const [impersonationToken, setImpersonationToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [recentSessions, setRecentSessions] = useState<RecentSession[]>(() => {
    // Rehydrate recent sessions on mount.
    try {
      const raw = localStorage.getItem(IMPERSONATION_RECENT_KEY);
      return raw ? (JSON.parse(raw) as RecentSession[]) : [];
    } catch {
      return [];
    }
  });
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Persist recent sessions whenever they change.
  useEffect(() => {
    try {
      localStorage.setItem(IMPERSONATION_RECENT_KEY, JSON.stringify(recentSessions));
    } catch {
      // Ignore storage errors (private mode, quota exceeded, etc.)
    }
  }, [recentSessions]);

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

  // On mount: refresh the recent-sessions list from the backend so a new browser
  // still sees recently impersonated tenants (the localStorage cache is empty).
  // Then poll every 30s while the admin is authenticated so the picker
  // stays current even when the admin has not opened it recently.
  useEffect(() => {
    if (!adminToken) return;
    void refreshRecentSessionsFromServer();
    const id = setInterval(() => {
      void refreshRecentSessionsFromServer();
    }, 30_000);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [adminToken]);

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

      // Resolve effective scope: default to tenant-wide if not provided.
      const scope: ImpersonationScope =
        params.scope ?? { kind: 'tenant', id: params.targetTenantId };

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
            scope_kind: scope.kind,
            scope_id: scope.id,
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
          scope_kind?: ImpersonationScopeKind;
          scope_id?: string;
        };

        const expiresAt = new Date(data.expires_at);
        const effectiveScope: ImpersonationScope = {
          kind: data.scope_kind ?? scope.kind,
          id: data.scope_id ?? scope.id,
        };
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
          scope: effectiveScope,
        };

        setSession(newSession);
        setImpersonationToken(data.access_token);
        persistSession(newSession, data.access_token);
        // Update recent sessions (deduped, newest first, max 5)
        addRecentSession({
          tenantId: data.tenant_id,
          tenantName: params.targetTenantName,
          lastUsedAt: Date.now(),
          mode: data.mode,
        });
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
      // Best-effort server-side END audit record; don't block local cleanup on failure.
      // Note: no longer sends tenant_id query param — the server recovers it from the audit row.
      await fetch(
        `${API_BASE}/admin/impersonate/${session.sessionId}`,
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
  // Recent sessions (top-5 most-recent impersonated tenants)
  // ---------------------------------------------------------------------------

  const addRecentSession = useCallback((entry: RecentSession) => {
    setRecentSessions((prev) => {
      // Dedup by tenantId: drop existing entry with the same id, then prepend.
      const filtered = prev.filter((s) => s.tenantId !== entry.tenantId);
      const next = [entry, ...filtered];
      return next.slice(0, 5);
    });
  }, []);

  const clearRecentSessions = useCallback(() => {
    setRecentSessions([]);
    try {
      localStorage.removeItem(IMPERSONATION_RECENT_KEY);
    } catch {
      // Ignore storage errors.
    }
  }, []);

  /**
   * Fetch recent sessions from the backend and merge them with the localStorage
   * cache. Backend wins on ties (most recent lastUsedAt wins). New tenants from
   * the backend are added; entries that no longer exist server-side are kept
   * (they may be tenants the admin lost access to, which is fine for the picker).
   */
  const refreshRecentSessionsFromServer = useCallback(async (): Promise<void> => {
    if (!adminToken) return;
    try {
      const resp = await fetch(`${API_BASE}/admin/impersonate/sessions/recent`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
      if (!resp.ok) return; // Silent fail — local cache is still usable.
      const data = await resp.json();
      const serverRecent: RecentSession[] = (data.recent_sessions ?? []).map(
        (s: { target_tenant_id: string; tenant_name?: string; mode: string; last_used_at: string }) => ({
          tenantId: s.target_tenant_id,
          tenantName: s.tenant_name || s.target_tenant_id.slice(0, 8),
          lastUsedAt: new Date(s.last_used_at).getTime(),
          mode: (s.mode as ImpersonationMode) || 'read_only',
        }),
      );
      setRecentSessions((prev) => {
        // Merge: server-first by lastUsedAt, dedup by tenantId, cap at 5.
        const byTenant = new Map<string, RecentSession>();
        for (const s of serverRecent) byTenant.set(s.tenantId, s);
        for (const s of prev) {
          const existing = byTenant.get(s.tenantId);
          if (!existing || s.lastUsedAt > existing.lastUsedAt) {
            byTenant.set(s.tenantId, s);
          }
        }
        return Array.from(byTenant.values())
          .sort((a, b) => b.lastUsedAt - a.lastUsedAt)
          .slice(0, 5);
      });
    } catch {
      // Network error — local cache still works.
    }
  }, [adminToken]);

  /**
   * Fetch the list of currently-active impersonation sessions for this admin.
   * Useful for the picker to warn about re-entry and for the banner to show
   * "1 session active" indicators.
   */
  const listActiveSessions = useCallback(async (): Promise<ActiveImpersonationSession[]> => {
    if (!adminToken) return [];
    try {
      const resp = await fetch(`${API_BASE}/admin/impersonate/sessions/active`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
      if (!resp.ok) return [];
      const data = await resp.json();
      return data.active_sessions ?? [];
    } catch {
      return [];
    }
  }, [adminToken]);

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
    recentSessions,
    assumeTenantContext,
    exitImpersonation,
    clearRecentSessions,
    refreshRecentSessionsFromServer,
    listActiveSessions,
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
