-- Migration 032: Business Calendar Engine
-- Multi-region business calendar support with inheritance and ISDA conventions

-- =============================================================================
-- 1. ENUM TYPES
-- =============================================================================

CREATE TYPE adjustment_convention AS ENUM (
    'FOLLOWING',          -- Move to next business day
    'MODIFIED_FOLLOWING', -- Following, unless crosses month boundary (then preceding)
    'PRECEDING',          -- Move to previous business day
    'UNADJUSTED'         -- No adjustment
);

-- =============================================================================
-- 2. BUSINESS CALENDARS
-- =============================================================================

CREATE TABLE business_calendars (
    calendar_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(tenant_id), -- NULL for global calendars
    
    -- Calendar identification
    calendar_code VARCHAR(50) UNIQUE NOT NULL, -- 'NYSE', 'LSE', 'US_FEDERAL', 'TARGET'
    calendar_name TEXT NOT NULL,
    description TEXT,
    
    -- Inheritance (NYSE inherits US_FEDERAL holidays)
    parent_calendar_ids UUID[], -- Array of parent calendar IDs
    
    -- Configuration
    timezone TEXT NOT NULL DEFAULT 'America/New_York',
    weekend_days INTEGER[] DEFAULT '{0,6}', -- Sunday=0, Saturday=6 (PostgreSQL day numbering)
    
    -- Status
    active BOOLEAN DEFAULT TRUE,
    is_global BOOLEAN DEFAULT FALSE, -- System vs tenant-specific
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(user_id),
    
    CONSTRAINT valid_weekend_days CHECK (
        array_length(weekend_days, 1) IS NULL OR 
        array_length(weekend_days, 1) BETWEEN 0 AND 7
    )
);

CREATE INDEX idx_calendars_code ON business_calendars(calendar_code) WHERE active = TRUE;
CREATE INDEX idx_calendars_tenant ON business_calendars(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_calendars_active ON business_calendars(active) WHERE active = TRUE;

-- RLS for multi-tenancy
ALTER TABLE business_calendars ENABLE ROW LEVEL SECURITY;

CREATE POLICY calendars_global_read ON business_calendars
    FOR SELECT
    USING (is_global = TRUE OR tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 3. CALENDAR HOLIDAYS
-- =============================================================================

CREATE TABLE calendar_holidays (
    holiday_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID NOT NULL REFERENCES business_calendars(calendar_id) ON DELETE CASCADE,
    
    -- Holiday details
    holiday_date DATE NOT NULL,
    holiday_name TEXT NOT NULL,
    holiday_type VARCHAR(50), -- 'NATIONAL', 'BANK', 'MARKET', 'RELIGIOUS', 'OBSERVANCE'
    
    -- Half-day trading (e.g., NYSE closes at 1pm on day before July 4th)
    is_half_day BOOLEAN DEFAULT FALSE,
    half_day_close_time TIME, -- e.g., '13:00:00' for 1 PM close
    
    -- Recurrence (for annual holidays like Christmas)
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_rule TEXT, -- iCal RRULE format: 'FREQ=YEARLY;BYMONTH=12;BYMONTHDAY=25'
    
    -- Observance rules (if holiday falls on weekend, observe on Friday/Monday)
    observed_date DATE, -- Actual observed date if different from holiday_date
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (calendar_id, holiday_date),
    INDEX idx_holidays_calendar_date (calendar_id, holiday_date),
    INDEX idx_holidays_date_range (holiday_date) -- For range queries
);

ALTER TABLE calendar_holidays ENABLE ROW LEVEL SECURITY;

CREATE POLICY holidays_via_calendar ON calendar_holidays
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM business_calendars
            WHERE business_calendars.calendar_id = calendar_holidays.calendar_id
            AND (business_calendars.is_global = TRUE OR business_calendars.tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID)
        )
    );

-- =============================================================================
-- 4. SEED DATA - US FEDERAL HOLIDAYS
-- =============================================================================

-- Create US Federal calendar (base for many financial markets)
INSERT INTO business_calendars (calendar_code, calendar_name, timezone, is_global, weekend_days)
VALUES ('US_FEDERAL', 'United States Federal Holidays', 'America/New_York', TRUE, '{0,6}');

-- Seed 2025 US Federal holidays with recurrence rules
INSERT INTO calendar_holidays (calendar_id, holiday_date, holiday_name, holiday_type, is_recurring, recurrence_rule)
SELECT 
    (SELECT calendar_id FROM business_calendars WHERE calendar_code = 'US_FEDERAL'),
    date::DATE,
    name,
    'NATIONAL',
    TRUE,
    rrule
FROM (VALUES
    ('2025-01-01', 'New Year''s Day', 'FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1'),
    ('2025-01-20', 'Martin Luther King Jr. Day', 'FREQ=YEARLY;BYMONTH=1;BYDAY=3MO'),
    ('2025-02-17', 'Presidents'' Day', 'FREQ=YEARLY;BYMONTH=2;BYDAY=3MO'),
    ('2025-05-26', 'Memorial Day', 'FREQ=YEARLY;BYMONTH=5;BYDAY=-1MO'),
    ('2025-06-19', 'Juneteenth', 'FREQ=YEARLY;BYMONTH=6;BYMONTHDAY=19'),
    ('2025-07-04', 'Independence Day', 'FREQ=YEARLY;BYMONTH=7;BYMONTHDAY=4'),
    ('2025-09-01', 'Labor Day', 'FREQ=YEARLY;BYMONTH=9;BYDAY=1MO'),
    ('2025-10-13', 'Columbus Day', 'FREQ=YEARLY;BYMONTH=10;BYDAY=2MO'),
    ('2025-11-11', 'Veterans Day', 'FREQ=YEARLY;BYMONTH=11;BYMONTHDAY=11'),
    ('2025-11-27', 'Thanksgiving', 'FREQ=YEARLY;BYMONTH=11;BYDAY=4TH'),
    ('2025-12-25', 'Christmas Day', 'FREQ=YEARLY;BYMONTH=12;BYMONTHDAY=25')
) AS holidays(date, name, rrule);

-- =============================================================================
-- 5. SEED DATA - NYSE CALENDAR
-- =============================================================================

-- Create NYSE calendar (inherits US_FEDERAL + market-specific holidays)
INSERT INTO business_calendars (
    calendar_code, 
    calendar_name, 
    parent_calendar_ids, 
    timezone, 
    is_global,
    weekend_days
)
VALUES (
    'NYSE',
    'New York Stock Exchange',
    ARRAY[(SELECT calendar_id FROM business_calendars WHERE calendar_code = 'US_FEDERAL')],
    'America/New_York',
    TRUE,
    '{0,6}'
);

-- NYSE-specific holidays/closures
INSERT INTO calendar_holidays (calendar_id, holiday_date, holiday_name, holiday_type, is_recurring, recurrence_rule)
SELECT 
    (SELECT calendar_id FROM business_calendars WHERE calendar_code = 'NYSE'),
    date::DATE,
    name,
    'MARKET',
    recurring,
    rrule
FROM (VALUES
    ('2025-04-18', 'Good Friday', TRUE, 'FREQ=YEARLY'), -- Complex calculation, simplified
    ('2025-07-03', 'Day Before Independence Day (Early Close)', TRUE, NULL) -- Half-day, 1 PM close
) AS nyse_holidays(date, name, recurring, rrule);

-- Mark July 3rd as half-day
UPDATE calendar_holidays 
SET is_half_day = TRUE, half_day_close_time = '13:00:00'
WHERE holiday_name = 'Day Before Independence Day (Early Close)';

-- =============================================================================
-- 6. SEED DATA - LONDON STOCK EXCHANGE (LSE)
-- =============================================================================

INSERT INTO business_calendars (calendar_code, calendar_name, timezone, is_global, weekend_days)
VALUES ('LSE', 'London Stock Exchange', 'Europe/London', TRUE, '{0,6}');

INSERT INTO calendar_holidays (calendar_id, holiday_date, holiday_name, holiday_type, is_recurring)
SELECT 
    (SELECT calendar_id FROM business_calendars WHERE calendar_code = 'LSE'),
    date::DATE,
    name,
    'MARKET',
    TRUE
FROM (VALUES
    ('2025-01-01', 'New Year''s Day', TRUE),
    ('2025-04-18', 'Good Friday', TRUE),
    ('2025-04-21', 'Easter Monday', TRUE),
    ('2025-05-05', 'Early May Bank Holiday', TRUE),
    ('2025-05-26', 'Spring Bank Holiday', TRUE),
    ('2025-08-25', 'Summer Bank Holiday', TRUE),
    ('2025-12-25', 'Christmas Day', TRUE),
    ('2025-12-26', 'Boxing Day', TRUE)
) AS lse_holidays(date, name, recurring);

-- =============================================================================
-- 7. HELPER FUNCTIONS
-- =============================================================================

-- Check if a date is a business day (considering calendar inheritance)
CREATE OR REPLACE FUNCTION is_business_day(
    p_calendar_code VARCHAR,
    p_date DATE
) RETURNS BOOLEAN AS $$
DECLARE
    v_calendar_id UUID;
    v_weekend_days INTEGER[];
    v_day_of_week INTEGER;
    v_is_holiday BOOLEAN;
BEGIN
    -- Get calendar
    SELECT calendar_id, weekend_days INTO v_calendar_id, v_weekend_days
    FROM business_calendars
    WHERE calendar_code = p_calendar_code AND active = TRUE;
    
    IF v_calendar_id IS NULL THEN
        RAISE EXCEPTION 'Calendar % not found', p_calendar_code;
    END IF;
    
    -- Check weekend (0=Sunday, 6=Saturday in PostgreSQL EXTRACT(DOW))
    v_day_of_week := EXTRACT(DOW FROM p_date);
    IF v_day_of_week = ANY(v_weekend_days) THEN
        RETURN FALSE;
    END IF;
    
    -- Check if holiday (with inheritance via recursive CTE)
    WITH RECURSIVE calendar_hierarchy AS (
        SELECT calendar_id FROM business_calendars WHERE calendar_id = v_calendar_id
        UNION
        SELECT unnest(parent_calendar_ids) 
        FROM business_calendars bc
        JOIN calendar_hierarchy ch ON bc.calendar_id = ch.calendar_id
        WHERE bc.parent_calendar_ids IS NOT NULL
    )
    SELECT EXISTS(
        SELECT 1 FROM calendar_holidays
        WHERE calendar_id IN (SELECT calendar_id FROM calendar_hierarchy)
        AND holiday_date = p_date
    ) INTO v_is_holiday;
    
    RETURN NOT v_is_holiday;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get next business day
CREATE OR REPLACE FUNCTION next_business_day(
    p_calendar_code VARCHAR,
    p_from_date DATE
) RETURNS DATE AS $$
DECLARE
    v_current DATE := p_from_date + 1;
    v_iterations INTEGER := 0;
BEGIN
    WHILE v_iterations < 30 LOOP -- Max 30 days to prevent infinite loop
        IF is_business_day(p_calendar_code, v_current) THEN
            RETURN v_current;
        END IF;
        v_current := v_current + 1;
        v_iterations := v_iterations + 1;
    END LOOP;
    
    RAISE EXCEPTION 'Could not find next business day within 30 days';
END;
$$ LANGUAGE plpgsql STABLE;

-- Add N business days
CREATE OR REPLACE FUNCTION add_business_days(
    p_calendar_code VARCHAR,
    p_from_date DATE,
    p_days INTEGER
) RETURNS DATE AS $$
DECLARE
    v_current DATE := p_from_date;
    v_remaining INTEGER := p_days;
BEGIN
    WHILE v_remaining > 0 LOOP
        v_current := v_current + 1;
        IF is_business_day(p_calendar_code, v_current) THEN
            v_remaining := v_remaining - 1;
        END IF;
    END LOOP;
    
    RETURN v_current;
END;
$$ LANGUAGE plpgsql STABLE;

-- Adjust date according to business day convention
CREATE OR REPLACE FUNCTION adjust_date(
    p_calendar_code VARCHAR,
    p_date DATE,
    p_convention adjustment_convention
) RETURNS DATE AS $$
DECLARE
    v_adjusted DATE;
BEGIN
    IF p_convention = 'UNADJUSTED' OR is_business_day(p_calendar_code, p_date) THEN
        RETURN p_date;
    END IF;
    
    CASE p_convention
        WHEN 'FOLLOWING' THEN
            RETURN next_business_day(p_calendar_code, p_date);
            
        WHEN 'MODIFIED_FOLLOWING' THEN
            v_adjusted := next_business_day(p_calendar_code, p_date);
            -- If crosses month boundary, use preceding instead
            IF EXTRACT(MONTH FROM v_adjusted) != EXTRACT(MONTH FROM p_date) THEN
                RETURN previous_business_day(p_calendar_code, p_date);
            END IF;
            RETURN v_adjusted;
            
        WHEN 'PRECEDING' THEN
            RETURN previous_business_day(p_calendar_code, p_date);
    END CASE;
    
    RETURN p_date;
END;
$$ LANGUAGE plpgsql STABLE;

-- Get previous business day
CREATE OR REPLACE FUNCTION previous_business_day(
    p_calendar_code VARCHAR,
    p_from_date DATE
) RETURNS DATE AS $$
DECLARE
    v_current DATE := p_from_date - 1;
    v_iterations INTEGER := 0;
BEGIN
    WHILE v_iterations < 30 LOOP
        IF is_business_day(p_calendar_code, v_current) THEN
            RETURN v_current;
        END IF;
        v_current := v_current - 1;
        v_iterations := v_iterations + 1;
    END LOOP;
    
    RAISE EXCEPTION 'Could not find previous business day within 30 days';
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- 8. COMMENTS
-- =============================================================================

COMMENT ON TABLE business_calendars IS 'Multi-region business calendars with inheritance support for global financial markets';
COMMENT ON TABLE calendar_holidays IS 'Holiday definitions with recurrence rules and half-day trading support';
COMMENT ON FUNCTION is_business_day IS 'Check if a date is a business day considering weekends and holidays (with inheritance)';
COMMENT ON FUNCTION next_business_day IS 'Find the next business day after a given date';
COMMENT ON FUNCTION add_business_days IS 'Add N business days to a date';
COMMENT ON FUNCTION adjust_date IS 'Adjust a date according to ISDA business day convention';
