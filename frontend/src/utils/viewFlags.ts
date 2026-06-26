import { View } from '../types/views';

const truthy = (value: unknown): boolean => value === true || value === 'true';

const normalizeTags = (tags: unknown): string[] => {
  if (!Array.isArray(tags)) return [];
  return tags
    .map((tag) => {
      if (typeof tag === 'string') return tag.toLowerCase();
      if (tag && typeof tag === 'object' && 'name' in tag) {
        const maybeName = (tag as Record<string, unknown>)['name'];
        if (typeof maybeName === 'string') return maybeName.toLowerCase();
      }
      return null;
    })
    .filter((tag): tag is string => Boolean(tag));
};

const hasTag = (tags: string[], patterns: string[]): boolean =>
  tags.some((tag) => patterns.some((pattern) => tag === pattern || tag.includes(pattern)));

export const deriveViewFlags = (view: Partial<View> | any): { isCore: boolean; isCustom: boolean } => {
  if (!view) {
    return { isCore: false, isCustom: false };
  }

  const tags = normalizeTags((view && typeof view === 'object') ? (view as Record<string, unknown>).tags : undefined);
  const v = (view && typeof view === 'object') ? (view as Record<string, unknown>) : undefined;
  const name = typeof view?.name === 'string' ? view.name.toLowerCase() : '';
  const title = typeof view?.title === 'string' ? view.title.toLowerCase() : '';
  const extendsValue = typeof view?.extends === 'string' ? view.extends.trim() : '';

  const baseIsCore =
    truthy(v?.is_core) ||
    truthy(v?.isCore) ||
    truthy(typeof v?.flags === 'object' && v?.flags ? (v.flags as Record<string, unknown>)['is_core'] : undefined) ||
    truthy(typeof v?.flags === 'object' && v?.flags ? (v.flags as Record<string, unknown>)['isCore'] : undefined) ||
    truthy(typeof v?.metadata === 'object' && v?.metadata ? (v.metadata as Record<string, unknown>)['is_core'] : undefined) ||
    truthy(typeof v?.metadata === 'object' && v?.metadata ? (v.metadata as Record<string, unknown>)['isCore'] : undefined) ||
    truthy(typeof v?.meta === 'object' && v?.meta ? (v.meta as Record<string, unknown>)['is_core'] : undefined) ||
    truthy(typeof v?.meta === 'object' && v?.meta ? (v.meta as Record<string, unknown>)['isCore'] : undefined) ||
    truthy(typeof v?.attributes === 'object' && v?.attributes ? (v.attributes as Record<string, unknown>)['is_core'] : undefined) ||
    truthy(typeof v?.attributes === 'object' && v?.attributes ? (v.attributes as Record<string, unknown>)['isCore'] : undefined);

  const baseIsCustom =
    truthy(v?.is_custom) ||
    truthy(v?.isCustom) ||
    truthy(typeof v?.flags === 'object' && v?.flags ? (v.flags as Record<string, unknown>)['is_custom'] : undefined) ||
    truthy(typeof v?.flags === 'object' && v?.flags ? (v.flags as Record<string, unknown>)['isCustom'] : undefined) ||
    truthy(typeof v?.metadata === 'object' && v?.metadata ? (v.metadata as Record<string, unknown>)['is_custom'] : undefined) ||
    truthy(typeof v?.metadata === 'object' && v?.metadata ? (v.metadata as Record<string, unknown>)['isCustom'] : undefined) ||
    truthy(typeof v?.meta === 'object' && v?.meta ? (v.meta as Record<string, unknown>)['is_custom'] : undefined) ||
    truthy(typeof v?.meta === 'object' && v?.meta ? (v.meta as Record<string, unknown>)['isCustom'] : undefined) ||
    truthy(typeof v?.attributes === 'object' && v?.attributes ? (v.attributes as Record<string, unknown>)['is_custom'] : undefined) ||
    truthy(typeof v?.attributes === 'object' && v?.attributes ? (v.attributes as Record<string, unknown>)['isCustom'] : undefined);

  const derivedCore =
    baseIsCore ||
    hasTag(tags, ['core', 'system-core', 'core-model', 'base']) ||
    name.startsWith('core_') ||
    name.startsWith('core-') ||
    name.endsWith('_core') ||
    name.endsWith('-core') ||
    title.startsWith('core ');

  const nameTokens = name.replace(/[_-]+/g, ' ');
  const titleTokens = title.replace(/[_-]+/g, ' ');

  const derivedCustom =
    baseIsCustom ||
    hasTag(tags, ['custom', 'client', 'client-specific']) ||
    /custom/.test(nameTokens) ||
    /custom/.test(titleTokens) ||
    (!derivedCore && Boolean(extendsValue));

  const isCustom = derivedCustom;
  const isCore = derivedCore || (!isCustom && !extendsValue);

  return { isCore, isCustom };
};
