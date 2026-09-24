/**
 * Design/preview artboard widths. Published pages (`PageBrowser` at
 * /pages/:slug) put a 280px menu beside the content (`p: 3`). "Page" uses
 * that same content column so widgets in Design occupy the same pixels as
 * they will for a viewer. Graded sizes are explicit content-column widths.
 */
export const PAGE_BROWSER_NAV_PX = 280;

export type CanvasSizeId = 'page' | '1440' | '1280' | '1024' | '768' | '390';

export const CANVAS_SIZES: { id: CanvasSizeId; label: string; hint: string }[] = [
  { id: 'page', label: 'Page', hint: 'Same width as /pages/:slug' },
  { id: '1440', label: '1440', hint: 'Wide desktop' },
  { id: '1280', label: '1280', hint: 'Desktop' },
  { id: '1024', label: '1024', hint: 'Laptop' },
  { id: '768', label: '768', hint: 'Tablet' },
  { id: '390', label: '390', hint: 'Phone' },
];

export const canvasWidthPx = (id: CanvasSizeId, viewportWidth: number): number => {
  if (id === 'page') return Math.max(360, viewportWidth - PAGE_BROWSER_NAV_PX);
  return Number(id);
};
