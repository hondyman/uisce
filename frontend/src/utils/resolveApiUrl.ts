import { devLog, devDebug, devWarn } from './devLogger';

// Utility to resolve API URLs against configured base (VITE_API_BASE_URL) and
// avoid accidentally resolving to the frontend origin (e.g. Vite dev server).
//
// Resolution priority:
//   1. If VITE_USE_PROXY=true (set in .env.local), return the relative path
//      so Vite's proxy (configured in vite.config.ts) intercepts /api requests
//      and forwards them to the platform backend. This is the recommended dev
//      setup and is the path that fixes the
//        net::ERR_CONNECTION_REFUSED (port 8001) and 404 (port 5173)
//      errors that the Business Object Details page was producing.
//   2. If VITE_API_BASE_URL is set, resolve against that base URL.
//   3. In dev mode (no env vars configured), return the relative path so the
//      always-on Vite proxy still intercepts /api requests. Falling back to
//      window.location.origin here would route /api calls to the Vite dev
//      server itself (port 5173), which is what previously produced 404s.
//   4. In production, default to window.location.origin (API reverse-proxied
//      at the same host is the recommended production deployment).
export function resolveApiUrl(pathOrUrl: string): string {
  const env = (import.meta as any).env || {};

  // Pass absolute URLs (http://..., https://...) through unchanged.
  if (/^https?:\/\//i.test(pathOrUrl)) {
    return pathOrUrl;
  }

  // 1. VITE_USE_PROXY=true — use relative paths and let Vite proxy forward.
  const useProxy = String(env?.VITE_USE_PROXY || 'false').toLowerCase() === 'true';
  if (useProxy && pathOrUrl.startsWith('/')) {
    // Small developer-friendly warning: if your VITE_BACKEND_TARGET is set to
    // host.docker.internal (which makes sense when running frontend in Docker),
    // but you run the frontend on your host machine instead, the dev proxy may
    // not reach the backend container. Prefer pointing the backend target at
    // http://localhost:8082 (mapped host port) when the frontend runs locally.
    try {
      const backendTarget = env?.VITE_BACKEND_TARGET || env?.VITE_API_BASE_URL || '';
      if (
        backendTarget.includes('host.docker.internal') &&
        typeof window !== 'undefined' &&
        window.location &&
        window.location.hostname === 'localhost'
      ) {
        devWarn(
          '[resolveApiUrl] ⚠ VITE_BACKEND_TARGET points at host.docker.internal. ' +
          'Since you are running the frontend on your host (not in Docker), ' +
          'set VITE_BACKEND_TARGET and VITE_API_BASE_URL to http://localhost:8082 ' +
          '(or appropriate host-mapped port) or use the provided frontend/.env.local.example'
        );
      }
    } catch (e) {
      // ignore — best-effort warning
    }

    return pathOrUrl;
  }

  // 2. Resolve against the configured API base, if any.
  const configuredBase: string | undefined = env?.VITE_API_BASE_URL;
  if (configuredBase) {
    try {
      return new URL(pathOrUrl, configuredBase).toString();
    } catch (e) {
      // fall through to dev/prod defaults
    }
  }

  // 3. Dev mode with no env vars: return relative path so the always-on Vite
  //    proxy (configured in vite.config.ts) intercepts /api/* requests.
  // 4. Production: default to the current origin (assumes reverse-proxy).
  try {
    if (env?.DEV && pathOrUrl.startsWith('/')) {
      return pathOrUrl;
    }
  } catch (e) {
    // ignore
  }

  return new URL(pathOrUrl, window.location.origin).toString();
}

export default resolveApiUrl;