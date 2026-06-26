import { FormControl, Select, MenuItem, SelectChangeEvent } from '@mui/material';
import { useState, useEffect } from 'react';

export interface Tenant {
  id: string;
  name: string;
}

export function TenantSwitcher() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [currentTenant, setCurrentTenant] = useState<string>('');

  useEffect(() => {
    // Load from localStorage or context
    const saved = localStorage.getItem('selectedTenant');
    if (saved) setCurrentTenant(saved);

    // Mock tenants - in real app, fetch from API
    setTenants([
      { id: 'tenant-1', name: 'Acme Capital' },
      { id: 'tenant-2', name: 'BlackRock Test' },
      { id: 'tenant-3', name: 'State Street Alpha' },
    ]);

    if (saved) setCurrentTenant(saved);
    else setCurrentTenant('tenant-1');
  }, []);

  const handleChange = (event: SelectChangeEvent) => {
    const value = event.target.value;
    setCurrentTenant(value);
    localStorage.setItem('selectedTenant', value);
    // In a real app, this would trigger a context update or URL change
  };

  return (
    <FormControl size="small" sx={{ minWidth: 200 }}>
      <Select value={currentTenant} onChange={handleChange} displayEmpty>
        {tenants.map((t) => (
          <MenuItem key={t.id} value={t.id}>
            {t.name}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}
