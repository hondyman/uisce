import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';

/**
 * Settings (slug fabric-settings), served at /fabric/settings via STUDIO_ROUTES.
 * Appearance is the one real setting: it is saved per browser by the app's own
 * chooser (features/platform-settings/studio.tsx). The IP-validation, tenant-
 * assignment and notification cards and the access-policy list were retired:
 * their values were only kept in this browser, nothing enforced them, and the
 * policies were a fixed sample list.
 */

const layout = (root: string, spec: Record<string, { type: 'Column'; children: string[]; style?: Record<string, string> }>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

function fabricSettingsPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    hdr: {
      id: 'hdr',
      type: 'PageHeader',
      props: { icon: 'settings', title: 'Settings', subtitle: 'Appearance for this browser.' },
      style: { flex: '1 1 320px' },
    },
    appearance: { id: 'appearance', type: 'platformSettings.Appearance', props: {} },
  };

  const main = layout('page_root', {
    page_root: { type: 'Column', children: ['hdr', 'appearance'], style: { gap: '16px' } },
  });

  return {
    name: 'Settings',
    slug: 'fabric-settings',
    description: 'Appearance for this browser. Built in Page Studio.',
    version: 1,
    isCore: true,
    status: 'published',
    layout: main,
    tabs: [],
    filterBar: layout('fb_root', { fb_root: { type: 'Column', children: [], style: { gap: '0px' } } }),
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 900, padding: 3 },
      variables: [],
      queries: [],
    },
  };
}

export const fabricSettingsBlueprint = () => structuredClone(fabricSettingsPage());
