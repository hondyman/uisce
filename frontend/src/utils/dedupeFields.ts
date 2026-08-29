export function dedupeFields<T extends { name: string }>(fields: T[]): T[] {
  const seen = new Set<string>();
  return fields.filter(f => {
    if (seen.has(f.name)) return false;
    seen.add(f.name);
    return true;
  });
}
