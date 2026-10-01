import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box, Paper, Typography, Stack, Button, IconButton, Dialog, DialogTitle, DialogContent,
  DialogActions, TextField, MenuItem, Select, InputLabel, FormControl, FormHelperText, Alert, Tooltip,
  CircularProgress, Chip,
} from '@mui/material';
import { SimpleTreeView } from '@mui/x-tree-view/SimpleTreeView';
import { TreeItem } from '@mui/x-tree-view/TreeItem';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import CreateNewFolderIcon from '@mui/icons-material/CreateNewFolder';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import DescriptionIcon from '@mui/icons-material/Description';
import LaunchIcon from '@mui/icons-material/Launch';
import LockIcon from '@mui/icons-material/Lock';
import { useNavigate } from 'react-router-dom';
import { NavigationMenuApi, NavigationMenuNode, NavigationMenuUpsert } from '../../api/navigationMenu';
import { PageStudioApi, PageStudioPage } from '../../api/pageStudio';

// Menu Designer - PeopleSoft/Workday/Salesforce-style: an arbitrarily deep
// tree of menu nodes (navigation_menu_nodes), where a leaf node is bound to
// a Page Studio page by slug (targetPageKey).
//
// requiredEntitlement is a target_profile_key: the same profile string that
// GET /api/capabilities resolves for the caller (via iam.user_roles, else the
// verified JWT role claims, else BASE_USER). MainNavigation enforces it. It is
// deliberately NOT a menu:* capability key - that is a separate mechanism on
// the hardcoded nav categories, and the two namespaces must not be mixed.
interface EditState {
  mode: 'create' | 'edit';
  parentId: string | null;
  node?: NavigationMenuNode;
}

function slugify(label: string): string {
  return label
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '') || `node_${Date.now()}`;
}

/**
 * Profile keys a node may be gated on. These are the two built-ins seeded by
 * backend/db/migrations/20260930_001_seed_menu_abac_policies.up.sql, plus the
 * convention for tenant-minted roles. A tenant that defines its own roles in
 * iam.roles can use those names here too; they are not enumerable from the
 * designer without an extra endpoint.
 */
const PROFILE_KEYS = ['BASE_USER', 'PLATFORM_OPERATOR'] as const;

const MenuDesignerPage: React.FC = () => {
  const navigate = useNavigate();
  const [tree, setTree] = useState<NavigationMenuNode[]>([]);
  const [pages, setPages] = useState<PageStudioPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string[]>([]);

  const [editState, setEditState] = useState<EditState | null>(null);
  const [formLabel, setFormLabel] = useState('');
  const [formNodeKey, setFormNodeKey] = useState('');
  const [formNodeKeyTouched, setFormNodeKeyTouched] = useState(false);
  const [formTargetPageKey, setFormTargetPageKey] = useState('');
  const [formEntitlement, setFormEntitlement] = useState('BASE_USER');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<NavigationMenuNode | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [treeRes, pagesRes] = await Promise.all([
        NavigationMenuApi.listTree(),
        PageStudioApi.listPages().catch(() => [] as PageStudioPage[]),
      ]);
      setTree(treeRes || []);
      setPages(pagesRes || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load navigation menu');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const allNodeIds = useMemo(() => {
    const ids: string[] = [];
    const walk = (nodes: NavigationMenuNode[]) => {
      for (const n of nodes) {
        ids.push(n.id);
        if (n.children?.length) walk(n.children);
      }
    };
    walk(tree);
    return ids;
  }, [tree]);

  const openCreate = (parentId: string | null) => {
    setEditState({ mode: 'create', parentId });
    setFormLabel('');
    setFormNodeKey('');
    setFormNodeKeyTouched(false);
    setFormTargetPageKey('');
    setFormEntitlement('BASE_USER');
    setSaveError(null);
  };

  const openEdit = (node: NavigationMenuNode) => {
    setEditState({ mode: 'edit', parentId: node.parentId ?? null, node });
    setFormLabel(node.label);
    setFormNodeKey(node.nodeKey);
    setFormNodeKeyTouched(true);
    setFormTargetPageKey(node.targetPageKey || '');
    setFormEntitlement(node.requiredEntitlement || 'BASE_USER');
    setSaveError(null);
  };

  const closeDialog = () => setEditState(null);

  const handleLabelChange = (value: string) => {
    setFormLabel(value);
    if (!formNodeKeyTouched) {
      setFormNodeKey(slugify(value));
    }
  };

  const handleSave = async () => {
    if (!editState) return;
    if (!formLabel.trim() || !formNodeKey.trim()) {
      setSaveError('Label and node key are required');
      return;
    }
    setSaving(true);
    setSaveError(null);
    const payload: NavigationMenuUpsert = {
      parentId: editState.parentId,
      nodeKey: formNodeKey.trim(),
      label: formLabel.trim(),
      targetPageKey: formTargetPageKey || null,
      displayOrder: editState.node?.displayOrder ?? 0,
      requiredEntitlement: formEntitlement || 'BASE_USER',
    };
    try {
      if (editState.mode === 'create') {
        await NavigationMenuApi.create(payload);
      } else if (editState.node) {
        await NavigationMenuApi.update(editState.node.id, payload);
      }
      closeDialog();
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save menu node');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await NavigationMenuApi.remove(deleteTarget.id);
      setDeleteTarget(null);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete menu node');
    }
  };

  const renderTree = (nodes: NavigationMenuNode[]) =>
    nodes.map((node) => (
      <TreeItem
        key={node.id}
        itemId={node.id}
        label={
          <Stack direction="row" alignItems="center" spacing={1} sx={{ py: 0.5, pr: 1 }}>
            {node.targetPageKey ? <DescriptionIcon fontSize="small" color="primary" /> : <MenuBookIcon fontSize="small" color="action" />}
            <Typography variant="body2" sx={{ flex: 1, fontWeight: node.targetPageKey ? 400 : 600 }}>
              {node.label}
            </Typography>
            {node.targetPageKey && (
              <Tooltip title={`Bound to page "${node.targetPageKey}"`}>
                <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                  {node.targetPageKey}
                </Typography>
              </Tooltip>
            )}
            {node.targetPageKey && (
              <Tooltip title="Preview this page">
                <IconButton
                  size="small"
                  onClick={(e) => {
                    e.stopPropagation();
                    navigate(`/pages/${node.targetPageKey}`);
                  }}
                >
                  <LaunchIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            )}
            <Tooltip title="Add child menu item">
              <IconButton size="small" onClick={(e) => { e.stopPropagation(); openCreate(node.id); }}>
                <CreateNewFolderIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            {node.inherited ? (
              // Gold-copy entries: every tenant has them; change them in the gold copy.
              <Tooltip title="From the gold copy - read-only here. You can add your own entries under it.">
                <Chip size="small" variant="outlined" color="primary" label="Core" />
              </Tooltip>
            ) : (
              <>
                <Tooltip title="Edit">
                  <IconButton size="small" onClick={(e) => { e.stopPropagation(); openEdit(node); }}>
                    <EditIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Delete (and any children)">
                  <IconButton size="small" onClick={(e) => { e.stopPropagation(); setDeleteTarget(node); }}>
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </>
            )}
            {node.requiredEntitlement && node.requiredEntitlement !== 'BASE_USER' && !node.inherited && (
              <Tooltip title={`Requires: ${node.requiredEntitlement}`}>
                <LockIcon fontSize="small" color="warning" />
              </Tooltip>
            )}
          </Stack>
        }
      >
        {node.children && node.children.length > 0 ? renderTree(node.children) : null}
      </TreeItem>
    ));

  return (
    <Box sx={{ p: 3, maxWidth: 900, mx: 'auto' }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 800 }}>Menu Designer</Typography>
          <Typography variant="body2" color="text.secondary">
            Build the navigation tree consumers see — menus, folders, and pages, PeopleSoft/Workday/Salesforce-style.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined" onClick={() => navigate('/pages')}>Browse as consumer</Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => openCreate(null)}>
            Add Top-Level Menu
          </Button>
        </Stack>
      </Stack>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Paper variant="outlined" sx={{ p: 2, minHeight: 300 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress size={24} />
          </Box>
        ) : tree.length === 0 ? (
          <Alert severity="info">No menus yet. Click "Add Top-Level Menu" to create one.</Alert>
        ) : (
          <SimpleTreeView expandedItems={expanded} onExpandedItemsChange={(_, ids) => setExpanded(ids)}>
            {renderTree(tree)}
          </SimpleTreeView>
        )}
      </Paper>

      <Dialog open={!!editState} onClose={closeDialog} maxWidth="sm" fullWidth>
        <DialogTitle>
          {editState?.mode === 'create' ? 'Add Menu Item' : 'Edit Menu Item'}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {saveError && <Alert severity="error">{saveError}</Alert>}
            <TextField
              label="Label"
              value={formLabel}
              onChange={(e) => handleLabelChange(e.target.value)}
              fullWidth
            />
            <TextField
              label="Node key"
              value={formNodeKey}
              onChange={(e) => { setFormNodeKey(e.target.value); setFormNodeKeyTouched(true); }}
              helperText="Stable identifier for this node, e.g. 'orders_menu'"
              fullWidth
            />
            <FormControl fullWidth>
              <InputLabel id="target-page-label">Target page (leave blank for a folder)</InputLabel>
              <Select
                labelId="target-page-label"
                label="Target page (leave blank for a folder)"
                value={formTargetPageKey}
                onChange={(e) => setFormTargetPageKey(e.target.value)}
              >
                <MenuItem value="">
                  <em>None — this is a folder/group</em>
                </MenuItem>
                {pages.map((p) => (
                  <MenuItem key={p.id} value={p.slug}>
                    {p.name} ({p.slug})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth>
              <InputLabel id="required-entitlement-label">Required entitlement</InputLabel>
              <Select
                labelId="required-entitlement-label"
                id="required-entitlement"
                label="Required entitlement"
                value={formEntitlement || 'BASE_USER'}
                onChange={(e) => setFormEntitlement(e.target.value)}
              >
                {PROFILE_KEYS.map((key) => (
                  <MenuItem key={key} value={key}>
                    {key === 'BASE_USER' ? `${key} — everyone` : key}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>
                Profile key from IAM. A node is shown only to callers resolved to this
                profile; unknown values hide the node.
              </FormHelperText>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog}>Cancel</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete "{deleteTarget?.label}"?</DialogTitle>
        <DialogContent>
          <Typography variant="body2">
            This will also delete any child menu items underneath it. This can't be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default MenuDesignerPage;
