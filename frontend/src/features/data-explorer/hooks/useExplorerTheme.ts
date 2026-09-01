import { useMemo } from 'react';
import { useTheme } from '@mui/material/styles';
import { alpha } from '@mui/material/styles';
import { uisceOcean, uisceGold, uisceSeafoam, uisceAbyss, uisceTide, semanticSuccess, semanticWarning, semanticError } from '../../../theme/palette';

export interface ExplorerTheme {
  accent: string;
  accentDark: string;
  accentHover: string;
  accentMuted: string;
  background: string;
  backgroundElevated: string;
  backgroundOverlay: string;
  border: string;
  borderSubtle: string;
  text: string;
  textSecondary: string;
  textMuted: string;
  success: string;
  successLight: string;
  warning: string;
  warningLight: string;
  error: string;
  errorLight: string;
  info: string;
  infoLight: string;
  chartPalette: string[];
  isDark: boolean;
}

export function useExplorerTheme(): ExplorerTheme {
  const muiTheme = useTheme();
  const isDark = muiTheme.palette.mode === 'dark';

  return useMemo(() => {
    if (isDark) {
      return {
        accent: uisceOcean.main,
        accentDark: uisceOcean.dark,
        accentHover: uisceOcean.light,
        accentMuted: uisceOcean.glowLight,
        background: uisceAbyss[900],
        backgroundElevated: uisceAbyss[800],
        backgroundOverlay: uisceAbyss[700],
        border: uisceAbyss.border,
        borderSubtle: uisceAbyss.divider,
        text: '#E8F4FF',
        textSecondary: '#7BA8C4',
        textMuted: 'rgba(232, 244, 255, 0.5)',
        success: semanticSuccess.dark,
        successLight: semanticSuccess.DEFAULT,
        warning: semanticWarning.dark,
        warningLight: semanticWarning.DEFAULT,
        error: semanticError.dark,
        errorLight: semanticError.DEFAULT,
        info: '#38BDF8',
        infoLight: '#BAE6FD',
        chartPalette: [uisceOcean.main, uisceGold.main, uisceSeafoam.main, semanticSuccess.DEFAULT, semanticWarning.DEFAULT, semanticError.DEFAULT, '#818CF8', '#F472B6'],
        isDark: true,
      };
    }

    return {
      accent: uisceOcean.main,
      accentDark: uisceOcean.dark,
      accentHover: uisceOcean.light,
      accentMuted: uisceOcean.glowLight,
      background: uisceTide[50],
      backgroundElevated: '#FFFFFF',
      backgroundOverlay: uisceTide[200],
      border: uisceTide.border,
      borderSubtle: uisceTide.divider,
      text: '#071526',
      textSecondary: 'rgba(7, 21, 38, 0.60)',
      textMuted: 'rgba(7, 21, 38, 0.40)',
      success: semanticSuccess.DEFAULT,
      successLight: semanticSuccess.light,
      warning: semanticWarning.DEFAULT,
      warningLight: semanticWarning.light,
      error: semanticError.DEFAULT,
      errorLight: semanticError.light,
      info: '#3B82F6',
      infoLight: '#DBEAFE',
      chartPalette: [uisceOcean.main, uisceGold.main, uisceSeafoam.main, semanticSuccess.DEFAULT, semanticWarning.DEFAULT, semanticError.DEFAULT, '#4F46E5', '#DB2777'],
      isDark: false,
    };
  }, [isDark]);
}

export function explorerAlpha(color: string, alphaValue: number): string {
  return alpha(color, alphaValue);
}
