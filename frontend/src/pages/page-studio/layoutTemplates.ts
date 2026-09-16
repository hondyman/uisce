import type { PageLayout, PanelNodeProps } from '../../types/pageStudio';

export interface LayoutTemplate {
  id: string;
  name: string;
  description: string;
  /** Grid used only for the picker's schematic thumbnail: rows of relative flex weights. */
  thumbnail: number[][];
  /** Empty container node ids, in the order widgets should be dropped into them. */
  build: () => { layout: PageLayout; sectionIds: string[] };
}

const col = (id: string, flex?: number): Record<string, unknown> => ({ id, type: 'Column', children: [], style: flex ? { flex: String(flex) } : undefined });

/**
 * Standard section-based body layouts an investment-management page
 * typically needs. Each template is a full PageLayout with named,
 * currently-empty section node ids (`sectionIds`) - the designer's
 * TemplatePickerDialog seeds a new tab/page with one of these, and
 * generatePageDraft.ts (the AI generator) fills sectionIds with generated
 * widgets in order. Side panels and a top tile row are NOT part of these
 * templates - they're added ad hoc from the palette (Panel/Tile component
 * types) since which pages need them is much more page-specific than the
 * body shape is.
 */
export const LAYOUT_TEMPLATES: LayoutTemplate[] = [
  {
    id: 'single-column',
    name: 'Single Column',
    description: 'One full-width section. Best for a focused table or form.',
    thumbnail: [[1]],
    build: () => {
      const rootId = 'root';
      return {
        layout: { root: rootId, nodes: { [rootId]: { id: rootId, type: 'Column', children: [] } } },
        sectionIds: [rootId],
      };
    },
  },
  {
    id: 'two-column',
    name: 'Two Column (60/40)',
    description: 'A wider primary section beside a narrower secondary one - e.g. a chart next to key figures.',
    thumbnail: [[3, 2]],
    build: () => {
      const rootId = 'root';
      const leftId = 'section_left';
      const rightId = 'section_right';
      return {
        layout: {
          root: rootId,
          nodes: {
            [rootId]: { id: rootId, type: 'Row', children: [leftId, rightId] },
            [leftId]: col(leftId, 3) as any,
            [rightId]: col(rightId, 2) as any,
          },
        },
        sectionIds: [leftId, rightId],
      };
    },
  },
  {
    id: 'three-column',
    name: 'Three Column',
    description: 'Three equal-width sections side by side - compare several breakdowns at once.',
    thumbnail: [[1, 1, 1]],
    build: () => {
      const rootId = 'root';
      const ids = ['section_a', 'section_b', 'section_c'];
      const nodes: Record<string, unknown> = { [rootId]: { id: rootId, type: 'Row', children: ids } };
      ids.forEach((id) => { nodes[id] = col(id, 1); });
      return { layout: { root: rootId, nodes: nodes as any }, sectionIds: ids };
    },
  },
  {
    id: 'dashboard-grid',
    name: 'Dashboard Grid',
    description: 'A KPI row across the top, a chart, then a full-width table below - the classic summary dashboard.',
    thumbnail: [[1], [1, 1], [1]],
    build: () => {
      const rootId = 'root';
      const kpiRowId = 'section_kpis';
      const chartRowId = 'row_chart';
      const chartId = 'section_chart';
      const secondaryId = 'section_secondary';
      const tableId = 'section_table';
      return {
        layout: {
          root: rootId,
          nodes: {
            [rootId]: { id: rootId, type: 'Column', children: [kpiRowId, chartRowId, tableId] },
            [kpiRowId]: { id: kpiRowId, type: 'Row', children: [] },
            [chartRowId]: { id: chartRowId, type: 'Row', children: [chartId, secondaryId] },
            [chartId]: col(chartId, 2) as any,
            [secondaryId]: col(secondaryId, 1) as any,
            [tableId]: { id: tableId, type: 'Column', children: [] },
          },
        },
        sectionIds: [kpiRowId, chartId, secondaryId, tableId],
      };
    },
  },
  {
    id: 'master-detail',
    name: 'Master-Detail',
    description: 'A narrow, collapsible list rail on the left with a main detail area - browse records, inspect one.',
    thumbnail: [[1, 3]],
    build: () => {
      const rootId = 'root';
      const panelId = 'panel_list';
      const listSectionId = 'section_list';
      const detailId = 'section_detail';
      const panelProps: PanelNodeProps = { side: 'left', collapsible: true, defaultOpen: true, widthPx: 320, label: 'List' };
      return {
        layout: {
          root: rootId,
          nodes: {
            [rootId]: { id: rootId, type: 'Row', children: [panelId, detailId] },
            [panelId]: { id: panelId, type: 'Panel', props: panelProps as unknown as Record<string, unknown>, children: [listSectionId] },
            [listSectionId]: { id: listSectionId, type: 'Column', children: [] },
            [detailId]: { id: detailId, type: 'Column', children: [], style: { flex: '1' } },
          },
        },
        sectionIds: [listSectionId, detailId],
      };
    },
  },
  {
    id: 'two-column-side-panel',
    name: 'Two Column + Side Panel',
    description: 'Main content with a collapsible right-hand rail - good for filters or a detail inspector that slides away.',
    thumbnail: [[3, 1]],
    build: () => {
      const rootId = 'root';
      const mainId = 'section_main';
      const panelId = 'panel_side';
      const panelSectionId = 'section_panel';
      const panelProps: PanelNodeProps = { side: 'right', collapsible: true, defaultOpen: true, widthPx: 340, label: 'Details' };
      return {
        layout: {
          root: rootId,
          nodes: {
            [rootId]: { id: rootId, type: 'Row', children: [mainId, panelId] },
            [mainId]: { id: mainId, type: 'Column', children: [], style: { flex: '1' } },
            [panelId]: { id: panelId, type: 'Panel', props: panelProps as unknown as Record<string, unknown>, children: [panelSectionId] },
            [panelSectionId]: { id: panelSectionId, type: 'Column', children: [] },
          },
        },
        sectionIds: [mainId, panelSectionId],
      };
    },
  },
];

export const getLayoutTemplate = (id: string): LayoutTemplate =>
  LAYOUT_TEMPLATES.find((t) => t.id === id) || LAYOUT_TEMPLATES[0];
