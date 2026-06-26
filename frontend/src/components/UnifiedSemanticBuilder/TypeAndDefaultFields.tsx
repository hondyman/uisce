// React default import removed — using automatic JSX runtime

interface Props {
  kind: string | null;
  formData: any;
  setFormData: (v: any) => void;
}

const TypeAndDefaultFields: React.FC<Props> = ({ kind, formData, setFormData }) => {
  return (
    <>
      {kind === 'filter' && (
        <div className="form-group">
          <label>Default Value</label>
          <input value={formData.defaultValue || ''} onChange={(e) => setFormData({ ...formData, defaultValue: e.target.value })} placeholder="Default value" title="Default value" />
        </div>
      )}
      <div className="form-group">
        <label>Type</label>
        <input value={formData.type || (kind === 'measure' ? 'number' : 'string')} onChange={(e) => setFormData({ ...formData, type: e.target.value })} placeholder="Data type" title="Data type" />
      </div>
      <div className="form-group">
        <label>Format</label>
        <select value={formData.format || 'default'} onChange={(e) => setFormData({ ...formData, format: e.target.value })} title="Display format">
          <option value="default">Default</option>
          <option value="currency">Currency</option>
          <option value="percentage">Percentage</option>
          <option value="decimal">Decimal</option>
          <option value="integer">Integer</option>
          <option value="date">Date</option>
          <option value="datetime">DateTime</option>
          <option value="time">Time</option>
        </select>
      </div>
    </>
  );
};

export default TypeAndDefaultFields;
