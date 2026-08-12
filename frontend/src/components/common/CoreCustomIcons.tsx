import React from 'react';
import { Tooltip, useTheme, Chip, Stack } from '@mui/material';
import WaterDropIcon from '@mui/icons-material/WaterDrop';
import BuildIcon from '@mui/icons-material/Build';
import { deriveViewFlags } from '../../utils/viewFlags';

export const CORE_COLOR = 'success.main';
export const CUSTOM_COLOR = 'secondary.main';

interface CoreIconProps {
  sx?: object;
  fontSize?: 'small' | 'medium' | 'large' | 'inherit';
}

export const CoreIcon: React.FC<CoreIconProps> = ({ sx = {}, fontSize = 'small' }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const color = isDark ? theme.palette.primary.light : theme.palette.info.main;

  return (
    <Tooltip title="Core" arrow placement="top">
      <WaterDropIcon
        sx={{
          color,
          fontSize,
          ...sx,
        }}
      />
    </Tooltip>
  );
};

interface CustomIconProps {
  sx?: object;
  fontSize?: 'small' | 'medium' | 'large' | 'inherit';
}

export const CustomIcon: React.FC<CustomIconProps> = ({ sx = {}, fontSize = 'small' }) => {
  return (
    <Tooltip title="Custom" arrow placement="top">
      <BuildIcon
        sx={{
          color: 'secondary.main',
          fontSize,
          ...sx,
        }}
      />
    </Tooltip>
  );
};

export function isCoreItemKind(item: unknown): boolean {
  const { isCore } = deriveViewFlags(item as any);
  return isCore;
}

export function isCustomItemKind(item: unknown): boolean {
  const { isCustom } = deriveViewFlags(item as any);
  return isCustom;
}

interface CoreCustomBadgeProps {
  item: unknown;
  label?: string;
  variant?: 'chip' | 'icon' | 'chip-icon';
  size?: 'small' | 'medium';
}

export const CoreCustomBadge: React.FC<CoreCustomBadgeProps> = ({
  item,
  label,
  variant = 'chip-icon',
  size = 'small',
}) => {
  const { isCore, isCustom } = deriveViewFlags(item as any);

  if (!isCore && !isCustom) return null;

  const isCoreKind = isCore;

  if (variant === 'icon') {
    return isCoreKind ? <CoreIcon fontSize={size} /> : <CustomIcon fontSize={size} />;
  }

  const color = isCoreKind ? CORE_COLOR : CUSTOM_COLOR;
  const displayLabel = label ?? (isCoreKind ? 'Core' : 'Custom');

  if (variant === 'chip') {
    return (
      <Chip
        label={displayLabel}
        size={size}
        variant="outlined"
        sx={{
          color,
          borderColor: color,
          backgroundColor: 'transparent',
        }}
      />
    );
  }

  return (
    <Chip
      icon={isCoreKind ? <WaterDropIcon sx={{ fontSize: size === 'small' ? 16 : 20, color }} /> : <BuildIcon sx={{ fontSize: size === 'small' ? 16 : 20, color }} />}
      label={displayLabel}
      size={size}
      variant="outlined"
      sx={{
        color,
        borderColor: color,
        backgroundColor: 'transparent',
      }}
    />
  );
};

interface RenderCoreCustomIconsProps {
  option?: unknown;
  variant?: 'chip' | 'icon' | 'chip-icon';
  size?: 'small' | 'medium';
}

export function renderCoreCustomIcons(props?: RenderCoreCustomIconsProps) {
  const { option, size = 'small' } = props ?? {};

  if (!option) return null;

  const isCore = Boolean(
    (option as any).isCore ?? (option as any).is_core
  );
  const isCustom = Boolean(
    (option as any).isCustom ?? (option as any).is_custom
  );

  const chips: React.ReactNode[] = [];

  if (isCore) {
    chips.push(
      <Chip
        key="core"
        icon={<WaterDropIcon sx={{ fontSize: size === 'small' ? 16 : 20, color: CORE_COLOR }} />}
        label="Core"
        size={size}
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
        icon={<BuildIcon sx={{ fontSize: size === 'small' ? 16 : 20, color: CUSTOM_COLOR }} />}
        label="Custom"
        size={size}
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

export default renderCoreCustomIcons;
