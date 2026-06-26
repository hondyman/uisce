import type { ReactNode } from 'react';
import { Chip, Stack } from '@mui/material';

// Centralized colors for Core and Custom chips. Use MUI palette names so
// colors remain theme-aware and consistent across the app.
const CORE_COLOR = 'success.main';
const CUSTOM_COLOR = 'secondary.main';

export function renderCoreCustomChips(option?: any) {
  if (!option) return null;
  const isCore = Boolean(option.isCore ?? option.is_core);
  const isCustom = Boolean(option.isCustom ?? option.is_custom);
  const chips: ReactNode[] = [];
  if (isCore) {
    chips.push(
      <Chip
        key="core"
        label="Core"
        size="small"
        variant="outlined"
        sx={{
          color: CORE_COLOR,
          borderColor: CORE_COLOR,
          backgroundColor: 'transparent',
        }}
      />
    );
  }
  if (isCustom) {
    chips.push(
      <Chip
        key="custom"
        label="Custom"
        size="small"
        variant="outlined"
        sx={{
          color: CUSTOM_COLOR,
          borderColor: CUSTOM_COLOR,
          backgroundColor: 'transparent',
        }}
      />
    );
  }
  if (chips.length === 0) return null;
  return <Stack direction="row" spacing={0.5} sx={{ ml: 1, flexWrap: 'nowrap' }}>{chips}</Stack>;
}

export default renderCoreCustomChips;
