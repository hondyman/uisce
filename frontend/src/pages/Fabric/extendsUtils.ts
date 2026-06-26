export function analyzeExtendsSelection(newBaseKey: string | null | undefined, selectedModelKey: string | null | undefined, currentParentKey: string | null | undefined) {
  const normalizeKey = (s: any) => {
    let v = (s || '').toString().trim().toLowerCase();
    if (!v) return '';
    // Convert dot notation to slash path
    if (v.includes('.')) v = v.replace(/\.+/g, '/');
    // Ensure single leading slash
    if (!v.startsWith('/')) v = '/' + v;
    v = v.replace(/\/+/, '/');
    return v;
  };
  const newNorm = normalizeKey(newBaseKey);
  const selNorm = normalizeKey(selectedModelKey);
  const parentNorm = normalizeKey(currentParentKey);
  if (!newNorm) return { valid: false, reason: 'empty' };
  if (newNorm === selNorm) return { valid: false, reason: 'self' };
  if (newNorm === parentNorm) return { valid: false, reason: 'idempotent' };
  return { valid: true };
}
