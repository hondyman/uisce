import React from 'react';
import { Box, Typography, Chip } from '@mui/material';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { useDroppable } from '@dnd-kit/core';

interface FilterDropZoneProps {
  selectedBO: any;
  onSwitchToFields: () => void;
}

const FilterDropZone: React.FC<FilterDropZoneProps> = ({ selectedBO, onSwitchToFields }) => {
  const { setNodeRef, isOver } = useDroppable({
    id: 'filter-drop-zone',
  });

  return (
    <Box
      ref={setNodeRef}
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <Box sx={{ px: 1.5, pt: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
          <FilterAltIcon sx={{ fontSize: 18, color: 'primary.main' }} />
          <Typography variant="subtitle2" fontWeight={700} sx={{ fontSize: '0.8rem' }}>
            Filter Builder
          </Typography>
        </Box>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.7rem', lineHeight: 1.4 }}>
          Drag a field from the BO Fields palette and drop it here to create a filter.
        </Typography>
      </Box>

      <Box
        sx={{
          mx: 1.5,
          flex: 1,
          border: '2px dashed',
          borderColor: isOver ? 'primary.main' : 'divider',
          bgcolor: isOver ? 'action.hover' : 'transparent',
          borderRadius: 2,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 1.5,
          p: 2,
          transition: 'all 0.15s',
        }}
      >
        <FilterAltIcon sx={{ fontSize: 32, color: isOver ? 'primary.main' : 'text.disabled' }} />
        <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', fontSize: '0.75rem' }}>
          {isOver ? 'Drop to add filter' : 'Drag field here'}
        </Typography>
      </Box>

      {selectedBO?.subtypes && Object.keys(selectedBO.subtypes).length > 0 && (
        <Box sx={{ px: 1.5 }}>
          <Typography variant="caption" fontWeight={700} sx={{ textTransform: 'uppercase', letterSpacing: 1, color: 'text.secondary', fontSize: '0.65rem' }}>
            Subtype Fields
          </Typography>
          <Box sx={{ mt: 0.5, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
            {Object.entries(selectedBO.subtypes).map(([key, subtype]: [string, any]) => (
              <Chip
                key={key}
                label={subtype.displayName || key}
                size="small"
                sx={{ height: 20, fontSize: '0.65rem' }}
                onClick={onSwitchToFields}
              />
            ))}
          </Box>
        </Box>
      )}
    </Box>
  );
};

export default FilterDropZone;
