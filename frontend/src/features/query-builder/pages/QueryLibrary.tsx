/**
 * Query Library - list/manage page for saved queries, mirroring Report
 * Builder's ReportLibrary UX (folder tree, search, run/edit/rename/
 * delete/favorite/share row actions) against the real saved-query backend
 * (backend/internal/querybuilder/saved_query_handler.go +
 * saved_query_folder_handler.go) instead of the disconnected mock this
 * replaces (components/reporting/QueryLibraryDashboard.tsx, which rendered
 * report templates mislabeled as queries).
 */
import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box, Paper, Typography, TextField, InputAdornment, Button, IconButton,
  List, ListItemButton, ListItemIcon, ListItemText, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, Chip, Menu, MenuItem,
  Dialog, DialogTitle, DialogContent, DialogActions, Alert, CircularProgress,
  Stack, Tooltip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import CreateNewFolderIcon from '@mui/icons-material/CreateNewFolder';
import FolderIcon from '@mui/icons-material/Folder';
import StorageIcon from '@mui/icons-material/Storage';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import StarIcon from '@mui/icons-material/Star';
import StarBorderIcon from '@mui/icons-material/StarBorder';
import PublicIcon from '@mui/icons-material/Public';
import LockIcon from '@mui/icons-material/Lock';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import {
  listSavedQueries, deleteSavedQuery, cloneSavedQuery, setSavedQueryFavorite,
  setSavedQueryVisibility, listSavedQueryFolders, createSavedQueryFolder,
  deleteSavedQueryFolder, updateSavedQuery,
} from '../services/savedQueryApi';
import type { SavedQuery, SavedQueryFolder } from '../types/queryDef';
import { useNotification } from '../../../hooks/useNotification';

export default function QueryLibrary() {
  const navigate = useNavigate();
  const notify = useNotification();

  const [queries, setQueries] = useState<SavedQuery[]>([]);
  const [folders, setFolders] = useState<SavedQueryFolder[]>([]);
  const [activeFolderId, setActiveFolderId] = useState<string | undefined>(undefined);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null);
  const [menuQuery, setMenuQuery] = useState<SavedQuery | null>(null);
  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [renameOpen, setRenameOpen] = useState(false);
  const [renameValue, setRenameValue] = useState('');

  const load = () => {
    setLoading(true);
    setError(null);
    Promise.all([listSavedQueries({ folderId: activeFolderId }), listSavedQueryFolders()])
      .then(([q, f]) => { setQueries(q); setFolders(f); })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(load, [activeFolderId]);

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return queries;
    return queries.filter((q) => q.name.toLowerCase().includes(term) || q.description?.toLowerCase().includes(term));
  }, [queries, search]);

  const rootFolders = folders.filter((f) => !f.parentId);

  const openMenu = (e: React.MouseEvent<HTMLElement>, q: SavedQuery) => {
    e.stopPropagation();
    setMenuAnchor(e.currentTarget);
    setMenuQuery(q);
  };
  const closeMenu = () => { setMenuAnchor(null); setMenuQuery(null); };

  const handleRun = (q: SavedQuery) => navigate(`/query-builder/editor/${q.id}`);
  const handleEdit = (q: SavedQuery) => navigate(`/query-builder/editor/${q.id}?return_to=${encodeURIComponent('/reports/queries')}`);

  const handleDelete = async (q: SavedQuery) => {
    closeMenu();
    if (!window.confirm(`Delete "${q.name}"? This cannot be undone.`)) return;
    try {
      await deleteSavedQuery(q.id);
      notify.success('Query deleted');
      load();
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  const handleClone = async (q: SavedQuery) => {
    closeMenu();
    try {
      await cloneSavedQuery(q.id);
      notify.success('Query duplicated');
      load();
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  const handleToggleFavorite = async (q: SavedQuery) => {
    try {
      const updated = await setSavedQueryFavorite(q.id, !q.isFavorite);
      setQueries((prev) => prev.map((x) => x.id === q.id ? updated : x));
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  const handleToggleVisibility = async (q: SavedQuery) => {
    closeMenu();
    try {
      const updated = await setSavedQueryVisibility(q.id, q.visibility === 'private' ? 'shared' : 'private');
      setQueries((prev) => prev.map((x) => x.id === q.id ? updated : x));
      notify.success(updated.visibility === 'shared' ? 'Shared with your tenant' : 'Made private');
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  const openRename = () => {
    if (!menuQuery) return;
    setRenameValue(menuQuery.name);
    setRenameOpen(true);
  };

  const submitRename = async () => {
    if (!menuQuery) return;
    try {
      await updateSavedQuery(menuQuery.id, {
        name: renameValue,
        description: menuQuery.description,
        boId: menuQuery.boId,
        bindingId: menuQuery.bindingId,
        relatedBoIds: menuQuery.relatedBoIds,
        chartType: menuQuery.chartType,
        state: menuQuery.state,
        tags: menuQuery.tags,
        folderId: menuQuery.folderId,
      });
      notify.success('Renamed');
      setRenameOpen(false);
      closeMenu();
      load();
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  const submitNewFolder = async () => {
    if (!newFolderName.trim()) return;
    try {
      await createSavedQueryFolder(newFolderName.trim(), activeFolderId);
      setNewFolderName('');
      setNewFolderOpen(false);
      load();
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  const handleDeleteFolder = async (f: SavedQueryFolder) => {
    if (!window.confirm(`Delete folder "${f.name}"? Queries inside become unfiled.`)) return;
    try {
      await deleteSavedQueryFolder(f.id);
      if (activeFolderId === f.id) setActiveFolderId(undefined);
      load();
    } catch (e: any) {
      notify.error(e.message);
    }
  };

  return (
    <Box display="flex" height="100%" minWidth={0}>
      {/* Folder tree */}
      <Paper variant="outlined" sx={{ width: 240, flexShrink: 0, p: 1, borderRadius: 0, overflowY: 'auto' }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" px={1}>
          <Typography variant="subtitle2" sx={{ px: 1 }}>Folders</Typography>
          <Tooltip title="New folder">
            <IconButton size="small" onClick={() => setNewFolderOpen(true)}><CreateNewFolderIcon fontSize="small" /></IconButton>
          </Tooltip>
        </Stack>
        <List dense>
          <ListItemButton selected={!activeFolderId} onClick={() => setActiveFolderId(undefined)}>
            <ListItemIcon sx={{ minWidth: 32 }}><StorageIcon fontSize="small" /></ListItemIcon>
            <ListItemText primary="All queries" />
          </ListItemButton>
          {rootFolders.map((f) => (
            <ListItemButton key={f.id} selected={activeFolderId === f.id} onClick={() => setActiveFolderId(f.id)}
              onContextMenu={(e) => { e.preventDefault(); handleDeleteFolder(f); }}>
              <ListItemIcon sx={{ minWidth: 32 }}><FolderIcon fontSize="small" /></ListItemIcon>
              <ListItemText primary={f.name} secondary={`${f.itemCount} item${f.itemCount === 1 ? '' : 's'}`} />
            </ListItemButton>
          ))}
        </List>
      </Paper>

      {/* Query list */}
      <Box flex={1} minWidth={0} p={2} sx={{ overflowX: 'auto' }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
          <TextField
            size="small" placeholder="Search queries…" value={search} onChange={(e) => setSearch(e.target.value)}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
            sx={{ width: 320 }}
          />
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => navigate('/query-builder/editor/new')}>
            New Query
          </Button>
        </Stack>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {loading ? (
          <Box display="flex" justifyContent="center" p={4}><CircularProgress /></Box>
        ) : filtered.length === 0 ? (
          <Paper variant="outlined" sx={{ p: 4, textAlign: 'center' }}>
            <Typography color="text.secondary">No queries yet. Create one to get started.</Typography>
          </Paper>
        ) : (
          <TableContainer component={Paper} variant="outlined" sx={{ overflowX: 'auto' }}>
            <Table size="small" sx={{ tableLayout: 'fixed', minWidth: 760 }}>
              <TableHead>
                <TableRow>
                  <TableCell padding="checkbox" sx={{ width: 48 }} />
                  <TableCell sx={{ width: '30%' }}>Name</TableCell>
                  <TableCell sx={{ width: '18%' }}>Object</TableCell>
                  <TableCell sx={{ width: '12%' }}>Visibility</TableCell>
                  <TableCell sx={{ width: '15%' }}>Owner</TableCell>
                  <TableCell sx={{ width: '15%' }}>Updated</TableCell>
                  <TableCell align="right" sx={{ width: 90 }} />
                </TableRow>
              </TableHead>
              <TableBody>
                {filtered.map((q) => (
                  <TableRow key={q.id} hover sx={{ cursor: 'pointer' }} onClick={() => handleRun(q)}>
                    <TableCell padding="checkbox">
                      <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleToggleFavorite(q); }}>
                        {q.isFavorite ? <StarIcon fontSize="small" color="warning" /> : <StarBorderIcon fontSize="small" />}
                      </IconButton>
                    </TableCell>
                    <TableCell sx={{ overflow: 'hidden' }}>
                      <Typography variant="body2" fontWeight={500} noWrap>{q.name}</Typography>
                      {q.description && (
                        <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                          {q.description}
                        </Typography>
                      )}
                      {q.isCore && <Chip label="Core" size="small" sx={{ mt: 0.5 }} />}
                    </TableCell>
                    <TableCell sx={{ overflow: 'hidden' }}>
                      <Tooltip title={q.boId}>
                        <Chip size="small" label={q.boId} sx={{ maxWidth: '100%', '& .MuiChip-label': { overflow: 'hidden', textOverflow: 'ellipsis' } }} />
                      </Tooltip>
                      {q.relatedBoIds?.length > 0 && (
                        <Chip size="small" variant="outlined" sx={{ ml: 0.5 }} label={`+${q.relatedBoIds.length}`} />
                      )}
                    </TableCell>
                    <TableCell>
                      {q.visibility === 'shared'
                        ? <Chip size="small" icon={<PublicIcon />} label="Shared" />
                        : <Chip size="small" icon={<LockIcon />} label="Private" variant="outlined" />}
                    </TableCell>
                    <TableCell><Typography variant="caption">{q.createdBy || q.userId}</Typography></TableCell>
                    <TableCell><Typography variant="caption">{new Date(q.updatedAt).toLocaleString()}</Typography></TableCell>
                    <TableCell align="right">
                      <Tooltip title="Run">
                        <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleRun(q); }}>
                          <PlayArrowIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <IconButton size="small" onClick={(e) => openMenu(e, q)}><MoreVertIcon fontSize="small" /></IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>

      <Menu anchorEl={menuAnchor} open={!!menuAnchor} onClose={closeMenu}>
        <MenuItem onClick={() => { handleEdit(menuQuery!); closeMenu(); }}>Edit</MenuItem>
        <MenuItem onClick={openRename}>Rename</MenuItem>
        <MenuItem onClick={() => handleClone(menuQuery!)}>Duplicate</MenuItem>
        <MenuItem onClick={() => handleToggleVisibility(menuQuery!)}>
          {menuQuery?.visibility === 'shared' ? 'Make private' : 'Share with tenant'}
        </MenuItem>
        <MenuItem onClick={() => handleDelete(menuQuery!)} sx={{ color: 'error.main' }}>Delete</MenuItem>
      </Menu>

      <Dialog open={newFolderOpen} onClose={() => setNewFolderOpen(false)}>
        <DialogTitle>New Folder</DialogTitle>
        <DialogContent>
          <TextField autoFocus fullWidth label="Folder name" value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)} sx={{ mt: 1 }} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setNewFolderOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={submitNewFolder}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={renameOpen} onClose={() => setRenameOpen(false)}>
        <DialogTitle>Rename Query</DialogTitle>
        <DialogContent>
          <TextField autoFocus fullWidth label="Name" value={renameValue}
            onChange={(e) => setRenameValue(e.target.value)} sx={{ mt: 1 }} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRenameOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={submitRename}>Save</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
