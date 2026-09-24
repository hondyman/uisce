import { describe, expect, it } from 'vitest';
import { PAGE_BROWSER_NAV_PX, canvasWidthPx } from '../../pages/page-studio/canvasSizes';

describe('canvasWidthPx', () => {
  it('Page size is the published content column (viewport minus PageBrowser menu)', () => {
    expect(canvasWidthPx('page', 1600)).toBe(1600 - PAGE_BROWSER_NAV_PX);
  });

  it('graded sizes are exact content widths', () => {
    expect(canvasWidthPx('1280', 2000)).toBe(1280);
    expect(canvasWidthPx('390', 2000)).toBe(390);
  });
});
