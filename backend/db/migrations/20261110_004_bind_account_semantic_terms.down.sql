-- Unbind ACCOUNT attribute_def rows whose terms were created by the bind seed.
-- Leaves pre-existing typed-column terms (account_cd, …) and their rules alone
-- when those terms were not solely created for attribute_def rows.

UPDATE public.attribute_def a
   SET semantic_term_id = NULL,
       updated_at = now()
 WHERE a.entity_type = 'ACCOUNT'
   AND a.semantic_term_id IN (
       SELECT cn.id
       FROM public.catalog_node cn
       WHERE cn.properties->>'source' = 'account_semantic_bind'
   );

DELETE FROM public.semantic_survivorship_rules r
 WHERE r.entity_type = 'ACCOUNT'
   AND r.semantic_term_id IN (
       SELECT cn.id FROM public.catalog_node cn
       WHERE cn.properties->>'source' = 'account_semantic_bind'
   );

-- Optional: leave catalog_node terms in place (other sessions may reference them).
