import React, { ReactNode } from 'react';
import { Box, Tooltip } from '@mui/material';
import { useCanEditCoreItem } from '../../hooks/useCanEditCoreItem';
import { CoreItemIcon } from './CoreItemIcon';

interface CoreItemGuardProps {
  item: { tenant_id?: string | null; isCore?: boolean; is_core?: boolean } | null | undefined;
  children: ReactNode;
  showIcon?: boolean;
  disableReason?: string | null;
}

export const CoreItemGuard: React.FC<CoreItemGuardProps> = ({
  item,
  children,
  showIcon = true,
  disableReason,
}) => {
  const { isCore, canEdit, disabledReason: hookReason } = useCanEditCoreItem(item);

  if (!isCore) {
    return <>{children}</>;
  }

  const reason = disableReason ?? hookReason;

  if (canEdit) {
    return (
      <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
        {children}
        {showIcon && <CoreItemIcon />}
      </Box>
    );
  }

  return (
    <Tooltip title={reason ?? 'Core item — editing disabled'} arrow placement="top">
      <Box
        sx={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 0.5,
          opacity: 0.6,
          cursor: 'not-allowed',
        }}
      >
        {children}
        {showIcon && <CoreItemIcon />}
      </Box>
    </Tooltip>
  );
};

export default CoreItemGuard;
