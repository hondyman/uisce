CREATE TABLE IF NOT EXISTS public.metric_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255),
    description text,
    domain varchar(255),
    granularity varchar(50), -- 'day', 'month', etc.
    aggregation_function varchar(50), -- 'SUM', 'AVG', etc.
    base_query text,
    dimensions jsonb DEFAULT '[]'::jsonb,
    sla_config jsonb DEFAULT '{}'::jsonb,
    owner varchar(255),
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    
    CONSTRAINT metric_definitions_pk PRIMARY KEY (id),
    CONSTRAINT metric_definitions_name_unique UNIQUE (name)
);

CREATE INDEX IF NOT EXISTS metric_definitions_domain_idx ON public.metric_definitions (domain);
