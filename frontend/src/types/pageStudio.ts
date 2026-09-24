/**
 * ZERO-PAGE WINDOW: no page definitions persist durably today. The frontend
 * (src/api/pageStudio.ts) calls `/api/page-studio`, but that route has no
 * backend implementation anywhere in the Go codebase (checked repo-wide) —
 * the same unwired-API pattern found in BO Studio's governance screens.
 * There is no serialized-format compatibility to preserve, but also no
 * existing page definitions this schema needs to be compatible *with* —
 * these types are defining the format for the first real pages ever saved.
 * That's the cheapest this design gets: once pages persist, decisions like
 * the closed breakpoint enum and whole-overlay-replace below become
 * migrations instead of type changes. Treat that as a closing window, not
 * a permanent freedom.
 */
export interface ComponentDefinition {
  id: string;
  type: string;
  label?: string;
  icon?: string;
  /** Palette-catalog defaults; a placed widget instance uses `props` instead. */
  defaultProps?: Record<string, unknown>;
  category?: string;
  /** Widget instance config (data binding, text content, expression source, etc). */
  props?: Record<string, unknown>;
  /** Free-form CSS-ish style overrides applied to the widget's wrapper (font, color, spacing). */
  style?: Record<string, string>;
}

/**
 * Breakpoints the responsive designer targets. Desktop is always the base
 * layout (LayoutNode's own props/style/children), never an override key —
 * there is no `responsive.desktop`. Closed set deliberately: adding a
 * fourth breakpoint later is a schema change here plus a new fallback-table
 * key in bo_widget_breakpoint_fallback, not a free extension.
 */
export type ResponsiveBreakpoint = 'mobile' | 'tablet';

/**
 * A per-breakpoint override. Whole-overlay replace, not field-by-field
 * merge: if a breakpoint has an override at all, every property below is
 * read from this overlay, falling back to the base LayoutNode's own value
 * only for fields the overlay omits *within this object* (e.g. an overlay
 * that sets `hidden` but not `span` still inherits `span` from base) —
 * but the overlay object itself is never merged with a sibling breakpoint's
 * overlay, and there is no cascade (mobile does not inherit from tablet).
 * A node is either overridden at a breakpoint or it isn't; the builder UI's
 * inherited-vs-overridden indicator reads directly off presence of this key.
 *
 * Structurally constrained on purpose: this can change presentation only.
 * It cannot add, remove, or re-parent nodes — the same node set and the
 * same entitlement/capability filtering apply at every breakpoint. Widgets
 * with no honest equivalent at this breakpoint are handled by
 * bo_widget_breakpoint_fallback, not by swapping componentId here.
 */
export interface ResponsiveOverride {
  /** Hide this node entirely at this breakpoint. */
  hidden?: boolean;
  /** Grid column span override (interpretation is the renderer's, same units as base). */
  span?: number;
  /** Sort position among siblings at this breakpoint. */
  order?: number;
  /** Stack direction for this node's children at this breakpoint. */
  stackDirection?: 'row' | 'column';
}

export interface LayoutNode {
  id: string;
  /**
   * Container kind - Row/Column stack their children horizontally/vertically.
   * Panel is a Column that can additionally slide open/collapsed as a side
   * region (see PanelNodeProps below, read out of `props`) - used for
   * things like a filters/detail rail next to a page's main content.
   */
  type: 'Row' | 'Column' | 'Panel';
  /** Unused by plain Row/Column containers; reserved for a future componentId-backed layout node kind. */
  componentId?: string;
  /** Row/Column: unused today. Panel: see PanelNodeProps. */
  props?: Record<string, unknown>;
  /**
   * Child ids, resolved against the SAME PageLayout.nodes map for a nested
   * container or against CorePageDefinition.components for a placed widget
   * - a flat id reference, not a nested LayoutNode tree (LayoutCanvas.tsx /
   * PageBrowser.tsx both resolve children this way).
   */
  children?: string[];
  style?: Record<string, string>;
  /**
   * Sparse per-breakpoint overrides. Omitted entirely for nodes that render
   * identically everywhere (the common case) — this is an overlay on top of
   * the base layout above, not a parallel tree per breakpoint.
   *
   * VALIDATION RULE (designed here, not yet wired): before a page is saved,
   * every componentId reachable from this tree that maps to a widget_key
   * must resolve via bo_widget_policy + bo_widget_breakpoint_fallback for
   * 'mobile' and 'tablet' — either the widget renders as-is (no fallback
   * row) or a fallback_widget_key exists. A widget resolving to
   * not_supported (a fallback row with a null fallback_widget_key) blocks
   * the save unless this node declares a `hidden: true` override at that
   * breakpoint. This rule has no save path to live in yet — implementing
   * it is part of the governance-routes unit, currently blocked on the
   * BO-service triage decision (see backend/internal/pagebuilder/BO_SERVICES.md).
   * Do not implement this check against a mocked wire format in the
   * interim; it should land against the real save endpoint.
   *
   * Any client-side preview of this rule (e.g. in the responsive designer
   * UI, before a save path exists) MUST be visibly marked as a preview —
   * "stub says unrenderable" is not the same claim as "server validated,"
   * and the UI must not let an author confuse the two. See
   * pages/page-studio/RESPONSIVE_DESIGNER.md's "unrenderable" state.
   */
  responsive?: Partial<Record<ResponsiveBreakpoint, ResponsiveOverride>>;
}

export interface DataSourceDefinition {
  id: string;
  name: string;
  type: 'api' | 'database' | 'static' | 'business_object';
  config: Record<string, unknown>;
}

/** DataSourceDefinition.config shape when type === 'business_object'. */
export interface BusinessObjectDataSourceConfig {
  boId: string;
  boKey: string;
  bindingId: string;
  displayName: string;
  relatedBoIds: string[];
  /**
   * Master-detail child scoping: when set, a Table/Form bound to this data
   * source only shows rows where `fkField` equals the record id of the
   * page's current selection (see SelectionContext.tsx) - e.g. an
   * "Order Allocations" data source with fkField: "order_id" only shows
   * allocations for the Order row currently selected in a master Table
   * elsewhere on the page (any tab). Absent for the master data source
   * itself and for data sources not participating in master-detail at all.
   */
  masterFilter?: { fkField: string };
}

/**
 * The tree of layout structure (Row/Column containers), keyed by node id.
 * `root` names the entry node. This is the actual runtime/persisted shape
 * (PageStudioPage.tsx, LayoutCanvas.tsx, PageBrowser.tsx all read/write it
 * this way, and it's what the page_studio_handler.go backend round-trips
 * as-is) - it was previously mis-declared here as a bare `LayoutNode[]`,
 * which never matched any real caller and produced a long tail of
 * spurious "Property 'nodes'/'root' does not exist" type errors.
 */
export interface PageLayout {
  root: string;
  nodes: Record<string, LayoutNode>;
}

/** LayoutNode.props shape when type === 'Panel'. */
export interface PanelNodeProps {
  side: 'left' | 'right';
  /** When false, the panel is a fixed-width rail with no toggle. */
  collapsible: boolean;
  defaultOpen: boolean;
  widthPx: number;
  /** Shown in the panel's collapse-toggle tooltip and its collapsed-rail strip. */
  label?: string;
}

/**
 * One tab of a multi-tab page. Each tab owns its own layout tree, but all
 * tabs on a page share the same `components` map (CorePageDefinition.components)
 * and `dataSources` - a widget id is unique across the whole page, not
 * per-tab, so switching tabs never needs to remap ids.
 */
export interface PageTab {
  id: string;
  label: string;
  layout: PageLayout;
}

/** PeopleSoft-style page events that only change presentation. */
export type PresentationEventKind = 'pageActivate' | 'fieldChange' | 'fieldEdit' | 'selectionChange';

export type PresentationTargetKind = 'widget' | 'section' | 'field';

export interface PresentationTarget {
  kind: PresentationTargetKind;
  /** Widget id, layout node id, or form widget id when kind is field. */
  id: string;
  fieldName?: string;
}

export interface PresentationAction {
  target: PresentationTarget;
  hidden?: boolean;
  readOnly?: boolean;
  collapsed?: boolean;
  style?: Partial<Record<'color' | 'backgroundColor' | 'fontWeight' | 'borderColor', string>>;
  /** Display caption only — does not rename the Business Object term. */
  label?: string;
  /**
   * Report Studio only (Crystal/SSRS's "suppress repeated value"): when the
   * value of this cell/field is unchanged from the previous row within
   * `scope`, blank it instead of repeating it. Additive, optional, and
   * deliberately unread by Page Studio's renderer (`PageComponentRenderer.tsx`
   * only checks `hidden`/`readOnly`/`style`/`label`) — see
   * HANDOFF_REPORT_BUILDER_SPINE_PLAN.md Phase 1 ticket 1.4. Do not add a
   * switch/effect-type dispatch for this; the action-merge model in
   * `applyActions` (presentationEvents.ts) stays flat and additive so new
   * report-only fields cost nothing for Page Studio to ignore.
   */
  suppressRepeat?: 'band' | 'page';
}

export interface PresentationRule {
  id: string;
  event: PresentationEventKind;
  label?: string;
  /** Optional: which field/widget this rule is authored against (UI grouping + FieldEdit filter). */
  source?: { kind: 'field' | 'widget'; id: string; termKey?: string };
  /** ASL expression. Empty = always apply on this event. */
  when: string;
  actions: PresentationAction[];
}

export interface CorePageDefinition {
  id: string;
  name: string;
  slug: string;
  description?: string;
  env?: string;
  tenantId?: string;
  /**
   * The page's layout when it has no tabs. Once `tabs` is set (even to a
   * single tab), `tabs[].layout` is authoritative and this field is a
   * stale leftover from before the page was split into tabs - kept only
   * so old, already-saved single-layout pages still load without a
   * migration step.
   */
  layout: PageLayout;
  /** When present, the page is multi-tab; each tab supplies its own layout. */
  tabs?: PageTab[];
  /**
   * Layout tree rendered ABOVE the tab strip, shared across every tab -
   * for page-wide filter widgets (Slicers) that should scope every tab's
   * data, not just one. Uses the same `components`/`dataSources` maps as
   * `tabs`/`layout`; a component id lives in exactly one tree (this one,
   * or a tab's), never both. Optional: most pages have no page-wide
   * filters and this is omitted entirely.
   */
  filterBar?: PageLayout;
  /** Placed widget instances, keyed by id (same keys layout nodes' children reference), shared across all tabs. */
  components: Record<string, ComponentDefinition>;
  dataSources: DataSourceDefinition[];
  /**
   * Presentation-only rules (PeopleSoft FieldChange analog). They hide,
   * restyle, collapse, or relabel widgets/sections. They never save,
   * delete, or set field values — Business Object events own CRUD.
   */
  presentationEvents?: PresentationRule[];
  dataBindings?: { sources: Record<string, unknown>; bindings: unknown[] };
  visibility?: { roles: string[] };
  /** True for a gold-copy-authored page every tenant inherits read-only; false for an ordinary tenant-authored page. Set at creation, not editable afterward. */
  isCore?: boolean;
  /** False when this is an inherited gold-copy page the current tenant cannot mutate. */
  editable?: boolean;
  status?: 'draft' | 'published';
  /** list | detail | master-detail | dashboard — authoring intent, not a layout id. */
  pageKind?: 'list' | 'detail' | 'master-detail' | 'dashboard';
  createdAt: string;
  updatedAt: string;
  version?: number;
}
