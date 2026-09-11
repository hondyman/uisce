-- 20260910_002_report_folders.down.sql
-- Down migration for private per-user report folders and folder item mappings
-- WARNING: Destructive operation. Irreversibly drops all user report folder structures and folder-item associations.

DROP TABLE IF EXISTS public.report_folder_items;
DROP TABLE IF EXISTS public.report_folders;
