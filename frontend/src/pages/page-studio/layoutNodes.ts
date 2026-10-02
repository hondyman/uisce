import type { LayoutNode, PageLayout } from '../../types/pageStudio';

/**
 * A layout's node map, or an empty one. `nodes` is optional on PageLayout because
 * grid-kind and empty layouts have none; tree-editing code treats that as "no nodes".
 */
export const nodesOf = (layout: Pick<PageLayout, 'nodes'>): Record<string, LayoutNode> => layout.nodes ?? {};
