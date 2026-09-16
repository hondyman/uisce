import { describe, expect, it } from 'vitest';
import { mergeGeneratedSpecIntoDraft } from '../../pages/page-studio/generatePageDraft';
import type { CorePageDefinition } from '../../types/pageStudio';
import type { GeneratedPageSpec } from '../../api/pageStudio';

const draft = (): CorePageDefinition => ({
  id: 'p1',
  name: 'Order',
  slug: 'order',
  layout: {
    root: 'root',
    nodes: { root: { id: 'root', type: 'Column', children: ['t1'] } },
  },
  components: {
    t1: { id: 't1', type: 'Table', props: { dataSourceId: 'bo_1' } },
  },
  dataSources: [{
    id: 'bo_1',
    name: 'order',
    type: 'business_object',
    config: { boId: '1', boKey: 'order', bindingId: '', displayName: 'Order', relatedBoIds: [] },
  }],
  createdAt: '',
  updatedAt: '',
});

describe('mergeGeneratedSpecIntoDraft', () => {
  it('adds a missing widget type without replacing the existing table', async () => {
    const spec: GeneratedPageSpec = {
      title: 'Order',
      layoutTemplate: 'single-column',
      relatedBusinessObjects: [],
      sections: [
        { boKey: '', type: 'Table', title: 'Orders' },
        { boKey: '', type: 'Form', title: 'Order form' },
      ],
      source: 'template',
    };
    const next = await mergeGeneratedSpecIntoDraft(draft(), spec);
    const types = Object.values(next.components).map((c) => c.type).sort();
    expect(types).toEqual(['Form', 'Table']);
    expect(next.layout.nodes.root.children).toContain('t1');
  });
});
