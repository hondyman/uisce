-- 20261227_003_region_postgres_cluster (up)
-- Which Postgres cluster a region's tenant databases are created on. Provisioning a tenant "in
-- US East" resolves the region here: a region with no active row has no cluster and is refused, so a
-- database is never created on a host nobody chose. No credential is stored: the worker holds the
-- administrator credential of the cluster it serves and refuses a region that names another.

CREATE TABLE IF NOT EXISTS public.region_postgres_cluster (
    region_code character varying(50) PRIMARY KEY
        REFERENCES public.region_config (region_code) ON DELETE RESTRICT,
    host        text    NOT NULL CHECK (host <> ''),
    port        integer NOT NULL DEFAULT 5432 CHECK (port BETWEEN 1 AND 65535),
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamp with time zone NOT NULL DEFAULT now(),
    updated_at  timestamp with time zone NOT NULL DEFAULT now()
);

COMMENT ON TABLE public.region_postgres_cluster IS
    'Postgres cluster that holds the tenant databases of a region. Read by tenant provisioning; holds no credentials.';

-- The first region and its cluster: the native Postgres host. The region row is added only if it is
-- missing, so an existing configuration is never changed.
INSERT INTO public.region_config (id, region_code, region_name, description, is_active)
SELECT gen_random_uuid(), 'us-east-1', 'US East (N. Virginia)', 'US East', true
WHERE NOT EXISTS (SELECT 1 FROM public.region_config WHERE region_code = 'us-east-1');

INSERT INTO public.region_postgres_cluster (region_code, host, port)
VALUES ('us-east-1', '100.84.50.65', 5432)
ON CONFLICT (region_code) DO NOTHING;
