// src/utils/pageMerge.ts
import {
    CorePageDefinition,
    PageOverlay,
    EffectivePageDefinition,
    LayoutTree,
    LayoutNode,
    LayoutOverrides,
    ComponentDefinition,
    ComponentOverride,
    VisibilityDefinition
} from "../types/pageStudio";

export function mergePage(
    core: CorePageDefinition,
    overlay?: PageOverlay
): EffectivePageDefinition {
    if (!overlay) return { ...core };

    const effective: EffectivePageDefinition = {
        ...core,
        layout: mergeLayout(core.layout, overlay.overrides.layout),
        components: mergeComponents(core.components, overlay.overrides.components),
        dataBindings: mergeDataBindings(
            core.dataBindings,
            overlay.overrides.dataBindings
        ),
        visibility: mergeVisibility(core.visibility, overlay.overrides.visibility),
        _inheritance: {
            components: {},
            layoutNodes: {},
        },
    };

    // Annotate inheritance
    Object.keys(core.components).forEach((id) => {
        if (overlay.overrides.components?.[id]) {
            effective._inheritance!.components[id] = { origin: "overridden" };
        } else {
            effective._inheritance!.components[id] = { origin: "core" };
        }
    });

    Object.keys(overlay.overrides.components || {}).forEach((id) => {
        if (!core.components[id]) {
            effective._inheritance!.components[id] = { origin: "tenant-only" };
        }
    });

    Object.keys(core.layout.nodes).forEach((id) => {
        if (overlay.overrides.layout?.nodes?.[id]) {
            effective._inheritance!.layoutNodes[id] = { origin: "overridden" };
        } else {
            effective._inheritance!.layoutNodes[id] = { origin: "core" };
        }
    });

    return effective;
}

function mergeLayout(core: LayoutTree, ov?: LayoutOverrides): LayoutTree {
    if (!ov) return JSON.parse(JSON.stringify(core));
    const result: LayoutTree = JSON.parse(JSON.stringify(core));

    if (ov.root) result.root = ov.root;
    if (ov.nodes) {
        Object.entries(ov.nodes).forEach(([id, nodeOv]) => {
            const base: LayoutNode | undefined = result.nodes[id];
            if (!base) {
                result.nodes[id] = { id, type: "Row", ...nodeOv } as LayoutNode;
            } else {
                if (nodeOv.children) base.children = nodeOv.children;
                if (nodeOv.props) base.props = { ...(base.props || {}), ...nodeOv.props };
            }
        });
    }
    return result;
}

function mergeComponents(
    core: Record<string, ComponentDefinition>,
    ov?: Record<string, ComponentOverride>
): Record<string, ComponentDefinition> {
    if (!ov) return JSON.parse(JSON.stringify(core));
    const result: Record<string, ComponentDefinition> = JSON.parse(JSON.stringify(core));

    Object.entries(ov).forEach(([id, cOv]) => {
        const base = result[id];
        if (!base) {
            result[id] = { id, type: "Custom", props: cOv.props };
        } else {
            base.props = { ...base.props, ...cOv.props };
        }
    });
    return result;
}

function mergeDataBindings(
    core: CorePageDefinition["dataBindings"],
    ov?: PageOverlay["overrides"]["dataBindings"]
): CorePageDefinition["dataBindings"] {
    if (!ov) return JSON.parse(JSON.stringify(core));
    const result = JSON.parse(JSON.stringify(core));

    if (ov.bindings) {
        ov.bindings.forEach((b) => {
            const idx = result.bindings.findIndex(
                (ex: any) => ex.componentId === b.componentId && ex.prop === b.prop
            );
            if (idx !== -1) result.bindings[idx] = b;
            else result.bindings.push(b);
        });
    }
    return result;
}

function mergeVisibility(
    core: VisibilityDefinition,
    ov?: VisibilityDefinition
): VisibilityDefinition {
    if (!ov) return { ...core };
    return {
        roles: Array.from(new Set([...(core.roles || []), ...(ov.roles || [])])),
        entitlement_policies: Array.from(
            new Set([...(core.entitlement_policies || []), ...(ov.entitlement_policies || [])])
        ),
    };
}
