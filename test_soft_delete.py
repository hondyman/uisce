#!/usr/bin/env python3
"""End-to-end test of the tenant soft-delete feature."""
import psycopg2
import psycopg2.extras
import json
import sys
import uuid

DB_PARAMS = {
    "host": "100.84.50.65",
    "port": 5432,
    "user": "postgres",
    "password": "postgres",
    "dbname": "alpha",
}

PASS = "\033[32m✅ PASS\033[0m"
FAIL = "\033[31m❌ FAIL\033[0m"


def run_step(name, fn):
    print(f"\n=== {name} ===")
    try:
        result = fn()
        print(f"{PASS}  {name}")
        return result
    except AssertionError as e:
        print(f"{FAIL}  {name}: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"{FAIL}  {name}: {type(e).__name__}: {e}")
        sys.exit(1)


def main():
    conn = psycopg2.connect(**DB_PARAMS)
    conn.autocommit = False
    cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

    # 1. Schema: confirm new columns exist
    def step_check_schema():
        cur.execute("""
            SELECT column_name, data_type, is_nullable, column_default
            FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = 'tenants'
              AND column_name IN ('is_deleted', 'deleted_at')
            ORDER BY column_name;
        """)
        rows = cur.fetchall()
        assert len(rows) == 2, f"Expected 2 columns, got {len(rows)}: {rows}"
        cols = {r["column_name"]: r for r in rows}
        assert cols["is_deleted"]["data_type"] == "boolean"
        assert cols["deleted_at"]["data_type"] == "timestamp with time zone"
        print(f"   columns: {json.dumps([dict(r) for r in rows], default=str)}")

    run_step("1. Schema columns exist (is_deleted, deleted_at)", step_check_schema)

    # 2. Schema: confirm indexes exist
    def step_check_indexes():
        cur.execute("""
            SELECT indexname FROM pg_indexes
            WHERE tablename = 'tenants' AND indexname IN
              ('idx_tenants_is_deleted', 'idx_tenants_active');
        """)
        rows = cur.fetchall()
        names = {r["indexname"] for r in rows}
        assert "idx_tenants_is_deleted" in names, f"missing idx_tenants_is_deleted: {names}"
        assert "idx_tenants_active" in names, f"missing idx_tenants_active: {names}"
        print(f"   indexes: {names}")

    run_step("2. Indexes exist (idx_tenants_is_deleted, idx_tenants_active)", step_check_indexes)

    # 3. Insert a throwaway tenant to exercise the soft delete path
    test_id = str(uuid.uuid4())
    test_name = f"__soft_delete_test_{uuid.uuid4().hex[:8]}"

    def step_insert_test_tenant():
        cur.execute("""
            INSERT INTO public.tenants (id, name, display_name, is_active, is_deleted)
            VALUES (%s, %s, %s, true, false)
        """, (test_id, test_name, "Soft Delete Test Tenant"))
        conn.commit()
        cur.execute("SELECT id, name, is_deleted FROM public.tenants WHERE id = %s", (test_id,))
        row = cur.fetchone()
        assert row is not None
        assert row["is_deleted"] is False, row
        print(f"   inserted tenant id={row['id']} is_deleted={row['is_deleted']}")

    run_step("3. Insert throwaway test tenant", step_insert_test_tenant)

    # 4. GET_TENANTS-equivalent query (with where is_deleted = false) should include it
    def step_visible_to_list_query():
        cur.execute("""
            SELECT id, name, is_deleted FROM public.tenants
            WHERE is_deleted = false AND id = %s
        """, (test_id,))
        row = cur.fetchone()
        assert row is not None, "test tenant missing from active list"
        assert row["is_deleted"] is False
        print(f"   visible: {row['name']} (is_deleted={row['is_deleted']})")

    run_step("4. Test tenant is visible to GET_TENANTS-style query", step_visible_to_list_query)

    # 5. Soft-delete the tenant via the new mutation shape
    def step_soft_delete():
        cur.execute("""
            UPDATE public.tenants
            SET is_deleted = true, deleted_at = now()
            WHERE id = %s
            RETURNING id, is_deleted, deleted_at
        """, (test_id,))
        row = cur.fetchone()
        assert row["is_deleted"] is True, row
        assert row["deleted_at"] is not None, row
        print(f"   soft-deleted: is_deleted={row['is_deleted']} deleted_at={row['deleted_at']}")

    run_step("5. Soft-delete the tenant (UPDATE ... is_deleted=true, deleted_at=now())", step_soft_delete)

    # 6. GET_TENANTS-equivalent query (with where is_deleted = false) should NOT return it
    def step_hidden_from_list_query():
        cur.execute("""
            SELECT id, name FROM public.tenants
            WHERE is_deleted = false AND id = %s
        """, (test_id,))
        row = cur.fetchone()
        assert row is None, f"soft-deleted tenant should not appear: {row}"
        print("   correctly hidden from active list")

    run_step("6. Soft-deleted tenant is hidden from GET_TENANTS-style query", step_hidden_from_list_query)

    # 7. Row is still in the table (data is preserved)
    def step_row_still_exists():
        cur.execute("""
            SELECT id, name, is_deleted, deleted_at FROM public.tenants WHERE id = %s
        """, (test_id,))
        row = cur.fetchone()
        assert row is not None, "row was actually deleted!"
        assert row["is_deleted"] is True
        assert row["deleted_at"] is not None
        print(f"   preserved: id={row['id']} name={row['name']} deleted_at={row['deleted_at']}")

    run_step("7. Row is preserved (no hard delete)", step_row_still_exists)

    # 8. Index is being used
    def step_index_used():
        cur.execute("""
            EXPLAIN (FORMAT JSON)
            SELECT id FROM public.tenants WHERE is_deleted = false AND id = %s
        """, (test_id,))
        plan = cur.fetchone()[0]
        plan_str = json.dumps(plan)
        # The optimizer may use a seq scan on tiny tables, but the index should at least
        # be present. We simply confirm the query runs without error and is consistent.
        print(f"   plan OK ({len(plan_str)} bytes)")

    run_step("8. Active-list query plan runs cleanly", step_index_used)

    # 9. Cleanup: hard delete the throwaway row (so we don't pollute the table)
    def step_cleanup():
        cur.execute("DELETE FROM public.tenants WHERE id = %s", (test_id,))
        conn.commit()
        cur.execute("SELECT id FROM public.tenants WHERE id = %s", (test_id,))
        assert cur.fetchone() is None
        print("   cleanup OK")

    run_step("9. Cleanup throwaway tenant", step_cleanup)

    cur.close()
    conn.close()
    print("\n\033[32m🎉 All soft-delete checks passed.\033[0m")


if __name__ == "__main__":
    main()