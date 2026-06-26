// React default import removed — using automatic JSX runtime

interface Props {
  formData: any;
  setFormData: (v: any) => void;
}

const AggregationFormatFields: React.FC<Props> = ({ formData, setFormData }) => {
  return (
    <div className="two-col">
      <div className="form-group">
        <label>Aggregation</label>
        <select value={formData.aggregationType || 'sum'} onChange={(e) => setFormData({ ...formData, aggregationType: e.target.value })} title="Aggregation type">
          <option value="sum">Sum</option>
          <option value="count">Count</option>
          <option value="avg">Average</option>
          <option value="min">Min</option>
          <option value="max">Max</option>
        </select>
      </div>
      <div className="form-group">
        <label>Format</label>
        <input value={formData.format || ''} onChange={(e) => setFormData({ ...formData, format: e.target.value })} placeholder="#,##0.00" title="Format mask" />
      </div>
    </div>
  );
};

export default AggregationFormatFields;
