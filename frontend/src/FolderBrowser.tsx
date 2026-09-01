import { useState, useEffect, useCallback } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import IconButton from '@mui/material/IconButton';
import { useNotification } from './hooks/useNotification';
import { useDrop } from 'react-dnd';
import { listFolders, addItemToFolder } from './api';
import type { FullFolder } from './types';
import FolderAnalyticsPanel from './FolderAnalyticsPanel';
import FolderDiffViewer from './FolderDiffViewer';

export const ItemTypes = {
  SAVED_ITEM: 'savedItem',
};

function Folder({ folder, onDropItem }: { folder: FullFolder; onDropItem: (_folderId: string, _item: any) => void }) {
  const theme = useTheme();
  const [{ isOver, canDrop }, drop] = useDrop(() => ({
    accept: ItemTypes.SAVED_ITEM,
    drop: (item: { id: string; type: 'query' | 'workbook' }) => onDropItem(folder.id, item),
    collect: (monitor: any) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  }));

  const [isAnalyticsVisible, setIsAnalyticsVisible] = useState(false);
  const [isDiffVisible, setIsDiffVisible] = useState(false);

  const isDark = theme.palette.mode === 'dark';

  return (
    <Paper
      ref={drop}
      elevation={0}
      sx={{
        p: 2,
        mb: 2,
        border: 1,
        borderColor: isOver ? 'primary.main' : canDrop ? 'success.main' : 'divider',
        bgcolor: isOver ? 'action.hover' : 'background.paper',
        transition: 'all 150ms ease',
        '&:hover': {
          bgcolor: 'action.hover',
        },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
          📁 {folder.name}
        </Typography>
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <IconButton size="small" onClick={() => setIsDiffVisible(true)} title="Compare Versions">
            🔄
          </IconButton>
          <IconButton size="small" onClick={() => setIsAnalyticsVisible(!isAnalyticsVisible)} title="Toggle Analytics">
            📊
          </IconButton>
        </Box>
      </Box>
      {isAnalyticsVisible && <FolderAnalyticsPanel folderId={folder.id} />}
      <Box component="ul" sx={{ listStyle: 'none', pl: 0, '& li': { py: 0.5, pl: 2 } }}>
        {folder.items.map(item => (
          <Typography component="li" key={item.item_id} variant="body2">
            {item.name}
          </Typography>
        ))}
        {folder.items.length === 0 && (
          <Typography component="li" variant="body2" sx={{ color: 'text.secondary', fontStyle: 'italic' }}>
            Drop items here
          </Typography>
        )}
      </Box>
      {isDiffVisible && <FolderDiffViewer folderId={folder.id} onClose={() => setIsDiffVisible(false)} />}
    </Paper>
  );
}

export default function FolderBrowser() {
  const theme = useTheme();
  const [folders, setFolders] = useState<FullFolder[]>([]);
  const notification = useNotification();

  const fetchFolders = useCallback(() => {
    listFolders().then(setFolders).catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); });
  }, []);

  useEffect(fetchFolders, [fetchFolders]);

  const handleDropItem = useCallback(async (folderId: string, item: { id: string; type: 'query' | 'workbook' }) => {
    try {
      await addItemToFolder(folderId, item.id, item.type);
      fetchFolders();
    } catch (error) {
      notification.error(`Failed to add item to folder: ${(error as Error).message}`);
    }
  }, [fetchFolders]);

  return (
    <Box sx={{ p: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          Folders
        </Typography>
        <Button variant="outlined" size="small" onClick={fetchFolders}>
          Refresh
        </Button>
      </Box>
      {folders.map(folder => (
        <Folder key={folder.id} folder={folder} onDropItem={handleDropItem} />
      ))}
      {folders.length === 0 && (
        <Typography variant="body2" sx={{ color: 'text.secondary', fontStyle: 'italic' }}>
          No folders found.
        </Typography>
      )}
    </Box>
  );
}
