import { useLayoutEffect, useState } from 'react';

/**
 * Height that fills the space below an element, down to the bottom of the
 * area that scrolls it (the app's main pane) or the viewport. The app header's
 * height varies (it wraps on narrow screens), so a fixed calc(100vh - Npx)
 * either leaves a gap or pushes the bottom off-screen. Attach the returned ref
 * to the page root.
 */
function scrollParent(el: HTMLElement): HTMLElement | null {
  for (let p = el.parentElement; p; p = p.parentElement) {
    if (/(auto|scroll)/.test(getComputedStyle(p).overflowY)) return p;
  }
  return null;
}

export function useFillHeight(min = 480) {
  const [el, setEl] = useState<HTMLElement | null>(null);
  const [height, setHeight] = useState<number>();
  useLayoutEffect(() => {
    if (!el) return;
    const measure = () => {
      const sp = scrollParent(el);
      const bottom = sp ? sp.getBoundingClientRect().top + sp.clientHeight : window.innerHeight;
      setHeight(Math.max(min, Math.floor(bottom - el.getBoundingClientRect().top)));
    };
    measure();
    window.addEventListener('resize', measure);
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(measure) : null;
    ro?.observe(document.body);
    return () => {
      window.removeEventListener('resize', measure);
      ro?.disconnect();
    };
  }, [el, min]);
  return [setEl, height] as const;
}
