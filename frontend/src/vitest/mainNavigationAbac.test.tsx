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
