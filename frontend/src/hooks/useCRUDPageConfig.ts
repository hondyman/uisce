import { useState, useCallback, useEffect } from 'react';
import type {
    CRUDPageConfig,
    FieldConfig,
    RelationshipConfig,
    LayoutConfig,
    LayoutSection,
    BODefinition,
    CRUDGeneratorOptions,
    FieldDataType,
    FieldWidget,
    RelationshipType,
    UIRole,
} from '../types/crud-generator-types';

/**
 * Hook for generating CRUD page configuration from a Business Object definition
 * Automatically infers field widgets, layouts, and relationship sections
 * based on the BO's structure and its semantic relationships.
 */
export const useCRUDPageConfig = (
    boId: string | undefined,
    tenantId: string,
    datasourceId: string,
    options: CRUDGeneratorOptions = {}
) => {
    const [config, setConfig] = useState<CRUDPageConfig | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchBODefinition = useCallback(async (): Promise<BODefinition | null> => {
        if (!boId || !tenantId || !datasourceId) return null;

        const res = await fetch(`/api/business-objects/${boId}`, {
            headers: {
                'X-Tenant-ID': tenantId,
                'X-Tenant-Datasource-ID': datasourceId,
            },
        });

        if (!res.ok) {
            if (res.status === 404) {
                // Business object not found - return null gracefully instead of throwing
                return null;
            }
            throw new Error(`Failed to fetch business object: ${res.status} ${res.statusText}`);
        }
        return res.json();
    }, [boId, tenantId, datasourceId]);

    const fetchBORelationships = useCallback(async (): Promise<RelationshipConfig[]> => {
        if (!boId || !tenantId || !datasourceId) return [];

        try {
            const res = await fetch(`/api/relationships/bo/${boId}`, {
                headers: {
                    'X-Tenant-ID': tenantId,
                    'X-Tenant-Datasource-ID': datasourceId,
                },
            });

            if (!res.ok) return [];
            const data = await res.json();

            return (data || []).map((rel: any, idx: number) => ({
                id: rel.id || `rel-${idx}`,
                targetBO: rel.target_bo_name || rel.targetBOName,
                displayName: rel.display_name || rel.target_bo_name,
                relationshipType: rel.relationship_type || 'M:1',
                uiRole: inferUIRole(rel),
                isLookup: rel.lookup || rel.is_lookup || false,
                displayFields: rel.display_fields || [],
                canAddNew: rel.relationship_type === '1:M' || rel.relationship_type === 'M:M',
                canRemove: true,
            }));
        } catch {
            return [];
        }
    }, [boId, tenantId, datasourceId]);

    const generateConfig = useCallback(async () => {
        if (!boId) return;

        try {
            setLoading(true);
            setError(null);

            const [boDefinition, relationships] = await Promise.all([
                fetchBODefinition(),
                fetchBORelationships(),
            ]);

            if (!boDefinition) {
                throw new Error('Business object not found');
            }

            // Generate field configurations
            const fields = generateFieldConfigs(boDefinition, options);

            // Generate layout
            const layout = generateLayout(fields, relationships, options);

            const pageConfig: CRUDPageConfig = {
                boName: boDefinition.name,
                displayName: boDefinition.displayName || boDefinition.name,
                description: boDefinition.description,
                icon: boDefinition.icon,
                fields,
                relationships,
                layout,
                actions: [
                    { id: 'save', label: 'Save', type: 'submit', variant: 'primary' },
                    { id: 'cancel', label: 'Cancel', type: 'cancel', variant: 'secondary' },
                    { id: 'delete', label: 'Delete', type: 'delete', variant: 'danger', confirmMessage: 'Are you sure you want to delete this record?' },
                ],
                permissions: {
                    canCreate: true,
                    canRead: true,
                    canUpdate: true,
                    canDelete: true,
                },
            };

            setConfig(pageConfig);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to generate config');
        } finally {
            setLoading(false);
        }
    }, [boId, fetchBODefinition, fetchBORelationships, options]);

    useEffect(() => {
        generateConfig();
    }, [generateConfig]);

    return { config, loading, error, refresh: generateConfig };
};

// ============================================================================
// Generator Functions
// ============================================================================

function generateFieldConfigs(
    bo: BODefinition,
    options: CRUDGeneratorOptions
): FieldConfig[] {
    const fields: FieldConfig[] = [];

    for (const field of bo.fields) {
        if (options.excludeFields?.includes(field.name)) continue;

        const dataType = inferDataType(field.type);
        const widget = options.widgetMappings?.[field.type] || inferWidget(dataType, field);

        fields.push({
            name: field.name,
            label: field.label || formatLabel(field.name),
            type: dataType,
            widget,
            required: field.required || false,
            readOnly: false,
            colSpan: widget === 'textarea' || widget === 'json-editor' ? 12 : 6,
        });
    }

    // Add audit fields if requested
    if (options.includeAuditFields) {
        fields.push(
            { name: 'created_at', label: 'Created At', type: 'datetime', widget: 'datetime', required: false, readOnly: true, colSpan: 6 },
            { name: 'updated_at', label: 'Updated At', type: 'datetime', widget: 'datetime', required: false, readOnly: true, colSpan: 6 },
            { name: 'created_by', label: 'Created By', type: 'string', widget: 'text', required: false, readOnly: true, colSpan: 6 },
            { name: 'updated_by', label: 'Updated By', type: 'string', widget: 'text', required: false, readOnly: true, colSpan: 6 }
        );
    }

    return fields;
}

function generateLayout(
    fields: FieldConfig[],
    relationships: RelationshipConfig[],
    options: CRUDGeneratorOptions
): LayoutConfig {
    const lookupRelationships = relationships.filter(r => r.uiRole === 'lookup' || r.isLookup);
    const childRelationships = relationships.filter(r => r.uiRole === 'child_collection');
    const detailRelationships = relationships.filter(r => r.uiRole === 'detail');
    const associationRelationships = relationships.filter(r => r.uiRole === 'association');

    // If many relationships, use tabs
    if (options.useTabsForRelationships && relationships.length > 2) {
        return generateTabbedLayout(fields, lookupRelationships, childRelationships, detailRelationships, associationRelationships);
    }

    // Otherwise, use sections
    const sections: LayoutSection[] = [];

    // Main fields section
    const mainFields = fields.filter(f => !['created_at', 'updated_at', 'created_by', 'updated_by'].includes(f.name));
    sections.push({
        id: 'main-fields',
        title: 'Details',
        type: 'fields',
        fields: mainFields.map(f => f.name),
        columns: 2,
    });

    // Detail relationships (1:1) - embedded inline
    for (const rel of detailRelationships) {
        sections.push({
            id: `rel-${rel.id}`,
            title: rel.displayName,
            type: 'relationship',
            relationshipId: rel.id,
            collapsible: true,
        });
    }

    // Child collections (1:M)
    for (const rel of childRelationships) {
        sections.push({
            id: `rel-${rel.id}`,
            title: rel.displayName,
            type: 'relationship',
            relationshipId: rel.id,
            collapsible: true,
            collapsed: true,
        });
    }

    // Associations (M:M)
    for (const rel of associationRelationships) {
        sections.push({
            id: `rel-${rel.id}`,
            title: rel.displayName,
            type: 'relationship',
            relationshipId: rel.id,
            collapsible: true,
            collapsed: true,
        });
    }

    // Audit section
    const auditFields = fields.filter(f => ['created_at', 'updated_at', 'created_by', 'updated_by'].includes(f.name));
    if (auditFields.length > 0) {
        sections.push({
            id: 'audit',
            title: 'Audit Information',
            type: 'fields',
            fields: auditFields.map(f => f.name),
            collapsible: true,
            collapsed: true,
            columns: 2,
        });
    }

    const layout: LayoutConfig = {
        type: 'single',
        sections,
    };

    // Side panel for lookups
    if (options.generateSidePanel && lookupRelationships.length > 0) {
        layout.sidePanel = {
            enabled: true,
            width: 300,
            sections: lookupRelationships.map(rel => ({
                id: `side-${rel.id}`,
                title: rel.displayName,
                type: 'relationship',
                relationshipId: rel.id,
            })),
        };
    }

    return layout;
}

function generateTabbedLayout(
    fields: FieldConfig[],
    lookups: RelationshipConfig[],
    children: RelationshipConfig[],
    details: RelationshipConfig[],
    associations: RelationshipConfig[]
): LayoutConfig {
    const tabs = [];

    // Main tab
    const mainFields = fields.filter(f => !['created_at', 'updated_at', 'created_by', 'updated_by'].includes(f.name));
    tabs.push({
        id: 'main',
        label: 'Details',
        icon: 'info',
        sections: [{
            id: 'main-fields',
            title: 'Details',
            type: 'fields' as const,
            fields: mainFields.map(f => f.name),
            columns: 2,
        }],
    });

    // Related tab for child collections
    if (children.length > 0) {
        tabs.push({
            id: 'related',
            label: 'Related',
            icon: 'link',
            sections: children.map(rel => ({
                id: `rel-${rel.id}`,
                title: rel.displayName,
                type: 'relationship' as const,
                relationshipId: rel.id,
            })),
        });
    }

    // Associations tab
    if (associations.length > 0) {
        tabs.push({
            id: 'associations',
            label: 'Associations',
            icon: 'group_work',
            sections: associations.map(rel => ({
                id: `rel-${rel.id}`,
                title: rel.displayName,
                type: 'relationship' as const,
                relationshipId: rel.id,
            })),
        });
    }

    return {
        type: 'tabs',
        tabs,
    };
}

// ============================================================================
// Inference Functions
// ============================================================================

function inferDataType(typeString: string): FieldDataType {
    const type = typeString.toLowerCase();

    if (['int', 'integer', 'bigint', 'smallint'].includes(type)) return 'integer';
    if (['float', 'double', 'decimal', 'numeric', 'real', 'number'].includes(type)) return 'number';
    if (['bool', 'boolean'].includes(type)) return 'boolean';
    if (['date'].includes(type)) return 'date';
    if (['datetime', 'timestamp', 'timestamptz'].includes(type)) return 'datetime';
    if (['time'].includes(type)) return 'time';
    if (['json', 'jsonb'].includes(type)) return 'json';
    if (['uuid'].includes(type)) return 'uuid';
    if (['array'].includes(type) || type.endsWith('[]')) return 'array';
    if (['enum'].includes(type) || type.startsWith('enum')) return 'enum';

    return 'string';
}

function inferWidget(dataType: FieldDataType, field: any): FieldWidget {
    // Check for lookup relationship
    if (field.semanticTermId || field.lookup) {
        return 'lookup';
    }

    switch (dataType) {
        case 'boolean':
            return 'switch';
        case 'integer':
        case 'number':
            return 'number';
        case 'date':
            return 'date';
        case 'datetime':
            return 'datetime';
        case 'time':
            return 'time';
        case 'json':
            return 'json-editor';
        case 'enum':
            return 'select';
        case 'array':
            return 'multiselect';
        default:
            // Check for long text
            if (field.name.includes('description') || field.name.includes('notes') || field.name.includes('comment')) {
                return 'textarea';
            }
            return 'text';
    }
}

function inferUIRole(rel: any): UIRole {
    if (rel.lookup || rel.is_lookup || rel.ui_role === 'lookup') {
        return 'lookup';
    }

    const type = rel.relationship_type || rel.relationshipType;
    switch (type) {
        case '1:1':
            return 'detail';
        case '1:M':
            return 'child_collection';
        case 'M:M':
            return 'association';
        default:
            return 'lookup';
    }
}

function formatLabel(name: string): string {
    return name
        .replace(/_/g, ' ')
        .replace(/([a-z])([A-Z])/g, '$1 $2')
        .split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
}
