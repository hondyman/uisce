-- 20261216_001_platform_lakehouse (down)
--
-- Drops the platform warehouse's provisioning record only. The bucket, the warehouse and the credential
-- are untouched: a bucket under Object Lock is not something a migration removes.
DROP TRIGGER IF EXISTS trg_platform_lakehouse_guard ON public.platform_lakehouse;
DROP FUNCTION IF EXISTS public.platform_lakehouse_guard();
DROP TABLE IF EXISTS public.platform_lakehouse;
