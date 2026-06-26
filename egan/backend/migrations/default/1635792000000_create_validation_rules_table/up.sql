CREATE TABLE validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'disabled',
    scope_type TEXT NOT NULL,
    scope_ref TEXT,
    message_type TEXT NOT NULL,
    message TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);