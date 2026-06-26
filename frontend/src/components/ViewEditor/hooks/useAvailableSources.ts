import { useState, useCallback } from 'react';
import { fetchViews } from '../../../services/viewsService';

export interface AvailableSource {
  id: string;
  name: string;
  type: 'cube' | 'extended_view';
  items: AvailableItem[];
  expanded?: boolean;
  filteredOutCount?: number;
  isCore?: boolean;
  isCustom?: boolean;
  promoted?: boolean;
}

export interface AvailableItem {
  id: string;
  name: string;
  type: 'dimension' | 'measure' | 'view';
  source: string;
  description?: string;
  datatype?: string;
}

export interface ExtendsOption {
  id?: string;
  name: string;
  title?: string;
  description?: string;
  isCore?: boolean;
  isCustom?: boolean;
}

export const useAvailableSources = (
  isValidTenantScope: () => boolean,
  tenantId?: string,
  datasourceId?: string,
  viewData?: any,
  selectedRefs?: Set<string>
) => {
  const [availableSources, setAvailableSources] = useState<AvailableSource[]>([]);
  const [sourcesLoading, setSourcesLoading] = useState(false);

  const fetchAvailableSources = useCallback(async () => {
    if (!isValidTenantScope()) {
      setAvailableSources([]);
      return;
    }

    setSourcesLoading(true);
    try {
      // Fetch cubes
      const cubesResponse = await fetch(`/api/fabric/models?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`);
      const cubesData = await cubesResponse.json();

      // Fetch extended views
      const extendsParams = new URLSearchParams();
      extendsParams.set('tenant_id', tenantId!);
      extendsParams.set('tenant_instance_id', datasourceId!);
      extendsParams.set('page_size', '200');
      // Use centralized fetchViews helper which respects tenant/datasource/status/q
      let extendsData: any = null;
      try {
        const views = await fetchViews({ tenantId, datasourceId, pageSize: 200 });
        extendsData = { views };
      } catch (e) {
        extendsData = null;
      }

      const sources: AvailableSource[] = [];

      // Process cubes
      if (Array.isArray(cubesData?.models)) {
        cubesData.models.forEach((cube: any) => {
          const cubeId = cube?.id ? String(cube.id).toLowerCase() : '';
          const cubeKey = cube?.model_key ? String(cube.model_key).toLowerCase() : '';
          const matches = (cubeId && selectedRefs?.has(cubeId)) || (cubeKey && selectedRefs?.has(cubeKey));
          if (!matches) return;

          const dimensions: AvailableItem[] = [];
          const measures: AvailableItem[] = [];

          const config = cube.resolved_config?.cubes?.[0];
          if (config) {
            const existingDimensionIds = new Set<string>((viewData?.dimensions || []).map((d: any) => (d?.id || d?.qualifiedName || d?.name || '').toLowerCase()));
            const existingMeasureIds = new Set<string>((viewData?.measures || []).map((m: any) => (m?.id || m?.qualifiedName || m?.name || '').toLowerCase()));

            if (config.dimensions) {
              Object.entries(config.dimensions).forEach(([key, dim]: [string, any]) => {
                const qid = `${cube.id}.${key}`.toLowerCase();
                if (existingDimensionIds.has(qid)) return;
                dimensions.push({
                  id: `${cube.id}.${key}`,
                  name: dim.title || key,
                  type: 'dimension',
                  source: cube.id,
                  description: dim.description,
                  datatype: dim.datatype || dim.type,
                });
              });
            }
            if (config.measures) {
              Object.entries(config.measures).forEach(([key, measure]: [string, any]) => {
                const qid = `${cube.id}.${key}`.toLowerCase();
                if (existingMeasureIds.has(qid)) return;
                measures.push({
                  id: `${cube.id}.${key}`,
                  name: measure.title || key,
                  type: 'measure',
                  source: cube.id,
                  description: measure.description,
                  datatype: measure.datatype || measure.type,
                });
              });
            }
          }

          if (dimensions.length > 0 || measures.length > 0) {
            const totalCount = ((config?.dimensions && Object.keys(config.dimensions).length) || 0) + ((config?.measures && Object.keys(config.measures).length) || 0);
            const shownCount = dimensions.length + measures.length;
            const filteredOutCount = Math.max(0, totalCount - shownCount);

            sources.push({
              id: cube.id,
              name: cube.display_name || cube.model_key,
              type: 'cube',
              items: [...dimensions, ...measures],
              expanded: false,
              filteredOutCount,
              isCore: Boolean(cube.is_core),
              isCustom: Boolean(cube.is_custom),
            });
          }
        });
      }

      // Process extended views
      if (Array.isArray(extendsData?.views)) {
            extendsData.views.forEach((view: any) => {
              if (!view || !view.id) return;

              const viewId = String(view.id).toLowerCase();
              const currentExtendsId = (viewData && viewData.extends && typeof viewData.extends === 'object') ? String(viewData.extends.id || viewData.extends.ID || viewData.extends.name || '').toLowerCase() : (typeof viewData?.extends === 'string' ? String(viewData.extends).toLowerCase() : '');
              const isSelectedExtends = currentExtendsId === viewId;

          const existingDimensionIds = new Set<string>((viewData?.dimensions || []).map((d: any) => (d?.id || d?.qualifiedName || d?.name || '').toLowerCase()));
          const existingMeasureIds = new Set<string>((viewData?.measures || []).map((m: any) => (m?.id || m?.qualifiedName || m?.name || '').toLowerCase()));

          const items: AvailableItem[] = [];
          if (view.dimensions) {
            view.dimensions.forEach((dim: any, index: number) => {
              const qid = `${view.id}.${dim.name || `dimension_${index}`}`.toLowerCase();
              if (existingDimensionIds.has(qid)) return;
              items.push({
                id: `${view.id}.${dim.name || `dimension_${index}`}`,
                name: dim.title || dim.name || `Dimension ${index + 1}`,
                type: 'dimension',
                source: view.id,
                description: dim.description,
                datatype: dim.datatype || dim.type,
              });
            });
          }
          if (view.measures) {
            view.measures.forEach((measure: any, index: number) => {
              const qid = `${view.id}.${measure.name || `measure_${index}`}`.toLowerCase();
              if (existingMeasureIds.has(qid)) return;
              items.push({
                id: `${view.id}.${measure.name || `measure_${index}`}`,
                name: measure.title || measure.name || `Measure ${index + 1}`,
                type: 'measure',
                source: view.id,
                description: measure.description,
                datatype: measure.datatype || measure.type,
              });
            });
          }

          if (items.length > 0) {
            const totalCount = ((view.dimensions && view.dimensions.length) || 0) + ((view.measures && view.measures.length) || 0);
            const shownCount = items.length;
            const filteredOutCount = Math.max(0, totalCount - shownCount);
            sources.push({
              id: view.id,
              name: view.title || view.name,
              type: 'extended_view',
              items,
              expanded: isSelectedExtends,
              filteredOutCount,
              isCore: Boolean(view.is_core || view.isCore),
              isCustom: Boolean(view.is_custom || view.isCustom),
            });
          }
        });
      }

      // If the current view extends a view that wasn't included in the fetched list,
      // fetch it individually (or use the in-memory extends object) so its items
      // appear immediately in the available components panel.
      try {
        const explicitExtends = viewData?.extends;
        const explicitExtendsId = typeof explicitExtends === 'string' && explicitExtends.trim() ? String(explicitExtends).toLowerCase() : '';
        const alreadyIncluded = explicitExtendsId && sources.some(s => String(s.id).toLowerCase() === explicitExtendsId);
        if (explicitExtendsId && !alreadyIncluded) {
          // Attempt to fetch the single view by id
          try {
            const url = `/api/views/${explicitExtendsId}?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`;
            const singleResp = await fetch(url);
            if (singleResp && singleResp.ok) {
              const data = await singleResp.json();
              const view = data?.view ?? data;
              if (view && view.id) {
                const existingDimensionIds = new Set<string>((viewData?.dimensions || []).map((d: any) => (d?.id || d?.qualifiedName || d?.name || '').toLowerCase()));
                const existingMeasureIds = new Set<string>((viewData?.measures || []).map((m: any) => (m?.id || m?.qualifiedName || m?.name || '').toLowerCase()));
                const items: AvailableItem[] = [];
                if (view.dimensions) {
                  view.dimensions.forEach((dim: any, index: number) => {
                    const qid = `${view.id}.${dim.name || `dimension_${index}`}`.toLowerCase();
                    if (existingDimensionIds.has(qid)) return;
                    items.push({
                      id: `${view.id}.${dim.name || `dimension_${index}`}`,
                      name: dim.title || dim.name || `Dimension ${index + 1}`,
                      type: 'dimension',
                      source: view.id,
                      description: dim.description,
                      datatype: dim.datatype || dim.type,
                    });
                  });
                }
                if (view.measures) {
                  view.measures.forEach((measure: any, index: number) => {
                    const qid = `${view.id}.${measure.name || `measure_${index}`}`.toLowerCase();
                    if (existingMeasureIds.has(qid)) return;
                    items.push({
                      id: `${view.id}.${measure.name || `measure_${index}`}`,
                      name: measure.title || measure.name || `Measure ${index + 1}`,
                      type: 'measure',
                      source: view.id,
                      description: measure.description,
                      datatype: measure.datatype || measure.type,
                    });
                  });
                }

                if (items.length > 0) {
                  sources.unshift({
                    id: view.id,
                    name: view.title || view.name,
                    type: 'extended_view',
                    items,
                    expanded: true,
                    filteredOutCount: Math.max(0, ((view.dimensions?.length || 0) + (view.measures?.length || 0)) - items.length),
                    isCore: Boolean(view.is_core || view.isCore),
                    isCustom: Boolean(view.is_custom || view.isCustom),
                  });
                }
              }
            }
          } catch (e) {
            // ignore single-view fetch failure
          }
        } else if (explicitExtends && typeof explicitExtends === 'object' && explicitExtends.id) {
          // The extends may be an in-memory object (newly added view) or from typeahead — fetch full details if needed
          const view = explicitExtends as any;
          let fullView = view;
          if (!view.dimensions && !view.measures) {
            // Object doesn't have dimensions/measures, fetch full view details
            try {
              const url = `/api/views/${view.id}?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`;
              const singleResp = await fetch(url);
              if (singleResp && singleResp.ok) {
                const data = await singleResp.json();
                fullView = data?.view ?? data;
              }
            } catch (e) {
              // ignore fetch failure, fall back to object
            }
          }
          const existingDimensionIds = new Set<string>((viewData?.dimensions || []).map((d: any) => (d?.id || d?.qualifiedName || d?.name || '').toLowerCase()));
          const existingMeasureIds = new Set<string>((viewData?.measures || []).map((m: any) => (m?.id || m?.qualifiedName || m?.name || '').toLowerCase()));
          const items: AvailableItem[] = [];
          if (fullView.dimensions) {
            fullView.dimensions.forEach((dim: any, index: number) => {
              const qid = `${fullView.id}.${dim.name || `dimension_${index}`}`.toLowerCase();
              if (existingDimensionIds.has(qid)) return;
              items.push({
                id: `${fullView.id}.${dim.name || `dimension_${index}`}`,
                name: dim.title || dim.name || `Dimension ${index + 1}`,
                type: 'dimension',
                source: fullView.id,
                description: dim.description,
                datatype: dim.datatype || dim.type,
              });
            });
          }
          if (fullView.measures) {
            fullView.measures.forEach((measure: any, index: number) => {
              const qid = `${fullView.id}.${measure.name || `measure_${index}`}`.toLowerCase();
              if (existingMeasureIds.has(qid)) return;
              items.push({
                id: `${fullView.id}.${measure.name || `measure_${index}`}`,
                name: measure.title || measure.name || `Measure ${index + 1}`,
                type: 'measure',
                source: fullView.id,
                description: measure.description,
                datatype: measure.datatype || measure.type,
              });
            });
          }
          if (items.length > 0) {
            sources.unshift({
              id: fullView.id,
              name: fullView.title || fullView.name,
              type: 'extended_view',
              items,
              expanded: true,
              filteredOutCount: Math.max(0, ((fullView.dimensions?.length || 0) + (fullView.measures?.length || 0)) - items.length),
              isCore: Boolean(fullView.is_core || fullView.isCore),
              isCustom: Boolean(fullView.is_custom || fullView.isCustom),
            });
          }
        }
      } catch (e) {
        // ignore errors in the explicit-extends handling
      }

      setAvailableSources(sources);
    } catch (error) {
      setAvailableSources([]);
    } finally {
      setSourcesLoading(false);
    }
  }, [isValidTenantScope, tenantId, datasourceId, viewData, selectedRefs]);

  return { availableSources, sourcesLoading, fetchAvailableSources };
};