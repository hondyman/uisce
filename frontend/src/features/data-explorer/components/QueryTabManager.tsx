import React, { useState } from 'react';
import {
  Box,
  Tabs,
  Tab,
  IconButton,
  Tooltip,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  TextField,
} from '@mui/material';
import {
  Add as AddIcon,
  Close as CloseIcon,
  MoreVert as MoreVertIcon,
  ContentCopy as DuplicateIcon,
  Edit as EditIcon,
} from '@mui/icons-material';
import type { QueryTabState } from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface QueryTabManagerProps {
  tabs: QueryTabState[];
  activeTabId: string;
  onSelectTab: (tabId: string) => void;
  onAddTab: () => void;
  onCloseTab: (tabId: string) => void;
  onRenameTab: (tabId: string, newName: string) => void;
  onDuplicateTab: (tabId: string) => void;
}

export const QueryTabManager: React.FC<QueryTabManagerProps> = ({
  tabs,
  activeTabId,
  onSelectTab,
  onAddTab,
  onCloseTab,
  onRenameTab,
  onDuplicateTab,
}) => {
  const theme = useExplorerTheme();
  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null);
  const [selectedMenuTabId, setSelectedMenuTabId] = useState<string | null>(null);
  const [editingTabId, setEditingTabId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState('');

  const handleOpenMenu = (e: React.MouseEvent<HTMLElement>, tabId: string) => {
    e.stopPropagation();
    setSelectedMenuTabId(tabId);
    setMenuAnchor(e.currentTarget);
  };

  const handleCloseMenu = () => {
    setMenuAnchor(null);
    setSelectedMenuTabId(null);
  };

  const handleStartRename = (tabId: string, currentName: string) => {
    setEditingTabId(tabId);
    setEditingName(currentName);
    handleCloseMenu();
  };

  const handleFinishRename = (tabId: string) => {
    if (editingName.trim()) {
      onRenameTab(tabId, editingName.trim());
    }
    setEditingTabId(null);
  };

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        bgcolor: theme.background,
        borderBottom: `1px solid ${theme.border}`,
        px: 1.5,
        gap: 1,
      }}
    >
      <Tabs
        value={tabs.findIndex((t) => t.id === activeTabId)}
        variant="scrollable"
        scrollButtons="auto"
        sx={{
          minHeight: 36,
          '& .MuiTabs-indicator': {
            bgcolor: theme.accent,
            height: 2.5,
          },
        }}
      >
        {tabs.map((tab) => {
          const isActive = tab.id === activeTabId;
          const isEditing = editingTabId === tab.id;

          return (
            <Tab
              component="div"
              key={tab.id}
              onClick={() => !isEditing && onSelectTab(tab.id)}
              label={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8 }}>
                  {isEditing ? (
                    <TextField
                      size="small"
                      variant="standard"
                      value={editingName}
                      onChange={(e) => setEditingName(e.target.value)}
                      onBlur={() => handleFinishRename(tab.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleFinishRename(tab.id);
                        if (e.key === 'Escape') setEditingTabId(null);
                      }}
                      autoFocus
                      InputProps={{
                        disableUnderline: true,
                        sx: { fontSize: '0.78rem', fontWeight: 700, width: 100, color: theme.text },
                      }}
                      onClick={(e) => e.stopPropagation()}
                    />
                  ) : (
                    <Box
                      onDoubleClick={(e) => {
                        e.stopPropagation();
                        handleStartRename(tab.id, tab.name);
                      }}
                      sx={{
                        fontSize: '0.78rem',
                        fontWeight: isActive ? 700 : 500,
                        color: isActive ? theme.text : theme.textMuted,
                        maxWidth: 140,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {tab.name}
                    </Box>
                  )}

                  {tab.isExecuting && (
                    <Box
                      sx={{
                        width: 6,
                        height: 6,
                        borderRadius: '50%',
                        bgcolor: theme.accent,
                        animation: 'pulse 1.5s infinite',
                      }}
                    />
                  )}

                  <IconButton
                    size="small"
                    onClick={(e) => handleOpenMenu(e, tab.id)}
                    sx={{ p: 0.2, ml: 0.2, opacity: isActive ? 0.8 : 0.4, color: theme.textMuted }}
                  >
                    <MoreVertIcon sx={{ fontSize: 13 }} />
                  </IconButton>

                  {tabs.length > 1 && (
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        onCloseTab(tab.id);
                      }}
                      sx={{ p: 0.2, opacity: 0.6, '&:hover': { opacity: 1, color: theme.error } }}
                    >
                      <CloseIcon sx={{ fontSize: 12 }} />
                    </IconButton>
                  )}
                </Box>
              }
              sx={{
                minHeight: 36,
                py: 0.5,
                px: 1.5,
                textTransform: 'none',
                bgcolor: isActive ? theme.backgroundElevated : 'transparent',
                borderRight: `1px solid ${theme.border}`,
                borderRadius: '6px 6px 0 0',
                transition: 'background 0.15s ease',
                color: isActive ? theme.text : theme.textMuted,
              }}
            />
          );
        })}
      </Tabs>

      <Tooltip title="New query tab">
        <IconButton
          size="small"
          onClick={onAddTab}
          sx={{
            p: 0.6,
            bgcolor: theme.backgroundElevated,
            border: `1px solid ${theme.border}`,
            borderRadius: 1.5,
            color: theme.accent,
            '&:hover': { bgcolor: theme.background },
          }}
        >
          <AddIcon sx={{ fontSize: 16 }} />
        </IconButton>
      </Tooltip>

      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={handleCloseMenu}
        PaperProps={{ sx: { minWidth: 140, boxShadow: 3, bgcolor: theme.backgroundElevated, color: theme.text } }}
      >
        <MenuItem
          onClick={() => {
            const target = tabs.find((t) => t.id === selectedMenuTabId);
            if (target) handleStartRename(target.id, target.name);
          }}
          sx={{ fontSize: '0.78rem', gap: 1, color: theme.text }}
        >
          <ListItemIcon><EditIcon fontSize="small" sx={{ color: theme.text }} /></ListItemIcon>
          <ListItemText primary="Rename Tab" primaryTypographyProps={{ fontSize: '0.78rem' }} />
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (selectedMenuTabId) {
              onDuplicateTab(selectedMenuTabId);
              handleCloseMenu();
            }
          }}
          sx={{ fontSize: '0.78rem', gap: 1, color: theme.text }}
        >
          <ListItemIcon><DuplicateIcon fontSize="small" sx={{ color: theme.text }} /></ListItemIcon>
          <ListItemText primary="Duplicate" primaryTypographyProps={{ fontSize: '0.78rem' }} />
        </MenuItem>
        {tabs.length > 1 && (
          <MenuItem
            onClick={() => {
              if (selectedMenuTabId) {
                onCloseTab(selectedMenuTabId);
                handleCloseMenu();
              }
            }}
            sx={{ fontSize: '0.78rem', gap: 1, color: theme.error }}
          >
            <ListItemIcon><CloseIcon fontSize="small" sx={{ color: theme.error }} /></ListItemIcon>
            <ListItemText primary="Close Tab" primaryTypographyProps={{ fontSize: '0.78rem' }} />
          </MenuItem>
        )}
      </Menu>
    </Box>
  );
};
