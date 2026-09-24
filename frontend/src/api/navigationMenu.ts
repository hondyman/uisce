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
  listTree: (): Promise<NavigationMenuNode[]> => apiClient<NavigationMenuNode[]>(BASE),

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
