// React default import removed — using automatic JSX runtime
import * as TablerIcons from '@tabler/icons-react';
import SqlEditorField from './SqlEditorField';
import NameTitleFields from './NameTitleFields';
import SourceFields from './SourceFields';
import AggregationFormatFields from './AggregationFormatFields';
import TypeAndDefaultFields from './TypeAndDefaultFields';
import LibraryPanel from './LibraryPanel';

interface Props {
  kind: string | null;
  targetTable?: string | { id: string; qualified_path: string } | null;
  formData: any;
  setFormData: (v: any) => void;
  setMode: (m: 'override' | 'custom' | null) => void;
  libraryPanelOpen: boolean;
  setLibraryPanelOpen: (b: boolean) => void;
  libraryByCategory: Record<string, any[]>;
  librarySearch: string;
  setLibrarySearch: (s: string) => void;
  onCreate: () => void;
  semanticModel?: any;
}

const AddElementCustomForm: React.FC<Props> = ({ kind, targetTable, formData, setFormData, setMode, libraryPanelOpen, setLibraryPanelOpen, libraryByCategory, librarySearch, setLibrarySearch, onCreate, semanticModel }) => {
  return (
    <div className="custom-form">
      {kind === 'measure' && (
        <div className="library-browse">
          <button className="btn btn-primary library-browse-btn" onClick={() => setLibraryPanelOpen(true)}>
            <TablerIcons.IconChartBar size={16} />
            Calculations Library
          </button>
        </div>
      )}

      {kind === 'measure' && (
        <LibraryPanel
          open={libraryPanelOpen}
          libraryByCategory={libraryByCategory}
          librarySearch={librarySearch}
          setLibrarySearch={setLibrarySearch}
          onClose={() => setLibraryPanelOpen(false)}
          onSelect={(opt) => { setFormData({ ...opt }); setLibraryPanelOpen(false); }}
        />
      )}

      <SourceFields formData={formData} setFormData={setFormData} disabledSourceTable={targetTable} semanticModel={semanticModel} />

  <NameTitleFields formData={formData} setFormData={setFormData} />
      <div className="form-group">
        <label>Description</label>
        <textarea value={formData.description || ''} onChange={(e) => setFormData({ ...formData, description: e.target.value })} rows={2} placeholder="Optional description" title="Description" />
      </div>
      {kind === 'measure' && (
        <AggregationFormatFields formData={formData} setFormData={setFormData} />
      )}
  <TypeAndDefaultFields kind={kind} formData={formData} setFormData={setFormData} />
      {/* SQL editor field */}
      <SqlEditorField
        value={formData.sql || ''}
        onChange={(value) => setFormData({ ...formData, sql: value })}
        placeholder={`SQL for ${kind}`}
        height={100}
      />
      <div className="actions">
        <button className="btn btn-secondary" onClick={() => setMode(null)}>Back</button>
        <button className="btn btn-primary" onClick={onCreate}>Create Custom</button>
      </div>
    </div>
  );
};

export default AddElementCustomForm;
