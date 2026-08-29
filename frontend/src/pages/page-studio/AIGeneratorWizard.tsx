import React, { useState } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField, Box, Typography } from '@mui/material';

interface AIGeneratorWizardProps {
  open: boolean;
  onClose: () => void;
  onGenerate: (spec: { description: string; components: string[] }) => void;
}

export const AIGeneratorWizard: React.FC<AIGeneratorWizardProps> = ({ open, onClose, onGenerate }) => {
  const [description, setDescription] = useState('');
  const [components, setComponents] = useState('');

  const handleGenerate = () => {
    onGenerate({
      description,
      components: components.split(',').map(c => c.trim()).filter(Boolean),
    });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>AI Page Generator</DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Page Description"
            multiline
            rows={3}
            fullWidth
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Describe the page you want to generate..."
          />
          <TextField
            label="Components (comma-separated)"
            fullWidth
            value={components}
            onChange={(e) => setComponents(e.target.value)}
            placeholder="Button, Card, Table"
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={handleGenerate}>Generate</Button>
      </DialogActions>
    </Dialog>
  );
};
