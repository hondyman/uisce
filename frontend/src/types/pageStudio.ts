// src/types/pageStudio.ts

export type LayoutNodeType = "Row" | "Column" | "Tabs" | "Card" | "Grid";

export interface LayoutNode {
    id: string;
    type: LayoutNodeType;
    children?: string[]; // IDs of child nodes or components
    props?: Record<string, any>;
}

export interface LayoutTree {
    root: string;
    nodes: Record<string, LayoutNode>;
}

export interface ActionDefinition {
    type: "navigate" | "mutate" | "refresh" | "openModal" | "closeModal" | "setState";
    targetPageId?: string;
    targetComponentId?: string;
    mutationSourceId?: string;
    params?: Record<string, any>;
    stateKey?: string;
    stateValue?: any;
}

export interface ComponentEventConfig {
    event: string; // e.g., "onRowClick", "onSubmit"
    actions: ActionDefinition[];
}

export interface ComponentDefinition {
    id: string;
    type: string;
    props: Record<string, any>;
    events?: ComponentEventConfig[];
    visibility?: { expression: string };
    dynamicProps?: { prop: string; expression: string }[];
}

export interface DataSourceDefinition {
    id: string;
    type: "rest" | "graphql";
    endpointId: string;
    query?: string; // For GraphQL
    args?: Record<string, any>;
}

export interface DataBinding {
    componentId: string;
    prop: string;
    sourceId: string;
    path: string;
}

export interface VisibilityDefinition {
    roles?: string[];
    entitlement_policies?: string[];
}

export interface CorePageDefinition {
    id: string;
    env: string;
    tenantId?: string; // null for core
    name: string;
    slug: string;
    description?: string;
    layout: LayoutTree;
    components: Record<string, ComponentDefinition>;
    dataBindings: {
        sources: Record<string, DataSourceDefinition>;
        bindings: DataBinding[];
    };
    visibility: VisibilityDefinition;
    version: number;
}

// Overlays
export interface LayoutNodeOverride {
    children?: string[];
    props?: Record<string, any>;
}

export interface LayoutOverrides {
    root?: string;
    nodes?: Record<string, LayoutNodeOverride>;
}

export interface ComponentOverride {
    props: Record<string, any>;
}

export interface PageOverlay {
    id: string;
    parentId: string;
    env: string;
    tenantId: string;
    overrides: {
        layout?: LayoutOverrides;
        components?: Record<string, ComponentOverride>;
        dataBindings?: {
            bindings?: DataBinding[];
        };
        visibility?: VisibilityDefinition;
    };
}

// Effective Page (Run-time)
export interface EffectivePageDefinition extends CorePageDefinition {
    _inheritance?: {
        components: Record<string, { origin: "core" | "overridden" | "tenant-only" }>;
        layoutNodes: Record<string, { origin: "core" | "overridden" | "tenant-only" }>;
    };
}

// Themes
export interface ThemeTokens {
    colors: Record<string, string>;
    typography: Record<string, any>;
    spacing: Record<string, number>;
    borderRadius: number;
}

export interface ThemeDefinition {
    id: string;
    name: string;
    tokens: ThemeTokens;
}

export interface TenantThemeOverride {
    tenantId: string;
    parentThemeId: string;
    overrides: Partial<ThemeTokens>;
}
// Tenant Upgrades
export type UpgradeStatus = "pending" | "accepted" | "partially_applied" | "dismissed";

export interface ConflictItem {
    type: "componentProp" | "layout" | "binding" | "visibility";
    componentId?: string;
    nodeId?: string;
    propName?: string;
    coreBefore: any;
    coreAfter: any;
    tenantOverride: any;
}

export interface ChangeItem {
    type: "componentProp" | "layout" | "binding" | "visibility";
    componentId?: string;
    nodeId?: string;
    propName?: string;
    before: any;
    after: any;
}

export interface UpgradeImpact {
    id: string;
    corePageId: string;
    coreOldVersion: number;
    coreNewVersion: number;
    tenantId: string;
    overlayPageId: string;
    summary: string;
    conflicts: ConflictItem[];
    inheritedChanges: ChangeItem[];
    newCoreComponents: string[];
    removedCoreComponents: string[];
    status: UpgradeStatus;
    createdAt: string;
    updatedAt: string;
}
