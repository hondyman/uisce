import React from 'react';
import FolderOutlinedIcon from '@mui/icons-material/FolderOutlined';
import type { NavigationMenuNode, RouteAlias } from '../api/navigationMenu';
import { PageIcon } from '../pages/page-studio/app/icons';
import type { CategoryConfig, NavigationItem, NavigationMenu } from './MainNavigation';

/**
 * Builds the top navigation from the Menu Designer tree (navigation_menu_nodes).
 * No menu content lives in code: labels, grouping, order, icons, gating and
 * targets all come from the tree. Shape: root = category, its child groups =
 * dropdown menus, and every descendant leaf under a group = an item.
 */

// Accent colours cycle by position; they carry no meaning, so the palette is presentation only.
const PALETTE = ['#607D8B', '#2196F3', '#4CAF50', '#FF9800', '#9C27B0', '#E91E63', '#009688', '#795548'];

function colorsFor(index: number): CategoryConfig['color'] {
  const primary = PALETTE[index % PALETTE.length];
  return { primary, light: `${primary}22`, dark: primary, background: `${primary}10` };
}

const icon = (name?: string | null): React.ReactNode =>
  name ? React.createElement(PageIcon, { name, fontSize: 'small' }) : React.createElement(FolderOutlinedIcon, { fontSize: 'small' });

/** Where a node opens: an explicit (interim) coded route, else its page's alias, else /p/:slug. */
export function nodeTarget(node: NavigationMenuNode, aliases: RouteAlias[]): string | undefined {
  if (node.targetRoute) return node.targetRoute;
  if (!node.targetPageKey) return undefined;
  const alias = aliases.find((a) => a.pageKey === node.targetPageKey && !a.path.includes(':'));
  return alias ? (alias.path.startsWith('/') ? alias.path : `/${alias.path}`) : `/p/${node.targetPageKey}`;
}

function sorted(nodes: NavigationMenuNode[] | undefined): NavigationMenuNode[] {
  return [...(nodes ?? [])].filter((n) => !n.hidden).sort((a, b) => a.displayOrder - b.displayOrder || a.label.localeCompare(b.label));
}

function leaves(node: NavigationMenuNode, aliases: RouteAlias[]): NavigationItem[] {
  const target = nodeTarget(node, aliases);
  const own: NavigationItem[] = target
    ? [
        {
          label: node.label,
          path: target,
          icon: icon(node.icon),
          requiredCapability: node.requiredCapability ?? undefined,
          requiredEntitlement: node.requiredEntitlement,
        },
      ]
    : [];
  return [...own, ...sorted(node.children).flatMap((c) => leaves(c, aliases))];
}

export function mapDesignerTreeToCategories(tree: NavigationMenuNode[], aliases: RouteAlias[] = []): CategoryConfig[] {
  const cats: CategoryConfig[] = [];
  sorted(tree).forEach((root) => {
    const groups: NavigationMenu[] = [];
    const direct: NavigationMenuNode[] = [];
    sorted(root.children).forEach((child) => {
      if ((child.children?.length ?? 0) > 0 && !nodeTarget(child, aliases)) {
        const items = leaves(child, aliases);
        if (items.length)
          groups.push({
            label: child.label,
            icon: icon(child.icon),
            items,
            requiredCapability: child.requiredCapability ?? undefined,
            requiredEntitlement: child.requiredEntitlement,
          });
      } else direct.push(child);
    });
    // Leaves sitting directly under the category form one group named after it.
    const directItems = direct.flatMap((n) => leaves(n, aliases));
    if (directItems.length) groups.unshift({ label: root.label, icon: icon(root.icon), items: directItems });
    if (!groups.length) return;
    cats.push({
      label: root.label,
      key: root.nodeKey,
      icon: icon(root.icon),
      defaultPath: groups[0].items[0].path,
      color: colorsFor(cats.length),
      menus: groups,
      requiredCapability: root.requiredCapability ?? undefined,
      requiredEntitlement: root.requiredEntitlement,
    });
  });
  return cats;
}
