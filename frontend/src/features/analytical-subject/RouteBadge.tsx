import React from 'react';
import { Box, Chip, Stack, Typography } from '@mui/material';
import type { CubeRouteBadge, ServedFrom } from './types';

export type RouteBadgeProps = {
  /** Route metadata from preview/execute (cubeHit / cubeMiss). */
  route?: CubeRouteBadge | null;
  /** When no cube routing ran at all (legacy BO-only without router). */
  emptyLabel?: string;
};

function tone(served: ServedFrom, stale?: boolean): {
  bg: string;
  border: string;
  text: string;
  label: string;
} {
  if (served === 'hot') {
    return {
      bg: stale ? 'rgba(245, 158, 11, 0.12)' : 'rgba(6, 182, 212, 0.12)',
      border: stale ? 'rgba(245, 158, 11, 0.45)' : 'rgba(6, 182, 212, 0.45)',
      text: stale ? '#fbbf24' : '#67e8f9',
      label: stale ? 'hot · stale' : 'hot',
    };
  }
  if (served === 'cold') {
    return {
      bg: 'rgba(139, 92, 246, 0.12)',
      border: 'rgba(139, 92, 246, 0.45)',
      text: '#c4b5fd',
      label: stale ? 'cold · stale' : 'cold',
    };
  }
  return {
    bg: 'rgba(16, 185, 129, 0.12)',
    border: 'rgba(16, 185, 129, 0.45)',
    text: '#6ee7b7',
    label: 'raw',
  };
}

/**
 * Dual-tier route badge for cube / BO acceleration. Replaces the coarse
 * OLAP/OLTP QueryEngineIndicator stub when cubeHit/cubeMiss is present.
 */
export const RouteBadge: React.FC<RouteBadgeProps> = ({
  route,
  emptyLabel = 'raw · base path',
}) => {
  const served: ServedFrom = route?.servedFrom
    ?? (route?.missReason || !route?.cubeId ? 'raw' : 'raw');
  const t = tone(served, route?.stale);
  const title = route?.cubeName || route?.cubeId;

  return (
    <Box
      sx={{
        borderRadius: 2,
        p: 1.5,
        border: `1px solid ${t.border}`,
        bgcolor: t.bg,
        color: t.text,
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
        fontSize: 12,
      }}
      data-testid="route-badge"
      data-served-from={served}
      data-stale={route?.stale ? 'true' : 'false'}
      data-miss={route?.missReason || ''}
    >
      <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={1}>
        <Stack direction="row" alignItems="center" spacing={1}>
          <Chip
            size="small"
            label={t.label}
            sx={{
              height: 22,
              fontWeight: 700,
              bgcolor: 'transparent',
              color: t.text,
              border: `1px solid ${t.border}`,
            }}
          />
          {title && (
            <Typography variant="caption" sx={{ color: t.text, opacity: 0.9 }}>
              {title}
              {route?.contractVersion ? ` · v${route.contractVersion}` : ''}
            </Typography>
          )}
        </Stack>
        {route?.materialization && (
          <Typography variant="caption" sx={{ opacity: 0.75, maxWidth: '45%', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {route.materialization}
          </Typography>
        )}
      </Stack>
      {route?.missReason && (
        <Typography variant="caption" sx={{ display: 'block', mt: 0.75, opacity: 0.85 }}>
          miss: {route.missReason}
        </Typography>
      )}
      {!route && (
        <Typography variant="caption" sx={{ display: 'block', opacity: 0.8 }}>
          {emptyLabel}
        </Typography>
      )}
    </Box>
  );
};

/** Build badge props from preview/execute wire fields. */
export function routeBadgeFromPreview(input: {
  cubeHit?: {
    cubeId?: string;
    cubeName?: string;
    materialization?: string;
    servedFrom?: string;
    contractVersion?: number;
    grain?: string[];
    stale?: boolean;
  } | null;
  cubeMiss?: string | null;
}): CubeRouteBadge | null {
  if (input.cubeHit) {
    const sf = (input.cubeHit.servedFrom || 'hot') as ServedFrom;
    return {
      servedFrom: sf === 'cold' || sf === 'hot' || sf === 'raw' ? sf : 'hot',
      stale: !!input.cubeHit.stale,
      cubeId: input.cubeHit.cubeId,
      cubeName: input.cubeHit.cubeName,
      materialization: input.cubeHit.materialization,
      contractVersion: input.cubeHit.contractVersion,
    };
  }
  if (input.cubeMiss) {
    return { servedFrom: 'raw', missReason: input.cubeMiss };
  }
  return null;
}

export default RouteBadge;
