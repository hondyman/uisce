-- Point the Security business object's MDM binding at the security master.
--
-- The Security BO (c2000000-...0003) is one business concept with two physical homes: the OMS
-- instrument (/orm/security, the default binding) and the MDM security master. Its second binding
-- ("Security MDM Golden Record Binding", added by db/manual_fixes/add_security_mdm_binding.sql) pointed
-- at mdm.security_golden_record, whose values live in jsonb, so no field could be mapped. The master
-- anchor mdm.security_master has real columns and is bitemporal (valid_from / valid_to), so the binding
-- now drives it. Unused until now: no field bindings, no rules scoped to it.
--
-- Idempotent.

UPDATE public.business_object_binding
   SET driving_node_id           = 'f50cd533-e786-5fa3-a7b5-305dd3f2e6aa',  -- /mdm/security_master
       binding_name              = 'Security Master Binding',
       temporal_mode             = 'BITEMPORAL',
       valid_from_column_node_id = '222638d2-befe-51d5-82dd-b4f25b79e293',  -- valid_from
       valid_to_column_node_id   = '9783addf-facc-5b14-8143-765e37c2a936',  -- valid_to
       updated_at                = now()
 WHERE bo_binding_id = '2ad873ed-2c77-4721-93a6-f2afac6546fa'
   AND bo_id = 'c2000000-0000-4000-8000-000000000003';
