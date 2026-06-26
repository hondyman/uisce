// React default import removed — using automatic JSX runtime
import * as TablerIcons from '@tabler/icons-react';

export function countExtensionChanges(changes: Record<string, any> | null | undefined): number {
  if (!changes) return 0;
  let count = 0;
  const listKeys = ['dimensions_added', 'measures_added', 'joins_added', 'filters_added'];
  listKeys.forEach(key => {
    if (Array.isArray(changes[key])) count += changes[key].length;
  });
  const mapKeys = ['dimensions_overridden', 'measures_overridden', 'joins_overridden', 'filters_overridden'];
  mapKeys.forEach(key => {
    if (changes[key] && typeof changes[key] === 'object') count += Object.keys(changes[key]).length;
  });
  return count;
}

export function renderExtensionChanges(changes: Record<string, any>) {
  const sections: Array<{ key: string; label: string; items?: string[]; map?: Record<string, string[]> }> = [];
  const listKeys = [
    { key: 'cube_fields', label: 'Cube Fields' },
    { key: 'dimensions_added', label: 'Dimensions Added' },
    { key: 'measures_added', label: 'Measures Added' },
    { key: 'joins_added', label: 'Joins Added' },
    { key: 'filters_added', label: 'Filters Added' },
  ];
  listKeys.forEach(({ key, label }) => {
    const v = changes[key];
    if (Array.isArray(v) && v.length > 0) sections.push({ key, label, items: v as string[] });
  });
  const mapKeys = [
    { key: 'dimensions_overridden', label: 'Dimensions Overridden' },
    { key: 'measures_overridden', label: 'Measures Overridden' },
    { key: 'joins_overridden', label: 'Joins Overridden' },
    { key: 'filters_overridden', label: 'Filters Overridden' },
  ];
  mapKeys.forEach(({ key, label }) => {
    const v = changes[key];
    if (v && typeof v === 'object') sections.push({ key, label, map: v as Record<string, string[]> });
  });

  if (sections.length === 0) {
    return <div className="changes-empty">No extension changes.</div>;
  }

  return (
    <>
      {sections.map((s) => (
        <div className="change-section" key={s.key}>
          <div className="change-title">{s.label}</div>
          {s.items && (
            <ul className="change-list">
              {s.items.map((it) => (
                <li key={it}>
                  <TablerIcons.IconPlus size={14} /> {it}
                </li>
              ))}
            </ul>
          )}
          {s.map && (
            <ul className="change-map">
              {Object.entries(s.map).map(([name, keys]) => (
                <li key={name}>
                  <TablerIcons.IconSettings size={14} /> <strong>{name}</strong>
                  {Array.isArray(keys) && keys.length > 0 && (
                    <span className="change-keys"> — {keys.join(', ')}</span>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      ))}
    </>
  );
}

export default {};
