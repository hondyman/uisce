/**
 * filterNavigationByCapabilities / mapDesignerTreeToCategories unit tests
 *
 * Covers the two authorization gates on the main navigation:
 *   - requiredCapability  : a menu:* action key from GET /api/capabilities
 *   - requiredEntitlement : a target_profile_key from a Menu Designer node,
 *                           compared against the server-resolved profile
 *
 * These are deliberately separate mechanisms and must not accept each other's
 * vocabulary. The security property under test throughout is fail-closed: when
 * the backend has not told us something, a restricted node stays hidden.
 *
 * Pure functions only — no network, no React rendering.
 */

import { describe, it, expect } from 'vitest';
import {
  filterNavigationByCapabilities,
  flattenDesignerPages,
  type CategoryConfig,
} from '@/components/MainNavigation';
import type { NavigationMenuNode } from '@/api/navigationMenu';

const ORG_ACCESS_VISIBLE = { isVisible: true, canRead: true, canWrite: true };

function category(overrides: Partial<CategoryConfig> = {}): CategoryConfig {
  return {
    label: 'Test',
    key: 'catalog',
    icon: null,
    defaultPath: '/x',
    color: { primary: '#000', light: '#111', dark: '#222', background: '#333' },
    menus: [
      {
        label: 'Group',
        icon: null,
        items: [{ label: 'Item', path: '/x/item', icon: null }],
      },
    ],
    ...overrides,
  };
}

describe('filterNavigationByCapabilities — capability gate', () => {
  it('returns every category for a platform operator, even with no capabilities', () => {
    const cats = [
      category({ label: 'A', requiredCapability: 'menu:platform' }),
      category({ label: 'B', requiredCapability: 'menu:system' }),
    ];
    const out = filterNavigationByCapabilities(cats, undefined, true, ORG_ACCESS_VISIBLE);
    expect(out.map((c) => c.label)).toEqual(['A', 'B']);
  });

  it('drops a category whose capability is explicitly false', () => {
    const cats = [
      category({ label: 'A', requiredCapability: 'menu:platform' }),
      category({ label: 'B' }),
    ];
    const out = filterNavigationByCapabilities(cats, { 'menu:platform': false }, false, ORG_ACCESS_VISIBLE);
    expect(out.map((c) => c.label)).toEqual(['B']);
  });

  it('keeps a category whose capability is true', () => {
    const cats = [category({ label: 'A', requiredCapability: 'menu:platform' })];
    const out = filterNavigationByCapabilities(cats, { 'menu:platform': true }, false, ORG_ACCESS_VISIBLE);
    expect(out.map((c) => c.label)).toEqual(['A']);
  });

  // While the feed is still loading we must not flash unauthorized admin menus.
  it('hides capability-gated categories while capabilities are undefined', () => {
    const cats = [
      category({ label: 'Gated', requiredCapability: 'menu:platform' }),
      category({ label: 'Open' }),
    ];
    const out = filterNavigationByCapabilities(cats, undefined, false, ORG_ACCESS_VISIBLE);
    expect(out.map((c) => c.label)).toEqual(['Open']);
  });

  it('removes a category once its last item is filtered out', () => {
    const cat = category({
      label: 'OnlyGated',
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [{ label: 'Secret', path: '/x', icon: null, requiredCapability: 'menu:system' }],
        },
      ],
    });
    const out = filterNavigationByCapabilities([cat], { 'menu:system': false }, false, ORG_ACCESS_VISIBLE);
    expect(out).toHaveLength(0);
  });
});

describe('filterNavigationByCapabilities — entitlement (profile) gate', () => {
  it('shows a BASE_USER node to everyone', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [{ label: 'Everyone', path: '/x', icon: null, requiredEntitlement: 'BASE_USER' }],
        },
      ],
    });
    for (const profile of [undefined, 'BASE_USER', 'PLATFORM_OPERATOR']) {
      const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, profile);
      expect(out).toHaveLength(1);
    }
  });

  it('shows an ungated node when requiredEntitlement is empty', () => {
    const cat = category();
    const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'BASE_USER');
    expect(out).toHaveLength(1);
  });

  it('hides a PLATFORM_OPERATOR node from a BASE_USER caller', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [
            { label: 'Open', path: '/a', icon: null },
            { label: 'AdminOnly', path: '/b', icon: null, requiredEntitlement: 'PLATFORM_OPERATOR' },
          ],
        },
      ],
    });
    const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'BASE_USER');
    expect(out[0].menus[0].items.map((i) => i.label)).toEqual(['Open']);
  });

  it('shows a PLATFORM_OPERATOR node to a PLATFORM_OPERATOR caller', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [{ label: 'AdminOnly', path: '/b', icon: null, requiredEntitlement: 'PLATFORM_OPERATOR' }],
        },
      ],
    });
    const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'PLATFORM_OPERATOR');
    expect(out[0].menus[0].items.map((i) => i.label)).toEqual(['AdminOnly']);
  });

  // Fail closed: we cannot prove the caller holds the profile yet.
  it('hides restricted nodes while the profile is still unknown', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [
            { label: 'Open', path: '/a', icon: null },
            { label: 'AdminOnly', path: '/b', icon: null, requiredEntitlement: 'PLATFORM_OPERATOR' },
          ],
        },
      ],
    });
    const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, undefined);
    expect(out[0].menus[0].items.map((i) => i.label)).toEqual(['Open']);
  });

  // A typo in the Menu Designer must not authorize everyone.
  it('hides a node gated by a profile key the caller does not hold', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [{ label: 'Typo', path: '/b', icon: null, requiredEntitlement: 'PLATFOM_OPERATR' }],
        },
      ],
    });
    const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'PLATFORM_OPERATOR');
    expect(out).toHaveLength(0);
  });

  it('removes the category when its only item is profile-gated away', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [{ label: 'AdminOnly', path: '/b', icon: null, requiredEntitlement: 'PLATFORM_OPERATOR' }],
        },
      ],
    });
    const out = filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'BASE_USER');
    expect(out).toHaveLength(0);
  });

  // Both gates apply independently; either one denying hides the item.
  it('hides an item gated by both when only the capability denies', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [
            {
              label: 'Both',
              path: '/b',
              icon: null,
              requiredCapability: 'menu:system',
              requiredEntitlement: 'PLATFORM_OPERATOR',
            },
          ],
        },
      ],
    });
    const out = filterNavigationByCapabilities(
      [cat],
      { 'menu:system': false },
      false,
      ORG_ACCESS_VISIBLE,
      'PLATFORM_OPERATOR'
    );
    expect(out).toHaveLength(0);
  });

  it('hides an item gated by both when only the profile denies', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [
            {
              label: 'Both',
              path: '/b',
              icon: null,
              requiredCapability: 'menu:system',
              requiredEntitlement: 'PLATFORM_OPERATOR',
            },
          ],
        },
      ],
    });
    const out = filterNavigationByCapabilities(
      [cat],
      { 'menu:system': true },
      false,
      ORG_ACCESS_VISIBLE,
      'BASE_USER'
    );
    expect(out).toHaveLength(0);
  });

  it('shows an item gated by both when both allow', () => {
    const cat = category({
      menus: [
        {
          label: 'Group',
          icon: null,
          items: [
            {
              label: 'Both',
              path: '/b',
              icon: null,
              requiredCapability: 'menu:system',
              requiredEntitlement: 'PLATFORM_OPERATOR',
            },
          ],
        },
      ],
    });
    const out = filterNavigationByCapabilities(
      [cat],
      { 'menu:system': true },
      false,
      ORG_ACCESS_VISIBLE,
      'PLATFORM_OPERATOR'
    );
    expect(out[0].menus[0].items.map((i) => i.label)).toEqual(['Both']);
  });
});

describe('flattenDesignerPages', () => {
  const node = (over: Partial<NavigationMenuNode>): NavigationMenuNode => ({
    id: 'id',
    nodeKey: 'key',
    label: 'Node',
    displayOrder: 0,
    requiredEntitlement: 'BASE_USER',
    ...over,
  });

  // Regression guard for the real bug: the navigation_menu_nodes tree is a
  // Page Studio page tree, not a model of the platform nav. Treating its roots
  // as top-level categories replaced the whole platform menu (Platform ->
  // Organization -> Tenants) with two folders.
  it('returns an empty list for an empty tree', () => {
    expect(flattenDesignerPages([])).toEqual([]);
  });

  it('skips grouping folders and surfaces only their page leaves', () => {
    const tree = [
      node({
        id: 'c1',
        label: 'Master Data',
        children: [
          node({ id: 'i1', label: 'Vendor registry', targetPageKey: 'mdm-vendors' }),
          node({ id: 'i2', label: 'Match rules', targetPageKey: 'mdm-match-rules' }),
        ],
      }),
    ];
    const out = flattenDesignerPages(tree);
    expect(out.map((i) => i.path)).toEqual(['/pages/mdm-vendors', '/pages/mdm-match-rules']);
    expect(out.map((i) => i.label)).toEqual(['Master Data › Vendor registry', 'Master Data › Match rules']);
  });

  it('never turns a folder into a clickable item', () => {
    const tree = [node({ id: 'c1', label: 'Orders', children: [] })];
    expect(flattenDesignerPages(tree)).toEqual([]);
  });

  it('binds a page to /pages/<slug>', () => {
    const tree = [
      node({ id: 'c1', label: 'Orders', children: [node({ id: 'i1', label: 'Order List', targetPageKey: 'order-list-tl48' })] }),
    ];
    expect(flattenDesignerPages(tree)[0].path).toBe('/pages/order-list-tl48');
  });

  it('carries requiredEntitlement through so the profile gate can apply', () => {
    const tree = [
      node({
        id: 'c1',
        label: 'Admin',
        children: [
          node({ id: 'i1', label: 'AdminPage', targetPageKey: 'admin-page', requiredEntitlement: 'PLATFORM_OPERATOR' }),
        ],
      }),
    ];
    expect(flattenDesignerPages(tree)[0].requiredEntitlement).toBe('PLATFORM_OPERATOR');
  });

  it('treats a blank requiredEntitlement as ungated', () => {
    const tree = [
      node({ id: 'c1', label: 'Open', targetPageKey: 'open', requiredEntitlement: '' }),
    ];
    expect(flattenDesignerPages(tree)[0].requiredEntitlement).toBeUndefined();
  });

  it('preserves the folder trail across three levels', () => {
    const tree = [
      node({
        id: 'c1',
        label: 'A',
        children: [node({ id: 'g1', label: 'B', children: [node({ id: 'i1', label: 'C', targetPageKey: 'c' })] })],
      }),
    ];
    expect(flattenDesignerPages(tree)[0].label).toBe('A › B › C');
  });

  // The designer item must still be subject to the same profile gate.
  it('feeds a designer item through the profile gate', () => {
    const cat: CategoryConfig = {
      label: 'Build',
      key: 'weave',
      icon: null,
      defaultPath: '/x',
      color: { primary: '#000', light: '#111', dark: '#222', background: '#333' },
      menus: [
        {
          label: 'Pages & APIs',
          icon: null,
          items: flattenDesignerPages([
            node({ id: 'i1', label: 'Secret', targetPageKey: 'secret', requiredEntitlement: 'PLATFORM_OPERATOR' }),
          ]),
        },
      ],
    };
    expect(filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'BASE_USER')).toHaveLength(0);
    expect(filterNavigationByCapabilities([cat], {}, false, ORG_ACCESS_VISIBLE, 'PLATFORM_OPERATOR')).toHaveLength(1);
  });
});
