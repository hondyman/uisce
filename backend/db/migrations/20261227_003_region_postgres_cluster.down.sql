-- 20261227_003_region_postgres_cluster (down)
-- Drops the region-to-cluster table. The us-east-1 region row it may have added is left: other
-- configuration can refer to it.
DROP TABLE IF EXISTS public.region_postgres_cluster;
