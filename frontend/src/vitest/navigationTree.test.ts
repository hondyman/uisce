import { describe, it, expect } from 'vitest';
import { mapDesignerTreeToCategories, nodeTarget } from '@/components/navigationTree';
import type { NavigationMenuNode } from '@/api/navigationMenu';

const n = (o: Partial<NavigationMenuNode> & { nodeKey: string }): NavigationMenuNode => ({
  id: o.nodeKey, label: o.nodeKey, displayOrder: 0, requiredEntitlement: 'BASE_USER', ...o,
});

describe('mapDesignerTreeToCategories', () => {
  const tree = [
    n({ nodeKey: 'b', label: 'B', displayOrder: 2, children: [n({ nodeKey: 'b1', targetRoute: '/b/one' })] }),
    n({
      nodeKey: 'a', label: 'A', displayOrder: 1, requiredCapability: 'menu:a',
      children: [
        n({ nodeKey: 'g', label: 'Group', children: [
          n({ nodeKey: 'p', label: 'Page', targetPageKey: 'my-page' }),
          n({ nodeKey: 'h', label: 'Hidden', targetRoute: '/h', hidden: true }),
          n({ nodeKey: 'sub', label: 'Sub', children: [n({ nodeKey: 'deep', targetRoute: '/deep' })] }),
        ] }),
        n({ nodeKey: 'empty', label: 'Empty', children: [] }),
      ],
    }),
    n({ nodeKey: 'dead', label: 'Dead', children: [n({ nodeKey: 'x', label: 'NoTarget' })] }),
  ];

  it('orders roots, prunes empty/targetless branches, hides hidden nodes, flattens depth', () => {
    const cats = mapDesignerTreeToCategories(tree);
    expect(cats.map((c) => c.key)).toEqual(['a', 'b']);
    expect(cats[0].requiredCapability).toBe('menu:a');
    expect(cats[0].menus[0].items.map((i) => i.path)).toEqual(['/p/my-page', '/deep']);
    expect(cats[0].defaultPath).toBe('/p/my-page');
  });

  it('puts leaves directly under a root into a group named after it', () => {
    const cats = mapDesignerTreeToCategories(tree);
    expect(cats[1].menus[0].label).toBe('B');
    expect(cats[1].menus[0].items[0].path).toBe('/b/one');
  });

  it('prefers an alias without params, else /p/:slug', () => {
    const node = n({ nodeKey: 'q', targetPageKey: 'pg' });
    expect(nodeTarget(node, [{ path: '/x/:id', pageKey: 'pg' }, { path: '/x', pageKey: 'pg' }])).toBe('/x');
    expect(nodeTarget(node, [{ path: '/x/:id', pageKey: 'pg' }])).toBe('/p/pg');
  });
});
