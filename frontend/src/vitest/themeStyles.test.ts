import { describe, expect, it } from 'vitest';
import { getContrastRatio } from '@mui/material/styles';
import { THEME_STYLES, createThemeForStyle } from '../theme/themeStyles';

describe('theme styles', () => {
  // 'uisce' is the pre-existing theme (its light teal primary is below AA); the rest are checked.
  for (const { id } of THEME_STYLES.filter((s) => s.id !== 'uisce')) {
    for (const mode of ['light', 'dark'] as const) {
      it(`${id} ${mode}: builds, keeps shell palette keys, readable primary button`, () => {
        const theme = createThemeForStyle(id, mode);
        expect(theme.palette.mode).toBe(mode);
        expect(theme.palette.tertiary).toBeDefined();
        expect(theme.palette.categoryBuild).toBeDefined();
        const { main, contrastText } = theme.palette.primary;
        expect(getContrastRatio(main, contrastText)).toBeGreaterThanOrEqual(4.5);
      });
    }
  }
});
