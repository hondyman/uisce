import React from 'react';
import { Box, Modal, Typography, Button, Paper } from '@mui/material';

export const PromotionImpactModal: React.FC<{ open: boolean; impact: any; onClose: () => void; onConfirm: () => void }> = ({ open, impact, onClose, onConfirm }) => {
  if (!open) return null;
  return (
    <Modal open={open} onClose={onClose}>
      <Box sx={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: 500 }}>
        <Paper sx={{ p: 3 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>Promotion Impact Summary</Typography>
          <ul>
            <li>Business Objects: {impact?.business_objects ?? 0}</li>
            <li>Fields: {impact?.fields ?? 0}</li>
            <li>Semantic Terms: {impact?.semantic_terms ?? 0}</li>
            <li>Related Rules: {impact?.related_rules ?? 0}</li>
            <li>Overrides: {impact?.overrides ?? 0}</li>
          </ul>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mt: 2 }}>
            <Button onClick={onClose}>Cancel</Button>
            <Button variant="contained" onClick={onConfirm}>Promote</Button>
          </Box>
        </Paper>
      </Box>
    </Modal>
  );
};

export default PromotionImpactModal;