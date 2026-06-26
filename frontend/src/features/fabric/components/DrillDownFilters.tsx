import type { FC } from 'react';
import {
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Checkbox,
  ListItemText,
  OutlinedInput,
  Typography,
} from '@mui/material';
import { useDrillDown } from '../../../contexts/DrillDownContext';

const SEVERITIES = ['breaking', 'medium', 'low'];

const SeverityFilter: FC = () => {
  const { filters, setFilters } = useDrillDown();

  const handleChange = (event: any) => {
    const {
      target: { value },
    } = event;
    setFilters((prev) => ({ ...prev, severity: typeof value === 'string' ? value.split(',') : value }));
  };

  return (
    <FormControl size="small" sx={{ m: 1, minWidth: 150 }}>
      <InputLabel>Severity</InputLabel>
      <Select
        multiple
        value={filters?.severity || []}
        onChange={handleChange}
        input={<OutlinedInput label="Severity" />}
        renderValue={(selected) => (selected as string[]).join(', ')}
      >
        {SEVERITIES.map((name) => (
          <MenuItem key={name} value={name}>
            <Checkbox checked={(filters?.severity || []).indexOf(name) > -1} />
            <ListItemText primary={name} />
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};

const DrillDownFilters: FC<{ context: string | null }> = ({ context }) => {
  if (!context) return null;

  const showSeverityFilter = ['historical', 'policy_compare'].includes(context);

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', p: 2, borderBottom: 1, borderColor: 'divider' }}>
      <Typography variant="subtitle2" sx={{ mr: 2 }}>Filters:</Typography>
      {showSeverityFilter && <SeverityFilter />}
      {/* Add other context-aware filters like <EnvironmentFilter /> here */}
    </Box>
  );
};

export default DrillDownFilters;