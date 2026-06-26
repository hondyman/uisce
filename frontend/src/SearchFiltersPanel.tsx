import type { SearchFilters } from './types';

// These would be real, styled components in a full app
const Select = ({ label, value, onChange, multiple }: any) => (
  <label>{label}: <select value={value} onChange={e => (onChange as any)(multiple ? [...e.target.selectedOptions].map((o: any) => o.value) : e.target.value)} multiple={multiple}><option>{Array.isArray(value) ? value.join(',') : value}</option></select></label>
);
const TagPicker = ({ label, value, onChange }: any) => (
  <label>{label}: <input value={value.join(',')} onChange={e => (onChange as any)(e.target.value.split(','))} /></label>
);

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export default function SearchFiltersPanel({ filters, onChange }: { filters: SearchFilters; onChange: (f: SearchFilters) => void }) {
  return (
    <div className="search-filters">
      <Select label="Type" options={['query', 'workbook', 'view']} multiple value={filters.type} onChange={(type: any) => onChange({ ...filters, type })} />
      <Select label="Scope" options={['mine', 'shared', 'all']} value={filters.scope} onChange={(scope: any) => onChange({ ...filters, scope })} />
      <TagPicker label="Tags" value={filters.tags} onChange={(tags: any) => onChange({ ...filters, tags })} />
    </div>
  );
}