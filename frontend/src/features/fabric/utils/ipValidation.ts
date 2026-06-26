/**
 * IP Pattern Validation Utilities
 * Frontend helper: validate IPv4 dotted string with '*' allowed in octets
 */

export function isValidIpOrWildcard(s: string): boolean {
  const parts = s.split('.');
  if (parts.length !== 4) return false;
  for (const p of parts) {
    if (p === '*') continue;
    const n = Number(p);
    if (!Number.isInteger(n) || n < 0 || n > 255) return false;
  }
  return true;
}

/**
 * Frontend overlap check mirroring backend logic
 * Determines if two IP patterns overlap
 */
export function ipPatternsOverlap(a: string, b: string): boolean {
  const pa = a.split('.');
  const pb = b.split('.');
  if (pa.length !== 4 || pb.length !== 4) return false;
  for (let i = 0; i < 4; i++) {
    if (pa[i] === '*' || pb[i] === '*') continue;
    if (pa[i] !== pb[i]) return false;
  }
  return true;
}

/**
 * Map tenant IDs to display names using the loaded tenants list
 */
export function mapTenantIdsToNames(
  tenantIds: string[],
  tenants: { id: string; displayName: string }[]
): string[] {
  return tenantIds.map(id => tenants.find(t => t.id === id)?.displayName || id);
}
