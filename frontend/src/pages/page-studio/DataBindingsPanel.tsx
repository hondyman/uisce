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
import { apiClient } from '../../utils/apiClient';
import { CorePageDefinition, DataSourceDefinition, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import { fetchBusinessObjectBindings } from '../../features/query-builder/services/queryBuilderApi';

interface BusinessObjectOption {
    id: string;
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
                boKey: bo.name,
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

    const toggleRelated = (source: DataSourceDefinition, relatedId: string) => {
        const cfg = source.config as unknown as BusinessObjectDataSourceConfig;
        const current = cfg.relatedBoIds || [];
        const next = current.includes(relatedId)
            ? current.filter((id) => id !== relatedId)
            : [...current, relatedId];
        setDraft((prev) => ({
            ...prev,
            dataSources: (prev.dataSources || []).map((d) =>
                d.id === source.id ? { ...d, config: { ...cfg, relatedBoIds: next } as unknown as Record<string, unknown> } : d
            ),
        }));
    };

    return (
        <Box sx={{ p: 2 }}>
            <Typography variant="overline" color="text.secondary">
                Business Object Data Sources
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
                    No Business Object data sources yet. Add one above to bind page widgets to it.
                </Typography>
            )}

            <List dense disablePadding>
                {boDataSources.map((ds) => {
                    const cfg = ds.config as unknown as BusinessObjectDataSourceConfig;
                    const related = relatedByBO[cfg.boId] || [];
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
                                <ListItemText primary={cfg.displayName} secondary={cfg.boKey} />
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
                                                        onClick={() => toggleRelated(ds, rel.targetObjectId)}
                                                    />
                                                </Tooltip>
                                            );
                                        })}
                                    </Box>
                                </Box>
                            )}
                            <Divider sx={{ mt: 2 }} />
                        </Box>
                    );
                })}
            </List>
        </Box>
    );
};

export default DataBindingsPanel;
