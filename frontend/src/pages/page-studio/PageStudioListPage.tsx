import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box, Typography, TextField, InputAdornment, Button, Card, CardActionArea, CardContent,
  Chip, Stack, CircularProgress, Alert, Grid, IconButton, Tooltip, Menu, MenuItem, Dialog,
  DialogTitle, DialogContent, DialogActions, Select, MenuItem as SelectMenuItem, InputLabel, FormControl,
  FormControlLabel, Switch,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import DescriptionIcon from '@mui/icons-material/Description';
import VerifiedIcon from '@mui/icons-material/Verified';
import PersonIcon from '@mui/icons-material/Person';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import EditIcon from '@mui/icons-material/Edit';
import DriveFileRenameOutlineIcon from '@mui/icons-material/DriveFileRenameOutline';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DeleteIcon from '@mui/icons-material/Delete';
import { PageStudioApi } from '../../api/pageStudio';
import type { CorePageDefinition } from '../../types/pageStudio';
import { useTenant } from '../../contexts/TenantContext';
import { apiClient } from '../../utils/apiClient';
import { generatePageDraft, type BOOption } from './generatePageDraft';
import type { GeneratedPageKind } from '../../api/pageStudio';
import { NavigationMenuApi, NavigationMenuNode } from '../../api/navigationMenu';

const flattenMenuNodes = (nodes: NavigationMenuNode[], depth = 0): { node: NavigationMenuNode; depth: number }[] =>
  nodes.flatMap((node) => [{ node, depth }, ...flattenMenuNodes(node.children || [], depth + 1)]);

/**
 * The searchable page list, split out from PageStudioPage.tsx's old
 * combined list+editor layout - that layout swapped the editor in over
 * the same URL on click, so there was no page to land on directly, share,
 * or bookmark. This is the list half; PageStudioDetailsPage.tsx is the
 * details/tabs half, reached via /page-studio/:id the same way Business
 * Object Manager splits its list and detail pages.
 */
const PageStudioListPage: React.FC = () => {
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const [pages, setPages] = useState<CorePageDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [menuAnchor, setMenuAnchor] = useState<{ el: HTMLElement; page: CorePageDefinition } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CorePageDefinition | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [cloning, setCloning] = useState(false);
  const [renameTarget, setRenameTarget] = useState<CorePageDefinition | null>(null);
  const [renameDraft, setRenameDraft] = useState('');
  const [renaming, setRenaming] = useState(false);

  const [aiOpen, setAiOpen] = useState(false);
  const [aiBOs, setAiBOs] = useState<BOOption[]>([]);
  const [aiBOId, setAiBOId] = useState('');
  const [aiDescription, setAiDescription] = useState('');
  const [aiPageKind, setAiPageKind] = useState<GeneratedPageKind>('dashboard');
  const [aiGenerating, setAiGenerating] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  const [aiAddToMenu, setAiAddToMenu] = useState(true);
  const [aiMenuParentId, setAiMenuParentId] = useState('');
  const [menuNodes, setMenuNodes] = useState<NavigationMenuNode[]>([]);

  const openAiDialog = () => {
    setAiError(null);
    setAiBOId('');
    setAiDescription('');
    setAiPageKind('dashboard');
    setAiAddToMenu(true);
    setAiMenuParentId('');
    setAiOpen(true);
    apiClient<unknown>('/business-objects', { headers: tenant?.id ? { 'X-Tenant-ID': tenant.id } : undefined })
      .then((data) => {
        const rawList = Array.isArray(data) ? data : data && typeof data === 'object' ? Object.values(data as Record<string, unknown>) : [];
        const normalized = rawList
          .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
          .map((item) => ({
            id: String(item.id ?? item.name),
            key: String(item.key ?? item.technicalName ?? item.technical_name ?? item.name ?? ''),
            name: String(item.name ?? ''),
            display_name: String(item.displayName ?? item.display_name ?? item.name ?? ''),
          }));
        setAiBOs(normalized);
      })
      .catch(() => setAiBOs([]));
    NavigationMenuApi.listTree().then(setMenuNodes).catch(() => setMenuNodes([]));
  };

  const handleGenerate = async () => {
    const bo = aiBOs.find((b) => b.id === aiBOId);
    if (!bo) return;
    setAiGenerating(true);
    setAiError(null);
    try {
      const spec = await PageStudioApi.generateSpec(bo.id, bo.key, bo.display_name || bo.name, aiDescription, aiPageKind);
      const draft = await generatePageDraft(bo, spec, aiDescription, tenant?.id || '');
      const saved = await PageStudioApi.savePage(draft);
      if (aiAddToMenu) {
        // A menu entry is best-effort: a page the AI just generated is
        // still real and saved even if this fails (e.g. duplicate
        // nodeKey), so a menu-wiring error surfaces as a warning, not a
        // failed generation.
        await NavigationMenuApi.create({
          parentId: aiMenuParentId || null,
          nodeKey: saved.slug,
          label: saved.name,
          targetPageKey: saved.slug,
          displayOrder: 0,
        }).catch((err) => console.error('Failed to add generated page to navigation menu', err));
      }
      setAiOpen(false);
      navigate(`${saved.id}?aiDraft=1`);
    } catch (err) {
      setAiError(err instanceof Error ? err.message : 'Failed to generate page');
    } finally {
      setAiGenerating(false);
    }
  };

  const load = () => {
    setLoading(true);
    PageStudioApi.listPages()
      .then((data) => setPages(data))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load pages'))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return pages;
    return pages.filter((p) =>
      p.name.toLowerCase().includes(q) ||
      p.slug.toLowerCase().includes(q) ||
      (p.description || '').toLowerCase().includes(q)
    );
  }, [pages, search]);

  const closeMenu = () => setMenuAnchor(null);

  const handleClone = async (page: CorePageDefinition) => {
    closeMenu();
    setCloning(true);
    try {
      const cloned = await PageStudioApi.clonePage(page.id);
      navigate(cloned.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clone page');
    } finally {
      setCloning(false);
    }
  };

  const handleRename = async () => {
    if (!renameTarget) return;
    const trimmed = renameDraft.trim();
    if (!trimmed) return;
    setRenaming(true);
    try {
      await PageStudioApi.updatePage(renameTarget.id, { name: trimmed });
      setRenameTarget(null);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rename page');
    } finally {
      setRenaming(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await PageStudioApi.deletePage(deleteTarget.id);
      setDeleteTarget(null);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete page');
    } finally {
      setDeleting(false);
    }
  };

  return (
    <Box sx={{ p: 4, maxWidth: 1100, mx: 'auto' }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={800}>Page Designer</Typography>
          <Typography variant="body2" color="text.secondary">Search and open a page, or start a new one.</Typography>
        </Box>
        <Stack direction="row" spacing={1.5}>
          <Button variant="outlined" startIcon={<AutoAwesomeIcon />} onClick={openAiDialog}>
            Generate with AI
          </Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => navigate('new')}>
            New Page
          </Button>
        </Stack>
      </Stack>

      <TextField
        fullWidth
        placeholder="Search pages by name, slug, or description…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        sx={{ mb: 3 }}
        InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
      />

      {loading && <Box sx={{ display: 'flex', justifyContent: 'center', p: 6 }}><CircularProgress /></Box>}
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      {cloning && <Alert severity="info" sx={{ mb: 2 }}>Cloning page…</Alert>}

      {!loading && filtered.length === 0 && (
        <Alert severity="info">
          {pages.length === 0 ? 'No pages yet - create your first one.' : 'No pages match your search.'}
        </Alert>
      )}

      <Grid container spacing={2}>
        {filtered.map((page) => (
          <Grid key={page.id} size={{ xs: 12, sm: 6, md: 4 }}>
            <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', position: 'relative' }}>
              <IconButton
                size="small"
                sx={{ position: 'absolute', top: 6, right: 6, zIndex: 1 }}
                onClick={(e) => { e.stopPropagation(); setMenuAnchor({ el: e.currentTarget, page }); }}
              >
                <MoreVertIcon fontSize="small" />
              </IconButton>
              <CardActionArea onClick={() => navigate(page.id)} sx={{ height: '100%', p: 0.5 }}>
                <CardContent>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1, pr: 3 }}>
                    <Tooltip title={page.isCore ? 'Core page (gold-copy, inherited read-only)' : 'Custom page'}>
                      {page.isCore ? <VerifiedIcon color="primary" fontSize="small" /> : <PersonIcon color="action" fontSize="small" />}
                    </Tooltip>
                    <DescriptionIcon color="disabled" fontSize="small" />
                    <Typography variant="subtitle1" fontWeight={700} noWrap>{page.name}</Typography>
                  </Stack>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                    /{page.slug}
                  </Typography>
                  {page.description && (
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5, display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
                      {page.description}
                    </Typography>
                  )}
                  <Stack direction="row" spacing={1}>
                    <Chip
                      size="small"
                      label={page.status === 'published' ? 'Published' : 'Draft'}
                      color={page.status === 'published' ? 'success' : 'default'}
                      variant={page.status === 'published' ? 'filled' : 'outlined'}
                    />
                    <Chip size="small" label={`v${page.version ?? 1}`} variant="outlined" />
                  </Stack>
                </CardContent>
              </CardActionArea>
            </Card>
          </Grid>
        ))}
      </Grid>

      <Menu anchorEl={menuAnchor?.el} open={!!menuAnchor} onClose={closeMenu}>
        <MenuItem onClick={() => { const p = menuAnchor!.page; closeMenu(); navigate(p.id); }}>
          <EditIcon fontSize="small" sx={{ mr: 1 }} /> Edit
        </MenuItem>
        <MenuItem onClick={() => { const p = menuAnchor!.page; setRenameDraft(p.name); setRenameTarget(p); closeMenu(); }}>
          <DriveFileRenameOutlineIcon fontSize="small" sx={{ mr: 1 }} /> Rename
        </MenuItem>
        <MenuItem onClick={() => handleClone(menuAnchor!.page)}>
          <ContentCopyIcon fontSize="small" sx={{ mr: 1 }} /> Clone
        </MenuItem>
        <MenuItem onClick={() => { setDeleteTarget(menuAnchor!.page); closeMenu(); }} sx={{ color: 'error.main' }}>
          <DeleteIcon fontSize="small" sx={{ mr: 1 }} /> Delete
        </MenuItem>
      </Menu>

      <Dialog open={!!renameTarget} onClose={() => !renaming && setRenameTarget(null)}>
        <DialogTitle>Rename page</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            label="Page name"
            value={renameDraft}
            onChange={(e) => setRenameDraft(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleRename(); }}
            sx={{ mt: 1 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRenameTarget(null)} disabled={renaming}>Cancel</Button>
          <Button variant="contained" onClick={handleRename} disabled={renaming || !renameDraft.trim()}>
            {renaming ? 'Saving…' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete "{deleteTarget?.name}"?</DialogTitle>
        <DialogContent>
          <Typography variant="body2">This can't be undone.</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={aiOpen} onClose={() => !aiGenerating && setAiOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Generate a page with AI</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Pick a Business Object and a page kind. Widgets bind to that object and its related objects — never the whole catalog. You will land in the editor to refine (copilot follow-ups do not wipe your layout).
          </Typography>
          {aiError && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setAiError(null)}>{aiError}</Alert>}
          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel id="ai-bo-select-label">Business Object</InputLabel>
            <Select
              labelId="ai-bo-select-label"
              label="Business Object"
              value={aiBOId}
              onChange={(e) => setAiBOId(e.target.value as string)}
            >
              {aiBOs.map((bo) => (
                <SelectMenuItem key={bo.id} value={bo.id}>{bo.display_name || bo.name}</SelectMenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl fullWidth sx={{ mb: 2 }}>
            <InputLabel id="ai-kind-label">Page kind</InputLabel>
            <Select
              labelId="ai-kind-label"
              label="Page kind"
              value={aiPageKind}
              onChange={(e) => setAiPageKind(e.target.value as GeneratedPageKind)}
            >
              <SelectMenuItem value="list">List — table of records (optional slicers)</SelectMenuItem>
              <SelectMenuItem value="detail">Detail — form plus related tables</SelectMenuItem>
              <SelectMenuItem value="master-detail">Master-detail — list rail + form</SelectMenuItem>
              <SelectMenuItem value="dashboard">Dashboard — KPIs, charts, table</SelectMenuItem>
            </Select>
          </FormControl>
          <TextField
            fullWidth
            multiline
            minRows={3}
            label="What should this page show? (optional)"
            placeholder="e.g. A summary dashboard with key totals and a full record table"
            value={aiDescription}
            onChange={(e) => setAiDescription(e.target.value)}
            sx={{ mb: 2 }}
          />
          <FormControlLabel
            control={<Switch checked={aiAddToMenu} onChange={(e) => setAiAddToMenu(e.target.checked)} />}
            label="Add this page to the navigation menu"
          />
          {aiAddToMenu && (
            <FormControl fullWidth size="small" sx={{ mt: 1 }}>
              <InputLabel id="ai-menu-parent-label">Menu location</InputLabel>
              <Select
                labelId="ai-menu-parent-label"
                label="Menu location"
                value={aiMenuParentId}
                onChange={(e) => setAiMenuParentId(e.target.value as string)}
              >
                <SelectMenuItem value="">Top level</SelectMenuItem>
                {flattenMenuNodes(menuNodes).map(({ node, depth }) => (
                  <SelectMenuItem key={node.id} value={node.id}>{'  '.repeat(depth)}{node.label}</SelectMenuItem>
                ))}
              </Select>
            </FormControl>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAiOpen(false)} disabled={aiGenerating}>Cancel</Button>
          <Button
            variant="contained"
            startIcon={aiGenerating ? <CircularProgress size={16} color="inherit" /> : <AutoAwesomeIcon />}
            onClick={handleGenerate}
            disabled={!aiBOId || aiGenerating}
          >
            {aiGenerating ? 'Generating…' : 'Generate'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default PageStudioListPage;
