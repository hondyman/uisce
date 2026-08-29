import React, { useState } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, TextField, Avatar, Stack } from '@mui/material';

interface TenantBrandingEditorProps {
  open: boolean;
  onClose: () => void;
  tenantId: string;
}

export const TenantBrandingEditor: React.FC<TenantBrandingEditorProps> = ({ open, onClose, tenantId }) => {
  const [logo, setLogo] = useState('');
  const [primaryColor, setPrimaryColor] = useState('#1976d2');

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Tenant Branding</DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Logo URL"
            fullWidth
            value={logo}
            onChange={(e) => setLogo(e.target.value)}
          />
          <TextField
            label="Primary Color"
            type="color"
            fullWidth
            value={primaryColor}
            onChange={(e) => setPrimaryColor(e.target.value)}
            InputLabelProps={{ shrink: true }}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={onClose}>Save</Button>
      </DialogActions>
    </Dialog>
  );
};
