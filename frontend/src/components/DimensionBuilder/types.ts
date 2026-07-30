// src/components/DimensionBuilder/types.ts
import TextFieldsIcon from '@mui/icons-material/TextFields';
import HashIcon from '@mui/icons-material/Tag'; // No direct Hash in MUI, using Tag as approximation
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import LanguageIcon from '@mui/icons-material/Language';

export interface CaseWhen {
  sql: string;
  label: string | { sql: string };
}

export interface CaseElse {
  label: string | { sql: string };
}

export interface Granularity {
  id: string;
  name: string;
  interval: string;
  offset?: string;
  origin?: string;
  title?: string;
}

export interface Dimension {
  id: string;
  name: string;
  sql: string;
  type: 'string' | 'number' | 'time' | 'boolean' | 'geo';
  title?: string;
  description?: string;
  format?: 'link' | 'id' | 'currency' | 'percent' | 'number' | 'external_url' | '';
  meta?: Record<string, any>;
  primary_key?: boolean;
  public?: boolean;
  sub_query?: boolean;
  propagate_filters_to_sub_query?: boolean;
  case?: {
    when: CaseWhen[];
    else: CaseElse;
  };
  granularities?: Granularity[];
  isEditing?: boolean;
}

export const dimensionTypes = [
  { value: 'string' as const, label: 'String', icon: TextFieldsIcon },
  { value: 'number' as const, label: 'Number', icon: HashIcon },
  { value: 'time' as const, label: 'Time', icon: AccessTimeIcon },
  { value: 'boolean' as const, label: 'Boolean', icon: CheckCircleIcon },
  { value: 'geo' as const, label: 'Geo', icon: LanguageIcon }
];

export const formatOptions = [
  { value: '' as const, label: 'No Format' },
  { value: 'link' as const, label: 'Link' },
  { value: 'id' as const, label: 'ID' },
  { value: 'currency' as const, label: 'Currency' },
  { value: 'percent' as const, label: 'Percent' },
  { value: 'number' as const, label: 'Number' },
  { value: 'external_url' as const, label: 'External URL' }
];

export const timeUnits = ['second', 'minute', 'hour', 'day', 'week', 'month', 'quarter', 'year'];