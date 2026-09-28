/**
 * The page application model: what a Page Studio page needs beyond layout
 * and Business Object widgets to be a working console - page state
 * (variables), governed data (queries against registered operations),
 * behavior (actions), and conditional display (conditions evaluated by the
 * one rule engine). Reverse-engineered from the hand-built mastering console
 * (the retired features/mastering/MasteringPage.tsx): every construct here exists
 * because that page needs it. See docs/page-studio-app-model.md.
 *
 * Governance boundaries (deliberate, do not loosen):
 *  - Queries and mutations name a registered operation id
 *    (studio-core/operations). A page never carries a URL, SQL, or script.
 *  - Bindings are path lookups and string templates only ({{vars.entity}}).
 *    Anything that decides - show/hide, enable, per-row state - is a
 *    ConditionNode evaluated by the rule engine's wasm build, never a JS
 *    expression and never a new operator switch.
 *  - Domain logic (derived fields, counts, messages) lives in the operation
 *    or domain-component adapter that owns the domain, not in the page.
 */

/** A value that may be a literal, or a string holding {{path}} templates. A string that is exactly one {{path}} resolves to the raw value. */
export type Binding = unknown;

/** Display text: a literal/template string (translated when it is a known i18n key) or an i18n key with template params. */
export type TextSpec = string | { t: string; params?: Record<string, Binding> };

/**
 * A condition, in the rule engine's own RuleNode JSON shape
 * (backend/internal/rules/vm/ast.go) - evaluated in the browser by
 * rule_engine.wasm's evaluateRule against the page scope. Field paths are
 * scope paths: vars.entity, queries.profile.data.series, row.state.
 */
export type ConditionNode =
  | { type: 'condition'; field: string; operator: string; value?: unknown; secondValue?: unknown }
  | { type: 'group'; operator: 'AND' | 'OR'; conditions: ConditionNode[] };

export interface PageVariable {
  name: string;
  /** Initial value. */
  default?: unknown;
  /** Seed from a query once it loads, while the variable is still empty (e.g. the first profile's entity). */
  initFrom?: { query: string; path: string };
  /** Mirror into the URL query string so the state survives reload and can be linked. */
  url?: boolean;
  description?: string;
}

export interface PageQuery {
  id: string;
  /** Registered operation id, e.g. mastering.golden.list. */
  operation: string;
  params?: Record<string, Binding>;
  /** Only run while this holds (e.g. an entity has been chosen). Absent = always. */
  enabledWhen?: ConditionNode;
  /** Keep showing the previous result while new params load (filters, paging). Search inputs debounce themselves. */
  keepPrevious?: boolean;
}

export interface PageAppModel {
  variables?: PageVariable[];
  queries?: PageQuery[];
  /** The variable the tab strip reads and writes, so actions can switch tabs. */
  tabVariable?: string;
  /** 'none' hides the default title + /slug chrome (a PageHeader widget replaces it). */
  chrome?: 'default' | 'none';
  /** Content width cap and padding, like a hand-built console's container. */
  surface?: { maxWidth?: number; padding?: number };
}

// ---------------------------------------------------------------------------
// Actions

export type Action =
  | { kind: 'setVariable'; name: string; value?: Binding }
  | {
      kind: 'runOperation';
      operation: string;
      params?: Record<string, Binding>;
      /** Ask first. */
      confirm?: { title: TextSpec; text?: TextSpec; confirmLabel?: TextSpec };
      /** Collect input first; its values are {{form.<name>}} in params and onSuccess. */
      form?: FormSpec;
      /** Run after success; the operation's result is {{result}}. */
      onSuccess?: Action[];
      /** Show on success. */
      successMessage?: TextSpec;
    }
  | { kind: 'navigate'; to: Binding }
  | { kind: 'notify'; severity: 'success' | 'info' | 'warning' | 'error'; text: TextSpec };

export interface FormFieldSpec {
  name: string;
  label: TextSpec;
  /** chips: a list of strings typed as chips; date: an ISO date. */
  kind: 'text' | 'multiline' | 'number' | 'radio' | 'select' | 'switch' | 'chips' | 'date';
  options?: { value: string; label: TextSpec }[];
  /** Options from a query: rows at rowsPath, value/label read from each row (omit = the row itself). */
  optionsFrom?: OptionsFrom;
  default?: Binding;
  required?: boolean;
  helperText?: TextSpec;
  /** Show the field only while this holds; read-only while readOnlyWhen holds. */
  visibleWhen?: ConditionNode;
  readOnlyWhen?: ConditionNode;
}

export interface OptionsFrom {
  query: string;
  rowsPath?: string;
  valueField?: string;
  labelField?: string;
}

export interface FormSpec {
  title: TextSpec;
  /** Explanatory text above the fields. */
  intro?: TextSpec;
  /** A callout shown when its condition holds (e.g. "needs 2 approvals"). */
  notice?: { severity: 'info' | 'warning'; text: TextSpec; visibleWhen?: ConditionNode };
  fields: FormFieldSpec[];
  submitLabel?: TextSpec;
  submitColor?: 'primary' | 'inherit' | 'error';
}

// ---------------------------------------------------------------------------
// Overlay and tab containers (layout nodes; app/containers.tsx)

/** A footer button of a Dialog (or Drawer). */
export interface ContainerButton {
  label: TextSpec;
  variant?: 'text' | 'outlined' | 'contained';
  color?: 'primary' | 'inherit' | 'error' | 'secondary';
  disabledWhen?: ConditionNode;
  visibleWhen?: ConditionNode;
  onClick: Action[];
}

/** LayoutNode.props when type === 'Drawer' or 'Dialog'. */
export interface OverlayNodeProps {
  /** Open while this holds, e.g. vars.goldenId is_not_empty. */
  openWhen?: ConditionNode;
  title?: TextSpec;
  subtitle?: TextSpec;
  /** Drawer: side and width. */
  anchor?: 'right' | 'left';
  width?: number;
  /** Dialog: width class. */
  maxWidth?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  /** Closing (X, backdrop, Escape) runs these - usually clearing the variable openWhen reads. */
  onClose?: Action[];
  /** Footer buttons. */
  buttons?: ContainerButton[];
}

/** LayoutNode.props when type === 'TabSet'; tab i shows children[i]. */
export interface TabSetNodeProps {
  tabs?: { id: string; label: TextSpec; badge?: Binding; visibleWhen?: ConditionNode }[];
  /** The page variable holding the active tab id (so actions can switch it); omit for local state. */
  variable?: string;
}

// ---------------------------------------------------------------------------
// Rich table cells (DataGrid columns)

export type ChipColor = 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info';

export type CellSpec = CellBody & {
  /** Show this cell only when the condition holds for the row. */
  visibleWhen?: ConditionNode;
};

export type CellBody =
  /** value (raw) or text (translated when it names an i18n key). */
  | { kind: 'text'; value?: Binding; text?: TextSpec; mono?: boolean; bold?: boolean; nowrap?: boolean; caption?: boolean; color?: 'error' | 'warning' | 'info' | 'secondary' }
  /** Primary line plus a caption line (name over code). */
  | { kind: 'twoLine'; primary: Binding; secondary?: Binding; secondaryMono?: boolean }
  | { kind: 'number'; value?: Binding; digits?: number; suffix?: Binding }
  /** 0..1 ratio shown as a whole percent. */
  | { kind: 'percent'; value?: Binding }
  /** A signed percent coloured by magnitude (warn/bad thresholds on |v|). */
  | { kind: 'delta'; value?: Binding; warn?: number; bad?: number }
  | { kind: 'datetime'; value?: Binding; caption?: boolean }
  /** A change: the old value struck through, then the new one in bold. */
  | { kind: 'diff'; before: Binding; after: Binding }
  | {
      kind: 'chip';
      value?: Binding;
      /** i18n key prefix for the label ('mastering.goldenStatus.' + value). */
      labelKey?: string;
      label?: TextSpec;
      colorMap?: Record<string, ChipColor>;
      /** Pick the colour from a different value than the label (severity colours a type chip). */
      colorBy?: Binding;
      color?: ChipColor;
      variant?: 'filled' | 'outlined';
      tooltip?: Binding;
      /** Extra caption after the chip (v{{row.version}}). */
      caption?: Binding;
    }
  /** Chips for each entry of an object (counts) or array; zero/empty entries skipped. */
  | { kind: 'chips'; value?: Binding; labelKey?: string; colorMap?: Record<string, ChipColor>; showCount?: boolean; keys?: string[] }
  /** A text button that runs actions (open a record). */
  | { kind: 'link'; label: TextSpec; onClick: Action[]; after?: Binding }
  /** Row buttons, each with its own condition. */
  | { kind: 'actions'; buttons: RowButton[]; caption?: { text: TextSpec; visibleWhen?: ConditionNode } }
  /** An inline input whose value is {{rowState.<name>}} for this row's actions. */
  | { kind: 'input'; name: string; placeholder?: TextSpec };

export interface RowButton {
  label: TextSpec;
  variant?: 'text' | 'outlined' | 'contained';
  color?: 'primary' | 'inherit' | 'error' | 'secondary';
  visibleWhen?: ConditionNode;
  onClick: Action[];
}

export interface ColumnDef {
  id: string;
  header?: TextSpec;
  align?: 'left' | 'right' | 'center';
  nowrap?: boolean;
  minWidth?: number;
  /** Shorthand for {kind:'text', value:'{{row.<field>}}'} when cell is absent. */
  field?: string;
  cell?: CellSpec;
  /** Wrap the cell in a tooltip. */
  tooltip?: Binding;
  /** Several cells stacked in one column (a caption under a chip, an input over buttons). */
  stack?: CellSpec[];
  /** Lay the stack out in a row instead of a column. */
  stackDirection?: 'row' | 'column';
  visibleWhen?: ConditionNode;
}
