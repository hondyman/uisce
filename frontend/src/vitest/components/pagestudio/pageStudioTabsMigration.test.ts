import { describe, it, expect } from 'vitest';
import { migrateLayoutSpecToV3, type PageStudioLayoutSpecV2, type CanvasWidget } from '../../../components/pagestudio/pageStudioTypes';

describe('migrateLayoutSpecToV3', () => {
  it('wraps a v2 spec into a single "Details" tab', () => {
    const canvas: CanvasWidget[] = [{ id: 'f1', type: 'field', fieldKey: 'a', label: 'A', dataType: 'string', controlType: 'text', subtypeKey: null }];
    const v2: PageStudioLayoutSpecV2 = {
      version: 2,
      pageKey: 'p1',
      title: 'Page 1',
      rootBoKey: 'account',
      rootBoId: 'bo-1',
      selectedSubtypeKeys: [],
      canvas,
    };
    const v3 = migrateLayoutSpecToV3(v2);
    expect(v3.version).toBe(3);
    expect(v3.tabs).toHaveLength(1);
    expect(v3.tabs[0].title).toBe('Details');
    expect(v3.tabs[0].canvas).toEqual(canvas);
    expect(v3.pageKey).toBe('p1');
  });

  it('passes an already-v3 spec through unchanged', () => {
    const v3in = {
      version: 3 as const,
      pageKey: 'p2',
      title: 'Page 2',
      rootBoKey: 'account',
      rootBoId: 'bo-1',
      selectedSubtypeKeys: [],
      tabs: [{ id: 't1', title: 'Tab 1', canvas: [] }],
    };
    const v3out = migrateLayoutSpecToV3(v3in);
    expect(v3out).toBe(v3in);
  });

  it('throws for an unrecognized version', () => {
    expect(() => migrateLayoutSpecToV3({ version: 1 } as any)).toThrow();
    expect(() => migrateLayoutSpecToV3({} as any)).toThrow();
  });
});
