import { FormControl, FormHelperText, MenuItem, Select, SelectChangeEvent, Tooltip } from '@mui/material';
import { useAccess } from '../contexts/AccessContext';
import { useRegions } from '../hooks/useRegions';
import { useSelectedRegion } from '../hooks/useSelectedRegion';
import { setSelectedRegion } from '../lib/region';

/**
 * Explicit region selection. Every request must state its region and the backend
 * refuses one that does not, so this shows exactly what is sent: the stored region,
 * or an unmistakable "Select region" state when there is none. It never pre-selects.
 */
export default function RegionPicker({ size = 'small' }: { size?: 'small' | 'medium' }) {
  const selected = useSelectedRegion();
  const { regions } = useRegions();
  const { currentTenant } = useAccess();

  // A region already in use (e.g. the tenant's own) must stay selectable even if
  // the lookup does not list it, or the Select would render an out-of-range value.
  const options = regions.some((r) => r.value === selected) || !selected
    ? regions
    : [...regions, { value: selected, label: selected, fromLookup: false }];
  const missing = !selected;

  return (
    <Tooltip
      title={
        missing
          ? 'No region selected: requests that need a region will be refused until you pick one.'
          : `Region ${selected}`
      }
    >
      <FormControl size={size} error={missing} sx={{ minWidth: 150 }}>
        <Select
          displayEmpty
          value={selected}
          disabled={!currentTenant}
          onChange={(e: SelectChangeEvent) => setSelectedRegion(e.target.value)}
          inputProps={{ 'aria-label': 'Region' }}
          renderValue={(v) => (v ? options.find((o) => o.value === v)?.label ?? v : 'Select region')}
        >
          {options.map((r) => (
            <MenuItem key={r.value} value={r.value}>
              {r.label}
            </MenuItem>
          ))}
        </Select>
        {missing && <FormHelperText sx={{ mx: 0 }}>Region required</FormHelperText>}
      </FormControl>
    </Tooltip>
  );
}
