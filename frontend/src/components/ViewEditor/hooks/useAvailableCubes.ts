import { useState, useCallback } from 'react';

export const useAvailableCubes = (
  isValidTenantScope: () => boolean,
  tenantId?: string,
  datasourceId?: string,
  viewData?: any
) => {
  const [availableCubes, setAvailableCubes] = useState<any[]>([]);

  const fetchAvailableCubes = useCallback(async () => {
    if (!isValidTenantScope()) {
      setAvailableCubes([]);
      return;
    }

    try {
      const response = await fetch(`/api/fabric/models?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`);

      if (!response.ok) {
        setAvailableCubes([]);
        return;
      }

      const data = await response.json();

      if (Array.isArray(data?.models)) {
        const existingCubeIds = new Set((viewData?.cubes || []).map((c: any) => String(c.id || c).toLowerCase()));
        const available = data.models.filter((cube: any) => !existingCubeIds.has(String(cube.id).toLowerCase()));
        setAvailableCubes(available);
      } else {
        setAvailableCubes([]);
      }
    } catch (error) {
      setAvailableCubes([]);
    }
  }, [isValidTenantScope, tenantId, datasourceId, viewData]);

  return { availableCubes, fetchAvailableCubes };
};