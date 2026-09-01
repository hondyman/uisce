import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Stack,
  Typography,
  Switch,
  FormControlLabel,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Alert,
  Chip,
  Box,
} from '@mui/material';
import {
  Share as ShareIcon,
  Delete as DeleteIcon,
  Security as SecurityIcon,
  Person as PersonIcon,
} from '@mui/icons-material';
import type { ShareConfig } from '../types/dataExplorerTypes';
import { EXPLORER_BORDER } from '../types/dataExplorerTypes';

interface ShareQueryModalProps {
  open: boolean;
  onClose: () => void;
  queryName: string;
  onShare: (config: ShareConfig) => void;
}

export const ShareQueryModal: React.FC<ShareQueryModalProps> = ({
  open,
  onClose,
  queryName,
  onShare,
}) => {
  const [recipient, setRecipient] = useState('');
  const [permission, setPermission] = useState<'view' | 'edit' | 'admin'>('view');
  const [watermark, setWatermark] = useState(true);
  const [sharedList, setSharedList] = useState<ShareConfig['sharedWith']>([
    {
      recipientId: 'team_portfolio_mgmt',
      recipientName: 'Portfolio Management Team',
      permission: 'view',
      watermark: true,
    },
  ]);
  const [success, setSuccess] = useState(false);

  const handleAddRecipient = () => {
    if (!recipient.trim()) return;
    const item = {
      recipientId: recipient.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_'),
      recipientName: recipient.trim(),
      permission,
      watermark,
    };
    setSharedList((prev) => [...prev, item]);
    setRecipient('');
  };

  const handleRemove = (id: string) => {
    setSharedList((prev) => prev.filter((item) => item.recipientId !== id));
  };

  const handleConfirmShare = () => {
    onShare({ sharedWith: sharedList });
    setSuccess(true);
    setTimeout(() => {
      setSuccess(false);
      onClose();
    }, 1000);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, fontWeight: 700 }}>
        <ShareIcon sx={{ color: '#0D9488' }} />
        Share Query & Access Control
      </DialogTitle>
      <DialogContent dividers sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
        {success && <Alert severity="success">Permissions and shared links updated successfully.</Alert>}

        <Typography variant="body2" color="text.secondary">
          Share <strong>{queryName}</strong> with users, departments, or role groups under GSIFI tenant isolation.
        </Typography>

        <Stack direction="row" spacing={1.5} alignItems="center">
          <TextField
            label="User Email, Role or Department"
            placeholder="e.g. risk-team@firm.com or compliance"
            value={recipient}
            onChange={(e) => setRecipient(e.target.value)}
            fullWidth
            size="small"
          />
          <FormControl size="small" sx={{ width: 120 }}>
            <InputLabel>Role</InputLabel>
            <Select
              value={permission}
              label="Role"
              onChange={(e) => setPermission(e.target.value as any)}
            >
              <MenuItem value="view">View</MenuItem>
              <MenuItem value="edit">Edit</MenuItem>
              <MenuItem value="admin">Admin</MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="contained"
            onClick={handleAddRecipient}
            disabled={!recipient.trim()}
            sx={{
              bgcolor: '#0D9488',
              color: '#FFF',
              textTransform: 'none',
              fontWeight: 700,
              flexShrink: 0,
            }}
          >
            Add
          </Button>
        </Stack>

        <FormControlLabel
          control={<Switch checked={watermark} onChange={(e) => setWatermark(e.target.checked)} color="primary" />}
          label={
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                Enforce Confidential GSIFI Watermarking
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Watermarks viewer identity and timestamp onto all visual and exported outputs
              </Typography>
            </Box>
          }
        />

        <Box>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#64748B', mb: 1, display: 'block' }}>
            ACTIVE SHARED RECIPIENTS ({sharedList.length})
          </Typography>
          <List dense sx={{ bgcolor: '#F8FAFC', borderRadius: 1.5, border: `1px solid ${EXPLORER_BORDER}` }}>
            {sharedList.map((item) => (
              <ListItem key={item.recipientId}>
                <PersonIcon sx={{ fontSize: 18, color: '#0D9488', mr: 1 }} />
                <ListItemText
                  primary={item.recipientName}
                  secondary={`Role: ${item.permission.toUpperCase()} · ${item.watermark ? 'Watermarked' : 'Standard'}`}
                  primaryTypographyProps={{ fontSize: '0.85rem', fontWeight: 600 }}
                  secondaryTypographyProps={{ fontSize: '0.75rem' }}
                />
                <ListItemSecondaryAction>
                  <IconButton size="small" onClick={() => handleRemove(item.recipientId)}>
                    <DeleteIcon sx={{ fontSize: 16, color: '#EF4444' }} />
                  </IconButton>
                </ListItemSecondaryAction>
              </ListItem>
            ))}
          </List>
        </Box>
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 1.5 }}>
        <Button onClick={onClose} sx={{ textTransform: 'none' }}>
          Cancel
        </Button>
        <Button
          onClick={handleConfirmShare}
          variant="contained"
          sx={{
            bgcolor: '#0D9488',
            color: '#FFF',
            textTransform: 'none',
            fontWeight: 700,
            '&:hover': { bgcolor: '#0F766E' },
          }}
        >
          Save Sharing Settings
        </Button>
      </DialogActions>
    </Dialog>
  );
};
