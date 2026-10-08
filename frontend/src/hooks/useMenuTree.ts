import { useEffect, useState } from 'react';
import { NavigationMenuApi, type NavigationMenuNode } from '../api/navigationMenu';

/** The tenant's Menu Designer tree (gold copy + own nodes). Empty until loaded or on failure. */
export function useMenuTree(): NavigationMenuNode[] {
  const [tree, setTree] = useState<NavigationMenuNode[]>([]);
  useEffect(() => {
    let live = true;
    NavigationMenuApi.listTree()
      .then((t) => live && setTree(Array.isArray(t) ? t : []))
      .catch(() => undefined);
    return () => {
      live = false;
    };
  }, []);
  return tree;
}
