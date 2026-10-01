-- 20261118_002a_seed_master_data_menu_parent.down.sql
-- Intentionally a no-op. The "Master Data" menu node pre-dates this migration on
-- alpha (it was inserted directly), so this migration does not own the row and a
-- rollback must not delete it: that would also orphan or cascade the menu nodes
-- that hang off it.
SELECT 1;
