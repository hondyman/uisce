import React, { useEffect, useState } from 'react';
import { Box, CircularProgress, Alert, Tabs, Tab } from '@mui/material';
import { PageStudioApi } from '../../api/pageStudio';
import type { CorePageDefinition, ComponentDefinition } from '../../types/pageStudio';
import PageComponentRenderer from './PageComponentRenderer';
import { SelectionProvider } from './SelectionContext';

interface EmbeddedPageContentProps {
  slug: string;
  tenantId: string;
  /**
   * When set, this embed is a Detail view for one record (a List Table's
   * "Open in a modal/side panel" row-click action - see
   * PageComponentRenderer.tsx) rather than a plain page embed - seeds a
   * SelectionProvider bound to the embedded page's primary Business Object
   * (its first data source), the same way PageBrowser.tsx's PageContent
   * does for the "Navigate to another page" full-page variant.
   */
  recordId?: string;
}

/**
 * Fetches a page by slug and renders its component tree - the shared
 * piece a Button's "Open Modal" action and PageBrowser.tsx's top-level
 * render both need. Kept separate from PageBrowser rather than importing
 * it directly, since PageBrowser also renders the page chrome (title,
 * nav) that a modal shouldn't repeat - this renders only the layout tree.
 */
const EmbeddedPageContent: React.FC<EmbeddedPageContentProps> = ({ slug, tenantId, recordId }) => {
  const [page, setPage] = useState<CorePageDefinition | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    PageStudioApi.getPageBySlug(slug)
      .then((p) => { if (!cancelled) setPage(p); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load page'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [slug]);

  if (loading) return <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress size={24} /></Box>;
  if (error) return <Alert severity="error">{error}</Alert>;
  if (!page) return null;

  const tabs = page.tabs && page.tabs.length > 0
    ? page.tabs
    : [{ id: '__default__', label: page.name, layout: page.layout }];
  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0];
  const layout = activeTab.layout;

  const renderNode = (nodeId: string): React.ReactNode => {
    const node = layout?.nodes?.[nodeId];
    if (!node) {
      const component = page.components?.[nodeId];
      if (!component) return null;
      return (
        <Box key={component.id} sx={{ mb: 2 }}>
          <PageComponentRenderer component={component as ComponentDefinition} dataSources={page.dataSources || []} tenantId={tenantId} />
        </Box>
      );
    }
    return (
      <Box key={nodeId} sx={{ display: 'flex', flexDirection: node.type === 'Row' ? 'row' : 'column', gap: 2, ...(node.style as React.CSSProperties | undefined) }}>
        {(node.children || []).map((childId) => renderNode(childId))}
      </Box>
    );
  };

  const primaryBoId = (page.dataSources?.[0] as { config?: { boId?: string } } | undefined)?.config?.boId;
  const initialSelection = recordId && primaryBoId ? { boId: primaryBoId, recordId } : null;

  const body = (
    <Box>
      {tabs.length > 1 && (
        <Tabs value={activeTab.id} onChange={(_, v) => setActiveTabId(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          {tabs.map((t) => <Tab key={t.id} value={t.id} label={t.label} sx={{ textTransform: 'none' }} />)}
        </Tabs>
      )}
      {layout?.root ? renderNode(layout.root) : <Alert severity="info">This page has no components yet.</Alert>}
    </Box>
  );

  return recordId
    ? <SelectionProvider key={`${page.id || slug}:${recordId}`} initialSelection={initialSelection}>{body}</SelectionProvider>
    : body;
};

export default EmbeddedPageContent;
