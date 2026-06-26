-- Phase 3: Insert Test Data for Existing Tenant
-- Use the LGM1 tenant for testing

-- Insert test calendar
INSERT INTO calendars (tenant_id, name, region, holidays) 
VALUES (
  '870361a8-87e2-4171-95ad-0473cc93791e',
  'Test - USA Federal Holidays',
  'US',
  jsonb_build_array(
    jsonb_build_object('date', '2026-01-01', 'name', 'New Year', 'severity', 'HIGH'),
    jsonb_build_object('date', '2026-07-04', 'name', 'Independence Day', 'severity', 'HIGH'),
    jsonb_build_object('date', '2026-12-25', 'name', 'Christmas', 'severity', 'HIGH'),
    jsonb_build_object('date', '2026-11-26', 'name', 'Thanksgiving', 'severity', 'MEDIUM'),
    jsonb_build_object('date', '2026-02-16', 'name', 'Presidents Day', 'severity', 'MEDIUM')
  )
)
ON CONFLICT DO NOTHING;

INSERT INTO schedule_profiles (tenant_id, name, timezone, conflict_resolution)
VALUES ('870361a8-87e2-4171-95ad-0473cc93791e', 'test-default', 'UTC', 'UNION')
ON CONFLICT DO NOTHING;

-- Link calendar to profile
DO $$
DECLARE
  test_tenant_id UUID := '870361a8-87e2-4171-95ad-0473cc93791e';
  calendar_id UUID;
  profile_id UUID;
BEGIN
  SELECT id INTO calendar_id FROM calendars WHERE tenant_id = test_tenant_id AND name = 'Test - USA Federal Holidays' LIMIT 1;
  SELECT id INTO profile_id FROM schedule_profiles WHERE tenant_id = test_tenant_id AND name = 'test-default' LIMIT 1;
  
  IF calendar_id IS NOT NULL AND profile_id IS NOT NULL THEN
    INSERT INTO profile_calendars (profile_id, calendar_id, weight)
    VALUES (profile_id, calendar_id, 100)
    ON CONFLICT (profile_id, calendar_id) DO NOTHING;
  END IF;
END $$;

-- Insert blackouts (recurring and one-time)
DO $$
DECLARE
  test_tenant_id UUID := '870361a8-87e2-4171-95ad-0473cc93791e';
  profile_id UUID;
BEGIN
  SELECT id INTO profile_id FROM schedule_profiles WHERE tenant_id = test_tenant_id AND name = 'test-default' LIMIT 1;
  
  IF profile_id IS NOT NULL THEN
    -- One-time maintenance window
    INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity)
    VALUES (
      test_tenant_id,
      profile_id,
      'Test - Monthly Maintenance',
      'Scheduled database maintenance',
      '2026-02-20 02:00:00+00',
      '2026-02-20 04:00:00+00',
      'MAINTENANCE',
      'HIGH'
    )
    ON CONFLICT DO NOTHING;
    
    -- Recurring: Every Monday 11 PM - 1 AM UTC for 52 weeks
    INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity, recurrence_rule)
    VALUES (
      test_tenant_id,
      profile_id,
      'Test - Weekly Batch Job',
      'Recurring batch processing window',
      '2026-02-23 23:00:00+00',
      '2026-02-24 01:00:00+00',
      'PLANNED_DOWNTIME',
      'MEDIUM',
      'FREQ=WEEKLY;BYDAY=MO;COUNT=52'
    )
    ON CONFLICT DO NOTHING;
    
    -- Recurring: Every Friday 3 PM - 5 PM UTC
    INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity, recurrence_rule)
    VALUES (
      test_tenant_id,
      profile_id,
      'Test - Weekly Deployment Window',
      'Scheduled deployments every Friday afternoon',
      '2026-02-20 15:00:00+00',
      '2026-02-20 17:00:00+00',
      'PLANNED_DOWNTIME',
      'LOW',
      'FREQ=WEEKLY;BYDAY=FR;COUNT=52'
    )
    ON CONFLICT DO NOTHING;
  END IF;
END $$;

-- Summary
SELECT '✅ Test Data Inserted' as status;
SELECT 'Calendars' as category, COUNT(*) as count FROM calendars WHERE tenant_id = '870361a8-87e2-4171-95ad-0473cc93791e' AND name LIKE 'Test%'
UNION ALL SELECT 'Profiles', COUNT(*) FROM schedule_profiles WHERE tenant_id = '870361a8-87e2-4171-95ad-0473cc93791e' AND name LIKE 'test%'
UNION ALL SELECT 'Blackouts', COUNT(*) FROM blackouts WHERE tenant_id = '870361a8-87e2-4171-95ad-0473cc93791e' AND name LIKE 'Test%';
