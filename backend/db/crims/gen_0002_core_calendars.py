#!/usr/bin/env python3
"""Generates 0002_seed_core_calendars.sql: the gold-copy core calendars
(NYSE, LSE, TARGET2) for the years below, from each market's published
holiday rules.

Tenants inherit these calendars (decision D5): a tenant never copies them,
it adds its own layer through mdm.calendar_hierarchy (ADDITIVE by default).

Rerun with a wider YEARS range to extend coverage; the SQL is idempotent.
Days are derived from rules, so they are seeded is_official = false until
reconciled against the exchange's own published calendar. One-off closures
(national days of mourning, etc.) cannot be derived and are listed in
SPECIAL.
"""
import datetime as dt
import pathlib

YEARS = range(2024, 2031)
GOLD = "99e99e99-99e9-49e9-89e9-99e99e99e999"
D = dt.date


def easter(y):  # anonymous Gregorian algorithm
    a, b, c = y % 19, y // 100, y % 100
    d, e = b // 4, b % 4
    f = (b + 8) // 25
    g = (b - f + 1) // 3
    h = (19 * a + b - d - g + 15) % 30
    i, k = c // 4, c % 4
    l = (32 + 2 * e + 2 * i - h - k) % 7
    m = (a + 11 * h + 22 * l) // 451
    month = (h + l - 7 * m + 114) // 31
    day = (h + l - 7 * m + 114) % 31 + 1
    return D(y, month, day)


def nth_weekday(y, m, wd, n):  # wd: 0=Mon; n=1.. or -1 for last
    if n > 0:
        d = D(y, m, 1)
        d += dt.timedelta((wd - d.weekday()) % 7)
        return d + dt.timedelta(7 * (n - 1))
    d = (D(y, m + 1, 1) if m < 12 else D(y + 1, 1, 1)) - dt.timedelta(1)
    return d - dt.timedelta((d.weekday() - wd) % 7)


def us_observed(d):  # Saturday -> Friday, Sunday -> Monday
    return d - dt.timedelta(1) if d.weekday() == 5 else d + dt.timedelta(1) if d.weekday() == 6 else d


def nyse(y):
    closed, half = [], []
    ny = D(y, 1, 1)
    # NYSE does not observe New Year's Day on the prior Friday (Dec 31).
    if ny.weekday() != 5:
        closed.append((us_observed(ny), "NEW_YEARS_DAY", "New Year's Day"))
    closed += [
        (nth_weekday(y, 1, 0, 3), "MLK_DAY", "Martin Luther King Jr. Day"),
        (nth_weekday(y, 2, 0, 3), "WASHINGTONS_BIRTHDAY", "Washington's Birthday"),
        (easter(y) - dt.timedelta(2), "GOOD_FRIDAY", "Good Friday"),
        (nth_weekday(y, 5, 0, -1), "MEMORIAL_DAY", "Memorial Day"),
        (us_observed(D(y, 6, 19)), "JUNETEENTH", "Juneteenth National Independence Day"),
        (us_observed(D(y, 7, 4)), "INDEPENDENCE_DAY", "Independence Day"),
        (nth_weekday(y, 9, 0, 1), "LABOR_DAY", "Labor Day"),
        (nth_weekday(y, 11, 3, 4), "THANKSGIVING", "Thanksgiving Day"),
        (us_observed(D(y, 12, 25)), "CHRISTMAS", "Christmas Day"),
    ]
    closed_days = {c[0] for c in closed}
    jul3, xmas_eve = D(y, 7, 3), D(y, 12, 24)
    if jul3.weekday() < 5 and jul3 not in closed_days:
        half.append((jul3, "INDEPENDENCE_EVE", "Day before Independence Day", "13:00"))
    half.append((nth_weekday(y, 11, 3, 4) + dt.timedelta(1), "DAY_AFTER_THANKSGIVING", "Day after Thanksgiving", "13:00"))
    if xmas_eve.weekday() < 5 and xmas_eve not in closed_days:
        half.append((xmas_eve, "CHRISTMAS_EVE", "Christmas Eve", "13:00"))
    return closed, half


def uk_bank(y):
    """England & Wales bank holidays with the UK substitution rules."""
    out = []
    ny = D(y, 1, 1)
    out.append((ny + dt.timedelta({5: 2, 6: 1}.get(ny.weekday(), 0)), "NEW_YEARS_DAY", "New Year's Day"))
    e = easter(y)
    out += [
        (e - dt.timedelta(2), "GOOD_FRIDAY", "Good Friday"),
        (e + dt.timedelta(1), "EASTER_MONDAY", "Easter Monday"),
        (nth_weekday(y, 5, 0, 1), "EARLY_MAY", "Early May bank holiday"),
        (nth_weekday(y, 5, 0, -1), "SPRING_BANK", "Spring bank holiday"),
        (nth_weekday(y, 8, 0, -1), "SUMMER_BANK", "Summer bank holiday"),
    ]
    xmas, boxing = D(y, 12, 25), D(y, 12, 26)
    if xmas.weekday() == 5:    # Sat: Christmas -> Mon 27, Boxing -> Tue 28
        xmas, boxing = D(y, 12, 27), D(y, 12, 28)
    elif xmas.weekday() == 6:  # Sun: Boxing Mon 26, Christmas -> Tue 27
        xmas = D(y, 12, 27)
    elif boxing.weekday() == 5:  # Fri Christmas: Boxing Sat -> Mon 28
        boxing = D(y, 12, 28)
    out += [(xmas, "CHRISTMAS", "Christmas Day"), (boxing, "BOXING_DAY", "Boxing Day")]
    return out


def lse(y):
    closed = uk_bank(y)
    closed_days = {c[0] for c in closed}
    half = []
    for d, code, name in ((D(y, 12, 24), "CHRISTMAS_EVE", "Christmas Eve"), (D(y, 12, 31), "NEW_YEARS_EVE", "New Year's Eve")):
        if d.weekday() < 5 and d not in closed_days:
            half.append((d, code, name, "12:30"))
    return closed, half


def target2(y):
    e = easter(y)
    return [
        (D(y, 1, 1), "NEW_YEARS_DAY", "New Year's Day"),
        (e - dt.timedelta(2), "GOOD_FRIDAY", "Good Friday"),
        (e + dt.timedelta(1), "EASTER_MONDAY", "Easter Monday"),
        (D(y, 5, 1), "LABOUR_DAY", "Labour Day"),
        (D(y, 12, 25), "CHRISTMAS", "Christmas Day"),
        (D(y, 12, 26), "BOXING_DAY", "Christmas Holiday"),
    ], []


# One-off closures that no rule produces.
SPECIAL = [
    ("XNYS", D(2025, 1, 9), "NATIONAL_DAY_OF_MOURNING", "National Day of Mourning for President Jimmy Carter"),
]

CALENDARS = {"XNYS": nyse, "XLON": lse, "TARGET2": target2}


def q(s):
    return "'" + s.replace("'", "''") + "'"


rows = []
for cd, fn in CALENDARS.items():
    for y in YEARS:
        closed, half = fn(y)
        rows += [f"({q(cd)}, DATE {q(d.isoformat())}, {q(code)}, {q(name)}, 'CLOSED', NULL)" for d, code, name in closed]
        rows += [f"({q(cd)}, DATE {q(d.isoformat())}, {q(code)}, {q(name)}, 'HALF', TIME {q(t)})" for d, code, name, t in half]
rows += [f"({q(cd)}, DATE {q(d.isoformat())}, {q(code)}, {q(name)}, 'CLOSED', NULL)" for cd, d, code, name in SPECIAL]

first, last = f"{YEARS[0]}-01-01", f"{YEARS[-1]}-12-31"
sql = f"""-- Core calendars owned by the gold-copy tenant (decision D5): NYSE (XNYS),
-- London Stock Exchange (XLON) and TARGET2, {first} .. {last}.
-- GENERATED by gen_0002_core_calendars.py - edit the generator, not this file.
--
-- Tenants inherit these; a tenant's own items go in a child calendar linked
-- through mdm.calendar_hierarchy (ADDITIVE by default; INHERIT_EXCEPT and
-- OVERRIDE through an approved mdm.calendar_change_request).
--
-- Holidays are derived from each market's published rules, plus listed
-- one-off closures, so days are seeded is_official = false until reconciled
-- with the exchange's own calendar. Idempotent: reruns add only what is
-- missing and never overwrite an edited day.
BEGIN;

CREATE TEMP TABLE core_cal_def (
    calendar_cd text, name text, short_name text, type_cd text, tz text, country_cd text,
    mic text, currency_cd text, open_time time, close_time time, cutoff_time time
) ON COMMIT DROP;
INSERT INTO core_cal_def VALUES
    ('XNYS', 'New York Stock Exchange', 'NYSE', 'EXCHANGE_TRADING', 'America/New_York', 'US', 'XNYS', 'USD', '09:30', '16:00', NULL),
    ('XLON', 'London Stock Exchange', 'LSE', 'EXCHANGE_TRADING', 'Europe/London', 'GB', 'XLON', 'GBP', '08:00', '16:30', NULL),
    ('TARGET2', 'TARGET2 (T2) euro payments', 'TARGET2', 'TARGET2', 'Europe/Frankfurt', NULL, NULL, 'EUR', '07:00', '18:00', '18:00');

INSERT INTO mdm.calendar_master (tenant_id, calendar_cd, name, short_name, calendar_type_id, primary_time_zone_id,
                                 country_cd, mic, currency_cd, effective_from, is_golden_record, custom_attributes)
SELECT '{GOLD}'::uuid, d.calendar_cd, d.name, d.short_name,
       (SELECT id FROM mdm.calendar_type WHERE calendar_type_cd = d.type_cd ORDER BY (tenant_id = '{GOLD}'::uuid) DESC LIMIT 1),
       (SELECT id FROM mdm.time_zone WHERE tz_name = d.tz ORDER BY (tenant_id = '{GOLD}'::uuid) DESC LIMIT 1),
       d.country_cd, d.mic, d.currency_cd, DATE '{first}', true,
       jsonb_build_object('seed', 'core-calendars-v1', 'coverage', '{YEARS[0]}-{YEARS[-1]}', 'rule_derived', true, 'core', true)
FROM core_cal_def d
ON CONFLICT (tenant_id, calendar_cd) DO NOTHING;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM mdm.calendar_master WHERE tenant_id = '{GOLD}'::uuid
               AND calendar_cd IN ('XNYS', 'XLON', 'TARGET2') AND (calendar_type_id IS NULL OR primary_time_zone_id IS NULL)) THEN
        RAISE EXCEPTION 'core calendar seed: a calendar type or time zone reference row is missing';
    END IF;
END $$;

INSERT INTO mdm.calendar_identifier (tenant_id, calendar_id, id_type, id_value, is_primary, effective_from, source)
SELECT m.tenant_id, m.id, 'MIC', m.mic, true, DATE '{first}', 'ISO 10383'
FROM mdm.calendar_master m
WHERE m.tenant_id = '{GOLD}'::uuid AND m.mic IS NOT NULL AND m.calendar_cd IN ('XNYS', 'XLON')
  AND NOT EXISTS (SELECT 1 FROM mdm.calendar_identifier i WHERE i.calendar_id = m.id AND i.id_type = 'MIC');

CREATE TEMP TABLE core_cal_holiday (calendar_cd text, d date, code text, name text, kind text, close_time time) ON COMMIT DROP;
INSERT INTO core_cal_holiday VALUES
{",\n".join(rows)};

INSERT INTO mdm.calendar_day (
    tenant_id, calendar_id, calendar_date, day_of_week, is_weekend,
    is_business_day, is_trading_day, is_settlement_day, is_payment_day, is_dealing_day, is_valuation_day,
    is_half_day, session_open_time, session_close_time, settlement_cutoff_time,
    is_month_end, is_quarter_end, is_year_end,
    is_last_business_day_of_month, is_first_business_day_of_month,
    business_day_of_month, business_day_of_quarter, business_day_of_year,
    additional_holidays, is_materialized, materialization_source, is_official, custom_attributes)
SELECT tenant_id, calendar_id, dt, dow, weekend,
       open, open, open, open, open, open,
       half, CASE WHEN open THEN open_time END, CASE WHEN open THEN COALESCE(half_close, close_time) END,
       CASE WHEN open THEN cutoff_time END,
       dt = (date_trunc('month', dt) + interval '1 month - 1 day')::date,
       dt = (date_trunc('quarter', dt) + interval '3 months - 1 day')::date,
       extract(month FROM dt) = 12 AND extract(day FROM dt) = 31,
       open AND dt = max(dt) FILTER (WHERE open) OVER (PARTITION BY calendar_id, date_trunc('month', dt)),
       open AND dt = min(dt) FILTER (WHERE open) OVER (PARTITION BY calendar_id, date_trunc('month', dt)),
       CASE WHEN open THEN count(*) FILTER (WHERE open) OVER (PARTITION BY calendar_id, date_trunc('month', dt) ORDER BY dt) END,
       CASE WHEN open THEN count(*) FILTER (WHERE open) OVER (PARTITION BY calendar_id, date_trunc('quarter', dt) ORDER BY dt) END,
       CASE WHEN open THEN count(*) FILTER (WHERE open) OVER (PARTITION BY calendar_id, date_trunc('year', dt) ORDER BY dt) END,
       holidays, true, 'core-calendars-v1', false,
       jsonb_build_object('rule_derived', true)
FROM (
    SELECT m.tenant_id, m.id AS calendar_id, g.d::date AS dt,
           extract(isodow FROM g.d)::int AS dow,
           extract(isodow FROM g.d) IN (6, 7) AS weekend,
           NOT (extract(isodow FROM g.d) IN (6, 7))
             AND NOT EXISTS (SELECT 1 FROM core_cal_holiday h WHERE h.calendar_cd = m.calendar_cd AND h.d = g.d::date AND h.kind = 'CLOSED') AS open,
           EXISTS (SELECT 1 FROM core_cal_holiday h WHERE h.calendar_cd = m.calendar_cd AND h.d = g.d::date AND h.kind = 'HALF') AS half,
           (SELECT min(h.close_time) FROM core_cal_holiday h WHERE h.calendar_cd = m.calendar_cd AND h.d = g.d::date AND h.kind = 'HALF') AS half_close,
           (SELECT jsonb_agg(jsonb_build_object('code', h.code, 'name', h.name, 'kind', h.kind) ORDER BY h.code)
              FROM core_cal_holiday h WHERE h.calendar_cd = m.calendar_cd AND h.d = g.d::date) AS holidays,
           c.open_time, c.close_time, c.cutoff_time
    FROM mdm.calendar_master m
    JOIN core_cal_def c ON c.calendar_cd = m.calendar_cd
    CROSS JOIN generate_series(DATE '{first}', DATE '{last}', interval '1 day') AS g(d)
    WHERE m.tenant_id = '{GOLD}'::uuid
) days
ON CONFLICT (tenant_id, calendar_id, calendar_date) DO NOTHING;

COMMIT;
"""
out = pathlib.Path(__file__).with_name("0002_seed_core_calendars.sql")
out.write_text(sql)
print(f"wrote {out.name}: {len(rows)} holiday/half-day rows")
