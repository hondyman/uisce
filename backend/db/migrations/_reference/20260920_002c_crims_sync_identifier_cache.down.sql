-- Rollback for 20260920_002c_crims_sync_identifier_cache.up.sql
-- Run: psql ... -f this_file  (against crims database)

BEGIN;

CREATE OR REPLACE FUNCTION orm.sync_identifier_cache()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.is_primary AND NEW.effective_to IS NULL THEN
        IF NEW.id_type = 'ISIN'  THEN UPDATE orm.security SET isin  = NEW.id_value WHERE id = NEW.security_id;
        ELSIF NEW.id_type = 'CUSIP' THEN UPDATE orm.security SET cusip = NEW.id_value WHERE id = NEW.security_id;
        ELSIF NEW.id_type = 'TICKER' THEN UPDATE orm.security SET ticker = NEW.id_value WHERE id = NEW.security_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_sec_ident_cache
    BEFORE INSERT OR UPDATE ON orm.security_identifier
    FOR EACH ROW EXECUTE FUNCTION orm.sync_identifier_cache();

COMMIT;
