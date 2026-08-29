import React from 'react';
import { CoreIcon, CustomIcon } from '../common/CoreCustomIcons';

interface CoreFieldBadgeProps {
  isCore?: boolean;
  size?: number | string;
  sx?: object;
}

export const CoreFieldBadge: React.FC<CoreFieldBadgeProps> = ({ isCore, size = 14, sx = {} }) => {
  if (isCore === undefined || isCore === null) return null;
  return isCore ? (
    <CoreIcon fontSize={size} sx={{ verticalAlign: 'middle', ...sx }} />
  ) : (
    <CustomIcon fontSize={size} sx={{ verticalAlign: 'middle', ...sx }} />
  );
};

export default CoreFieldBadge;
