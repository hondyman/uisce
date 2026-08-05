import React from 'react';
import { Tooltip, useTheme } from '@mui/material';
import WaterDropIcon from '@mui/icons-material/WaterDrop';

interface CoreItemIconProps {
  sx?: object;
  fontSize?: 'small' | 'medium' | 'large' | 'inherit';
}

export const CoreItemIcon: React.FC<CoreItemIconProps> = ({ sx = {}, fontSize = 'small' }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const color = isDark ? theme.palette.primary.light : theme.palette.info.main;

  return (
    <Tooltip title="Core" arrow placement="top">
      <WaterDropIcon
        sx={{
          color,
          fontSize,
          ...sx,
        }}
      />
    </Tooltip>
  );
};

export default CoreItemIcon;
