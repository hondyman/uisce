-- Remove business_object catalog_node_type added by 20260812000001_add_business_object_catalog_node_type.up.sql

DELETE FROM public.catalog_node_type WHERE catalog_type_name = 'business_object';
