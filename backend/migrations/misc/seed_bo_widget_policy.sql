-- Seeds bo_widget_policy and bo_widget_breakpoint_fallback from the two
-- mappings this consolidates: the frontend FieldDataType->FieldWidget
-- inference in frontend/src/hooks/useCRUDPageConfig.ts (inferWidget,
-- re-verified present at that exact path in the main tree, not a worktree,
-- during the 2026-09-06 recovery pass) and the backend
-- component_extensibility/forms/generator.go 3-type mock stub
-- ("string"/"number"/"reference"). The seed is the consolidation — once the
-- backend generator reads from bo_widget_policy instead of its own switch,
-- generator.go's stub can be deleted.
-- Idempotent: ON CONFLICT DO NOTHING on both tables.
--
-- RECOVERY NOTE (2026-09-06): this file was deleted, pre-commit, by a
-- concurrent session's git-clean-class operation on a shared working tree.
-- The 13 policy rows and 2 fallback rows below were reconstructed by
-- dumping the live table content from alpha (`SELECT * FROM
-- bo_widget_policy`), not retyped from memory — confirmed byte-for-byte
-- match against what's actually live, and re-running this file against
-- alpha is a confirmed no-op.

INSERT INTO public.bo_widget_policy (field_type, cardinality, default_widget_key, allowed_widget_keys) VALUES
    ('string',    'one',  'text',        ARRAY['text', 'textarea', 'autocomplete', 'hidden']),
    ('number',    'one',  'number',      ARRAY['number', 'hidden']),
    ('integer',   'one',  'number',      ARRAY['number', 'hidden']),
    ('boolean',   'one',  'switch',      ARRAY['switch', 'checkbox']),
    ('date',      'one',  'date',        ARRAY['date']),
    ('datetime',  'one',  'datetime',    ARRAY['datetime', 'date']),
    ('time',      'one',  'time',        ARRAY['time']),
    ('json',      'one',  'json-editor', ARRAY['json-editor', 'textarea']),
    ('enum',      'one',  'select',      ARRAY['select', 'radio', 'autocomplete']),
    ('array',     'many', 'multiselect', ARRAY['multiselect']),
    ('uuid',      'one',  'text',        ARRAY['text', 'hidden', 'lookup']),
    -- 'reference' consolidates generator.go's "reference" -> Select mapping.
    ('reference', 'one',  'lookup',      ARRAY['lookup', 'select', 'autocomplete']),
    ('reference', 'many', 'multiselect', ARRAY['multiselect', 'lookup'])
ON CONFLICT (field_type, cardinality) DO NOTHING;

-- Widgets with no honest mobile equivalent get a declared fallback rather
-- than rendering squeezed. Conservative starter set — extend as real
-- mobile-designer usage surfaces more cases; do not fabricate not_supported
-- rows without a concrete widget behind them.
INSERT INTO public.bo_widget_breakpoint_fallback (widget_key, breakpoint, fallback_widget_key) VALUES
    ('json-editor', 'mobile', 'textarea'),
    ('rich-text',   'mobile', 'textarea')
ON CONFLICT (widget_key, breakpoint) DO NOTHING;
