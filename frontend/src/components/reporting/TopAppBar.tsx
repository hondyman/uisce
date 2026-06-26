import type { FC, ReactNode } from 'react';
import { AppBar, Toolbar, Box, SxProps, Theme } from '@mui/material';

type Props = {
  children?: ReactNode;
  sx?: SxProps<Theme>;
};

const TopAppBar: FC<Props> = ({ children, sx }) => {
  return (
    <AppBar position="relative" sx={sx}>
      <Toolbar>
        <Box sx={{ display: 'flex', alignItems: 'center', width: '100%' }}>
          {children}
        </Box>
      </Toolbar>
    </AppBar>
  );
};

export default TopAppBar;
