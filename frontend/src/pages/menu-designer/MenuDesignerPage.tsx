import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box, Paper, Typography, Stack, Button, IconButton, Dialog, DialogTitle, DialogContent,
  DialogActions, TextField, MenuItem, Select, InputLabel, FormControl, FormHelperText, Alert, Tooltip,
  CircularProgress, Chip, FormControlLabel, Switch, Divider,
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
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import SecurityIcon from '@mui/icons-material/Security';
import { useNavigate } from 'react-router-dom';
import { NavigationMenuApi, NavigationMenuNode, NavigationMenuUpsert } from '../../api/navigationMenu';
import { PageStudioApi, PageStudioPage } from '../../api/pageStudio';
import { PAGE_ICONS, PageIcon } from '../page-studio/app/icons';

// Menu Designer - PeopleSoft/Workday/Salesforce-style: an arbitrarily deep
// tree of menu nodes (navigation_menu_nodes), where a leaf node is bound to
// a Page Studio page by slug (targetPageKey).
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

const PROFILE_KEYS = ['BASE_USER', 'PLATFORM_OPERATOR'] as const;

const AVAILABLE_CAPABILITIES = [
  { key: '', label: 'None (Unrestricted)' },
  { key: 'menu:admin', label: 'menu:admin (Administration & Platform)' },
  { key: 'menu:catalog', label: 'menu:catalog (Catalog & Metadata)' },
  { key: 'menu:compliance', label: 'menu:compliance (Compliance & Surveillance)' },
  { key: 'menu:operations', label: 'menu:operations (Operations & Workflows)' },
  { key: 'menu:analytics', label: 'menu:analytics (Analytics & Cubes)' },
  { key: 'menu:fabric', label: 'menu:fabric (Data Fabric & Pipelines)' },
  { key: 'menu:reporting', label: 'menu:reporting (Reporting Studio)' },
  { key: 'menu:security', label: 'menu:security (Security & IAM)' },
];

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
  const [formIcon, setFormIcon] = useState('');
  const [formEntitlement, setFormEntitlement] = useState('BASE_USER');
  const [formCapability, setFormCapability] = useState('');
  const [formHidden, setFormHidden] = useState(false);
  const [formDisplayOrder, setFormDisplayOrder] = useState(0);

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

  const openCreate = (parentId: string | null) => {
    setEditState({ mode: 'create', parentId });
    setFormLabel('');
    setFormNodeKey('');
    setFormNodeKeyTouched(false);
    setFormTargetPageKey('');
    setFormIcon('');
    setFormEntitlement('BASE_USER');
    setFormCapability('');
    setFormHidden(false);
    setFormDisplayOrder(0);
    setSaveError(null);
  };

  const openEdit = (node: NavigationMenuNode) => {
    setEditState({ mode: 'edit', parentId: node.parentId ?? null, node });
    setFormLabel(node.label);
    setFormNodeKey(node.nodeKey);
    setFormNodeKeyTouched(true);
    setFormTargetPageKey(node.targetPageKey || '');
    setFormIcon(node.icon || '');
    setFormEntitlement(node.requiredEntitlement || 'BASE_USER');
    setFormCapability(node.requiredCapability || '');
    setFormHidden(!!node.hidden);
    setFormDisplayOrder(node.displayOrder ?? 0);
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
      icon: formIcon.trim() || null,
      targetPageKey: formTargetPageKey || null,
      displayOrder: Number(formDisplayOrder) || 0,
      requiredEntitlement: formEntitlement || 'BASE_USER',
      requiredCapability: formCapability.trim() || null,
      hidden: formHidden,
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
          <Stack direction="row" alignItems="center" spacing={1} sx={{ py: 0.5, pr: 1, opacity: node.hidden ? 0.5 : 1 }}>
            {node.icon ? (
              <PageIcon name={node.icon} fontSize="small" color={node.targetPageKey ? 'primary' : 'action'} />
            ) : node.targetPageKey ? (
              <DescriptionIcon fontSize="small" color="primary" />
            ) : (
              <MenuBookIcon fontSize="small" color="action" />
            )}
            <Typography variant="body2" sx={{ flex: 1, fontWeight: node.targetPageKey ? 400 : 600 }}>
              {node.label}
            </Typography>
            {node.hidden && (
              <Tooltip title="Hidden from consumer navigation">
                <Chip size="small" icon={<VisibilityOffIcon fontSize="small" />} label="Hidden" color="default" variant="outlined" />
              </Tooltip>
            )}
            {node.requiredCapability && (
              <Tooltip title={`Required capability: ${node.requiredCapability}`}>
                <Chip size="small" icon={<SecurityIcon fontSize="small" />} label={node.requiredCapability} color="secondary" variant="outlined" />
              </Tooltip>
            )}
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
              <Tooltip title="From the gold copy core taxonomy.">
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
    <Box sx={{ p: 3, maxWidth: 960, mx: 'auto' }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 800 }}>Menu Designer</Typography>
          <Typography variant="body2" color="text.secondary">
            Single source of truth for application navigation: categories, menus, capabilities, and Page Studio page bindings.
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
              required
            />
            <TextField
              label="Node key"
              value={formNodeKey}
              onChange={(e) => { setFormNodeKey(e.target.value); setFormNodeKeyTouched(true); }}
              helperText="Stable identifier for this node, e.g. 'orders_menu'"
              fullWidth
              required
            />
            <Stack direction="row" spacing={2}>
              <FormControl fullWidth>
                <InputLabel id="icon-select-label">Icon</InputLabel>
                <Select
                  labelId="icon-select-label"
                  label="Icon"
                  value={formIcon}
                  onChange={(e) => setFormIcon(e.target.value)}
                  renderValue={(selected) => (
                    <Stack direction="row" spacing={1} alignItems="center">
                      {selected && <PageIcon name={selected} fontSize="small" />}
                      <Typography variant="body2">{selected || 'None'}</Typography>
                    </Stack>
                  )}
                >
                  <MenuItem value="">
                    <em>None</em>
                  </MenuItem>
                  {Object.keys(PAGE_ICONS).sort().map((iconKey) => (
                    <MenuItem key={iconKey} value={iconKey}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <PageIcon name={iconKey} fontSize="small" />
                        <Typography variant="body2">{iconKey}</Typography>
                      </Stack>
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <TextField
                label="Display Order"
                type="number"
                value={formDisplayOrder}
                onChange={(e) => setFormDisplayOrder(parseInt(e.target.value, 10) || 0)}
                sx={{ width: 140 }}
                helperText="Sort position"
              />
            </Stack>

            <FormControl fullWidth>
              <InputLabel id="target-page-label">Target page (leave blank for a folder/group)</InputLabel>
              <Select
                labelId="target-page-label"
                label="Target page (leave blank for a folder/group)"
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

            <Divider />

            <FormControl fullWidth>
              <InputLabel id="required-capability-label">Required Capability (ABAC Gate)</InputLabel>
              <Select
                labelId="required-capability-label"
                label="Required Capability (ABAC Gate)"
                value={formCapability}
                onChange={(e) => setFormCapability(e.target.value)}
              >
                {AVAILABLE_CAPABILITIES.map((cap) => (
                  <MenuItem key={cap.key} value={cap.key}>
                    {cap.label}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>
                Category or menu-level capability key enforced for role-based navigation visibility.
              </FormHelperText>
            </FormControl>

            <FormControl fullWidth>
              <InputLabel id="required-entitlement-label">Required Entitlement Profile</InputLabel>
              <Select
                labelId="required-entitlement-label"
                id="required-entitlement"
                label="Required Entitlement Profile"
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
                Profile key from IAM. A node is shown only to callers resolved to this profile.
              </FormHelperText>
            </FormControl>

            <FormControlLabel
              control={
                <Switch
                  checked={formHidden}
                  onChange={(e) => setFormHidden(e.target.checked)}
                  color="warning"
                />
              }
              label="Hide from Navigation (keep node in tree for designer/direct route aliases)"
            />
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
