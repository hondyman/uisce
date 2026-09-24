import React from 'react';
import { Box, Stack, Typography, IconButton, Tooltip, Divider } from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import AddIcon from '@mui/icons-material/Add';
import LinkIcon from '@mui/icons-material/Link';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { isToMany, type RelationshipResult } from './pageStudioTypes';

/** One draggable "add related object" tile - relies on an ancestor DndContext (mounted in PageStudioPalette). */
const RelatedObjectTile: React.FC<{ rel: RelationshipResult; onAdd: () => void }> = ({ rel, onAdd }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `related-${rel.id}`,
    data: { kind: 'relatedobject', relationship: rel },
  });
  return (
    <Box
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        p: 1,
        bgcolor: 'rgba(255,255,255,0.03)',
        border: '1px solid rgba(255,255,255,0.08)',
        borderRadius: 1,
        cursor: 'grab',
        opacity: isDragging ? 0.4 : 1,
        '&:hover': { borderColor: 'primary.main', bgcolor: 'rgba(99,102,241,0.08)' },
      }}
    >
      <Stack direction="row" alignItems="center" gap={1} sx={{ minWidth: 0 }}>
        <LinkIcon fontSize="small" color="action" />
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="body2" fontWeight={700} noWrap>{rel.relatedObjectName}</Typography>
          <Typography variant="caption" color="text.secondary" noWrap display="block">
            {rel.cardinality} relationship
          </Typography>
        </Box>
      </Stack>
      <Tooltip title={`Add ${rel.relatedObjectName}`}>
        <IconButton size="small" aria-label={`Add ${rel.relatedObjectName}`} onClick={onAdd}>
          <AddIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Box>
  );
};

export interface RelatedObjectsPaletteProps {
  relationships: RelationshipResult[] | null;
  onAddRelatedObject: (relationship: RelationshipResult) => void;
}

export const RelatedObjectsPalette: React.FC<RelatedObjectsPaletteProps> = ({
  relationships,
  onAddRelatedObject,
}) => {
  if (!relationships) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
        <Typography variant="caption">Loading relationships…</Typography>
      </Box>
    );
  }

  const toMany = relationships.filter((r) => isToMany(r.cardinality));
  const toOne = relationships.filter((r) => !isToMany(r.cardinality));

  if (relationships.length === 0) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
        <Typography variant="caption">
          No relationships found for this Business Object yet.
        </Typography>
      </Box>
    );
  }

  return (
    <Stack spacing={1} sx={{ p: 1.5 }}>
      {toMany.length > 0 && (
        <>
          <Typography variant="overline" color="text.secondary" sx={{ px: 0.5 }}>
            Drag onto canvas
          </Typography>
          {toMany.map((rel) => (
            <RelatedObjectTile key={rel.id} rel={rel} onAdd={() => onAddRelatedObject(rel)} />
          ))}
        </>
      )}

      {toOne.length > 0 && (
        <>
          {toMany.length > 0 && <Divider sx={{ my: 1 }} />}
          <Typography variant="overline" color="text.secondary" sx={{ px: 0.5 }}>
            Reference cards (shown automatically)
          </Typography>
          {toOne.map((rel) => (
            <Box
              key={rel.id}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1,
                p: 1,
                bgcolor: 'rgba(255,255,255,0.02)',
                border: '1px dashed rgba(255,255,255,0.08)',
                borderRadius: 1,
              }}
            >
              <VisibilityIcon fontSize="small" color="disabled" />
              <Box sx={{ minWidth: 0 }}>
                <Typography variant="body2" fontWeight={600} noWrap>{rel.relatedObjectName}</Typography>
                <Typography variant="caption" color="text.secondary" noWrap display="block">
                  {rel.cardinality} · appears at the top of every tab
                </Typography>
              </Box>
            </Box>
          ))}
        </>
      )}
    </Stack>
  );
};

export default RelatedObjectsPalette;
