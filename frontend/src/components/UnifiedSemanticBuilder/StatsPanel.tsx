// React default import removed — using automatic JSX runtime

interface Props {
  issueCount?: number;
  modelSize?: number;
}

const StatsPanel: React.FC<Props> = ({ issueCount = 0, modelSize = 0 }) => {
  return (
    <aside className="workspace-stats-panel">
      <div className="workspace-stats-inner">
        <h4>Model stats</h4>
        <div>Issues: {issueCount}</div>
        <div>Elements: {modelSize}</div>
      </div>
    </aside>
  );
};

export default StatsPanel;
