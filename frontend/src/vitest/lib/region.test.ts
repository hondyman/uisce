import { describe, it, expect, beforeEach, vi } from 'vitest';
import { clearSelectedRegion, getSelectedRegion, setSelectedRegion, subscribeSelectedRegion } from '../../lib/region';

describe('selected region (no default)', () => {
  beforeEach(() => localStorage.clear());

  it('is empty when nothing is selected: it is never guessed', () => {
    expect(getSelectedRegion()).toBe('');
  });

  it('returns what was selected', () => {
    setSelectedRegion('eu-central-1');
    expect(getSelectedRegion()).toBe('eu-central-1');
  });

  it('clearing forgets the selection instead of reverting to a default', () => {
    setSelectedRegion('eu-central-1');
    clearSelectedRegion();
    expect(getSelectedRegion()).toBe('');
  });

  it('selecting a blank region clears it', () => {
    setSelectedRegion('eu-central-1');
    setSelectedRegion('   ');
    expect(getSelectedRegion()).toBe('');
  });

  it('notifies subscribers on change and stops after unsubscribe', () => {
    const cb = vi.fn();
    const off = subscribeSelectedRegion(cb);
    setSelectedRegion('us-east-1');
    clearSelectedRegion();
    expect(cb).toHaveBeenCalledTimes(2);
    off();
    setSelectedRegion('us-west-2');
    expect(cb).toHaveBeenCalledTimes(2);
  });
});
