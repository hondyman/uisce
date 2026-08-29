import React from 'react';
import { Box, Typography, Paper, Divider } from '@mui/material';

interface PropertiesPanelProps {
  selectedNode: any;
  onUpdate: (updates: Record<string, unknown>) => void;
}

const PropertiesPanel: React.FC<PropertiesPanelProps> = ({ selectedNode, onUpdate }) => {
  if (!selectedNode) {
    return (
      <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper' }}>
        <Typography variant="body2" color="text.secondary">
          Select a component to edit its properties
        </Typography>
      </Paper>
    );
  }

  return (
    <Paper elevation={0} sx={{ p: 2, height: '100%', bgcolor: 'background.paper' }}>
      <Typography variant="subtitle2" sx={{ mb: 2 }}>
        Properties: {selectedNode.componentId || 'Unknown'}
      </Typography>
      <Divider sx={{ mb: 2 }} />
      <Typography variant="body2" color="text.secondary">
        Component ID: {selectedNode.id}
      </Typography>
    </Paper>
  );
};

export default PropertiesPanel;
