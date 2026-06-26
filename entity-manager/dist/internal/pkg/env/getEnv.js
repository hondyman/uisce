export function getEnv(nodeKey, viteKey, defaultValue = '') {
    if (typeof process !== 'undefined' && process.env) {
        if (process.env[nodeKey]) {
            return process.env[nodeKey];
        }
        if (viteKey && process.env[viteKey]) {
            return process.env[viteKey];
        }
    }
    return defaultValue;
}
//# sourceMappingURL=getEnv.js.map