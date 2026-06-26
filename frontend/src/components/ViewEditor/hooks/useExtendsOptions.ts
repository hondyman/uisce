import { useState, useCallback, useRef } from 'react';
import { ExtendsOption } from './useAvailableSources';
import { fetchViews } from '../../../services/viewsService';

export const useExtendsOptions = (
  isValidTenantScope: () => boolean,
  tenantId?: string,
  datasourceId?: string,
  viewName?: string,
  viewData?: any
) => {
  const [extendsOptions, setExtendsOptions] = useState<ExtendsOption[]>([]);
  const [extendsLoading, setExtendsLoading] = useState(false);
  const extendsFetchIdRef = useRef(0);

  const fetchExtendsOptions = useCallback(
    async (query: string) => {
      if (!isValidTenantScope()) {
        setExtendsOptions([]);
        return;
      }

      const fetchId = ++extendsFetchIdRef.current;
      setExtendsLoading(true);

      try {
        const views = await fetchViews({
          tenantId,
          datasourceId,
          pageSize: 100,
          status: 'published',
          q: (query || '').trim(),
        });

        if (fetchId !== extendsFetchIdRef.current) return;

        const options: ExtendsOption[] = [];
        if (Array.isArray(views)) {
          for (const view of views) {
            if (!view || !view.id) continue;

            const currentId = String(viewData?.id || '').toLowerCase();
            const candidateId = String(view.id).toLowerCase();
            if (currentId && candidateId && currentId === candidateId) continue;

            const currentName = String((viewData?.name || viewName || '')).toLowerCase();
            const candidateName = String(view.name || '').toLowerCase();
            if (currentName && candidateName && currentName === candidateName) continue;

            options.push({
              id: String((view as Record<string, unknown>).id || ''),
              name: String((view as Record<string, unknown>).name || ''),
              title: String((view as Record<string, unknown>).title || ''),
              description: String((view as Record<string, unknown>).description || ''),
              isCore: Boolean((view as Record<string, unknown>).is_core || (view as Record<string, unknown>).isCore),
              isCustom: Boolean((view as Record<string, unknown>).is_custom || (view as Record<string, unknown>).isCustom),
            });
          }
        }

        setExtendsOptions(options);
      } catch (err) {
        setExtendsOptions([]);
      } finally {
        if (fetchId === extendsFetchIdRef.current) setExtendsLoading(false);
      }
    },
    [isValidTenantScope, tenantId, datasourceId, viewName, viewData]
  );

  return { extendsOptions, extendsLoading, fetchExtendsOptions };
};