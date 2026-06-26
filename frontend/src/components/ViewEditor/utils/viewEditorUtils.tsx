import {
  ViewColumn,
  Functions,
  TextFields,
  Numbers,
  Calculate,
  CheckBox,
  CalendarToday,
  Fingerprint,
  Schedule,
  AccessTime,
  ArrowForward,
} from '@mui/icons-material';

export const getDatatypeIcon = (datatype: string, itemType?: string, itemName?: string) => {
  const lowerType = datatype?.toLowerCase() || '';
  const lowerName = itemName?.toLowerCase() || '';
  const iconProps = { fontSize: 'small' as const, sx: { opacity: 0.7 } };

  // For measures, try to detect aggregation type from name
  if (itemType === 'measure') {
    if (lowerName.includes('count') || lowerName.includes('cnt')) {
      return <Numbers {...iconProps} sx={{ ...iconProps.sx, color: 'success.main' }} />;
    }
    if (lowerName.includes('sum') || lowerName.includes('total')) {
      return <Calculate {...iconProps} sx={{ ...iconProps.sx, color: 'warning.main' }} />;
    }
    if (lowerName.includes('avg') || lowerName.includes('average')) {
      return <Calculate {...iconProps} sx={{ ...iconProps.sx, color: 'info.main' }} />;
    }
    if (lowerName.includes('min')) {
      return <ArrowForward {...iconProps} sx={{ ...iconProps.sx, color: 'error.main', transform: 'rotate(-90deg)' }} />;
    }
    if (lowerName.includes('max')) {
      return <ArrowForward {...iconProps} sx={{ ...iconProps.sx, color: 'success.main', transform: 'rotate(90deg)' }} />;
    }
  }

  // Datatype-based icons
  switch (lowerType) {
    case 'string':
    case 'text':
      return <TextFields {...iconProps} sx={{ ...iconProps.sx, color: 'text.secondary' }} />;
    case 'integer':
    case 'int':
    case 'number':
      return <Numbers {...iconProps} sx={{ ...iconProps.sx, color: 'warning.main' }} />;
    case 'decimal':
    case 'float':
    case 'double':
      return <Calculate {...iconProps} sx={{ ...iconProps.sx, color: 'info.main' }} />;
    case 'boolean':
    case 'bool':
      return <CheckBox {...iconProps} sx={{ ...iconProps.sx, color: 'success.main' }} />;
    case 'date':
      return <CalendarToday {...iconProps} sx={{ ...iconProps.sx, color: 'primary.main' }} />;
    case 'time':
      return <Schedule {...iconProps} sx={{ ...iconProps.sx, color: 'primary.main' }} />;
    case 'datetime':
    case 'timestamp':
      return <AccessTime {...iconProps} sx={{ ...iconProps.sx, color: 'primary.main' }} />;
    case 'uuid':
      return <Fingerprint {...iconProps} sx={{ ...iconProps.sx, color: 'secondary.main' }} />;
    default:
      return <TextFields {...iconProps} sx={{ ...iconProps.sx, color: 'text.secondary' }} />;
  }
};

export const getDimensionMeasureIcon = (type: string) => {
  const iconProps = { fontSize: 'small' as const };

  switch (type) {
    case 'dimension':
      return <ViewColumn sx={{ color: 'success.main', ...iconProps }} />;
    case 'measure':
      return <Functions sx={{ color: 'info.main', ...iconProps }} />;
    default:
      return null;
  }
};

export const buildSelectedRefs = (viewData?: any): Set<string> => {
  const s = new Set<string>();

  const addRef = (value?: string | null) => {
    if (!value) return;
    const normalized = value.trim().toLowerCase();
    if (normalized) {
      s.add(normalized);
      const stripped = normalized
        .replace(/\s*\(custom\)\s*/gi, '')
        .replace(/\s*\(core\)\s*/gi, '')
        .replace(/^\/public\//, '')
        .replace(/^\//, '')
        .trim();
      if (stripped && stripped !== normalized) s.add(stripped);
    }
  };

  // Add cubes
  const cubesRefList = Array.isArray(viewData?.cubes) ? viewData.cubes : [];
  cubesRefList.forEach((c: any) => {
    if (typeof c === 'string') {
      addRef(c);
    } else if (c && typeof c === 'object') {
      addRef(c.id ? String(c.id) : undefined);
      addRef(c.model_key ? String(c.model_key) : undefined);
      addRef(c.name ? String(c.name) : undefined);
    }
  });

  // Add join paths
  const joinPathsList = Array.isArray(viewData?.join_paths) ? viewData.join_paths : [];
  joinPathsList.forEach((jp: any) => {
    if (typeof jp === 'string') {
      addRef(jp);
    } else if (jp && typeof jp === 'object') {
      addRef(jp.id ? String(jp.id) : undefined);
      addRef(jp.path ? String(jp.path) : undefined);
      addRef(jp.label ? String(jp.label) : undefined);
    }
  });

  // Add extends
  const extendsId = (viewData && typeof viewData.extends === 'string') ? viewData.extends : (viewData && viewData.extends && (viewData.extends.id || viewData.extends.ID || viewData.extends.name));
  if (extendsId && String(extendsId).trim()) addRef(String(extendsId));

  return s;
};

export const getExtendsId = (viewData?: any): string => {
  if (!viewData) return '';
  if (typeof viewData.extends === 'string') return String(viewData.extends);
  if (viewData.extends && typeof viewData.extends === 'object') return String(viewData.extends.id || viewData.extends.ID || viewData.extends.name || '');
  return '';
};