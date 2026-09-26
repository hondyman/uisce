-- Safe cast helpers for custom_attributes preview/validation on crims.
-- Idempotent. One bad JSONB value must not abort the whole query.

CREATE OR REPLACE FUNCTION mdm.safe_int(text)
RETURNS int LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE
        WHEN $1 IS NULL THEN NULL
        WHEN $1 ~ '^-?\d+$' THEN $1::int
        ELSE NULL
    END
$$;

CREATE OR REPLACE FUNCTION mdm.safe_numeric(text)
RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE
        WHEN $1 IS NULL THEN NULL
        WHEN $1 ~ '^-?\d+(\.\d+)?([eE][-+]?\d+)?$' THEN $1::numeric
        ELSE NULL
    END
$$;

CREATE OR REPLACE FUNCTION mdm.safe_date(text)
RETURNS date LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE
        WHEN $1 IS NULL THEN NULL
        WHEN $1 ~ '^\d{4}-\d{2}-\d{2}$' THEN $1::date
        ELSE NULL
    END
$$;

CREATE OR REPLACE FUNCTION mdm.safe_bool(text)
RETURNS bool LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE
        WHEN $1 IS NULL THEN NULL
        WHEN lower($1) IN ('true','t','1','yes','y') THEN true
        WHEN lower($1) IN ('false','f','0','no','n') THEN false
        ELSE NULL
    END
$$;
