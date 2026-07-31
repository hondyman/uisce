import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface IdentityContextValue {
  tenantId: string;
  userId: string;
  tenantIds: string[];
  email: string;
  functionalRole: string;
  clearanceLevel: string;
  roles: string[];
  isLoading: boolean;
  error: string | null;
}

const IdentityContext = createContext<IdentityContextValue | null>(null);

export function useIdentityContext(): IdentityContextValue {
  const ctx = useContext(IdentityContext);
  if (!ctx) {
    return {
      tenantId: 'anonymous',
      userId: 'anonymous',
      tenantIds: [],
      email: '',
      functionalRole: 'guest',
      clearanceLevel: 'restricted',
      roles: [],
      isLoading: false,
      error: 'IdentityContext not mounted',
    };
  }
  return ctx;
}

interface IdentityContextProviderProps {
  children: ReactNode;
}

export const IdentityContextProvider: React.FC<IdentityContextProviderProps> = ({ children }) => {
  const [state, setState] = useState<IdentityContextValue>({
    tenantId: '',
    userId: '',
    tenantIds: [],
    email: '',
    functionalRole: 'analyst',
    clearanceLevel: 'standard',
    roles: [],
    isLoading: true,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        const res = await fetch('/api/v1/personalization/context', {
          credentials: 'include',
        });
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`);
        }
        const data = await res.json();
        if (!cancelled) {
          setState({
            tenantId: data.tenant_id || '',
            userId: data.user_id || '',
            tenantIds: data.tenant_ids || [],
            email: data.email || '',
            functionalRole: data.functional_role || 'analyst',
            clearanceLevel: data.clearance_level || 'standard',
            roles: data.roles || [],
            isLoading: false,
            error: null,
          });
        }
      } catch (e: any) {
        if (!cancelled) {
          setState((prev) => ({
            ...prev,
            isLoading: false,
            error: e.message || 'Failed to load identity context',
          }));
        }
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  return <IdentityContext.Provider value={state}>{children}</IdentityContext.Provider>;
};

export function useABACFilter<T extends { label: string; clearanceLevel?: string }>(commands: T[]): T[] {
  const { clearanceLevel } = useIdentityContext();

  if (clearanceLevel === 'elevated' || clearanceLevel === 'confidential') {
    return commands;
  }

  const restrictedCommands = commands.filter((cmd) => {
    if (!cmd.clearanceLevel) return true;
    if (cmd.clearanceLevel === 'standard') return true;
    if (cmd.clearanceLevel === 'restricted') return true;
    return false;
  });

  return restrictedCommands;
}
