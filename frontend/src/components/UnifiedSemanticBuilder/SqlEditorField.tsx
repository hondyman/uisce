// React default import removed — using automatic JSX runtime
import SqlMonacoEditor from '../SqlMonacoEditor';

interface Props {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  height?: number;
}

const SqlEditorField: React.FC<Props> = ({ value, onChange, placeholder, height = 100 }) => {
  return (
    <div className="form-group">
      <label>SQL Expression</label>
      <SqlMonacoEditor value={value} onChange={onChange} placeholder={placeholder} height={height} />
    </div>
  );
};

export default SqlEditorField;
