import React, { useEffect, useState } from 'react';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    Alert,
    Box,
    Typography,
    IconButton,
    Chip,
    TextField,
    Button,
    Tooltip,
} from '@mui/material';
import {
    Add as AddIcon,
    Delete as DeleteIcon,
    Business as BusinessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { useDraggable } from '@dnd-kit/core';
import { CorePageDefinition, DataSourceDefinition, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import {
    listBusinessObjects, fetchBORelationships, fetchBusinessObjectBindings, fetchBOTerms,
    type BusinessObjectOption, type BORelationship, type SemanticTermView,
} from '../../studio-core/binding/businessObjectApi';
import { buildBODataSource } from './generatePageDraft';
import { fkColumnFromJoinCondition } from '../../studio-core/binding/boRelationships';

/**
 * Drag payload shape for dragging a field out of the Design palette onto a
 * widget — LayoutCanvas.tsx onDragEnd reads this via active.data.current.
 * `kind: 'field'` distinguishes bind-field from new-widget-from-palette.
 */
export interface FieldDragPayload {
    boId: string;
    termNodeId: string;
    termKey: string;
    displayName: string;
    role: string;
}

type FieldsStatus = 'loading' | 'ready' | 'error';

/**
 * Chip is visual only. MUI Chip as the drag root swallows pointer events
 * (same class of bug as related-object chips). Box owns useDraggable.
 */
export const FieldChip: React.FC<{ payload: FieldDragPayload; isMeasure: boolean }> = ({ payload, isMeasure }) => {
    const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
        id: `field-${payload.boId}-${payload.termNodeId}`,
        data: { kind: 'field', payload },
    });
    return (
        <Box
            ref={setNodeRef}
            {...listeners}
            {...attributes}
            sx={{ display: 'inline-block', cursor: 'grab', opacity: isDragging ? 0.4 : 1 }}
        >
            <Chip
                size="small"
                label={payload.displayName}
                variant="outlined"
                color={isMeasure ? 'secondary' : 'default'}
                sx={{ pointerEvents: 'none' }}
            />
        </Box>
    );
};

interface DataBindingsPanelProps {
    draft: CorePageDefinition;
    setDraft: (updater: (prev: CorePageDefinition) => CorePageDefinition) => void;
    tenantId: string;
}

const boCfg = (ds: DataSourceDefinition): BusinessObjectDataSourceConfig =>
    ds.config as unknown as BusinessObjectDataSourceConfig;

const DataBindingsPanel: React.FC<DataBindingsPanelProps> = ({ draft, setDraft, tenantId }) => {
    const [businessObjects, setBusinessObjects] = useState<BusinessObjectOption[]>([]);
    const [selectedBOId, setSelectedBOId] = useState('');
    const [relatedByBO, setRelatedByBO] = useState<Record<string, BORelationship[]>>({});
    const [fieldsByBO, setFieldsByBO] = useState<Record<string, SemanticTermView[]>>({});
    const [fieldsStatusByBO, setFieldsStatusByBO] = useState<Record<string, FieldsStatus>>({});
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        let cancelled = false;
        listBusinessObjects()
            .then((list) => { if (!cancelled) setBusinessObjects(list); })
            .catch(() => setBusinessObjects([]));
        return () => {
            cancelled = true;
        };
    }, [tenantId]);

    const boDataSources = (draft.dataSources || []).filter((ds) => ds.type === 'business_object');
    const primary = boDataSources[0];
    const primaryCfg = primary ? boCfg(primary) : undefined;
    const primaryName = primaryCfg?.displayName || primaryCfg?.boKey || 'the primary Business Object';

    const fetchRelated = async (boId: string) => {
        if (relatedByBO[boId]) return relatedByBO[boId];
        try {
            const related = await fetchBORelationships(boId);
            setRelatedByBO((prev) => ({ ...prev, [boId]: related }));
            return related;
        } catch {
            setRelatedByBO((prev) => ({ ...prev, [boId]: [] }));
            return [];
        }
    };

    useEffect(() => {
        if (!primaryCfg?.boId) return;
        void fetchRelated(primaryCfg.boId);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [primaryCfg?.boId]);

    useEffect(() => {
        let cancelled = false;
        boDataSources.forEach((ds) => {
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
        return () => {
            cancelled = true;
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [boDataSources.map((d) => boCfg(d)?.boId).join(',')]);

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
                displayName: bo.displayName,
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

    const toggleRelated = async (source: DataSourceDefinition, rel: BORelationship) => {
        const cfg = boCfg(source);
        const current = cfg.relatedBoIds || [];
        const including = current.includes(rel.targetObjectId) ||
            (draft.dataSources || []).some((d) => d.id === `bo_${rel.targetObjectId}`);
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

    const relatedOfPrimary = (primaryCfg ? relatedByBO[primaryCfg.boId] : []) || [];
    const relatedBOs = relatedOfPrimary.filter((rel) => rel.kind !== 'relatedTable');
    const includedBoIds = new Set(boDataSources.map((ds) => boCfg(ds).boId));
    const isRelatedIncluded = (rel: BORelationship) =>
        (primaryCfg?.relatedBoIds || []).includes(rel.targetObjectId) || includedBoIds.has(rel.targetObjectId);
    const onPage = relatedBOs.filter(isRelatedIncluded);
    const notYet = relatedBOs.filter((rel) => !isRelatedIncluded(rel));

    const fieldCountLabel = (boId: string) => {
        const status = fieldsStatusByBO[boId];
        if (status === 'loading') return 'loading fields…';
        if (status === 'error') return 'fields unavailable';
        const n = (fieldsByBO[boId] || []).length;
        return `${n} field${n === 1 ? '' : 's'}`;
    };

    const joinCaptionFor = (boId: string) => {
        const rel = relatedBOs.find((r) => r.targetObjectId === boId);
        if (!rel) return '';
        const join = rel.joinCondition ? ` · ${rel.joinCondition}` : '';
        return `${rel.cardinality || 'related'}${join}`;
    };

    return (
        <Box sx={{ p: 2 }}>
            <Typography variant="overline" color="text.secondary">
                Page data
            </Typography>
            <Alert severity="info" sx={{ mt: 1, mb: 2, py: 0.5, '& .MuiAlert-message': { fontSize: 12 } }}>
                <strong>Include</strong> a related object here to make it available (join is recorded; nothing is placed on the canvas).
                Then switch to <strong>Design</strong> to <strong>place</strong> a Table/Form and <strong>bind</strong> fields onto that widget.
                {primary ? ` This page’s primary object is ${primaryName}.` : ' Add a primary Business Object first.'}
            </Alert>

            {boDataSources.length === 0 && (
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                    Bind a Business Object first. Widgets on this page can only use that object&apos;s terms (and related objects you include).
                </Typography>
            )}

            <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                <TextField
                    select
                    fullWidth
                    size="small"
                    value={selectedBOId}
                    onChange={(e) => setSelectedBOId(e.target.value)}
                    SelectProps={{ native: true }}
                    disabled={boDataSources.length > 0}
                >
                    <option value="">{boDataSources.length > 0 ? 'Primary already set' : 'Select Business Object...'}</option>
                    {businessObjects
                        .filter((bo) => !boDataSources.some((ds) => ds.id === `bo_${bo.id}`))
                        .map((bo) => (
                            <option key={bo.id} value={bo.id}>
                                {bo.displayName}
                            </option>
                        ))}
                </TextField>
                <Button variant="contained" size="small" startIcon={<AddIcon />} onClick={handleAddBO} disabled={!selectedBOId || loading || boDataSources.length > 0}>
                    Add
                </Button>
            </Box>

            {boDataSources.map((ds, index) => {
                const cfg = boCfg(ds);
                const isPrimary = index === 0;
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
                            sx={{ px: 0, minHeight: 48, '& .MuiAccordionSummary-content': { alignItems: 'center', my: 1, overflow: 'hidden' } }}
                        >
                            <BusinessIcon fontSize="small" sx={{ mr: 1, color: 'primary.main', flexShrink: 0 }} />
                            <Box sx={{ flex: 1, minWidth: 0, mr: 1 }}>
                                <Typography variant="body2" fontWeight={700} noWrap>
                                    {cfg.displayName || cfg.boKey}{isPrimary ? ' (primary)' : ''}
                                </Typography>
                                <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                                    {isPrimary ? cfg.boKey : (joinCaptionFor(cfg.boId) || cfg.boKey)}
                                    {' · '}
                                    {fieldCountLabel(cfg.boId)}
                                </Typography>
                            </Box>
                            <IconButton
                                size="small"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    handleRemoveSource(ds.id);
                                }}
                            >
                                <DeleteIcon fontSize="small" />
                            </IconButton>
                        </AccordionSummary>
                        <AccordionDetails sx={{ px: 0, pt: 0, pb: 1.5 }}>
                            {isPrimary ? (
                                <>
                                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                                        On this page — click to remove (does not delete widgets already placed)
                                    </Typography>
                                    {onPage.length === 0 ? (
                                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                                            Only {primaryName} so far.
                                        </Typography>
                                    ) : (
                                        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 1.5 }}>
                                            {onPage.map((rel) => (
                                                <Tooltip
                                                    key={rel.id}
                                                    title={`${rel.cardinality}${rel.joinCondition ? ` — ${rel.joinCondition}` : ''} · included (data source). Click to exclude.`}
                                                >
                                                    <Chip
                                                        size="small"
                                                        label={rel.relatedObjectName}
                                                        color="primary"
                                                        variant="filled"
                                                        onClick={() => { void toggleRelated(ds, rel); }}
                                                    />
                                                </Tooltip>
                                            ))}
                                        </Box>
                                    )}
                                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                                        Related — not on this page yet. Click to include (available for widgets; nothing is placed).
                                    </Typography>
                                    {notYet.length === 0 ? (
                                        <Typography variant="caption" color="text.secondary">
                                            {relatedBOs.length === 0 ? 'No related Business Objects on the relationship graph.' : 'All related objects are included.'}
                                        </Typography>
                                    ) : (
                                        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                                            {notYet.map((rel) => (
                                                <Tooltip
                                                    key={rel.id}
                                                    title={`${rel.cardinality}${rel.joinCondition ? ` — ${rel.joinCondition}` : ''} · click to include`}
                                                >
                                                    <Chip
                                                        size="small"
                                                        label={rel.relatedObjectName}
                                                        color="default"
                                                        variant="outlined"
                                                        onClick={() => { void toggleRelated(ds, rel); }}
                                                    />
                                                </Tooltip>
                                            ))}
                                        </Box>
                                    )}
                                </>
                            ) : (
                                <Typography variant="caption" color="text.secondary">
                                    Included as a data source. Switch to Design to place a widget and bind fields.
                                    {cfg.boKey ? ` Technical name: ${cfg.boKey}.` : ''}
                                </Typography>
                            )}
                        </AccordionDetails>
                    </Accordion>
                );
            })}
        </Box>
    );
};

export default DataBindingsPanel;
