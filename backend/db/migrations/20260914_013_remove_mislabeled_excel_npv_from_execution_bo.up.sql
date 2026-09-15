-- Remove the mislabeled "Excel NPV" field from the northwind tenant's execution
-- Business Object. The field references semantic term
-- semantic_term/Performance.excel_npv (a calculated valuation formula:
-- =NPV({rate}, {cash_flows})) but was attached to the execution BO with
-- field_role='DIMENSION' and binding_requirement='REQUIRED' - mislabeling a
-- calculated valuation concept as a required execution attribute. No physical
-- column on orm.execution matches it; QueryBORecords' defensive column-existence
-- guard was dropping it on every /data read, leaving the field present in
-- metadata but invisible at the SQL layer - the worst kind of metadata drift:
-- visible enough to be misleading, dead enough to never fire.
--
-- Removal (not re-purposing) is the right action: the term lives on
-- semantic_term/Performance.excel_npv in the catalog with its full formula
-- intact, ready to be re-attached to a BO that actually models valuation
-- (Order, Position, or a future Valuation BO) when one of those needs NPV.
-- For the Execution BO, where it never belonged, the field is just noise.
--
-- Tenant-scoped: only the northwind tenant's execution BO is touched. Other
-- tenants' BOs are unaffected - some legitimate user may have a use for this
-- field elsewhere.
--
-- Pre-flight diagnostic for the runbook:
--   SELECT bf.id, bf.field_name, cn.qualified_path, bo.bo_key, t.name AS tenant
--   FROM business_object_fields bf
--   JOIN business_objects bo ON bo.id = bf.bo_id
--   JOIN tenants t ON t.id = bf.tenant_id
--   LEFT JOIN catalog_node cn ON cn.id = bf.term_node_id
--   WHERE bf.field_name = 'Excel NPV';

BEGIN;

-- Guard: only touch the northwind tenant's execution BO's field, so a
-- coincidental field_name collision in another tenant doesn't get caught
-- in this fix.
DELETE FROM public.business_object_fields bf
USING public.business_objects bo, public.tenants t
WHERE bf.bo_id = bo.id
  AND bo.tenant_id = t.id
  AND t.name = 'northwind'
  AND bo.bo_key = 'execution'
  AND bf.field_name = 'Excel NPV';

-- Verify the deletion landed (the row count should be > 0 on a fresh apply;
-- zero on a re-apply, which is also fine because DELETE is idempotent in this
-- context - the WHERE clause would match nothing the second time).
DO $$
DECLARE
    remaining INT;
BEGIN
    SELECT COUNT(*) INTO remaining
    FROM public.business_object_fields bf
    JOIN public.business_objects bo ON bo.id = bf.bo_id
    JOIN public.tenants t ON t.id = bo.tenant_id
    WHERE t.name = 'northwind'
      AND bo.bo_key = 'execution'
      AND bf.field_name = 'Excel NPV';

    IF remaining > 0 THEN
        RAISE EXCEPTION 'Excel NPV field still present on northwind execution BO after DELETE (% rows remaining)', remaining;
    END IF;

    RAISE NOTICE 'Excel NPV field removed from northwind execution BO';
END $$;

COMMIT;
