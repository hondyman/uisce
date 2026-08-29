import React from 'react';
import ViewAgendaIcon from '@mui/icons-material/ViewAgenda';
import ViewStreamIcon from '@mui/icons-material/ViewStream';
import TableChartIcon from '@mui/icons-material/TableChart';
import BarChartIcon from '@mui/icons-material/BarChart';
import SpeedIcon from '@mui/icons-material/Speed';
import LinkIcon from '@mui/icons-material/Link';
import type { ContainerWidgetType } from './pageStudioTypes';

export interface WidgetMeta {
  type: ContainerWidgetType | 'field' | 'relatedObject';
  label: string;
  description: string;
  Icon: React.ComponentType<{ fontSize?: number | string; sx?: object; color?: string }>;
  /** Widget primitives are always core (gold-copy primitives). Field items inherit from their BO. */
  isCoreDefault: boolean;
}

export const WIDGET_REGISTRY: Record<WidgetMeta['type'], WidgetMeta> = {
  field: {
    type: 'field',
    label: 'Form Field',
    description: 'A single form input bound to a Business Object field',
    Icon: (props) => <LinkIcon {...props} />,
    isCoreDefault: false,
  },
  section: {
    type: 'section',
    label: 'Section',
    description: 'Grouping container with a title — stacks children vertically',
    Icon: ViewAgendaIcon,
    isCoreDefault: true,
  },
  row: {
    type: 'row',
    label: 'Row',
    description: 'Horizontal layout container — places children side-by-side',
    Icon: ViewStreamIcon,
    isCoreDefault: true,
  },
  grid: {
    type: 'grid',
    label: 'Data Grid',
    description: 'Tabular data list bound to a Business Object',
    Icon: TableChartIcon,
    isCoreDefault: true,
  },
  chart: {
    type: 'chart',
    label: 'Analytics Chart',
    description: 'Chart bound to a Business Object via a query',
    Icon: BarChartIcon,
    isCoreDefault: true,
  },
  kpi: {
    type: 'kpi',
    label: 'KPI Tile',
    description: 'Single-value metric tile with optional trend indicator',
    Icon: SpeedIcon,
    isCoreDefault: true,
  },
  relatedObject: {
    type: 'relatedObject',
    label: 'Related Object',
    description: 'Embedded related Business Object viewer',
    Icon: LinkIcon,
    isCoreDefault: false,
  },
};

export const CONTAINER_WIDGET_DEFS = (Object.values(WIDGET_REGISTRY).filter(
  (w): w is WidgetMeta & { type: ContainerWidgetType } =>
    w.type !== 'field' && w.type !== 'relatedObject'
) as WidgetMeta[]);

export default WIDGET_REGISTRY;
