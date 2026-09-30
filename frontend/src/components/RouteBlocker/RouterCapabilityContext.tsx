import React, { createContext, useContext, useMemo } from 'react';
import { useBlockableNavigate } from './useBlockableNavigate';
import { useRouteBlocker } from './RouteBlocker';
import { useExtensionsService, type ValidationIssue } from '../../services/extensions';

type BlockHandler = (tx: any) => Promise<boolean> | boolean;
type RouteBlockerShape = {
  register: (h: BlockHandler) => () => void;
  run: (tx: any) => Promise<boolean>;
};
type BlockableNavigateFn = (to: string | number, options?: { replace?: boolean; state?: any }) => Promise<boolean>;
// Mirrors the service's own signature exactly, so no cast is needed to publish it.
type ValidateExtensionFn = (datasourceId: string, payload: Record<string, any>) => Promise<{ issues: ValidationIssue[] }>;

export interface RouterCapability {
  /**
   * Null outside a Router. Matches the `catch { navigate = null }` fallback these
   * call sites used before, byte for byte.
   */
  navigate: BlockableNavigateFn | null;
  /**
   * Never null. The no-op object is exactly what `useRouteBlocker` already returns
   * when its context is absent (see RouteBlocker.tsx), so this preserves current
   * behaviour: outside a Router the blocker is a harmless pass-through, not null.
   */
  blocker: RouteBlockerShape;
  /**
   * Null outside a Router. `useExtensionsService` reaches the router through
   * `useAuthFetch` -> `useBlockableNavigate` + `useLocation`, so it cannot be called
   * outside one. Null matches the previous
   * `catch { validateExtension: async () => ({ issues: [] }) }` fallback.
   */
  validateExtension: ValidateExtensionFn | null;
}

/**
 * The degradation contract. This is a module-level singleton on purpose: consumers
 * compare against it by identity, and a stable identity means effects that depend on
 * the blocker do not re-run on every render (today's `routeBlocker = null` was also
 * stable, so this is identity-stable in the same way).
 */
const DEGRADED_ROUTER_CAPABILITY: RouterCapability = {
  navigate: null,
  blocker: {
    register: (_h: BlockHandler) => () => {},
    run: async (_tx: any) => true,
  },
  validateExtension: null,
};

const RouterCapabilityContext = createContext<RouterCapability>(DEGRADED_ROUTER_CAPABILITY);

// Dedupe per capability, not per wave: a missing Router is one root cause but two
// independently-callable capabilities, and each should warn once, not once per render.
let warnedNavigate = false;
let warnedValidateExtension = false;

const warnOnce = (flag: 'navigate' | 'validateExtension', message: string) => {
  if (process.env.NODE_ENV === 'production') return;
  if (flag === 'navigate') {
    if (warnedNavigate) return;
    warnedNavigate = true;
  } else {
    if (warnedValidateExtension) return;
    warnedValidateExtension = true;
  }
  // eslint-disable-next-line no-console
  console.warn(message);
};

/**
 * Read router-dependent capabilities without ever calling a router hook.
 *
 * These call sites used to wrap `useBlockableNavigate()` / `useExtensionsService()` in
 * `try/catch` and fall back when the throw came from a missing Router. A hook inside a
 * `try` is a conditional hook call: the hook count depended on whether a Router was
 * mounted, which throws "Rendered fewer hooks than expected" when that changes. Absence
 * is now detected by reading this context instead of catching a throw.
 *
 * Outside <RouterCapabilityProvider> this returns the degraded capability, so these
 * components keep rendering — including in tests that mount them bare.
 */
export const useRouterCapability = (): RouterCapability => {
  const capability = useContext(RouterCapabilityContext);
  if (capability === DEGRADED_ROUTER_CAPABILITY) {
    warnOnce(
      'navigate',
      '[RouterCapability] No <RouterCapabilityProvider> found. Router-dependent navigation and '
        + 'extension validation are disabled. This provider must be mounted inside the Router; '
        + 'check that it is not being mocked away or rendered above <BrowserRouter>.',
    );
    warnOnce(
      'validateExtension',
      '[RouterCapability] No <RouterCapabilityProvider> found: extension validation is disabled.',
    );
  }
  return capability;
};

/**
 * Must be mounted INSIDE the Router. This is the single place that calls the
 * router-dependent hooks, and it calls them unconditionally, so it cannot throw:
 * `useNavigate()` throws only via `useInRouterContext()` (verified in
 * react-router 6.24 — there is no other throw path), and `useRouteBlocker` never throws.
 */
export const RouterCapabilityProvider: React.FC<{ children?: React.ReactNode }> = ({ children }) => {
  const navigate = useBlockableNavigate();
  const blocker = useRouteBlocker();
  const { validateExtension } = useExtensionsService();

  const value = useMemo<RouterCapability>(
    () => ({ navigate, blocker, validateExtension }),
    [navigate, blocker, validateExtension],
  );

  return <RouterCapabilityContext.Provider value={value}>{children}</RouterCapabilityContext.Provider>;
};

export default RouterCapabilityProvider;
