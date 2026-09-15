import React, { useState } from 'react';
import { Box, Tabs, Tab, Typography, Alert } from '@mui/material';
import type { CorePageDefinition } from '../../types/pageStudio';
import RenderLayoutTree from './RenderLayoutTree';
import { SelectionProvider } from './SelectionContext';

interface DraftPreviewProps {
  draft: CorePageDefinition;
  tenantId: string;
}

/**
 * Renders the CURRENT in-memory draft exactly as a consumer would see it
 * (same tabs, same PageComponentRenderer used by PageBrowser.tsx) - unlike
 * EmbeddedPageContent.tsx, which fetches a page by slug, this renders
 * draft state directly so unsaved edits show up in Preview without a
 * save-then-reload round trip. This is the Preview button's content;
 * previously that button had no onClick handler at all.
 */
const DraftPreview: React.FC<DraftPreviewProps> = ({ draft, tenantId }) => {
  const tabs = draft.tabs && draft.tabs.length > 0
    ? draft.tabs
    : [{ id: '__default__', label: draft.name || 'Page 1', layout: draft.layout }];
  const [activeTabId, setActiveTabId] = useState(tabs[0].id);
  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0];

  return (
    <SelectionProvider key={draft.id || 'new'}>
    <Box sx={{ p: 3, height: '100%', overflowY: 'auto', bgcolor: 'background.default' }}>
      <Typography variant="h5" sx={{ fontWeight: 800, mb: 0.5 }}>{draft.name}</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: tabs.length > 1 ? 2 : 3 }}>/{draft.slug}</Typography>
      {draft.filterBar?.root && (
        <Box sx={{ mb: 2 }}>
          <RenderLayoutTree
            nodeId={draft.filterBar.root}
            nodes={draft.filterBar.nodes}
            components={draft.components}
            dataSources={draft.dataSources || []}
            tenantId={tenantId}
          />
        </Box>
      )}
      {tabs.length > 1 && (
        <Tabs value={activeTab.id} onChange={(_, v) => setActiveTabId(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          {tabs.map((t) => <Tab key={t.id} value={t.id} label={t.label} sx={{ textTransform: 'none' }} />)}
        </Tabs>
      )}
      {activeTab.layout.root ? (
        <RenderLayoutTree
          nodeId={activeTab.layout.root}
          nodes={activeTab.layout.nodes}
          components={draft.components}
          dataSources={draft.dataSources || []}
          tenantId={tenantId}
        />
      ) : (
        <Alert severity="info">This page has no components yet.</Alert>
      )}
    </Box>
    </SelectionProvider>
  );
};

export default DraftPreview;
