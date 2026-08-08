import { FormControl, Select, MenuItem, SelectChangeEvent, CircularProgress } from '@mui/material';
import { useAccess } from '../contexts/AccessContext';

export function TenantSwitcher() {
  const { accessibleTenants, currentTenant, setTenantScope, isLoading } = useAccess();

  const handleChange = (event: SelectChangeEvent) => {
    const tenantId = event.target.value;
    const tenant = accessibleTenants.find(t => t.id === tenantId);
    if (tenant) {
      setTenantScope(tenant);
    }
  };

  if (isLoading) {
    return (
      <FormControl size="small" sx={{ minWidth: 200 }}>
        <CircularProgress size={20} sx={{ mx: 2 }} />
      </FormControl>
    );
  }

  return (
    <FormControl size="small" sx={{ minWidth: 200 }}>
      <Select
        value={currentTenant?.id || ''}
        onChange={handleChange}
        displayEmpty
      >
        {accessibleTenants.map((t) => (
          <MenuItem key={t.id} value={t.id}>
            {t.display_name || t.name}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}
