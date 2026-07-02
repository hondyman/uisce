import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback, useRef } from 'react';
import { User as OidcUser } from 'oidc-client-ts';
import { devLog, devError } from '../utils/devLogger';
import { userManager } from '../config/oidc';
import { useToast } from '../hooks/use-toast';

interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  organization: string;
  permissions: string[];
  is_active: boolean;
  roles?: string[];
  is_core_admin?: boolean;
  isCoreAdmin?: boolean;
  is_admin?: boolean;
  is_global_admin?: boolean;
  /** Raw Keycloak groups claim (group paths / names) */
  groups?: string[];
  /** Top-level `operator_role` claim injected by the Keycloak profile scope */
  operator_role?: string;
  /** Nested `uisce_metadata` claim from Keycloak, if present */
  uisce_metadata?: Record<string, unknown>;
  /** Tenant assignments for multi-tenant access control */
  tenant_assignments?: Array<{
    tenantId: string;
    tenantName?: string;
    accessLevel: 'platform_operator' | 'tenant_admin' | 'tenant_user';
    isReadOnly?: boolean;
  }>;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  refreshTokenValue: string | null;
  tokenExpiresAt: number | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isAdmin: () => boolean;
  isCoreAdmin: () => boolean;
  /** True when the user holds the global_admin or global_ops role from Keycloak */
  isGlobalAdmin: () => boolean;
  canManageCoreAssets: () => boolean;
  canManageCustomAssets: () => boolean;
  login: (email?: string, password?: string) => Promise<void>;
  register: (email: string, password: string, name: string, organization?: string) => Promise<void>;
  forgotPassword: (email: string) => Promise<void>;
  resetPassword: (token: string, newPassword: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
  isTokenExpired: () => boolean;
  getValidToken: () => Promise<string | null>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

const AUTH_TOKEN_KEY = 'auth_token';
const AUTH_REFRESH_TOKEN_KEY = 'auth_refresh_token';
const AUTH_USER_KEY = 'auth_user';
const AUTH_EXPIRES_AT_KEY = 'auth_expires_at';

function extractRoles(profile: Record<string, unknown>): string[] {
  const roles: string[] = [];

  if (Array.isArray(profile.roles)) {
    for (const r of profile.roles) {
      if (typeof r === 'string') roles.push(r);
    }
  }

  const realmAccess = profile.realm_access as Record<string, unknown> | undefined;
  if (realmAccess && Array.isArray(realmAccess.roles)) {
    for (const r of realmAccess.roles) {
      if (typeof r === 'string') roles.push(r);
    }
  }

  const resourceAccess = profile.resource_access as Record<string, Record<string, unknown>> | undefined;
  if (resourceAccess) {
    for (const client of Object.keys(resourceAccess)) {
      const clientAccess = resourceAccess[client];
      if (clientAccess && Array.isArray(clientAccess.roles)) {
        for (const r of clientAccess.roles) {
          if (typeof r === 'string') roles.push(`${client}:${r}`);
        }
      }
    }
  }

  return [...new Set(roles)];
}

// Recognise the platform-operator status from a Keycloak `groups` claim.
// Keycloak emits the *path* of the group (e.g. "/Uisce-Global-Admins") when full.path=true,
// or just the leaf name (e.g. "Uisce-Global-Admins") otherwise. Be permissive.
const GLOBAL_ADMIN_GROUP_RE = /(^|\/)uisce[-_ ]?global[-_ ]?admins?$/i;
const GLOBAL_OPS_GROUP_RE = /(^|\/)uisce[-_ ]?(global[-_ ]?ops|ops)$/i;

function extractGroups(profile: Record<string, unknown>): string[] {
  const result: string[] = [];
  const raw = profile.groups;
  if (Array.isArray(raw)) {
    for (const g of raw) {
      if (typeof g === 'string') result.push(g);
    }
  }
  return result;
}

function mapProfileToUser(profile: Record<string, unknown>, roles: string[]): User {
  const email = (profile.email as string) || (profile.preferred_username as string) || '';
  const name =
    (profile.name as string) ||
    (profile.preferred_username as string) ||
    email;

  const isCoreAdmin = roles.some(
    (r) => r === 'admin' || r === 'realm-admin' || r === 'core_admin' || r === 'core-admin',
  );

  // Global admin check: read from the custom uisce_metadata claim injected by Keycloak,
  // OR fall back to checking the roles array for global_admin / global_ops,
  // OR recognise the federated IdP group (Uisce-Global-Admins) when present.
  const uisceMetadata = profile.uisce_metadata as Record<string, unknown> | undefined;
  const operatorRole = (profile.operator_role as string | undefined) || '';
  const groups = extractGroups(profile);

  const hasGlobalAdminGroup = groups.some((g) => GLOBAL_ADMIN_GROUP_RE.test(g));
  const hasGlobalOpsGroup = groups.some((g) => GLOBAL_OPS_GROUP_RE.test(g));

  const isGlobalAdmin =
    uisceMetadata?.is_global_admin === true ||
    uisceMetadata?.operator_role === 'global_admin' ||
    operatorRole === 'global_admin' ||
    operatorRole === 'global_ops' ||
    roles.includes('global_admin') ||
    roles.includes('global_ops') ||
    hasGlobalAdminGroup ||
    hasGlobalOpsGroup;

  return {
    id: (profile.sub as string) || (profile.preferred_username as string) || email,
    email,
    name,
    role: roles[0] || 'user',
    organization: (profile.organization as string) || '',
    permissions: [],
    is_active: true,
    roles,
    groups,
    operator_role: operatorRole || undefined,
    uisce_metadata: uisceMetadata,
    is_core_admin: isCoreAdmin,
    isCoreAdmin: isCoreAdmin,
    is_admin: isCoreAdmin || roles.includes('admin'),
    is_global_admin: isGlobalAdmin,
  };
}

function persistOidcUser(oidcUser: OidcUser): void {
  try {
    // Send the OIDC ID token to the backend verifier.
    const token = oidcUser.id_token || oidcUser.access_token;
    const refreshToken = oidcUser.refresh_token;
    const roles = extractRoles(oidcUser.profile);
    const user = mapProfileToUser(oidcUser.profile, roles);

    let expiresAt: number | null = null;
    if (typeof oidcUser.expires_at === 'number') {
      expiresAt = oidcUser.expires_at * 1000;
    } else if (oidcUser.expires_in && typeof oidcUser.expires_in === 'number') {
      expiresAt = Date.now() + oidcUser.expires_in * 1000;
    }

    localStorage.setItem(AUTH_TOKEN_KEY, token);
    if (refreshToken) {
      localStorage.setItem(AUTH_REFRESH_TOKEN_KEY, refreshToken);
    }
    localStorage.setItem(AUTH_USER_KEY, JSON.stringify(user));
    if (expiresAt) {
      localStorage.setItem(AUTH_EXPIRES_AT_KEY, expiresAt.toString());
    }
  } catch (error) {
    devError('Error persisting OIDC user:', error);
  }
}

function clearPersistedAuth(): void {
  localStorage.removeItem(AUTH_TOKEN_KEY);
  localStorage.removeItem(AUTH_REFRESH_TOKEN_KEY);
  localStorage.removeItem(AUTH_USER_KEY);
  localStorage.removeItem(AUTH_EXPIRES_AT_KEY);
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [refreshTokenValue, setRefreshTokenValue] = useState<string | null>(null);
  const [tokenExpiresAt, setTokenExpiresAt] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const oidcUserRef = useRef<OidcUser | null>(null);
  const toast = useToast();

  const hydrateFromOidcUser = useCallback((oidcUser: OidcUser | null) => {
    oidcUserRef.current = oidcUser;
    if (oidcUser && !oidcUser.expired) {
      persistOidcUser(oidcUser);
      const storedUser = localStorage.getItem(AUTH_USER_KEY);
      const storedToken = localStorage.getItem(AUTH_TOKEN_KEY);
      const storedRefreshToken = localStorage.getItem(AUTH_REFRESH_TOKEN_KEY);
      const storedExpiresAt = localStorage.getItem(AUTH_EXPIRES_AT_KEY);
      setUser(storedUser ? JSON.parse(storedUser) : null);
      setToken(storedToken);
      setRefreshTokenValue(storedRefreshToken);
      setTokenExpiresAt(storedExpiresAt ? parseInt(storedExpiresAt, 10) : null);
    } else {
      clearPersistedAuth();
      setUser(null);
      setToken(null);
      setRefreshTokenValue(null);
      setTokenExpiresAt(null);
    }
  }, []);

  // Load user on mount and subscribe to OIDC events.
  useEffect(() => {
    let mounted = true;

    const loadUser = async () => {
      try {
        const oidcUser = await userManager.getUser();
        if (!mounted) return;
        hydrateFromOidcUser(oidcUser);
      } catch (error) {
        devError('Error loading OIDC user:', error);
      } finally {
        if (mounted) setIsLoading(false);
      }
    };

    void loadUser();

    const onUserLoaded = (oidcUser: OidcUser) => {
      devLog('OIDC user loaded event');
      hydrateFromOidcUser(oidcUser);
    };

    const onUserUnloaded = () => {
      devLog('OIDC user unloaded event');
      hydrateFromOidcUser(null);
    };

    const onSilentRenewError = (error: Error) => {
      devError('OIDC silent renew error:', error);
      // Clear auth state on silent renew failure so the user is redirected to login.
      hydrateFromOidcUser(null);
    };

    userManager.events.addUserLoaded(onUserLoaded);
    userManager.events.addUserUnloaded(onUserUnloaded);
    userManager.events.addSilentRenewError(onSilentRenewError);

    return () => {
      mounted = false;
      userManager.events.removeUserLoaded(onUserLoaded);
      userManager.events.removeUserUnloaded(onUserUnloaded);
      userManager.events.removeSilentRenewError(onSilentRenewError);
    };
  }, [hydrateFromOidcUser]);

  const login = async (): Promise<void> => {
    try {
      await userManager.signinRedirect();
    } catch (error) {
      devError('OIDC login redirect failed:', error);
      throw error;
    }
  };

  const register = async (): Promise<void> => {
    throw new Error('User registration is managed in Keycloak.');
  };

  const forgotPassword = async (): Promise<void> => {
    throw new Error('Password reset is managed in Keycloak.');
  };

  const resetPassword = async (): Promise<void> => {
    throw new Error('Password reset is managed in Keycloak.');
  };

  const logout = async (): Promise<void> => {
    const wasLoggedIn = !!user;
    hydrateFromOidcUser(null);

    try {
      await userManager.signoutRedirect();
    } catch (error) {
      devError('OIDC signout redirect failed:', error);
      // Fallback: remove user locally and clear storage.
      await userManager.removeUser();
      clearPersistedAuth();
    }

    if (wasLoggedIn) {
      toast.toast({
        title: 'Logged out',
        description: 'You have been signed out.',
        variant: 'default',
      });
    }
  };

  const refreshToken = useCallback(async (): Promise<void> => {
    try {
      const oidcUser = await userManager.signinSilent();
      if (!oidcUser) {
        throw new Error('Silent authentication returned no user');
      }
      hydrateFromOidcUser(oidcUser);
      devLog('OIDC token refreshed successfully');
    } catch (error) {
      devError('OIDC token refresh failed:', error);
      hydrateFromOidcUser(null);
      throw error;
    }
  }, [hydrateFromOidcUser]);

  const isTokenExpired = (): boolean => {
    if (!tokenExpiresAt) return false;
    return Date.now() >= tokenExpiresAt - 30000;
  };

  const getValidToken = useCallback(async (): Promise<string | null> => {
    const current = oidcUserRef.current;
    if (!current || current.expired) {
      if (!current) {
        // Attempt to load a stored user in case the ref was cleared.
        try {
          const loaded = await userManager.getUser();
          if (loaded && !loaded.expired) {
            hydrateFromOidcUser(loaded);
            return loaded.id_token || loaded.access_token || null;
          }
        } catch (error) {
          devError('Error loading user for valid token:', error);
        }
        return null;
      }
      devLog('OIDC token expired, attempting silent refresh');
      try {
        await refreshToken();
        return oidcUserRef.current?.id_token || oidcUserRef.current?.access_token || null;
      } catch (error) {
        devError('Failed to refresh OIDC token:', error);
        return null;
      }
    }
    return current.id_token || current.access_token || null;
  }, [hydrateFromOidcUser, refreshToken]);

  const computeIsCoreAdmin = (): boolean => {
    const u: any = user;
    if (!u) return false;
    if (u.is_core_admin) return true;
    if (u.isCoreAdmin) return true;
    if (Array.isArray(u.roles) && u.roles.includes('core_admin')) {
      return true;
    }
    if (u.email === 'admin@example.com') return true;
    return false;
  };

  const computeIsAdmin = (): boolean => {
    const u: any = user;
    if (!u) return false;
    if (computeIsCoreAdmin()) return true;
    if (u.is_admin) return true;
    if (Array.isArray(u.roles) && u.roles.includes('admin')) return true;
    if (u.email === 'admin@example.com') return true;
    return false;
  };

  const computeIsGlobalAdmin = (): boolean => {
    const u: any = user;
    if (!u) return false;
    if (u.is_global_admin) return true;
    if (Array.isArray(u.roles) && (u.roles.includes('global_admin') || u.roles.includes('global_ops'))) return true;
    return false;
  };

  const value: AuthContextType = {
    user,
    token,
    refreshTokenValue,
    tokenExpiresAt,
    isAuthenticated: !!user && !!token && !isTokenExpired(),
    isLoading,
    isAdmin: () => computeIsAdmin(),
    isCoreAdmin: () => computeIsCoreAdmin(),
    isGlobalAdmin: () => computeIsGlobalAdmin(),
    canManageCoreAssets: () => computeIsCoreAdmin(),
    canManageCustomAssets: () => computeIsAdmin(),
    login,
    register,
    forgotPassword,
    resetPassword,
    logout,
    refreshToken,
    isTokenExpired,
    getValidToken,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
