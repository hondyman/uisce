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
  label: string;
  icon?: string;
  defaultProps?: Record<string, unknown>;
  category?: string;
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
  componentId: string;
  props?: Record<string, unknown>;
  children?: LayoutNode[];
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
  type: 'api' | 'database' | 'static';
  config: Record<string, unknown>;
}

export interface CorePageDefinition {
  id: string;
  name: string;
  slug: string;
  description?: string;
  layout: LayoutNode[];
  components: ComponentDefinition[];
  dataSources: DataSourceDefinition[];
  createdAt: string;
  updatedAt: string;
  version?: number;
}
