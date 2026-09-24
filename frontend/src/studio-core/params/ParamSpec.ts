/**
 * The one typed parameter shape for any studio surface that prompts for
 * runtime values: Report Studio's report parameters, a future Page Studio
 * filter/param surface, and the API/Query builder's param inputs. Per
 * HANDOFF_REPORT_BUILDER_SPINE_PLAN.md Phase 1 (spine extraction, ticket
 * 1.2) — before this, `ReportParameter` was declared three times
 * (SSRSReportBuilder.tsx, ParametersDialog.tsx, ReportParametersToolbar.tsx)
 * with byte-identical fields and no shared source of truth.
 *
 * Field names deliberately kept as the original `ReportParameter` shape
 * (`id`/`name`/`prompt`/`allowBlank`/`allowMultiple`) rather than the
 * `key`/`label`/`required`/`multiSelect` rename an earlier draft of this
 * ticket proposed: `FilterBuilderPanel.tsx` (`handleCreateParameter`,
 * `ValueInput`, `ConditionRow`) is a fourth real consumer of this shape,
 * untyped (`parameters?: any[]`), that reads/writes `.name`/`.prompt`
 * directly — a rename there would compile clean (nothing to catch it) and
 * break filter param-binding at runtime. Renaming is still worth doing
 * later, just as its own ticket that touches `FilterBuilderPanel.tsx`
 * deliberately, not as a silent side effect of this one. This ticket is a
 * pure additive extension: zero behavior change for every existing
 * consumer, new optional fields for the capabilities the spine plan needs.
 */
export type ParamValueType = 'string' | 'number' | 'date' | 'boolean' | 'ref';

export interface ParamOption {
  value: string | number | boolean;
  label: string;
}

/**
 * Where a param's value or option list comes from at runtime. Omitted on
 * `ParamSpec` = 'static' (a plain user-entered/default value) — today's
 * only behavior for every existing param. `'ref'` is the BO-term fence:
 * options must come from `termKey` on `boKey`'s BO (the same fenced
 * vocabulary `GetBOTerms` gives the field picker), never free SQL.
 */
export type ParamSource =
  | { kind: 'static'; options?: ParamOption[] }
  | { kind: 'url' }
  | { kind: 'ref'; boKey: string; termKey: string };

export interface ParamSpec {
  id: string;
  name: string;
  type: ParamValueType;
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
  /** Where the value/options come from. Omitted = static, unchanged from today. */
  source?: ParamSource;
  /**
   * Cascading: this param's option set is filtered by the value(s) of its
   * parent param(s). `relationshipId` names an edge from
   * `GetBusinessObjectRelationships` — the one relationship API
   * (studio-core/binding/boRelationships.ts) — so the cascade is graph
   * traversal, never a hand-written dependency. The runtime filter this
   * implies ("parent's selected value -> the join condition -> my option
   * set") is the same shape `masterFilter` already resolves for the Slicer
   * widget; a future cascade implementation should be a thin wrapper over
   * that, not a second implementation to keep in agreement with the first.
   */
  cascadeFrom?: {
    parentParamId: string;
    relationshipId: string;
    direction: 'parent-to-child' | 'child-to-parent';
  }[];
  /** Format-only display concerns; no behavior. */
  presentation?: { helpText?: string; hidden?: boolean };
}

/** @deprecated Import `ParamSpec` directly — kept only during the migration window. */
export type ReportParameter = ParamSpec;
