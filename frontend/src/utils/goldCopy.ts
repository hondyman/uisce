interface GoldCopyResponse {
  id: string;
  gold_copy: boolean;
  resolved: boolean;
}

let cachedGoldCopyId: string | null = null;
let resolvePromise: Promise<string | null> | null = null;

export async function resolveGoldCopyTenantId(): Promise<string | null> {
  if (cachedGoldCopyId !== null) {
    return cachedGoldCopyId;
  }

  if (resolvePromise !== null) {
    return resolvePromise;
  }

  resolvePromise = fetch('/api/tenants/gold-copy')
    .then(res => {
      if (!res.ok) {
        console.warn('[goldCopy] Failed to resolve gold copy tenant:', res.status);
        return null;
      }
      return res.json() as Promise<GoldCopyResponse>;
    })
    .then(data => {
      if (data && data.resolved && data.id) {
        cachedGoldCopyId = data.id;
        return cachedGoldCopyId;
      }
      return null;
    })
    .catch(err => {
      console.warn('[goldCopy] Error resolving gold copy tenant:', err);
      return null;
    })
    .finally(() => {
      resolvePromise = null;
    });

  return resolvePromise;
}

export async function isGoldCopyTenant(tenantId: string | null | undefined): Promise<boolean> {
  if (!tenantId) return false;
  const goldCopyId = await resolveGoldCopyTenantId();
  if (!goldCopyId) return false;
  return tenantId === goldCopyId;
}

export async function isCoreItem(
  itemTenantId: string | null | undefined,
  requestTenantId: string | null | undefined
): Promise<boolean> {
  if (await isGoldCopyTenant(itemTenantId)) return true;
  if (await isGoldCopyTenant(requestTenantId)) return true;
  return false;
}

export function isGoldCopyTenantSync(tenantId: string | null | undefined): boolean {
  if (!tenantId) return false;
  return cachedGoldCopyId !== null && tenantId === cachedGoldCopyId;
}

export function isCoreItemSync(
  itemTenantId: string | null | undefined,
  requestTenantId: string | null | undefined
): boolean {
  if (isGoldCopyTenantSync(itemTenantId)) return true;
  if (isGoldCopyTenantSync(requestTenantId)) return true;
  return false;
}

export function getCachedGoldCopyId(): string | null {
  return cachedGoldCopyId;
}
