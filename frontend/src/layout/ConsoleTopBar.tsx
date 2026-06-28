import { Box, Stack, AppBar, Toolbar } from '@mui/material';
import { GlobalSearch } from './GlobalSearch';
import { TenantSwitcher } from './TenantSwitcher';
import { ActiveSessionsIndicator } from '../components/admin/ActiveSessionsIndicator';
import { useAuth } from '../contexts/AuthContext';
import { useImpersonation } from '../contexts/ImpersonationContext';
import { useState, useCallback } from 'react';
import { ImpersonationTenantPicker } from '../components/admin/ImpersonationTenantPicker';
import type { ActiveImpersonationSession } from '../contexts/ImpersonationContext';

export function ConsoleTopBar() {
  const { token: adminToken } = useAuth();
  const { exitImpersonation, recentSessions, clearRecentSessions } = useImpersonation();
  const [pickerOpen, setPickerOpen] = useState(false);

  // Switching to a different active session: exit current, then open the picker
  // pre-populated with that tenant. The actual assume happens once the admin
  // re-confirms through the modal flow.
  const handleSwitchToTenant = useCallback(
    async (_s: ActiveImpersonationSession) => {
      await exitImpersonation();
      setPickerOpen(true);
    },
    [exitImpersonation],
  );

  return (
    <AppBar position="static" sx={{ backgroundColor: 'white', color: 'black' }}>
      <Toolbar>
        <Box sx={{ flexGrow: 1 }} />
        <Stack direction="row" spacing={2} alignItems="center">
          <ActiveSessionsIndicator onSwitchToTenant={handleSwitchToTenant} />
          <GlobalSearch />
          <TenantSwitcher />
        </Stack>
      </Toolbar>

      {/* Mount the picker so the topbar can launch it directly when the admin
          clicks "Switch" on an active session. The picker is otherwise
          opened from the TenantsPage (which also wires scope handoff). */}
      {adminToken && (
        <ImpersonationTenantPicker
          open={pickerOpen}
          onClose={() => setPickerOpen(false)}
          adminToken={adminToken}
          recentSessions={recentSessions}
          onClearRecentSessions={clearRecentSessions}
          onSelect={() => setPickerOpen(false)}
          initialTenant={null}
        />
      )}
    </AppBar>
  );
}
