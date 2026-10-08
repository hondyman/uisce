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
  /** ABAC capability key gating the node (e.g. menu:platform). */
  requiredCapability?: string | null;
  /** Interim: opens a not-yet-redesigned coded screen until it is rebuilt in Page Designer. */
  targetRoute?: string | null;
  hidden?: boolean;
  children?: NavigationMenuNode[];
  /** From the gold copy: every tenant has it, read-only outside the gold copy. */
  inherited?: boolean;
}

export interface NavigationMenuUpsert {
  parentId?: string | null;
  nodeKey: string;
  label: string;
  icon?: string | null;
  targetPageKey?: string | null;
  displayOrder: number;
  requiredEntitlement?: string;
  requiredCapability?: string | null;
  hidden?: boolean;
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

/** App URL path (may hold :params) -> the Page Designer page that serves it. */
export interface RouteAlias {
  path: string;
  pageKey: string;
}

export const RouteAliasApi = {
  list: (): Promise<RouteAlias[]> => apiClient<RouteAlias[]>('/route-aliases'),
};
