import React from 'react';

/**
 * RTL bootstrap. MUI v7 + theme.direction handles MUI-generated styles; an
 * Emotion cache + stylis-plugin-rtl is needed for sx / custom emotion
 * styles, and plain `.css` files need postcss-rtlcss (already installed)
 * since `dir=rtl` only flips CSS logical properties that the author used.
 *
 * Stubbed: pure pass-through while we verify what `ar` actually flips on
 * its own. Will be replaced with:
 *   const rtlCache = createCache({ key: 'mui-rtl', stylisPlugins: [prefixer, rtlPlugin] });
 * after the ar locale smoke test.
 */
export function DirectionProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
