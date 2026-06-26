import React, { useState, useEffect } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField, Box } from '@mui/material';

interface Props {
  open: boolean;
  title?: string;
  label?: string;
  defaultValue?: string;
  ariaLabel?: string;
  onClose: () => void;
  onSubmit: (value: string) => void;
}

export default function TextPromptDialog({ open, title = 'Enter value', label = '', defaultValue = '', ariaLabel = 'text-prompt', onClose, onSubmit }: Props) {
  const [value, setValue] = useState(defaultValue);

  useEffect(() => setValue(defaultValue), [defaultValue]);

  return (
    <Dialog open={open} onClose={onClose} aria-labelledby="text-prompt-title">
      <DialogTitle id="text-prompt-title">{title}</DialogTitle>
      <DialogContent>
        <Box sx={{ width: 500 }}>
          <TextField
            autoFocus
            fullWidth
            label={label}
            inputProps={{ 'aria-label': ariaLabel }}
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={() => onSubmit(value)} variant="contained">OK</Button>
      </DialogActions>
    </Dialog>
  );
}
