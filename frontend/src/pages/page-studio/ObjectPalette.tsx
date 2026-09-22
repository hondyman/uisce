import React, { useEffect, useState } from 'react';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Typography,
  Chip,
  Tooltip,
} from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { CorePageDefinition, BusinessObjectDataSourceConfig, DataSourceDefinition } from '../../types/pageStudio';
import type { RelatedObjectDragPayload } from '../../studio-core/binding/boRelationships';
import { widgetTypeForCardinality } from '../../studio-core/binding/boRelationships';
import { fetchBOTerms, fetchBORelationships, type BORelationship, type SemanticTermView } from '../../studio-core/binding/businessObjectApi';
import { FieldChip } from './DataBindingsPanel';

type FieldsStatus = 'loading' | 'ready' | 'error';

const boCfg = (ds: DataSourceDefinition): BusinessObjectDataSourceConfig =>
  ds.config as unknown as BusinessObjectDataSourceConfig;

/**
 * Graph-first half of the Design palette: the page's bound Business Objects
 * and their related objects from GET /api/business-objects/{id}/relationships.
 * Dropping a related object onto the canvas creates a Table or Form already
 * scoped by the FK — LayoutCanvas.tsx handles the drop.
 */
const RelatedObjectTile: React.FC<{ payload: RelatedObjectDragPayload; caption?: string }> = ({ payload, caption }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `related-${payload.parentSourceId}-${payload.targetObjectId}`,
    data: { kind: 'related-object', payload },
  });
  const widgetType = widgetTypeForCardinality(payload.cardinality);
  const join = payload.joinCondition ? ` · ${payload.joinCondition}` : '';
  return (
    <Tooltip title={caption || `${payload.cardinality || 'related'} → drops as ${widgetType}${join}`}>
      <Box
        ref={setNodeRef}
        {...listeners}
        {...attributes}
        sx={{ display: 'inline-block', cursor: 'grab', opacity: isDragging ? 0.4 : 1 }}
      >
        <Chip
          size="small"
          icon={<AccountTreeIcon sx={{ fontSize: 14 }} />}
          label={`${payload.relatedObjectName} · ${widgetType}`}
          variant="outlined"
          sx={{ pointerEvents: 'none' }}
        />
      </Box>
    </Tooltip>
  );
};

const dragPayload = (parent: DataSourceDefinition, rel: BORelationship): RelatedObjectDragPayload => {
  const cfg = boCfg(parent);
  return {
    parentSourceId: parent.id,
    parentBoId: cfg.boId,
    targetObjectId: rel.targetObjectId,
    relatedObjectName: rel.relatedObjectName,
    cardinality: rel.cardinality,
    joinCondition: rel.joinCondition,
    joinColumns: rel.joinColumns,
    kind: rel.kind || 'bo',
    linkTable: rel.linkTable,
  };
};

interface ObjectPaletteProps {
  draft: CorePageDefinition;
  tenantId: string;
}

const ObjectPalette: React.FC<ObjectPaletteProps> = ({ draft, tenantId }) => {
  const boSources = (draft.dataSources || []).filter((d) => d.type === 'business_object');
  const [relatedByBO, setRelatedByBO] = useState<Record<string, BORelationship[]>>({});
  const [fieldsByBO, setFieldsByBO] = useState<Record<string, SemanticTermView[]>>({});
  const [fieldsStatusByBO, setFieldsStatusByBO] = useState<Record<string, FieldsStatus>>({});

  const primary = boSources[0];
  const primaryCfg = primary ? boCfg(primary) : undefined;

  useEffect(() => {
    let cancelled = false;
    if (!primaryCfg?.boId || relatedByBO[primaryCfg.boId]) return;
    fetchBORelationships(primaryCfg.boId)
      .then((rels) => {
        if (!cancelled) setRelatedByBO((prev) => ({ ...prev, [primaryCfg.boId]: rels }));
      })
      .catch(() => {
        if (!cancelled) setRelatedByBO((prev) => ({ ...prev, [primaryCfg.boId]: [] }));
      });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [primaryCfg?.boId, tenantId]);

  useEffect(() => {
    let cancelled = false;
    boSources.forEach((ds) => {
      const cfg = boCfg(ds);
      if (!cfg?.boId || fieldsStatusByBO[cfg.boId] === 'ready' || fieldsStatusByBO[cfg.boId] === 'error') return;
      setFieldsStatusByBO((prev) => ({ ...prev, [cfg.boId]: 'loading' }));
      fetchBOTerms(cfg.boId, cfg.bindingId || '')
        .then((terms) => {
          if (cancelled) return;
          setFieldsByBO((prev) => ({ ...prev, [cfg.boId]: terms }));
          setFieldsStatusByBO((prev) => ({ ...prev, [cfg.boId]: 'ready' }));
        })
        .catch(() => {
          if (cancelled) return;
          setFieldsByBO((prev) => ({ ...prev, [cfg.boId]: [] }));
          setFieldsStatusByBO((prev) => ({ ...prev, [cfg.boId]: 'error' }));
        });
    });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boSources.map((d) => boCfg(d)?.boId).join(',')]);

  if (boSources.length === 0) {
    return (
      <Box sx={{ px: 2, pt: 2, pb: 1 }}>
        <Typography variant="overline" color="text.secondary" fontWeight="bold">Business Objects</Typography>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
          Bind a Business Object in Data Binding (or create the page from the wizard). Then drag fields onto widgets on the page.
        </Typography>
      </Box>
    );
  }

  const relatedOfPrimary = (primaryCfg ? relatedByBO[primaryCfg.boId] : []) || [];
  const relatedBOs = relatedOfPrimary.filter((r) => r.kind !== 'relatedTable');
  const includedIds = new Set(boSources.map((s) => boCfg(s).boId));
  const availableRelated = relatedBOs.filter((r) => !includedIds.has(r.targetObjectId));
  const relatedTables = relatedOfPrimary.filter((r) => r.kind === 'relatedTable');

  const fieldsBlock = (boId: string) => {
    const status = fieldsStatusByBO[boId];
    const fields = fieldsByBO[boId] || [];
    if (status === 'loading' || !status) {
      return (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
          Loading fields…
        </Typography>
      );
    }
    if (status === 'error') {
      return (
        <Typography variant="caption" color="error" sx={{ display: 'block', mt: 0.5 }}>
          Could not load fields — this must be a published Business Object (id {boId}).
        </Typography>
      );
    }
    if (fields.length === 0) {
      return (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
          No bound fields.
        </Typography>
      );
    }
    return (
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
        {fields.map((field) => (
          <FieldChip
            key={field.termNodeId}
            payload={{
              boId,
              termNodeId: field.termNodeId,
              termKey: field.termKey,
              displayName: field.displayName,
              role: field.role,
            }}
            isMeasure={field.role === 'MEASURE' || field.role === 'CALCULATED'}
          />
        ))}
      </Box>
    );
  };

  return (
    <Box sx={{ px: 2, pt: 2, pb: 1 }}>
      <Typography variant="overline" color="text.secondary" fontWeight="bold">Business Objects</Typography>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
        Drag a field onto a Table, Form, chart, KPI, or slicer already on the page. Drop a related object chip onto a slot to place a widget.
      </Typography>
      {boSources.map((ds, index) => {
        const cfg = boCfg(ds);
        const isPrimary = index === 0;
        const relFromPrimary = relatedBOs.find((r) => r.targetObjectId === cfg.boId);
        const widgetType = relFromPrimary ? widgetTypeForCardinality(relFromPrimary.cardinality) : 'Table';
        return (
          <Accordion
            key={ds.id}
            defaultExpanded={isPrimary}
            disableGutters
            elevation={0}
            sx={{ '&:before': { display: 'none' }, bgcolor: 'transparent', borderBottom: '1px solid', borderColor: 'divider' }}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon />}
              sx={{ px: 0, minHeight: 40, '& .MuiAccordionSummary-content': { my: 0.75, overflow: 'hidden' } }}
            >
              <Box sx={{ minWidth: 0 }}>
                <Typography variant="caption" fontWeight={700} sx={{ display: 'block' }}>
                  {cfg.displayName || cfg.boKey}{isPrimary ? ' (primary)' : ''}
                </Typography>
                <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                  {isPrimary
                    ? cfg.boKey
                    : relFromPrimary
                      ? `${relFromPrimary.cardinality}${relFromPrimary.joinCondition ? ` · ${relFromPrimary.joinCondition}` : ''}`
                      : cfg.boKey}
                </Typography>
              </Box>
            </AccordionSummary>
            <AccordionDetails sx={{ px: 0, pt: 0, pb: 1.5 }}>
              {fieldsBlock(cfg.boId)}
              {!isPrimary && primary && relFromPrimary && (
                <Box sx={{ mt: 1 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                    Drop on the page to add a {widgetType.toLowerCase()}
                    {relFromPrimary.joinCondition ? ` (${relFromPrimary.joinCondition})` : ''}.
                  </Typography>
                  <RelatedObjectTile
                    payload={dragPayload(primary, relFromPrimary)}
                    caption={`Place ${relFromPrimary.relatedObjectName} as ${widgetType}${relFromPrimary.joinCondition ? ` · ${relFromPrimary.joinCondition}` : ''}`}
                  />
                </Box>
              )}
              {isPrimary && availableRelated.length > 0 && (
                <Box sx={{ mt: 1.5 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                    Available related — drop on the page to include and place
                  </Typography>
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                    {availableRelated.map((rel) => (
                      <RelatedObjectTile
                        key={`${ds.id}-${rel.targetObjectId}`}
                        payload={dragPayload(ds, rel)}
                      />
                    ))}
                  </Box>
                </Box>
              )}
              {isPrimary && relatedTables.length > 0 && (
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                  Related tables: {relatedTables.map((r) => r.relatedObjectName).join(', ')}
                </Typography>
              )}
            </AccordionDetails>
          </Accordion>
        );
      })}
    </Box>
  );
};

export default ObjectPalette;
