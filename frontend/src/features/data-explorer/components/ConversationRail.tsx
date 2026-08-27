import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  TextField,
  InputAdornment,
  Stack,
  IconButton,
  Tooltip,
  Button,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  CircularProgress,
  Chip,
  Divider,
} from '@mui/material';
import {
  Search as SearchIcon,
  Add as AddIcon,
  History as HistoryIcon,
  Star as StarIcon,
  People as PeopleIcon,
  Folder as FolderIcon,
  MoreVert as MoreVertIcon,
  PushPin as PinIcon,
  PushPinOutlined as PinOutlinedIcon,
  Share as ShareIcon,
  ShareOutlined as ShareOutlinedIcon,
  Edit as EditIcon,
  DeleteOutline as DeleteIcon,
  Close as CloseIcon,
  ChatBubbleOutline as ChatIcon,
} from '@mui/icons-material';
import {
  EXPLORER_ACCENT,
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';
import type {
  Conversation,
  ConversationFolder,
} from '../types/conversationTypes';
import { useExplorerConversations } from '../hooks/useExplorerConversations';

interface ConversationRailProps {
  activeId: string | null;
  onSelect: (conversation: Conversation) => void;
  onNewConversation: () => void;
}

function relativeTime(iso?: string): string {
  if (!iso) return 'Just now';
  const ts = new Date(iso).getTime();
  const diff = Math.max(0, Date.now() - ts);
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return 'Just now';
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d`;
  return new Date(iso).toLocaleDateString();
}

type Section = 'recent' | 'pinned' | 'shared' | 'folders' | { folder: string };

function NavItem({
  icon,
  label,
  count,
  active,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  count?: number;
  active?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      fullWidth
      onClick={onClick}
      startIcon={icon}
      sx={{
        justifyContent: 'flex-start',
        color: active ? EXPLORER_TEXT : EXPLORER_MUTED,
        bgcolor: active ? EXPLORER_BG : 'transparent',
        borderRadius: 2,
        textTransform: 'none',
        fontWeight: 600,
        px: 1.5,
        py: 1,
        '&:hover': { bgcolor: EXPLORER_BG, color: EXPLORER_TEXT },
      }}
    >
      <Box sx={{ flex: 1, textAlign: 'left' }}>{label}</Box>
      {typeof count === 'number' && (
        <Chip
          label={count}
          size="small"
          sx={{
            height: 18,
            fontSize: 10,
            fontWeight: 700,
            bgcolor: EXPLORER_BG,
            color: EXPLORER_MUTED,
          }}
        />
      )}
    </Button>
  );
}

export const ConversationRail: React.FC<ConversationRailProps> = ({
  activeId,
  onSelect,
  onNewConversation,
}) => {
  const {
    conversations,
    folders,
    conversationsByScope,
    conversationsInFolder,
    isLoading,
    createFolder,
    renameFolder,
    deleteFolder,
    pin,
    share,
    moveToFolder,
    deleteConversation,
    refresh,
  } = useExplorerConversations();

  const [scope, setScope] = useState<Section>('recent');
  const [search, setSearch] = useState('');
  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [moveTarget, setMoveTarget] = useState<{ id: string; anchor: HTMLElement | null }>({
    id: '',
    anchor: null,
  });
  const [shareDialog, setShareDialog] = useState<{ id: string; value: string }>({
    id: '',
    value: '',
  });
  const [renameDialog, setRenameDialog] = useState<{ id: string; value: string; kind: 'folder' | 'conversation' }>({
    id: '',
    value: '',
    kind: 'folder',
  });
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; kind: 'folder' | 'conversation'; name: string }>({
    id: '',
    kind: 'conversation',
    name: '',
  });

  const visibleConversations: Conversation[] = useMemo(() => {
    const filtered = (list: Conversation[]) => {
      if (!search.trim()) return list;
      const q = search.toLowerCase();
      return list.filter(
        (c) =>
          c.title.toLowerCase().includes(q) ||
          c.messages.some((m) => m.content.toLowerCase().includes(q))
      );
    };
    if (typeof scope === 'string') {
      return filtered(conversationsByScope[scope as keyof typeof conversationsByScope] || []);
    }
    return filtered(conversationsInFolder(scope.folder));
  }, [scope, conversationsByScope, conversationsInFolder, search]);

  const openMoveMenu = (id: string, anchor: HTMLElement) =>
    setMoveTarget({ id, anchor });
  const closeMoveMenu = () => setMoveTarget({ id: '', anchor: null });

  const performMove = async (folderId: string | null) => {
    await moveToFolder(moveTarget.id, folderId);
    closeMoveMenu();
  };

  const performShare = async () => {
    await share(shareDialog.id, shareDialog.value.split(/[,\s]+/).filter(Boolean));
    setShareDialog({ id: '', value: '' });
  };

  const openRenameFolder = (folder: ConversationFolder) => {
    setRenameDialog({ id: folder.id, value: folder.name, kind: 'folder' });
  };

  const openRenameConversation = (conv: Conversation) => {
    setRenameDialog({ id: conv.id, value: conv.title, kind: 'conversation' });
  };

  const performRename = async () => {
    const { id, value, kind } = renameDialog;
    if (!id || !value.trim()) {
      setRenameDialog({ id: '', value: '', kind });
      return;
    }
    if (kind === 'folder') {
      await renameFolder(id, value.trim());
    } else {
      const target = conversations.find((c) => c.id === id);
      if (target) {
        const updated = { ...target, title: value.trim() };
        onSelect(updated);
        await refresh();
      }
    }
    setRenameDialog({ id: '', value: '', kind });
  };

  const performCreateFolder = async () => {
    if (!newFolderName.trim()) {
      setNewFolderOpen(false);
      return;
    }
    await createFolder(newFolderName.trim());
    setNewFolderName('');
    setNewFolderOpen(false);
  };

  const currentFolder =
    typeof scope === 'object' ? folders.find((f) => f.id === scope.folder) ?? null : null;

  const renderConversationRow = (conv: Conversation) => (
    <Box
      key={conv.id}
      onClick={() => onSelect(conv)}
      sx={{
        position: 'relative',
        p: 1.25,
        pl: 1.5,
        borderRadius: 2,
        cursor: 'pointer',
        bgcolor: activeId === conv.id ? 'rgba(249, 245, 6, 0.18)' : 'transparent',
        border: activeId === conv.id ? `1px solid ${EXPLORER_ACCENT}` : '1px solid transparent',
        '&:hover': {
          bgcolor: activeId === conv.id ? 'rgba(249, 245, 6, 0.22)' : EXPLORER_BG,
        },
      }}
    >
      <Stack direction="row" alignItems="flex-start" spacing={1}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Stack direction="row" spacing={0.5} alignItems="center">
            {conv.isPinned && (
              <PinIcon sx={{ fontSize: 12, color: EXPLORER_ACCENT }} />
            )}
            {conv.isShared && (
              <ShareIcon sx={{ fontSize: 12, color: '#3b82f6' }} />
            )}
            <Typography
              variant="body2"
              fontWeight={activeId === conv.id ? 700 : 500}
              sx={{ color: EXPLORER_TEXT, flex: 1, minWidth: 0 }}
              noWrap
            >
              {conv.title || 'Untitled conversation'}
            </Typography>
          </Stack>
          <Stack direction="row" spacing={0.5} alignItems="center" sx={{ mt: 0.5 }}>
            <Typography variant="caption" sx={{ color: EXPLORER_MUTED }} noWrap>
              {conv.messages.length} messages
            </Typography>
            <Typography variant="caption" sx={{ color: EXPLORER_MUTED }}>
              · {relativeTime(conv.updatedAt)}
            </Typography>
          </Stack>
        </Box>
        <IconButton
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            openMoveMenu(conv.id, e.currentTarget);
          }}
          sx={{ color: EXPLORER_MUTED, p: 0.5 }}
        >
          <MoreVertIcon fontSize="small" />
        </IconButton>
      </Stack>
    </Box>
  );

  const folderMenus: Record<string, ConversationFolder[]> = useMemo(() => {
    const result: Record<string, ConversationFolder[]> = {};
    folders.forEach((f) => {
      const parent = f.parentId ?? 'root';
      (result[parent] = result[parent] || []).push(f);
    });
    return result;
  }, [folders]);

  return (
    <Paper
      elevation={0}
      sx={{
        width: 280,
        borderRight: `1px solid ${EXPLORER_BORDER}`,
        bgcolor: EXPLORER_BG,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        flexShrink: 0,
      }}
    >
      <Box sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
        <Box
          sx={{
            width: 28,
            height: 28,
            borderRadius: 1.5,
            bgcolor: EXPLORER_ACCENT,
            color: EXPLORER_TEXT,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <ChatIcon sx={{ fontSize: 18, color: EXPLORER_TEXT }} />
        </Box>
        <Typography variant="subtitle2" sx={{ fontWeight: 700, color: EXPLORER_TEXT }}>
          Conversations
        </Typography>
        <Box sx={{ flex: 1 }} />
        <Tooltip title="Refresh">
          <IconButton size="small" onClick={() => refresh()} sx={{ color: EXPLORER_MUTED }}>
            <HistoryIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>

      <Stack spacing={0.5} sx={{ px: 1 }}>
        <NavItem
          icon={<AddIcon fontSize="small" />}
          label="New conversation"
          active={false}
          onClick={onNewConversation}
        />
        <TextField
          size="small"
          placeholder="Search…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          sx={{
            mx: 0.5,
            mb: 1,
            '& .MuiOutlinedInput-root': {
              borderRadius: 2,
              bgcolor: 'white',
              '& fieldset': { borderColor: EXPLORER_BORDER },
            },
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ fontSize: 16, color: EXPLORER_MUTED }} />
              </InputAdornment>
            ),
          }}
        />
        <NavItem
          icon={<HistoryIcon fontSize="small" />}
          label="Recent"
          count={conversationsByScope.recent.length}
          active={scope === 'recent'}
          onClick={() => setScope('recent')}
        />
        <NavItem
          icon={<StarIcon fontSize="small" />}
          label="Saved"
          count={conversationsByScope.pinned.length}
          active={scope === 'pinned'}
          onClick={() => setScope('pinned')}
        />
        <NavItem
          icon={<PeopleIcon fontSize="small" />}
          label="Shared"
          count={conversationsByScope.shared.length}
          active={scope === 'shared'}
          onClick={() => setScope('shared')}
        />
      </Stack>

      <Box sx={{ px: 1, mt: 1, mb: 1 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography
            variant="caption"
            sx={{
              fontWeight: 700,
              color: EXPLORER_MUTED,
              letterSpacing: 0.5,
              textTransform: 'uppercase',
            }}
          >
            Project folders
          </Typography>
          <Tooltip title="New folder">
            <IconButton
              size="small"
              onClick={() => setNewFolderOpen(true)}
              sx={{ color: EXPLORER_MUTED }}
            >
              <AddIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Stack>
        <Stack spacing={0.25} sx={{ mt: 0.5 }}>
          <NavItem
            icon={<FolderIcon fontSize="small" />}
            label="Unsorted"
            count={conversationsInFolder(null).length}
            active={typeof scope === 'object' && scope.folder === ''}
            onClick={() => setScope({ folder: '' })}
          />
          {(folderMenus.root || []).map((folder) => (
            <Stack key={folder.id} direction="row" alignItems="center" spacing={0.5}>
              <Box sx={{ flex: 1 }}>
                <NavItem
                  icon={<FolderIcon fontSize="small" />}
                  label={folder.name}
                  count={conversationsInFolder(folder.id).length}
                  active={typeof scope === 'object' && scope.folder === folder.id}
                  onClick={() => setScope({ folder: folder.id })}
                />
              </Box>
              <IconButton
                size="small"
                onClick={(e) => openMoveMenu(`folder-${folder.id}`, e.currentTarget)}
                sx={{ color: EXPLORER_MUTED, p: 0.25 }}
              >
                <MoreVertIcon sx={{ fontSize: 14 }} />
              </IconButton>
            </Stack>
          ))}
        </Stack>
      </Box>

      <Divider sx={{ mx: 2, my: 0.5 }} />

      <Box sx={{ flex: 1, overflow: 'auto', px: 1, pb: 2 }}>
        {isLoading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <CircularProgress size={18} sx={{ color: EXPLORER_ACCENT }} />
          </Box>
        )}
        {!isLoading && visibleConversations.length === 0 && (
          <Box
            sx={{
              p: 3,
              textAlign: 'center',
              borderRadius: 2,
              border: `1px dashed ${EXPLORER_BORDER}`,
              color: EXPLORER_MUTED,
            }}
          >
            <Typography variant="caption">
              No conversations yet. Start one with the “New conversation” button above.
            </Typography>
          </Box>
        )}
        <Stack spacing={0.5}>{visibleConversations.map(renderConversationRow)}</Stack>
      </Box>

      {/* Right-click / overflow menu */}
      <Menu
        anchorEl={moveTarget.anchor}
        open={Boolean(moveTarget.anchor)}
        onClose={closeMoveMenu}
      >
        {moveTarget.id.startsWith('folder-') ? (
          [
            <MenuItem
              key="rename-folder"
              onClick={() => {
                const folder = folders.find((f) => `folder-${f.id}` === moveTarget.id);
                if (folder) openRenameFolder(folder);
                closeMoveMenu();
              }}
            >
              <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
              <ListItemText>Rename folder</ListItemText>
            </MenuItem>,
            <MenuItem
              key="delete-folder"
              onClick={() => {
                const folder = folders.find((f) => `folder-${f.id}` === moveTarget.id);
                if (folder) {
                  setConfirmDelete({
                    id: folder.id,
                    kind: 'folder',
                    name: folder.name,
                  });
                }
                closeMoveMenu();
              }}
            >
              <ListItemIcon><DeleteIcon fontSize="small" color="error" /></ListItemIcon>
              <ListItemText sx={{ color: 'error.main' }}>Delete folder</ListItemText>
            </MenuItem>,
          ]
        ) : (
          [
            <MenuItem
              key="rename-conversation"
              onClick={() => {
                const conv = conversations.find((c) => c.id === moveTarget.id);
                if (conv) openRenameConversation(conv);
                closeMoveMenu();
              }}
            >
              <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
              <ListItemText>Rename</ListItemText>
            </MenuItem>,
            <MenuItem
              key="pin"
              onClick={async () => {
                await pin(moveTarget.id);
                closeMoveMenu();
              }}
            >
              <ListItemIcon>
                {conversations.find((c) => c.id === moveTarget.id)?.isPinned ? (
                  <PinOutlinedIcon fontSize="small" />
                ) : (
                  <PinIcon fontSize="small" />
                )}
              </ListItemIcon>
              <ListItemText>
                {conversations.find((c) => c.id === moveTarget.id)?.isPinned
                  ? 'Unpin'
                  : 'Pin'}
              </ListItemText>
            </MenuItem>,
            <MenuItem
              key="share"
              onClick={async () => {
                const conv = conversations.find((c) => c.id === moveTarget.id);
                if (conv) {
                  setShareDialog({
                    id: conv.id,
                    value: conv.sharedWith.join(', '),
                  });
                }
                closeMoveMenu();
              }}
            >
              <ListItemIcon>
                {conversations.find((c) => c.id === moveTarget.id)?.isShared ? (
                  <ShareOutlinedIcon fontSize="small" />
                ) : (
                  <ShareIcon fontSize="small" />
                )}
              </ListItemIcon>
              <ListItemText>
                {conversations.find((c) => c.id === moveTarget.id)?.isShared
                  ? 'Edit share list'
                  : 'Share'}
              </ListItemText>
            </MenuItem>,
            <MenuItem
              key="move-header"
              disabled
              sx={{ opacity: '1 !important', fontSize: 11, fontWeight: 700, color: EXPLORER_MUTED }}
            >
              <ListItemText>Move to folder…</ListItemText>
            </MenuItem>,
            <MenuItem key="move-root" onClick={() => performMove(null)} sx={{ pl: 4 }}>
              <ListItemIcon><FolderIcon fontSize="small" /></ListItemIcon>
              <ListItemText>Unsorted</ListItemText>
            </MenuItem>,
            ...folders.map((f) => (
              <MenuItem key={`move-${f.id}`} onClick={() => performMove(f.id)} sx={{ pl: 4 }}>
                <ListItemIcon><FolderIcon fontSize="small" /></ListItemIcon>
                <ListItemText>{f.name}</ListItemText>
              </MenuItem>
            )),
            <Divider key="div" />,
            <MenuItem
              key="delete-conv"
              onClick={() => {
                const conv = conversations.find((c) => c.id === moveTarget.id);
                if (conv) {
                  setConfirmDelete({ id: conv.id, kind: 'conversation', name: conv.title });
                }
                closeMoveMenu();
              }}
            >
              <ListItemIcon><DeleteIcon fontSize="small" color="error" /></ListItemIcon>
              <ListItemText sx={{ color: 'error.main' }}>Delete</ListItemText>
            </MenuItem>,
          ]
        )}
      </Menu>

      {/* New folder dialog */}
      <Dialog open={newFolderOpen} onClose={() => setNewFolderOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>Create project folder</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            label="Folder name"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            sx={{ mt: 1 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setNewFolderOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={performCreateFolder}
            sx={{
              bgcolor: EXPLORER_ACCENT,
              color: EXPLORER_TEXT,
              textTransform: 'none',
              fontWeight: 700,
              '&:hover': { bgcolor: '#e6e205' },
            }}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Rename dialog */}
      <Dialog open={Boolean(renameDialog.id)} onClose={() => setRenameDialog({ id: '', value: '', kind: 'folder' })} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>
          {renameDialog.kind === 'folder' ? 'Rename folder' : 'Rename conversation'}
        </DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            label="Name"
            value={renameDialog.value}
            onChange={(e) => setRenameDialog((prev) => ({ ...prev, value: e.target.value }))}
            sx={{ mt: 1 }}
          />
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setRenameDialog({ id: '', value: '', kind: 'folder' })}
            sx={{ textTransform: 'none' }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={performRename}
            sx={{
              bgcolor: EXPLORER_ACCENT,
              color: EXPLORER_TEXT,
              textTransform: 'none',
              fontWeight: 700,
              '&:hover': { bgcolor: '#e6e205' },
            }}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Share dialog */}
      <Dialog
        open={Boolean(shareDialog.id)}
        onClose={() => setShareDialog({ id: '', value: '' })}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle sx={{ fontWeight: 700 }}>Share conversation</DialogTitle>
        <DialogContent>
          <Typography variant="caption" sx={{ color: EXPLORER_MUTED }}>
            Comma- or space-separated list of teammates (emails or user IDs).
          </Typography>
          <TextField
            autoFocus
            fullWidth
            label="Share with"
            value={shareDialog.value}
            onChange={(e) => setShareDialog((prev) => ({ ...prev, value: e.target.value }))}
            sx={{ mt: 2 }}
            placeholder="alice@acme.com, bob@acme.com"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShareDialog({ id: '', value: '' })} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={performShare}
            sx={{
              bgcolor: EXPLORER_ACCENT,
              color: EXPLORER_TEXT,
              textTransform: 'none',
              fontWeight: 700,
              '&:hover': { bgcolor: '#e6e205' },
            }}
          >
            Save share
          </Button>
        </DialogActions>
      </Dialog>

      {/* Confirm delete */}
      <Dialog
        open={Boolean(confirmDelete.id)}
        onClose={() => setConfirmDelete({ id: '', kind: 'conversation', name: '' })}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle sx={{ fontWeight: 700 }}>
          Delete {confirmDelete.kind === 'folder' ? 'folder' : 'conversation'}?
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ color: EXPLORER_TEXT }}>
            “{confirmDelete.name}” will be removed
            {confirmDelete.kind === 'folder' ? '. Conversations inside are moved to Unsorted.' : ' permanently.'}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setConfirmDelete({ id: '', kind: 'conversation', name: '' })}
            sx={{ textTransform: 'none' }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={async () => {
              if (!confirmDelete.id) return;
              if (confirmDelete.kind === 'folder') {
                await deleteFolder(confirmDelete.id);
              } else {
                await deleteConversation(confirmDelete.id);
                if (activeId === confirmDelete.id) {
                  onNewConversation();
                }
              }
              setConfirmDelete({ id: '', kind: 'conversation', name: '' });
            }}
            sx={{ textTransform: 'none', fontWeight: 700 }}
          >
            <CloseIcon sx={{ fontSize: 16, mr: 0.5 }} /> Delete
          </Button>
        </DialogActions>
      </Dialog>
      {scope === 'recent' && currentFolder === null && null}
      <ConversationScopeDebug scope={scope} />
    </Paper>
  );
};

// Small dev-only helper to make scope state visible from devtools
function ConversationScopeDebug({ scope }: { scope: Section }) {
  if (typeof window === 'undefined') return null;
  if (!window.location.search.includes('debug=rail')) return null;
  return (
    <Box sx={{ p: 1, fontSize: 11, color: EXPLORER_MUTED }}>
      scope = {typeof scope === 'string' ? scope : `folder:${scope.folder}`}
    </Box>
  );
}

export default ConversationRail;
