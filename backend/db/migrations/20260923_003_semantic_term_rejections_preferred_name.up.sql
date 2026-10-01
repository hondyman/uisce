-- preferred_name: when a user rejects a derived name and retypes a replacement,
-- the replacement is stored here. pickFirstNonRejected checks this column first:
-- if preferred_name is non-NULL and non-empty, it wins immediately.

ALTER TABLE sml.semantic_term_rejections
    ADD COLUMN IF NOT EXISTS preferred_name text;
