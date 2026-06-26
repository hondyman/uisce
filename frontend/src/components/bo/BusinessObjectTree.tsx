import React from 'react';
import { Box, List, ListItem, ListItemText } from '@mui/material';

interface FieldNode {
  fullPath: string; // e.g., 'customer.name'
  label: string;
}

interface BusinessObjectTreeProps {
  fields: FieldNode[];
  highlightedFields?: Set<string>;
}

const BusinessObjectTree: React.FC<BusinessObjectTreeProps> = ({ fields, highlightedFields }) => {
  return (
    <Box>
      <List>
        {fields.map(f => (
          <ListItem key={f.fullPath} sx={highlightedFields && highlightedFields.has(f.fullPath) ? { backgroundColor: 'rgba(255,215,0,0.15)', borderLeft: '3px solid #f5c518' } : {}}>
            <ListItemText primary={f.label} secondary={f.fullPath} />
          </ListItem>
        ))}
      </List>
    </Box>
  );
};

export default BusinessObjectTree;
