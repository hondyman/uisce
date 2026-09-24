-- Migration: End-to-End Report Filters with Calendar Support
-- Version: 20260824_001
-- Description: Adds filter persistence table, tenant default calendar columns, and SQL calendar functions

-- ============================================================================
-- 1. FILTER MODEL PERSISTENCE
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.report_filters (
    filter_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    report_id UUID,
    filter_model JSONB NOT NULL DEFAULT '{"groups":[],"groupCombinator":"AND"}'::jsonb,
    compiled_where TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    UNIQUE(tenant_id, report_id)
);

CREATE INDEX IF NOT EXISTS idx_report_filters_tenant ON public.report_filters(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_filters_report ON public.report_filters(report_id);

-- ============================================================================
-- 2. TENANT DEFAULT CALENDAR COLUMNS
-- ============================================================================

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenants' AND column_name = 'default_calendar_code'
    ) THEN
        ALTER TABLE public.tenants ADD COLUMN default_calendar_code VARCHAR(50);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenants' AND column_name = 'default_fiscal_year'
    ) THEN
        ALTER TABLE public.tenants ADD COLUMN default_fiscal_year INTEGER;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tenants' AND column_name = 'default_region'
    ) THEN
        ALTER TABLE public.tenants ADD COLUMN default_region VARCHAR(50);
    END IF;
END $$;

-- Seed sensible defaults for existing tenants
UPDATE public.tenants
SET default_calendar_code = COALESCE(default_calendar_code, 'US'),
    default_fiscal_year = COALESCE(default_fiscal_year, EXTRACT(YEAR FROM CURRENT_DATE)::INTEGER),
    default_region = COALESCE(default_region, 'us-east-1')
WHERE default_calendar_code IS NULL OR default_fiscal_year IS NULL OR default_region IS NULL;

-- ============================================================================
-- 3. CALENDAR SQL FUNCTIONS
-- ============================================================================

CREATE OR REPLACE FUNCTION calendar_next_business_day(p_date DATE, p_cal VARCHAR)
RETURNS DATE AS $$
DECLARE
    next_day DATE := COALESCE(p_date, CURRENT_DATE) + 1;
    max_iter INT := 30;
    iter INT := 0;
BEGIN
    WHILE iter < max_iter LOOP
        IF NOT EXISTS (
            SELECT 1 FROM public.tenant_calendar_holidays h
            JOIN public.tenant_exchange_calendars c ON h.calendar_id = c.id
            WHERE c.calendar_code = p_cal AND h.holiday_date = next_day
        )
        AND EXTRACT(DOW FROM next_day) NOT IN (0, 6) THEN
            RETURN next_day;
        END IF;
        next_day := next_day + 1;
        iter := iter + 1;
    END LOOP;
    RETURN next_day;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION calendar_previous_business_day(p_date DATE, p_cal VARCHAR)
RETURNS DATE AS $$
DECLARE
    prev_day DATE := COALESCE(p_date, CURRENT_DATE) - 1;
    max_iter INT := 30;
    iter INT := 0;
BEGIN
    WHILE iter < max_iter LOOP
        IF NOT EXISTS (
            SELECT 1 FROM public.tenant_calendar_holidays h
            JOIN public.tenant_exchange_calendars c ON h.calendar_id = c.id
            WHERE c.calendar_code = p_cal AND h.holiday_date = prev_day
        )
        AND EXTRACT(DOW FROM prev_day) NOT IN (0, 6) THEN
            RETURN prev_day;
        END IF;
        prev_day := prev_day - 1;
        iter := iter + 1;
    END LOOP;
    RETURN prev_day;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION calendar_add_business_days(p_date DATE, p_cal VARCHAR, p_days INT)
RETURNS DATE AS $$
DECLARE
    result_date DATE := COALESCE(p_date, CURRENT_DATE);
    direction INT := SIGN(p_days);
    remaining INT := ABS(p_days);
BEGIN
    WHILE remaining > 0 LOOP
        result_date := result_date + direction;
        IF NOT EXISTS (
            SELECT 1 FROM public.tenant_calendar_holidays h
            JOIN public.tenant_exchange_calendars c ON h.calendar_id = c.id
            WHERE c.calendar_code = p_cal AND h.holiday_date = result_date
        )
        AND EXTRACT(DOW FROM result_date) NOT IN (0, 6) THEN
            remaining := remaining - 1;
        END IF;
    END LOOP;
    RETURN result_date;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION calendar_is_business_day(p_date DATE, p_cal VARCHAR)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXTRACT(DOW FROM p_date) NOT IN (0, 6)
        AND NOT EXISTS (
            SELECT 1 FROM public.tenant_calendar_holidays h
            JOIN public.tenant_exchange_calendars c ON h.calendar_id = c.id
            WHERE c.calendar_code = p_cal AND h.holiday_date = p_date
        );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION calendar_is_holiday(p_date DATE, p_cal VARCHAR)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM public.tenant_calendar_holidays h
        JOIN public.tenant_exchange_calendars c ON h.calendar_id = c.id
        WHERE c.calendar_code = p_cal AND h.holiday_date = p_date
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;
