import React, { ReactNode } from 'react';
import { useAuth } from '../contexts/AuthContext';

interface RequireCapabilityProps {
  /** Capability key returned by /api/auth/me/entitlements (e.g. "menu:platform") */
  capability: string;
  /** Optional fallback rendered while entitlements are loading. */
  fallback?: ReactNode;
  children: ReactNode;
}

/**
 * Capability-driven access guard.  The component knows nothing about roles;
 * it simply checks the capability map returned by the backend ABAC engine.
 * If the capability is missing or false, the children are silently dropped
 * from the DOM.
 */
export const RequireCapability: React.FC<RequireCapabilityProps> = ({
  capability,
  fallback = null,
  children,
}) => {
  const { entitlements, entitlementsLoading } = useAuth();

  if (entitlementsLoading) {
    return <>{fallback}</>;
  }

  if (!entitlements?.capabilities[capability]) {
    return null;
  }

  return <>{children}</>;
};

export default RequireCapability;
