import React, { useEffect, useState } from 'react';
import { Box, Typography, Chip, Tooltip } from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import { apiClient } from '../../utils/apiClient';
import type { CorePageDefinition, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import type { BORelationship, RelatedObjectDragPayload } from './boRelationships';
import { widgetTypeForCardinality } from './boRelationships';
import { fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';
import { FieldChip } from './DataBindingsPanel';

/**
 * Graph-first half of the Design palette: the page's bound Business Objects
 * and their related objects from GET /api/business-objects/{id}/relationships.
 * Dropping a related object onto the canvas creates a Table or Form already
 * scoped by the FK — LayoutCanvas.tsx handles the drop.
 */
const RelatedObjectTile: React.FC<{ payload: RelatedObjectDragPayload }> = ({ payload }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `related-${payload.parentSourceId}-${payload.targetObjectId}`,
    data: { kind: 'related-object', payload },
  });
  const widgetType = widgetTypeForCardinality(payload.cardinality);
  return (
    <Tooltip title={`${payload.cardinality || 'related'} → drops as ${widgetType}${payload.joinCondition ? ` (${payload.joinCondition})` : ''}`}>
      <Chip
        ref={setNodeRef}
        {...listeners}
        {...attributes}
        size="small"
        icon={<AccountTreeIcon sx={{ fontSize: 14 }} />}
        label={`${payload.relatedObjectName} · ${widgetType}`}
        variant="outlined"
        sx={{ cursor: 'grab', opacity: isDragging ? 0.4 : 1 }}
      />
    </Tooltip>
  );
};

interface ObjectPaletteProps {
  draft: CorePageDefinition;
  tenantId: string;
}

const ObjectPalette: React.FC<ObjectPaletteProps> = ({ draft, tenantId }) => {
  const boSources = (draft.dataSources || []).filter((d) => d.type === 'business_object');
  const [relatedByBO, setRelatedByBO] = useState<Record<string, BORelationship[]>>({});
  const [fieldsByBO, setFieldsByBO] = useState<Record<string, SemanticTermView[]>>({});

  useEffect(() => {
    let cancelled = false;
    boSources.forEach((ds) => {
      const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
      if (!cfg?.boId || relatedByBO[cfg.boId]) return;
      apiClient<{ relatedObjects?: BORelationship[] }>(`/business-objects/${cfg.boId}/relationships`, {
        headers: tenantId ? { 'X-Tenant-ID': tenantId } : undefined,
      })
        .then((data) => {
          if (!cancelled) setRelatedByBO((prev) => ({ ...prev, [cfg.boId]: data?.relatedObjects || [] }));
        })
        .catch(() => {
          if (!cancelled) setRelatedByBO((prev) => ({ ...prev, [cfg.boId]: [] }));
        });
    });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boSources.map((d) => (d.config as unknown as BusinessObjectDataSourceConfig)?.boId).join(','), tenantId]);

  useEffect(() => {
    boSources.forEach((ds) => {
      const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
      if (!cfg?.boId || fieldsByBO[cfg.boId]) return;
      fetchBOTerms(cfg.boId, cfg.bindingId || '')
        .then((terms) => setFieldsByBO((prev) => ({ ...prev, [cfg.boId]: terms })))
        .catch(() => setFieldsByBO((prev) => ({ ...prev, [cfg.boId]: [] })));
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boSources.map((d) => (d.config as unknown as BusinessObjectDataSourceConfig)?.boId).join(',')]);

  if (boSources.length === 0) {
    return (
      <Box sx={{ px: 2, pt: 2, pb: 1 }}>
        <Typography variant="overline" color="text.secondary" fontWeight="bold">Business Objects</Typography>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
          Bind a Business Object in Data Binding (or create the page from the wizard). Related lists come from the relationship graph, not from a hardcoded object list.
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ px: 2, pt: 2, pb: 1 }}>
      <Typography variant="overline" color="text.secondary" fontWeight="bold">Business Objects</Typography>
      {boSources.map((ds, index) => {
        const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
        const related = relatedByBO[cfg.boId] || [];
        const fields = fieldsByBO[cfg.boId] || [];
        return (
          <Box key={ds.id} sx={{ mt: 1 }}>
            <Typography variant="caption" fontWeight={700} sx={{ display: 'block' }}>
              {cfg.displayName || cfg.boKey}{index === 0 ? ' (primary)' : ''}
            </Typography>
            {fields.length > 0 && (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
                {fields.map((field) => (
                  <FieldChip
                    key={field.termNodeId}
                    payload={{
                      boId: cfg.boId,
                      termNodeId: field.termNodeId,
                      termKey: field.termKey,
                      displayName: field.displayName,
                      role: field.role,
                    }}
                    isMeasure={field.role === 'MEASURE' || field.role === 'CALCULATED'}
                  />
                ))}
              </Box>
            )}
            {related.length > 0 && (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
                {related.map((rel) => (
                  <RelatedObjectTile
                    key={`${ds.id}-${rel.targetObjectId}`}
                    payload={{
                      parentSourceId: ds.id,
                      parentBoId: cfg.boId,
                      targetObjectId: rel.targetObjectId,
                      relatedObjectName: rel.relatedObjectName,
                      cardinality: rel.cardinality,
                      joinCondition: rel.joinCondition,
                    }}
                  />
                ))}
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
};

export default ObjectPalette;
