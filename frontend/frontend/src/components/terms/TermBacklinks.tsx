import React from 'react';
import { useQuery } from 'react-query';
import { rulesApi } from '../../services/rulesApi';
import { Box, Typography, List, ListItem, ListItemText } from '@mui/material';

export const TermBacklinks: React.FC<{ termId: string }> = ({ termId }) => {
  const { data: rules } = useQuery(['termRules', termId], () => rulesApi.fetchRulesBySemanticTerm(termId));

  if (!rules || rules.length === 0) return null;

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 1 }}>Rules referencing this term</Typography>
      <List>
        {rules.map((r: any) => (
          <ListItem key={r.id} component="a" href={`/rules/${r.id}`} button>
            <ListItemText primary={r.name} secondary={r.description || ''} />
          </ListItem>
        ))}
      </List>
    </Box>
  );
};

export default TermBacklinks;