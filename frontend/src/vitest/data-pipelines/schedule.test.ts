import { describe, expect, it } from 'vitest';
import { cronFor, presetOf } from '../../features/data-pipelines/ScheduleDialog';

describe('schedule presets', () => {
  it('builds cron from presets', () => {
    expect(cronFor('weekdays', '06:30', '')).toBe('30 6 * * 1-5');
    expect(cronFor('daily', '22:05', '')).toBe('5 22 * * *');
    expect(cronFor('hourly', '00:15', '')).toBe('15 * * * *');
    expect(cronFor('custom', '', ' 0 6 1 * * ')).toBe('0 6 1 * *');
  });
  it('round-trips presets and falls back to custom', () => {
    for (const [p, t] of [['weekdays', '06:30'], ['daily', '22:05'], ['hourly', '00:15']] as const) {
      expect(presetOf(cronFor(p, t, ''))).toEqual({ preset: p, time: t });
    }
    expect(presetOf('0 6 1 * *').preset).toBe('custom');
  });
});
