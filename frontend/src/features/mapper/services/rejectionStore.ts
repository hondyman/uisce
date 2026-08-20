/**
 * Persistent Rejection Memory Store
 * Remembers user-rejected suggestions per tenant so they are never suggested again.
 */

const STORAGE_PREFIX = 'uisce_mapper_rejections_';

function getStorageKey(tenantId: string): string {
  return `${STORAGE_PREFIX}${tenantId || 'global'}`;
}

export function loadRejections(tenantId: string): Set<string> {
  try {
    const raw = localStorage.getItem(getStorageKey(tenantId));
    if (!raw) return new Set();
    const arr = JSON.parse(raw);
    return new Set(Array.isArray(arr) ? arr : []);
  } catch (e) {
    console.error('Failed to load rejection store:', e);
    return new Set();
  }
}

export function saveRejections(tenantId: string, rejections: Set<string>): void {
  try {
    const arr = Array.from(rejections);
    localStorage.setItem(getStorageKey(tenantId), JSON.stringify(arr));
  } catch (e) {
    console.error('Failed to save rejection store:', e);
  }
}

/**
 * Creates a unique rejection key for a source node and target node/name
 */
export function makeRejectionKey(sourceNodeId: string, targetKeyOrName: string): string {
  return `${sourceNodeId}::${(targetKeyOrName || '').toLowerCase().trim()}`;
}

/**
 * Record a rejection
 */
export function recordRejection(tenantId: string, sourceNodeId: string, targetKeyOrName: string): void {
  const current = loadRejections(tenantId);
  const key = makeRejectionKey(sourceNodeId, targetKeyOrName);
  current.add(key);
  saveRejections(tenantId, current);
}

/**
 * Check if a suggestion was previously rejected
 */
export function isSuggestionRejected(
  tenantId: string,
  sourceNodeId: string,
  targetNodeId: string,
  targetNodeName: string
): boolean {
  const rejections = loadRejections(tenantId);
  const keyById = makeRejectionKey(sourceNodeId, targetNodeId);
  const keyByName = makeRejectionKey(sourceNodeId, targetNodeName);
  return rejections.has(keyById) || rejections.has(keyByName);
}

/**
 * Remove a rejection
 */
export function unrecordRejection(tenantId: string, sourceNodeId: string, targetKeyOrName: string): void {
  const current = loadRejections(tenantId);
  const key = makeRejectionKey(sourceNodeId, targetKeyOrName);
  current.delete(key);
  saveRejections(tenantId, current);
}

/**
 * Clear all rejections for a tenant
 */
export function clearAllRejections(tenantId: string): void {
  try {
    localStorage.removeItem(getStorageKey(tenantId));
  } catch (e) {
    console.error('Failed to clear rejections:', e);
  }
}
