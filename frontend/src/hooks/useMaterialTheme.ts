import { useTheme } from '@mui/material/styles';

/**
 * useMaterialTheme - Returns theme-aware colors for consistent MUI styling
 * Production-ready hook for accessing Material UI theme colors
 * 
 * @returns Object containing theme-aware color values
 */
export const useMaterialTheme = () => {
  const theme = useTheme();

  return {
    // Text colors
    textColor: theme.palette.text.primary,
    textSecondaryColor: theme.palette.text.secondary,

    // Background colors
    backgroundColor: theme.palette.background.paper,
    backgroundSecondaryColor: theme.palette.background.default,

    // Border colors
    borderColor: theme.palette.divider,

    // Chart colors
    gridColor: theme.palette.mode === 'dark'
      ? 'rgba(255, 255, 255, 0.1)'
      : 'rgba(0, 0, 0, 0.1)',

    // Status colors
    successColor: theme.palette.success.main,
    errorColor: theme.palette.error.main,
    warningColor: theme.palette.warning.main,
    infoColor: theme.palette.info.main,

    // Semantic colors for data viz
    positiveColor: theme.palette.success.main,
    negativeColor: theme.palette.error.main,
    neutralColor: theme.palette.grey[500],

    // Hover states
    hoverBackgroundColor: theme.palette.action.hover,
    selectedBackgroundColor: theme.palette.action.selected,

    // Borders
    lightBorder: theme.palette.divider,
  };
};
