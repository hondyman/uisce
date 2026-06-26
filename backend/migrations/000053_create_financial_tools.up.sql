CREATE TABLE IF NOT EXISTS public.financial_tools (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name varchar(255) NOT NULL,
    description text,
    parameters_schema jsonb DEFAULT '{}'::jsonb,
    handler_type varchar(50) NOT NULL, -- 'internal', 'script', 'api'
    handler_config jsonb DEFAULT '{}'::jsonb,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    
    CONSTRAINT financial_tools_pk PRIMARY KEY (id),
    CONSTRAINT financial_tools_name_unique UNIQUE (name)
);

CREATE INDEX financial_tools_handler_type_idx ON public.financial_tools (handler_type);
