export const GOLD_COPY_TENANT_ID = '99e99e99-99e9-49e9-89e9-99e99e99e999';

export function isGoldCopyTenant(tenantId: string | null | undefined): boolean {
  if (!tenantId) return false;
  return tenantId === GOLD_COPY_TENANT_ID;
}

export function isCoreItem(
  itemTenantId: string | null | undefined,
  requestTenantId: string | null | undefined
): boolean {
  if (isGoldCopyTenant(itemTenantId)) return true;
  if (isGoldCopyTenant(requestTenantId)) return true;
  return false;
}
