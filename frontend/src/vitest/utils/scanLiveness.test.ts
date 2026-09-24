import { describe, expect, it } from 'vitest';
import { SCAN_STALL_AFTER_SECONDS, formatDuration, scanLiveness } from '../../features/tenants/components/scanLiveness';

describe('formatDuration', () => {
  it('formats seconds, minutes and hours', () => {
    expect(formatDuration(0)).toBe('0s');
    expect(formatDuration(59.9)).toBe('59s');
    expect(formatDuration(125)).toBe('2m 05s');
    expect(formatDuration(3720)).toBe('1h 02m');
  });
  it('never goes negative', () => {
    expect(formatDuration(-5)).toBe('0s');
  });
});

describe('scanLiveness', () => {
  const t0 = 1_000_000;
  it('reports elapsed time and time since the last update', () => {
    const l = scanLiveness(t0, t0 + 62_000, t0 + 65_000);
    expect(l.elapsed).toBe('1m 05s');
    expect(l.sinceLastUpdate).toBe('3s');
    expect(l.stalled).toBe(false);
  });
  it('is stalled only after the silence threshold, however long the scan has run', () => {
    const justUnder = scanLiveness(t0, t0, t0 + (SCAN_STALL_AFTER_SECONDS - 1) * 1000);
    const atLimit = scanLiveness(t0, t0, t0 + SCAN_STALL_AFTER_SECONDS * 1000);
    expect(justUnder.stalled).toBe(false);
    expect(atLimit.stalled).toBe(true);
    expect(scanLiveness(t0, t0 + 590_000, t0 + 600_000).stalled).toBe(false); // long scan, but heard from 10s ago
  });
});
