import React from 'react';
import { Chip, Tooltip, Box, Stack, Typography } from '@mui/material';
import {
  Layers as InheritedIcon,
  Star as CoreIcon,
  Extension as CustomIcon,
} from '@mui/icons-material';

// ============================================================================
// FIELD TYPE BADGES
// ============================================================================

export interface FieldTypeBadgeProps {
  isCore?: boolean;
  isCustom?: boolean;
  isInherited?: boolean;
  inheritedFrom?: string;
  size?: 'small' | 'medium';
  showLabel?: boolean;
}

export const FieldTypeBadge: React.FC<FieldTypeBadgeProps> = ({
  isCore = false,
  isCustom = false,
  isInherited = false,
  inheritedFrom,
  size = 'small',
  showLabel = true,
}) => {
  if (isInherited && inheritedFrom) {
    return (
      <Tooltip title={`Inherited from ${inheritedFrom}`}>
        <Chip
          icon={<InheritedIcon />}
          label={showLabel ? 'Inherited' : undefined}
          size={size}
          sx={{
            bgcolor: '#e3f2fd',
            color: '#1976d2',
            fontWeight: 500,
            fontSize: size === 'small' ? '0.7rem' : '0.8rem',
            height: size === 'small' ? 20 : 24,
          }}
        />
      </Tooltip>
    );
  }

  if (isCustom) {
    return (
      <Tooltip title="Custom field (tenant-specific customization)">
        <Chip
          icon={<CustomIcon />}
          label={showLabel ? 'Custom' : undefined}
          size={size}
          sx={{
            bgcolor: '#fff3e0',
            color: '#f57c00',
            fontWeight: 500,
            fontSize: size === 'small' ? '0.7rem' : '0.8rem',
            height: size === 'small' ? 20 : 24,
          }}
        />
      </Tooltip>
    );
  }

  if (isCore) {
    return (
      <Tooltip title="Core field (platform-provided)">
        <Chip
          icon={<CoreIcon />}
          label={showLabel ? 'Core' : undefined}
          size={size}
          sx={{
            bgcolor: '#f3e5f5',
            color: '#7b1fa2',
            fontWeight: 500,
            fontSize: size === 'small' ? '0.7rem' : '0.8rem',
            height: size === 'small' ? 20 : 24,
          }}
        />
      </Tooltip>
    );
  }

  return null;
};

// ============================================================================
// FIELD ROW WITH BADGES
// ============================================================================

export interface FieldRowBadgesProps {
  field: {
    is_core?: boolean;
    is_custom?: boolean;
    is_inherited?: boolean;
    inherited_from_key?: string;
    is_required?: boolean;
  };
  showRequired?: boolean;
}

export const FieldRowBadges: React.FC<FieldRowBadgesProps> = ({
  field,
  showRequired = true,
}) => {
  return (
    <Stack direction="row" spacing={0.5} alignItems="center">
      <FieldTypeBadge
        isCore={field.is_core}
        isCustom={field.is_custom}
        isInherited={field.is_inherited}
        inheritedFrom={field.inherited_from_key}
        size="small"
        showLabel={false}
      />
      {showRequired && field.is_required && (
        <Chip
          label="Required"
          size="small"
          color="error"
          variant="outlined"
          sx={{
            height: 20,
            fontSize: '0.65rem',
            fontWeight: 500,
          }}
        />
      )}
    </Stack>
  );
};

// ============================================================================
// FIELD LEGEND (for showing what badges mean)
// ============================================================================

export const FieldLegend: React.FC = () => {
  return (
    <Box
      sx={{
        p: 2,
        bgcolor: 'grey.50',
        borderRadius: 1,
        border: '1px solid',
        borderColor: 'divider',
      }}
    >
      <Typography variant="subtitle2" gutterBottom fontWeight="bold">
        Field Types
      </Typography>
      <Stack spacing={1}>
        <Stack direction="row" spacing={1} alignItems="center">
          <FieldTypeBadge isInherited inheritedFrom="Parent BO" />
          <Typography variant="body2" color="text.secondary">
            Inherited from parent business object
          </Typography>
        </Stack>
        <Stack direction="row" spacing={1} alignItems="center">
          <FieldTypeBadge isCore />
          <Typography variant="body2" color="text.secondary">
            Core field provided by the platform (subtype-specific)
          </Typography>
        </Stack>
        <Stack direction="row" spacing={1} alignItems="center">
          <FieldTypeBadge isCustom />
          <Typography variant="body2" color="text.secondary">
            Custom field added by tenant admin
          </Typography>
        </Stack>
      </Stack>
    </Box>
  );
};

export default FieldTypeBadge;
