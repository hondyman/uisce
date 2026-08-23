ALTER TABLE catalog_node
ADD CONSTRAINT catalog_node_tenant_path_uniq UNIQUE (tenant_id, qualified_path);
