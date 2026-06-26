/**
 * Get environment variable with fallback support
 * Works in both Node.js and Vite environments
 */
export function getEnv(nodeKey: string, viteKey?: string, defaultValue: string = ''): string {
    // Node.js environment
    if (typeof process !== 'undefined' && process.env) {
        if (process.env[nodeKey]) {
            return process.env[nodeKey] as string;
        }
        if (viteKey && process.env[viteKey]) {
            return process.env[viteKey] as string;
        }
    }

    return defaultValue;
}
