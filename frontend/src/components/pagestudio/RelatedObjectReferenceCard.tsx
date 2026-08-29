import React from 'react';
import { Paper, Stack, Typography } from '@mui/material';
import VisibilityIcon from '@mui/icons-material/Visibility';
import type { RelationshipResult } from './pageStudioTypes';

export interface RelatedObjectReferenceCardProps {
  relationship: RelationshipResult;
}

// A to-one (1:1 / M:1) related object is never authored onto the canvas — it's always shown as a
// read-only reference, automatically, for every to-one relationship the root BO has. Live-fetching
// the actual referenced record is follow-up work; today this is an informational placeholder.
export const RelatedObjectReferenceCard: React.FC<RelatedObjectReferenceCardProps> = ({ relationship }) => {
  return (
    <Paper variant="outlined" sx={{ p: 1.25, mb: 1, bgcolor: 'rgba(255,255,255,0.02)' }}>
      <Stack direction="row" alignItems="center" gap={1}>
        <VisibilityIcon fontSize="small" color="disabled" />
        <Stack sx={{ minWidth: 0 }}>
          <Typography variant="body2" fontWeight={700} noWrap>{relationship.relatedObjectName}</Typography>
          <Typography variant="caption" color="text.secondary" noWrap display="block">
            {relationship.cardinality} reference · read-only
          </Typography>
        </Stack>
      </Stack>
    </Paper>
  );
};

export default RelatedObjectReferenceCard;
