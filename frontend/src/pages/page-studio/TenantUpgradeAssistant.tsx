import React, { useState } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, Typography, LinearProgress } from '@mui/material';

interface TenantUpgradeAssistantProps {
  open: boolean;
  onClose: () => void;
  tenantId: string;
}

export const TenantUpgradeAssistant: React.FC<TenantUpgradeAssistantProps> = ({ open, onClose, tenantId }) => {
  const [progress, setProgress] = useState(0);

  const handleUpgrade = async () => {
    for (let i = 0; i <= 100; i += 10) {
      await new Promise(r => setTimeout(r, 200));
      setProgress(i);
    }
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Tenant Upgrade Assistant</DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 2 }}>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Upgrading tenant: {tenantId}
          </Typography>
          <LinearProgress variant="determinate" value={progress} sx={{ mb: 2 }} />
          <Typography variant="caption" color="text.secondary">
            Running schema migrations and applying defaults...
          </Typography>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={progress < 100}>Cancel</Button>
        <Button variant="contained" onClick={handleUpgrade}>Start Upgrade</Button>
      </DialogActions>
    </Dialog>
  );
};
