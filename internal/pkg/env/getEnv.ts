/**
 * Get environment variable with fallback support
 * Works in Node.js environments (for Docker/server builds)
 */
export function getEnv(legacyKey: string, viteKey: string, defaultValue?: string): string | undefined {
  // Node-side environment variable lookup
  if (typeof process !== 'undefined' && process.env) {
    // Try legacy key first
    const legacy = process.env[legacyKey];
    if (legacy) return legacy;

    // Try Vite-style key as fallback (in case it's set in Node env)
    const vite = process.env[viteKey];
    if (vite) return vite;
  }

  return defaultValue;
}
