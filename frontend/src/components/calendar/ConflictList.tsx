import React from 'react';
import { Box, Typography, Accordion, AccordionSummary, AccordionDetails, Chip, Alert } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { SyncConflict } from '../../hooks/useConflictResolution';
import { ConflictDetail } from './ConflictDetail';
import { ConflictResolution } from './ConflictResolution';

interface Props {
  conflicts: SyncConflict[];
  onResolve: (conflictId: string, strategy: string) => void;
  isResolving: boolean;
}

export const ConflictList: React.FC<Props> = ({ conflicts, onResolve, isResolving }) => {
  if (!conflicts || conflicts.length === 0) {
    return <Alert severity="success">No pending conflicts found.</Alert>;
  }

  return (
    <Box>
      <Typography variant="h6" gutterBottom>Pending Conflicts ({conflicts.length})</Typography>
      {conflicts.map((conflict) => (
        <Accordion key={conflict.id} sx={{ mb: 1 }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Box display="flex" alignItems="center" gap={2}>
              <Chip 
                size="small" 
                label={conflict.severity} 
                color={conflict.severity === 'critical' ? 'error' : (conflict.severity === 'warning' ? 'warning' : 'info')} 
              />
              <Typography variant="subtitle1">
                {conflict.internal_event_data?.title || conflict.google_event_data?.summary || 'Unknown Event'}
              </Typography>
              <Typography variant="caption" color="textSecondary">
                ({conflict.conflict_type})
              </Typography>
            </Box>
          </AccordionSummary>
          <AccordionDetails>
            <ConflictDetail conflict={conflict} />
            <ConflictResolution 
              conflictId={conflict.id} 
              onResolve={(strategy) => onResolve(conflict.id, strategy)} 
              isResolving={isResolving} 
            />
          </AccordionDetails>
        </Accordion>
      ))}
    </Box>
  );
};
