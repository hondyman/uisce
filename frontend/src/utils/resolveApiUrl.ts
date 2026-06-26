import { devLog, devDebug, devWarn } from './devLogger';

// Utility to resolve API URLs against configured base (VITE_API_BASE_URL) and
// avoid accidentally resolving to the frontend origin (e.g. Vite dev server).
export function resolveApiUrl(pathOrUrl: string): string {
  const env = (import.meta as any).env || {};
  
  // When VITE_USE_PROXY is enabled, use relative paths so Vite's proxy can intercept them.
  // The proxy will forward /api/* requests to the configured backend.
  const useProxy = String(env?.VITE_USE_PROXY || 'false').toLowerCase() === 'true';
  if (useProxy && pathOrUrl.startsWith('/')) {
    // Small developer-friendly warning: if your VITE_BACKEND_TARGET is set to
    // host.docker.internal (which makes sense when running frontend in Docker),
    // but you run the frontend on your host machine instead, the dev proxy may
    // not reach the backend container. Prefer pointing the backend target at
    // http://localhost:8082 (mapped host port) when the frontend runs locally.
    try {
      const backendTarget = env?.VITE_BACKEND_TARGET || env?.VITE_API_BASE_URL || '';
      if (backendTarget.includes('host.docker.internal') && window && window.location && window.location.hostname === 'localhost') {
        // eslint-disable-next-line no-console
        devWarn('[resolveApiUrl] ⚠ VITE_BACKEND_TARGET points at host.docker.internal. Since you are running the frontend on your host (not in Docker), set VITE_BACKEND_TARGET and VITE_API_BASE_URL to http://localhost:8082 (or appropriate host-mapped port) or use the provided frontend/.env.local.example');
      }
    } catch (e) { }

    // Return relative path to let Vite proxy handle it
    return pathOrUrl;
  }

  let configuredBase: string | undefined = env?.VITE_API_BASE_URL;
  try {
    if (!configuredBase && env?.DEV) {
      configuredBase = 'http://localhost:8001';
    }
  } catch (e) {}

  // If we have a configured base prefer it; otherwise default to current origin
  const base = configuredBase || window.location.origin;
  let final = new URL(pathOrUrl, base);

  // If the computed URL ended up pointing at the frontend origin (for example
  // because `pathOrUrl` was absolute and included `window.location.origin`),
  // rebase the path against the configured base so the browser calls the API
  // gateway instead of the Vite dev server.
  try {
    const frontendOrigin = window.location.origin;
    if (final.origin === frontendOrigin && configuredBase) {
      const baseObj = new URL(configuredBase);
      final = new URL(final.pathname + final.search, baseObj.origin);
    }
  } catch (e) {
    // ignore and keep final as-is
  }

  return final.toString();
}

export default resolveApiUrl;
