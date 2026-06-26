// mergeProperties.ts
// Merge type-level defaults (typeConfig) with instance-level values (instanceProps).
// Instance values take precedence. The merge is shallow for top-level keys, but
// will deep-merge plain objects one level deep to preserve nested config.
export const mergeProperties = (typeConfig: any, instanceProps: any) => {
  if (!typeConfig && !instanceProps) return undefined;
  if (!typeConfig) return instanceProps;
  if (!instanceProps) return typeConfig;

  const result: any = { ...typeConfig };

  for (const k of Object.keys(instanceProps)) {
    const v = instanceProps[k];
    const tv = typeConfig[k];
    // If both are plain objects, shallow merge their keys
    if (v && typeof v === 'object' && !Array.isArray(v) && tv && typeof tv === 'object' && !Array.isArray(tv)) {
      result[k] = { ...tv, ...v };
    } else {
      result[k] = v;
    }
  }

  return result;
};

export default mergeProperties;
