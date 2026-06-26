import React from 'react';
import { Chip, ChipProps } from '@mui/material';

interface DataTypeChipProps extends Omit<ChipProps, 'color'> {
  type?: string;
}

export const DataTypeChip: React.FC<DataTypeChipProps> = ({ type = 'string', sx, ...props }) => {
  const normalizedType = (type || 'string').toLowerCase();
  
  let styles = {
    bgcolor: 'success.50', 
    color: 'success.700', 
    borderColor: 'success.200'
  };

  if (/int|double|float|decimal|number/.test(normalizedType)) {
    styles = { bgcolor: 'primary.50', color: 'primary.700', borderColor: 'primary.200' };
  } else if (/bool|boolean/.test(normalizedType)) {
    styles = { bgcolor: 'secondary.50', color: 'secondary.700', borderColor: 'secondary.200' };
  } else if (/date|time|timestamp/.test(normalizedType)) {
    styles = { bgcolor: 'warning.50', color: 'warning.800', borderColor: 'warning.200' };
  }

  return (
    <Chip
      label={type}
      size="small"
      variant="outlined"
      sx={{
        height: 22,
        fontSize: '0.7rem',
        fontWeight: 500,
        border: '1px solid',
        ...styles,
        ...sx,
      }}
      {...props}
    />
  );
};
