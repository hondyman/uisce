import React, { useState } from 'react';
import { Box, Paper, Typography, Button, TextField, List, ListItem, ListItemText, Chip } from '@mui/material';

interface AITestGeneratorProps {
  componentId: string;
}

export const AITestGenerator: React.FC<AITestGeneratorProps> = ({ componentId }) => {
  const [testCases, setTestCases] = useState<{ name: string; status: string }[]>([]);

  const handleGenerate = () => {
    setTestCases([
      { name: `test-${componentId}-renders`, status: 'pending' },
      { name: `test-${componentId}-interaction`, status: 'pending' },
    ]);
  };

  return (
    <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper' }}>
      <Typography variant="subtitle2" sx={{ mb: 2 }}>AI Test Generator</Typography>
      <Button variant="outlined" size="small" onClick={handleGenerate} sx={{ mb: 2 }}>
        Generate Tests
      </Button>
      <List dense>
        {testCases.map((tc, i) => (
          <ListItem key={i}>
            <ListItemText primary={tc.name} />
            <Chip label={tc.status} size="small" color={tc.status === 'passing' ? 'success' : 'default'} />
          </ListItem>
        ))}
      </List>
    </Paper>
  );
};
