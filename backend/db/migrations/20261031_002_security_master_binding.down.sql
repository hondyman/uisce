UPDATE public.business_object_binding
   SET driving_node_id = (SELECT id FROM public.catalog_node WHERE qualified_path = '/mdm/security_golden_record'
                            AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' LIMIT 1),
       binding_name = 'Security MDM Golden Record Binding', temporal_mode = 'NONE',
       valid_from_column_node_id = NULL, valid_to_column_node_id = NULL, updated_at = now()
 WHERE bo_binding_id = '2ad873ed-2c77-4721-93a6-f2afac6546fa';
