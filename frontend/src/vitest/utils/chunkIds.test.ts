import { describe, it, expect } from 'vitest';
import { chunkIds } from '../../utils/chunkIds';

describe('chunkIds', () => {
  it('returns no chunks for empty input', () => {
    expect(chunkIds([], 10)).toEqual([]);
  });

  it('keeps a single chunk when the list fits', () => {
    expect(chunkIds([1, 2, 3], 10)).toEqual([[1, 2, 3]]);
  });

  it('splits at the size boundary, not over it', () => {
    expect(chunkIds([1, 2, 3, 4], 2)).toEqual([[1, 2], [3, 4]]);
  });

  it('does not exceed the size in the trailing chunk', () => {
    expect(chunkIds([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]]);
  });

  it('emits exactly one chunk per full batch', () => {
    const ids = Array.from({ length: 5361 }, (_, i) => `id-${i}`);
    const chunks = chunkIds(ids, 2000);
    expect(chunks).toHaveLength(3);
    expect(chunks.every(c => c.length <= 2000)).toBe(true);
    expect(chunks[0]).toHaveLength(2000);
    expect(chunks[2]).toHaveLength(1361);
  });

  it('preserves every id in order', () => {
    const ids = Array.from({ length: 250 }, (_, i) => i);
    expect(chunkIds(ids, 30).flat()).toEqual(ids);
  });

  it('rejects a non-positive size rather than looping forever', () => {
    expect(() => chunkIds([1, 2, 3], 0)).toThrow();
  });
});
