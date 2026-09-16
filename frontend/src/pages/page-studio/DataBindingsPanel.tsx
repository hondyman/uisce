import React, { useEffect, useState } from 'react';
import {
    Box,
    Typography,
    List,
    ListItem,
    ListItemText,
    IconButton,
    Chip,
    TextField,
    Button,
    Tooltip,
    Divider,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon, Business as BusinessIcon } from '@mui/icons-material';
import { useDraggable } from '@dnd-kit/core';
import { apiClient } from '../../utils/apiClient';
import { CorePageDefinition, DataSourceDefinition, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import { fetchBusinessObjectBindings, fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';
import { buildBODataSource } from './generatePageDraft';
import { fkColumnFromJoinCondition } from './boRelationships';

/**
 * Drag payload shape for dragging a field out of this panel onto a widget in
 * the canvas - see LayoutCanvas.tsx's onDragEnd, which reads this back via
 * @dnd-kit's `active.data.current` and patches the target widget's
 * dimension/measure field selection accordingly. `kind: 'field'` (set on the
 * useDraggable below) is how that handler tells "new widget from
 * ComponentPalette" and "bind this field" apart, replacing the old
 * dataTransfer custom-MIME-type trick.
 */
export interface FieldDragPayload {
    boId: string;
    termNodeId: string;
    termKey: string;
    displayName: string;
    role: string;
}

interface BusinessObjectOption {
    id: string;
    /** bo_key - the technical name every bo-scoped endpoint (bo-fields, terms, etc.) actually expects, not the display name. */
    key: string;
    name: string;
    display_name: string;
}

// Shape returned by GET /api/business-objects/{id}/relationships
interface RelatedBusinessObject {
    id: string;
    relatedObjectName: string;
    targetObjectId: string;
    relationshipType: string;
    cardinality: string;
    joinCondition: string;
}

/** One draggable field chip - see FieldDragPayload doc comment above. */
export const FieldChip: React.FC<{ payload: FieldDragPayload; isMeasure: boolean }> = ({ payload, isMeasure }) => {
    const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
        id: `field-${payload.boId}-${payload.termNodeId}`,
        data: { kind: 'field', payload },
    });
    return (
        <Chip
            ref={setNodeRef}
            {...listeners}
            {...attributes}
            size="small"
            label={payload.displayName}
            variant="outlined"
            color={isMeasure ? 'secondary' : 'default'}
            sx={{ cursor: 'grab', opacity: isDragging ? 0.4 : 1 }}
        />
    );
};

interface DataBindingsPanelProps {
    draft: CorePageDefinition;
    setDraft: (updater: (prev: CorePageDefinition) => CorePageDefinition) => void;
    tenantId: string;
}

// Lets a page bind to a Business Object as a data source, and pick from its
// related Business Objects (resolved via the cached BO relationship graph)
// so widgets on the page can pull in related data without the user having
// to know the underlying join.
const DataBindingsPanel: React.FC<DataBindingsPanelProps> = ({ draft, setDraft, tenantId }) => {
    const [businessObjects, setBusinessObjects] = useState<BusinessObjectOption[]>([]);
    const [selectedBOId, setSelectedBOId] = useState('');
    const [relatedByBO, setRelatedByBO] = useState<Record<string, RelatedBusinessObject[]>>({});
    const [fieldsByBO, setFieldsByBO] = useState<Record<string, SemanticTermView[]>>({});
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        let cancelled = false;
        apiClient<unknown>('/business-objects', {
            headers: tenantId ? { 'X-Tenant-ID': tenantId } : undefined,
        })
            .then((data) => {
                if (cancelled) return;
                const rawList = Array.isArray(data)
                    ? data
                    : data && typeof data === 'object'
                        ? Object.values(data as Record<string, unknown>)
                        : [];
                const normalized = rawList
                    .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
                    .map((item) => ({
                        id: String(item.id ?? item.name),
                        key: String(item.key ?? item.technicalName ?? item.technical_name ?? item.name ?? ''),
                        name: String(item.name ?? ''),
                        display_name: String(item.displayName ?? item.display_name ?? item.name ?? ''),
                    }));
                setBusinessObjects(normalized);
            })
            .catch(() => setBusinessObjects([]));
        return () => {
            cancelled = true;
        };
    }, [tenantId]);

    const boDataSources = (draft.dataSources || []).filter((ds) => ds.type === 'business_object');

    const fetchRelated = async (boId: string) => {
        if (relatedByBO[boId]) return relatedByBO[boId];
        try {
            const data = await apiClient<{ relatedObjects?: RelatedBusinessObject[] }>(
                `/business-objects/${boId}/relationships`,
                { headers: tenantId ? { 'X-Tenant-ID': tenantId } : undefined }
            );
            const related = data?.relatedObjects || [];
            setRelatedByBO((prev) => ({ ...prev, [boId]: related }));
            return related;
        } catch {
            setRelatedByBO((prev) => ({ ...prev, [boId]: [] }));
            return [];
        }
    };

    useEffect(() => {
        boDataSources.forEach((ds) => {
            const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
            if (cfg?.boId) fetchRelated(cfg.boId);
        });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [(draft.dataSources || []).length]);

    // Field list per bound BO, for the drag-and-drop palette below each
    // source - same term data every widget's own field pickers already use
    // (fetchBOTerms), just surfaced here as drag sources instead of a
    // per-widget dropdown.
    useEffect(() => {
        boDataSources.forEach((ds) => {
            const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
            if (!cfg?.boId || fieldsByBO[cfg.boId]) return;
            fetchBOTerms(cfg.boId, cfg.bindingId || '')
                .then((terms) => setFieldsByBO((prev) => ({ ...prev, [cfg.boId]: terms })))
                .catch(() => setFieldsByBO((prev) => ({ ...prev, [cfg.boId]: [] })));
        });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [(draft.dataSources || []).length]);

    const handleAddBO = async () => {
        if (!selectedBOId) return;
        const bo = businessObjects.find((b) => b.id === selectedBOId);
        if (!bo) return;

        setLoading(true);
        try {
            await fetchRelated(bo.id);
            const bindings = await fetchBusinessObjectBindings(bo.id).catch(() => []);
            const defaultBinding = bindings.find((b) => b.isDefault) || bindings[0];
            const config: BusinessObjectDataSourceConfig = {
                boId: bo.id,
                boKey: bo.key,
                bindingId: defaultBinding?.bindingId || '',
                displayName: bo.display_name,
                relatedBoIds: [],
            };
            const newSource: DataSourceDefinition = {
                id: `bo_${bo.id}`,
                name: bo.name,
                type: 'business_object',
                config: config as unknown as Record<string, unknown>,
            };
            setDraft((prev) => ({
                ...prev,
                dataSources: [...(prev.dataSources || []).filter((d) => d.id !== newSource.id), newSource],
            }));
            setSelectedBOId('');
        } finally {
            setLoading(false);
        }
    };

    const handleRemoveSource = (id: string) => {
        setDraft((prev) => ({ ...prev, dataSources: (prev.dataSources || []).filter((d) => d.id !== id) }));
    };

    const toggleRelated = async (source: DataSourceDefinition, rel: RelatedBusinessObject) => {
        const cfg = source.config as unknown as BusinessObjectDataSourceConfig;
        const current = cfg.relatedBoIds || [];
        const including = current.includes(rel.targetObjectId);
        const next = including
            ? current.filter((id) => id !== rel.targetObjectId)
            : [...current, rel.targetObjectId];
        let extra: DataSourceDefinition | null = null;
        if (!including) {
            const fkField = fkColumnFromJoinCondition(rel.joinCondition);
            extra = await buildBODataSource(rel.targetObjectId, rel.relatedObjectName, rel.relatedObjectName, rel.relatedObjectName, []);
            extra = {
                ...extra,
                config: {
                    ...(extra.config as object),
                    ...(fkField ? { masterFilter: { fkField } } : {}),
                } as unknown as Record<string, unknown>,
            };
        }
        setDraft((prev) => {
            const sources = (prev.dataSources || []).map((d) =>
                d.id === source.id ? { ...d, config: { ...cfg, relatedBoIds: next } as unknown as Record<string, unknown> } : d
            );
            if (including) {
                return { ...prev, dataSources: sources.filter((d) => d.id !== `bo_${rel.targetObjectId}`) };
            }
            const withoutDup = sources.filter((d) => d.id !== extra!.id);
            return { ...prev, dataSources: [...withoutDup, extra!] };
        });
    };

    return (
        <Box sx={{ p: 2 }}>
            <Typography variant="overline" color="text.secondary">
                Primary Business Object
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                First source is the page&apos;s main Business Object. Related objects come from the relationship graph — click to add them (and their terms) to this page.
            </Typography>

            <Box sx={{ display: 'flex', gap: 1, mt: 1, mb: 2 }}>
                <TextField
                    select
                    fullWidth
                    size="small"
                    value={selectedBOId}
                    onChange={(e) => setSelectedBOId(e.target.value)}
                    SelectProps={{ native: true }}
                >
                    <option value="">Select Business Object...</option>
                    {businessObjects
                        .filter((bo) => !boDataSources.some((ds) => ds.id === `bo_${bo.id}`))
                        .map((bo) => (
                            <option key={bo.id} value={bo.id}>
                                {bo.display_name}
                            </option>
                        ))}
                </TextField>
                <Button variant="contained" size="small" startIcon={<AddIcon />} onClick={handleAddBO} disabled={!selectedBOId || loading}>
                    Add
                </Button>
            </Box>

            {boDataSources.length === 0 && (
                <Typography variant="caption" color="text.secondary">
                    Bind a Business Object first. Widgets on this page can only use that object&apos;s terms (and related objects you include).
                </Typography>
            )}

            <List dense disablePadding>
                {boDataSources.map((ds, index) => {
                    const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
                    const related = relatedByBO[cfg.boId] || [];
                    const fields = fieldsByBO[cfg.boId] || [];
                    const isPrimary = index === 0;
                    return (
                        <Box key={ds.id} sx={{ mb: 2 }}>
                            <ListItem
                                disableGutters
                                secondaryAction={
                                    <IconButton size="small" onClick={() => handleRemoveSource(ds.id)}>
                                        <DeleteIcon fontSize="small" />
                                    </IconButton>
                                }
                            >
                                <BusinessIcon fontSize="small" sx={{ mr: 1, color: 'primary.main' }} />
                                <ListItemText
                                    primary={<>{cfg.displayName}{isPrimary ? ' (primary)' : ''}</>}
                                    secondary={cfg.boKey}
                                />
                            </ListItem>

                            {related.length > 0 && (
                                <Box sx={{ pl: 4 }}>
                                    <Typography variant="caption" color="text.secondary">
                                        Related objects (click to include on this page)
                                    </Typography>
                                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
                                        {related.map((rel) => {
                                            const active = (cfg.relatedBoIds || []).includes(rel.targetObjectId);
                                            return (
                                                <Tooltip
                                                    key={rel.id}
                                                    title={`${rel.relationshipType} (${rel.cardinality})${rel.joinCondition ? ` — ${rel.joinCondition}` : ''}`}
                                                >
                                                    <Chip
                                                        size="small"
                                                        label={rel.relatedObjectName}
                                                        color={active ? 'primary' : 'default'}
                                                        variant={active ? 'filled' : 'outlined'}
                                                        onClick={() => { void toggleRelated(ds, rel); }}
                                                    />
                                                </Tooltip>
                                            );
                                        })}
                                    </Box>
                                </Box>
                            )}

                            <Box sx={{ pl: 4, mt: 1.5 }}>
                                <Typography variant="caption" color="text.secondary">
                                    Fields — drag onto a Table, Form, Slicer, Chart, or KPI
                                </Typography>
                                {fields.length === 0 ? (
                                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                                        Loading fields…
                                    </Typography>
                                ) : (
                                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
                                        {fields.map((field) => {
                                            const isMeasure = field.role === 'MEASURE' || field.role === 'CALCULATED';
                                            const payload: FieldDragPayload = {
                                                boId: cfg.boId,
                                                termNodeId: field.termNodeId,
                                                termKey: field.termKey,
                                                displayName: field.displayName,
                                                role: field.role,
                                            };
                                            return <FieldChip key={field.termNodeId} payload={payload} isMeasure={isMeasure} />;
                                        })}
                                    </Box>
                                )}
                            </Box>

                            <Divider sx={{ mt: 2 }} />
                        </Box>
                    );
                })}
            </List>
        </Box>
    );
};

export default DataBindingsPanel;
