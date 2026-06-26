import { Box, Stack, AppBar, Toolbar } from '@mui/material';
import { GlobalSearch } from './GlobalSearch';
import { TenantSwitcher } from './TenantSwitcher';

export function ConsoleTopBar() {
  return (
    <AppBar position="static" sx={{ backgroundColor: 'white', color: 'black' }}>
      <Toolbar>
        <Box sx={{ flexGrow: 1 }} />
        <Stack direction="row" spacing={2} alignItems="center">
          <GlobalSearch />
          <TenantSwitcher />
        </Stack>
      </Toolbar>
    </AppBar>
  );
}
