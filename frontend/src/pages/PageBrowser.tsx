import React, { useCallback, useEffect, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import {
  Box, Paper, Typography, List, ListItemButton, ListItemIcon, ListItemText, Collapse,
  CircularProgress, Alert, Divider, Tabs, Tab,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import DescriptionIcon from '@mui/icons-material/Description';
import DashboardIcon from '@mui/icons-material/Dashboard';
import { NavigationMenuApi, NavigationMenuNode } from '../api/navigationMenu';
import { PageStudioApi, PageStudioPage } from '../api/pageStudio';
import type { PresentationRule } from '../types/pageStudio';
import { useTenant } from '../contexts/TenantContext';
import RenderLayoutTree from './page-studio/RenderLayoutTree';
import { SelectionProvider } from './page-studio/SelectionContext';
import { PresentationProvider } from './page-studio/PresentationRuntime';

// The consumer-facing side of the Menu Designer: a persistent nav tree
// (same data the designer edits) next to whichever page is selected,
// rendered read-only against live data via the same PageComponentRenderer
// the Page Studio editor's canvas uses. No access-control enforcement yet
// (requiredEntitlement exists on each node but is intentionally not
// checked here) - the user asked for browse-by-menu now, security later.

// Runtime shapes actually stored in page_definitions.layout/components -
interface RuntimeLayoutNode {
  id: string;
  type: string;
  children?: string[];
  props?: Record<string, unknown>;
  style?: Record<string, string>;
}
interface RuntimeComponent {
  id: string;
  type: string;
  props?: Record<string, unknown>;
  style?: Record<string, string>;
}
interface RuntimeTab {
  id: string;
  label: string;
  layout: { root: string; nodes: Record<string, RuntimeLayoutNode> };
}

const NavTree: React.FC<{
  nodes: NavigationMenuNode[];
  activeKey?: string;
  depth?: number;
  onSelect: (key: string) => void;
}> = ({ nodes, activeKey, depth = 0, onSelect }) => {
  const [openIds, setOpenIds] = useState<Set<string>>(new Set(nodes.map((n) => n.id)));

  const toggle = (id: string) => {
    setOpenIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  return (
    <List dense disablePadding>
      {nodes.map((node) => {
        const hasChildren = !!node.children?.length;
        const isLeaf = !!node.targetPageKey;
        const isActive = isLeaf && node.targetPageKey === activeKey;
        return (
          <React.Fragment key={node.id}>
            <ListItemButton
              sx={{ pl: 2 + depth * 2 }}
              selected={isActive}
              onClick={() => {
                if (isLeaf && node.targetPageKey) onSelect(node.targetPageKey);
                else if (hasChildren) toggle(node.id);
              }}
            >
              <ListItemIcon sx={{ minWidth: 32 }}>
                {isLeaf ? <DescriptionIcon fontSize="small" /> : <MenuBookIcon fontSize="small" />}
              </ListItemIcon>
              <ListItemText primaryTypographyProps={{ fontWeight: isLeaf ? 400 : 700, fontSize: 14 }}>
                {node.label}
              </ListItemText>
              {hasChildren && (openIds.has(node.id) ? <ExpandLess fontSize="small" /> : <ExpandMore fontSize="small" />)}
            </ListItemButton>
            {hasChildren && (
              <Collapse in={openIds.has(node.id)} timeout="auto" unmountOnExit>
                <NavTree nodes={node.children!} activeKey={activeKey} depth={depth + 1} onSelect={onSelect} />
              </Collapse>
            )}
          </React.Fragment>
        );
      })}
    </List>
  );
};

const PageContent: React.FC<{ slug: string; recordId?: string }> = ({ slug, recordId }) => {
  const { tenant } = useTenant();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const isCreate = recordId === 'new';
  const isView = searchParams.get('mode') === 'view';
  const [page, setPage] = useState<PageStudioPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
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

  if (loading) {
    return <Box sx={{ display: 'flex', justifyContent: 'center', p: 6 }}><CircularProgress /></Box>;
  }
  if (error) {
    return <Alert severity="error" sx={{ m: 3 }}>{error}</Alert>;
  }
  if (!page) return null;

  const rawTabs = (page as unknown as { tabs?: RuntimeTab[] }).tabs;
  const allTabs: RuntimeTab[] = rawTabs && rawTabs.length > 0
    ? rawTabs
    : [{ id: '__default__', label: page.name, layout: page.layout as unknown as RuntimeTab['layout'] }];
  const tabs = isCreate
    ? allTabs.filter((t) => t.id === 'tab_order' || t.id === allTabs[0]?.id).slice(0, 1)
    : allTabs;
  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0];
  const layout = activeTab.layout;
  const allComponents = (page.components as unknown as Record<string, RuntimeComponent>) || {};
  const components = isCreate
    ? Object.fromEntries(Object.entries(allComponents).filter(([, c]) => c.type !== 'FixCommand'))
    : allComponents;
  const dataSources = (page as unknown as { dataSources?: unknown[] }).dataSources || [];
  const filterBar = (page as unknown as { filterBar?: { root: string; nodes: Record<string, RuntimeLayoutNode> } }).filterBar;

  // A List page's row navigates here with the chosen record's id in the
  // URL (see PageComponentRenderer.tsx's Table "Navigate to another page"
  // row-click behavior) - bind it to this Detail page's primary Business
  // Object (its first data source) so every widget here (child tables via
  // masterFilter, a Form) scopes to that one record without the page
  // needing its own master Table to click through first.
  const primaryBoId = (dataSources[0] as { config?: { boId?: string } } | undefined)?.config?.boId;
  const initialSelection = !isCreate && recordId && primaryBoId ? { boId: primaryBoId, recordId } : null;

  return (
    <SelectionProvider key={`${page.id || slug}:${recordId || ''}`} initialSelection={initialSelection}>
    <PresentationProvider rules={(page as PageStudioPage).presentationEvents || [] as PresentationRule[]}>
    <Box sx={{ p: 3 }}>
      {(recordId || isCreate) && (
        <Box
          onClick={() => navigate(-1)}
          sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, mb: 1, cursor: 'pointer', color: 'primary.main', fontSize: 13, fontWeight: 600 }}
        >
          <ArrowBackIcon fontSize="inherit" /> Back to list
        </Box>
      )}
      <Typography variant="h5" sx={{ fontWeight: 800, mb: 0.5 }}>
        {isCreate ? `New ${page.name.replace(/ detail$/i, '')}` : isView ? page.name : page.name}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: tabs.length > 1 ? 2 : 3 }}>/{page.slug}</Typography>
      {filterBar?.root && (
        <Box sx={{ mb: 2 }}>
          <RenderLayoutTree
            nodeId={filterBar.root}
            nodes={filterBar.nodes}
            components={components}
            dataSources={dataSources}
            tenantId={tenant?.id || ''}
          />
        </Box>
      )}
      {tabs.length > 1 && (
        <Tabs value={activeTab.id} onChange={(_, v) => setActiveTabId(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
          {tabs.map((t) => <Tab key={t.id} value={t.id} label={t.label} sx={{ textTransform: 'none' }} />)}
        </Tabs>
      )}
      {layout?.root ? (
        <RenderLayoutTree
          nodeId={layout.root}
          nodes={layout.nodes}
          components={components}
          dataSources={dataSources}
          tenantId={tenant?.id || ''}
        />
      ) : (
        <Alert severity="info">This page has no components yet.</Alert>
      )}
    </Box>
    </PresentationProvider>
    </SelectionProvider>
  );
};

const PageBrowser: React.FC = () => {
  const { slug, recordId } = useParams<{ slug?: string; recordId?: string }>();
  const navigate = useNavigate();
  const [tree, setTree] = useState<NavigationMenuNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await NavigationMenuApi.listTree();
      setTree(result || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load navigation menu');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <Paper elevation={0} sx={{ width: 280, borderRight: '1px solid', borderColor: 'divider', overflowY: 'auto' }}>
        <Box sx={{ p: 2 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 800, display: 'flex', alignItems: 'center', gap: 1 }}>
            <DashboardIcon fontSize="small" /> Menu
          </Typography>
        </Box>
        <Divider />
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}><CircularProgress size={20} /></Box>
        ) : error ? (
          <Alert severity="error" sx={{ m: 2 }}>{error}</Alert>
        ) : tree.length === 0 ? (
          <Alert severity="info" sx={{ m: 2 }}>No menus configured yet.</Alert>
        ) : (
          <NavTree nodes={tree} activeKey={slug} onSelect={(key) => navigate(`/pages/${key}`)} />
        )}
      </Paper>
      <Box sx={{ flex: 1, overflowY: 'auto' }}>
        {slug ? (
          <PageContent slug={slug} recordId={recordId} />
        ) : (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', opacity: 0.6 }}>
            <Typography>Select a page from the menu</Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export const StandalonePageRenderer: React.FC<{ slug?: string; recordId?: string }> = (props) => {
  const params = useParams<{ slug: string; recordId?: string }>();
  const slug = props.slug || params.slug;
  const recordId = props.recordId || params.recordId;
  if (!slug) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="info">No page slug specified.</Alert>
      </Box>
    );
  }
  return <PageContent slug={slug} recordId={recordId} />;
};

export default PageBrowser;

