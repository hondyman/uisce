import type { ReactNode } from 'react';
import { Chip, Stack } from '@mui/material';
import WaterDropIcon from '@mui/icons-material/WaterDrop';
import BuildIcon from '@mui/icons-material/Build';
import { CORE_COLOR, CUSTOM_COLOR } from './CoreCustomIcons';

export { CORE_COLOR, CUSTOM_COLOR };

export function renderCoreCustomChips(option?: any) {
  if (!option) return null;
  const isCore = Boolean(option.isCore ?? option.is_core);
  const isCustom = Boolean(option.isCustom ?? option.is_custom);
  const chips: ReactNode[] = [];
  if (isCore) {
    chips.push(
      <Chip
        key="core"
        icon={<WaterDropIcon sx={{ fontSize: 16, color: CORE_COLOR }} />}
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
        icon={<BuildIcon sx={{ fontSize: 16, color: CUSTOM_COLOR }} />}
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
