import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Box,
  Typography,
  Switch,
  Button,
} from '@mui/material';
import { usePersonalization } from '../contexts/PersonalizationContext';

interface PersonalSettingsDialogProps {
  open: boolean;
  onClose: () => void;
}

const PersonalSettingsDialog: React.FC<PersonalSettingsDialogProps> = ({ open, onClose }) => {
  const { menuDisplayMode, toggleMenuDisplayMode } = usePersonalization();

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Personal Settings</DialogTitle>
      <DialogContent dividers>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            py: 2,
            gap: 3,
          }}
        >
          <Box sx={{ flex: 1 }}>
            <Typography variant="body1" fontWeight="600" gutterBottom>
              Menu as Cards
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Display menu items as cards instead of dropdown menus
            </Typography>
          </Box>
          <Switch
            checked={menuDisplayMode === 'cards'}
            onChange={toggleMenuDisplayMode}
          />
        </Box>
      </DialogContent>
      <DialogActions sx={{ px: 2, py: 1.5 }}>
        <Button onClick={onClose} variant="contained">
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default PersonalSettingsDialog;
