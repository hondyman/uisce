import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { devLog, devError } from '../utils/devLogger';
import { apiPost } from '../utils/api';
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
  /** Tenant assignments for multi-tenant access control */
  tenant_assignments?: Array<{
    tenantId: string;
    tenantName?: string;
    accessLevel: 'platform_operator' | 'tenant_admin' | 'tenant_user';
    isReadOnly?: boolean;
  }>;
}

interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  expires_at?: number; // Add expiration timestamp
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  refreshTokenValue: string | null;
  tokenExpiresAt: number | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  // convenience helper to determine admin status
  isAdmin: () => boolean;
  isCoreAdmin: () => boolean;
  canManageCoreAssets: () => boolean;
  canManageCustomAssets: () => boolean;
  login: (email: string, password: string) => Promise<void>;
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

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [refreshTokenValue, setRefreshTokenValue] = useState<string | null>(null);
  const [tokenExpiresAt, setTokenExpiresAt] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Load auth state from localStorage on mount
  useEffect(() => {
    const loadAuthState = () => {
      try {
        const storedToken = localStorage.getItem('auth_token');
        const storedRefreshToken = localStorage.getItem('auth_refresh_token');
        const storedUser = localStorage.getItem('auth_user');
        const storedExpiresAt = localStorage.getItem('auth_expires_at');

        if (storedToken && storedUser) {
          setToken(storedToken);
          setRefreshTokenValue(storedRefreshToken);
          setUser(JSON.parse(storedUser));
          setTokenExpiresAt(storedExpiresAt ? parseInt(storedExpiresAt) : null);
        }
      } catch (error) {
        devError('Error loading auth state from localStorage:', error);
        // Clear invalid data
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_refresh_token');
        localStorage.removeItem('auth_user');
        localStorage.removeItem('auth_expires_at');
      } finally {
        setIsLoading(false);
      }
    };

    loadAuthState();
  }, []);

  const login = async (email: string, password: string): Promise<void> => {
    try {
      const authData: AuthResponse = await apiPost('auth/login', { email, password });

      // Calculate expiration time
      const expiresAt = authData.expires_at || (Date.now() + (authData.expires_in * 1000));

      // Store auth data
      setUser(authData.user);
      setToken(authData.access_token);
      setRefreshTokenValue(authData.refresh_token);
      setTokenExpiresAt(expiresAt);

      // Persist to localStorage
      localStorage.setItem('auth_token', authData.access_token);
      localStorage.setItem('auth_refresh_token', authData.refresh_token);
      localStorage.setItem('auth_user', JSON.stringify(authData.user));
      localStorage.setItem('auth_expires_at', expiresAt.toString());

      devLog('User logged in successfully:', authData.user.email);
    } catch (error) {
      devLog('Login error:', error);
      throw error;
    }
  };

  const register = async (email: string, password: string, name: string, organization?: string): Promise<void> => {
    try {
      const authData: AuthResponse = await apiPost('auth/register', { 
        email, 
        password, 
        name, 
        organization: organization || 'Default Organization' 
      });

      // Calculate expiration time
      const expiresAt = authData.expires_at || (Date.now() + (authData.expires_in * 1000));

      // Store auth data
      setUser(authData.user);
      setToken(authData.access_token);
      setRefreshTokenValue(authData.refresh_token);
      setTokenExpiresAt(expiresAt);

      // Persist to localStorage
      localStorage.setItem('auth_token', authData.access_token);
      localStorage.setItem('auth_refresh_token', authData.refresh_token);
      localStorage.setItem('auth_user', JSON.stringify(authData.user));
      localStorage.setItem('auth_expires_at', expiresAt.toString());

      devLog('User registered successfully:', authData.user.email);
    } catch (error) {
      devLog('Registration error:', error);
      throw error;
    }
  };

  const forgotPassword = async (email: string): Promise<void> => {
    try {
      await apiPost('auth/forgot-password', { email });
      devLog('Password reset email sent to:', email);
    } catch (error) {
      devLog('Forgot password error:', error);
      throw error;
    }
  };

  const resetPassword = async (token: string, newPassword: string): Promise<void> => {
    try {
      await apiPost('auth/reset-password', { token, password: newPassword });
      devLog('Password reset successfully');
    } catch (error) {
      devLog('Reset password error:', error);
      throw error;
    }
  };

  const toast = useToast()

  const logout = async () => {
    // Clear local auth state first to ensure user can always log out
    const wasLoggedIn = !!user;
    setUser(null);
    setToken(null);
    setRefreshTokenValue(null);
    setTokenExpiresAt(null);
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_refresh_token');
    localStorage.removeItem('auth_user');
    localStorage.removeItem('auth_expires_at');

    // Show toast immediately after clearing
    if (wasLoggedIn) {
      toast.toast({ title: 'Logged out', description: 'You have been signed out.', variant: 'default' });
      // Attempt to notify backend to invalidate session (don't await to avoid blocking)
      apiPost('auth/logout', {}).catch((err) => {
        devLog('Logout request failed:', err);
      });
    }

    devLog('User logged out');
  };

  const refreshToken = async (): Promise<void> => {
    if (!refreshTokenValue) {
      devLog('No refresh token available');
      throw new Error('No refresh token available');
    }

    try {
      const response = await apiPost('auth/refresh', { refresh_token: refreshTokenValue });
      
      // Calculate new expiration time
      const expiresAt = response.expires_at || (Date.now() + (response.expires_in * 1000));

      // Update auth data
      setToken(response.access_token);
      setRefreshTokenValue(response.refresh_token);
      setTokenExpiresAt(expiresAt);

      // Update localStorage
      localStorage.setItem('auth_token', response.access_token);
      localStorage.setItem('auth_refresh_token', response.refresh_token);
      localStorage.setItem('auth_expires_at', expiresAt.toString());

      devLog('Token refreshed successfully');
    } catch (error) {
      devError('Token refresh failed:', error);
      // Clear tokens on refresh failure
      setToken(null);
      setRefreshTokenValue(null);
      setTokenExpiresAt(null);
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_refresh_token');
      localStorage.removeItem('auth_expires_at');
      throw error;
    }
  };

  const isTokenExpired = (): boolean => {
    if (!tokenExpiresAt) return false;
    // Add 30 second buffer before expiration
    return Date.now() >= (tokenExpiresAt - 30000);
  };

  const getValidToken = async (): Promise<string | null> => {
    if (!token) return null;
    
    if (isTokenExpired()) {
      devLog('Token expired, attempting refresh');
      try {
  await refreshToken();
  // Read the latest token from localStorage to avoid stale state
  const newToken = localStorage.getItem('auth_token');
  return newToken;
      } catch (error) {
        devError('Failed to refresh token:', error);
        // Don't throw - return null so caller can handle gracefully
        return null;
      }
    }
    
    return token;
  };

  const computeIsCoreAdmin = (): boolean => {
    const u: any = user;
    if (!u) return false;
    if (u.is_core_admin !== undefined) {
      if (u.is_core_admin) return true;
    }
    if (u.isCoreAdmin !== undefined) {
      if (u.isCoreAdmin) return true;
    }
    if (Array.isArray(u.roles) && typeof u.roles.includes === 'function') {
      if (u.roles.includes('core_admin') || u.roles.includes('core-admin')) {
        return true;
      }
    }
    if (u.email === 'admin@example.com') {
      return true;
    }
    return false;
  };

  const computeIsAdmin = (): boolean => {
    const u: any = user;
    if (!u) return false;
    if (computeIsCoreAdmin()) return true;
    if (u.is_admin) return true;
    if (u.isAdmin) return true;
    if (Array.isArray(u.roles) && typeof u.roles.includes === 'function') {
      if (u.roles.includes('admin')) return true;
    }
    if (u.email === 'admin@example.com') return true;
    return false;
  };

  const value: AuthContextType = {
    user,
    token,
    refreshTokenValue,
    tokenExpiresAt,
  // Treat expired tokens as unauthenticated so ProtectedRoute can redirect
  isAuthenticated: !!user && !!token && !isTokenExpired(),
    isLoading,
    isAdmin: () => computeIsAdmin(),
    isCoreAdmin: () => computeIsCoreAdmin(),
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

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
