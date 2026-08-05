import { useTenant } from '../contexts/TenantContext';
import { isCoreItem, GOLD_COPY_TENANT_ID } from '../utils/goldCopy';

interface UseCanEditCoreItemOptions {
  itemTenantId?: string | null;
}

interface UseCanEditCoreItemResult {
  isCore: boolean;
  canEdit: boolean;
  disabledReason: string | null;
  isGoldCopyTenant: boolean;
}

export function useCanEditCoreItem(
  item: { tenant_id?: string | null; isCore?: boolean; is_core?: boolean } | null | undefined,
  options?: UseCanEditCoreItemOptions
): UseCanEditCoreItemResult {
  const tenant = useTenant();

  const itemTenantId = options?.itemTenantId ?? item?.tenant_id;
  const isCoreItemDetected = Boolean(item?.isCore ?? item?.is_core);
  const isGoldCopyTenant = tenant?.id === GOLD_COPY_TENANT_ID;

  const isCore = isCoreItemDetected || isCoreItem(itemTenantId, tenant?.id);

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
