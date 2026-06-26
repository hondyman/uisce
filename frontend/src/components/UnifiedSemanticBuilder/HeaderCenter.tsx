// React default import removed — using automatic JSX runtime

interface Props {
  modelName: string;
  setModelName: (name: string) => void;
  selectedModelNameReadonly?: boolean;
}

const HeaderCenter: React.FC<Props> = ({ modelName, setModelName, selectedModelNameReadonly = false }) => (
  <div className="header-center">
    <div className="model-config">
      <div className="model-name-row">
        <label htmlFor="model-name" className="form-label-inline">Model Name</label>
        <input id="model-name" type="text" value={modelName} onChange={(e) => setModelName(e.target.value)} placeholder="Enter semantic model name" className="form-input" readOnly={selectedModelNameReadonly} />
      </div>
    </div>
  </div>
);

export default HeaderCenter;
