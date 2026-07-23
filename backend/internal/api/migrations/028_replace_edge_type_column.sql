-- Migration: Replace edge_type with edge_type_name in catalog_edge
-- Created: 2026-01-23

-- 1. Ensure edge_type_name column exists (redundant if already added)
ALTER TABLE public.catalog_edge ADD COLUMN IF NOT EXISTS edge_type_name text;

-- 2. Populate edge_type_name from catalog_edge_type if possible
UPDATE public.catalog_edge ce
SET edge_type_name = cet.edge_type_name
FROM public.catalog_edge_type cet
WHERE ce.edge_type_id = cet.id
AND ce.edge_type_name IS NULL;

-- 3. Drop the old unique constraint
ALTER TABLE public.catalog_edge DROP CONSTRAINT IF EXISTS catalog_edge_unique;

-- 4. Create the new unique constraint using edge_type_name
ALTER TABLE public.catalog_edge 
ADD CONSTRAINT catalog_edge_unique 
UNIQUE (tenant_datasource_id, source_node_id, edge_type_name, target_node_id);

-- 5. Update index for performance
DROP INDEX IF EXISTS catalog_edge_tenant_datasource_id_idx;
CREATE INDEX IF NOT EXISTS catalog_edge_tenant_datasource_id_idx 
ON public.catalog_edge (tenant_datasource_id, source_node_id, edge_type_name, target_node_id);
