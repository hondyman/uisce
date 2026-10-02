import React from 'react';
import type { CorePageDefinition } from '../../types/pageStudio';
import { SelectionProvider } from './SelectionContext';
import { PresentationProvider } from './PresentationRuntime';
import RuntimePage from './app/RuntimePage';
import { useResolvedPage } from './app/fragmentLoad';

interface DraftPreviewProps {
  draft: CorePageDefinition;
  tenantId: string;
  /** When true, parent (PageArtboard) already supplies page padding. */
  framed?: boolean;
}

/**
 * Renders the CURRENT in-memory draft exactly as a consumer would see it -
 * the same RuntimePage PageBrowser.tsx uses (tabs, badges, app runtime) -
 * so unsaved edits show up in Preview without a save-then-reload round
 * trip. Unlike EmbeddedPageContent.tsx, which fetches a page by slug.
 */
const DraftPreview: React.FC<DraftPreviewProps> = ({ draft: unresolved, tenantId, framed }) => {
  // The preview shows the page with the fragments it names applied; the draft itself is untouched.
  const { page: draft } = useResolvedPage(unresolved);
  const tabs = draft.tabs && draft.tabs.length > 0
    ? draft.tabs
    : [{ id: '__default__', label: draft.name || 'Page 1', layout: draft.layout }];
  return (
    <SelectionProvider key={draft.id || 'new'}>
      <PresentationProvider rules={draft.presentationEvents || []}>
        <RuntimePage
          name={draft.name}
          slug={draft.slug}
          tabs={tabs}
          filterBar={draft.filterBar}
          components={draft.components}
          dataSources={draft.dataSources || []}
          tenantId={tenantId}
          app={draft.app}
          padded={!framed}
        />
      </PresentationProvider>
    </SelectionProvider>
  );
};

export default DraftPreview;
