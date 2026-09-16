import React from 'react';
import { Box } from '@mui/material';
import { PanelNodeProps } from '../../types/pageStudio';
import PageComponentRenderer from './PageComponentRenderer';
import PanelRegion from './PanelRegion';

const DEFAULT_PANEL_PROPS: PanelNodeProps = { side: 'right', collapsible: true, defaultOpen: true, widthPx: 320, label: 'Panel' };

// Minimal runtime shapes: PageBrowser.tsx reads page_definitions JSON
// straight off the wire (RuntimeLayoutNode/RuntimeComponent there), while
// DraftPreview.tsx already has real CorePageDefinition/ComponentDefinition
// values - both fit this looser shape, which is all rendering needs.
interface RenderableNode {
  id: string;
  type: string;
  children?: string[];
  props?: Record<string, unknown>;
  style?: Record<string, string>;
}
interface RenderableComponent {
  id: string;
  type: string;
  props?: Record<string, unknown>;
  style?: Record<string, string>;
}

interface RenderLayoutTreeProps {
  nodeId: string;
  nodes: Record<string, RenderableNode>;
  components: Record<string, RenderableComponent>;
  dataSources: unknown[];
  tenantId: string;
}

/**
 * Read-only render of one layout tree, shared by DraftPreview.tsx (unsaved
 * in-editor preview) and PageBrowser.tsx (the real page a consumer visits)
 * so Panel slide/collapse behavior and resizable-widget sizing (both driven
 * by data these two used to walk with separately copy-pasted `renderNode`
 * functions) can't drift between "what the author previewed" and "what a
 * viewer actually gets". LayoutCanvas.tsx keeps its own version because
 * editing (selection, drag-drop, delete) is a fundamentally different
 * rendering job than display.
 */
const RenderLayoutTree: React.FC<RenderLayoutTreeProps> = ({ nodeId, nodes, components, dataSources, tenantId }) => {
  const node = nodes[nodeId];
  if (!node) {
    const component = components[nodeId];
    if (!component) return null;
    return (
      <Box key={component.id} sx={{ flex: 1, minWidth: 0 }}>
        <PageComponentRenderer component={component as any} dataSources={dataSources as any} tenantId={tenantId} />
      </Box>
    );
  }

  const body = (
    <Box
      key={node.type === 'Panel' ? undefined : nodeId}
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        flexDirection: node.type === 'Row' ? 'row' : 'column',
        gap: 2,
        ...(node.style as React.CSSProperties | undefined),
      }}
    >
      {(node.children || []).map((childId) => (
        <RenderLayoutTree key={childId} nodeId={childId} nodes={nodes} components={components} dataSources={dataSources} tenantId={tenantId} />
      ))}
    </Box>
  );

  if (node.type === 'Panel') {
    const panelProps = { ...DEFAULT_PANEL_PROPS, ...(node.props as Partial<PanelNodeProps> | undefined) };
    return <Box key={nodeId}><PanelRegion {...panelProps}>{body}</PanelRegion></Box>;
  }
  return body;
};

export default RenderLayoutTree;
