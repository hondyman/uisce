import React from 'react';
import { Box, Typography, Breadcrumbs, Chip } from '@mui/material';
import { ChevronRight, Home, CornerUpLeft } from 'lucide-react';
import { ExplorerQueryDefinition } from '../types/explorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface DrilldownBreadcrumbsProps {
  history: ExplorerQueryDefinition[];
  currentQuery: ExplorerQueryDefinition;
  onNavigateBack: (historyIndex: number) => void;
}

export const DrilldownBreadcrumbs: React.FC<DrilldownBreadcrumbsProps> = ({
  history,
  currentQuery,
  onNavigateBack,
}) => {
  const theme = useExplorerTheme();

  if (history.length === 0) return null;

  return (
    <Box
      sx={{
        px: 2.5,
        py: 0.8,
        bgcolor: theme.background,
        borderBottom: `1px solid ${theme.border}`,
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
      }}
    >
      <Typography
        variant="caption"
        sx={{
          color: theme.textMuted,
          fontWeight: 800,
          display: 'flex',
          alignItems: 'center',
          gap: 0.5,
          fontSize: '0.68rem',
          letterSpacing: 0.5,
        }}
      >
        <CornerUpLeft size={13} /> DRILL PATH:
      </Typography>

      <Breadcrumbs
        separator={<ChevronRight size={13} color={theme.textMuted} />}
        aria-label="drill-down-path"
      >
        {history.map((pastQuery, index) => {
          const lastFilter = pastQuery.filters[pastQuery.filters.length - 1];
          const label =
            index === 0
              ? 'Base Query'
              : lastFilter
              ? `${lastFilter.fieldId}: ${lastFilter.value}`
              : `Step ${index}`;

          return (
            <Chip
              key={`history-${index}`}
              label={label}
              size="small"
              icon={index === 0 ? <Home size={12} /> : undefined}
              onClick={() => onNavigateBack(index)}
              sx={{
                height: 22,
                fontSize: '0.7rem',
                fontWeight: 600,
                cursor: 'pointer',
                bgcolor: theme.accentMuted,
                color: theme.accent,
                border: `1px solid ${theme.border}`,
                '&:hover': { bgcolor: theme.accentMuted },
              }}
            />
          );
        })}

        <Chip
          label={
            currentQuery.filters.length > 0
              ? `${currentQuery.filters[currentQuery.filters.length - 1].fieldId}: ${
                  currentQuery.filters[currentQuery.filters.length - 1].value
                }`
              : 'Current View'
          }
          size="small"
          sx={{
            height: 22,
            fontSize: '0.7rem',
            fontWeight: 800,
            bgcolor: theme.accent,
            color: theme.isDark ? theme.background : '#FFFFFF',
            border: '1px solid transparent',
          }}
        />
      </Breadcrumbs>
    </Box>
  );
};

export default DrilldownBreadcrumbs;
