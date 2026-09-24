import React from 'react';
import { Box, Paper, Typography, IconButton } from '@mui/material';
import { Delete as DeleteIcon } from '@mui/icons-material';
import { useDroppable, useDndMonitor, type DragEndEvent } from '@dnd-kit/core';
import { PageLayout, ComponentDefinition, DataSourceDefinition, PanelNodeProps, BusinessObjectDataSourceConfig } from '../../types/pageStudio';
import PageComponentRenderer from './PageComponentRenderer';
import PanelRegion from './PanelRegion';
import type { FieldDragPayload } from './DataBindingsPanel';
import type { FieldLayoutEntry } from './FormFieldsDesigner';
import type { RelatedObjectDragPayload } from '../../studio-core/binding/boRelationships';
import { widgetTypeForCardinality } from '../../studio-core/binding/boRelationships';
import { ensureRelatedDataSource, isBindableWidget } from '../../studio-core/binding/ensureRelatedDataSource';

/** Selecting one field of a Form (Design mode) reuses the same `selectedId`
 * string LayoutCanvas already tracks for nodes/components, as
 * `${componentId}::field::${fieldName}` - PropertiesPanel.tsx parses this
 * same convention to show that field's properties instead of the whole
 * Form's. */
const FIELD_SELECTION_SEP = '::field::';

const DEFAULT_PANEL_PROPS: PanelNodeProps = { side: 'right', collapsible: true, defaultOpen: true, widthPx: 320, label: 'Panel' };
const FIELD_DROP_TYPES = ['Slicer', 'LineChart', 'KPIGroup', 'Table', 'Form'];

interface LayoutCanvasProps {
    layout: PageLayout;
    onLayoutChange: (updater: (layout: PageLayout) => PageLayout) => void;
    components: Record<string, ComponentDefinition>;
    onComponentsChange: (updater: (components: Record<string, ComponentDefinition>) => Record<string, ComponentDefinition>) => void;
    dataSources: DataSourceDefinition[];
    onDataSourcesChange: (updater: (dataSources: DataSourceDefinition[]) => DataSourceDefinition[]) => void;
    tenantId: string;
    selectedId: string | null;
    onSelect: (id: string | null) => void;
}

/** Drop target wrapping a Row/Column so related-BO chips and palette tiles
 * can land even when the container already has children. */
const ContainerSlot: React.FC<{
    nodeId: string;
    nodeType: string;
    selected: boolean;
    empty: boolean;
    onSelect: () => void;
    style?: React.CSSProperties;
    children: React.ReactNode;
}> = ({ nodeId, nodeType, selected, empty, onSelect, style, children }) => {
    const { setNodeRef, isOver } = useDroppable({ id: `slot:${nodeId}`, data: { kind: 'slot', nodeId } });
    return (
        <Box
            ref={setNodeRef}
            onClick={(e) => { e.stopPropagation(); onSelect(); }}
            sx={{
                outline: selected ? '2px solid' : isOver ? '2px dashed' : '1px dashed',
                outlineColor: selected ? 'primary.main' : isOver ? 'primary.main' : 'rgba(0,0,0,0.12)',
                outlineOffset: 2,
                p: 0,
                position: 'relative',
                minWidth: 0,
                flex: 1,
            }}
        >
            <Typography variant="caption" sx={{ position: 'absolute', top: -14, left: 0, px: 0.5, color: 'text.secondary', pointerEvents: 'none', fontSize: 10 }}>
                {nodeType}
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', flexDirection: nodeType === 'Row' ? 'row' : 'column', gap: 2, ...style }}>
                {children}
                {empty && (
                    <Box sx={{ p: 0.5, textAlign: 'center', flex: 1, minHeight: 32, border: '1px dashed rgba(0,0,0,0.1)', borderRadius: 1 }}>
                        <Typography variant="caption" color="textSecondary">Drop here</Typography>
                    </Box>
                )}
            </Box>
        </Box>
    );
};

/**
 * One placed widget - a field-drop target when its type accepts one. Hoisted
 * to module scope (not defined inside LayoutCanvas) so its useDroppable
 * hook keeps a stable component identity across re-renders instead of
 * remounting - and with it losing drag state - every time LayoutCanvas
 * re-renders, which happens on every keystroke while editing a widget.
 */
const ComponentBlock: React.FC<{
    comp: ComponentDefinition;
    selectedId: string | null;
    dataSources: DataSourceDefinition[];
    tenantId: string;
    onSelect: (id: string | null) => void;
    onDelete: (id: string) => void;
    onUpdateFieldLayout: (compId: string, fieldName: string, entry: Partial<FieldLayoutEntry>) => void;
    onReorderFields: (compId: string, orderedFieldNames: string[]) => void;
    onUnhideField: (compId: string, fieldName: string) => void;
}> = ({ comp, selectedId, dataSources, tenantId, onSelect, onDelete, onUpdateFieldLayout, onReorderFields, onUnhideField }) => {
    const acceptsField = FIELD_DROP_TYPES.includes(comp.type);
    const { setNodeRef, isOver } = useDroppable({ id: `comp:${comp.id}`, data: { kind: 'component-target' }, disabled: !acceptsField });
    const fieldPrefix = `${comp.id}${FIELD_SELECTION_SEP}`;
    const selectedFieldName = selectedId?.startsWith(fieldPrefix) ? selectedId.slice(fieldPrefix.length) : null;
    const selected = selectedId === comp.id || selectedFieldName !== null;
    return (
        <Paper
            ref={setNodeRef}
            onClick={(e) => { e.stopPropagation(); onSelect(comp.id); }}
            elevation={0}
            sx={{
                p: 0,
                position: 'relative',
                border: isOver ? '2px dashed' : '1px solid',
                borderColor: isOver ? 'secondary.main' : selected ? 'primary.main' : 'transparent',
                borderRadius: 1,
                // A resizable widget (comp.style.resize set via the Properties
                // panel's Layout section) needs `flex: 0 0 auto` so its own
                // width/height style takes effect and the resize handle has
                // room to shrink/grow it independently of the row/column -
                // `flex: 1` would just snap it back to fill its container.
                flex: comp.style?.resize ? '0 0 auto' : 1,
                minWidth: 0,
                bgcolor: 'transparent',
                '&:hover .widget-chrome': { opacity: 1 },
            }}
        >
            <Box
                className="widget-chrome"
                sx={{
                    position: 'absolute', top: 4, right: 4, zIndex: 2,
                    display: 'flex', alignItems: 'center', gap: 0.5,
                    opacity: selected ? 1 : 0,
                    bgcolor: 'background.paper', borderRadius: 1, px: 0.5,
                    boxShadow: 1,
                }}
            >
                <Typography variant="caption" color="text.secondary">{comp.type}</Typography>
                <IconButton size="small" onClick={(e) => { e.stopPropagation(); onDelete(comp.id); }}>
                    <DeleteIcon fontSize="small" />
                </IconButton>
            </Box>
            <PageComponentRenderer
                component={comp}
                dataSources={dataSources || []}
                tenantId={tenantId || 'default'}
                mode="design"
                selectedFieldName={selectedFieldName}
                onSelectField={(name) => onSelect(name ? `${fieldPrefix}${name}` : comp.id)}
                onUpdateFieldLayout={(name, entry) => onUpdateFieldLayout(comp.id, name, entry)}
                onReorderFields={(names) => onReorderFields(comp.id, names)}
                onUnhideField={(name) => onUnhideField(comp.id, name)}
            />
        </Paper>
    );
};

/**
 * Renders and edits ONE tab's layout tree. Operates on `layout`/`components`
 * directly rather than the whole CorePageDefinition draft, so PageEditor.tsx
 * can point it at whichever tab is active without this component needing
 * any tab awareness itself - it was previously coupled to the full draft,
 * which made it impossible to reuse per-tab without a larger rewrite.
 *
 * Two instances of this component are mounted simultaneously by PageEditor
 * (the page-wide filter bar, and the active tab's canvas), sharing ONE
 * @dnd-kit DndContext mounted there. Each instance's useDndMonitor below
 * only acts on a drop when the target node/component id actually belongs to
 * ITS OWN layout tree, so the two canvases never step on each other's drops.
 */
const LayoutCanvas: React.FC<LayoutCanvasProps> = ({
    layout, onLayoutChange, components, onComponentsChange, dataSources, onDataSourcesChange, tenantId, selectedId, onSelect,
}) => {
    const handleFieldDrop = (payload: FieldDragPayload, comp: ComponentDefinition) => {
        const isMeasure = payload.role === 'MEASURE' || payload.role === 'CALCULATED';
        onComponentsChange((prev) => {
            const existing = prev[comp.id];
            if (!existing) return prev;
            const patch: Record<string, unknown> = {};
            // Rebind this widget to the dropped field's Business Object too,
            // in case it wasn't bound yet or was pointed at a different one -
            // dropping a field is a complete "use this data" gesture, not
            // just a field pick within an already-chosen source.
            const boSource = dataSources.find(
                (d) => d.type === 'business_object' && (d.config as unknown as BusinessObjectDataSourceConfig)?.boId === payload.boId
            );
            if (boSource) patch.dataSourceId = boSource.id;
            if (existing.type === 'Slicer') {
                patch.dimensionTermIds = [payload.termNodeId];
            } else if (existing.type === 'KPIGroup') {
                if (isMeasure) patch.measureTermIds = [payload.termNodeId];
            } else if (existing.type === 'LineChart') {
                if (isMeasure) patch.measureTermIds = [payload.termNodeId];
                else patch.dimensionTermIds = [payload.termNodeId];
            } else if (existing.type === 'Table') {
                const key = payload.termKey || payload.displayName;
                const cols = existing.props?.visibleColumns as string[] | undefined;
                if (cols && cols.length > 0 && key && !cols.includes(key)) patch.visibleColumns = [...cols, key];
            } else if (existing.type === 'Form') {
                // Form shows the BO schema; drop just rebinds the source.
            } else {
                return prev;
            }
            return { ...prev, [comp.id]: { ...existing, props: { ...existing.props, ...patch } } };
        });
        onSelect(comp.id);
    };

    const handleDrop = (componentType: string, parentId: string) => {
        const newId = `${componentType.toLowerCase()}_${Math.random().toString(36).substr(2, 5)}`;

        if (['Row', 'Column', 'Panel'].includes(componentType)) {
            onLayoutChange((prev) => {
                const newNode = componentType === 'Panel'
                    ? { id: newId, type: 'Panel' as const, children: [], props: { ...DEFAULT_PANEL_PROPS } as Record<string, unknown> }
                    : { id: newId, type: componentType as 'Row' | 'Column', children: [] };
                const nodes = { ...prev.nodes, [newId]: newNode };
                const parent = nodes[parentId];
                nodes[parentId] = { ...parent, children: [...(parent.children || []), newId] };
                return { ...prev, nodes };
            });
        } else {
            onComponentsChange((prev) => ({ ...prev, [newId]: { id: newId, type: componentType, props: {} } }));
            onLayoutChange((prev) => {
                const parent = prev.nodes[parentId];
                return { ...prev, nodes: { ...prev.nodes, [parentId]: { ...parent, children: [...(parent.children || []), newId] } } };
            });
        }

        onSelect(newId);
    };

    // Graph-driven related-object drop: cardinality picks Table vs Form;
    // joinCondition supplies the child FK for masterFilter. The related BO
    // is added as its own data source if this page does not already have one.
    const applyRelatedSource = async (payload: RelatedObjectDragPayload): Promise<string | null> => {
        const result = await ensureRelatedDataSource(dataSources, payload);
        if (!result) return null;
        onDataSourcesChange(() => result.sources);
        return result.sourceId;
    };

    const handleRelatedObjectDrop = async (payload: RelatedObjectDragPayload, parentId: string) => {
        if (!(parentId in layout.nodes)) return;
        const sourceId = await applyRelatedSource(payload);
        if (!sourceId) return;
        const widgetType = widgetTypeForCardinality(payload.cardinality);
        const newId = `${widgetType.toLowerCase()}_${Math.random().toString(36).substr(2, 5)}`;
        onComponentsChange((prev) => ({
            ...prev,
            [newId]: { id: newId, type: widgetType, label: payload.relatedObjectName, props: { dataSourceId: sourceId } },
        }));
        onLayoutChange((prev) => {
            const parent = prev.nodes[parentId];
            if (!parent) return prev;
            return { ...prev, nodes: { ...prev.nodes, [parentId]: { ...parent, children: [...(parent.children || []), newId] } } };
        });
        onSelect(newId);
    };

    const handleRelatedBindExisting = async (payload: RelatedObjectDragPayload, compId: string) => {
        const comp = components[compId];
        if (!comp || !isBindableWidget(comp.type)) {
            const parentId = Object.keys(layout.nodes).find((id) => (layout.nodes[id].children || []).includes(compId));
            if (parentId) await handleRelatedObjectDrop(payload, parentId);
            return;
        }
        const unbound = !comp.props?.dataSourceId;
        if (!unbound) {
            const parentId = Object.keys(layout.nodes).find((id) => (layout.nodes[id].children || []).includes(compId));
            if (parentId) await handleRelatedObjectDrop(payload, parentId);
            return;
        }
        const sourceId = await applyRelatedSource(payload);
        if (!sourceId) return;
        onComponentsChange((prev) => {
            const existing = prev[compId];
            if (!existing) return prev;
            return { ...prev, [compId]: { ...existing, props: { ...existing.props, dataSourceId: sourceId }, label: payload.relatedObjectName } };
        });
        onSelect(compId);
    };

    // Design-mode field resize/reorder for a Form widget (FormFieldsDesigner.tsx) -
    // stored on that component's own props, keyed by field name, same shape
    // BOFormWidget.tsx reads at Preview/runtime.
    const handleUpdateFieldLayout = (compId: string, fieldName: string, entry: Partial<FieldLayoutEntry>) => {
        onComponentsChange((prev) => {
            const existing = prev[compId];
            if (!existing) return prev;
            const fieldLayout = { ...(existing.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined) };
            const current = fieldLayout[fieldName];
            fieldLayout[fieldName] = { order: current?.order ?? 0, colSpan: current?.colSpan ?? 12, ...entry };
            return { ...prev, [compId]: { ...existing, props: { ...existing.props, fieldLayout } } };
        });
    };

    const handleReorderFields = (compId: string, orderedFieldNames: string[]) => {
        onComponentsChange((prev) => {
            const existing = prev[compId];
            if (!existing) return prev;
            const prevLayout = (existing.props?.fieldLayout as Record<string, FieldLayoutEntry> | undefined) || {};
            // Keep hidden-field colSpan/order entries; the designer only
            // passes the currently visible names.
            const fieldLayout: Record<string, FieldLayoutEntry> = { ...prevLayout };
            orderedFieldNames.forEach((name, i) => {
                fieldLayout[name] = { colSpan: prevLayout[name]?.colSpan ?? 12, order: i };
            });
            return { ...prev, [compId]: { ...existing, props: { ...existing.props, fieldLayout } } };
        });
    };

    // Brings a deleted (hidden) field back - see PropertiesPanel.tsx's field
    // delete button and FormFieldsDesigner.tsx's "Deleted fields" chip list.
    const handleUnhideField = (compId: string, fieldName: string) => {
        onComponentsChange((prev) => {
            const existing = prev[compId];
            if (!existing) return prev;
            const fieldOverrides = { ...(existing.props?.fieldOverrides as Record<string, { hidden?: boolean }> | undefined) };
            fieldOverrides[fieldName] = { ...fieldOverrides[fieldName], hidden: false };
            return { ...prev, [compId]: { ...existing, props: { ...existing.props, fieldOverrides } } };
        });
    };

    const handleDelete = (id: string) => {
        onComponentsChange((prev) => {
            const next = { ...prev };
            delete next[id];
            return next;
        });
        onLayoutChange((prev) => {
            const nodes = { ...prev.nodes };
            delete nodes[id];
            for (const key of Object.keys(nodes)) {
                if (nodes[key].children) {
                    nodes[key] = { ...nodes[key], children: nodes[key].children!.filter((cid) => cid !== id) };
                }
            }
            return { ...prev, nodes };
        });
        onSelect(null);
    };

    // Only react to a drop whose target actually belongs to THIS instance's
    // own tree (see class doc comment above) - both the filter-bar canvas
    // and the active tab's canvas mount this monitor simultaneously against
    // one shared DndContext.
    useDndMonitor({
        onDragEnd(event: DragEndEvent) {
            const { active, over } = event;
            if (!over) return;
            const overId = String(over.id);
            const data = active.data.current as { kind?: string; componentType?: string; payload?: FieldDragPayload | RelatedObjectDragPayload } | undefined;
            if (!data) return;
            // Form field reorder is handled by FormFieldsDesigner.tsx's own
            // useDndMonitor against this same DndContext.
            if (data.kind === 'form-field') return;

            if (data.kind === 'related-object' && data.payload) {
                const payload = data.payload as RelatedObjectDragPayload;
                if (overId.startsWith('slot:')) {
                    const nodeId = overId.slice(5);
                    if (nodeId in layout.nodes) void handleRelatedObjectDrop(payload, nodeId);
                    return;
                }
                if (overId.startsWith('comp:')) {
                    const compId = overId.slice(5);
                    if (isReachable(layout, compId) || components[compId]) void handleRelatedBindExisting(payload, compId);
                    return;
                }
                return;
            }
            if (overId.startsWith('slot:') && data.kind === 'component' && data.componentType) {
                const nodeId = overId.slice(5);
                if (nodeId in layout.nodes) handleDrop(data.componentType, nodeId);
                return;
            }
            if (overId.startsWith('comp:') && data.kind === 'field' && data.payload) {
                const compId = overId.slice(5);
                const comp = components[compId];
                // A component belongs to this instance's tree only if its id
                // is reachable from this layout's root (see isReachable below) -
                // components are shared across the filter-bar and tab trees,
                // so this is what keeps the two canvases from double-handling.
                if (comp && isReachable(layout, compId)) handleFieldDrop(data.payload as FieldDragPayload, comp);
            }
        },
    });

    const renderNode = (nodeId: string): React.ReactNode => {
        if (!nodeId) return null;
        const node = layout?.nodes?.[nodeId];
        if (!node) {
            const component = components?.[nodeId];
            if (component) {
                return (
                    <ComponentBlock
                        key={component.id}
                        comp={component}
                        selectedId={selectedId}
                        dataSources={dataSources}
                        tenantId={tenantId}
                        onSelect={onSelect}
                        onDelete={handleDelete}
                        onUpdateFieldLayout={handleUpdateFieldLayout}
                        onReorderFields={handleReorderFields}
                        onUnhideField={handleUnhideField}
                    />
                );
            }
            return null;
        }

        const body = (
            <ContainerSlot
                key={node.type === 'Panel' ? undefined : nodeId}
                nodeId={nodeId}
                nodeType={node.type}
                selected={selectedId === nodeId}
                empty={!node.children || node.children.length === 0}
                onSelect={() => onSelect(nodeId)}
                style={node.style as React.CSSProperties | undefined}
            >
                {(node.children || []).map((childId) => renderNode(childId))}
            </ContainerSlot>
        );

        if (node.type === 'Panel') {
            const panelProps = { ...DEFAULT_PANEL_PROPS, ...(node.props as Partial<PanelNodeProps> | undefined) };
            return <Box key={nodeId}><PanelRegion {...panelProps}>{body}</PanelRegion></Box>;
        }
        return body;
    };

    // Boundary guard: a layout missing `root`/`nodes` (legacy shape drift on
    // an old saved draft, or a hand-edited/stale JSON blob) must not crash
    // the whole editor - render a visible notice instead of a silent blank
    // canvas, so an author sees why nothing is there rather than assuming
    // the page is genuinely empty. See HANDOFF_REPORT_BUILDER_SPINE_PLAN.md's
    // process-hygiene note for the root-cause follow-up (this guard fixes
    // the crash; it does not explain how a tab's layout got malformed).
    if (!layout?.root || !layout?.nodes) {
        return (
            <Box sx={{ minHeight: '100%', p: 2 }}>
                <Typography variant="body2" color="error">
                    This tab's layout data is malformed (missing root/nodes) and can't be rendered. This usually means the page was saved under an older layout shape.
                </Typography>
            </Box>
        );
    }

    return (
        <Box sx={{ minHeight: '100%' }}>
            {renderNode(layout.root)}
        </Box>
    );
};

/** Whether a component id is reachable (as a leaf, non-node child) from this layout's root. */
function isReachable(layout: PageLayout, id: string): boolean {
    const seen = new Set<string>();
    const walk = (nodeId: string): boolean => {
        if (nodeId === id && !layout.nodes[nodeId]) return true;
        const node = layout.nodes[nodeId];
        if (!node || seen.has(nodeId)) return false;
        seen.add(nodeId);
        return (node.children || []).some((childId) => (childId === id && !layout.nodes[childId]) || walk(childId));
    };
    return walk(layout.root);
}

export default LayoutCanvas;
