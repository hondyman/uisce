import apiClient from '../utils/apiClient';

export interface NavigationMenuNode {
  id: string;
  parentId?: string | null;
  nodeKey: string;
  label: string;
  icon?: string | null;
  targetPageKey?: string | null;
  displayOrder: number;
  requiredEntitlement: string;
  children?: NavigationMenuNode[];
  /** From the gold copy: every tenant has it, read-only outside the gold copy. */
  inherited?: boolean;
  /** Gold-copy entry this tenant has switched off (with everything under it). Only returned to the Menu Designer. */
  hidden?: boolean;
}

export interface NavigationMenuUpsert {
  parentId?: string | null;
  nodeKey: string;
  label: string;
  icon?: string | null;
  targetPageKey?: string | null;
  displayOrder: number;
  requiredEntitlement?: string;
}

const BASE = '/navigation-menu';

export const NavigationMenuApi = {
  /** Hidden entries are left out unless `includeHidden` (the Menu Designer asks for them). */
  listTree: (includeHidden = false): Promise<NavigationMenuNode[]> =>
    apiClient<NavigationMenuNode[]>(includeHidden ? `${BASE}?includeHidden=true` : BASE),

  /** Switch a gold-copy entry off (or back on) for this tenant only. */
  setHidden: (id: string, hidden: boolean): Promise<void> =>
    apiClient<void>(`${BASE}/${id}/hidden`, {
      method: 'PUT',
      body: JSON.stringify({ hidden }),
    }),

  create: (node: NavigationMenuUpsert): Promise<NavigationMenuNode> =>
    apiClient<NavigationMenuNode>(BASE, {
      method: 'POST',
      body: JSON.stringify(node),
    }),

  update: (id: string, node: NavigationMenuUpsert): Promise<NavigationMenuNode> =>
    apiClient<NavigationMenuNode>(`${BASE}/${id}`, {
      method: 'PUT',
      body: JSON.stringify(node),
    }),

  remove: (id: string): Promise<void> =>
    apiClient<void>(`${BASE}/${id}`, { method: 'DELETE' }),
};

export default NavigationMenuApi;
