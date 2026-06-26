/**
 * Helper to get environment variables with support for a fallback default.
 * Handles both Vite's import.meta.env and potentially other env objects.
 *
 * @param _scope - Optional scope prefix (currently unused but kept for compatibility)
 * @param key - The environment variable key (e.g., 'VITE_API_BASE_URL')
 * @param defaultValue - The fallback value if key is not found
 * @returns The environment variable value or the default
 */
export const getEnv = (_scope: string, key: string, defaultValue: string): string => {
    // Vite exposes env vars on import.meta.env
    if (import.meta.env && import.meta.env[key]) {
        return import.meta.env[key] as string;
    }
    return defaultValue;
};
