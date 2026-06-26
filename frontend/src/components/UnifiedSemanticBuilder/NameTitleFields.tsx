// React default import removed — using automatic JSX runtime

interface Props {
  formData: any;
  setFormData: (v: any) => void;
}

const NameTitleFields: React.FC<Props> = ({ formData, setFormData }) => {
  return (
    <div className="two-col">
      <div className="form-group">
        <label>Name</label>
        <input value={formData.name || ''} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="unique_name" title="Element name" />
      </div>
      <div className="form-group">
        <label>Title</label>
        <input value={formData.title || ''} onChange={(e) => setFormData({ ...formData, title: e.target.value })} placeholder="Display Title" title="Display title" />
      </div>
    </div>
  );
};

export default NameTitleFields;
