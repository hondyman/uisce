import type { FC } from 'react';
import { Box, Typography, Chip } from '@mui/material';
import { ViewColumn, Functions } from '@mui/icons-material';
import FilterAltIcon from '@mui/icons-material/FilterAlt';

interface FilterTotalsProps {
  dimensionCount: number;
  measureCount: number;
  typeFilter: 'all' | 'dimension' | 'measure';
  setTypeFilter: (v: 'all' | 'dimension' | 'measure') => void;
}

export const FilterTotals: FC<FilterTotalsProps> = ({ dimensionCount, measureCount, typeFilter, setTypeFilter }) => {
  return (
    <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'center' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <ViewColumn sx={{ color: 'success.main' }} fontSize="small" />
        <Typography variant="caption">Dimensions</Typography>
        <Chip
          label={dimensionCount}
          size="small"
          color={typeFilter === 'dimension' ? 'primary' : 'success'}
          variant={typeFilter === 'dimension' ? 'filled' : 'outlined'}
          icon={typeFilter === 'dimension' ? <FilterAltIcon fontSize="small" /> : undefined}
          sx={{
            ml: 0.5,
            cursor: 'pointer',
            boxShadow: typeFilter === 'dimension' ? '0 4px 12px rgba(25, 118, 210, 0.12)' : 'none',
            border: typeFilter === 'dimension' ? '1px solid rgba(25,118,210,0.15)' : undefined,
          }}
          onClick={() => setTypeFilter(typeFilter === 'dimension' ? 'all' : 'dimension')}
        />
      </Box>

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <Functions sx={{ color: 'info.main' }} fontSize="small" />
        <Typography variant="caption">Measures</Typography>
        <Chip
          label={measureCount}
          size="small"
          color={typeFilter === 'measure' ? 'primary' : 'info'}
          variant={typeFilter === 'measure' ? 'filled' : 'outlined'}
          icon={typeFilter === 'measure' ? <FilterAltIcon fontSize="small" /> : undefined}
          sx={{
            ml: 0.5,
            cursor: 'pointer',
            boxShadow: typeFilter === 'measure' ? '0 4px 12px rgba(25, 118, 210, 0.12)' : 'none',
            border: typeFilter === 'measure' ? '1px solid rgba(25,118,210,0.15)' : undefined,
          }}
          onClick={() => setTypeFilter(typeFilter === 'measure' ? 'all' : 'measure')}
        />
      </Box>
    </Box>
  );
};

export default FilterTotals;
