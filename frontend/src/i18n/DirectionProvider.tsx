import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import RtlProvider from '@mui/system/RtlProvider';
import { isRtl } from './locales';

/**
 * RTL bootstrap. In MUI v7, the RtlProvider sets up a React context that
 * components like Drawer/MenuItem/Popper read to flip their physical properties
 * (borderLeft/Right, left/Right anchors) when dir=rtl. Without it, the
 * `theme.direction: 'rtl'` flag on its own only affects MUI-generated styles
 * via styleOverrides — every component using `borderLeft` etc. directly
 * stays LTR.
 *
 * The export is from `@mui/system/RtlProvider` as the default export.
 * (`@mui/material` does not re-export it; `useRtl` is the named hook export.)
 */
export function DirectionProvider({ children }: { children: React.ReactNode }) {
  const { i18n } = useTranslation();
  const dir = isRtl(i18n.language) ? 'rtl' : 'ltr';
  useEffect(() => {
    document.documentElement.dir = dir;
  }, [dir]);
  return isRtl(i18n.language) ? <RtlProvider>{children}</RtlProvider> : <>{children}</>;
}
