CREATE TABLE rule_criteria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL,
    group_id TEXT,
    field TEXT NOT NULL,
    operator TEXT NOT NULL,
    comparison_type TEXT NOT NULL,
    comparison_value TEXT NOT NULL,
    FOREIGN KEY (rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE
);