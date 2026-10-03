import { useEffect, useState } from 'react';
import { useTenant } from '../contexts/TenantContext';
import { resolveGoldCopyTenantId } from '../utils/goldCopy';

interface UseCanEditCoreItemOptions {
  itemTenantId?: string | null;
}

interface UseCanEditCoreItemResult {
  isCore: boolean;
  canEdit: boolean;
  disabledReason: string | null;
  isGoldCopyTenant: boolean;
}

/**
 * Advisory UI gate only; the backend enforces core-item immutability.
 *
 * The gold-copy tenant id is resolved from the API (no hardcoded id), so until
 * it is known we cannot tell a core item from a tenant's own. Fail closed in
 * that window: never report an edit as allowed on an id we have not resolved.
 */
export function useCanEditCoreItem(
  item: { tenant_id?: string | null; isCore?: boolean; is_core?: boolean } | null | undefined,
  options?: UseCanEditCoreItemOptions
): UseCanEditCoreItemResult {
  const { tenant } = useTenant();
  const [goldCopyId, setGoldCopyId] = useState<string | null | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    resolveGoldCopyTenantId().then(id => {
      if (!cancelled) setGoldCopyId(id);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const itemTenantId = options?.itemTenantId ?? item?.tenant_id;
  const flaggedCore = Boolean(item?.isCore ?? item?.is_core);
  const resolved = goldCopyId !== undefined;
  const isGoldCopyTenant = Boolean(goldCopyId) && tenant?.id === goldCopyId;
  const ownedByGoldCopy = Boolean(goldCopyId) && itemTenantId === goldCopyId;
  const isCore = flaggedCore || ownedByGoldCopy || isGoldCopyTenant;

  if (!resolved && !flaggedCore) {
    return { isCore: false, canEdit: false, disabledReason: null, isGoldCopyTenant: false };
  }

  if (!isCore) {
    return { isCore: false, canEdit: true, disabledReason: null, isGoldCopyTenant };
  }

  if (isGoldCopyTenant) {
    return { isCore: true, canEdit: true, disabledReason: null, isGoldCopyTenant };
  }

  return {
    isCore: true,
    canEdit: false,
    disabledReason: 'Core items owned by the gold copy tenant cannot be edited.',
    isGoldCopyTenant,
  };
}

export default useCanEditCoreItem;
