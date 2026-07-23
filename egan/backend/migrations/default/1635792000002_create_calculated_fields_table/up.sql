CREATE TABLE calculated_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    object_type TEXT NOT NULL,
    condition_def TEXT,
    aggregation_func TEXT,
    source_fields TEXT,
    return_type TEXT NOT NULL
);